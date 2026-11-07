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
	"testing"

	"google.com/apf/api/core"
)

func TestDirectoryReaderSuppression_DebugFalse(t *testing.T) {
	ctx := context.Background()
	mockStore := &mockMessageStore{}
	originalStore := Store
	originalConfig := config
	Store = mockStore
	defer func() {
		Store = originalStore
		config = originalConfig
	}()

	cfg := &core.Configuration{
		DebugMode: false,
		LogLevel:  "INFO",
	}
	SetConfig(cfg)

	eventCtx := EventContext{
		OrganizationID: "test-org",
		ProjectID:      "test-proj",
		UserID:         "test-user",
		SessionID:      "test-sess",
	}

	callEvent := ADKEvent{
		Author:       SenderAgent,
		InvocationID: "inv-dir-1",
		Content: &EventContent{
			Parts: []map[string]interface{}{
				{
					KeyFunctionCall: map[string]interface{}{
						KeyName: "directory_reader",
						KeyArgs: map[string]interface{}{
							"directoryPathOrGlob": "/tmp",
						},
					},
				},
			},
		},
	}

	handleToolCallRequest(ctx, callEvent, eventCtx)

	if len(mockStore.messages) != 0 {
		t.Fatalf("Expected 0 messages stored for directory_reader call when debug mode is false, got %d", len(mockStore.messages))
	}

	resultEvent := ADKEvent{
		Author:       SenderAgent,
		InvocationID: "inv-dir-1",
		Content: &EventContent{
			Parts: []map[string]interface{}{
				{
					KeyFunctionResponse: map[string]interface{}{
						KeyName: "directory_reader",
						KeyResponse: map[string]interface{}{
							"filesPaths": []string{"/tmp/a.txt"},
						},
					},
				},
			},
		},
	}

	handleToolResultProvided(ctx, resultEvent, eventCtx)

	if len(mockStore.messages) != 0 {
		t.Fatalf("Expected 0 messages stored for directory_reader result when debug mode is false, got %d", len(mockStore.messages))
	}
}

func TestDirectoryReaderSuppression_DebugTrue(t *testing.T) {
	ctx := context.Background()
	mockStore := &mockMessageStore{}
	originalStore := Store
	originalConfig := config
	Store = mockStore
	defer func() {
		Store = originalStore
		config = originalConfig
	}()

	cfg := &core.Configuration{
		DebugMode: true,
		LogLevel:  "INFO",
	}
	SetConfig(cfg)

	eventCtx := EventContext{
		OrganizationID: "test-org",
		ProjectID:      "test-proj",
		UserID:         "test-user",
		SessionID:      "test-sess",
	}

	callEvent := ADKEvent{
		Author:       SenderAgent,
		InvocationID: "inv-dir-2",
		Content: &EventContent{
			Parts: []map[string]interface{}{
				{
					KeyFunctionCall: map[string]interface{}{
						KeyName: "directory_reader",
						KeyArgs: map[string]interface{}{
							"directoryPathOrGlob": "/tmp",
						},
					},
				},
			},
		},
	}

	handleToolCallRequest(ctx, callEvent, eventCtx)

	if len(mockStore.messages) != 1 {
		t.Fatalf("Expected 1 message stored for directory_reader call when debug mode is true, got %d", len(mockStore.messages))
	}

	resultEvent := ADKEvent{
		Author:       SenderAgent,
		InvocationID: "inv-dir-2",
		Content: &EventContent{
			Parts: []map[string]interface{}{
				{
					KeyFunctionResponse: map[string]interface{}{
						KeyName: "directory_reader",
						KeyResponse: map[string]interface{}{
							"filesPaths": []string{"/tmp/a.txt"},
						},
					},
				},
			},
		},
	}

	handleToolResultProvided(ctx, resultEvent, eventCtx)

	if len(mockStore.messages) != 2 {
		t.Fatalf("Expected 2 messages stored for directory_reader call+result when debug mode is true, got %d", len(mockStore.messages))
	}
}

func TestOtherToolsNotSuppressed_DebugFalse(t *testing.T) {
	ctx := context.Background()
	mockStore := &mockMessageStore{}
	originalStore := Store
	originalConfig := config
	Store = mockStore
	defer func() {
		Store = originalStore
		config = originalConfig
	}()

	cfg := &core.Configuration{
		DebugMode: false,
		LogLevel:  "INFO",
	}
	SetConfig(cfg)

	eventCtx := EventContext{
		OrganizationID: "test-org",
		ProjectID:      "test-proj",
		UserID:         "test-user",
		SessionID:      "test-sess",
	}

	callEvent := ADKEvent{
		Author:       SenderAgent,
		InvocationID: "inv-grep-1",
		Content: &EventContent{
			Parts: []map[string]interface{}{
				{
					KeyFunctionCall: map[string]interface{}{
						KeyName: "grep_search",
						KeyArgs: map[string]interface{}{
							"query": "foo",
						},
					},
				},
			},
		},
	}

	handleToolCallRequest(ctx, callEvent, eventCtx)

	if len(mockStore.messages) != 1 {
		t.Fatalf("Expected 1 message stored for non-directory_reader call even when debug mode is false, got %d", len(mockStore.messages))
	}
}
