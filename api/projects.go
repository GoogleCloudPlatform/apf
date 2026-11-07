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
	"google.com/apf/api/workspace"

	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.com/apf/api/core"
	"google.com/apf/api/db"
	"google.com/apf/api/integrations"
)

var store db.ProjectStore
var requestStore db.AccessRequestStore
var configStore db.ConfigStore
var autojoinStore db.AutojoinStore
var FirestoreClient *firestore.Client

func listProjectsHandler(w http.ResponseWriter, r *http.Request) {
	claims, _ := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	userId := ""
	if claims != nil {
		userId = claims.Subject
	}

	if userId == "" {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "You dont have access to see projects", Status: "ERROR", Title: "Unauthorized"})
		return
	}

	organizationID, errOrg := core.GetOrganizationID(r.Context())
	if errOrg != nil {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized: Organization ID missing", Status: "ERROR"})
		return
	}

	dbMyProjects, dbBrowseProjects, err := store.ListProjectsByUser(r.Context(), organizationID, userId)
	if err != nil {
		fmt.Printf("Error listing projects for user: %v\n", err)
		sendJSON(w, http.StatusInternalServerError, Response{Message: "Failed to list projects", Status: "ERROR", Title: "Error listing projects"})
		return
	}

	myProjects := []*db.Project{}
	for i := range dbMyProjects {
		p := &dbMyProjects[i]
		// Fetch artifacts for this project
		artifacts, err := workspace.ArtifactStore.ListArtifactsForProject(r.Context(), organizationID, p.ID)
		if err != nil {
			fmt.Printf("Error listing artifacts for project %s: %v\n", p.ID, err)
			artifacts = []db.Artifact{}
		}
		p.Artifacts = artifacts

		// Fetch read-only status for this user/project
		userProj, errUserProj := userProjectStore.GetProjectAppSetting(r.Context(), organizationID, userId, p.ID, "github")
		if errUserProj == nil && userProj != nil {
			p.IsReadOnly = userProj.IsReadOnly
		}

		myProjects = append(myProjects, p)
	}

	browseProjects := []*db.Project{}
	for i := range dbBrowseProjects {
		p := &dbBrowseProjects[i]
		// Fetch artifacts for this project
		artifacts, err := workspace.ArtifactStore.ListArtifactsForProject(r.Context(), organizationID, p.ID)
		if err != nil {
			fmt.Printf("Error listing artifacts for project %s: %v\n", p.ID, err)
			artifacts = []db.Artifact{}
		}
		p.Artifacts = artifacts
		browseProjects = append(browseProjects, p)
	}

	response := map[string][]*db.Project{
		"myProjects":     myProjects,
		"browseProjects": browseProjects,
	}
	sendJSON(w, http.StatusOK, response)
}

func getProjectHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "new" {
		status := "New"
		visibility := "public"
		baseBranch := "main"
		icon := "🚀"
		members := make(map[string]string)
		govGates := map[string]string{
			"ONBOARDING":     "Not Started",
			"ANALYSIS":       "Not Started",
			"PLANNING":       "Not Started",
			"SOLUTIONING":    "Not Started",
			"IMPLEMENTATION": "Not Started",
		}

		claims, _ := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
		var ownerId *string
		if claims != nil {
			ownerId = &claims.Subject
			members[claims.Subject] = "owner"
		}

		blankProject := &db.Project{
			ID:              "new",
			Status:          &status,
			Visibility:      &visibility,
			BaseBranch:      &baseBranch,
			Icon:            &icon,
			Members:         &members,
			GovernanceGates: &govGates,
			OwnerId:         ownerId,
			BmadConfig: &db.BmadConfig{
				SpecFolder:                  "docs/spec",
				ImplementationArtifactsPath: "implementation",
				PlanningArtifactsPath:       "plans",
				TestingArtifactsPath:        "test",
			},
		}
		sendJSON(w, http.StatusOK, blankProject)
		return
	}

	organizationID, errOrg := core.GetOrganizationID(r.Context())
	if errOrg != nil {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized: Organization ID missing", Status: "ERROR"})
		return
	}

	project, err := store.GetProject(r.Context(), organizationID, id)
	if err != nil {
		sendJSON(w, http.StatusNotFound, Response{Message: fmt.Sprintf("Project not found: %v", err), Status: "ERROR"})
		return
	}

	claims, _ := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if claims != nil && claims.Subject != "" {
		userProj, errUserProj := userProjectStore.GetProjectAppSetting(r.Context(), organizationID, claims.Subject, project.ID, "github")
		if errUserProj == nil && userProj != nil {
			project.IsReadOnly = userProj.IsReadOnly
		}
	}

	sendJSON(w, http.StatusOK, project)
}

func createProjectHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized context", Status: "ERROR"})
		return
	}

	organizationID, errOrg := core.GetOrganizationID(r.Context())
	if errOrg != nil {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized: Organization ID missing", Status: "ERROR"})
		return
	}

	if !core.HasPermission(r.Context(), claims.Roles, core.PermissionProjectCreate) {
		sendJSON(w, http.StatusForbidden, Response{Message: "Forbidden: Requires project:create permission", Status: "ERROR"})
		return
	}

	var p db.Project
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Invalid request payload", Status: "ERROR"})
		return
	}

	status := "New"
	p.Status = &status
	p.OwnerId = &claims.Subject
	p.CreatedAt = time.Now()

	if p.Visibility == nil || *p.Visibility == "" {
		visibility := "public"
		p.Visibility = &visibility
	}

	if p.Members == nil {
		m := make(map[string]string)
		p.Members = &m
	}
	(*p.Members)[claims.Subject] = "owner"

	if p.GovernanceGates == nil {
		g := make(map[string]string)
		p.GovernanceGates = &g
	}
	(*p.GovernanceGates)["ONBOARDING"] = "Not Started"
	(*p.GovernanceGates)["ANALYSIS"] = "Not Started"
	(*p.GovernanceGates)["PLANNING"] = "Not Started"
	(*p.GovernanceGates)["SOLUTIONING"] = "Not Started"
	(*p.GovernanceGates)["IMPLEMENTATION"] = "Not Started"

	if p.BaseBranch == nil || *p.BaseBranch == "" {
		baseBranch := "main"
		p.BaseBranch = &baseBranch
	}

	if err := store.CreateProject(r.Context(), organizationID, &p); err != nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to create project: %v", err), Status: "ERROR"})
		return
	}

	// Initialize workspace (clone repo or create folder and inject required folders) as soon as project is saved
	var accessToken string
	if p.GetRepoUrl() != "" {
		accessToken, _ = integrations.ResolveGitHubToken(r.Context(), claims.Subject, p.ID)
	}
	if errInit := workspace.InitializeProjectWorkspace(r.Context(), organizationID, claims.Subject, &p, accessToken); errInit != nil {
		slog.Warn("Failed to initialize workspace upon project creation", "error", errInit, "projectId", p.ID, "organizationID", organizationID)
	}

	sendJSON(w, http.StatusCreated, p)
}

func updateProjectHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	organizationID, errOrg := core.GetOrganizationID(r.Context())
	if errOrg != nil {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized: Organization ID missing", Status: "ERROR"})
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Failed to read request body", Status: "ERROR"})
		return
	}

	existingProject, err := store.GetProject(r.Context(), organizationID, id)
	if err != nil {
		sendJSON(w, http.StatusNotFound, Response{Message: fmt.Sprintf("Project not found: %v", err), Status: "ERROR"})
		return
	}

	var payload db.Project
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Invalid request payload", Status: "ERROR"})
		return
	}

	// Apply updates
	if payload.Name != nil {
		existingProject.Name = payload.Name
	}
	if payload.Desc != nil {
		existingProject.Desc = payload.Desc
	}
	if payload.Status != nil {
		existingProject.Status = payload.Status
		if *payload.Status == "Onboarded" {
			if existingProject.GovernanceGates == nil {
				g := make(map[string]string)
				existingProject.GovernanceGates = &g
			}
			(*existingProject.GovernanceGates)["ONBOARDING"] = "Complete"
		}
	}
	if id == "foundry-showcase-project" {
		visibility := "private"
		existingProject.Visibility = &visibility
	} else if payload.Visibility != nil {
		existingProject.Visibility = payload.Visibility
	}
	if payload.BaseBranch != nil {
		existingProject.BaseBranch = payload.BaseBranch
	}
	if payload.Icon != nil {
		existingProject.Icon = payload.Icon
	}
	if payload.Members != nil {
		existingProject.Members = payload.Members
	}
	if payload.GovernanceGates != nil {
		existingProject.GovernanceGates = payload.GovernanceGates
	}
	if payload.LinkedWorkflows != nil {
		existingProject.LinkedWorkflows = payload.LinkedWorkflows
	}
	if payload.BmadConfig != nil {
		existingProject.BmadConfig = payload.BmadConfig
	}

	if err := store.UpdateProject(r.Context(), organizationID, existingProject); err != nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to update project: %v", err), Status: "ERROR"})
		return
	}

	// Ensure workspace is initialized upon project save/update
	claims, _ := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	userID := ""
	if claims != nil {
		userID = claims.Subject
	} else if existingProject.OwnerId != nil {
		userID = *existingProject.OwnerId
	}

	if userID != "" {
		var accessToken string
		if existingProject.GetRepoUrl() != "" {
			accessToken, _ = integrations.ResolveGitHubToken(r.Context(), userID, existingProject.ID)
		}
		if errInit := workspace.InitializeProjectWorkspace(r.Context(), organizationID, userID, existingProject, accessToken); errInit != nil {
			slog.Warn("Failed to initialize workspace upon project update", "error", errInit, "projectId", existingProject.ID, "organizationID", organizationID)
		}
	}

	sendJSON(w, http.StatusOK, existingProject)
}

func deleteProjectHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	organizationID, errOrg := core.GetOrganizationID(r.Context())
	if errOrg != nil {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized: Organization ID missing", Status: "ERROR"})
		return
	}

	// Fetch existing project to get owner and members for workspace folder cleanup
	existingProject, _ := store.GetProject(r.Context(), organizationID, id)

	if err := store.DeleteProject(r.Context(), organizationID, id); err != nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to delete project: %v", err), Status: "ERROR"})
		return
	}

	// Clean up workspace filesystem folder(s): workspace mount path + OrgId + UserId + ProjectId
	var callerUserID string
	if claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims); ok && claims != nil {
		callerUserID = claims.Subject
	}

	if existingProject != nil {
		deleteProjectWorkspaceFolders(organizationID, existingProject, callerUserID)
	} else if id != "" {
		dummyProj := &db.Project{ID: id}
		deleteProjectWorkspaceFolders(organizationID, dummyProj, callerUserID)
	}

	w.WriteHeader(http.StatusNoContent)
}

func createAccessRequestHandler(w http.ResponseWriter, r *http.Request) {
	projectId := r.PathValue("id")
	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized context", Status: "ERROR"})
		return
	}

	organizationID, errOrg := core.GetOrganizationID(r.Context())
	if errOrg != nil {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized: Organization ID missing", Status: "ERROR"})
		return
	}

	var req db.AccessRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Invalid request payload", Status: "ERROR"})
		return
	}
	req.ProjectId = projectId
	req.UserId = claims.Subject
	req.UserName = claims.Username
	req.Status = "pending"
	req.RequestDate = time.Now().Format(time.RFC3339)

	if err := requestStore.CreateRequest(r.Context(), organizationID, &req); err != nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to create request: %v", err), Status: "ERROR"})
		return
	}
	sendJSON(w, http.StatusCreated, req)
}

func updateAccessRequestHandler(w http.ResponseWriter, r *http.Request) {
	projectId := r.PathValue("id")
	requestId := r.PathValue("requestId")
	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized context", Status: "ERROR"})
		return
	}

	organizationID, errOrg := core.GetOrganizationID(r.Context())
	if errOrg != nil {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized: Organization ID missing", Status: "ERROR"})
		return
	}

	project, err := store.GetProject(r.Context(), organizationID, projectId)
	if err != nil {
		sendJSON(w, http.StatusNotFound, Response{Message: fmt.Sprintf("Project not found: %v", err), Status: "ERROR"})
		return
	}
	if project.OwnerId != nil && *project.OwnerId != claims.Subject {
		sendJSON(w, http.StatusForbidden, Response{Message: "Forbidden: Only project owner can manage requests", Status: "ERROR"})
		return
	}

	var req db.AccessRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Invalid request payload", Status: "ERROR"})
		return
	}
	req.ID = requestId
	req.ProjectId = projectId

	if err := requestStore.UpdateRequest(r.Context(), organizationID, &req); err != nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to update request: %v", err), Status: "ERROR"})
		return
	}

	if req.Status == "approved" {
		if project.Members == nil {
			m := make(map[string]string)
			project.Members = &m
		}
		(*project.Members)[req.UserId] = req.RequestedRole
		if err := store.UpdateProject(r.Context(), organizationID, project); err != nil {
			sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to add member to project: %v", err), Status: "ERROR"})
			return
		}
	}

	sendJSON(w, http.StatusOK, req)
}

