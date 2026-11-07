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

	"cloud.google.com/go/firestore"
	"google.com/apf/api/core"
	"google.com/apf/api/db"
)

func TestSeedLocal(t *testing.T) {
	emulatorHost := os.Getenv("FIRESTORE_EMULATOR_HOST")
	if emulatorHost == "" {
		t.Skip("Skipping Firestore integration test: FIRESTORE_EMULATOR_HOST is not set")
	}

	ctx := context.WithValue(context.Background(), core.OrganizationIDKey, "default-organization")

	// Connect to emulator using a distinct mock project ID to isolate the test database
	client, err := firestore.NewClient(ctx, "agenticproductfoundry-seed-test")
	if err != nil {
		t.Fatalf("Failed to connect to Firestore emulator: %v", err)
	}
	defer client.Close()

	// Set the environment variables
	t.Setenv("SIMULATED_USER_ID", "custom-test-user-id")
	t.Setenv("DEV_REPO_ACCESS_TOKEN", "test-dev-repo-token")

	// Re-initialize package-level appConfig to pick up env change
	appConfig = core.NewConfig()

	// Run local seeding (this will init stores and run all seeding steps)
	initStores(client)
	local(ctx, client)

	// Verify that the credentials have been saved with the correct PAT for the simulated owner
	simulatedUsers := []string{
		"custom-test-user-id",
	}

	for _, uID := range simulatedUsers {
		cred, err := appCredentialStore.GetCredential(ctx, "default-organization", uID, "github")
		if err != nil {
			t.Errorf("Expected GitHub credential to be saved for user %s, got error: %v", uID, err)
			continue
		}
		if cred == nil {
			t.Errorf("Expected GitHub credential to be saved for user %s, got nil", uID)
			continue
		}
		patVal := db.GetFieldValue(cred, "pat")
		if patVal != "test-dev-repo-token" {
			t.Errorf("Expected PAT value %q, got %q for user %s", "test-dev-repo-token", patVal, uID)
		}

		user, err := userStore.GetUser(ctx, "default-organization", uID)
		if err != nil {
			t.Errorf("Expected user profile to be saved for user %s, got error: %v", uID, err)
		} else if user == nil || user.TourStatus == nil || !user.TourStatus.Completed {
			t.Errorf("Expected user %s tour status to be completed", uID)
		}
	}

	// Verify the seeded projects have fallback repository details pre-configured
	projects, err := projectStore.ListProjects(ctx, "default-organization")
	if err != nil {
		t.Fatalf("Failed to list projects: %v", err)
	}

	if len(projects) == 0 {
		t.Fatalf("Expected at least one seeded project, got 0")
	}

	foundProject := false
	for _, p := range projects {
		if p.Name != nil && *p.Name == "Foundry Showcase" {
			foundProject = true
			if p.BaseBranch == nil || *p.BaseBranch != "main" {
				t.Errorf("Expected BaseBranch 'main', got %v", p.BaseBranch)
			}
			if p.GetPlanningArtifactsLocation() != "docs/specs/plans" {
				t.Errorf("Expected PlanningArtifactsLocation 'docs/specs/plans', got %v", p.GetPlanningArtifactsLocation())
			}
			if p.GetSpecLocation() != "docs/specs" {
				t.Errorf("Expected SpecLocation 'docs/specs', got %v", p.GetSpecLocation())
			}
			if p.GetImplementationArtifactsLocation() != "docs/specs/implementation" {
				t.Errorf("Expected ImplementationArtifactsLocation 'docs/specs/implementation', got %v", p.GetImplementationArtifactsLocation())
			}
		}
	}
	if !foundProject {
		t.Errorf("Expected to find 'Foundry Showcase' in seeded projects list")
	}

	// Verify feature flags seeding
	projectListFeat, err := featureStore.GetFeature(ctx, "project_list")
	if err != nil {
		t.Errorf("Expected 'project_list' feature flag to be seeded, got error: %v", err)
	} else if val, ok := projectListFeat.Properties["enable_share_projects"].(bool); !ok || val != false {
		t.Errorf("Expected 'enable_share_projects' property to be false, got: %v", projectListFeat.Properties["enable_share_projects"])
	}

	workflowsFeat, err := featureStore.GetFeature(ctx, "workflows")
	if err != nil {
		t.Errorf("Expected 'workflows' feature flag to be seeded, got error: %v", err)
	} else if val, ok := workflowsFeat.Properties["enable_workflow_run"].(bool); !ok || val != false {
		t.Errorf("Expected 'enable_workflow_run' property to be false, got: %v", workflowsFeat.Properties["enable_workflow_run"])
	}

	appBannersFeat, err := featureStore.GetFeature(ctx, "app_banners")
	if err != nil {
		t.Errorf("Expected 'app_banners' feature flag to be seeded, got error: %v", err)
	} else {
		if val, ok := appBannersFeat.Properties["show_alpha_notice"].(bool); !ok || val != true {
			t.Errorf("Expected 'show_alpha_notice' property to be true, got: %v", appBannersFeat.Properties["show_alpha_notice"])
		}
		if val, ok := appBannersFeat.Properties["show_version"].(bool); !ok || val != true {
			t.Errorf("Expected 'show_version' property to be true, got: %v", appBannersFeat.Properties["show_version"])
		}
	}
}
