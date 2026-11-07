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
	"time"

	"cloud.google.com/go/firestore"
	"google.com/apf/api/core"
	"google.com/apf/api/db"
)

func init() {
	_ = Register(&NFSAuditJob{})
}

// NFSAuditJob implements the Job interface for auditing NFS storage disk usage and updating metadata.
type NFSAuditJob struct {
	DryRun    bool
	NFSStore  db.NFSStore
	MountPath string
}

// Name returns the unique job identifier.
func (j *NFSAuditJob) Name() string {
	return "nfs-audit"
}

// Description returns a summary of the job's purpose.
func (j *NFSAuditJob) Description() string {
	return "Audits NFS workspace storage usage and updates metadata"
}

func (j *NFSAuditJob) isDryRun(ctx context.Context) bool {
	if IsDryRun(ctx) {
		return true
	}
	if j.DryRun {
		return true
	}
	env := strings.ToLower(strings.TrimSpace(os.Getenv("DRY_RUN")))
	return env == "true" || env == "1" || env == "yes"
}

// Run executes the NFS storage audit job.
func (j *NFSAuditJob) Run(ctx context.Context, cfg *core.Configuration) error {
	if j.isDryRun(ctx) {
		ctx = ContextWithDryRun(ctx, true)
	}

	store := j.NFSStore
	if store == nil {
		if cfg != nil && cfg.GoogleCloudProject != "" {
			var client *firestore.Client
			var err error
			if cfg.FirestoreDbName != "" {
				client, err = firestore.NewClientWithDatabase(ctx, cfg.GoogleCloudProject, cfg.FirestoreDbName)
			} else {
				client, err = firestore.NewClient(ctx, cfg.GoogleCloudProject)
			}
			if err != nil {
				return fmt.Errorf("failed to initialize firestore client for nfs-audit: %w", err)
			}
			defer client.Close()
			store = db.NewFirestoreNFSStore(client)
		}
	}

	if store == nil {
		return fmt.Errorf("NFSStore is not initialized and firestore configuration is missing")
	}

	effectiveCfg := cfg
	if j.MountPath != "" {
		if effectiveCfg == nil {
			effectiveCfg = &core.Configuration{}
		} else {
			// Clone configuration to avoid modifying caller's struct
			cfgCopy := *effectiveCfg
			effectiveCfg = &cfgCopy
		}
		effectiveCfg.WorkspaceMountPath = j.MountPath
	}

	slog.Info("Starting NFS storage audit job", "job", j.Name())
	auditedCount, err := RunNFSAuditScan(ctx, effectiveCfg, store)
	if err != nil {
		slog.Error("NFS storage audit scan failed", "job", j.Name(), "error", err)
		return fmt.Errorf("nfs storage audit scan failed: %w", err)
	}

	slog.Info("NFS storage audit job completed successfully", "job", j.Name(), "foldersAudited", auditedCount)
	return nil
}

// WorkspaceDirStats holds scanned disk usage metrics for a workspace directory.
type WorkspaceDirStats struct {
	WorkspacePath  string
	OrganizationID string
	ProjectID      string
	TotalBytes     int64
	FileCount      int64
}

// ScanWorkspaceDirectory recursively calculates total size in bytes and file count for a directory.
// It continues traversal upon encountering unreadable directories, broken symlinks, or permission errors.
// It avoids infinite recursion when encountering circular or nested directory symlinks.
func ScanWorkspaceDirectory(dirPath string) (totalBytes int64, fileCount int64, err error) {
	return scanWorkspaceDirWithVisited(dirPath, make(map[string]bool))
}

func scanWorkspaceDirWithVisited(dirPath string, visited map[string]bool) (totalBytes int64, fileCount int64, err error) {
	targetPath := dirPath
	if resolved, evalErr := filepath.EvalSymlinks(dirPath); evalErr == nil {
		targetPath = resolved
	}
	if visited[targetPath] {
		return 0, 0, nil
	}
	visited[targetPath] = true

	err = filepath.WalkDir(targetPath, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if os.IsNotExist(walkErr) {
				return nil
			}
			slog.Warn("Encountered inaccessible path during directory scan", "path", path, "err", walkErr)
			return nil
		}

		if path == targetPath {
			return nil
		}

		if d.Type()&fs.ModeSymlink != 0 {
			info, statErr := os.Stat(path)
			if statErr != nil {
				slog.Warn("Encountered broken symlink or inaccessible target during scan", "path", path, "err", statErr)
				return nil
			}
			if info.IsDir() {
				subBytes, subFiles, _ := scanWorkspaceDirWithVisited(path, visited)
				totalBytes += subBytes
				fileCount += subFiles
			} else {
				fileCount++
				totalBytes += info.Size()
			}
			return nil
		}

		if !d.IsDir() {
			info, infoErr := d.Info()
			if infoErr != nil {
				slog.Warn("Failed to get file info during scan", "path", path, "err", infoErr)
				return nil
			}
			fileCount++
			totalBytes += info.Size()
		}
		return nil
	})
	return totalBytes, fileCount, err
}

