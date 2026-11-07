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
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"google.com/apf/ai/core"
	"google.golang.org/adk/tool"
)

const (
	maxOutputBytes   = 64 * 1024 // 64 KB limit
	defaultUVTimeout = 30 * time.Second
)

var (
	uvPathOnce   sync.Once
	cachedUVPath string
	cachedUVErr  error
)

// CommandExecutor defines an abstraction over os/exec for deterministic unit testing.
type CommandExecutor interface {
	LookPath(file string) (string, error)
	Run(ctx context.Context, dir string, env []string, name string, args ...string) (stdout, stderr string, exitCode int, err error)
}

type osCommandExecutor struct{}

func (e *osCommandExecutor) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

func (e *osCommandExecutor) Run(ctx context.Context, dir string, env []string, name string, args ...string) (string, string, int, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = env
	cmd.Stdin = nil
	cmd.Cancel = func() error {
		if cmd.Process != nil {
			return cmd.Process.Kill()
		}
		return nil
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}
	return stdoutBuf.String(), stderrBuf.String(), exitCode, err
}

// UVRunnerInput defines the structured arguments for the UVRunnerTool.
type UVRunnerInput struct {
	ScriptPath string   `json:"scriptPath" description:"Path to the Python script to run with uv (e.g. '_bmad/scripts/memlog.py')"`
	Args       []string `json:"args,omitempty" description:"List of arguments to pass to the Python script"`
}

// UVRunnerOutput defines the result returned to the agent.
type UVRunnerOutput struct {
	Stdout       string   `json:"stdout"`
	Stderr       string   `json:"stderr"`
	ExitCode     int      `json:"exitCode"`
	Success      bool     `json:"success"`
	CreatedFiles []string `json:"createdFiles,omitempty"`
}

type UVRunnerTool struct {
	Cfg       *core.Configuration
	Executor  CommandExecutor
	semaphore chan struct{}
	uvOnce    sync.Once
	uvPath    string
	uvErr     error
}

// NewUVRunnerTool initializes the runner with semaphore capacity matching Cloud Run container concurrency.
func NewUVRunnerTool(cfg *core.Configuration) *UVRunnerTool {
	concurrency := 80
	if cfg != nil && cfg.ContainerConcurrency > 0 {
		concurrency = cfg.ContainerConcurrency
	}
	return &UVRunnerTool{
		Cfg:       cfg,
		Executor:  &osCommandExecutor{},
		semaphore: make(chan struct{}, concurrency),
	}
}

func (t *UVRunnerTool) Name() string {
	return "uv_run"
}

func (t *UVRunnerTool) Description() string {
	return "Executes Python scripts and BMad utilities (such as memlog.py, resolve_config.py, brain.py) using the uv toolchain within the project workspace."
}

