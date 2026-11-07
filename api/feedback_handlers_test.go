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
	"strings"
	"testing"

	"google.com/apf/api/core"
	"google.com/apf/api/db"
)

func TestCreateFeedbackHandler_Success(t *testing.T) {
	mockStore := db.NewMockFeedbackStore()
	handler := NewFeedbackHandler(mockStore)

	payload := map[string]string{
		"type":      "bug",
		"subject":   "Test Bug Subject",
		"message":   "Detailed error log message",
		"userEmail": "testuser@example.com",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/v1/feedback", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// Inject user claims into context
	claims := &core.UserClaims{
		Username:       "testuser@example.com",
		OrganizationID: "org-test-123",
	}
	ctx := req.Context()
	ctx = core.WithOrganizationID(ctx, "org-test-123")
	ctx = context.WithValue(ctx, core.UserClaimsKey, claims)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.CreateFeedbackHandler(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201 Created, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp["status"] != "success" {
		t.Errorf("expected status 'success', got %v", resp["status"])
	}
	if resp["id"] == nil || resp["id"] == "" {
		t.Errorf("expected non-empty feedback id in response")
	}
}

func TestCreateFeedbackHandler_InvalidPayload(t *testing.T) {
	mockStore := db.NewMockFeedbackStore()
	handler := NewFeedbackHandler(mockStore)

	testCases := []struct {
		name    string
		payload map[string]string
	}{
		{
			name: "invalid feedback type",
			payload: map[string]string{
				"type":    "unknown_type",
				"subject": "Valid Subject",
				"message": "Valid Message",
			},
		},
		{
			name: "missing subject",
			payload: map[string]string{
				"type":    "bug",
				"subject": "",
				"message": "Valid Message",
			},
		},
		{
			name: "subject exceeding max length",
			payload: map[string]string{
				"type":    "feature_request",
				"subject": strings.Repeat("a", 201),
				"message": "Valid Message",
			},
		},
		{
			name: "missing message",
			payload: map[string]string{
				"type":    "general",
				"subject": "Valid Subject",
				"message": "",
			},
		},
		{
			name: "message exceeding max length",
			payload: map[string]string{
				"type":    "general",
				"subject": "Valid Subject",
				"message": strings.Repeat("m", 4001),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			body, _ := json.Marshal(tc.payload)
			req := httptest.NewRequest("POST", "/api/v1/feedback", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			handler.CreateFeedbackHandler(rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Errorf("[%s] expected status 400 Bad Request, got %d", tc.name, rr.Code)
			}

			var resp map[string]interface{}
			_ = json.Unmarshal(rr.Body.Bytes(), &resp)
			if resp["code"] != "INVALID_FEEDBACK_PAYLOAD" {
				t.Errorf("[%s] expected error code INVALID_FEEDBACK_PAYLOAD, got %v", tc.name, resp["code"])
			}
		})
	}
}
