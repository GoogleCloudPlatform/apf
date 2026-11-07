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
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"google.com/apf/api/agent"
	"google.com/apf/api/core"
	"google.com/apf/api/db"
)

type mockSaveArtifactStore struct {
	db.ArtifactStore
	artifacts []db.Artifact
}

func (m *mockSaveArtifactStore) ListArtifactsForProject(ctx context.Context, organizationID string, projectId string) ([]db.Artifact, error) {
	return m.artifacts, nil
}

func (m *mockSaveArtifactStore) CreateArtifact(ctx context.Context, organizationID string, a *db.Artifact) error {
	m.artifacts = append(m.artifacts, *a)
	return nil
}

func (m *mockSaveArtifactStore) UpdateArtifact(ctx context.Context, organizationID string, a *db.Artifact) error {
	for i, art := range m.artifacts {
		if art.ID == a.ID || art.Path == a.Path {
			m.artifacts[i] = *a
			return nil
		}
	}
	m.artifacts = append(m.artifacts, *a)
	return nil
}

type safeDummyUserProjectStore struct {
	db.UserProjectStore
}

func (s *safeDummyUserProjectStore) GetProjectAppSetting(ctx context.Context, organizationID string, userId, projectId, appId string) (*db.UserProject, error) {
	return &db.UserProject{UserID: userId, ProjectID: projectId, AppID: appId}, nil
}

func (s *safeDummyUserProjectStore) SaveProjectAppSetting(ctx context.Context, organizationID string, setting *db.UserProject) error {
	return nil
}

func TestSaveArtifactContentHandler_Success_RawString(t *testing.T) {
	tempMount := t.TempDir()
	os.Setenv("ARTIFACT_SERVICE_TYPE", "FS")
	os.Setenv("WORKSPACE_MOUNT_PATH", tempMount)
	defer os.Unsetenv("ARTIFACT_SERVICE_TYPE")
	defer os.Unsetenv("WORKSPACE_MOUNT_PATH")

	origArtifactStore := ArtifactStore
	origProjectStore := db.ProjectStoreInstance
	origMessageStore := agent.Store
	origUserProjectStore := UserProjectStore
	origHook := agent.OnFileWriteHook
	defer func() {
		ArtifactStore = origArtifactStore
		db.ProjectStoreInstance = origProjectStore
		agent.Store = origMessageStore
		UserProjectStore = origUserProjectStore
		agent.OnFileWriteHook = origHook
	}()

	targetArt := db.Artifact{
		ID:        "art123",
		ProjectId: "proj123",
		Title:     "prd.md",
		Path:      "docs/spec/prd.md",
		Phase:     "PLANNING",
		Status:    db.ArtifactStatusInProgress,
	}

	mStore := &mockSaveArtifactStore{
		artifacts: []db.Artifact{targetArt},
	}
	ArtifactStore = mStore

	updatedChan := make(chan *db.Project, 1)
	mProjectStore := &mockDeleteProjectStore{
		project: &db.Project{
			ID: "proj123",
		},
		updatedChan: updatedChan,
	}
	db.ProjectStoreInstance = mProjectStore
	agent.Store = &safeDummyMessageStore{}
	UserProjectStore = &safeDummyUserProjectStore{}

	var hookFilePath string
	agent.OnFileWriteHook = func(ctx context.Context, projectId string, organizationID string, phase string, filePath string) {
		hookFilePath = filePath
		_ = SyncOrRegisterArtifact(ctx, organizationID, projectId, phase, filePath)
		_ = recalculateGates(ctx, organizationID, projectId)
	}

	newContent := "# Updated Product Requirements\n\n- Feature A\n- Feature B\n"

	req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/proj123/artifacts/art123/content", strings.NewReader(newContent))
	req.SetPathValue("id", "proj123")
	req.SetPathValue("artifactId", "art123")

	claims := &core.UserClaims{Subject: "user@google.com"}
	req = withTestCtx(req, claims)

	w := httptest.NewRecorder()
	SaveArtifactContentHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}

	// Verify file was written to disk
	expectedFilePath := filepath.Join(tempMount, "test-organization", "user@google.com", "proj123", "docs/spec/prd.md")
	savedBytes, err := os.ReadFile(expectedFilePath)
	if err != nil {
		t.Fatalf("Failed to read saved file from %s: %v", expectedFilePath, err)
	}
	if string(savedBytes) != newContent {
		t.Errorf("Expected content %q, got %q", newContent, string(savedBytes))
	}

	if hookFilePath != "docs/spec/prd.md" {
		t.Errorf("Expected hookFilePath to be docs/spec/prd.md, got %q", hookFilePath)
	}

	// Verify gate calculation triggered
	select {
	case updatedProj := <-updatedChan:
		if updatedProj.ID != "proj123" {
			t.Errorf("Expected project ID proj123, got %s", updatedProj.ID)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timed out waiting for gate recalculation update")
	}
}

