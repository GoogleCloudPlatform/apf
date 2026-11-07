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
	"os"
	"testing"

	"cloud.google.com/go/firestore"
)

func TestUserStore_GetOrCreateUser(t *testing.T) {
	emulatorHost := os.Getenv("FIRESTORE_EMULATOR_HOST")
	if emulatorHost == "" {
		t.Skip("Skipping Firestore integration test: FIRESTORE_EMULATOR_HOST is not set")
	}

	ctx := testCtx()
	client, err := firestore.NewClient(ctx, "agenticproductfoundry-489717")
	if err != nil {
		t.Fatalf("Failed to create Firestore client: %v", err)
	}
	defer client.Close()

	userStore := NewFirestoreUserStore(client, nil)
	testUserID := "test-user-pref-uuid"

	// Ensure clean slate by deleting test doc first
	_, _ = organizationCollectionRef(client, "users").Doc(testUserID).Delete(ctx)

	// 1. GetOrCreateUser (Creates default profile)
	user, err := userStore.GetOrCreateUser(ctx, "test-organization", testUserID)
	if err != nil {
		t.Fatalf("GetOrCreateUser failed to initialize: %v", err)
	}
	if user.ID != testUserID {
		t.Errorf("Expected user ID to be %s, got %s", testUserID, user.ID)
	}
	if user.Preferences.Theme != "dark" {
		t.Errorf("Expected default theme to be 'dark', got %s", user.Preferences.Theme)
	}

	// 2. GetOrCreateUser (Retrieves existing profile)
	user2, err := userStore.GetOrCreateUser(ctx, "test-organization", testUserID)
	if err != nil {
		t.Fatalf("GetOrCreateUser failed to retrieve existing: %v", err)
	}
	if user2.Preferences.Theme != "dark" {
		t.Errorf("Expected retrieved theme to be 'dark', got %s", user2.Preferences.Theme)
	}

	// Clean up
	_, _ = organizationCollectionRef(client, "users").Doc(testUserID).Delete(ctx)
}

func TestUserStore_UpdatePreferences(t *testing.T) {
	emulatorHost := os.Getenv("FIRESTORE_EMULATOR_HOST")
	if emulatorHost == "" {
		t.Skip("Skipping Firestore integration test: FIRESTORE_EMULATOR_HOST is not set")
	}

	ctx := testCtx()
	client, err := firestore.NewClient(ctx, "agenticproductfoundry-489717")
	if err != nil {
		t.Fatalf("Failed to create Firestore client: %v", err)
	}
	defer client.Close()

	userStore := NewFirestoreUserStore(client, nil)
	testUserID := "test-user-pref-update-uuid"

	// Ensure base profile exists
	_, err = userStore.GetOrCreateUser(ctx, "test-organization", testUserID)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// Update to light theme
	newPrefs := &UserPreferences{Theme: "light"}
	err = userStore.UpdateUserPreferences(ctx, "test-organization", testUserID, newPrefs)
	if err != nil {
		t.Fatalf("UpdateUserPreferences failed: %v", err)
	}

	// Verify update
	user, err := userStore.GetOrCreateUser(ctx, "test-organization", testUserID)
	if err != nil {
		t.Fatalf("GetOrCreateUser lookup failed: %v", err)
	}
	if user.Preferences.Theme != "light" {
		t.Errorf("Expected updated theme to be 'light', got %s", user.Preferences.Theme)
	}

	// Clean up
	_, _ = organizationCollectionRef(client, "users").Doc(testUserID).Delete(ctx)
}

