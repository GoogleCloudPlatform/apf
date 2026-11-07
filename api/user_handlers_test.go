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
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"google.com/apf/api/core"
	"google.com/apf/api/db"
	"google.com/apf/api/integrations"
)

// Mocks

type mockUserStore struct {
	db.UserStore
	users map[string]*db.User
}

func (m *mockUserStore) GetUser(ctx context.Context, organizationID string, id string) (*db.User, error) {
	if m.users == nil {
		return nil, db.ErrNotFound
	}
	u, ok := m.users[id]
	if !ok {
		return nil, db.ErrNotFound
	}
	return u, nil
}

func (m *mockUserStore) GetUserByEmail(ctx context.Context, organizationID string, email string) (*db.User, error) {
	if email == "" {
		return nil, db.ErrNotFound
	}
	if m.users == nil {
		return nil, db.ErrNotFound
	}
	for _, u := range m.users {
		if strings.EqualFold(u.Email, email) {
			return u, nil
		}
	}
	return nil, db.ErrNotFound
}

func (m *mockUserStore) DeleteUser(ctx context.Context, organizationID string, id string) error {
	if m.users == nil {
		return db.ErrNotFound
	}
	u, ok := m.users[id]
	if !ok {
		return db.ErrNotFound
	}

	// Last admin check
	if u.Role == db.RoleOrgAdmin {
		adminCount := 0
		for _, usr := range m.users {
			if usr.Role == db.RoleOrgAdmin && usr.UserID != "" && !strings.Contains(usr.ID, "@") {
				adminCount++
			}
		}
		if adminCount <= 1 {
			return db.ErrLastAdmin
		}
	}

	delete(m.users, id)
	return nil
}

func (m *mockUserStore) ListUsers(ctx context.Context, organizationID string) ([]db.User, error) {
	var list []db.User
	for _, u := range m.users {
		list = append(list, *u)
	}
	return list, nil
}

func (m *mockUserStore) GetUserCount(ctx context.Context, organizationID string) (int, error) {
	users, err := m.ListUsers(ctx, organizationID)
	if err != nil {
		return 0, err
	}
	return len(users), nil
}

func (m *mockUserStore) DiscoverUserOrganizations(ctx context.Context, email string) ([]string, error) {
	var orgs []string
	for _, u := range m.users {
		if u.Email == email {
			orgs = append(orgs, "org1")
		}
	}
	return orgs, nil
}

func (m *mockUserStore) CreateUser(ctx context.Context, organizationID string, user *db.User) error {
	if m.users == nil {
		m.users = make(map[string]*db.User)
	}
	key := user.UserID
	if key == "" {
		key = user.Email
	}
	m.users[key] = user
	return nil
}

func (m *mockUserStore) UpdateUserRole(ctx context.Context, organizationID string, id string, role db.UserRole) error {
	if m.users == nil {
		return db.ErrNotFound
	}
	u, ok := m.users[id]
	if !ok {
		return db.ErrNotFound
	}

	// Last admin check
	if u.Role == db.RoleOrgAdmin && role != db.RoleOrgAdmin {
		adminCount := 0
		for _, usr := range m.users {
			if usr.Role == db.RoleOrgAdmin && usr.UserID != "" && !strings.Contains(usr.ID, "@") {
				adminCount++
			}
		}
		if adminCount <= 1 {
			return db.ErrLastAdmin
		}
	}

	u.Role = role
	return nil
}

func (m *mockUserStore) GetOrCreateUser(ctx context.Context, organizationID string, id string) (*db.User, error) {
	if m.users == nil {
		m.users = make(map[string]*db.User)
	}
	u, ok := m.users[id]
	if !ok {
		var invitedRole db.UserRole
		var email string
		claims, ok := ctx.Value(core.UserClaimsKey).(*core.UserClaims)
		if ok && claims != nil && claims.Username != "" {
			email = claims.Username
			invite, inviteExists := m.users[email]
			if inviteExists {
				invitedRole = invite.Role
				delete(m.users, email)
			}
		}

		userRole := db.RoleOrgViewer
		if invitedRole != "" {
			userRole = invitedRole
		}

		u = &db.User{
			ID:     id,
			UserID: id,
			Email:  email,
			Role:   userRole,
			Preferences: db.UserPreferences{
				Theme: "dark",
			},
			TourStatus: &db.TourStatus{
				Completed:         false,
				CompletedSections: []string{},
				LastActiveStep: db.LastActiveStep{
					SectionID: "",
					StepIndex: 0,
				},
			},
		}
		m.users[id] = u
	}
	return u, nil
}

func (m *mockUserStore) UpdateUserPreferences(ctx context.Context, organizationID string, id string, prefs *db.UserPreferences) error {
	if m.users == nil {
		m.users = make(map[string]*db.User)
	}
	u, ok := m.users[id]
	if !ok {
		u = &db.User{ID: id}
		m.users[id] = u
	}
	u.Preferences = *prefs
	return nil
}

func (m *mockUserStore) UpdateTourStatus(ctx context.Context, organizationID string, id string, status *db.TourStatus) error {
	if m.users == nil {
		m.users = make(map[string]*db.User)
	}
	u, ok := m.users[id]
	if !ok {
		u = &db.User{ID: id}
		m.users[id] = u
	}
	u.TourStatus = status
	return nil
}

func (m *mockUserStore) ListUserOrganizations(ctx context.Context, userID string) ([]string, error) {
	return []string{"default-organization"}, nil
}

type mockAppCredentialStore struct {
	db.AppCredentialStore
	creds map[string]map[string]*db.AppCredential
}

func (m *mockAppCredentialStore) GetCredential(ctx context.Context, organizationID string, userId, appId string) (*db.AppCredential, error) {
	if m.creds == nil {
		m.creds = make(map[string]map[string]*db.AppCredential)
	}
	userMap, ok := m.creds[userId]
	if !ok {
		return nil, errors.New("not found")
	}
	cred, ok := userMap[appId]
	if !ok {
		return nil, errors.New("not found")
	}
	return cred, nil
}

