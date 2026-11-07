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

package integrations

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"google.com/apf/api/core"
	"google.com/apf/api/db"
)

// mockStore is an in-memory implementation of db.IntegrationStore
type mockStore struct {
	integrations map[string]*db.Integration
}

func newMockStore() *mockStore {
	return &mockStore{
		integrations: make(map[string]*db.Integration),
	}
}

func (m *mockStore) GetIntegration(ctx context.Context, userId, name string) (*db.Integration, error) {
	key := userId + "_" + name
	if integration, ok := m.integrations[key]; ok {
		// Return a copy to avoid side effects from mutation in tests
		copied := &db.Integration{
			ID:       integration.ID,
			UserID:   integration.UserID,
			Name:     integration.Name,
			Settings: make(map[string]interface{}),
		}
		for k, v := range integration.Settings {
			copied.Settings[k] = v
		}
		return copied, nil
	}
	return nil, nil
}

func (m *mockStore) SaveIntegration(ctx context.Context, integration *db.Integration) error {
	key := integration.UserID + "_" + integration.Name
	m.integrations[key] = integration
	return nil
}

func (m *mockStore) ListUserIntegrations(ctx context.Context, userId string) ([]db.Integration, error) {
	return nil, nil
}

func (m *mockStore) DeleteIntegration(ctx context.Context, userId, name string) error {
	key := userId + "_" + name
	delete(m.integrations, key)
	return nil
}

func TestGitHubCallbackHandler_StoresAcquiredAt(t *testing.T) {
	// Backup and restore globals
	origStore := Store
	origTokenURL := githubTokenExchangeURL
	origUserURL := githubUserAPIURL
	defer func() {
		Store = origStore
		githubTokenExchangeURL = origTokenURL
		githubUserAPIURL = origUserURL
	}()

	mStore := newMockStore()
	Store = mStore

	// Start a local test server to mock GitHub API
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login/oauth/access_token" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"access_token":  "test-token-123",
				"refresh_token": "test-refresh-123",
			})
			return
		}
		if r.URL.Path == "/user" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"login": "test-user",
			})
			return
		}
		http.Error(w, "Not Found", http.StatusNotFound)
	}))
	defer ts.Close()

	githubTokenExchangeURL = ts.URL + "/login/oauth/access_token"
	githubUserAPIURL = ts.URL + "/user"

	// Mock core.Config for ClientSecret detection
	t.Setenv("GITHUB_CLIENT_ID", "client-id")
	t.Setenv("GITHUB_CLIENT_SECRET", "client-secret")

	// Call the handler
	req := httptest.NewRequest("GET", "/api/v1/integrations/github/callback?code=valid-code&state=default-organization", nil)
	w := httptest.NewRecorder()

	// Inject context claims
	claims := &core.UserClaims{Subject: "user-1"}
	ctx := context.WithValue(req.Context(), core.UserClaimsKey, claims)
	ctx = context.WithValue(ctx, core.OrganizationIDKey, "default-organization")
	req = req.WithContext(ctx)

	GitHubCallbackHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	// Verify integration in store has acquiredAt
	integration, err := mStore.GetIntegration(testCtx(), "user-1", "github")
	if err != nil {
		t.Fatalf("Failed to get integration: %v", err)
	}
	if integration == nil {
		t.Fatal("Integration was not saved")
	}

	acquiredAtStr, ok := integration.Settings["acquiredAt"].(string)
	if !ok || acquiredAtStr == "" {
		t.Fatal("Integration Settings['acquiredAt'] is missing or empty")
	}

	acquiredTime, err := time.Parse(time.RFC3339, acquiredAtStr)
	if err != nil {
		t.Fatalf("Failed to parse acquiredAt: %v", err)
	}

	if time.Since(acquiredTime) > 5*time.Second {
		t.Errorf("acquiredTime %v is too old", acquiredTime)
	}
}

