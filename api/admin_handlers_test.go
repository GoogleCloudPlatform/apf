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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"google.com/apf/api/core"
	"google.com/apf/api/db"
)

func TestAdminMiddleware_Gating(t *testing.T) {
	// Setup user store with non-admin user
	userStore = &mockUserStore{
		users: map[string]*db.User{
			"user1": {
				ID:   "user1",
				Role: db.RoleOrgViewer,
			},
		},
	}

	handler := orgPermissionMiddleware(core.PermissionOrgUpdate)(func(w http.ResponseWriter, r *http.Request) {
		orgID, err := core.GetOrganizationID(r.Context())
		if err == nil && orgID != "" {
			w.Header().Set("X-Test-Org-ID", orgID)
		}
		w.WriteHeader(http.StatusOK)
	})

	// Case 1: Viewer user (lacks org:update permission) gets 403
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/organizations/org1/users", nil)
	req.SetPathValue("orgId", "org1")
	claims := &core.UserClaims{Subject: "user1"}
	req = withTestCtx(req, claims)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403 for user lacking permission, got %d", w.Code)
	}

	// Case 2: Admin user (has org:update permission) gets 200
	userStore = &mockUserStore{
		users: map[string]*db.User{
			"admin1": {
				ID:   "admin1",
				Role: db.RoleOrgAdmin,
			},
		},
	}

	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/admin/organizations/org1/users", nil)
	req2.SetPathValue("orgId", "org1")
	claims2 := &core.UserClaims{Subject: "admin1"}
	req2 = withTestCtx(req2, claims2)

	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("Expected status 200 for user with permission, got %d. Body: %s", w2.Code, w2.Body.String())
	}
	if w2.Header().Get("X-Test-Org-ID") != "org1" {
		t.Errorf("Expected organization ID org1 in context, got %q", w2.Header().Get("X-Test-Org-ID"))
	}

	// Case 3: Middleware does NOT reconcile invitation for admin (reconciliation only on selectOrganizationHandler)
	userStore = &mockUserStore{
		users: map[string]*db.User{
			"invited-admin@example.com": {
				ID:    "invited-admin@example.com",
				Email: "invited-admin@example.com",
				Role:  db.RoleOrgAdmin,
			},
		},
	}

	req3 := httptest.NewRequest(http.MethodGet, "/api/v1/admin/organizations/org1/users", nil)
	req3.SetPathValue("orgId", "org1")
	claims3 := &core.UserClaims{
		Subject:  "admin-uid",
		Username: "invited-admin@example.com",
	}
	req3 = withTestCtx(req3, claims3)

	w3 := httptest.NewRecorder()
	handler.ServeHTTP(w3, req3)

	if w3.Code != http.StatusForbidden {
		t.Errorf("Expected status 403 for unreconciled admin, got %d", w3.Code)
	}

	// Verify user is NOT created in store with UID
	mStore, ok := userStore.(*mockUserStore)
	if !ok {
		t.Fatal("userStore is not a *mockUserStore")
	}
	_, err := mStore.GetUser(context.Background(), "org1", "admin-uid")
	if !errors.Is(err, db.ErrNotFound) {
		t.Errorf("Expected user to NOT be created, but GetUser returned: %v", err)
	}

	// Verify invitation still exists
	invite, err := mStore.GetUser(context.Background(), "org1", "invited-admin@example.com")
	if err != nil {
		t.Fatalf("Expected invitation doc to still exist: %v", err)
	}
	if invite.Role != db.RoleOrgAdmin {
		t.Errorf("Expected invitation role to still be admin, got %v", invite.Role)
	}

	// Case 4: Super Admin bypasses database check and gets 200
	userStore = &mockUserStore{users: map[string]*db.User{}} // empty store

	req4 := httptest.NewRequest(http.MethodGet, "/api/v1/admin/organizations/org1/users", nil)
	req4.SetPathValue("orgId", "org1")
	claims4 := &core.UserClaims{
		Subject:      "super-admin-uid",
		IsSuperAdmin: true,
	}
	req4 = withTestCtx(req4, claims4)

	w4 := httptest.NewRecorder()
	handler.ServeHTTP(w4, req4)

	if w4.Code != http.StatusOK {
		t.Errorf("Expected status 200 for Super Admin bypass, got %d. Body: %s", w4.Code, w4.Body.String())
	}
	if w4.Header().Get("X-Test-Org-ID") != "org1" {
		t.Errorf("Expected organization ID org1 in context for Super Admin, got %q", w4.Header().Get("X-Test-Org-ID"))
	}
}

