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

package db

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"google.com/apf/api/core"
)

func TestNFSVolumeStats_LocalDir(t *testing.T) {
	ctx := context.Background()
	store := &FirestoreNFSStore{}

	// Test with current directory
	pwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd failed: %v", err)
	}

	stats, err := store.GetNFSVolumeStats(ctx, pwd)
	if err != nil {
		t.Fatalf("GetNFSVolumeStats failed: %v", err)
	}

	if !stats.IsHealthy {
		t.Errorf("Expected IsHealthy = true for valid path %s", pwd)
	}
	if stats.TotalBytes <= 0 {
		t.Errorf("Expected TotalBytes > 0, got %d", stats.TotalBytes)
	}
}

func TestCheckFolderWritePermitted_NoStore(t *testing.T) {
	ctx := context.Background()
	store := &FirestoreNFSStore{}

	// When workspace path is empty
	err := store.CheckFolderWritePermitted(ctx, "")
	if err != nil {
		t.Errorf("Expected nil error for empty workspace path, got %v", err)
	}
}

func TestFirestoreNFSStore_CRUD(t *testing.T) {
	emulatorHost := os.Getenv("FIRESTORE_EMULATOR_HOST")
	if emulatorHost == "" {
		t.Skip("Skipping Firestore integration test: FIRESTORE_EMULATOR_HOST is not set")
	}

	ctx := context.Background()
	client, err := firestore.NewClient(ctx, "agenticproductfoundry-489717")
	if err != nil {
		t.Fatalf("Failed to create Firestore client: %v", err)
	}
	defer client.Close()

	store := NewFirestoreNFSStore(client)

	folderID := "test-folder-123"
	workspacePath := filepath.Join(os.TempDir(), "test-workspace-nfs-123")
	meta := &core.NFSFolderMetadata{
		FolderID:        folderID,
		OrganizationID:  "org-abc",
		ProjectID:       "proj-xyz",
		WorkspacePath:   workspacePath,
		QuotaSizeBytes:  1073741824, // 1 GB
		UsedSizeBytes:   52428800,   // 50 MB
		FileCount:       120,
		RetentionPolicy: "30d",
		LastAuditedAt:   time.Now(),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	// Clean up after test
	defer func() {
		_ = store.DeleteFolderMetadata(ctx, folderID)
	}()

	// 1. Save Folder Metadata
	err = store.SaveFolderMetadata(ctx, meta)
	if err != nil {
		t.Fatalf("SaveFolderMetadata failed: %v", err)
	}

	// 2. Get Folder Metadata by ID
	fetched, err := store.GetFolderMetadata(ctx, folderID)
	if err != nil {
		t.Fatalf("GetFolderMetadata failed: %v", err)
	}
	if fetched.OrganizationID != meta.OrganizationID {
		t.Errorf("Expected OrgID %s, got %s", meta.OrganizationID, fetched.OrganizationID)
	}
	if fetched.QuotaSizeBytes != meta.QuotaSizeBytes {
		t.Errorf("Expected Quota %d, got %d", meta.QuotaSizeBytes, fetched.QuotaSizeBytes)
	}

	// 3. Get Folder Metadata by Workspace Path
	byPath, err := store.GetFolderMetadataByPath(ctx, workspacePath)
	if err != nil {
		t.Fatalf("GetFolderMetadataByPath failed: %v", err)
	}
	if byPath.FolderID != folderID {
		t.Errorf("Expected FolderID %s, got %s", folderID, byPath.FolderID)
	}

	// 4. Check write permitted when below quota
	if err := store.CheckFolderWritePermitted(ctx, workspacePath); err != nil {
		t.Errorf("Expected write permitted, got %v", err)
	}

	// 5. Update to Exceeded Quota
	newQuota := int64(100)
	_, err = store.UpdateFolderMetadata(ctx, folderID, core.NFSFolderMetadataUpdate{
		QuotaSizeBytes: &newQuota,
	})
	if err != nil {
		t.Fatalf("UpdateFolderMetadata failed: %v", err)
	}
	if err := store.CheckFolderWritePermitted(ctx, workspacePath); err != core.ErrNFSQuotaExceeded {
		t.Errorf("Expected ErrNFSQuotaExceeded, got %v", err)
	}
}
