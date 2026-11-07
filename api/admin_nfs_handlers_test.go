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
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"google.com/apf/api/core"
	"google.com/apf/api/db"
)

type mockNFSStore struct {
	folders map[string]*core.NFSFolderMetadata
}

type nfsTestMockProjectStore struct{}

func (m *nfsTestMockProjectStore) ListProjects(ctx context.Context, organizationID string) ([]db.Project, error) {
	if organizationID == "org1" {
		owner := "user1"
		return []db.Project{
			{
				ID:      "proj1",
				OwnerId: &owner,
			},
		}, nil
	}
	return nil, nil
}
func (m *nfsTestMockProjectStore) ListProjectsByUser(ctx context.Context, organizationID string, userId string) ([]db.Project, []db.Project, error) { return nil, nil, nil }
func (m *nfsTestMockProjectStore) GetProject(ctx context.Context, organizationID string, id string) (*db.Project, error) { return nil, nil }
func (m *nfsTestMockProjectStore) CreateProject(ctx context.Context, organizationID string, p *db.Project) error { return nil }
func (m *nfsTestMockProjectStore) UpdateProject(ctx context.Context, organizationID string, p *db.Project) error { return nil }
func (m *nfsTestMockProjectStore) DeleteProject(ctx context.Context, organizationID string, id string) error { return nil }
func (m *nfsTestMockProjectStore) ReconcileMemberID(ctx context.Context, organizationID string, oldID string, newID string) error { return nil }
type nfsTestMockOrgStore struct {
	orgs []db.Organization
}

func (m *nfsTestMockOrgStore) GetOrganization(ctx context.Context, id string) (*db.Organization, error) {
	for _, o := range m.orgs {
		if o.ID == id {
			return &o, nil
		}
	}
	return nil, db.ErrNotFound
}
func (m *nfsTestMockOrgStore) GetOrganizationByName(ctx context.Context, name string) (*db.Organization, error) { return nil, nil }
func (m *nfsTestMockOrgStore) CreateOrganization(ctx context.Context, org *db.Organization) (string, error) { return "", nil }
func (m *nfsTestMockOrgStore) CreateOrganizationWithID(ctx context.Context, id string, org *db.Organization) error { return nil }
func (m *nfsTestMockOrgStore) ListOrganizations(ctx context.Context) ([]db.Organization, error) {
	return m.orgs, nil
}
func (m *nfsTestMockOrgStore) UpdateOrganizationConfig(ctx context.Context, id string, config *db.OrganizationConfig) error { return nil }
func (m *nfsTestMockOrgStore) DeleteOrganization(ctx context.Context, id string) error { return nil }
func (m *nfsTestMockOrgStore) UpdateOrganizationName(ctx context.Context, id string, name string) error { return nil }

func newMockNFSStore() *mockNFSStore {
	return &mockNFSStore{
		folders: make(map[string]*core.NFSFolderMetadata),
	}
}

func (m *mockNFSStore) GetFolderMetadata(ctx context.Context, folderID string) (*core.NFSFolderMetadata, error) {
	folder, ok := m.folders[folderID]
	if !ok {
		return nil, db.ErrNotFound
	}
	return folder, nil
}

func (m *mockNFSStore) GetFolderMetadataByPath(ctx context.Context, workspacePath string) (*core.NFSFolderMetadata, error) {
	for _, f := range m.folders {
		if f.WorkspacePath == workspacePath {
			return f, nil
		}
	}
	return nil, nil
}

func (m *mockNFSStore) UpdateFolderMetadata(ctx context.Context, folderID string, updates core.NFSFolderMetadataUpdate) (*core.NFSFolderMetadata, error) {
	folder, ok := m.folders[folderID]
	if !ok {
		folder = &core.NFSFolderMetadata{FolderID: folderID}
		m.folders[folderID] = folder
	}
	if updates.QuotaSizeBytes != nil {
		folder.QuotaSizeBytes = *updates.QuotaSizeBytes
	}
	if updates.RetentionPolicy != nil {
		folder.RetentionPolicy = *updates.RetentionPolicy
	}
	folder.UpdatedAt = time.Now()
	return folder, nil
}

func (m *mockNFSStore) ListFolderMetadata(ctx context.Context, filter core.NFSFolderFilter) ([]core.NFSFolderMetadata, error) {
	var res []core.NFSFolderMetadata
	for _, f := range m.folders {
		if filter.OrganizationID != "" && f.OrganizationID != filter.OrganizationID {
			continue
		}
		res = append(res, *f)
	}
	return res, nil
}