func TestUpdateOrganizationConfigHandler(t *testing.T) {
	mockOrgStore := &mockOrganizationStore{
		organizations: map[string]*db.Organization{
			"org1": {
				ID:   "org1",
				Name: "Org One",
			},
		},
	}
	h := NewAdminHandler(nil, mockOrgStore, &core.Configuration{
		Permissions:           core.AllPermissions(),
		CustomRolePermissions: core.CustomRolePermissions(),
	})

	themeDark := "dark"
	config := db.OrganizationConfig{
		Branding: &db.BrandingConfig{
			Theme: &themeDark,
		},
	}
	body, _ := json.Marshal(config)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/organizations/org1/config", bytes.NewReader(body))
	req.SetPathValue("orgId", "org1")
	ctx := context.WithValue(req.Context(), core.OrganizationIDKey, "org1")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	h.UpdateOrganizationConfig(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var org db.Organization
	_ = json.NewDecoder(w.Body).Decode(&org)
	if org.Config == nil || org.Config.Branding == nil || org.Config.Branding.Theme == nil || *org.Config.Branding.Theme != "dark" {
		t.Errorf("Expected theme to be updated to dark")
	}

	// Test invalid theme validation
	themeBlue := "blue"
	invalidConfig := db.OrganizationConfig{
		Branding: &db.BrandingConfig{
			Theme: &themeBlue,
		},
	}
	body2, _ := json.Marshal(invalidConfig)
	req2 := httptest.NewRequest(http.MethodPut, "/api/v1/admin/organizations/org1/config", bytes.NewReader(body2))
	req2.SetPathValue("orgId", "org1")
	req2 = req2.WithContext(ctx)

	w2 := httptest.NewRecorder()
	h.UpdateOrganizationConfig(w2, req2)

	if w2.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid theme, got %d", w2.Code)
	}
}

