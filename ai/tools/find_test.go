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
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	aitesting "google.com/apf/ai/testing"
)

func setupTestDirectoryStructure(t *testing.T) string {
	t.Helper()
	// Create a temporary directory structure for testing:
	// tmpDir/
	//   file1.txt
	//   file2.log
	//   SKILL.md
	//   .git/
	//     git_file.txt
	//   node_modules/
	//     module_file.js
	//   vendor/
	//     vendor_file.go
	//   custom_ignored/
	//     ignored.txt
	//   subdir1/
	//     file3.txt
	//     file4.go
	//     SKILL.md
	//     subdir2/
	//       file5.txt
	//       subdir3/
	//         file6.txt
	tmpDir, err := os.MkdirTemp("", "find_files_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	filesToCreate := []string{
		"file1.txt",
		"file2.log",
		"SKILL.md",
		filepath.Join(".git", "git_file.txt"),
		filepath.Join("node_modules", "module_file.js"),
		filepath.Join("vendor", "vendor_file.go"),
		filepath.Join("custom_ignored", "ignored.txt"),
		filepath.Join("subdir1", "file3.txt"),
		filepath.Join("subdir1", "file4.go"),
		filepath.Join("subdir1", "SKILL.md"),
		filepath.Join("subdir1", "subdir2", "file5.txt"),
		filepath.Join("subdir1", "subdir2", "subdir3", "file6.txt"),
	}

	for _, relPath := range filesToCreate {
		absPath := filepath.Join(tmpDir, relPath)
		if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
			t.Fatalf("failed to create parent dir for %s: %v", relPath, err)
		}
		if err := os.WriteFile(absPath, []byte("test content"), 0644); err != nil {
			t.Fatalf("failed to write file %s: %v", relPath, err)
		}
	}

	return tmpDir
}

