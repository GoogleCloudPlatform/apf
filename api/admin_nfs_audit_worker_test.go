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

package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"google.com/apf/api/core"
	"google.com/apf/api/db"
	"google.com/apf/api/jobs"
)

type mockNFSStoreForAudit struct {
	folders map[string]*core.NFSFolderMetadata
}

func newMockNFSStoreForAudit() *mockNFSStoreForAudit {
	return &mockNFSStoreForAudit{
		folders: make(map[string]*core.NFSFolderMetadata),
	}
}

func (m *mockNFSStoreForAudit) GetFolderMetadata(ctx context.Context, folderID string) (*core.NFSFolderMetadata, error) {
	if meta, exists := m.folders[folderID]; exists {
		return meta, nil
	}
	return nil, dbErrNotFound()
}

func (m *mockNFSStoreForAudit) GetFolderMetadataByPath(ctx context.Context, workspacePath string) (*core.NFSFolderMetadata, error) {
	for _, f := range m.folders {
		if f.WorkspacePath == workspacePath {
			return f, nil
		}
	}
	return nil, dbErrNotFound()
}

func (m *mockNFSStoreForAudit) SaveFolderMetadata(ctx context.Context, meta *core.NFSFolderMetadata) error {
	m.folders[meta.FolderID] = meta
	return nil
}

func (m *mockNFSStoreForAudit) UpdateFolderMetadata(ctx context.Context, folderID string, updates core.NFSFolderMetadataUpdate) (*core.NFSFolderMetadata, error) {
	meta, exists := m.folders[folderID]
	if !exists {
		return nil, dbErrNotFound()
	}
	if updates.QuotaSizeBytes != nil {
		meta.QuotaSizeBytes = *updates.QuotaSizeBytes
	}
	if updates.RetentionPolicy != nil {
		meta.RetentionPolicy = *updates.RetentionPolicy
	}
	return meta, nil
}

func (m *mockNFSStoreForAudit) ListFolderMetadata(ctx context.Context, filter core.NFSFolderFilter) ([]core.NFSFolderMetadata, error) {
	var list []core.NFSFolderMetadata
	for _, f := range m.folders {
		if filter.OrganizationID != "" && f.OrganizationID != filter.OrganizationID {
			continue
		}
		if filter.ProjectID != "" && f.ProjectID != filter.ProjectID {
			continue
		}
		list = append(list, *f)
	}
	return list, nil
}

func (m *mockNFSStoreForAudit) UpsertFolderAuditStats(ctx context.Context, folderID string, usedBytes int64, fileCount int64) error {
	meta, exists := m.folders[folderID]
	if !exists {
		meta = &core.NFSFolderMetadata{FolderID: folderID}
		m.folders[folderID] = meta
	}
	meta.UsedSizeBytes = usedBytes
	meta.FileCount = fileCount
	return nil
}

func (m *mockNFSStoreForAudit) DeleteFolderMetadata(ctx context.Context, folderID string) error {
	delete(m.folders, folderID)
	return nil
}

func (m *mockNFSStoreForAudit) GetNFSVolumeStats(ctx context.Context, mountPath string) (*core.NFSVolumeStats, error) {
	return &core.NFSVolumeStats{
		MountPath:      mountPath,
		TotalBytes:     1000000,
		AvailableBytes: 500000,
		UsedBytes:      500000,
		IsHealthy:      true,
	}, nil
}

func (m *mockNFSStoreForAudit) CheckFolderWritePermitted(ctx context.Context, workspacePath string) error {
	meta, err := m.GetFolderMetadataByPath(ctx, workspacePath)
	if err != nil {
		return nil
	}
	if meta.QuotaSizeBytes > 0 && meta.UsedSizeBytes >= meta.QuotaSizeBytes {
		return core.ErrNFSQuotaExceeded
	}
	return nil
}

func dbErrNotFound() error {
	return db.ErrNotFound
}

func TestDiscoverWorkspaceFoldersAndAudit(t *testing.T) {
	tempMount := t.TempDir()

	// Create dummy workspace structure: tempMount/org-test/user-test/proj-test
	wsPath := filepath.Join(tempMount, "org-test", "user-test", "proj-test")
	if err := os.MkdirAll(wsPath, 0755); err != nil {
		t.Fatalf("Failed to create dummy workspace dir: %v", err)
	}

	// Write dummy files: file1.txt (10 bytes), file2.txt (20 bytes)
	if err := os.WriteFile(filepath.Join(wsPath, "file1.txt"), []byte("0123456789"), 0644); err != nil {
		t.Fatalf("Failed to write file1: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wsPath, "file2.txt"), []byte("01234567890123456789"), 0644); err != nil {
		t.Fatalf("Failed to write file2: %v", err)
	}

	// 1. Test DiscoverWorkspaceFolders
	discovered, err := jobs.DiscoverWorkspaceFolders(tempMount)
	if err != nil {
		t.Fatalf("DiscoverWorkspaceFolders failed: %v", err)
	}

	if len(discovered) != 1 {
		t.Fatalf("Expected 1 discovered workspace folder, got %d", len(discovered))
	}

	if discovered[0].OrganizationID != "org-test" {
		t.Errorf("Expected org-test, got %s", discovered[0].OrganizationID)
	}
	if discovered[0].ProjectID != "All Projects" {
		t.Errorf("Expected All Projects, got %s", discovered[0].ProjectID)
	}
	if discovered[0].FileCount != 2 {
		t.Errorf("Expected 2 files, got %d", discovered[0].FileCount)
	}
	if discovered[0].TotalBytes != 30 {
		t.Errorf("Expected 30 total bytes, got %d", discovered[0].TotalBytes)
	}

	// 2. Test RunNFSAuditScan
	store := newMockNFSStoreForAudit()
	cfg := &core.Configuration{
		WorkspaceMountPath: tempMount,
	}

	ctx := context.Background()
	count, err := RunNFSAuditScan(ctx, cfg, store)
	if err != nil {
		t.Fatalf("RunNFSAuditScan failed: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 folder audited, got %d", count)
	}

	// Verify Firestore/NFSStore record updated at Organization level
	folderID := "org-org-test"
	meta, err := store.GetFolderMetadata(ctx, folderID)
	if err != nil {
		t.Fatalf("Failed to get folder metadata after audit: %v", err)
	}

	if meta.UsedSizeBytes != 30 {
		t.Errorf("Expected UsedSizeBytes=30 in store, got %d", meta.UsedSizeBytes)
	}
	if meta.FileCount != 2 {
		t.Errorf("Expected FileCount=2 in store, got %d", meta.FileCount)
	}
}
