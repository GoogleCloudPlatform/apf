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
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"google.golang.org/adk/tool"
)

var defaultExcludes = []string{".git", "node_modules", "vendor", "dev", "proc", "sys", ".tmp"}

// FindInput defines the arguments for the FindTool.
type FindInput struct {
	DirectoryPath string   `json:"directoryPath,omitempty"`
	Pattern       string   `json:"pattern,omitempty"`
	MaxDepth      int      `json:"maxDepth,omitempty"`
	Type          string   `json:"type,omitempty"` // "file", "directory", "any"
	Excludes      []string `json:"excludes,omitempty"`
}

// FindOutput defines the return values for the FindTool.
type FindOutput struct {
	FilesPaths []string `json:"filesPaths"`
	TotalFound int      `json:"totalFound"`
}

// FindFiles recursively searches the filesystem matching patterns, depth, type, and exclusions.
func FindFiles(ctx tool.Context, input *FindInput) (*FindOutput, error) {
	if input == nil {
		input = &FindInput{}
	}

	dirPath := input.DirectoryPath
	if dirPath == "" {
		dirPath = "."
	}
	dirPath = filepath.Clean(dirPath)

	info, err := os.Stat(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			slog.Warn("Directory does not exist", "directoryPath", dirPath)
			return &FindOutput{
				FilesPaths: []string{},
				TotalFound: 0,
			}, nil
		}
		slog.Warn("Failed to stat directory", "directoryPath", dirPath, "error", err)
		return &FindOutput{
			FilesPaths: []string{},
			TotalFound: 0,
		}, nil
	}

	excludes := make([]string, 0, len(defaultExcludes)+len(input.Excludes))
	excludes = append(excludes, defaultExcludes...)
	excludes = append(excludes, input.Excludes...)

	var results []string = []string{}

	compiledRe := compileMatcher(input.Pattern)

	// If the provided path is a single file rather than a directory
	if !info.IsDir() {
		relPath := filepath.Base(dirPath)
		if matchesType(false, input.Type) && !isExcluded(info.Name(), relPath, excludes) && matchPatternCompiled(compiledRe, input.Pattern, relPath, info.Name()) {
			results = append(results, dirPath)
		}
		return &FindOutput{
			FilesPaths: results,
			TotalFound: len(results),
		}, nil
	}

	maxDepth := input.MaxDepth
	if maxDepth <= 0 {
		maxDepth = 10
	}

	visitedDirs := make(map[string]bool)
	absDirPath, errAbs := filepath.Abs(dirPath)
	if errAbs != nil {
		absDirPath = dirPath
	}
	if realPath, err := filepath.EvalSymlinks(absDirPath); err == nil {
		visitedDirs[realPath] = true
	} else {
		visitedDirs[absDirPath] = true
	}

	err = walkFindDir(ctx, dirPath, dirPath, 1, maxDepth, compiledRe, input.Pattern, input.Type, excludes, &results, visitedDirs)
	if err != nil {
		slog.Warn("Error while searching files", "directoryPath", dirPath, "error", err)
	}

	sort.Strings(results)

	return &FindOutput{
		FilesPaths: results,
		TotalFound: len(results),
	}, nil
}

