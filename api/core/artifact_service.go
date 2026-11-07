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

package core

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
)

// NFSWritePermittedChecker checks if folder write operations are permitted.
type NFSWritePermittedChecker interface {
	CheckFolderWritePermitted(ctx context.Context, workspacePath string) error
}

// ArtifactService defines the interface for managing artifacts.
type ArtifactService interface {
	GetArtifact(ctx context.Context, organizationID string, userId string, workspaceID string, path string) ([]byte, error)
	WriteArtifact(ctx context.Context, organizationID string, userId string, workspaceID string, path string, data []byte) error
	DeleteArtifact(ctx context.Context, organizationID string, userId string, workspaceID string, path string) error
	ListArtifacts(ctx context.Context, organizationID string, userId string, workspaceID string, prefix string) ([]string, error)
	ListTopLevelFiles(ctx context.Context, organizationID string, userId string, workspaceID string) ([]string, error)
	SaveSessionArtifact(ctx context.Context, projectId, sessionId, filename string, r io.Reader) (storageType string, storagePath string, bytesWritten int64, err error)
	ReadSessionArtifact(ctx context.Context, storageType string, storagePath string) ([]byte, error)
}

// FileSystemArtifactService implements ArtifactService using local file system.
type FileSystemArtifactService struct {
	basePath          string
	localArtifactPath string
	nfsStore          NFSWritePermittedChecker
}

// NewFileSystemArtifactService creates a new FileSystemArtifactService.
func NewFileSystemArtifactService(basePath string) *FileSystemArtifactService {
	return &FileSystemArtifactService{basePath: basePath}
}

// WithLocalArtifactPath sets the base directory for local artifact storage.
func (s *FileSystemArtifactService) WithLocalArtifactPath(localArtifactPath string) *FileSystemArtifactService {
	s.localArtifactPath = localArtifactPath
	return s
}

// WithNFSStore attaches an NFS store to check write permissions before file creation/update.
func (s *FileSystemArtifactService) WithNFSStore(store NFSWritePermittedChecker) *FileSystemArtifactService {
	s.nfsStore = store
	return s
}

// GetArtifact reads artifact content from file system.
func (s *FileSystemArtifactService) GetArtifact(ctx context.Context, organizationID string, userId string, workspaceID string, path string) ([]byte, error) {
	fullPath := filepath.Join(s.basePath, organizationID, userId, workspaceID, path)
	return os.ReadFile(fullPath)
}

// WriteArtifact writes artifact content to file system with NFS quota & lock checking.
func (s *FileSystemArtifactService) WriteArtifact(ctx context.Context, organizationID string, userId string, workspaceID string, path string, data []byte) error {
	workspaceRoot := filepath.Join(s.basePath, organizationID, userId, workspaceID)
	if s.nfsStore != nil {
		if err := s.nfsStore.CheckFolderWritePermitted(ctx, workspaceRoot); err != nil {
			return err
		}
	}

	fullPath := filepath.Join(workspaceRoot, path)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	return os.WriteFile(fullPath, data, 0644)
}

// DeleteArtifact deletes artifact file from file system after verifying NFS lock/quota.
func (s *FileSystemArtifactService) DeleteArtifact(ctx context.Context, organizationID string, userId string, workspaceID string, path string) error {
	workspaceRoot := filepath.Join(s.basePath, organizationID, userId, workspaceID)
	if s.nfsStore != nil {
		if err := s.nfsStore.CheckFolderWritePermitted(ctx, workspaceRoot); err != nil {
			return err
		}
	}
	fullPath := filepath.Join(workspaceRoot, path)
	return os.Remove(fullPath)
}

// ListArtifacts lists files recursively on the local file system under the basePath/organizationId/userId/workspaceID/prefix.
func (s *FileSystemArtifactService) ListArtifacts(ctx context.Context, organizationID string, userId string, workspaceID string, prefix string) ([]string, error) {
	workspaceRoot := filepath.Join(s.basePath, organizationID, userId, workspaceID)
	dirPath := filepath.Join(workspaceRoot, prefix)

	var filePaths []string
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if !info.IsDir() {
			// Get path relative to workspaceRoot
			rel, err := filepath.Rel(workspaceRoot, path)
			if err != nil {
				return err
			}
			filePaths = append(filePaths, rel)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk local filesystem: %w", err)
	}
	return filePaths, nil
}

