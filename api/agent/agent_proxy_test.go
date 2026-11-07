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
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"google.com/apf/api/core"
	"google.com/apf/api/db"
)

func init() {
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err == nil {
		pubBytes, err := x509.MarshalPKIXPublicKey(&privKey.PublicKey)
		if err == nil {
			pubPEM := pem.EncodeToMemory(&pem.Block{
				Type:  "PUBLIC KEY",
				Bytes: pubBytes,
			})
			os.Setenv("SECRET_ENCRYPTION_PUBLIC_KEY", string(pubPEM))
		}
	}
}

type mockProjectStore struct {
	db.ProjectStore
}

func (m *mockProjectStore) GetProject(ctx context.Context, organizationID string, id string) (*db.Project, error) {
	name := "mock-project-name"
	baseBranch := "main"
	return &db.Project{
		ID:         id,
		Name:       &name,
		BaseBranch: &baseBranch,
	}, nil
}

type mockUserProjectStore struct {
	db.UserProjectStore
	isReadOnly bool
}

func (m *mockUserProjectStore) GetProjectAppSetting(ctx context.Context, organizationID string, userId, projectId, appId string) (*db.UserProject, error) {
	return &db.UserProject{
		UserID:     userId,
		ProjectID:  projectId,
		AppID:      appId,
		Enabled:    true,
		IsReadOnly: m.isReadOnly,
	}, nil
}

type mockAppCredentialStore struct {
	db.AppCredentialStore
}

func (m *mockAppCredentialStore) GetCredential(ctx context.Context, organizationID string, userId, appId string) (*db.AppCredential, error) {
	if appId == "gemini" {
		return &db.AppCredential{
			IntegrationID: "gemini",
			Fields: []db.CredentialAttribute{
				{Name: "gemini_api_key", Value: "gemini_simulated_api_key_12345"},
			},
		}, nil
	}
	return &db.AppCredential{
		IntegrationID: appId,
		Fields: []db.CredentialAttribute{
			{Name: "pat", Value: "mock-pat-token"},
		},
	}, nil
}


func initTestMocks() {
	db.ProjectStoreInstance = &mockProjectStore{}
	db.UserProjectStoreInstance = &mockUserProjectStore{}
	db.AppCredentialStoreInstance = &mockAppCredentialStore{}
}

func TestAgentProxy_CreateSession_Success(t *testing.T) {
	initTestMocks()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &core.Configuration{
		AgentBaseURL: server.URL,
	}
	proxy := NewAgentProxy(cfg.AgentBaseURL, cfg)

	claims := &core.UserClaims{
		Subject: "test-user@example.com",
		Token:   "mock-token-123",
	}
	ctx := context.WithValue(context.Background(), core.UserClaimsKey, claims)
	ctx = context.WithValue(ctx, core.OrganizationIDKey, "default-organization")

	err := proxy.CreateSession(ctx, "APF Overwatch Agent", "session-456", "project-789", "test-user@example.com", "default-organization")
	if err != nil {
		t.Fatalf("expected no error creating session, got %v", err)
	}
}

func TestAgentProxy_CreateSession_AlreadyExists(t *testing.T) {
	initTestMocks()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("session session-456 already exists in db"))
	}))
	defer server.Close()

	cfg := &core.Configuration{
		AgentBaseURL: server.URL,
	}
	proxy := NewAgentProxy(cfg.AgentBaseURL, cfg)

	claims := &core.UserClaims{
		Subject: "user-123",
		Token:   "mock-token-123",
	}
	ctx := context.WithValue(context.Background(), core.UserClaimsKey, claims)
	ctx = context.WithValue(ctx, core.OrganizationIDKey, "default-organization")

	err := proxy.CreateSession(ctx, "my-app", "session-456", "project-789", "user-123", "default-organization")
	if err != nil {
		t.Fatalf("expected no error when session already exists, got %v", err)
	}
}

func TestAgentProxy_CreateSession_Error(t *testing.T) {
	initTestMocks()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal server error"))
	}))
	defer server.Close()

	cfg := &core.Configuration{
		AgentBaseURL: server.URL,
	}
	proxy := NewAgentProxy(cfg.AgentBaseURL, cfg)

	claims := &core.UserClaims{
		Subject: "test-user@example.com",
		Token:   "mock-token-123",
	}
	ctx := context.WithValue(context.Background(), core.UserClaimsKey, claims)
	ctx = context.WithValue(ctx, core.OrganizationIDKey, "default-organization")

	err := proxy.CreateSession(ctx, "APF Overwatch Agent", "session-456", "project-789", "test-user@example.com", "default-organization")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unexpected error creating session") {
		t.Errorf("expected error message to contain %q, got %q", "unexpected error creating session", err.Error())
	}
}

