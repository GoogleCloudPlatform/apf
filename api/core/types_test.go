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

package core

import (
	"context"
	"testing"
)

func TestGetOrganizationID(t *testing.T) {
	tests := []struct {
		name        string
		setupCtx    func() context.Context
		expectedID  string
		expectedErr bool
	}{
		{
			name: "OrganizationID directly in context",
			setupCtx: func() context.Context {
				return context.WithValue(context.Background(), OrganizationIDKey, "org-123")
			},
			expectedID:  "org-123",
			expectedErr: false,
		},
		{
			name: "OrganizationID in UserClaims in context",
			setupCtx: func() context.Context {
				claims := &UserClaims{
					Subject:        "user-123",
					Username:       "user",
					OrganizationID: "org-456",
				}
				return context.WithValue(context.Background(), UserClaimsKey, claims)
			},
			expectedID:  "org-456",
			expectedErr: false,
		},
		{
			name: "OrganizationID directly in context takes precedence over UserClaims",
			setupCtx: func() context.Context {
				claims := &UserClaims{
					Subject:        "user-123",
					Username:       "user",
					OrganizationID: "org-456",
				}
				ctx := context.WithValue(context.Background(), UserClaimsKey, claims)
				return context.WithValue(ctx, OrganizationIDKey, "org-123")
			},
			expectedID:  "org-123",
			expectedErr: false,
		},
		{
			name: "No OrganizationID or UserClaims",
			setupCtx: func() context.Context {
				return context.Background()
			},
			expectedID:  "",
			expectedErr: true,
		},
		{
			name: "UserClaims without OrganizationID",
			setupCtx: func() context.Context {
				claims := &UserClaims{
					Subject:  "user-123",
					Username: "user",
				}
				return context.WithValue(context.Background(), UserClaimsKey, claims)
			},
			expectedID:  "",
			expectedErr: true,
		},
		{
			name: "Empty OrganizationID directly in context",
			setupCtx: func() context.Context {
				return context.WithValue(context.Background(), OrganizationIDKey, "")
			},
			expectedID:  "",
			expectedErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := tc.setupCtx()
			id, err := GetOrganizationID(ctx)
			if (err != nil) != tc.expectedErr {
				t.Fatalf("expected error: %v, got: %v", tc.expectedErr, err)
			}
			if id != tc.expectedID {
				t.Errorf("expected organization ID: %q, got: %q", tc.expectedID, id)
			}
		})
	}
}

func TestWithOrganizationID(t *testing.T) {
	ctx := context.Background()
	orgID := "custom-org-456"

	newCtx := WithOrganizationID(ctx, orgID)
	retrievedID, err := GetOrganizationID(newCtx)
	if err != nil {
		t.Fatalf("expected no error retrieving organization ID, got: %v", err)
	}
	if retrievedID != orgID {
		t.Errorf("expected %q, got %q", orgID, retrievedID)
	}
}

func TestUserClaims_HasAccessToOrg(t *testing.T) {
	t.Run("SuperAdmin has access to any org", func(t *testing.T) {
		claims := &UserClaims{
			Subject:      "admin-user",
			IsSuperAdmin: true,
		}
		if !claims.HasAccessToOrg("any-org-id") {
			t.Errorf("expected SuperAdmin to have access to any org")
		}
	})

	t.Run("Primary OrganizationID matches", func(t *testing.T) {
		claims := &UserClaims{
			Subject:        "regular-user",
			OrganizationID: "org-primary",
		}
		if !claims.HasAccessToOrg("org-primary") {
			t.Errorf("expected access to matching primary organization")
		}
		if claims.HasAccessToOrg("other-org") {
			t.Errorf("expected no access to non-matching org when Organizations list is empty")
		}
	})

	t.Run("Secondary Organizations list matches", func(t *testing.T) {
		claims := &UserClaims{
			Subject:       "multi-org-user",
			Organizations: []string{"org-alpha", "org-beta"},
		}
		if !claims.HasAccessToOrg("org-beta") {
			t.Errorf("expected access to org in Organizations list")
		}
		if claims.HasAccessToOrg("org-gamma") {
			t.Errorf("expected no access to unlisted org")
		}
	})

	t.Run("Empty claims has no access", func(t *testing.T) {
		claims := &UserClaims{
			Subject: "empty-user",
		}
		if claims.HasAccessToOrg("org-test") {
			t.Errorf("expected empty claims not to have access")
		}
	})
}

