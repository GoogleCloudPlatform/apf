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
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"google.com/apf/api/core"
	"google.com/apf/api/db"
	"google.golang.org/api/iterator"
)

var (
	appConfig          *core.Configuration
	projectStore       db.ProjectStore
	workflowStore      db.WorkflowStore
	configStore        db.ConfigStore
	artifactStore      db.ArtifactStore
	appCredentialStore db.AppCredentialStore
	userProjectStore   db.UserProjectStore
	organizationStore  db.OrganizationStore
	userStore          db.UserStore
	featureStore       db.FeatureStore
	authClient         *auth.Client
)

func initStores(client *firestore.Client) {
	projectStore = db.NewFirestoreProjectStore(client)
	db.ProjectStoreInstance = projectStore
	workflowStore = db.NewFirestoreWorkflowStore(client)
	artifactStore = db.NewFirestoreArtifactStore(client)
	configStore = db.NewFirestoreConfigStore(client)
	appCredentialStore = db.NewFirestoreAppCredentialStore(client)
	db.AppCredentialStoreInstance = appCredentialStore
	userProjectStore = db.NewFirestoreUserProjectStore(client)
	db.UserProjectStoreInstance = userProjectStore
	organizationStore = db.NewFirestoreOrganizationStore(client)
	db.OrganizationStoreInstance = organizationStore
	featureStore = db.NewFirestoreFeatureStore(client)
	
	fUserStore := db.NewFirestoreUserStore(client, authClient)
	userStore = fUserStore
}

var envPrefixes = map[string]string{
	"local":   "apf-local",
	"staging": "apf-stg",
	"prod":    "apf-prod",
}

