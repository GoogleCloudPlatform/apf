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

package jobs_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"google.com/apf/api/core"
	"google.com/apf/api/db"
	"google.com/apf/api/jobs"
)

type mockOrgStore struct {
	orgs []db.Organization
}

func (m *mockOrgStore) ListOrganizations(ctx context.Context) ([]db.Organization, error) {
	return m.orgs, nil
}

func (m *mockOrgStore) GetOrganization(ctx context.Context, id string) (*db.Organization, error) {
	for _, o := range m.orgs {
		if o.ID == id {
			return &o, nil
		}
	}
	return nil, db.ErrNotFound
}

func (m *mockOrgStore) GetOrganizationByName(ctx context.Context, name string) (*db.Organization, error) {
	for _, o := range m.orgs {
		if o.Name == name {
			return &o, nil
		}
	}
	return nil, db.ErrNotFound
}

func (m *mockOrgStore) CreateOrganization(ctx context.Context, o *db.Organization) (string, error) {
	return o.ID, nil
}

func (m *mockOrgStore) CreateOrganizationWithID(ctx context.Context, id string, o *db.Organization) error {
	return nil
}

func (m *mockOrgStore) UpdateOrganizationConfig(ctx context.Context, id string, config *db.OrganizationConfig) error {
	return nil
}

func (m *mockOrgStore) DeleteOrganization(ctx context.Context, id string) error {
	return nil
}

func (m *mockOrgStore) UpdateOrganizationName(ctx context.Context, id string, name string) error {
	return nil
}

type mockProjectStore struct {
	projects map[string][]db.Project
}

func newMockProjectStore() *mockProjectStore {
	return &mockProjectStore{
		projects: make(map[string][]db.Project),
	}
}

func (m *mockProjectStore) ListProjects(ctx context.Context, organizationID string) ([]db.Project, error) {
	return m.projects[organizationID], nil
}

func (m *mockProjectStore) ListProjectsByUser(ctx context.Context, organizationID string, userId string) ([]db.Project, []db.Project, error) {
	return nil, nil, nil
}

func (m *mockProjectStore) GetProject(ctx context.Context, organizationID string, id string) (*db.Project, error) {
	for _, p := range m.projects[organizationID] {
		if p.ID == id {
			return &p, nil
		}
	}
	return nil, db.ErrNotFound
}

func (m *mockProjectStore) CreateProject(ctx context.Context, organizationID string, p *db.Project) error {
	return nil
}

func (m *mockProjectStore) UpdateProject(ctx context.Context, organizationID string, p *db.Project) error {
	return nil
}

func (m *mockProjectStore) DeleteProject(ctx context.Context, organizationID string, id string) error {
	return nil
}

func (m *mockProjectStore) ReconcileMemberID(ctx context.Context, organizationID string, oldID string, newID string) error {
	return nil
}

func TestHangingWorkspaceCleanupJob_Registration(t *testing.T) {
	job, ok := jobs.Get("hanging-cleanup")
	if !ok {
		t.Fatalf("expected hanging-cleanup to be registered in DefaultRegistry")
	}
	if job.Name() != "hanging-cleanup" {
		t.Errorf("expected job name 'hanging-cleanup', got %q", job.Name())
	}
	if job.Description() == "" {
		t.Errorf("expected non-empty job description")
	}
}

