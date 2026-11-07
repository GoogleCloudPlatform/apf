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

package agent

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"google.com/apf/api/core"
	"google.com/apf/api/db"
)

type mockArtifactStore struct {
	db.ArtifactStore
	artifacts []db.Artifact
}

func (m *mockArtifactStore) ListArtifactsForProject(ctx context.Context, organizationID string, projectId string) ([]db.Artifact, error) {
	return m.artifacts, nil
}

func (m *mockArtifactStore) UpdateArtifact(ctx context.Context, organizationID string, a *db.Artifact) error {
	for i, art := range m.artifacts {
		if art.ID == a.ID {
			m.artifacts[i] = *a
			return nil
		}
	}
	return nil
}

func (m *mockArtifactStore) CreateArtifact(ctx context.Context, organizationID string, a *db.Artifact) error {
	m.artifacts = append(m.artifacts, *a)
	return nil
}

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
	return nil, nil
}

func (m *mockMessageStore) StreamProjectActivities(ctx context.Context, organizationID string, projectId string, since time.Time) (<-chan db.Message, <-chan error) {
	return nil, nil
}

func processEventsForTest(ctx context.Context, payload []byte, projectID, organizationID, userID, phase, sessionID string) {
	var events []ADKEvent
	_ = json.Unmarshal(payload, &events)
	for _, event := range events {
		ProcessAgentEvent(ctx, event, projectID, organizationID, userID, phase, sessionID)
	}
}

func TestProcessAgentResponse_FileWriteMessageCreation(t *testing.T) {
	// Setup mock stores
	mockArtStore := &mockArtifactStore{}
	mockMsgStore := &mockMessageStore{}

	SetArtifactStore(mockArtStore)
	Store = mockMsgStore

	// Set config with AgentNameToDocNameMap
	cfg := &core.Configuration{
		AgentNameToDocNameMap: map[string]string{
			"APF Create PRD": "product-req-doc",
		},
	}
	SetConfig(cfg)

	// Create a mock file-write event from the agent
	eventJSON := `[
		{
			"author": "APF Create PRD",
			"invocation_id": "inv123",
			"content": null,
			"partial": false,
			"turn_complete": true,
			"actions": {
				"stateDelta": {
					"file-write-notification:123": "docs/specs/prd.md"
				}
			}
		}
	]`

	ctx := context.Background()
	processEventsForTest(ctx, []byte(eventJSON), "proj-123", "org-123", "user-123", "PLANNING", "sess-123")

	// Check that NO artifacts were written to the DB in the API tier
	if len(mockArtStore.artifacts) != 0 {
		t.Fatalf("Expected 0 artifacts to be created, got %d", len(mockArtStore.artifacts))
	}

	// Check that a file-write notification message was created instead
	var fileWriteMsgs []db.Message
	for _, msg := range mockMsgStore.messages {
		if msg.NotificationDetails != nil && msg.NotificationDetails.Name == "file-write" {
			fileWriteMsgs = append(fileWriteMsgs, msg)
		}
	}

	if len(fileWriteMsgs) != 1 {
		t.Fatalf("Expected 1 file-write message to be created, got %d", len(fileWriteMsgs))
	}

	msg := fileWriteMsgs[0]
	if msg.NotificationDetails == nil || msg.NotificationDetails.Name != "file-write" {
		t.Errorf("Expected message notification name 'file-write', got '%+v'", msg.NotificationDetails)
	}
	filePathParam := msg.NotificationDetails.Params["filePath"].(string)
	if filePathParam != "docs/specs/prd.md" {
		t.Errorf("Expected message filePath param 'docs/specs/prd.md', got '%s'", filePathParam)
	}
}

func TestProcessAgentResponse_MultipleFileWriteMessages(t *testing.T) {
	// Setup mock stores
	mockArtStore := &mockArtifactStore{}
	mockMsgStore := &mockMessageStore{}

	SetArtifactStore(mockArtStore)
	Store = mockMsgStore

	// Set config with AgentNameToDocNameMap
	cfg := &core.Configuration{
		AgentNameToDocNameMap: map[string]string{
			"APF Create Specs": "product-req-doc",
		},
	}
	SetConfig(cfg)

	// Create a mock events body containing unique key-value pairs, with a duplicate path to test deduplication
	eventJSON := `[
		{
			"author": "APF Create Specs",
			"invocation_id": "inv123",
			"content": null,
			"partial": false,
			"turn_complete": true,
			"actions": {
				"stateDelta": {
					"file-write-notification:0": "docs/specs/prd.md",
					"file-write-notification:1": "docs/specs/architecture.md",
					"file-write-notification:2": "docs/specs/brief.md",
					"file-write-notification:3": "docs/specs/prd.md"
				}
			}
		}
	]`

	ctx := context.Background()
	processEventsForTest(ctx, []byte(eventJSON), "proj-123", "org-123", "user-123", "PLANNING", "sess-123")

	// Check that NO artifacts were written to the DB in the API tier
	if len(mockArtStore.artifacts) != 0 {
		t.Fatalf("Expected 0 artifacts to be created, got %d", len(mockArtStore.artifacts))
	}

	// Check that correct number of file-write messages were created (should be 3 due to deduplication)
	var fileWriteMsgs []db.Message
	for _, msg := range mockMsgStore.messages {
		if msg.NotificationDetails != nil && msg.NotificationDetails.Name == "file-write" {
			fileWriteMsgs = append(fileWriteMsgs, msg)
		}
	}

	if len(fileWriteMsgs) != 3 {
		t.Fatalf("Expected 3 file-write messages to be created, got %d", len(fileWriteMsgs))
	}

	paths := make(map[string]bool)
	for _, msg := range fileWriteMsgs {
		if fp, ok := msg.NotificationDetails.Params["filePath"].(string); ok {
			paths[fp] = true
		}
	}

	expectedPaths := []string{"docs/specs/prd.md", "docs/specs/architecture.md", "docs/specs/brief.md"}
	for _, expectedPath := range expectedPaths {
		if !paths[expectedPath] {
			t.Errorf("Expected message for path '%s' to be created", expectedPath)
		}
	}
}

