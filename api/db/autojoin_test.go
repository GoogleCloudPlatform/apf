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
	"os"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
)

func TestFirestoreAutojoinStore_CRUD(t *testing.T) {
	emulatorHost := os.Getenv("FIRESTORE_EMULATOR_HOST")
	if emulatorHost == "" {
		t.Skip("Skipping Firestore integration test: FIRESTORE_EMULATOR_HOST is not set")
	}

	ctx := context.Background()
	client, err := firestore.NewClient(ctx, "agenticproductfoundry-489717")
	if err != nil {
		t.Fatalf("Failed to create Firestore client: %v", err)
	}
	defer client.Close()

	store := NewFirestoreAutojoinStore(client)

	// 1. Create a rule
	rule := &AutojoinRule{
		Domain:         "google.com",
		OrganizationID: "google-cloud-org",
		DefaultRoles:   []UserRole{RoleOrgEditor, RoleProjectCreator},
		IsActive:       true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	id, err := store.CreateRule(ctx, rule)
	if err != nil {
		t.Fatalf("CreateRule failed: %v", err)
	}
	if id == "" {
		t.Fatal("Expected non-empty rule ID")
	}

	defer func() {
		// Clean up rule document
		_, _ = client.Collection("autojoin_rules").Doc(id).Delete(ctx)
	}()

	// 2. Retrieve the rule
	fetched, err := store.GetRule(ctx, id)
	if err != nil {
		t.Fatalf("GetRule failed: %v", err)
	}
	if fetched.Domain != rule.Domain {
		t.Errorf("Expected domain %s, got %s", rule.Domain, fetched.Domain)
	}
	if len(fetched.DefaultRoles) != 2 || fetched.DefaultRoles[0] != RoleOrgEditor {
		t.Errorf("Expected 2 default roles with primary role OrgEditor, got %v", fetched.DefaultRoles)
	}

	// 3. List rules
	rules, err := store.ListRules(ctx)
	if err != nil {
		t.Fatalf("ListRules failed: %v", err)
	}
	found := false
	for _, r := range rules {
		if r.ID == id {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected to find rule %s in listed rules", id)
	}

	// 4. Update the rule
	fetched.DefaultRoles = []UserRole{RoleOrgViewer}
	err = store.UpdateRule(ctx, id, fetched)
	if err != nil {
		t.Fatalf("UpdateRule failed: %v", err)
	}

	updated, err := store.GetRule(ctx, id)
	if err != nil {
		t.Fatalf("GetRule after update failed: %v", err)
	}
	if len(updated.DefaultRoles) != 1 || updated.DefaultRoles[0] != RoleOrgViewer {
		t.Errorf("Expected updated default role OrgViewer, got %v", updated.DefaultRoles)
	}

	// 5. Query active rule by domain
	activeRule, err := store.FindActiveRuleByDomain(ctx, "google.com")
	if err != nil {
		t.Fatalf("FindActiveRuleByDomain failed: %v", err)
	}
	if activeRule.ID != id {
		t.Errorf("Expected active rule ID %s, got %s", id, activeRule.ID)
	}

	// 6. Delete the rule
	err = store.DeleteRule(ctx, id)
	if err != nil {
		t.Fatalf("DeleteRule failed: %v", err)
	}

	_, err = store.GetRule(ctx, id)
	if err == nil {
		t.Fatal("Expected GetRule to fail after deletion, but it succeeded")
	}
}
