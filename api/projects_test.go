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
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"google.com/apf/api/core"
	"google.com/apf/api/db"
	"google.com/apf/api/integrations"
	"google.com/apf/api/workspace"
)

type testMockProjectStore struct {
	db.ProjectStore
	projects map[string]*db.Project
}

func (m *testMockProjectStore) DeleteProject(ctx context.Context, organizationID string, id string) error {
	if m.projects != nil {
		delete(m.projects, id)
	}
	return nil
}


type mockIntegrationStore struct {
	db.IntegrationStore
	integrations map[string]*db.Integration
}

func (m *mockIntegrationStore) GetIntegration(ctx context.Context, userId, name string) (*db.Integration, error) {
	if m.integrations == nil {
		return nil, fmt.Errorf("not found")
	}
	docID := fmt.Sprintf("%s_%s", userId, name)
	i, ok := m.integrations[docID]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return i, nil
}

func (m *mockIntegrationStore) SaveIntegration(ctx context.Context, integration *db.Integration) error {
	if m.integrations == nil {
		m.integrations = make(map[string]*db.Integration)
	}
	docID := fmt.Sprintf("%s_%s", integration.UserID, integration.Name)
	m.integrations[docID] = integration
	return nil
}

func (m *testMockProjectStore) CreateProject(ctx context.Context, organizationID string, p *db.Project) error {
	if m.projects == nil {
		m.projects = make(map[string]*db.Project)
	}
	p.ID = fmt.Sprintf("project-%d", len(m.projects)+1)
	m.projects[p.ID] = p
	return nil
}

func (m *testMockProjectStore) GetProject(ctx context.Context, organizationID string, id string) (*db.Project, error) {
	if m.projects == nil {
		return nil, fmt.Errorf("project not found")
	}
	p, ok := m.projects[id]
	if !ok {
		return nil, fmt.Errorf("project not found")
	}
	return p, nil
}

func (m *testMockProjectStore) UpdateProject(ctx context.Context, organizationID string, p *db.Project) error {
	if m.projects == nil {
		return fmt.Errorf("project not found")
	}
	m.projects[p.ID] = p
	return nil
}

func (m *testMockProjectStore) ReconcileMemberID(ctx context.Context, organizationID string, oldID string, newID string) error {
	if m.projects == nil {
		return nil
	}
	for _, p := range m.projects {
		if p.Members != nil {
			role, ok := (*p.Members)[oldID]
			if ok {
				(*p.Members)[newID] = role
				delete(*p.Members, oldID)
			}
		}
	}
	return nil
}

func (m *testMockProjectStore) ListProjects(ctx context.Context, organizationID string) ([]db.Project, error) {
	if m.projects == nil {
		return nil, nil
	}
	var list []db.Project
	for _, p := range m.projects {
		list = append(list, *p)
	}
	return list, nil
}