func TestGitHubLinkPATHandler_StoresAcquiredAt(t *testing.T) {
	origStore := Store
	origUserURL := githubUserAPIURL
	defer func() {
		Store = origStore
		githubUserAPIURL = origUserURL
	}()

	mStore := newMockStore()
	Store = mStore

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/user" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"login": "pat-user",
			})
			return
		}
	}))
	defer ts.Close()
	githubUserAPIURL = ts.URL + "/user"

	reqBody, _ := json.Marshal(PATRequest{Token: "ghp_personal_token"})
	req := httptest.NewRequest("POST", "/api/v1/integrations/github/pat", bytes.NewReader(reqBody))
	w := httptest.NewRecorder()

	claims := &core.UserClaims{Subject: "user-pat"}
	ctx := context.WithValue(req.Context(), core.UserClaimsKey, claims)
	ctx = context.WithValue(ctx, core.OrganizationIDKey, "default-organization")
	req = req.WithContext(ctx)

	GitHubLinkPATHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	integration, _ := mStore.GetIntegration(testCtx(), "user-pat", "github")
	if integration == nil {
		t.Fatal("PAT integration was not saved")
	}

	acquiredAtStr, _ := integration.Settings["acquiredAt"].(string)
	if acquiredAtStr == "" {
		t.Fatal("PAT acquiredAt is missing")
	}
}

func TestGetOrRefreshToken_YoungToken_NoRefresh(t *testing.T) {
	origStore := Store
	defer func() { Store = origStore }()

	mStore := newMockStore()
	Store = mStore

	// Set up an integration with a young access token (10 mins old)
	integration := &db.Integration{
		UserID: "user-young",
		Name:   "github",
		Settings: map[string]interface{}{
			"accessToken":  "young-token-123",
			"refreshToken": "refresh-token-123",
			"acquiredAt":   time.Now().Add(-10 * time.Minute).Format(time.RFC3339),
		},
	}
	mStore.SaveIntegration(testCtx(), integration)

	token, err := GetOrRefreshToken(testCtx(), "user-young", false)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if token != "young-token-123" {
		t.Errorf("Expected token 'young-token-123', got '%s'", token)
	}
}

func TestGetOrRefreshToken_OldToken_TriggersRefresh(t *testing.T) {
	origStore := Store
	origTokenURL := githubTokenExchangeURL
	defer func() {
		Store = origStore
		githubTokenExchangeURL = origTokenURL
	}()

	mStore := newMockStore()
	Store = mStore

	// Set up an integration with an old access token (2 hours old)
	integration := &db.Integration{
		UserID: "user-old",
		Name:   "github",
		Settings: map[string]interface{}{
			"accessToken":  "old-token-123",
			"refreshToken": "refresh-token-123",
			"acquiredAt":   time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
		},
	}
	mStore.SaveIntegration(testCtx(), integration)

	// Mock GitHub Token exchange server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]string
		json.NewDecoder(r.Body).Decode(&req)

		if req["grant_type"] != "refresh_token" || req["refresh_token"] != "refresh-token-123" {
			http.Error(w, "Invalid refresh request", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"access_token":  "brand-new-access-token",
			"refresh_token": "rotated-refresh-token",
		})
	}))
	defer ts.Close()
	githubTokenExchangeURL = ts.URL

	t.Setenv("GITHUB_CLIENT_ID", "client-id")
	t.Setenv("GITHUB_CLIENT_SECRET", "client-secret")

	token, err := GetOrRefreshToken(testCtx(), "user-old", false)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if token != "brand-new-access-token" {
		t.Errorf("Expected refreshed token 'brand-new-access-token', got '%s'", token)
	}

	// Verify store got updated with new credentials and acquiredAt
	updated, _ := mStore.GetIntegration(testCtx(), "user-old", "github")
	if updated.Settings["accessToken"] != "brand-new-access-token" {
		t.Errorf("Expected updated accessToken in db, got %v", updated.Settings["accessToken"])
	}
	if updated.Settings["refreshToken"] != "rotated-refresh-token" {
		t.Errorf("Expected rotated refresh token in db, got %v", updated.Settings["refreshToken"])
	}

	acquiredAtStr, _ := updated.Settings["acquiredAt"].(string)
	acquiredTime, _ := time.Parse(time.RFC3339, acquiredAtStr)
	if time.Since(acquiredTime) > 5*time.Second {
		t.Errorf("New acquiredTime %v should be close to now", acquiredTime)
	}
}

