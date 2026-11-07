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
	"net/http"
	"net/http/httptest"
	"testing"

	"google.com/apf/api/core"
	"google.com/apf/api/db"
)

type mockAutojoinStore struct {
	db.AutojoinStore
	rules map[string]*db.AutojoinRule
}

func (m *mockAutojoinStore) FindActiveRuleByDomain(ctx context.Context, domain string) (*db.AutojoinRule, error) {
	if m.rules == nil {
		return nil, db.ErrNotFound
	}
	rule, ok := m.rules[domain]
	if !ok {
		return nil, db.ErrNotFound
	}
	return rule, nil
}

func TestListProjectMembers(t *testing.T) {
	ownerID := "owner-123"
	editorID := "editor-123"
	viewerID := "viewer-123"
	pendingEmail := "pending.user@test.com"

	proj := &db.Project{
		ID:      "proj-123",
		OwnerId: &ownerID,
		Members: &map[string]string{
			editorID:     "Editor",
			viewerID:     "Viewer",
			pendingEmail: "Viewer",
		},
	}

	store = &testMockProjectStore{
		projects: map[string]*db.Project{
			"proj-123": proj,
		},
	}

	userStore = &mockUserStore{
		users: map[string]*db.User{
			ownerID: {
				UserID:      ownerID,
				DisplayName: "Owner User",
				Email:       "owner@test.com",
			},
			editorID: {
				UserID:      editorID,
				DisplayName: "Editor User",
				Email:       "editor@test.com",
			},
			viewerID: {
				UserID:      viewerID,
				DisplayName: "Viewer User",
				Email:       "viewer@test.com",
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/proj-123/members", nil)
	req.SetPathValue("id", "proj-123")
	claims := &core.UserClaims{Subject: ownerID}
	req = withTestCtx(req, claims)

	w := httptest.NewRecorder()
	listProjectMembersHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}

	var members []ProjectMemberResponse
	if err := json.NewDecoder(w.Body).Decode(&members); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(members) != 4 {
		t.Errorf("Expected 4 members, got %d", len(members))
	}

	// Verify pending member name derivation ("pending.user@test.com" -> "Pending User")
	foundPending := false
	for _, m := range members {
		if m.ID == pendingEmail {
			foundPending = true
			if !m.IsPending {
				t.Error("Expected member to be pending")
			}
			if m.Name != "Pending User" {
				t.Errorf("Expected name 'Pending User', got '%s'", m.Name)
			}
		}
	}
	if !foundPending {
		t.Error("Pending member not found")
	}
}

func TestAddProjectMembers(t *testing.T) {
	ownerID := "owner-123"
	existingUserID := "existing-123"
	autojoinEmail := "newguy@autojoin.com"
	badEmail := "stranger@danger.com"

	proj := &db.Project{
		ID:      "proj-123",
		OwnerId: &ownerID,
		Members: &map[string]string{
			ownerID: "owner",
		},
	}

	store = &testMockProjectStore{
		projects: map[string]*db.Project{
			"proj-123": proj,
		},
	}

	userStore = &mockUserStore{
		users: map[string]*db.User{
			ownerID: {
				UserID:      ownerID,
				DisplayName: "Owner User",
				Email:       "owner@test.com",
			},
			existingUserID: {
				UserID:      existingUserID,
				DisplayName: "Existing User",
				Email:       "existing@test.com",
			},
		},
	}

	autojoinStore = &mockAutojoinStore{
		rules: map[string]*db.AutojoinRule{
			"autojoin.com": {
				OrganizationID: "test-organization", // matches test context org
				Domain:         "autojoin.com",
			},
		},
	}

	// Test Case 1: Add existing user (Success)
	payload := AddProjectMembersRequest{
		Emails: []string{"existing@test.com"},
		Role:   "Editor",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/proj-123/members", bytes.NewBuffer(body))
	req.SetPathValue("id", "proj-123")
	req = withTestCtx(req, &core.UserClaims{Subject: ownerID})

	w := httptest.NewRecorder()
	addProjectMembersHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}

	// Verify member added by UserID
	if (*proj.Members)[existingUserID] != "Editor" {
		t.Errorf("Expected existing user to be added as Editor, got %s", (*proj.Members)[existingUserID])
	}

	// Test Case 2: Add autojoin email (Success - stored as email)
	payload = AddProjectMembersRequest{
		Emails: []string{autojoinEmail},
		Role:   "Viewer",
	}
	body, _ = json.Marshal(payload)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/projects/proj-123/members", bytes.NewBuffer(body))
	req.SetPathValue("id", "proj-123")
	req = withTestCtx(req, &core.UserClaims{Subject: ownerID})

	w = httptest.NewRecorder()
	addProjectMembersHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}

	// Verify member added by Email
	if (*proj.Members)[autojoinEmail] != "Viewer" {
		t.Errorf("Expected autojoin email to be added as Viewer, got %s", (*proj.Members)[autojoinEmail])
	}

	// Test Case 3: Add bad email (Failure - 404)
	payload = AddProjectMembersRequest{
		Emails: []string{badEmail},
		Role:   "Viewer",
	}
	body, _ = json.Marshal(payload)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/projects/proj-123/members", bytes.NewBuffer(body))
	req.SetPathValue("id", "proj-123")
	req = withTestCtx(req, &core.UserClaims{Subject: ownerID})

	w = httptest.NewRecorder()
	addProjectMembersHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("Expected 404 Not Found, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestEditorPermissions(t *testing.T) {
	ownerID := "owner-123"
	editorID := "editor-123"
	otherID := "other-123"

	proj := &db.Project{
		ID:      "proj-123",
		OwnerId: &ownerID,
		Members: &map[string]string{
			ownerID:  "owner",
			editorID: "Editor",
			otherID:  "Viewer",
		},
	}

	store = &testMockProjectStore{
		projects: map[string]*db.Project{
			"proj-123": proj,
		},
	}

	userStore = &mockUserStore{
		users: map[string]*db.User{
			ownerID:    {UserID: ownerID, Email: "owner@test.com"},
			editorID:   {UserID: editorID, Email: "editor@test.com"},
			otherID:    {UserID: otherID, Email: "other@test.com"},
		},
	}

	// Test Case 1: Editor tries to grant "Owner" role (Forbidden)
	payload := AddProjectMembersRequest{
		Emails: []string{"other@test.com"},
		Role:   "Owner", // invalid role for add (only Editor/Viewer allowed, but also Editor cannot grant Owner)
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/proj-123/members", bytes.NewBuffer(body))
	req.SetPathValue("id", "proj-123")
	req = withTestCtx(req, &core.UserClaims{Subject: editorID}) // Caller is Editor

	w := httptest.NewRecorder()
	addProjectMembersHandler(w, req)

	// Should fail because Owner is not in allowed roles (only Editor/Viewer allowed in request)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request, got %d", w.Code)
	}

	// Test Case 2: Editor tries to modify Project Owner (Forbidden)
	updatePayload := UpdateProjectMemberRoleRequest{Role: "Viewer"}
	body, _ = json.Marshal(updatePayload)
	req = httptest.NewRequest(http.MethodPut, "/api/v1/projects/proj-123/members/"+ownerID, bytes.NewBuffer(body))
	req.SetPathValue("id", "proj-123")
	req.SetPathValue("memberId", ownerID)
	req = withTestCtx(req, &core.UserClaims{Subject: editorID}) // Caller is Editor

	w = httptest.NewRecorder()
	updateProjectMemberRoleHandler(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected 403 Forbidden for Editor modifying Owner, got %d. Body: %s", w.Code, w.Body.String())
	}

	// Test Case 3: Editor tries to remove Project Owner (Forbidden)
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/projects/proj-123/members/"+ownerID, nil)
	req.SetPathValue("id", "proj-123")
	req.SetPathValue("memberId", ownerID)
	req = withTestCtx(req, &core.UserClaims{Subject: editorID}) // Caller is Editor

	w = httptest.NewRecorder()
	removeProjectMemberHandler(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected 403 Forbidden for Editor removing Owner, got %d. Body: %s", w.Code, w.Body.String())
	}

	// Test Case 4: Editor removes regular member (Success)
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/projects/proj-123/members/"+otherID, nil)
	req.SetPathValue("id", "proj-123")
	req.SetPathValue("memberId", otherID)
	req = withTestCtx(req, &core.UserClaims{Subject: editorID}) // Caller is Editor

	w = httptest.NewRecorder()
	removeProjectMemberHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}

	// Verify member removed
	if _, ok := (*proj.Members)[otherID]; ok {
		t.Error("Expected member to be removed by Editor")
	}
}

func TestReconciliationMock(t *testing.T) {
	// Test the mock implementation of ReconcileMemberID to ensure our test logic works
	pendingEmail := "pending@test.com"
	userID := "user-999"

	proj := &db.Project{
		ID: "proj-123",
		Members: &map[string]string{
			pendingEmail: "Editor",
		},
	}

	mockStore := &testMockProjectStore{
		projects: map[string]*db.Project{
			"proj-123": proj,
		},
	}

	err := mockStore.ReconcileMemberID(context.Background(), "org1", pendingEmail, userID)
	if err != nil {
		t.Fatalf("ReconcileMemberID failed: %v", err)
	}

	// Verify email key is gone and userID key is present with same role
	if _, ok := (*proj.Members)[pendingEmail]; ok {
		t.Error("Expected pending email key to be removed")
	}
	if (*proj.Members)[userID] != "Editor" {
		t.Errorf("Expected userID key to have role 'Editor', got '%s'", (*proj.Members)[userID])
	}
}