func TestAddUserHandler(t *testing.T) {
	// Backup and restore global state
	oldUserStore := userStore
	oldOrgStore := organizationStore
	oldAppConfig := appConfig
	t.Cleanup(func() {
		userStore = oldUserStore
		organizationStore = oldOrgStore
		appConfig = oldAppConfig
	})

	mUserStore := &mockUserStore{
		users: map[string]*db.User{
			"caller-admin": {
				UserID: "caller-admin",
				Role:   db.RoleOrgAdmin,
			},
		},
	}
	h := NewAdminHandler(mUserStore, nil, &core.Configuration{
		Permissions:           core.AllPermissions(),
		CustomRolePermissions: core.CustomRolePermissions(),
	})

	mOrgStore := &mockOrganizationStore{
		organizations: map[string]*db.Organization{
			"org1": {
				ID:   "org1",
				Name: "Org One",
			},
		},
	}
	h.OrganizationStore = mOrgStore

	appConfig = &core.Configuration{
		UiUrl: "http://localhost:8082",
	}

	tests := []struct {
		name           string
		request        AddUserRequest
		expectedStatus int
		verify         func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name: "Add Admin Success",
			request: AddUserRequest{
				Email:    "new-admin@example.com",
				Password: "Password123!",
				Role:     "admin",
			},
			expectedStatus: http.StatusCreated,
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if resp["email"] != "new-admin@example.com" || resp["role"] != "org:admin" || resp["status"] != "invited" {
					t.Errorf("Unexpected response: %v", resp)
				}
				id := resp["id"].(string)
				// Verify in store
				added, err := mUserStore.GetUser(context.Background(), "org1", id)
				if err != nil {
					t.Fatalf("Expected user doc to exist: %v", err)
				}
				if added.Email != "new-admin@example.com" || added.Role != db.RoleOrgAdmin {
					t.Errorf("Unexpected user data in store: %v", added)
				}
			},
		},
		{
			name: "Add Editor Success",
			request: AddUserRequest{
				Email:    "new-editor@example.com",
				Password: "Password123!",
				Role:     "editor",
			},
			expectedStatus: http.StatusCreated,
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if resp["email"] != "new-editor@example.com" || resp["role"] != "org:editor" || resp["status"] != "invited" {
					t.Errorf("Unexpected response: %v", resp)
				}
				id := resp["id"].(string)
				// Verify in store
				added, err := mUserStore.GetUser(context.Background(), "org1", id)
				if err != nil {
					t.Fatalf("Expected user doc to exist: %v", err)
				}
				if added.Email != "new-editor@example.com" || added.Role != db.RoleOrgEditor {
					t.Errorf("Unexpected user data in store: %v", added)
				}
			},
		},
		{
			name: "Add Viewer Success",
			request: AddUserRequest{
				Email:    "new-viewer@example.com",
				Password: "Password123!",
				Role:     "viewer",
			},
			expectedStatus: http.StatusCreated,
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				id := resp["id"].(string)
				// Verify in store
				added, err := mUserStore.GetUser(context.Background(), "org1", id)
				if err != nil {
					t.Fatalf("Expected user doc to exist: %v", err)
				}
				if added.Email != "new-viewer@example.com" || added.Role != db.RoleOrgViewer {
					t.Errorf("Unexpected user data in store: %v", added)
				}
			},
		},
		{
			name: "Add Multiple Roles Success",
			request: AddUserRequest{
				Email:    "multi-role@example.com",
				Password: "Password123!",
				Roles:    []string{"viewer", "project:editor"},
			},
			expectedStatus: http.StatusCreated,
			verify: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if resp["email"] != "multi-role@example.com" || resp["role"] != "org:viewer" || resp["status"] != "invited" {
					t.Errorf("Unexpected response: %v", resp)
				}
				roles, ok := resp["roles"].([]interface{})
				if !ok || len(roles) != 2 || roles[0] != "org:viewer" || roles[1] != "project:editor" {
					t.Errorf("Unexpected roles in response: %v", resp["roles"])
				}
				id := resp["id"].(string)
				// Verify in store
				added, err := mUserStore.GetUser(context.Background(), "org1", id)
				if err != nil {
					t.Fatalf("Expected user doc to exist: %v", err)
				}
				if added.Email != "multi-role@example.com" || added.Role != db.RoleOrgViewer {
					t.Errorf("Unexpected user data in store: %v", added)
				}
				if len(added.Roles) != 2 || added.Roles[0] != db.RoleOrgViewer || added.Roles[1] != db.RoleProjectEditor {
					t.Errorf("Unexpected user roles in store: %v", added.Roles)
				}
			},
		},
		{
			name: "Invalid Role",
			request: AddUserRequest{
				Email:    "bad-role@example.com",
				Password: "Password123!",
				Role:     "invalid-role",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Missing Email",
			request: AddUserRequest{
				Email:    "",
				Password: "Password123!",
				Role:     "viewer",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Missing Password",
			request: AddUserRequest{
				Email:    "no-pass@example.com",
				Password: "",
				Role:     "viewer",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Short Password",
			request: AddUserRequest{
				Email:    "short-pass@example.com",
				Password: "123",
				Role:     "viewer",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Add Owner Blocked (Privilege Escalation)",
			request: AddUserRequest{
				Email:    "new-owner@example.com",
				Password: "Password123!",
				Role:     "owner",
			},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body, _ := json.Marshal(tc.request)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/organizations/org1/users", bytes.NewReader(body))
			req.SetPathValue("orgId", "org1")
			claims := &core.UserClaims{Subject: "caller-admin"}
			req = withTestCtx(req, claims)

			w := httptest.NewRecorder()
			h.AddUser(w, req)

			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tc.expectedStatus, w.Code, w.Body.String())
			}

			if tc.verify != nil && w.Code == tc.expectedStatus {
				tc.verify(t, w)
			}
		})
	}
}

