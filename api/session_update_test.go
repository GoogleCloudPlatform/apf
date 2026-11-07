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

func TestUpdateSessionHandler_Success(t *testing.T) {
	targetSess := db.Session{
		ID:        "sess-123",
		ProjectId: "proj-123",
		UserId:    "user@google.com",
		Phase:     "PLANNING",
		CreatedAt: time.Now(),
	}

	mSessionStore := &mockTestSessionStore{
		sessions: []db.Session{targetSess},
	}
	mMessageStore := &mockTestMessageStore{}

	handler := NewSessionHandler(mSessionStore, mMessageStore, nil)

	payload := map[string]string{
		"title": "Help - Aug 14, 06:30 PM",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/projects/proj-123/sessions/sess-123", bytes.NewReader(body))
	req.SetPathValue("id", "proj-123")
	req.SetPathValue("sessionId", "sess-123")

	claims := &core.UserClaims{Subject: "user@google.com"}
	ctx := context.WithValue(req.Context(), core.UserClaimsKey, claims)
	ctx = context.WithValue(ctx, core.OrganizationIDKey, "test-org")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.updateSessionHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}

	if len(mSessionStore.sessions) != 1 {
		t.Fatalf("Expected 1 session in store, got %d", len(mSessionStore.sessions))
	}

	if mSessionStore.sessions[0].Title != "Help - Aug 14, 06:30 PM" {
		t.Errorf("Expected session title to be updated to %q, got %q", "Help - Aug 14, 06:30 PM", mSessionStore.sessions[0].Title)
	}
}