func walkFindDir(ctx tool.Context, currentDir string, rootDir string, currentDepth int, maxDepth int, compiledRe *regexp.Regexp, pattern string, typeFilter string, excludes []string, results *[]string, visitedDirs map[string]bool) error {
	if ctx != nil {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	if maxDepth > 0 && currentDepth > maxDepth {
		return nil
	}

	entries, err := os.ReadDir(currentDir)
	if err != nil {
		slog.Warn("Failed to read directory", "directory", currentDir, "error", err)
		return nil
	}

	for _, entry := range entries {
		if ctx != nil {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
		}

		name := entry.Name()
		fullPath := filepath.Join(currentDir, name)

		relPath, err := filepath.Rel(rootDir, fullPath)
		if err != nil {
			relPath = fullPath
		}

		if isExcluded(name, relPath, excludes) {
			continue
		}

		isDir := entry.IsDir()
		isSymlink := entry.Type()&os.ModeSymlink != 0
		var realPath string

		if isSymlink {
			if stat, err := os.Stat(fullPath); err == nil && stat.IsDir() {
				isDir = true
				if resolved, errEval := filepath.EvalSymlinks(fullPath); errEval == nil {
					realPath = resolved
				} else {
					realPath = fullPath
				}
			}
		} else if isDir {
			if resolved, errEval := filepath.EvalSymlinks(fullPath); errEval == nil {
				realPath = resolved
			} else {
				realPath = fullPath
			}
		}

		if isDir {
			if matchesType(true, typeFilter) && matchPatternCompiled(compiledRe, pattern, relPath, name) {
				*results = append(*results, fullPath)
			}

			// Check recursion depth limit
			if currentDepth < maxDepth {
				if realPath == "" {
					if resolved, errEval := filepath.EvalSymlinks(fullPath); errEval == nil {
						realPath = resolved
					} else {
						realPath = fullPath
					}
				}

				isAncestorOrVisited := visitedDirs[realPath] || isAncestor(realPath, fullPath)

				if !isAncestorOrVisited {
					visitedDirs[realPath] = true
					_ = walkFindDir(ctx, fullPath, rootDir, currentDepth+1, maxDepth, compiledRe, pattern, typeFilter, excludes, results, visitedDirs)
				}
			}
		} else {
			if matchesType(false, typeFilter) && matchPatternCompiled(compiledRe, pattern, relPath, name) {
				*results = append(*results, fullPath)
			}
		}
	}

	return nil
}

func isAncestor(ancestor, path string) bool {
	absAncestor, err1 := filepath.Abs(ancestor)
	absPath, err2 := filepath.Abs(path)
	if err1 != nil || err2 != nil {
		return false
	}
	rel, err := filepath.Rel(absAncestor, absPath)
	if err != nil {
		return false
	}
	return rel != "." && !strings.HasPrefix(rel, "..")
}

func matchesType(isDir bool, typeFilter string) bool {
	tf := strings.ToLower(strings.TrimSpace(typeFilter))
	switch tf {
	case "directory", "dir", "directories":
		return isDir
	case "file", "files":
		return !isDir
	case "any", "":
		return true
	default:
		return true
	}
}

func isExcluded(name string, relPath string, excludes []string) bool {
	relPathSlash := filepath.ToSlash(relPath)
	parts := strings.Split(relPathSlash, "/")

	for _, pattern := range excludes {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}

		lowerPat := strings.ToLower(pattern)
		lowerName := strings.ToLower(name)
		lowerRel := strings.ToLower(relPathSlash)

		// Direct name match
		if lowerName == lowerPat {
			return true
		}

		// Check path segments
		for _, part := range parts {
			if strings.ToLower(part) == lowerPat {
				return true
			}
		}

		// Glob match
		if strings.ContainsAny(pattern, "*?[]") {
			if matched, _ := filepath.Match(lowerPat, lowerName); matched {
				return true
			}
			if matched, _ := filepath.Match(lowerPat, lowerRel); matched {
				return true
			}
		}

		// Prefix / prefix path match e.g. "node_modules/" or "build"
		cleanPat := strings.Trim(lowerPat, "/")
		if strings.HasPrefix(lowerRel, cleanPat+"/") || lowerRel == cleanPat {
			return true
		}
	}
	return false
}

func compileMatcher(pattern string) *regexp.Regexp {
	if pattern == "" || pattern == "*" {
		return nil
	}
	if strings.ContainsAny(pattern, "*?[]") || strings.Contains(pattern, "/") {
		if re, err := globToRegexp(pattern); err == nil {
			if ciRe, err := regexp.Compile("(?i)" + re.String()); err == nil {
				return ciRe
			}
		}
	}
	return nil
}

func matchPatternCompiled(compiledRe *regexp.Regexp, pattern string, relPath string, name string) bool {
	if pattern == "" || pattern == "*" {
		return true
	}

	relPathSlash := filepath.ToSlash(relPath)
	lowerPattern := strings.ToLower(pattern)
	lowerName := strings.ToLower(name)
	lowerRel := strings.ToLower(relPathSlash)

	// 1. If compiled regex exists
	if compiledRe != nil {
		if compiledRe.MatchString(relPathSlash) || compiledRe.MatchString(name) {
			return true
		}
		if matched, _ := filepath.Match(lowerPattern, lowerName); matched {
			return true
		}
		if matched, _ := filepath.Match(lowerPattern, lowerRel); matched {
			return true
		}
		return false
	}

	// 2. Exact match (case-insensitive) on name or relPath
	if lowerName == lowerPattern || lowerRel == lowerPattern {
		return true
	}

	// 3. Substring match (case-insensitive) on name or relPath
	if strings.Contains(lowerName, lowerPattern) || strings.Contains(lowerRel, lowerPattern) {
		return true
	}

	return false
}

func matchPattern(pattern string, relPath string, name string) bool {
	return matchPatternCompiled(compileMatcher(pattern), pattern, relPath, name)
}

// FindTool is an ADK tool for recursively searching files and directories.
type FindTool struct{}

// Name returns the name of the tool.
func (t *FindTool) Name() string {
	return "find_files"
}

// Description returns a description of the tool.
func (t *FindTool) Description() string {
	return "Recursively searches the filesystem for files and directories matching a pattern, with support for glob matching, depth limits, type filtering, and path exclusions."
}

// Execute runs the tool with the given arguments.
func (t *FindTool) Execute(ctx tool.Context, input *FindInput) (*FindOutput, error) {
	return FindFiles(ctx, input)
}
