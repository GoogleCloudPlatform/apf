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
	"path/filepath"
	"regexp"
	"strings"

	"google.com/apf/ai/core"
	"google.golang.org/adk/tool"
)

type ContentSanitizerInterceptor struct {
	BaseInterceptor
	cfg *core.Configuration
}

func (c *ContentSanitizerInterceptor) Name() string { return "ContentSanitizer" }

func (c *ContentSanitizerInterceptor) Before(ctx tool.Context, tool tool.Tool, args map[string]any) (map[string]any, error) {
	if tool.Name() != "file_writer" {
		return nil, nil
	}

	repoCfg, err := core.NewRepoConfigFromState(ctx, c.cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to load repo configuration: %w", err)
	}

	userId := ctx.UserID()
	projectId := repoCfg.ProjectId
	if userId == "" || projectId == "" {
		return nil, nil
	}

	contentVal, ok := args["content"]
	if !ok {
		return nil, nil
	}

	content, ok := contentVal.(string)
	if !ok || content == "" {
		return nil, nil
	}

	args["content"] = c.sanitizeContent(content, userId, projectId)

	return nil, nil
}

func (c *ContentSanitizerInterceptor) After(ctx tool.Context, tool tool.Tool, args, result map[string]any, err error) (map[string]any, error) {
	if tool.Name() != "file_writer" && tool.Name() != "file_reader" && tool.Name() != "directory_reader" {
		return result, err
	}

	repoCfg, errRepo := core.NewRepoConfigFromState(ctx, c.cfg)
	if errRepo != nil {
		return result, err
	}

	userId := ctx.UserID()
	projectId := repoCfg.ProjectId
	if userId == "" || projectId == "" {
		return result, err
	}

	absWorkspaceMountPath, errAbs := filepath.Abs(c.cfg.WorkspaceMountPath)
	if errAbs != nil {
		return result, err
	}
	projectRoot := filepath.Join(absWorkspaceMountPath, repoCfg.OrganizationId, userId, projectId)

	if err != nil {
		return result, c.sanitizeError(err, projectRoot)
	}

	if result == nil {
		return nil, nil
	}

	switch tool.Name() {
	case "file_reader":
		if val, ok := result["filePath"]; ok {
			if pathStr, ok := val.(string); ok {
				result["filePath"] = c.sanitizePath(pathStr, projectRoot)
			}
		}
		if val, ok := result["content"]; ok {
			if contentStr, ok := val.(string); ok {
				result["content"] = c.sanitizeContent(contentStr, userId, projectId)
			}
		}
	case "directory_reader":
		if val, ok := result["filesPaths"]; ok {
			result["filesPaths"] = c.sanitizePaths(val, projectRoot)
		}
	}

	return result, nil
}

func (c *ContentSanitizerInterceptor) sanitizePath(path string, projectRoot string) string {
	if filepath.IsAbs(path) {
		rel, err := filepath.Rel(projectRoot, path)
		if err == nil && !strings.HasPrefix(rel, "..") {
			return rel
		}
	}
	return path
}

func (c *ContentSanitizerInterceptor) sanitizePaths(paths any, projectRoot string) any {
	if slice, ok := paths.([]string); ok {
		res := make([]string, len(slice))
		for i, p := range slice {
			res[i] = c.sanitizePath(p, projectRoot)
		}
		return res
	}
	if slice, ok := paths.([]any); ok {
		res := make([]any, len(slice))
		for i, p := range slice {
			if pStr, ok := p.(string); ok {
				res[i] = c.sanitizePath(pStr, projectRoot)
			} else {
				res[i] = p
			}
		}
		return res
	}
	return paths
}

func (c *ContentSanitizerInterceptor) sanitizeError(err error, projectRoot string) error {
	if err == nil {
		return nil
	}
	errMsg := err.Error()
	errMsg = strings.ReplaceAll(errMsg, "file://"+projectRoot, "file://")
	errMsg = strings.ReplaceAll(errMsg, projectRoot+string(filepath.Separator), "")
	errMsg = strings.ReplaceAll(errMsg, projectRoot, "")
	return fmt.Errorf("%s", errMsg)
}

func (c *ContentSanitizerInterceptor) sanitizeContent(content string, userId, projectId string) string {
	pattern := fmt.Sprintf(`(?:(^|[\"\'`+"`"+`\(\[\{\<\n\r ])( ))?(file://)?(/[^ \)^\"\'`+"`"+`<>\[\]\n\t\r]*)%s/%s/(([^ \)^\"\'`+"`"+`<>\[\]\n\t\r]*))`, regexp.QuoteMeta(userId), regexp.QuoteMeta(projectId))
	re, err := regexp.Compile(pattern)
	if err != nil {
		slog.Error("Failed to compile regex pattern in sanitizeContent", "error", err)
		return content
	}

	return re.ReplaceAllStringFunc(content, func(match string) string {
		submatches := re.FindStringSubmatch(match)
		if len(submatches) >= 6 {
			preceding := submatches[1]
			scheme := submatches[3]
			remainder := submatches[5]
			if scheme != "" {
				return preceding + "file:///" + remainder
			}
			return preceding + remainder
		}
		return match
	})
}