func TestProcessAgentResponse_StreamingThought(t *testing.T) {
	mockArtStore := &mockArtifactStore{}
	mockMsgStore := &mockMessageStore{}

	SetArtifactStore(mockArtStore)
	Store = mockMsgStore

	eventJSON := `[
		{
			"author": "APF Planning Agent",
			"invocation_id": "inv456",
			"content": {
				"role": "model",
				"parts": [
					{
						"thought": "Analyzing repository requirements for the user goal..."
					}
				]
			},
			"partial": true,
			"turn_complete": false,
			"actions": null
		}
	]`

	ctx := context.Background()
	processEventsForTest(ctx, []byte(eventJSON), "proj-123", "org-123", "user-123", "PLANNING", "sess-123")

	// Check that a transient thought message was created
	var thoughtMsgs []db.Message
	for _, msg := range mockMsgStore.messages {
		if msg.NotificationDetails != nil && msg.NotificationDetails.Name == "thought" {
			thoughtMsgs = append(thoughtMsgs, msg)
		}
	}

	if len(thoughtMsgs) != 1 {
		t.Fatalf("Expected 1 thought notification message to be created, got %d", len(thoughtMsgs))
	}

	msg := thoughtMsgs[0]
	if msg.Sender != "APF Planning Agent" {
		t.Errorf("Expected message sender 'APF Planning Agent', got '%s'", msg.Sender)
	}
	if msg.Text != "Analyzing repository requirements for the user goal..." {
		t.Errorf("Expected message text to be 'Analyzing repository requirements for the user goal...', got '%s'", msg.Text)
	}
	if !msg.IsNotification {
		t.Errorf("Expected message to be a notification")
	}
}

func TestProcessAgentResponse_ToolCallAndResultFormatting(t *testing.T) {
	mockArtStore := &mockArtifactStore{}
	mockMsgStore := &mockMessageStore{}

	SetArtifactStore(mockArtStore)
	Store = mockMsgStore

	// Test Tool Call
	toolCallJSON := `[
		{
			"author": "APF Planning Agent",
			"invocation_id": "inv789",
			"content": {
				"role": "model",
				"parts": [
					{
						"function_call": {
							"name": "read_file",
							"args": {
								"path": "/foo/bar.go"
							}
						}
					}
				]
			},
			"partial": false,
			"turn_complete": false
		}
	]`

	ctx := context.Background()
	processEventsForTest(ctx, []byte(toolCallJSON), "proj-123", "org-123", "user-123", "PLANNING", "sess-123")

	if len(mockMsgStore.messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(mockMsgStore.messages))
	}

	callMsg := mockMsgStore.messages[0]
	expectedCallText := "Executing tool: `read_file` (`/foo/bar.go`)"
	if callMsg.Text != expectedCallText {
		t.Errorf("Expected text %q, got %q", expectedCallText, callMsg.Text)
	}

	// Test Tool Result
	mockMsgStore.messages = nil
	toolResultJSON := `[
		{
			"author": "APF Planning Agent",
			"invocation_id": "inv789",
			"content": {
				"role": "function",
				"parts": [
					{
						"functionResponse": {
							"name": "read_file",
							"response": {
								"result": "package main\n\nfunc main() {}\n"
							}
						}
					}
				]
			},
			"partial": false,
			"turn_complete": false
		}
	]`

	processEventsForTest(ctx, []byte(toolResultJSON), "proj-123", "org-123", "user-123", "PLANNING", "sess-123")

	if len(mockMsgStore.messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(mockMsgStore.messages))
	}

	resultMsg := mockMsgStore.messages[0]
	if !strings.Contains(resultMsg.Text, "Read file: /foo/bar.go") {
		t.Errorf("Expected result msg text to mention Read file: /foo/bar.go, got %q", resultMsg.Text)
	}
	if !strings.Contains(resultMsg.Text, "package main") {
		t.Errorf("Expected result msg text to contain read content 'package main', got %q", resultMsg.Text)
	}
	if resultMsg.NotificationDetails == nil || resultMsg.NotificationDetails.Params["content"] != "package main\n\nfunc main() {}\n" {
		t.Errorf("Expected content param in NotificationDetails, got %+v", resultMsg.NotificationDetails)
	}
	if resultMsg.NotificationDetails.Params["file_name"] != "/foo/bar.go" || resultMsg.NotificationDetails.Params["is_file_reader"] != true {
		t.Errorf("Expected file_name and is_file_reader in NotificationDetails Params, got %+v", resultMsg.NotificationDetails.Params)
	}
	if resultMsg.IsNotification {
		t.Errorf("Expected IsNotification to be false for file reader tool result")
	}
}


