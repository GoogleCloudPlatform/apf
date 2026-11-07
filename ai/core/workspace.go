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
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"google.golang.org/adk/agent"
)

const (
	stateWorkspaceReady = "workspace_ready"
)

// Workspace manages a directory within a local directory that is mounted from GCS.
type Workspace struct {
	destRootDir string
	cfg         *Configuration
	ctx         agent.CallbackContext
}

func NewWorkspace(ctx agent.CallbackContext, cfg *Configuration) (*Workspace, error) {
	ws := &Workspace{
		cfg:         cfg,
		ctx:         ctx,
		destRootDir: cfg.WorkspaceMountPath,
	}

	repoCfg, err := NewRepoConfigFromState(ctx, cfg)
	if err != nil {
		return nil, err
	}
	ws.cfg.RepoCfg = *repoCfg

	userId := ctx.UserID()
	clonePath := filepath.Join(cfg.WorkspaceMountPath, repoCfg.OrganizationId, userId, repoCfg.ProjectId)
	if _, err := os.Stat(clonePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("workspace directory does not exist: %s (must be initialized by API tier)", clonePath)
	}

	// Execute skill injection logic during workspace initialization only if required files do not exist
	if !VerifySkillsPresence(clonePath) {
		bmadRoot := cfg.BmadCfg.BmadRootPath
		if bmadRoot == "" {
			bmadRoot = os.Getenv("BMAD_ROOT_PATH")
		}
		if bmadRoot != "" && !filepath.IsAbs(bmadRoot) {
			if absPath, err := filepath.Abs(bmadRoot); err == nil {
				bmadRoot = absPath
			}
		}

		if bmadRoot != "" {
			if info, err := os.Stat(bmadRoot); err == nil && info.IsDir() {
				agentsDir := filepath.Join(clonePath, ".agents")
				_ = os.MkdirAll(agentsDir, 0755)

				// Copy _bmad folder
				srcBmad := filepath.Join(bmadRoot, "_bmad")
				if _, err := os.Stat(srcBmad); os.IsNotExist(err) {
					srcBmad = filepath.Join(bmadRoot, "bmad")
				}
				if info, err := os.Stat(srcBmad); err == nil && info.IsDir() {
					dstBmad := filepath.Join(agentsDir, "_bmad")
					_ = copyDir(srcBmad, dstBmad)
				}

				// Copy skills into .agents/skills
				srcSkills := filepath.Join(bmadRoot, "skills")
				if info, err := os.Stat(srcSkills); err == nil && info.IsDir() {
					dstSkills := filepath.Join(agentsDir, "skills")
					_ = os.MkdirAll(dstSkills, 0755)
					entries, err := os.ReadDir(srcSkills)
					if err == nil {
						for _, entry := range entries {
							srcItem := filepath.Join(srcSkills, entry.Name())
							dstItem := filepath.Join(dstSkills, entry.Name())
							if entry.IsDir() {
								_ = copyDir(srcItem, dstItem)
							} else {
								_ = copyFile(srcItem, dstItem, 0644)
							}
						}
					}
				}
			}
		}
	}

	// Ensure upstream BMad compatibility symlink (_bmad -> .agents/_bmad) exists
	agentsBmad := filepath.Join(clonePath, ".agents", "_bmad")
	if _, err := os.Stat(agentsBmad); err == nil {
		rootBmad := filepath.Join(clonePath, "_bmad")
		if _, err := os.Lstat(rootBmad); os.IsNotExist(err) {
			_ = os.Symlink(filepath.Join(".agents", "_bmad"), rootBmad)
		}

		// Seed and synchronize configuration files with project metadata
		specLoc := repoCfg.SpecLocation
		if specLoc == "" {
			specLoc = "docs/spec"
		}
		planLoc := repoCfg.PlanningArtifactsLocation
		if planLoc == "" {
			planLoc = "docs/spec/planning-artifacts"
		}
		implLoc := repoCfg.ImplementationArtifactsLocation
		if implLoc == "" {
			implLoc = "docs/spec/implementation-artifacts"
		}

		sharedFields := map[string]string{
			"planning_artifacts":       fmt.Sprintf("{project-root}/%s", planLoc),
			"implementation_artifacts": fmt.Sprintf("{project-root}/%s", implLoc),
			"project_knowledge":        fmt.Sprintf("{project-root}/%s", specLoc),
			"output_folder":            fmt.Sprintf("{project-root}/%s", specLoc),
			"project_name":             repoCfg.ProjectName,
			"user_name":                "developer",
		}
		userFields := map[string]string{
			"user_name": userId,
		}

		_ = updateTomlConfigFile(filepath.Join(agentsBmad, "config.toml"), sharedFields)
		_ = updateYamlConfigFile(filepath.Join(agentsBmad, "core", "config.yaml"), sharedFields)
		_ = updateYamlConfigFile(filepath.Join(agentsBmad, "bmm", "config.yaml"), sharedFields)
		_ = updateTomlConfigFile(filepath.Join(agentsBmad, "config.user.toml"), userFields)

		customDir := filepath.Join(agentsBmad, "custom")
		_ = os.MkdirAll(customDir, 0755)
		_ = updateTomlConfigFile(filepath.Join(customDir, "config.user.toml"), userFields)
	}

	// Ensure .gitignore exists and contains _bmad, .tmp/, *.user.toml
	if err := ensureGitignoreWithBmad(clonePath); err != nil {
		slog.Warn("Failed to ensure .gitignore", "error", err, "path", clonePath)
	}

	// Set workspace ready in state just in case
	_ = ctx.State().Set(stateWorkspaceReady, "true")

	return ws, nil
}