// DiscoverWorkspaceFolders scans mountPath for organization workspace directories.
// Aggregates total storage size and file count at the Organization level.
func DiscoverWorkspaceFolders(mountPath string) ([]WorkspaceDirStats, error) {
	if mountPath == "" {
		mountPath = "~/workspaces"
	}
	cleanMount := filepath.Clean(core.ExpandPath(mountPath))

	if _, err := os.Stat(cleanMount); os.IsNotExist(err) {
		slog.Debug("Workspace mount path does not exist", "path", cleanMount)
		return []WorkspaceDirStats{}, nil
	}

	orgsDir := cleanMount
	if info, err := os.Stat(filepath.Join(cleanMount, "organizations")); err == nil && info.IsDir() {
		orgsDir = filepath.Join(cleanMount, "organizations")
	}

	var results []WorkspaceDirStats

	orgEntries, err := os.ReadDir(orgsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []WorkspaceDirStats{}, nil
		}
		slog.Warn("Failed to read organizations directory", "path", orgsDir, "err", err)
		return nil, fmt.Errorf("failed to read orgs dir: %w", err)
	}

	for _, orgEntry := range orgEntries {
		if strings.HasPrefix(orgEntry.Name(), ".") {
			continue
		}
		isDir := orgEntry.IsDir()
		if !isDir && orgEntry.Type()&fs.ModeSymlink != 0 {
			if info, statErr := os.Stat(filepath.Join(orgsDir, orgEntry.Name())); statErr == nil && info.IsDir() {
				isDir = true
			}
		}
		if !isDir {
			continue
		}
		orgID := orgEntry.Name()
		orgDir := filepath.Join(orgsDir, orgID)

		bytes, files, scanErr := ScanWorkspaceDirectory(orgDir)
		if scanErr != nil {
			slog.Warn("Failed to scan organization directory", "path", orgDir, "err", scanErr)
			continue
		}

		results = append(results, WorkspaceDirStats{
			WorkspacePath:  orgDir,
			OrganizationID: orgID,
			ProjectID:      "All Projects",
			TotalBytes:     bytes,
			FileCount:      files,
		})
	}

	return results, nil
}

// RunNFSAuditScan scans all registered or discovered organization workspace folders and updates Firestore metadata.
func RunNFSAuditScan(ctx context.Context, config *core.Configuration, nfsStore db.NFSStore) (int, error) {
	if nfsStore == nil {
		return 0, fmt.Errorf("NFSStore is nil")
	}

	env := strings.ToLower(strings.TrimSpace(os.Getenv("DRY_RUN")))
	dryRun := IsDryRun(ctx) || env == "true" || env == "1" || env == "yes"
	if dryRun {
		slog.Info("Running NFS storage audit in DRY-RUN mode (metadata writes skipped)")
	}

	mountPath := "~/workspaces"
	if config != nil && config.WorkspaceMountPath != "" {
		mountPath = config.WorkspaceMountPath
	}

	// 1. Scan disk for organization workspace folders
	discovered, err := DiscoverWorkspaceFolders(mountPath)
	if err != nil {
		slog.Error("Failed to discover workspace folders", "err", err)
		return 0, fmt.Errorf("failed to discover workspace folders: %w", err)
	}

	// 2. Fetch existing folder metadata records from Firestore
	existing, err := nfsStore.ListFolderMetadata(ctx, core.NFSFolderFilter{})
	if err != nil {
		return 0, fmt.Errorf("failed to list existing folder metadata: %w", err)
	}

	metaMap := make(map[string]*core.NFSFolderMetadata)
	for i := range existing {
		if existing[i].FolderID != "" {
			metaMap[existing[i].FolderID] = &existing[i]
		}
		if existing[i].WorkspacePath != "" {
			metaMap[existing[i].WorkspacePath] = &existing[i]
		}
	}

	auditedCount := 0

	// 3. Update or create organization-level metadata records
	for _, d := range discovered {
		folderID := fmt.Sprintf("org-%s", d.OrganizationID)
		meta, found := metaMap[folderID]
		if !found {
			meta, found = metaMap[d.WorkspacePath]
		}

		if !dryRun {
			if !found {
				meta = &core.NFSFolderMetadata{
					FolderID:        folderID,
					OrganizationID:  d.OrganizationID,
					ProjectID:       "All Projects",
					WorkspacePath:   d.WorkspacePath,
					QuotaSizeBytes:  107374182400, // Default 100GB per Organization
					UsedSizeBytes:   d.TotalBytes,
					FileCount:       d.FileCount,
					RetentionPolicy: "90d",
					LastAuditedAt:   time.Now(),
				}
				if saveErr := nfsStore.SaveFolderMetadata(ctx, meta); saveErr != nil {
					slog.Error("Failed to save organization metadata during audit", "folderID", folderID, "err", saveErr)
					continue
				}
			} else {
				if upsertErr := nfsStore.UpsertFolderAuditStats(ctx, meta.FolderID, d.TotalBytes, d.FileCount); upsertErr != nil {
					slog.Error("Failed to upsert audit stats for org", "folderID", meta.FolderID, "err", upsertErr)
					continue
				}
			}
		}

		auditedCount++
		slog.Info("Audited organization workspace folder",
			"dryRun", dryRun,
			"orgId", d.OrganizationID,
			"folderId", folderID,
			"path", d.WorkspacePath,
			"bytes", d.TotalBytes,
			"files", d.FileCount,
		)
	}

	// 4. Clean up legacy project-level records from Firestore
	if !dryRun {
		for _, m := range existing {
			if strings.HasPrefix(m.FolderID, "folder-") && m.ProjectID != "All Projects" && m.ProjectID != "" {
				if delErr := nfsStore.DeleteFolderMetadata(ctx, m.FolderID); delErr != nil {
					slog.Warn("Failed to delete legacy project metadata record", "folderID", m.FolderID, "err", delErr)
				}
			}
		}
	}

	return auditedCount, nil
}
