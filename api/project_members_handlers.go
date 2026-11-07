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
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"google.com/apf/api/core"
	"google.com/apf/api/db"
)

type ProjectMemberResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	IsPending bool   `json:"isPending"`
}

// Helper to check if user is Project Owner or Org Admin
func isProjectOwnerOrOrgAdmin(project *db.Project, claims *core.UserClaims) bool {
	// Org Owner and Admin have rights
	for _, role := range claims.Roles {
		if role == string(db.RoleOrgOwner) || role == string(db.RoleOrgAdmin) {
			return true
		}
	}
	// Project Owner has rights
	if project.OwnerId != nil && *project.OwnerId == claims.Subject {
		return true
	}
	return false
}

// Helper to check if user can manage members (Owner, Org Admin, or Project Editor)
func canManageMembers(project *db.Project, claims *core.UserClaims) bool {
	if isProjectOwnerOrOrgAdmin(project, claims) {
		return true
	}
	// Check if Project Editor
	if project.Members != nil {
		role, ok := (*project.Members)[claims.Subject]
		if ok && (role == "Editor" || role == "project:editor") {
			return true
		}
	}
	return false
}

// GET /api/v1/projects/{id}/members
func listProjectMembersHandler(w http.ResponseWriter, r *http.Request) {
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

	members := []ProjectMemberResponse{}

	// Add Owner first (from project.OwnerId)
	if project.OwnerId != nil {
		ownerUser, err := userStore.GetUser(r.Context(), organizationID, *project.OwnerId)
		if err == nil {
			members = append(members, ProjectMemberResponse{
				ID:    *project.OwnerId,
				Name:  ownerUser.DisplayName,
				Email: ownerUser.Email,
				Role:  "Owner",
			})
		} else {
			members = append(members, ProjectMemberResponse{
				ID:    *project.OwnerId,
				Name:  "Owner",
				Email: "",
				Role:  "Owner",
			})
		}
	}

	// Add other members
	if project.Members != nil {
		for userID, role := range *project.Members {
			// Skip owner if they are also in the members map
			if project.OwnerId != nil && userID == *project.OwnerId {
				continue
			}

			// Map roles to capitalized version for frontend
			displayRole := role
			switch role {
			case "project:editor", "Editor":
				displayRole = "Editor"
			case "project:viewer", "Viewer":
				displayRole = "Viewer"
			case "owner":
				displayRole = "Owner"
			}

			// Check if it's a pending email (contains @)
			if strings.Contains(userID, "@") {
				parts := strings.Split(strings.Split(userID, "@")[0], ".")
				for i, part := range parts {
					if len(part) > 0 {
						parts[i] = strings.ToUpper(part[:1]) + part[1:]
					}
				}
				name := strings.Join(parts, " ")
				members = append(members, ProjectMemberResponse{
					ID:        userID,
					Name:      name,
					Email:     userID,
					Role:      displayRole,
					IsPending: true,
				})
				continue
			}

			// Regular user
			u, err := userStore.GetUser(r.Context(), organizationID, userID)
			if err != nil {
				members = append(members, ProjectMemberResponse{
					ID:    userID,
					Name:  "Unknown User",
					Email: "",
					Role:  displayRole,
				})
				continue
			}

			members = append(members, ProjectMemberResponse{
				ID:    userID,
				Name:  u.DisplayName,
				Email: u.Email,
				Role:  displayRole,
			})
		}
	}

	sendJSON(w, http.StatusOK, members)
}

type AddProjectMembersRequest struct {
	Emails []string `json:"emails"`
	Role   string   `json:"role"`
}