func TestHangingWorkspaceCleanupJob_DryRun_NoDeletion(t *testing.T) {
	tempMount := t.TempDir()

	// Structure:
	// tempMount/org-1/user-1/proj-active (Active in DB)
	// tempMount/org-1/user-1/proj-orphan (Missing in DB -> Orphan)
	// tempMount/org-1/user-1/proj-orphan.lock (Lock file)
	activeDir := filepath.Join(tempMount, "org-1", "user-1", "proj-active")
	orphanDir := filepath.Join(tempMount, "org-1", "user-1", "proj-orphan")
	orphanLock := orphanDir + ".lock"

	_ = os.MkdirAll(activeDir, 0755)
	_ = os.MkdirAll(orphanDir, 0755)
	_ = os.WriteFile(filepath.Join(activeDir, "main.go"), []byte("package main"), 0644)
	_ = os.WriteFile(filepath.Join(orphanDir, "junk.txt"), []byte("orphaned data"), 0644)
	_ = os.WriteFile(orphanLock, []byte("locked"), 0644)

	orgStore := &mockOrgStore{
		orgs: []db.Organization{{ID: "org-1", Name: "Org 1"}},
	}
	projStore := newMockProjectStore()
	owner := "user-1"
	projStore.projects["org-1"] = []db.Project{
		{ID: "proj-active", OwnerId: &owner},
	}

	// 1. Dry run via struct field
	jobField := &jobs.HangingWorkspaceCleanupJob{
		DryRun:            true,
		MountPath:         tempMount,
		OrganizationStore: orgStore,
		ProjectStore:      projStore,
	}

	ctx := context.Background()
	cfg := &core.Configuration{WorkspaceMountPath: tempMount}

	if err := jobField.Run(ctx, cfg); err != nil {
		t.Fatalf("jobField.Run failed: %v", err)
	}

	if _, err := os.Stat(orphanDir); os.IsNotExist(err) {
		t.Errorf("expected orphan directory to still exist after struct dry-run")
	}
	if _, err := os.Stat(orphanLock); os.IsNotExist(err) {
		t.Errorf("expected orphan lock file to still exist after struct dry-run")
	}

	// 2. Dry run via context
	jobCtx := &jobs.HangingWorkspaceCleanupJob{
		DryRun:            false,
		MountPath:         tempMount,
		OrganizationStore: orgStore,
		ProjectStore:      projStore,
	}
	dryCtx := jobs.ContextWithDryRun(ctx, true)

	if err := jobCtx.Run(dryCtx, cfg); err != nil {
		t.Fatalf("jobCtx.Run failed: %v", err)
	}

	if _, err := os.Stat(orphanDir); os.IsNotExist(err) {
		t.Errorf("expected orphan directory to still exist after context dry-run")
	}

	// 3. Dry run via environment variable
	t.Setenv("DRY_RUN", "true")
	jobEnv := &jobs.HangingWorkspaceCleanupJob{
		DryRun:            false,
		MountPath:         tempMount,
		OrganizationStore: orgStore,
		ProjectStore:      projStore,
	}

	if err := jobEnv.Run(ctx, cfg); err != nil {
		t.Fatalf("jobEnv.Run failed: %v", err)
	}

	if _, err := os.Stat(orphanDir); os.IsNotExist(err) {
		t.Errorf("expected orphan directory to still exist after env dry-run")
	}
}

func TestHangingWorkspaceCleanupJob_LiveRun_DeletesOrphans(t *testing.T) {
	tempMount := t.TempDir()

	activeDir := filepath.Join(tempMount, "org-1", "user-1", "proj-active")
	orphanDir := filepath.Join(tempMount, "org-1", "user-1", "proj-orphan")
	orphanLock := orphanDir + ".lock"

	_ = os.MkdirAll(activeDir, 0755)
	_ = os.MkdirAll(orphanDir, 0755)
	_ = os.WriteFile(filepath.Join(activeDir, "main.go"), []byte("package main"), 0644)
	_ = os.WriteFile(filepath.Join(orphanDir, "junk.txt"), []byte("orphaned data"), 0644)
	_ = os.WriteFile(orphanLock, []byte("locked"), 0644)

	orgStore := &mockOrgStore{
		orgs: []db.Organization{{ID: "org-1", Name: "Org 1"}},
	}
	projStore := newMockProjectStore()
	owner := "user-1"
	projStore.projects["org-1"] = []db.Project{
		{ID: "proj-active", OwnerId: &owner},
	}

	job := &jobs.HangingWorkspaceCleanupJob{
		DryRun:            false,
		MountPath:         tempMount,
		OrganizationStore: orgStore,
		ProjectStore:      projStore,
	}

	ctx := context.Background()
	cfg := &core.Configuration{WorkspaceMountPath: tempMount}

	if err := job.Run(ctx, cfg); err != nil {
		t.Fatalf("job.Run failed: %v", err)
	}

	// Verify orphan directory and lock file are deleted
	if _, err := os.Stat(orphanDir); !os.IsNotExist(err) {
		t.Errorf("expected orphan directory to be deleted")
	}
	if _, err := os.Stat(orphanLock); !os.IsNotExist(err) {
		t.Errorf("expected orphan lock file to be deleted")
	}

	// Verify active project and parent directories are PRESERVED
	if _, err := os.Stat(activeDir); os.IsNotExist(err) {
		t.Errorf("active project directory should NOT have been deleted")
	}
	if _, err := os.Stat(filepath.Join(tempMount, "org-1", "user-1")); os.IsNotExist(err) {
		t.Errorf("parent user directory should NOT have been deleted")
	}
}