func (m *mockNFSStore) UpsertFolderAuditStats(ctx context.Context, folderID string, usedBytes int64, fileCount int64) error {
	folder, ok := m.folders[folderID]
	if !ok {
		folder = &core.NFSFolderMetadata{FolderID: folderID}
		m.folders[folderID] = folder
	}
	folder.UsedSizeBytes = usedBytes
	folder.FileCount = fileCount
	folder.LastAuditedAt = time.Now()
	return nil
}

func (m *mockNFSStore) SaveFolderMetadata(ctx context.Context, meta *core.NFSFolderMetadata) error {
	m.folders[meta.FolderID] = meta
	return nil
}

func (m *mockNFSStore) DeleteFolderMetadata(ctx context.Context, folderID string) error {
	delete(m.folders, folderID)
	return nil
}

func (m *mockNFSStore) GetNFSVolumeStats(ctx context.Context, mountPath string) (*core.NFSVolumeStats, error) {
	return &core.NFSVolumeStats{
		MountPath:      mountPath,
		TotalBytes:     1000000000,
		AvailableBytes: 500000000,
		UsedBytes:      500000000,
		IsHealthy:      true,
	}, nil
}

func (m *mockNFSStore) CheckFolderWritePermitted(ctx context.Context, workspacePath string) error {
	for _, f := range m.folders {
		if f.WorkspacePath == workspacePath {
			if f.QuotaSizeBytes > 0 && f.UsedSizeBytes >= f.QuotaSizeBytes {
				return core.ErrNFSQuotaExceeded
			}
		}
	}
	return nil
}

