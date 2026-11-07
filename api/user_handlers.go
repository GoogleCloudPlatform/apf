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
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"google.com/apf/api/core"
	"google.com/apf/api/db"
	"google.com/apf/api/integrations"
	"google.com/apf/api/workspace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var userStore db.UserStore
var appCredentialStore db.AppCredentialStore
var userProjectStore db.UserProjectStore

const credentialMaskPlaceholder = "••••••••••••••••••••••••••••••••"

// AppCredentialDTO represents the secure data transfer model for credentials
type AppCredentialDTO struct {
	IntegrationID string                   `json:"integrationId"`
	Fields        []db.CredentialAttribute `json:"fields"`
}

// UserProjectDTO represents user project status payload
type UserProjectDTO struct {
	Enabled      bool                     `json:"enabled"`
	CredentialID string                   `json:"credentialId,omitempty"`
	IsOverridden bool                     `json:"isOverridden"`
	Fields       []db.CredentialAttribute `json:"fields,omitempty"`
	IsReadOnly   bool                     `json:"isReadOnly"`
}

// BulkUserProjectDTO represents an item in the bulk save request
type BulkUserProjectDTO struct {
	AppID        string                   `json:"appId"`
	Enabled      bool                     `json:"enabled"`
	CredentialID string                   `json:"credentialId,omitempty"`
	IsOverridden bool                     `json:"isOverridden"`
	Fields       []db.CredentialAttribute `json:"fields,omitempty"`
	IsReadOnly   bool                     `json:"isReadOnly"`
}

func getUserIDParam(r *http.Request, key string) string {
	val := r.PathValue(key)
	if unescaped, err := url.PathUnescape(val); err == nil && unescaped != "" {
		return unescaped
	}
	return val
}

func checkUserAccess(claims *core.UserClaims, targetUserID string) bool {
	targetUnescaped, errTarget := url.PathUnescape(targetUserID)
	if errTarget != nil {
		targetUnescaped = targetUserID
	}
	subjectUnescaped, errSub := url.PathUnescape(claims.Subject)
	if errSub != nil {
		subjectUnescaped = claims.Subject
	}
	if subjectUnescaped == targetUnescaped || claims.Subject == targetUserID || claims.Subject == targetUnescaped {
		return true
	}
	for _, r := range claims.Roles {
		if r == "admin" {
			return true
		}
	}
	return false
}

// GET /api/v1/users/{id}/preferences
func getUserPreferencesHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || claims == nil {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized context", Status: "ERROR"})
		return
	}

	id := getUserIDParam(r, "id")
	if !checkUserAccess(claims, id) {
		sendJSON(w, http.StatusForbidden, Response{Message: "Forbidden: You do not have access to this resource", Status: "ERROR"})
		return
	}

	organizationID, errOrg := core.GetOrganizationID(r.Context())
	if errOrg != nil {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized context: missing organization ID", Status: "ERROR"})
		return
	}

	user, err := userStore.GetOrCreateUser(r.Context(), organizationID, id)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Database error: %v", err), Status: "ERROR"})
		return
	}

	sendJSON(w, http.StatusOK, user.Preferences)
}

// PUT /api/v1/users/{id}/preferences
func updateUserPreferencesHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || claims == nil {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized context", Status: "ERROR"})
		return
	}

	id := getUserIDParam(r, "id")
	if !checkUserAccess(claims, id) {
		sendJSON(w, http.StatusForbidden, Response{Message: "Forbidden: You do not have access to this resource", Status: "ERROR"})
		return
	}

	var prefs db.UserPreferences
	if err := json.NewDecoder(r.Body).Decode(&prefs); err != nil {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Invalid request payload", Status: "ERROR"})
		return
	}

	if prefs.Theme != "dark" && prefs.Theme != "light" {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Invalid theme value: must be 'dark' or 'light'", Status: "ERROR"})
		return
	}

	organizationID, errOrg := core.GetOrganizationID(r.Context())
	if errOrg != nil {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized context: missing organization ID", Status: "ERROR"})
		return
	}

	if err := userStore.UpdateUserPreferences(r.Context(), organizationID, id, &prefs); err != nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to update preferences: %v", err), Status: "ERROR"})
		return
	}

	sendJSON(w, http.StatusOK, prefs)
}

