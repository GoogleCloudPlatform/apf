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
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkGitClient_Status(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "git_bench_status_*")
	if err != nil {
		b.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	client := NewGitClient(tempDir)
	repoCfg := &GitRepoConfig{
		UserId:       "bench-user",
		ProjectId:    "bench-proj",
		SpecLocation: "docs/spec",
		BaseBranch:   "main",
		SpecBranch:   "specs/bench-user",
	}

	ctx := context.Background()
	orgID := "bench-org"
	if err := client.EnsureRepoInitialized(ctx, orgID, repoCfg); err != nil {
		b.Fatalf("EnsureRepoInitialized failed: %v", err)
	}

	clonePath := client.getClonePath(orgID, repoCfg)
	// Create sample spec files
	for i := 0; i < 20; i++ {
		p := filepath.Join(clonePath, "docs", "spec", fmt.Sprintf("story_%d.md", i))
		_ = os.MkdirAll(filepath.Dir(p), 0755)
		_ = os.WriteFile(p, []byte("# Benchmark Story Content\n"), 0644)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		items, err := client.GetWorkspaceStatus(ctx, orgID, repoCfg)
		if err != nil {
			b.Fatalf("GetWorkspaceStatus failed: %v", err)
		}
		if len(items) == 0 {
			b.Fatalf("expected items, got 0")
		}
	}
}

func BenchmarkGitClient_LocalCommit(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "git_bench_commit_*")
	if err != nil {
		b.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	client := NewGitClient(tempDir)
	repoCfg := &GitRepoConfig{
		UserId:       "bench-user",
		ProjectId:    "bench-proj",
		SpecLocation: "docs/spec",
		BaseBranch:   "main",
		SpecBranch:   "specs/bench-user",
	}

	ctx := context.Background()
	orgID := "bench-org"
	if err := client.EnsureRepoInitialized(ctx, orgID, repoCfg); err != nil {
		b.Fatalf("EnsureRepoInitialized failed: %v", err)
	}

	clonePath := client.getClonePath(orgID, repoCfg)
	p := filepath.Join(clonePath, "docs", "spec", "bench.md")
	_ = os.MkdirAll(filepath.Dir(p), 0755)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = os.WriteFile(p, []byte(fmt.Sprintf("Iteration %d\n", i)), 0644)
		err := client.LocalCommit(ctx, orgID, repoCfg, "docs/spec/bench.md", fmt.Sprintf("Commit %d", i), "Bench Bot", "bot@example.com")
		if err != nil {
			b.Fatalf("LocalCommit failed: %v", err)
		}
	}
}
