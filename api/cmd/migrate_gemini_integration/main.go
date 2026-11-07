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
	"log"
	"os"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.com/apf/api/core"
	"google.com/apf/api/db"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "Preview migration changes without modifying Firestore")
	gcpProject := flag.String("project-id", "", "Google Cloud Project ID (defaults to config)")
	flag.Parse()

	cfg := core.NewConfig()
	targetProject := cfg.GoogleCloudProject
	if *gcpProject != "" {
		targetProject = *gcpProject
	}

	ctx := context.Background()
	var client *firestore.Client
	var err error

	if os.Getenv("FIRESTORE_EMULATOR_HOST") != "" {
		log.Printf("[Migration] Using Firestore emulator: %s", os.Getenv("FIRESTORE_EMULATOR_HOST"))
		client, err = firestore.NewClientWithDatabase(ctx, targetProject, cfg.FirestoreDbName)
	} else {
		log.Printf("[Migration] Connecting to Firestore project %q (db: %q)...", targetProject, cfg.FirestoreDbName)
		client, err = firestore.NewClientWithDatabase(ctx, targetProject, cfg.FirestoreDbName)
	}
	if err != nil {
		log.Fatalf("[Migration Error] Failed to create Firestore client: %v", err)
	}
	defer client.Close()

	geminiIntegration := &db.AppIntegration{
		ID:           "gemini",
		Name:         "Gemini Models",
		Description:  "Google Gemini AI integration for model calls and agent reasoning.",
		AuthType:     "apikey",
		Icon:         "gemini",
		IsEnabled:    true,
		DisplayOrder: 1,
		Labels:       []string{"AI Model"},
		Fields: []db.CredentialAttribute{
			{
				Name:        "gemini_api_key",
				DisplayName: "Gemini API Key",
				Enabled:     true,
				HelpText:    "Enter your Google Gemini API Key.",
				IsSecret:    true,
			},
		},
	}

	if *dryRun {
		log.Println("[Migration DRY-RUN] Previewing upsert of global app integration 'gemini':")
		log.Printf("[Migration DRY-RUN]   ID: %s, Name: %s, AuthType: %s, IsEnabled: %t",
			geminiIntegration.ID, geminiIntegration.Name, geminiIntegration.AuthType, geminiIntegration.IsEnabled)
	} else {
		log.Println("[Migration] Upserting global app integration 'gemini'...")
		_, err := client.Collection("app_integrations").Doc(geminiIntegration.ID).Set(ctx, geminiIntegration)
		if err != nil {
			log.Fatalf("[Migration Error] Failed to upsert global gemini integration: %v", err)
		}
		log.Println("[Migration] Successfully upserted global app integration 'gemini'.")
	}

	// Iterate over all organization collections to upsert org-level catalog document
	orgIter := client.Collection("organizations").Documents(ctx)
	orgCount := 0
	for {
		doc, err := orgIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatalf("[Migration Error] Error iterating organizations: %v", err)
		}


		orgID := doc.Ref.ID
		orgCount++

		if *dryRun {
			log.Printf("[Migration DRY-RUN] Would upsert 'gemini' integration in organization collection: %s/app_integrations", orgID)
		} else {
			orgIntegrationRef := client.Collection("organizations").Doc(orgID).Collection("app_integrations").Doc(geminiIntegration.ID)
			_, err := orgIntegrationRef.Set(ctx, geminiIntegration)
			if err != nil {
				log.Printf("[Migration Error] Failed to upsert gemini integration for org %s: %v", orgID, err)
			} else {
				log.Printf("[Migration] Upserted gemini integration for org %s", orgID)
			}
		}
	}

	if *dryRun {
		log.Printf("[Migration DRY-RUN Complete] Processed 1 global catalog and %d organization(s). Zero writes performed.", orgCount)
	} else {
		log.Printf("[Migration Complete] Processed 1 global catalog and %d organization(s). Migration finished successfully!", orgCount)
	}
}