// POST /api/v1/projects/{id}/members
func addProjectMembersHandler(w http.ResponseWriter, r *http.Request) {
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

	// Auth Check
	if !canManageMembers(project, claims) {
		sendJSON(w, http.StatusForbidden, Response{Message: "Forbidden: You do not have permission to manage members", Status: "ERROR"})
		return
	}

	var req AddProjectMembersRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Invalid request payload", Status: "ERROR"})
		return
	}

	req.Role = strings.TrimSpace(req.Role)
	if req.Role != "Editor" && req.Role != "Viewer" {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Invalid role. Only 'Editor' and 'Viewer' are allowed", Status: "ERROR"})
		return
	}

	// Editor Constraint: Editors cannot grant Owner role
	isOwnerOrAdmin := isProjectOwnerOrOrgAdmin(project, claims)
	if !isOwnerOrAdmin && req.Role == "Owner" {
		sendJSON(w, http.StatusForbidden, Response{Message: "Forbidden: Editors cannot grant Owner role", Status: "ERROR"})
		return
	}

	if project.Members == nil {
		m := make(map[string]string)
		project.Members = &m
	}

	addedCount := 0
	var errors []string

	for _, email := range req.Emails {
		email = strings.ToLower(strings.TrimSpace(email))
		if email == "" {
			continue
		}

		// 1. Try to find user in Org
		u, err := userStore.GetUserByEmail(r.Context(), organizationID, email)
		if err == nil {
			(*project.Members)[u.UserID] = req.Role
			addedCount++
			continue
		}

		// 2. User not found, check autojoin rules
		parts := strings.Split(email, "@")
		if len(parts) != 2 {
			errors = append(errors, fmt.Sprintf("Invalid email format: %s", email))
			continue
		}
		domain := parts[1]

		rule, err := autojoinStore.FindActiveRuleByDomain(r.Context(), domain)
		if err == nil && rule != nil && rule.OrganizationID == organizationID {
			(*project.Members)[email] = req.Role
			addedCount++
			log.Printf("INFO: Added pending member %s to project %s via autojoin rule", email, project.ID)
		} else {
			errors = append(errors, fmt.Sprintf("User %s not found in organization and domain is not eligible for autojoin", email))
		}
	}

	if addedCount > 0 {
		if err := store.UpdateProject(r.Context(), organizationID, project); err != nil {
			sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to update project members: %v", err), Status: "ERROR"})
			return
		}
	}

	if len(errors) > 0 && addedCount == 0 {
		sendJSON(w, http.StatusNotFound, Response{Message: strings.Join(errors, "; "), Status: "ERROR"})
		return
	}

	status := "SUCCESS"
	if len(errors) > 0 {
		status = "PARTIAL_SUCCESS"
	}

	sendJSON(w, http.StatusOK, map[string]interface{}{
		"status":     status,
		"addedCount": addedCount,
		"errors":     errors,
	})
}

type UpdateProjectMemberRoleRequest struct {
	Role string `json:"role"`
}

// PUT /api/v1/projects/{id}/members/{memberId}
func updateProjectMemberRoleHandler(w http.ResponseWriter, r *http.Request) {
	projectId := r.PathValue("id")
	memberId := r.PathValue("memberId")
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

	// Auth Check
	if !canManageMembers(project, claims) {
		sendJSON(w, http.StatusForbidden, Response{Message: "Forbidden: You do not have permission to manage members", Status: "ERROR"})
		return
	}

	var req UpdateProjectMemberRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Invalid request payload", Status: "ERROR"})
		return
	}

	req.Role = strings.TrimSpace(req.Role)
	if req.Role != "Editor" && req.Role != "Viewer" {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Invalid role. Only 'Editor' and 'Viewer' are allowed", Status: "ERROR"})
		return
	}

	if project.Members == nil || (*project.Members)[memberId] == "" {
		sendJSON(w, http.StatusNotFound, Response{Message: "Member not found in this project", Status: "ERROR"})
		return
	}

	isOwnerOrAdmin := isProjectOwnerOrOrgAdmin(project, claims)
	if !isOwnerOrAdmin {
		// Editor Constraint: Editors cannot modify Owner
		if project.OwnerId != nil && memberId == *project.OwnerId {
			sendJSON(w, http.StatusForbidden, Response{Message: "Forbidden: Editors cannot modify the Project Owner", Status: "ERROR"})
			return
		}
	}

	(*project.Members)[memberId] = req.Role

	if err := store.UpdateProject(r.Context(), organizationID, project); err != nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to update member role: %v", err), Status: "ERROR"})
		return
	}

	sendJSON(w, http.StatusOK, Response{Message: "Member role updated successfully", Status: "SUCCESS"})
}

// DELETE /api/v1/projects/{id}/members/{memberId}
func removeProjectMemberHandler(w http.ResponseWriter, r *http.Request) {
	projectId := r.PathValue("id")
	memberId := r.PathValue("memberId")
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

	// Auth Check
	if !canManageMembers(project, claims) {
		sendJSON(w, http.StatusForbidden, Response{Message: "Forbidden: You do not have permission to manage members", Status: "ERROR"})
		return
	}

	if project.Members == nil || (*project.Members)[memberId] == "" {
		sendJSON(w, http.StatusNotFound, Response{Message: "Member not found in this project", Status: "ERROR"})
		return
	}

	isOwnerOrAdmin := isProjectOwnerOrOrgAdmin(project, claims)
	if !isOwnerOrAdmin {
		// Editor Constraint: Editors cannot remove Owner
		if project.OwnerId != nil && memberId == *project.OwnerId {
			sendJSON(w, http.StatusForbidden, Response{Message: "Forbidden: Editors cannot remove the Project Owner", Status: "ERROR"})
			return
		}
	}

	delete(*project.Members, memberId)

	if err := store.UpdateProject(r.Context(), organizationID, project); err != nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to remove member: %v", err), Status: "ERROR"})
		return
	}

	sendJSON(w, http.StatusOK, Response{Message: "Member removed successfully", Status: "SUCCESS"})
}
