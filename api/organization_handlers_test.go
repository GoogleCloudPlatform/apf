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
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"google.com/apf/api/core"
	"google.com/apf/api/db"
)

type mockOrganizationStore struct {
	db.OrganizationStore
	organizations map[string]*db.Organization
}

func (m *mockOrganizationStore) GetOrganization(ctx context.Context, id string) (*db.Organization, error) {
	if m.organizations == nil {
		return nil, db.ErrNotFound
	}
	org, ok := m.organizations[id]
	if !ok {
		return nil, db.ErrNotFound
	}
	return org, nil
}

func (m *mockOrganizationStore) UpdateOrganizationConfig(ctx context.Context, id string, config *db.OrganizationConfig) error {
	if m.organizations == nil {
		return db.ErrNotFound
	}
	org, ok := m.organizations[id]
	if !ok {
		return db.ErrNotFound
	}
	org.Config = config
	return nil
}


func (m *mockOrganizationStore) GetOrganizationByName(ctx context.Context, name string) (*db.Organization, error) {
	if m.organizations == nil {
		return nil, db.ErrNotFound
	}
	for _, org := range m.organizations {
		if strings.EqualFold(org.Name, name) {
			return org, nil
		}
	}
	return nil, db.ErrNotFound
}

func (m *mockOrganizationStore) CreateOrganization(ctx context.Context, org *db.Organization) (string, error) {
	if m.organizations == nil {
		m.organizations = make(map[string]*db.Organization)
	}
	for _, existing := range m.organizations {
		if strings.EqualFold(existing.Name, org.Name) {
			return "", db.ErrConflict
		}
	}
	id := fmt.Sprintf("org-%d", len(m.organizations)+1)
	org.ID = id
	m.organizations[id] = org
	return id, nil
}

func (m *mockOrganizationStore) CreateOrganizationWithID(ctx context.Context, id string, org *db.Organization) error {
	if m.organizations == nil {
		m.organizations = make(map[string]*db.Organization)
	}
	for _, existing := range m.organizations {
		if strings.EqualFold(existing.Name, org.Name) {
			return db.ErrConflict
		}
	}
	org.ID = id
	m.organizations[id] = org
	return nil
}

func (m *mockOrganizationStore) DeleteOrganization(ctx context.Context, id string) error {
	if m.organizations == nil {
		return db.ErrNotFound
	}
	if _, ok := m.organizations[id]; !ok {
		return db.ErrNotFound
	}
	delete(m.organizations, id)
	return nil
}

func (m *mockOrganizationStore) UpdateOrganizationName(ctx context.Context, id string, name string) error {
	if m.organizations == nil {
		return db.ErrNotFound
	}
	org, ok := m.organizations[id]
	if !ok {
		return db.ErrNotFound
	}
	org.Name = name
	return nil
}

