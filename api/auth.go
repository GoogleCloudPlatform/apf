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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"cloud.google.com/go/firestore"
	"firebase.google.com/go/v4/auth"
	"google.com/apf/api/core"
	"google.com/apf/api/db"
)

// Authenticator defines interface for swappable auth providers.
type Authenticator interface {
	VerifyToken(token string) (*core.UserClaims, error)
	GenerateMockToken(username string) (string, error)
	CreateUser(ctx context.Context, orgID string, email string, password string, displayName string) (string, error)
	SetUserClaims(ctx context.Context, orgID string, uid string, claims map[string]interface{}) error
}

// mockPermissionsForRoles maps mock roles to their predefined permissions for simulation.
func mockPermissionsForRoles(roles []string) []string {
	if len(roles) == 0 {
		return nil
	}
	r := core.UserRole(roles[0])
	if perms, ok := core.PredefinedRolePermissions[r]; ok {
		return perms
	}
	return nil
}

// MockAuthenticator is a standard local-velocity simulation provider.
type MockAuthenticator struct{}

func (m *MockAuthenticator) VerifyToken(token string) (*core.UserClaims, error) {
	if !strings.HasPrefix(token, "simulated-jwt-") {
		return nil, fmt.Errorf("invalid mock token prefix")
	}
	username := strings.TrimPrefix(token, "simulated-jwt-token-for-")
	organizationID := ""
	if idx := strings.Index(username, "-organization-"); idx != -1 {
		organizationID = username[idx+len("-organization-"):]
		username = username[:idx]
	}
	roles := []string{"viewer"}
	isSuperAdmin := false
	switch username {
	case "admin":
		roles = []string{"owner"}
	case "super-admin":
		roles = []string{"owner"}
		isSuperAdmin = true
	}
	subject := username
	if username == "admin" {
		if appConfig != nil {
			subject = appConfig.SimulatedUserID
		} else {
			subject = core.NewConfig().SimulatedUserID
		}
	}
	var organizations []string
	if organizationID != "" {
		organizations = []string{organizationID}
	}
	return &core.UserClaims{
		Subject:        subject,
		Username:       username,
		Roles:          roles,
		OrganizationID: organizationID,
		Organizations:  organizations,
		IsSuperAdmin:   isSuperAdmin,
		Permissions:    mockPermissionsForRoles(roles),
	}, nil
}

func (m *MockAuthenticator) GenerateMockToken(username string) (string, error) {
	return "simulated-jwt-token-for-" + username, nil
}

func (m *MockAuthenticator) CreateUser(ctx context.Context, orgID string, email string, password string, displayName string) (string, error) {
	uid := "simulated-uid-" + strings.ReplaceAll(strings.Split(email, "@")[0], ".", "-")
	return uid, nil
}

func (m *MockAuthenticator) SetUserClaims(ctx context.Context, orgID string, uid string, claims map[string]interface{}) error {
	return nil
}

// FirebaseAuthAuthenticator verifies tokens using Firebase Admin SDK.
type FirebaseAuthAuthenticator struct {
	Client          *auth.Client
	FirestoreClient *firestore.Client
}

func (f *FirebaseAuthAuthenticator) VerifyToken(token string) (*core.UserClaims, error) {
	if strings.HasPrefix(token, "simulated-jwt-token-for-") {
		return f.verifySimulatedToken(token)
	}
	return f.verifyFirebaseToken(token)
}

func (f *FirebaseAuthAuthenticator) verifySimulatedToken(token string) (*core.UserClaims, error) {
	fullUsername := strings.TrimPrefix(token, "simulated-jwt-token-for-")
	username := fullUsername
	organizationID := ""
	if idx := strings.Index(fullUsername, "-organization-"); idx != -1 {
		organizationID = fullUsername[idx+len("-organization-"):]
		username = fullUsername[:idx]
	}

	roles := []string{"viewer"}
	subject := "simulated-" + username + "-uid"
	isSuperAdmin := false

	switch username {
	case "admin":
		subject = "simulated-admin-uid"
		if appConfig != nil {
			subject = appConfig.SimulatedUserID
		} else {
			subject = core.NewConfig().SimulatedUserID
		}
		roles = []string{string(core.RoleOrgOwner), string(core.RoleProjectCreator)}
	case "user1":
		subject = "simulated-user1-uid"
		roles = []string{string(core.RoleOrgViewer), string(core.RoleProjectCreator)}
	case "user2":
		subject = "simulated-user2-uid"
		roles = []string{string(core.RoleOrgViewer)}
	case "super-admin":
		subject = "simulated-super-admin-uid"
		roles = []string{string(core.RoleOrgOwner)}
		isSuperAdmin = true
	case "org-owner":
		subject = "simulated-org-owner-uid"
		roles = []string{string(core.RoleOrgOwner)}
	case "org-owner@the-high-council.com":
		subject = "org-owner@the-high-council.com"
		roles = []string{string(core.RoleOrgOwner)}
	case "org-admin@the-high-council.com":
		subject = "org-admin@the-high-council.com"
		roles = []string{string(core.RoleOrgAdmin)}
	case "user-admin@the-high-council.com":
		subject = "user-admin@the-high-council.com"
		roles = []string{string(core.RoleUserAdmin)}
	}

	var organizations []string
	if organizationID != "" {
		organizations = []string{organizationID}
	}

	return &core.UserClaims{
		Subject:        subject,
		Username:       username,
		Roles:          roles,
		OrganizationID: organizationID,
		Organizations:  organizations,
		IsSuperAdmin:   isSuperAdmin,
		Permissions:    mockPermissionsForRoles(roles),
	}, nil
}