func TestSaveArtifactContentHandler_Success_JSONPayload(t *testing.T) {
	tempMount := t.TempDir()
	os.Setenv("ARTIFACT_SERVICE_TYPE", "FS")
	os.Setenv("WORKSPACE_MOUNT_PATH", tempMount)
	defer os.Unsetenv("ARTIFACT_SERVICE_TYPE")
	defer os.Unsetenv("WORKSPACE_MOUNT_PATH")

	origArtifactStore := ArtifactStore
	origProjectStore := db.ProjectStoreInstance
	origMessageStore := agent.Store
	origUserProjectStore := UserProjectStore
	origHook := agent.OnFileWriteHook
	defer func() {
		ArtifactStore = origArtifactStore
		db.ProjectStoreInstance = origProjectStore
		agent.Store = origMessageStore
		UserProjectStore = origUserProjectStore
		agent.OnFileWriteHook = origHook
	}()

	targetArt := db.Artifact{
		ID:        "art456",
		ProjectId: "proj123",
		Title:     "architecture.md",
		Path:      "docs/architecture.md",
		Phase:     "SOLUTIONING",
		Status:    db.ArtifactStatusInProgress,
	}

	mStore := &mockSaveArtifactStore{
		artifacts: []db.Artifact{targetArt},
	}
	ArtifactStore = mStore

	updatedChan := make(chan *db.Project, 1)
	mProjectStore := &mockDeleteProjectStore{
		project: &db.Project{
			ID: "proj123",
		},
		updatedChan: updatedChan,
	}
	db.ProjectStoreInstance = mProjectStore
	agent.Store = &safeDummyMessageStore{}
	UserProjectStore = &safeDummyUserProjectStore{}

	agent.OnFileWriteHook = func(ctx context.Context, projectId string, organizationID string, phase string, filePath string) {
		_ = SyncOrRegisterArtifact(ctx, organizationID, projectId, phase, filePath)
		_ = recalculateGates(ctx, organizationID, projectId)
	}

	jsonBody, _ := json.Marshal(map[string]string{
		"content": "### Architecture Design\n\nMicroservices layout.",
	})

	req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/proj123/artifacts/art456/content", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "proj123")
	req.SetPathValue("artifactId", "art456")

	claims := &core.UserClaims{Subject: "developer@google.com"}
	req = withTestCtx(req, claims)

	w := httptest.NewRecorder()
	SaveArtifactContentHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}

	expectedFilePath := filepath.Join(tempMount, "test-organization", "developer@google.com", "proj123", "docs/architecture.md")
	savedBytes, err := os.ReadFile(expectedFilePath)
	if err != nil {
		t.Fatalf("Failed to read saved file from %s: %v", expectedFilePath, err)
	}
	expectedContent := "### Architecture Design\n\nMicroservices layout."
	if string(savedBytes) != expectedContent {
		t.Errorf("Expected content %q, got %q", expectedContent, string(savedBytes))
	}
}

