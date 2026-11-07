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

package jobs_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"google.com/apf/api/core"
	"google.com/apf/api/db"
	"google.com/apf/api/jobs"
)

type mockNFSStore struct {
	folders map[string]*core.NFSFolderMetadata
}

func newMockNFSStore() *mockNFSStore {
	return &mockNFSStore{
		folders: make(map[string]*core.NFSFolderMetadata),
	}
}

func (m *mockNFSStore) GetFolderMetadata(ctx context.Context, folderID string) (*core.NFSFolderMetadata, error) {
	if meta, exists := m.folders[folderID]; exists {
		return meta, nil
	}
	return nil, db.ErrNotFound
}

func (m *mockNFSStore) GetFolderMetadataByPath(ctx context.Context, workspacePath string) (*core.NFSFolderMetadata, error) {
	for _, f := range m.folders {
		if f.WorkspacePath == workspacePath {
			return f, nil
		}
	}
	return nil, db.ErrNotFound
}

func (m *mockNFSStore) SaveFolderMetadata(ctx context.Context, meta *core.NFSFolderMetadata) error {
	m.folders[meta.FolderID] = meta
	return nil
}

func (m *mockNFSStore) UpdateFolderMetadata(ctx context.Context, folderID string, updates core.NFSFolderMetadataUpdate) (*core.NFSFolderMetadata, error) {
	meta, exists := m.folders[folderID]
	if !exists {
		return nil, db.ErrNotFound
	}
	if updates.QuotaSizeBytes != nil {
		meta.QuotaSizeBytes = *updates.QuotaSizeBytes
	}
	if updates.RetentionPolicy != nil {
		meta.RetentionPolicy = *updates.RetentionPolicy
	}
	return meta, nil
}