func (f *FirebaseAuthAuthenticator) verifyFirebaseToken(token string) (*core.UserClaims, error) {
	// Extract tenant ID from the token payload to support multi-tenancy.
	var tenantID string
	parts := strings.Split(token, ".")
	if len(parts) == 3 {
		payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
		if err == nil {
			var payload struct {
				Firebase struct {
					Tenant string `json:"tenant"`
				} `json:"firebase"`
			}
			if json.Unmarshal(payloadBytes, &payload) == nil {
				tenantID = payload.Firebase.Tenant
			}
		}
	}

	var decodedToken *auth.Token
	var err error
	if tenantID != "" {
		tenantClient, errTenant := f.Client.TenantManager.AuthForTenant(tenantID)
		if errTenant != nil {
			return nil, fmt.Errorf("failed to get auth client for tenant %s: %w", tenantID, errTenant)
		}
		decodedToken, err = tenantClient.VerifyIDToken(context.Background(), token)
	} else {
		decodedToken, err = f.Client.VerifyIDToken(context.Background(), token)
	}

	if err != nil {
		return nil, err
	}

	roles := []string{}
	if rolesClaim, ok := decodedToken.Claims["roles"].([]interface{}); ok {
		for _, v := range rolesClaim {
			if s, ok := v.(string); ok {
				roles = append(roles, s)
			}
		}
	}
	if len(roles) == 0 {
		roles = []string{"viewer"} // Fallback
	}

	permissions := []string{}
	if permissionsClaim, ok := decodedToken.Claims["permissions"].([]interface{}); ok {
		for _, v := range permissionsClaim {
			if s, ok := v.(string); ok {
				permissions = append(permissions, s)
			}
		}
	}
	if len(permissions) == 0 {
		for _, r := range roles {
			perms, err := core.GetPermissionsForRole(context.Background(), core.UserRole(r))
			if err == nil {
				permissions = append(permissions, perms...)
			}
		}
	}

	username, _ := decodedToken.Claims["email"].(string)
	if username == "" {
		username = decodedToken.UID // Fallback
	}

	organizationID, _ := decodedToken.Claims["organization_id"].(string)
	if organizationID == "" {
		organizationID = decodedToken.Firebase.Tenant
	}

	isSuperAdmin := false
	if isa, ok := decodedToken.Claims["is_super_admin"].(bool); ok && isa {
		if organizationID == "" {
			isSuperAdmin = true
		}
	}

	organizations := []string{}
	if orgs, ok := decodedToken.Claims["organizations"].([]interface{}); ok {
		for _, v := range orgs {
			if s, ok := v.(string); ok {
				organizations = append(organizations, s)
			}
		}
	}
	if len(organizations) == 0 && organizationID != "" {
		organizations = []string{organizationID}
	}

	return &core.UserClaims{
		Subject:        decodedToken.UID,
		Username:       username,
		Roles:          roles,
		OrganizationID: organizationID,
		Organizations:  organizations,
		IsSuperAdmin:   isSuperAdmin,
		Permissions:    permissions,
	}, nil
}

func (f *FirebaseAuthAuthenticator) GenerateMockToken(username string) (string, error) {
	return "simulated-jwt-token-for-" + username, nil
}

func (f *FirebaseAuthAuthenticator) resolveTenantID(ctx context.Context, orgID string) (string, error) {
	if f.FirestoreClient == nil {
		return "", fmt.Errorf("firestore client is not initialized in authenticator")
	}
	doc, err := f.FirestoreClient.Collection("organizations").Doc(orgID).Get(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to load organization %s: %w", orgID, err)
	}
	var org struct {
		TenantID string `firestore:"tenant_id"`
	}
	if err := doc.DataTo(&org); err != nil {
		return "", fmt.Errorf("failed to decode organization %s: %w", orgID, err)
	}
	if org.TenantID == "" {
		return "", fmt.Errorf("organization %s has no tenant ID configured", orgID)
	}
	return org.TenantID, nil
}