func TestSaveArtifactContentHandler_URLEncodedPath(t *testing.T) {
	tempMount := t.TempDir()
	os.Setenv("ARTIFACT_SERVICE_TYPE", "FS")
	os.Setenv("WORKSPACE_MOUNT_PATH", tempMount)
	defer os.Unsetenv("ARTIFACT_SERVICE_TYPE")
	defer os.Unsetenv("WORKSPACE_MOUNT_PATH")

	origArtifactStore := ArtifactStore
	origProjectStore := db.ProjectStoreInstance
	origMessageStore := agent.Store
	origUserProjectStore := UserProjectStore
	origHook := agent.OnFileWriteHook
	defer func() {
		ArtifactStore = origArtifactStore
		db.ProjectStoreInstance = origProjectStore
		agent.Store = origMessageStore
		UserProjectStore = origUserProjectStore
		agent.OnFileWriteHook = origHook
	}()

	mStore := &mockSaveArtifactStore{
		artifacts: []db.Artifact{},
	}
	ArtifactStore = mStore

	mProjectStore := &mockDeleteProjectStore{
		project: &db.Project{
			ID: "proj123",
		},
		updatedChan: make(chan *db.Project, 1),
	}
	db.ProjectStoreInstance = mProjectStore
	agent.Store = &safeDummyMessageStore{}
	UserProjectStore = &safeDummyUserProjectStore{}

	agent.OnFileWriteHook = func(ctx context.Context, projectId string, organizationID string, phase string, filePath string) {
		_ = SyncOrRegisterArtifact(ctx, organizationID, projectId, phase, filePath)
		_ = recalculateGates(ctx, organizationID, projectId)
	}

	fileContent := "# New file created via path\n"
	req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/proj123/artifacts/docs%2Fspec%2Fcustom.md/content", strings.NewReader(fileContent))
	req.SetPathValue("id", "proj123")
	req.SetPathValue("artifactId", "docs%2Fspec%2Fcustom.md")

	claims := &core.UserClaims{Subject: "user@google.com"}
	req = withTestCtx(req, claims)

	w := httptest.NewRecorder()
	SaveArtifactContentHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}

	expectedFilePath := filepath.Join(tempMount, "test-organization", "user@google.com", "proj123", "docs/spec/custom.md")
	savedBytes, err := os.ReadFile(expectedFilePath)
	if err != nil {
		t.Fatalf("Failed to read saved file from %s: %v", expectedFilePath, err)
	}
	if string(savedBytes) != fileContent {
		t.Errorf("Expected content %q, got %q", fileContent, string(savedBytes))
	}
}

func TestSaveArtifactContentHandler_MissingOrgContext(t *testing.T) {
	req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/proj123/artifacts/art123/content", strings.NewReader("content"))
	req.SetPathValue("id", "proj123")
	req.SetPathValue("artifactId", "art123")

	w := httptest.NewRecorder()
	SaveArtifactContentHandler(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 Unauthorized for missing org context, got %d", w.Code)
	}
}

func TestGetArtifactContentHandler_UnindexedFile_Success(t *testing.T) {
	tempMount := t.TempDir()
	os.Setenv("ARTIFACT_SERVICE_TYPE", "FS")
	os.Setenv("WORKSPACE_MOUNT_PATH", tempMount)
	defer os.Unsetenv("ARTIFACT_SERVICE_TYPE")
	defer os.Unsetenv("WORKSPACE_MOUNT_PATH")

	origArtifactStore := ArtifactStore
	defer func() {
		ArtifactStore = origArtifactStore
	}()
	ArtifactStore = &mockSaveArtifactStore{artifacts: []db.Artifact{}} // Empty, file not in Firestore

	// Write file directly on disk in workspace mount path
	fileDir := filepath.Join(tempMount, "test-organization", "user@google.com", "proj123", "src")
	if err := os.MkdirAll(fileDir, 0755); err != nil {
		t.Fatalf("Failed to create dir: %v", err)
	}
	expectedContent := "// Go file unindexed in firestore\npackage main\n"
	filePath := filepath.Join(fileDir, "main.go")
	if err := os.WriteFile(filePath, []byte(expectedContent), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/proj123/artifacts/src%2Fmain.go/content", nil)
	req.SetPathValue("id", "proj123")
	req.SetPathValue("artifactId", "src%2Fmain.go")

	claims := &core.UserClaims{Subject: "user@google.com"}
	req = withTestCtx(req, claims)

	w := httptest.NewRecorder()
	GetArtifactContentHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}
	if w.Body.String() != expectedContent {
		t.Errorf("Expected body %q, got %q", expectedContent, w.Body.String())
	}
}

func TestGetArtifactContentHandler_MissingOrgContext(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/proj123/artifacts/src%2Fmain.go/content", nil)
	req.SetPathValue("id", "proj123")
	req.SetPathValue("artifactId", "src%2Fmain.go")

	w := httptest.NewRecorder()
	GetArtifactContentHandler(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 Unauthorized for missing org context, got %d", w.Code)
	}
}

