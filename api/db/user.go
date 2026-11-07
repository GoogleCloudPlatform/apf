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

package db

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"firebase.google.com/go/v4/auth"
	"google.com/apf/api/core"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ErrNotFound is returned when a requested firestore document is not found.
var ErrNotFound = errors.New("record not found")

// ErrLastAdmin is returned when an action would leave the organization with zero administrators.
var ErrLastAdmin = errors.New("cannot demote or remove the last administrator of the organization")

// UserPreferences stores general user settings like theme choice
type UserPreferences struct {
	Theme string `firestore:"theme" json:"theme"` // "dark" or "light"
}

// LastActiveStep represents the last step the user was active on in a section
type LastActiveStep struct {
	SectionID string `firestore:"section_id" json:"section_id"`
	StepIndex int    `firestore:"step_index" json:"step_index"`
}

// TourStatus tracks the user's progress through the onboarding tour
type TourStatus struct {
	Completed         bool           `firestore:"completed" json:"completed"`
	CompletedSections []string       `firestore:"completed_sections" json:"completed_sections"`
	LastActiveStep    LastActiveStep `firestore:"last_active_step" json:"last_active_step"`
}

// UserRole defines the role of a user in an organization
type UserRole = core.UserRole

const (
	RoleOrgOwner      UserRole = core.RoleOrgOwner
	RoleOrgAdmin      UserRole = core.RoleOrgAdmin
	RoleOrgEditor     UserRole = core.RoleOrgEditor
	RoleOrgViewer     UserRole = core.RoleOrgViewer

	RoleProjectAdmin  UserRole = core.RoleProjectAdmin
	RoleProjectEditor UserRole = core.RoleProjectEditor
	RoleProjectViewer UserRole = core.RoleProjectViewer
	RoleProjectCreator UserRole = core.RoleProjectCreator

	RoleUserAdmin     UserRole = core.RoleUserAdmin
	RoleUserEditor    UserRole = core.RoleUserEditor
	RoleUserViewer    UserRole = core.RoleUserViewer

)



// User represents global user metadata and options
type User struct {
	ID          string          `firestore:"-" json:"id"`
	UserID      string          `firestore:"userId,omitempty" json:"userId,omitempty"`
	Email       string          `firestore:"email,omitempty" json:"email,omitempty"`
	Role        UserRole        `firestore:"role,omitempty" json:"role,omitempty"`
	Roles       []UserRole      `firestore:"roles,omitempty" json:"roles,omitempty"`
	DisplayName string          `firestore:"displayName,omitempty" json:"displayName,omitempty"`
	Preferences UserPreferences `firestore:"preferences" json:"preferences"`
	TourStatus  *TourStatus     `firestore:"tour_status,omitempty" json:"tour_status,omitempty"`
	LastLogin   *time.Time      `firestore:"last_login,omitempty" json:"lastLogin,omitempty"`
}

// AppCredential represents decrypted/encrypted credential parameters for a connected application
type AppCredential struct {
	ID            string                `firestore:"-" json:"id"`
	IntegrationID string                `firestore:"integrationId" json:"integrationId"` // e.g. "github", "figma"
	Fields        []CredentialAttribute `firestore:"fields" json:"fields"`
	LastLogin     time.Time             `firestore:"lastLogin" json:"lastLogin"`
}

// UserProject represents individual user configurations scoped to a specific project and app
type UserProject struct {
	ID                   string                `firestore:"-" json:"id"`
	UserID               string                `firestore:"userId" json:"userId"`
	ProjectID            string                `firestore:"projectId" json:"projectId"`
	AppID                string                `firestore:"appId" json:"appId"`
	Enabled              bool                  `firestore:"enabled" json:"enabled"`
	CredentialID         string                `firestore:"credentialId,omitempty" json:"credentialId,omitempty"`
	IsOverridden         bool                  `firestore:"isOverridden" json:"isOverridden"`
	Fields               []CredentialAttribute `firestore:"fields,omitempty" json:"fields,omitempty"`
	IsReadOnly           bool                  `firestore:"isReadOnly" json:"isReadOnly"`
	CanSubmitPullRequest bool                  `firestore:"canSubmitPullRequest" json:"canSubmitPullRequest"`
	CanWrite             bool                  `firestore:"canWrite" json:"canWrite"`
	UpdatedAt            time.Time             `firestore:"updatedAt" json:"updatedAt"`
}

