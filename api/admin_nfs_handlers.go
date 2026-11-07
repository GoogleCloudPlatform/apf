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
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"google.com/apf/api/core"
	"google.com/apf/api/db"
	"google.com/apf/api/jobs"
)

// WithNFSStore registers the NFS database store on the AdminHandler instance.
func (h *AdminHandler) WithNFSStore(store db.NFSStore) *AdminHandler {
	h.NFSStore = store
	return h
}

// superAdminAuthMiddleware ensures that only authenticated Super Administrators can access the endpoint.
func superAdminAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
		if !ok || claims == nil || !claims.IsSuperAdmin {
			sendJSON(w, http.StatusForbidden, Response{
				Message: "Forbidden: Super Administrator access required",
				Status:  "ERROR",
			})
			return
		}
		next(w, r)
	}
}

// GetNFSVolumeStatus handles GET /api/v1/admin/nfs/volumes
func (h *AdminHandler) GetNFSVolumeStatus(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || !claims.IsSuperAdmin {
		sendJSON(w, http.StatusForbidden, Response{Message: "Forbidden: Super Administrator access required", Status: "ERROR"})
		return
	}

	if h.NFSStore == nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: "NFSStore is not initialized", Status: "ERROR"})
		return
	}

	mountPath := r.URL.Query().Get("mountPath")
	if mountPath == "" && h.Config != nil && h.Config.WorkspaceMountPath != "" {
		mountPath = h.Config.WorkspaceMountPath
	}

	volumeStats, err := h.NFSStore.GetNFSVolumeStats(r.Context(), mountPath)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to get NFS volume stats: %v", err), Status: "ERROR"})
		return
	}

	folders, err := h.NFSStore.ListFolderMetadata(r.Context(), core.NFSFolderFilter{})
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to list folder metadata: %v", err), Status: "ERROR"})
		return
	}

	sendJSON(w, http.StatusOK, map[string]interface{}{
		"volume":       volumeStats,
		"totalFolders": len(folders),
		"status":       "SUCCESS",
	})
}

// ListNFSFolderMetadata handles GET /api/v1/admin/nfs/folders
func (h *AdminHandler) ListNFSFolderMetadata(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || !claims.IsSuperAdmin {
		sendJSON(w, http.StatusForbidden, Response{Message: "Forbidden: Super Administrator access required", Status: "ERROR"})
		return
	}

	if h.NFSStore == nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: "NFSStore is not initialized", Status: "ERROR"})
		return
	}

	filter := core.NFSFolderFilter{
		OrganizationID: r.URL.Query().Get("organizationId"),
		ProjectID:      r.URL.Query().Get("projectId"),
	}
	if filter.OrganizationID == "" {
		filter.OrganizationID = r.URL.Query().Get("orgId")
	}

	folders, err := h.NFSStore.ListFolderMetadata(r.Context(), filter)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to list folder metadata: %v", err), Status: "ERROR"})
		return
	}

	if folders == nil {
		folders = []core.NFSFolderMetadata{}
	}

	sendJSON(w, http.StatusOK, map[string]interface{}{
		"folders": folders,
		"total":   len(folders),
		"status":  "SUCCESS",
	})
}

// UpdateNFSFolderMetadata handles PUT /api/v1/admin/nfs/folders/{folderId}
func (h *AdminHandler) UpdateNFSFolderMetadata(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || !claims.IsSuperAdmin {
		sendJSON(w, http.StatusForbidden, Response{Message: "Forbidden: Super Administrator access required", Status: "ERROR"})
		return
	}

	if h.NFSStore == nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: "NFSStore is not initialized", Status: "ERROR"})
		return
	}

	folderID := r.PathValue("folderId")
	if folderID == "" {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Missing folder ID", Status: "ERROR"})
		return
	}

	var updates core.NFSFolderMetadataUpdate
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Invalid request payload", Status: "ERROR"})
		return
	}

	updated, err := h.NFSStore.UpdateFolderMetadata(r.Context(), folderID, updates)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to update folder metadata: %v", err), Status: "ERROR"})
		return
	}

	sendJSON(w, http.StatusOK, updated)
}

