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
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"google.com/apf/api/core"
	"google.com/apf/api/db"
)

// CreateOrganizationRequest defines the payload for creating a new organization.
type CreateOrganizationRequest struct {
	Name       string `json:"name"`
	OwnerEmail string `json:"ownerEmail"`
}

// CreateOrganizationResponse defines the response payload for a successfully created organization.
type CreateOrganizationResponse struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	OwnerUID string `json:"ownerUid"`
}

func createOrganizationHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || claims == nil {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized context", Status: "ERROR"})
		return
	}

	// Verify the user is a super admin
	if !claims.IsSuperAdmin {
		sendJSON(w, http.StatusForbidden, Response{Message: "Forbidden: Super Admin role required", Status: "ERROR"})
		return
	}

	// Limit request body size to 1MB
	r.Body = http.MaxBytesReader(w, r.Body, 1048576)

	var req CreateOrganizationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Invalid request payload", Status: "ERROR"})
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Organization name cannot be empty", Status: "ERROR"})
		return
	}

	req.OwnerEmail = strings.TrimSpace(req.OwnerEmail)
	if req.OwnerEmail == "" || !strings.Contains(req.OwnerEmail, "@") {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Valid owner email is required", Status: "ERROR"})
		return
	}

	// Check if organization name already exists
	existing, err := db.OrganizationStoreInstance.GetOrganizationByName(r.Context(), req.Name)
	if err == nil && existing != nil {
		sendJSON(w, http.StatusConflict, Response{Message: fmt.Sprintf("Organization name '%s' already exists", req.Name), Status: "ERROR"})
		return
	}
	// If error is not ErrNotFound, it means a database error occurred
	if err != nil && err != db.ErrNotFound {
		log.Printf("ERROR: Database error checking organization name: %v", err)
		sendJSON(w, http.StatusInternalServerError, Response{Message: "Internal server error", Status: "ERROR"})
		return
	}

	var (
		authOrgCreated     = false
		dbOrgCreated       = false
		ownerCreatedInAuth = false
		tenantID           string
		orgID              string
		ownerUID           string
	)

	// Rollback defers
	defer func() {
		if authOrgCreated && (!dbOrgCreated || !ownerCreatedInAuth || err != nil) {
			log.Printf("ROLLBACK: Deleting Firebase organization tenant %s due to subsequent failures", tenantID)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if rErr := globalOrganizationProvisioner.DeleteOrganization(ctx, tenantID); rErr != nil {
				log.Printf("ROLLBACK_ERROR: Failed to delete Firebase organization tenant %s: %v", tenantID, rErr)
			}
		}
	}()

	defer func() {
		if dbOrgCreated && (!ownerCreatedInAuth || err != nil) {
			log.Printf("ROLLBACK: Deleting Firestore organization doc %s due to subsequent failures", orgID)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if rErr := db.OrganizationStoreInstance.DeleteOrganization(ctx, orgID); rErr != nil {
				log.Printf("ROLLBACK_ERROR: Failed to delete Firestore organization doc %s: %v", orgID, rErr)
			}
			if ownerUID != "" {
				log.Printf("ROLLBACK: Deleting Firestore owner user doc %s/%s due to subsequent failures", orgID, ownerUID)
				if rErr := userStore.DeleteUser(ctx, orgID, ownerUID); rErr != nil {
					log.Printf("ROLLBACK_ERROR: Failed to delete Firestore owner user doc %s/%s: %v", orgID, ownerUID, rErr)
				}
			}
		}
	}()

	// 1. Create Tenant in Firebase Auth
	tenantID, err = globalOrganizationProvisioner.CreateOrganization(r.Context(), req.Name)
	if err != nil {
		log.Printf("ERROR: Failed to create organization in Firebase: %v", err)
		sendJSON(w, http.StatusInternalServerError, Response{Message: "Failed to provision organization in Firebase Auth", Status: "ERROR"})
		return
	}
	authOrgCreated = true

	// 2. Create Organization in Firestore (Generates independent Org ID)
	org := &db.Organization{
		Name:     req.Name,
		TenantID: tenantID,
	}
	orgID, err = db.OrganizationStoreInstance.CreateOrganization(r.Context(), org)
	if err != nil {
		log.Printf("ERROR: Failed to create organization in Firestore: %v", err)
		if errors.Is(err, db.ErrConflict) {
			sendJSON(w, http.StatusConflict, Response{Message: fmt.Sprintf("Organization name '%s' already exists", req.Name), Status: "ERROR"})
		} else {
			sendJSON(w, http.StatusInternalServerError, Response{Message: "Failed to create organization record in database", Status: "ERROR"})
		}
		return
	}
	dbOrgCreated = true

	// 3. Create Owner in Firebase Auth Tenant, passing both Tenant ID and Firestore Org ID
	ownerUID, err = globalOrganizationProvisioner.CreateOrganizationOwner(r.Context(), tenantID, orgID, req.OwnerEmail)
	if err != nil {
		log.Printf("ERROR: Failed to create organization owner in Firebase Auth: %v", err)
		sendJSON(w, http.StatusInternalServerError, Response{Message: "Failed to create organization owner in Firebase Auth", Status: "ERROR"})
		return
	}
	ownerCreatedInAuth = true

	// 4. Create Owner record in Firestore under /organizations/{orgID}/users/{ownerUID}
	ownerUser := &db.User{
		UserID: ownerUID,
		Email:  req.OwnerEmail,
		Role:   db.RoleOrgOwner,
		Preferences: db.UserPreferences{
			Theme: "dark",
		},
	}
	err = userStore.CreateUser(r.Context(), orgID, ownerUser)
	if err != nil {
		log.Printf("ERROR: Failed to create owner user record in Firestore: %v", err)
		sendJSON(w, http.StatusInternalServerError, Response{Message: "Failed to save owner user record in database", Status: "ERROR"})
		return
	}

	resp := CreateOrganizationResponse{
		ID:       orgID,
		Name:     req.Name,
		OwnerUID: ownerUID,
	}
	sendJSON(w, http.StatusCreated, resp)
}