func (m *mockAppCredentialStore) SaveCredential(ctx context.Context, organizationID string, userId, appId string, cred *db.AppCredential) error {
	if m.creds == nil {
		m.creds = make(map[string]map[string]*db.AppCredential)
	}
	userMap, ok := m.creds[userId]
	if !ok {
		userMap = make(map[string]*db.AppCredential)
		m.creds[userId] = userMap
	}
	userMap[appId] = cred
	return nil
}

func (m *mockAppCredentialStore) DeleteCredential(ctx context.Context, organizationID string, userId, appId string) error {
	if m.creds == nil {
		return nil
	}
	userMap, ok := m.creds[userId]
	if ok {
		delete(userMap, appId)
	}
	return nil
}

type mockUserProjectStore struct {
	db.UserProjectStore
	settings map[string]*db.UserProject
}

func (m *mockUserProjectStore) GetProjectAppSetting(ctx context.Context, organizationID string, userId, projectId, appId string) (*db.UserProject, error) {
	if m.settings == nil {
		m.settings = make(map[string]*db.UserProject)
	}
	key := userId + "_" + projectId + "_" + appId
	s, ok := m.settings[key]
	if !ok {
		return &db.UserProject{
			UserID:    userId,
			ProjectID: projectId,
			AppID:     appId,
			Enabled:   false,
		}, nil
	}
	return s, nil
}

func (m *mockUserProjectStore) SaveProjectAppSetting(ctx context.Context, organizationID string, setting *db.UserProject) error {
	if m.settings == nil {
		m.settings = make(map[string]*db.UserProject)
	}
	key := setting.UserID + "_" + setting.ProjectID + "_" + setting.AppID
	m.settings[key] = setting
	return nil
}

type mockConfigStore struct {
	db.ConfigStore
	integrations    []db.AppIntegration
	orgIntegrations []db.AppIntegration
}

func (m *mockConfigStore) ListGlobalAppIntegrations(ctx context.Context) ([]db.AppIntegration, error) {
	return m.integrations, nil
}

func (m *mockConfigStore) ListOrgAppIntegrations(ctx context.Context, orgID string) ([]db.AppIntegration, error) {
	return m.orgIntegrations, nil
}

func (m *mockConfigStore) SaveOrgAppIntegration(ctx context.Context, orgID string, integration *db.AppIntegration) error {
	return nil
}

type mockProjectStore struct {
	db.ProjectStore
	projects map[string]*db.Project
}

func (m *mockProjectStore) GetProject(ctx context.Context, organizationID string, id string) (*db.Project, error) {
	if m.projects == nil {
		return nil, errors.New("project not found")
	}
	p, ok := m.projects[id]
	if !ok {
		return nil, errors.New("project not found")
	}
	return p, nil
}

// Tests

