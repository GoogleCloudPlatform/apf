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
	"os"
	"path/filepath"
	"strings"
	"testing"

	aitesting "google.com/apf/ai/testing"
)

func TestNewWorkspace_Success(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "workspace_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	userId := "test-user"
	projectId := "test-project"
	projectDir := filepath.Join(tempDir, "default-organization", userId, projectId)
	err = os.MkdirAll(projectDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	cfg := &Configuration{
		WorkspaceMountPath: tempDir,
	}

	mockState := &aitesting.MockState{Data: make(map[string]interface{})}
	mockState.Data["projectId"] = projectId
	mockState.Data["repoUrl"] = "https://github.com/example/repo.git"

	mockCtx := &aitesting.MockCallbackContext{
		UserId:    userId,
		SessionId: "test-session",
		StateObj:  mockState,
	}

	ws, err := NewWorkspace(mockCtx, cfg)
	if err != nil {
		t.Fatalf("NewWorkspace failed: %v", err)
	}
	if ws == nil {
		t.Fatal("Expected workspace to be non-nil")
	}

	val, err := mockCtx.State().Get(stateWorkspaceReady)
	if err != nil {
		t.Errorf("Failed to get stateWorkspaceReady: %v", err)
	}
	if val != "true" {
		t.Errorf("Expected stateWorkspaceReady to be 'true', got %v", val)
	}
}

func TestNewWorkspace_Failure_DirNotExists(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "workspace_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	cfg := &Configuration{
		WorkspaceMountPath: tempDir,
	}

	mockState := &aitesting.MockState{Data: make(map[string]interface{})}
	mockState.Data["projectId"] = "non-existent-project"
	mockState.Data["repoUrl"] = "https://github.com/example/repo.git"

	mockCtx := &aitesting.MockCallbackContext{
		UserId:    "test-user",
		SessionId: "test-session",
		StateObj:  mockState,
	}

	_, err = NewWorkspace(mockCtx, cfg)
	if err == nil {
		t.Fatal("Expected NewWorkspace to fail when directory does not exist")
	}
}

func TestNewWorkspace_CreatesBmadSymlink(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "workspace_symlink_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	userId := "test-user"
	projectId := "test-project"
	projectDir := filepath.Join(tempDir, "default-organization", userId, projectId)
	agentsBmadDir := filepath.Join(projectDir, ".agents", "_bmad")
	if err := os.MkdirAll(agentsBmadDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := &Configuration{
		WorkspaceMountPath: tempDir,
	}

	mockState := &aitesting.MockState{Data: make(map[string]interface{})}
	mockState.Data["projectId"] = projectId
	mockState.Data["repoUrl"] = "https://github.com/example/repo.git"

	mockCtx := &aitesting.MockCallbackContext{
		UserId:    userId,
		SessionId: "test-session",
		StateObj:  mockState,
	}

	_, err = NewWorkspace(mockCtx, cfg)
	if err != nil {
		t.Fatalf("NewWorkspace failed: %v", err)
	}

	rootBmad := filepath.Join(projectDir, "_bmad")
	fi, err := os.Lstat(rootBmad)
	if err != nil {
		t.Fatalf("Expected _bmad symlink at project root, got error: %v", err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Errorf("Expected _bmad to be a symlink, got mode: %v", fi.Mode())
	}
}

func TestNewWorkspace_CreatesGitignoreWithBmad_WhenMissing(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "workspace_gitignore_missing")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	userId := "test-user"
	projectId := "test-project"
	projectDir := filepath.Join(tempDir, "default-organization", userId, projectId)
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := &Configuration{
		WorkspaceMountPath: tempDir,
	}

	mockState := &aitesting.MockState{Data: make(map[string]interface{})}
	mockState.Data["projectId"] = projectId
	mockState.Data["repoUrl"] = "https://github.com/example/repo.git"

	mockCtx := &aitesting.MockCallbackContext{
		UserId:    userId,
		SessionId: "test-session",
		StateObj:  mockState,
	}

	_, err = NewWorkspace(mockCtx, cfg)
	if err != nil {
		t.Fatalf("NewWorkspace failed: %v", err)
	}

	gitignorePath := filepath.Join(projectDir, ".gitignore")
	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("Expected .gitignore to be created, got error: %v", err)
	}
	if !containsLine(string(data), "_bmad") {
		t.Errorf("Expected .gitignore to contain '_bmad', got: %s", string(data))
	}
}

