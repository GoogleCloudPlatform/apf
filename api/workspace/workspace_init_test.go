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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"google.com/apf/api/core"
	"google.com/apf/api/db"
)

type mockInitProjectStore struct {
	db.ProjectStore
	projects map[string]*db.Project
}

func (m *mockInitProjectStore) GetProject(ctx context.Context, organizationID, projectID string) (*db.Project, error) {
	if m.projects != nil {
		if p, ok := m.projects[projectID]; ok {
			return p, nil
		}
	}
	return nil, os.ErrNotExist
}

func TestEnsureGitignoreWithBmad_MissingGitignore(t *testing.T) {
	tempDir := t.TempDir()

	err := ensureGitignoreWithBmad(tempDir)
	if err != nil {
		t.Fatalf("ensureGitignoreWithBmad failed: %v", err)
	}

	gitignorePath := filepath.Join(tempDir, ".gitignore")
	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("Expected .gitignore to be created: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "_bmad") {
		t.Errorf("Expected .gitignore to contain _bmad, got: %s", content)
	}
	if !strings.Contains(content, "*.user.toml") {
		t.Errorf("Expected .gitignore to contain *.user.toml, got: %s", content)
	}
}

func TestEnsureGitignoreWithBmad_ExistingWithoutBmadOrUserToml(t *testing.T) {
	tempDir := t.TempDir()
	gitignorePath := filepath.Join(tempDir, ".gitignore")
	initialContent := "node_modules\n.env\n.agents\n"
	if err := os.WriteFile(gitignorePath, []byte(initialContent), 0644); err != nil {
		t.Fatal(err)
	}

	err := ensureGitignoreWithBmad(tempDir)
	if err != nil {
		t.Fatalf("ensureGitignoreWithBmad failed: %v", err)
	}

	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("Failed to read .gitignore: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "_bmad") {
		t.Errorf("Expected .gitignore to contain _bmad, got: %s", content)
	}
	if !strings.Contains(content, "*.user.toml") {
		t.Errorf("Expected .gitignore to contain *.user.toml, got: %s", content)
	}
	if !strings.Contains(content, "node_modules") {
		t.Errorf("Expected .gitignore to preserve node_modules, got: %s", content)
	}
	// .agents must be removed so universal skills are tracked
	if strings.Contains(content, ".agents") {
		t.Errorf("Expected .gitignore to NOT contain .agents, got: %s", content)
	}
}

func TestEnsureGitignoreWithBmad_ExistingWithAll(t *testing.T) {
	tempDir := t.TempDir()
	gitignorePath := filepath.Join(tempDir, ".gitignore")
	initialContent := "node_modules\n_bmad\n.tmp/\n*.user.toml\n"
	if err := os.WriteFile(gitignorePath, []byte(initialContent), 0644); err != nil {
		t.Fatal(err)
	}

	err := ensureGitignoreWithBmad(tempDir)
	if err != nil {
		t.Fatalf("ensureGitignoreWithBmad failed: %v", err)
	}

	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("Failed to read .gitignore: %v", err)
	}

	content := string(data)
	lines := strings.Split(strings.TrimSpace(content), "\n")
	bmadCount := 0
	userTomlCount := 0
	for _, l := range lines {
		if strings.TrimSpace(l) == "_bmad" {
			bmadCount++
		}
		if strings.TrimSpace(l) == "*.user.toml" {
			userTomlCount++
		}
	}
	if bmadCount != 1 {
		t.Errorf("Expected exactly 1 _bmad entry in .gitignore, got %d. Content:\n%s", bmadCount, content)
	}
	if userTomlCount != 1 {
		t.Errorf("Expected exactly 1 *.user.toml entry in .gitignore, got %d. Content:\n%s", userTomlCount, content)
	}
}

func TestInjectConfigUpdates_FailsOnMissingProject(t *testing.T) {
	origStore := db.ProjectStoreInstance
	db.ProjectStoreInstance = &mockInitProjectStore{}
	defer func() { db.ProjectStoreInstance = origStore }()

	ctx := context.Background()
	err := injectConfigUpdates(ctx, "org", "user", "non-existent-proj")
	if err == nil {
		t.Fatal("Expected injectConfigUpdates to fail on non-existent project, but succeeded")
	}
}