// GET /api/v1/users/{id}/tour-status
func getUserTourStatusHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || claims == nil {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized context", Status: "ERROR"})
		return
	}

	id := getUserIDParam(r, "id")
	if !checkUserAccess(claims, id) {
		sendJSON(w, http.StatusForbidden, Response{Message: "Forbidden: You do not have access to this resource", Status: "ERROR"})
		return
	}

	organizationID, errOrg := core.GetOrganizationID(r.Context())
	if errOrg != nil {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized context: missing organization ID", Status: "ERROR"})
		return
	}

	user, err := userStore.GetOrCreateUser(r.Context(), organizationID, id)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Database error: %v", err), Status: "ERROR"})
		return
	}

	status := user.TourStatus
	if status == nil {
		status = &db.TourStatus{
			Completed:         false,
			CompletedSections: []string{},
			LastActiveStep: db.LastActiveStep{
				SectionID: "",
				StepIndex: 0,
			},
		}
	}

	sendJSON(w, http.StatusOK, status)
}

// PUT /api/v1/users/{id}/tour-status
func updateUserTourStatusHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || claims == nil {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized context", Status: "ERROR"})
		return
	}

	id := getUserIDParam(r, "id")
	if !checkUserAccess(claims, id) {
		sendJSON(w, http.StatusForbidden, Response{Message: "Forbidden: You do not have access to this resource", Status: "ERROR"})
		return
	}

	var status db.TourStatus
	if err := json.NewDecoder(r.Body).Decode(&status); err != nil {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Invalid request payload", Status: "ERROR"})
		return
	}

	organizationID, errOrg := core.GetOrganizationID(r.Context())
	if errOrg != nil {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized context: missing organization ID", Status: "ERROR"})
		return
	}

	if err := userStore.UpdateTourStatus(r.Context(), organizationID, id, &status); err != nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to update tour status: %v", err), Status: "ERROR"})
		return
	}

	sendJSON(w, http.StatusOK, status)
}

// GET /api/v1/users/{id}/apps/{appId}/credentials
func getAppCredentialHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || claims == nil {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized context", Status: "ERROR"})
		return
	}

	id := r.PathValue("id")
	appId := r.PathValue("appId")
	if !checkUserAccess(claims, id) {
		sendJSON(w, http.StatusForbidden, Response{Message: "Forbidden: You do not have access to this resource", Status: "ERROR"})
		return
	}

	organizationID, errOrg := core.GetOrganizationID(r.Context())
	if errOrg != nil {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized context: missing organization ID", Status: "ERROR"})
		return
	}

	cred, err := appCredentialStore.GetCredential(r.Context(), organizationID, id, appId)
	if err != nil {
		if status.Code(err) == codes.NotFound || strings.Contains(err.Error(), "not found") {
			sendJSON(w, http.StatusNotFound, Response{Message: "Credentials not found", Status: "ERROR"})
			return
		}
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to retrieve credentials: %v", err), Status: "ERROR"})
		return
	}

	dto := AppCredentialDTO{
		IntegrationID: cred.IntegrationID,
		Fields:        make([]db.CredentialAttribute, len(cred.Fields)),
	}

	for i, field := range cred.Fields {
		dto.Fields[i] = field
		if field.IsSecret && field.Value != "" {
			dto.Fields[i].Value = credentialMaskPlaceholder
		}
	}



	sendJSON(w, http.StatusOK, dto)
}