func (f *FirebaseAuthAuthenticator) CreateUser(ctx context.Context, orgID string, email string, password string, displayName string) (string, error) {
	tenantID, err := f.resolveTenantID(ctx, orgID)
	if err != nil {
		return "", err
	}
	tenantClient, err := f.Client.TenantManager.AuthForTenant(tenantID)
	if err != nil {
		return "", fmt.Errorf("failed to get auth client for tenant %s: %w", tenantID, err)
	}
	params := (&auth.UserToCreate{}).
		Email(email).
		EmailVerified(true).
		Password(password)

	if displayName != "" {
		params.DisplayName(displayName)
	}

	u, err := tenantClient.CreateUser(ctx, params)
	if err != nil {
		return "", fmt.Errorf("failed to create user in Firebase Auth: %w", err)
	}
	return u.UID, nil
}

func (f *FirebaseAuthAuthenticator) SetUserClaims(ctx context.Context, orgID string, uid string, claims map[string]interface{}) error {
	tenantID, err := f.resolveTenantID(ctx, orgID)
	if err != nil {
		return err
	}
	tenantClient, err := f.Client.TenantManager.AuthForTenant(tenantID)
	if err != nil {
		return fmt.Errorf("failed to get auth client for tenant %s: %w", tenantID, err)
	}
	err = tenantClient.SetCustomUserClaims(ctx, uid, claims)
	if err != nil {
		return fmt.Errorf("failed to set custom user claims: %w", err)
	}
	return nil
}

var globalAuthenticator Authenticator = &MockAuthenticator{}

// OrganizationProvisioner defines interface for organization lifecycle operations.
type OrganizationProvisioner interface {
	CreateOrganization(ctx context.Context, name string) (string, error)
	CreateOrganizationOwner(ctx context.Context, tenantID string, orgID string, email string) (string, error)
	DeleteOrganization(ctx context.Context, orgID string) error
}

// MockOrganizationProvisioner is a simulation of OrganizationProvisioner for local development.
type MockOrganizationProvisioner struct{}

func (m *MockOrganizationProvisioner) CreateOrganization(ctx context.Context, name string) (string, error) {
	orgID := "mock-org-" + strings.ToLower(strings.ReplaceAll(name, " ", "-"))
	return orgID, nil
}

func (m *MockOrganizationProvisioner) CreateOrganizationOwner(ctx context.Context, tenantID string, orgID string, email string) (string, error) {
	ownerUID := "mock-owner-" + strings.ToLower(strings.ReplaceAll(email, "@", "-at-"))
	return ownerUID, nil
}

func (m *MockOrganizationProvisioner) DeleteOrganization(ctx context.Context, orgID string) error {
	return nil
}

// FirebaseOrganizationProvisioner implements OrganizationProvisioner using Firebase Admin SDK.
type FirebaseOrganizationProvisioner struct {
	TenantManager *auth.TenantManager
}

func (f *FirebaseOrganizationProvisioner) CreateOrganization(ctx context.Context, name string) (string, error) {
	tenantToCreate := (&auth.TenantToCreate{}).
		DisplayName(name).
		AllowPasswordSignUp(true).
		EnableEmailLinkSignIn(true)
	tenant, err := f.TenantManager.CreateTenant(ctx, tenantToCreate)
	if err != nil {
		return "", fmt.Errorf("failed to create organization in Firebase: %w", err)
	}
	return tenant.ID, nil
}

func (f *FirebaseOrganizationProvisioner) CreateOrganizationOwner(ctx context.Context, tenantID string, orgID string, email string) (string, error) {
	tenantClient, err := f.TenantManager.AuthForTenant(tenantID)
	if err != nil {
		return "", fmt.Errorf("failed to get organization client for tenant %s: %w", tenantID, err)
	}

	userToCreate := (&auth.UserToCreate{}).Email(email).EmailVerified(true)
	userRecord, err := tenantClient.CreateUser(ctx, userToCreate)
	if err != nil {
		return "", fmt.Errorf("failed to create user in organization: %w", err)
	}

	claims := map[string]interface{}{
		"role":            "owner",
		"organization_id": orgID,
	}
	if err := tenantClient.SetCustomUserClaims(ctx, userRecord.UID, claims); err != nil {
		return "", fmt.Errorf("failed to set owner claim for organization user: %w", err)
	}

	return userRecord.UID, nil
}

func (f *FirebaseOrganizationProvisioner) DeleteOrganization(ctx context.Context, orgID string) error {
	if err := f.TenantManager.DeleteTenant(ctx, orgID); err != nil {
		return fmt.Errorf("failed to delete organization in Firebase: %w", err)
	}
	return nil
}

var globalOrganizationProvisioner OrganizationProvisioner = &MockOrganizationProvisioner{}

var organizationStore db.OrganizationStore