func TestAgentProxy_RunAgent_Success(t *testing.T) {
	initTestMocks()

	var receivedMethod string
	var receivedPath string
	var receivedRawQuery string
	var receivedContentType string
	var receivedPayload map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		receivedRawQuery = r.URL.RawQuery
		receivedContentType = r.Header.Get("Content-Type")

		bodyBytes, err := io.ReadAll(r.Body)
		if err == nil {
			json.Unmarshal(bodyBytes, &receivedPayload)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Mock agent response body"))
	}))
	defer server.Close()

	cfg := &core.Configuration{
		AgentBaseURL: server.URL,
	}
	proxy := NewAgentProxy(cfg.AgentBaseURL, cfg)

	claims := &core.UserClaims{
		Subject: "test-user@example.com",
		Token:   "mock-token-123",
	}
	ctx := context.WithValue(context.Background(), core.UserClaimsKey, claims)
	ctx = context.WithValue(ctx, core.OrganizationIDKey, "default-organization")

	resp, err := proxy.RunAgent(ctx, "APF Overwatch Agent", "session-456", "project-789", "test-user@example.com", "hello agent")
	if err != nil {
		t.Fatalf("expected no error in RunAgent, got %v", err)
	}

	if string(resp) != "Mock agent response body" {
		t.Errorf("expected response %q, got %q", "Mock agent response body", string(resp))
	}

	if receivedMethod != "POST" {
		t.Errorf("expected method POST, got %q", receivedMethod)
	}

	if receivedPath != "/api/run" {
		t.Errorf("expected path /api/run, got %q", receivedPath)
	}

	if receivedRawQuery != "" {
		t.Errorf("expected query to be empty, got %q", receivedRawQuery)
	}

	if receivedContentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", receivedContentType)
	}

	if receivedPayload["appName"] != "APF Overwatch Agent" {
		t.Errorf("expected appName %q, got %v", "APF Overwatch Agent", receivedPayload["appName"])
	}

	if receivedPayload["userId"] != "test-user@example.com" {
		t.Errorf("expected userId %q, got %v", "test-user@example.com", receivedPayload["userId"])
	}

	if receivedPayload["sessionId"] != "session-456" {
		t.Errorf("expected sessionId %q, got %v", "session-456", receivedPayload["sessionId"])
	}

	if receivedPayload["userToken"] != "mock-token-123" {
		t.Errorf("expected userToken %q, got %v", "mock-token-123", receivedPayload["userToken"])
	}

	// Check newMessage structure
	newMsg, exists := receivedPayload["newMessage"].(map[string]interface{})
	if !exists {
		t.Fatal("expected newMessage in payload")
	}
	if newMsg["role"] != "user" {
		t.Errorf("expected role user, got %v", newMsg["role"])
	}
	parts, ok := newMsg["parts"].([]interface{})
	if !ok || len(parts) != 1 {
		t.Fatal("expected parts array of size 1")
	}
	partObj, ok := parts[0].(map[string]interface{})
	if !ok {
		t.Fatal("expected part to be an object")
	}
	if partObj["text"] != "hello agent" {
		t.Errorf("expected text %q, got %v", "hello agent", partObj["text"])
	}
}

func TestAgentProxy_GeminiApiKey_MissingInDatabase_ReturnsHTTP400(t *testing.T) {
	initTestMocks()
	// Override mock credential store to return error for gemini
	emptyStore := &emptyMockAppCredentialStore{}
	db.AppCredentialStoreInstance = emptyStore

	proxy := NewAgentProxy("http://mock-agent", &core.Configuration{})
	claims := &core.UserClaims{Subject: "test-user@example.com", Token: "mock-token"}
	ctx := context.WithValue(context.Background(), core.UserClaimsKey, claims)
	ctx = context.WithValue(ctx, core.OrganizationIDKey, "default-organization")

	targetMap := make(map[string]interface{})
	err := proxy.populateAuthAndSecrets(ctx, targetMap, "project-123", "test-user@example.com")
	if err == nil {
		t.Fatal("expected error when Gemini API key is missing from database, got nil")
	}
	if !strings.Contains(err.Error(), "GEMINI_API_KEY_REQUIRED") {
		t.Errorf("expected error message to contain 'GEMINI_API_KEY_REQUIRED', got %v", err)
	}
}

type emptyMockAppCredentialStore struct {
	db.AppCredentialStore
}

func (m *emptyMockAppCredentialStore) GetCredential(ctx context.Context, organizationID string, userId, appId string) (*db.AppCredential, error) {
	return nil, nil
}

