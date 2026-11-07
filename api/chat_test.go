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
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"google.com/apf/api/agent"
	core "google.com/apf/api/core"
	"google.com/apf/api/db"
)

func TestCancelMessageHandler(t *testing.T) {
	handler := NewChatHandler(nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	handler.RegisterExecution("proj1", "sess1", cancel)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/proj1/messages/cancel?sessionId=sess1", nil)
	req.SetPathValue("id", "proj1")

	claims := &core.UserClaims{Subject: "user1"}
	req = req.WithContext(context.WithValue(req.Context(), core.UserClaimsKey, claims))

	rr := httptest.NewRecorder()
	handler.CancelMessageHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %d", rr.Code)
	}

	select {
	case <-ctx.Done():
		// Success
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected context to be cancelled")
	}
}

func TestStreamLargeChunk_ScannerBuffer(t *testing.T) {
	// Simulate an SSE line larger than standard bufio.MaxScanTokenSize (65536 bytes)
	largePayload := strings.Repeat("A", 100*1024) // 100 KB single token
	sseLine := "data: {\"content\":\"" + largePayload + "\", \"turnComplete\": true}\n"

	reader := strings.NewReader(sseLine)
	scanner := bufio.NewScanner(reader)
	const maxCapacity = 32 * 1024 * 1024 // 32 MB max line buffer
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, maxCapacity)

	scanned := scanner.Scan()
	if !scanned {
		t.Fatalf("Expected scanner to scan large line (>64KB) cleanly, got err: %v", scanner.Err())
	}
	if scanner.Err() != nil {
		t.Fatalf("Unexpected scanner error on large line: %v", scanner.Err())
	}
	if len(scanner.Text()) < 100*1024 {
		t.Fatalf("Expected scanned text length >= 100KB, got %d", len(scanner.Text()))
	}
}

type testMessageStore struct {
	db.MessageStore
	savedMsg *db.Message
}

func (m *testMessageStore) CreateMessage(ctx context.Context, organizationID string, projectId string, userId string, msg *db.Message) (*db.Message, error) {
	m.savedMsg = msg
	copy := *msg
	copy.ID = "msg-123"
	return &copy, nil
}

