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

package jobs_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"google.com/apf/api/core"
	"google.com/apf/api/jobs"
)

// mockJob implements jobs.Job for testing.
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

func TestRegistry_RegisterAndGet(t *testing.T) {
	reg := jobs.NewRegistry()

	job1 := &mockJob{
		name:        "test-job-1",
		description: "Test job description 1",
	}

	if err := reg.Register(job1); err != nil {
		t.Fatalf("unexpected error registering job: %v", err)
	}

	retrieved, ok := reg.Get("test-job-1")
	if !ok {
		t.Fatalf("expected to find job 'test-job-1'")
	}
	if retrieved.Name() != job1.Name() {
		t.Errorf("expected name %q, got %q", job1.Name(), retrieved.Name())
	}
	if retrieved.Description() != job1.Description() {
		t.Errorf("expected description %q, got %q", job1.Description(), retrieved.Description())
	}

	_, ok = reg.Get("non-existent")
	if ok {
		t.Errorf("expected Get for non-existent job to return false")
	}
}

func TestRegistry_RegisterErrors(t *testing.T) {
	reg := jobs.NewRegistry()

	// Nil job
	if err := reg.Register(nil); !errors.Is(err, jobs.ErrNilJob) {
		t.Errorf("expected ErrNilJob, got %v", err)
	}

	// Empty name
	emptyJob := &mockJob{name: "", description: "no name"}
	if err := reg.Register(emptyJob); !errors.Is(err, jobs.ErrEmptyJobName) {
		t.Errorf("expected ErrEmptyJobName, got %v", err)
	}

	// Whitespace name
	spaceJob := &mockJob{name: "   ", description: "spaces only"}
	if err := reg.Register(spaceJob); !errors.Is(err, jobs.ErrEmptyJobName) {
		t.Errorf("expected ErrEmptyJobName, got %v", err)
	}

	// Valid job
	validJob := &mockJob{name: "valid-job", description: "valid"}
	if err := reg.Register(validJob); err != nil {
		t.Fatalf("unexpected error registering valid job: %v", err)
	}

	// Duplicate job
	duplicateJob := &mockJob{name: "valid-job", description: "duplicate"}
	if err := reg.Register(duplicateJob); !errors.Is(err, jobs.ErrJobAlreadyRegistered) {
		t.Errorf("expected ErrJobAlreadyRegistered, got %v", err)
	}
}

func TestRegistry_List(t *testing.T) {
	reg := jobs.NewRegistry()

	// Empty list
	if list := reg.List(); len(list) != 0 {
		t.Fatalf("expected empty list, got %d items", len(list))
	}

	j1 := &mockJob{name: "zeta-job", description: "zeta"}
	j2 := &mockJob{name: "alpha-job", description: "alpha"}
	j3 := &mockJob{name: "beta-job", description: "beta"}

	_ = reg.Register(j1)
	_ = reg.Register(j2)
	_ = reg.Register(j3)

	list := reg.List()
	if len(list) != 3 {
		t.Fatalf("expected 3 jobs, got %d", len(list))
	}

	// Sorted alphabetically by name
	if list[0].Name() != "alpha-job" || list[1].Name() != "beta-job" || list[2].Name() != "zeta-job" {
		t.Errorf("expected sorted jobs [alpha-job, beta-job, zeta-job], got [%s, %s, %s]",
			list[0].Name(), list[1].Name(), list[2].Name())
	}
}