// TriggerNFSAudit handles POST /api/v1/admin/nfs/audit
func (h *AdminHandler) TriggerNFSAudit(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || !claims.IsSuperAdmin {
		sendJSON(w, http.StatusForbidden, Response{Message: "Forbidden: Super Administrator access required", Status: "ERROR"})
		return
	}

	if h.NFSStore == nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: "NFSStore is not initialized", Status: "ERROR"})
		return
	}

	auditedCount, err := RunNFSAuditScan(r.Context(), h.Config, h.NFSStore)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("NFS storage audit failed: %v", err), Status: "ERROR"})
		return
	}

	sendJSON(w, http.StatusOK, map[string]interface{}{
		"status":         "SUCCESS",
		"message":        "NFS storage audit completed successfully",
		"foldersAudited": auditedCount,
	})
}

// HangingProjectItem represents a project folder on disk that has no matching user-project record in the database.
type HangingProjectItem = jobs.HangingProjectItem

// DeleteHangingProjectsRequest holds payload for deleting discovered hanging projects.
type DeleteHangingProjectsRequest struct {
	IndexKeys []string `json:"indexKeys,omitempty"`
	Paths     []string `json:"paths,omitempty"`
}

// DiscoverHangingProjects scans filesystem at orgid/userid/projectid (index 1) and diffs against DB (index 2).
func DiscoverHangingProjects(ctx context.Context, mountPath string, orgStore db.OrganizationStore) ([]HangingProjectItem, error) {
	return jobs.DiscoverHangingProjects(ctx, mountPath, orgStore, db.ProjectStoreInstance, FirestoreClient)
}

// GetHangingProjects handles GET /api/v1/admin/nfs/hanging-projects
func (h *AdminHandler) GetHangingProjects(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || !claims.IsSuperAdmin {
		sendJSON(w, http.StatusForbidden, Response{Message: "Forbidden: Super Administrator access required", Status: "ERROR"})
		return
	}

	mountPath := "~/workspaces"
	if h.Config != nil && h.Config.WorkspaceMountPath != "" {
		mountPath = h.Config.WorkspaceMountPath
	}

	hanging, err := DiscoverHangingProjects(r.Context(), mountPath, h.OrganizationStore)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to discover hanging projects: %v", err), Status: "ERROR"})
		return
	}

	if hanging == nil {
		hanging = []HangingProjectItem{}
	}

	sendJSON(w, http.StatusOK, map[string]interface{}{
		"status":          "SUCCESS",
		"hangingProjects": hanging,
		"total":           len(hanging),
	})
}