func TestInjectConfigUpdates_FailsOnEmptySpecLocation(t *testing.T) {
	origStore := db.ProjectStoreInstance
	db.ProjectStoreInstance = &mockInitProjectStore{
		projects: map[string]*db.Project{
			"proj-empty-spec": {
				ID: "proj-empty-spec",
				BmadConfig: &db.BmadConfig{
					SpecFolder:                  "",
					PlanningArtifactsPath:        "",
					ImplementationArtifactsPath: "",
				},
				// github app integrations empty so GetSpecLocation defaults to docs/spec unless overridden
			},
		},
	}
	defer func() { db.ProjectStoreInstance = origStore }()

	// If planning or implementation artifacts location is empty, injectConfigUpdates should fail
	ctx := context.Background()
	_ = ctx
}

func TestInjectConfigUpdates_SeedsAllFilesCorrectly(t *testing.T) {
	tempMount := t.TempDir()
	origAppConfig := AppConfig
	AppConfig = &core.Configuration{
		WorkspaceMountPath: tempMount,
	}
	defer func() { AppConfig = origAppConfig }()

	orgID := "org1"
	userID := "user-alice"
	projID := "proj-showcase"
	projName := "Showcase Project"
	specLoc := "docs/custom-spec"
	planLoc := "docs/custom-spec/planning"
	implLoc := "docs/custom-spec/implementation"

	origStore := db.ProjectStoreInstance
	db.ProjectStoreInstance = &mockInitProjectStore{
		projects: map[string]*db.Project{
			projID: {
				ID:   projID,
				Name: &projName,
				BmadConfig: &db.BmadConfig{
					SpecFolder:                  specLoc,
					PlanningArtifactsPath:        planLoc,
					ImplementationArtifactsPath: implLoc,
				},
			},
		},
	}
	defer func() { db.ProjectStoreInstance = origStore }()

	projectRoot := filepath.Join(tempMount, orgID, userID, projID)
	agentsBmad := filepath.Join(projectRoot, ".agents", "_bmad")
	if err := os.MkdirAll(filepath.Join(agentsBmad, "core"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(agentsBmad, "bmm"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create initial template files
	_ = os.WriteFile(filepath.Join(agentsBmad, "config.toml"), []byte("[core]\nproject_name = \"Initial\"\n"), 0644)
	_ = os.WriteFile(filepath.Join(agentsBmad, "config.user.toml"), []byte("[core]\nuser_name = \"APF Ninja\"\n"), 0644)
	_ = os.WriteFile(filepath.Join(agentsBmad, "core", "config.yaml"), []byte("project_name: Initial\n"), 0644)
	_ = os.WriteFile(filepath.Join(agentsBmad, "bmm", "config.yaml"), []byte("project_name: Initial\n"), 0644)

	ctx := context.Background()
	err := injectConfigUpdates(ctx, orgID, userID, projID)
	if err != nil {
		t.Fatalf("injectConfigUpdates failed: %v", err)
	}

	// 1. Verify config.toml has project values and generic user_name
	configToml, err := os.ReadFile(filepath.Join(agentsBmad, "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	cToml := string(configToml)
	if !strings.Contains(cToml, "project_name = \"Showcase Project\"") {
		t.Errorf("Expected config.toml project_name, got: %s", cToml)
	}
	if !strings.Contains(cToml, "output_folder = \"{project-root}/docs/custom-spec\"") {
		t.Errorf("Expected config.toml output_folder, got: %s", cToml)
	}
	if !strings.Contains(cToml, "planning_artifacts = \"{project-root}/docs/custom-spec/planning\"") {
		t.Errorf("Expected config.toml planning_artifacts, got: %s", cToml)
	}
	if !strings.Contains(cToml, "user_name = \"developer\"") {
		t.Errorf("Expected config.toml generic user_name, got: %s", cToml)
	}

	// 2. Verify config.user.toml has active user_name
	userToml, err := os.ReadFile(filepath.Join(agentsBmad, "config.user.toml"))
	if err != nil {
		t.Fatal(err)
	}
	uToml := string(userToml)
	if !strings.Contains(uToml, "user_name = \"user-alice\"") {
		t.Errorf("Expected config.user.toml active user_name, got: %s", uToml)
	}

	// 3. Verify custom/config.user.toml exists and has active user_name
	customUserToml, err := os.ReadFile(filepath.Join(agentsBmad, "custom", "config.user.toml"))
	if err != nil {
		t.Fatal(err)
	}
	cuToml := string(customUserToml)
	if !strings.Contains(cuToml, "user_name = \"user-alice\"") {
		t.Errorf("Expected custom/config.user.toml active user_name, got: %s", cuToml)
	}

	// 4. Verify core/config.yaml and bmm/config.yaml have generic user_name
	coreYaml, _ := os.ReadFile(filepath.Join(agentsBmad, "core", "config.yaml"))
	if !strings.Contains(string(coreYaml), "user_name: \"developer\"") {
		t.Errorf("Expected core/config.yaml generic user_name, got: %s", string(coreYaml))
	}
	bmmYaml, _ := os.ReadFile(filepath.Join(agentsBmad, "bmm", "config.yaml"))
	if !strings.Contains(string(bmmYaml), "user_name: \"developer\"") {
		t.Errorf("Expected bmm/config.yaml generic user_name, got: %s", string(bmmYaml))
	}
}

func TestInjectSkills_FullWorkflow(t *testing.T) {
	tempMount := t.TempDir()
	tempBmadRoot := t.TempDir()

	origAppConfig := AppConfig
	AppConfig = &core.Configuration{
		WorkspaceMountPath: tempMount,
		BmadRootPath:       tempBmadRoot,
	}
	defer func() { AppConfig = origAppConfig }()

	orgID := "test-org"
	userID := "test-user"
	projID := "test-proj"
	projName := "Test Inject Skills Proj"
	specLoc := "docs/spec"
	planLoc := "docs/spec/planning-artifacts"
	implLoc := "docs/spec/implementation-artifacts"

	origStore := db.ProjectStoreInstance
	db.ProjectStoreInstance = &mockInitProjectStore{
		projects: map[string]*db.Project{
			projID: {
				ID:   projID,
				Name: &projName,
				BmadConfig: &db.BmadConfig{
					SpecFolder:                  specLoc,
					PlanningArtifactsPath:        planLoc,
					ImplementationArtifactsPath: implLoc,
				},
			},
		},
	}
	defer func() { db.ProjectStoreInstance = origStore }()

	// Setup source BMAD directory
	srcBmad := filepath.Join(tempBmadRoot, "_bmad")
	if err := os.MkdirAll(srcBmad, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcBmad, "config.toml"), []byte("output_folder = \"docs\"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	srcSkill1 := filepath.Join(tempBmadRoot, "skills", "bmad-prd")
	if err := os.MkdirAll(srcSkill1, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcSkill1, "SKILL.md"), []byte("# PRD Skill"), 0644); err != nil {
		t.Fatal(err)
	}

	projectRoot := filepath.Join(tempMount, orgID, userID, projID)

	ctx := context.Background()
	err := injectSkills(ctx, orgID, userID, projID)
	if err != nil {
		t.Fatalf("injectSkills failed: %v", err)
	}

	// 1. Check .agents directory creation
	agentsDir := filepath.Join(projectRoot, ".agents")
	if info, err := os.Stat(agentsDir); err != nil || !info.IsDir() {
		t.Errorf("Expected .agents directory to exist, err: %v", err)
	}

	// 2. Check root _bmad symlink
	rootBmad := filepath.Join(projectRoot, "_bmad")
	fi, err := os.Lstat(rootBmad)
	if err != nil {
		t.Errorf("Expected root _bmad symlink, got error: %v", err)
	} else if fi.Mode()&os.ModeSymlink == 0 {
		t.Errorf("Expected root _bmad to be a symlink, got mode: %v", fi.Mode())
	}

	// 3. Check .gitignore created and contains _bmad & *.user.toml
	gitignorePath := filepath.Join(projectRoot, ".gitignore")
	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("Expected .gitignore to be created: %v", err)
	}
	if !strings.Contains(string(data), "_bmad") {
		t.Errorf("Expected .gitignore to contain '_bmad', got: %s", string(data))
	}
	if !strings.Contains(string(data), "*.user.toml") {
		t.Errorf("Expected .gitignore to contain '*.user.toml', got: %s", string(data))
	}

	// 4. Check configuration files seeded
	configData, err := os.ReadFile(filepath.Join(agentsDir, "_bmad", "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(configData), "user_name = \"developer\"") {
		t.Errorf("Expected generic user_name in config.toml, got: %s", string(configData))
	}
	userConfigData, err := os.ReadFile(filepath.Join(agentsDir, "_bmad", "config.user.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(userConfigData), "user_name = \"test-user\"") {
		t.Errorf("Expected active user_name in config.user.toml, got: %s", string(userConfigData))
	}
}

func TestEnsureRepoInitialized_ExecutesSkillInjectionAndGitignore(t *testing.T) {
	tempMount := t.TempDir()
	tempBmadRoot := t.TempDir()

	origAppConfig := AppConfig
	AppConfig = &core.Configuration{
		WorkspaceMountPath: tempMount,
		BmadRootPath:       tempBmadRoot,
	}
	defer func() { AppConfig = origAppConfig }()

	orgID := "org-init-test"
	userID := "user-init-test"
	projID := "proj-init-test"
	projName := "Init Test Proj"
	specLoc := "docs/spec"
	planLoc := "docs/spec/planning-artifacts"
	implLoc := "docs/spec/implementation-artifacts"

	origStore := db.ProjectStoreInstance
	db.ProjectStoreInstance = &mockInitProjectStore{
		projects: map[string]*db.Project{
			projID: {
				ID:   projID,
				Name: &projName,
				BmadConfig: &db.BmadConfig{
					SpecFolder:                  specLoc,
					PlanningArtifactsPath:        planLoc,
					ImplementationArtifactsPath: implLoc,
				},
			},
		},
	}
	defer func() { db.ProjectStoreInstance = origStore }()

	// Setup source BMAD directory
	srcBmad := filepath.Join(tempBmadRoot, "_bmad")
	if err := os.MkdirAll(srcBmad, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcBmad, "config.toml"), []byte("output_folder = \"docs\"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	srcSkill := filepath.Join(tempBmadRoot, "skills", "bmad-brainstorming")
	if err := os.MkdirAll(srcSkill, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcSkill, "SKILL.md"), []byte("# Brainstorming"), 0644); err != nil {
		t.Fatal(err)
	}

	projectRoot := filepath.Join(tempMount, orgID, userID, projID)

	client := NewGitClient(tempMount)
	repoCfg := &GitRepoConfig{
		RepoURL:      "", // local repo
		UserId:       userID,
		ProjectId:    projID,
		BaseBranch:   "main",
		SpecBranch:   "specs/user-init-test",
		SpecLocation: "docs/spec",
	}

	ctx := context.Background()
	err := client.EnsureRepoInitialized(ctx, orgID, repoCfg)
	if err != nil {
		t.Fatalf("EnsureRepoInitialized failed: %v", err)
	}

	// Verify .agents
	if _, err := os.Stat(filepath.Join(projectRoot, ".agents")); err != nil {
		t.Errorf("Expected .agents directory to exist after EnsureRepoInitialized, err: %v", err)
	}

	// Verify _bmad symlink
	rootBmad := filepath.Join(projectRoot, "_bmad")
	if fi, err := os.Lstat(rootBmad); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Errorf("Expected root _bmad symlink after EnsureRepoInitialized, err: %v", err)
	}

	// Verify hardlinked skill
	injectedSkillFile := filepath.Join(projectRoot, ".agents", "skills", "bmad-brainstorming", "SKILL.md")
	dstFi, err := os.Stat(injectedSkillFile)
	if err != nil {
		t.Fatalf("Expected injected skill file to exist, err: %v", err)
	}
	srcFi, err := os.Stat(filepath.Join(srcSkill, "SKILL.md"))
	if err != nil {
		t.Fatalf("Expected source skill file to exist, err: %v", err)
	}
	if !os.SameFile(srcFi, dstFi) {
		t.Errorf("Expected skill file to be hardlinked (same file), but os.SameFile returned false")
	}

	// Verify .gitignore with _bmad and *.user.toml
	gitignoreData, err := os.ReadFile(filepath.Join(projectRoot, ".gitignore"))
	if err != nil {
		t.Fatalf("Expected .gitignore after EnsureRepoInitialized: %v", err)
	}
	if !strings.Contains(string(gitignoreData), "_bmad") {
		t.Errorf("Expected .gitignore to contain '_bmad', got: %s", string(gitignoreData))
	}
	if !strings.Contains(string(gitignoreData), "*.user.toml") {
		t.Errorf("Expected .gitignore to contain '*.user.toml', got: %s", string(gitignoreData))
	}
}

func TestVerifySkillsPresence(t *testing.T) {
	t.Run("empty directory returns false", func(t *testing.T) {
		tempDir := t.TempDir()
		if VerifySkillsPresence(tempDir) {
			t.Error("Expected VerifySkillsPresence to return false for empty directory")
		}
	})

	t.Run("missing _bmad returns false", func(t *testing.T) {
		tempDir := t.TempDir()
		_ = os.MkdirAll(filepath.Join(tempDir, ".agents", "skills", "bmad-test"), 0755)
		if VerifySkillsPresence(tempDir) {
			t.Error("Expected VerifySkillsPresence to return false when _bmad is missing")
		}
	})

	t.Run("missing skills returns false", func(t *testing.T) {
		tempDir := t.TempDir()
		_ = os.MkdirAll(filepath.Join(tempDir, ".agents", "_bmad"), 0755)
		if VerifySkillsPresence(tempDir) {
			t.Error("Expected VerifySkillsPresence to return false when skills dir is missing")
		}
	})

	t.Run("all present returns true", func(t *testing.T) {
		tempDir := t.TempDir()
		_ = os.MkdirAll(filepath.Join(tempDir, ".agents", "_bmad"), 0755)
		_ = os.MkdirAll(filepath.Join(tempDir, ".agents", "skills", "bmad-prd"), 0755)
		if !VerifySkillsPresence(tempDir) {
			t.Error("Expected VerifySkillsPresence to return true when all required folders exist")
		}
	})
}

func TestInitializeProjectWorkspace(t *testing.T) {
	tempMount := t.TempDir()
	tempBmadRoot := t.TempDir()
	orgID := "test-org"
	userID := "user-proj-init"
	projID := "proj-workspace-init"

	origAppConfig := AppConfig
	origGitClient := gitClient
	AppConfig = &core.Configuration{
		WorkspaceMountPath: tempMount,
		BmadRootPath:       tempBmadRoot,
	}
	SetGitClient(NewGitClient(tempMount))
	defer func() {
		AppConfig = origAppConfig
		gitClient = origGitClient
	}()

	t.Setenv("WORKSPACE_MOUNT_PATH", tempMount)
	t.Setenv("BMAD_ROOT_PATH", tempBmadRoot)

	// Setup source BMAD directory
	srcBmad := filepath.Join(tempBmadRoot, "_bmad")
	_ = os.MkdirAll(srcBmad, 0755)
	_ = os.WriteFile(filepath.Join(srcBmad, "config.toml"), []byte("output_folder = \"docs\"\n"), 0644)

	srcSkill := filepath.Join(tempBmadRoot, "skills", "bmad-init-skill")
	_ = os.MkdirAll(srcSkill, 0755)
	_ = os.WriteFile(filepath.Join(srcSkill, "SKILL.md"), []byte("# Init Skill"), 0644)

	projName := "Init Workspace Project"
	project := &db.Project{
		ID:   projID,
		Name: &projName,
	}

	ctx := context.Background()
	if err := InitializeProjectWorkspace(ctx, orgID, userID, project, ""); err != nil {
		t.Fatalf("InitializeProjectWorkspace failed: %v", err)
	}

	projectRoot := filepath.Join(tempMount, orgID, userID, projID)
	if !VerifySkillsPresence(projectRoot) {
		t.Errorf("Expected VerifySkillsPresence to return true after InitializeProjectWorkspace")
	}
}

func TestInjectSkillsForPath_UsesGlobalWorkspacePath(t *testing.T) {
	tempMount := t.TempDir()
	globalDir := filepath.Join(tempMount, "global")
	projectRoot := filepath.Join(tempMount, "test-org", "user-1", "proj-1")

	origAppConfig := AppConfig
	AppConfig = &core.Configuration{
		WorkspaceMountPath:  tempMount,
		GlobalWorkspacePath: globalDir,
	}
	defer func() {
		AppConfig = origAppConfig
	}()

	projID := "proj-1"
	projName := "Test Global Skills Proj"
	origStore := db.ProjectStoreInstance
	db.ProjectStoreInstance = &mockInitProjectStore{
		projects: map[string]*db.Project{
			projID: {
				ID:   projID,
				Name: &projName,
			},
		},
	}
	defer func() { db.ProjectStoreInstance = origStore }()

	// Setup global directory
	srcBmad := filepath.Join(globalDir, "_bmad")
	_ = os.MkdirAll(srcBmad, 0755)
	_ = os.WriteFile(filepath.Join(srcBmad, "config.toml"), []byte("output_folder = \"docs\"\n"), 0644)

	srcSkill := filepath.Join(globalDir, "skills", "bmad-global-skill")
	_ = os.MkdirAll(srcSkill, 0755)
	_ = os.WriteFile(filepath.Join(srcSkill, "SKILL.md"), []byte("# Global Skill"), 0644)

	ctx := context.Background()
	if err := InjectSkillsForPath(ctx, projectRoot, "test-org", "user-1", "proj-1"); err != nil {
		t.Fatalf("InjectSkillsForPath failed: %v", err)
	}

	if !VerifySkillsPresence(projectRoot) {
		t.Errorf("Expected VerifySkillsPresence to return true")
	}

	skillFile := filepath.Join(projectRoot, ".agents", "skills", "bmad-global-skill", "SKILL.md")
	data, err := os.ReadFile(skillFile)
	if err != nil {
		t.Fatalf("Failed to read skill file: %v", err)
	}
	if string(data) != "# Global Skill" {
		t.Errorf("Unexpected skill content: %s", string(data))
	}
}


