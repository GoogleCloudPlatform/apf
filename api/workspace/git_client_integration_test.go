// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package workspace

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"google.com/apf/api/core"
)

func runLocalGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_TERMINAL_PROMPT=0",
		"GIT_AUTHOR_NAME=Test User",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test User",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s in %s failed: %v, output: %s", strings.Join(args, " "), dir, err, string(out))
	}
	return strings.TrimSpace(string(out))
}

func setupBareRemote(t *testing.T) (string, func()) {
	t.Helper()
	remoteDir, err := os.MkdirTemp("", "git_remote_bare")
	if err != nil {
		t.Fatalf("failed to create temp dir for remote: %v", err)
	}

	runLocalGit(t, remoteDir, "init", "--bare", "-b", "main")

	cleanup := func() {
		os.RemoveAll(remoteDir)
	}

	return remoteDir, cleanup
}

func setupApfTempWorkspace(t *testing.T) string {
	t.Helper()
	home, err := os.UserHomeDir()
	var baseDir string
	if err == nil && home != "" {
		baseDir = filepath.Join(home, ".apf", "tmp")
	} else {
		baseDir = os.TempDir()
	}
	_ = os.MkdirAll(baseDir, 0755)

	tempDir, err := os.MkdirTemp(baseDir, "test_ws_*")
	if err != nil {
		t.Fatalf("failed to create temporary workspace: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(tempDir)
	})
	return tempDir
}

