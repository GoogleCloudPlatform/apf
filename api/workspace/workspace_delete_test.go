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

import (	"google.com/apf/api/agent"

	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"google.com/apf/api/core"
	"google.com/apf/api/db"
)

// Safe dummy mocks to prevent async panics after test exits
type safeDummyArtifactStore struct{ db.ArtifactStore }
func (s *safeDummyArtifactStore) ListArtifactsForProject(ctx context.Context, organizationID string, projectId string) ([]db.Artifact, error) {
	return nil, nil
}
func (s *safeDummyArtifactStore) DeleteArtifact(ctx context.Context, organizationID string, artifactId string) error {
	return nil
}

type safeDummyProjectStore struct{ db.ProjectStore }
func (s *safeDummyProjectStore) GetProject(ctx context.Context, organizationID string, id string) (*db.Project, error) {
	return &db.Project{ID: id}, nil
}
func (s *safeDummyProjectStore) UpdateProject(ctx context.Context, organizationID string, p *db.Project) error {
	return nil
}

type safeDummyMessageStore struct{ db.MessageStore }
func (s *safeDummyMessageStore) CreateMessage(ctx context.Context, organizationID string, projectId string, userId string, msg *db.Message) (*db.Message, error) {
	return msg, nil
}

// Test specific mocks
type mockDeleteArtifactStore struct {
	db.ArtifactStore
	artifacts []db.Artifact
}

func (m *mockDeleteArtifactStore) ListArtifactsForProject(ctx context.Context, organizationID string, projectId string) ([]db.Artifact, error) {
	return m.artifacts, nil
}

