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
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"google.com/apf/api/core"
	"google.com/apf/api/db"
)

//go:embed templates/default/*
var defaultTemplates embed.FS

func seedLocalData(ctx context.Context) {
	organizationID, errOrg := core.GetOrganizationID(ctx)
	if errOrg != nil {
		log.Fatalf("Failed to resolve organization ID in seedLocalData: %v", errOrg)
	}

	ownerID := "simulated-admin-uid"
	if appConfig != nil {
		ownerID = appConfig.SimulatedUserID
	} else {
		ownerID = core.NewConfig().SimulatedUserID
	}
	status := "Onboarded"
	visibility := "public"

	var projects []db.Project

	// Load default project from JSON template
	defaultProjBytes, errDef := defaultTemplates.ReadFile("templates/default/project.json")
	if errDef != nil {
		log.Printf("Warning: Failed to read default project template: %v", errDef)
	} else {
		var defProj db.Project
		if errUnmarshal := json.Unmarshal(defaultProjBytes, &defProj); errUnmarshal != nil {
			log.Printf("Warning: Failed to unmarshal default project template: %v", errUnmarshal)
		} else {
			if defProj.Members == nil {
				m := make(map[string]string)
				defProj.Members = &m
			}
			(*defProj.Members)[ownerID] = "owner"
			if defProj.CreatedAt.IsZero() {
				defProj.CreatedAt = time.Now().Add(-1 * time.Hour)
			}
			projects = append(projects, defProj)
		}
	}

	// Showcase Project
	showcaseName := "Foundry Showcase"
	showcaseDesc := "Building the next-generation autonomous pair programming AI assistant."
	baseBranch := "main"
	showcaseProj := db.Project{
		ID:         "foundry-showcase-project",
		Name:       &showcaseName,
		Desc:       &showcaseDesc,
		Status:     &status,
		OwnerId:    &ownerID,
		Visibility: &visibility,
		BaseBranch: &baseBranch,
		AppIntegrations: []db.ProjectIntegration{
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
		},
		LinkedWorkflows: []string{"showcase-workflow-dev-loop"},
		Members: &map[string]string{
			ownerID: "owner",
		},
		GovernanceGates: &map[string]string{
			"ONBOARDING":     "Complete",
			"ANALYSIS":       "Not Started",
			"PLANNING":       "Not Started",
			"SOLUTIONING":    "Not Started",
			"IMPLEMENTATION": "Not Started",
		},
		CreatedAt: time.Now().Add(-1 * time.Hour),
	}
	projects = append(projects, showcaseProj)

	// Delete existing projects with these names/IDs to force clean re-seeding
	existingProjects, err := projectStore.ListProjects(ctx, organizationID)
	if err == nil {
		for _, p := range existingProjects {
			for _, projData := range projects {
				pName := ""
				if p.Name != nil {
					pName = *p.Name
				}
				projName := ""
				if projData.Name != nil {
					projName = *projData.Name
				}
				if (pName != "" && pName == projName) || p.ID == projData.ID {
					log.Printf("Deleting existing project %q (%s) to re-seed...", pName, p.ID)
					deleteProjectWorkspaceFolders(ctx, organizationID, &p)
					_ = projectStore.DeleteProject(ctx, organizationID, p.ID)
					break
				}
			}
		}
	}

	for _, p := range projects {
		projName := p.ID
		if p.Name != nil {
			projName = *p.Name
		}

		err := projectStore.CreateProject(ctx, organizationID, &p)
		if err != nil {
			log.Printf("Failed to create project %q: %v", projName, err)
			continue
		}
		log.Printf("Created seeded project: %s with ID: %s", projName, p.ID)

		// Seed artifacts for showcase project
		if p.ID == "foundry-showcase-project" {
			if errShowcaseArts := ensureShowcaseArtifactsInStore(ctx, organizationID, ownerID, p.ID); errShowcaseArts != nil {
				log.Printf("Failed to seed showcase artifacts for project %s: %v", p.ID, errShowcaseArts)
			}
			if errShowcaseFiles := extractShowcaseWorkspaceFiles(organizationID, ownerID, p.ID); errShowcaseFiles != nil {
				log.Printf("Failed to extract showcase workspace files for project %s: %v", p.ID, errShowcaseFiles)
			}
			continue
		}
	}

	devToken := os.Getenv("DEV_REPO_ACCESS_TOKEN")
	if devToken == "" {
		log.Fatalf("\n================================================================================\nDATABASE SEEDING ERROR: DEV_REPO_ACCESS_TOKEN environment variable is missing!\nPlease set DEV_REPO_ACCESS_TOKEN in your environment or config.env before seeding.\n================================================================================\n")
	}

	geminiAPIKey := os.Getenv("GEMINI_API_KEY")
	if geminiAPIKey == "" {
		log.Fatalf("\n================================================================================\nDATABASE SEEDING ERROR: GEMINI_API_KEY environment variable is missing!\nPlease set GEMINI_API_KEY in your environment or config.env before seeding.\n================================================================================\n")
	}
	type seedUserInfo struct {
		UID  string
		Role db.UserRole
	}
	simulatedUsers := []seedUserInfo{
		{UID: ownerID, Role: db.RoleOrgOwner},
		{UID: "org-owner@the-high-council.com", Role: db.RoleOrgOwner},
		{UID: "org-admin@the-high-council.com", Role: db.RoleOrgAdmin},
		{UID: "org-editor@the-high-council.com", Role: db.RoleOrgEditor},
		{UID: "org-viewer@the-high-council.com", Role: db.RoleOrgViewer},
		{UID: "project-admin@the-high-council.com", Role: db.RoleProjectAdmin},
		{UID: "project-editor@the-high-council.com", Role: db.RoleProjectEditor},
		{UID: "project-viewer@the-high-council.com", Role: db.RoleProjectViewer},
		{UID: "project-creator@the-high-council.com", Role: db.RoleProjectCreator},
		{UID: "user-admin@the-high-council.com", Role: db.RoleUserAdmin},
		{UID: "user-editor@the-high-council.com", Role: db.RoleUserEditor},
		{UID: "user-viewer@the-high-council.com", Role: db.RoleUserViewer},
		{UID: "no-role@the-high-council.com", Role: ""},
		{UID: "simulated-user1-uid", Role: db.RoleOrgEditor},
		{UID: "simulated-user2-uid", Role: db.RoleOrgViewer},
	}

	for _, uInfo := range simulatedUsers {
		uID := uInfo.UID
		if userStore != nil {
			log.Printf("Seeding user %s in organization...", uID)
			
			// Inject simulated claims into context so GetOrCreateUser can resolve the email
			email := uID
			if uID == "simulated-admin-uid" {
				email = "admin@agentic-foundry.local"
			} else if uID == "simulated-user1-uid" {
				email = "user1@agentic-foundry.local"
			} else if uID == "simulated-user2-uid" {
				email = "user2@agentic-foundry.local"
			} else if strings.Contains(uID, "@") {
				email = uID
			}

			claimsCtx := context.WithValue(ctx, core.UserClaimsKey, &core.UserClaims{
				Subject:  uID,
				Username: email,
			})

			_, err := userStore.GetOrCreateUser(claimsCtx, organizationID, uID)
			if err != nil {
				log.Printf("Failed to seed user %s: %v", uID, err)
				continue
			}

			if err := userStore.UpdateUserRole(claimsCtx, organizationID, uID, uInfo.Role); err != nil {
				log.Printf("Failed to set role %s for user %s: %v", uInfo.Role, uID, err)
			} else {
				log.Printf("Seeded role %s for user %s", uInfo.Role, uID)
			}

			if err := userStore.UpdateTourStatus(claimsCtx, organizationID, uID, &db.TourStatus{Completed: true}); err != nil {
				log.Printf("Failed to set completed tour status for user %s: %v", uID, err)
			} else {
				log.Printf("Seeded completed tour status for user %s", uID)
			}
		}
	}

	for _, uInfo := range simulatedUsers {
		uID := uInfo.UID
		userEmail := uID
		if !strings.Contains(userEmail, "@") {
			userEmail = "octocat@github.com"
		}
		cred := &db.AppCredential{
			IntegrationID: "github",
			Fields: []db.CredentialAttribute{
				{
					Name:        "pat",
					DisplayName: "Personal Access Token",
					Enabled:     true,
					Required:    true,
					HelpText:    "Enter your GitHub Personal Access Token (PAT) with repo and user scopes.",
					IsSecret:    true,
					Value:       devToken,
				},
				{
					Name:        "email",
					DisplayName: "Git Author Email",
					Enabled:     true,
					Required:    true,
					HelpText:    "Your GitHub-verified author email used for Git commits and sync.",
					IsSecret:    false,
					Value:       userEmail,
				},
			},
		}
		if err := appCredentialStore.SaveCredential(ctx, organizationID, uID, "github", cred); err != nil {
			log.Printf("Failed to save GitHub credential for user %s: %v", uID, err)
		} else {
			log.Printf("Successfully seeded GitHub app credential for user %s", uID)
		}


		encryptedGeminiKey, errEnc := db.EncryptSecret(geminiAPIKey)
		if errEnc != nil {
			log.Fatalf("Failed to encrypt Gemini API key during local seeding: %v", errEnc)
		}

		geminiCred := &db.AppCredential{
			IntegrationID: "gemini",
			Fields: []db.CredentialAttribute{
				{
					Name:        "gemini_api_key",
					DisplayName: "Gemini API Key",
					Enabled:     true,
					Required:    true,
					HelpText:    "Enter your Google Gemini API Key.",
					IsSecret:    true,
					Value:       encryptedGeminiKey,
				},
			},
		}
		if err := appCredentialStore.SaveCredential(ctx, organizationID, uID, "gemini", geminiCred); err != nil {
			log.Printf("Failed to save Gemini credential for user %s: %v", uID, err)
		} else {
			log.Printf("Successfully seeded Gemini app credential for user %s", uID)
		}

		if userProjectStore != nil {
			for _, projData := range projects {
				githubSetting := &db.UserProject{
					UserID:       uID,
					ProjectID:    projData.ID,
					AppID:        "github",
					Enabled:      true,
					CredentialID: "github",
					IsOverridden: false,
					IsReadOnly:   false,
				}
				if err := userProjectStore.SaveProjectAppSetting(ctx, organizationID, githubSetting); err != nil {
					log.Printf("Failed to save GitHub user project setting for user %s, project %s: %v", uID, projData.ID, err)
				} else {
					log.Printf("Successfully seeded GitHub user project setting for user %s (%s)", uID, projData.ID)
				}

				geminiSetting := &db.UserProject{
					UserID:       uID,
					ProjectID:    projData.ID,
					AppID:        "gemini",
					Enabled:      true,
					CredentialID: "gemini",
					IsOverridden: false,
					IsReadOnly:   false,
				}
				if err := userProjectStore.SaveProjectAppSetting(ctx, organizationID, geminiSetting); err != nil {
					log.Printf("Failed to save Gemini user project setting for user %s, project %s: %v", uID, projData.ID, err)
				} else {
					log.Printf("Successfully seeded Gemini user project setting for user %s (%s)", uID, projData.ID)
				}
			}
		}
	}
}

