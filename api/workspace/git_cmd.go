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
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
)

var (
	ErrGitCommandFailed = errors.New("git command failed")
)

// GitCmdRunner defines the interface for running git CLI commands.
type GitCmdRunner interface {
	Run(ctx context.Context, dir string, token string, args ...string) (stdout string, stderr string, err error)
}

// DefaultGitCmdRunner executes git CLI in an isolated subprocess.
type DefaultGitCmdRunner struct{}

// Run executes a git CLI command in dir with optional GitHub token auth via in-memory credential helper.
func (r *DefaultGitCmdRunner) Run(ctx context.Context, dir string, token string, args ...string) (string, string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir

	// Clean base environment with non-interactive flags, Protocol v2, and default identity
	cmd.Env = append(os.Environ(),
		"GIT_TERMINAL_PROMPT=0",
		"GIT_PROTOCOL_FROM_USER=2",
		"GIT_AUTHOR_NAME=System Bot",
		"GIT_AUTHOR_EMAIL=bot@foundry.google.com",
		"GIT_COMMITTER_NAME=System Bot",
		"GIT_COMMITTER_EMAIL=bot@foundry.google.com",
	)

	// Inject secure in-memory credential helper when an access token is provided
	if token != "" {
		cmd.Env = append(cmd.Env,
			"APF_GIT_TOKEN="+token,
			"GIT_CONFIG_PARAMETERS='credential.helper=!f() { echo username=oauth2; echo password=$APF_GIT_TOKEN; }; f'",
		)
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
	stdout := strings.TrimRight(stdoutBuf.String(), "\r\n")
	stderr := strings.TrimSpace(stderrBuf.String())

	if err != nil {
		sanitizedStderr := SanitizeGitOutput(stderr, token)
		slog.Debug("git command returned error",
			"args", args,
			"dir", dir,
			"stderr", sanitizedStderr,
			"err", err,
		)
		return stdout, stderr, fmt.Errorf("%w: %s (stderr: %s)", ErrGitCommandFailed, err, sanitizedStderr)
	}

	return stdout, stderr, nil
}

// SanitizeGitOutput redacts sensitive access tokens from git output strings.
func SanitizeGitOutput(output, token string) string {
	if token == "" || output == "" {
		return output
	}
	return strings.ReplaceAll(output, token, "[REDACTED_TOKEN]")
}