func (m *mockDeleteArtifactStore) DeleteArtifact(ctx context.Context, organizationID string, artifactId string) error {
	for i, a := range m.artifacts {
		if a.ID == artifactId {
			m.artifacts = append(m.artifacts[:i], m.artifacts[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("artifact not found in mock db.ProjectStoreInstance")
}

type mockDeleteProjectStore struct {
	db.ProjectStore
	project     *db.Project
	updatedChan chan *db.Project
}

func (m *mockDeleteProjectStore) GetProject(ctx context.Context, organizationID string, id string) (*db.Project, error) {
	if m.project == nil {
		return nil, errors.New("project not found in mock db.ProjectStoreInstance")
	}
	return m.project, nil
}

func (m *mockDeleteProjectStore) UpdateProject(ctx context.Context, organizationID string, p *db.Project) error {
	if m.updatedChan != nil {
		m.updatedChan <- p
	}
	return nil
}

type mockDeleteMessageStore struct {
	db.MessageStore
	messages []*db.Message
}

func (m *mockDeleteMessageStore) CreateMessage(ctx context.Context, organizationID string, projectId string, userId string, msg *db.Message) (*db.Message, error) {
	m.messages = append(m.messages, msg)
	return msg, nil
}

func TestDeleteArtifactHandler_Success(t *testing.T) {
	// Force FileSystem artifact service for testing to avoid GCS initialization
	os.Setenv("ARTIFACT_SERVICE_TYPE", "FS")
	os.Setenv("WORKSPACE_MOUNT_PATH", t.TempDir())
	defer os.Unsetenv("ARTIFACT_SERVICE_TYPE")
	defer os.Unsetenv("WORKSPACE_MOUNT_PATH")

	// Save original stores
	origArtifactStore := ArtifactStore
	origStore := db.ProjectStoreInstance
	origMessageStore := agent.Store
	defer func() {
		// Restore to safe dummy stores instead of nil to prevent late async goroutine panics
		ArtifactStore = &safeDummyArtifactStore{}
		db.ProjectStoreInstance = &safeDummyProjectStore{}
		agent.Store = &safeDummyMessageStore{}
	}()

	// Setup mock data
	targetArt := db.Artifact{
		ID:        "art123",
		ProjectId: "proj123",
		Title:     "test_spec.md",
		Path:      "specs/test_spec.md",
		Phase:     "ANALYSIS",
		Status:    db.ArtifactStatusComplete,
	}

	mArtifactStore := &mockDeleteArtifactStore{
		artifacts: []db.Artifact{targetArt},
	}
	ArtifactStore = mArtifactStore

	// Create the physical file to ensure it gets deleted
	tempDir := os.Getenv("WORKSPACE_MOUNT_PATH")
	projectId := "proj123"
	filePath := "specs/test_spec.md"
	
	organizationId := "test-organization"
	fullPath := filepath.Join(tempDir, organizationId, "user@google.com", projectId, filePath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		t.Fatalf("Failed to create mock file directory: %v", err)
	}
	if err := os.WriteFile(fullPath, []byte("mock file content"), 0644); err != nil {
		t.Fatalf("Failed to write mock file: %v", err)
	}

	updatedChan := make(chan *db.Project, 1)
	mProjectStore := &mockDeleteProjectStore{
		project: &db.Project{
			ID: "proj123",
		},
		updatedChan: updatedChan,
	}
	db.ProjectStoreInstance = mProjectStore

	mMessageStore := &mockDeleteMessageStore{}
	agent.Store = mMessageStore

	// Create request
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/proj123/artifacts/art123", nil)
	req.SetPathValue("id", "proj123")
	req.SetPathValue("artifactId", "art123")

	claims := &core.UserClaims{Subject: "user@google.com"}
	req = withTestCtx(req, claims)

	w := httptest.NewRecorder()
	DeleteArtifactHandler(w, req)

	// Assert response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if resp["message"] != "Artifact deleted successfully" {
		t.Errorf("Expected success message, got %q", resp["message"])
	}

	// Assert database deletion
	if len(mArtifactStore.artifacts) != 0 {
		t.Errorf("Expected artifact list to be empty after deletion, got %d items", len(mArtifactStore.artifacts))
	}

	// Assert physical file deletion
	if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
		t.Errorf("Expected physical file to be deleted, but it still exists at %s (err: %v)", fullPath, err)
	}

	// Assert message broadcasted (synchronous delete notification)
	foundDeleteNotify := false
	for _, m := range mMessageStore.messages {
		if m.NotificationDetails != nil && m.NotificationDetails.Name == "artifact-delete" {
			foundDeleteNotify = true
			if m.NotificationDetails.Params["artifactId"] != "art123" {
				t.Errorf("Expected broadcasted artifactId to be 'art123', got %v", m.NotificationDetails.Params["artifactId"])
			}
			break
		}
	}
	if !foundDeleteNotify {
		t.Error("Expected to find an 'artifact-delete' notification message broadcasted")
	}

	// Wait for async gate recalculation and assert it updated the project
	select {
	case updatedProject := <-updatedChan:
		if updatedProject.ID != "proj123" {
			t.Errorf("Expected updated project ID to be 'proj123', got %q", updatedProject.ID)
		}
		if updatedProject.GovernanceGates == nil {
			t.Error("Expected project governance gates to be updated/recalculated")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for async gate recalculation to trigger UpdateProject")
	}

	// Restore original stores just in case (though defer will set them to safe dummies)
	_ = origArtifactStore
	_ = origStore
	_ = origMessageStore
}

func TestDeleteArtifactHandler_NotFound(t *testing.T) {
	origArtifactStore := ArtifactStore
	defer func() {
		ArtifactStore = &safeDummyArtifactStore{}
	}()

	// Mock empty artifact db.ProjectStoreInstance
	ArtifactStore = &mockDeleteArtifactStore{
		artifacts: []db.Artifact{},
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/proj123/artifacts/art-missing", nil)
	req.SetPathValue("id", "proj123")
	req.SetPathValue("artifactId", "art-missing")

	claims := &core.UserClaims{Subject: "user@google.com"}
	req = withTestCtx(req, claims)

	w := httptest.NewRecorder()
	DeleteArtifactHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 Not Found, got %d", w.Code)
	}

	_ = origArtifactStore
}

func TestDeleteArtifactHandler_UnindexedFile_Success(t *testing.T) {
	os.Setenv("ARTIFACT_SERVICE_TYPE", "FS")
	tempDir := t.TempDir()
	os.Setenv("WORKSPACE_MOUNT_PATH", tempDir)
	defer os.Unsetenv("ARTIFACT_SERVICE_TYPE")
	defer os.Unsetenv("WORKSPACE_MOUNT_PATH")

	origArtifactStore := ArtifactStore
	defer func() {
		ArtifactStore = &safeDummyArtifactStore{}
	}()

	ArtifactStore = &mockDeleteArtifactStore{
		artifacts: []db.Artifact{},
	}

	organizationId := "test-organization"
	projectId := "proj123"
	filePath := "src/utils/unindexed.ts"
	fullPath := filepath.Join(tempDir, organizationId, "user@google.com", projectId, filePath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		t.Fatalf("Failed to create mock file directory: %v", err)
	}
	if err := os.WriteFile(fullPath, []byte("export const x = 1;"), 0644); err != nil {
		t.Fatalf("Failed to write mock file: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/proj123/artifacts/"+url.PathEscape(filePath), nil)
	req.SetPathValue("id", "proj123")
	req.SetPathValue("artifactId", url.PathEscape(filePath))

	claims := &core.UserClaims{Subject: "user@google.com"}
	req = withTestCtx(req, claims)

	w := httptest.NewRecorder()
	DeleteArtifactHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
		t.Errorf("Expected physical file to be deleted, but it still exists")
	}

	_ = origArtifactStore
}