func TestCreateOrganizationSuccess(t *testing.T) {
	oldStore := db.OrganizationStoreInstance
	oldUserStore := userStore
	t.Cleanup(func() {
		db.OrganizationStoreInstance = oldStore
		userStore = oldUserStore
	})

	mockStore := &mockOrganizationStore{
		organizations: make(map[string]*db.Organization),
	}
	db.OrganizationStoreInstance = mockStore

	mockUStore := &mockUserStore{
		users: make(map[string]*db.User),
	}
	userStore = mockUStore

	reqPayload := CreateOrganizationRequest{
		Name:       "New Enterprise Org",
		OwnerEmail: "owner@enterprise.com",
	}
	body, _ := json.Marshal(reqPayload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/organizations", bytes.NewBuffer(body))
	claims := &core.UserClaims{
		Subject:      "super-admin-user",
		IsSuperAdmin: true,
	}
	req = withTestCtx(req, claims)

	w := httptest.NewRecorder()
	createOrganizationHandler(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected status 201 Created, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp CreateOrganizationResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.ID == "" {
		t.Error("Expected returned organization ID to not be empty")
	}
	if resp.Name != "New Enterprise Org" {
		t.Errorf("Expected organization name %q, got %q", "New Enterprise Org", resp.Name)
	}
	if resp.OwnerUID == "" {
		t.Error("Expected returned owner UID to not be empty")
	}

	// Verify organization was stored
	stored, err := mockStore.GetOrganizationByName(context.Background(), "New Enterprise Org")
	if err != nil {
		t.Fatalf("Expected organization to be saved, got error: %v", err)
	}
	if stored.ID != resp.ID {
		t.Errorf("Saved organization ID %q doesn't match response ID %q", stored.ID, resp.ID)
	}

	// Verify owner was created in database
	owner, err := mockUStore.GetUser(context.Background(), resp.ID, resp.OwnerUID)
	if err != nil {
		t.Fatalf("Expected owner user to be created in DB, got error: %v", err)
	}
	if owner.Email != "owner@enterprise.com" {
		t.Errorf("Expected owner email %q, got %q", "owner@enterprise.com", owner.Email)
	}
	if owner.Role != db.RoleOrgOwner {
		t.Errorf("Expected owner role %q, got %q", db.RoleOrgOwner, owner.Role)
	}
}

func TestCreateOrganizationNotSuperAdmin(t *testing.T) {
	oldStore := db.OrganizationStoreInstance
	t.Cleanup(func() { db.OrganizationStoreInstance = oldStore })

	mockStore := &mockOrganizationStore{
		organizations: make(map[string]*db.Organization),
	}
	db.OrganizationStoreInstance = mockStore

	reqPayload := CreateOrganizationRequest{
		Name:       "Forbidden Org",
		OwnerEmail: "owner@forbidden.com",
	}
	body, _ := json.Marshal(reqPayload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/organizations", bytes.NewBuffer(body))
	claims := &core.UserClaims{
		Subject:      "regular-user",
		IsSuperAdmin: false, // Not super admin
	}
	req = withTestCtx(req, claims)

	w := httptest.NewRecorder()
	createOrganizationHandler(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("Expected status 403 Forbidden, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestCreateOrganizationEmptyName(t *testing.T) {
	oldStore := db.OrganizationStoreInstance
	t.Cleanup(func() { db.OrganizationStoreInstance = oldStore })

	mockStore := &mockOrganizationStore{
		organizations: make(map[string]*db.Organization),
	}
	db.OrganizationStoreInstance = mockStore

	reqPayload := CreateOrganizationRequest{
		Name:       "",
		OwnerEmail: "owner@emptyname.com",
	}
	body, _ := json.Marshal(reqPayload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/organizations", bytes.NewBuffer(body))
	claims := &core.UserClaims{
		Subject:      "super-admin-user",
		IsSuperAdmin: true,
	}
	req = withTestCtx(req, claims)

	w := httptest.NewRecorder()
	createOrganizationHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected status 400 Bad Request, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestCreateOrganizationDuplicateName(t *testing.T) {
	oldStore := db.OrganizationStoreInstance
	t.Cleanup(func() { db.OrganizationStoreInstance = oldStore })

	existingOrg := &db.Organization{
		ID:   "org-1",
		Name: "Existing Org",
	}
	mockStore := &mockOrganizationStore{
		organizations: map[string]*db.Organization{
			"org-1": existingOrg,
		},
	}
	db.OrganizationStoreInstance = mockStore

	reqPayload := CreateOrganizationRequest{
		Name:       "Existing Org",
		OwnerEmail: "owner@duplicate.com",
	}
	body, _ := json.Marshal(reqPayload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/organizations", bytes.NewBuffer(body))
	claims := &core.UserClaims{
		Subject:      "super-admin-user",
		IsSuperAdmin: true,
	}
	req = withTestCtx(req, claims)

	w := httptest.NewRecorder()
	createOrganizationHandler(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("Expected status 409 Conflict, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestCreateOrganizationNoAuth(t *testing.T) {
	reqPayload := CreateOrganizationRequest{
		Name:       "Unauthorized Org",
		OwnerEmail: "owner@noauth.com",
	}
	body, _ := json.Marshal(reqPayload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/organizations", bytes.NewBuffer(body))
	// No claims in context

	w := httptest.NewRecorder()
	createOrganizationHandler(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("Expected status 401 Unauthorized, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestCreateOrganization_Integration(t *testing.T) {
	emulatorHost := os.Getenv("FIRESTORE_EMULATOR_HOST")
	if emulatorHost == "" {
		t.Skip("Skipping Firestore integration test: FIRESTORE_EMULATOR_HOST is not set")
	}

	oldStore := db.OrganizationStoreInstance
	oldUserStore := userStore
	t.Cleanup(func() {
		db.OrganizationStoreInstance = oldStore
		userStore = oldUserStore
	})

	ctx := context.Background()
	// Connect to the emulator using a mock project ID
	client, err := firestore.NewClient(ctx, "agenticproductfoundry-489717")
	if err != nil {
		t.Fatalf("Failed to connect to Firestore emulator: %v", err)
	}
	t.Cleanup(func() { client.Close() })

	// Initialize the real stores backed by emulator client
	realStore := db.NewFirestoreOrganizationStore(client)
	db.OrganizationStoreInstance = realStore
	userStore = db.NewFirestoreUserStore(client, nil)

	uniqueName := fmt.Sprintf("Integration Org %d", time.Now().UnixNano())
	reqPayload := CreateOrganizationRequest{
		Name:       uniqueName,
		OwnerEmail: "owner@integration.com",
	}
	body, _ := json.Marshal(reqPayload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/organizations", bytes.NewBuffer(body))
	claims := &core.UserClaims{
		Subject:      "super-admin-user",
		IsSuperAdmin: true,
	}
	req = withTestCtx(req, claims)

	w := httptest.NewRecorder()
	createOrganizationHandler(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected status 201 Created, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp CreateOrganizationResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.ID == "" {
		t.Fatal("Expected returned organization ID to not be empty")
	}
	if resp.OwnerUID == "" {
		t.Fatal("Expected returned owner UID to not be empty")
	}

	// Register cleanup for integration test docs
	t.Cleanup(func() {
		_, _ = client.Collection("organizations").Doc(resp.ID).Collection("users").Doc(resp.OwnerUID).Delete(ctx)
		_, _ = client.Collection("organizations").Doc(resp.ID).Delete(ctx)
	})

	if resp.Name != uniqueName {
		t.Errorf("Expected organization name %q, got %q", uniqueName, resp.Name)
	}

	// Verify that the organization document was actually created in firestore emulator
	docRef := client.Collection("organizations").Doc(resp.ID)
	docSnap, err := docRef.Get(ctx)
	if err != nil {
		t.Fatalf("Failed to retrieve created organization document: %v", err)
	}

	var storedOrg db.Organization
	if err := docSnap.DataTo(&storedOrg); err != nil {
		t.Fatalf("Failed to map document data: %v", err)
	}

	if storedOrg.Name != uniqueName {
		t.Errorf("Expected stored organization name %q, got %q", uniqueName, storedOrg.Name)
	}

	expectedTenantID := "mock-org-" + strings.ToLower(strings.ReplaceAll(uniqueName, " ", "-"))
	if storedOrg.TenantID != expectedTenantID {
		t.Errorf("Expected stored tenant ID %q, got %q", expectedTenantID, storedOrg.TenantID)
	}

	// Verify that the owner user document was actually created in firestore emulator
	userRef := client.Collection("organizations").Doc(resp.ID).Collection("users").Doc(resp.OwnerUID)
	userSnap, err := userRef.Get(ctx)
	if err != nil {
		t.Fatalf("Failed to retrieve created owner user document: %v", err)
	}

	var storedUser db.User
	if err := userSnap.DataTo(&storedUser); err != nil {
		t.Fatalf("Failed to map user document data: %v", err)
	}

	if storedUser.Email != "owner@integration.com" {
		t.Errorf("Expected stored owner email %q, got %q", "owner@integration.com", storedUser.Email)
	}
	if storedUser.Role != db.RoleOrgOwner {
		t.Errorf("Expected stored owner role %q, got %q", db.RoleOrgOwner, storedUser.Role)
	}
}

func TestCreateOrganizationInvalidJSON(t *testing.T) {
	oldStore := db.OrganizationStoreInstance
	t.Cleanup(func() { db.OrganizationStoreInstance = oldStore })

	mockStore := &mockOrganizationStore{
		organizations: make(map[string]*db.Organization),
	}
	db.OrganizationStoreInstance = mockStore

	invalidBody := []byte(`{invalid json`)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/organizations", bytes.NewBuffer(invalidBody))
	claims := &core.UserClaims{
		Subject:      "super-admin-user",
		IsSuperAdmin: true,
	}
	req = withTestCtx(req, claims)

	w := httptest.NewRecorder()
	createOrganizationHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected status 400 Bad Request, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestCreateOrganizationCaseInsensitiveDuplicate(t *testing.T) {
	oldStore := db.OrganizationStoreInstance
	t.Cleanup(func() { db.OrganizationStoreInstance = oldStore })

	existingOrg := &db.Organization{
		ID:   "org-1",
		Name: "Existing Org",
	}
	mockStore := &mockOrganizationStore{
		organizations: map[string]*db.Organization{
			"org-1": existingOrg,
		},
	}
	db.OrganizationStoreInstance = mockStore

	// Request with different casing
	reqPayload := CreateOrganizationRequest{
		Name:       "existing org",
		OwnerEmail: "owner@duplicate.com",
	}
	body, _ := json.Marshal(reqPayload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/organizations", bytes.NewBuffer(body))
	claims := &core.UserClaims{
		Subject:      "super-admin-user",
		IsSuperAdmin: true,
	}
	req = withTestCtx(req, claims)

	w := httptest.NewRecorder()
	createOrganizationHandler(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("Expected status 409 Conflict, got %d. Body: %s", w.Code, w.Body.String())
	}
}


