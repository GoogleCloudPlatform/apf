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

package bmad

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"google.com/apf/ai/core"
	"google.golang.org/adk/tool"
)

type PathSafetyInterceptor struct {
	BaseInterceptor
	cfg *core.Configuration
}

func (p *PathSafetyInterceptor) Name() string { return "PathSafety" }

func isScriptOrScratchFile(path string) bool {
	clean := filepath.ToSlash(filepath.Clean(path))
	if strings.HasPrefix(clean, ".tmp/") || clean == ".tmp" || strings.HasPrefix(clean, "tmp/") || strings.HasPrefix(clean, "/tmp/") || strings.Contains(clean, "/.tmp/") {
		return true
	}
	if strings.HasPrefix(clean, "_bmad/") || strings.Contains(clean, "/_bmad/") || strings.HasPrefix(clean, "scripts/") || strings.Contains(clean, "/scripts/") {
		return true
	}
	ext := strings.ToLower(filepath.Ext(clean))
	switch ext {
	case ".py", ".sh", ".bash", ".js", ".ts", ".mjs", ".cjs", ".rb", ".php", ".go", ".rs":
		return true
	}
	return false
}

func (p *PathSafetyInterceptor) Before(ctx tool.Context, tool tool.Tool, args map[string]any) (map[string]any, error) {
	if tool.Name() != "file_writer" && tool.Name() != "file_reader" && tool.Name() != "directory_reader" && tool.Name() != "find_files" {
		return nil, nil
	}

	repoCfg, err := core.NewRepoConfigFromState(ctx, p.cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to load repo configuration: %w", err)
	}
	bmadCfg := core.NewBmadConfigFromState(ctx, p.cfg)

	p.cfg.BmadCfg = *bmadCfg
	p.cfg.RepoCfg = *repoCfg

	userId := ctx.UserID()
	projectId := repoCfg.ProjectId
	if userId == "" || projectId == "" {
		return nil, fmt.Errorf("user ID or Project ID is missing, cannot perform path safety check")
	}

	absWorkspaceMountPath, err := filepath.Abs(p.cfg.WorkspaceMountPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for WorkspaceMountPath: %w", err)
	}

	projectRoot := filepath.Join(absWorkspaceMountPath, repoCfg.OrganizationId, userId, projectId)
	absProjectRoot := filepath.Clean(projectRoot)

	pathParamName := "filePath"
	if tool.Name() == "directory_reader" {
		pathParamName = "directoryPathOrGlob"
	} else if tool.Name() == "find_files" {
		pathParamName = "directoryPath"
	}

	filePathVal, ok := args[pathParamName]
	if !ok && tool.Name() != "find_files" {
		return nil, nil
	}

	filePath, _ := filePathVal.(string)
	if filePath == "" && tool.Name() == "find_files" {
		filePath = "."
	}
	if filePath == "" {
		return nil, nil
	}

	// Strip "file:" URI prefix if present (e.g. from persistent_facts in BMad)
	filePath = strings.TrimPrefix(filePath, "file:")

	// Expand placeholders
	if strings.Contains(filePath, "{project-root}") {
		filePath = strings.ReplaceAll(filePath, "{project-root}", absProjectRoot)
	}
	if strings.Contains(filePath, "{skill-root}") {
		filePath = strings.ReplaceAll(filePath, "{skill-root}", filepath.Join(absProjectRoot, ".agents", "skills"))
	}
	if strings.Contains(filePath, "{project-name}") {
		filePath = strings.ReplaceAll(filePath, "{project-name}", repoCfg.ProjectName)
	}

	var finalPath string

	// Check if the path is target-bound to local framework directories
	if tool.Name() == "file_reader" || tool.Name() == "directory_reader" || tool.Name() == "find_files" {
		if strings.Contains(filePath, "_bmad/") {
			// Check if candidate exists in project workspace (.agents/_bmad or _bmad)
			idx := strings.Index(filePath, "_bmad/")
			bmadSubPath := filepath.Clean(filePath[idx:])
			projectAgentsCandidate := filepath.Join(absProjectRoot, ".agents", bmadSubPath)
			projectCandidate := filepath.Join(absProjectRoot, bmadSubPath)
			if _, statErr := os.Stat(projectAgentsCandidate); statErr == nil {
				finalPath = projectAgentsCandidate
				args[pathParamName] = projectAgentsCandidate
			} else if _, statErr := os.Stat(projectCandidate); statErr == nil {
				finalPath = projectCandidate
				args[pathParamName] = projectCandidate
			}
		}
	}

	if finalPath == "" {
		if !filepath.IsAbs(filePath) {
			// Anchor relative paths to the user's project workspace root
			finalPath = filepath.Join(absProjectRoot, filePath)
			slog.Info("Detected relative path, anchoring to project workspace", "oldPath", filePath, "newPath", finalPath)
			args[pathParamName] = finalPath
		} else {
			finalPath = filePath
			// Safely resolve absolute path to enforce spec pattern correction
			if bmadCfg.GetSpecLocation() != "" {
				specPathPattern := filepath.Join(repoCfg.OrganizationId, userId, projectId, bmadCfg.GetSpecLocation())
				if strings.Contains(filePath, specPathPattern) {
					absFilePath, errAbs := filepath.Abs(filePath)
					if errAbs != nil {
						absFilePath = filepath.Clean(filePath)
					}
					if !strings.HasPrefix(absFilePath, absWorkspaceMountPath) {
						idx := strings.Index(filePath, specPathPattern)
						if idx != -1 {
							suffix := filePath[idx:]
							newPath := filepath.Join(absWorkspaceMountPath, suffix)
							slog.Warn("Detected file path outside workspace mount path containing spec path pattern, correcting path", "oldPath", filePath, "newPath", newPath)
							finalPath = newPath
							args[pathParamName] = newPath
						}
					}
				}
			}
			args[pathParamName] = finalPath
		}
	}

	// Zero-Trust Guardrail: Force all file writes to be inside specs folder, .tmp folder, or root instructions
	if tool.Name() == "file_writer" && finalPath != "" {
		absSpecsDir := filepath.Clean(filepath.Join(absProjectRoot, bmadCfg.GetSpecLocation()))
		absTmpDir := filepath.Clean(filepath.Join(absProjectRoot, ".tmp"))
		absFilePath := filepath.Clean(finalPath)

		isInsideSpecs := bmadCfg.GetSpecLocation() != "" && (absFilePath == absSpecsDir || strings.HasPrefix(absFilePath, absSpecsDir+string(filepath.Separator)))
		isInsideTmp := absFilePath == absTmpDir || strings.HasPrefix(absFilePath, absTmpDir+string(filepath.Separator))

		baseNameLower := strings.ToLower(filepath.Base(absFilePath))
		isRootInstructionFile := baseNameLower == "agents.md" || baseNameLower == "project-context.md"

		if !isInsideSpecs && !isInsideTmp && !isRootInstructionFile {
			baseName := filepath.Base(absFilePath)
			if isScriptOrScratchFile(filePath) || isScriptOrScratchFile(finalPath) {
				newPath := filepath.Clean(filepath.Join(absTmpDir, baseName))
				slog.Warn("Zero-Trust Guardrail: Redirecting script/scratch file write to .tmp folder", "oldPath", finalPath, "newPath", newPath)
				args["filePath"] = newPath
			} else if bmadCfg.GetSpecLocation() != "" {
				relToProj, err := filepath.Rel(absProjectRoot, absFilePath)
				if err != nil || strings.HasPrefix(relToProj, "..") {
					newPath := filepath.Clean(filepath.Join(absSpecsDir, baseName))
					slog.Warn("Zero-Trust Guardrail: Forcing path to allowed directory (outside project root)", "oldPath", finalPath, "newPath", newPath)
					args["filePath"] = newPath
				} else {
					cleanSpecLoc := filepath.Clean(bmadCfg.GetSpecLocation())
					specSegments := strings.Split(cleanSpecLoc, string(filepath.Separator))
					relSegments := strings.Split(relToProj, string(filepath.Separator))

					matchCount := 0
					for matchCount < len(specSegments) && matchCount < len(relSegments) {
						if specSegments[matchCount] == relSegments[matchCount] {
							matchCount++
						} else {
							break
						}
					}

					var targetRelPath string
					if matchCount > 0 {
						remainingSpec := ""
						if matchCount < len(specSegments) {
							remainingSpec = filepath.Join(specSegments[matchCount:]...)
						}
						remainingRel := ""
						if matchCount < len(relSegments) {
							remainingRel = filepath.Join(relSegments[matchCount:]...)
						}
						targetRelPath = filepath.Join(
							filepath.Join(specSegments[:matchCount]...),
							remainingSpec,
							remainingRel,
						)
					} else {
						targetRelPath = filepath.Join(cleanSpecLoc, relToProj)
					}
					newPath := filepath.Clean(filepath.Join(absProjectRoot, targetRelPath))
					slog.Warn("Zero-Trust Guardrail: Forcing path to specs directory", "oldPath", finalPath, "newPath", newPath)
					args["filePath"] = newPath
				}
			}
		}
	}

	// Verify allowed reading boundaries for file_reader, directory_reader, and find_files
	if (tool.Name() == "file_reader" || tool.Name() == "directory_reader" || tool.Name() == "find_files") && finalPath != "" {
		absFilePath := filepath.Clean(finalPath)

		isInsideProject := absFilePath == absProjectRoot || strings.HasPrefix(absFilePath, absProjectRoot+string(filepath.Separator))

		if !isInsideProject {
			return nil, fmt.Errorf("failed to read %s: path resides outside allowed directories (zero-trust perimeter breach)", filePath)
		}
	}

	return nil, nil
}
