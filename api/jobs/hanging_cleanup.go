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

package jobs

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"cloud.google.com/go/firestore"
	"google.com/apf/api/core"
	"google.com/apf/api/db"
)

func init() {
	_ = Register(&HangingWorkspaceCleanupJob{})
}

// HangingProjectItem represents a project folder on disk that has no matching project record in the database.
type HangingProjectItem struct {
	IndexKey  string `json:"indexKey"` // orgid-userid-projectid
	OrgID     string `json:"orgId"`
	UserID    string `json:"userId"`
	ProjectID string `json:"projectId"`
	Path      string `json:"path"`
	SizeBytes int64  `json:"sizeBytes"`
	FileCount int64  `json:"fileCount"`
}

// HangingWorkspaceCleanupJob implements the Job interface for discovering and deleting orphaned workspace folders.
type HangingWorkspaceCleanupJob struct {
	DryRun            bool
	MountPath         string
	OrganizationStore db.OrganizationStore
	ProjectStore      db.ProjectStore
	FirestoreClient   *firestore.Client
}

// Name returns the unique job identifier.
func (j *HangingWorkspaceCleanupJob) Name() string {
	return "hanging-cleanup"
}

// Description returns a summary of the job's purpose.
func (j *HangingWorkspaceCleanupJob) Description() string {
	return "Discovers and cleans up orphaned workspace folders and lock files absent in the database"
}

func (j *HangingWorkspaceCleanupJob) isDryRun(ctx context.Context) bool {
	if IsDryRun(ctx) {
		return true
	}
	if j.DryRun {
		return true
	}
	env := strings.ToLower(strings.TrimSpace(os.Getenv("DRY_RUN")))
	return env == "true" || env == "1" || env == "yes"
}

// Run executes the orphaned workspace cleanup job.
func (j *HangingWorkspaceCleanupJob) Run(ctx context.Context, cfg *core.Configuration) error {
	dryRun := j.isDryRun(ctx)

	mountPath := j.MountPath
	if mountPath == "" && cfg != nil && cfg.WorkspaceMountPath != "" {
		mountPath = cfg.WorkspaceMountPath
	}
	if mountPath == "" {
		mountPath = "~/workspaces"
	}
	cleanMount := filepath.Clean(core.ExpandPath(mountPath))

	baseDir := cleanMount
	if info, err := os.Stat(filepath.Join(cleanMount, "organizations")); err == nil && info.IsDir() {
		baseDir = filepath.Join(cleanMount, "organizations")
	}

	var client *firestore.Client
	if j.FirestoreClient != nil {
		client = j.FirestoreClient
	} else if cfg != nil && cfg.GoogleCloudProject != "" {
		var err error
		if cfg.FirestoreDbName != "" {
			client, err = firestore.NewClientWithDatabase(ctx, cfg.GoogleCloudProject, cfg.FirestoreDbName)
		} else {
			client, err = firestore.NewClient(ctx, cfg.GoogleCloudProject)
		}
		if err != nil {
			return fmt.Errorf("failed to initialize firestore client for hanging-cleanup: %w", err)
		}
		if client != nil {
			defer client.Close()
		}
	}

	orgStore := j.OrganizationStore
	if orgStore == nil && client != nil {
		orgStore = db.NewFirestoreOrganizationStore(client)
	}

	projectStore := j.ProjectStore
	if projectStore == nil && client != nil {
		projectStore = db.NewFirestoreProjectStore(client)
	}
	if projectStore == nil {
		projectStore = db.ProjectStoreInstance
	}

	if client == nil && (orgStore == nil || projectStore == nil) {
		return fmt.Errorf("hanging workspace cleanup requires a database connection or configured organization and project stores")
	}

	slog.Info("Starting hanging workspace cleanup job", "job", j.Name(), "dryRun", dryRun, "mountPath", mountPath)

	hanging, err := DiscoverHangingProjects(ctx, mountPath, orgStore, projectStore, client)
	if err != nil {
		slog.Error("Failed to discover hanging projects", "job", j.Name(), "error", err)
		return fmt.Errorf("failed to discover hanging projects: %w", err)
	}

	if dryRun {
		slog.Info("Hanging workspace cleanup running in DRY-RUN mode (no disk deletions will be performed)")
		var totalCandidateBytes int64
		for _, item := range hanging {
			totalCandidateBytes += item.SizeBytes
			slog.Info("Discovered candidate orphaned workspace folder",
				"dryRun", true,
				"indexKey", item.IndexKey,
				"orgId", item.OrgID,
				"userId", item.UserID,
				"projectId", item.ProjectID,
				"path", item.Path,
				"sizeBytes", item.SizeBytes,
				"fileCount", item.FileCount,
			)
		}
		slog.Info("Hanging workspace cleanup dry-run scan completed",
			"candidateCount", len(hanging),
			"totalCandidateBytes", totalCandidateBytes,
		)
		return nil
	}

	deletedCount := 0
	var freedBytes int64

	for _, item := range hanging {
		rel, err := filepath.Rel(baseDir, item.Path)
		if err != nil || strings.HasPrefix(rel, "..") {
			slog.Warn("Refusing to delete hanging workspace: path is outside base workspace directory", "path", item.Path, "baseDir", baseDir)
			continue
		}

		relParts := strings.Split(filepath.ToSlash(rel), "/")
		if len(relParts) != 3 || relParts[0] == "" || relParts[1] == "" || relParts[2] == "" {
			slog.Warn("Refusing to delete hanging workspace: path must be at project level (org/user/project)", "path", item.Path)
			continue
		}

		lockPath := item.Path + ".lock"
		size := item.SizeBytes

		if err := os.RemoveAll(item.Path); err != nil {
			slog.Warn("Failed to remove hanging project directory", "path", item.Path, "error", err)
			continue
		}

		deletedCount++
		freedBytes += size

		if lockErr := os.RemoveAll(lockPath); lockErr != nil && !os.IsNotExist(lockErr) {
			slog.Warn("Failed to remove hanging project lock path", "path", lockPath, "error", lockErr)
		}

		slog.Info("Removed orphaned workspace directory",
			"indexKey", item.IndexKey,
			"path", item.Path,
			"freedBytes", size,
		)
	}

	slog.Info("Hanging workspace cleanup completed successfully",
		"deletedCount", deletedCount,
		"freedBytes", freedBytes,
	)

	return nil
}