// UserStore manages users collection
type UserStore interface {
	GetUser(ctx context.Context, organizationID string, id string) (*User, error)
	GetUserByEmail(ctx context.Context, organizationID string, email string) (*User, error)
	GetOrCreateUser(ctx context.Context, organizationID string, id string) (*User, error)
	UpdateUserPreferences(ctx context.Context, organizationID string, id string, prefs *UserPreferences) error
	UpdateTourStatus(ctx context.Context, organizationID string, id string, status *TourStatus) error
	ListUserOrganizations(ctx context.Context, userID string) ([]string, error)
	DiscoverUserOrganizations(ctx context.Context, email string) ([]string, error)
	ReconcileUserMemberships(ctx context.Context, email string, targetUserID string) error
	ReconcileAutojoinRules(ctx context.Context, email string, userID string) error
	UpdateUserRole(ctx context.Context, organizationID string, id string, role UserRole) error
	DeleteUser(ctx context.Context, organizationID string, id string) error
	ListUsers(ctx context.Context, organizationID string) ([]User, error)
	GetUserCount(ctx context.Context, organizationID string) (int, error)
	CreateUser(ctx context.Context, organizationID string, user *User) error
}

// AppCredentialStore manages users/{userId}/app_credentials subcollection
type AppCredentialStore interface {
	GetCredential(ctx context.Context, organizationID string, userId, appId string) (*AppCredential, error)
	SaveCredential(ctx context.Context, organizationID string, userId, appId string, cred *AppCredential) error
	DeleteCredential(ctx context.Context, organizationID string, userId, appId string) error
}

// UserProjectStore manages user-project-app scoped settings
type UserProjectStore interface {
	GetProjectAppSetting(ctx context.Context, organizationID string, userId, projectId, appId string) (*UserProject, error)
	SaveProjectAppSetting(ctx context.Context, organizationID string, setting *UserProject) error
}

// FirestoreUserStore implements UserStore using Cloud Firestore and Google Identity Platform (Firebase Auth)
type FirestoreUserStore struct {
	client        *firestore.Client
	authClient    *auth.Client
	autojoinStore AutojoinStore
}

// NewFirestoreUserStore creates a new FirestoreUserStore
func NewFirestoreUserStore(client *firestore.Client, authClient *auth.Client) *FirestoreUserStore {
	return &FirestoreUserStore{
		client:        client,
		authClient:    authClient,
		autojoinStore: NewFirestoreAutojoinStore(client),
	}
}

// WithAutojoinStore sets the autojoin store instance for the user store (useful for sharing cache).
func (s *FirestoreUserStore) WithAutojoinStore(autojoinStore AutojoinStore) *FirestoreUserStore {
	s.autojoinStore = autojoinStore
	return s
}


func (s *FirestoreUserStore) getTenantClient(ctx context.Context, orgID string) (*auth.TenantClient, error) {
	if s.authClient == nil {
		return nil, nil
	}
	
	// Load organization to resolve decoupled tenant ID
	doc, err := s.client.Collection("organizations").Doc(orgID).Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load organization %s to resolve tenant: %w", orgID, err)
	}
	
	var org struct {
		TenantID string `firestore:"tenant_id"`
	}
	if err := doc.DataTo(&org); err != nil {
		return nil, fmt.Errorf("failed to decode organization %s: %w", orgID, err)
	}
	
	if org.TenantID == "" {
		return nil, fmt.Errorf("organization %s has no tenant ID configured", orgID)
	}

	tenantManager := s.authClient.TenantManager
	return tenantManager.AuthForTenant(org.TenantID)
}