// ListTopLevelFiles lists direct, non-directory files at the root of the workspace directory.
func (s *FileSystemArtifactService) ListTopLevelFiles(ctx context.Context, organizationID string, userId string, workspaceID string) ([]string, error) {
	workspaceRoot := filepath.Join(s.basePath, organizationID, userId, workspaceID)
	entries, err := os.ReadDir(workspaceRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read workspace root directory: %w", err)
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, entry.Name())
		}
	}
	return files, nil
}

// SaveSessionArtifact persists a session artifact to the local filesystem at {localArtifactPath}/{projectId}/{sessionId}/{filename}.
func (s *FileSystemArtifactService) SaveSessionArtifact(ctx context.Context, projectId, sessionId, filename string, r io.Reader) (string, string, int64, error) {
	sanitized := filepath.Base(filepath.Clean(filename))
	if sanitized == "." || sanitized == "/" || sanitized == "" {
		return "", "", 0, fmt.Errorf("invalid filename: %s", filename)
	}

	base := s.localArtifactPath
	if base == "" {
		if s.basePath != "" {
			base = s.basePath
		} else {
			base = ".foundry/artifacts"
		}
	}

	targetDir := filepath.Join(base, projectId, sessionId)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", "", 0, fmt.Errorf("failed to create artifact directory: %w", err)
	}

	targetPath := filepath.Join(targetDir, sanitized)
	f, err := os.Create(targetPath)
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to create artifact file: %w", err)
	}
	defer f.Close()

	written, err := io.Copy(f, r)
	if err != nil {
		return "", "", written, fmt.Errorf("failed to write artifact data: %w", err)
	}
	return "local_file", targetPath, written, nil
}

// ReadSessionArtifact reads the content of a session artifact from the local filesystem.
func (s *FileSystemArtifactService) ReadSessionArtifact(ctx context.Context, storageType string, storagePath string) ([]byte, error) {
	if storagePath == "" {
		return nil, fmt.Errorf("storage path is empty")
	}
	if storageType != "" && storageType != "local_file" && storageType != "local" {
		return nil, fmt.Errorf("unsupported storage type for filesystem artifact service: %s", storageType)
	}
	cleanedPath := filepath.Clean(storagePath)
	if s.basePath != "" {
		rel, err := filepath.Rel(s.basePath, cleanedPath)
		if err != nil || strings.HasPrefix(rel, "..") {
			return nil, fmt.Errorf("storage path %q is outside base path", storagePath)
		}
	}
	return os.ReadFile(cleanedPath)
}

// GcsArtifactService implements ArtifactService using Google Cloud Storage.
type GcsArtifactService struct {
	client     *storage.Client
	bucketName string
}

// NewGcsArtifactService creates a new GcsArtifactService.
func NewGcsArtifactService(client *storage.Client, bucketName string) *GcsArtifactService {
	return &GcsArtifactService{client: client, bucketName: bucketName}
}

// GetArtifact reads artifact content from GCS.
func (s *GcsArtifactService) GetArtifact(ctx context.Context, organizationID string, userId string, workspaceID string, path string) ([]byte, error) {
	objectName := fmt.Sprintf("%s/%s/%s/docs/%s", organizationID, userId, workspaceID, path)

	rc, err := s.client.Bucket(s.bucketName).Object(objectName).NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read from GCS: %w", err)
	}
	defer rc.Close()

	return io.ReadAll(rc)
}

// WriteArtifact writes artifact content to GCS.
func (s *GcsArtifactService) WriteArtifact(ctx context.Context, organizationID string, userId string, workspaceID string, path string, data []byte) error {
	objectName := fmt.Sprintf("%s/%s/%s/docs/%s", organizationID, userId, workspaceID, path)
	wc := s.client.Bucket(s.bucketName).Object(objectName).NewWriter(ctx)
	if _, err := wc.Write(data); err != nil {
		_ = wc.Close()
		return fmt.Errorf("failed to write to GCS: %w", err)
	}
	return wc.Close()
}

// DeleteArtifact deletes artifact from GCS.
func (s *GcsArtifactService) DeleteArtifact(ctx context.Context, organizationID string, userId string, workspaceID string, path string) error {
	objectName := fmt.Sprintf("%s/%s/%s/docs/%s", organizationID, userId, workspaceID, path)
	err := s.client.Bucket(s.bucketName).Object(objectName).Delete(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete from GCS: %w", err)
	}
	return nil
}

