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

import (	"google.com/apf/api/workspace"

	"context"
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.com/apf/api/db"
	"google.com/apf/api/workflow"
)

//go:embed seeding/templates/showcase/*
var showcaseTemplates embed.FS

type tempMessage struct {
	SessionId              string `json:"sessionId"`
	Sender                 string `json:"sender"`
	Text                   string `json:"text"`
	IsUser                 bool   `json:"isUser"`
	TimestampOffsetSeconds int    `json:"timestamp_offset_seconds"`
}

// resolveProjectPath resolves the absolute path to the project's workspace directory.
func resolveProjectPath(orgID, userID, projectID string) (string, error) {
	mountPath := appConfig.WorkspaceMountPath
	if strings.HasPrefix(mountPath, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to resolve home directory: %w", err)
		}
		mountPath = filepath.Join(home, mountPath[2:])
	}
	return filepath.Join(mountPath, orgID, userID, projectID), nil
}

// seedShowcase seeds the showcase project. If projectID is empty, a deterministic one is generated.
func seedShowcase(ctx context.Context, orgID, userID, projectID string) (*db.Project, error) {
	if projectID == "" {
		projectID = "foundry-showcase-project"
	}

	// Fail-safe cleanup on seeding failure
	success := false
	defer func() {
		if !success {
			// Clean up partially seeded Firestore records
			log.Printf("[Showcase Seeder] Seeding failed. Triggering rollback cleanup for project %s", projectID)
			_ = store.DeleteProject(ctx, orgID, projectID)
			
			// Clean up filesystem
			projectPath, err := resolveProjectPath(orgID, userID, projectID)
			if err == nil {
				_ = os.RemoveAll(projectPath)
			}
		}
	}()

	// 1. Load and create Project
	projectData, err := showcaseTemplates.ReadFile("seeding/templates/showcase/project.json")
	if err != nil {
		return nil, fmt.Errorf("failed to read project template: %w", err)
	}

	var p db.Project
	if err := json.Unmarshal(projectData, &p); err != nil {
		return nil, fmt.Errorf("failed to unmarshal project template: %w", err)
	}

	p.ID = projectID
	p.OwnerId = &userID
	p.CreatedAt = time.Now()
	visibility := "private"
	p.Visibility = &visibility
	
	if p.Members == nil {
		m := make(map[string]string)
		p.Members = &m
	}
	(*p.Members)[userID] = "owner"

	p.AppIntegrations = []db.ProjectIntegration{
		{
			IntegrationID: "github",
			Enabled:       true,
			Settings:      map[string]interface{}{},
		},
		{
			IntegrationID: "gemini",
			Enabled:       true,
			Settings:      map[string]interface{}{},
		},
	}

	if err := store.CreateProject(ctx, orgID, &p); err != nil {
		return nil, fmt.Errorf("failed to create project in DB: %w", err)
	}

	// 2. Load and create Sessions
	sessionsData, err := showcaseTemplates.ReadFile("seeding/templates/showcase/sessions.json")
	if err != nil {
		return nil, fmt.Errorf("failed to read sessions template: %w", err)
	}

	var templatesSessions []db.Session
	if err := json.Unmarshal(sessionsData, &templatesSessions); err != nil {
		return nil, fmt.Errorf("failed to unmarshal sessions template: %w", err)
	}

	sessionIDMap := make(map[string]string)
	for _, sess := range templatesSessions {
		templateID := sess.ID
		newSessID := "showcase-session-" + uuid.New().String()
		sessionIDMap[templateID] = newSessID

		sess.ID = newSessID
		sess.ProjectId = projectID
		sess.UserId = userID
		sess.CreatedAt = time.Now().Add(-1 * time.Hour)
		sess.LastActiveAt = time.Now()

		if err := sessionStore.CreateSession(ctx, orgID, &sess); err != nil {
			return nil, fmt.Errorf("failed to create session %s: %w", newSessID, err)
		}
	}

	// Create the global session document explicitly to prevent orphaned messages (Ghost Sessions)
	globalSess := db.Session{
		ID:             "project_global_" + projectID,
		ProjectId:      projectID,
		UserId:         userID,
		Phase:          "GLOBAL",
		Status:         "active",
		CreatedAt:      time.Now().Add(-1 * time.Hour),
		LastActiveAt:   time.Now(),
	}
	if err := sessionStore.CreateSession(ctx, orgID, &globalSess); err != nil {
		return nil, fmt.Errorf("failed to create global session: %w", err)
	}

	// 3. Load and create Messages
	messagesData, err := showcaseTemplates.ReadFile("seeding/templates/showcase/messages.json")
	if err != nil {
		return nil, fmt.Errorf("failed to read messages template: %w", err)
	}

	var tempMsgs []tempMessage
	if err := json.Unmarshal(messagesData, &tempMsgs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal messages template: %w", err)
	}

	for _, tMsg := range tempMsgs {
		newSessID, ok := sessionIDMap[tMsg.SessionId]
		if !ok {
			// Fallback to the explicitly seeded global session document
			newSessID = "project_global_" + projectID
		}

		msg := db.Message{
			SessionId:      newSessID,
			ProjectId:      projectID,
			OrganizationID: orgID,
			Sender:         tMsg.Sender,
			Text:           tMsg.Text,
			IsUser:         tMsg.IsUser,
			Timestamp:      time.Now().Add(time.Duration(tMsg.TimestampOffsetSeconds) * time.Second),
		}

		if _, err := Store.CreateMessage(ctx, orgID, projectID, userID, &msg); err != nil {
			return nil, fmt.Errorf("failed to create message: %w", err)
		}
	}

	// 4. Load and create Artifacts matching showcase templates
	if err := ensureShowcaseArtifactsInStore(ctx, orgID, userID, projectID); err != nil {
		return nil, fmt.Errorf("failed to ensure showcase artifacts: %w", err)
	}

	// 4.5. Load and seed Showcase Workflow
	workflowData, err := showcaseTemplates.ReadFile("seeding/templates/showcase/auto_dev_loop_workflow.json")
	if err != nil {
		return nil, fmt.Errorf("failed to read showcase workflow template: %w", err)
	}

	var wf db.WorkflowTemplate
	if err := json.Unmarshal(workflowData, &wf); err != nil {
		return nil, fmt.Errorf("failed to unmarshal showcase workflow template: %w", err)
	}

	wf.ID = "showcase-workflow-dev-loop"
	if err := workflow.Store.CreateWorkflow(ctx, &wf); err != nil {
		return nil, fmt.Errorf("failed to create showcase workflow in DB: %w", err)
	}
	log.Printf("[Showcase Seeder] Successfully seeded showcase workflow: %s", wf.ID)

	// 5. Extract and write workspace spec files to the exact paths defined in artifacts metadata
	if err := extractShowcaseWorkspaceFiles(orgID, userID, projectID); err != nil {
		return nil, err
	}

	success = true
	return &p, nil
}