func commitFileToBare(t *testing.T, remoteDir, branchName, relFilePath, content, commitMsg string) {
	t.Helper()
	tempClone, err := os.MkdirTemp("", "git_remote_clone")
	if err != nil {
		t.Fatalf("failed to create temp clone dir: %v", err)
	}
	defer os.RemoveAll(tempClone)

	// Try cloning
	cmd := exec.Command("git", "clone", "--branch", branchName, remoteDir, ".")
	cmd.Dir = tempClone
	if err := cmd.Run(); err != nil {
		// If branch doesn't exist or repo is empty
		runLocalGit(t, tempClone, "init", "-b", branchName)
		runLocalGit(t, tempClone, "remote", "add", "origin", remoteDir)
	}

	absPath := filepath.Join(tempClone, relFilePath)
	_ = os.MkdirAll(filepath.Dir(absPath), 0755)
	if err := os.WriteFile(absPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	runLocalGit(t, tempClone, "add", relFilePath)
	runLocalGit(t, tempClone, "commit", "-m", commitMsg)
	runLocalGit(t, tempClone, "push", "origin", fmt.Sprintf("refs/heads/%s:refs/heads/%s", branchName, branchName))
}


func TestGitClient_Integration_Flow(t *testing.T) {
	remoteDir, cleanupRemote := setupBareRemote(t)
	defer cleanupRemote()

	// Initial commit to main branch in remote
	commitFileToBare(t, remoteDir, "main", "README.md", "# Test Repo", "Initial commit")

	workspaceMount := setupApfTempWorkspace(t)
	client := NewGitClient(workspaceMount)
	repoCfg := &GitRepoConfig{
		RepoURL:         remoteDir,
		RepoAccessToken: "dummy",
		BaseBranch:      "main",
		SpecBranch:      "specs/test-user",
		UserId:          "test-user",
		ProjectId:       "proj123",
		SpecLocation:    "docs/spec",
	}

	ctx := testCtx()

	// 1. Initial Pull (should clone and create spec branch off main)
	err := client.Pull(ctx, "test-organization", repoCfg)
	if err != nil {
		t.Fatalf("Initial Pull failed: %v", err)
	}

	organizationID, _ := core.GetOrganizationID(ctx)
	clonePath := client.getClonePath(organizationID, repoCfg)
	if _, err := os.Stat(filepath.Join(clonePath, ".git")); os.IsNotExist(err) {
		t.Error("Expected .git directory to exist after Pull (clone)")
	}

	// Verify we are on spec branch
	branchName, err := client.runGit(ctx, clonePath, "", "symbolic-ref", "--short", "HEAD")
	if err != nil {
		t.Fatalf("Failed to get current branch: %v", err)
	}
	if branchName != repoCfg.SpecBranch {
		t.Errorf("Expected branch %s, got %s", repoCfg.SpecBranch, branchName)
	}

	// 2. Local Commit (should commit a spec file)
	specFileRel := "docs/spec/prd.md"
	specFileAbs := filepath.Join(clonePath, specFileRel)
	_ = os.MkdirAll(filepath.Dir(specFileAbs), 0755)
	if err := os.WriteFile(specFileAbs, []byte("## PRD V1"), 0644); err != nil {
		t.Fatal(err)
	}

	err = client.LocalCommit(ctx, "test-organization", repoCfg, specFileRel, "Add PRD V1", "User A", "usera@example.com")
	if err != nil {
		t.Fatalf("LocalCommit failed: %v", err)
	}

	// Verify that an empty commit (no changes) does not throw an error
	err = client.LocalCommit(ctx, "test-organization", repoCfg, specFileRel, "Add PRD V1 Again (Empty)", "User A", "usera@example.com")
	if err != nil {
		t.Fatalf("LocalCommit with clean working tree failed: %v", err)
	}

	// Verify commit exists locally
	lastMsg, err := client.runGit(ctx, clonePath, "", "log", "-1", "--pretty=%B")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(lastMsg, "Add PRD V1") {
		t.Errorf("Expected commit message 'Add PRD V1', got %q", lastMsg)
	}

	// 3. Push (should push local commit to remote)
	err = client.Push(ctx, "test-organization", repoCfg)
	if err != nil {
		t.Fatalf("Push failed: %v", err)
	}

	// Verify remote has the branch and commit
	remoteHead, err := client.runGit(ctx, remoteDir, "", "rev-parse", "refs/heads/"+repoCfg.SpecBranch)
	if err != nil {
		t.Fatalf("Remote spec branch not found: %v", err)
	}
	localHead, err := client.runGit(ctx, clonePath, "", "rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("Local HEAD not found: %v", err)
	}
	if remoteHead != localHead {
		t.Errorf("Remote branch ref %s doesn't match local head %s", remoteHead, localHead)
	}

	// 4. Remote changes occur (divergence)
	commitFileToBare(t, remoteDir, repoCfg.SpecBranch, "docs/spec/architecture.md", "## Arch", "Add Arch on remote")

	// Local change and commit
	if err := os.WriteFile(specFileAbs, []byte("## PRD V2"), 0644); err != nil {
		t.Fatal(err)
	}
	err = client.LocalCommit(ctx, "test-organization", repoCfg, specFileRel, "Update PRD V2", "User A", "usera@example.com")
	if err != nil {
		t.Fatalf("LocalCommit V2 failed: %v", err)
	}

	// 5. Pull (should fetch and rebase local "Update PRD V2" on top of remote "Add Arch on remote")
	err = client.Pull(ctx, "test-organization", repoCfg)
	if err != nil {
		t.Fatalf("Pull after divergence failed: %v", err)
	}

	// Verify commit history order
	logHistory, err := client.runGit(ctx, clonePath, "", "log", "--oneline")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(logHistory, "Update PRD V2") || !strings.Contains(logHistory, "Add Arch on remote") {
		t.Errorf("Expected rebased commit history containing both changes, got:\n%s", logHistory)
	}

	// 6. Push with recovery after divergence
	commitFileToBare(t, remoteDir, repoCfg.SpecBranch, "docs/spec/architecture.md", "## Arch V2", "Update Arch on remote V2")

	if err := os.WriteFile(specFileAbs, []byte("## PRD V3"), 0644); err != nil {
		t.Fatal(err)
	}
	err = client.LocalCommit(ctx, "test-organization", repoCfg, specFileRel, "Update PRD V3", "User A", "usera@example.com")
	if err != nil {
		t.Fatal(err)
	}

	err = client.Push(ctx, "test-organization", repoCfg)
	if err != nil {
		t.Fatalf("Push with conflict failed to auto-recover: %v", err)
	}
}

func TestGitClient_Integration_Locking(t *testing.T) {
	remoteDir, cleanupRemote := setupBareRemote(t)
	defer cleanupRemote()
	commitFileToBare(t, remoteDir, "main", "README.md", "# Test Repo", "Initial commit")

	workspaceMount := setupApfTempWorkspace(t)
	client := NewGitClient(workspaceMount)
	repoCfg := &GitRepoConfig{
		RepoURL:         remoteDir,
		RepoAccessToken: "dummy",
		BaseBranch:      "main",
		SpecBranch:      "specs/test-user",
		UserId:          "test-user",
		ProjectId:       "proj123",
		SpecLocation:    "docs/spec",
	}

	ctx := testCtx()

	// Acquire lock manually to block client
	lock, err := client.acquireLock(ctx, "test-organization", repoCfg)
	if err != nil {
		t.Fatal(err)
	}

	ctxTimeout, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	err = client.Pull(ctxTimeout, "test-organization", repoCfg)
	if err == nil {
		t.Error("Expected Pull to fail due to lock")
	} else if !strings.Contains(err.Error(), "workspace lock timeout") && !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Expected lock timeout error, got: %v", err)
	}

	lock.Unlock()

	// Now it should succeed
	err = client.Pull(ctx, "test-organization", repoCfg)
	if err != nil {
		t.Errorf("Pull failed after unlocking: %v", err)
	}
}

