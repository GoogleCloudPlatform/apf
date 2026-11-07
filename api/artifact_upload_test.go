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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"google.com/apf/api/core"
	"google.com/apf/api/db"
)

type mockUploadArtifactStore struct {
	artifacts []db.Artifact
}

func (m *mockUploadArtifactStore) ListArtifactsForProject(ctx context.Context, organizationID string, projectId string) ([]db.Artifact, error) {
	return m.artifacts, nil
}

func (m *mockUploadArtifactStore) ListSessionArtifacts(ctx context.Context, organizationID string, projectId string, sessionId string) ([]db.Artifact, error) {
	var result []db.Artifact
	for _, a := range m.artifacts {
		if a.ProjectId == projectId && a.SessionId == sessionId {
			result = append(result, a)
		}
	}
	return result, nil
}

func (m *mockUploadArtifactStore) UpdateArtifact(ctx context.Context, organizationID string, a *db.Artifact) error {
	return nil
}

func (m *mockUploadArtifactStore) CreateArtifact(ctx context.Context, organizationID string, a *db.Artifact) error {
	m.artifacts = append(m.artifacts, *a)
	return nil
}

func (m *mockUploadArtifactStore) DeleteArtifact(ctx context.Context, organizationID string, artifactId string) error {
	return nil
}

func (m *mockUploadArtifactStore) SyncArtifacts(ctx context.Context, organizationID string, projectId string, discovered []db.Artifact) (int, int, error) {
	return 0, 0, nil
}

type mockFailingArtifactService struct{}

func (m *mockFailingArtifactService) GetArtifact(ctx context.Context, organizationID string, userId string, workspaceID string, path string) ([]byte, error) {
	return nil, nil
}

func (m *mockFailingArtifactService) WriteArtifact(ctx context.Context, organizationID string, userId string, workspaceID string, path string, data []byte) error {
	return nil
}

func (m *mockFailingArtifactService) DeleteArtifact(ctx context.Context, organizationID string, userId string, workspaceID string, path string) error {
	return nil
}

func (m *mockFailingArtifactService) ListArtifacts(ctx context.Context, organizationID string, userId string, workspaceID string, prefix string) ([]string, error) {
	return nil, nil
}

func (m *mockFailingArtifactService) ListTopLevelFiles(ctx context.Context, organizationID string, userId string, workspaceID string) ([]string, error) {
	return nil, nil
}

func (m *mockFailingArtifactService) SaveSessionArtifact(ctx context.Context, projectId, sessionId, filename string, r io.Reader) (string, string, int64, error) {
	return "", "", 0, errors.New("gcs connection error: permission denied")
}

func (m *mockFailingArtifactService) ReadSessionArtifact(ctx context.Context, storageType string, storagePath string) ([]byte, error) {
	return nil, errors.New("gcs connection error: permission denied")
}

type mockGcsArtifactService struct {
	bucketName string
}

func (m *mockGcsArtifactService) GetArtifact(ctx context.Context, organizationID string, userId string, workspaceID string, path string) ([]byte, error) {
	return nil, nil
}

func (m *mockGcsArtifactService) WriteArtifact(ctx context.Context, organizationID string, userId string, workspaceID string, path string, data []byte) error {
	return nil
}

func (m *mockGcsArtifactService) DeleteArtifact(ctx context.Context, organizationID string, userId string, workspaceID string, path string) error {
	return nil
}

func (m *mockGcsArtifactService) ListArtifacts(ctx context.Context, organizationID string, userId string, workspaceID string, prefix string) ([]string, error) {
	return nil, nil
}

func (m *mockGcsArtifactService) ListTopLevelFiles(ctx context.Context, organizationID string, userId string, workspaceID string) ([]string, error) {
	return nil, nil
}

func (m *mockGcsArtifactService) SaveSessionArtifact(ctx context.Context, projectId, sessionId, filename string, r io.Reader) (string, string, int64, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return "", "", 0, err
	}
	gcsPath := fmt.Sprintf("gs://%s/projects/%s/sessions/%s/%s", m.bucketName, projectId, sessionId, filename)
	return "gcs", gcsPath, int64(len(data)), nil
}

func (m *mockGcsArtifactService) ReadSessionArtifact(ctx context.Context, storageType string, storagePath string) ([]byte, error) {
	return []byte("mock gcs session artifact content"), nil
}

