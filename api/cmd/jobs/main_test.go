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

package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"google.com/apf/api/core"
	"google.com/apf/api/jobs"
)

type mockJob struct {
	name        string
	description string
	runFn       func(ctx context.Context, cfg *core.Configuration) error
}

func (m *mockJob) Name() string {
	return m.name
}

func (m *mockJob) Description() string {
	return m.description
}

func (m *mockJob) Run(ctx context.Context, cfg *core.Configuration) error {
	if m.runFn != nil {
		return m.runFn(ctx, cfg)
	}
	return nil
}

func testConfigLoader() *core.Configuration {
	return &core.Configuration{
		ServiceName: "test-jobs-service",
		Environment: "test",
	}
}

func TestCLI_ListFlag(t *testing.T) {
	reg := jobs.NewRegistry()
	_ = reg.Register(&mockJob{name: "workspace-cleanup", description: "Clean inactive workspaces"})
	_ = reg.Register(&mockJob{name: "nfs-audit", description: "Audit NFS disk usage"})

	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{"--list"}, &stdout, &stderr, reg, testConfigLoader)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	out := stdout.String()
	if !strings.Contains(out, "Available Housekeeping Jobs:") {
		t.Errorf("expected header 'Available Housekeeping Jobs:', got %q", out)
	}
	if !strings.Contains(out, "nfs-audit") || !strings.Contains(out, "Audit NFS disk usage") {
		t.Errorf("expected nfs-audit in output, got %q", out)
	}
	if !strings.Contains(out, "workspace-cleanup") || !strings.Contains(out, "Clean inactive workspaces") {
		t.Errorf("expected workspace-cleanup in output, got %q", out)
	}
}

func TestCLI_ListFlag_Empty(t *testing.T) {
	reg := jobs.NewRegistry()

	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{"--list"}, &stdout, &stderr, reg, testConfigLoader)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	out := stdout.String()
	if !strings.Contains(out, "(no jobs registered)") {
		t.Errorf("expected '(no jobs registered)' in output, got %q", out)
	}
}

func TestCLI_PositionalJobExecution(t *testing.T) {
	reg := jobs.NewRegistry()
	executed := false

	_ = reg.Register(&mockJob{
		name:        "sync-workspaces",
		description: "Syncs workspaces",
		runFn: func(ctx context.Context, cfg *core.Configuration) error {
			executed = true
			if cfg.ServiceName != "test-jobs-service" {
				t.Errorf("expected ServiceName 'test-jobs-service', got %q", cfg.ServiceName)
			}
			return nil
		},
	})

	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{"sync-workspaces"}, &stdout, &stderr, reg, testConfigLoader)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d. stderr: %s", code, stderr.String())
	}
	if !executed {
		t.Errorf("expected job to be executed")
	}
	if !strings.Contains(stdout.String(), "completed successfully") {
		t.Errorf("expected success message in stdout, got %q", stdout.String())
	}
}

func TestCLI_FlagJobExecution(t *testing.T) {
	reg := jobs.NewRegistry()
	executed := false

	_ = reg.Register(&mockJob{
		name:        "prune-records",
		description: "Prunes old database records",
		runFn: func(ctx context.Context, cfg *core.Configuration) error {
			executed = true
			return nil
		},
	})

	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{"--name", "prune-records"}, &stdout, &stderr, reg, testConfigLoader)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d. stderr: %s", code, stderr.String())
	}
	if !executed {
		t.Errorf("expected job to be executed")
	}
	if !strings.Contains(stdout.String(), "completed successfully") {
		t.Errorf("expected success message in stdout, got %q", stdout.String())
	}
}

func TestCLI_MissingArguments(t *testing.T) {
	reg := jobs.NewRegistry()
	_ = reg.Register(&mockJob{name: "sample-job", description: "Sample"})

	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{}, &stdout, &stderr, reg, testConfigLoader)

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}

	errOut := stderr.String()
	if !strings.Contains(errOut, "missing required job name") {
		t.Errorf("expected missing job name error, got %q", errOut)
	}
	if !strings.Contains(errOut, "Usage:") {
		t.Errorf("expected usage instructions, got %q", errOut)
	}
	if !strings.Contains(errOut, "sample-job") {
		t.Errorf("expected sample-job listed in available jobs, got %q", errOut)
	}
}

func TestCLI_UnknownJob(t *testing.T) {
	reg := jobs.NewRegistry()
	_ = reg.Register(&mockJob{name: "sample-job", description: "Sample"})

	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{"invalid-job"}, &stdout, &stderr, reg, testConfigLoader)

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}

	errOut := stderr.String()
	if !strings.Contains(errOut, "unknown job \"invalid-job\"") {
		t.Errorf("expected unknown job error, got %q", errOut)
	}
	if !strings.Contains(errOut, "sample-job") {
		t.Errorf("expected sample-job listed in available jobs, got %q", errOut)
	}
}