// ListArtifacts lists objects on GCS with the prefix workspaces/organizationId/userId/workspaceID/docs/prefix.
func (s *GcsArtifactService) ListArtifacts(ctx context.Context, organizationID string, userId string, workspaceID string, prefix string) ([]string, error) {
	gcsPrefix := fmt.Sprintf("%s/%s/%s/docs/%s", organizationID, userId, workspaceID, prefix)
	if !strings.HasSuffix(gcsPrefix, "/") && prefix != "" {
		gcsPrefix += "/"
	}

	var filePaths []string
	it := s.client.Bucket(s.bucketName).Objects(ctx, &storage.Query{Prefix: gcsPrefix})
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to list GCS objects: %w", err)
		}
		// Trim prefix to keep the path relative to workspace root
		relPath := strings.TrimPrefix(attrs.Name, fmt.Sprintf("%s/%s/%s/docs/", organizationID, userId, workspaceID))
		if !strings.HasSuffix(relPath, "/") {
			filePaths = append(filePaths, relPath)
		}
	}
	return filePaths, nil
}

// ListTopLevelFiles lists direct objects located directly in the workspace root without descending into subdirectories.
func (s *GcsArtifactService) ListTopLevelFiles(ctx context.Context, organizationID string, userId string, workspaceID string) ([]string, error) {
	gcsPrefix := fmt.Sprintf("%s/%s/%s/docs/", organizationID, userId, workspaceID)
	it := s.client.Bucket(s.bucketName).Objects(ctx, &storage.Query{
		Prefix:    gcsPrefix,
		Delimiter: "/",
	})

	var files []string
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to list GCS root objects: %w", err)
		}
		if attrs.Name != "" {
			relPath := strings.TrimPrefix(attrs.Name, gcsPrefix)
			if relPath != "" && !strings.HasSuffix(relPath, "/") {
				files = append(files, relPath)
			}
		}
	}
	return files, nil
}

// SaveSessionArtifact persists a session artifact to GCS at projects/{projectId}/sessions/{sessionId}/{filename}.
func (s *GcsArtifactService) SaveSessionArtifact(ctx context.Context, projectId, sessionId, filename string, r io.Reader) (string, string, int64, error) {
	sanitized := filepath.Base(filepath.Clean(filename))
	if sanitized == "." || sanitized == "/" || sanitized == "" {
		return "", "", 0, fmt.Errorf("invalid filename: %s", filename)
	}

	objectName := fmt.Sprintf("projects/%s/sessions/%s/%s", projectId, sessionId, sanitized)
	wc := s.client.Bucket(s.bucketName).Object(objectName).NewWriter(ctx)
	written, err := io.Copy(wc, r)
	if err != nil {
		_ = wc.Close()
		return "", "", written, fmt.Errorf("failed to stream artifact to GCS: %w", err)
	}
	if err := wc.Close(); err != nil {
		return "", "", written, fmt.Errorf("failed to close GCS writer: %w", err)
	}
	gcsURI := fmt.Sprintf("gs://%s/%s", s.bucketName, objectName)
	return "gcs", gcsURI, written, nil
}

// ReadSessionArtifact reads the content of a session artifact from GCS.
func (s *GcsArtifactService) ReadSessionArtifact(ctx context.Context, storageType string, storagePath string) ([]byte, error) {
	if storagePath == "" {
		return nil, fmt.Errorf("storage path is empty")
	}
	if storageType != "" && storageType != "gcs" {
		return nil, fmt.Errorf("unsupported storage type for gcs artifact service: %s", storageType)
	}
	if s.client == nil {
		return nil, fmt.Errorf("gcs client is uninitialized")
	}

	bucket := s.bucketName
	objectName := storagePath
	if strings.HasPrefix(storagePath, "gs://") {
		trimmed := strings.TrimPrefix(storagePath, "gs://")
		parts := strings.SplitN(trimmed, "/", 2)
		if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
			bucket = parts[0]
			objectName = parts[1]
		} else {
			return nil, fmt.Errorf("invalid gcs storage path: %s", storagePath)
		}
	}

	rc, err := s.client.Bucket(bucket).Object(objectName).NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read from GCS: %w", err)
	}
	defer rc.Close()

	return io.ReadAll(rc)
}