type UpdateIntegrationPayload struct {
	IntegrationID string                 `json:"integration_id"`
	Enabled       *bool                  `json:"enabled"`
	Settings      map[string]interface{} `json:"settings"`
}

func updateProjectIntegrationsHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || claims == nil {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized context", Status: "ERROR"})
		return
	}

	organizationID, errOrg := core.GetOrganizationID(r.Context())
	if errOrg != nil {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized: Organization ID missing", Status: "ERROR"})
		return
	}

	var payload UpdateIntegrationPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Invalid request payload", Status: "ERROR"})
		return
	}

	if payload.IntegrationID == "" {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Missing integration_id", Status: "ERROR"})
		return
	}

	if payload.Enabled == nil {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Missing enabled status", Status: "ERROR"})
		return
	}

	// If enabled is true, perform strict validation on required fields
	if *payload.Enabled {
		if payload.IntegrationID == "github" {
			repoUrl, _ := payload.Settings["repoUrl"].(string)
			specLocation, _ := payload.Settings["specLocation"].(string)
			baseBranch, _ := payload.Settings["baseBranch"].(string)
			specBranchPrefix, _ := payload.Settings["specBranchPrefix"].(string)

			if repoUrl == "" || specLocation == "" || baseBranch == "" || specBranchPrefix == "" {
				sendJSON(w, http.StatusBadRequest, Response{
					Message: "Missing required GitHub settings (repoUrl, specLocation, baseBranch, specBranchPrefix)",
					Status:  "ERROR",
				})
				return
			}
		}
	}

	userId := claims.Subject

	existingProject, err := store.GetProject(r.Context(), organizationID, id)
	if err != nil {
		sendJSON(w, http.StatusNotFound, Response{Message: fmt.Sprintf("Project not found: %v", err), Status: "ERROR"})
		return
	}

	processRepoStateTransition(r.Context(), organizationID, userId, existingProject, payload)

	if FirestoreClient == nil {
		found := false
		for i, integration := range existingProject.AppIntegrations {
			if integration.IntegrationID == payload.IntegrationID {
				found = true
				existingProject.AppIntegrations[i].Enabled = *payload.Enabled
				if *payload.Enabled {
					existingProject.AppIntegrations[i].Settings = payload.Settings
				}
				break
			}
		}

		if !found {
			existingProject.AppIntegrations = append(existingProject.AppIntegrations, db.ProjectIntegration{
				IntegrationID: payload.IntegrationID,
				Enabled:       *payload.Enabled,
				Settings:      payload.Settings,
			})
		}

		if payload.IntegrationID == "github" && *payload.Enabled {
			baseBranchStr := payload.Settings["baseBranch"].(string)
			existingProject.BaseBranch = &baseBranchStr
		}

		if err := store.UpdateProject(r.Context(), organizationID, existingProject); err != nil {
			sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to update project: %v", err), Status: "ERROR"})
			return
		}

		sendJSON(w, http.StatusOK, existingProject)
		return
	}

	docRef := db.OrganizationCollection(organizationID, FirestoreClient, "projects").Doc(id)
	err = FirestoreClient.RunTransaction(r.Context(), func(ctx context.Context, tx *firestore.Transaction) error {
		doc, err := tx.Get(docRef)
		if err != nil {
			return err
		}

		var p db.Project
		if err := doc.DataTo(&p); err != nil {
			return err
		}

		// Find or create the integration entry
		found := false
		for i, integration := range p.AppIntegrations {
			if integration.IntegrationID == payload.IntegrationID {
				found = true
				p.AppIntegrations[i].Enabled = *payload.Enabled
				if *payload.Enabled {
					p.AppIntegrations[i].Settings = payload.Settings
				}
				break
			}
		}

		if !found {
			p.AppIntegrations = append(p.AppIntegrations, db.ProjectIntegration{
				IntegrationID: payload.IntegrationID,
				Enabled:       *payload.Enabled,
				Settings:      payload.Settings,
			})
		}

		// If github integration is enabled, sync base branch
		if payload.IntegrationID == "github" && *payload.Enabled {
			baseBranchStr := payload.Settings["baseBranch"].(string)
			p.BaseBranch = &baseBranchStr
		}

		return tx.Set(docRef, p)
	})

	if err != nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to update project integration: %v", err), Status: "ERROR"})
		return
	}

	// Fetch updated project to return
	updatedProj, err := store.GetProject(r.Context(), organizationID, id)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to fetch updated project: %v", err), Status: "ERROR"})
		return
	}

	sendJSON(w, http.StatusOK, updatedProj)
}

