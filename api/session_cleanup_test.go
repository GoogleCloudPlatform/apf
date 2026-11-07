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
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"google.com/apf/api/core"
	"google.com/apf/api/db"
	"google.com/apf/api/workspace"
)

type mockCleanupProjectStore struct {
	db.ProjectStore
	projects map[string]*db.Project
}

func (m *mockCleanupProjectStore) GetProject(ctx context.Context, organizationID string, id string) (*db.Project, error) {
	if p, ok := m.projects[id]; ok {
		return p, nil
	}
	return nil, db.ErrNotFound
}

type mockCleanupSessionStore struct {
	sessions []db.Session
}

func (m *mockCleanupSessionStore) CreateSession(ctx context.Context, organizationID string, s *db.Session) error {
	if s.ID == "" {
		s.ID = fmt.Sprintf("mock-sess-%d", len(m.sessions)+1)
	}
	m.sessions = append(m.sessions, *s)
	return nil
}

func (m *mockCleanupSessionStore) ListSessionsForProject(ctx context.Context, organizationID string, projectId string, userId string, phase string) ([]db.Session, error) {
	var result []db.Session
	for _, s := range m.sessions {
		if s.ProjectId == projectId && (phase == "" || s.Phase == phase) {
			result = append(result, s)
		}
	}
	return result, nil
}

func (m *mockCleanupSessionStore) DeleteSession(ctx context.Context, organizationID string, id string) error {
	for i, s := range m.sessions {
		if s.ID == id {
			m.sessions = append(m.sessions[:i], m.sessions[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("session not found in mock store")
}

func (m *mockCleanupSessionStore) UpdateSessionStatus(ctx context.Context, organizationID string, id string, status string) error {
	for i, s := range m.sessions {
		if s.ID == id {
			m.sessions[i].Status = status
			return nil
		}
	}
	return fmt.Errorf("session not found in mock store")
}

func (m *mockCleanupSessionStore) UpdateSessionTitle(ctx context.Context, organizationID string, id string, title string) error {
	for i, s := range m.sessions {
		if s.ID == id {
			m.sessions[i].Title = title
			return nil
		}
	}
	return fmt.Errorf("session not found in mock store")
}

func TestConnectSessionHandler_RollbackOnFailure(t *testing.T) {
	origStore := store
	origProjInst := db.ProjectStoreInstance
	origWsStore := workspace.SessionStore

	mProjectStore := &mockCleanupProjectStore{
		projects: map[string]*db.Project{
			"proj123": {
				ID: "proj123",
			},
		},
	}
	store = mProjectStore
	db.ProjectStoreInstance = mProjectStore

	t.Setenv("AGENT_BASE_URL", "http://127.0.0.1:54321")
	mSessionStore := &mockCleanupSessionStore{
		sessions: []db.Session{},
	}
	workspace.SessionStore = mSessionStore
	mMessageStore := &mockTestMessageStore{}

	defer func() {
		store = origStore
		db.ProjectStoreInstance = origProjInst
		workspace.SessionStore = origWsStore
	}()

	handler := NewSessionHandler(mSessionStore, mMessageStore, nil)

	body := []byte(`{"phase":"ONBOARDING"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/proj123/sessions?createNew=true", bytes.NewReader(body))
	req.SetPathValue("id", "proj123")

	claims := &core.UserClaims{Subject: "user@google.com"}
	req = withTestCtx(req, claims)

	w := httptest.NewRecorder()
	handler.connectSessionHandler(w, req)

	// Since agent backend is not running, initialization fails with 500
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500 InternalServerError, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp["code"] != "SESSION_INIT_FAILED" {
		t.Errorf("Expected code 'SESSION_INIT_FAILED', got %v", resp["code"])
	}

	// Crucial assertion: Newly created session must be rolled back and NOT linger in the database
	if len(mSessionStore.sessions) != 0 {
		t.Errorf("Expected session store to have 0 sessions after rollback, but found %d: %+v", len(mSessionStore.sessions), mSessionStore.sessions)
	}
}

func TestConnectSessionHandler_ExistingSessionRetryFailure(t *testing.T) {
	origStore := store
	origProjInst := db.ProjectStoreInstance
	origWsStore := workspace.SessionStore

	t.Setenv("AGENT_BASE_URL", "http://127.0.0.1:54321")
	mProjectStore := &mockCleanupProjectStore{
		projects: map[string]*db.Project{
			"proj123": {
				ID: "proj123",
			},
		},
	}
	store = mProjectStore
	db.ProjectStoreInstance = mProjectStore

	existingSess := db.Session{
		ID:        "sess-existing",
		ProjectId: "proj123",
		UserId:    "user@google.com",
		Phase:     "ONBOARDING",
		Status:    "INITIALIZING",
		CreatedAt: time.Now(),
	}

	mSessionStore := &mockCleanupSessionStore{
		sessions: []db.Session{existingSess},
	}
	workspace.SessionStore = mSessionStore
	mMessageStore := &mockTestMessageStore{}

	defer func() {
		store = origStore
		db.ProjectStoreInstance = origProjInst
		workspace.SessionStore = origWsStore
	}()

	handler := NewSessionHandler(mSessionStore, mMessageStore, nil)

	body := []byte(`{"phase":"ONBOARDING"}`)
	// Connect to existing session (createNew=false)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/proj123/sessions", bytes.NewReader(body))
	req.SetPathValue("id", "proj123")

	claims := &core.UserClaims{Subject: "user@google.com"}
	req = withTestCtx(req, claims)

	w := httptest.NewRecorder()
	handler.connectSessionHandler(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500 InternalServerError, got %d. Body: %s", w.Code, w.Body.String())
	}

	// For existing session retry, session record is preserved and updated to FAILED
	if len(mSessionStore.sessions) != 1 {
		t.Fatalf("Expected 1 session in store, got %d", len(mSessionStore.sessions))
	}
	if mSessionStore.sessions[0].Status != "FAILED" {
		t.Errorf("Expected existing session status to be updated to FAILED, got %q", mSessionStore.sessions[0].Status)
	}
}