func TestNewWorkspace_AppendsBmadToGitignore_WhenMissingInExisting(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "workspace_gitignore_existing")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	userId := "test-user"
	projectId := "test-project"
	projectDir := filepath.Join(tempDir, "default-organization", userId, projectId)
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Pre-create .gitignore without _bmad
	gitignorePath := filepath.Join(projectDir, ".gitignore")
	if err := os.WriteFile(gitignorePath, []byte("node_modules\n*.log\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &Configuration{
		WorkspaceMountPath: tempDir,
	}

	mockState := &aitesting.MockState{Data: make(map[string]interface{})}
	mockState.Data["projectId"] = projectId
	mockState.Data["repoUrl"] = "https://github.com/example/repo.git"

	mockCtx := &aitesting.MockCallbackContext{
		UserId:    userId,
		SessionId: "test-session",
		StateObj:  mockState,
	}

	_, err = NewWorkspace(mockCtx, cfg)
	if err != nil {
		t.Fatalf("NewWorkspace failed: %v", err)
	}

	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("Failed to read .gitignore: %v", err)
	}
	content := string(data)
	if !containsLine(content, "_bmad") {
		t.Errorf("Expected .gitignore to contain '_bmad', got: %s", content)
	}
	if !containsLine(content, "node_modules") {
		t.Errorf("Expected .gitignore to preserve 'node_modules', got: %s", content)
	}
}

func TestNewWorkspace_PreservesBmadInGitignore_WhenAlreadyPresent(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "workspace_gitignore_present")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	userId := "test-user"
	projectId := "test-project"
	projectDir := filepath.Join(tempDir, "default-organization", userId, projectId)
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Pre-create .gitignore with _bmad/, .tmp/, *.user.toml
	gitignorePath := filepath.Join(projectDir, ".gitignore")
	initialContent := "node_modules\n_bmad/\n.tmp/\n*.user.toml\n*.log\n"
	if err := os.WriteFile(gitignorePath, []byte(initialContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &Configuration{
		WorkspaceMountPath: tempDir,
	}

	mockState := &aitesting.MockState{Data: make(map[string]interface{})}
	mockState.Data["projectId"] = projectId
	mockState.Data["repoUrl"] = "https://github.com/example/repo.git"

	mockCtx := &aitesting.MockCallbackContext{
		UserId:    userId,
		SessionId: "test-session",
		StateObj:  mockState,
	}

	_, err = NewWorkspace(mockCtx, cfg)
	if err != nil {
		t.Fatalf("NewWorkspace failed: %v", err)
	}

	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("Failed to read .gitignore: %v", err)
	}
	if string(data) != initialContent {
		t.Errorf("Expected .gitignore to remain unchanged, got: %s", string(data))
	}
}