func TestFindFiles(t *testing.T) {
	tmpDir := setupTestDirectoryStructure(t)
	defer os.RemoveAll(tmpDir)

	mockCtx := &aitesting.MockToolContext{
		MockCallbackContext: &aitesting.MockCallbackContext{
			UserId:    "test-user",
			SessionId: "test-session",
		},
	}

	t.Run("Recursive Extension Search", func(t *testing.T) {
		input := &FindInput{
			DirectoryPath: tmpDir,
			Pattern:       "*.go",
		}
		output, err := FindFiles(mockCtx, input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		wantPaths := []string{
			filepath.Join(tmpDir, "subdir1", "file4.go"),
		}
		// Note: vendor/vendor_file.go is excluded by default

		if output.TotalFound != len(wantPaths) {
			t.Errorf("expected TotalFound %d, got %d", len(wantPaths), output.TotalFound)
		}

		sort.Strings(output.FilesPaths)
		sort.Strings(wantPaths)
		if !reflect.DeepEqual(output.FilesPaths, wantPaths) {
			t.Errorf("unexpected paths: got %v, want %v", output.FilesPaths, wantPaths)
		}
	})

	t.Run("Filename Search", func(t *testing.T) {
		input := &FindInput{
			DirectoryPath: tmpDir,
			Pattern:       "SKILL.md",
		}
		output, err := FindFiles(mockCtx, input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		wantPaths := []string{
			filepath.Join(tmpDir, "SKILL.md"),
			filepath.Join(tmpDir, "subdir1", "SKILL.md"),
		}

		if output.TotalFound != len(wantPaths) {
			t.Errorf("expected TotalFound %d, got %d", len(wantPaths), output.TotalFound)
		}

		sort.Strings(output.FilesPaths)
		sort.Strings(wantPaths)
		if !reflect.DeepEqual(output.FilesPaths, wantPaths) {
			t.Errorf("unexpected paths: got %v, want %v", output.FilesPaths, wantPaths)
		}
	})

	t.Run("Case Insensitive Search", func(t *testing.T) {
		input := &FindInput{
			DirectoryPath: tmpDir,
			Pattern:       "skill.md",
		}
		output, err := FindFiles(mockCtx, input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		wantPaths := []string{
			filepath.Join(tmpDir, "SKILL.md"),
			filepath.Join(tmpDir, "subdir1", "SKILL.md"),
		}

		if output.TotalFound != len(wantPaths) {
			t.Errorf("expected TotalFound %d, got %d", len(wantPaths), output.TotalFound)
		}

		sort.Strings(output.FilesPaths)
		sort.Strings(wantPaths)
		if !reflect.DeepEqual(output.FilesPaths, wantPaths) {
			t.Errorf("unexpected paths: got %v, want %v", output.FilesPaths, wantPaths)
		}
	})

	t.Run("Depth-Limited Search maxDepth 1", func(t *testing.T) {
		input := &FindInput{
			DirectoryPath: tmpDir,
			Pattern:       "*",
			MaxDepth:      1,
		}
		output, err := FindFiles(mockCtx, input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Only top-level items under tmpDir (excluding default ignored .git, node_modules, vendor)
		wantPaths := []string{
			filepath.Join(tmpDir, "file1.txt"),
			filepath.Join(tmpDir, "file2.log"),
			filepath.Join(tmpDir, "SKILL.md"),
			filepath.Join(tmpDir, "custom_ignored"),
			filepath.Join(tmpDir, "subdir1"),
		}

		if output.TotalFound != len(wantPaths) {
			t.Errorf("expected TotalFound %d, got %d (%v)", len(wantPaths), output.TotalFound, output.FilesPaths)
		}

		sort.Strings(output.FilesPaths)
		sort.Strings(wantPaths)
		if !reflect.DeepEqual(output.FilesPaths, wantPaths) {
			t.Errorf("unexpected paths: got %v, want %v", output.FilesPaths, wantPaths)
		}
	})

	t.Run("Type Filter Directories Only", func(t *testing.T) {
		input := &FindInput{
			DirectoryPath: tmpDir,
			Type:          "directory",
		}
		output, err := FindFiles(mockCtx, input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		wantPaths := []string{
			filepath.Join(tmpDir, "custom_ignored"),
			filepath.Join(tmpDir, "subdir1"),
			filepath.Join(tmpDir, "subdir1", "subdir2"),
			filepath.Join(tmpDir, "subdir1", "subdir2", "subdir3"),
		}

		if output.TotalFound != len(wantPaths) {
			t.Errorf("expected TotalFound %d, got %d (%v)", len(wantPaths), output.TotalFound, output.FilesPaths)
		}

		sort.Strings(output.FilesPaths)
		sort.Strings(wantPaths)
		if !reflect.DeepEqual(output.FilesPaths, wantPaths) {
			t.Errorf("unexpected paths: got %v, want %v", output.FilesPaths, wantPaths)
		}
	})

	t.Run("Type Filter Files Only", func(t *testing.T) {
		input := &FindInput{
			DirectoryPath: tmpDir,
			Pattern:       "*",
			Type:          "file",
		}
		output, err := FindFiles(mockCtx, input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		wantPaths := []string{
			filepath.Join(tmpDir, "file1.txt"),
			filepath.Join(tmpDir, "file2.log"),
			filepath.Join(tmpDir, "SKILL.md"),
			filepath.Join(tmpDir, "custom_ignored", "ignored.txt"),
			filepath.Join(tmpDir, "subdir1", "file3.txt"),
			filepath.Join(tmpDir, "subdir1", "file4.go"),
			filepath.Join(tmpDir, "subdir1", "SKILL.md"),
			filepath.Join(tmpDir, "subdir1", "subdir2", "file5.txt"),
			filepath.Join(tmpDir, "subdir1", "subdir2", "subdir3", "file6.txt"),
		}

		if output.TotalFound != len(wantPaths) {
			t.Errorf("expected TotalFound %d, got %d (%v)", len(wantPaths), output.TotalFound, output.FilesPaths)
		}

		sort.Strings(output.FilesPaths)
		sort.Strings(wantPaths)
		if !reflect.DeepEqual(output.FilesPaths, wantPaths) {
			t.Errorf("unexpected paths: got %v, want %v", output.FilesPaths, wantPaths)
		}
	})

	t.Run("Exclusion Filtering", func(t *testing.T) {
		input := &FindInput{
			DirectoryPath: tmpDir,
			Pattern:       "*.txt",
			Excludes:      []string{"custom_ignored", "subdir2"},
		}
		output, err := FindFiles(mockCtx, input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		wantPaths := []string{
			filepath.Join(tmpDir, "file1.txt"),
			filepath.Join(tmpDir, "subdir1", "file3.txt"),
		}

		if output.TotalFound != len(wantPaths) {
			t.Errorf("expected TotalFound %d, got %d (%v)", len(wantPaths), output.TotalFound, output.FilesPaths)
		}

		sort.Strings(output.FilesPaths)
		sort.Strings(wantPaths)
		if !reflect.DeepEqual(output.FilesPaths, wantPaths) {
			t.Errorf("unexpected paths: got %v, want %v", output.FilesPaths, wantPaths)
		}
	})

	t.Run("Non-Existent Directory", func(t *testing.T) {
		input := &FindInput{
			DirectoryPath: filepath.Join(tmpDir, "non", "existent", "path"),
			Pattern:       "*",
		}
		output, err := FindFiles(mockCtx, input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if output == nil {
			t.Fatal("expected non-nil output")
		}

		if output.TotalFound != 0 {
			t.Errorf("expected TotalFound 0, got %d", output.TotalFound)
		}

		if len(output.FilesPaths) != 0 {
			t.Errorf("expected empty FilesPaths, got %v", output.FilesPaths)
		}
	})

	t.Run("FindTool Name Description and Execute", func(t *testing.T) {
		findTool := &FindTool{}

		if findTool.Name() != "find_files" {
			t.Errorf("expected tool name 'find_files', got '%s'", findTool.Name())
		}

		if findTool.Description() == "" {
			t.Error("expected non-empty tool description")
		}

		output, err := findTool.Execute(mockCtx, &FindInput{
			DirectoryPath: tmpDir,
			Pattern:       "*.log",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if output.TotalFound != 1 {
			t.Errorf("expected 1 file, got %d", output.TotalFound)
		}
		if len(output.FilesPaths) != 1 || output.FilesPaths[0] != filepath.Join(tmpDir, "file2.log") {
			t.Errorf("unexpected filesPaths: %v", output.FilesPaths)
		}
	})

	t.Run("Empty Pattern and Default Dir", func(t *testing.T) {
		// Change working directory temporarily to tmpDir
		origWd, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get wd: %v", err)
		}
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatalf("failed to chdir to tmpDir: %v", err)
		}
		defer func() {
			_ = os.Chdir(origWd)
		}()

		output, err := FindFiles(mockCtx, &FindInput{
			Pattern:  "*.log",
			MaxDepth: 1,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if output.TotalFound != 1 {
			t.Errorf("expected 1 file, got %d (%v)", output.TotalFound, output.FilesPaths)
		}
	})

	t.Run("Glob Pattern with Wildcards", func(t *testing.T) {
		input := &FindInput{
			DirectoryPath: tmpDir,
			Pattern:       "**/*.txt",
		}
		output, err := FindFiles(mockCtx, input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		wantPaths := []string{
			filepath.Join(tmpDir, "file1.txt"),
			filepath.Join(tmpDir, "custom_ignored", "ignored.txt"),
			filepath.Join(tmpDir, "subdir1", "file3.txt"),
			filepath.Join(tmpDir, "subdir1", "subdir2", "file5.txt"),
			filepath.Join(tmpDir, "subdir1", "subdir2", "subdir3", "file6.txt"),
		}

		if output.TotalFound != len(wantPaths) {
			t.Errorf("expected TotalFound %d, got %d (%v)", len(wantPaths), output.TotalFound, output.FilesPaths)
		}

		sort.Strings(output.FilesPaths)
		sort.Strings(wantPaths)
		if !reflect.DeepEqual(output.FilesPaths, wantPaths) {
			t.Errorf("unexpected paths: got %v, want %v", output.FilesPaths, wantPaths)
		}
	})

	t.Run("System Directory Exclusions", func(t *testing.T) {
		sysDirs := []string{"dev", "proc", "sys", ".tmp"}
		for _, dir := range sysDirs {
			d := filepath.Join(tmpDir, dir)
			_ = os.MkdirAll(d, 0755)
			_ = os.WriteFile(filepath.Join(d, "system_file.txt"), []byte("system"), 0644)
		}

		input := &FindInput{
			DirectoryPath: tmpDir,
			Pattern:       "system_file.txt",
		}
		output, err := FindFiles(mockCtx, input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if output.TotalFound != 0 {
			t.Errorf("expected 0 files in system directories, got %d: %v", output.TotalFound, output.FilesPaths)
		}
	})

	t.Run("Symlink Cycle Prevention", func(t *testing.T) {
		cycleDir := filepath.Join(tmpDir, "cycle_test")
		_ = os.MkdirAll(filepath.Join(cycleDir, "child"), 0755)
		_ = os.WriteFile(filepath.Join(cycleDir, "child", "valid.txt"), []byte("valid"), 0644)

		// Create cyclic symlink child/link_to_parent -> cycleDir
		symlinkPath := filepath.Join(cycleDir, "child", "link_to_parent")
		if err := os.Symlink(cycleDir, symlinkPath); err != nil {
			t.Skipf("skipping symlink test (symlink creation not supported/permitted): %v", err)
		}

		input := &FindInput{
			DirectoryPath: cycleDir,
			Pattern:       "*.txt",
		}
		output, err := FindFiles(mockCtx, input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// It should find valid.txt exactly once without hanging or infinite loop
		wantPaths := []string{
			filepath.Join(cycleDir, "child", "valid.txt"),
		}
		if output.TotalFound != 1 || !reflect.DeepEqual(output.FilesPaths, wantPaths) {
			t.Errorf("expected %v, got %v", wantPaths, output.FilesPaths)
		}
	})
}