func TestGitClient_Integration_EmptyRemote(t *testing.T) {
	remoteDir, cleanupRemote := setupBareRemote(t)
	defer cleanupRemote()

	workspaceMount := setupApfTempWorkspace(t)
	client := NewGitClient(workspaceMount)
	repoCfg := &GitRepoConfig{
		RepoURL:         remoteDir,
		RepoAccessToken: "dummy",
		BaseBranch:      "main",
		SpecBranch:      "specs/test-user",
		UserId:          "test-user",
		ProjectId:       "proj123",
		SpecLocation:    "docs/spec",
	}

	ctx := testCtx()

	err := client.Pull(ctx, "test-organization", repoCfg)
	if err != nil {
		t.Fatalf("Pull failed: %v", err)
	}

	organizationID, _ := core.GetOrganizationID(ctx)
	clonePath := client.getClonePath(organizationID, repoCfg)
	specFileRel := "docs/spec/prd.md"
	specFileAbs := filepath.Join(clonePath, specFileRel)
	_ = os.MkdirAll(filepath.Dir(specFileAbs), 0755)
	if err := os.WriteFile(specFileAbs, []byte("## PRD V1"), 0644); err != nil {
		t.Fatal(err)
	}

	err = client.LocalCommit(ctx, "test-organization", repoCfg, specFileRel, "Add PRD V1", "User A", "usera@example.com")
	if err != nil {
		t.Fatalf("LocalCommit to empty remote failed: %v", err)
	}

	err = client.Push(ctx, "test-organization", repoCfg)
	if err != nil {
		t.Fatalf("Push to empty remote failed: %v", err)
	}
}

