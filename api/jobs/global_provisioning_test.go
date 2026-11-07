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
	"os"
	"path/filepath"
	"testing"

	"google.com/apf/api/core"
)

func TestGlobalWorkspaceProvisioningJob_Registration(t *testing.T) {
	job, ok := Get("global-workspace-provisioning")
	if !ok {
		t.Fatalf("Expected global-workspace-provisioning job to be registered in DefaultRegistry")
	}
	if job.Name() != "global-workspace-provisioning" {
		t.Errorf("Expected job name 'global-workspace-provisioning', got %s", job.Name())
	}
	if job.Description() == "" {
		t.Errorf("Expected non-empty job description")
	}
}

func TestGlobalWorkspaceProvisioningJob_Run(t *testing.T) {
	tempDir := t.TempDir()
	srcBmad := filepath.Join(tempDir, "source_bmad")
	globalDir := filepath.Join(tempDir, "global")

	// Create sample source structure
	_ = os.MkdirAll(filepath.Join(srcBmad, "_bmad"), 0755)
	_ = os.WriteFile(filepath.Join(srcBmad, "_bmad", "config.toml"), []byte("project_name = \"test\"\n"), 0644)

	_ = os.MkdirAll(filepath.Join(srcBmad, "skills", "bmad-build"), 0755)
	_ = os.WriteFile(filepath.Join(srcBmad, "skills", "bmad-build", "SKILL.md"), []byte("# Build Skill\n"), 0644)

	job := &GlobalWorkspaceProvisioningJob{
		BmadRootPath:        srcBmad,
		GlobalWorkspacePath: globalDir,
	}

	cfg := &core.Configuration{
		BmadRootPath:        srcBmad,
		GlobalWorkspacePath: globalDir,
	}

	ctx := context.Background()

	// 1. Test Run Execution
	if err := job.Run(ctx, cfg); err != nil {
		t.Fatalf("Job Run failed: %v", err)
	}

	// Verify _bmad synced
	bmadDst := filepath.Join(globalDir, "_bmad", "config.toml")
	data, err := os.ReadFile(bmadDst)
	if err != nil {
		t.Fatalf("Expected %s to exist: %v", bmadDst, err)
	}
	if string(data) != "project_name = \"test\"\n" {
		t.Errorf("Unexpected content: %s", string(data))
	}

	// Verify skill synced
	skillDst := filepath.Join(globalDir, "skills", "bmad-build", "SKILL.md")
	dataSkill, err := os.ReadFile(skillDst)
	if err != nil {
		t.Fatalf("Expected %s to exist: %v", skillDst, err)
	}
	if string(dataSkill) != "# Build Skill\n" {
		t.Errorf("Unexpected skill content: %s", string(dataSkill))
	}

	// Verify .version exists
	versionDst := filepath.Join(globalDir, ".version")
	if _, err := os.Stat(versionDst); err != nil {
		t.Errorf("Expected .version file to exist: %v", err)
	}
}

func TestGlobalWorkspaceProvisioningJob_DryRun(t *testing.T) {
	tempDir := t.TempDir()
	srcBmad := filepath.Join(tempDir, "source_bmad")
	globalDir := filepath.Join(tempDir, "global")

	_ = os.MkdirAll(filepath.Join(srcBmad, "skills", "bmad-build"), 0755)
	_ = os.WriteFile(filepath.Join(srcBmad, "skills", "bmad-build", "SKILL.md"), []byte("# Build Skill\n"), 0644)

	job := &GlobalWorkspaceProvisioningJob{
		BmadRootPath:        srcBmad,
		GlobalWorkspacePath: globalDir,
	}

	cfg := &core.Configuration{
		BmadRootPath:        srcBmad,
		GlobalWorkspacePath: globalDir,
	}

	// Run in Dry Run mode
	ctx := ContextWithDryRun(context.Background(), true)
	if err := job.Run(ctx, cfg); err != nil {
		t.Fatalf("Dry run failed: %v", err)
	}

	// Destination should NOT exist
	if _, err := os.Stat(globalDir); !os.IsNotExist(err) {
		t.Errorf("Expected globalDir to not exist during dry run")
	}
}