// ensureGitignoreWithBmad checks for .gitignore in projectRoot, creates it if missing,
// and ensures _bmad, .tmp/, and *.user.toml are included in the file.
func ensureGitignoreWithBmad(projectRoot string) error {
	gitignorePath := filepath.Join(projectRoot, ".gitignore")
	data, err := os.ReadFile(gitignorePath)
	if os.IsNotExist(err) {
		return os.WriteFile(gitignorePath, []byte("_bmad\n.tmp/\n*.user.toml\n"), 0644)
	} else if err != nil {
		return fmt.Errorf("failed to read .gitignore: %w", err)
	}

	content := string(data)
	lines := strings.Split(content, "\n")
	var newLines []string
	modified := false
	hasTmp := false
	hasBmad := false
	hasUserToml := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == ".agents" || trimmed == ".agents/" {
			modified = true
			continue
		}
		if trimmed == ".tmp" || trimmed == ".tmp/" || trimmed == "/.tmp" || trimmed == "/.tmp/" {
			hasTmp = true
		}
		if trimmed == "_bmad" || trimmed == "_bmad/" || trimmed == "/_bmad" || trimmed == "/_bmad/" {
			hasBmad = true
		}
		if trimmed == "*.user.toml" || trimmed == "config.user.toml" {
			hasUserToml = true
		}
		newLines = append(newLines, line)
	}

	if !hasTmp {
		newLines = append(newLines, ".tmp/")
		modified = true
	}
	if !hasBmad {
		newLines = append(newLines, "_bmad")
		modified = true
	}
	if !hasUserToml {
		newLines = append(newLines, "*.user.toml")
		modified = true
	}

	if modified {
		output := strings.Join(newLines, "\n")
		if !strings.HasSuffix(output, "\n") {
			output += "\n"
		}
		if err := os.WriteFile(gitignorePath, []byte(output), 0644); err != nil {
			return fmt.Errorf("failed to update .gitignore: %w", err)
		}
	}
	return nil
}

func updateTomlConfigFile(filePath string, fields map[string]string) error {
	var content string
	contentBytes, err := os.ReadFile(filePath)
	if os.IsNotExist(err) {
		content = "[core]\n"
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else {
		content = string(contentBytes)
	}

	for key, val := range fields {
		if val == "" {
			continue
		}
		target := fmt.Sprintf("%s = \"%s\"", key, val)
		re := regexp.MustCompile(fmt.Sprintf(`(?m)^\s*%s\s*=.*$`, regexp.QuoteMeta(key)))
		if re.MatchString(content) {
			content = re.ReplaceAllString(content, target)
		} else {
			if !strings.HasSuffix(content, "\n") && len(content) > 0 {
				content += "\n"
			}
			content += target + "\n"
		}
	}
	return os.WriteFile(filePath, []byte(content), 0644)
}

func updateYamlConfigFile(filePath string, fields map[string]string) error {
	var content string
	contentBytes, err := os.ReadFile(filePath)
	if os.IsNotExist(err) {
		content = ""
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else {
		content = string(contentBytes)
	}

	for key, val := range fields {
		if val == "" {
			continue
		}
		target := fmt.Sprintf("%s: \"%s\"", key, val)
		re := regexp.MustCompile(fmt.Sprintf(`(?m)^\s*%s:\s*.*$`, regexp.QuoteMeta(key)))
		if re.MatchString(content) {
			content = re.ReplaceAllString(content, target)
		} else {
			if !strings.HasSuffix(content, "\n") && len(content) > 0 {
				content += "\n"
			}
			content += target + "\n"
		}
	}
	return os.WriteFile(filePath, []byte(content), 0644)
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, relPath)
		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}
		return copyFile(path, dstPath, info.Mode())
	})
}

func copyFile(src, dst string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

// VerifySkillsPresence checks if .agents, .agents/_bmad, and at least one
// .agents/skills/bmad-* subfolder exist in the workspace directory.
// Returns true if all exist, false otherwise.
func VerifySkillsPresence(workspaceDir string) bool {
	agentsDir := filepath.Join(workspaceDir, ".agents")
	if info, err := os.Stat(agentsDir); err != nil || !info.IsDir() {
		return false
	}

	bmadDir := filepath.Join(agentsDir, "_bmad")
	if info, err := os.Stat(bmadDir); err != nil || !info.IsDir() {
		return false
	}

	skillsDir := filepath.Join(agentsDir, "skills")
	if info, err := os.Stat(skillsDir); err != nil || !info.IsDir() {
		return false
	}

	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return false
	}

	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "bmad-") {
			return true
		}
	}

	return false
}
