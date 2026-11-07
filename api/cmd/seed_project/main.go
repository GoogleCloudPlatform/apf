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
	"encoding/json"
	"flag"
	"io"
	"log"
	"os"
	"path/filepath"

	"cloud.google.com/go/firestore"
	"google.com/apf/api/core"
	"google.com/apf/api/db"
)

func main() {
	mode := flag.String("mode", "seed", "Operation mode: 'seed' (seed project from template) or 'extract' (extract project to template)")
	projectID := flag.String("project-id", "", "The ID of the existing project (required)")
	userID := flag.String("user-id", "", "The ID of the user owning/seeding the data (required for 'seed' mode)")
	sourceDir := flag.String("source", "", "The source folder path (required)")
	destDir := flag.String("destination", "", "The destination folder path (required)")
	artifactsJSONPath := flag.String("artifacts-json", "", "Path to the JSON file containing the list of artifacts (required for 'seed', optional for 'extract')")

	flag.Parse()

	if *mode != "seed" && *mode != "extract" {
		log.Fatalf("Invalid mode %q. Supported modes: 'seed', 'extract'", *mode)
	}

	if *projectID == "" || *sourceDir == "" || *destDir == "" {
		flag.Usage()
		os.Exit(1)
	}

	// Mode-specific validation
	if *mode == "seed" {
		if *userID == "" || *artifactsJSONPath == "" {
			log.Println("Error: -user-id and -artifacts-json are required for 'seed' mode.")
			flag.Usage()
			os.Exit(1)
		}
	}

	// 1. Validate source folder
	srcInfo, err := os.Stat(*sourceDir)
	if err != nil {
		log.Fatalf("Error checking source directory %q: %v", *sourceDir, err)
	}
	if !srcInfo.IsDir() {
		log.Fatalf("Source path %q is not a directory", *sourceDir)
	}

	// 2. Copy directory from source to destination recursively
	log.Printf("[%s mode] Copying files from %q to %q...", *mode, *sourceDir, *destDir)
	if err := copyDir(*sourceDir, *destDir); err != nil {
		log.Fatalf("Failed to copy directory: %v", err)
	}
	log.Println("Directory copy completed successfully.")

	// 3. Initialize database connection using core configuration
	cfg := core.NewConfig()
	ctx := context.Background()

	var client *firestore.Client
	if os.Getenv("FIRESTORE_EMULATOR_HOST") != "" {
		log.Printf("Detected Firebase Emulator environment: %s", os.Getenv("FIRESTORE_EMULATOR_HOST"))
		client, err = firestore.NewClientWithDatabase(ctx, cfg.GoogleCloudProject, cfg.FirestoreDbName)
	} else {
		log.Printf("Using production Firestore database: %s", cfg.GoogleCloudProject)
		client, err = firestore.NewClientWithDatabase(ctx, cfg.GoogleCloudProject, cfg.FirestoreDbName)
	}
	if err != nil {
		log.Fatalf("Failed to create Firestore client: %v", err)
	}
	defer client.Close()

	const defaultOrgID = "default-organization"

	// Verify project exists (optional but recommended log check)
	projectStore := db.NewFirestoreProjectStore(client)
	proj, err := projectStore.GetProject(ctx, defaultOrgID, *projectID)
	if err != nil {
		log.Printf("Warning: Could not fetch project %q from database: %v. Proceeding.", *projectID, err)
	} else {
		log.Printf("Verified project exists: %s (ID: %s)", *proj.Name, proj.ID)
	}

	artifactStore := db.NewFirestoreArtifactStore(client)

	if *mode == "seed" {
		// 4a. Read and parse artifacts JSON file
		artifactsJSONBytes, err := os.ReadFile(*artifactsJSONPath)
		if err != nil {
			log.Fatalf("Error reading artifacts JSON file %q: %v", *artifactsJSONPath, err)
		}

		var templateArtifacts []db.Artifact
		if err := json.Unmarshal(artifactsJSONBytes, &templateArtifacts); err != nil {
			log.Fatalf("Error parsing artifacts JSON: %v", err)
		}
		log.Printf("Successfully parsed %d artifacts from JSON", len(templateArtifacts))

		// 5a. Seed artifacts into Firestore (Idempotent by Path)
		existingArtifacts, err := artifactStore.ListArtifactsForProject(ctx, defaultOrgID, *projectID)
		if err != nil {
			log.Fatalf("Failed to list existing artifacts for project %q: %v", *projectID, err)
		}

		// Map existing artifacts by path for easy lookup
		existingMap := make(map[string]db.Artifact)
		for _, a := range existingArtifacts {
			existingMap[a.Path] = a
		}

		for _, art := range templateArtifacts {
			// Enrich/Override fields with CLI parameter context
			art.ProjectId = *projectID
			if art.Owner == "" {
				art.Owner = *userID
			}
			if art.ArtifactType == "" {
				art.ArtifactType = db.DetermineArtifactType(art.Path)
			}

			if existing, exists := existingMap[art.Path]; exists {
				// Update existing artifact
				art.ID = existing.ID
				if err := artifactStore.UpdateArtifact(ctx, defaultOrgID, &art); err != nil {
					log.Fatalf("Failed to update existing artifact for path %q: %v", art.Path, err)
				}
				log.Printf("Updated existing artifact ID %s for path %q", art.ID, art.Path)
			} else {
				// Create new artifact
				if err := artifactStore.CreateArtifact(ctx, defaultOrgID, &art); err != nil {
					log.Fatalf("Failed to create new artifact for path %q: %v", art.Path, err)
				}
				log.Printf("Created new artifact ID %s for path %q", art.ID, art.Path)
			}
		}
		log.Println("All project data and database artifacts successfully seeded!")

	} else if *mode == "extract" {
		// 4b. Query all artifacts for project
		log.Printf("Retrieving artifacts for project %q from Firestore...", *projectID)
		artifacts, err := artifactStore.ListArtifactsForProject(ctx, defaultOrgID, *projectID)
		if err != nil {
			log.Fatalf("Failed to list artifacts for project %q: %v", *projectID, err)
		}
		log.Printf("Retrieved %d artifacts from database", len(artifacts))

		// Clean up ID and ProjectId if we want to export a clean template
		for i := range artifacts {
			artifacts[i].ID = ""
			artifacts[i].ProjectId = ""
		}

		// 5b. Marshal to formatted JSON
		log.Println("Serializing artifacts to JSON...")
		artifactsJSONBytes, err := json.MarshalIndent(artifacts, "", "  ")
		if err != nil {
			log.Fatalf("Failed to serialize artifacts: %v", err)
		}

		// 6b. Write JSON file to target destination
		targetJSONPath := *artifactsJSONPath
		if targetJSONPath == "" {
			targetJSONPath = filepath.Join(*destDir, "artifacts.json")
		}

		if err := os.WriteFile(targetJSONPath, artifactsJSONBytes, 0644); err != nil {
			log.Fatalf("Failed to write artifacts JSON file to %q: %v", targetJSONPath, err)
		}
		log.Printf("Successfully exported artifacts JSON to %q!", targetJSONPath)
		log.Println("Project extraction and artifact serialization completed successfully!")
	}
}

// copyDir copies a directory tree, attempting to merge and overwrite existing files.
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		targetPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(targetPath, info.Mode())
		}

		return copyFile(path, targetPath)
	})
}

// copyFile copies a single file, overwriting the target file if it exists.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
