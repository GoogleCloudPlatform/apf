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
	"testing"
)

func TestMockFeedbackStore_CreateAndGet(t *testing.T) {
	ctx := context.Background()
	store := NewMockFeedbackStore()

	fb := &Feedback{
		Type:           "bug",
		Subject:        "Button click fails",
		Message:        "Clicking submit throws error",
		UserEmail:      "user@example.com",
		OrganizationID: "org-123",
		ProjectID:      "proj-456",
	}

	created, err := store.CreateFeedback(ctx, fb)
	if err != nil {
		t.Fatalf("expected no error creating feedback, got: %v", err)
	}

	if created.ID == "" {
		t.Errorf("expected auto-generated feedback ID, got empty")
	}

	if created.CreatedAt.IsZero() {
		t.Errorf("expected CreatedAt timestamp to be set")
	}

	fetched, err := store.GetFeedback(ctx, created.ID)
	if err != nil {
		t.Fatalf("expected to fetch created feedback, got error: %v", err)
	}

	if fetched.Type != fb.Type || fetched.Subject != fb.Subject || fetched.Message != fb.Message {
		t.Errorf("fetched feedback fields mismatch with original")
	}
}

func TestMockFeedbackStore_InvalidFeedback(t *testing.T) {
	ctx := context.Background()
	store := NewMockFeedbackStore()

	invalidFb := &Feedback{
		Type:    "bug",
		Subject: "", // empty subject
		Message: "Some message",
	}

	_, err := store.CreateFeedback(ctx, invalidFb)
	if err == nil {
		t.Errorf("expected error for invalid feedback with empty subject, got nil")
	}
}
