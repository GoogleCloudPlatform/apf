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

package tools

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"google.com/apf/ai/core"
	aitesting "google.com/apf/ai/testing"
)

type mockCommandExecutor struct {
	lookPathFn func(file string) (string, error)
	runFn      func(ctx context.Context, dir string, env []string, name string, args ...string) (string, string, int, error)
}

func (m *mockCommandExecutor) LookPath(file string) (string, error) {
	if m.lookPathFn != nil {
		return m.lookPathFn(file)
	}
	return "/usr/local/bin/uv", nil
}

func (m *mockCommandExecutor) Run(ctx context.Context, dir string, env []string, name string, args ...string) (string, string, int, error) {
	if m.runFn != nil {
		return m.runFn(ctx, dir, env, name, args...)
	}
	return "mock stdout", "", 0, nil
}

func setupTestWorkspace(t *testing.T) (projectRoot, bmadRoot, mountPath string) {
	tmpDir, err := os.MkdirTemp("", "uv_runner_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	mountPath = filepath.Join(tmpDir, "mount")
	projectRoot = filepath.Join(mountPath, "test-org", "test-user", "test-proj")
	bmadRoot = filepath.Join(tmpDir, "bmad")

	if err := os.MkdirAll(filepath.Join(projectRoot, "docs", "spec"), 0755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectRoot, ".agents", "skills", "scripts"), 0755); err != nil {
		t.Fatalf("failed to create skills dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectRoot, ".agents", "_bmad", "scripts"), 0755); err != nil {
		t.Fatalf("failed to create _bmad dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(bmadRoot, "scripts"), 0755); err != nil {
		t.Fatalf("failed to create bmad dir: %v", err)
	}

	return projectRoot, bmadRoot, mountPath
}

func TestUVRunner_Success(t *testing.T) {
	projectRoot, bmadRoot, mountPath := setupTestWorkspace(t)
	defer os.RemoveAll(filepath.Dir(mountPath))

	scriptFile := filepath.Join(projectRoot, "test_script.py")
	if err := os.WriteFile(scriptFile, []byte("print('hello')"), 0644); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	cfg := &core.Configuration{
		WorkspaceMountPath: mountPath,
		RepoCfg: core.RepoConfiguration{
			OrganizationId: "test-org",
			ProjectId:      "test-proj",
		},
		BmadCfg: core.BmadConfiguration{
			BmadRootPath:   bmadRoot,
			BmadOutputPath: "docs/spec",
		},
		ContainerConcurrency: 10,
	}

	var capturedDir string
	var capturedArgs []string
	var capturedEnv []string

	mockExec := &mockCommandExecutor{
		lookPathFn: func(file string) (string, error) {
			return "/usr/bin/uv", nil
		},
		runFn: func(ctx context.Context, dir string, env []string, name string, args ...string) (string, string, int, error) {
			capturedDir = dir
			capturedArgs = args
			capturedEnv = env
			return "execution successful\n", "", 0, nil
		},
	}

	runner := NewUVRunnerTool(cfg)
	runner.Executor = mockExec

	mockCtx := &aitesting.MockToolContext{
		MockCallbackContext: &aitesting.MockCallbackContext{
			UserId:   "test-user",
			StateObj: &aitesting.MockState{},
		},
	}

	input := &UVRunnerInput{
		ScriptPath: "{project-root}/test_script.py",
		Args:       []string{"--workspace", "{project-root}/docs/spec", "--bmad", "{skill-root}"},
	}

	out, err := runner.Execute(mockCtx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !out.Success || out.ExitCode != 0 {
		t.Errorf("expected success, got exitCode=%d", out.ExitCode)
	}
	if out.Stdout != "execution successful\n" {
		t.Errorf("unexpected stdout: %q", out.Stdout)
	}
	if capturedDir != projectRoot {
		t.Errorf("expected cmd.Dir=%s, got %s", projectRoot, capturedDir)
	}
	if len(capturedArgs) < 5 || capturedArgs[0] != "run" || capturedArgs[1] != scriptFile {
		t.Errorf("unexpected command args: %v", capturedArgs)
	}
	if capturedArgs[3] != filepath.Join(projectRoot, "docs", "spec") {
		t.Errorf("token {project-root} was not expanded in arg: %s", capturedArgs[3])
	}
	expectedSkillRoot := filepath.Join(projectRoot, ".agents", "skills")
	if capturedArgs[5] != expectedSkillRoot {
		t.Errorf("token {skill-root} was not expanded to %s in arg: %s", expectedSkillRoot, capturedArgs[5])
	}

	// Verify environment variables
	var hasPythonUnbuffered, hasUvSystemPython, hasHomeTmp, hasUvCache bool
	for _, e := range capturedEnv {
		if e == "PYTHONUNBUFFERED=1" {
			hasPythonUnbuffered = true
		}
		if e == "UV_SYSTEM_PYTHON=1" {
			hasUvSystemPython = true
		}
		if e == "HOME=/tmp" {
			hasHomeTmp = true
		}
		if e == "UV_CACHE_DIR=/tmp/uv-cache" {
			hasUvCache = true
		}
	}
	if !hasPythonUnbuffered || !hasUvSystemPython || !hasHomeTmp || !hasUvCache {
		t.Errorf("missing expected environment variables: %v", capturedEnv)
	}
}

func TestUVRunner_ZeroTrustBoundaryViolation(t *testing.T) {
	_, bmadRoot, mountPath := setupTestWorkspace(t)
	defer os.RemoveAll(filepath.Dir(mountPath))

	cfg := &core.Configuration{
		WorkspaceMountPath: mountPath,
		RepoCfg: core.RepoConfiguration{
			OrganizationId: "test-org",
			ProjectId:      "test-proj",
		},
		BmadCfg: core.BmadConfiguration{
			BmadRootPath:   bmadRoot,
			BmadOutputPath: "docs/spec",
		},
	}

	runner := NewUVRunnerTool(cfg)
	runner.Executor = &mockCommandExecutor{}

	mockCtx := &aitesting.MockToolContext{
		MockCallbackContext: &aitesting.MockCallbackContext{
			UserId:   "test-user",
			StateObj: &aitesting.MockState{},
		},
	}

	input := &UVRunnerInput{
		ScriptPath: "/etc/shadow",
	}

	_, err := runner.Execute(mockCtx, input)
	if err == nil {
		t.Fatalf("expected error for path outside boundary, got nil")
	}
	if !strings.Contains(err.Error(), "zero-trust security violation") {
		t.Errorf("expected zero-trust security error, got: %v", err)
	}
}

func TestUVRunner_NonZeroExitCode(t *testing.T) {
	projectRoot, bmadRoot, mountPath := setupTestWorkspace(t)
	defer os.RemoveAll(filepath.Dir(mountPath))

	scriptFile := filepath.Join(projectRoot, ".agents", "skills", "scripts", "failing.py")
	if err := os.WriteFile(scriptFile, []byte("import sys; sys.exit(2)"), 0644); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	cfg := &core.Configuration{
		WorkspaceMountPath: mountPath,
		RepoCfg: core.RepoConfiguration{
			OrganizationId: "test-org",
			ProjectId:      "test-proj",
		},
		BmadCfg: core.BmadConfiguration{
			BmadRootPath:   bmadRoot,
			BmadOutputPath: "docs/spec",
		},
	}

	runner := NewUVRunnerTool(cfg)
	runner.Executor = &mockCommandExecutor{
		runFn: func(ctx context.Context, dir string, env []string, name string, args ...string) (string, string, int, error) {
			return "", "Traceback (most recent call last):\nValueError: bad arg", 2, nil
		},
	}

	mockCtx := &aitesting.MockToolContext{
		MockCallbackContext: &aitesting.MockCallbackContext{
			UserId:   "test-user",
			StateObj: &aitesting.MockState{},
		},
	}

	input := &UVRunnerInput{
		ScriptPath: "{skill-root}/scripts/failing.py",
	}

	out, err := runner.Execute(mockCtx, input)
	if err != nil {
		t.Fatalf("unexpected error for non-zero exit code (should return structured output): %v", err)
	}
	if out.Success {
		t.Errorf("expected Success=false, got true")
	}
	if out.ExitCode != 2 {
		t.Errorf("expected ExitCode=2, got %d", out.ExitCode)
	}
	if !strings.Contains(out.Stderr, "ValueError: bad arg") {
		t.Errorf("unexpected stderr: %s", out.Stderr)
	}
}

func TestUVRunner_OutputTruncation(t *testing.T) {
	projectRoot, bmadRoot, mountPath := setupTestWorkspace(t)
	defer os.RemoveAll(filepath.Dir(mountPath))

	scriptFile := filepath.Join(projectRoot, "large_output.py")
	if err := os.WriteFile(scriptFile, []byte("print('x')"), 0644); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	cfg := &core.Configuration{
		WorkspaceMountPath: mountPath,
		RepoCfg: core.RepoConfiguration{
			OrganizationId: "test-org",
			ProjectId:      "test-proj",
		},
		BmadCfg: core.BmadConfiguration{
			BmadRootPath:   bmadRoot,
			BmadOutputPath: "docs/spec",
		},
	}

	largeStdout := strings.Repeat("A", 70*1024)

	runner := NewUVRunnerTool(cfg)
	runner.Executor = &mockCommandExecutor{
		runFn: func(ctx context.Context, dir string, env []string, name string, args ...string) (string, string, int, error) {
			return largeStdout, "", 0, nil
		},
	}

	mockCtx := &aitesting.MockToolContext{
		MockCallbackContext: &aitesting.MockCallbackContext{
			UserId:   "test-user",
			StateObj: &aitesting.MockState{},
		},
	}

	out, err := runner.Execute(mockCtx, &UVRunnerInput{ScriptPath: scriptFile})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out.Stdout, "...[stdout truncated at 64KB]...") {
		t.Errorf("expected output to be truncated at 64KB")
	}
	if len(out.Stdout) > 70*1024 {
		t.Errorf("stdout was not truncated, length: %d", len(out.Stdout))
	}
}

func TestUVRunner_FileModificationStateNotification(t *testing.T) {
	projectRoot, bmadRoot, mountPath := setupTestWorkspace(t)
	defer os.RemoveAll(filepath.Dir(mountPath))

	scriptFile := filepath.Join(projectRoot, "generate.py")
	if err := os.WriteFile(scriptFile, []byte("print('create')"), 0644); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	cfg := &core.Configuration{
		WorkspaceMountPath: mountPath,
		RepoCfg: core.RepoConfiguration{
			OrganizationId: "test-org",
			ProjectId:      "test-proj",
		},
		BmadCfg: core.BmadConfiguration{
			BmadRootPath:   bmadRoot,
			BmadOutputPath: "docs/spec",
		},
	}

	mockState := &aitesting.MockState{Data: make(map[string]interface{})}
	mockCtx := &aitesting.MockToolContext{
		MockCallbackContext: &aitesting.MockCallbackContext{
			UserId:   "test-user",
			StateObj: mockState,
		},
	}

	runner := NewUVRunnerTool(cfg)
	runner.Executor = &mockCommandExecutor{
		runFn: func(ctx context.Context, dir string, env []string, name string, args ...string) (string, string, int, error) {
			// Simulate script creating a file in docs/spec and projectRoot
			specOut := filepath.Join(projectRoot, "docs", "spec", "brainstorm.html")
			_ = os.WriteFile(specOut, []byte("<html>content</html>"), 0644)
			_ = os.WriteFile(filepath.Join(projectRoot, "AGENTS.md"), []byte("# Agents"), 0644)
			return "done", "", 0, nil
		},
	}

	out, err := runner.Execute(mockCtx, &UVRunnerInput{ScriptPath: scriptFile})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(out.CreatedFiles) < 2 {
		t.Errorf("expected at least 2 created files, got %v", out.CreatedFiles)
	}

	// Verify state notifications
	var notificationCount int
	for k, v := range mockState.Data {
		if strings.HasPrefix(k, "file-write-notification:") {
			notificationCount++
			valStr, _ := v.(string)
			if valStr != filepath.Join("docs", "spec", "brainstorm.html") && valStr != "AGENTS.md" {
				t.Errorf("unexpected notification value: %v", valStr)
			}
		}
	}
	if notificationCount < 2 {
		t.Errorf("expected 2 state notifications, got %d", notificationCount)
	}
}

func TestUVRunner_RealUV(t *testing.T) {
	uvPath, err := exec.LookPath("uv")
	if err != nil {
		t.Skip("uv binary not found in PATH; skipping integration test")
	}

	projectRoot, bmadRoot, mountPath := setupTestWorkspace(t)
	defer os.RemoveAll(filepath.Dir(mountPath))

	scriptPath := filepath.Join(projectRoot, "hello.py")
	if err := os.WriteFile(scriptPath, []byte("import sys; print(f'Hello from uv python {sys.version_info.major}')"), 0755); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	cfg := &core.Configuration{
		WorkspaceMountPath: mountPath,
		RepoCfg: core.RepoConfiguration{
			OrganizationId: "test-org",
			ProjectId:      "test-proj",
		},
		BmadCfg: core.BmadConfiguration{
			BmadRootPath:   bmadRoot,
			BmadOutputPath: "docs/spec",
		},
	}

	runner := NewUVRunnerTool(cfg)
	mockCtx := &aitesting.MockToolContext{
		MockCallbackContext: &aitesting.MockCallbackContext{
			UserId:   "test-user",
			StateObj: &aitesting.MockState{},
		},
	}

	out, err := runner.Execute(mockCtx, &UVRunnerInput{
		ScriptPath: "{project-root}/hello.py",
	})
	if err != nil {
		t.Fatalf("real uv execution failed: %v", err)
	}
	if !out.Success {
		t.Fatalf("real uv failed with exitCode=%d, stderr=%s", out.ExitCode, out.Stderr)
	}
	if !strings.Contains(out.Stdout, "Hello from uv python") {
		t.Errorf("unexpected stdout from real uv (%s): %s", uvPath, out.Stdout)
	}
}

func TestUVRunner_MissingBinary(t *testing.T) {
	_, bmadRoot, mountPath := setupTestWorkspace(t)
	defer os.RemoveAll(filepath.Dir(mountPath))

	cfg := &core.Configuration{
		WorkspaceMountPath: mountPath,
		RepoCfg: core.RepoConfiguration{
			OrganizationId: "test-org",
			ProjectId:      "test-proj",
		},
		BmadCfg: core.BmadConfiguration{
			BmadRootPath:   bmadRoot,
			BmadOutputPath: "docs/spec",
		},
	}

	runner := NewUVRunnerTool(cfg)
	runner.Executor = &mockCommandExecutor{
		lookPathFn: func(file string) (string, error) {
			return "", os.ErrNotExist
		},
	}

	mockCtx := &aitesting.MockToolContext{
		MockCallbackContext: &aitesting.MockCallbackContext{
			UserId:   "test-user",
			StateObj: &aitesting.MockState{},
		},
	}

	_, err := runner.Execute(mockCtx, &UVRunnerInput{
		ScriptPath: "{project-root}/any.py",
	})
	if err == nil {
		t.Fatalf("expected error for missing uv binary, got nil")
	}
	if !strings.Contains(err.Error(), "uv executable not found in container PATH") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestUVRunner_FrameworkScriptFallback(t *testing.T) {
	projectRoot, bmadRoot, mountPath := setupTestWorkspace(t)
	defer os.RemoveAll(filepath.Dir(mountPath))

	// Create script in project workspace .agents/_bmad/scripts
	scriptFile := filepath.Join(projectRoot, ".agents", "_bmad", "scripts", "brain.py")
	if err := os.WriteFile(scriptFile, []byte("print('brain')"), 0755); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	cfg := &core.Configuration{
		WorkspaceMountPath: mountPath,
		RepoCfg: core.RepoConfiguration{
			OrganizationId: "test-org",
			ProjectId:      "test-proj",
		},
		BmadCfg: core.BmadConfiguration{
			BmadRootPath:   bmadRoot,
			BmadOutputPath: "docs/spec",
		},
	}

	var capturedArgs []string
	var capturedDir string
	runner := NewUVRunnerTool(cfg)
	runner.Executor = &mockCommandExecutor{
		runFn: func(ctx context.Context, dir string, env []string, name string, args ...string) (string, string, int, error) {
			capturedDir = dir
			capturedArgs = args
			return "brain ok", "", 0, nil
		},
	}

	mockCtx := &aitesting.MockToolContext{
		MockCallbackContext: &aitesting.MockCallbackContext{
			UserId:   "test-user",
			StateObj: &aitesting.MockState{},
		},
	}

	// Relative path that exists in projectRoot/.agents/_bmad/scripts/
	out, err := runner.Execute(mockCtx, &UVRunnerInput{
		ScriptPath: "scripts/brain.py",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out.Success {
		t.Errorf("expected success, got false")
	}
	if capturedDir != projectRoot {
		t.Errorf("expected cmd.Dir=%s, got %s", projectRoot, capturedDir)
	}
	if len(capturedArgs) < 2 || capturedArgs[1] != scriptFile {
		t.Errorf("expected fallback to local framework script (%s), got: %v", scriptFile, capturedArgs)
	}
}

func TestUVRunner_BmadPathPriority(t *testing.T) {
	projectRoot, bmadRoot, mountPath := setupTestWorkspace(t)
	defer os.RemoveAll(filepath.Dir(mountPath))

	// Create script in projectRoot/.agents/_bmad/scripts/resolve_config.py
	bmadScriptDir := filepath.Join(projectRoot, ".agents", "_bmad", "scripts")
	if err := os.MkdirAll(bmadScriptDir, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	bmadScriptFile := filepath.Join(bmadScriptDir, "resolve_config.py")
	if err := os.WriteFile(bmadScriptFile, []byte("print('bmad project config')"), 0755); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	cfg := &core.Configuration{
		WorkspaceMountPath: mountPath,
		RepoCfg: core.RepoConfiguration{
			OrganizationId: "test-org",
			ProjectId:      "test-proj",
		},
		BmadCfg: core.BmadConfiguration{
			BmadRootPath:   bmadRoot,
			BmadOutputPath: "docs/spec",
		},
	}

	var capturedArgs []string
	var capturedDir string
	runner := NewUVRunnerTool(cfg)
	runner.Executor = &mockCommandExecutor{
		runFn: func(ctx context.Context, dir string, env []string, name string, args ...string) (string, string, int, error) {
			capturedDir = dir
			capturedArgs = args
			return "config ok", "", 0, nil
		},
	}

	mockCtx := &aitesting.MockToolContext{
		MockCallbackContext: &aitesting.MockCallbackContext{
			UserId:   "test-user",
			StateObj: &aitesting.MockState{},
		},
	}

	// Call uv_run with {project-root}/_bmad/scripts/resolve_config.py
	out, err := runner.Execute(mockCtx, &UVRunnerInput{
		ScriptPath: "{project-root}/_bmad/scripts/resolve_config.py",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out.Success {
		t.Errorf("expected success, got false")
	}
	if capturedDir != projectRoot {
		t.Errorf("expected cmd.Dir=%s, got %s", projectRoot, capturedDir)
	}
	if len(capturedArgs) < 2 || capturedArgs[1] != bmadScriptFile {
		t.Errorf("expected _bmad path to resolve to local project script (%s), got: %v", bmadScriptFile, capturedArgs)
	}
}

func TestUVRunner_ProjectScriptPrecedence(t *testing.T) {
	projectRoot, bmadRoot, mountPath := setupTestWorkspace(t)
	defer os.RemoveAll(filepath.Dir(mountPath))

	// Create script in projectRoot/.tmp/scratch.py
	projTmpDir := filepath.Join(projectRoot, ".tmp")
	if err := os.MkdirAll(projTmpDir, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	projScriptFile := filepath.Join(projTmpDir, "scratch.py")
	if err := os.WriteFile(projScriptFile, []byte("print('project scratch')"), 0755); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	cfg := &core.Configuration{
		WorkspaceMountPath: mountPath,
		RepoCfg: core.RepoConfiguration{
			OrganizationId: "test-org",
			ProjectId:      "test-proj",
		},
		BmadCfg: core.BmadConfiguration{
			BmadRootPath:   bmadRoot,
			BmadOutputPath: "docs/spec",
		},
	}

	var capturedArgs []string
	var capturedDir string
	runner := NewUVRunnerTool(cfg)
	runner.Executor = &mockCommandExecutor{
		runFn: func(ctx context.Context, dir string, env []string, name string, args ...string) (string, string, int, error) {
			capturedDir = dir
			capturedArgs = args
			return "scratch ok", "", 0, nil
		},
	}

	mockCtx := &aitesting.MockToolContext{
		MockCallbackContext: &aitesting.MockCallbackContext{
			UserId:   "test-user",
			StateObj: &aitesting.MockState{},
		},
	}

	out, err := runner.Execute(mockCtx, &UVRunnerInput{
		ScriptPath: "{project-root}/.tmp/scratch.py",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out.Success {
		t.Errorf("expected success, got false")
	}
	if capturedDir != projectRoot {
		t.Errorf("expected cmd.Dir=%s, got %s", projectRoot, capturedDir)
	}
	if len(capturedArgs) < 2 || capturedArgs[1] != projScriptFile {
		t.Errorf("expected project script (%s), got: %v", projScriptFile, capturedArgs)
	}
}

func TestUVRunner_AgentsBmadPrecedence(t *testing.T) {
	projectRoot, bmadRoot, mountPath := setupTestWorkspace(t)
	defer os.RemoveAll(filepath.Dir(mountPath))

	// Central bmad script
	bmadScriptDir := filepath.Join(bmadRoot, "_bmad", "scripts")
	_ = os.MkdirAll(bmadScriptDir, 0755)
	_ = os.WriteFile(filepath.Join(bmadScriptDir, "memlog.py"), []byte("print('central')"), 0755)

	// Project .agents/_bmad script override
	projAgentsScriptDir := filepath.Join(projectRoot, ".agents", "_bmad", "scripts")
	_ = os.MkdirAll(projAgentsScriptDir, 0755)
	projAgentsScriptFile := filepath.Join(projAgentsScriptDir, "memlog.py")
	_ = os.WriteFile(projAgentsScriptFile, []byte("print('custom agents memlog')"), 0755)

	cfg := &core.Configuration{
		WorkspaceMountPath: mountPath,
		RepoCfg: core.RepoConfiguration{
			OrganizationId: "test-org",
			ProjectId:      "test-proj",
		},
		BmadCfg: core.BmadConfiguration{
			BmadRootPath:   bmadRoot,
			BmadOutputPath: "docs/spec",
		},
	}

	var capturedArgs []string
	var capturedDir string
	runner := NewUVRunnerTool(cfg)
	runner.Executor = &mockCommandExecutor{
		runFn: func(ctx context.Context, dir string, env []string, name string, args ...string) (string, string, int, error) {
			capturedDir = dir
			capturedArgs = args
			return "memlog ok", "", 0, nil
		},
	}

	mockCtx := &aitesting.MockToolContext{
		MockCallbackContext: &aitesting.MockCallbackContext{
			UserId:   "test-user",
			StateObj: &aitesting.MockState{},
		},
	}

	out, err := runner.Execute(mockCtx, &UVRunnerInput{
		ScriptPath: "{project-root}/_bmad/scripts/memlog.py",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out.Success {
		t.Errorf("expected success, got false")
	}
	if capturedDir != projectRoot {
		t.Errorf("expected cmd.Dir=%s, got %s", projectRoot, capturedDir)
	}
	if len(capturedArgs) < 2 || capturedArgs[1] != projAgentsScriptFile {
		t.Errorf("expected .agents override script (%s), got: %v", projAgentsScriptFile, capturedArgs)
	}
}

func TestUVRunner_EnsuresBmadSymlink(t *testing.T) {
	projectRoot, bmadRoot, mountPath := setupTestWorkspace(t)
	defer os.RemoveAll(filepath.Dir(mountPath))

	// Create .agents/_bmad in project workspace
	projAgentsBmad := filepath.Join(projectRoot, ".agents", "_bmad")
	_ = os.MkdirAll(projAgentsBmad, 0755)

	cfg := &core.Configuration{
		WorkspaceMountPath: mountPath,
		RepoCfg: core.RepoConfiguration{
			OrganizationId: "test-org",
			ProjectId:      "test-proj",
		},
		BmadCfg: core.BmadConfiguration{
			BmadRootPath:   bmadRoot,
			BmadOutputPath: "docs/spec",
		},
	}

	runner := NewUVRunnerTool(cfg)
	runner.Executor = &mockCommandExecutor{
		runFn: func(ctx context.Context, dir string, env []string, name string, args ...string) (string, string, int, error) {
			return "ok", "", 0, nil
		},
	}

	mockCtx := &aitesting.MockToolContext{
		MockCallbackContext: &aitesting.MockCallbackContext{
			UserId:   "test-user",
			StateObj: &aitesting.MockState{},
		},
	}

	_, err := runner.Execute(mockCtx, &UVRunnerInput{
		ScriptPath: "{project-root}/test.py",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rootBmad := filepath.Join(projectRoot, "_bmad")
	fi, err := os.Lstat(rootBmad)
	if err != nil {
		t.Fatalf("expected _bmad symlink at project root, got error: %v", err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Errorf("expected _bmad to be a symlink, got mode: %v", fi.Mode())
	}
}



