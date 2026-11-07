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
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

func TestFileSystemArtifactService_Lifecycle(t *testing.T) {
	tempDir := t.TempDir()
	service := NewFileSystemArtifactService(tempDir)
	ctx := context.Background()

	orgID := "org-100"
	userID := "user-200"
	workspaceID := "ws-300"

	// 1. Write nested artifacts
	file1Path := "docs/spec/architecture.md"
	file1Data := []byte("# Architecture Spec\nDetails here.")
	if err := service.WriteArtifact(ctx, orgID, userID, workspaceID, file1Path, file1Data); err != nil {
		t.Fatalf("WriteArtifact file1 failed: %v", err)
	}

	file2Path := "docs/plans/roadmap.md"
	file2Data := []byte("# Roadmap\nMilestones.")
	if err := service.WriteArtifact(ctx, orgID, userID, workspaceID, file2Path, file2Data); err != nil {
		t.Fatalf("WriteArtifact file2 failed: %v", err)
	}

	file3Path := "readme.txt"
	file3Data := []byte("Root file")
	if err := service.WriteArtifact(ctx, orgID, userID, workspaceID, file3Path, file3Data); err != nil {
		t.Fatalf("WriteArtifact file3 failed: %v", err)
	}

	// 2. GetArtifact (Success & Failure)
	gotData, err := service.GetArtifact(ctx, orgID, userID, workspaceID, file1Path)
	if err != nil {
		t.Fatalf("GetArtifact failed: %v", err)
	}
	if string(gotData) != string(file1Data) {
		t.Errorf("GetArtifact content mismatch: expected %q, got %q", string(file1Data), string(gotData))
	}

	_, err = service.GetArtifact(ctx, orgID, userID, workspaceID, "nonexistent/file.txt")
	if err == nil {
		t.Errorf("expected error reading nonexistent artifact, got nil")
	}

	// 3. ListArtifacts
	// List with prefix "docs"
	docsList, err := service.ListArtifacts(ctx, orgID, userID, workspaceID, "docs")
	if err != nil {
		t.Fatalf("ListArtifacts 'docs' failed: %v", err)
	}
	if len(docsList) != 2 {
		t.Errorf("expected 2 files under 'docs', got %d: %v", len(docsList), docsList)
	}

	// List with root prefix ""
	allList, err := service.ListArtifacts(ctx, orgID, userID, workspaceID, "")
	if err != nil {
		t.Fatalf("ListArtifacts '' failed: %v", err)
	}
	if len(allList) != 3 {
		t.Errorf("expected 3 files total, got %d: %v", len(allList), allList)
	}

	// List with nonexistent prefix (should return empty slice, not error)
	emptyList, err := service.ListArtifacts(ctx, orgID, userID, workspaceID, "nonexistent-dir")
	if err != nil {
		t.Fatalf("ListArtifacts nonexistent prefix failed: %v", err)
	}
	if len(emptyList) != 0 {
		t.Errorf("expected empty list for nonexistent dir, got %v", emptyList)
	}

	// 3b. ListTopLevelFiles (should only return root files, not nested subdirectories)
	topFiles, err := service.ListTopLevelFiles(ctx, orgID, userID, workspaceID)
	if err != nil {
		t.Fatalf("ListTopLevelFiles failed: %v", err)
	}
	if len(topFiles) != 1 || topFiles[0] != "readme.txt" {
		t.Errorf("expected 1 top-level file ['readme.txt'], got %v", topFiles)
	}

	// ListTopLevelFiles on nonexistent workspace returns nil, nil
	nonexistentTopFiles, err := service.ListTopLevelFiles(ctx, "nonexistent", "nonexistent", "nonexistent")
	if err != nil {
		t.Fatalf("ListTopLevelFiles nonexistent workspace failed: %v", err)
	}
	if len(nonexistentTopFiles) != 0 {
		t.Errorf("expected 0 top-level files for nonexistent workspace, got %v", nonexistentTopFiles)
	}

	// 4. DeleteArtifact (Success & Failure)
	if err := service.DeleteArtifact(ctx, orgID, userID, workspaceID, file3Path); err != nil {
		t.Fatalf("DeleteArtifact failed: %v", err)
	}

	// Verify deleted
	_, err = service.GetArtifact(ctx, orgID, userID, workspaceID, file3Path)
	if err == nil {
		t.Errorf("expected error getting deleted artifact, got nil")
	}

	// Delete nonexistent file returns error
	err = service.DeleteArtifact(ctx, orgID, userID, workspaceID, file3Path)
	if err == nil {
		t.Errorf("expected error deleting already deleted artifact, got nil")
	}
}

