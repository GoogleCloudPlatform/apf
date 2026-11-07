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

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"google.com/apf/api/agent"
	"google.com/apf/api/core"
	"google.com/apf/api/db"
)

type mockHealthProjectStore struct {
	db.ProjectStore
	project *db.Project
	updated *db.Project
	err     error
}

func (m *mockHealthProjectStore) GetProject(ctx context.Context, organizationID string, id string) (*db.Project, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.project == nil {
		return nil, errors.New("project not found")
	}
	return m.project, nil
}

func (m *mockHealthProjectStore) UpdateProject(ctx context.Context, organizationID string, p *db.Project) error {
	m.updated = p
	return nil
}

type mockHealthArtifactStore struct {
	db.ArtifactStore
	artifacts []db.Artifact
}

func (m *mockHealthArtifactStore) ListArtifactsForProject(ctx context.Context, organizationID string, projectId string) ([]db.Artifact, error) {
	return m.artifacts, nil
}

func (m *mockHealthArtifactStore) SyncArtifacts(ctx context.Context, organizationID string, projectId string, discovered []db.Artifact) (int, int, error) {
	m.artifacts = discovered
	return len(discovered), 0, nil
}

func (m *mockHealthArtifactStore) CreateArtifact(ctx context.Context, organizationID string, a *db.Artifact) error {
	m.artifacts = append(m.artifacts, *a)
	return nil
}

func (m *mockHealthArtifactStore) UpdateArtifact(ctx context.Context, organizationID string, a *db.Artifact) error {
	return nil
}

func (m *mockHealthArtifactStore) DeleteArtifact(ctx context.Context, organizationID string, artifactId string) error {
	return nil
}

type mockHealthMessageStore struct {
	db.MessageStore
	created []*db.Message
}

func (m *mockHealthMessageStore) CreateMessage(ctx context.Context, organizationID string, projectId string, userId string, msg *db.Message) (*db.Message, error) {
	m.created = append(m.created, msg)
	return msg, nil
}

