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

package workspace

import (	"google.com/apf/api/agent"

	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"google.com/apf/api/core"
	"google.com/apf/api/db"
)

type mockSyncArtifactStore struct {
	db.ArtifactStore
	syncCalled   bool
	discovered   []db.Artifact
	addedCount   int
	deletedCount int
}

func (m *mockSyncArtifactStore) SyncArtifacts(ctx context.Context, organizationID string, projectId string, discovered []db.Artifact) (int, int, error) {
	m.syncCalled = true
	m.discovered = discovered
	return m.addedCount, m.deletedCount, nil
}

func (m *mockSyncArtifactStore) ListArtifactsForProject(ctx context.Context, organizationID string, projectId string) ([]db.Artifact, error) {
	return nil, nil
}

type mockSyncProjectStore struct {
	db.ProjectStore
	project *db.Project
}

func (m *mockSyncProjectStore) GetProject(ctx context.Context, organizationID string, id string) (*db.Project, error) {
	return m.project, nil
}

func (m *mockSyncProjectStore) UpdateProject(ctx context.Context, organizationID string, p *db.Project) error {
	return nil
}

func TestSyncArtifactsHandler_Success(t *testing.T) {
	// Force FileSystem artifact service for testing
	tempDir := t.TempDir()
	os.Setenv("ARTIFACT_SERVICE_TYPE", "FS")
	os.Setenv("WORKSPACE_MOUNT_PATH", tempDir)
	defer os.Unsetenv("ARTIFACT_SERVICE_TYPE")
	defer os.Unsetenv("WORKSPACE_MOUNT_PATH")

	// Save original stores
	origArtifactStore := ArtifactStore
	origStore := db.ProjectStoreInstance
	origMessageStore := agent.Store
	defer func() {
		ArtifactStore = origArtifactStore
		db.ProjectStoreInstance = origStore
		agent.Store = origMessageStore
	}()

	// Create physical files to scan
	userId := "user@google.com"
	projectId := "proj123"
	specLoc := "docs/spec"
	planningLoc := specLoc + "/plans"
	
	organizationId := "test-organization"
	fullPlanningPath := filepath.Join(tempDir, organizationId, userId, projectId, planningLoc, "epic_planning_spec.md")
	fullSpecPath := filepath.Join(tempDir, organizationId, userId, projectId, specLoc, "prd.md")
	fullBriefPath := filepath.Join(tempDir, organizationId, userId, projectId, specLoc, "brief.md")
	fullOtherPath := filepath.Join(tempDir, organizationId, userId, projectId, specLoc, "other_spec.md")

	fullAgentsPath := filepath.Join(tempDir, organizationId, userId, projectId, "AGENTS.md")

	if err := os.MkdirAll(filepath.Dir(fullPlanningPath), 0755); err != nil {
		t.Fatalf("Failed to create planning directory: %v", err)
	}
	if err := os.WriteFile(fullPlanningPath, []byte("planning spec"), 0644); err != nil {
		t.Fatalf("Failed to write planning file: %v", err)
	}
	if err := os.WriteFile(fullSpecPath, []byte("general spec prd"), 0644); err != nil {
		t.Fatalf("Failed to write spec file: %v", err)
	}
	if err := os.WriteFile(fullBriefPath, []byte("general brief"), 0644); err != nil {
		t.Fatalf("Failed to write brief file: %v", err)
	}
	if err := os.WriteFile(fullOtherPath, []byte("other neutral spec"), 0644); err != nil {
		t.Fatalf("Failed to write other spec file: %v", err)
	}
	if err := os.WriteFile(fullAgentsPath, []byte("root agents instructions"), 0644); err != nil {
		t.Fatalf("Failed to write agents file: %v", err)
	}

	// Configure project
	proj := &db.Project{
		ID: projectId,
		AppIntegrations: []db.ProjectIntegration{
			{
				IntegrationID: "github",
				Enabled:       true,
				Settings: map[string]interface{}{
					"specLocation": specLoc,
				},
			},
		},
	}

	mArtStore := &mockSyncArtifactStore{addedCount: 5, deletedCount: 0}
	mProjStore := &mockSyncProjectStore{project: proj}

	ArtifactStore = mArtStore
	db.ProjectStoreInstance = mProjStore
	agent.Store = &safeDummyMessageStore{} // dummy message db.ProjectStoreInstance to prevent nil pointer in recalculateGates

	// Create HTTP request
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/proj123/artifacts/sync", nil)
	req.SetPathValue("id", projectId)

	claims := &core.UserClaims{Subject: userId}
	req = withTestCtx(req, claims)

	w := httptest.NewRecorder()
	SyncArtifactsHandler(w, req)

	// Assert response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp["status"] != "SUCCESS" {
		t.Errorf("Expected status SUCCESS, got %q", resp["status"])
	}

	if !mArtStore.syncCalled {
		t.Error("Expected SyncArtifacts to be called on mock db.ProjectStoreInstance")
	}

	// Check that 5 artifacts were discovered
	if len(mArtStore.discovered) != 5 {
		t.Errorf("Expected 5 discovered artifacts, got %d", len(mArtStore.discovered))
	}

	// Verify discovered metadata
	foundPlanning := false
	foundSpec := false
	foundBrief := false
	foundOther := false
	foundAgents := false
	for _, a := range mArtStore.discovered {
		if a.Title == "epic_planning_spec.md" {
			foundPlanning = true
			if a.Phase != "SOLUTIONING" {
				t.Errorf("Expected epic_planning_spec.md to be in SOLUTIONING phase, got %q", a.Phase)
			}
		}
		if a.Title == "prd.md" {
			foundSpec = true
			if a.Phase != "PLANNING" {
				t.Errorf("Expected prd.md to be in PLANNING phase, got %q", a.Phase)
			}
		}
		if a.Title == "brief.md" {
			foundBrief = true
			if a.Phase != "ANALYSIS" {
				t.Errorf("Expected brief.md to be in ANALYSIS phase, got %q", a.Phase)
			}
		}
		if a.Title == "other_spec.md" {
			foundOther = true
			if a.Phase != "PLANNING" {
				t.Errorf("Expected other_spec.md to be in PLANNING phase, got %q", a.Phase)
			}
		}
		if a.Title == "AGENTS.md" {
			foundAgents = true
			if a.Phase != "ONBOARDING" {
				t.Errorf("Expected AGENTS.md to be in ONBOARDING phase, got %q", a.Phase)
			}
		}
	}

	if !foundPlanning {
		t.Error("Expected to discover planning artifact epic_planning_spec.md")
	}
	if !foundSpec {
		t.Error("Expected to discover spec artifact prd.md")
	}
	if !foundBrief {
		t.Error("Expected to discover brief artifact brief.md")
	}
	if !foundOther {
		t.Error("Expected to discover other artifact other_spec.md")
	}
	if !foundAgents {
		t.Error("Expected to discover root instruction artifact AGENTS.md")
	}
}