func TestFileSystemArtifactService_NFSQuota_Delete(t *testing.T) {
	tempDir := t.TempDir()
	mockStore := &mockNFSStoreForService{
		quota: make(map[string]bool),
	}
	service := NewFileSystemArtifactService(tempDir).WithNFSStore(mockStore)
	ctx := context.Background()

	orgID := "org-test"
	userID := "user-test"
	workspaceID := "ws-test"
	filePath := "test.md"

	// Create file first
	if err := service.WriteArtifact(ctx, orgID, userID, workspaceID, filePath, []byte("content")); err != nil {
		t.Fatalf("WriteArtifact failed: %v", err)
	}

	// Block write/delete via quota
	workspaceRoot := filepath.Join(tempDir, orgID, userID, workspaceID)
	mockStore.quota[workspaceRoot] = true

	// Delete should fail with ErrNFSQuotaExceeded
	err := service.DeleteArtifact(ctx, orgID, userID, workspaceID, filePath)
	if err != ErrNFSQuotaExceeded {
		t.Errorf("expected ErrNFSQuotaExceeded on DeleteArtifact, got %v", err)
	}

	// Unblock quota
	mockStore.quota[workspaceRoot] = false
	if err := service.DeleteArtifact(ctx, orgID, userID, workspaceID, filePath); err != nil {
		t.Errorf("expected DeleteArtifact to succeed when quota is restored, got %v", err)
	}
}

func TestGcsArtifactService_Methods(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/b/") && strings.HasSuffix(r.URL.Path, "/o") {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"items": []}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	client, err := storage.NewClient(context.Background(),
		option.WithoutAuthentication(),
		option.WithEndpoint(ts.URL),
	)
	if err != nil {
		t.Fatalf("failed to create storage client: %v", err)
	}
	defer client.Close()

	svc := NewGcsArtifactService(client, "test-bucket")
	ctx := context.Background()

	// 1. GetArtifact returns error on missing object
	_, err = svc.GetArtifact(ctx, "org1", "user1", "ws1", "missing.md")
	if err == nil {
		t.Errorf("expected error from GetArtifact on missing object, got nil")
	}

	// 2. DeleteArtifact returns error on missing object
	err = svc.DeleteArtifact(ctx, "org1", "user1", "ws1", "missing.md")
	if err == nil {
		t.Errorf("expected error from DeleteArtifact on missing object, got nil")
	}

	// 3. ListArtifacts returns empty list
	list, err := svc.ListArtifacts(ctx, "org1", "user1", "ws1", "specs")
	if err != nil {
		t.Fatalf("ListArtifacts failed: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected 0 items, got %d", len(list))
	}

	// 3b. ListTopLevelFiles returns empty list
	topList, err := svc.ListTopLevelFiles(ctx, "org1", "user1", "ws1")
	if err != nil {
		t.Fatalf("ListTopLevelFiles failed: %v", err)
	}
	if len(topList) != 0 {
		t.Errorf("expected 0 items, got %d", len(topList))
	}

	// 4. WriteArtifact with canceled context returns error
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	err = svc.WriteArtifact(canceledCtx, "org1", "user1", "ws1", "newfile.md", []byte("hello"))
	if err == nil {
		t.Errorf("expected error from WriteArtifact with canceled context, got nil")
	}

	// 5. SaveSessionArtifact with canceled context returns error
	errCancelCtx, cancel2 := context.WithCancel(context.Background())
	cancel2()
	_, _, _, err = svc.SaveSessionArtifact(errCancelCtx, "proj1", "sess1", "upload.png", strings.NewReader("image-data"))
	if err == nil {
		t.Errorf("expected error from SaveSessionArtifact with canceled context, got nil")
	}
}

