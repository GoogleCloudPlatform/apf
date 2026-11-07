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
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	"google.com/apf/api/core"
)

var (
	// ErrJobNotFound is returned when attempting to retrieve or execute an unregistered job.
	ErrJobNotFound = errors.New("job not found")
	// ErrNilJob is returned when attempting to register a nil job.
	ErrNilJob = errors.New("cannot register nil job")
	// ErrEmptyJobName is returned when attempting to register a job with an empty name.
	ErrEmptyJobName = errors.New("job name cannot be empty")
	// ErrJobAlreadyRegistered is returned when attempting to register a job with a duplicate name.
	ErrJobAlreadyRegistered = errors.New("job already registered")
)

// Job defines the standard interface for executable housekeeping jobs.
type Job interface {
	Name() string
	Description() string
	Run(ctx context.Context, cfg *core.Configuration) error
}

// Registry manages a thread-safe catalog of executable housekeeping jobs.
type Registry struct {
	mu   sync.RWMutex
	jobs map[string]Job
}

// NewRegistry creates and returns an initialized, empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		jobs: make(map[string]Job),
	}
}

// DefaultRegistry is the package-level default job registry.
var DefaultRegistry = NewRegistry()

// Register adds a job to the registry.
// Returns an error if the job is nil, has an empty name, or is already registered.
func (r *Registry) Register(job Job) error {
	if job == nil {
		return ErrNilJob
	}
	name := strings.TrimSpace(job.Name())
	if name == "" {
		return ErrEmptyJobName
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.jobs == nil {
		r.jobs = make(map[string]Job)
	}

	if _, exists := r.jobs[name]; exists {
		return fmt.Errorf("%w: %s", ErrJobAlreadyRegistered, name)
	}
	r.jobs[name] = job
	return nil
}

// Get retrieves a registered job by name.
func (r *Registry) Get(name string) (Job, bool) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return nil, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.jobs == nil {
		return nil, false
	}

	job, ok := r.jobs[trimmed]
	return job, ok
}

// List returns a slice of all registered jobs, sorted alphabetically by name.
func (r *Registry) List() []Job {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.jobs == nil || len(r.jobs) == 0 {
		return []Job{}
	}

	list := make([]Job, 0, len(r.jobs))
	for _, j := range r.jobs {
		list = append(list, j)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Name() < list[j].Name()
	})
	return list
}

// Execute looks up a registered job by name and runs it with the provided context and configuration.
func (r *Registry) Execute(ctx context.Context, name string, cfg *core.Configuration) error {
	trimmed := strings.TrimSpace(name)
	job, ok := r.Get(trimmed)
	if !ok {
		return fmt.Errorf("%w: %s", ErrJobNotFound, trimmed)
	}
	return job.Run(ctx, cfg)
}

// Register registers a job in the DefaultRegistry.
func Register(job Job) error {
	return DefaultRegistry.Register(job)
}

// Get retrieves a job from the DefaultRegistry.
func Get(name string) (Job, bool) {
	return DefaultRegistry.Get(name)
}

// List returns all registered jobs from the DefaultRegistry sorted by name.
func List() []Job {
	return DefaultRegistry.List()
}

// Execute runs a job by name from the DefaultRegistry.
func Execute(ctx context.Context, name string, cfg *core.Configuration) error {
	return DefaultRegistry.Execute(ctx, name, cfg)
}

type dryRunKey struct{}

// ContextWithDryRun returns a new Context carrying the dryRun boolean flag.
func ContextWithDryRun(ctx context.Context, dryRun bool) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, dryRunKey{}, dryRun)
}

// IsDryRun reports whether the context is configured for dry-run execution.
func IsDryRun(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	if v, ok := ctx.Value(dryRunKey{}).(bool); ok {
		return v
	}
	return false
}