func TestProjectHealthCheckHandler(t *testing.T) {
	origStore := db.ProjectStoreInstance
	origArtifactStore := ArtifactStore
	origMessageStore := agent.Store
	defer func() {
		db.ProjectStoreInstance = origStore
		ArtifactStore = origArtifactStore
		agent.Store = origMessageStore
	}()

	mockProj := &db.Project{
		ID: "proj123",
	}
	mockArts := []db.Artifact{
		{
			ID:        "art1",
			ProjectId: "proj123",
			Phase:     "ANALYSIS",
			Status:    db.ArtifactStatusComplete,
			Required:  true,
		},
	}

	mProjectStore := &mockHealthProjectStore{project: mockProj}
	mArtifactStore := &mockHealthArtifactStore{artifacts: mockArts}
	mMessageStore := &mockHealthMessageStore{}

	db.ProjectStoreInstance = mProjectStore
	ArtifactStore = mArtifactStore
	agent.Store = mMessageStore

	// Set up temporary directories for workspace mount and BMAD root
	tempMount := t.TempDir()
	tempBmadRoot := t.TempDir()

	origAppConfig := AppConfig
	AppConfig = &core.Configuration{
		WorkspaceMountPath: tempMount,
		SimulatedUserID:    "user123",
	}
	defer func() { AppConfig = origAppConfig }()

	t.Setenv("BMAD_ROOT_PATH", tempBmadRoot)

	// Create dummy BMAD root structure
	bmadDir := filepath.Join(tempBmadRoot, "_bmad")
	bmadCoreDir := filepath.Join(tempBmadRoot, "_bmad", "core")
	skillsDir := filepath.Join(tempBmadRoot, "skills", "bmad-test-skill")
	if err := os.MkdirAll(bmadCoreDir, 0755); err != nil {
		t.Fatalf("Failed to create bmadCoreDir: %v", err)
	}
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		t.Fatalf("Failed to create skillsDir: %v", err)
	}
	initialToml := `planning_artifacts = "{project-root}/docs/spec/planning-artifacts"
implementation_artifacts = "{project-root}/_bmad-output/implementation-artifacts"`
	if err := os.WriteFile(filepath.Join(bmadDir, "config.toml"), []byte(initialToml), 0644); err != nil {
		t.Fatalf("Failed to write config.toml: %v", err)
	}
	initialYaml := `output_folder: "{project-root}"`
	if err := os.WriteFile(filepath.Join(bmadCoreDir, "config.yaml"), []byte(initialYaml), 0644); err != nil {
		t.Fatalf("Failed to write config.yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillsDir, "SKILL.md"), []byte("# BMAD Skill"), 0644); err != nil {
		t.Fatalf("Failed to write SKILL.md: %v", err)
	}

	// Create existing custom skill in project workspace
	projRoot := filepath.Join(tempMount, "test-organization", "user123", "proj123")
	customSkillDir := filepath.Join(projRoot, ".agents", "skills", "custom-user-skill")
	if err := os.MkdirAll(customSkillDir, 0755); err != nil {
		t.Fatalf("Failed to create customSkillDir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(customSkillDir, "SKILL.md"), []byte("# Custom Skill"), 0644); err != nil {
		t.Fatalf("Failed to write custom SKILL.md: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/proj123/health-check", nil)
	req.SetPathValue("id", "proj123")
	claims := &core.UserClaims{Subject: "user123"}
	req = withTestCtx(req, claims)

	w := httptest.NewRecorder()
	ProjectHealthCheckHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp Response
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Status != "SUCCESS" {
		t.Errorf("Expected status SUCCESS, got %s", resp.Status)
	}

	if mProjectStore.updated == nil {
		t.Fatal("Expected project update, got nil")
	}

	gates := *mProjectStore.updated.GovernanceGates
	if gates["ANALYSIS"] != "Complete" {
		t.Errorf("Expected ANALYSIS gate to be Complete, got %s", gates["ANALYSIS"])
	}

	// Verify .agents/_bmad/config.toml was injected and updated
	injectedBmadConfig := filepath.Join(projRoot, ".agents", "_bmad", "config.toml")
	tomlBytes, err := os.ReadFile(injectedBmadConfig)
	if err != nil {
		t.Fatalf("Expected injected _bmad config at %s, but got error: %v", injectedBmadConfig, err)
	}
	tomlStr := string(tomlBytes)
	if !containsSubstring(tomlStr, `planning_artifacts = "{project-root}/docs/spec/plans"`) {
		t.Errorf("Expected config.toml to contain updated planning_artifacts, got: %s", tomlStr)
	}
	if !containsSubstring(tomlStr, `implementation_artifacts = "{project-root}/docs/spec/implementation"`) {
		t.Errorf("Expected config.toml to contain updated implementation_artifacts, got: %s", tomlStr)
	}
	if !containsSubstring(tomlStr, `output_folder = "{project-root}/docs/spec"`) {
		t.Errorf("Expected config.toml to contain updated output_folder, got: %s", tomlStr)
	}

	// Verify .agents/_bmad/core/config.yaml was injected and updated
	injectedYamlConfig := filepath.Join(projRoot, ".agents", "_bmad", "core", "config.yaml")
	yamlBytes, err := os.ReadFile(injectedYamlConfig)
	if err != nil {
		t.Fatalf("Expected injected core/config.yaml at %s, but got error: %v", injectedYamlConfig, err)
	}
	yamlStr := string(yamlBytes)
	if !containsSubstring(yamlStr, `output_folder: "{project-root}/docs/spec"`) {
		t.Errorf("Expected config.yaml to contain updated output_folder, got: %s", yamlStr)
	}

	// Verify .agents/skills/bmad-test-skill/SKILL.md was injected
	injectedSkillFile := filepath.Join(projRoot, ".agents", "skills", "bmad-test-skill", "SKILL.md")
	if _, err := os.Stat(injectedSkillFile); os.IsNotExist(err) {
		t.Errorf("Expected injected BMAD skill at %s, but not found", injectedSkillFile)
	}

	// Verify pre-existing .agents/skills/custom-user-skill/SKILL.md was preserved
	preservedSkillFile := filepath.Join(projRoot, ".agents", "skills", "custom-user-skill", "SKILL.md")
	if _, err := os.Stat(preservedSkillFile); os.IsNotExist(err) {
		t.Errorf("Expected preserved custom skill at %s, but not found", preservedSkillFile)
	}
}

func containsSubstring(s, sub string) bool {
	return filepath.Clean(s) != "" && (s == sub || len(s) > len(sub) && (s[:len(sub)] == sub || findSubstring(s, sub)))
}

func findSubstring(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestGetWorkspaceMountPath_PanicOnEmpty(t *testing.T) {
	origAppConfig := AppConfig
	defer func() { AppConfig = origAppConfig }()

	AppConfig = &core.Configuration{WorkspaceMountPath: ""}

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected getWorkspaceMountPath to panic when WorkspaceMountPath is empty, but it did not")
		}
	}()

	_ = getWorkspaceMountPath()
}

func TestInjectConfigUpdates_RequiresProjectInstance(t *testing.T) {
	origStore := db.ProjectStoreInstance
	defer func() { db.ProjectStoreInstance = origStore }()

	db.ProjectStoreInstance = nil
	ctx := context.Background()

	err := injectConfigUpdates(ctx, "org", "user", "proj")
	if err == nil {
		t.Errorf("Expected injectConfigUpdates to fail when ProjectStoreInstance is nil, but it succeeded")
	}
}