// PUT /api/v1/users/{id}/apps/{appId}/credentials
func saveAppCredentialHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || claims == nil {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized context", Status: "ERROR"})
		return
	}

	id := r.PathValue("id")
	appId := r.PathValue("appId")
	if !checkUserAccess(claims, id) {
		sendJSON(w, http.StatusForbidden, Response{Message: "Forbidden: You do not have access to this resource", Status: "ERROR"})
		return
	}

	var payload AppCredentialDTO
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Invalid request payload", Status: "ERROR"})
		return
	}

	organizationID, errOrg := core.GetOrganizationID(r.Context())
	if errOrg != nil {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized context: missing organization ID", Status: "ERROR"})
		return
	}

	var existing *db.AppCredential
	var fetchErr error
	existing, fetchErr = appCredentialStore.GetCredential(r.Context(), organizationID, id, appId)
	if fetchErr != nil {
		if !strings.Contains(fetchErr.Error(), "not found") && status.Code(fetchErr) != codes.NotFound {
			sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Database lookup failure: %v", fetchErr), Status: "ERROR"})
			return
		}
		existing = nil
	}

	newCred := db.AppCredential{
		IntegrationID: appId,
		Fields:        make([]db.CredentialAttribute, len(payload.Fields)),
	}

	copy(newCred.Fields, payload.Fields)
	hydrateSecretsFromExisting(&newCred, existing)

	if configStore != nil {
		catalog, errCatalog := configStore.ListGlobalAppIntegrations(r.Context())
		if errCatalog == nil {
			var catalogApp *db.AppIntegration
			for i := range catalog {
				if catalog[i].ID == appId {
					catalogApp = &catalog[i]
					break
				}
			}
			if catalogApp != nil {
				for i, field := range newCred.Fields {
					for _, catField := range catalogApp.Fields {
						if catField.Name == field.Name {
							if catField.Required {
								newCred.Fields[i].Required = true
							}
							if newCred.Fields[i].DisplayName == "" {
								newCred.Fields[i].DisplayName = catField.DisplayName
							}
							break
						}
					}
				}
			}
		}
	}

	for _, field := range newCred.Fields {
		if field.Enabled && field.Required && strings.TrimSpace(field.Value) == "" {
			displayName := field.DisplayName
			if displayName == "" {
				displayName = field.Name
			}
			sendJSON(w, http.StatusBadRequest, Response{
				Message: fmt.Sprintf("%s is required", displayName),
				Status:  "ERROR",
			})
			return
		}
	}
	if appId == "github" {
		patVal := db.GetFieldValue(&newCred, "pat")
		emailVal := db.GetFieldValue(&newCred, "email")
		if strings.TrimSpace(patVal) == "" {
			sendJSON(w, http.StatusBadRequest, Response{
				Message: "Personal Access Token is required for GitHub credentials",
				Status:  "ERROR",
			})
			return
		}
		if strings.TrimSpace(emailVal) == "" {
			sendJSON(w, http.StatusBadRequest, Response{
				Message: "Git Author Email is required for GitHub credentials",
				Status:  "ERROR",
			})
			return
		}
	}

	if err := appCredentialStore.SaveCredential(r.Context(), organizationID, id, appId, &newCred); err != nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to save credentials: %v", err), Status: "ERROR"})
		return
	}

	dto := AppCredentialDTO{
		IntegrationID: appId,
		Fields:        make([]db.CredentialAttribute, len(newCred.Fields)),
	}
	for i, field := range newCred.Fields {
		dto.Fields[i] = field
		if field.IsSecret && field.Value != "" {
			dto.Fields[i].Value = credentialMaskPlaceholder
		}
	}



	sendJSON(w, http.StatusOK, dto)
}

// DELETE /api/v1/users/{id}/apps/{appId}/credentials
func deleteAppCredentialHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || claims == nil {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized context", Status: "ERROR"})
		return
	}

	id := r.PathValue("id")
	appId := r.PathValue("appId")
	if !checkUserAccess(claims, id) {
		sendJSON(w, http.StatusForbidden, Response{Message: "Forbidden: You do not have access to this resource", Status: "ERROR"})
		return
	}

	organizationID, errOrg := core.GetOrganizationID(r.Context())
	if errOrg != nil {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized context: missing organization ID", Status: "ERROR"})
		return
	}

	if err := appCredentialStore.DeleteCredential(r.Context(), organizationID, id, appId); err != nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to delete credentials: %v", err), Status: "ERROR"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GET /api/v1/users/{userId}/projects/{projectId}/apps/{appId}/credentials
func getProjectAppSettingHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || claims == nil {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized context", Status: "ERROR"})
		return
	}

	userId := r.PathValue("userId")
	projectId := r.PathValue("projectId")
	appId := r.PathValue("appId")

	if !checkUserAccess(claims, userId) {
		sendJSON(w, http.StatusForbidden, Response{Message: "Forbidden: You do not have access to this resource", Status: "ERROR"})
		return
	}

	organizationID, errOrg := core.GetOrganizationID(r.Context())
	if errOrg != nil {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized context: missing organization ID", Status: "ERROR"})
		return
	}

	// Verify target project access
	project, err := store.GetProject(r.Context(), organizationID, projectId)
	if err != nil {
		sendJSON(w, http.StatusNotFound, Response{Message: fmt.Sprintf("Project not found: %v", err), Status: "ERROR"})
		return
	}

	isMember := false
	if project.Members != nil && (*project.Members)[claims.Subject] != "" {
		isMember = true
	}
	if project.OwnerId != nil && *project.OwnerId != claims.Subject && !isMember {
		sendJSON(w, http.StatusForbidden, Response{Message: "Forbidden: You are not a member or owner of this project", Status: "ERROR"})
		return
	}

	setting, err := userProjectStore.GetProjectAppSetting(r.Context(), organizationID, userId, projectId, appId)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Database error: %v", err), Status: "ERROR"})
		return
	}

	sendJSON(w, http.StatusOK, setting)
}

func hydrateUserProjectSecretsFromExisting(incoming *db.UserProject, existing *db.UserProject) {
	if incoming == nil || existing == nil {
		return
	}
	for i, field := range incoming.Fields {
		if field.IsSecret && field.Value == credentialMaskPlaceholder {
			for _, extField := range existing.Fields {
				if extField.Name == field.Name {
					incoming.Fields[i].Value = extField.Value
					break
				}
			}
		}
	}
}

func checkAndUpdateUserProjectPermissions(ctx context.Context, setting *db.UserProject) {
	if setting.AppID != "github" || !setting.Enabled {
		return
	}

	// Keep track of user's request
	userRequestedReadOnly := setting.IsReadOnly

	// Default to read-only for github if enabled, until proven otherwise.
	setting.IsReadOnly = true

	organizationID, errOrg := core.GetOrganizationID(ctx)
	if errOrg != nil {
		slog.Error("Failed to fetch organization ID for user project write permission check", "projectId", setting.ProjectID, "error", errOrg)
		return
	}

	project, err := store.GetProject(ctx, organizationID, setting.ProjectID)
	if err != nil {
		slog.Error("Failed to fetch project for user project write permission check", "projectId", setting.ProjectID, "error", err)
		return
	}

	repoURL := project.GetRepoUrl()
	slog.Info("checkAndUpdateUserProjectPermissions: Resolved repo URL", "repoUrl", repoURL)
	if repoURL == "" {
		return
	}

	owner, repo, err := workspace.ParseGitHubURL(repoURL)
	if err != nil {
		slog.Error("Failed to parse github url for user project write permission check", "repoUrl", repoURL, "error", err)
		return
	}

	var token string
	if setting.IsOverridden {
		for _, field := range setting.Fields {
			if field.Name == "pat" {
				token = field.Value
				break
			}
		}
	}

	if token == "" {
		resolvedToken, err := integrations.ResolveGitHubToken(ctx, setting.UserID, setting.ProjectID)
		if err == nil {
			token = resolvedToken
		}
	}

	slog.Info("checkAndUpdateUserProjectPermissions: Resolved token", "hasToken", token != "")

	if token == "" {
		slog.Warn("No GitHub token found for user project write permission check", "userId", setting.UserID, "projectId", setting.ProjectID)
		return
	}

	if integrations.IsSimulatedToken(token) {
		slog.Info("checkAndUpdateUserProjectPermissions: Simulated token detected, forcing read-write")
		setting.IsReadOnly = userRequestedReadOnly
		setting.CanWrite = true
		setting.CanSubmitPullRequest = true
		return
	}

	client := &integrations.GitHubPRClient{Token: token}
	_, hasWrite, hasPR, err := client.CheckPermissions(ctx, owner, repo)
	if err != nil {
		slog.Error("GitHub permissions check failed", "owner", owner, "repo", repo, "error", err)
		return
	}

	slog.Info("checkAndUpdateUserProjectPermissions: Permissions check result", "hasWrite", hasWrite, "hasPR", hasPR)

	setting.CanWrite = hasWrite
	setting.CanSubmitPullRequest = hasPR
	setting.IsReadOnly = userRequestedReadOnly || !(hasWrite || hasPR)
	slog.Info("checkAndUpdateUserProjectPermissions: Final settings", "isReadOnly", setting.IsReadOnly, "canWrite", setting.CanWrite, "canSubmitPR", setting.CanSubmitPullRequest)
}