// GetOrCreateUser retrieves user preferences or boots up default settings
func (s *FirestoreUserStore) GetOrCreateUser(ctx context.Context, organizationID string, id string) (*User, error) {
	if id == "" {
		return nil, fmt.Errorf("userId cannot be empty")
	}
	col := organizationCollection(organizationID, s.client, "users")
	docRef := col.Doc(id)
	doc, err := docRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			var email string
			claims, ok := ctx.Value(core.UserClaimsKey).(*core.UserClaims)
			if ok && claims != nil {
				email = claims.Username
				if email == "" && strings.Contains(claims.Subject, "@") {
					email = claims.Subject
				}
			}

			userRoles := []UserRole{RoleOrgViewer} // default roles

			// Initialize default record
			user := &User{
				ID:     id,
				UserID: id,
				Email:  email,
				Role:   userRoles[0],
				Roles:  userRoles,
				Preferences: UserPreferences{
					Theme: "dark",
				},
				TourStatus: &TourStatus{
					Completed:         false,
					CompletedSections: []string{},
					LastActiveStep: LastActiveStep{
						SectionID: "",
						StepIndex: 0,
					},
				},
				LastLogin: func() *time.Time { t := time.Now(); return &t }(),
			}
			_, err = docRef.Set(ctx, user)
			if err != nil {
				return nil, err
			}

			return user, nil
		}
		return nil, err
	}

	var user User
	if err := doc.DataTo(&user); err != nil {
		return nil, err
	}
	user.ID = doc.Ref.ID

	if len(user.Roles) == 0 && user.Role != "" {
		user.Roles = []UserRole{user.Role}
	}

	// Reconcile in place if it's an invitation doc (UserID is empty)
	if user.UserID == "" {
		user.UserID = id
		if len(user.Roles) == 0 && user.Role != "" {
			user.Roles = []UserRole{user.Role}
		}
		user.Preferences = UserPreferences{
			Theme: "dark",
		}
		user.TourStatus = &TourStatus{
			Completed:         false,
			CompletedSections: []string{},
			LastActiveStep: LastActiveStep{
				SectionID: "",
				StepIndex: 0,
			},
		}
		_, err = docRef.Set(ctx, &user)
		if err != nil {
			return nil, err
		}
	}

	return &user, nil
}

// GetUser retrieves a user document from Firestore without creating it
func (s *FirestoreUserStore) GetUser(ctx context.Context, organizationID string, id string) (*User, error) {
	if id == "" {
		return nil, fmt.Errorf("userId cannot be empty")
	}
	col := organizationCollection(organizationID, s.client, "users")
	doc, err := col.Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	var user User
	if err := doc.DataTo(&user); err != nil {
		return nil, err
	}
	user.ID = doc.Ref.ID
	if len(user.Roles) == 0 && user.Role != "" {
		user.Roles = []UserRole{user.Role}
	}
	return &user, nil
}

// GetUserByEmail retrieves a user document by email within an organization
func (s *FirestoreUserStore) GetUserByEmail(ctx context.Context, organizationID string, email string) (*User, error) {
	if email == "" {
		log.Printf("WARN: GetUserByEmail called with empty email")
		return nil, ErrNotFound
	}
	col := organizationCollection(organizationID, s.client, "users")
	iter := col.Where("email", "==", strings.ToLower(strings.TrimSpace(email))).Limit(1).Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if err == iterator.Done {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	var user User
	if err := doc.DataTo(&user); err != nil {
		return nil, err
	}
	user.ID = doc.Ref.ID
	if len(user.Roles) == 0 && user.Role != "" {
		user.Roles = []UserRole{user.Role}
	}
	return &user, nil
}

// CreateUser creates a new user document in Firestore under /organizations/{orgID}/users/{userID}
func (s *FirestoreUserStore) CreateUser(ctx context.Context, organizationID string, user *User) error {
	if organizationID == "" {
		return fmt.Errorf("organizationID cannot be empty")
	}
	if user == nil {
		return fmt.Errorf("user cannot be nil")
	}
	docID := user.UserID
	if docID == "" {
		docID = user.Email
	}
	if docID == "" {
		return fmt.Errorf("userId and email cannot both be empty")
	}
	col := organizationCollection(organizationID, s.client, "users")
	_, err := col.Doc(docID).Set(ctx, user)
	return err
}


// UpdateUserPreferences updates custom preference variables (like theme mode)
func (s *FirestoreUserStore) UpdateUserPreferences(ctx context.Context, organizationID string, id string, prefs *UserPreferences) error {
	if id == "" {
		return fmt.Errorf("userId cannot be empty")
	}
	col := organizationCollection(organizationID, s.client, "users")
	_, err := col.Doc(id).Set(ctx, map[string]interface{}{
		"preferences": prefs,
	}, firestore.MergeAll)
	return err
}

// UpdateTourStatus updates the user's tour progress in Firestore
func (s *FirestoreUserStore) UpdateTourStatus(ctx context.Context, organizationID string, id string, status *TourStatus) error {
	if id == "" {
		return fmt.Errorf("userId cannot be empty")
	}
	col := organizationCollection(organizationID, s.client, "users")
	_, err := col.Doc(id).Set(ctx, map[string]interface{}{
		"tour_status": status,
	}, firestore.MergeAll)
	return err
}

// ListUserOrganizations retrieves all organization IDs the user is part of by scanning for matching user docs.
func (s *FirestoreUserStore) ListUserOrganizations(ctx context.Context, userID string) ([]string, error) {
	if userID == "" {
		return nil, fmt.Errorf("userId cannot be empty")
	}

	iter := s.client.CollectionGroup("users").Where("userId", "==", userID).Documents(ctx)
	defer iter.Stop()

	var orgs []string
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		if doc.Ref.Parent != nil && doc.Ref.Parent.Parent != nil {
			orgs = append(orgs, doc.Ref.Parent.Parent.ID)
		}
	}
	return orgs, nil
}

