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
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"google.com/apf/api/core"
	"google.com/apf/api/db"
)

func init() {
	if os.Getenv("SECRET_ENCRYPTION_PUBLIC_KEY") == "" {
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
}

type mockProjectStoreForRoutes struct {
	db.ProjectStore
}

func (m *mockProjectStoreForRoutes) ListProjectsByUser(ctx context.Context, organizationID, userId string) ([]db.Project, []db.Project, error) {
	return []db.Project{}, []db.Project{}, nil
}

func (m *mockProjectStoreForRoutes) GetProject(ctx context.Context, organizationID string, id string) (*db.Project, error) {
	if id == "proj1" {
		owner := "user1"
		return &db.Project{
			ID:      "proj1",
			OwnerId: &owner,
		}, nil
	}
	return nil, fmt.Errorf("project not found")
}

type testRouteAuthenticator struct {
	claimsMap map[string]*core.UserClaims
}

func (t *testRouteAuthenticator) VerifyToken(token string) (*core.UserClaims, error) {
	if claims, ok := t.claimsMap[token]; ok {
		return claims, nil
	}
	return nil, fmt.Errorf("invalid token")
}

func (t *testRouteAuthenticator) GenerateMockToken(username string) (string, error) {
	return username, nil
}

func (t *testRouteAuthenticator) CreateUser(ctx context.Context, orgID string, email string, password string, displayName string) (string, error) {
	return "simulated-uid-" + email, nil
}

func (t *testRouteAuthenticator) SetUserClaims(ctx context.Context, orgID string, uid string, claims map[string]interface{}) error {
	return nil
}

func TestRoutesMiddlewareIntegration(t *testing.T) {
	// Backup global state
	oldStore := store
	oldAuth := globalAuthenticator
	oldUserProjStore := userProjectStore
	oldUserStore := userStore
	oldConfig := appConfig
	t.Cleanup(func() {
		store = oldStore
		globalAuthenticator = oldAuth
		userProjectStore = oldUserProjStore
		userStore = oldUserStore
		appConfig = oldConfig
	})

	appConfig = &core.Configuration{
		BypassIAPValidation: true,
		Environment:         "test",
	}

	// Initialize global stores and authenticator
	store = &mockProjectStoreForRoutes{}
	userProjectStore = &mockUserProjectStore{}
	userStore = &mockUserStore{}

	auth := &testRouteAuthenticator{
		claimsMap: make(map[string]*core.UserClaims),
	}
	globalAuthenticator = auth

	// Setup claims for tests
	auth.claimsMap["token-org1-scoped"] = &core.UserClaims{
		Subject:        "user1",
		OrganizationID: "org1",
	}
	auth.claimsMap["token-org-admin"] = &core.UserClaims{
		Subject:        "admin1",
		OrganizationID: "org1",
		Roles:          []string{string(core.RoleOrgAdmin)},
	}
	auth.claimsMap["token-super-admin"] = &core.UserClaims{
		Subject:        "superadmin1",
		OrganizationID: "org1",
		IsSuperAdmin:   true,
	}
	auth.claimsMap["token-primary-allowed"] = &core.UserClaims{
		Subject:       "user1",
		Organizations: []string{"org1", "org2"},
	}

	mux := http.NewServeMux()
	chatHandler := &ChatHandler{}
	sessionHandler := &SessionHandler{}
	adminHandler := &AdminHandler{
		UserStore:         userStore,
		OrganizationStore: nil,
	}
	authHandler := NewAuthHandler(
		nil, // provisioner
		auth, // authenticator
		nil, // validator
		userStore,
		nil, // orgStore
		appConfig,
	)
	featureHandler := NewFeatureHandler(db.NewMockFeatureStore())
	feedbackHandler := NewFeedbackHandler(db.NewMockFeedbackStore())
	telemetryHandler := NewTelemetryHandler(nil, nil, nil)
	registerRoutes(mux, chatHandler, sessionHandler, adminHandler, authHandler, featureHandler, feedbackHandler, telemetryHandler)

	tests := []struct {
		name           string
		method         string
		path           string
		token          string
		orgHeader      string
		expectedStatus int
	}{
		{
			name:           "Unauthenticated -> 401",
			method:         "GET",
			path:           "/api/v1/projects",
			token:          "",
			orgHeader:      "org1",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Org-scoped token, matching header -> 200",
			method:         "GET",
			path:           "/api/v1/projects",
			token:          "token-org1-scoped",
			orgHeader:      "org1",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Org-scoped token, mismatched header -> 403",
			method:         "GET",
			path:           "/api/v1/projects",
			token:          "token-org1-scoped",
			orgHeader:      "org2",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Primary token, allowed org header -> Forbidden (must select tenant first)",
			method:         "GET",
			path:           "/api/v1/projects",
			token:          "token-primary-allowed",
			orgHeader:      "org2",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Primary token, disallowed org header -> 403",
			method:         "GET",
			path:           "/api/v1/projects",
			token:          "token-primary-allowed",
			orgHeader:      "org3",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Primary token, missing header -> 403",
			method:         "GET",
			path:           "/api/v1/projects",
			token:          "token-primary-allowed",
			orgHeader:      "",
			expectedStatus: http.StatusForbidden,
		},
		// Parameterized route & project access tests
		{
			name:           "Project access - Success with matching org",
			method:         "GET",
			path:           "/api/v1/projects/proj1",
			token:          "token-org1-scoped",
			orgHeader:      "org1",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Project access - Forbidden with mismatched org",
			method:         "GET",
			path:           "/api/v1/projects/proj1",
			token:          "token-org1-scoped",
			orgHeader:      "org2",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Project access - Primary token -> Forbidden (must select tenant first)",
			method:         "GET",
			path:           "/api/v1/projects/proj1",
			token:          "token-primary-allowed",
			orgHeader:      "org1",
			expectedStatus: http.StatusForbidden,
		},
		// User preferences tests (verifies wrapping of user endpoints)
		{
			name:           "User preferences - Success with matching org",
			method:         "GET",
			path:           "/api/v1/users/user1/preferences",
			token:          "token-org1-scoped",
			orgHeader:      "org1",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "User preferences - Forbidden with mismatched org",
			method:         "GET",
			path:           "/api/v1/users/user1/preferences",
			token:          "token-org1-scoped",
			orgHeader:      "org2",
			expectedStatus: http.StatusForbidden,
		},
		// Telemetry endpoint tests (verifies route registration & auth middleware wrapping)
		{
			name:           "Telemetry usage - Unauthorized without token",
			method:         "GET",
			path:           "/api/v1/telemetry/usage",
			token:          "",
			orgHeader:      "org1",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Telemetry usage - Database unavailable with valid token",
			method:         "GET",
			path:           "/api/v1/telemetry/usage?organizationId=org1",
			token:          "token-super-admin",
			orgHeader:      "org1",
			expectedStatus: http.StatusServiceUnavailable,
		},
		// Format validation tests
		{
			name:           "Malformed org header -> 400",
			method:         "GET",
			path:           "/api/v1/projects",
			token:          "token-org1-scoped",
			orgHeader:      "org1;invalid",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			if tc.token != "" {
				req.Header.Set("Authorization", "Bearer "+tc.token)
			}
			if tc.orgHeader != "" {
				req.Header.Set(core.OrganizationIDHeader, tc.orgHeader)
			}

			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)

			if rr.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tc.expectedStatus, rr.Code, rr.Body.String())
			}
		})
	}
}