func TestRegistry_Execute(t *testing.T) {
	reg := jobs.NewRegistry()
	cfg := &core.Configuration{ServiceName: "test-service"}

	executed := false
	var receivedCtx context.Context
	var receivedCfg *core.Configuration

	testJob := &mockJob{
		name:        "run-job",
		description: "runs a test task",
		runFn: func(ctx context.Context, c *core.Configuration) error {
			executed = true
			receivedCtx = ctx
			receivedCfg = c
			return nil
		},
	}

	if err := reg.Register(testJob); err != nil {
		t.Fatalf("failed to register job: %v", err)
	}

	ctx := context.WithValue(context.Background(), "key", "val")
	if err := reg.Execute(ctx, "run-job", cfg); err != nil {
		t.Fatalf("unexpected error executing job: %v", err)
	}

	if !executed {
		t.Errorf("expected job to be executed")
	}
	if receivedCtx != ctx {
		t.Errorf("expected context to be passed to job")
	}
	if receivedCfg != cfg {
		t.Errorf("expected cfg to be passed to job")
	}

	// Non-existent job execution
	err := reg.Execute(ctx, "unknown-job", cfg)
	if err == nil {
		t.Fatalf("expected error executing unknown job, got nil")
	}
	if !errors.Is(err, jobs.ErrJobNotFound) {
		t.Errorf("expected ErrJobNotFound, got %v", err)
	}

	// Failing job execution
	failJob := &mockJob{
		name:        "fail-job",
		description: "fails task",
		runFn: func(ctx context.Context, c *core.Configuration) error {
			return errors.New("simulated failure")
		},
	}
	_ = reg.Register(failJob)
	if err := reg.Execute(ctx, "fail-job", cfg); err == nil || err.Error() != "simulated failure" {
		t.Errorf("expected 'simulated failure' error, got %v", err)
	}
}

func TestDefaultRegistry_Helpers(t *testing.T) {
	jobName := "default-test-job"
	j := &mockJob{
		name:        jobName,
		description: "test default registry",
	}

	if err := jobs.Register(j); err != nil {
		t.Fatalf("failed to register in DefaultRegistry: %v", err)
	}

	retrieved, ok := jobs.Get(jobName)
	if !ok || retrieved.Name() != jobName {
		t.Errorf("failed to Get from DefaultRegistry")
	}

	list := jobs.List()
	found := false
	for _, item := range list {
		if item.Name() == jobName {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected %s in DefaultRegistry.List()", jobName)
	}

	if err := jobs.Execute(context.Background(), jobName, nil); err != nil {
		t.Errorf("unexpected error executing from DefaultRegistry: %v", err)
	}
}

func TestRegistry_Concurrency(t *testing.T) {
	reg := jobs.NewRegistry()
	cfg := &core.Configuration{}
	ctx := context.Background()

	const numGoroutines = 50
	const opsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				jobName := fmt.Sprintf("job-%d-%d", workerID, j)
				job := &mockJob{
					name:        jobName,
					description: "concurrent job",
					runFn: func(ctx context.Context, c *core.Configuration) error {
						return nil
					},
				}

				// Register
				_ = reg.Register(job)

				// Get
				_, _ = reg.Get(jobName)

				// List
				_ = reg.List()

				// Execute
				_ = reg.Execute(ctx, jobName, cfg)
			}
		}(i)
	}

	wg.Wait()
}

func TestRegistry_ZeroValueSafety(t *testing.T) {
	var reg jobs.Registry

	if retrieved, ok := reg.Get("any"); ok || retrieved != nil {
		t.Errorf("expected Get on zero value Registry to return false")
	}
	if list := reg.List(); len(list) != 0 {
		t.Errorf("expected List on zero value Registry to return empty slice, got %d", len(list))
	}

	job := &mockJob{name: "zero-val-job", description: "testing zero value map init"}
	if err := reg.Register(job); err != nil {
		t.Fatalf("expected Register on zero value Registry to succeed, got %v", err)
	}

	if retrieved, ok := reg.Get("  zero-val-job  "); !ok || retrieved.Name() != "zero-val-job" {
		t.Errorf("expected Get with whitespace to find trimmed registered job")
	}
}

func TestDryRunContext(t *testing.T) {
	ctx := context.Background()

	// Default / empty context is not dry-run
	if jobs.IsDryRun(ctx) {
		t.Errorf("expected IsDryRun(context.Background()) to be false")
	}
	if jobs.IsDryRun(nil) {
		t.Errorf("expected IsDryRun(nil) to be false")
	}

	// Explicitly set dry-run to true
	dryCtx := jobs.ContextWithDryRun(ctx, true)
	if !jobs.IsDryRun(dryCtx) {
		t.Errorf("expected IsDryRun(dryCtx) to be true")
	}

	// Explicitly set dry-run to false
	liveCtx := jobs.ContextWithDryRun(dryCtx, false)
	if jobs.IsDryRun(liveCtx) {
		t.Errorf("expected IsDryRun(liveCtx) to be false")
	}
}