func TestUploadSessionArtifactHandler_LocalSuccess(t *testing.T) {
	tempDir := t.TempDir()
	artSvc := core.NewFileSystemArtifactService(tempDir)
	artStore := &mockUploadArtifactStore{}

	handler := NewSessionHandler(nil, nil, nil).
		WithArtifactService(artSvc).
		WithArtifactStore(artStore)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "test-notes.md")
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}
	fileContent := "# Hello World\nThis is a test session artifact."
	if _, err := part.Write([]byte(fileContent)); err != nil {
		t.Fatalf("Failed to write file content: %v", err)
	}
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/proj-123/sessions/sess-456/artifacts/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.SetPathValue("id", "proj-123")
	req.SetPathValue("sessionId", "sess-456")

	ctx := core.WithOrganizationID(req.Context(), "org-test")
	ctx = context.WithValue(ctx, core.UserClaimsKey, &core.UserClaims{Subject: "user-1"})
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.uploadSessionArtifactHandler(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("Expected status 201 Created, got %d: %s", rec.Code, rec.Body.String())
	}

	var createdArt db.Artifact
	if err := json.Unmarshal(rec.Body.Bytes(), &createdArt); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if createdArt.ID == "" {
		t.Errorf("Expected artifact ID to be populated, got empty string")
	}
	if createdArt.ProjectId != "proj-123" {
		t.Errorf("Expected projectId 'proj-123', got %q", createdArt.ProjectId)
	}
	if createdArt.SessionId != "sess-456" {
		t.Errorf("Expected sessionId 'sess-456', got %q", createdArt.SessionId)
	}
	if createdArt.Name != "test-notes.md" {
		t.Errorf("Expected name 'test-notes.md', got %q", createdArt.Name)
	}
	if createdArt.StorageType != "local_file" {
		t.Errorf("Expected storageType 'local_file', got %q", createdArt.StorageType)
	}
	if createdArt.Size != int64(len(fileContent)) {
		t.Errorf("Expected size %d, got %d", len(fileContent), createdArt.Size)
	}
	if createdArt.CreatedAt.IsZero() {
		t.Errorf("Expected createdAt timestamp to be set")
	}

	// Verify file actually written to local disk
	savedContent, err := os.ReadFile(createdArt.StoragePath)
	if err != nil {
		t.Fatalf("Failed to read persisted file at %s: %v", createdArt.StoragePath, err)
	}
	if string(savedContent) != fileContent {
		t.Errorf("Persisted file content mismatch: expected %q, got %q", fileContent, string(savedContent))
	}

	// Verify metadata in mock store
	if len(artStore.artifacts) != 1 {
		t.Errorf("Expected 1 artifact in store, got %d", len(artStore.artifacts))
	}
}

func TestUploadSessionArtifactHandler_MissingFile(t *testing.T) {
	handler := NewSessionHandler(nil, nil, nil)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("otherField", "value")
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/proj-123/sessions/sess-456/artifacts/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.SetPathValue("id", "proj-123")
	req.SetPathValue("sessionId", "sess-456")

	ctx := core.WithOrganizationID(req.Context(), "org-test")
	ctx = context.WithValue(ctx, core.UserClaimsKey, &core.UserClaims{Subject: "user-1"})
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.uploadSessionArtifactHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 Bad Request for missing file, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "File is required") {
		t.Errorf("Expected error message 'File is required', got %s", rec.Body.String())
	}
}

func TestUploadSessionArtifactHandler_ExceedsSizeLimit(t *testing.T) {
	tempDir := t.TempDir()
	artSvc := core.NewFileSystemArtifactService(tempDir)
	handler := NewSessionHandler(nil, nil, nil).WithArtifactService(artSvc)

	// Create oversized payload (> 10MB)
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "large-video.mp4")
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}
	oversized := make([]byte, 11*1024*1024)
	part.Write(oversized)
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/proj-123/sessions/sess-456/artifacts/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.SetPathValue("id", "proj-123")
	req.SetPathValue("sessionId", "sess-456")

	ctx := core.WithOrganizationID(req.Context(), "org-test")
	ctx = context.WithValue(ctx, core.UserClaimsKey, &core.UserClaims{Subject: "user-1"})
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.uploadSessionArtifactHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 Bad Request for oversized file, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "10MB") {
		t.Errorf("Expected size limit error message, got %s", rec.Body.String())
	}
}