// PUT /api/v1/users/{userId}/projects/{projectId}/apps/{appId}/credentials
func saveProjectAppSettingHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || claims == nil {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized context", Status: "ERROR"})
		return
	}

	userId := r.PathValue("userId")
	projectId := r.PathValue("projectId")
	appId := r.PathValue("appId")

	if !checkUserAccess(claims, userId) {
		sendJSON(w, http.StatusForbidden, Response{Message: "Forbidden: You do not have access to this resource", Status: "ERROR"})
		return
	}

	organizationID, errOrg := core.GetOrganizationID(r.Context())
	if errOrg != nil {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized context: missing organization ID", Status: "ERROR"})
		return
	}

	// Verify target project access
	project, err := store.GetProject(r.Context(), organizationID, projectId)
	if err != nil {
		sendJSON(w, http.StatusNotFound, Response{Message: fmt.Sprintf("Project not found: %v", err), Status: "ERROR"})
		return
	}

	isMember := false
	if project.Members != nil && (*project.Members)[claims.Subject] != "" {
		isMember = true
	}
	if project.OwnerId != nil && *project.OwnerId != claims.Subject && !isMember {
		sendJSON(w, http.StatusForbidden, Response{Message: "Forbidden: You are not a member or owner of this project", Status: "ERROR"})
		return
	}

	var payload UserProjectDTO
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Invalid request payload", Status: "ERROR"})
		return
	}

	var existing *db.UserProject
	if payload.IsOverridden {
		existing, _ = userProjectStore.GetProjectAppSetting(r.Context(), organizationID, userId, projectId, appId)
	}

	// Double check: if user is enabling this, they must have valid credentials globally configured or they receive a warning!
	// (Note: The prompt says warning is shown in UI if they enable but don't have credentials configured.
	// We allow saving here but validate state. We can also cross-check if the global app credential exists.)
	hasGlobalCredential := false
	_, credErr := appCredentialStore.GetCredential(r.Context(), organizationID, userId, appId)
	if credErr == nil {
		hasGlobalCredential = true
	}

	setting := db.UserProject{
		UserID:       userId,
		ProjectID:    projectId,
		AppID:        appId,
		Enabled:      payload.Enabled,
		CredentialID: payload.CredentialID,
		IsOverridden: payload.IsOverridden,
		Fields:       payload.Fields,
		IsReadOnly:   payload.IsReadOnly,
	}

	if setting.IsOverridden && existing != nil {
		hydrateUserProjectSecretsFromExisting(&setting, existing)
	}

	// Default credential ID to the appId itself if enabled and credentialId is empty but global credential exists
	if setting.Enabled && setting.CredentialID == "" && hasGlobalCredential {
		setting.CredentialID = appId
	}

	// Perform isReadOnly write check validation
	checkAndUpdateUserProjectPermissions(r.Context(), &setting)

	if err := userProjectStore.SaveProjectAppSetting(r.Context(), organizationID, &setting); err != nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to save project settings: %v", err), Status: "ERROR"})
		return
	}

	sendJSON(w, http.StatusOK, setting)
}