func TestCreateProjectDefaultStatusNew(t *testing.T) {
	// Set up mock store
	mockStore := &testMockProjectStore{
		projects: make(map[string]*db.Project),
	}
	store = mockStore

	projName := "New Test Project"
	payload := db.Project{
		Name: &projName,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", bytes.NewBuffer(body))
	claims := &core.UserClaims{
		Subject: "user-1",
		Roles:   []string{string(core.RoleProjectCreator)},
	}
	req = withTestCtx(req, claims)

	w := httptest.NewRecorder()
	createProjectHandler(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected status 201 Created, got %d. Body: %s", w.Code, w.Body.String())
	}

	var created db.Project
	if err := json.NewDecoder(w.Body).Decode(&created); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if created.Status == nil {
		t.Error("Expected project status to not be nil")
	} else if *created.Status != "New" {
		t.Errorf("Expected project status to be 'New', got '%s'", *created.Status)
	}
}

func TestUpdateProjectStatusToOnboarding(t *testing.T) {
	// Set up mock store with an existing project owned by user-1
	projName := "Existing Project"
	status := "New"
	ownerId := "user-1"
	members := map[string]string{
		"user-1": "owner",
	}
	existingProj := &db.Project{
		ID:      "project-123",
		Name:    &projName,
		Status:  &status,
		OwnerId: &ownerId,
		Members: &members,
	}

	mockStore := &testMockProjectStore{
		projects: map[string]*db.Project{
			"project-123": existingProj,
		},
	}
	store = mockStore

	// Payload to update status to "Onboarding"
	newStatus := "Onboarding"
	payload := db.Project{
		Status: &newStatus,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/project-123", bytes.NewBuffer(body))
	req.SetPathValue("id", "project-123")
	claims := &core.UserClaims{
		Subject: "user-1",
		Roles:   []string{"member"},
	}
	req = withTestCtx(req, claims)

	w := httptest.NewRecorder()
	// Call via middleware to verify projectAccessMiddleware works too
	handler := projectAccessMiddleware(updateProjectHandler)
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}

	var updated db.Project
	if err := json.NewDecoder(w.Body).Decode(&updated); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if updated.Status == nil {
		t.Error("Expected project status to not be nil")
	} else if *updated.Status != "Onboarding" {
		t.Errorf("Expected project status to be 'Onboarding', got '%s'", *updated.Status)
	}
}

func TestUpdateProjectStatusForbidden(t *testing.T) {
	// Set up mock store with an existing project owned by user-1
	projName := "Existing Project"
	status := "New"
	ownerId := "user-1"
	members := map[string]string{
		"user-1": "owner",
	}
	existingProj := &db.Project{
		ID:      "project-123",
		Name:    &projName,
		Status:  &status,
		OwnerId: &ownerId,
		Members: &members,
	}

	mockStore := &testMockProjectStore{
		projects: map[string]*db.Project{
			"project-123": existingProj,
		},
	}
	store = mockStore

	// Payload to update status to "Onboarding"
	newStatus := "Onboarding"
	payload := db.Project{
		Status: &newStatus,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/project-123", bytes.NewBuffer(body))
	req.SetPathValue("id", "project-123")
	// Different user (user-999) who is not owner or member
	claims := &core.UserClaims{
		Subject: "user-999",
		Roles:   []string{"member"},
	}
	req = withTestCtx(req, claims)

	w := httptest.NewRecorder()
	// Call via middleware to verify projectAccessMiddleware works too and blocks access
	handler := projectAccessMiddleware(updateProjectHandler)
	handler(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403 Forbidden, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestUpdateProjectStatus_SuperAdminBypass(t *testing.T) {
	// Set up mock store with an existing project owned by user-1
	projName := "Existing Project"
	status := "Onboarding"
	ownerId := "user-1"
	members := map[string]string{
		"user-1": "owner",
	}
	existingProj := &db.Project{
		ID:      "project-123",
		Name:    &projName,
		Status:  &status,
		OwnerId: &ownerId,
		Members: &members,
	}

	mockStore := &testMockProjectStore{
		projects: map[string]*db.Project{
			"project-123": existingProj,
		},
	}
	store = mockStore

	newStatus := "Active"
	payload := db.Project{
		Status: &newStatus,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/project-123", bytes.NewBuffer(body))
	req.SetPathValue("id", "project-123")
	
	// Super Admin claims (not owner, not member)
	claims := &core.UserClaims{
		Subject:      "super-admin-uid",
		IsSuperAdmin: true,
	}
	
	// Inject claims AND organization ID into context (simulating orgScopeMiddleware running first)
	ctx := context.WithValue(req.Context(), core.UserClaimsKey, claims)
	ctx = core.WithOrganizationID(ctx, "org-123")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler := projectAccessMiddleware(updateProjectHandler)
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 OK for Super Admin bypass, got %d. Body: %s", w.Code, w.Body.String())
	}

	var updated db.Project
	if err := json.NewDecoder(w.Body).Decode(&updated); err == nil {
		if updated.Status == nil || *updated.Status != "Active" {
			t.Errorf("Expected project status to be updated to 'Active' by Super Admin, got %v", updated.Status)
		}
	} else {
		t.Fatalf("Failed to decode response: %v", err)
	}
}

func TestUpdateProjectStatusToOnboarded(t *testing.T) {
	// Set up mock store with an existing project owned by user-1
	projName := "Existing Project"
	status := "Onboarding"
	ownerId := "user-1"
	members := map[string]string{
		"user-1": "owner",
	}
	existingProj := &db.Project{
		ID:      "project-123",
		Name:    &projName,
		Status:  &status,
		OwnerId: &ownerId,
		Members: &members,
	}

	mockStore := &testMockProjectStore{
		projects: map[string]*db.Project{
			"project-123": existingProj,
		},
	}
	store = mockStore

	// Payload to update status to "Onboarded"
	newStatus := "Onboarded"
	payload := db.Project{
		Status: &newStatus,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/project-123", bytes.NewBuffer(body))
	req.SetPathValue("id", "project-123")
	claims := &core.UserClaims{
		Subject: "user-1",
		Roles:   []string{"member"},
	}
	req = withTestCtx(req, claims)

	w := httptest.NewRecorder()
	handler := projectAccessMiddleware(updateProjectHandler)
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}

	var updated db.Project
	if err := json.NewDecoder(w.Body).Decode(&updated); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if updated.Status == nil {
		t.Error("Expected project status to not be nil")
	} else if *updated.Status != "Onboarded" {
		t.Errorf("Expected project status to be 'Onboarded', got '%s'", *updated.Status)
	}

	if updated.GovernanceGates == nil {
		t.Fatal("Expected GovernanceGates to not be nil")
	}
	gates := *updated.GovernanceGates
	if gates["ONBOARDING"] != "Complete" {
		t.Errorf("Expected ONBOARDING gate to be 'Complete', got '%s'", gates["ONBOARDING"])
	}
}

func TestUpdateProjectIntegrationsHandler_Success(t *testing.T) {
	oldClient := FirestoreClient
	FirestoreClient = nil
	defer func() { FirestoreClient = oldClient }()

	existingProj := &db.Project{
		ID:      "project-123",
		OwnerId: func() *string { s := "user-1"; return &s }(),
		Members: &map[string]string{"user-1": "owner"},
	}

	mockStore := &testMockProjectStore{
		projects: map[string]*db.Project{
			"project-123": existingProj,
		},
	}
	store = mockStore

	oldIntegrationStore := integrations.Store
	mockIntStore := &mockIntegrationStore{
		integrations: map[string]*db.Integration{
			"user-1_github": {
				UserID: "user-1",
				Name:   "github",
				Settings: map[string]interface{}{
					"accessToken": "ghu_simulated_access_token_987654321",
				},
			},
		},
	}
	integrations.Store = mockIntStore
	defer func() { integrations.Store = oldIntegrationStore }()

	payload := UpdateIntegrationPayload{
		IntegrationID: "github",
		Enabled:       func() *bool { b := true; return &b }(),
		Settings: map[string]interface{}{
			"repoUrl":          "https://github.com/test/repo",
			"specLocation":     "docs/specs",
			"baseBranch":       "main",
			"specBranchPrefix": "spec-",
			"email":            "test@example.com",
		},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/project-123/integrations", bytes.NewBuffer(body))
	req.SetPathValue("id", "project-123")
	claims := &core.UserClaims{
		Subject: "user-1",
		Roles:   []string{"member"},
	}
	req = withTestCtx(req, claims)

	w := httptest.NewRecorder()
	handler := projectAccessMiddleware(updateProjectIntegrationsHandler)
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}

	var updated db.Project
	if err := json.NewDecoder(w.Body).Decode(&updated); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(updated.AppIntegrations) != 1 {
		t.Fatalf("Expected 1 app integration, got %d", len(updated.AppIntegrations))
	}

	integration := updated.AppIntegrations[0]
	if integration.IntegrationID != "github" || !integration.Enabled {
		t.Errorf("Expected github integration enabled, got %+v", integration)
	}

	if updated.GetRepoUrl() != "https://github.com/test/repo" {
		t.Errorf("Expected dynamic GetRepoUrl to resolve, got %v", updated.GetRepoUrl())
	}

	if updated.GetGitHubEmail() != "test@example.com" {
		t.Errorf("Expected dynamic GetGitHubEmail to resolve, got %v", updated.GetGitHubEmail())
	}
}

func TestUpdateProjectIntegrationsHandler_ValidationFailure(t *testing.T) {
	oldClient := FirestoreClient
	FirestoreClient = nil
	defer func() { FirestoreClient = oldClient }()

	existingProj := &db.Project{
		ID:      "project-123",
		OwnerId: func() *string { s := "user-1"; return &s }(),
		Members: &map[string]string{"user-1": "owner"},
	}

	mockStore := &testMockProjectStore{
		projects: map[string]*db.Project{
			"project-123": existingProj,
		},
	}
	store = mockStore

	// Missing required Settings fields for GitHub integration when enabled = true
	payload := UpdateIntegrationPayload{
		IntegrationID: "github",
		Enabled:       func() *bool { b := true; return &b }(),
		Settings: map[string]interface{}{
			"repoUrl": "https://github.com/test/repo",
			// specLocation, baseBranch, specBranchPrefix are missing
		},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/project-123/integrations", bytes.NewBuffer(body))
	req.SetPathValue("id", "project-123")
	claims := &core.UserClaims{
		Subject: "user-1",
		Roles:   []string{"member"},
	}
	req = withTestCtx(req, claims)

	w := httptest.NewRecorder()
	handler := projectAccessMiddleware(updateProjectIntegrationsHandler)
	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 Bad Request, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestProjectAccessMiddleware_SuperAdminBypass(t *testing.T) {
	// Set up mock store with an existing project owned by user-1
	projName := "Existing Project"
	status := "New"
	ownerId := "user-1"
	members := map[string]string{
		"user-1": "owner",
	}
	existingProj := &db.Project{
		ID:      "project-123",
		Name:    &projName,
		Status:  &status,
		OwnerId: &ownerId,
		Members: &members,
	}

	mockStore := &testMockProjectStore{
		projects: map[string]*db.Project{
			"project-123": existingProj,
		},
	}
	store = mockStore

	// Payload to update status to "Onboarding"
	newStatus := "Onboarding"
	payload := db.Project{
		Status: &newStatus,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/project-123", bytes.NewBuffer(body))
	req.SetPathValue("id", "project-123")
	// Super Admin claims (not owner, not member)
	claims := &core.UserClaims{
		Subject:      "super-admin-uid",
		IsSuperAdmin: true,
	}
	req = withTestCtx(req, claims)

	w := httptest.NewRecorder()
	handler := projectAccessMiddleware(updateProjectHandler)
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 OK for Super Admin bypass, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestCheckRepoHandler(t *testing.T) {
	t.Setenv("DEV_REPO_ACCESS_TOKEN", "mock-token") // Ensure token resolve succeeds

	// Mock integrations store to avoid panic in ResolveGitHubToken fallback to OAuth
	oldIntegrationStore := integrations.Store
	integrations.Store = &mockIntegrationStore{}
	defer func() { integrations.Store = oldIntegrationStore }()

	ownerID := "owner-123"
	otherID := "other-123"
	repoURL := "https://github.com/test/repo"

	proj := &db.Project{
		ID:      "proj-123",
		OwnerId: &ownerID,
		Name:    stringPtr("Existing Project"),
		AppIntegrations: []db.ProjectIntegration{
			{
				IntegrationID: "github",
				Enabled:       true,
				Settings: map[string]interface{}{
					"repoUrl": repoURL,
				},
			},
		},
		Members: &map[string]string{
			ownerID: "owner",
		},
	}

	mockStore := &testMockProjectStore{
		projects: map[string]*db.Project{
			"proj-123": proj,
		},
	}
	store = mockStore

	// Test Case 1: Repo exists, user HAS access (is owner)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/check-repo?repoUrl="+repoURL, nil)
	req = withTestCtx(req, &core.UserClaims{Subject: ownerID})

	w := httptest.NewRecorder()
	checkRepoHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["exists"] != true {
		t.Errorf("Expected exists to be true, got %v", resp["exists"])
	}
	if resp["hasAccess"] != true {
		t.Errorf("Expected hasAccess to be true, got %v", resp["hasAccess"])
	}
	if resp["projectId"] != "proj-123" {
		t.Errorf("Expected projectId to be 'proj-123', got '%v'", resp["projectId"])
	}

	// Test Case 2: Repo exists, user DOES NOT have access (should return potentialRole: Editor due to mock token)
	req = httptest.NewRequest(http.MethodGet, "/api/v1/projects/check-repo?repoUrl="+repoURL, nil)
	req = withTestCtx(req, &core.UserClaims{Subject: otherID})

	w = httptest.NewRecorder()
	checkRepoHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}

	json.NewDecoder(w.Body).Decode(&resp)

	if resp["exists"] != true {
		t.Errorf("Expected exists to be true, got %v", resp["exists"])
	}
	if resp["hasAccess"] != false {
		t.Errorf("Expected hasAccess to be false, got %v", resp["hasAccess"])
	}
	if resp["potentialRole"] != "Editor" {
		t.Errorf("Expected potentialRole to be 'Editor', got '%v'", resp["potentialRole"])
	}

	// Test Case 3: Repo does not exist
	req = httptest.NewRequest(http.MethodGet, "/api/v1/projects/check-repo?repoUrl=https://github.com/new/repo", nil)
	req = withTestCtx(req, &core.UserClaims{Subject: ownerID})

	w = httptest.NewRecorder()
	checkRepoHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}

	json.NewDecoder(w.Body).Decode(&resp)

	if resp["exists"] != false {
		t.Errorf("Expected exists to be false, got %v", resp["exists"])
	}
}

func TestJoinProjectHandler(t *testing.T) {
	t.Setenv("DEV_REPO_ACCESS_TOKEN", "mock-token") // Ensure token resolve succeeds

	// Mock integrations store to avoid panic in ResolveGitHubToken fallback to OAuth
	oldIntegrationStore := integrations.Store
	integrations.Store = &mockIntegrationStore{}
	defer func() { integrations.Store = oldIntegrationStore }()

	ownerID := "owner-123"
	otherID := "other-123"
	repoURL := "https://github.com/test/repo"

	proj := &db.Project{
		ID:      "proj-123",
		OwnerId: &ownerID,
		Name:    stringPtr("Existing Project"),
		AppIntegrations: []db.ProjectIntegration{
			{
				IntegrationID: "github",
				Enabled:       true,
				Settings: map[string]interface{}{
					"repoUrl": repoURL,
				},
			},
		},
		Members: &map[string]string{
			ownerID: "owner",
		},
	}

	mockStore := &testMockProjectStore{
		projects: map[string]*db.Project{
			"proj-123": proj,
		},
	}
	store = mockStore

	// Test Case 1: Join successfully (mock-token auto-grants Editor)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/proj-123/join", nil)
	req.SetPathValue("id", "proj-123")
	req = withTestCtx(req, &core.UserClaims{Subject: otherID})

	w := httptest.NewRecorder()
	joinProjectHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["status"] != "SUCCESS" {
		t.Errorf("Expected status 'SUCCESS', got '%v'", resp["status"])
	}
	if resp["role"] != "Editor" {
		t.Errorf("Expected role 'Editor' (due to mock token), got '%v'", resp["role"])
	}

	// Verify member was actually added to project in store
	updatedProj, _ := store.GetProject(context.Background(), "test-organization", "proj-123")
	if updatedProj.Members == nil || (*updatedProj.Members)[otherID] != "Editor" {
		t.Errorf("Expected user to be added as Editor in store, got %v", updatedProj.Members)
	}

	// Test Case 2: Join when already a member (should fail)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/projects/proj-123/join", nil)
	req.SetPathValue("id", "proj-123")
	req = withTestCtx(req, &core.UserClaims{Subject: otherID}) // already joined in Case 1

	w = httptest.NewRecorder()
	joinProjectHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request for duplicate join, got %d. Body: %s", w.Code, w.Body.String())
	}

	// Test Case 3: Join with no token resolved (should fail)
	t.Setenv("DEV_REPO_ACCESS_TOKEN", "") // clear token fallback
	strangerID := "stranger-123"
	req = httptest.NewRequest(http.MethodPost, "/api/v1/projects/proj-123/join", nil)
	req.SetPathValue("id", "proj-123")
	req = withTestCtx(req, &core.UserClaims{Subject: strangerID})

	w = httptest.NewRecorder()
	joinProjectHandler(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected 403 Forbidden for no token, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func stringPtr(s string) *string {
	return &s
}

func TestDeleteProjectHandler_DeletesWorkspaceFolder(t *testing.T) {
	tempMount := t.TempDir()
	oldConfig := appConfig
	appConfig = &core.Configuration{
		WorkspaceMountPath: tempMount,
	}
	defer func() { appConfig = oldConfig }()

	orgID := "org-test-123"
	ownerID := "user-owner"
	memberID := "user-member"
	projID := "proj-delete-123"

	// Create dummy filesystem directories and lock files for owner and member: mountPath + OrgId + UserId + ProjectId
	ownerDir := filepath.Join(tempMount, orgID, ownerID, projID)
	memberDir := filepath.Join(tempMount, orgID, memberID, projID)
	ownerLock := ownerDir + ".lock"
	memberLock := memberDir + ".lock"
	if err := os.MkdirAll(ownerDir, 0755); err != nil {
		t.Fatalf("Failed to create ownerDir: %v", err)
	}
	if err := os.MkdirAll(memberDir, 0755); err != nil {
		t.Fatalf("Failed to create memberDir: %v", err)
	}
	os.WriteFile(ownerLock, []byte("lock"), 0644)
	os.WriteFile(memberLock, []byte("lock"), 0644)

	mockStore := &testMockProjectStore{
		projects: map[string]*db.Project{
			projID: {
				ID:      projID,
				OwnerId: &ownerID,
				Members: &map[string]string{
					ownerID:  "owner",
					memberID: "member",
				},
			},
		},
	}
	store = mockStore

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/"+projID, nil)
	req.SetPathValue("id", projID)
	claims := &core.UserClaims{
		Subject: ownerID,
	}
	req = withTestCtx(req, claims)
	ctx := core.WithOrganizationID(req.Context(), orgID)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	deleteProjectHandler(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("Expected 204 No Content, got %d. Body: %s", w.Code, w.Body.String())
	}

	// Verify project document was deleted from store
	if _, err := mockStore.GetProject(context.Background(), orgID, projID); err == nil {
		t.Errorf("Expected project to be deleted from store")
	}

	// Verify filesystem folders and lock files were deleted: mountPath + OrgId + UserId + ProjectId
	if _, err := os.Stat(ownerDir); !os.IsNotExist(err) {
		t.Errorf("Expected owner workspace folder at %s to be deleted, but it still exists", ownerDir)
	}
	if _, err := os.Stat(memberDir); !os.IsNotExist(err) {
		t.Errorf("Expected member workspace folder at %s to be deleted, but it still exists", memberDir)
	}
	if _, err := os.Stat(ownerLock); !os.IsNotExist(err) {
		t.Errorf("Expected owner lock file at %s to be deleted, but it still exists", ownerLock)
	}
	if _, err := os.Stat(memberLock); !os.IsNotExist(err) {
		t.Errorf("Expected member lock file at %s to be deleted, but it still exists", memberLock)
	}
}

func TestProvisionShowcaseHandler_ExistingProjectAddsMember(t *testing.T) {
	tempMount := t.TempDir()
	oldConfig := appConfig
	appConfig = &core.Configuration{
		WorkspaceMountPath: tempMount,
	}
	defer func() { appConfig = oldConfig }()

	adminID := "simulated-admin-uid"
	userID := "user-requesting-showcase"
	projName := "Foundry Showcase"
	status := "Onboarded"
	visibility := "public"

	// Pre-existing showcase project owned by simulated-admin-uid
	existingProj := &db.Project{
		ID:         "foundry-showcase-project",
		Name:       &projName,
		Status:     &status,
		Visibility: &visibility,
		OwnerId:    &adminID,
		Members: &map[string]string{
			adminID: "owner",
		},
	}

	mockStore := &testMockProjectStore{
		projects: map[string]*db.Project{
			"foundry-showcase-project": existingProj,
		},
	}
	oldStore := store
	store = mockStore
	defer func() { store = oldStore }()

	// Request provision-showcase as user-requesting-showcase
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/provision-showcase", nil)
	claims := &core.UserClaims{
		Subject: userID,
	}
	req = withTestCtx(req, claims)

	w := httptest.NewRecorder()
	provisionShowcaseHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}

	// Verify project in store now has userID in Members
	updatedProj, err := mockStore.GetProject(req.Context(), "test-organization", "foundry-showcase-project")
	if err != nil {
		t.Fatalf("Failed to fetch project after provision: %v", err)
	}

	if updatedProj.Members == nil {
		t.Fatal("Expected project Members not to be nil")
	}

	role, isMember := (*updatedProj.Members)[userID]
	if !isMember {
		t.Errorf("Expected user %s to be added to project Members", userID)
	}
	if role != "owner" {
		t.Errorf("Expected role 'owner', got '%s'", role)
	}
}

type testMockArtifactStore struct {
	db.ArtifactStore
	arts map[string]db.Artifact
}

func (m *testMockArtifactStore) ListArtifactsForProject(ctx context.Context, organizationID string, projectId string) ([]db.Artifact, error) {
	var res []db.Artifact
	for _, a := range m.arts {
		if a.ProjectId == projectId {
			res = append(res, a)
		}
	}
	return res, nil
}

func (m *testMockArtifactStore) CreateArtifact(ctx context.Context, organizationID string, a *db.Artifact) error {
	if m.arts == nil {
		m.arts = make(map[string]db.Artifact)
	}
	m.arts[a.ID] = *a
	return nil
}

func (m *testMockArtifactStore) DeleteArtifact(ctx context.Context, organizationID string, id string) error {
	delete(m.arts, id)
	return nil
}

func TestEnsureShowcaseArtifactsInStore_MismatchedArtifactCleanup(t *testing.T) {
	orgID := "test-organization"
	userID := "user-123"
	projectID := "foundry-showcase-project"

	mockArts := &testMockArtifactStore{
		arts: map[string]db.Artifact{
			"mismatched_1": {
				ID:        "mismatched_1",
				ProjectId: projectID,
				Title:     "brief.md",
				Path:      "docs/specs/plans/briefs/brief-test-project-2025-05-14/brief.md",
			},
		},
	}

	origStore := workspace.ArtifactStore
	workspace.ArtifactStore = mockArts
	defer func() { workspace.ArtifactStore = origStore }()

	err := ensureShowcaseArtifactsInStore(context.Background(), orgID, userID, projectID)
	if err != nil {
		t.Fatalf("ensureShowcaseArtifactsInStore failed: %v", err)
	}

	// Mismatched artifact should have been deleted and replaced with valid showcase template artifacts
	arts, err := mockArts.ListArtifactsForProject(context.Background(), orgID, projectID)
	if err != nil {
		t.Fatalf("Failed to list artifacts: %v", err)
	}

	if len(arts) == 0 {
		t.Fatal("Expected showcase template artifacts to be created")
	}

	for _, a := range arts {
		if a.ID == "mismatched_1" {
			t.Errorf("Expected mismatched artifact 'mismatched_1' to be deleted")
		}
		if a.Path == "docs/specs/plans/briefs/brief-test-project-2025-05-14/brief.md" {
			t.Errorf("Expected invalid path to be removed")
		}
	}
}

func TestCreateProject_InitializesWorkspace(t *testing.T) {
	tempMount := t.TempDir()
	tempBmadRoot := t.TempDir()

	workspace.AppConfig = &core.Configuration{
		WorkspaceMountPath: tempMount,
		BmadRootPath:       tempBmadRoot,
	}
	workspace.SetGitClient(workspace.NewGitClient(tempMount))
	defer workspace.ResetGitClient()
	t.Setenv("WORKSPACE_MOUNT_PATH", tempMount)
	t.Setenv("BMAD_ROOT_PATH", tempBmadRoot)

	// Setup source BMAD directory
	srcBmad := filepath.Join(tempBmadRoot, "_bmad")
	_ = os.MkdirAll(srcBmad, 0755)
	_ = os.WriteFile(filepath.Join(srcBmad, "config.toml"), []byte("output_folder = \"docs\"\n"), 0644)

	srcSkill := filepath.Join(tempBmadRoot, "skills", "bmad-prd")
	_ = os.MkdirAll(srcSkill, 0755)
	_ = os.WriteFile(filepath.Join(srcSkill, "SKILL.md"), []byte("# PRD Skill"), 0644)

	mockStore := &testMockProjectStore{
		projects: make(map[string]*db.Project),
	}
	store = mockStore
	origProjectStoreInstance := db.ProjectStoreInstance
	db.ProjectStoreInstance = mockStore
	defer func() { db.ProjectStoreInstance = origProjectStoreInstance }()

	projName := "Project With Immediate Workspace"
	payload := db.Project{
		Name: &projName,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", bytes.NewBuffer(body))
	claims := &core.UserClaims{
		Subject: "test-user-save",
		Roles:   []string{string(core.RoleProjectCreator)},
	}
	req = withTestCtx(req, claims)

	w := httptest.NewRecorder()
	createProjectHandler(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected status 201 Created, got %d. Body: %s", w.Code, w.Body.String())
	}

	var created db.Project
	if err := json.NewDecoder(w.Body).Decode(&created); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify that the workspace and skill folders exist immediately on disk
	projectRoot := filepath.Join(tempMount, "test-organization", "test-user-save", created.ID)
	if !workspace.VerifySkillsPresence(projectRoot) {
		t.Errorf("Expected workspace folders to be initialized on disk immediately after project is created at %s", projectRoot)
	}
}