func TestAppCredentialStore_EncryptionLoop(t *testing.T) {
	emulatorHost := os.Getenv("FIRESTORE_EMULATOR_HOST")
	if emulatorHost == "" {
		t.Skip("Skipping Firestore integration test: FIRESTORE_EMULATOR_HOST is not set")
	}

	ctx := testCtx()
	client, err := firestore.NewClient(ctx, "agenticproductfoundry-489717")
	if err != nil {
		t.Fatalf("Failed to create Firestore client: %v", err)
	}
	defer client.Close()

	credStore := NewFirestoreAppCredentialStore(client)
	testUserID := "test-user-cred-enc-uuid"
	appId := "figma"

	// Pre-cleanup
	_, _ = organizationCollectionRef(client, "users").Doc(testUserID).Collection("app_credentials").Doc(appId).Delete(ctx)

	// Save plain values
	cred := &AppCredential{
		IntegrationID: appId,
		Fields: []CredentialAttribute{
			{Name: "username", DisplayName: "Username", Value: "figma-username-test", IsSecret: false},
			{Name: "password", DisplayName: "Password", Value: "my-super-secret-password-123", IsSecret: true},
			{Name: "accessToken", DisplayName: "Access Token", Value: "my-secret-access-token-abc", IsSecret: true},
			{Name: "refreshToken", DisplayName: "Refresh Token", Value: "my-secret-refresh-token-xyz", IsSecret: true},
		},
	}

	err = credStore.SaveCredential(ctx, "test-organization", testUserID, appId, cred)
	if err != nil {
		t.Fatalf("SaveCredential failed: %v", err)
	}

	// 1. Verify Raw DB fields are Encrypted (we do not read via transparent store)
	doc, err := organizationCollectionRef(client, "users").Doc(testUserID).Collection("app_credentials").Doc(appId).Get(ctx)
	if err != nil {
		t.Fatalf("Failed to load raw doc: %v", err)
	}
	var rawData map[string]interface{}
	_ = doc.DataTo(&rawData)

	fieldsArray := rawData["fields"].([]interface{})
	var rawPassword, rawAccessToken, rawRefreshToken string
	for _, f := range fieldsArray {
		fieldMap := f.(map[string]interface{})
		name := fieldMap["name"].(string)
		val := fieldMap["value"].(string)
		switch name {
		case "password":
			rawPassword = val
		case "accessToken":
			rawAccessToken = val
		case "refreshToken":
			rawRefreshToken = val
		}
	}

	if rawPassword == "my-super-secret-password-123" {
		t.Error("Expected password to be encrypted, but it was stored in plaintext")
	}
	if rawAccessToken == "my-secret-access-token-abc" {
		t.Error("Expected accessToken to be encrypted, but it was stored in plaintext")
	}
	if rawRefreshToken == "my-secret-refresh-token-xyz" {
		t.Error("Expected refreshToken to be encrypted, but it was stored in plaintext")
	}

	// 2. Verify Transparent Decryption on Get
	fetched, err := credStore.GetCredential(ctx, "test-organization", testUserID, appId)
	if err != nil {
		t.Fatalf("GetCredential failed: %v", err)
	}

	var usernameVal, passwordVal, tokenVal, refreshVal string
	for _, field := range fetched.Fields {
		switch field.Name {
		case "username":
			usernameVal = field.Value
		case "password":
			passwordVal = field.Value
		case "accessToken":
			tokenVal = field.Value
		case "refreshToken":
			refreshVal = field.Value
		}
	}

	if usernameVal != "figma-username-test" {
		t.Errorf("Expected username to be figma-username-test, got %s", usernameVal)
	}
	if passwordVal != "my-super-secret-password-123" {
		t.Errorf("Expected decrypted password to match original, got %s", passwordVal)
	}
	if tokenVal != "my-secret-access-token-abc" {
		t.Errorf("Expected decrypted accessToken to match, got %s", tokenVal)
	}
	if refreshVal != "my-secret-refresh-token-xyz" {
		t.Errorf("Expected decrypted refreshToken to match, got %s", refreshVal)
	}

	// Clean up
	_ = credStore.DeleteCredential(ctx, "test-organization", testUserID, appId)
}

func TestUserProjectStore_SaveAndGet(t *testing.T) {
	emulatorHost := os.Getenv("FIRESTORE_EMULATOR_HOST")
	if emulatorHost == "" {
		t.Skip("Skipping Firestore integration test: FIRESTORE_EMULATOR_HOST is not set")
	}

	ctx := testCtx()
	client, err := firestore.NewClient(ctx, "agenticproductfoundry-489717")
	if err != nil {
		t.Fatalf("Failed to create Firestore client: %v", err)
	}
	defer client.Close()

	upStore := NewFirestoreUserProjectStore(client)
	testUserID := "test-user-project-uuid"
	projectId := "test-project-id"
	appId := "figma"

	parentDocID := "test-user-project-uuid_test-project-id"
	_, _ = organizationCollectionRef(client, "user_project").Doc(parentDocID).Collection("app_credentials").Doc(appId).Delete(ctx)
	_, _ = organizationCollectionRef(client, "user_project").Doc(parentDocID).Delete(ctx)

	// 1. Get default settings (not enabled by default)
	setting, err := upStore.GetProjectAppSetting(ctx, "test-organization", testUserID, projectId, appId)
	if err != nil {
		t.Fatalf("GetProjectAppSetting error: %v", err)
	}
	if setting.Enabled != false {
		t.Errorf("Expected default Enabled state to be false, got true")
	}

	// 2. Save settings
	newSetting := &UserProject{
		UserID:               testUserID,
		ProjectID:            projectId,
		AppID:                appId,
		Enabled:              true,
		CredentialID:         "figma-global-credential-ref",
		IsReadOnly:           true,
		CanSubmitPullRequest: true,
		CanWrite:             true,
	}
	err = upStore.SaveProjectAppSetting(ctx, "test-organization", newSetting)
	if err != nil {
		t.Fatalf("SaveProjectAppSetting failed: %v", err)
	}

	// 3. Verify lookup
	fetched, err := upStore.GetProjectAppSetting(ctx, "test-organization", testUserID, projectId, appId)
	if err != nil {
		t.Fatalf("GetProjectAppSetting error: %v", err)
	}
	if fetched.Enabled != true {
		t.Errorf("Expected Enabled to be true, got false")
	}
	if fetched.CredentialID != "figma-global-credential-ref" {
		t.Errorf("Expected CredentialID to be 'figma-global-credential-ref', got %s", fetched.CredentialID)
	}
	if fetched.IsReadOnly != true {
		t.Errorf("Expected IsReadOnly to be true, got false")
	}
	if fetched.CanSubmitPullRequest != true {
		t.Errorf("Expected CanSubmitPullRequest to be true, got false")
	}
	if fetched.CanWrite != true {
		t.Errorf("Expected CanWrite to be true, got false")
	}

	// Clean up
	_, _ = organizationCollectionRef(client, "user_project").Doc(parentDocID).Collection("app_credentials").Doc(appId).Delete(ctx)
	_, _ = organizationCollectionRef(client, "user_project").Doc(parentDocID).Delete(ctx)
}

