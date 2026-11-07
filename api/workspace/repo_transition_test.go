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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"google.com/apf/api/db"
)

func setupTestGitClient(t *testing.T) (string, *GitClient) {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "repo_transition_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	client := NewGitClient(tempDir)
	gitClient = client

	t.Cleanup(func() {
		os.RemoveAll(tempDir)
	})

	return tempDir, client
}

func TestScenario1_AttachRemoteRepo_WithExistingLocalGitRepo(t *testing.T) {
	mountDir, client := setupTestGitClient(t)

	orgID := "org-123"
	userID := "user-456"
	projectID := "proj-attach-test"

	clonePath := filepath.Join(mountDir, orgID, userID, projectID)
	err := os.MkdirAll(clonePath, 0755)
	if err != nil {
		t.Fatalf("failed to create workspace dir: %v", err)
	}

	ctx := context.Background()

	// 1. Initialize local git repo
	if _, err := client.runGit(ctx, clonePath, "", "init", "-b", "main"); err != nil {
		if _, errFallback := client.runGit(ctx, clonePath, "", "init"); errFallback != nil {
			t.Fatalf("failed to init local repo: %v", errFallback)
		}
	}

	// Create and commit a local file
	testFile := filepath.Join(clonePath, "docs", "specs", "prd.md")
	_ = os.MkdirAll(filepath.Dir(testFile), 0755)
	if err := os.WriteFile(testFile, []byte("# PRD Document"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	if _, err := client.runGit(ctx, clonePath, "", "add", "docs/specs/prd.md"); err != nil {
		t.Fatalf("failed to add file: %v", err)
	}

	if _, err := client.runGit(ctx, clonePath, "",
		"-c", "user.name=Tester",
		"-c", "user.email=test@example.com",
		"commit", "-m", "Initial local commit",
	); err != nil {
		t.Fatalf("failed to commit local file: %v", err)
	}

	// 2. Attach remote repo
	newRepoURL := "https://github.com/example/attached-repo.git"
	proj := &db.Project{
		ID: projectID,
	}

	settings := map[string]interface{}{
		"baseBranch":       "main",
		"specBranchPrefix": "spec-",
	}

	err = HandleAttachRemoteRepo(ctx, orgID, userID, proj, newRepoURL, settings)
	if err != nil {
		t.Fatalf("HandleAttachRemoteRepo failed: %v", err)
	}

	// 3. Verify origin remote was configured
	remoteURL, err := client.runGit(ctx, clonePath, "", "remote", "get-url", "origin")
	if err != nil {
		t.Fatalf("origin remote not found after attach: %v", err)
	}
	if remoteURL != newRepoURL {
		t.Errorf("expected origin URL %q, got %q", newRepoURL, remoteURL)
	}

	// Verify local file remains intact
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Errorf("expected local file %s to be preserved, but it was deleted", testFile)
	}
}

func TestScenario1_AttachRemoteRepo_WorkspaceNeverOpened(t *testing.T) {
	mountDir, _ := setupTestGitClient(t)

	orgID := "org-123"
	userID := "user-456"
	projectID := "proj-never-opened"

	proj := &db.Project{
		ID: projectID,
	}

	newRepoURL := "https://github.com/example/never-opened.git"
	settings := map[string]interface{}{
		"baseBranch": "main",
	}

	ctx := context.Background()
	err := HandleAttachRemoteRepo(ctx, orgID, userID, proj, newRepoURL, settings)
	if err != nil {
		t.Fatalf("HandleAttachRemoteRepo should not fail for non-existent workspace: %v", err)
	}

	// Workspace directory should not exist or not have .git
	clonePath := filepath.Join(mountDir, orgID, userID, projectID)
	gitDir := filepath.Join(clonePath, ".git")
	if _, err := os.Stat(gitDir); !os.IsNotExist(err) {
		t.Errorf("expected .git directory to not be created, but it exists")
	}
}

func TestScenario2_DetachRemoteRepo(t *testing.T) {
	mountDir, client := setupTestGitClient(t)

	orgID := "org-123"
	userID := "user-456"
	projectID := "proj-detach-test"

	clonePath := filepath.Join(mountDir, orgID, userID, projectID)
	_ = os.MkdirAll(clonePath, 0755)

	ctx := context.Background()
	if _, err := client.runGit(ctx, clonePath, "", "init", "-b", "main"); err != nil {
		if _, errFallback := client.runGit(ctx, clonePath, "", "init"); errFallback != nil {
			t.Fatalf("failed to init repo: %v", errFallback)
		}
	}

	// Add origin remote
	repoURL := "https://github.com/example/detach-repo.git"
	if _, err := client.runGit(ctx, clonePath, "", "remote", "add", "origin", repoURL); err != nil {
		t.Fatalf("failed to add origin remote: %v", err)
	}

	// Write a test file
	testFile := filepath.Join(clonePath, "README.md")
	_ = os.WriteFile(testFile, []byte("Hello"), 0644)

	// Detach remote
	err := HandleDetachRemoteRepo(ctx, orgID, userID, projectID)
	if err != nil {
		t.Fatalf("HandleDetachRemoteRepo failed: %v", err)
	}

	// Verify origin remote is removed
	remotes, _ := client.runGit(ctx, clonePath, "", "remote")
	if strings.Contains(remotes, "origin") {
		t.Errorf("expected origin remote to be removed, but it still exists")
	}

	// Verify local file and .git remain intact
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Errorf("expected local file to remain intact after detach")
	}
}