func TestAdminNFSHandlers_SuperAdminAuth(t *testing.T) {
	mockStore := newMockNFSStore()
	mockStore.folders["f1"] = &core.NFSFolderMetadata{
		FolderID:       "f1",
		OrganizationID: "org-1",
		ProjectID:      "proj-1",
		WorkspacePath:  "/tmp/workspaces/org-1/proj-1",
		QuotaSizeBytes: 1000,
	}

	handler := NewAdminHandler(nil, nil, &core.Configuration{}).WithNFSStore(mockStore)

	t.Run("GetNFSVolumeStatus - SuperAdmin", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/admin/nfs/volumes", nil)
		claims := &core.UserClaims{IsSuperAdmin: true, Username: "super-admin"}
		ctx := context.WithValue(req.Context(), core.UserClaimsKey, claims)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.GetNFSVolumeStatus(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
		}
	})

	t.Run("GetNFSVolumeStatus - NonSuperAdmin (403 Forbidden)", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/admin/nfs/volumes", nil)
		claims := &core.UserClaims{IsSuperAdmin: false, Username: "regular-user"}
		ctx := context.WithValue(req.Context(), core.UserClaimsKey, claims)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.GetNFSVolumeStatus(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected 403 Forbidden, got %d", w.Code)
		}
	})

	t.Run("ListNFSFolderMetadata - SuperAdmin", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/admin/nfs/folders", nil)
		claims := &core.UserClaims{IsSuperAdmin: true, Username: "super-admin"}
		ctx := context.WithValue(req.Context(), core.UserClaimsKey, claims)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.ListNFSFolderMetadata(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 OK, got %d", w.Code)
		}
	})

	t.Run("UpdateNFSFolderMetadata - SuperAdmin", func(t *testing.T) {
		newQuota := int64(5000)
		body, _ := json.Marshal(core.NFSFolderMetadataUpdate{
			QuotaSizeBytes: &newQuota,
		})

		req := httptest.NewRequest("PUT", "/api/v1/admin/nfs/folders/f1", bytes.NewReader(body))
		req.SetPathValue("folderId", "f1")
		claims := &core.UserClaims{IsSuperAdmin: true, Username: "super-admin"}
		ctx := context.WithValue(req.Context(), core.UserClaimsKey, claims)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.UpdateNFSFolderMetadata(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 OK, got %d", w.Code)
		}
		if mockStore.folders["f1"].QuotaSizeBytes != 5000 {
			t.Errorf("Expected quota 5000, got %d", mockStore.folders["f1"].QuotaSizeBytes)
		}
	})

	t.Run("GetHangingProjects and DeleteHangingProjects", func(t *testing.T) {
		tempDir := t.TempDir()
		// Create mock directory structure:
		// tempDir/org1/user1/proj1 (valid in DB mock)
		// tempDir/org1/user1/proj_hanging (hanging under matched org1)
		// tempDir/unmatched_org/user2/proj_unmatched (hanging under unmatched org -> should NOT be listed)
		org1User1Proj1 := filepath.Join(tempDir, "org1", "user1", "proj1")
		org1User1Hanging := filepath.Join(tempDir, "org1", "user1", "proj_hanging")
		unmatchedOrgHanging := filepath.Join(tempDir, "unmatched_org", "user2", "proj_unmatched")

		os.MkdirAll(org1User1Proj1, 0755)
		os.MkdirAll(org1User1Hanging, 0755)
		os.MkdirAll(unmatchedOrgHanging, 0755)
		os.WriteFile(filepath.Join(org1User1Hanging, "dummy.txt"), []byte("test content"), 0644)
		lockFile := org1User1Hanging + ".lock"
		os.WriteFile(lockFile, []byte("lock content"), 0644)

		orgStore := &nfsTestMockOrgStore{
			orgs: []db.Organization{{ID: "org1", Name: "Org 1"}},
		}

		cfg := &core.Configuration{WorkspaceMountPath: tempDir}
		db.ProjectStoreInstance = &nfsTestMockProjectStore{}
		h := NewAdminHandler(nil, orgStore, cfg).WithNFSStore(mockStore)

		// Call GetHangingProjects
		req := httptest.NewRequest("GET", "/api/v1/admin/nfs/hanging-projects", nil)
		claims := &core.UserClaims{IsSuperAdmin: true, Username: "super-admin"}
		req = req.WithContext(context.WithValue(req.Context(), core.UserClaimsKey, claims))

		w := httptest.NewRecorder()
		h.GetHangingProjects(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
		}

		var res struct {
			Status          string               `json:"status"`
			HangingProjects []HangingProjectItem `json:"hangingProjects"`
			Total           int                  `json:"total"`
		}
		json.Unmarshal(w.Body.Bytes(), &res)

		if res.Total != 1 {
			t.Errorf("Expected exactly 1 hanging project (under matched org1), got %d", res.Total)
		}
		if len(res.HangingProjects) > 0 && res.HangingProjects[0].OrgID != "org1" {
			t.Errorf("Expected orgID org1, got %s", res.HangingProjects[0].OrgID)
		}

		// Call DeleteHangingProjects with both indexKeys and paths
		delReqBody, _ := json.Marshal(DeleteHangingProjectsRequest{
			IndexKeys: []string{"org1-user1-proj_hanging"},
			Paths:     []string{org1User1Hanging},
		})
		reqDel := httptest.NewRequest("POST", "/api/v1/admin/nfs/hanging-projects/delete", bytes.NewReader(delReqBody))
		reqDel = reqDel.WithContext(context.WithValue(reqDel.Context(), core.UserClaimsKey, claims))

		wDel := httptest.NewRecorder()
		h.DeleteHangingProjects(wDel, reqDel)

		if wDel.Code != http.StatusOK {
			t.Fatalf("Expected 200 OK on delete, got %d. Body: %s", wDel.Code, wDel.Body.String())
		}

		// Verify hanging dir and lock file are deleted, but parent org1/user1 is NOT deleted
		if _, err := os.Stat(org1User1Hanging); !os.IsNotExist(err) {
			t.Errorf("Expected hanging project dir to be deleted, but it still exists")
		}
		if _, err := os.Stat(lockFile); !os.IsNotExist(err) {
			t.Errorf("Expected hanging project lock file to be deleted, but it still exists")
		}
		if _, err := os.Stat(filepath.Join(tempDir, "org1", "user1")); os.IsNotExist(err) {
			t.Errorf("Parent user directory should not have been deleted")
		}
	})

	t.Run("DeleteHangingProjects with Hyphenated IDs and Organizations Subdirectory", func(t *testing.T) {
		tempDir := t.TempDir()
		orgsDir := filepath.Join(tempDir, "organizations")
		// tempDir/organizations/org-100/user-200/proj-hanging-300
		projHanging := filepath.Join(orgsDir, "org-100", "user-200", "proj-hanging-300")
		os.MkdirAll(projHanging, 0755)
		os.WriteFile(filepath.Join(projHanging, "file.txt"), []byte("data"), 0644)

		orgStore := &nfsTestMockOrgStore{
			orgs: []db.Organization{{ID: "org-100", Name: "Org 100"}},
		}
		cfg := &core.Configuration{WorkspaceMountPath: tempDir}
		h := NewAdminHandler(nil, orgStore, cfg).WithNFSStore(mockStore)

		claims := &core.UserClaims{IsSuperAdmin: true, Username: "super-admin"}

		// 1. Discover should find it and format indexKey as org-100-user-200-proj-hanging-300
		hanging, err := DiscoverHangingProjects(context.Background(), tempDir, orgStore)
		if err != nil {
			t.Fatalf("DiscoverHangingProjects failed: %v", err)
		}
		if len(hanging) != 1 {
			t.Fatalf("Expected 1 hanging project, got %d", len(hanging))
		}
		if hanging[0].IndexKey != "org-100-user-200-proj-hanging-300" {
			t.Errorf("Expected indexKey org-100-user-200-proj-hanging-300, got %s", hanging[0].IndexKey)
		}

		// 2. Delete using IndexKey
		delReqBody, _ := json.Marshal(DeleteHangingProjectsRequest{
			IndexKeys: []string{"org-100-user-200-proj-hanging-300"},
		})
		reqDel := httptest.NewRequest("POST", "/api/v1/admin/nfs/hanging-projects/delete", bytes.NewReader(delReqBody))
		reqDel = reqDel.WithContext(context.WithValue(reqDel.Context(), core.UserClaimsKey, claims))

		wDel := httptest.NewRecorder()
		h.DeleteHangingProjects(wDel, reqDel)

		if wDel.Code != http.StatusOK {
			t.Fatalf("Expected 200 OK on delete, got %d. Body: %s", wDel.Code, wDel.Body.String())
		}

		var delRes struct {
			Status       string `json:"status"`
			DeletedCount int    `json:"deletedCount"`
		}
		json.Unmarshal(wDel.Body.Bytes(), &delRes)

		if delRes.DeletedCount != 1 {
			t.Errorf("Expected deletedCount 1, got %d. Response: %s", delRes.DeletedCount, wDel.Body.String())
		}

		if _, err := os.Stat(projHanging); !os.IsNotExist(err) {
			t.Errorf("Expected proj-hanging-300 to be deleted, but it still exists")
		}
	})

	t.Run("DeleteHangingProjects with Failures and Warnings", func(t *testing.T) {
		tempDir := t.TempDir()
		orgsDir := filepath.Join(tempDir, "organizations")
		validProj := filepath.Join(orgsDir, "org1", "user1", "proj1")
		os.MkdirAll(validProj, 0755)
		os.WriteFile(filepath.Join(validProj, "file.txt"), []byte("valid"), 0644)

		orgStore := &nfsTestMockOrgStore{
			orgs: []db.Organization{{ID: "org1", Name: "Org 1"}},
		}
		cfg := &core.Configuration{WorkspaceMountPath: tempDir}
		h := NewAdminHandler(nil, orgStore, cfg).WithNFSStore(mockStore)
		claims := &core.UserClaims{IsSuperAdmin: true, Username: "super-admin"}

		// Delete with an invalid outside path, an invalid non-project-level path, and a valid path
		delReqBody, _ := json.Marshal(DeleteHangingProjectsRequest{
			IndexKeys: []string{"org1-user1-proj1", "org1-user1", "outside-path"},
			Paths:     []string{"/tmp/completely/outside/workspace"},
		})
		reqDel := httptest.NewRequest("POST", "/api/v1/admin/nfs/hanging-projects/delete", bytes.NewReader(delReqBody))
		reqDel = reqDel.WithContext(context.WithValue(reqDel.Context(), core.UserClaimsKey, claims))

		wDel := httptest.NewRecorder()
		h.DeleteHangingProjects(wDel, reqDel)

		if wDel.Code != http.StatusOK {
			t.Fatalf("Expected 200 OK, got %d. Body: %s", wDel.Code, wDel.Body.String())
		}

		var delRes struct {
			Status       string   `json:"status"`
			Message      string   `json:"message"`
			DeletedCount int      `json:"deletedCount"`
			Errors       []string `json:"errors"`
		}
		if err := json.Unmarshal(wDel.Body.Bytes(), &delRes); err != nil {
			t.Fatalf("Failed to parse delete response: %v", err)
		}

		if delRes.DeletedCount != 1 {
			t.Errorf("Expected deletedCount 1, got %d", delRes.DeletedCount)
		}
		if delRes.Status != "WARNING" {
			t.Errorf("Expected status WARNING for partial failure, got %s", delRes.Status)
		}
		if len(delRes.Errors) == 0 {
			t.Errorf("Expected non-empty errors array for invalid paths, got 0 errors")
		}
		if _, err := os.Stat(validProj); !os.IsNotExist(err) {
			t.Errorf("Expected validProj to be deleted, but it still exists")
		}
	})
}
