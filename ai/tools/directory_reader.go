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
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"google.golang.org/adk/tool"
)

// DirectoryReaderInput defines the arguments for the DirectoryReaderTool.
type DirectoryReaderInput struct {
	DirectoryPathOrGlob string `json:"directoryPathOrGlob"`
}

// DirectoryReaderOutput defines the return values for the DirectoryReaderTool.
type DirectoryReaderOutput struct {
	FilesPaths []string `json:"filesPaths"`
}

// ReadDirectoryFromLocal reads a directory from the local filesystem and returns its content.
func ReadDirectoryFromLocal(ctx tool.Context, directoryPathOrGlob string) (*DirectoryReaderOutput, error) {
	if isGlobPath(directoryPathOrGlob) {
		return readGlobFromLocal(directoryPathOrGlob)
	}

	if _, err := os.Stat(directoryPathOrGlob); os.IsNotExist(err) {
		slog.Warn("Directory does not exist", "directoryPath", directoryPathOrGlob)
		return &DirectoryReaderOutput{
			FilesPaths: []string{},
		}, nil
	}

	filesPaths, err := walkDirDepth(directoryPathOrGlob, 1, 3)
	if err != nil {
		return nil, fmt.Errorf("failed to walk directory %q: %w", directoryPathOrGlob, err)
	}

	return &DirectoryReaderOutput{
		FilesPaths: filesPaths,
	}, nil
}

func isGlobPath(path string) bool {
	return strings.ContainsAny(path, "*?[]")
}

func getBaseDir(pattern string) string {
	idx := strings.IndexAny(pattern, "*?[]")
	if idx == -1 {
		return filepath.Clean(pattern)
	}
	dir := pattern[:idx]
	lastSlash := strings.LastIndexAny(dir, "/\\")
	if lastSlash == -1 {
		return "."
	}
	return filepath.Clean(dir[:lastSlash])
}

func globToRegexp(pattern string) (*regexp.Regexp, error) {
	pattern = filepath.ToSlash(pattern)
	var sb strings.Builder
	sb.WriteString("^")
	i := 0
	n := len(pattern)

	// Check if we have "**/" at the very start
	if n >= 3 && pattern[0] == '*' && pattern[1] == '*' && pattern[2] == '/' {
		sb.WriteString("(?:.*/)?")
		i += 3
	}

	for i < n {
		// Check if we have "/**/"
		if i+3 < n && pattern[i] == '/' && pattern[i+1] == '*' && pattern[i+2] == '*' && pattern[i+3] == '/' {
			sb.WriteString("/(?:.*/)?")
			i += 4
			continue
		}
		// Check if we have "/**" at the very end
		if i+2 < n && pattern[i] == '/' && pattern[i+1] == '*' && pattern[i+2] == '*' && i+3 == n {
			sb.WriteString("(?:/.*)?")
			i += 3
			continue
		}

		char := pattern[i]
		switch char {
		case '*':
			if i+1 < n && pattern[i+1] == '*' {
				sb.WriteString(".*")
				i += 2
			} else {
				sb.WriteString("[^/]*")
				i++
			}
		case '?':
			sb.WriteString("[^/]")
			i++
		case '[', ']', '^', '$', '.', '+', '(', ')', '|', '{', '}', '\\':
			sb.WriteString("\\" + string(char))
			i++
		default:
			sb.WriteByte(char)
			i++
		}
	}
	sb.WriteString("$")
	return regexp.Compile(sb.String())
}

func readGlobFromLocal(globPattern string) (*DirectoryReaderOutput, error) {
	baseDir := getBaseDir(globPattern)
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		slog.Warn("Base directory for glob does not exist", "baseDir", baseDir, "globPattern", globPattern)
		return &DirectoryReaderOutput{
			FilesPaths: []string{},
		}, nil
	}

	allFiles, err := walkDirDepth(baseDir, 1, 3)
	if err != nil {
		return nil, fmt.Errorf("failed to walk directory %q for glob: %w", baseDir, err)
	}

	re, err := globToRegexp(globPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to compile glob pattern %q: %w", globPattern, err)
	}

	var matchedFiles []string
	for _, path := range allFiles {
		normalized := filepath.ToSlash(path)
		if re.MatchString(normalized) {
			matchedFiles = append(matchedFiles, path)
		}
	}

	return &DirectoryReaderOutput{
		FilesPaths: matchedFiles,
	}, nil
}

func walkDirDepth(dirPath string, currentDepth int, maxDepth int) ([]string, error) {
	if currentDepth > maxDepth {
		return nil, nil
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		path := filepath.Join(dirPath, entry.Name())
		if entry.IsDir() {
			subFiles, err := walkDirDepth(path, currentDepth+1, maxDepth)
			if err != nil {
				return nil, err
			}
			files = append(files, subFiles...)
		} else {
			files = append(files, path)
		}
	}
	return files, nil
}

// DirectoryReaderTool is an ADK tool for reading files from the local filesystem.
type DirectoryReaderTool struct{}

// Name returns the name of the tool.
func (t *DirectoryReaderTool) Name() string {
	return "directory_reader"
}

// Description returns a description of the tool.
func (t *DirectoryReaderTool) Description() string {
	return "Given a directory path or a glob pattern, returns the file paths of the files in the directory. It recurses into sub-directories for 2 levels deep. It does not return the content of the files. It does not return the directory paths."
}

// Execute runs the tool with the given arguments.
func (t *DirectoryReaderTool) Execute(ctx tool.Context, input *DirectoryReaderInput) (*DirectoryReaderOutput, error) {
	return ReadDirectoryFromLocal(ctx, input.DirectoryPathOrGlob)
}