// DiscoverUserOrganizations retrieves all organization IDs associated with an email address by scanning user docs.
func (s *FirestoreUserStore) DiscoverUserOrganizations(ctx context.Context, email string) ([]string, error) {
	if email == "" {
		return nil, fmt.Errorf("email cannot be empty")
	}

	iter := s.client.CollectionGroup("users").Where("email", "==", email).Documents(ctx)
	defer iter.Stop()

	var orgs []string
	seen := make(map[string]bool)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		if doc.Ref.Parent != nil && doc.Ref.Parent.Parent != nil {
			orgID := doc.Ref.Parent.Parent.ID
			if !seen[orgID] {
				seen[orgID] = true
				orgs = append(orgs, orgID)
			}
		}
	}

	// Fallback: If no existing memberships are found, check active autojoin rules
	if len(orgs) == 0 {
		parts := strings.Split(email, "@")
		if len(parts) == 2 {
			domain := strings.ToLower(parts[1])
			if s.autojoinStore != nil {
				rule, err := s.autojoinStore.FindActiveRuleByDomain(ctx, domain)
				if err == nil && rule != nil {
					orgs = append(orgs, rule.OrganizationID)
				}
			}
		}
	}

	return orgs, nil
}

// ReconcileUserMemberships finds all user documents with the given email and,
// if their document ID / userId field does not match targetUserID (the IAP Subject),
// it reconciles them by creating a new document keyed by targetUserID and deleting the old one.
func (s *FirestoreUserStore) ReconcileUserMemberships(ctx context.Context, email string, targetUserID string) error {
	if email == "" {
		return fmt.Errorf("email cannot be empty")
	}
	if targetUserID == "" {
		return fmt.Errorf("targetUserID cannot be empty")
	}

	// Query all users with this email across all organizations
	iter := s.client.CollectionGroup("users").Where("email", "==", email).Documents(ctx)
	defer iter.Stop()

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to iterate users: %w", err)
		}

		// Get organization ID from path
		if doc.Ref.Parent == nil || doc.Ref.Parent.Parent == nil {
			continue
		}
		orgID := doc.Ref.Parent.Parent.ID

		// If the document ID is already the targetUserID, it is already reconciled
		if doc.Ref.ID == targetUserID {
			continue
		}

		log.Printf("RECONCILE: Found unreconciled membership for %s in org %s. Old ID: %s, New ID: %s", email, orgID, doc.Ref.ID, targetUserID)

		// Run a transaction to safely copy and delete
		err = s.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
			// Read the old document again in transaction
			oldDocRef := doc.Ref
			oldDoc, err := tx.Get(oldDocRef)
			if err != nil {
				return err
			}

			var user User
			if err := oldDoc.DataTo(&user); err != nil {
				return err
			}

			// Update fields for the reconciled user
			user.ID = targetUserID
			user.UserID = targetUserID

			// Create new document with targetUserID as ID
			newDocRef := s.client.Collection("organizations").Doc(orgID).Collection("users").Doc(targetUserID)
			
			// Check if new document already exists (should not, but just in case)
			_, err = tx.Get(newDocRef)
			if err == nil {
				log.Printf("RECONCILE_WARN: Target document %s already exists in org %s, overwriting", targetUserID, orgID)
			}

			err = tx.Set(newDocRef, user)
			if err != nil {
				return err
			}

			// Delete old document
			return tx.Delete(oldDocRef)
		})

		if err != nil {
			return fmt.Errorf("failed to reconcile membership for %s in org %s: %w", email, orgID, err)
		}
		log.Printf("RECONCILE_SUCCESS: Reconciled membership for %s in org %s to ID %s", email, orgID, targetUserID)

		if ProjectStoreInstance != nil {
			err = ProjectStoreInstance.ReconcileMemberID(ctx, orgID, doc.Ref.ID, targetUserID)
			if err != nil {
				log.Printf("RECONCILE_PROJECT_WARN: Failed to reconcile project memberships for %s in org %s: %v", email, orgID, err)
			}
		}
	}

	return nil
}