func main() {
	env := flag.String("env", "local", "Environment to seed: local, staging, prod")
	flag.Parse()

	ctx := context.Background()
	appConfig = core.NewConfig()

	if *env != "local" {
		os.Unsetenv("FIRESTORE_EMULATOR_HOST")
		os.Unsetenv("FIREBASE_AUTH_EMULATOR_HOST")
	} else {
		if os.Getenv("FIRESTORE_EMULATOR_HOST") == "" {
			os.Setenv("FIRESTORE_EMULATOR_HOST", "localhost:8084")
		}
		if os.Getenv("FIREBASE_AUTH_EMULATOR_HOST") == "" {
			os.Setenv("FIREBASE_AUTH_EMULATOR_HOST", "localhost:8085")
		}
		log.Printf("Connecting to Firebase Auth Emulator: %s", os.Getenv("FIREBASE_AUTH_EMULATOR_HOST"))
	}

	// Initialize Firebase Auth Client if possible
	app, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: appConfig.GoogleCloudProject})
	if err != nil {
		log.Printf("Warning: Failed to initialize Firebase App: %v. Running without Firebase Auth.", err)
	} else {
		var authErr error
		authClient, authErr = app.Auth(ctx)
		if authErr != nil {
			log.Printf("Warning: Failed to create Firebase Auth client: %v. Running without Firebase Auth.", authErr)
		} else {
			log.Println("Firebase Auth client initialized successfully")
		}
	}

	prefix, ok := envPrefixes[*env]
	if !ok {
		log.Fatalf("Invalid env %q. Supported envs: local, staging, prod", *env)
	}
	dbName := prefix + "-firestore"

	var client *firestore.Client

	if os.Getenv("FIRESTORE_EMULATOR_HOST") != "" {
		log.Printf("Connecting to Firestore Emulator: %s", os.Getenv("FIRESTORE_EMULATOR_HOST"))
	} else {
		log.Printf("Connecting to Firestore Project: %s", appConfig.GoogleCloudProject)
	}
	log.Printf("Using Database: %s", dbName)

	client, err = firestore.NewClientWithDatabase(ctx, appConfig.GoogleCloudProject, dbName)
	if err != nil {
		log.Fatalf("Failed to create Firestore client: %v", err)
	}
	defer client.Close()

	initStores(client)

	switch *env {
	case "local":
		if os.Getenv("FIRESTORE_EMULATOR_HOST") != "" {
			clearFirestoreEmulator(ctx, os.Getenv("FIRESTORE_EMULATOR_HOST"), appConfig.GoogleCloudProject, dbName)
		}

		var tenantID1, tenantID2 string
		var err error

		// Decoupled Firestore Org IDs
		orgID := "org-high-council"
		orgID2 := "org-sith-order"

		adminUID := "simulated-admin-uid"
		adminEmail := "admin@agentic-foundry.local"
		if appConfig != nil && appConfig.SimulatedUserID != "" {
			adminUID = appConfig.SimulatedUserID
			if strings.Contains(appConfig.SimulatedUserID, "@") {
				adminEmail = appConfig.SimulatedUserID
			}
		}

		usersToSeed := []SeedUser{
			{
				UID:         adminUID,
				Email:       adminEmail,
				Password:    "password",
				DisplayName: "Administrator",
				Role:        db.RoleOrgAdmin,
			},
			{
				UID:         "org-owner@the-high-council.com",
				Email:       "org-owner@the-high-council.com",
				Password:    "password",
				DisplayName: "Org Owner",
				Role:        db.RoleOrgOwner,
			},
			{
				UID:         "org-admin@the-high-council.com",
				Email:       "org-admin@the-high-council.com",
				Password:    "password",
				DisplayName: "Org Admin",
				Role:        db.RoleOrgAdmin,
			},
			{
				UID:         "org-editor@the-high-council.com",
				Email:       "org-editor@the-high-council.com",
				Password:    "password",
				DisplayName: "Org Editor",
				Role:        db.RoleOrgEditor,
			},
			{
				UID:         "org-viewer@the-high-council.com",
				Email:       "org-viewer@the-high-council.com",
				Password:    "password",
				DisplayName: "Org Viewer",
				Role:        db.RoleOrgViewer,
			},
			{
				UID:         "project-admin@the-high-council.com",
				Email:       "project-admin@the-high-council.com",
				Password:    "password",
				DisplayName: "Project Admin",
				Role:        db.RoleProjectAdmin,
			},
			{
				UID:         "project-editor@the-high-council.com",
				Email:       "project-editor@the-high-council.com",
				Password:    "password",
				DisplayName: "Project Editor",
				Role:        db.RoleProjectEditor,
			},
			{
				UID:         "project-viewer@the-high-council.com",
				Email:       "project-viewer@the-high-council.com",
				Password:    "password",
				DisplayName: "Project Viewer",
				Role:        db.RoleProjectViewer,
			},
			{
				UID:         "project-creator@the-high-council.com",
				Email:       "project-creator@the-high-council.com",
				Password:    "password",
				DisplayName: "Project Creator",
				Role:        db.RoleProjectCreator,
			},
			{
				UID:         "user-admin@the-high-council.com",
				Email:       "user-admin@the-high-council.com",
				Password:    "password",
				DisplayName: "User Admin",
				Role:        db.RoleUserAdmin,
			},
			{
				UID:         "user-editor@the-high-council.com",
				Email:       "user-editor@the-high-council.com",
				Password:    "password",
				DisplayName: "User Editor",
				Role:        db.RoleUserEditor,
			},
			{
				UID:         "user-viewer@the-high-council.com",
				Email:       "user-viewer@the-high-council.com",
				Password:    "password",
				DisplayName: "User Viewer",
				Role:        db.RoleUserViewer,
			},
			{
				UID:         "no-role@the-high-council.com",
				Email:       "no-role@the-high-council.com",
				Password:    "password",
				DisplayName: "No Role User",
				Role:        "",
			},
			{
				UID:         "simulated-user1-uid",
				Email:       "user1@agentic-foundry.local",
				Password:    "password",
				DisplayName: "User One (Project Creator)",
				Role:        db.RoleOrgEditor,
			},
			{
				UID:         "simulated-user2-uid",
				Email:       "user2@agentic-foundry.local",
				Password:    "password",
				DisplayName: "User Two",
				Role:        db.RoleOrgViewer,
			},
		}

		if authClient != nil {
			log.Println("Firebase Auth emulator detected. Preparing clean tenants...")
			// 1. Clean up all existing tenants in Firebase Auth Emulator to ensure deterministic seeding
			tenantsIter := authClient.TenantManager.Tenants(ctx, "")
			for {
				tenant, err := tenantsIter.Next()
				if err == iterator.Done {
					break
				}
				if err == nil {
					log.Printf("Cleaning up existing Firebase tenant: %s (%s)", tenant.DisplayName, tenant.ID)
					_ = authClient.TenantManager.DeleteTenant(ctx, tenant.ID)
				}
			}

			// 2. Create Tenants in Firebase Auth Emulator
			tenantHC, errHC := authClient.TenantManager.CreateTenant(ctx, (&auth.TenantToCreate{}).
				DisplayName("The-High-Council").
				AllowPasswordSignUp(true).
				EnableEmailLinkSignIn(true))
			if errHC != nil {
				log.Fatalf("Failed to create High Council tenant in Firebase Auth: %v", errHC)
			}
			tenantID1 = tenantHC.ID
			err = organizationStore.CreateOrganizationWithID(ctx, orgID, &db.Organization{
				Name:     "The High Council",
				TenantID: tenantID1,
			})
			if err != nil {
				log.Fatalf("Failed to save High Council organization in Firestore: %v", err)
			}
			log.Printf("Created organization %q with ID %q (Tenant ID: %q)", "The High Council", orgID, tenantID1)

			tenantSO, errSO := authClient.TenantManager.CreateTenant(ctx, (&auth.TenantToCreate{}).
				DisplayName("The-Sith-Order").
				AllowPasswordSignUp(true).
				EnableEmailLinkSignIn(true))
			if errSO != nil {
				log.Fatalf("Failed to create Sith Order tenant in Firebase Auth: %v", errSO)
			}
			tenantID2 = tenantSO.ID
			err = organizationStore.CreateOrganizationWithID(ctx, orgID2, &db.Organization{
				Name:     "The Sith Order",
				TenantID: tenantID2,
			})
			if err != nil {
				log.Fatalf("Failed to save Sith Order organization in Firestore: %v", err)
			}
			log.Printf("Created organization %q with ID %q (Tenant ID: %q)", "The Sith Order", orgID2, tenantID2)

			// 3. Seed users into the Firebase Auth Tenants
			for _, u := range usersToSeed {
				if err := seedUserInTenant(ctx, tenantID1, orgID, u); err != nil {
					log.Printf("Warning: Failed to seed user %s in High Council tenant: %v", u.Email, err)
				}
				if err := seedUserInTenant(ctx, tenantID2, orgID2, u); err != nil {
					log.Printf("Warning: Failed to seed user %s in Sith Order tenant: %v", u.Email, err)
				}
			}
		} else {
			// Fallback (Mock/offline mode)
			orgID, err = organizationStore.CreateOrganization(ctx, &db.Organization{
				Name:     "The High Council",
				TenantID: "mock-tenant-high-council",
			})
			if err != nil {
				log.Fatalf("Failed to create default organization: %v", err)
			}
			log.Printf("Created default organization %q with ID %q", "The High Council", orgID)

			orgID2, err = organizationStore.CreateOrganization(ctx, &db.Organization{
				Name:     "The Sith Order",
				TenantID: "mock-tenant-sith-order",
			})
			if err != nil {
				log.Fatalf("Failed to create default organization: %v", err)
			}
			log.Printf("Created default organization %q with ID %q", "The Sith Order", orgID2)
		}

		// Seed base configuration (workflows, local user) in The Sith Order (using Firestore Org ID)
		sithCtx := context.WithValue(ctx, core.OrganizationIDKey, orgID2)
		seedWorkflows(sithCtx)
		enableOrgIntegration(ctx, orgID2, "gitlab")
		enableOrgIntegration(ctx, orgID2, "jira")
		enableOrgIntegration(ctx, orgID2, "gemini")

		if userStore != nil {
			log.Printf("Seeding local user %s in Sith Order...", adminUID)
			claimsCtx := context.WithValue(sithCtx, core.UserClaimsKey, &core.UserClaims{
				Subject:        adminUID,
				Username:       adminEmail,
				OrganizationID: orgID2,
			})
			_, err := userStore.GetOrCreateUser(claimsCtx, orgID2, adminUID)
			if err != nil {
				log.Printf("Failed to seed user %s in Sith Order: %v", adminUID, err)
			} else {
				if err := userStore.UpdateTourStatus(claimsCtx, orgID2, adminUID, &db.TourStatus{Completed: true}); err != nil {
					log.Printf("Failed to set completed tour status for user %s in Sith Order: %v", adminUID, err)
				}
			}
		}

		localCtx := context.WithValue(ctx, core.OrganizationIDKey, orgID)
		local(localCtx, client)

		// Seed global app integrations once at the root level
		seedAppIntegrations(ctx)
		enableOrgIntegration(ctx, orgID, "github")
		enableOrgIntegration(ctx, orgID, "gemini")
	case "staging":
		staging(ctx, client)
	case "prod":
		prod(ctx, client)
	}
}

