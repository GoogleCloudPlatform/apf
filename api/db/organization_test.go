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

func strPtr(s string) *string { return &s }
func boolPtr(b bool) *bool     { return &b }

func TestOrganizationStore_UpdateOrganizationConfig(t *testing.T) {
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

	orgStore := NewFirestoreOrganizationStore(client)

	// Create a temp organization to test config updates
	org := &Organization{
		Name: "Test Config Org",
	}
	orgID, err := orgStore.CreateOrganization(ctx, org)
	if err != nil {
		t.Fatalf("Failed to create organization: %v", err)
	}
	defer func() {
		// Clean up organization doc
		_, _ = client.Collection("organizations").Doc(orgID).Delete(ctx)
	}()

	config := &OrganizationConfig{
		Branding: &BrandingConfig{
			LogoURL: strPtr("https://example.com/logo.png"),
			Theme:   strPtr("dark"),
		},
		SSO: &SSOConfig{
			Enabled:     boolPtr(true),
			ProviderURL: strPtr("https://sso.example.com"),
		},
		EnvSettings: map[string]string{
			"VAR_A": "VALUE_A",
			"VAR_B": "VALUE_B",
		},
	}

	err = orgStore.UpdateOrganizationConfig(ctx, orgID, config)
	if err != nil {
		t.Fatalf("UpdateOrganizationConfig failed: %v", err)
	}

	// Verify
	fetched, err := orgStore.GetOrganization(ctx, orgID)
	if err != nil {
		t.Fatalf("GetOrganization failed: %v", err)
	}

	if fetched.Config == nil {
		t.Fatal("Expected Config to be populated, got nil")
	}
	if fetched.Config.Branding == nil || fetched.Config.Branding.LogoURL == nil || *fetched.Config.Branding.LogoURL != *config.Branding.LogoURL {
		t.Errorf("Expected LogoURL %v, got %v", config.Branding.LogoURL, fetched.Config.Branding.LogoURL)
	}
	if fetched.Config.Branding.Theme == nil || *fetched.Config.Branding.Theme != *config.Branding.Theme {
		t.Errorf("Expected Theme %v, got %v", config.Branding.Theme, fetched.Config.Branding.Theme)
	}
	if fetched.Config.SSO == nil || fetched.Config.SSO.Enabled == nil || *fetched.Config.SSO.Enabled != *config.SSO.Enabled {
		t.Errorf("Expected SSO Enabled %v, got %v", config.SSO.Enabled, fetched.Config.SSO.Enabled)
	}
	if fetched.Config.SSO.ProviderURL == nil || *fetched.Config.SSO.ProviderURL != *config.SSO.ProviderURL {
		t.Errorf("Expected SSO ProviderURL %v, got %v", config.SSO.ProviderURL, fetched.Config.SSO.ProviderURL)
	}
	if len(fetched.Config.EnvSettings) != 2 || fetched.Config.EnvSettings["VAR_A"] != "VALUE_A" {
		t.Errorf("Expected EnvSettings to match, got %v", fetched.Config.EnvSettings)
	}
}