// GET /api/v1/projects/check-repo
func checkRepoHandler(w http.ResponseWriter, r *http.Request) {
	repoUrl := r.URL.Query().Get("repoUrl")
	if repoUrl == "" {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Missing repoUrl parameter", Status: "ERROR"})
		return
	}
	repoUrl = strings.TrimSpace(repoUrl)
	repoUrlLower := strings.ToLower(repoUrl)
	repoUrlLower = strings.TrimSuffix(repoUrlLower, ".git")

	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || claims == nil {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized context", Status: "ERROR"})
		return
	}

	organizationID, errOrg := core.GetOrganizationID(r.Context())
	if errOrg != nil {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized: Organization ID missing", Status: "ERROR"})
		return
	}

	projects, err := store.ListProjects(r.Context(), organizationID)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to list projects: %v", err), Status: "ERROR"})
		return
	}

	for _, p := range projects {
		pRepoUrl := p.GetRepoUrl()
		if pRepoUrl != "" {
			pRepoUrlLower := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(pRepoUrl)), ".git")

			if pRepoUrlLower == repoUrlLower {
				hasAccess := false
				if p.OwnerId != nil && *p.OwnerId == claims.Subject {
					hasAccess = true
				} else if p.Members != nil {
					if _, ok := (*p.Members)[claims.Subject]; ok {
						hasAccess = true
					}
				}

				potentialRole := ""
				if !hasAccess {
					// Check GitHub permissions
					owner, repo, err := parseGitHubURL(pRepoUrl)
					if err == nil {
						// Resolve GitHub token for the current user
						token, errToken := integrations.ResolveGitHubToken(r.Context(), claims.Subject, "") // no project ID yet
						if errToken == nil && token != "" {
							if integrations.IsSimulatedToken(token) {
								potentialRole = "Editor"
							} else {
								client := &integrations.GitHubPRClient{Token: token}
								read, write, _, errCheck := client.CheckPermissions(r.Context(), owner, repo)
								if errCheck == nil {
									if write {
										potentialRole = "Editor"
									} else if read {
										potentialRole = "Viewer"
									}
								} else {
									log.Printf("WARNING: Failed to check GitHub permissions for user %s on %s/%s: %v", claims.Subject, owner, repo, errCheck)
								}
							}
						} else {
							log.Printf("INFO: Could not resolve GitHub token for user %s: %v", claims.Subject, errToken)
						}
					}
				}

				sendJSON(w, http.StatusOK, map[string]interface{}{
					"exists":        true,
					"projectId":     p.ID,
					"projectName":   *p.Name,
					"hasAccess":     hasAccess,
					"potentialRole": potentialRole,
				})
				return
			}
		}
	}

	sendJSON(w, http.StatusOK, map[string]interface{}{
		"exists": false,
	})
}

func parseGitHubURL(repoURL string) (string, string, error) {
	if !strings.HasPrefix(repoURL, "https://github.com/") {
		return "", "", fmt.Errorf("invalid repository URL: only HTTPS github.com URLs are supported")
	}
	trimmed := strings.TrimPrefix(repoURL, "https://github.com/")
	trimmed = strings.TrimSuffix(trimmed, ".git")
	trimmed = strings.TrimRight(trimmed, "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("failed to parse owner and repository name from URL")
	}
	return parts[0], parts[1], nil
}