func TestFileSystemArtifactService_SaveSessionArtifact(t *testing.T) {
	tempDir := t.TempDir()
	service := NewFileSystemArtifactService(tempDir)
	ctx := context.Background()

	projectID := "proj-123"
	sessionID := "sess-456"
	filename := "notes.md"
	content := "# My Session Notes\nSome content."

	storageType, storagePath, bytesWritten, err := service.SaveSessionArtifact(ctx, projectID, sessionID, filename, strings.NewReader(content))
	if err != nil {
		t.Fatalf("SaveSessionArtifact failed: %v", err)
	}

	if storageType != "local_file" {
		t.Errorf("expected storageType 'local_file', got %q", storageType)
	}

	expectedPath := filepath.Join(tempDir, projectID, sessionID, filename)
	if storagePath != expectedPath {
		t.Errorf("expected storagePath %q, got %q", expectedPath, storagePath)
	}

	if bytesWritten != int64(len(content)) {
		t.Errorf("expected bytesWritten %d, got %d", len(content), bytesWritten)
	}

	// Verify file exists on disk and content matches
	readData, err := os.ReadFile(storagePath)
	if err != nil {
		t.Fatalf("failed to read saved file: %v", err)
	}
	if string(readData) != content {
		t.Errorf("file content mismatch: expected %q, got %q", content, string(readData))
	}
}

func TestFileSystemArtifactService_ReadSessionArtifact(t *testing.T) {
	tempDir := t.TempDir()
	service := NewFileSystemArtifactService(tempDir)
	ctx := context.Background()

	projectID := "proj-123"
	sessionID := "sess-456"
	filename := "schema.sql"
	content := "CREATE TABLE users (id TEXT PRIMARY KEY);"

	storageType, storagePath, _, err := service.SaveSessionArtifact(ctx, projectID, sessionID, filename, strings.NewReader(content))
	if err != nil {
		t.Fatalf("SaveSessionArtifact failed: %v", err)
	}

	// 1. Success reading saved artifact
	data, err := service.ReadSessionArtifact(ctx, storageType, storagePath)
	if err != nil {
		t.Fatalf("ReadSessionArtifact failed: %v", err)
	}
	if string(data) != content {
		t.Errorf("ReadSessionArtifact content mismatch: expected %q, got %q", content, string(data))
	}

	// 2. Error on empty storage path
	_, err = service.ReadSessionArtifact(ctx, storageType, "")
	if err == nil {
		t.Errorf("expected error on empty storage path, got nil")
	}

	// 3. Error on non-existent storage path
	_, err = service.ReadSessionArtifact(ctx, storageType, filepath.Join(tempDir, "missing.txt"))
	if err == nil {
		t.Errorf("expected error on nonexistent file, got nil")
	}

	// 4. Error on unsupported storage type
	_, err = service.ReadSessionArtifact(ctx, "unsupported_storage", storagePath)
	if err == nil {
		t.Errorf("expected error on unsupported storage type, got nil")
	}

	// 5. Error on path traversal attempt outside base path
	_, err = service.ReadSessionArtifact(ctx, storageType, filepath.Join(tempDir, "..", "..", "etc", "passwd"))
	if err == nil {
		t.Errorf("expected error on path traversal outside base path, got nil")
	}
}

func TestGcsArtifactService_ReadSessionArtifact(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "notes.md") || strings.Contains(r.RequestURI, "notes.md") {
			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = w.Write([]byte("# Notes Content"))
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	client, err := storage.NewClient(context.Background(),
		option.WithoutAuthentication(),
		option.WithEndpoint(ts.URL),
	)
	if err != nil {
		t.Fatalf("failed to create storage client: %v", err)
	}
	defer client.Close()

	svc := NewGcsArtifactService(client, "test-bucket")
	ctx := context.Background()

	// 1. Success reading with gs:// URI
	data, err := svc.ReadSessionArtifact(ctx, "gcs", "gs://test-bucket/projects/p1/sessions/s1/notes.md")
	if err != nil {
		t.Fatalf("ReadSessionArtifact with gs:// URI failed: %v", err)
	}
	if string(data) != "# Notes Content" {
		t.Errorf("expected '# Notes Content', got %q", string(data))
	}

	// 2. Success reading with relative object path
	data, err = svc.ReadSessionArtifact(ctx, "gcs", "projects/p1/sessions/s1/notes.md")
	if err != nil {
		t.Fatalf("ReadSessionArtifact with relative object path failed: %v", err)
	}
	if string(data) != "# Notes Content" {
		t.Errorf("expected '# Notes Content', got %q", string(data))
	}

	// 3. Error on empty storage path
	_, err = svc.ReadSessionArtifact(ctx, "gcs", "")
	if err == nil {
		t.Errorf("expected error on empty storage path, got nil")
	}

	// 4. Error on non-existent object
	_, err = svc.ReadSessionArtifact(ctx, "gcs", "gs://test-bucket/missing.png")
	if err == nil {
		t.Errorf("expected error on non-existent object, got nil")
	}
}