// DeleteHangingProjects handles POST /api/v1/admin/nfs/hanging-projects/delete
func (h *AdminHandler) DeleteHangingProjects(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || !claims.IsSuperAdmin {
		sendJSON(w, http.StatusForbidden, Response{Message: "Forbidden: Super Administrator access required", Status: "ERROR"})
		return
	}

	var req DeleteHangingProjectsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Invalid request payload", Status: "ERROR"})
		return
	}

	if len(req.IndexKeys) == 0 && len(req.Paths) == 0 {
		sendJSON(w, http.StatusBadRequest, Response{Message: "No hanging projects specified for deletion", Status: "ERROR"})
		return
	}

	mountPath := "~/workspaces"
	if h.Config != nil && h.Config.WorkspaceMountPath != "" {
		mountPath = h.Config.WorkspaceMountPath
	}
	if strings.HasPrefix(mountPath, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			mountPath = filepath.Join(home, mountPath[2:])
		}
	}
	cleanMount := filepath.Clean(mountPath)

	baseDir := cleanMount
	if info, err := os.Stat(filepath.Join(cleanMount, "organizations")); err == nil && info.IsDir() {
		baseDir = filepath.Join(cleanMount, "organizations")
	}

	deletedCount := 0
	var freedBytes int64
	var errMsgs []string

	pathsToDelete := make(map[string]bool)
	for _, p := range req.Paths {
		if p != "" {
			pathsToDelete[filepath.Clean(p)] = true
		}
	}

	// Build map from discovered hanging projects to handle index keys with hyphens (e.g. UUIDs)
	discovered, _ := DiscoverHangingProjects(r.Context(), mountPath, h.OrganizationStore)
	discoveredMap := make(map[string]string)
	for _, item := range discovered {
		discoveredMap[item.IndexKey] = item.Path
	}

	for _, key := range req.IndexKeys {
		if path, found := discoveredMap[key]; found {
			pathsToDelete[filepath.Clean(path)] = true
		} else {
			parts := strings.Split(key, "-")
			if len(parts) == 3 && parts[0] != "" && parts[1] != "" && parts[2] != "" {
				pathsToDelete[filepath.Clean(filepath.Join(baseDir, parts[0], parts[1], parts[2]))] = true
			}
		}
	}

	for targetPath := range pathsToDelete {
		rel, err := filepath.Rel(baseDir, targetPath)
		if err != nil || strings.HasPrefix(rel, "..") {
			slog.Warn("Failed to delete hanging project: path is outside base workspace directory", "path", targetPath, "baseDir", baseDir, "error", err)
			errMsgs = append(errMsgs, fmt.Sprintf("Path %s is outside base workspace directory %s", targetPath, baseDir))
			continue
		}

		relParts := strings.Split(filepath.ToSlash(rel), "/")
		if len(relParts) != 3 || relParts[0] == "" || relParts[1] == "" || relParts[2] == "" {
			slog.Warn("Refusing to delete hanging project: path must be at project level (org/user/project)", "path", targetPath)
			errMsgs = append(errMsgs, fmt.Sprintf("Refusing to delete %s: path must be at project level (org/user/project)", targetPath))
			continue
		}

		lockPath := targetPath + ".lock"
		size, _, _ := ScanWorkspaceDirectory(targetPath)
		if lockInfo, err := os.Stat(lockPath); err == nil {
			if lockInfo.IsDir() {
				lSize, _, _ := ScanWorkspaceDirectory(lockPath)
				size += lSize
			} else {
				size += lockInfo.Size()
			}
		}

		if err := os.RemoveAll(targetPath); err != nil {
			slog.Warn("Failed to remove hanging project directory", "path", targetPath, "error", err)
			errMsgs = append(errMsgs, fmt.Sprintf("Failed to remove directory %s: %v", targetPath, err))
		} else {
			deletedCount++
			freedBytes += size
			if lockErr := os.RemoveAll(lockPath); lockErr != nil && !os.IsNotExist(lockErr) {
				slog.Warn("Failed to remove hanging project lock path", "path", lockPath, "error", lockErr)
			}
		}
	}

	if len(errMsgs) > 0 {
		slog.Warn("Hanging projects delete operation encountered errors", "deletedCount", deletedCount, "failedCount", len(errMsgs), "errors", errMsgs)
	}

	status := "SUCCESS"
	if len(errMsgs) > 0 && deletedCount == 0 {
		status = "ERROR"
	} else if len(errMsgs) > 0 {
		status = "WARNING"
	}

	msg := fmt.Sprintf("Successfully deleted %d hanging project folder(s)", deletedCount)
	if len(errMsgs) > 0 {
		if deletedCount > 0 {
			msg = fmt.Sprintf("Deleted %d hanging project folder(s), but failed to delete %d folder(s)", deletedCount, len(errMsgs))
		} else {
			msg = fmt.Sprintf("Failed to delete %d hanging project folder(s)", len(errMsgs))
		}
	}

	sendJSON(w, http.StatusOK, map[string]interface{}{
		"status":       status,
		"message":      msg,
		"deletedCount": deletedCount,
		"freedBytes":   freedBytes,
		"errors":       errMsgs,
	})
}
