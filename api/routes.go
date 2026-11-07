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
	"net/http"

	"google.com/apf/api/core"
	"google.com/apf/api/integrations"
	"google.com/apf/api/workflow"
	"google.com/apf/api/workspace"
)

func registerRoutes(mux *http.ServeMux, chatHandler *ChatHandler, sessionHandler *SessionHandler, adminHandler *AdminHandler, authHandler *AuthHandler, featureHandler *FeatureHandler, feedbackHandler *FeedbackHandler, telemetryHandler *TelemetryHandler) {
	// Simple health check and version endpoints
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("GET /api/v1/health", healthHandler)
	mux.HandleFunc("GET /api/v1/version", versionHandler)
	mux.HandleFunc("OPTIONS /api/v1/version", preflightHandler)

	// Auth endpoints
	mux.HandleFunc("POST /api/v1/auth/login", authHandler.Login)
	mux.HandleFunc("OPTIONS /api/v1/auth/login", authHandler.Login) // Options for preflight
	mux.HandleFunc("POST /api/v1/auth/logout", authHandler.Logout)
	mux.HandleFunc("OPTIONS /api/v1/auth/logout", authHandler.Logout) // Options for preflight
	mux.HandleFunc("GET /api/v1/auth/me", authMiddleware(authHandler.Me))
	mux.HandleFunc("OPTIONS /api/v1/auth/me", preflightHandler)
	mux.HandleFunc("GET /api/v1/auth/discover", authHandler.DiscoverOrganization)
	mux.HandleFunc("OPTIONS /api/v1/auth/discover", preflightHandler)

	// Note: listUserOrganizationsHandler and selectOrganizationHandler are now methods on authHandler
	mux.HandleFunc("GET /api/v1/auth/organizations", authMiddleware(authHandler.ListUserOrganizations))
	mux.HandleFunc("OPTIONS /api/v1/auth/organizations", preflightHandler)
	mux.HandleFunc("POST /api/v1/auth/select-organization", authMiddleware(authHandler.SelectOrganization))
	mux.HandleFunc("OPTIONS /api/v1/auth/select-organization", preflightHandler)

	// Administrative Organization endpoints
	mux.HandleFunc("POST /api/v1/organizations", authMiddleware(createOrganizationHandler))
	mux.HandleFunc("OPTIONS /api/v1/organizations", preflightHandler)

	// Modular Integrations endpoints
	mux.HandleFunc("GET /api/v1/integrations/github/connect", authMiddleware(orgScopeMiddleware(integrations.GitHubConnectHandler)))
	mux.HandleFunc("OPTIONS /api/v1/integrations/github/connect", preflightHandler)
	mux.HandleFunc("GET /api/v1/integrations/github/callback", optionalAuthMiddleware(integrations.GitHubCallbackHandler)) // Callback cannot be wrapped with orgScopeMiddleware as it lacks header
	mux.HandleFunc("POST /api/v1/auth/github/pat", authMiddleware(orgScopeMiddleware(integrations.GitHubLinkPATHandler)))
	mux.HandleFunc("OPTIONS /api/v1/auth/github/pat", preflightHandler)
	mux.HandleFunc("GET /api/v1/integrations/github/status", authMiddleware(orgScopeMiddleware(integrations.GitHubStatusHandler)))
	mux.HandleFunc("OPTIONS /api/v1/integrations/github/status", preflightHandler)
	mux.HandleFunc("GET /api/v1/integrations/github/profile", authMiddleware(orgScopeMiddleware(integrations.GitHubProfileHandler)))
	mux.HandleFunc("OPTIONS /api/v1/integrations/github/profile", preflightHandler)
	mux.HandleFunc("GET /api/v1/integrations/github/repos", authMiddleware(orgScopeMiddleware(integrations.GitHubReposHandler)))
	mux.HandleFunc("OPTIONS /api/v1/integrations/github/repos", preflightHandler)
	mux.HandleFunc("POST /api/v1/integrations/github/validate-permissions", authMiddleware(orgScopeMiddleware(integrations.ValidateGitHubPermissionsHandler)))
	mux.HandleFunc("OPTIONS /api/v1/integrations/github/validate-permissions", preflightHandler)
	mux.HandleFunc("DELETE /api/v1/integrations/github", authMiddleware(orgScopeMiddleware(integrations.GitHubDisconnectHandler)))
	mux.HandleFunc("OPTIONS /api/v1/integrations/github", preflightHandler)

	// Projects resource endpoints
	mux.HandleFunc("GET /api/v1/projects", authMiddleware(orgScopeMiddleware(listProjectsHandler)))
	mux.HandleFunc("GET /api/v1/projects/{id}", authMiddleware(orgScopeMiddleware(projectAccessMiddleware(getProjectHandler))))
	mux.HandleFunc("OPTIONS /api/v1/projects/{id}", preflightHandler)
	mux.HandleFunc("GET /api/v1/projects/check-repo", authMiddleware(orgScopeMiddleware(checkRepoHandler)))
	mux.HandleFunc("OPTIONS /api/v1/projects/check-repo", preflightHandler)
	mux.HandleFunc("POST /api/v1/projects/{id}/join", authMiddleware(orgScopeMiddleware(joinProjectHandler)))
	mux.HandleFunc("OPTIONS /api/v1/projects/{id}/join", preflightHandler)
	mux.HandleFunc("POST /api/v1/projects", authMiddleware(orgScopeMiddleware(createProjectHandler))) // Protected
	mux.HandleFunc("OPTIONS /api/v1/projects", preflightHandler)                  // Options for preflight
	mux.HandleFunc("PUT /api/v1/projects/{id}", authMiddleware(orgScopeMiddleware(projectAccessMiddleware(updateProjectHandler))))
	mux.HandleFunc("DELETE /api/v1/projects/{id}", authMiddleware(orgScopeMiddleware(projectAccessMiddleware(deleteProjectHandler))))
	mux.HandleFunc("POST /api/v1/projects/{id}/integrations", authMiddleware(orgScopeMiddleware(projectAccessMiddleware(updateProjectIntegrationsHandler))))
	mux.HandleFunc("OPTIONS /api/v1/projects/{id}/integrations", preflightHandler)
	
	// Showcase Project endpoints
	mux.HandleFunc("POST /api/v1/projects/provision-showcase", authMiddleware(orgScopeMiddleware(provisionShowcaseHandler)))
	mux.HandleFunc("OPTIONS /api/v1/projects/provision-showcase", preflightHandler)
	mux.HandleFunc("POST /api/v1/projects/{id}/reset", authMiddleware(orgScopeMiddleware(projectAccessMiddleware(resetShowcaseHandler))))
	mux.HandleFunc("OPTIONS /api/v1/projects/{id}/reset", preflightHandler)
	
	// Project Onboarding endpoints (Story 7.2)
	mux.HandleFunc("GET /api/v1/projects/{id}/onboard/report", authMiddleware(orgScopeMiddleware(projectAccessMiddleware(getOnboardingReportHandler))))
	mux.HandleFunc("OPTIONS /api/v1/projects/{id}/onboard/report", preflightHandler)

	mux.HandleFunc("POST /api/v1/projects/{id}/requests", authMiddleware(orgScopeMiddleware(createAccessRequestHandler)))
	mux.HandleFunc("OPTIONS /api/v1/projects/{id}/requests", preflightHandler)
	mux.HandleFunc("PUT /api/v1/projects/{id}/requests/{requestId}", authMiddleware(orgScopeMiddleware(updateAccessRequestHandler)))
	mux.HandleFunc("OPTIONS /api/v1/projects/{id}/requests/{requestId}", preflightHandler)

	// Project Members endpoints
	mux.HandleFunc("GET /api/v1/projects/{id}/members", authMiddleware(orgScopeMiddleware(projectAccessMiddleware(listProjectMembersHandler))))
	mux.HandleFunc("POST /api/v1/projects/{id}/members", authMiddleware(orgScopeMiddleware(projectAccessMiddleware(addProjectMembersHandler))))
	mux.HandleFunc("PUT /api/v1/projects/{id}/members/{memberId}", authMiddleware(orgScopeMiddleware(projectAccessMiddleware(updateProjectMemberRoleHandler))))
	mux.HandleFunc("DELETE /api/v1/projects/{id}/members/{memberId}", authMiddleware(orgScopeMiddleware(projectAccessMiddleware(removeProjectMemberHandler))))
	mux.HandleFunc("OPTIONS /api/v1/projects/{id}/members", preflightHandler)
	mux.HandleFunc("OPTIONS /api/v1/projects/{id}/members/{memberId}", preflightHandler)

	// Workspace resource endpoints (Milestone B)
	mux.HandleFunc("GET /api/v1/projects/{id}/messages", authMiddleware(orgScopeMiddleware(projectAccessMiddleware(chatHandler.ListMessagesHandler))))
	mux.HandleFunc("GET /api/v1/projects/{id}/messages/stream", authMiddleware(orgScopeMiddleware(projectAccessMiddleware(chatHandler.StreamMessageHandler))))
	mux.HandleFunc("POST /api/v1/projects/{id}/messages", authMiddleware(orgScopeMiddleware(projectAccessMiddleware(chatHandler.UserMessageHandler))))
	mux.HandleFunc("POST /api/v1/projects/{id}/messages/cancel", authMiddleware(orgScopeMiddleware(projectAccessMiddleware(chatHandler.CancelMessageHandler))))
	mux.HandleFunc("OPTIONS /api/v1/projects/{id}/messages", preflightHandler)
	mux.HandleFunc("OPTIONS /api/v1/projects/{id}/messages/stream", preflightHandler)
	mux.HandleFunc("OPTIONS /api/v1/projects/{id}/messages/cancel", preflightHandler)

	// Project Activities endpoints
	mux.HandleFunc("GET /api/v1/projects/{id}/activities/stream", authMiddleware(orgScopeMiddleware(projectAccessMiddleware(StreamActivityHandler))))
	mux.HandleFunc("POST /api/v1/projects/{id}/activities", authMiddleware(orgScopeMiddleware(projectAccessMiddleware(IngestActivityHandler))))
	mux.HandleFunc("OPTIONS /api/v1/projects/{id}/activities/stream", preflightHandler)
	mux.HandleFunc("OPTIONS /api/v1/projects/{id}/activities", preflightHandler)

	// Workspace Session endpoints
	mux.HandleFunc("POST /api/v1/projects/{id}/sessions", authMiddleware(orgScopeMiddleware(projectAccessMiddleware(sessionHandler.connectSessionHandler))))
	mux.HandleFunc("GET /api/v1/projects/{id}/sessions", authMiddleware(orgScopeMiddleware(projectAccessMiddleware(sessionHandler.listSessionsHandler))))
	mux.HandleFunc("PATCH /api/v1/projects/{id}/sessions/{sessionId}", authMiddleware(orgScopeMiddleware(projectAccessMiddleware(sessionHandler.updateSessionHandler))))
	mux.HandleFunc("PUT /api/v1/projects/{id}/sessions/{sessionId}", authMiddleware(orgScopeMiddleware(projectAccessMiddleware(sessionHandler.updateSessionHandler))))
	mux.HandleFunc("DELETE /api/v1/projects/{id}/sessions/{sessionId}", authMiddleware(orgScopeMiddleware(projectAccessMiddleware(sessionHandler.deleteSessionHandler))))
	mux.HandleFunc("POST /api/v1/projects/{id}/sessions/{sessionId}/artifacts/upload", authMiddleware(orgScopeMiddleware(projectAccessMiddleware(sessionHandler.uploadSessionArtifactHandler))))
	mux.HandleFunc("OPTIONS /api/v1/projects/{id}/sessions", preflightHandler)
	mux.HandleFunc("OPTIONS /api/v1/projects/{id}/sessions/{sessionId}", preflightHandler)
	mux.HandleFunc("OPTIONS /api/v1/projects/{id}/sessions/{sessionId}/artifacts/upload", preflightHandler)

	// Sandbox Stack Discovery endpoint (Story 19.1)
	mux.HandleFunc("POST /api/v1/projects/{id}/sessions/{sessionId}/sandbox/discover", authMiddleware(orgScopeMiddleware(projectAccessMiddleware(SandboxDiscoverHandler))))
	mux.HandleFunc("OPTIONS /api/v1/projects/{id}/sessions/{sessionId}/sandbox/discover", preflightHandler)

	// Sandbox Build endpoint (Story 19.3)
	mux.HandleFunc("POST /api/v1/projects/{id}/sessions/{sessionId}/sandbox/build", authMiddleware(orgScopeMiddleware(projectAccessMiddleware(SandboxBuildHandler))))
	mux.HandleFunc("OPTIONS /api/v1/projects/{id}/sessions/{sessionId}/sandbox/build", preflightHandler)

	// Sandbox Synthesize endpoint (Story 19.4)
	mux.HandleFunc("POST /api/v1/projects/{id}/sessions/{sessionId}/sandbox/synthesize", authMiddleware(orgScopeMiddleware(projectAccessMiddleware(SandboxSynthesizeHandler))))
	mux.HandleFunc("OPTIONS /api/v1/projects/{id}/sessions/{sessionId}/sandbox/synthesize", preflightHandler)

	// Sandbox Verify endpoint (Story 19.6)
	mux.HandleFunc("POST /api/v1/projects/{id}/sessions/{sessionId}/sandbox/verify", authMiddleware(orgScopeMiddleware(projectAccessMiddleware(SandboxVerifyHandler))))
	mux.HandleFunc("OPTIONS /api/v1/projects/{id}/sessions/{sessionId}/sandbox/verify", preflightHandler)

	// Sandbox Teardown endpoint (Story 19.6)
	mux.HandleFunc("POST /api/v1/projects/{id}/sessions/{sessionId}/sandbox/teardown", authMiddleware(orgScopeMiddleware(projectAccessMiddleware(SandboxTeardownHandler))))
	mux.HandleFunc("OPTIONS /api/v1/projects/{id}/sessions/{sessionId}/sandbox/teardown", preflightHandler)

	// VCS Re-Hydration Pull endpoint
	mux.HandleFunc("POST /api/v1/projects/{id}/git/pull", authMiddleware(orgScopeMiddleware(projectAccessMiddleware(workspace.GitPullHandler))))
	mux.HandleFunc("OPTIONS /api/v1/projects/{id}/git/pull", preflightHandler)

	// VCS Push & PR Sync endpoint
	mux.HandleFunc("POST /api/v1/projects/{id}/git/sync", authMiddleware(orgScopeMiddleware(projectAccessMiddleware(workspace.GitSyncHandler))))
	mux.HandleFunc("OPTIONS /api/v1/projects/{id}/git/sync", preflightHandler)

	// VCS Merge endpoint
	mux.HandleFunc("POST /api/v1/projects/{id}/git/merge", authMiddleware(orgScopeMiddleware(projectAccessMiddleware(workspace.GitMergeHandler))))
	mux.HandleFunc("OPTIONS /api/v1/projects/{id}/git/merge", preflightHandler)

	// VCS Worktree Status endpoint
	mux.HandleFunc("GET /api/v1/projects/{id}/git/local-changes", authMiddleware(orgScopeMiddleware(projectAccessMiddleware(workspace.GitStatusHandler))))
	mux.HandleFunc("OPTIONS /api/v1/projects/{id}/git/local-changes", preflightHandler)

	// VCS Preview Diff & Mergeability endpoint
	mux.HandleFunc("GET /api/v1/projects/{id}/git/preview", authMiddleware(orgScopeMiddleware(projectAccessMiddleware(workspace.GitPreviewHandler))))
	mux.HandleFunc("OPTIONS /api/v1/projects/{id}/git/preview", preflightHandler)

	// Local Commit / Approve endpoint
	mux.HandleFunc("POST /api/v1/projects/{id}/git/commit", authMiddleware(orgScopeMiddleware(projectAccessMiddleware(workspace.GitCommitHandler))))
	mux.HandleFunc("OPTIONS /api/v1/projects/{id}/git/commit", preflightHandler)
	mux.HandleFunc("POST /api/v1/projects/{id}/git/approve", authMiddleware(orgScopeMiddleware(projectAccessMiddleware(workspace.GitCommitHandler))))
	mux.HandleFunc("OPTIONS /api/v1/projects/{id}/git/approve", preflightHandler)


	// Workspace health check and gate recalculation
	mux.HandleFunc("POST /api/v1/projects/{id}/health-check", authMiddleware(orgScopeMiddleware(projectAccessMiddleware(workspace.ProjectHealthCheckHandler))))
	mux.HandleFunc("OPTIONS /api/v1/projects/{id}/health-check", preflightHandler)

	mux.HandleFunc("POST /api/v1/projects/{id}/inject-skills", authMiddleware(orgScopeMiddleware(projectAccessMiddleware(workspace.InjectSkillsHandler))))
	mux.HandleFunc("OPTIONS /api/v1/projects/{id}/inject-skills", preflightHandler)

	mux.HandleFunc("POST /api/v1/projects/{id}/artifacts/sync", authMiddleware(orgScopeMiddleware(projectAccessMiddleware(workspace.SyncArtifactsHandler))))
	mux.HandleFunc("OPTIONS /api/v1/projects/{id}/artifacts/sync", preflightHandler)

	mux.HandleFunc("GET /api/v1/projects/{id}/artifacts", authMiddleware(orgScopeMiddleware(projectAccessMiddleware(workspace.ListArtifactsHandler))))
	mux.HandleFunc("OPTIONS /api/v1/projects/{id}/artifacts", preflightHandler)

	// File Explorer tree endpoint
	mux.HandleFunc("GET /api/v1/projects/{id}/files", authMiddleware(orgScopeMiddleware(projectAccessMiddleware(workspace.GetProjectFileTreeHandler))))
	mux.HandleFunc("OPTIONS /api/v1/projects/{id}/files", preflightHandler)
	mux.HandleFunc("POST /api/v1/projects/{id}/files/upload", authMiddleware(orgScopeMiddleware(projectAccessMiddleware(workspace.UploadProjectFilesHandler))))
	mux.HandleFunc("OPTIONS /api/v1/projects/{id}/files/upload", preflightHandler)
	mux.HandleFunc("DELETE /api/v1/projects/{id}/folders", authMiddleware(orgScopeMiddleware(projectAccessMiddleware(workspace.DeleteProjectFolderHandler))))
	mux.HandleFunc("OPTIONS /api/v1/projects/{id}/folders", preflightHandler)
	mux.HandleFunc("PUT /api/v1/projects/{id}/artifacts/{artifactId}", authMiddleware(orgScopeMiddleware(projectAccessMiddleware(workspace.UpdateArtifactHandler))))
	mux.HandleFunc("DELETE /api/v1/projects/{id}/artifacts/{artifactId}", authMiddleware(orgScopeMiddleware(projectAccessMiddleware(workspace.DeleteArtifactHandler))))
	mux.HandleFunc("GET /api/v1/projects/{id}/artifacts/{artifactId}/content", authMiddleware(orgScopeMiddleware(projectAccessMiddleware(workspace.GetArtifactContentHandler))))
	mux.HandleFunc("PUT /api/v1/projects/{id}/artifacts/{artifactId}/content", authMiddleware(orgScopeMiddleware(projectAccessMiddleware(workspace.SaveArtifactContentHandler))))
	mux.HandleFunc("POST /api/v1/projects/{id}/artifacts/{artifactId}/content", authMiddleware(orgScopeMiddleware(projectAccessMiddleware(workspace.SaveArtifactContentHandler))))
	mux.HandleFunc("OPTIONS /api/v1/projects/{id}/artifacts/{artifactId}/content", preflightHandler)

	// Project Workflow Runs endpoints
	mux.HandleFunc("GET /api/v1/projects/{id}/workflow-runs", authMiddleware(orgScopeMiddleware(projectAccessMiddleware(workflow.ListWorkflowRunsHandler))))
	mux.HandleFunc("DELETE /api/v1/projects/{id}/workflow-runs/{runId}", authMiddleware(orgScopeMiddleware(projectAccessMiddleware(workflow.DeleteWorkflowRunHandler))))
	mux.HandleFunc("OPTIONS /api/v1/projects/{id}/workflow-runs", preflightHandler)
	mux.HandleFunc("OPTIONS /api/v1/projects/{id}/workflow-runs/{runId}", preflightHandler)

	// Workflows resource endpoints
	mux.HandleFunc("GET /api/v1/workflows", authMiddleware(workflow.ListWorkflowsHandler))
	mux.HandleFunc("GET /api/v1/workflows/{id}", authMiddleware(workflow.GetWorkflowHandler))
	mux.HandleFunc("POST /api/v1/workflows", authMiddleware(workflow.CreateWorkflowHandler))
	mux.HandleFunc("PUT /api/v1/workflows/{id}", authMiddleware(workflow.UpdateWorkflowHandler))
	mux.HandleFunc("DELETE /api/v1/workflows/{id}", authMiddleware(workflow.DeleteWorkflowHandler))
	mux.HandleFunc("POST /api/v1/workflows/{id}/run", authMiddleware(orgScopeMiddleware(workflow.RunWorkflowBufferedHandler)))
	mux.HandleFunc("OPTIONS /api/v1/workflows", preflightHandler)
	mux.HandleFunc("OPTIONS /api/v1/workflows/{id}", preflightHandler)
	mux.HandleFunc("OPTIONS /api/v1/workflows/{id}/run", preflightHandler)
	mux.HandleFunc("POST /api/v1/workflows/{id}/runs/{runId}/resume", authMiddleware(orgScopeMiddleware(workflow.ResumeWorkflowHandler)))
	mux.HandleFunc("DELETE /api/v1/workflows/{id}/runs/{runId}", authMiddleware(orgScopeMiddleware(workflow.DeleteWorkflowRunByWorkflowHandler)))
	mux.HandleFunc("OPTIONS /api/v1/workflows/{id}/runs/{runId}/resume", preflightHandler)
	mux.HandleFunc("OPTIONS /api/v1/workflows/{id}/runs/{runId}", preflightHandler)

	// Refactored Centralized User Preferences endpoints
	mux.HandleFunc("GET /api/v1/users/{id}/preferences", authMiddleware(orgScopeMiddleware(getUserPreferencesHandler)))
	mux.HandleFunc("PUT /api/v1/users/{id}/preferences", authMiddleware(orgScopeMiddleware(updateUserPreferencesHandler)))
	mux.HandleFunc("OPTIONS /api/v1/users/{id}/preferences", preflightHandler)

	// Onboarding Tour Status endpoints
	mux.HandleFunc("GET /api/v1/users/{id}/tour-status", authMiddleware(orgScopeMiddleware(getUserTourStatusHandler)))
	mux.HandleFunc("PUT /api/v1/users/{id}/tour-status", authMiddleware(orgScopeMiddleware(updateUserTourStatusHandler)))
	mux.HandleFunc("OPTIONS /api/v1/users/{id}/tour-status", preflightHandler)

	// Refactored Centralized User App Credentials subcollection endpoints
	mux.HandleFunc("GET /api/v1/users/{id}/apps/{appId}/credentials", authMiddleware(orgScopeMiddleware(getAppCredentialHandler)))
	mux.HandleFunc("PUT /api/v1/users/{id}/apps/{appId}/credentials", authMiddleware(orgScopeMiddleware(saveAppCredentialHandler)))
	mux.HandleFunc("DELETE /api/v1/users/{id}/apps/{appId}/credentials", authMiddleware(orgScopeMiddleware(deleteAppCredentialHandler)))
	mux.HandleFunc("POST /api/v1/users/{id}/apps/{appId}/credentials/validate", authMiddleware(orgScopeMiddleware(validateAppCredentialHandler)))
	mux.HandleFunc("OPTIONS /api/v1/users/{id}/apps/{appId}/credentials", preflightHandler)
	mux.HandleFunc("OPTIONS /api/v1/users/{id}/apps/{appId}/credentials/validate", preflightHandler)

	// Refactored Centralized User Project Credentials endpoints
	mux.HandleFunc("GET /api/v1/users/{userId}/projects/{projectId}/apps/{appId}/credentials", authMiddleware(orgScopeMiddleware(getProjectAppSettingHandler)))
	mux.HandleFunc("PUT /api/v1/users/{userId}/projects/{projectId}/apps/{appId}/credentials", authMiddleware(orgScopeMiddleware(saveProjectAppSettingHandler)))
	mux.HandleFunc("PUT /api/v1/users/{userId}/projects/{projectId}/apps/credentials", authMiddleware(orgScopeMiddleware(saveBulkProjectAppSettingsHandler)))
	mux.HandleFunc("OPTIONS /api/v1/users/{userId}/projects/{projectId}/apps/{appId}/credentials", preflightHandler)
	mux.HandleFunc("OPTIONS /api/v1/users/{userId}/projects/{projectId}/apps/credentials", preflightHandler)

	// Administrative API endpoints
	mux.HandleFunc("PUT /api/v1/admin/organizations/{orgId}/config", authMiddleware(orgPermissionMiddleware(core.PermissionOrgUpdate)(adminHandler.UpdateOrganizationConfig)))
	mux.HandleFunc("POST /api/v1/admin/organizations/{orgId}/users", authMiddleware(orgPermissionMiddleware(core.PermissionUserCreate)(adminHandler.AddUser)))
	mux.HandleFunc("GET /api/v1/admin/organizations/{orgId}/users", authMiddleware(orgPermissionMiddleware(core.PermissionUserList)(adminHandler.ListOrganizationUsers)))
	mux.HandleFunc("DELETE /api/v1/admin/organizations/{orgId}/users/{userId}", authMiddleware(orgPermissionMiddleware(core.PermissionUserDelete)(adminHandler.RemoveUser)))
	mux.HandleFunc("PUT /api/v1/admin/organizations/{orgId}/users/{userId}/role", authMiddleware(orgPermissionMiddleware(core.PermissionUserUpdate)(adminHandler.UpdateUserRole)))
	mux.HandleFunc("GET /api/v1/admin/organizations/{orgId}/roles", authMiddleware(orgPermissionMiddleware(core.PermissionRoleList)(adminHandler.ListRoles)))

	// App Integrations enablement control (Admin only)
	mux.HandleFunc("POST /api/v1/admin/organizations/{orgId}/config/app-integrations/{id}/enable", authMiddleware(orgPermissionMiddleware(core.PermissionOrgUpdate)(EnableAppIntegrationHandler)))
	mux.HandleFunc("POST /api/v1/admin/organizations/{orgId}/config/app-integrations/{id}/disable", authMiddleware(orgPermissionMiddleware(core.PermissionOrgUpdate)(DisableAppIntegrationHandler)))

	// Super Admin Autojoin Rules Management
	mux.HandleFunc("GET /api/v1/admin/autojoin", authMiddleware(adminHandler.ListAutojoinRules))
	mux.HandleFunc("POST /api/v1/admin/autojoin", authMiddleware(adminHandler.CreateAutojoinRule))
	mux.HandleFunc("PUT /api/v1/admin/autojoin/{id}", authMiddleware(adminHandler.UpdateAutojoinRule))
	mux.HandleFunc("DELETE /api/v1/admin/autojoin/{id}", authMiddleware(adminHandler.DeleteAutojoinRule))
	mux.HandleFunc("OPTIONS /api/v1/admin/autojoin", preflightHandler)
	mux.HandleFunc("OPTIONS /api/v1/admin/autojoin/{id}", preflightHandler)
	mux.HandleFunc("GET /api/v1/admin/organizations", authMiddleware(adminHandler.ListOrganizations))
	mux.HandleFunc("OPTIONS /api/v1/admin/organizations", preflightHandler)
	mux.HandleFunc("GET /api/v1/admin/roles", authMiddleware(adminHandler.ListPlatformRoles))
	mux.HandleFunc("OPTIONS /api/v1/admin/roles", preflightHandler)

	mux.HandleFunc("GET /api/v1/admin/system-info", authMiddleware(adminHandler.GetSystemInfo))
	mux.HandleFunc("OPTIONS /api/v1/admin/system-info", preflightHandler)

	mux.HandleFunc("GET /api/v1/admin/organizations/{orgId}", authMiddleware(adminHandler.GetOrganization))
	mux.HandleFunc("PUT /api/v1/admin/organizations/{orgId}", authMiddleware(adminHandler.UpdateOrganization))
	mux.HandleFunc("DELETE /api/v1/admin/organizations/{orgId}", authMiddleware(adminHandler.DeleteOrganization))
	mux.HandleFunc("OPTIONS /api/v1/admin/organizations/{orgId}", preflightHandler)

	mux.HandleFunc("OPTIONS /api/v1/admin/organizations/{orgId}/config", preflightHandler)
	mux.HandleFunc("OPTIONS /api/v1/admin/organizations/{orgId}/users", preflightHandler)
	mux.HandleFunc("OPTIONS /api/v1/admin/organizations/{orgId}/users/{userId}", preflightHandler)
	mux.HandleFunc("OPTIONS /api/v1/admin/organizations/{orgId}/users/{userId}/role", preflightHandler)
	mux.HandleFunc("OPTIONS /api/v1/admin/organizations/{orgId}/roles", preflightHandler)
	mux.HandleFunc("OPTIONS /api/v1/admin/organizations/{orgId}/config/app-integrations/{id}/enable", preflightHandler)
	mux.HandleFunc("OPTIONS /api/v1/admin/organizations/{orgId}/config/app-integrations/{id}/disable", preflightHandler)

	// Super Admin NFS Volume & Folder Metadata Management
	mux.HandleFunc("GET /api/v1/admin/nfs/volumes", authMiddleware(superAdminAuthMiddleware(adminHandler.GetNFSVolumeStatus)))
	mux.HandleFunc("OPTIONS /api/v1/admin/nfs/volumes", preflightHandler)
	mux.HandleFunc("GET /api/v1/admin/nfs/folders", authMiddleware(superAdminAuthMiddleware(adminHandler.ListNFSFolderMetadata)))
	mux.HandleFunc("OPTIONS /api/v1/admin/nfs/folders", preflightHandler)
	mux.HandleFunc("PUT /api/v1/admin/nfs/folders/{folderId}", authMiddleware(superAdminAuthMiddleware(adminHandler.UpdateNFSFolderMetadata)))
	mux.HandleFunc("OPTIONS /api/v1/admin/nfs/folders/{folderId}", preflightHandler)
	mux.HandleFunc("POST /api/v1/admin/nfs/audit", authMiddleware(superAdminAuthMiddleware(adminHandler.TriggerNFSAudit)))
	mux.HandleFunc("OPTIONS /api/v1/admin/nfs/audit", preflightHandler)

	// Hanging Projects Discovery & Cleanup
	mux.HandleFunc("GET /api/v1/admin/nfs/hanging-projects", authMiddleware(superAdminAuthMiddleware(adminHandler.GetHangingProjects)))
	mux.HandleFunc("OPTIONS /api/v1/admin/nfs/hanging-projects", preflightHandler)
	mux.HandleFunc("POST /api/v1/admin/nfs/hanging-projects/delete", authMiddleware(superAdminAuthMiddleware(adminHandler.DeleteHangingProjects)))
	mux.HandleFunc("OPTIONS /api/v1/admin/nfs/hanging-projects/delete", preflightHandler)

	// Super Admin Jobs Catalog & Cloud Run Execution History (Story 12.13)
	mux.HandleFunc("GET /api/v1/admin/jobs", authMiddleware(superAdminAuthMiddleware(adminHandler.ListJobCatalog)))
	mux.HandleFunc("OPTIONS /api/v1/admin/jobs", preflightHandler)
	mux.HandleFunc("GET /api/v1/admin/jobs/executions", authMiddleware(superAdminAuthMiddleware(adminHandler.ListJobExecutions)))
	mux.HandleFunc("OPTIONS /api/v1/admin/jobs/executions", preflightHandler)

	// Super Admin On-Demand Job Execution (Story 12.12)
	mux.HandleFunc("POST /api/v1/admin/jobs/{jobName}/run", authMiddleware(superAdminAuthMiddleware(adminHandler.RunCloudRunJob)))
	mux.HandleFunc("OPTIONS /api/v1/admin/jobs/{jobName}/run", preflightHandler)

	// Configurations list endpoint (wrapped in orgScopeMiddleware to resolve active org state)
	mux.HandleFunc("GET /api/v1/config/app-integrations", authMiddleware(orgScopeMiddleware(getAppIntegrationsHandler)))
	mux.HandleFunc("OPTIONS /api/v1/config/app-integrations", preflightHandler)

	// Application-Level Feature Flag endpoints (Story 15.2)
	mux.HandleFunc("GET /api/v1/features/{featureId}", authMiddleware(featureHandler.GetFeatureHandler))
	mux.HandleFunc("OPTIONS /api/v1/features/{featureId}", preflightHandler)

	// Admin Feature Flag Management
	mux.HandleFunc("GET /api/v1/admin/features", authMiddleware(superAdminAuthMiddleware(featureHandler.ListFeaturesHandler)))
	mux.HandleFunc("PUT /api/v1/admin/features/{featureId}", authMiddleware(superAdminAuthMiddleware(featureHandler.SaveFeatureHandler)))
	mux.HandleFunc("OPTIONS /api/v1/admin/features", preflightHandler)
	mux.HandleFunc("OPTIONS /api/v1/admin/features/{featureId}", preflightHandler)

	// Feedback collection endpoint (Story 15.3)
	mux.HandleFunc("POST /api/v1/feedback", authMiddleware(feedbackHandler.CreateFeedbackHandler))
	mux.HandleFunc("OPTIONS /api/v1/feedback", preflightHandler)

	// Telemetry Usage Aggregation endpoint (Story 5.4)
	if telemetryHandler != nil {
		mux.HandleFunc("GET /api/v1/telemetry/usage", authMiddleware(telemetryHandler.GetTelemetryUsage))
		mux.HandleFunc("OPTIONS /api/v1/telemetry/usage", preflightHandler)
	}
}
