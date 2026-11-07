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
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"google.com/apf/api/core"
	"google.com/apf/api/db"
)

var domainRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-]{0,61}[a-zA-Z0-9](?:\.[a-zA-Z0-9-]{1,61}[a-zA-Z0-9])*\.[a-zA-Z]{2,}$`)

func validateAutojoinRuleFields(rule *db.AutojoinRule) error {
	if rule.Domain == "" {
		return fmt.Errorf("email domain cannot be empty")
	}
	if !domainRegex.MatchString(rule.Domain) {
		return fmt.Errorf("invalid email domain format: %s", rule.Domain)
	}
	rule.Domain = strings.ToLower(rule.Domain)

	if rule.OrganizationID == "" {
		return fmt.Errorf("target organization ID cannot be empty")
	}

	if len(rule.DefaultRoles) == 0 {
		return fmt.Errorf("at least one default role must be specified")
	}
	return nil
}

// AdminHandler handles administrative operations for organizations
type AdminHandler struct {
	UserStore         db.UserStore
	OrganizationStore db.OrganizationStore
	AutojoinStore     db.AutojoinStore
	NFSStore          db.NFSStore
	JobRunner         CloudRunJobRunner
	Config            *core.Configuration
}

// NewAdminHandler creates a new AdminHandler with dependencies
func NewAdminHandler(userStore db.UserStore, orgStore db.OrganizationStore, config *core.Configuration) *AdminHandler {
	return &AdminHandler{
		UserStore:         userStore,
		OrganizationStore: orgStore,
		Config:            config,
	}
}

// WithAutojoinStore registers the autojoin database store on the handler
func (h *AdminHandler) WithAutojoinStore(store db.AutojoinStore) *AdminHandler {
	h.AutojoinStore = store
	return h
}

// WithJobRunner registers the Cloud Run job runner on the handler
func (h *AdminHandler) WithJobRunner(runner CloudRunJobRunner) *AdminHandler {
	h.JobRunner = runner
	return h
}

// isValidRole checks if a role is predefined.
// It returns a boolean indicating validity, the canonical role ID (slug) if valid, and an error if any.
func (h *AdminHandler) isValidRole(_ context.Context, _ string, role string) (bool, string, error) {
	lowerRole := strings.ToLower(role)

	// Map backwards-compatible short names to new canonical organization role IDs
	switch lowerRole {
	case "owner":
		return true, string(db.RoleOrgOwner), nil
	case "admin":
		return true, string(db.RoleOrgAdmin), nil
	case "editor":
		return true, string(db.RoleOrgEditor), nil
	case "viewer":
		return true, string(db.RoleOrgViewer), nil
	}

	// Check if it matches any of the new canonical predefined roles
	predefinedRoles := map[db.UserRole]bool{
		db.RoleOrgOwner:       true,
		db.RoleOrgAdmin:       true,
		db.RoleOrgEditor:      true,
		db.RoleOrgViewer:      true,
		db.RoleProjectAdmin:   true,
		db.RoleProjectEditor:  true,
		db.RoleProjectViewer:  true,
		db.RoleProjectCreator: true,
		db.RoleUserAdmin:      true,
		db.RoleUserEditor:     true,
		db.RoleUserViewer:     true,
	}
	if predefinedRoles[db.UserRole(lowerRole)] {
		return true, lowerRole, nil
	}

	return false, "", nil
}

// UpdateOrganizationConfig handles PUT /api/v1/admin/organizations/{orgId}/config
func (h *AdminHandler) UpdateOrganizationConfig(w http.ResponseWriter, r *http.Request) {
	orgID := r.PathValue("orgId")
	if orgID == "" {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Missing organization ID", Status: "ERROR"})
		return
	}

	var config db.OrganizationConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Invalid request payload", Status: "ERROR"})
		return
	}

	// Basic validation
	if config.Branding != nil && config.Branding.Theme != nil {
		theme := *config.Branding.Theme
		if theme != "" && theme != "dark" && theme != "light" {
			sendJSON(w, http.StatusBadRequest, Response{Message: "Invalid theme: must be 'dark' or 'light'", Status: "ERROR"})
			return
		}
	}

	if err := h.OrganizationStore.UpdateOrganizationConfig(r.Context(), orgID, &config); err != nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to update config: %v", err), Status: "ERROR"})
		return
	}

	// Fetch updated org to return
	org, err := h.OrganizationStore.GetOrganization(r.Context(), orgID)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to retrieve updated organization: %v", err), Status: "ERROR"})
		return
	}

	sendJSON(w, http.StatusOK, org)
}

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// AddUserRequest holds direct user creation payload
type AddUserRequest struct {
	Email       string   `json:"email"`
	Password    string   `json:"password"`
	DisplayName string   `json:"displayName,omitempty"`
	Role        string   `json:"role,omitempty"`
	Roles       []string `json:"roles,omitempty"`
}

// AddUser handles POST /api/v1/admin/organizations/{orgId}/users (direct user creation)
func (h *AdminHandler) AddUser(w http.ResponseWriter, r *http.Request) {
	orgID := r.PathValue("orgId")
	if orgID == "" {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Missing organization ID", Status: "ERROR"})
		return
	}

	var req AddUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Invalid request payload", Status: "ERROR"})
		return
	}

	// Normalize email to lowercase
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	if req.Email == "" {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Email is required", Status: "ERROR"})
		return
	}
	if !emailRegex.MatchString(req.Email) {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Invalid email format", Status: "ERROR"})
		return
	}

	if len(req.Password) < 6 {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Password must be at least 6 characters", Status: "ERROR"})
		return
	}

	// Populate Roles if empty but Role is set (for backwards compatibility)
	if len(req.Roles) == 0 && req.Role != "" {
		req.Roles = []string{req.Role}
	}

	if len(req.Roles) == 0 {
		sendJSON(w, http.StatusBadRequest, Response{Message: "At least one role is required", Status: "ERROR"})
		return
	}

	// Restrict role assignment to prevent privilege escalation
	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || claims == nil {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized", Status: "ERROR"})
		return
	}

	var dbRoles []db.UserRole
	var allPermissions []string
	seenPerms := make(map[string]bool)

	for i, roleStr := range req.Roles {
		valid, canonicalRole, err := h.isValidRole(r.Context(), orgID, roleStr)
		if err != nil {
			sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to validate role: %v", err), Status: "ERROR"})
			return
		}
		if !valid {
			sendJSON(w, http.StatusBadRequest, Response{Message: fmt.Sprintf("Invalid role specified: %s", roleStr), Status: "ERROR"})
			return
		}
		req.Roles[i] = canonicalRole

		if ok, errMsg := h.validateRoleAssignment(r.Context(), orgID, claims, canonicalRole); !ok {
			sendJSON(w, http.StatusForbidden, Response{Message: fmt.Sprintf("Role %s: %s", roleStr, errMsg), Status: "ERROR"})
			return
		}
		dbRoles = append(dbRoles, db.UserRole(canonicalRole))

		// Aggregate permissions for Custom Claims
		perms, err := core.GetPermissionsForRole(r.Context(), core.UserRole(canonicalRole))
		if err == nil {
			for _, p := range perms {
				if !seenPerms[p] {
					seenPerms[p] = true
					allPermissions = append(allPermissions, p)
				}
			}
		}
	}

	// Check if user already exists in organization
	dbUsers, err := h.UserStore.ListUsers(r.Context(), orgID)
	if err == nil {
		for _, u := range dbUsers {
			if strings.ToLower(u.Email) == req.Email {
				sendJSON(w, http.StatusBadRequest, Response{Message: "User already exists in this organization", Status: "ERROR"})
				return
			}
		}
	}

	// 1. Create User programmatically in Firebase Auth
	uid, err := globalAuthenticator.CreateUser(r.Context(), orgID, req.Email, req.Password, req.DisplayName)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to create user in Auth provider: %v", err), Status: "ERROR"})
		return
	}

	// 2. Set Custom User Claims in Firebase Auth (for security & token verification)
	firebaseClaims := map[string]interface{}{
		"roles":       req.Roles,
		"role":        req.Roles[0], // primary role
		"permissions": allPermissions,
	}
	err = globalAuthenticator.SetUserClaims(r.Context(), orgID, uid, firebaseClaims)
	if err != nil {
		slog.Error("Failed to set custom claims for newly created user", "uid", uid, "error", err)
	}

	// 3. Create user document in Firestore (with empty UserID so it starts in "invited" status)
	primaryRole := dbRoles[0]
	newUser := &db.User{
		UserID:      "", // Empty UserID starts user in "invited" status until first login
		Email:       req.Email,
		Role:        primaryRole,
		Roles:       dbRoles,
		DisplayName: req.DisplayName,
	}

	err = h.UserStore.CreateUser(r.Context(), orgID, newUser)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to create user profile in database: %v", err), Status: "ERROR"})
		return
	}

	response := map[string]interface{}{
		"id":          req.Email,
		"email":       req.Email,
		"role":        string(primaryRole),
		"roles":       req.Roles,
		"status":      "invited",
		"displayName": req.DisplayName,
	}

	sendJSON(w, http.StatusCreated, response)
}

// UserListItem represents a user in the list response
type UserListItem struct {
	ID          string     `json:"id"`
	UserID      string     `json:"userId,omitempty"`
	Email       string     `json:"email"`
	Role        string     `json:"role"`
	Roles       []string   `json:"roles"`
	Status      string     `json:"status"`
	DisplayName string     `json:"displayName,omitempty"`
	LastLogin   *time.Time `json:"lastLogin,omitempty"`
}

// ListOrganizationUsersResponse is the response format for listing users
type ListOrganizationUsersResponse struct {
	Users []UserListItem `json:"users"`
}

// ListOrganizationUsers handles GET /api/v1/admin/organizations/{orgId}/users
func (h *AdminHandler) ListOrganizationUsers(w http.ResponseWriter, r *http.Request) {
	orgID := r.PathValue("orgId")
	if orgID == "" {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Missing organization ID", Status: "ERROR"})
		return
	}

	dbUsers, err := h.UserStore.ListUsers(r.Context(), orgID)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to list users: %v", err), Status: "ERROR"})
		return
	}

	query := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	if query == "" {
		query = strings.ToLower(strings.TrimSpace(r.URL.Query().Get("search")))
	}

	var users []UserListItem = []UserListItem{}
	for _, u := range dbUsers {
		status := "pending" // Default to pending onboarding
		id := u.UserID
		if u.UserID == "" {
			status = "invited"
			id = u.Email
		} else if u.TourStatus != nil && u.TourStatus.Completed {
			status = "active"
		}
		email := u.Email

		var roles []string
		for _, r := range u.Roles {
			roles = append(roles, string(r))
		}
		if len(roles) == 0 && u.Role != "" {
			roles = []string{string(u.Role)}
		}

		if query != "" {
			match := strings.Contains(strings.ToLower(email), query) ||
				strings.Contains(strings.ToLower(u.DisplayName), query) ||
				strings.Contains(strings.ToLower(id), query) ||
				strings.Contains(strings.ToLower(u.UserID), query) ||
				strings.Contains(strings.ToLower(status), query)
			if !match {
				for _, r := range roles {
					if strings.Contains(strings.ToLower(r), query) {
						match = true
						break
					}
				}
			}
			if !match {
				continue
			}
		}

		users = append(users, UserListItem{
			ID:          id,
			UserID:      u.UserID,
			Email:       email,
			Role:        string(u.Role),
			Roles:       roles,
			Status:      status,
			DisplayName: u.DisplayName,
			LastLogin:   u.LastLogin,
		})
	}

	sendJSON(w, http.StatusOK, ListOrganizationUsersResponse{Users: users})
}

// RemoveUser handles DELETE /api/v1/admin/organizations/{orgId}/users/{userId}
func (h *AdminHandler) RemoveUser(w http.ResponseWriter, r *http.Request) {
	orgID := r.PathValue("orgId")
	userId := r.PathValue("userId")
	if orgID == "" || userId == "" {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Missing orgId or userId", Status: "ERROR"})
		return
	}

	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || claims == nil {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized", Status: "ERROR"})
		return
	}

	// 1. Prevent self-deletion
	if claims.Subject == userId {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Cannot remove yourself from the organization", Status: "ERROR"})
		return
	}

	// 2. Delete user (handles last-admin check and Firebase integration atomically)
	err := h.UserStore.DeleteUser(r.Context(), orgID, userId)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			sendJSON(w, http.StatusNotFound, Response{Message: "User not found", Status: "ERROR"})
			return
		}
		if errors.Is(err, db.ErrLastAdmin) {
			sendJSON(w, http.StatusBadRequest, Response{Message: err.Error(), Status: "ERROR"})
			return
		}
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to remove user: %v", err), Status: "ERROR"})
		return
	}

	sendJSON(w, http.StatusOK, Response{Message: "User removed successfully", Status: "SUCCESS"})
}

// UpdateUserRoleRequest holds role update payload
type UpdateUserRoleRequest struct {
	Role string `json:"role"`
}

// UpdateUserRole handles PUT /api/v1/admin/organizations/{orgId}/users/{userId}/role
func (h *AdminHandler) UpdateUserRole(w http.ResponseWriter, r *http.Request) {
	orgID := r.PathValue("orgId")
	userID := r.PathValue("userId")
	if orgID == "" || userID == "" {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Missing orgId or userId", Status: "ERROR"})
		return
	}

	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || claims == nil {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized", Status: "ERROR"})
		return
	}

	// Prevent self-demotion
	if claims.Subject == userID {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Cannot update your own role", Status: "ERROR"})
		return
	}

	var req UpdateUserRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Invalid request payload", Status: "ERROR"})
		return
	}

	valid, canonicalRole, err := h.isValidRole(r.Context(), orgID, req.Role)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to validate role: %v", err), Status: "ERROR"})
		return
	}
	if !valid {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Invalid role specified", Status: "ERROR"})
		return
	}
	req.Role = canonicalRole

	// Restrict role assignment to prevent privilege escalation
	if valid, errMsg := h.validateRoleAssignment(r.Context(), orgID, claims, req.Role); !valid {
		sendJSON(w, http.StatusForbidden, Response{Message: errMsg, Status: "ERROR"})
		return
	}

	err = h.UserStore.UpdateUserRole(r.Context(), orgID, userID, db.UserRole(req.Role))
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			sendJSON(w, http.StatusNotFound, Response{Message: "User not found", Status: "ERROR"})
			return
		}
		if errors.Is(err, db.ErrLastAdmin) {
			sendJSON(w, http.StatusBadRequest, Response{Message: err.Error(), Status: "ERROR"})
			return
		}
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to update user role: %v", err), Status: "ERROR"})
		return
	}

	sendJSON(w, http.StatusOK, Response{Message: "User role updated successfully", Status: "SUCCESS"})
}

// isSuperset checks if superset slice contains all elements of subset slice.
func isSuperset(superset, subset []string) bool {
	superMap := make(map[string]bool)
	for _, p := range superset {
		superMap[p] = true
	}
	for _, p := range subset {
		if !superMap[p] {
			return false
		}
	}
	return true
}

// validateRoleAssignment prevents privilege escalation by ensuring that the caller
// has all permissions of the role they are attempting to assign.
func (h *AdminHandler) validateRoleAssignment(ctx context.Context, orgID string, claims *core.UserClaims, targetRole string) (bool, string) {
	if claims.IsSuperAdmin {
		return true, ""
	}

	// 1. Fetch caller's current role
	caller, err := h.UserStore.GetUser(ctx, orgID, claims.Subject)
	if err != nil {
		return false, "Caller user not found in organization"
	}

	// 2. Fetch caller's permissions
	callerPerms, err := core.GetPermissionsForRole(ctx, caller.Role)
	if err != nil {
		return false, fmt.Sprintf("Failed to resolve caller permissions: %v", err)
	}

	// 3. Fetch target role's permissions
	targetPerms, err := core.GetPermissionsForRole(ctx, core.UserRole(targetRole))
	if err != nil {
		return false, fmt.Sprintf("Failed to resolve target role permissions: %v", err)
	}

	// 4. Ensure caller possesses all permissions of the target role
	if !isSuperset(callerPerms, targetPerms) {
		return false, "Forbidden: You cannot assign a role with higher permissions than your own"
	}

	return true, ""
}

// ListRolesResponse represents the response containing the list of roles
type ListRolesResponse struct {
	Roles []RoleResponse `json:"roles"`
}

// RoleResponse represents the JSON structure for a role in API responses
type RoleResponse struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
	IsCustom    bool     `json:"isCustom"`
}

// ListRoles handles GET /api/v1/admin/organizations/{orgId}/roles
func (h *AdminHandler) ListRoles(w http.ResponseWriter, r *http.Request) {
	var roles []RoleResponse

	predefinedOrder := []db.UserRole{
		db.RoleOrgOwner, db.RoleOrgAdmin, db.RoleOrgEditor, db.RoleOrgViewer,
		db.RoleProjectAdmin, db.RoleProjectEditor, db.RoleProjectViewer, db.RoleProjectCreator,
		db.RoleUserAdmin, db.RoleUserEditor, db.RoleUserViewer,
	}
	for _, role := range predefinedOrder {
		if perms, ok := core.PredefinedRolePermissions[role]; ok {
			roles = append(roles, RoleResponse{
				ID:          string(role),
				Name:        capitalize(string(role)),
				Description: core.PredefinedRoleDescriptions[role],
				Permissions: perms,
				IsCustom:    false,
			})
		}
	}

	sendJSON(w, http.StatusOK, ListRolesResponse{Roles: roles})
}

// ListPlatformRoles handles GET /api/v1/admin/roles
func (h *AdminHandler) ListPlatformRoles(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || !claims.IsSuperAdmin {
		sendJSON(w, http.StatusForbidden, Response{Message: "Forbidden: Super Administrator access required", Status: "ERROR"})
		return
	}

	var roles []RoleResponse

	predefinedOrder := []db.UserRole{
		db.RoleOrgOwner, db.RoleOrgAdmin, db.RoleOrgEditor, db.RoleOrgViewer,
		db.RoleProjectAdmin, db.RoleProjectEditor, db.RoleProjectViewer, db.RoleProjectCreator,
		db.RoleUserAdmin, db.RoleUserEditor, db.RoleUserViewer,
	}
	for _, role := range predefinedOrder {
		if perms, ok := core.PredefinedRolePermissions[role]; ok {
			roles = append(roles, RoleResponse{
				ID:          string(role),
				Name:        capitalize(string(role)),
				Description: core.PredefinedRoleDescriptions[role],
				Permissions: perms,
				IsCustom:    false,
			})
		}
	}

	sendJSON(w, http.StatusOK, ListRolesResponse{Roles: roles})
}

func capitalize(s string) string {
	if len(s) == 0 {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// EnableAppIntegrationHandler handles POST /api/v1/admin/organizations/{orgId}/config/app-integrations/{id}/enable
func EnableAppIntegrationHandler(w http.ResponseWriter, r *http.Request) {
	orgID := r.PathValue("orgId")
	if orgID == "" {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Missing organization ID", Status: "ERROR"})
		return
	}
	integrationID := r.PathValue("id")
	if integrationID == "" {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Missing integration ID", Status: "ERROR"})
		return
	}

	// 1. Load from global catalog to verify it exists and is available in the system
	globalIntegrations, err := configStore.ListGlobalAppIntegrations(r.Context())
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to fetch global catalog: %v", err), Status: "ERROR"})
		return
	}

	var target *db.AppIntegration
	for i := range globalIntegrations {
		if globalIntegrations[i].ID == integrationID {
			target = &globalIntegrations[i]
			break
		}
	}

	if target == nil {
		sendJSON(w, http.StatusNotFound, Response{Message: fmt.Sprintf("Integration %s not found in global catalog", integrationID), Status: "ERROR"})
		return
	}

	// 2. Enable it by saving to the organization-scoped collection
	target.IsEnabled = true
	err = configStore.SaveOrgAppIntegration(r.Context(), orgID, target)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to enable integration for organization: %v", err), Status: "ERROR"})
		return
	}

	sendJSON(w, http.StatusOK, Response{Message: fmt.Sprintf("Successfully enabled integration %s", integrationID), Status: "SUCCESS"})
}

// DisableAppIntegrationHandler handles POST /api/v1/admin/organizations/{orgId}/config/app-integrations/{id}/disable
func DisableAppIntegrationHandler(w http.ResponseWriter, r *http.Request) {
	orgID := r.PathValue("orgId")
	if orgID == "" {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Missing organization ID", Status: "ERROR"})
		return
	}
	integrationID := r.PathValue("id")
	if integrationID == "" {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Missing integration ID", Status: "ERROR"})
		return
	}

	globalIntegrations, err := configStore.ListGlobalAppIntegrations(r.Context())
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to fetch global catalog: %v", err), Status: "ERROR"})
		return
	}

	var target *db.AppIntegration
	for i := range globalIntegrations {
		if globalIntegrations[i].ID == integrationID {
			target = &globalIntegrations[i]
			break
		}
	}

	if target == nil {
		sendJSON(w, http.StatusNotFound, Response{Message: fmt.Sprintf("Integration %s not found in global catalog", integrationID), Status: "ERROR"})
		return
	}

	target.IsEnabled = false
	err = configStore.SaveOrgAppIntegration(r.Context(), orgID, target)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to disable integration for organization: %v", err), Status: "ERROR"})
		return
	}

	sendJSON(w, http.StatusOK, Response{Message: fmt.Sprintf("Successfully disabled integration %s", integrationID), Status: "SUCCESS"})
}

// ListAutojoinRules handles GET /api/v1/admin/autojoin
func (h *AdminHandler) ListAutojoinRules(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || !claims.IsSuperAdmin {
		sendJSON(w, http.StatusForbidden, Response{Message: "Forbidden: Super Administrator access required", Status: "ERROR"})
		return
	}

	if h.AutojoinStore == nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: "AutojoinStore is not initialized", Status: "ERROR"})
		return
	}

	rules, err := h.AutojoinStore.ListRules(r.Context())
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to list rules: %v", err), Status: "ERROR"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rules)
}

// CreateAutojoinRule handles POST /api/v1/admin/autojoin
func (h *AdminHandler) CreateAutojoinRule(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || !claims.IsSuperAdmin {
		sendJSON(w, http.StatusForbidden, Response{Message: "Forbidden: Super Administrator access required", Status: "ERROR"})
		return
	}

	if h.AutojoinStore == nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: "AutojoinStore is not initialized", Status: "ERROR"})
		return
	}

	var rule db.AutojoinRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Invalid request payload", Status: "ERROR"})
		return
	}

	// 1. Validate fields and normalize domain
	if err := validateAutojoinRuleFields(&rule); err != nil {
		sendJSON(w, http.StatusBadRequest, Response{Message: err.Error(), Status: "ERROR"})
		return
	}

	// 2. Verify target organization exists
	_, err := h.OrganizationStore.GetOrganization(r.Context(), rule.OrganizationID)
	if err != nil {
		sendJSON(w, http.StatusBadRequest, Response{Message: fmt.Sprintf("Target organization %s does not exist", rule.OrganizationID), Status: "ERROR"})
		return
	}

	// 3. Verify roles are valid
	for _, role := range rule.DefaultRoles {
		isValid, _, err := h.isValidRole(r.Context(), rule.OrganizationID, string(role))
		if err != nil || !isValid {
			sendJSON(w, http.StatusBadRequest, Response{Message: fmt.Sprintf("Invalid role: %s", role), Status: "ERROR"})
			return
		}
	}

	// 4. Verify no other active rule exists for this domain
	if rule.IsActive {
		existing, err := h.AutojoinStore.FindActiveRuleByDomain(r.Context(), rule.Domain)
		if err == nil && existing != nil {
			sendJSON(w, http.StatusConflict, Response{Message: fmt.Sprintf("An active autojoin rule already exists for domain: %s", rule.Domain), Status: "ERROR"})
			return
		}
	}

	id, err := h.AutojoinStore.CreateRule(r.Context(), &rule)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to create rule: %v", err), Status: "ERROR"})
		return
	}

	sendJSON(w, http.StatusCreated, map[string]interface{}{
		"id":      id,
		"message": "Autojoin rule created successfully",
		"status":  "SUCCESS",
	})
}

// UpdateAutojoinRule handles PUT /api/v1/admin/autojoin/{id}
func (h *AdminHandler) UpdateAutojoinRule(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || !claims.IsSuperAdmin {
		sendJSON(w, http.StatusForbidden, Response{Message: "Forbidden: Super Administrator access required", Status: "ERROR"})
		return
	}

	id := r.PathValue("id")
	if id == "" {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Missing rule ID", Status: "ERROR"})
		return
	}

	if h.AutojoinStore == nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: "AutojoinStore is not initialized", Status: "ERROR"})
		return
	}

	var rule db.AutojoinRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Invalid request payload", Status: "ERROR"})
		return
	}

	// 1. Validate fields and normalize domain
	if err := validateAutojoinRuleFields(&rule); err != nil {
		sendJSON(w, http.StatusBadRequest, Response{Message: err.Error(), Status: "ERROR"})
		return
	}

	// 2. Verify target organization exists
	_, err := h.OrganizationStore.GetOrganization(r.Context(), rule.OrganizationID)
	if err != nil {
		sendJSON(w, http.StatusBadRequest, Response{Message: fmt.Sprintf("Target organization %s does not exist", rule.OrganizationID), Status: "ERROR"})
		return
	}

	// 3. Verify roles are valid
	for _, role := range rule.DefaultRoles {
		isValid, _, err := h.isValidRole(r.Context(), rule.OrganizationID, string(role))
		if err != nil || !isValid {
			sendJSON(w, http.StatusBadRequest, Response{Message: fmt.Sprintf("Invalid role: %s", role), Status: "ERROR"})
			return
		}
	}

	// 4. Verify no other active rule exists for this domain
	if rule.IsActive {
		existing, err := h.AutojoinStore.FindActiveRuleByDomain(r.Context(), rule.Domain)
		if err == nil && existing != nil && existing.ID != id {
			sendJSON(w, http.StatusConflict, Response{Message: fmt.Sprintf("An active autojoin rule already exists for domain: %s", rule.Domain), Status: "ERROR"})
			return
		}
	}

	err = h.AutojoinStore.UpdateRule(r.Context(), id, &rule)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			sendJSON(w, http.StatusNotFound, Response{Message: "Autojoin rule not found", Status: "ERROR"})
			return
		}
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to update rule: %v", err), Status: "ERROR"})
		return
	}

	sendJSON(w, http.StatusOK, Response{Message: "Autojoin rule updated successfully", Status: "SUCCESS"})
}

// DeleteAutojoinRule handles DELETE /api/v1/admin/autojoin/{id}
func (h *AdminHandler) DeleteAutojoinRule(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || !claims.IsSuperAdmin {
		sendJSON(w, http.StatusForbidden, Response{Message: "Forbidden: Super Administrator access required", Status: "ERROR"})
		return
	}

	id := r.PathValue("id")
	if id == "" {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Missing rule ID", Status: "ERROR"})
		return
	}

	if h.AutojoinStore == nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: "AutojoinStore is not initialized", Status: "ERROR"})
		return
	}

	err := h.AutojoinStore.DeleteRule(r.Context(), id)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to delete rule: %v", err), Status: "ERROR"})
		return
	}

	sendJSON(w, http.StatusOK, Response{Message: "Autojoin rule deleted successfully", Status: "SUCCESS"})
}

// ListOrganizations handles GET /api/v1/admin/organizations
func (h *AdminHandler) ListOrganizations(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || !claims.IsSuperAdmin {
		sendJSON(w, http.StatusForbidden, Response{Message: "Forbidden: Super Administrator access required", Status: "ERROR"})
		return
	}

	orgs, err := h.OrganizationStore.ListOrganizations(r.Context())
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to list organizations: %v", err), Status: "ERROR"})
		return
	}

	for i := range orgs {
		if h.UserStore != nil {
			count, err := h.UserStore.GetUserCount(r.Context(), orgs[i].ID)
			if err == nil {
				orgs[i].UserCount = count
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orgs)
}

// SystemInfoResponse represents the system configuration info for super admins
type SystemInfoResponse struct {
	Version          string   `json:"version"`
	Region           string   `json:"region"`
	SuperAdminEmails []string `json:"superAdminEmails"`
}

//go:embed version.json
var embeddedVersionFile []byte

type versionFile struct {
	Version string `json:"version"`
}

func getPlatformVersion() string {
	if len(embeddedVersionFile) > 0 {
		var vf versionFile
		if err := json.Unmarshal(embeddedVersionFile, &vf); err == nil && strings.TrimSpace(vf.Version) != "" {
			v := strings.TrimSpace(vf.Version)
			if !strings.HasPrefix(v, "v") {
				v = "v" + v
			}
			return v
		}
	}

	paths := []string{
		"version.json",
		"api/version.json",
		"../version.json",
		"../../version.json",
	}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err == nil {
			var vf versionFile
			if err := json.Unmarshal(data, &vf); err == nil && strings.TrimSpace(vf.Version) != "" {
				v := strings.TrimSpace(vf.Version)
				if !strings.HasPrefix(v, "v") {
					v = "v" + v
				}
				return v
			}
		}
	}
	return "v1.0.0"
}

// GetSystemInfo handles GET /api/v1/admin/system-info
func (h *AdminHandler) GetSystemInfo(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || !claims.IsSuperAdmin {
		sendJSON(w, http.StatusForbidden, Response{Message: "Forbidden: Super Administrator access required", Status: "ERROR"})
		return
	}

	region := "unknown"
	var superAdminEmails []string
	if h.Config != nil {
		region = h.Config.GoogleCloudLocation
		superAdminEmails = h.Config.SuperAdminEmails
	}

	resp := SystemInfoResponse{
		Version:          getPlatformVersion(),
		Region:           region,
		SuperAdminEmails: superAdminEmails,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// UpdateOrganizationRequest holds the fields allowed to be updated directly on the organization
type UpdateOrganizationRequest struct {
	Name string `json:"name"`
}

// UpdateOrganization handles PUT /api/v1/admin/organizations/{orgId}
func (h *AdminHandler) UpdateOrganization(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || !claims.IsSuperAdmin {
		sendJSON(w, http.StatusForbidden, Response{Message: "Forbidden: Super Administrator access required", Status: "ERROR"})
		return
	}

	orgID := r.PathValue("orgId")
	if orgID == "" {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Missing organization ID", Status: "ERROR"})
		return
	}

	var req UpdateOrganizationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Invalid request payload", Status: "ERROR"})
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Organization name cannot be empty", Status: "ERROR"})
		return
	}

	// Verify name uniqueness (excluding current org)
	existing, err := h.OrganizationStore.GetOrganizationByName(r.Context(), req.Name)
	if err == nil && existing != nil && existing.ID != orgID {
		sendJSON(w, http.StatusConflict, Response{Message: "An organization with this name already exists", Status: "ERROR"})
		return
	}

	err = h.OrganizationStore.UpdateOrganizationName(r.Context(), orgID, req.Name)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to update organization name: %v", err), Status: "ERROR"})
		return
	}

	sendJSON(w, http.StatusOK, Response{Message: "Organization updated successfully", Status: "SUCCESS"})
}

// DeleteOrganization handles DELETE /api/v1/admin/organizations/{orgId}
func (h *AdminHandler) DeleteOrganization(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || !claims.IsSuperAdmin {
		sendJSON(w, http.StatusForbidden, Response{Message: "Forbidden: Super Administrator access required", Status: "ERROR"})
		return
	}

	orgID := r.PathValue("orgId")
	if orgID == "" {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Missing organization ID", Status: "ERROR"})
		return
	}

	// 1. Get organization to find tenant ID for Firebase cleanup
	org, err := h.OrganizationStore.GetOrganization(r.Context(), orgID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			sendJSON(w, http.StatusNotFound, Response{Message: "Organization not found", Status: "ERROR"})
			return
		}
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to retrieve organization: %v", err), Status: "ERROR"})
		return
	}

	// 2. Clean up Firebase Tenant if it exists
	if org.TenantID != "" && globalOrganizationProvisioner != nil {
		err = globalOrganizationProvisioner.DeleteOrganization(r.Context(), org.TenantID)
		if err != nil {
			slog.Error("Failed to delete Firebase tenant during organization deletion", "tenantID", org.TenantID, "error", err)
			// We continue to delete from DB to avoid orphan DB records if Firebase is desynced
		}
	}

	// 3. Delete from Firestore
	err = h.OrganizationStore.DeleteOrganization(r.Context(), orgID)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to delete organization from database: %v", err), Status: "ERROR"})
		return
	}

	sendJSON(w, http.StatusOK, Response{Message: "Organization deleted successfully", Status: "SUCCESS"})
}

// GetOrganization handles GET /api/v1/admin/organizations/{orgId}
func (h *AdminHandler) GetOrganization(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || !claims.IsSuperAdmin {
		sendJSON(w, http.StatusForbidden, Response{Message: "Forbidden: Super Administrator access required", Status: "ERROR"})
		return
	}

	orgID := r.PathValue("orgId")
	if orgID == "" {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Missing organization ID", Status: "ERROR"})
		return
	}

	org, err := h.OrganizationStore.GetOrganization(r.Context(), orgID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			sendJSON(w, http.StatusNotFound, Response{Message: "Organization not found", Status: "ERROR"})
			return
		}
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to retrieve organization: %v", err), Status: "ERROR"})
		return
	}

	if h.UserStore != nil {
		count, err := h.UserStore.GetUserCount(r.Context(), orgID)
		if err == nil {
			org.UserCount = count
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(org)
}