func TestAgentProxy_GeminiApiKey_ResolutionHierarchy(t *testing.T) {
	initTestMocks()

	claims := &core.UserClaims{Subject: "test-user@example.com", Token: "mock-token"}
	ctx := context.WithValue(context.Background(), core.UserClaimsKey, claims)
	ctx = context.WithValue(ctx, core.OrganizationIDKey, "default-organization")

	cred, err := db.ResolveAppCredential(ctx, "default-organization", nil, nil, "test-user@example.com", "project-123", "gemini")
	if err != nil {
		t.Fatalf("expected no error resolving gemini credential, got %v", err)
	}
	if cred == nil {
		t.Fatal("expected non-nil credential resolved")
	}
	val := db.GetFieldValue(cred, "gemini_api_key")
	if val != "gemini_simulated_api_key_12345" {
		t.Errorf("expected resolved key 'gemini_simulated_api_key_12345', got %q", val)
	}
}

func TestAgentProxy_GeminiApiKey_EncryptedJWEPayload(t *testing.T) {
	initTestMocks()

	proxy := NewAgentProxy("http://mock-agent", &core.Configuration{})
	claims := &core.UserClaims{Subject: "test-user@example.com", Token: "mock-token"}
	ctx := context.WithValue(context.Background(), core.UserClaimsKey, claims)
	ctx = context.WithValue(ctx, core.OrganizationIDKey, "default-organization")

	targetMap := make(map[string]interface{})
	err := proxy.populateAuthAndSecrets(ctx, targetMap, "project-123", "test-user@example.com")
	if err != nil {
		t.Fatalf("expected no error populating auth and secrets, got %v", err)
	}
	if targetMap["userSecretsEncrypted"] == nil {
		t.Error("expected userSecretsEncrypted in targetMap")
	}
}

func TestAgentProxy_RunAgentStreamingWithParts_Success(t *testing.T) {
	initTestMocks()

	var receivedPath string
	var receivedPayload map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		bodyBytes, err := io.ReadAll(r.Body)
		if err == nil {
			json.Unmarshal(bodyBytes, &receivedPayload)
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("data: {\"content\":{\"parts\":[{\"text\":\"chunk 1\"}]},\"partial\":true}\n\n"))
	}))
	defer server.Close()

	cfg := &core.Configuration{
		AgentBaseURL: server.URL,
	}
	proxy := NewAgentProxy(cfg.AgentBaseURL, cfg)

	claims := &core.UserClaims{
		Subject: "test-user@example.com",
		Token:   "mock-token-123",
	}
	ctx := context.WithValue(context.Background(), core.UserClaimsKey, claims)
	ctx = context.WithValue(ctx, core.OrganizationIDKey, "default-organization")

	parts := []ProxyMessagePart{
		{Text: "Explain diagram\n\n[Attached File: notes.md]\n# Notes"},
		{InlineData: &ProxyInlineData{MimeType: "image/png", Data: "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII="}},
	}

	stream, err := proxy.RunAgentStreamingWithParts(ctx, "APF Help Agent", "session-456", "project-789", "test-user@example.com", parts)
	if err != nil {
		t.Fatalf("expected no error in RunAgentStreamingWithParts, got %v", err)
	}
	defer stream.Close()

	if receivedPath != "/api/run_sse" {
		t.Errorf("expected path /api/run_sse, got %q", receivedPath)
	}

	newMsg, exists := receivedPayload["newMessage"].(map[string]interface{})
	if !exists {
		t.Fatal("expected newMessage in payload")
	}
	receivedParts, ok := newMsg["parts"].([]interface{})
	if !ok || len(receivedParts) != 2 {
		t.Fatalf("expected 2 parts in newMessage, got %d", len(receivedParts))
	}

	part0, _ := receivedParts[0].(map[string]interface{})
	if part0["text"] != "Explain diagram\n\n[Attached File: notes.md]\n# Notes" {
		t.Errorf("unexpected text in part 0: %v", part0["text"])
	}

	part1, _ := receivedParts[1].(map[string]interface{})
	inlineData, _ := part1["inlineData"].(map[string]interface{})
	if inlineData["mimeType"] != "image/png" {
		t.Errorf("unexpected mimeType in inlineData: %v", inlineData["mimeType"])
	}
	if inlineData["data"] != "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=" {
		t.Errorf("unexpected data in inlineData: %v", inlineData["data"])
	}

	body, err := io.ReadAll(stream)
	if err != nil {
		t.Fatalf("failed to read stream body: %v", err)
	}
	if !strings.Contains(string(body), "chunk 1") {
		t.Errorf("expected stream body to contain 'chunk 1', got %q", string(body))
	}
}



