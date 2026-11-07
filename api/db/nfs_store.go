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

package db

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"cloud.google.com/go/firestore"
	"google.com/apf/api/core"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func normalizePath(path string) string {
	if path == "" {
		return ""
	}
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			path = filepath.Join(home, path[2:])
		}
	}
	return filepath.Clean(path)
}

// Collection name under which folder metadata documents are stored in Firestore:
// Document path: /admin_nfs_metadata/folders/folders/{folderId}
const (
	NFSAdminParentCollection = "admin_nfs_metadata"
	NFSAdminFoldersDocument  = "folders"
	NFSAdminFoldersSubcoll   = "folders"
)

// NFSStore defines data access methods for NFS Volume and Folder-level metadata management.
type NFSStore interface {
	GetFolderMetadata(ctx context.Context, folderID string) (*core.NFSFolderMetadata, error)
	GetFolderMetadataByPath(ctx context.Context, workspacePath string) (*core.NFSFolderMetadata, error)
	UpdateFolderMetadata(ctx context.Context, folderID string, updates core.NFSFolderMetadataUpdate) (*core.NFSFolderMetadata, error)
	ListFolderMetadata(ctx context.Context, filter core.NFSFolderFilter) ([]core.NFSFolderMetadata, error)
	UpsertFolderAuditStats(ctx context.Context, folderID string, usedBytes int64, fileCount int64) error
	SaveFolderMetadata(ctx context.Context, meta *core.NFSFolderMetadata) error
	DeleteFolderMetadata(ctx context.Context, folderID string) error
	GetNFSVolumeStats(ctx context.Context, mountPath string) (*core.NFSVolumeStats, error)
	CheckFolderWritePermitted(ctx context.Context, workspacePath string) error
}

// FirestoreNFSStore implements NFSStore using Google Cloud Firestore.
type FirestoreNFSStore struct {
	client *firestore.Client
}

// NewFirestoreNFSStore creates a new FirestoreNFSStore instance.
func NewFirestoreNFSStore(client *firestore.Client) *FirestoreNFSStore {
	return &FirestoreNFSStore{client: client}
}

func (s *FirestoreNFSStore) folderCollection() *firestore.CollectionRef {
	return s.client.Collection(NFSAdminParentCollection).Doc(NFSAdminFoldersDocument).Collection(NFSAdminFoldersSubcoll)
}

// GetFolderMetadata retrieves a folder metadata record by folder ID.
func (s *FirestoreNFSStore) GetFolderMetadata(ctx context.Context, folderID string) (*core.NFSFolderMetadata, error) {
	if folderID == "" {
		return nil, fmt.Errorf("folderID cannot be empty")
	}

	doc, err := s.folderCollection().Doc(folderID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get folder metadata for %s: %w", folderID, err)
	}

	var meta core.NFSFolderMetadata
	if err := doc.DataTo(&meta); err != nil {
		return nil, fmt.Errorf("failed to decode folder metadata: %w", err)
	}
	meta.FolderID = doc.Ref.ID
	return &meta, nil
}

// GetFolderMetadataByPath retrieves a folder metadata record by workspace path.
func (s *FirestoreNFSStore) GetFolderMetadataByPath(ctx context.Context, workspacePath string) (*core.NFSFolderMetadata, error) {
	if workspacePath == "" {
		return nil, fmt.Errorf("workspacePath cannot be empty")
	}

	cleanPath := normalizePath(workspacePath)
	iter := s.folderCollection().Where("workspacePath", "==", cleanPath).Limit(1).Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if err == nil {
		var meta core.NFSFolderMetadata
		if err := doc.DataTo(&meta); err == nil {
			meta.FolderID = doc.Ref.ID
			return &meta, nil
		}
	}

	// Fallback 1: Raw path query
	iter2 := s.folderCollection().Where("workspacePath", "==", workspacePath).Limit(1).Documents(ctx)
	defer iter2.Stop()
	doc2, err2 := iter2.Next()
	if err2 == nil {
		var meta core.NFSFolderMetadata
		if err := doc2.DataTo(&meta); err == nil {
			meta.FolderID = doc2.Ref.ID
			return &meta, nil
		}
	}

	// Fallback 2: Check if cleanPath is a subpath of an indexed Organization workspacePath
	allDocs, err := s.folderCollection().Documents(ctx).GetAll()
	if err == nil {
		var bestMatch *core.NFSFolderMetadata
		var longestMatch int
		for _, d := range allDocs {
			var meta core.NFSFolderMetadata
			if err := d.DataTo(&meta); err == nil {
				meta.FolderID = d.Ref.ID
				if meta.WorkspacePath != "" && (cleanPath == meta.WorkspacePath || strings.HasPrefix(cleanPath, meta.WorkspacePath+"/")) {
					if len(meta.WorkspacePath) > longestMatch {
						longestMatch = len(meta.WorkspacePath)
						bestMatch = &meta
					}
				}
			}
		}
		if bestMatch != nil {
			return bestMatch, nil
		}
	}

	return nil, ErrNotFound
}

// SaveFolderMetadata creates or overwrites a folder metadata record.
func (s *FirestoreNFSStore) SaveFolderMetadata(ctx context.Context, meta *core.NFSFolderMetadata) error {
	if meta == nil || meta.FolderID == "" {
		return fmt.Errorf("folder metadata and FolderID must not be empty")
	}
	if meta.CreatedAt.IsZero() {
		meta.CreatedAt = time.Now()
	}
	meta.UpdatedAt = time.Now()
	meta.WorkspacePath = normalizePath(meta.WorkspacePath)

	_, err := s.folderCollection().Doc(meta.FolderID).Set(ctx, meta)
	if err != nil {
		return fmt.Errorf("failed to save folder metadata: %w", err)
	}
	return nil
}