// DiscoverHangingProjects scans filesystem at baseDir/orgID/userID/projectID and diffs against database records.
// Traversal is resilient to broken symlinks, unreadable directories, or folder permission errors.
func DiscoverHangingProjects(ctx context.Context, mountPath string, orgStore db.OrganizationStore, projectStore db.ProjectStore, firestoreClient *firestore.Client) ([]HangingProjectItem, error) {
	if mountPath == "" {
		mountPath = "~/workspaces"
	}
	cleanMount := filepath.Clean(core.ExpandPath(mountPath))

	baseDir := cleanMount
	if info, err := os.Stat(filepath.Join(cleanMount, "organizations")); err == nil && info.IsDir() {
		baseDir = filepath.Join(cleanMount, "organizations")
	}

	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		slog.Debug("Workspace base directory does not exist", "path", baseDir)
		return []HangingProjectItem{}, nil
	}

	// 0. Build set of valid Organization IDs from DB
	validOrgs := make(map[string]bool)

	if firestoreClient != nil {
		orgDocs, err := firestoreClient.Collection("organizations").Documents(ctx).GetAll()
		if err != nil {
			slog.Error("Failed to list organizations from Firestore client", "error", err)
			return nil, fmt.Errorf("failed to list organizations from firestore: %w", err)
		}
		for _, d := range orgDocs {
			validOrgs[d.Ref.ID] = true
		}
	}

	if orgStore != nil {
		orgs, err := orgStore.ListOrganizations(ctx)
		if err != nil {
			slog.Error("Failed to list organizations from orgStore", "error", err)
			return nil, fmt.Errorf("failed to list organizations from org store: %w", err)
		}
		for _, o := range orgs {
			validOrgs[o.ID] = true
		}
	}

	// 1. Scan filesystem for project folders (depth 3: baseDir/orgID/userID/projectID)
	fsIndex := make(map[string]HangingProjectItem)

	orgEntries, err := os.ReadDir(baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []HangingProjectItem{}, nil
		}
		slog.Warn("Failed to read workspace base dir", "path", baseDir, "error", err)
		return []HangingProjectItem{}, nil
	}

	for _, orgEntry := range orgEntries {
		if strings.HasPrefix(orgEntry.Name(), ".") {
			continue
		}
		isDir := orgEntry.IsDir()
		if !isDir && orgEntry.Type()&fs.ModeSymlink != 0 {
			if info, statErr := os.Stat(filepath.Join(baseDir, orgEntry.Name())); statErr == nil && info.IsDir() {
				isDir = true
			}
		}
		if !isDir {
			continue
		}
		orgID := orgEntry.Name()

		// Do not list anything for which there is no organization match in the database
		if len(validOrgs) > 0 && !validOrgs[orgID] {
			continue
		}

		orgPath := filepath.Join(baseDir, orgID)

		userEntries, err := os.ReadDir(orgPath)
		if err != nil {
			slog.Warn("Failed to read organization directory during scan", "path", orgPath, "error", err)
			continue
		}

		for _, userEntry := range userEntries {
			if strings.HasPrefix(userEntry.Name(), ".") {
				continue
			}
			isUserDir := userEntry.IsDir()
			if !isUserDir && userEntry.Type()&fs.ModeSymlink != 0 {
				if info, statErr := os.Stat(filepath.Join(orgPath, userEntry.Name())); statErr == nil && info.IsDir() {
					isUserDir = true
				}
			}
			if !isUserDir {
				continue
			}
			userID := userEntry.Name()
			userPath := filepath.Join(orgPath, userID)

			projEntries, err := os.ReadDir(userPath)
			if err != nil {
				slog.Warn("Failed to read user directory during scan", "path", userPath, "error", err)
				continue
			}

			for _, projEntry := range projEntries {
				if strings.HasPrefix(projEntry.Name(), ".") {
					continue
				}

				projPath := filepath.Join(userPath, projEntry.Name())
				isProjDir := projEntry.IsDir()
				if !isProjDir && projEntry.Type()&fs.ModeSymlink != 0 {
					if info, statErr := os.Stat(projPath); statErr == nil && info.IsDir() {
						isProjDir = true
					}
				}
				if !isProjDir {
					continue
				}

				projID := projEntry.Name()
				key := fmt.Sprintf("%s-%s-%s", orgID, userID, projID)
				fsIndex[key] = HangingProjectItem{
					IndexKey:  key,
					OrgID:     orgID,
					UserID:    userID,
					ProjectID: projID,
					Path:      projPath,
				}
			}
		}
	}

	if len(fsIndex) == 0 {
		return []HangingProjectItem{}, nil
	}

	// 2. Build DB index (orgid-userid-projectid)
	dbIndex := make(map[string]bool)

	if firestoreClient != nil {
		projDocs, err := firestoreClient.CollectionGroup("projects").Documents(ctx).GetAll()
		if err != nil {
			slog.Error("Failed to query projects collection group from firestore", "error", err)
			return nil, fmt.Errorf("failed to query projects collection group: %w", err)
		}
		for _, doc := range projDocs {
			projID := doc.Ref.ID
			var p db.Project
			if err := doc.DataTo(&p); err == nil {
				orgID := ""
				if doc.Ref.Parent != nil && doc.Ref.Parent.Parent != nil {
					orgID = doc.Ref.Parent.Parent.ID
				}
				if orgID != "" {
					if p.OwnerId != nil && *p.OwnerId != "" {
						dbIndex[fmt.Sprintf("%s-%s-%s", orgID, *p.OwnerId, projID)] = true
					}
					if p.Members != nil {
						for mID := range *p.Members {
							dbIndex[fmt.Sprintf("%s-%s-%s", orgID, mID, projID)] = true
						}
					}
				}
			}
		}

		upDocs, err := firestoreClient.CollectionGroup("user_project").Documents(ctx).GetAll()
		if err != nil {
			slog.Error("Failed to query user_project collection group from firestore", "error", err)
			return nil, fmt.Errorf("failed to query user_project collection group: %w", err)
		}
		for _, doc := range upDocs {
			orgID := ""
			if doc.Ref.Parent != nil && doc.Ref.Parent.Parent != nil {
				orgID = doc.Ref.Parent.Parent.ID
			}
			var up struct {
				UserID    string `firestore:"userId"`
				ProjectID string `firestore:"projectId"`
			}
			if err := doc.DataTo(&up); err == nil && orgID != "" && up.UserID != "" && up.ProjectID != "" {
				dbIndex[fmt.Sprintf("%s-%s-%s", orgID, up.UserID, up.ProjectID)] = true
			}
		}
	}

	if orgStore != nil {
		orgs, err := orgStore.ListOrganizations(ctx)
		if err != nil {
			slog.Error("Failed to list organizations for projects lookup", "error", err)
			return nil, fmt.Errorf("failed to list organizations for projects lookup: %w", err)
		}
		for _, org := range orgs {
			targetStore := projectStore
			if targetStore == nil {
				targetStore = db.ProjectStoreInstance
			}
			if targetStore != nil {
				projects, err := targetStore.ListProjects(ctx, org.ID)
				if err != nil {
					slog.Error("Failed to list projects for org", "orgId", org.ID, "error", err)
					return nil, fmt.Errorf("failed to list projects for organization %s: %w", org.ID, err)
				}
				for _, p := range projects {
					if p.OwnerId != nil && *p.OwnerId != "" {
						dbIndex[fmt.Sprintf("%s-%s-%s", org.ID, *p.OwnerId, p.ID)] = true
					}
					if p.Members != nil {
						for mID := range *p.Members {
							dbIndex[fmt.Sprintf("%s-%s-%s", org.ID, mID, p.ID)] = true
						}
					}
				}
			}
		}
	}

	// 3. Diff: Find items in fsIndex missing from dbIndex
	var hanging []HangingProjectItem
	for key, item := range fsIndex {
		if !dbIndex[key] {
			bytes, files, _ := ScanWorkspaceDirectory(item.Path)
			lockPath := item.Path + ".lock"
			if lockInfo, err := os.Stat(lockPath); err == nil {
				if lockInfo.IsDir() {
					lBytes, lFiles, _ := ScanWorkspaceDirectory(lockPath)
					bytes += lBytes
					files += lFiles
				} else {
					bytes += lockInfo.Size()
					files++
				}
			}
			item.SizeBytes = bytes
			item.FileCount = files
			hanging = append(hanging, item)
		}
	}

	return hanging, nil
}
