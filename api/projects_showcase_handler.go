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
	"log"
	"net/http"
	"strings"

	"google.com/apf/api/core"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func provisionShowcaseHandler(w http.ResponseWriter, r *http.Request) {
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

	userId := claims.Subject

	// Check if already provisioned using the fixed ID
	showcaseID := "foundry-showcase-project"
	p, err := store.GetProject(r.Context(), organizationID, showcaseID)
	if err == nil && p != nil {
		// Already provisioned in DB. Ensure requesting user is added as an owner/member.
		needUpdate := false
		if p.Members == nil {
			m := make(map[string]string)
			p.Members = &m
			needUpdate = true
		}
		if role, ok := (*p.Members)[userId]; !ok || role == "" {
			(*p.Members)[userId] = "owner"
			needUpdate = true
		}
		if needUpdate {
			if errUp := store.UpdateProject(r.Context(), organizationID, p); errUp != nil {
				log.Printf("[Error] Failed to update showcase project membership for user %s: %v", userId, errUp)
				sendJSON(w, http.StatusInternalServerError, Response{
					Message: "Failed to update showcase project membership",
					Status:  "ERROR",
				})
				return
			}
		}

		// Ensure workspace spec files exist on disk for this user
		if errFiles := extractShowcaseWorkspaceFiles(organizationID, userId, showcaseID); errFiles != nil {
			log.Printf("[Warning] Failed to extract showcase workspace files for user %s: %v", userId, errFiles)
		}

		// Ensure showcase artifact records in store match showcase template artifacts
		if errArts := ensureShowcaseArtifactsInStore(r.Context(), organizationID, userId, showcaseID); errArts != nil {
			log.Printf("[Warning] Failed to ensure showcase artifacts in store for user %s: %v", userId, errArts)
		}

		// Already provisioned, return updated project safely (idempotent)
		sendJSON(w, http.StatusOK, p)
		return
	}

	// If it failed with a real DB error (not a simple NotFound), we must fail and report it
	if err != nil && status.Code(err) != codes.NotFound {
		log.Printf("[Error] Failed to check showcase pre-existence for user %s: %v", userId, err)
		sendJSON(w, http.StatusInternalServerError, Response{
			Message: "Database error checking showcase status",
			Status:  "ERROR",
		})
		return
	}

	// Provision showcase
	p, err = seedShowcase(r.Context(), organizationID, userId, showcaseID)
	if err != nil {
		log.Printf("[Error] Failed to provision showcase for user %s in org %s: %v", userId, organizationID, err)
		sendJSON(w, http.StatusInternalServerError, Response{
			Message: "Failed to provision showcase project",
			Status:  "ERROR",
		})
		return
	}

	sendJSON(w, http.StatusCreated, p)
}

func resetShowcaseHandler(w http.ResponseWriter, r *http.Request) {
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

	projectId := r.PathValue("id")
	userId := claims.Subject

	// Security Guard 1: Verify the project is actually a showcase project by fixed ID
	if projectId != "foundry-showcase-project" {
		sendJSON(w, http.StatusBadRequest, Response{
			Message: "Access Denied: Reset is only permitted for the showcase project",
			Status:  "ERROR",
		})
		return
	}

	// Security Guard 2: Prevent path traversal attacks
	if strings.Contains(projectId, "..") || strings.Contains(projectId, "/") || strings.Contains(projectId, "\\") {
		sendJSON(w, http.StatusBadRequest, Response{
			Message: "Access Denied: Invalid project ID structure",
			Status:  "ERROR",
		})
		return
	}

	// Reset showcase
	err := ResetShowcase(r.Context(), organizationID, userId, projectId)
	if err != nil {
		log.Printf("[Error] Failed to reset showcase project %s for user %s: %v", projectId, userId, err)
		sendJSON(w, http.StatusInternalServerError, Response{
			Message: "Failed to restore showcase project to default state",
			Status:  "ERROR",
		})
		return
	}

	sendJSON(w, http.StatusOK, Response{Message: "Showcase project restored to default state", Status: "SUCCESS"})
}
