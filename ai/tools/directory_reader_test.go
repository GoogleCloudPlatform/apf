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

func TestReadDirectoryFromLocal(t *testing.T) {
	// Create a temporary directory structure for testing:
	// tmpDir/
	//   file1.txt
	//   file2.log
	//   subdir1/
	//     file3.txt
	//     file4.go
	//     subdir2/
	//       file5.txt   (depth 3, should be returned)
	//       subdir3/
	//         file6.txt (depth 4, should not be returned)
	tmpDir, err := os.MkdirTemp("", "dir_reader_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create files
	filesToCreate := []string{
		"file1.txt",
		"file2.log",
		filepath.Join("subdir1", "file3.txt"),
		filepath.Join("subdir1", "file4.go"),
		filepath.Join("subdir1", "subdir2", "file5.txt"),
		filepath.Join("subdir1", "subdir2", "subdir3", "file6.txt"),
	}

	for _, relPath := range filesToCreate {
		absPath := filepath.Join(tmpDir, relPath)
		if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
			t.Fatalf("failed to create parent dir for %s: %v", relPath, err)
		}
		if err := os.WriteFile(absPath, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to write file %s: %v", relPath, err)
		}
	}

	mockCtx := &aitesting.MockToolContext{
		MockCallbackContext: &aitesting.MockCallbackContext{
			UserId:    "test-user",
			SessionId: "test-session",
		},
	}

	tests := []struct {
		name        string
		pathOrGlob  string
		wantPaths   []string
		expectError bool
	}{
		{
			name:       "Standard directory read - 3 levels deep",
			pathOrGlob: tmpDir,
			wantPaths: []string{
				filepath.Join(tmpDir, "file1.txt"),
				filepath.Join(tmpDir, "file2.log"),
				filepath.Join(tmpDir, "subdir1", "file3.txt"),
				filepath.Join(tmpDir, "subdir1", "file4.go"),
				filepath.Join(tmpDir, "subdir1", "subdir2", "file5.txt"),
			},
			expectError: false,
		},
		{
			name:       "Glob pattern matching specific extension",
			pathOrGlob: filepath.Join(tmpDir, "*.txt"),
			wantPaths: []string{
				filepath.Join(tmpDir, "file1.txt"),
			},
			expectError: false,
		},
		{
			name:       "Recursive glob pattern matching",
			pathOrGlob: filepath.Join(tmpDir, "**", "*.txt"),
			wantPaths: []string{
				filepath.Join(tmpDir, "file1.txt"),
				filepath.Join(tmpDir, "subdir1", "file3.txt"),
				filepath.Join(tmpDir, "subdir1", "subdir2", "file5.txt"),
			},
			expectError: false,
		},
		{
			name:       "Glob pattern matching specific subdirectory files",
			pathOrGlob: filepath.Join(tmpDir, "subdir1", "*.go"),
			wantPaths: []string{
				filepath.Join(tmpDir, "subdir1", "file4.go"),
			},
			expectError: false,
		},
		{
			name:        "Non-existent directory",
			pathOrGlob:  filepath.Join(tmpDir, "non-existent"),
			wantPaths:   []string{},
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			output, err := ReadDirectoryFromLocal(mockCtx, tc.pathOrGlob)
			if (err != nil) != tc.expectError {
				t.Fatalf("expected error status: %v, got: %v", tc.expectError, err)
			}

			if err == nil {
				// Sort both slices to ensure order doesn't affect equality check
				sort.Strings(output.FilesPaths)
				sort.Strings(tc.wantPaths)

				if !reflect.DeepEqual(output.FilesPaths, tc.wantPaths) {
					t.Errorf("unexpected files: got %v, want %v", output.FilesPaths, tc.wantPaths)
				}
			}
		})
	}
}
