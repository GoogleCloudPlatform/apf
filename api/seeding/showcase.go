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
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"google.com/apf/api/core"
	"google.com/apf/api/db"
)

//go:embed templates/showcase/*
var showcaseTemplates embed.FS

func extractShowcaseWorkspaceFiles(organizationID, userID, projectID string) error {
	var mountPath string
	if appConfig != nil {
		mountPath = appConfig.GetWorkspaceMountPath()
	} else {
		mountPath = core.NewConfig().GetWorkspaceMountPath()
	}
	if strings.HasPrefix(mountPath, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to resolve home directory: %w", err)
		}
		mountPath = filepath.Join(home, mountPath[2:])
	}
	projectPath := filepath.Join(mountPath, organizationID, userID, projectID)

	artifactsData, err := showcaseTemplates.ReadFile("templates/showcase/artifacts.json")
	if err != nil {
		return fmt.Errorf("failed to read artifacts template: %w", err)
	}

	var templatesArtifacts []db.Artifact
	if err := json.Unmarshal(artifactsData, &templatesArtifacts); err != nil {
		return fmt.Errorf("failed to unmarshal artifacts template: %w", err)
	}

	log.Printf("[Seeding Showcase] Extracting spec files to workspace path: %s", projectPath)
	for _, art := range templatesArtifacts {
		filename := filepath.Base(art.Path)
		content, err := showcaseTemplates.ReadFile("templates/showcase/workspace/" + filename)
		if err != nil {
			return fmt.Errorf("missing template file for artifact %s (%s) in embed: %w", art.ID, filename, err)
		}

		destPath := filepath.Join(projectPath, art.Path)
		destDir := filepath.Dir(destPath)
		if err := os.MkdirAll(destDir, 0755); err != nil {
			return fmt.Errorf("failed to create destination directory %s: %w", destDir, err)
		}

		if err := os.WriteFile(destPath, content, 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", destPath, err)
		}
		log.Printf("[Seeding Showcase] Wrote artifact file: %s", destPath)
	}
	return nil
}

func ensureShowcaseArtifactsInStore(ctx context.Context, organizationID, userID, projectID string) error {
	artifactsData, err := showcaseTemplates.ReadFile("templates/showcase/artifacts.json")
	if err != nil {
		return fmt.Errorf("failed to read artifacts template: %w", err)
	}

	var templatesArtifacts []db.Artifact
	if err := json.Unmarshal(artifactsData, &templatesArtifacts); err != nil {
		return fmt.Errorf("failed to unmarshal artifacts template: %w", err)
	}

	store := artifactStore
	if store == nil {
		log.Printf("[Seeding Showcase] artifactStore is nil, skipping store artifact sync")
		return nil
	}

	existingArts, errList := store.ListArtifactsForProject(ctx, organizationID, projectID)
	if errList == nil && len(existingArts) > 0 {
		hasMismatched := false
		for _, eArt := range existingArts {
			matched := false
			for _, tArt := range templatesArtifacts {
				tID := fmt.Sprintf("%s_%s", projectID, filepath.Base(tArt.Path))
				if eArt.ID == tID || eArt.Path == tArt.Path {
					matched = true
					break
				}
			}
			if !matched {
				hasMismatched = true
				break
			}
		}

		if hasMismatched {
			log.Printf("[Seeding Showcase] Found mismatched artifact paths for project %s. Cleaning up and re-seeding artifacts...", projectID)
			for _, eArt := range existingArts {
				_ = store.DeleteArtifact(ctx, organizationID, eArt.ID)
			}
			existingArts = nil
		}
	}

	if len(existingArts) == 0 {
		for _, art := range templatesArtifacts {
			art.ID = fmt.Sprintf("%s_%s", projectID, filepath.Base(art.Path))
			art.ProjectId = projectID
			art.Owner = userID

			if err := store.CreateArtifact(ctx, organizationID, &art); err != nil {
				log.Printf("[Seeding Showcase] Failed to create artifact %s: %v", art.ID, err)
			}
		}
	}
	return nil
}
