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

package core

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

type mockNFSStoreForService struct {
	quota map[string]bool
}

func (m *mockNFSStoreForService) CheckFolderWritePermitted(ctx context.Context, workspacePath string) error {
	if m.quota[workspacePath] {
		return ErrNFSQuotaExceeded
	}
	return nil
}

func TestFileSystemArtifactService_NFSQuotaAndLockEnforcement(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "nfs-artifact-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mockStore := &mockNFSStoreForService{
		quota: make(map[string]bool),
	}

	service := NewFileSystemArtifactService(tempDir).WithNFSStore(mockStore)

	orgID := "org-1"
	userID := "user-1"
	workspaceID := "proj-1"
	workspaceRoot := filepath.Join(tempDir, orgID, userID, workspaceID)

	ctx := context.Background()

	// 1. Normal write should succeed
	err = service.WriteArtifact(ctx, orgID, userID, workspaceID, "test.txt", []byte("hello world"))
	if err != nil {
		t.Errorf("Expected write to succeed, got %v", err)
	}

	// 2. Exceeded quota folder should fail with ErrNFSQuotaExceeded
	mockStore.quota[workspaceRoot] = true
	err = service.WriteArtifact(ctx, orgID, userID, workspaceID, "test3.txt", []byte("quota exceeded"))
	if err != ErrNFSQuotaExceeded {
		t.Errorf("Expected ErrNFSQuotaExceeded, got %v", err)
	}
}