func (t *UVRunnerTool) Execute(ctx tool.Context, input *UVRunnerInput) (*UVRunnerOutput, error) {
	startTime := time.Now()

	if input == nil || input.ScriptPath == "" {
		return nil, fmt.Errorf("scriptPath is required")
	}

	// 1. Verify `uv` binary is present in container PATH
	if t.Executor == nil {
		t.Executor = &osCommandExecutor{}
	}
	t.uvOnce.Do(func() {
		t.uvPath, t.uvErr = t.Executor.LookPath("uv")
	})
	if t.uvErr != nil {
		return nil, fmt.Errorf("uv executable not found in container PATH. Ensure 'uv' is installed in the container image: %w", t.uvErr)
	}

	// 2. Resolve project workspace directory
	var workspaceMount, userId, projectId, orgId, specLocation string
	if t.Cfg != nil {
		bmadCfg := core.NewBmadConfigFromState(ctx, t.Cfg)
		workspaceMount, _ = filepath.Abs(t.Cfg.WorkspaceMountPath)
		projectId = t.Cfg.RepoCfg.ProjectId
		orgId = t.Cfg.RepoCfg.OrganizationId
		specLocation = bmadCfg.GetSpecLocation()
	}
	if ctx != nil {
		userId = ctx.UserID()
	}
	if specLocation == "" {
		specLocation = "docs/spec"
	}

	projectRoot := filepath.Join(workspaceMount, orgId, userId, projectId)
	if projectRoot != "" {
		projectRoot = filepath.Clean(projectRoot)

		// Ensure upstream BMad compatibility symlink (_bmad -> .agents/_bmad) exists
		agentsBmad := filepath.Join(projectRoot, ".agents", "_bmad")
		if _, err := os.Stat(agentsBmad); err == nil {
			rootBmad := filepath.Join(projectRoot, "_bmad")
			if _, err := os.Lstat(rootBmad); os.IsNotExist(err) {
				_ = os.Symlink(filepath.Join(".agents", "_bmad"), rootBmad)
			}
		}
	}

	localSkillsRoot := filepath.Join(projectRoot, ".agents", "skills")

	// 3. Expand placeholder tokens `{project-root}` and `{skill-root}`
	scriptPath := strings.ReplaceAll(input.ScriptPath, "{project-root}", projectRoot)
	scriptPath = strings.ReplaceAll(scriptPath, "{skill-root}", localSkillsRoot)

	var expandedArgs []string
	for _, arg := range input.Args {
		arg = strings.ReplaceAll(arg, "{project-root}", projectRoot)
		arg = strings.ReplaceAll(arg, "{skill-root}", localSkillsRoot)
		expandedArgs = append(expandedArgs, arg)
	}

	// 4. Resolve absolute script path
	var resolvedScriptPath string

	// Rule 1: If path contains "_bmad/", check project workspace (.agents/_bmad or _bmad)
	if strings.Contains(input.ScriptPath, "_bmad/") || strings.Contains(scriptPath, "_bmad/") {
		targetPath := input.ScriptPath
		if !strings.Contains(targetPath, "_bmad/") {
			targetPath = scriptPath
		}
		idx := strings.Index(targetPath, "_bmad/")
		bmadSubPath := filepath.Clean(targetPath[idx:])
		if projectRoot != "" {
			projectAgentsCandidate := filepath.Clean(filepath.Join(projectRoot, ".agents", bmadSubPath))
			projectCandidate := filepath.Clean(filepath.Join(projectRoot, bmadSubPath))
			if _, err := os.Stat(projectAgentsCandidate); err == nil {
				resolvedScriptPath = projectAgentsCandidate
			} else if _, err := os.Stat(projectCandidate); err == nil {
				resolvedScriptPath = projectCandidate
			}
		}
	}

	// Rule 2: Standard resolution for project scripts and fallback
	if resolvedScriptPath == "" {
		if filepath.IsAbs(scriptPath) {
			resolvedScriptPath = filepath.Clean(scriptPath)
		} else if projectRoot != "" {
			resolvedScriptPath = filepath.Clean(filepath.Join(projectRoot, scriptPath))
		} else {
			resolvedScriptPath = filepath.Clean(scriptPath)
		}

		// Fallback for standard framework scripts under projectRoot/.agents/_bmad/scripts/ or projectRoot/_bmad/scripts/
		if projectRoot != "" {
			if _, err := os.Stat(resolvedScriptPath); os.IsNotExist(err) {
				scriptBase := filepath.Base(scriptPath)
				agentsFwCandidate := filepath.Clean(filepath.Join(projectRoot, ".agents", "_bmad", "scripts", scriptBase))
				rootFwCandidate := filepath.Clean(filepath.Join(projectRoot, "_bmad", "scripts", scriptBase))
				if _, errFw := os.Stat(agentsFwCandidate); errFw == nil {
					resolvedScriptPath = agentsFwCandidate
				} else if _, errFw := os.Stat(rootFwCandidate); errFw == nil {
					resolvedScriptPath = rootFwCandidate
				}
			}
		}
	}

	if info, errStat := os.Stat(resolvedScriptPath); errStat == nil && !info.IsDir() && info.Mode()&0111 == 0 {
		_ = os.Chmod(resolvedScriptPath, info.Mode()|0755)
	}

	// 5. Zero-Trust Security Boundary: Ensure script is strictly within projectRoot
	isInsideProject := projectRoot != "" && (resolvedScriptPath == projectRoot || strings.HasPrefix(resolvedScriptPath, projectRoot+string(filepath.Separator)))

	if !isInsideProject {
		slog.Error("Zero-trust security violation in uv_run", "scriptPath", input.ScriptPath, "resolvedPath", resolvedScriptPath, "projectRoot", projectRoot)
		return nil, fmt.Errorf("zero-trust security violation: script %q is outside allowed project boundary", input.ScriptPath)
	}

	// 6. Track snapshot of specs directory & root instructions before run to detect generated files
	specDir := filepath.Join(projectRoot, specLocation)
	beforeModTimes := getSnapshotModTimes(projectRoot, specDir)

	// 7. Concurrency Control (Matched to Cloud Run concurrency)
	if t.semaphore == nil {
		t.semaphore = make(chan struct{}, 80)
	}
	select {
	case t.semaphore <- struct{}{}:
		defer func() { <-t.semaphore }()
	case <-time.After(15 * time.Second):
		return nil, fmt.Errorf("timed out waiting for available uv runner slot in container")
	}

	execCtx, cancel := context.WithTimeout(context.Background(), defaultUVTimeout)
	defer cancel()

	localBmadRoot := filepath.Join(projectRoot, ".agents", "_bmad")
	if _, err := os.Stat(localBmadRoot); os.IsNotExist(err) {
		localBmadRoot = filepath.Join(projectRoot, "_bmad")
	}

	// 8. Execute `uv run <script> <args...>` anchored in projectRoot
	cmdArgs := append([]string{"run", resolvedScriptPath}, expandedArgs...)
	env := os.Environ()
	env = append(env,
		"UV_SYSTEM_PYTHON=1",
		"UV_CACHE_DIR=/tmp/uv-cache",
		"HOME=/tmp",
		"PYTHONUNBUFFERED=1",
		fmt.Sprintf("PROJECT_ROOT=%s", projectRoot),
		fmt.Sprintf("BMAD_ROOT=%s", localBmadRoot),
	)

	stdoutStr, stderrStr, exitCode, _ := t.Executor.Run(execCtx, projectRoot, env, t.uvPath, cmdArgs...)

	if len(stdoutStr) > maxOutputBytes {
		stdoutStr = stdoutStr[:maxOutputBytes] + "\n...[stdout truncated at 64KB]..."
	}
	if len(stderrStr) > maxOutputBytes {
		stderrStr = stderrStr[:maxOutputBytes] + "\n...[stderr truncated at 64KB]..."
	}

	// 9. Detect created or modified files and register state notifications
	afterModTimes := getSnapshotModTimes(projectRoot, specDir)
	var modifiedFiles []string
	for path, newTime := range afterModTimes {
		oldTime, exists := beforeModTimes[path]
		if !exists || newTime.After(oldTime) {
			relToProj, errRel := filepath.Rel(projectRoot, path)
			if errRel == nil {
				modifiedFiles = append(modifiedFiles, relToProj)
				if ctx != nil && ctx.State() != nil {
					ctx.State().Set(fmt.Sprintf("file-write-notification:%s", uuid.New().String()), relToProj)
				}
			}
		}
	}

	duration := time.Since(startTime)
	logLevel := slog.LevelInfo
	if exitCode != 0 {
		logLevel = slog.LevelWarn
	}
	slog.Log(context.Background(), logLevel, "uv_run execution finished",
		"scriptPath", input.ScriptPath,
		"exitCode", exitCode,
		"durationMs", duration.Milliseconds(),
		"createdFilesCount", len(modifiedFiles),
		"userId", userId,
		"projectId", projectId,
	)

	return &UVRunnerOutput{
		Stdout:       stdoutStr,
		Stderr:       stderrStr,
		ExitCode:     exitCode,
		Success:      exitCode == 0,
		CreatedFiles: modifiedFiles,
	}, nil
}

func getSnapshotModTimes(projectRoot, specDir string) map[string]time.Time {
	res := make(map[string]time.Time)

	// Scan spec directory recursively
	if specDir != "" {
		_ = filepath.WalkDir(specDir, func(p string, d fs.DirEntry, err error) error {
			if err == nil && !d.IsDir() {
				if info, errInfo := d.Info(); errInfo == nil {
					res[p] = info.ModTime()
				}
			}
			return nil
		})
	}

	// Also check root instruction files in projectRoot
	if projectRoot != "" {
		rootFiles := []string{"AGENTS.md", "project-context.md"}
		for _, rf := range rootFiles {
			p := filepath.Join(projectRoot, rf)
			if info, err := os.Stat(p); err == nil && !info.IsDir() {
				res[p] = info.ModTime()
			}
		}
	}

	return res
}