func TestGetArtifactContentHandler_SessionArtifact_Success(t *testing.T) {
	tempMount := t.TempDir()
	os.Setenv("ARTIFACT_SERVICE_TYPE", "FS")
	os.Setenv("WORKSPACE_MOUNT_PATH", tempMount)
	defer os.Unsetenv("ARTIFACT_SERVICE_TYPE")
	defer os.Unsetenv("WORKSPACE_MOUNT_PATH")

	origArtifactStore := ArtifactStore
	defer func() {
		ArtifactStore = origArtifactStore
	}()

	// Write session artifact file on disk
	sessionDir := filepath.Join(tempMount, "proj123", "sess456")
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		t.Fatalf("Failed to create session dir: %v", err)
	}
	expectedContent := "This is a session attachment content."
	filePath := filepath.Join(sessionDir, "notes.txt")
	if err := os.WriteFile(filePath, []byte(expectedContent), 0644); err != nil {
		t.Fatalf("Failed to write session file: %v", err)
	}

	sessionArt := db.Artifact{
		ID:          "art-session-1",
		ProjectId:   "proj123",
		SessionId:   "sess456",
		Name:        "notes.txt",
		Title:       "notes.txt",
		MimeType:    "text/plain",
		Size:        int64(len(expectedContent)),
		StorageType: "local_file",
		StoragePath: filePath,
		Phase:       "PLANNING",
		Status:      db.ArtifactStatusComplete,
	}

	ArtifactStore = &mockSaveArtifactStore{artifacts: []db.Artifact{sessionArt}}

	// 1. Fetch by artifact ID
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/proj123/artifacts/art-session-1/content", nil)
	req.SetPathValue("id", "proj123")
	req.SetPathValue("artifactId", "art-session-1")
	claims := &core.UserClaims{Subject: "user@google.com"}
	req = withTestCtx(req, claims)

	w := httptest.NewRecorder()
	GetArtifactContentHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}
	if w.Body.String() != expectedContent {
		t.Errorf("Expected content %q, got %q", expectedContent, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/plain" {
		t.Errorf("Expected Content-Type 'text/plain', got %q", ct)
	}

	// 2. Fetch by artifact Name
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/projects/proj123/artifacts/notes.txt/content", nil)
	req2.SetPathValue("id", "proj123")
	req2.SetPathValue("artifactId", "notes.txt")
	req2 = withTestCtx(req2, claims)

	w2 := httptest.NewRecorder()
	GetArtifactContentHandler(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("Expected status 200 OK for lookup by name, got %d. Body: %s", w2.Code, w2.Body.String())
	}
	if w2.Body.String() != expectedContent {
		t.Errorf("Expected content %q, got %q", expectedContent, w2.Body.String())
	}
}

func TestGetArtifactContentHandler_UnindexedImage_MimeTypeFallback(t *testing.T) {
	tempMount := t.TempDir()
	os.Setenv("ARTIFACT_SERVICE_TYPE", "FS")
	os.Setenv("WORKSPACE_MOUNT_PATH", tempMount)
	defer os.Unsetenv("ARTIFACT_SERVICE_TYPE")
	defer os.Unsetenv("WORKSPACE_MOUNT_PATH")

	origArtifactStore := ArtifactStore
	defer func() {
		ArtifactStore = origArtifactStore
	}()
	ArtifactStore = &mockSaveArtifactStore{artifacts: []db.Artifact{}} // Empty

	fileDir := filepath.Join(tempMount, "test-organization", "user@google.com", "proj123", "assets")
	if err := os.MkdirAll(fileDir, 0755); err != nil {
		t.Fatalf("Failed to create dir: %v", err)
	}
	rawImageData := []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR...")
	filePath := filepath.Join(fileDir, "mock_logo.png")
	if err := os.WriteFile(filePath, rawImageData, 0644); err != nil {
		t.Fatalf("Failed to write image file: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/proj123/artifacts/assets%2Fmock_logo.png/content", nil)
	req.SetPathValue("id", "proj123")
	req.SetPathValue("artifactId", "assets%2Fmock_logo.png")

	claims := &core.UserClaims{Subject: "user@google.com"}
	req = withTestCtx(req, claims)

	w := httptest.NewRecorder()
	GetArtifactContentHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}
	if !bytes.Equal(w.Body.Bytes(), rawImageData) {
		t.Errorf("Expected raw image bytes, got %v", w.Body.Bytes())
	}
	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "image/png") {
		t.Errorf("Expected Content-Type to start with 'image/png', got %q", ct)
	}
}