func (m *mockNFSStore) ListFolderMetadata(ctx context.Context, filter core.NFSFolderFilter) ([]core.NFSFolderMetadata, error) {
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

func (m *mockNFSStore) UpsertFolderAuditStats(ctx context.Context, folderID string, usedBytes int64, fileCount int64) error {
	meta, exists := m.folders[folderID]
	if !exists {
		meta = &core.NFSFolderMetadata{FolderID: folderID}
		m.folders[folderID] = meta
	}
	meta.UsedSizeBytes = usedBytes
	meta.FileCount = fileCount
	return nil
}

func (m *mockNFSStore) DeleteFolderMetadata(ctx context.Context, folderID string) error {
	delete(m.folders, folderID)
	return nil
}

func (m *mockNFSStore) GetNFSVolumeStats(ctx context.Context, mountPath string) (*core.NFSVolumeStats, error) {
	return &core.NFSVolumeStats{
		MountPath:      mountPath,
		TotalBytes:     1000000,
		AvailableBytes: 500000,
		UsedBytes:      500000,
		IsHealthy:      true,
	}, nil
}

func (m *mockNFSStore) CheckFolderWritePermitted(ctx context.Context, workspacePath string) error {
	meta, err := m.GetFolderMetadataByPath(ctx, workspacePath)
	if err != nil {
		return nil
	}
	if meta.QuotaSizeBytes > 0 && meta.UsedSizeBytes >= meta.QuotaSizeBytes {
		return core.ErrNFSQuotaExceeded
	}
	return nil
}

func TestNFSAuditJob_Registration(t *testing.T) {
	job, ok := jobs.Get("nfs-audit")
	if !ok {
		t.Fatalf("expected nfs-audit to be registered in DefaultRegistry")
	}
	if job.Name() != "nfs-audit" {
		t.Errorf("expected job name 'nfs-audit', got %q", job.Name())
	}
	if job.Description() == "" {
		t.Errorf("expected non-empty job description")
	}
}

func TestNFSAuditJob_Run_WithMockStore(t *testing.T) {
	tempMount := t.TempDir()

	// Create workspace directory structure: tempMount/org-alpha/user-1/proj-1
	wsPath := filepath.Join(tempMount, "org-alpha", "user-1", "proj-1")
	if err := os.MkdirAll(wsPath, 0755); err != nil {
		t.Fatalf("failed to create dummy workspace dir: %v", err)
	}

	// Write dummy files: file1 (15 bytes), file2 (25 bytes) -> 40 bytes total
	if err := os.WriteFile(filepath.Join(wsPath, "file1.txt"), []byte("0123456789abcde"), 0644); err != nil {
		t.Fatalf("failed to write file1: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wsPath, "file2.txt"), []byte("01234567890123456789abcde"), 0644); err != nil {
		t.Fatalf("failed to write file2: %v", err)
	}

	store := newMockNFSStore()
	job := &jobs.NFSAuditJob{
		NFSStore:  store,
		MountPath: tempMount,
	}

	ctx := context.Background()
	cfg := &core.Configuration{
		WorkspaceMountPath: tempMount,
	}

	if err := job.Run(ctx, cfg); err != nil {
		t.Fatalf("job.Run failed: %v", err)
	}

	// Verify metadata in store
	folderID := "org-org-alpha"
	meta, err := store.GetFolderMetadata(ctx, folderID)
	if err != nil {
		t.Fatalf("failed to get folder metadata for %s: %v", folderID, err)
	}

	if meta.UsedSizeBytes != 40 {
		t.Errorf("expected UsedSizeBytes 40, got %d", meta.UsedSizeBytes)
	}
	if meta.FileCount != 2 {
		t.Errorf("expected FileCount 2, got %d", meta.FileCount)
	}
	if meta.OrganizationID != "org-alpha" {
		t.Errorf("expected OrganizationID 'org-alpha', got %q", meta.OrganizationID)
	}

	// Run audit a second time to test upsert branch
	if err := job.Run(ctx, cfg); err != nil {
		t.Fatalf("second job.Run failed: %v", err)
	}
	meta2, _ := store.GetFolderMetadata(ctx, folderID)
	if meta2.UsedSizeBytes != 40 {
		t.Errorf("expected UsedSizeBytes 40 on second run, got %d", meta2.UsedSizeBytes)
	}
}

func TestNFSAuditJob_Run_MissingStoreAndConfig(t *testing.T) {
	job := &jobs.NFSAuditJob{}
	err := job.Run(context.Background(), nil)
	if err == nil {
		t.Fatalf("expected error when running with nil store and empty config, got nil")
	}
}

func TestScanWorkspaceDirectory_EdgeCases(t *testing.T) {
	tempDir := t.TempDir()

	// Non-existent directory
	bytes, count, err := jobs.ScanWorkspaceDirectory(filepath.Join(tempDir, "non-existent"))
	if err != nil {
		t.Fatalf("expected nil error on non-existent dir, got %v", err)
	}
	if bytes != 0 || count != 0 {
		t.Errorf("expected 0 bytes and 0 files, got %d bytes, %d files", bytes, count)
	}

	// Directory with valid file and broken symlink
	subDir := filepath.Join(tempDir, "sub")
	os.MkdirAll(subDir, 0755)
	os.WriteFile(filepath.Join(subDir, "regular.txt"), []byte("hello"), 0644) // 5 bytes

	brokenSymlink := filepath.Join(subDir, "broken_link")
	_ = os.Symlink(filepath.Join(tempDir, "target_does_not_exist"), brokenSymlink)

	bytes, count, err = jobs.ScanWorkspaceDirectory(subDir)
	if err != nil {
		t.Fatalf("expected scan to succeed despite broken symlink, got %v", err)
	}
	if bytes != 5 || count != 1 {
		t.Errorf("expected 5 bytes and 1 file, got %d bytes, %d files", bytes, count)
	}
}

func TestDiscoverWorkspaceFolders_EdgeCases(t *testing.T) {
	// 1. Missing mount path
	results, err := jobs.DiscoverWorkspaceFolders("/path/does/not/exist/at/all")
	if err != nil {
		t.Fatalf("expected nil error on missing mount path, got %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected empty results on missing mount path, got %d", len(results))
	}

	// 2. Organizations subdirectory handling and hidden file skipping
	tempMount := t.TempDir()
	orgsDir := filepath.Join(tempMount, "organizations")
	os.MkdirAll(filepath.Join(orgsDir, ".hidden-dir"), 0755)
	os.MkdirAll(filepath.Join(orgsDir, "valid-org", "user1", "proj1"), 0755)
	os.WriteFile(filepath.Join(orgsDir, "valid-org", "user1", "proj1", "data.bin"), []byte("12345678"), 0644)

	results, err = jobs.DiscoverWorkspaceFolders(tempMount)
	if err != nil {
		t.Fatalf("DiscoverWorkspaceFolders failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 organization discovered, got %d", len(results))
	}
	if results[0].OrganizationID != "valid-org" {
		t.Errorf("expected OrganizationID 'valid-org', got %q", results[0].OrganizationID)
	}
	if results[0].TotalBytes != 8 || results[0].FileCount != 1 {
		t.Errorf("expected 8 bytes and 1 file, got %d bytes, %d files", results[0].TotalBytes, results[0].FileCount)
	}
}

func TestRunNFSAuditScan_LegacyProjectCleanup(t *testing.T) {
	tempMount := t.TempDir()
	store := newMockNFSStore()

	// Seed legacy project-level folder record
	legacyRecord := &core.NFSFolderMetadata{
		FolderID:       "folder-legacy-proj-1",
		OrganizationID: "org-legacy",
		ProjectID:      "proj-legacy-1",
		WorkspacePath:  "/old/path",
	}
	_ = store.SaveFolderMetadata(context.Background(), legacyRecord)

	cfg := &core.Configuration{
		WorkspaceMountPath: tempMount,
	}

	count, err := jobs.RunNFSAuditScan(context.Background(), cfg, store)
	if err != nil {
		t.Fatalf("RunNFSAuditScan failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 audited folders in empty mount dir, got %d", count)
	}

	// Verify legacy record was pruned
	_, err = store.GetFolderMetadata(context.Background(), "folder-legacy-proj-1")
	if err == nil {
		t.Errorf("expected legacy folder record to be deleted, but still exists")
	}
}

func TestNFSAuditJob_DryRun(t *testing.T) {
	tempMount := t.TempDir()
	wsPath := filepath.Join(tempMount, "org-beta", "user-1", "proj-1")
	_ = os.MkdirAll(wsPath, 0755)
	_ = os.WriteFile(filepath.Join(wsPath, "file.txt"), []byte("sample-data"), 0644)

	store := newMockNFSStore()
	job := &jobs.NFSAuditJob{
		NFSStore:  store,
		MountPath: tempMount,
	}

	dryCtx := jobs.ContextWithDryRun(context.Background(), true)
	cfg := &core.Configuration{WorkspaceMountPath: tempMount}

	if err := job.Run(dryCtx, cfg); err != nil {
		t.Fatalf("dry-run job.Run failed: %v", err)
	}

	// Verify store was NOT written in dry-run mode
	_, err := store.GetFolderMetadata(context.Background(), "org-org-beta")
	if err != db.ErrNotFound {
		t.Errorf("expected store to not contain folder metadata in dry-run mode, got %v", err)
	}

	// Test struct-level DryRun field
	jobStruct := &jobs.NFSAuditJob{
		DryRun:    true,
		NFSStore:  store,
		MountPath: tempMount,
	}
	if err := jobStruct.Run(context.Background(), cfg); err != nil {
		t.Fatalf("struct dry-run job.Run failed: %v", err)
	}
	_, err = store.GetFolderMetadata(context.Background(), "org-org-beta")
	if err != db.ErrNotFound {
		t.Errorf("expected store to not contain folder metadata in struct dry-run mode, got %v", err)
	}
}

func TestDiscoverWorkspaceFolders_SymlinkOrgDir(t *testing.T) {
	tempMount := t.TempDir()
	realOrgDir := filepath.Join(tempMount, "real-org")
	projDir := filepath.Join(realOrgDir, "user1", "proj1")
	_ = os.MkdirAll(projDir, 0755)
	_ = os.WriteFile(filepath.Join(projDir, "data.bin"), []byte("1234567890"), 0644)

	orgsDir := filepath.Join(tempMount, "organizations")
	_ = os.MkdirAll(orgsDir, 0755)
	_ = os.Symlink(realOrgDir, filepath.Join(orgsDir, "symlinked-org"))

	results, err := jobs.DiscoverWorkspaceFolders(tempMount)
	if err != nil {
		t.Fatalf("DiscoverWorkspaceFolders failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 organization discovered, got %d", len(results))
	}
	if results[0].OrganizationID != "symlinked-org" {
		t.Errorf("expected OrganizationID 'symlinked-org', got %q", results[0].OrganizationID)
	}
	if results[0].TotalBytes != 10 || results[0].FileCount != 1 {
		t.Errorf("expected 10 bytes and 1 file, got %d bytes, %d files", results[0].TotalBytes, results[0].FileCount)
	}
}

func TestScanWorkspaceDirectory_SymlinkCycle(t *testing.T) {
	tempDir := t.TempDir()
	file := filepath.Join(tempDir, "file.txt")
	_ = os.WriteFile(file, []byte("cycle-test"), 0644) // 10 bytes

	// Create a circular directory symlink pointing back to tempDir
	cycleLink := filepath.Join(tempDir, "cycle")
	_ = os.Symlink(tempDir, cycleLink)

	bytes, count, err := jobs.ScanWorkspaceDirectory(tempDir)
	if err != nil {
		t.Fatalf("expected ScanWorkspaceDirectory to handle cycle without error, got %v", err)
	}
	if bytes != 10 || count != 1 {
		t.Errorf("expected 10 bytes and 1 file despite circular symlink, got %d bytes, %d files", bytes, count)
	}
}
