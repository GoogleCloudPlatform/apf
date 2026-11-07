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
	"fmt"
	"os"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
)

func TestCreateMessage_TimestampBehavior(t *testing.T) {
	emulatorHost := os.Getenv("FIRESTORE_EMULATOR_HOST")
	if emulatorHost == "" {
		t.Skip("Skipping Firestore integration test: FIRESTORE_EMULATOR_HOST is not set")
	}

	ctx := testCtx()
	client, err := firestore.NewClient(ctx, "agenticproductfoundry-489717")
	if err != nil {
		t.Fatalf("Failed to create Firestore client: %v", err)
	}
	defer client.Close()

	// We need a sessionStore to initialize FirestoreMessageStore, but we can pass nil if we don't use session-related methods that need it.
	// CreateMessage doesn't seem to use sessionStore.
	store := NewFirestoreMessageStore(client, nil)

	uniqueSuffix := time.Now().UnixNano()
	projectId := fmt.Sprintf("test-project-msg-%d", uniqueSuffix)
	userId := "test-user-msg"
	sessionId := fmt.Sprintf("test-session-msg-%d", uniqueSuffix)

	t.Run("sets timestamp if zero", func(t *testing.T) {
		msg := &Message{
			Sender:    "User",
			Text:      "Hello world",
			IsUser:    true,
			SessionId: sessionId,
		}

		const organizationID = "org1"

		before := time.Now().Add(-1 * time.Second)
		created, err := store.CreateMessage(ctx, organizationID, projectId, userId, msg)
		after := time.Now().Add(1 * time.Second)

		if err != nil {
			t.Fatalf("Failed to create message: %v", err)
		}

		if created.Timestamp.IsZero() {
			t.Error("Expected created message timestamp to be non-zero")
		}

		if created.Timestamp.Before(before) || created.Timestamp.After(after) {
			t.Errorf("Expected created message timestamp to be around now, got %v (before: %v, after: %v)", created.Timestamp, before, after)
		}

		// Fetch from DB to verify it was saved with the timestamp
		messages, err := store.ListMessages(ctx, organizationID, projectId, userId, sessionId, true)
		if err != nil {
			t.Fatalf("Failed to list messages: %v", err)
		}

		if len(messages) != 1 {
			t.Fatalf("Expected 1 message, got %d", len(messages))
		}

		savedMsg := messages[0]
		if savedMsg.Timestamp.IsZero() {
			t.Error("Expected saved message timestamp to be non-zero")
		}
		// Firestore truncates to microsecond precision, so we truncate the in-memory timestamp for comparison
		expectedTime := created.Timestamp.Truncate(time.Microsecond)
		if !savedMsg.Timestamp.Equal(expectedTime) {
			t.Errorf("Expected saved message timestamp %v to match expected timestamp %v (created: %v)", savedMsg.Timestamp, expectedTime, created.Timestamp)
		}
	})

	t.Run("preserves timestamp if non-zero", func(t *testing.T) {
		specificTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
		msg := &Message{
			Sender:    "User",
			Text:      "Hello historical world",
			IsUser:    true,
			SessionId: sessionId,
			Timestamp: specificTime,
		}

		const organizationID = "org1"

		created, err := store.CreateMessage(ctx, organizationID, projectId, userId, msg)
		if err != nil {
			t.Fatalf("Failed to create message: %v", err)
		}

		if !created.Timestamp.Equal(specificTime) {
			t.Errorf("Expected created message timestamp to be %v, got %v", specificTime, created.Timestamp)
		}

		// Fetch from DB to verify
		// Wait, we have another message in the session now from the previous test if they share the same sessionId.
		// Let's use a different sessionId or clean up.
		// Actually, ListMessages orders by timestamp, so the 2023 one should be first.
		messages, err := store.ListMessages(ctx, organizationID, projectId, userId, sessionId, true)
		if err != nil {
			t.Fatalf("Failed to list messages: %v", err)
		}

		// We expect 2 messages now, ordered by timestamp Asc.
		// The 2023 one should be first.
		if len(messages) < 2 {
			t.Fatalf("Expected at least 2 messages, got %d", len(messages))
		}

		firstMsg := messages[0]
		if !firstMsg.Timestamp.Equal(specificTime) {
			t.Errorf("Expected first message timestamp to be %v, got %v", specificTime, firstMsg.Timestamp)
		}
		if firstMsg.Text != "Hello historical world" {
			t.Errorf("Expected first message to be the historical one, got text: %q", firstMsg.Text)
		}
	})
}