func TestUploadSessionArtifactHandler_PathSanitization(t *testing.T) {
	tempDir := t.TempDir()
	artSvc := core.NewFileSystemArtifactService(tempDir)
	artStore := &mockUploadArtifactStore{}

	handler := NewSessionHandler(nil, nil, nil).
		WithArtifactService(artSvc).
		WithArtifactStore(artStore)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "../../../../etc/passwd")
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}
	part.Write([]byte("root:x:0:0:root"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p1/sessions/s1/artifacts/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.SetPathValue("id", "p1")
	req.SetPathValue("sessionId", "s1")

	ctx := core.WithOrganizationID(req.Context(), "org-test")
	ctx = context.WithValue(ctx, core.UserClaimsKey, &core.UserClaims{Subject: "user-1"})
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.uploadSessionArtifactHandler(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("Expected status 201 Created, got %d: %s", rec.Code, rec.Body.String())
	}

	var createdArt db.Artifact
	if err := json.Unmarshal(rec.Body.Bytes(), &createdArt); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if createdArt.Name != "passwd" {
		t.Errorf("Expected sanitized filename 'passwd', got %q", createdArt.Name)
	}
	if !strings.HasPrefix(createdArt.StoragePath, tempDir) {
		t.Errorf("StoragePath %q must remain within tempDir %q", createdArt.StoragePath, tempDir)
	}
	expectedFile := filepath.Join(tempDir, "p1", "s1", "passwd")
	if createdArt.StoragePath != expectedFile {
		t.Errorf("Expected %q, got %q", expectedFile, createdArt.StoragePath)
	}
}

func TestUploadSessionArtifactHandler_GcsSuccess(t *testing.T) {
	artSvc := &mockGcsArtifactService{bucketName: "apf-test-bucket"}
	artStore := &mockUploadArtifactStore{}

	handler := NewSessionHandler(nil, nil, nil).
		WithArtifactService(artSvc).
		WithArtifactStore(artStore)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "diagram.png")
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}
	imageBytes := []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")
	part.Write(imageBytes)
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p1/sessions/s1/artifacts/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.SetPathValue("id", "p1")
	req.SetPathValue("sessionId", "s1")

	ctx := core.WithOrganizationID(req.Context(), "org-test")
	ctx = context.WithValue(ctx, core.UserClaimsKey, &core.UserClaims{Subject: "user-1"})
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.uploadSessionArtifactHandler(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("Expected status 201 Created, got %d: %s", rec.Code, rec.Body.String())
	}

	var createdArt db.Artifact
	if err := json.Unmarshal(rec.Body.Bytes(), &createdArt); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if createdArt.StorageType != "gcs" {
		t.Errorf("Expected storageType 'gcs', got %q", createdArt.StorageType)
	}
	expectedURI := "gs://apf-test-bucket/projects/p1/sessions/s1/diagram.png"
	if createdArt.StoragePath != expectedURI {
		t.Errorf("Expected storagePath %q, got %q", expectedURI, createdArt.StoragePath)
	}
	if createdArt.Name != "diagram.png" {
		t.Errorf("Expected name 'diagram.png', got %q", createdArt.Name)
	}
}

func TestUploadSessionArtifactHandler_GcsError(t *testing.T) {
	artSvc := &mockFailingArtifactService{}
	artStore := &mockUploadArtifactStore{}

	handler := NewSessionHandler(nil, nil, nil).
		WithArtifactService(artSvc).
		WithArtifactStore(artStore)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "diagram.png")
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}
	part.Write([]byte("image content"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p1/sessions/s1/artifacts/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.SetPathValue("id", "p1")
	req.SetPathValue("sessionId", "s1")

	ctx := core.WithOrganizationID(req.Context(), "org-test")
	ctx = context.WithValue(ctx, core.UserClaimsKey, &core.UserClaims{Subject: "user-1"})
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.uploadSessionArtifactHandler(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500 Internal Server Error, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestUploadSessionArtifactHandler_EmptyFile(t *testing.T) {
	handler := NewSessionHandler(nil, nil, nil)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "empty.txt")
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}
	// Zero bytes written
	_ = part
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p1/sessions/s1/artifacts/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.SetPathValue("id", "p1")
	req.SetPathValue("sessionId", "s1")

	ctx := core.WithOrganizationID(req.Context(), "org-test")
	ctx = context.WithValue(ctx, core.UserClaimsKey, &core.UserClaims{Subject: "user-1"})
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.uploadSessionArtifactHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 Bad Request for empty file, got %d: %s", rec.Code, rec.Body.String())
	}
}