func deleteProjectWorkspaceFolders(ctx context.Context, organizationID string, p *db.Project, extraUserIDs ...string) {
	if p == nil || p.ID == "" || p.ID == "." || p.ID == ".." || strings.Contains(p.ID, "/") || strings.Contains(p.ID, "\\") {
		return
	}
	if organizationID == "" || organizationID == "." || organizationID == ".." || strings.Contains(organizationID, "/") || strings.Contains(organizationID, "\\") {
		return
	}

	var mountPath string
	if appConfig != nil {
		mountPath = appConfig.GetWorkspaceMountPath()
	} else {
		mountPath = core.NewConfig().GetWorkspaceMountPath()
	}
	if strings.HasPrefix(mountPath, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			mountPath = filepath.Join(home, mountPath[2:])
		}
	}
	cleanMount := filepath.Clean(mountPath)

	userMap := make(map[string]bool)
	if p.OwnerId != nil && *p.OwnerId != "" {
		userMap[*p.OwnerId] = true
	}
	if p.Members != nil {
		for memberID := range *p.Members {
			if memberID != "" {
				userMap[memberID] = true
			}
		}
	}
	for _, uID := range extraUserIDs {
		if uID != "" {
			userMap[uID] = true
		}
	}

	baseDirs := []string{cleanMount, filepath.Join(cleanMount, "organizations")}
	for _, bDir := range baseDirs {
		orgDir := filepath.Join(bDir, organizationID)
		if userEntries, err := os.ReadDir(orgDir); err == nil {
			for _, uEntry := range userEntries {
				if uEntry.IsDir() && !strings.HasPrefix(uEntry.Name(), ".") {
					userMap[uEntry.Name()] = true
				}
			}
		}
	}

	for userID := range userMap {
		if userID == "" || userID == "." || userID == ".." || strings.Contains(userID, "/") || strings.Contains(userID, "\\") {
			continue
		}
		for _, bDir := range baseDirs {
			targetPath := filepath.Join(bDir, organizationID, userID, p.ID)
			if err := os.RemoveAll(targetPath); err != nil {
				log.Printf("Warning: Failed to clean up workspace path %s: %v", targetPath, err)
			} else {
				log.Printf("Successfully cleaned up workspace path %s", targetPath)
			}

			lockPath := targetPath + ".lock"
			_ = os.RemoveAll(lockPath)
		}
	}
}