// extractShowcaseWorkspaceFiles extracts embedded showcase spec files to the workspace path for a user.
func extractShowcaseWorkspaceFiles(orgID, userID, projectID string) error {
	projectPath, err := resolveProjectPath(orgID, userID, projectID)
	if err != nil {
		return fmt.Errorf("failed to resolve project path: %w", err)
	}

	artifactsData, err := showcaseTemplates.ReadFile("seeding/templates/showcase/artifacts.json")
	if err != nil {
		return fmt.Errorf("failed to read artifacts template: %w", err)
	}

	var templatesArtifacts []db.Artifact
	if err := json.Unmarshal(artifactsData, &templatesArtifacts); err != nil {
		return fmt.Errorf("failed to unmarshal artifacts template: %w", err)
	}

	log.Printf("[Showcase Seeder] Beginning spec files extraction to workspace path: %s", projectPath)
	for _, art := range templatesArtifacts {
		filename := filepath.Base(art.Path)
		content, err := showcaseTemplates.ReadFile("seeding/templates/showcase/workspace/" + filename)
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
		log.Printf("[Showcase Seeder] Successfully wrote artifact file: %s", destPath)
	}
	return nil
}

// ensureShowcaseArtifactsInStore ensures that showcase artifact records in store match showcase template artifacts.
func ensureShowcaseArtifactsInStore(ctx context.Context, orgID, userID, projectID string) error {
	artifactsData, err := showcaseTemplates.ReadFile("seeding/templates/showcase/artifacts.json")
	if err != nil {
		return fmt.Errorf("failed to read artifacts template: %w", err)
	}

	var templatesArtifacts []db.Artifact
	if err := json.Unmarshal(artifactsData, &templatesArtifacts); err != nil {
		return fmt.Errorf("failed to unmarshal artifacts template: %w", err)
	}

	if workspace.ArtifactStore == nil {
		log.Printf("[Showcase Seeder] workspace.ArtifactStore is nil, skipping store artifact sync")
		return nil
	}

	existingArts, errList := workspace.ArtifactStore.ListArtifactsForProject(ctx, orgID, projectID)
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
			log.Printf("[Showcase Seeder] Found mismatched artifact paths for project %s. Cleaning up and re-seeding artifacts...", projectID)
			for _, eArt := range existingArts {
				_ = workspace.ArtifactStore.DeleteArtifact(ctx, orgID, eArt.ID)
			}
			existingArts = nil
		}
	}

	if len(existingArts) == 0 {
		for _, art := range templatesArtifacts {
			art.ID = fmt.Sprintf("%s_%s", projectID, filepath.Base(art.Path))
			art.ProjectId = projectID
			art.Owner = userID

			if err := workspace.ArtifactStore.CreateArtifact(ctx, orgID, &art); err != nil {
				log.Printf("[Showcase Seeder] Failed to create artifact %s: %v", art.ID, err)
			}
		}
	}
	return nil
}


// ResetShowcase resets the showcase project by deleting its data and re-seeding it.
func ResetShowcase(ctx context.Context, orgID, userID, projectID string) error {
	// Security Guard: Prevent path traversal attacks
	if strings.Contains(projectID, "..") || strings.Contains(projectID, "/") || strings.Contains(projectID, "\\") {
		return fmt.Errorf("invalid project ID: path traversal detected")
	}

	// 1. Delete all Firestore records for this project
	if err := store.DeleteProject(ctx, orgID, projectID); err != nil {
		return fmt.Errorf("failed to delete project records for reset: %w", err)
	}

	// 2. Clear the filesystem directory
	projectPath, err := resolveProjectPath(orgID, userID, projectID)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(projectPath); err != nil {
		return fmt.Errorf("failed to clear project workspace directory: %w", err)
	}

	// 3. Re-seed with the SAME projectID
	_, err = seedShowcase(ctx, orgID, userID, projectID)
	if err != nil {
		return fmt.Errorf("failed to re-seed project: %w", err)
	}

	return nil
}
