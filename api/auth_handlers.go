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
	"fmt"
	"log/slog"
	"net/http"

	"google.com/apf/api/core"
	"google.com/apf/api/db"
)

type AuthHandler struct {
	provisioner OrganizationProvisioner
	authenticator Authenticator
	validator     TokenValidator
	userStore     db.UserStore
	orgStore      db.OrganizationStore
	config        *core.Configuration
}

func NewAuthHandler(
	provisioner OrganizationProvisioner,
	authenticator Authenticator,
	validator TokenValidator,
	userStore db.UserStore,
	orgStore db.OrganizationStore,
	config *core.Configuration,
) *AuthHandler {
	return &AuthHandler{
		provisioner: provisioner,
		authenticator: authenticator,
		validator:     validator,
		userStore:     userStore,
		orgStore:      orgStore,
		config:        config,
	}
}

// Me returns the current user's claims.
func (a *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Bootstrap/create user profile and preferences document on first login
	// Only if organization ID can be resolved (e.g. org-scoped token).
	// For primary token on first login (before org selection), we skip bootstrapping here.
	if orgID, err := core.GetOrganizationID(r.Context()); err == nil && orgID != "" {
		if err := bootstrapUserProfile(r.Context(), orgID, claims.Subject); err != nil {
			http.Error(w, "Failed to bootstrap user profile: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(claims)
}

// Login issues simulated JWT tokens for local velocity.
func (a *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// For local simulation, allow any non-empty password, or specific admin/admin
	if req.Username == "" || req.Password == "" {
		http.Error(w, "Missing credentials", http.StatusUnauthorized)
		return
	}

	token, err := a.authenticator.GenerateMockToken(req.Username)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(LoginResponse{Token: token})
}

// Logout handles simulated logout.
func (a *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message": "Logged out successfully"}`))
}

// ListUserOrganizations lists all organizations the authenticated user belongs to.
func (a *AuthHandler) ListUserOrganizations(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok {
		http.Error(w, "Unauthorized context", http.StatusUnauthorized)
		return
	}

	// Automatically reconcile any pending memberships for this email on the fly
	if claims.Username != "" {
		err := a.userStore.ReconcileUserMemberships(r.Context(), claims.Username, claims.Subject)
		if err != nil {
			slog.Error("Failed to reconcile user memberships on list", "email", claims.Username, "subject", claims.Subject, "error", err)
		}
		
		// Automatically check and reconcile any domain-based autojoin rules
		err = a.userStore.ReconcileAutojoinRules(r.Context(), claims.Username, claims.Subject)
		if err != nil {
			slog.Error("Failed to reconcile autojoin rules on list", "email", claims.Username, "subject", claims.Subject, "error", err)
		}
	}

	orgIDs, err := a.userStore.ListUserOrganizations(r.Context(), claims.Subject)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to query organizations: %v", err), http.StatusInternalServerError)
		return
	}

	if len(orgIDs) == 0 {
		orgIDs = claims.Organizations
	}
	if len(orgIDs) == 0 && a.orgStore != nil {
		allOrgs, err := a.orgStore.ListOrganizations(r.Context())
		if err == nil {
			for _, o := range allOrgs {
				orgIDs = append(orgIDs, o.ID)
			}
		}
	}

	var orgInfos []OrganizationInfo
	for _, id := range orgIDs {
		name := id
		tenantID := ""
		if a.orgStore != nil {
			org, err := a.orgStore.GetOrganization(r.Context(), id)
			if err == nil && org != nil {
				name = org.Name
				tenantID = org.TenantID
			}
		}
		orgInfos = append(orgInfos, OrganizationInfo{ID: id, Name: name, TenantID: tenantID})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ListOrganizationsResponse{Organizations: orgInfos})
}

// DiscoverOrganization checks which organizations an email belongs to and returns their IDs/Names.
func (a *AuthHandler) DiscoverOrganization(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	email := r.URL.Query().Get("email")
	if email == "" {
		http.Error(w, "email query parameter is required", http.StatusBadRequest)
		return
	}

	orgIDs, err := a.userStore.DiscoverUserOrganizations(r.Context(), email)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to discover organizations: %v", err), http.StatusInternalServerError)
		return
	}

	var orgInfos []OrganizationInfo
	for _, id := range orgIDs {
		name := id
		tenantID := ""
		if a.orgStore != nil {
			org, err := a.orgStore.GetOrganization(r.Context(), id)
			if err == nil && org != nil {
				name = org.Name
				tenantID = org.TenantID
			}
		}
		orgInfos = append(orgInfos, OrganizationInfo{ID: id, Name: name, TenantID: tenantID})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(DiscoverOrganizationResponse{Organizations: orgInfos})
}

// SelectOrganization validates membership to a chosen org and returns a signed Org Session JWT.
func (a *AuthHandler) SelectOrganization(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok {
		http.Error(w, "Unauthorized context", http.StatusUnauthorized)
		return
	}

	var req SelectOrganizationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if req.OrgID == "" {
		http.Error(w, "org_id is required", http.StatusBadRequest)
		return
	}

	orgIDs, err := a.userStore.ListUserOrganizations(r.Context(), claims.Subject)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to query organizations: %v", err), http.StatusInternalServerError)
		return
	}

	if len(orgIDs) == 0 {
		orgIDs = claims.Organizations
	}
	if len(orgIDs) == 0 && a.orgStore != nil {
		allOrgs, err := a.orgStore.ListOrganizations(r.Context())
		if err == nil {
			for _, o := range allOrgs {
				orgIDs = append(orgIDs, o.ID)
			}
		}
	}

	authorized := false
	for _, id := range orgIDs {
		if id == req.OrgID {
			authorized = true
			break
		}
	}

	if authorized {
		err := a.loadUserRolesAndPermissions(r.Context(), req.OrgID, claims)
		if err != nil {
			slog.Warn("Failed to load user profile for authorized org", "orgID", req.OrgID, "error", err)
		}
	}

	if !authorized {
		http.Error(w, "Forbidden: User is not a member of this organization", http.StatusForbidden)
		return
	}

	jwtSecret := a.config.JwtSecret
	if jwtSecret == "" {
		jwtSecret = "dev-secret-key-that-must-be-changed-in-production-12345"
	}

	token, err := MintOrgSessionToken(claims.Subject, req.OrgID, claims.Roles, claims.Permissions, claims.IsSuperAdmin, jwtSecret)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to mint session token: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(SelectOrganizationResponse{Token: token})
}

func (a *AuthHandler) loadUserRolesAndPermissions(ctx context.Context, orgID string, claims *core.UserClaims) error {
	var userRoles []core.UserRole
	user, err := a.userStore.GetUser(ctx, orgID, claims.Subject)
	if err == nil && user != nil {
		userRoles = user.Roles
		if len(userRoles) == 0 && user.Role != "" {
			userRoles = []core.UserRole{user.Role}
		}
	} else {
		for _, r := range claims.Roles {
			userRoles = append(userRoles, core.UserRole(r))
		}
	}

	var roles []string
	var allPerms []string
	seenPerms := make(map[string]bool)

	for _, roleVal := range userRoles {
		roles = append(roles, string(roleVal))
		perms, err := core.GetPermissionsForRole(ctx, roleVal)
		if err == nil {
			for _, p := range perms {
				if !seenPerms[p] {
					seenPerms[p] = true
					allPerms = append(allPerms, p)
				}
			}
		}
	}
	claims.Roles = roles
	claims.Permissions = allPerms
	return nil
}


// DiscoverOrganizationResponse holds the discovered organization lists.
type DiscoverOrganizationResponse struct {
	Organizations []OrganizationInfo `json:"organizations"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

type OrganizationInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	TenantID string `json:"tenant_id,omitempty"`
}

type ListOrganizationsResponse struct {
	Organizations []OrganizationInfo `json:"organizations"`
}

type SelectOrganizationRequest struct {
	OrgID string `json:"org_id"`
}

type SelectOrganizationResponse struct {
	Token string `json:"token"`
}