func TestDiscoverHangingProjects_EdgeCases(t *testing.T) {
	// 1. Missing mount path
	results, err := jobs.DiscoverHangingProjects(context.Background(), "/path/that/does/not/exist", nil, nil, nil)
	if err != nil {
		t.Fatalf("expected nil error on missing mount path, got %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected empty results, got %d", len(results))
	}

	// 2. Organizations subdirectory with broken symlinks inside projects
	tempMount := t.TempDir()
	orgsDir := filepath.Join(tempMount, "organizations")
	projDir := filepath.Join(orgsDir, "org-2", "user-2", "proj-symlink")
	_ = os.MkdirAll(projDir, 0755)
	_ = os.WriteFile(filepath.Join(projDir, "file.txt"), []byte("test"), 0644)
	_ = os.Symlink(filepath.Join(tempMount, "no-target"), filepath.Join(projDir, "broken_link"))

	orgStore := &mockOrgStore{
		orgs: []db.Organization{{ID: "org-2", Name: "Org 2"}},
	}

	hanging, err := jobs.DiscoverHangingProjects(context.Background(), tempMount, orgStore, nil, nil)
	if err != nil {
		t.Fatalf("DiscoverHangingProjects failed: %v", err)
	}
	if len(hanging) != 1 {
		t.Fatalf("expected 1 hanging project, got %d", len(hanging))
	}
	if hanging[0].IndexKey != "org-2-user-2-proj-symlink" {
		t.Errorf("expected IndexKey 'org-2-user-2-proj-symlink', got %q", hanging[0].IndexKey)
	}
}

func TestHangingWorkspaceCleanupJob_MissingStoresError(t *testing.T) {
	job := &jobs.HangingWorkspaceCleanupJob{}
	err := job.Run(context.Background(), nil)
	if err == nil {
		t.Fatalf("expected error when running cleanup job without database stores, got nil")
	}
}

type errorOrgStore struct {
	mockOrgStore
}

func (e *errorOrgStore) ListOrganizations(ctx context.Context) ([]db.Organization, error) {
	return nil, os.ErrPermission
}

func TestDiscoverHangingProjects_StoreError(t *testing.T) {
	tempMount := t.TempDir()
	_, err := jobs.DiscoverHangingProjects(context.Background(), tempMount, &errorOrgStore{}, nil, nil)
	if err == nil {
		t.Fatalf("expected error when orgStore fails, got nil")
	}
}

func TestDiscoverHangingProjects_SymlinkOrgDir(t *testing.T) {
	tempMount := t.TempDir()
	realOrgDir := filepath.Join(tempMount, "real-org")
	projDir := filepath.Join(realOrgDir, "user1", "proj1")
	_ = os.MkdirAll(projDir, 0755)
	_ = os.WriteFile(filepath.Join(projDir, "file.txt"), []byte("data"), 0644)

	orgsDir := filepath.Join(tempMount, "organizations")
	_ = os.MkdirAll(orgsDir, 0755)
	_ = os.Symlink(realOrgDir, filepath.Join(orgsDir, "symlinked-org"))

	orgStore := &mockOrgStore{
		orgs: []db.Organization{{ID: "symlinked-org", Name: "Symlinked Org"}},
	}

	hanging, err := jobs.DiscoverHangingProjects(context.Background(), tempMount, orgStore, nil, nil)
	if err != nil {
		t.Fatalf("DiscoverHangingProjects failed: %v", err)
	}
	if len(hanging) != 1 {
		t.Fatalf("expected 1 hanging project from symlinked org, got %d", len(hanging))
	}
	if hanging[0].IndexKey != "symlinked-org-user1-proj1" {
		t.Errorf("expected IndexKey 'symlinked-org-user1-proj1', got %q", hanging[0].IndexKey)
	}
}

func TestDiscoverHangingProjects_SymlinkUserDir(t *testing.T) {
	tempMount := t.TempDir()
	realUserDir := filepath.Join(tempMount, "real-user")
	projDir := filepath.Join(realUserDir, "proj1")
	_ = os.MkdirAll(projDir, 0755)
	_ = os.WriteFile(filepath.Join(projDir, "file.txt"), []byte("data"), 0644)

	orgDir := filepath.Join(tempMount, "org-symlink-user")
	_ = os.MkdirAll(orgDir, 0755)
	_ = os.Symlink(realUserDir, filepath.Join(orgDir, "symlinked-user"))

	orgStore := &mockOrgStore{
		orgs: []db.Organization{{ID: "org-symlink-user", Name: "Org"}},
	}

	hanging, err := jobs.DiscoverHangingProjects(context.Background(), tempMount, orgStore, nil, nil)
	if err != nil {
		t.Fatalf("DiscoverHangingProjects failed: %v", err)
	}
	if len(hanging) != 1 {
		t.Fatalf("expected 1 hanging project from symlinked user dir, got %d", len(hanging))
	}
	if hanging[0].IndexKey != "org-symlink-user-symlinked-user-proj1" {
		t.Errorf("expected IndexKey 'org-symlink-user-symlinked-user-proj1', got %q", hanging[0].IndexKey)
	}
}