func TestGetOrRefreshToken_PAT_NeverRefreshes(t *testing.T) {
	origStore := Store
	defer func() { Store = origStore }()

	mStore := newMockStore()
	Store = mStore

	// PAT has NO refreshToken in Settings
	integration := &db.Integration{
		UserID: "user-pat",
		Name:   "github",
		Settings: map[string]interface{}{
			"accessToken": "ghp_personal_token_123",
			"acquiredAt":  time.Now().Add(-5 * time.Hour).Format(time.RFC3339),
		},
	}
	mStore.SaveIntegration(testCtx(), integration)

	token, err := GetOrRefreshToken(testCtx(), "user-pat", false)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if token != "ghp_personal_token_123" {
		t.Errorf("Expected direct token return, got %s", token)
	}
}

func TestGetOrRefreshToken_Simulated_RefreshesSuccessfully(t *testing.T) {
	origStore := Store
	defer func() { Store = origStore }()

	mStore := newMockStore()
	Store = mStore

	// Set up simulated integration
	integration := &db.Integration{
		UserID: "user-simulated",
		Name:   "github",
		Settings: map[string]interface{}{
			"accessToken":  "ghu_simulated_token",
			"refreshToken": "ghr_simulated_token",
			"acquiredAt":   time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
		},
	}
	mStore.SaveIntegration(testCtx(), integration)

	token, err := GetOrRefreshToken(testCtx(), "user-simulated", false)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !strings.HasPrefix(token, "ghu_simulated_access_token_refreshed_") {
		t.Errorf("Expected refreshed simulated token prefix, got '%s'", token)
	}

	updated, _ := mStore.GetIntegration(testCtx(), "user-simulated", "github")
	refreshedAccessToken := updated.Settings["accessToken"].(string)
	if !strings.HasPrefix(refreshedAccessToken, "ghu_simulated_access_token_refreshed_") {
		t.Errorf("Expected saved accessToken to be refreshed, got %s", refreshedAccessToken)
	}
}

func TestGitHubReposHandler_AutoRetryOn401(t *testing.T) {
	origStore := Store
	origTokenURL := githubTokenExchangeURL
	origReposURL := githubReposAPIURL
	defer func() {
		Store = origStore
		githubTokenExchangeURL = origTokenURL
		githubReposAPIURL = origReposURL
	}()

	mStore := newMockStore()
	Store = mStore

	integration := &db.Integration{
		UserID: "user-401",
		Name:   "github",
		Settings: map[string]interface{}{
			"accessToken":  "stale-token",
			"refreshToken": "refresh-token-123",
			"acquiredAt":   time.Now().Add(-10 * time.Minute).Format(time.RFC3339), // young token so proactive refresh is skipped
		},
	}
	mStore.SaveIntegration(testCtx(), integration)

	// Keep track of API requests to verify sequence
	var refreshCalled bool
	var repoCalls int

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos" {
			repoCalls++
			authHeader := r.Header.Get("Authorization")
			if authHeader == "Bearer stale-token" {
				// First call returns 401 Unauthorized
				w.WriteHeader(http.StatusUnauthorized)
				return
			} else if authHeader == "Bearer fresh-token-abc" {
				// Second call with refreshed token succeeds
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode([]map[string]string{
					{"name": "my-repo", "html_url": "https://github.com/user/my-repo"},
				})
				return
			}
			w.WriteHeader(http.StatusForbidden)
			return
		}

		if r.URL.Path == "/refresh" {
			refreshCalled = true
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"access_token":  "fresh-token-abc",
				"refresh_token": "new-refresh-token-xyz",
			})
			return
		}
	}))
	defer ts.Close()

	githubReposAPIURL = ts.URL + "/repos"
	githubTokenExchangeURL = ts.URL + "/refresh"

	t.Setenv("GITHUB_CLIENT_ID", "client-id")
	t.Setenv("GITHUB_CLIENT_SECRET", "client-secret")

	req := httptest.NewRequest("GET", "/api/v1/integrations/github/repos", nil)
	w := httptest.NewRecorder()

	claims := &core.UserClaims{Subject: "user-401"}
	ctx := context.WithValue(req.Context(), core.UserClaimsKey, claims)
	ctx = context.WithValue(ctx, core.OrganizationIDKey, "default-organization")
	req = req.WithContext(ctx)

	GitHubReposHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	if repoCalls != 2 {
		t.Errorf("Expected 2 repo API calls, got %d", repoCalls)
	}

	if !refreshCalled {
		t.Error("Expected token refresh to have been called")
	}

	// Verify updated database state
	updated, _ := mStore.GetIntegration(testCtx(), "user-401", "github")
	if updated.Settings["accessToken"] != "fresh-token-abc" {
		t.Errorf("Expected token to be fresh-token-abc, got %v", updated.Settings["accessToken"])
	}
}

