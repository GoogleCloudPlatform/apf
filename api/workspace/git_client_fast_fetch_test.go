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
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

type spyGitCmdRunner struct {
	mu           sync.Mutex
	commands     [][]string
	defaultRunner *DefaultGitCmdRunner
}

func newSpyGitCmdRunner() *spyGitCmdRunner {
	return &spyGitCmdRunner{
		defaultRunner: &DefaultGitCmdRunner{},
	}
}

func (s *spyGitCmdRunner) Run(ctx context.Context, dir string, token string, args ...string) (string, string, error) {
	s.mu.Lock()
	s.commands = append(s.commands, append([]string(nil), args...))
	s.mu.Unlock()
	return s.defaultRunner.Run(ctx, dir, token, args...)
}

func (s *spyGitCmdRunner) getCommands() [][]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	copied := make([][]string, len(s.commands))
	copy(copied, s.commands)
	return copied
}

func (s *spyGitCmdRunner) clearCommands() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.commands = nil
}

func TestGitClient_PreFlightShortCircuit(t *testing.T) {
	remoteDir, cleanupRemote := setupBareRemote(t)
	defer cleanupRemote()

	commitFileToBare(t, remoteDir, "main", "README.md", "# Test Repo", "Initial commit")

	workspaceMount := setupApfTempWorkspace(t)
	spyRunner := newSpyGitCmdRunner()
	client := NewGitClient(workspaceMount).WithRunner(spyRunner)

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

	// 1. Initial Pull: Clones and syncs repo
	err := client.Pull(ctx, "test-organization", repoCfg)
	if err != nil {
		t.Fatalf("Initial Pull failed: %v", err)
	}

	clonePath := filepath.Join(workspaceMount, "test-organization", repoCfg.UserId, repoCfg.ProjectId)

	// Verify optimization configurations were written
	ver, _ := client.runGit(ctx, clonePath, "", "config", "protocol.version")
	if strings.TrimSpace(ver) != "2" {
		t.Errorf("Expected protocol.version=2, got %q", ver)
	}

	algo, _ := client.runGit(ctx, clonePath, "", "config", "fetch.negotiationAlgorithm")
	if strings.TrimSpace(algo) != "skipping" {
		t.Errorf("Expected fetch.negotiationAlgorithm=skipping, got %q", algo)
	}

	cg, _ := client.runGit(ctx, clonePath, "", "config", "fetch.writeCommitGraph")
	if strings.TrimSpace(cg) != "true" {
		t.Errorf("Expected fetch.writeCommitGraph=true, got %q", cg)
	}

	// 2. Clear command spy and run GetWorkspaceStatus (which triggers syncBaseBranchFromRemote)
	spyRunner.clearCommands()
	_, err = client.GetWorkspaceStatus(ctx, "test-organization", repoCfg)
	if err != nil {
		t.Fatalf("GetWorkspaceStatus failed: %v", err)
	}

	// In the second run, remote tip matches local tracking ref.
	// We expect ls-remote was run, but "fetch" was NEVER called (Short-Circuited!).
	cmds := spyRunner.getCommands()
	hasLsRemote := false
	hasFetch := false
	for _, cmd := range cmds {
		for i, arg := range cmd {
			if arg == "ls-remote" {
				hasLsRemote = true
			}
			if arg == "fetch" {
				// check if it's a fetch command
				if i == 0 || (i > 0 && cmd[i-1] != "-c") {
					hasFetch = true
				}
			}
		}
	}

	if !hasLsRemote {
		t.Errorf("Expected ls-remote to be executed for pre-flight check, but it was not found in: %+v", cmds)
	}
	if hasFetch {
		t.Errorf("Expected git fetch to be SHORT-CIRCUITED, but fetch command was executed: %+v", cmds)
	}

	// 3. Now push a new commit to bare remote's main branch
	commitFileToBare(t, remoteDir, "main", "new_file.md", "# New File", "Commit 2")

	// 4. Run GetWorkspaceStatus again (should detect mismatch, run fetch with optimizations, and update local main)
	spyRunner.clearCommands()
	_, err = client.GetWorkspaceStatus(ctx, "test-organization", repoCfg)
	if err != nil {
		t.Fatalf("GetWorkspaceStatus after remote commit failed: %v", err)
	}

	cmdsAfter := spyRunner.getCommands()
	hasFetchAfter := false
	hasNoTags := false
	for _, cmd := range cmdsAfter {
		for _, arg := range cmd {
			if arg == "fetch" {
				hasFetchAfter = true
			}
			if arg == "--no-tags" {
				hasNoTags = true
			}
		}
	}

	if !hasFetchAfter {
		t.Errorf("Expected git fetch to execute when remote has changed, but it was not found in: %+v", cmdsAfter)
	}
	if !hasNoTags {
		t.Errorf("Expected --no-tags in fetch arguments, but it was not found in: %+v", cmdsAfter)
	}
}