func TestUserMessageHandler_PromptAndDisplayPrompt(t *testing.T) {
	mockStore := &testMessageStore{}
	handler := NewChatHandler(mockStore)

	body := `{"sender":"User","text":"Generate project context","displayText":"Generate project context","displayPrompt":"Generate project context","prompt":"/bmad-project-context","isUser":false,"phase":"ONBOARDING","sessionId":"sess1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/proj1/messages?sessionId=sess1", strings.NewReader(body))
	req.SetPathValue("id", "proj1")

	claims := &core.UserClaims{Subject: "user1", OrganizationID: "org1"}
	req = req.WithContext(context.WithValue(req.Context(), core.UserClaimsKey, claims))
	req = req.WithContext(core.WithOrganizationID(req.Context(), "org1"))

	rr := httptest.NewRecorder()
	handler.UserMessageHandler(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("Expected status Created, got %d: %s", rr.Code, rr.Body.String())
	}

	if mockStore.savedMsg == nil {
		t.Fatal("Expected message to be saved to store")
	}

	if mockStore.savedMsg.Prompt != "/bmad-project-context" {
		t.Errorf("Expected Prompt to be '/bmad-project-context', got '%s'", mockStore.savedMsg.Prompt)
	}

	if mockStore.savedMsg.Text != "Generate project context" {
		t.Errorf("Expected Text to be 'Generate project context', got '%s'", mockStore.savedMsg.Text)
	}

	if mockStore.savedMsg.DisplayText != "Generate project context" {
		t.Errorf("Expected DisplayText to be 'Generate project context', got '%s'", mockStore.savedMsg.DisplayText)
	}

	if mockStore.savedMsg.DisplayPrompt != "Generate project context" {
		t.Errorf("Expected DisplayPrompt to be 'Generate project context', got '%s'", mockStore.savedMsg.DisplayPrompt)
	}
}

func TestUserMessageHandler_Artifacts(t *testing.T) {
	mockStore := &testMessageStore{}
	handler := NewChatHandler(mockStore)

	body := `{"sender":"User","text":"Review architecture and notes","prompt":"Review architecture and notes","isUser":false,"phase":"PLANNING","sessionId":"sess1","artifactIds":["art-1","art-2"],"artifacts":[{"id":"art-1","name":"diagram.png","mimeType":"image/png","size":2048000,"storageType":"local_file","storagePath":"/artifacts/diagram.png"},{"id":"art-2","name":"notes.md","mimeType":"text/markdown","size":1024,"storageType":"local_file","storagePath":"/artifacts/notes.md"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/proj1/messages?sessionId=sess1", strings.NewReader(body))
	req.SetPathValue("id", "proj1")

	claims := &core.UserClaims{Subject: "user1", OrganizationID: "org1"}
	req = req.WithContext(context.WithValue(req.Context(), core.UserClaimsKey, claims))
	req = req.WithContext(core.WithOrganizationID(req.Context(), "org1"))

	rr := httptest.NewRecorder()
	handler.UserMessageHandler(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("Expected status Created, got %d: %s", rr.Code, rr.Body.String())
	}

	if mockStore.savedMsg == nil {
		t.Fatal("Expected message to be saved to store")
	}

	if len(mockStore.savedMsg.ArtifactIDs) != 2 {
		t.Fatalf("Expected 2 ArtifactIDs, got %d", len(mockStore.savedMsg.ArtifactIDs))
	}
	if mockStore.savedMsg.ArtifactIDs[0] != "art-1" || mockStore.savedMsg.ArtifactIDs[1] != "art-2" {
		t.Errorf("Unexpected ArtifactIDs: %v", mockStore.savedMsg.ArtifactIDs)
	}

	if len(mockStore.savedMsg.Artifacts) != 2 {
		t.Fatalf("Expected 2 Artifacts, got %d", len(mockStore.savedMsg.Artifacts))
	}
	if mockStore.savedMsg.Artifacts[0].ID != "art-1" || mockStore.savedMsg.Artifacts[0].Name != "diagram.png" || mockStore.savedMsg.Artifacts[0].MimeType != "image/png" {
		t.Errorf("Unexpected Artifacts[0]: %+v", mockStore.savedMsg.Artifacts[0])
	}
	if mockStore.savedMsg.Artifacts[1].ID != "art-2" || mockStore.savedMsg.Artifacts[1].Name != "notes.md" || mockStore.savedMsg.Artifacts[1].MimeType != "text/markdown" {
		t.Errorf("Unexpected Artifacts[1]: %+v", mockStore.savedMsg.Artifacts[1])
	}
}

type mockChatProjectStore struct {
	db.ProjectStore
}

func (m *mockChatProjectStore) GetProject(ctx context.Context, organizationID string, id string) (*db.Project, error) {
	name := "test-project"
	baseBranch := "main"
	return &db.Project{
		ID:         id,
		Name:       &name,
		BaseBranch: &baseBranch,
	}, nil
}

type mockChatAppCredentialStore struct {
	db.AppCredentialStore
}

func (m *mockChatAppCredentialStore) GetCredential(ctx context.Context, organizationID string, userId, appId string) (*db.AppCredential, error) {
	if appId == "gemini" {
		return &db.AppCredential{
			IntegrationID: "gemini",
			Fields: []db.CredentialAttribute{
				{Name: "gemini_api_key", Value: "simulated_gemini_key"},
			},
		}, nil
	}
	return nil, nil
}

type mockChatMemoryArtifactService struct {
	files map[string][]byte
}

func (m *mockChatMemoryArtifactService) GetArtifact(ctx context.Context, organizationID string, userId string, workspaceID string, path string) ([]byte, error) {
	return nil, nil
}
func (m *mockChatMemoryArtifactService) WriteArtifact(ctx context.Context, organizationID string, userId string, workspaceID string, path string, data []byte) error {
	return nil
}
func (m *mockChatMemoryArtifactService) DeleteArtifact(ctx context.Context, organizationID string, userId string, workspaceID string, path string) error {
	return nil
}
func (m *mockChatMemoryArtifactService) ListArtifacts(ctx context.Context, organizationID string, userId string, workspaceID string, prefix string) ([]string, error) {
	return nil, nil
}
func (m *mockChatMemoryArtifactService) ListTopLevelFiles(ctx context.Context, organizationID string, userId string, workspaceID string) ([]string, error) {
	return nil, nil
}
func (m *mockChatMemoryArtifactService) SaveSessionArtifact(ctx context.Context, projectId, sessionId, filename string, r io.Reader) (string, string, int64, error) {
	return "local_file", "", 0, nil
}
func (m *mockChatMemoryArtifactService) ReadSessionArtifact(ctx context.Context, storageType string, storagePath string) ([]byte, error) {
	if data, exists := m.files[storagePath]; exists {
		return data, nil
	}
	return nil, errors.New("file not found: " + storagePath)
}

type mockChatArtifactStore struct {
	db.ArtifactStore
	artifacts []db.Artifact
}

func (m *mockChatArtifactStore) ListArtifactsForProject(ctx context.Context, organizationID string, projectId string) ([]db.Artifact, error) {
	return m.artifacts, nil
}

func TestUserMessageHandler_ProcessesAttachmentsAndDispatchesMultipart(t *testing.T) {
	db.ProjectStoreInstance = &mockChatProjectStore{}
	db.AppCredentialStoreInstance = &mockChatAppCredentialStore{}

	var receivedPayload map[string]interface{}
	payloadReceivedChan := make(chan struct{}, 1)

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && strings.Contains(r.URL.Path, "/sessions/") {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method == "POST" && r.URL.Path == "/api/run_sse" {
			var body map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&body)
			receivedPayload = body
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("data: {\"content\":{\"parts\":[{\"text\":\"response\"}]},\"turnComplete\":true}\n\n"))
			select {
			case payloadReceivedChan <- struct{}{}:
			default:
			}
			return
		}
		http.NotFound(w, r)
	}))
	defer mockServer.Close()

	cfg := &core.Configuration{
		AgentBaseURL:     mockServer.URL,
		AgentHTTPTimeout: 5 * time.Second,
	}

	artSvc := &mockChatMemoryArtifactService{
		files: map[string][]byte{
			"/artifacts/readme.txt": []byte("Hello from readme!"),
			"/artifacts/chart.png":  []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A},
		},
	}

	mockMsgStore := &testMessageStore{}
	agent.Store = mockMsgStore
	handler := NewChatHandler(mockMsgStore).
		WithArtifactService(artSvc).
		WithAppConfig(cfg)

	body := `{
		"sender": "User",
		"text": "Please analyze this doc and diagram",
		"isUser": true,
		"phase": "ANALYSIS",
		"sessionId": "sess-multipart",
		"artifacts": [
			{"id": "art-doc", "name": "readme.txt", "mimeType": "text/plain", "storageType": "local_file", "storagePath": "/artifacts/readme.txt"},
			{"id": "art-img", "name": "chart.png", "mimeType": "image/png", "storageType": "local_file", "storagePath": "/artifacts/chart.png"}
		]
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/proj1/messages?sessionId=sess-multipart", strings.NewReader(body))
	req.SetPathValue("id", "proj1")

	claims := &core.UserClaims{Subject: "user1", Token: "mock-token", OrganizationID: "org1"}
	req = req.WithContext(context.WithValue(req.Context(), core.UserClaimsKey, claims))
	req = req.WithContext(core.WithOrganizationID(req.Context(), "org1"))

	rr := httptest.NewRecorder()
	handler.UserMessageHandler(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("Expected status Created, got %d: %s", rr.Code, rr.Body.String())
	}

	select {
	case <-payloadReceivedChan:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for agent run_sse request")
	}

	newMsg, exists := receivedPayload["newMessage"].(map[string]interface{})
	if !exists {
		t.Fatal("expected newMessage in payload")
	}
	parts, ok := newMsg["parts"].([]interface{})
	if !ok || len(parts) != 2 {
		t.Fatalf("expected 2 parts (1 text with demarcation + 1 image), got %d", len(parts))
	}

	part0, _ := parts[0].(map[string]interface{})
	textVal, _ := part0["text"].(string)
	if !strings.Contains(textVal, "Please analyze this doc and diagram") {
		t.Errorf("expected prompt text in part 0, got %q", textVal)
	}
	if !strings.Contains(textVal, "[Attached File: readme.txt]\nHello from readme!") {
		t.Errorf("expected demarcation block in part 0, got %q", textVal)
	}

	part1, _ := parts[1].(map[string]interface{})
	inlineData, _ := part1["inlineData"].(map[string]interface{})
	if inlineData["mimeType"] != "image/png" {
		t.Errorf("expected image/png, got %v", inlineData["mimeType"])
	}
	if inlineData["data"] == "" {
		t.Errorf("expected non-empty base64 data in inlineData")
	}
}

func TestUserMessageHandler_ResolvesArtifactIDsFromStore(t *testing.T) {
	db.ProjectStoreInstance = &mockChatProjectStore{}
	db.AppCredentialStoreInstance = &mockChatAppCredentialStore{}

	var receivedPayload map[string]interface{}
	payloadReceivedChan := make(chan struct{}, 1)

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && strings.Contains(r.URL.Path, "/sessions/") {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method == "POST" && r.URL.Path == "/api/run_sse" {
			var body map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&body)
			receivedPayload = body
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("data: {\"content\":{\"parts\":[{\"text\":\"response\"}]},\"turnComplete\":true}\n\n"))
			select {
			case payloadReceivedChan <- struct{}{}:
			default:
			}
			return
		}
		http.NotFound(w, r)
	}))
	defer mockServer.Close()

	cfg := &core.Configuration{
		AgentBaseURL:     mockServer.URL,
		AgentHTTPTimeout: 5 * time.Second,
	}

	artSvc := &mockChatMemoryArtifactService{
		files: map[string][]byte{
			"/artifacts/stored.sql": []byte("SELECT * FROM table;"),
		},
	}

	artStore := &mockChatArtifactStore{
		artifacts: []db.Artifact{
			{
				ID:          "art-stored-id",
				Name:        "stored.sql",
				MimeType:    "application/sql",
				StorageType: "local_file",
				StoragePath: "/artifacts/stored.sql",
			},
		},
	}

	mockMsgStore := &testMessageStore{}
	agent.Store = mockMsgStore
	handler := NewChatHandler(mockMsgStore).
		WithArtifactService(artSvc).
		WithArtifactStore(artStore).
		WithAppConfig(cfg)

	body := `{
		"sender": "User",
		"text": "Review stored schema",
		"isUser": true,
		"phase": "PLANNING",
		"sessionId": "sess-stored",
		"artifactIds": ["art-stored-id"]
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/proj1/messages?sessionId=sess-stored", strings.NewReader(body))
	req.SetPathValue("id", "proj1")

	claims := &core.UserClaims{Subject: "user1", Token: "mock-token", OrganizationID: "org1"}
	req = req.WithContext(context.WithValue(req.Context(), core.UserClaimsKey, claims))
	req = req.WithContext(core.WithOrganizationID(req.Context(), "org1"))

	rr := httptest.NewRecorder()
	handler.UserMessageHandler(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("Expected status Created, got %d: %s", rr.Code, rr.Body.String())
	}

	select {
	case <-payloadReceivedChan:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for agent run_sse request")
	}

	newMsg, exists := receivedPayload["newMessage"].(map[string]interface{})
	if !exists {
		t.Fatal("expected newMessage in payload")
	}
	parts, ok := newMsg["parts"].([]interface{})
	if !ok || len(parts) != 1 {
		t.Fatalf("expected 1 text part, got %d", len(parts))
	}

	part0, _ := parts[0].(map[string]interface{})
	textVal, _ := part0["text"].(string)
	if !strings.Contains(textVal, "Review stored schema") {
		t.Errorf("expected prompt in part 0, got %q", textVal)
	}
	if !strings.Contains(textVal, "[Attached File: stored.sql]\nSELECT * FROM table;") {
		t.Errorf("expected demarcation block for stored artifact, got %q", textVal)
	}
}

