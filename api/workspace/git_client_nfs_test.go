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

	"google.com/apf/api/core"
)

type mockNFSStoreForGit struct {
	quota map[string]bool
}

func (m *mockNFSStoreForGit) CheckFolderWritePermitted(ctx context.Context, workspacePath string) error {
	if m.quota[workspacePath] {
		return core.ErrNFSQuotaExceeded
	}
	return nil
}

func TestGitClient_NFSQuotaAndLockEnforcement(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "nfs-git-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mockStore := &mockNFSStoreForGit{
		quota: make(map[string]bool),
	}

	gitClient := NewGitClient(tempDir).WithNFSStore(mockStore)

	orgID := "org-1"
	repoCfg := &GitRepoConfig{
		UserId:       "user-1",
		ProjectId:    "proj-1",
		SpecLocation: "docs/spec/",
		BaseBranch:   "main",
		SpecBranch:   "specs/proj-1",
	}

	clonePath := filepath.Join(tempDir, orgID, repoCfg.UserId, repoCfg.ProjectId)
	ctx := context.Background()

	// 1. Quota Exceeded -> should fail with ErrNFSQuotaExceeded
	mockStore.quota[clonePath] = true

	err = gitClient.EnsureRepoInitialized(ctx, orgID, repoCfg)
	if err != core.ErrNFSQuotaExceeded {
		t.Errorf("Expected ErrNFSQuotaExceeded on EnsureRepoInitialized, got %v", err)
	}

	err = gitClient.LocalCommit(ctx, orgID, repoCfg, "docs/spec/test.md", "commit msg", "Author", "author@email.com")
	if err != core.ErrNFSQuotaExceeded {
		t.Errorf("Expected ErrNFSQuotaExceeded on LocalCommit, got %v", err)
	}

	err = gitClient.Pull(ctx, orgID, repoCfg)
	if err != core.ErrNFSQuotaExceeded {
		t.Errorf("Expected ErrNFSQuotaExceeded on Pull, got %v", err)
	}
}