// POST /api/v1/projects/{id}/join
func joinProjectHandler(w http.ResponseWriter, r *http.Request) {
	projectId := r.PathValue("id")
	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || claims == nil {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized context", Status: "ERROR"})
		return
	}

	organizationID, errOrg := core.GetOrganizationID(r.Context())
	if errOrg != nil {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized: Organization ID missing", Status: "ERROR"})
		return
	}

	project, err := store.GetProject(r.Context(), organizationID, projectId)
	if err != nil {
		sendJSON(w, http.StatusNotFound, Response{Message: fmt.Sprintf("Project not found: %v", err), Status: "ERROR"})
		return
	}

	// Check if already a member
	if project.OwnerId != nil && *project.OwnerId == claims.Subject {
		sendJSON(w, http.StatusBadRequest, Response{Message: "You are already the owner of this project", Status: "ERROR"})
		return
	}
	if project.Members != nil {
		if _, ok := (*project.Members)[claims.Subject]; ok {
			sendJSON(w, http.StatusBadRequest, Response{Message: "You are already a member of this project", Status: "ERROR"})
			return
		}
	}

	repoURL := project.GetRepoUrl()
	if repoURL == "" {
		sendJSON(w, http.StatusBadRequest, Response{Message: "This project is not tied to a repository", Status: "ERROR"})
		return
	}

	owner, repo, err := parseGitHubURL(repoURL)
	if err != nil {
		sendJSON(w, http.StatusBadRequest, Response{Message: fmt.Sprintf("Failed to parse project repository URL: %v", err), Status: "ERROR"})
		return
	}

	// Resolve GitHub token for the current user
	token, errToken := integrations.ResolveGitHubToken(r.Context(), claims.Subject, "") // no project ID yet
	if errToken != nil || token == "" {
		sendJSON(w, http.StatusForbidden, Response{Message: "Forbidden: Could not resolve your GitHub token. Please link your GitHub account first", Status: "ERROR"})
		return
	}

	assignedRole := ""
	if integrations.IsSimulatedToken(token) {
		assignedRole = "Editor"
	} else {
		client := &integrations.GitHubPRClient{Token: token}
		read, write, _, errCheck := client.CheckPermissions(r.Context(), owner, repo)
		if errCheck != nil {
			sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to check repository permissions: %v", errCheck), Status: "ERROR"})
			return
		}

		if write {
			assignedRole = "Editor"
		} else if read {
			assignedRole = "Viewer"
		} else {
			sendJSON(w, http.StatusForbidden, Response{Message: "Forbidden: You do not have read access to the associated repository on GitHub", Status: "ERROR"})
			return
		}
	}

	if project.Members == nil {
		m := make(map[string]string)
		project.Members = &m
	}

	(*project.Members)[claims.Subject] = assignedRole

	if err := store.UpdateProject(r.Context(), organizationID, project); err != nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to join project: %v", err), Status: "ERROR"})
		return
	}

	sendJSON(w, http.StatusOK, map[string]interface{}{
		"status": "SUCCESS",
		"role":   assignedRole,
	})
}

func processRepoStateTransition(ctx context.Context, organizationID string, userId string, existingProject *db.Project, payload UpdateIntegrationPayload) {
	if payload.IntegrationID != "github" {
		return
	}

	oldRepoUrl := existingProject.GetRepoUrl()
	newRepoUrl := ""
	if payload.Enabled != nil && *payload.Enabled && payload.Settings != nil {
		if url, ok := payload.Settings["repoUrl"].(string); ok {
			newRepoUrl = strings.TrimSpace(url)
		}
	}

	if oldRepoUrl == "" && newRepoUrl != "" {
		// Scenario 1: Local Git Repo -> Remote Connected Git Repo
		if err := workspace.HandleAttachRemoteRepo(ctx, organizationID, userId, existingProject, newRepoUrl, payload.Settings); err != nil {
			slog.Error("Failed to handle attach remote repo transition", "projectId", existingProject.ID, "error", err)
		}
	} else if oldRepoUrl != "" && newRepoUrl == "" {
		// Scenario 2: Remote Connected Git Repo -> Disconnected Repo
		if err := workspace.HandleDetachRemoteRepo(ctx, organizationID, userId, existingProject.ID); err != nil {
			slog.Error("Failed to handle detach remote repo transition", "projectId", existingProject.ID, "error", err)
		}
	} else if oldRepoUrl != "" && newRepoUrl != "" && oldRepoUrl != newRepoUrl {
		// Scenario 3: Change in Remote Connected Git Repo (Repo A -> Repo B)
		if err := workspace.HandleSwitchRemoteRepo(ctx, organizationID, userId, existingProject, newRepoUrl, payload.Settings); err != nil {
			slog.Error("Failed to handle switch remote repo transition", "projectId", existingProject.ID, "error", err)
		}
	}
}
