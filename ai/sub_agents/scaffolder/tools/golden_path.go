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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v3"

	"google.golang.org/adk/tool"
)

// GetGoldenPathArgs defines the input for the get_golden_path tool.
type GetGoldenPathArgs struct {
	GoldenPathID string `json:"golden_path_id" jsonschema:"Golden path ID"`
}

// GetGoldenPathResult defines the output for the get_golden_path tool.
type GetGoldenPathResult struct {
	GoldenPath string `json:"golden_path"`
}

// GetGoldenPath returns the golden path for a given golden path ID.
func GetGoldenPath(_ tool.Context, args GetGoldenPathArgs) (GetGoldenPathResult, error) {
	catalogDir, err := resolveServiceCatalogDir()
	if err != nil {
		return GetGoldenPathResult{}, err
	}
	catalogPath := filepath.Join(catalogDir, "catalog.yaml")

	data, err := os.ReadFile(catalogPath)
	if err != nil {
		return GetGoldenPathResult{}, fmt.Errorf("failed to read catalog file: %w", err)
	}

	type Catalog struct {
		Archetypes []map[string]interface{} `yaml:"archetypes"`
	}
	var catalog Catalog
	if err := yaml.Unmarshal(data, &catalog); err != nil {
		return GetGoldenPathResult{}, fmt.Errorf("failed to parse catalog: %w", err)
	}

	for _, entry := range catalog.Archetypes {
		id, ok := entry["id"].(string)
		if ok && id == args.GoldenPathID {
			res, err := json.Marshal(entry)
			if err != nil {
				return GetGoldenPathResult{}, fmt.Errorf("failed to marshal entry: %w", err)
			}
			return GetGoldenPathResult{GoldenPath: string(res)}, nil
		}
	}

	return GetGoldenPathResult{}, fmt.Errorf("golden path '%s' not found in catalog", args.GoldenPathID)
}

// GetGoldenPathTemplateArgs defines the input for the get_golden_path_template tool.
type GetGoldenPathTemplateArgs struct {
	GoldenPathID string `json:"golden_path" jsonschema:"Golden path archetype"`
}

// GetGoldenPathTemplateResult defines the output for the get_golden_path_template tool.
type GetGoldenPathTemplateResult struct {
	FilesContent map[string]string `json:"files_content" jsonschema:"Files content"`
}

// GetGoldenPathTemplate returns the golden path template for a given golden path ID.
func GetGoldenPathTemplate(_ tool.Context, args GetGoldenPathTemplateArgs) (GetGoldenPathTemplateResult, error) {
	var archetype map[string]interface{}
	id := args.GoldenPathID
	if err := json.Unmarshal([]byte(args.GoldenPathID), &archetype); err == nil {
		if parsedID, ok := archetype["id"].(string); ok && parsedID != "" {
			id = parsedID
		}
	}

	catalogDir, err := resolveServiceCatalogDir()
	if err != nil {
		return GetGoldenPathTemplateResult{}, err
	}
	targetDir := filepath.Join(catalogDir, "archetypes", id)

	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		return GetGoldenPathTemplateResult{}, fmt.Errorf("archetype directory not found: %s", targetDir)
	}

	filesContent := make(map[string]string)
	err = filepath.Walk(targetDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		relPath, err := filepath.Rel(targetDir, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil // Skip files we cannot read
		}
		filesContent[relPath] = string(data)
		return nil
	})

	if err != nil {
		return GetGoldenPathTemplateResult{}, fmt.Errorf("failed to read local archetype files: %w", err)
	}

	return GetGoldenPathTemplateResult{FilesContent: filesContent}, nil
}

// resolveServiceCatalogDir dynamically locates the service-catalog directory
// whether it is being run alongside the binary in production, or via go run.
func resolveServiceCatalogDir() (string, error) {
	pathsToCheck := []string{}

	// 1. Relative to the executable
	exePath, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exePath)
		pathsToCheck = append(pathsToCheck,
			filepath.Join(exeDir, "sub_agents/scaffolder/service-catalog"),
			filepath.Join(exeDir, "service-catalog"),
		)
	}

	// 2. Current working directory
	wd, err := os.Getwd()
	if err == nil {
		pathsToCheck = append(pathsToCheck,
			filepath.Join(wd, "sub_agents/scaffolder/service-catalog"),
			filepath.Join(wd, "service-catalog"),
		)
	}

	// 3. Fallback to build-time source path via runtime.Caller
	_, filename, _, ok := runtime.Caller(0)
	if ok {
		// filename is .../scaffolder/tools/golden_path.go
		// filepath.Dir(filename) is .../scaffolder/tools
		// filepath.Dir(filepath.Dir(filename)) is .../scaffolder
		pathsToCheck = append(pathsToCheck,
			filepath.Join(filepath.Dir(filepath.Dir(filename)), "service-catalog"),
		)
	}

	for _, target := range pathsToCheck {
		if info, err := os.Stat(target); err == nil && info.IsDir() {
			return target, nil
		}
	}

	return "", fmt.Errorf("could not locate service-catalog directory (checked %v)", pathsToCheck)
}