func TestListOrganizationUsersHandler(t *testing.T) {
	mUserStore := &mockUserStore{
		users: map[string]*db.User{
			"user1-uid": {
				UserID:     "user1-uid",
				Email:      "user1@example.com",
				Role:       db.RoleOrgViewer,
				TourStatus: &db.TourStatus{Completed: true},
			},
			"pending@example.com": {
				Email: "pending@example.com",
				Role:  db.RoleOrgAdmin,
			},
		},
	}
	h := NewAdminHandler(mUserStore, nil, &core.Configuration{
		Permissions:           core.AllPermissions(),
		CustomRolePermissions: core.CustomRolePermissions(),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/organizations/org1/users", nil)
	req.SetPathValue("orgId", "org1")

	w := httptest.NewRecorder()
	h.ListOrganizationUsers(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp ListOrganizationUsersResponse
	_ = json.NewDecoder(w.Body).Decode(&resp)

	if len(resp.Users) != 2 {
		t.Fatalf("Expected 2 users, got %d", len(resp.Users))
	}

	var uActive, uInvited *UserListItem
	for i := range resp.Users {
		u := &resp.Users[i]
		switch u.ID {
		case "user1-uid":
			uActive = u
		case "pending@example.com":
			uInvited = u
		}
	}

	if uActive == nil || uActive.UserID != "user1-uid" || uActive.Status != "active" || uActive.ID != "user1-uid" {
		t.Errorf("Unexpected active user data: %v", uActive)
	} else {
		if len(uActive.Roles) != 1 || uActive.Roles[0] != string(db.RoleOrgViewer) {
			t.Errorf("Expected active user roles to be [%s], got %v", db.RoleOrgViewer, uActive.Roles)
		}
	}
	if uInvited == nil || uInvited.UserID != "" || uInvited.Status != "invited" || uInvited.ID != "pending@example.com" {
		t.Errorf("Unexpected invited user data: %v", uInvited)
	} else {
		if len(uInvited.Roles) != 1 || uInvited.Roles[0] != string(db.RoleOrgAdmin) {
			t.Errorf("Expected invited user roles to be [%s], got %v", db.RoleOrgAdmin, uInvited.Roles)
		}
	}
}

func TestListOrganizationUsers_Search(t *testing.T) {
	mUserStore := &mockUserStore{
		users: map[string]*db.User{
			"user1-uid": {
				UserID:      "user1-uid",
				Email:       "alice@example.com",
				DisplayName: "Alice Smith",
				Role:        db.RoleOrgViewer,
				Roles:       []db.UserRole{db.RoleOrgViewer},
			},
			"user2-uid": {
				UserID:      "user2-uid",
				Email:       "bob@company.com",
				DisplayName: "Bob Jones",
				Role:        db.RoleOrgAdmin,
				Roles:       []db.UserRole{db.RoleOrgAdmin},
			},
		},
	}
	h := NewAdminHandler(mUserStore, nil, &core.Configuration{
		Permissions:           core.AllPermissions(),
		CustomRolePermissions: core.CustomRolePermissions(),
	})

	tests := []struct {
		name          string
		query         string
		expectedCount int
		expectedEmail string
	}{
		{
			name:          "Search by email substring",
			query:         "alice",
			expectedCount: 1,
			expectedEmail: "alice@example.com",
		},
		{
			name:          "Search by display name",
			query:         "Jones",
			expectedCount: 1,
			expectedEmail: "bob@company.com",
		},
		{
			name:          "Search by role",
			query:         "admin",
			expectedCount: 1,
			expectedEmail: "bob@company.com",
		},
		{
			name:          "Search with no matches",
			query:         "charlie",
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/organizations/org1/users?q="+tt.query, nil)
			req.SetPathValue("orgId", "org1")

			w := httptest.NewRecorder()
			h.ListOrganizationUsers(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("Expected status 200, got %d", w.Code)
			}

			var resp ListOrganizationUsersResponse
			_ = json.NewDecoder(w.Body).Decode(&resp)

			if len(resp.Users) != tt.expectedCount {
				t.Errorf("Expected %d users, got %d", tt.expectedCount, len(resp.Users))
			}
			if tt.expectedCount == 1 && resp.Users[0].Email != tt.expectedEmail {
				t.Errorf("Expected user email %s, got %s", tt.expectedEmail, resp.Users[0].Email)
			}
		})
	}
}

func TestRemoveUserHandler(t *testing.T) {
	mStore := &mockUserStore{
		users: map[string]*db.User{
			"user-to-remove": {
				UserID: "user-to-remove",
				Role:   db.RoleOrgViewer,
			},
		},
	}
	h := NewAdminHandler(mStore, nil, &core.Configuration{
		Permissions:           core.AllPermissions(),
		CustomRolePermissions: core.CustomRolePermissions(),
	})

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/organizations/org1/users/user-to-remove", nil)
	req.SetPathValue("orgId", "org1")
	req.SetPathValue("userId", "user-to-remove")
	claims := &core.UserClaims{Subject: "admin-caller"}
	req = withTestCtx(req, claims)

	w := httptest.NewRecorder()
	h.RemoveUser(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	_, err := mStore.GetUser(context.Background(), "org1", "user-to-remove")
	if !errors.Is(err, db.ErrNotFound) {
		t.Errorf("Expected user to be removed, but GetUser returned: %v", err)
	}
}

func TestUpdateUserRoleHandler(t *testing.T) {
	mUserStore := &mockUserStore{
		users: map[string]*db.User{
			"user-to-update": {
				UserID: "user-to-update",
				Role:   db.RoleOrgViewer,
			},
			"admin-caller": {
				UserID: "admin-caller",
				Role:   db.RoleOrgAdmin,
			},
		},
	}
	h := NewAdminHandler(mUserStore, nil, &core.Configuration{
		Permissions:           core.AllPermissions(),
		CustomRolePermissions: core.CustomRolePermissions(),
	})

	// Case 1: Success promoting viewer to admin
	roleReq := UpdateUserRoleRequest{Role: "admin"}
	body, _ := json.Marshal(roleReq)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/organizations/org1/users/user-to-update/role", bytes.NewReader(body))
	req.SetPathValue("orgId", "org1")
	req.SetPathValue("userId", "user-to-update")
	claims := &core.UserClaims{Subject: "admin-caller"}
	req = withTestCtx(req, claims)

	w := httptest.NewRecorder()
	h.UpdateUserRole(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	updatedUser, _ := mUserStore.GetUser(context.Background(), "org1", "user-to-update")
	if updatedUser.Role != db.RoleOrgAdmin {
		t.Errorf("Expected role to be updated to admin, got %v", updatedUser.Role)
	}

	// Case 2: Prevent self-demotion
	req2 := httptest.NewRequest(http.MethodPut, "/api/v1/admin/organizations/org1/users/admin-caller/role", bytes.NewReader(body))
	req2.SetPathValue("orgId", "org1")
	req2.SetPathValue("userId", "admin-caller")
	req2 = withTestCtx(req2, claims)

	w2 := httptest.NewRecorder()
	h.UpdateUserRole(w2, req2)

	if w2.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for self-demotion, got %d", w2.Code)
	}

	// Case 3: Prevent demoting the last active admin
	// Currently we have two admins: admin-caller (active) and user-to-update (active).
	// If we demote user-to-update, it should work because admin-caller is still active.
	roleReqViewer := UpdateUserRoleRequest{Role: "viewer"}
	bodyViewer, _ := json.Marshal(roleReqViewer)
	req3 := httptest.NewRequest(http.MethodPut, "/api/v1/admin/organizations/org1/users/user-to-update/role", bytes.NewReader(bodyViewer))
	req3.SetPathValue("orgId", "org1")
	req3.SetPathValue("userId", "user-to-update")
	req3 = withTestCtx(req3, claims)

	w3 := httptest.NewRecorder()
	h.UpdateUserRole(w3, req3)

	if w3.Code != http.StatusOK {
		t.Errorf("Expected status 200 for demoting one of multiple admins, got %d. Body: %s", w3.Code, w3.Body.String())
	}

	// Now user-to-update is viewer again. admin-caller is the ONLY active admin.
	// If a different caller (simulated by changing claims Subject, e.g. a super admin)
	// tries to demote admin-caller, it should be BLOCKED because admin-caller is the last active admin.
	req4 := httptest.NewRequest(http.MethodPut, "/api/v1/admin/organizations/org1/users/admin-caller/role", bytes.NewReader(bodyViewer))
	req4.SetPathValue("orgId", "org1")
	req4.SetPathValue("userId", "admin-caller")
	superClaims := &core.UserClaims{Subject: "super-admin", IsSuperAdmin: true} // different from target admin-caller
	req4 = withTestCtx(req4, superClaims)

	w4 := httptest.NewRecorder()
	h.UpdateUserRole(w4, req4)

	if w4.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 when demoting the last active admin, got %d. Body: %s", w4.Code, w4.Body.String())
	}

	// Case 4: Prevent privilege escalation (admin caller promoting viewer to owner)
	roleReqOwner := UpdateUserRoleRequest{Role: "owner"}
	bodyOwner, _ := json.Marshal(roleReqOwner)
	req5 := httptest.NewRequest(http.MethodPut, "/api/v1/admin/organizations/org1/users/user-to-update/role", bytes.NewReader(bodyOwner))
	req5.SetPathValue("orgId", "org1")
	req5.SetPathValue("userId", "user-to-update")
	req5 = withTestCtx(req5, claims) // caller claims: admin-caller (org:admin)

	w5 := httptest.NewRecorder()
	h.UpdateUserRole(w5, req5)

	if w5.Code != http.StatusForbidden {
		t.Errorf("Expected status 403 when admin attempts to assign owner role, got %d. Body: %s", w5.Code, w5.Body.String())
	}
}

func TestGetSystemInfo(t *testing.T) {
	h := NewAdminHandler(nil, nil, &core.Configuration{
		GoogleCloudLocation: "us-west-2",
		SuperAdminEmails:    []string{"admin@example.com"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/system-info", nil)
	claims := &core.UserClaims{
		Subject:      "admin-caller",
		IsSuperAdmin: true,
	}
	req = withTestCtx(req, claims)

	w := httptest.NewRecorder()
	h.GetSystemInfo(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp SystemInfoResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Version == "" || !strings.HasPrefix(resp.Version, "v") {
		t.Errorf("Expected version starting with 'v', got %s", resp.Version)
	}
	if resp.Region != "us-west-2" {
		t.Errorf("Expected region 'us-west-2', got %s", resp.Region)
	}
}