// ReconcileAutojoinRules checks for active domain rules and auto-provisions membership if matched.
func (s *FirestoreUserStore) ReconcileAutojoinRules(ctx context.Context, email string, userID string) error {
	if email == "" || userID == "" {
		return nil
	}

	// 1. Extract email domain (e.g., "user@google.com" -> "google.com")
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return fmt.Errorf("invalid email format: %s", email)
	}
	domain := strings.ToLower(parts[1])

	// 2. Query active autojoin rule for domain
	if s.autojoinStore == nil {
		return fmt.Errorf("autojoinStore is not initialized")
	}
	rule, err := s.autojoinStore.FindActiveRuleByDomain(ctx, domain)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil // No autojoin rule, exit gracefully
		}
		return err
	}

	// 3. Run a transaction to verify and safely provision the user
	err = s.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		userDocRef := s.client.Collection("organizations").Doc(rule.OrganizationID).Collection("users").Doc(userID)

		doc, err := tx.Get(userDocRef)
		if err != nil {
			if status.Code(err) != codes.NotFound {
				return err // Return transient error to let Firestore abort & retry!
			}
		} else if doc.Exists() {
			return nil // User is already a member, nothing to do
		}

		// If user doc doesn't exist, create it inside the organization
		primaryRole := UserRole("viewer") // fallback default
		if len(rule.DefaultRoles) > 0 {
			primaryRole = rule.DefaultRoles[0]
		}

		newUser := &User{
			UserID:      userID,
			Email:       email,
			Role:        primaryRole,
			Roles:       rule.DefaultRoles,
			DisplayName: strings.Split(email, "@")[0], // Default display name to localpart
			Preferences: UserPreferences{Theme: "light"},
		}

		return tx.Create(userDocRef, newUser)
	})

	if err != nil {
		return fmt.Errorf("failed to auto-provision membership for %s in org %s: %w", email, rule.OrganizationID, err)
	}
	log.Printf("AUTOJOIN_SUCCESS: Auto-provisioned membership for %s in org %s with roles %v", email, rule.OrganizationID, rule.DefaultRoles)

	if ProjectStoreInstance != nil {
		err = ProjectStoreInstance.ReconcileMemberID(ctx, rule.OrganizationID, email, userID)
		if err != nil {
			log.Printf("AUTOJOIN_PROJECT_WARN: Failed to reconcile project memberships for %s in org %s: %v", email, rule.OrganizationID, err)
		}
	}

	return nil
}

// UpdateUserRole updates the role of a user document in Firestore and custom claims in Firebase Auth Tenant
func (s *FirestoreUserStore) UpdateUserRole(ctx context.Context, organizationID string, id string, role UserRole) error {
	if id == "" {
		return fmt.Errorf("user ID cannot be empty")
	}

	// Fetch permissions for the new role
	permissions, err := core.GetPermissionsForRole(ctx, role)
	if err != nil {
		return fmt.Errorf("failed to get permissions for role %q: %w", role, err)
	}

	col := organizationCollection(organizationID, s.client, "users")
	
	// Fetch all user document references outside the transaction to use for last-admin verification
	refs, err := col.DocumentRefs(ctx).GetAll()
	if err != nil {
		return fmt.Errorf("failed to get user references: %w", err)
	}

	var oldRole UserRole
	var oldRoles []UserRole

	// 1. Update Firestore inside a transaction to ensure atomicity of the last-admin check
	err = s.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		docRef := col.Doc(id)
		doc, err := tx.Get(docRef)
		if err != nil {
			return err
		}
		
		var targetUser User
		if err := doc.DataTo(&targetUser); err != nil {
			return err
		}
		oldRole = targetUser.Role
		oldRoles = targetUser.Roles

		isAdministrative := func(user User) bool {
			roles := user.Roles
			if len(roles) == 0 && user.Role != "" {
				roles = []UserRole{user.Role}
			}
			for _, r := range roles {
				if core.HasAdminPermissions(ctx, r) {
					return true
				}
			}
			return false
		}

		isNewRoleAdmin := core.HasAdminPermissions(ctx, role)

		// Prevent demoting the last admin/owner
		if isAdministrative(targetUser) && !isNewRoleAdmin {
			snapshots, err := tx.GetAll(refs)
			if err != nil {
				return fmt.Errorf("failed to fetch all users in transaction: %w", err)
			}
			
			adminCount := 0
			for _, snap := range snapshots {
				if !snap.Exists() {
					continue
				}
				var u User
				if err := snap.DataTo(&u); err != nil {
					return err
				}
				// Only count active admins/owners (have UserID, not a pending invite)
				if isAdministrative(u) && u.UserID != "" && !strings.Contains(snap.Ref.ID, "@") {
					adminCount++
				}
			}
			
			if adminCount <= 1 {
				return ErrLastAdmin
			}
		}

		newRoles := []UserRole{role}
		if role == "" {
			newRoles = []UserRole{}
		}
		return tx.Update(docRef, []firestore.Update{
			{Path: "role", Value: role},
			{Path: "roles", Value: newRoles},
		})
	})
	if err != nil {
		return err
	}

	// 2. Update Firebase Auth Tenant if enabled
	tenantClient, err := s.getTenantClient(ctx, organizationID)
	if err != nil {
		return fmt.Errorf("failed to get tenant client: %w", err)
	}
	if tenantClient != nil {
		if !strings.Contains(id, "@") {
			rolesClaim := []string{string(role)}
			if role == "" {
				rolesClaim = []string{}
			}
			claims := map[string]interface{}{
				"roles":       rolesClaim,
				"role":        string(role),
				"permissions": permissions,
			}
			err = tenantClient.SetCustomUserClaims(ctx, id, claims)
			if err != nil {
				// Rollback Firestore update on Firebase failure to maintain consistency
				rollbackErr := s.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
					return tx.Update(col.Doc(id), []firestore.Update{
						{Path: "role", Value: oldRole},
						{Path: "roles", Value: oldRoles},
					})
				})
				if rollbackErr != nil {
					log.Printf("CRITICAL: failed to roll back Firestore role update for user %s after Firebase failure: %v", id, rollbackErr)
				}
				return fmt.Errorf("failed to update firebase custom claims (Firestore rolled back): %w", err)
			}
		}
	}

	return nil
}