func local(ctx context.Context, client *firestore.Client) {
	log.Println("Starting local database seeding...")
	seedWorkflows(ctx)
	seedAppIntegrations(ctx)
	seedFeatures(ctx)
	seedLocalData(ctx)
	log.Println("Local database seeding completed successfully.")
}

func staging(ctx context.Context, client *firestore.Client) {
	log.Println("Starting staging database seeding...")
	seedWorkflows(ctx)
	seedAppIntegrations(ctx)
	seedFeatures(ctx)
	seedStagingData(ctx)
	log.Println("Staging database seeding completed successfully.")
}

func prod(ctx context.Context, client *firestore.Client) {
	log.Println("Starting production database seeding...")
	seedWorkflows(ctx)
	seedAppIntegrations(ctx)
	seedFeatures(ctx)
	seedProdData(ctx)
	log.Println("Production database seeding completed successfully.")
}

type SeedUser struct {
	UID         string
	Email       string
	Password    string
	DisplayName string
	Role        db.UserRole
}

func seedUserInTenant(ctx context.Context, tenantID string, orgID string, user SeedUser) error {
	if authClient == nil {
		return nil
	}

	tenantClient, err := authClient.TenantManager.AuthForTenant(tenantID)
	if err != nil {
		return fmt.Errorf("failed to get auth client for tenant %s: %w", tenantID, err)
	}

	// Try to create the user in Firebase Auth Tenant
	u := (&auth.UserToCreate{}).
		UID(user.UID).
		Email(user.Email).
		EmailVerified(true).
		Password(user.Password).
		DisplayName(user.DisplayName)

	userRecord, err := tenantClient.CreateUser(ctx, u)
	if err != nil {
		// If user already exists, we can update them instead of failing
		if auth.IsEmailAlreadyExists(err) || auth.IsUIDAlreadyExists(err) {
			log.Printf("User %s already exists in tenant %s, updating...", user.Email, tenantID)
			uu := (&auth.UserToUpdate{}).
				Email(user.Email).
				DisplayName(user.DisplayName).
				Password(user.Password)
			_, err = tenantClient.UpdateUser(ctx, user.UID, uu)
			if err != nil {
				return fmt.Errorf("failed to update user in tenant: %w", err)
			}
		} else {
			return fmt.Errorf("failed to create user in tenant: %w", err)
		}
	} else {
		log.Printf("Created user %s (%s) in tenant %s", userRecord.Email, userRecord.UID, tenantID)
	}

	// Set custom claims (roles, permissions & organization_id)
	permissions, err := core.GetPermissionsForRole(ctx, user.Role)
	if err != nil {
		log.Printf("Warning: Failed to fetch permissions for role %s: %v", user.Role, err)
	}
	claims := map[string]interface{}{
		"roles":           []string{string(user.Role)},
		"role":            string(user.Role),
		"permissions":     permissions,
		"organization_id": orgID, // Set Firestore Org ID
	}
	err = tenantClient.SetCustomUserClaims(ctx, user.UID, claims)
	if err != nil {
		return fmt.Errorf("failed to set custom claims for user %s: %w", user.UID, err)
	}

	return nil
}

func clearFirestoreEmulator(ctx context.Context, host string, projectID string, databaseID string) {
	url := fmt.Sprintf("http://%s/emulator/v1/projects/%s/databases/%s/documents", host, projectID, databaseID)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		log.Printf("Warning: Failed to create request to clear Firestore emulator: %v", err)
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Warning: Failed to clear Firestore emulator: %v", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		log.Printf("Successfully cleared Firestore emulator database %s", databaseID)
	} else {
		log.Printf("Warning: Failed to clear Firestore emulator, status: %s", resp.Status)
	}
}

func enableOrgIntegration(ctx context.Context, orgID string, integrationID string) {
	integration := &db.AppIntegration{
		IsEnabled: true,
	}
	integration.ID = integrationID
	if err := configStore.SaveOrgAppIntegration(ctx, orgID, integration); err != nil {
		log.Printf("Failed to enable integration %s for organization %s: %v", integrationID, orgID, err)
	} else {
		log.Printf("Enabled integration %s for organization %s", integrationID, orgID)
	}
}
