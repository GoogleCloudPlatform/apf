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

package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"

	"google.com/apf/api/agent"
	"google.com/apf/api/core"
)

// SubprocessStream wraps a running subprocess stdout/stderr and implements io.ReadCloser.
type SubprocessStream struct {
	stdout io.ReadCloser
	stderr io.ReadCloser
	cmd    *exec.Cmd
}

// Read reads from the subprocess's stdout pipe.
func (s *SubprocessStream) Read(p []byte) (n int, err error) {
	return s.stdout.Read(p)
}

// Close closes the pipes and waits for the subprocess to finish.
func (s *SubprocessStream) Close() error {
	s.stdout.Close()
	s.stderr.Close()
	if err := s.cmd.Wait(); err != nil {
		slog.Error("Subprocess execution failed on Close", "error", err)
		return err
	}
	return nil
}

// RunWorkflowRequest represents the payload structure required by run_agent_cli.py
type RunWorkflowRequest struct {
	AppName    string                 `json:"appName"`
	UserID     string                 `json:"userId"`
	SessionID  string                 `json:"sessionId"`
	NewMessage agent.ProxyNewMessage  `json:"newMessage"`
	State      map[string]interface{} `json:"state"`
}

// RunWorkflowDirect spawns the run_agent_cli.py python process directly and returns its stdout stream.
func RunWorkflowDirect(
	ctx context.Context,
	workflowID string,
	sessionID string,
	projectID string,
	userID string,
	organizationID string,
	newMessage agent.ProxyNewMessage,
) (io.ReadCloser, error) {
	cfg := core.NewConfig()

	// 1. Resolve working directory path
	if organizationID == "" || userID == "" || projectID == "" {
		return nil, fmt.Errorf("missing organizationID, userID, or projectID for workspace resolution")
	}
	workingDir := filepath.Join(cfg.WorkspaceMountPath, organizationID, userID, projectID)

	// Verify working directory exists
	if _, err := os.Stat(workingDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("workspace directory does not exist: %s", workingDir)
	}

	// 2. Resolve BMad root path from environment or default
	bmadRootPath := os.Getenv("BMAD_ROOT_PATH")
	if bmadRootPath == "" {
		bmadRootPath = "workspaces/config/bmad"
	}
	// If relative, resolve to absolute path
	if !filepath.IsAbs(bmadRootPath) {
		absPath, err := filepath.Abs(bmadRootPath)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve absolute path for bmad root: %w", err)
		}
		bmadRootPath = absPath
	}

	// 3. Construct payload JSON
	state := map[string]interface{}{
		"organizationId": organizationID,
		"projectId":      projectID,
	}
	payload := RunWorkflowRequest{
		AppName:    workflowID,
		UserID:     userID,
		SessionID:  sessionID,
		NewMessage: newMessage,
		State:      state,
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize runner payload: %w", err)
	}

	// 4. Resolve Python executable and runner script path
	// Typically python is in the virtualenv of the runners folder: runners/adk/.venv/bin/python3
	// We'll find the relative path from the current workspace root
	wd, err := os.Getwd()
	var runnerRoot string
	if err == nil {
		// If we are running from api/, the runners folder is in the parent directory
		runnerRoot = filepath.Join(wd, "..")
	} else {
		runnerRoot = "."
	}

	pythonPath := filepath.Join(runnerRoot, "runners", "adk", ".venv", "bin", "python3")
	cliScriptPath := filepath.Join(runnerRoot, "runners", "adk", "run_agent_cli.py")

	// Fallback to standard python3 if the venv one doesn't exist (e.g. during dev/custom setups)
	if _, err := os.Stat(pythonPath); os.IsNotExist(err) {
		slog.Warn("Virtual environment python not found, falling back to system python3", "path", pythonPath)
		pythonPath = "python3"
	}

	// 5. Setup and spawn command
	cmd := exec.CommandContext(
		ctx,
		pythonPath,
		cliScriptPath,
		string(payloadJSON),
		workingDir,
		bmadRootPath,
	)
	cmd.Dir = workingDir
	cmd.Env = os.Environ() // Pass host environment variables

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	slog.Info("Spawning isolated runner subprocess directly from Go backend",
		"python", pythonPath,
		"script", cliScriptPath,
		"cwd", workingDir,
		"bmadRoot", bmadRootPath,
	)

	if err := cmd.Start(); err != nil {
		stdoutPipe.Close()
		stderrPipe.Close()
		return nil, fmt.Errorf("failed to start runner subprocess: %w", err)
	}

	// 6. Monitor stderr logs in a background goroutine to log any internal python logs/errors
	go func() {
		defer stderrPipe.Close()
		buf := make([]byte, 1024)
		for {
			n, err := stderrPipe.Read(buf)
			if n > 0 {
				slog.Debug(fmt.Sprintf("[Runner Subprocess] %s", string(buf[:n])))
			}
			if err != nil {
				break
			}
		}
	}()

	return &SubprocessStream{
		stdout: stdoutPipe,
		stderr: stderrPipe,
		cmd:    cmd,
	}, nil
}