func TestSyncProjectArtifactsFromDisk_CaseInsensitiveRootInstructions(t *testing.T) {
	tempDir := t.TempDir()
	os.Setenv("ARTIFACT_SERVICE_TYPE", "FS")
	os.Setenv("WORKSPACE_MOUNT_PATH", tempDir)
	defer os.Unsetenv("ARTIFACT_SERVICE_TYPE")
	defer os.Unsetenv("WORKSPACE_MOUNT_PATH")

	origArtifactStore := ArtifactStore
	origStore := db.ProjectStoreInstance
	origMessageStore := agent.Store

	mArtStore := &mockSyncArtifactStore{}
	ArtifactStore = mArtStore
	mProjStore := &mockSyncProjectStore{}
	db.ProjectStoreInstance = mProjStore
	agent.Store = &safeDummyMessageStore{}

	defer func() {
		ArtifactStore = origArtifactStore
		db.ProjectStoreInstance = origStore
		agent.Store = origMessageStore
	}()

	orgID := "org-case-test"
	userID := "user-case-test"
	projectID := "proj-case-test"
	specLoc := "docs/spec"

	wsRoot := filepath.Join(tempDir, orgID, userID, projectID)
	if err := os.MkdirAll(filepath.Join(wsRoot, specLoc), 0755); err != nil {
		t.Fatalf("Failed to create workspace dirs: %v", err)
	}

	// Create root files with varying casing
	if err := os.WriteFile(filepath.Join(wsRoot, "agents.md"), []byte("lowercase agents"), 0644); err != nil {
		t.Fatalf("Failed to write agents.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wsRoot, "Project-Context.md"), []byte("mixed case context"), 0644); err != nil {
		t.Fatalf("Failed to write Project-Context.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wsRoot, "PROJECT-STATUS.MD"), []byte("uppercase status"), 0644); err != nil {
		t.Fatalf("Failed to write PROJECT-STATUS.MD: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wsRoot, specLoc, "spec.md"), []byte("some spec"), 0644); err != nil {
		t.Fatalf("Failed to write spec.md: %v", err)
	}

	proj := &db.Project{
		ID: projectID,
		AppIntegrations: []db.ProjectIntegration{
			{
				IntegrationID: "github",
				Enabled:       true,
				Settings: map[string]interface{}{
					"specLocation": specLoc,
				},
			},
		},
	}
	mProjStore.project = proj

	_, _, err := SyncProjectArtifactsFromDisk(context.Background(), orgID, userID, proj, true)
	if err != nil {
		t.Fatalf("SyncProjectArtifactsFromDisk failed: %v", err)
	}
	if len(mArtStore.discovered) != 4 {
		t.Errorf("Expected 4 discovered artifacts, got %d", len(mArtStore.discovered))
	}

	foundLowercaseAgents := false
	foundMixedContext := false
	foundUpperStatus := false
	foundSpec := false

	for _, a := range mArtStore.discovered {
		if a.Title == "agents.md" {
			foundLowercaseAgents = true
			if a.Phase != "ONBOARDING" {
				t.Errorf("Expected agents.md in ONBOARDING phase, got %q", a.Phase)
			}
			if a.ArtifactType != db.ArtifactTypeProjectContext {
				t.Errorf("Expected agents.md type %q, got %q", db.ArtifactTypeProjectContext, a.ArtifactType)
			}
		}
		if a.Title == "Project-Context.md" {
			foundMixedContext = true
			if a.Phase != "ONBOARDING" {
				t.Errorf("Expected Project-Context.md in ONBOARDING phase, got %q", a.Phase)
			}
			if a.ArtifactType != db.ArtifactTypeProjectContext {
				t.Errorf("Expected Project-Context.md type %q, got %q", db.ArtifactTypeProjectContext, a.ArtifactType)
			}
		}
		if a.Title == "PROJECT-STATUS.MD" {
			foundUpperStatus = true
			if a.Phase != "ONBOARDING" {
				t.Errorf("Expected PROJECT-STATUS.MD in ONBOARDING phase, got %q", a.Phase)
			}
			if a.ArtifactType != db.ArtifactTypeProjectStatus {
				t.Errorf("Expected PROJECT-STATUS.MD type %q, got %q", db.ArtifactTypeProjectStatus, a.ArtifactType)
			}
		}
		if a.Title == "spec.md" {
			foundSpec = true
		}
	}

	if !foundLowercaseAgents {
		t.Error("Expected to discover lowercase agents.md")
	}
	if !foundMixedContext {
		t.Error("Expected to discover mixed-case Project-Context.md")
	}
	if !foundUpperStatus {
		t.Error("Expected to discover uppercase PROJECT-STATUS.MD")
	}
	if !foundSpec {
		t.Error("Expected to discover spec.md")
	}
}

func TestSyncProjectArtifactsFromDisk_DeepRecursiveDocsAndPatterns(t *testing.T) {
	tempDir := t.TempDir()
	os.Setenv("ARTIFACT_SERVICE_TYPE", "FS")
	os.Setenv("WORKSPACE_MOUNT_PATH", tempDir)
	defer os.Unsetenv("ARTIFACT_SERVICE_TYPE")
	defer os.Unsetenv("WORKSPACE_MOUNT_PATH")

	origArtifactStore := ArtifactStore
	origStore := db.ProjectStoreInstance
	origMessageStore := agent.Store

	mArtStore := &mockSyncArtifactStore{}
	ArtifactStore = mArtStore
	mProjStore := &mockSyncProjectStore{}
	db.ProjectStoreInstance = mProjStore
	agent.Store = &safeDummyMessageStore{}

	defer func() {
		ArtifactStore = origArtifactStore
		db.ProjectStoreInstance = origStore
		agent.Store = origMessageStore
	}()

	orgID := "org-deep-test"
	userID := "user-deep-test"
	projectID := "proj-deep-test"

	wsRoot := filepath.Join(tempDir, orgID, userID, projectID)

	// Create 4-level deep directory structure under docs/
	deepDir := filepath.Join(wsRoot, "docs", "spec", "plans", "architecture", "v1", "sub")
	if err := os.MkdirAll(deepDir, 0755); err != nil {
		t.Fatalf("Failed to create deep dirs: %v", err)
	}
	prdDir := filepath.Join(wsRoot, "docs", "spec", "plans", "prds")
	if err := os.MkdirAll(prdDir, 0755); err != nil {
		t.Fatalf("Failed to create prd dir: %v", err)
	}
	storyDir := filepath.Join(wsRoot, "docs", "spec", "implementation", "stories")
	if err := os.MkdirAll(storyDir, 0755); err != nil {
		t.Fatalf("Failed to create story dir: %v", err)
	}

	// 1. Deep nested architecture file (4+ levels)
	if err := os.WriteFile(filepath.Join(deepDir, "architecture-2026.md"), []byte("deep arch"), 0644); err != nil {
		t.Fatalf("Failed to write deep arch file: %v", err)
	}
	// 2. PRD with pattern & date
	if err := os.WriteFile(filepath.Join(prdDir, "prd-checkout-flow-2026.md"), []byte("prd"), 0644); err != nil {
		t.Fatalf("Failed to write prd file: %v", err)
	}
	// 3. Brief with date at docs/ root
	if err := os.WriteFile(filepath.Join(wsRoot, "docs", "brief-apf-2026-08-22.md"), []byte("brief"), 0644); err != nil {
		t.Fatalf("Failed to write brief file: %v", err)
	}
	// 4. Retro with date
	if err := os.WriteFile(filepath.Join(wsRoot, "docs", "epic-1-retro-2026-07-11.md"), []byte("retro"), 0644); err != nil {
		t.Fatalf("Failed to write retro file: %v", err)
	}
	// 5. Sprint status with YAML extension under docs/
	if err := os.WriteFile(filepath.Join(wsRoot, "docs", "sprint-status.yaml"), []byte("sprint: active"), 0644); err != nil {
		t.Fatalf("Failed to write sprint yaml file: %v", err)
	}
	// 6. Story with YAML extension
	if err := os.WriteFile(filepath.Join(storyDir, "1-1-login.yaml"), []byte("story: login"), 0644); err != nil {
		t.Fatalf("Failed to write story yaml file: %v", err)
	}
	// 7. Root-level sprint-status.yaml
	if err := os.WriteFile(filepath.Join(wsRoot, "sprint-status.yaml"), []byte("root sprint"), 0644); err != nil {
		t.Fatalf("Failed to write root sprint yaml file: %v", err)
	}

	// Hidden files that MUST be ignored
	if err := os.WriteFile(filepath.Join(wsRoot, "docs", ".DS_Store"), []byte("junk"), 0644); err != nil {
		t.Fatalf("Failed to write .DS_Store: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wsRoot, ".hidden_root_file"), []byte("hidden"), 0644); err != nil {
		t.Fatalf("Failed to write .hidden_root_file: %v", err)
	}

	proj := &db.Project{
		ID: projectID,
	}
	mProjStore.project = proj

	_, _, err := SyncProjectArtifactsFromDisk(context.Background(), orgID, userID, proj, true)
	if err != nil {
		t.Fatalf("SyncProjectArtifactsFromDisk failed: %v", err)
	}

	// Verify exactly 7 valid artifacts were discovered (hidden files ignored)
	if len(mArtStore.discovered) != 7 {
		t.Fatalf("Expected exactly 7 discovered artifacts, got %d: %+v", len(mArtStore.discovered), mArtStore.discovered)
	}

	discoveredMap := make(map[string]db.Artifact)
	for _, a := range mArtStore.discovered {
		discoveredMap[a.Path] = a
	}

	// Check Deep Architecture file
	archPath := filepath.Join("docs", "spec", "plans", "architecture", "v1", "sub", "architecture-2026.md")
	if archArt, ok := discoveredMap[archPath]; !ok {
		t.Errorf("Missing deep architecture artifact %q", archPath)
	} else {
		if archArt.Phase != "SOLUTIONING" {
			t.Errorf("Expected deep arch Phase 'SOLUTIONING', got %q", archArt.Phase)
		}
		if archArt.ArtifactType != db.ArtifactTypeArchitecture {
			t.Errorf("Expected deep arch Type %q, got %q", db.ArtifactTypeArchitecture, archArt.ArtifactType)
		}
	}

	// Check PRD with pattern & date
	prdPath := filepath.Join("docs", "spec", "plans", "prds", "prd-checkout-flow-2026.md")
	if prdArt, ok := discoveredMap[prdPath]; !ok {
		t.Errorf("Missing PRD artifact %q", prdPath)
	} else {
		if prdArt.Phase != "PLANNING" {
			t.Errorf("Expected PRD Phase 'PLANNING', got %q", prdArt.Phase)
		}
		if prdArt.ArtifactType != db.ArtifactTypePrd {
			t.Errorf("Expected PRD Type %q, got %q", db.ArtifactTypePrd, prdArt.ArtifactType)
		}
	}

	// Check Brief with date at docs/ root
	briefPath := filepath.Join("docs", "brief-apf-2026-08-22.md")
	if briefArt, ok := discoveredMap[briefPath]; !ok {
		t.Errorf("Missing brief artifact %q", briefPath)
	} else {
		if briefArt.Phase != "ANALYSIS" {
			t.Errorf("Expected Brief Phase 'ANALYSIS', got %q", briefArt.Phase)
		}
		if briefArt.ArtifactType != db.ArtifactTypeProductBrief {
			t.Errorf("Expected Brief Type %q, got %q", db.ArtifactTypeProductBrief, briefArt.ArtifactType)
		}
	}

	// Check Retro with date
	retroPath := filepath.Join("docs", "epic-1-retro-2026-07-11.md")
	if retroArt, ok := discoveredMap[retroPath]; !ok {
		t.Errorf("Missing retro artifact %q", retroPath)
	} else {
		if retroArt.Phase != "IMPLEMENTATION" {
			t.Errorf("Expected Retro Phase 'IMPLEMENTATION', got %q", retroArt.Phase)
		}
		if retroArt.ArtifactType != db.ArtifactTypeRetrospective {
			t.Errorf("Expected Retro Type %q, got %q", db.ArtifactTypeRetrospective, retroArt.ArtifactType)
		}
	}

	// Check YAML story
	storyPath := filepath.Join("docs", "spec", "implementation", "stories", "1-1-login.yaml")
	if storyArt, ok := discoveredMap[storyPath]; !ok {
		t.Errorf("Missing YAML story artifact %q", storyPath)
	} else {
		if storyArt.Phase != "IMPLEMENTATION" {
			t.Errorf("Expected Story Phase 'IMPLEMENTATION', got %q", storyArt.Phase)
		}
		if storyArt.ArtifactType != db.ArtifactTypeStory {
			t.Errorf("Expected Story Type %q, got %q", db.ArtifactTypeStory, storyArt.ArtifactType)
		}
	}

	// Check Root sprint-status.yaml
	if rootSprintArt, ok := discoveredMap["sprint-status.yaml"]; !ok {
		t.Errorf("Missing root sprint-status.yaml")
	} else {
		if rootSprintArt.Phase != "IMPLEMENTATION" {
			t.Errorf("Expected Root sprint-status Phase 'IMPLEMENTATION', got %q", rootSprintArt.Phase)
		}
	}
}
