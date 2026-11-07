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
	"testing"
)

func setupTestGitRepo(t *testing.T) (string, *GitClient) {
	tempDir, err := os.MkdirTemp("", "git-snapshot-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	clonePath := filepath.Join(tempDir, "org1", "user1", "proj1")
	if err := os.MkdirAll(clonePath, 0755); err != nil {
		t.Fatalf("failed to create clone path: %v", err)
	}

	client := NewGitClient(tempDir)
	ctx := context.Background()

	if _, err := client.runGit(ctx, clonePath, "", "init", "-b", "main"); err != nil {
		if _, errFallback := client.runGit(ctx, clonePath, "", "init"); errFallback != nil {
			t.Fatalf("failed to init git repo: %v", errFallback)
		}
	}

	testFile := filepath.Join(clonePath, "README.md")
	if err := os.WriteFile(testFile, []byte("# Initial Commit\n"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	if _, err := client.runGit(ctx, clonePath, "", "add", "README.md"); err != nil {
		t.Fatalf("failed to add file: %v", err)
	}

	if _, err := client.runGit(ctx, clonePath, "",
		"-c", "user.name=Test User",
		"-c", "user.email=test@example.com",
		"commit", "-m", "Initial commit",
	); err != nil {
		t.Fatalf("failed to commit initial file: %v", err)
	}

	return tempDir, client
}

func TestGitSnapshot_CreateAndRestore(t *testing.T) {
	tempDir, client := setupTestGitRepo(t)
	defer os.RemoveAll(tempDir)

	ctx := context.Background()
	orgID := "org1"
	repoCfg := &GitRepoConfig{
		UserId:    "user1",
		ProjectId: "proj1",
	}

	// 1. Create snapshot
	snapshot, err := client.CreateSnapshot(ctx, orgID, repoCfg, "Pre-sync safety snapshot")
	if err != nil {
		t.Fatalf("CreateSnapshot failed: %v", err)
	}

	if snapshot.ID == "" {
		t.Fatalf("Expected non-empty snapshot ID")
	}

	// 2. Modify working tree with new edit
	clonePath := filepath.Join(tempDir, "org1", "user1", "proj1")
	testFile := filepath.Join(clonePath, "README.md")
	if err := os.WriteFile(testFile, []byte("# Modified Commit\n"), 0644); err != nil {
		t.Fatalf("failed to overwrite file: %v", err)
	}

	// 3. Restore snapshot
	err = client.RestoreSnapshot(ctx, orgID, repoCfg, snapshot.ID)
	if err != nil {
		t.Fatalf("RestoreSnapshot failed: %v", err)
	}

	// 4. Verify file content reverted back to initial commit
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read file after restore: %v", err)
	}

	if string(content) != "# Initial Commit\n" {
		t.Fatalf("Expected content '# Initial Commit\\n', got '%s'", string(content))
	}

	// 5. List snapshots
	snapshots, err := client.ListSnapshots(ctx, orgID, repoCfg)
	if err != nil {
		t.Fatalf("ListSnapshots failed: %v", err)
	}

	if len(snapshots) == 0 {
		t.Fatalf("Expected at least 1 snapshot in list")
	}
}