func TestGetUserPreferencesHandler(t *testing.T) {
	userStore = &mockUserStore{
		users: map[string]*db.User{
			"user123": {
				ID: "user123",
				Preferences: db.UserPreferences{
					Theme: "light",
				},
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/user123/preferences", nil)
	req.SetPathValue("id", "user123")
	claims := &core.UserClaims{Subject: "user123"}
	req = withTestCtx(req, claims)

	w := httptest.NewRecorder()
	getUserPreferencesHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var prefs db.UserPreferences
	_ = json.NewDecoder(w.Body).Decode(&prefs)
	if prefs.Theme != "light" {
		t.Errorf("Expected theme 'light', got '%s'", prefs.Theme)
	}

	// Forbidden case
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/users/user456/preferences", nil)
	req2.SetPathValue("id", "user456")
	req2 = withTestCtx(req2, claims)

	w2 := httptest.NewRecorder()
	getUserPreferencesHandler(w2, req2)

	if w2.Code != http.StatusForbidden {
		t.Errorf("Expected forbidden access 403, got %d", w2.Code)
	}
}

func TestUpdateUserPreferencesHandler(t *testing.T) {
	userStore = &mockUserStore{}

	payload := db.UserPreferences{Theme: "light"}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/user123/preferences", bytes.NewBuffer(body))
	req.SetPathValue("id", "user123")
	claims := &core.UserClaims{Subject: "user123"}
	req = withTestCtx(req, claims)

	w := httptest.NewRecorder()
	updateUserPreferencesHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	mock := userStore.(*mockUserStore)
	if mock.users["user123"].Preferences.Theme != "light" {
		t.Errorf("Expected theme to be updated to 'light'")
	}
}

func TestGetUserTourStatusHandler(t *testing.T) {
	userStore = &mockUserStore{
		users: map[string]*db.User{
			"user123": {
				ID: "user123",
				TourStatus: &db.TourStatus{
					Completed:         true,
					CompletedSections: []string{"catalog", "settings"},
					LastActiveStep: db.LastActiveStep{
						SectionID: "workspace",
						StepIndex: 1,
					},
				},
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/user123/tour-status", nil)
	req.SetPathValue("id", "user123")
	claims := &core.UserClaims{Subject: "user123"}
	req = withTestCtx(req, claims)

	w := httptest.NewRecorder()
	getUserTourStatusHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var status db.TourStatus
	_ = json.NewDecoder(w.Body).Decode(&status)
	if !status.Completed {
		t.Errorf("Expected completed to be true")
	}
	if len(status.CompletedSections) != 2 {
		t.Errorf("Expected 2 completed sections, got %d", len(status.CompletedSections))
	}

	// Forbidden case
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/users/user456/tour-status", nil)
	req2.SetPathValue("id", "user456")
	req2 = withTestCtx(req2, claims)

	w2 := httptest.NewRecorder()
	getUserTourStatusHandler(w2, req2)

	if w2.Code != http.StatusForbidden {
		t.Errorf("Expected forbidden access 403, got %d", w2.Code)
	}

	// URL encoded user ID case
	userStore = &mockUserStore{
		users: map[string]*db.User{
			"accounts.google.com:1077645123": {
				ID: "accounts.google.com:1077645123",
				TourStatus: &db.TourStatus{
					Completed: false,
				},
			},
		},
	}
	req3 := httptest.NewRequest(http.MethodGet, "/api/v1/users/accounts.google.com%3A1077645123/tour-status", nil)
	req3.SetPathValue("id", "accounts.google.com%3A1077645123")
	encodedClaims := &core.UserClaims{Subject: "accounts.google.com:1077645123"}
	req3 = withTestCtx(req3, encodedClaims)

	w3 := httptest.NewRecorder()
	getUserTourStatusHandler(w3, req3)

	if w3.Code != http.StatusOK {
		t.Errorf("Expected status 200 for URL encoded user ID, got %d", w3.Code)
	}
}

func TestUpdateUserTourStatusHandler(t *testing.T) {
	userStore = &mockUserStore{}

	payload := db.TourStatus{
		Completed:         false,
		CompletedSections: []string{"catalog"},
		LastActiveStep: db.LastActiveStep{
			SectionID: "settings",
			StepIndex: 3,
		},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/user123/tour-status", bytes.NewBuffer(body))
	req.SetPathValue("id", "user123")
	claims := &core.UserClaims{Subject: "user123"}
	req = withTestCtx(req, claims)

	w := httptest.NewRecorder()
	updateUserTourStatusHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	mock := userStore.(*mockUserStore)
	savedStatus := mock.users["user123"].TourStatus
	if savedStatus == nil {
		t.Fatal("Expected TourStatus to be saved, got nil")
	}
	if savedStatus.Completed {
		t.Errorf("Expected completed to be false")
	}
	if savedStatus.LastActiveStep.SectionID != "settings" || savedStatus.LastActiveStep.StepIndex != 3 {
		t.Errorf("Expected last active step to be settings:3, got %s:%d", savedStatus.LastActiveStep.SectionID, savedStatus.LastActiveStep.StepIndex)
	}
}

func TestGetAppCredentialHandler(t *testing.T) {
	appCredentialStore = &mockAppCredentialStore{
		creds: map[string]map[string]*db.AppCredential{
			"user123": {
				"figma": {
					IntegrationID: "figma",
					Fields: []db.CredentialAttribute{
						{Name: "username", DisplayName: "Username", Value: "figma-user", IsSecret: false},
						{Name: "password", DisplayName: "Password", Value: "plain-secret-pass", IsSecret: true},
					},
				},
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/user123/apps/figma/credentials", nil)
	req.SetPathValue("id", "user123")
	req.SetPathValue("appId", "figma")
	claims := &core.UserClaims{Subject: "user123"}
	req = withTestCtx(req, claims)

	w := httptest.NewRecorder()
	getAppCredentialHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var dto AppCredentialDTO
	_ = json.NewDecoder(w.Body).Decode(&dto)

	var usernameVal, passwordVal string
	for _, field := range dto.Fields {
		switch field.Name {
		case "username":
			usernameVal = field.Value
		case "password":
			passwordVal = field.Value
		}
	}

	if usernameVal != "figma-user" {
		t.Errorf("Expected username 'figma-user', got '%s'", usernameVal)
	}
	if passwordVal != credentialMaskPlaceholder {
		t.Errorf("Expected password to be masked, got '%s'", passwordVal)
	}

	// Test Not Found
	reqNotFound := httptest.NewRequest(http.MethodGet, "/api/v1/users/user123/apps/stitch/credentials", nil)
	reqNotFound.SetPathValue("id", "user123")
	reqNotFound.SetPathValue("appId", "stitch")
	reqNotFound = withTestCtx(reqNotFound, claims)

	wNotFound := httptest.NewRecorder()
	getAppCredentialHandler(wNotFound, reqNotFound)

	if wNotFound.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", wNotFound.Code)
	}
}

func TestSaveAppCredentialHandler(t *testing.T) {
	appCredentialStore = &mockAppCredentialStore{
		creds: map[string]map[string]*db.AppCredential{
			"user123": {
				"figma": {
					IntegrationID: "figma",
					Fields: []db.CredentialAttribute{
						{Name: "username", DisplayName: "Username", Value: "old-username", IsSecret: false},
						{Name: "password", DisplayName: "Password", Value: "old-secret-password", IsSecret: true},
					},
				},
			},
		},
	}

	// Payload uses standard password placeholder but updates username and adds access token
	payload := AppCredentialDTO{
		IntegrationID: "figma",
		Fields: []db.CredentialAttribute{
			{Name: "username", DisplayName: "Username", Value: "new-username", IsSecret: false},
			{Name: "password", DisplayName: "Password", Value: credentialMaskPlaceholder, IsSecret: true},
			{Name: "accessToken", DisplayName: "Access Token", Value: "new-access-token", IsSecret: true},
		},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/user123/apps/figma/credentials", bytes.NewBuffer(body))
	req.SetPathValue("id", "user123")
	req.SetPathValue("appId", "figma")
	claims := &core.UserClaims{Subject: "user123"}
	req = withTestCtx(req, claims)

	w := httptest.NewRecorder()
	saveAppCredentialHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	mock := appCredentialStore.(*mockAppCredentialStore)
	saved := mock.creds["user123"]["figma"]

	var usernameVal, passwordVal, tokenVal string
	for _, field := range saved.Fields {
		switch field.Name {
		case "username":
			usernameVal = field.Value
		case "password":
			passwordVal = field.Value
		case "accessToken":
			tokenVal = field.Value
		}
	}

	if usernameVal != "new-username" {
		t.Errorf("Expected updated username, got '%s'", usernameVal)
	}
	if passwordVal != "old-secret-password" {
		t.Errorf("Expected masked password placeholder to preserve existing value, got '%s'", passwordVal)
	}
	if tokenVal != "new-access-token" {
		t.Errorf("Expected new access token to be saved, got '%s'", tokenVal)
	}
}

func TestSaveAppCredentialHandler_GitHubValidation(t *testing.T) {
	appCredentialStore = &mockAppCredentialStore{
		creds: make(map[string]map[string]*db.AppCredential),
	}

	// 1. Missing email should fail
	payloadNoEmail := AppCredentialDTO{
		IntegrationID: "github",
		Fields: []db.CredentialAttribute{
			{Name: "pat", DisplayName: "Personal Access Token", Enabled: true, Required: true, Value: "ghp_test123", IsSecret: true},
			{Name: "email", DisplayName: "Git Author Email", Enabled: true, Required: true, Value: "", IsSecret: false},
		},
	}
	bodyNoEmail, _ := json.Marshal(payloadNoEmail)
	req1 := httptest.NewRequest(http.MethodPut, "/api/v1/users/user123/apps/github/credentials", bytes.NewBuffer(bodyNoEmail))
	req1.SetPathValue("id", "user123")
	req1.SetPathValue("appId", "github")
	claims := &core.UserClaims{Subject: "user123"}
	req1 = withTestCtx(req1, claims)
	w1 := httptest.NewRecorder()
	saveAppCredentialHandler(w1, req1)

	if w1.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 when email is missing, got %d. Body: %s", w1.Code, w1.Body.String())
	}

	// 2. Both PAT and Email provided should succeed
	payloadValid := AppCredentialDTO{
		IntegrationID: "github",
		Fields: []db.CredentialAttribute{
			{Name: "pat", DisplayName: "Personal Access Token", Enabled: true, Required: true, Value: "ghp_test123", IsSecret: true},
			{Name: "email", DisplayName: "Git Author Email", Enabled: true, Required: true, Value: "octocat@github.com", IsSecret: false},
		},
	}
	bodyValid, _ := json.Marshal(payloadValid)
	req2 := httptest.NewRequest(http.MethodPut, "/api/v1/users/user123/apps/github/credentials", bytes.NewBuffer(bodyValid))
	req2.SetPathValue("id", "user123")
	req2.SetPathValue("appId", "github")
	req2 = withTestCtx(req2, claims)
	w2 := httptest.NewRecorder()
	saveAppCredentialHandler(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("Expected status 200 when both PAT and email provided, got %d. Body: %s", w2.Code, w2.Body.String())
	}
}

func TestGetProjectAppSettingHandler(t *testing.T) {
	userProjectStore = &mockUserProjectStore{
		settings: map[string]*db.UserProject{
			"user123_project456_figma": {
				UserID:       "user123",
				ProjectID:    "project456",
				AppID:        "figma",
				Enabled:      true,
				CredentialID: "figma",
			},
		},
	}

	owner := "user123"
	store = &mockProjectStore{
		projects: map[string]*db.Project{
			"project456": {
				ID:      "project456",
				OwnerId: &owner,
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/user123/projects/project456/apps/figma/credentials", nil)
	req.SetPathValue("userId", "user123")
	req.SetPathValue("projectId", "project456")
	req.SetPathValue("appId", "figma")
	claims := &core.UserClaims{Subject: "user123"}
	req = withTestCtx(req, claims)

	w := httptest.NewRecorder()
	getProjectAppSettingHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var res db.UserProject
	_ = json.NewDecoder(w.Body).Decode(&res)

	if !res.Enabled {
		t.Error("Expected enabled project-scoped figma integration to be true")
	}
	if res.CredentialID != "figma" {
		t.Errorf("Expected linked CredentialID 'figma', got '%s'", res.CredentialID)
	}
}

func TestSaveProjectAppSettingHandler(t *testing.T) {
	userProjectStore = &mockUserProjectStore{}
	appCredentialStore = &mockAppCredentialStore{
		creds: map[string]map[string]*db.AppCredential{
			"user123": {
				"figma": {IntegrationID: "figma"},
			},
		},
	}

	owner := "user123"
	store = &mockProjectStore{
		projects: map[string]*db.Project{
			"project456": {
				ID:      "project456",
				OwnerId: &owner,
			},
		},
	}

	payload := UserProjectDTO{
		Enabled:      true,
		CredentialID: "figma",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/user123/projects/project456/apps/figma/credentials", bytes.NewBuffer(body))
	req.SetPathValue("userId", "user123")
	req.SetPathValue("projectId", "project456")
	req.SetPathValue("appId", "figma")
	claims := &core.UserClaims{Subject: "user123"}
	req = withTestCtx(req, claims)

	w := httptest.NewRecorder()
	saveProjectAppSettingHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	mock := userProjectStore.(*mockUserProjectStore)
	saved := mock.settings["user123_project456_figma"]

	if !saved.Enabled {
		t.Error("Expected project figma setting to be saved as enabled")
	}
	if saved.CredentialID != "figma" {
		t.Errorf("Expected credential ID link to be saved, got '%s'", saved.CredentialID)
	}
}

func TestGetAppIntegrationsHandler(t *testing.T) {
	configStore = &mockConfigStore{
		integrations: []db.AppIntegration{
			{ID: "github", Name: "GitHub", AuthType: "oauth", Icon: "github"},
			{ID: "figma", Name: "Figma", AuthType: "apikey", Icon: "figma"},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/config/app-integrations", nil)
	claims := &core.UserClaims{Subject: "user123"}
	req = withTestCtx(req, claims)

	w := httptest.NewRecorder()
	getAppIntegrationsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var list []db.AppIntegration
	_ = json.NewDecoder(w.Body).Decode(&list)

	if len(list) != 2 {
		t.Errorf("Expected list of 2 integrations, got %d", len(list))
	}
	if list[0].Name != "GitHub" || list[1].Name != "Figma" {
		t.Errorf("Integrations names mismatch, got first: %s, second: %s", list[0].Name, list[1].Name)
	}
}

type mockIntegration struct {
	valid bool
	err   error
}

func (m *mockIntegration) Validate(ctx context.Context, cred *db.AppCredential) (bool, error) {
	return m.valid, m.err
}

func TestValidateAppCredentialHandler(t *testing.T) {
	appCredentialStore = &mockAppCredentialStore{
		creds: map[string]map[string]*db.AppCredential{
			"user123": {
				"figma": {
					IntegrationID: "figma",
					Fields: []db.CredentialAttribute{
						{Name: "accessToken", DisplayName: "Token", Value: "figma-mock-key-abc", IsSecret: true},
					},
				},
			},
		},
	}

	// Register a dynamic test mock
	integrations.Registry["FigmaIntegration"] = &mockIntegration{valid: true}

	// Case 1: Validate already saved credentials (no payload body passed)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/user123/apps/figma/credentials/validate", nil)
	req.SetPathValue("id", "user123")
	req.SetPathValue("appId", "figma")
	claims := &core.UserClaims{Subject: "user123"}
	req = withTestCtx(req, claims)

	w := httptest.NewRecorder()
	validateAppCredentialHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var res map[string]interface{}
	_ = json.NewDecoder(w.Body).Decode(&res)
	if res["valid"] != true {
		t.Errorf("Expected valid flag to be true, got %v", res["valid"])
	}

	// Case 2: Validate a specific unsaved payload (payload body passed)
	payload := AppCredentialDTO{
		IntegrationID: "figma",
		Fields: []db.CredentialAttribute{
			{Name: "accessToken", DisplayName: "Token", Value: "new-user-unsaved-key", IsSecret: true},
		},
	}
	body, _ := json.Marshal(payload)
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/users/user123/apps/figma/credentials/validate", bytes.NewBuffer(body))
	req2.SetPathValue("id", "user123")
	req2.SetPathValue("appId", "figma")
	req2 = withTestCtx(req2, claims)

	w2 := httptest.NewRecorder()
	validateAppCredentialHandler(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w2.Code)
	}

	var res2 map[string]interface{}
	_ = json.NewDecoder(w2.Body).Decode(&res2)
	if res2["valid"] != true {
		t.Errorf("Expected valid flag to be true, got %v", res2["valid"])
	}

	// Case 3: Validation failure check
	integrations.Registry["FigmaIntegration"] = &mockIntegration{valid: false, err: errors.New("invalid API token")}
	req3 := httptest.NewRequest(http.MethodPost, "/api/v1/users/user123/apps/figma/credentials/validate", nil)
	req3.SetPathValue("id", "user123")
	req3.SetPathValue("appId", "figma")
	req3 = withTestCtx(req3, claims)

	w3 := httptest.NewRecorder()
	validateAppCredentialHandler(w3, req3)

	var res3 map[string]interface{}
	_ = json.NewDecoder(w3.Body).Decode(&res3)
	if res3["valid"] == true || res3["error"] != "invalid API token" {
		t.Errorf("Expected invalid with validation error, got valid: %v, error: %v", res3["valid"], res3["error"])
	}

	// Case 4: GitHub validation without email succeeds and returns profile email
	req4Body, _ := json.Marshal(AppCredentialDTO{
		IntegrationID: "github",
		Fields: []db.CredentialAttribute{
			{Name: "pat", DisplayName: "Token", Value: "simulated_pat_token", IsSecret: true},
		},
	})
	req4 := httptest.NewRequest(http.MethodPost, "/api/v1/users/user123/apps/github/credentials/validate", bytes.NewBuffer(req4Body))
	req4.SetPathValue("id", "user123")
	req4.SetPathValue("appId", "github")
	req4 = withTestCtx(req4, claims)

	w4 := httptest.NewRecorder()
	validateAppCredentialHandler(w4, req4)

	if w4.Code != http.StatusOK {
		t.Errorf("Expected status 200 for GitHub validation, got %d", w4.Code)
	}
	var res4 map[string]interface{}
	_ = json.NewDecoder(w4.Body).Decode(&res4)
	if res4["valid"] != true {
		t.Errorf("Expected GitHub validation to be valid without email, got %v", res4)
	}
	if res4["email"] != "octocat@github.com" {
		t.Errorf("Expected GitHub validation to return profile email 'octocat@github.com', got %v", res4["email"])
	}
}

func TestSaveBulkProjectAppSettingsHandler(t *testing.T) {
	userProjectStore = &mockUserProjectStore{}
	appCredentialStore = &mockAppCredentialStore{
		creds: map[string]map[string]*db.AppCredential{
			"user123": {
				"figma":  {IntegrationID: "figma"},
				"github": {IntegrationID: "github"},
			},
		},
	}

	owner := "user123"
	store = &mockProjectStore{
		projects: map[string]*db.Project{
			"project456": {
				ID:      "project456",
				OwnerId: &owner,
			},
		},
	}

	payload := []BulkUserProjectDTO{
		{
			AppID:        "figma",
			Enabled:      true,
			CredentialID: "figma",
		},
		{
			AppID:        "github",
			Enabled:      true,
			CredentialID: "github",
		},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/user123/projects/project456/apps/credentials", bytes.NewBuffer(body))
	req.SetPathValue("userId", "user123")
	req.SetPathValue("projectId", "project456")
	claims := &core.UserClaims{Subject: "user123"}
	req = withTestCtx(req, claims)

	w := httptest.NewRecorder()
	saveBulkProjectAppSettingsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	mock := userProjectStore.(*mockUserProjectStore)
	savedFigma := mock.settings["user123_project456_figma"]
	savedGithub := mock.settings["user123_project456_github"]

	if savedFigma == nil || !savedFigma.Enabled {
		t.Error("Expected project figma setting to be saved as enabled")
	}
	if savedGithub == nil || !savedGithub.Enabled {
		t.Error("Expected project github setting to be saved as enabled")
	}
}

func TestSaveProjectAppSettingHandler_IsReadOnly(t *testing.T) {
	origStore := integrations.Store
	defer func() { integrations.Store = origStore }()
	integrations.Store = &mockIntegrationStore{}

	var pushPermissionResponse bool
	var prPermissionResponse bool
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-github-pat" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// 1. GET /repos/{owner}/{repo}
		if r.Method == "GET" && strings.HasSuffix(r.URL.Path, "/repos/owner/test-repo") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(fmt.Sprintf(`{
				"permissions": {
					"pull": true,
					"push": %t
				}
			}`, pushPermissionResponse)))
			return
		}

		// 2. PUT /repos/{owner}/{repo}/contents/...
		if r.Method == "PUT" && strings.Contains(r.URL.Path, "/contents/permission-check-dummy-file-do-not-create") {
			if pushPermissionResponse {
				w.WriteHeader(http.StatusUnprocessableEntity) // 422 means write allowed
			} else {
				w.WriteHeader(http.StatusForbidden) // 403 means no write
			}
			return
		}

		// 3. POST /repos/{owner}/{repo}/pulls
		if r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/pulls") {
			if prPermissionResponse {
				w.WriteHeader(http.StatusUnprocessableEntity) // 422 means PR allowed
			} else {
				w.WriteHeader(http.StatusForbidden) // 403 means no PR
			}
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	defer integrations.SetGitHubAPIURLForTesting("https://api.github.com")
	integrations.SetGitHubAPIURLForTesting(ts.URL)

	userProjectStore = &mockUserProjectStore{}
	appCredentialStore = &mockAppCredentialStore{
		creds: map[string]map[string]*db.AppCredential{
			"user123": {
				"github": {IntegrationID: "github"},
			},
		},
	}

	owner := "user123"
	store = &mockProjectStore{
		projects: map[string]*db.Project{
			"project456": {
				ID:      "project456",
				OwnerId: &owner,
				AppIntegrations: []db.ProjectIntegration{
					{
						IntegrationID: "github",
						Enabled:       true,
						Settings: map[string]interface{}{
							"repoUrl": "https://github.com/owner/test-repo",
						},
					},
				},
			},
		},
	}

	// Case A: Token has write (push) permission. IsReadOnly should be false.
	pushPermissionResponse = true
	prPermissionResponse = true
	payload := UserProjectDTO{
		Enabled:      true,
		CredentialID: "github",
		IsOverridden: true,
		Fields: []db.CredentialAttribute{
			{Name: "pat", Value: "test-github-pat", IsSecret: true},
		},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/user123/projects/project456/apps/github/credentials", bytes.NewBuffer(body))
	req.SetPathValue("userId", "user123")
	req.SetPathValue("projectId", "project456")
	req.SetPathValue("appId", "github")
	claims := &core.UserClaims{Subject: "user123"}
	req = withTestCtx(req, claims)

	w := httptest.NewRecorder()
	saveProjectAppSettingHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	mock := userProjectStore.(*mockUserProjectStore)
	saved := mock.settings["user123_project456_github"]
	if saved.IsReadOnly != false {
		t.Errorf("Expected IsReadOnly to be false when token has write permission, got true")
	}
	if saved.CanWrite != true {
		t.Errorf("Expected CanWrite to be true, got false")
	}
	if saved.CanSubmitPullRequest != true {
		t.Errorf("Expected CanSubmitPullRequest to be true, got false")
	}

	// Case B: Token has read-only (no push) permission, but can PR. IsReadOnly should be false (user has PR permission), CanSubmitPR should be true.
	pushPermissionResponse = false
	prPermissionResponse = true
	body2, _ := json.Marshal(payload)

	req2 := httptest.NewRequest(http.MethodPut, "/api/v1/users/user123/projects/project456/apps/github/credentials", bytes.NewBuffer(body2))
	req2.SetPathValue("userId", "user123")
	req2.SetPathValue("projectId", "project456")
	req2.SetPathValue("appId", "github")
	req2 = withTestCtx(req2, claims)

	w2 := httptest.NewRecorder()
	saveProjectAppSettingHandler(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w2.Code)
	}

	saved2 := mock.settings["user123_project456_github"]
	if saved2.IsReadOnly != false {
		t.Errorf("Expected IsReadOnly to be false when token has PR permission, got true")
	}
	if saved2.CanWrite != false {
		t.Errorf("Expected CanWrite to be false when token has only PR permission, got true")
	}
	if saved2.CanSubmitPullRequest != true {
		t.Errorf("Expected CanSubmitPullRequest to be true, got false")
	}

	// Case C: User requested read-only, but token has write permission. IsReadOnly should be true, but CanWrite should be true.
	pushPermissionResponse = true
	prPermissionResponse = true
	payloadReadOnly := UserProjectDTO{
		Enabled:      true,
		CredentialID: "github",
		IsOverridden: true,
		IsReadOnly:   true, // Explicitly requested
		Fields: []db.CredentialAttribute{
			{Name: "pat", Value: "test-github-pat", IsSecret: true},
		},
	}
	body3, _ := json.Marshal(payloadReadOnly)

	req3 := httptest.NewRequest(http.MethodPut, "/api/v1/users/user123/projects/project456/apps/github/credentials", bytes.NewBuffer(body3))
	req3.SetPathValue("userId", "user123")
	req3.SetPathValue("projectId", "project456")
	req3.SetPathValue("appId", "github")
	req3 = withTestCtx(req3, claims)

	w3 := httptest.NewRecorder()
	saveProjectAppSettingHandler(w3, req3)

	if w3.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w3.Code)
	}

	saved3 := mock.settings["user123_project456_github"]
	if saved3.IsReadOnly != true {
		t.Errorf("Expected IsReadOnly to be true when user explicitly requested it, got false")
	}
	if saved3.CanWrite != true {
		t.Errorf("Expected CanWrite to be true, got false")
	}
	if saved3.CanSubmitPullRequest != true {
		t.Errorf("Expected CanSubmitPullRequest to be true, got false")
	}

	// Case D: No token found. IsReadOnly should default to true, permissions to false.
	payloadNoToken := UserProjectDTO{
		Enabled:      true,
		CredentialID: "github",
		IsOverridden: true,
		Fields:       []db.CredentialAttribute{}, // No PAT
	}
	body4, _ := json.Marshal(payloadNoToken)

	req4 := httptest.NewRequest(http.MethodPut, "/api/v1/users/user123/projects/project456/apps/github/credentials", bytes.NewBuffer(body4))
	req4.SetPathValue("userId", "user123")
	req4.SetPathValue("projectId", "project456")
	req4.SetPathValue("appId", "github")
	req4 = withTestCtx(req4, claims)

	w4 := httptest.NewRecorder()
	saveProjectAppSettingHandler(w4, req4)

	if w4.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w4.Code)
	}

	saved4 := mock.settings["user123_project456_github"]
	if saved4.IsReadOnly != true {
		t.Errorf("Expected IsReadOnly to default to true when no token is found, got false")
	}
	if saved4.CanWrite != false {
		t.Errorf("Expected CanWrite to be false, got true")
	}
	if saved4.CanSubmitPullRequest != false {
		t.Errorf("Expected CanSubmitPullRequest to be false, got true")
	}

	// Case E: Token has neither push nor PR permission. IsReadOnly should be true, permissions false.
	pushPermissionResponse = false
	prPermissionResponse = false
	body5, _ := json.Marshal(payload)

	req5 := httptest.NewRequest(http.MethodPut, "/api/v1/users/user123/projects/project456/apps/github/credentials", bytes.NewBuffer(body5))
	req5.SetPathValue("userId", "user123")
	req5.SetPathValue("projectId", "project456")
	req5.SetPathValue("appId", "github")
	req5 = withTestCtx(req5, claims)

	w5 := httptest.NewRecorder()
	saveProjectAppSettingHandler(w5, req5)

	if w5.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w5.Code)
	}

	saved5 := mock.settings["user123_project456_github"]
	if saved5.IsReadOnly != true {
		t.Errorf("Expected IsReadOnly to be true when token has neither push nor PR permission, got false")
	}
	if saved5.CanWrite != false {
		t.Errorf("Expected CanWrite to be false, got true")
	}
	if saved5.CanSubmitPullRequest != false {
		t.Errorf("Expected CanSubmitPullRequest to be false, got true")
	}
}

func TestSaveBulkProjectAppSettingsHandler_IsReadOnly(t *testing.T) {
	var pushPermissionResponse bool
	var prPermissionResponse bool
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. GET /repos/{owner}/{repo}
		if r.Method == "GET" && strings.HasSuffix(r.URL.Path, "/repos/owner/test-repo") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(fmt.Sprintf(`{
				"permissions": {
					"pull": true,
					"push": %t
				}
			}`, pushPermissionResponse)))
			return
		}

		// 2. PUT /repos/{owner}/{repo}/contents/...
		if r.Method == "PUT" && strings.Contains(r.URL.Path, "/contents/permission-check-dummy-file-do-not-create") {
			if pushPermissionResponse {
				w.WriteHeader(http.StatusUnprocessableEntity)
			} else {
				w.WriteHeader(http.StatusForbidden)
			}
			return
		}

		// 3. POST /repos/{owner}/{repo}/pulls
		if r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/pulls") {
			if prPermissionResponse {
				w.WriteHeader(http.StatusUnprocessableEntity)
			} else {
				w.WriteHeader(http.StatusForbidden)
			}
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	defer integrations.SetGitHubAPIURLForTesting("https://api.github.com")
	integrations.SetGitHubAPIURLForTesting(ts.URL)

	userProjectStore = &mockUserProjectStore{}
	appCredentialStore = &mockAppCredentialStore{
		creds: map[string]map[string]*db.AppCredential{
			"user123": {
				"github": {IntegrationID: "github"},
			},
		},
	}

	owner := "user123"
	store = &mockProjectStore{
		projects: map[string]*db.Project{
			"project456": {
				ID:      "project456",
				OwnerId: &owner,
				AppIntegrations: []db.ProjectIntegration{
					{
						IntegrationID: "github",
						Enabled:       true,
						Settings: map[string]interface{}{
							"repoUrl": "https://github.com/owner/test-repo",
						},
					},
				},
			},
		},
	}

	// Scenario 1: Token has PR permission but no push. IsReadOnly should be false.
	pushPermissionResponse = false
	prPermissionResponse = true
	payload := []BulkUserProjectDTO{
		{
			AppID:        "github",
			Enabled:      true,
			IsOverridden: true,
			Fields: []db.CredentialAttribute{
				{Name: "pat", Value: "test-github-pat", IsSecret: true},
			},
		},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/user123/projects/project456/apps/credentials", bytes.NewBuffer(body))
	req.SetPathValue("userId", "user123")
	req.SetPathValue("projectId", "project456")
	claims := &core.UserClaims{Subject: "user123"}
	req = withTestCtx(req, claims)

	w := httptest.NewRecorder()
	saveBulkProjectAppSettingsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	mock := userProjectStore.(*mockUserProjectStore)
	saved := mock.settings["user123_project456_github"]
	if saved == nil || saved.IsReadOnly != false {
		t.Errorf("Expected bulk save to mark github integration IsReadOnly as false when user has PR permission, got: %+v", saved)
	}
	if saved.CanWrite != false {
		t.Errorf("Expected CanWrite to be false when user only has PR permission, got true")
	}
	if saved.CanSubmitPullRequest != true {
		t.Errorf("Expected CanSubmitPullRequest to be true, got false")
	}

	// Scenario 2: Token has neither push nor PR permission. IsReadOnly should be true, response WARNING.
	pushPermissionResponse = false
	prPermissionResponse = false
	body2, _ := json.Marshal(payload)

	req2 := httptest.NewRequest(http.MethodPut, "/api/v1/users/user123/projects/project456/apps/credentials", bytes.NewBuffer(body2))
	req2.SetPathValue("userId", "user123")
	req2.SetPathValue("projectId", "project456")
	req2 = withTestCtx(req2, claims)

	w2 := httptest.NewRecorder()
	saveBulkProjectAppSettingsHandler(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w2.Code)
	}

	saved2 := mock.settings["user123_project456_github"]
	if saved2 == nil || saved2.IsReadOnly != true {
		t.Errorf("Expected bulk save to mark github integration IsReadOnly as true when user has neither push nor PR permission, got: %+v", saved2)
	}

	var resp Response
	_ = json.NewDecoder(w2.Body).Decode(&resp)
	if resp.Status != "WARNING" {
		t.Errorf("Expected response status 'WARNING', got '%s'", resp.Status)
	}
	if !strings.Contains(resp.Message, "write access") {
		t.Errorf("Expected warning message to contain 'write access', got '%s'", resp.Message)
	}

	// Scenario 3: Token has push permission but no PR permission. IsReadOnly false, response WARNING.
	pushPermissionResponse = true
	prPermissionResponse = false
	body3, _ := json.Marshal(payload)

	req3 := httptest.NewRequest(http.MethodPut, "/api/v1/users/user123/projects/project456/apps/credentials", bytes.NewBuffer(body3))
	req3.SetPathValue("userId", "user123")
	req3.SetPathValue("projectId", "project456")
	req3 = withTestCtx(req3, claims)

	w3 := httptest.NewRecorder()
	saveBulkProjectAppSettingsHandler(w3, req3)

	if w3.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w3.Code)
	}

	saved3 := mock.settings["user123_project456_github"]
	if saved3 == nil || saved3.IsReadOnly != false {
		t.Errorf("Expected bulk save to mark github integration IsReadOnly as false when user has push permission, got: %+v", saved3)
	}
	if saved3.CanWrite != true {
		t.Errorf("Expected CanWrite to be true, got false")
	}
	if saved3.CanSubmitPullRequest != false {
		t.Errorf("Expected CanSubmitPullRequest to be false, got true")
	}

	var resp3 Response
	_ = json.NewDecoder(w3.Body).Decode(&resp3)
	if resp3.Status != "WARNING" {
		t.Errorf("Expected response status 'WARNING', got '%s'", resp3.Status)
	}
	if !strings.Contains(resp3.Message, "permission to submit pull request") {
		t.Errorf("Expected warning message to contain 'permission to submit pull request', got '%s'", resp3.Message)
	}
}

func TestUserHandlers_ValidateGeminiApiKey_SynchronousValidation(t *testing.T) {
	mockCredStore := &mockAppCredentialStore{}
	appCredentialStore = mockCredStore

	claims := &core.UserClaims{Subject: "user123"}
	payload := AppCredentialDTO{
		IntegrationID: "gemini",
		Fields: []db.CredentialAttribute{
			{Name: "gemini_api_key", Value: "AIzaSy1234567890SecretKeyA1B2", IsSecret: true},
		},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/user123/apps/gemini/credentials", bytes.NewBuffer(body))
	req.SetPathValue("id", "user123")
	req.SetPathValue("appId", "gemini")
	req = withTestCtx(req, claims)

	w := httptest.NewRecorder()
	saveAppCredentialHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var respDTO AppCredentialDTO
	_ = json.NewDecoder(w.Body).Decode(&respDTO)
	if len(respDTO.Fields) == 0 {
		t.Fatal("expected non-empty fields in response")
	}
	if respDTO.Fields[0].Value != "••••••••••••••••••••••••••••••••" {
		t.Errorf("expected masked placeholder '••••••••••••••••••••••••••••••••', got %q", respDTO.Fields[0].Value)
	}
}

func TestUserHandlers_GetGeminiApiKeyStatus_MaskedPreview(t *testing.T) {
	mockCredStore := &mockAppCredentialStore{
		creds: map[string]map[string]*db.AppCredential{
			"user123": {
				"gemini": &db.AppCredential{
					IntegrationID: "gemini",
					Fields: []db.CredentialAttribute{
						{Name: "gemini_api_key", Value: "AIzaSy1234567890SecretKeyA1B2", IsSecret: true},
					},
				},
			},
		},
	}
	appCredentialStore = mockCredStore

	claims := &core.UserClaims{Subject: "user123"}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/user123/apps/gemini/credentials", nil)
	req.SetPathValue("id", "user123")
	req.SetPathValue("appId", "gemini")
	req = withTestCtx(req, claims)

	w := httptest.NewRecorder()
	getAppCredentialHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var respDTO AppCredentialDTO
	_ = json.NewDecoder(w.Body).Decode(&respDTO)
	if respDTO.Fields[0].Value != "••••••••••••••••••••••••••••••••" {
		t.Errorf("expected masked placeholder '••••••••••••••••••••••••••••••••', got %q", respDTO.Fields[0].Value)
	}
}

func TestUserHandlers_DeleteGeminiApiKey_ImmediateRevocation(t *testing.T) {
	mockCredStore := &mockAppCredentialStore{
		creds: map[string]map[string]*db.AppCredential{
			"user123": {
				"gemini": &db.AppCredential{
					IntegrationID: "gemini",
					Fields: []db.CredentialAttribute{
						{Name: "gemini_api_key", Value: "secret"},
					},
				},
			},
		},
	}
	appCredentialStore = mockCredStore

	claims := &core.UserClaims{Subject: "user123"}
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/user123/apps/gemini/credentials", nil)
	req.SetPathValue("id", "user123")
	req.SetPathValue("appId", "gemini")
	req = withTestCtx(req, claims)

	w := httptest.NewRecorder()
	deleteAppCredentialHandler(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", w.Code)
	}

	if mockCredStore.creds["user123"]["gemini"] != nil {
		t.Error("expected gemini credential document to be hard-deleted from store")
	}
}
