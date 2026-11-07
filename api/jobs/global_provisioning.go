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
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"google.com/apf/api/core"
)

func init() {
	_ = Register(&GlobalWorkspaceProvisioningJob{})
}

// GlobalWorkspaceProvisioningJob provisions and updates shared assets in GLOBAL_WORKSPACE_PATH.
type GlobalWorkspaceProvisioningJob struct {
	DryRun              bool
	BmadRootPath        string
	GlobalWorkspacePath string
}

// Name returns the unique job identifier.
func (j *GlobalWorkspaceProvisioningJob) Name() string {
	return "global-workspace-provisioning"
}

// Description returns a brief human-readable summary of the job's purpose.
func (j *GlobalWorkspaceProvisioningJob) Description() string {
	return "Provisions and updates shared assets in GLOBAL_WORKSPACE_PATH from container image"
}

func (j *GlobalWorkspaceProvisioningJob) isDryRun(ctx context.Context) bool {
	if IsDryRun(ctx) {
		return true
	}
	if j.DryRun {
		return true
	}
	env := strings.ToLower(strings.TrimSpace(os.Getenv("DRY_RUN")))
	return env == "true" || env == "1" || env == "yes"
}

// Run executes the global workspace provisioning job.
func (j *GlobalWorkspaceProvisioningJob) Run(ctx context.Context, cfg *core.Configuration) error {
	dryRun := j.isDryRun(ctx)
	bmadRoot := j.BmadRootPath
	if bmadRoot == "" && cfg != nil {
		bmadRoot = cfg.BmadRootPath
	}
	if bmadRoot == "" {
		bmadRoot = os.Getenv("BMAD_ROOT_PATH")
	}
	if bmadRoot == "" {
		bmadRoot = "/config/bmad"
	}
	if !filepath.IsAbs(bmadRoot) {
		absPath, err := filepath.Abs(bmadRoot)
		if err == nil {
			bmadRoot = absPath
		}
	}

	globalPath := j.GlobalWorkspacePath
	if globalPath == "" && cfg != nil {
		globalPath = cfg.GlobalWorkspacePath
	}
	if globalPath == "" {
		globalPath = os.Getenv("GLOBAL_WORKSPACE_PATH")
	}
	if globalPath == "" {
		mountPath := "/workspaces"
		if cfg != nil && cfg.WorkspaceMountPath != "" {
			mountPath = cfg.WorkspaceMountPath
		}
		globalPath = filepath.Join(mountPath, "global")
	}
	if !filepath.IsAbs(globalPath) {
		absPath, err := filepath.Abs(globalPath)
		if err == nil {
			globalPath = absPath
		}
	}

	slog.Info("Running GlobalWorkspaceProvisioningJob",
		"bmadRoot", bmadRoot,
		"globalPath", globalPath,
		"dryRun", dryRun,
	)

	// Validate source directory exists
	info, err := os.Stat(bmadRoot)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("invalid or non-existent bmad root directory: %s (err: %w)", bmadRoot, err)
	}

	if dryRun {
		slog.Info("[DRY-RUN] Would synchronize assets from bmadRoot to globalPath", "bmadRoot", bmadRoot, "globalPath", globalPath)
		return nil
	}

	// Ensure destination directory exists
	if err := os.MkdirAll(globalPath, 0755); err != nil {
		return fmt.Errorf("failed to create global workspace directory %s: %w", globalPath, err)
	}

	// Acquire file lock on .seed.lock to ensure multi-instance / concurrent job safety
	lockFilePath := filepath.Join(globalPath, ".seed.lock")
	lockFile, err := os.OpenFile(lockFilePath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("failed to open lock file %s: %w", lockFilePath, err)
	}
	defer lockFile.Close()

	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("failed to acquire file lock on %s: %w", lockFilePath, err)
	}
	defer func() {
		_ = syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN)
	}()

	// Copy _bmad directory
	srcBmad := filepath.Join(bmadRoot, "_bmad")
	if _, err := os.Stat(srcBmad); os.IsNotExist(err) {
		srcBmad = filepath.Join(bmadRoot, "bmad")
	}
	if bmadInfo, err := os.Stat(srcBmad); err == nil && bmadInfo.IsDir() {
		dstBmad := filepath.Join(globalPath, "_bmad")
		if err := syncDirectory(srcBmad, dstBmad); err != nil {
			return fmt.Errorf("failed to sync _bmad directory: %w", err)
		}
	}

	// Copy skills directory
	srcSkills := filepath.Join(bmadRoot, "skills")
	if skillsInfo, err := os.Stat(srcSkills); err == nil && skillsInfo.IsDir() {
		dstSkills := filepath.Join(globalPath, "skills")
		if err := syncDirectory(srcSkills, dstSkills); err != nil {
			return fmt.Errorf("failed to sync skills directory: %w", err)
		}
	}

	// Copy template files if present
	srcTemplates := filepath.Join(bmadRoot, "templates")
	if templatesInfo, err := os.Stat(srcTemplates); err == nil && templatesInfo.IsDir() {
		dstTemplates := filepath.Join(globalPath, "templates")
		if err := syncDirectory(srcTemplates, dstTemplates); err != nil {
			return fmt.Errorf("failed to sync templates directory: %w", err)
		}
	}

	// Write .version metadata
	versionPath := filepath.Join(globalPath, ".version")
	timestamp := time.Now().UTC().Format(time.RFC3339)
	versionContent := fmt.Sprintf("provisioned_at: %s\nsource: %s\n", timestamp, bmadRoot)
	if err := os.WriteFile(versionPath, []byte(versionContent), 0644); err != nil {
		slog.Warn("Failed to write .version metadata file in global workspace", "error", err)
	}

	slog.Info("GlobalWorkspaceProvisioningJob completed successfully", "globalPath", globalPath)
	return nil
}

// syncDirectory copies files and subdirectories from src to dst.
func syncDirectory(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		targetPath := filepath.Join(dst, relPath)
		if d.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return err
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		return copyFileContents(path, targetPath, info.Mode())
	})
}

// copyFileContents copies a single file from src to dst.
func copyFileContents(src, dst string, mode os.FileMode) error {
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

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	return out.Sync()
}