func TestUserStore_UpdateTourStatus(t *testing.T) {
	emulatorHost := os.Getenv("FIRESTORE_EMULATOR_HOST")
	if emulatorHost == "" {
		t.Skip("Skipping Firestore integration test: FIRESTORE_EMULATOR_HOST is not set")
	}

	ctx := testCtx()
	client, err := firestore.NewClient(ctx, "agenticproductfoundry-489717")
	if err != nil {
		t.Fatalf("Failed to create Firestore client: %v", err)
	}
	defer client.Close()

	userStore := NewFirestoreUserStore(client, nil)
	testUserID := "test-user-tour-update-uuid"

	// Ensure base profile exists
	_, err = userStore.GetOrCreateUser(ctx, "test-organization", testUserID)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// Update tour status
	newStatus := &TourStatus{
		Completed:         false,
		CompletedSections: []string{"catalog"},
		LastActiveStep: LastActiveStep{
			SectionID: "workspace",
			StepIndex: 2,
		},
	}
	err = userStore.UpdateTourStatus(ctx, "test-organization", testUserID, newStatus)
	if err != nil {
		t.Fatalf("UpdateTourStatus failed: %v", err)
	}

	// Verify update
	user, err := userStore.GetOrCreateUser(ctx, "test-organization", testUserID)
	if err != nil {
		t.Fatalf("GetOrCreateUser lookup failed: %v", err)
	}
	if user.TourStatus == nil {
		t.Fatal("Expected TourStatus to be populated, got nil")
	}
	if user.TourStatus.Completed != false {
		t.Errorf("Expected Completed to be false, got %t", user.TourStatus.Completed)
	}
	if len(user.TourStatus.CompletedSections) != 1 || user.TourStatus.CompletedSections[0] != "catalog" {
		t.Errorf("Expected CompletedSections to contain 'catalog', got %v", user.TourStatus.CompletedSections)
	}
	if user.TourStatus.LastActiveStep.SectionID != "workspace" || user.TourStatus.LastActiveStep.StepIndex != 2 {
		t.Errorf("Expected LastActiveStep to be workspace:2, got %s:%d", user.TourStatus.LastActiveStep.SectionID, user.TourStatus.LastActiveStep.StepIndex)
	}

	// Clean up
	_, _ = organizationCollectionRef(client, "users").Doc(testUserID).Delete(ctx)
}


func TestUserStore_UpdateUserRole(t *testing.T) {
	emulatorHost := os.Getenv("FIRESTORE_EMULATOR_HOST")
	if emulatorHost == "" {
		t.Skip("Skipping Firestore integration test: FIRESTORE_EMULATOR_HOST is not set")
	}

	ctx := testCtx()
	client, err := firestore.NewClient(ctx, "agenticproductfoundry-489717")
	if err != nil {
		t.Fatalf("Failed to create Firestore client: %v", err)
	}
	defer client.Close()

	userStore := NewFirestoreUserStore(client, nil)
	orgID := "test-org-role-update"
	userID := "test-user-role-update-uid"

	// Setup user
	col := organizationCollection(orgID, client, "users")
	_, err = col.Doc(userID).Set(ctx, &User{
		ID:     userID,
		UserID: userID,
		Role:   RoleOrgViewer,
	})
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// Update role
	err = userStore.UpdateUserRole(ctx, orgID, userID, RoleOrgAdmin)
	if err != nil {
		t.Fatalf("UpdateUserRole failed: %v", err)
	}

	// Verify
	doc, err := col.Doc(userID).Get(ctx)
	if err != nil {
		t.Fatalf("Failed to get user doc: %v", err)
	}

	var user User
	if err := doc.DataTo(&user); err != nil {
		t.Fatalf("Failed to parse user doc: %v", err)
	}

	if user.Role != RoleOrgAdmin {
		t.Errorf("Expected role 'admin', got %s", user.Role)
	}

	// Clean up
	_, _ = col.Doc(userID).Delete(ctx)
}