func TestGitClient_Integration_NonExistentBaseBranch(t *testing.T) {
	remoteDir, cleanupRemote := setupBareRemote(t)
	defer cleanupRemote()

	commitFileToBare(t, remoteDir, "main", "README.md", "# Test Repo", "Initial commit")

	workspaceMount := setupApfTempWorkspace(t)
	client := NewGitClient(workspaceMount)
	repoCfg := &GitRepoConfig{
		RepoURL:         remoteDir,
		RepoAccessToken: "dummy",
		BaseBranch:      "custom-base-branch",
		SpecBranch:      "specs/custom-user",
		UserId:          "custom-user",
		ProjectId:       "proj456",
		SpecLocation:    "docs/spec",
	}

	ctx := testCtx()

	// 1. Initial Pull
	err := client.Pull(ctx, "test-organization", repoCfg)
	if err != nil {
		t.Fatalf("Pull with non-existent base branch failed: %v", err)
	}

	organizationID, _ := core.GetOrganizationID(ctx)
	clonePath := client.getClonePath(organizationID, repoCfg)

	// Verify local HEAD is on SpecBranch
	headBranch, err := client.runGit(ctx, clonePath, "", "symbolic-ref", "--short", "HEAD")
	if err != nil {
		t.Fatalf("Failed to get HEAD branch: %v", err)
	}
	if headBranch != repoCfg.SpecBranch {
		t.Errorf("Expected branch %s, got %s", repoCfg.SpecBranch, headBranch)
	}

	// 2. Commit a file locally
	specFileRel := "docs/spec/prd.md"
	specFileAbs := filepath.Join(clonePath, specFileRel)
	_ = os.MkdirAll(filepath.Dir(specFileAbs), 0755)
	if err := os.WriteFile(specFileAbs, []byte("## Custom Base PRD"), 0644); err != nil {
		t.Fatal(err)
	}

	err = client.LocalCommit(ctx, "test-organization", repoCfg, specFileRel, "Add Custom Base PRD", "User B", "userb@example.com")
	if err != nil {
		t.Fatalf("LocalCommit failed: %v", err)
	}

	// 3. Push to remote
	err = client.Push(ctx, "test-organization", repoCfg)
	if err != nil {
		t.Fatalf("Push with non-existent base branch failed: %v", err)
	}

	// 4. Verify remote has the branch
	specRef, err := client.runGit(ctx, remoteDir, "", "rev-parse", "refs/heads/"+repoCfg.SpecBranch)
	if err != nil || specRef == "" {
		t.Errorf("Expected %s to exist on remote after push, but got error: %v", repoCfg.SpecBranch, err)
	}
}

