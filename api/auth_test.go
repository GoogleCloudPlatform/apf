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
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"google.golang.org/api/idtoken"

	"google.com/apf/api/core"
	"google.com/apf/api/db"
)

func TestMockAuthenticator_VerifyToken(t *testing.T) {
	auth := &MockAuthenticator{}

	tests := []struct {
		name                   string
		token                  string
		expectedUsername       string
		expectedOrganizationID string
		expectedIsSuperAdmin   bool
		expectedErr            bool
	}{
		{
			name:                   "Valid token with organization",
			token:                  "simulated-jwt-token-for-user1-organization-organizationA",
			expectedUsername:       "user1",
			expectedOrganizationID: "organizationA",
			expectedIsSuperAdmin:   false,
			expectedErr:            false,
		},
		{
			name:                   "Valid token without organization",
			token:                  "simulated-jwt-token-for-user1",
			expectedUsername:       "user1",
			expectedOrganizationID: "",
			expectedIsSuperAdmin:   false,
			expectedErr:            false,
		},
		{
			name:                   "Admin token with organization",
			token:                  "simulated-jwt-token-for-admin-organization-organizationB",
			expectedUsername:       "admin",
			expectedOrganizationID: "organizationB",
			expectedIsSuperAdmin:   false,
			expectedErr:            false,
		},
		{
			name:                   "Super Admin token without organization",
			token:                  "simulated-jwt-token-for-super-admin",
			expectedUsername:       "super-admin",
			expectedOrganizationID: "",
			expectedIsSuperAdmin:   true,
			expectedErr:            false,
		},
		{
			name:                   "Invalid token format",
			token:                  "invalid-token",
			expectedUsername:       "",
			expectedOrganizationID: "",
			expectedIsSuperAdmin:   false,
			expectedErr:            true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			claims, err := auth.VerifyToken(tc.token)
			if (err != nil) != tc.expectedErr {
				t.Fatalf("expected error: %v, got: %v", tc.expectedErr, err)
			}
			if tc.expectedErr {
				return
			}
			if claims.Username != tc.expectedUsername {
				t.Errorf("expected username: %q, got: %q", tc.expectedUsername, claims.Username)
			}
			if claims.OrganizationID != tc.expectedOrganizationID {
				t.Errorf("expected organization ID: %q, got: %q", tc.expectedOrganizationID, claims.OrganizationID)
			}
			if claims.IsSuperAdmin != tc.expectedIsSuperAdmin {
				t.Errorf("expected isSuperAdmin: %t, got: %t", tc.expectedIsSuperAdmin, claims.IsSuperAdmin)
			}
		})
	}
}

func TestFirebaseAuthAuthenticator_VerifyToken_Simulated(t *testing.T) {
	oldConfig := appConfig
	defer func() { appConfig = oldConfig }()

	// We don't initialize Client since we are testing simulated paths
	auth := &FirebaseAuthAuthenticator{Client: nil}

	tests := []struct {
		name                   string
		token                  string
		expectedUsername       string
		expectedOrganizationID string
		expectedIsSuperAdmin   bool
		expectedErr            bool
	}{
		{
			name:                   "Simulated admin with organization",
			token:                  "simulated-jwt-token-for-admin-organization-organizationA",
			expectedUsername:       "admin",
			expectedOrganizationID: "organizationA",
			expectedIsSuperAdmin:   false,
			expectedErr:            false,
		},
		{
			name:                   "Simulated user1 with organization",
			token:                  "simulated-jwt-token-for-user1-organization-organizationB",
			expectedUsername:       "user1",
			expectedOrganizationID: "organizationB",
			expectedIsSuperAdmin:   false,
			expectedErr:            false,
		},
		{
			name:                   "Simulated arbitrary user with organization",
			token:                  "simulated-jwt-token-for-someguy-organization-organizationC",
			expectedUsername:       "someguy",
			expectedOrganizationID: "organizationC",
			expectedIsSuperAdmin:   false,
			expectedErr:            false,
		},
		{
			name:                   "Simulated user without organization",
			token:                  "simulated-jwt-token-for-user2",
			expectedUsername:       "user2",
			expectedOrganizationID: "",
			expectedIsSuperAdmin:   false,
			expectedErr:            false,
		},
		{
			name:                   "Simulated super-admin without organization",
			token:                  "simulated-jwt-token-for-super-admin",
			expectedUsername:       "super-admin",
			expectedOrganizationID: "",
			expectedIsSuperAdmin:   true,
			expectedErr:            false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Initialize appConfig with dummy values to avoid nil pointer if needed,
			// but VerifyToken only uses appConfig.SimulatedUserID if username is admin.
			// Let's make sure it doesn't crash.
			if appConfig == nil {
				appConfig = &core.Configuration{
					SimulatedUserID:     "admin-uid",
					BypassIAPValidation: true,
				}
			}

			claims, err := auth.VerifyToken(tc.token)
			if (err != nil) != tc.expectedErr {
				t.Fatalf("expected error: %v, got: %v", tc.expectedErr, err)
			}
			if tc.expectedErr {
				return
			}
			if claims.Username != tc.expectedUsername {
				t.Errorf("expected username: %q, got: %q", tc.expectedUsername, claims.Username)
			}
			if claims.OrganizationID != tc.expectedOrganizationID {
				t.Errorf("expected organization ID: %q, got: %q", tc.expectedOrganizationID, claims.OrganizationID)
			}
			if claims.IsSuperAdmin != tc.expectedIsSuperAdmin {
				t.Errorf("expected isSuperAdmin: %t, got: %t", tc.expectedIsSuperAdmin, claims.IsSuperAdmin)
			}
		})
	}
}
func TestOrgSessionTokens(t *testing.T) {
	secret := "test-secret-key-12345"
	subject := "user@example.com"
	orgID := "org-test"
	roles := []string{"viewer", "writer"}
	permissions := []string{"read:data", "write:data"}

	// 1. Test Mint and Verify
	token, err := MintOrgSessionToken(subject, orgID, roles, permissions, false, secret)
	if err != nil {
		t.Fatalf("failed to mint org session token: %v", err)
	}

	claims, err := VerifyOrgSessionToken(token, secret)
	if err != nil {
		t.Fatalf("failed to verify org session token: %v", err)
	}

	if claims.Subject != subject {
		t.Errorf("expected subject: %q, got: %q", subject, claims.Subject)
	}
	if claims.OrgID != orgID {
		t.Errorf("expected org ID: %q, got: %q", orgID, claims.OrgID)
	}
	if len(claims.Roles) != 2 || claims.Roles[0] != "viewer" || claims.Roles[1] != "writer" {
		t.Errorf("expected roles: %v, got: %v", roles, claims.Roles)
	}
	if len(claims.Permissions) != 2 || claims.Permissions[0] != "read:data" || claims.Permissions[1] != "write:data" {
		t.Errorf("expected permissions: %v, got: %v", permissions, claims.Permissions)
	}

	// 2. Test Invalid Secret
	_, err = VerifyOrgSessionToken(token, "wrong-secret")
	if err == nil {
		t.Error("expected error when verifying with wrong secret, got nil")
	}

	// 3. Test Invalid Token Format
	_, err = VerifyOrgSessionToken("garbage-token", secret)
	if err == nil {
		t.Error("expected error when verifying invalid token format, got nil")
	}
}

