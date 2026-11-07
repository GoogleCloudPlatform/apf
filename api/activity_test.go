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
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"google.com/apf/api/core"
	"google.com/apf/api/db"
)

type mockMessageStore struct {
	db.MessageStore
	messages []db.Message
}

func (m *mockMessageStore) CreateMessage(ctx context.Context, organizationID string, projectId string, userId string, msg *db.Message) (*db.Message, error) {
	msg.Timestamp = time.Now()
	m.messages = append(m.messages, *msg)
	return msg, nil
}

func (m *mockMessageStore) ListMessages(ctx context.Context, organizationID string, projectId string, userId string, sessionId string, includeTransient bool) ([]db.Message, error) {
	return m.messages, nil
}

func (m *mockMessageStore) StreamMessages(ctx context.Context, organizationID string, projectId string, userId string, sessionId string, since time.Time) (<-chan db.Message, <-chan error) {
	msgChan := make(chan db.Message)
	errChan := make(chan error)
	go func() {
		defer close(msgChan)
		defer close(errChan)
		for _, msg := range m.messages {
			msgChan <- msg
		}
	}()
	return msgChan, errChan
}

func (m *mockMessageStore) StreamProjectActivities(ctx context.Context, organizationID string, projectId string, since time.Time) (<-chan db.Message, <-chan error) {
	msgChan := make(chan db.Message)
	errChan := make(chan error)
	go func() {
		defer close(msgChan)
		defer close(errChan)
		for _, msg := range m.messages {
			if msg.IsNotification {
				msgChan <- msg
			}
		}
	}()
	return msgChan, errChan
}

func TestIngestActivityHandler(t *testing.T) {
	mockStore := &mockMessageStore{}
	Store = mockStore

	payload := map[string]interface{}{
		"sender": "Developer CLI",
		"text":   "Updated PRD artifact",
		"phase":  "PLANNING",
		"notificationDetails": map[string]interface{}{
			"name": "artifact-update",
			"params": map[string]interface{}{
				"artifact": "PRD",
				"status":   string(db.ArtifactStatusComplete),
			},
		},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/proj123/activities", bytes.NewBuffer(body))
	req.SetPathValue("id", "proj123")

	claims := &core.UserClaims{Subject: "user123", Username: "Developer CLI"}
	req = withTestCtx(req, claims)

	w := httptest.NewRecorder()
	IngestActivityHandler(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201 Created, got %d", w.Code)
	}

	if len(mockStore.messages) == 0 {
		t.Fatalf("Expected message to be saved in store")
	}

	saved := mockStore.messages[0]
	if !saved.IsNotification {
		t.Errorf("Expected IsNotification to be true")
	}
	if saved.Sender != "Developer CLI" {
		t.Errorf("Expected sender 'Developer CLI', got '%s'", saved.Sender)
	}
}