func TestNewWorkspace_SkillInjection(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "workspace_skill_inject")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	tempBmadRoot, err := os.MkdirTemp("", "bmad_root")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempBmadRoot)

	// Create dummy BMAD structure
	srcBmad := filepath.Join(tempBmadRoot, "_bmad")
	if err := os.MkdirAll(srcBmad, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcBmad, "config.toml"), []byte("planning_artifacts = \"docs\""), 0644); err != nil {
		t.Fatal(err)
	}

	srcSkill := filepath.Join(tempBmadRoot, "skills", "bmad-prd")
	if err := os.MkdirAll(srcSkill, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcSkill, "SKILL.md"), []byte("# PRD Skill"), 0644); err != nil {
		t.Fatal(err)
	}

	userId := "test-user"
	projectId := "test-project"
	projectDir := filepath.Join(tempDir, "default-organization", userId, projectId)
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := &Configuration{
		WorkspaceMountPath: tempDir,
		BmadCfg: BmadConfiguration{
			BmadRootPath: tempBmadRoot,
		},
	}

	mockState := &aitesting.MockState{Data: make(map[string]interface{})}
	mockState.Data["projectId"] = projectId
	mockState.Data["repoUrl"] = "https://github.com/example/repo.git"

	mockCtx := &aitesting.MockCallbackContext{
		UserId:    userId,
		SessionId: "test-session",
		StateObj:  mockState,
	}

	_, err = NewWorkspace(mockCtx, cfg)
	if err != nil {
		t.Fatalf("NewWorkspace failed: %v", err)
	}

	// Verify .agents/.bmad
	injectedBmad := filepath.Join(projectDir, ".agents", "_bmad", "config.toml")
	if _, err := os.Stat(injectedBmad); err != nil {
		t.Errorf("Expected injected _bmad config at %s, got: %v", injectedBmad, err)
	}

	// Verify .agents/skills/bmad-prd/SKILL.md
	injectedSkill := filepath.Join(projectDir, ".agents", "skills", "bmad-prd", "SKILL.md")
	if _, err := os.Stat(injectedSkill); err != nil {
		t.Errorf("Expected injected skill at %s, got: %v", injectedSkill, err)
	}

	// Verify root _bmad symlink
	rootBmad := filepath.Join(projectDir, "_bmad")
	fi, err := os.Lstat(rootBmad)
	if err != nil {
		t.Errorf("Expected root _bmad symlink, got error: %v", err)
	} else if fi.Mode()&os.ModeSymlink == 0 {
		t.Errorf("Expected root _bmad to be symlink, got: %v", fi.Mode())
	}

	// Verify .gitignore exists and contains _bmad and *.user.toml
	gitignoreData, err := os.ReadFile(filepath.Join(projectDir, ".gitignore"))
	if err != nil {
		t.Errorf("Expected .gitignore to exist, got error: %v", err)
	} else {
		if !containsLine(string(gitignoreData), "_bmad") {
			t.Errorf("Expected .gitignore to contain '_bmad', got: %s", string(gitignoreData))
		}
		if !containsLine(string(gitignoreData), "*.user.toml") {
			t.Errorf("Expected .gitignore to contain '*.user.toml', got: %s", string(gitignoreData))
		}
	}

	// Verify configuration files seeded with generic vs personal user_name
	configTomlData, err := os.ReadFile(injectedBmad)
	if err != nil {
		t.Errorf("Expected injected config.toml, got: %v", err)
	} else {
		if !strings.Contains(string(configTomlData), "user_name = \"developer\"") {
			t.Errorf("Expected config.toml generic user_name, got: %s", string(configTomlData))
		}
		if !strings.Contains(string(configTomlData), "project_name = \"test-project\"") {
			t.Errorf("Expected config.toml project_name, got: %s", string(configTomlData))
		}
	}

	userTomlData, err := os.ReadFile(filepath.Join(projectDir, ".agents", "_bmad", "config.user.toml"))
	if err != nil {
		t.Errorf("Expected config.user.toml to exist, got: %v", err)
	} else if !strings.Contains(string(userTomlData), "user_name = \"test-user\"") {
		t.Errorf("Expected config.user.toml active user_name, got: %s", string(userTomlData))
	}

	customUserTomlData, err := os.ReadFile(filepath.Join(projectDir, ".agents", "_bmad", "custom", "config.user.toml"))
	if err != nil {
		t.Errorf("Expected custom/config.user.toml to exist, got: %v", err)
	} else if !strings.Contains(string(customUserTomlData), "user_name = \"test-user\"") {
		t.Errorf("Expected custom/config.user.toml active user_name, got: %s", string(customUserTomlData))
	}
}

func containsLine(content, target string) bool {
	for _, line := range osLines(content) {
		trimmed := strings.TrimSpace(line)
		if trimmed == target || line == target || line == target+"/" {
			return true
		}
	}
	return false
}

func osLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func TestVerifySkillsPresence(t *testing.T) {
	t.Run("empty directory returns false", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "skills_presence_test")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tempDir)

		if VerifySkillsPresence(tempDir) {
			t.Error("Expected VerifySkillsPresence to return false for empty directory")
		}
	})

	t.Run("missing _bmad returns false", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "skills_presence_test")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tempDir)

		_ = os.MkdirAll(filepath.Join(tempDir, ".agents", "skills", "bmad-test"), 0755)
		if VerifySkillsPresence(tempDir) {
			t.Error("Expected VerifySkillsPresence to return false when _bmad is missing")
		}
	})

	t.Run("missing skills returns false", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "skills_presence_test")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tempDir)

		_ = os.MkdirAll(filepath.Join(tempDir, ".agents", "_bmad"), 0755)
		if VerifySkillsPresence(tempDir) {
			t.Error("Expected VerifySkillsPresence to return false when skills dir is missing")
		}
	})

	t.Run("skills dir without bmad-* prefix returns false", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "skills_presence_test")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tempDir)

		_ = os.MkdirAll(filepath.Join(tempDir, ".agents", "_bmad"), 0755)
		_ = os.MkdirAll(filepath.Join(tempDir, ".agents", "skills", "other-skill"), 0755)
		if VerifySkillsPresence(tempDir) {
			t.Error("Expected VerifySkillsPresence to return false when no bmad-* skill exists")
		}
	})

	t.Run("all present returns true", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "skills_presence_test")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tempDir)

		_ = os.MkdirAll(filepath.Join(tempDir, ".agents", "_bmad"), 0755)
		_ = os.MkdirAll(filepath.Join(tempDir, ".agents", "skills", "bmad-prd"), 0755)
		if !VerifySkillsPresence(tempDir) {
			t.Error("Expected VerifySkillsPresence to return true when all required folders exist")
		}
	})
}

func TestNewWorkspace_SkipsCopyWhenSkillsAlreadyPresent(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "workspace_skip_copy_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	tempBmadRoot, err := os.MkdirTemp("", "bmad_root_source")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempBmadRoot)

	// Create a new skill in the source root that should NOT be copied because workspace already has skills
	srcSkill := filepath.Join(tempBmadRoot, "skills", "bmad-new-skill")
	if err := os.MkdirAll(srcSkill, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcSkill, "SKILL.md"), []byte("# New Skill"), 0644); err != nil {
		t.Fatal(err)
	}

	userId := "test-user"
	projectId := "test-project"
	projectDir := filepath.Join(tempDir, "default-organization", userId, projectId)

	// Pre-populate projectDir with existing .agents/_bmad and .agents/skills/bmad-existing
	existingBmad := filepath.Join(projectDir, ".agents", "_bmad")
	if err := os.MkdirAll(existingBmad, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(existingBmad, "config.toml"), []byte("original = true"), 0644); err != nil {
		t.Fatal(err)
	}

	existingSkill := filepath.Join(projectDir, ".agents", "skills", "bmad-existing")
	if err := os.MkdirAll(existingSkill, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := &Configuration{
		WorkspaceMountPath: tempDir,
		BmadCfg: BmadConfiguration{
			BmadRootPath: tempBmadRoot,
		},
	}

	mockState := &aitesting.MockState{Data: make(map[string]interface{})}
	mockState.Data["projectId"] = projectId
	mockState.Data["repoUrl"] = "https://github.com/example/repo.git"

	mockCtx := &aitesting.MockCallbackContext{
		UserId:    userId,
		SessionId: "test-session",
		StateObj:  mockState,
	}

	_, err = NewWorkspace(mockCtx, cfg)
	if err != nil {
		t.Fatalf("NewWorkspace failed: %v", err)
	}

	// Verify that bmad-new-skill was NOT copied into projectDir
	newSkillInProject := filepath.Join(projectDir, ".agents", "skills", "bmad-new-skill")
	if _, err := os.Stat(newSkillInProject); !os.IsNotExist(err) {
		t.Errorf("Expected bmad-new-skill to NOT be copied when skills already exist, but it was found")
	}
}


