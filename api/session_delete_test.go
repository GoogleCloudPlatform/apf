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
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"google.com/apf/api/core"
	"google.com/apf/api/db"
)

type mockTestSessionStore struct {
	db.SessionStore
	sessions []db.Session
}

func (m *mockTestSessionStore) ListSessionsForProject(ctx context.Context, organizationID string, projectId string, userId string, phase string) ([]db.Session, error) {
	return m.sessions, nil
}

func (m *mockTestSessionStore) DeleteSession(ctx context.Context, organizationID string, id string) error {
	for i, s := range m.sessions {
		if s.ID == id {
			m.sessions = append(m.sessions[:i], m.sessions[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("session not found in mock store")
}

func (m *mockTestSessionStore) UpdateSessionStatus(ctx context.Context, organizationID string, id string, status string) error {
	for i, s := range m.sessions {
		if s.ID == id {
			m.sessions[i].Status = status
			return nil
		}
	}
	return fmt.Errorf("session not found in mock store")
}

func (m *mockTestSessionStore) UpdateSessionTitle(ctx context.Context, organizationID string, id string, title string) error {
	for i, s := range m.sessions {
		if s.ID == id {
			m.sessions[i].Title = title
			return nil
		}
	}
	return fmt.Errorf("session not found in mock store")
}


type mockTestMessageStore struct {
	db.MessageStore
	messages []*db.Message
}

func (m *mockTestMessageStore) CreateMessage(ctx context.Context, organizationID string, projectId string, userId string, msg *db.Message) (*db.Message, error) {
	m.messages = append(m.messages, msg)
	return msg, nil
}

func TestDeleteSessionHandler_Success(t *testing.T) {
	targetSess := db.Session{
		ID:        "sess123",
		ProjectId: "proj123",
		UserId:    "user@google.com",
		Phase:     "ANALYSIS",
		CreatedAt: time.Now(),
	}

	mSessionStore := &mockTestSessionStore{
		sessions: []db.Session{targetSess},
	}
	mMessageStore := &mockTestMessageStore{}

	// Instantiate handler with mocks
	handler := NewSessionHandler(mSessionStore, mMessageStore, nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/proj123/sessions/sess123", nil)
	req.SetPathValue("id", "proj123")
	req.SetPathValue("sessionId", "sess123")

	claims := &core.UserClaims{Subject: "user@google.com"}
	req = withTestCtx(req, claims)

	w := httptest.NewRecorder()
	handler.deleteSessionHandler(w, req)

	// Assert Response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if resp["message"] != "Session deleted successfully" {
		t.Errorf("Expected success message, got %q", resp["message"])
	}

	// Assert DB deletion
	if len(mSessionStore.sessions) != 0 {
		t.Errorf("Expected mock sessions list to be empty after deletion, got %d items", len(mSessionStore.sessions))
	}

	// Assert Broadcast message is created
	foundDeleteNotify := false
	for _, m := range mMessageStore.messages {
		if m.NotificationDetails != nil && m.NotificationDetails.Name == "session-delete" {
			foundDeleteNotify = true
			if m.NotificationDetails.Params["sessionId"] != "sess123" {
				t.Errorf("Expected broadcasted sessionId to be 'sess123', got %v", m.NotificationDetails.Params["sessionId"])
			}
			if m.NotificationDetails.Params["phase"] != "ANALYSIS" {
				t.Errorf("Expected broadcasted phase to be 'ANALYSIS', got %v", m.NotificationDetails.Params["phase"])
			}
			break
		}
	}
	if !foundDeleteNotify {
		t.Error("Expected to find a 'session-delete' notification message broadcasted")
	}
}

func TestDeleteSessionHandler_NotFound(t *testing.T) {
	mSessionStore := &mockTestSessionStore{
		sessions: []db.Session{},
	}
	mMessageStore := &mockTestMessageStore{}

	handler := NewSessionHandler(mSessionStore, mMessageStore, nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/proj123/sessions/sess-missing", nil)
	req.SetPathValue("id", "proj123")
	req.SetPathValue("sessionId", "sess-missing")

	claims := &core.UserClaims{Subject: "user@google.com"}
	req = withTestCtx(req, claims)

	w := httptest.NewRecorder()
	handler.deleteSessionHandler(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500 Internal Server Error when session is not found, got %d. Body: %s", w.Code, w.Body.String())
	}
}