// PUT /api/v1/users/{userId}/projects/{projectId}/apps/credentials
func saveBulkProjectAppSettingsHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || claims == nil {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized context", Status: "ERROR"})
		return
	}

	userId := r.PathValue("userId")
	projectId := r.PathValue("projectId")

	if !checkUserAccess(claims, userId) {
		sendJSON(w, http.StatusForbidden, Response{Message: "Forbidden: You do not have access to this resource", Status: "ERROR"})
		return
	}

	organizationID, errOrg := core.GetOrganizationID(r.Context())
	if errOrg != nil {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized context: missing organization ID", Status: "ERROR"})
		return
	}

	// Verify target project access
	project, err := store.GetProject(r.Context(), organizationID, projectId)
	if err != nil {
		sendJSON(w, http.StatusNotFound, Response{Message: fmt.Sprintf("Project not found: %v", err), Status: "ERROR"})
		return
	}

	isMember := false
	if project.Members != nil && (*project.Members)[claims.Subject] != "" {
		isMember = true
	}
	if project.OwnerId != nil && *project.OwnerId != claims.Subject && !isMember {
		sendJSON(w, http.StatusForbidden, Response{Message: "Forbidden: You are not a member or owner of this project", Status: "ERROR"})
		return
	}

	var payload []BulkUserProjectDTO
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Invalid request payload", Status: "ERROR"})
		return
	}

	hasWarnings := false
	warningMessage := ""
	for _, item := range payload {
		if item.AppID == "" {
			sendJSON(w, http.StatusBadRequest, Response{Message: "appId is required for all bulk integration items", Status: "ERROR"})
			return
		}

		var existing *db.UserProject
		if item.IsOverridden {
			existing, _ = userProjectStore.GetProjectAppSetting(r.Context(), organizationID, userId, projectId, item.AppID)
		}

		hasGlobalCredential := false
		_, credErr := appCredentialStore.GetCredential(r.Context(), organizationID, userId, item.AppID)
		if credErr == nil {
			hasGlobalCredential = true
		}

		setting := db.UserProject{
			UserID:       userId,
			ProjectID:    projectId,
			AppID:        item.AppID,
			Enabled:      item.Enabled,
			CredentialID: item.CredentialID,
			IsOverridden: item.IsOverridden,
			Fields:       item.Fields,
			IsReadOnly:   item.IsReadOnly,
		}

		if setting.IsOverridden && existing != nil {
			hydrateUserProjectSecretsFromExisting(&setting, existing)
		}

		if setting.Enabled && setting.CredentialID == "" && hasGlobalCredential {
			setting.CredentialID = item.AppID
		}

		// Perform isReadOnly write check validation
		checkAndUpdateUserProjectPermissions(r.Context(), &setting)

		if setting.AppID == "github" && setting.Enabled {
			if setting.IsReadOnly {
				hasWarnings = true
				warningMessage = "You dont have write access to the code repo. You will not be able to sync your changes and collaborate with others."
			} else if !setting.CanSubmitPullRequest {
				hasWarnings = true
				warningMessage = "You dont have permission to submit pull request. If your orgnaization prevents direct pushes, you will need this permission to collbaorate effectively."
			}
		}

		if err := userProjectStore.SaveProjectAppSetting(r.Context(), organizationID, &setting); err != nil {
			sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to save project settings for %s: %v", item.AppID, err), Status: "ERROR"})
			return
		}
	}

	if hasWarnings {
		sendJSON(w, http.StatusOK, Response{
			Title:   "Project Created with Warnings",
			Message: warningMessage,
			Status:  "WARNING",
		})
		return
	}

	sendJSON(w, http.StatusOK, Response{Message: "Successfully updated project app integrations", Status: "SUCCESS"})
}

// GET /api/v1/config/app-integrations
func getAppIntegrationsHandler(w http.ResponseWriter, r *http.Request) {
	_, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized context", Status: "ERROR"})
		return
	}

	// 1. Load global catalog
	globalList, err := configStore.ListGlobalAppIntegrations(r.Context())
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to fetch global catalog: %v", err), Status: "ERROR"})
		return
	}

	// 2. Try to load organization-scoped integrations if organization ID is in context
	orgID, err := core.GetOrganizationID(r.Context())
	if err == nil && orgID != "" {
		orgList, err := configStore.ListOrgAppIntegrations(r.Context(), orgID)
		if err == nil {
			// Create a map of org-scoped integrations for fast lookup
			orgMap := make(map[string]db.AppIntegration)
			for _, oi := range orgList {
				orgMap[oi.ID] = oi
			}

			// Merge: Update IsEnabled based on org-scoped state
			for i := range globalList {
				if oi, exists := orgMap[globalList[i].ID]; exists {
					globalList[i].IsEnabled = oi.IsEnabled
				} else {
					// If it doesn't exist in org-scoped, it is disabled for this org by default!
					globalList[i].IsEnabled = false
				}
			}
		}
	} else {
		// If no organization ID in context, default all to disabled
		for i := range globalList {
			globalList[i].IsEnabled = false
		}
	}

	if len(globalList) == 0 {
		globalList = []db.AppIntegration{}
	}

	sendJSON(w, http.StatusOK, globalList)
}