// DeleteUser removes a user document from Firestore and deletes them from Firebase Auth Tenant
func (s *FirestoreUserStore) DeleteUser(ctx context.Context, organizationID string, id string) error {
	if id == "" {
		return fmt.Errorf("userId cannot be empty")
	}

	col := organizationCollection(organizationID, s.client, "users")
	
	// Fetch all user document references outside the transaction to use for last-admin verification
	refs, err := col.DocumentRefs(ctx).GetAll()
	if err != nil {
		return fmt.Errorf("failed to get user references: %w", err)
	}

	// 1. Delete from Firestore inside a transaction to ensure atomicity of the last-admin check
	err = s.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		docRef := col.Doc(id)
		doc, err := tx.Get(docRef)
		if err != nil {
			return err
		}
		
		var targetUser User
		if err := doc.DataTo(&targetUser); err != nil {
			return err
		}

		isAdministrative := func(user User) bool {
			roles := user.Roles
			if len(roles) == 0 && user.Role != "" {
				roles = []UserRole{user.Role}
			}
			for _, r := range roles {
				if core.HasAdminPermissions(ctx, r) {
					return true
				}
			}
			return false
		}

		// Prevent removing the last admin/owner
		if isAdministrative(targetUser) {
			snapshots, err := tx.GetAll(refs)
			if err != nil {
				return fmt.Errorf("failed to fetch all users in transaction: %w", err)
			}
			
			adminCount := 0
			for _, snap := range snapshots {
				if !snap.Exists() {
					continue
				}
				var u User
				if err := snap.DataTo(&u); err != nil {
					return err
				}
				if isAdministrative(u) && u.UserID != "" && !strings.Contains(snap.Ref.ID, "@") {
					adminCount++
				}
			}
			
			if adminCount <= 1 {
				return ErrLastAdmin
			}
		}

		if err := tx.Delete(docRef); err != nil {
			return err
		}
		if targetUser.Email != "" {
			if err := tx.Delete(col.Doc(targetUser.Email)); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	// 2. Delete from Firebase Auth Tenant if enabled
	tenantClient, err := s.getTenantClient(ctx, organizationID)
	if err != nil {
		return fmt.Errorf("failed to get tenant client: %w", err)
	}
	if tenantClient != nil {
		if !strings.Contains(id, "@") {
			err = tenantClient.DeleteUser(ctx, id)
			if err != nil && !auth.IsUserNotFound(err) {
				// Return the error instead of silently logging to ensure visibility of failures
				return fmt.Errorf("failed to delete user from firebase auth: %w", err)
			}
		}
	}

	return nil
}

// ListUsers retrieves all users in the organization, merging active Firebase tenant users and pending Firestore invites
func (s *FirestoreUserStore) ListUsers(ctx context.Context, organizationID string) ([]User, error) {
	// 1. Fetch from Firestore (contains pending invites and local copies of active users)
	col := organizationCollection(organizationID, s.client, "users")
	docs, err := col.Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}
	
	var users []User
	var pendingInvites []User
	existingEmails := make(map[string]bool)

	for _, doc := range docs {
		var user User
		if err := doc.DataTo(&user); err != nil {
			return nil, err
		}
		user.ID = doc.Ref.ID
		
		if len(user.Roles) == 0 && user.Role != "" {
			user.Roles = []UserRole{user.Role}
		}

		if user.UserID == "" {
			// It's a pending invitation (keyed by email or has empty UserID)
			pendingInvites = append(pendingInvites, user)
		} else {
			users = append(users, user)
			if user.Email != "" {
				existingEmails[strings.ToLower(user.Email)] = true
			}
		}
	}

	// 2. Append pending invites, filtering out those that already have active accounts
	for _, invite := range pendingInvites {
		if invite.Email != "" && existingEmails[strings.ToLower(invite.Email)] {
			log.Printf("Info: Skipping pending invite for %s as they already have an active account", invite.Email)
			continue
		}
		users = append(users, invite)
	}

	return users, nil
}