// UpdateFolderMetadata updates specified fields (quota, read-only status, retention policy) on a folder metadata record.
func (s *FirestoreNFSStore) UpdateFolderMetadata(ctx context.Context, folderID string, updates core.NFSFolderMetadataUpdate) (*core.NFSFolderMetadata, error) {
	if folderID == "" {
		return nil, fmt.Errorf("folderID cannot be empty")
	}

	docRef := s.folderCollection().Doc(folderID)
	doc, err := docRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to retrieve folder metadata: %w", err)
	}

	var meta core.NFSFolderMetadata
	if err := doc.DataTo(&meta); err != nil {
		return nil, fmt.Errorf("failed to decode folder metadata: %w", err)
	}

	var firestoreUpdates []firestore.Update
	if updates.QuotaSizeBytes != nil {
		meta.QuotaSizeBytes = *updates.QuotaSizeBytes
		firestoreUpdates = append(firestoreUpdates, firestore.Update{Path: "quotaSizeBytes", Value: *updates.QuotaSizeBytes})
	}
	if updates.RetentionPolicy != nil {
		meta.RetentionPolicy = *updates.RetentionPolicy
		firestoreUpdates = append(firestoreUpdates, firestore.Update{Path: "retentionPolicy", Value: *updates.RetentionPolicy})
	}

	meta.UpdatedAt = time.Now()
	firestoreUpdates = append(firestoreUpdates, firestore.Update{Path: "updatedAt", Value: meta.UpdatedAt})

	if len(firestoreUpdates) > 0 {
		_, err = docRef.Update(ctx, firestoreUpdates)
		if err != nil {
			return nil, fmt.Errorf("failed to update folder metadata: %w", err)
		}
	}

	return &meta, nil
}

// ListFolderMetadata lists all folder metadata records, optionally filtered by organization ID or project ID.
func (s *FirestoreNFSStore) ListFolderMetadata(ctx context.Context, filter core.NFSFolderFilter) ([]core.NFSFolderMetadata, error) {
	query := s.folderCollection().Query

	if filter.OrganizationID != "" {
		query = query.Where("organizationId", "==", filter.OrganizationID)
	}
	if filter.ProjectID != "" {
		query = query.Where("projectId", "==", filter.ProjectID)
	}

	docs, err := query.Documents(ctx).GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to list folder metadata: %w", err)
	}

	results := make([]core.NFSFolderMetadata, 0, len(docs))
	for _, doc := range docs {
		var meta core.NFSFolderMetadata
		if err := doc.DataTo(&meta); err != nil {
			return nil, fmt.Errorf("failed to decode folder metadata item: %w", err)
		}
		meta.FolderID = doc.Ref.ID
		results = append(results, meta)
	}

	return results, nil
}

// UpsertFolderAuditStats updates usedSizeBytes, fileCount, and lastAuditedAt for a folder record.
func (s *FirestoreNFSStore) UpsertFolderAuditStats(ctx context.Context, folderID string, usedBytes int64, fileCount int64) error {
	if folderID == "" {
		return fmt.Errorf("folderID cannot be empty")
	}

	now := time.Now()
	docRef := s.folderCollection().Doc(folderID)
	_, err := docRef.Set(ctx, map[string]interface{}{
		"usedSizeBytes": usedBytes,
		"fileCount":     fileCount,
		"lastAuditedAt": now,
		"updatedAt":     now,
	}, firestore.MergeAll)

	if err != nil {
		return fmt.Errorf("failed to upsert audit stats for %s: %w", folderID, err)
	}
	return nil
}

// DeleteFolderMetadata removes a folder metadata record.
func (s *FirestoreNFSStore) DeleteFolderMetadata(ctx context.Context, folderID string) error {
	if folderID == "" {
		return fmt.Errorf("folderID cannot be empty")
	}
	_, err := s.folderCollection().Doc(folderID).Delete(ctx)
	return err
}

// GetNFSVolumeStats queries disk volume capacity and health for a mount path using statfs syscall.
func (s *FirestoreNFSStore) GetNFSVolumeStats(ctx context.Context, mountPath string) (*core.NFSVolumeStats, error) {
	if mountPath == "" {
		mountPath = "/var/nfs/workspaces" // default mount path
	}

	var stat syscall.Statfs_t
	err := syscall.Statfs(mountPath, &stat)
	if err != nil {
		return &core.NFSVolumeStats{
			MountPath:      mountPath,
			TotalBytes:     0,
			AvailableBytes: 0,
			UsedBytes:      0,
			IsHealthy:      false,
		}, fmt.Errorf("statfs failed for mount path %s: %w", mountPath, err)
	}

	totalBytes := int64(stat.Blocks) * int64(stat.Bsize)
	freeBytes := int64(stat.Bavail) * int64(stat.Bsize)
	usedBytes := totalBytes - freeBytes

	return &core.NFSVolumeStats{
		MountPath:      mountPath,
		TotalBytes:     totalBytes,
		AvailableBytes: freeBytes,
		UsedBytes:      usedBytes,
		IsHealthy:      true,
	}, nil
}

// CheckFolderWritePermitted checks if write operations are allowed on the given workspace path.
func (s *FirestoreNFSStore) CheckFolderWritePermitted(ctx context.Context, workspacePath string) error {
	if workspacePath == "" {
		return nil
	}

	meta, err := s.GetFolderMetadataByPath(ctx, workspacePath)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			// If folder metadata doc does not exist yet, allow by default
			return nil
		}
		return err
	}

	if meta.QuotaSizeBytes > 0 && meta.UsedSizeBytes >= meta.QuotaSizeBytes {
		return core.ErrNFSQuotaExceeded
	}

	return nil
}