// Hook to be invoked from first login (within meHandler) to guarantee profile document bootstrap.
func bootstrapUserProfile(ctx context.Context, organizationID string, userId string) error {
	if userId == "" {
		return errors.New("cannot bootstrap empty userId")
	}
	if organizationID == "" {
		return errors.New("cannot bootstrap empty organizationID")
	}
	_, err := userStore.GetOrCreateUser(ctx, organizationID, userId)
	return err
}

// POST /api/v1/users/{id}/apps/{appId}/credentials/validate
func validateAppCredentialHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || claims == nil {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized context", Status: "ERROR"})
		return
	}

	id := r.PathValue("id")
	appId := r.PathValue("appId")
	if !checkUserAccess(claims, id) {
		sendJSON(w, http.StatusForbidden, Response{Message: "Forbidden: You do not have access to this resource", Status: "ERROR"})
		return
	}

	organizationID, errOrg := core.GetOrganizationID(r.Context())
	if errOrg != nil {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized context: missing organization ID", Status: "ERROR"})
		return
	}

	// 1. Fetch existing credentials from db if present
	var existing *db.AppCredential
	existing, fetchErr := appCredentialStore.GetCredential(r.Context(), organizationID, id, appId)
	if fetchErr != nil && status.Code(fetchErr) != codes.NotFound && !strings.Contains(fetchErr.Error(), "not found") {
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Database lookup failure: %v", fetchErr), Status: "ERROR"})
		return
	}

	var cred db.AppCredential
	var payload AppCredentialDTO
	hasPayload := true

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		if err == io.EOF {
			hasPayload = false
		} else {
			sendJSON(w, http.StatusBadRequest, Response{Message: "Invalid request payload: " + err.Error(), Status: "ERROR"})
			return
		}
	}

	if hasPayload {
		cred = db.AppCredential{
			IntegrationID: appId,
			Fields:        make([]db.CredentialAttribute, len(payload.Fields)),
		}

		copy(cred.Fields, payload.Fields)
		hydrateSecretsFromExisting(&cred, existing)
	} else {
		// No payload passed, validate already saved credentials in DB
		if existing == nil {
			sendJSON(w, http.StatusNotFound, Response{Message: "No saved credentials found to validate", Status: "ERROR"})
			return
		}
		cred = *existing
	}

	// 3. Resolve dynamic integration implementation
	impl, err := integrations.GetIntegration(appId)
	if err != nil {
		sendJSON(w, http.StatusNotImplemented, Response{Message: fmt.Sprintf("Validation not supported: %v", err), Status: "ERROR"})
		return
	}

	// 4. Run dynamic validation routine
	valid, err := impl.Validate(r.Context(), &cred)
	if err != nil {
		sendJSON(w, http.StatusOK, map[string]interface{}{
			"valid": false,
			"error": err.Error(),
		})
		return
	}

	respData := map[string]interface{}{
		"valid": valid,
	}
	if valid && appId == "github" {
		patVal := db.GetFieldValue(&cred, "pat")
		if profile, err := integrations.FetchGitHubUserProfile(r.Context(), patVal); err == nil && profile != nil && profile.Email != "" {
			respData["email"] = profile.Email
		}
	}

	sendJSON(w, http.StatusOK, respData)
}

// hydrateSecretsFromExisting restores actual secret values from existing credentials
// if the incoming fields contain the credentialMaskPlaceholder.
func hydrateSecretsFromExisting(incoming *db.AppCredential, existing *db.AppCredential) {
	if incoming == nil {
		return
	}
	for i, field := range incoming.Fields {
		if field.IsSecret && field.Value == credentialMaskPlaceholder {
			if existing != nil {
				var foundExisting bool
				for _, extField := range existing.Fields {
					if extField.Name == field.Name {
						incoming.Fields[i].Value = extField.Value
						foundExisting = true
						break
					}
				}
				if !foundExisting {
					incoming.Fields[i].Value = ""
				}
			} else {
				incoming.Fields[i].Value = ""
			}
		}
	}
}