type mockTransitionArtifactStore struct {
	artifacts  []db.Artifact
	deletedIDs []string
}

func (m *mockTransitionArtifactStore) ListArtifactsForProject(ctx context.Context, organizationID string, projectId string) ([]db.Artifact, error) {
	return m.artifacts, nil
}
func (m *mockTransitionArtifactStore) DeleteArtifact(ctx context.Context, organizationID string, artifactId string) error {
	m.deletedIDs = append(m.deletedIDs, artifactId)
	return nil
}
func (m *mockTransitionArtifactStore) UpdateArtifact(ctx context.Context, organizationID string, a *db.Artifact) error {
	return nil
}
func (m *mockTransitionArtifactStore) CreateArtifact(ctx context.Context, organizationID string, a *db.Artifact) error {
	return nil
}
func (m *mockTransitionArtifactStore) SyncArtifacts(ctx context.Context, organizationID string, projectId string, discovered []db.Artifact) (int, int, error) {
	return len(discovered), 0, nil
}

func TestScenario3_SwitchRemoteRepo(t *testing.T) {
	mountDir, _ := setupTestGitClient(t)

	orgID := "org-123"
	userID := "user-456"
	projectID := "proj-switch-test"

	clonePath := filepath.Join(mountDir, orgID, userID, projectID)
	_ = os.MkdirAll(clonePath, 0755)

	testFile := filepath.Join(clonePath, "old_spec.md")
	_ = os.WriteFile(testFile, []byte("Old Spec"), 0644)

	mockStore := &mockTransitionArtifactStore{
		artifacts: []db.Artifact{
			{ID: "art-old-1", ProjectId: projectID, Title: "old_spec.md"},
			{ID: "art-old-2", ProjectId: projectID, Title: "old_prd.md"},
		},
	}
	oldStore := ArtifactStore
	ArtifactStore = mockStore
	t.Cleanup(func() {
		ArtifactStore = oldStore
	})

	proj := &db.Project{
		ID: projectID,
	}

	ctx := context.Background()
	err := HandleSwitchRemoteRepo(ctx, orgID, userID, proj, "", nil)
	if err != nil {
		t.Fatalf("HandleSwitchRemoteRepo failed: %v", err)
	}

	// Verify clonePath was purged on disk
	if _, err := os.Stat(clonePath); !os.IsNotExist(err) {
		t.Errorf("expected workspace directory %s to be purged, but it still exists", clonePath)
	}

	// Verify existing Firestore DB artifacts were purged
	if len(mockStore.deletedIDs) != 2 {
		t.Errorf("expected 2 Firestore artifacts to be deleted, got %d (%v)", len(mockStore.deletedIDs), mockStore.deletedIDs)
	}
}