func TestMockTokenValidator(t *testing.T) {
	mockVal := &MockTokenValidator{
		ExpectedAudience: "expected-aud",
		Payloads: map[string]*idtoken.Payload{
			"valid-token": {
				Subject: "user-123",
				Claims: map[string]interface{}{
					"email": "user@example.com",
					"firebase": map[string]interface{}{
						"tenant": "tenant-abc",
					},
				},
			},
		},
		Errors: map[string]error{
			"invalid-token": errors.New("invalid signature"),
		},
	}

	t.Run("Valid Token and Audience", func(t *testing.T) {
		payload, err := mockVal.Validate(context.Background(), "valid-token", "expected-aud")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if payload.Subject != "user-123" {
			t.Errorf("expected subject user-123, got %s", payload.Subject)
		}
		email, ok := payload.Claims["email"].(string)
		if !ok || email != "user@example.com" {
			t.Errorf("expected email user@example.com, got %v", payload.Claims["email"])
		}
	})

	t.Run("Audience Mismatch", func(t *testing.T) {
		_, err := mockVal.Validate(context.Background(), "valid-token", "wrong-aud")
		if err == nil {
			t.Fatal("expected error due to audience mismatch, got nil")
		}
		if !strings.Contains(err.Error(), "audience mismatch") {
			t.Errorf("expected audience mismatch error, got %q", err.Error())
		}
	})

	t.Run("Invalid Token", func(t *testing.T) {
		_, err := mockVal.Validate(context.Background(), "invalid-token", "expected-aud")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err.Error() != "invalid signature" {
			t.Errorf("expected error 'invalid signature', got %q", err.Error())
		}
	})

	t.Run("Missing Token (not configured)", func(t *testing.T) {
		_, err := mockVal.Validate(context.Background(), "unknown-token", "expected-aud")
		if err == nil {
			t.Fatal("expected error for unconfigured token, got nil")
		}
	})
}

func TestDiscoverOrganizationHandler(t *testing.T) {
	// Setup user store with mock users
	mStore := &mockUserStore{
		users: map[string]*db.User{
			"user1": {
				Email: "test-discover@example.com",
				Role:  db.RoleOrgViewer,
			},
		},
	}
	userStore = mStore

	// Case 1: Valid Discover
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/discover?email=test-discover@example.com", nil)
	w := httptest.NewRecorder()
	authHandler := NewAuthHandler(
		nil, // provisioner
		nil, // authenticator
		nil, // validator
		mStore, // userStore
		nil, // orgStore
		&core.Configuration{}, // config
	)
	authHandler.DiscoverOrganization(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp DiscoverOrganizationResponse
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if len(resp.Organizations) != 1 || resp.Organizations[0].ID != "org1" {
		t.Errorf("Expected discovered organizations list to contain 'org1', got: %+v", resp.Organizations)
	}

	// Case 2: Missing email param
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/auth/discover", nil)
	w2 := httptest.NewRecorder()
	authHandler.DiscoverOrganization(w2, req2)

	if w2.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing email, got %d", w2.Code)
	}
}