func TestGitClient_StagePath(t *testing.T) {
	remoteDir, cleanupRemote := setupBareRemote(t)
	defer cleanupRemote()

	commitFileToBare(t, remoteDir, "main", "README.md", "# Test Repo", "Initial commit")

	workspaceMount := setupApfTempWorkspace(t)
	client := NewGitClient(workspaceMount)
	repoCfg := &GitRepoConfig{
		RepoURL:         remoteDir,
		RepoAccessToken: "dummy",
		BaseBranch:      "main",
		SpecBranch:      "specs/test-user",
		UserId:          "test-user",
		ProjectId:       "proj123",
		SpecLocation:    "docs/spec",
	}

	ctx := context.Background()
	err := client.EnsureRepoInitialized(ctx, "test-organization", repoCfg)
	if err != nil {
		t.Fatalf("EnsureRepoInitialized failed: %v", err)
	}

	clonePath := filepath.Join(workspaceMount, "test-organization", repoCfg.UserId, repoCfg.ProjectId)

	// Create .agents file
	bmadConfigAbs := filepath.Join(clonePath, ".agents", "_bmad", "config.toml")
	_ = os.MkdirAll(filepath.Dir(bmadConfigAbs), 0755)
	if err := os.WriteFile(bmadConfigAbs, []byte("key=val"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := client.StagePath(ctx, "test-organization", repoCfg, ".agents"); err != nil {
		t.Fatalf("StagePath failed for .agents: %v", err)
	}

	statusOut, err := client.runGit(ctx, clonePath, "", "status", "--porcelain")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(statusOut, "A  .agents/_bmad/config.toml") && !strings.Contains(statusOut, ".agents") {
		t.Errorf("Expected .agents to be staged, got status: %s", statusOut)
	}
}

func TestGitClient_GetWorkspaceStatus_IncludesBranchCommits(t *testing.T) {
	remoteDir, cleanupRemote := setupBareRemote(t)
	defer cleanupRemote()

	commitFileToBare(t, remoteDir, "main", "README.md", "# Test Repo", "Initial commit")

	workspaceMount := setupApfTempWorkspace(t)
	client := NewGitClient(workspaceMount)
	repoCfg := &GitRepoConfig{
		RepoURL:         remoteDir,
		RepoAccessToken: "dummy",
		BaseBranch:      "main",
		SpecBranch:      "specs/test-user",
		UserId:          "test-user",
		ProjectId:       "proj123",
		SpecLocation:    "docs/spec",
	}

	ctx := context.Background()
	if err := client.EnsureRepoInitialized(ctx, "test-organization", repoCfg); err != nil {
		t.Fatalf("EnsureRepoInitialized failed: %v", err)
	}

	clonePath := filepath.Join(workspaceMount, "test-organization", repoCfg.UserId, repoCfg.ProjectId)

	// 1. Commit a file to specs/test-user using LocalCommit (working tree is clean for this file)
	prdAbs := filepath.Join(clonePath, "docs", "spec", "prd.md")
	_ = os.MkdirAll(filepath.Dir(prdAbs), 0755)
	if err := os.WriteFile(prdAbs, []byte("# PRD Content"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := client.LocalCommit(ctx, "test-organization", repoCfg, "docs/spec/prd.md", "Add PRD", "Author", "author@example.com"); err != nil {
		t.Fatalf("LocalCommit failed: %v", err)
	}

	// 2. Create an uncommitted untracked file
	archAbs := filepath.Join(clonePath, "docs", "spec", "architecture.md")
	if err := os.WriteFile(archAbs, []byte("# Architecture Content"), 0644); err != nil {
		t.Fatal(err)
	}

	// 3. Call GetWorkspaceStatus
	items, err := client.GetWorkspaceStatus(ctx, "test-organization", repoCfg)
	if err != nil {
		t.Fatalf("GetWorkspaceStatus failed: %v", err)
	}

	foundPRD := false
	foundArch := false
	for _, item := range items {
		if item.Path == "docs/spec/prd.md" {
			foundPRD = true
			if item.Staged != "committed" {
				t.Errorf("Expected PRD Staged='committed', got %q", item.Staged)
			}
			if !item.IsChecked {
				t.Errorf("Expected PRD IsChecked=true")
			}
		}
		if item.Path == "docs/spec/architecture.md" {
			foundArch = true
			if item.Worktree != "untracked" {
				t.Errorf("Expected Arch Worktree='untracked', got %q", item.Worktree)
			}
			if !item.IsChecked {
				t.Errorf("Expected Arch IsChecked=true")
			}
		}
	}

	if !foundPRD {
		t.Errorf("Expected committed file 'docs/spec/prd.md' to be included in GetWorkspaceStatus items: %+v", items)
	}
	if !foundArch {
		t.Errorf("Expected uncommitted file 'docs/spec/architecture.md' to be included in GetWorkspaceStatus items: %+v", items)
	}
}

func TestGitClient_GetWorkspaceStatus_Deduplication(t *testing.T) {
	remoteDir, cleanupRemote := setupBareRemote(t)
	defer cleanupRemote()

	commitFileToBare(t, remoteDir, "main", "README.md", "# Test Repo", "Initial commit")

	workspaceMount := setupApfTempWorkspace(t)
	client := NewGitClient(workspaceMount)
	repoCfg := &GitRepoConfig{
		RepoURL:         remoteDir,
		RepoAccessToken: "dummy",
		BaseBranch:      "main",
		SpecBranch:      "specs/test-user",
		UserId:          "test-user",
		ProjectId:       "proj123",
		SpecLocation:    "docs/spec",
	}

	ctx := context.Background()
	if err := client.EnsureRepoInitialized(ctx, "test-organization", repoCfg); err != nil {
		t.Fatalf("EnsureRepoInitialized failed: %v", err)
	}

	clonePath := filepath.Join(workspaceMount, "test-organization", repoCfg.UserId, repoCfg.ProjectId)

	// 1. Commit docs/spec/prd.md to specBranch
	prdAbs := filepath.Join(clonePath, "docs", "spec", "prd.md")
	_ = os.MkdirAll(filepath.Dir(prdAbs), 0755)
	if err := os.WriteFile(prdAbs, []byte("# PRD v1"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := client.LocalCommit(ctx, "test-organization", repoCfg, "docs/spec/prd.md", "Add PRD", "Author", "author@example.com"); err != nil {
		t.Fatalf("LocalCommit failed: %v", err)
	}

	// 2. Now edit docs/spec/prd.md in the working tree (dirty uncommitted changes on top of commit)
	if err := os.WriteFile(prdAbs, []byte("# PRD v1 with dirty edits"), 0644); err != nil {
		t.Fatal(err)
	}

	// 3. Call GetWorkspaceStatus
	items, err := client.GetWorkspaceStatus(ctx, "test-organization", repoCfg)
	if err != nil {
		t.Fatalf("GetWorkspaceStatus failed: %v", err)
	}

	countPRD := 0
	for _, item := range items {
		if item.Path == "docs/spec/prd.md" {
			countPRD++
			if item.Worktree != "modified" {
				t.Errorf("Expected Worktree='modified', got %q", item.Worktree)
			}
		}
	}

	if countPRD != 1 {
		t.Errorf("Expected exactly 1 entry for 'docs/spec/prd.md', got %d. Items: %+v", countPRD, items)
	}
}

func TestGitClient_GetWorkspaceStatus_SyncsBaseBranchFromRemote(t *testing.T) {
	remoteDir, cleanupRemote := setupBareRemote(t)
	defer cleanupRemote()

	commitFileToBare(t, remoteDir, "main", "README.md", "# Test Repo", "Initial commit")

	workspaceMount := setupApfTempWorkspace(t)
	client := NewGitClient(workspaceMount)
	repoCfg := &GitRepoConfig{
		RepoURL:         remoteDir,
		RepoAccessToken: "dummy",
		BaseBranch:      "main",
		SpecBranch:      "specs/test-user",
		UserId:          "test-user",
		ProjectId:       "proj123",
		SpecLocation:    "docs/spec",
	}

	ctx := context.Background()
	if err := client.EnsureRepoInitialized(ctx, "test-organization", repoCfg); err != nil {
		t.Fatalf("EnsureRepoInitialized failed: %v", err)
	}

	clonePath := filepath.Join(workspaceMount, "test-organization", repoCfg.UserId, repoCfg.ProjectId)

	// 1. Commit docs/spec/prd.md to specBranch
	prdAbs := filepath.Join(clonePath, "docs", "spec", "prd.md")
	_ = os.MkdirAll(filepath.Dir(prdAbs), 0755)
	if err := os.WriteFile(prdAbs, []byte("# PRD Content"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := client.LocalCommit(ctx, "test-organization", repoCfg, "docs/spec/prd.md", "Add PRD", "Author", "author@example.com"); err != nil {
		t.Fatalf("LocalCommit failed: %v", err)
	}

	// 2. Push a new commit to bare remote's main branch
	commitFileToBare(t, remoteDir, "main", "remote_update.md", "# Remote Update", "Remote commit on main")

	// Verify local main branch is behind remote main before GetWorkspaceStatus
	localBaseBefore, err := client.runGit(ctx, clonePath, "", "rev-parse", "refs/heads/main")
	if err != nil {
		t.Fatalf("Failed to rev-parse local main: %v", err)
	}

	// 3. Call GetWorkspaceStatus (should sync main with remote origin/main)
	items, err := client.GetWorkspaceStatus(ctx, "test-organization", repoCfg)
	if err != nil {
		t.Fatalf("GetWorkspaceStatus failed: %v", err)
	}

	// Verify local main branch ref has been updated
	localBaseAfter, err := client.runGit(ctx, clonePath, "", "rev-parse", "refs/heads/main")
	if err != nil {
		t.Fatalf("Failed to rev-parse local main after status: %v", err)
	}
	remoteBaseHash, err := client.runGit(ctx, clonePath, "", "rev-parse", "refs/remotes/origin/main")
	if err != nil {
		t.Fatalf("Failed to rev-parse origin/main: %v", err)
	}

	if localBaseAfter == localBaseBefore {
		t.Errorf("Expected local main to be updated after GetWorkspaceStatus, but stayed %s", localBaseBefore)
	}
	if localBaseAfter != remoteBaseHash {
		t.Errorf("Expected local main (%s) to match remote origin/main (%s)", localBaseAfter, remoteBaseHash)
	}

	// Verify committed file on specBranch is included
	foundPRD := false
	for _, item := range items {
		if item.Path == "docs/spec/prd.md" {
			foundPRD = true
			if item.Staged != "committed" {
				t.Errorf("Expected PRD Staged='committed', got %q", item.Staged)
			}
		}
	}
	if !foundPRD {
		t.Errorf("Expected committed file 'docs/spec/prd.md' in items: %+v", items)
	}
}