func TestCLI_JobExecutionError(t *testing.T) {
	reg := jobs.NewRegistry()
	_ = reg.Register(&mockJob{
		name:        "failing-job",
		description: "Always fails",
		runFn: func(ctx context.Context, cfg *core.Configuration) error {
			return errors.New("database connection refused")
		},
	})

	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{"failing-job"}, &stdout, &stderr, reg, testConfigLoader)

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}

	errOut := stderr.String()
	if !strings.Contains(errOut, "database connection refused") {
		t.Errorf("expected error message in stderr, got %q", errOut)
	}
}

func TestCLI_ContextCancellation(t *testing.T) {
	reg := jobs.NewRegistry()
	ctx, cancel := context.WithCancel(context.Background())

	_ = reg.Register(&mockJob{
		name:        "slow-job",
		description: "Slow task interrupted by signal",
		runFn: func(c context.Context, cfg *core.Configuration) error {
			select {
			case <-c.Done():
				return c.Err()
			case <-time.After(5 * time.Second):
				return nil
			}
		},
	})

	// Cancel context immediately to simulate SIGINT/SIGTERM
	cancel()

	var stdout, stderr bytes.Buffer
	code := run(ctx, []string{"slow-job"}, &stdout, &stderr, reg, testConfigLoader)

	if code != 1 {
		t.Fatalf("expected exit code 1 on cancellation, got %d", code)
	}

	errOut := stderr.String()
	if !strings.Contains(errOut, "context canceled") {
		t.Errorf("expected context canceled in stderr, got %q", errOut)
	}
}

func TestCLI_InvalidFlags(t *testing.T) {
	reg := jobs.NewRegistry()

	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{"--invalid-flag-xyz"}, &stdout, &stderr, reg, testConfigLoader)

	if code != 1 {
		t.Fatalf("expected exit code 1 on invalid flag, got %d", code)
	}

	errOut := stderr.String()
	if !strings.Contains(errOut, "Error parsing arguments") {
		t.Errorf("expected argument parse error, got %q", errOut)
	}
}

func TestCLI_HelpFlag(t *testing.T) {
	reg := jobs.NewRegistry()
	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{"-h"}, &stdout, &stderr, reg, testConfigLoader)

	if code != 0 {
		t.Fatalf("expected exit code 0 for help flag, got %d", code)
	}
}

func TestCLI_NilWritersSafety(t *testing.T) {
	reg := jobs.NewRegistry()
	_ = reg.Register(&mockJob{
		name:        "nil-writer-job",
		description: "Runs without stdout/stderr",
		runFn: func(ctx context.Context, cfg *core.Configuration) error {
			return nil
		},
	})

	code := run(context.Background(), []string{"nil-writer-job"}, nil, nil, reg, testConfigLoader)
	if code != 0 {
		t.Fatalf("expected exit code 0 when stdout/stderr are nil, got %d", code)
	}
}

func TestCLI_DryRunFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "positional first", args: []string{"test-dry-run-job", "--dry-run"}},
		{name: "flag first", args: []string{"--dry-run", "test-dry-run-job"}},
		{name: "name flag with dry-run", args: []string{"--name", "test-dry-run-job", "--dry-run"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reg := jobs.NewRegistry()
			receivedDryRun := false

			_ = reg.Register(&mockJob{
				name:        "test-dry-run-job",
				description: "Tests dry run",
				runFn: func(ctx context.Context, cfg *core.Configuration) error {
					receivedDryRun = jobs.IsDryRun(ctx)
					return nil
				},
			})

			var stdout, stderr bytes.Buffer
			code := run(context.Background(), tc.args, &stdout, &stderr, reg, testConfigLoader)
			if code != 0 {
				t.Fatalf("expected exit code 0, got %d. stderr: %s", code, stderr.String())
			}
			if !receivedDryRun {
				t.Errorf("expected job to receive dry-run context as true for args %v", tc.args)
			}
		})
	}
}

func TestCLI_DefaultRegistry_AutoRegisteredJobs(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{"--list"}, &stdout, &stderr, nil, testConfigLoader)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d. stderr: %s", code, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "nfs-audit") {
		t.Errorf("expected 'nfs-audit' in DefaultRegistry list, got %q", out)
	}
	if !strings.Contains(out, "hanging-cleanup") {
		t.Errorf("expected 'hanging-cleanup' in DefaultRegistry list, got %q", out)
	}
}

func TestCLI_UnexpectedExtraArguments(t *testing.T) {
	reg := jobs.NewRegistry()
	_ = reg.Register(&mockJob{
		name:        "my-job",
		description: "My job",
	})

	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{"my-job", "extra-arg"}, &stdout, &stderr, reg, testConfigLoader)
	if code != 1 {
		t.Errorf("expected exit code 1 for extra arguments, got %d", code)
	}
	if !strings.Contains(stderr.String(), "unexpected extra arguments") {
		t.Errorf("expected stderr to mention unexpected extra arguments, got %q", stderr.String())
	}
}