// GetUserCount returns the count of users in the organization using a Firestore aggregation count query.
func (s *FirestoreUserStore) GetUserCount(ctx context.Context, organizationID string) (int, error) {
	col := organizationCollection(organizationID, s.client, "users")
	if col == nil {
		return 0, nil
	}
	res, err := col.NewAggregationQuery().WithCount("all").Get(ctx)
	if err != nil {
		return 0, err
	}
	if val, ok := res["all"]; ok {
		switch v := val.(type) {
		case int64:
			return int(v), nil
		case int:
			return v, nil
		case interface{ GetIntegerValue() int64 }:
			return int(v.GetIntegerValue()), nil
		}
	}
	return 0, nil
}




// FirestoreAppCredentialStore implements AppCredentialStore using Cloud Firestore
type FirestoreAppCredentialStore struct {
	client *firestore.Client
}

// NewFirestoreAppCredentialStore creates a new FirestoreAppCredentialStore
func NewFirestoreAppCredentialStore(client *firestore.Client) *FirestoreAppCredentialStore {
	return &FirestoreAppCredentialStore{client: client}
}

// GetCredential retrieves a user credential and transparently decrypts sensitive fields
func (s *FirestoreAppCredentialStore) GetCredential(ctx context.Context, organizationID string, userId, appId string) (*AppCredential, error) {
	if userId == "" || appId == "" {
		return nil, fmt.Errorf("userId and appId are required")
	}
	usersCol := organizationCollection(organizationID, s.client, "users")
	doc, err := usersCol.Doc(userId).Collection("app_credentials").Doc(appId).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, status.Error(codes.NotFound, "credential not found")
		}
		return nil, err
	}

	var cred AppCredential
	if err := doc.DataTo(&cred); err != nil {
		return nil, err
	}
	cred.ID = doc.Ref.ID

	// Transparent decryption
	for i, field := range cred.Fields {
		if field.IsSecret && field.Value != "" {
			decrypted, err := decryptSecret(field.Value)
			if err != nil {
				return nil, fmt.Errorf("failed to decrypt field %s: %w", field.Name, err)
			}
			cred.Fields[i].Value = decrypted
		}
	}

	return &cred, nil
}

// SaveCredential saves a user credential and transparently encrypts sensitive fields
func (s *FirestoreAppCredentialStore) SaveCredential(ctx context.Context, organizationID string, userId, appId string, cred *AppCredential) error {
	if userId == "" || appId == "" {
		return fmt.Errorf("userId and appId are required")
	}

	// Transparent encryption
	var encryptedFields []CredentialAttribute
	if cred.Fields != nil {
		encryptedFields = make([]CredentialAttribute, len(cred.Fields))
		for i, field := range cred.Fields {
			encryptedFields[i] = field
			if field.IsSecret && field.Value != "" && !strings.HasPrefix(field.Value, keyVersionHeader) {
				encrypted, err := encryptSecret(field.Value)
				if err != nil {
					return fmt.Errorf("failed to encrypt field %s: %w", field.Name, err)
				}
				encryptedFields[i].Value = encrypted
			}
		}
	}

	encryptedCred := AppCredential{
		IntegrationID: cred.IntegrationID,
		Fields:        encryptedFields,
		LastLogin:     time.Now(),
	}

	usersCol := organizationCollection(organizationID, s.client, "users")
	_, err := usersCol.Doc(userId).Collection("app_credentials").Doc(appId).Set(ctx, encryptedCred)
	return err
}