func TestResolveGitHubToken_FallbackToDevRepoAccessToken(t *testing.T) {
	origStore := Store
	defer func() { Store = origStore }()

	mStore := newMockStore()
	Store = mStore

	// Set DEV_REPO_ACCESS_TOKEN environment variable
	t.Setenv("DEV_REPO_ACCESS_TOKEN", "mock-dev-token-999")

	// Ensure database has no integration for user
	// Ensure ResolveGitHubToken falls back to DEV_REPO_ACCESS_TOKEN
	resolved, err := ResolveGitHubToken(testCtx(), "user-no-creds", "proj-abc")
	if err != nil {
		t.Fatalf("ResolveGitHubToken failed: %v", err)
	}

	if resolved != "mock-dev-token-999" {
		t.Errorf("Expected token to resolve to 'mock-dev-token-999', got %q", resolved)
	}
}

func TestValidateGitHubPermissionsHandler_BaseBranchCheck(t *testing.T) {
	// Test simulated token with existing base branch
	reqBody1, _ := json.Marshal(ValidatePermissionsRequest{
		RepoURL:    "https://github.com/test-owner/test-repo",
		Token:      "simulated_token",
		BaseBranch: "main",
	})
	req1 := httptest.NewRequest("POST", "/api/v1/integrations/github/validate-permissions", bytes.NewReader(reqBody1))
	claims := &core.UserClaims{Subject: "user-test"}
	ctx1 := context.WithValue(req1.Context(), core.UserClaimsKey, claims)
	req1 = req1.WithContext(ctx1)
	w1 := httptest.NewRecorder()

	ValidateGitHubPermissionsHandler(w1, req1)

	if w1.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d. Body: %s", w1.Code, w1.Body.String())
	}

	var res1 ValidatePermissionsResponse
	json.Unmarshal(w1.Body.Bytes(), &res1)

	if !res1.BaseBranchExists {
		t.Errorf("Expected BaseBranchExists to be true for 'main', got false")
	}

	// Test simulated token with non-existent base branch
	reqBody2, _ := json.Marshal(ValidatePermissionsRequest{
		RepoURL:    "https://github.com/test-owner/test-repo",
		Token:      "simulated_token",
		BaseBranch: "non-existent-branch",
	})
	req2 := httptest.NewRequest("POST", "/api/v1/integrations/github/validate-permissions", bytes.NewReader(reqBody2))
	ctx2 := context.WithValue(req2.Context(), core.UserClaimsKey, claims)
	req2 = req2.WithContext(ctx2)
	w2 := httptest.NewRecorder()

	ValidateGitHubPermissionsHandler(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d. Body: %s", w2.Code, w2.Body.String())
	}

	var res2 ValidatePermissionsResponse
	json.Unmarshal(w2.Body.Bytes(), &res2)

	if res2.BaseBranchExists {
		t.Errorf("Expected BaseBranchExists to be false for 'non-existent-branch', got true")
	}
}

func TestGitHubProfileHandler(t *testing.T) {
	origStore := Store
	defer func() { Store = origStore }()

	mStore := newMockStore()
	Store = mStore

	req := httptest.NewRequest("GET", "/api/v1/integrations/github/profile?token=ghu_simulated_test_token", nil)
	claims := &core.UserClaims{Subject: "user-test"}
	ctx := context.WithValue(req.Context(), core.UserClaimsKey, claims)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	GitHubProfileHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var res map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if res["username"] != "octocat" {
		t.Errorf("Expected username 'octocat', got %v", res["username"])
	}
	if res["email"] != "octocat@github.com" {
		t.Errorf("Expected email 'octocat@github.com', got %v", res["email"])
	}
}

func testCtx() context.Context {
	return context.WithValue(context.Background(), core.OrganizationIDKey, "default-organization")
}
