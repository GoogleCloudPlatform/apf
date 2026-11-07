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
	"os"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"google.com/apf/api/db"
)

func TestAutojoinIntegration(t *testing.T) {
	emulatorHost := os.Getenv("FIRESTORE_EMULATOR_HOST")
	if emulatorHost == "" {
		t.Skip("Skipping Autojoin integration test: FIRESTORE_EMULATOR_HOST is not set")
	}

	ctx := context.Background()
	client, err := firestore.NewClient(ctx, "agenticproductfoundry-489717")
	if err != nil {
		t.Fatalf("Failed to create Firestore client: %v", err)
	}
	defer client.Close()

	// Initialize stores
	autojoinStore := db.NewFirestoreAutojoinStore(client)
	userStore := db.NewFirestoreUserStore(client, nil) // authClient nil for unit tests
	orgStore := db.NewFirestoreOrganizationStore(client)

	// Set up target organization
	targetOrg := &db.Organization{
		Name: "Google Cloud Test Org",
	}
	orgID, err := orgStore.CreateOrganization(ctx, targetOrg)
	if err != nil {
		t.Fatalf("Failed to create target organization: %v", err)
	}
	defer func() {
		_, _ = client.Collection("organizations").Doc(orgID).Delete(ctx)
	}()

	// 1. Create active autojoin rule
	rule := &db.AutojoinRule{
		Domain:         "google.com",
		OrganizationID: orgID,
		DefaultRoles:   []db.UserRole{db.RoleOrgEditor, db.RoleProjectCreator},
		IsActive:       true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	ruleID, err := autojoinStore.CreateRule(ctx, rule)
	if err != nil {
		t.Fatalf("Failed to create autojoin rule: %v", err)
	}
	defer func() {
		_, _ = client.Collection("autojoin_rules").Doc(ruleID).Delete(ctx)
	}()

	// 2. Test Case 1: Discover tenancy for new user with mapped domain
	discoveredOrgs, err := userStore.DiscoverUserOrganizations(ctx, "newuser@google.com")
	if err != nil {
		t.Fatalf("DiscoverUserOrganizations failed for mapped domain: %v", err)
	}
	if len(discoveredOrgs) != 1 || discoveredOrgs[0] != orgID {
		t.Errorf("Expected to discover target organization %s, got %v", orgID, discoveredOrgs)
	}

	// 3. Test Case 2: Reconcile / Auto-provision mapped user
	userID := "google-test-user-999"
	err = userStore.ReconcileAutojoinRules(ctx, "newuser@google.com", userID)
	if err != nil {
		t.Fatalf("ReconcileAutojoinRules failed: %v", err)
	}
	defer func() {
		// Clean up provisioned user
		_, _ = client.Collection("organizations").Doc(orgID).Collection("users").Doc(userID).Delete(ctx)
	}()

	// Verify user doc in Firestore
	userDocRef := client.Collection("organizations").Doc(orgID).Collection("users").Doc(userID)
	doc, err := userDocRef.Get(ctx)
	if err != nil {
		t.Fatalf("Failed to retrieve auto-provisioned user document: %v", err)
	}

	var user db.User
	if err := doc.DataTo(&user); err != nil {
		t.Fatalf("Failed to decode user document: %v", err)
	}

	if user.UserID != userID {
		t.Errorf("Expected UserID %s, got %s", userID, user.UserID)
	}
	if user.Email != "newuser@google.com" {
		t.Errorf("Expected Email newuser@google.com, got %s", user.Email)
	}
	if user.Role != db.RoleOrgEditor {
		t.Errorf("Expected primary Role %s, got %s", db.RoleOrgEditor, user.Role)
	}
	if len(user.Roles) != 2 || user.Roles[1] != db.RoleProjectCreator {
		t.Errorf("Expected 2 roles [org:editor, project:creator], got %v", user.Roles)
	}

	// 4. Test Case 3: Discover tenancy for user with UNMAPPED domain
	unmappedOrgs, err := userStore.DiscoverUserOrganizations(ctx, "newuser@unmapped.com")
	if err != nil {
		t.Fatalf("DiscoverUserOrganizations failed for unmapped domain: %v", err)
	}
	if len(unmappedOrgs) != 0 {
		t.Errorf("Expected 0 discovered organizations for unmapped domain, got %v", unmappedOrgs)
	}

	// 5. Test Case 4: Reconcile user with UNMAPPED domain (should not provision)
	unmappedUserID := "unmapped-user-777"
	err = userStore.ReconcileAutojoinRules(ctx, "newuser@unmapped.com", unmappedUserID)
	if err != nil {
		t.Fatalf("ReconcileAutojoinRules failed for unmapped user: %v", err)
	}

	// Verify no user doc exists
	unmappedUserDocRef := client.Collection("organizations").Doc(orgID).Collection("users").Doc(unmappedUserID)
	_, err = unmappedUserDocRef.Get(ctx)
	if err == nil {
		t.Errorf("Expected unmapped user to NOT be provisioned, but document was found")
		_, _ = unmappedUserDocRef.Delete(ctx) // cleanup just in case
	}
}