// DeleteCredential deletes a user credential
func (s *FirestoreAppCredentialStore) DeleteCredential(ctx context.Context, organizationID string, userId, appId string) error {
	if userId == "" || appId == "" {
		return fmt.Errorf("userId and appId are required")
	}
	usersCol := organizationCollection(organizationID, s.client, "users")
	_, err := usersCol.Doc(userId).Collection("app_credentials").Doc(appId).Delete(ctx)
	return err
}

// FirestoreUserProjectStore implements UserProjectStore using Cloud Firestore
type FirestoreUserProjectStore struct {
	client *firestore.Client
}

// NewFirestoreUserProjectStore creates a new FirestoreUserProjectStore
func NewFirestoreUserProjectStore(client *firestore.Client) *FirestoreUserProjectStore {
	return &FirestoreUserProjectStore{client: client}
}

// GetProjectAppSetting retrieves scoped settings for user project or default values
func (s *FirestoreUserProjectStore) GetProjectAppSetting(ctx context.Context, organizationID string, userId, projectId, appId string) (*UserProject, error) {
	if userId == "" || projectId == "" || appId == "" {
		return nil, fmt.Errorf("userId, projectId and appId are required")
	}
	parentDocID := fmt.Sprintf("%s_%s", userId, projectId)
	col := organizationCollection(organizationID, s.client, "user_project")
	doc, err := col.Doc(parentDocID).Collection("app_credentials").Doc(appId).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			// Return successful blank struct representing disabled default setting
			return &UserProject{
				UserID:    userId,
				ProjectID: projectId,
				AppID:     appId,
				Enabled:   false,
			}, nil
		}
		return nil, err
	}

	var setting UserProject
	if err := doc.DataTo(&setting); err != nil {
		return nil, err
	}
	setting.ID = doc.Ref.ID

	// Transparent decryption for overridden project-specific secrets
	if setting.IsOverridden && setting.Fields != nil {
		for i, field := range setting.Fields {
			if field.IsSecret && field.Value != "" {
				decrypted, err := decryptSecret(field.Value)
				if err != nil {
					return nil, fmt.Errorf("failed to decrypt user project field %s: %w", field.Name, err)
				}
				setting.Fields[i].Value = decrypted
			}
		}
	}

	return &setting, nil
}

// SaveProjectAppSetting updates project settings for this specific user
func (s *FirestoreUserProjectStore) SaveProjectAppSetting(ctx context.Context, organizationID string, setting *UserProject) error {
	if setting.UserID == "" || setting.ProjectID == "" || setting.AppID == "" {
		return fmt.Errorf("userId, projectId and appId are required in setting struct")
	}

	col := organizationCollection(organizationID, s.client, "user_project")

	// 1. Set parent base document to ensure 1 entry per user/per project exists!
	parentDocID := fmt.Sprintf("%s_%s", setting.UserID, setting.ProjectID)
	_, err := col.Doc(parentDocID).Set(ctx, map[string]interface{}{
		"userId":    setting.UserID,
		"projectId": setting.ProjectID,
		"updatedAt": time.Now(),
	}, firestore.MergeAll)
	if err != nil {
		return fmt.Errorf("failed to save parent user_project record: %w", err)
	}

	// 2. Transparent encryption for overridden project-specific secrets
	var encryptedFields []CredentialAttribute
	if setting.IsOverridden && setting.Fields != nil {
		encryptedFields = make([]CredentialAttribute, len(setting.Fields))
		for i, field := range setting.Fields {
			encryptedFields[i] = field
			if field.IsSecret && field.Value != "" && !strings.HasPrefix(field.Value, keyVersionHeader) {
				encrypted, err := encryptSecret(field.Value)
				if err != nil {
					return fmt.Errorf("failed to encrypt user project field %s: %w", field.Name, err)
				}
				encryptedFields[i].Value = encrypted
			}
		}
	} else {
		encryptedFields = setting.Fields
	}

	encryptedSetting := UserProject{
		UserID:               setting.UserID,
		ProjectID:            setting.ProjectID,
		AppID:                setting.AppID,
		Enabled:              setting.Enabled,
		CredentialID:         setting.CredentialID,
		IsOverridden:         setting.IsOverridden,
		Fields:               encryptedFields,
		IsReadOnly:           setting.IsReadOnly,
		CanSubmitPullRequest: setting.CanSubmitPullRequest,
		CanWrite:             setting.CanWrite,
		UpdatedAt:            time.Now(),
	}

	// 3. Set the subcollection settings document
	_, err = col.Doc(parentDocID).Collection("app_credentials").Doc(setting.AppID).Set(ctx, encryptedSetting)
	return err
}
