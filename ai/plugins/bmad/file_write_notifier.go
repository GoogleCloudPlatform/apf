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
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	"google.com/apf/ai/core"
	"google.golang.org/adk/tool"
)

type FileWriteNotifierInterceptor struct {
	BaseInterceptor
	cfg *core.Configuration
}

func (f *FileWriteNotifierInterceptor) Name() string { return "FileWriteNotifier" }

func (f *FileWriteNotifierInterceptor) After(ctx tool.Context, tool tool.Tool, args, result map[string]any, err error) (map[string]any, error) {
	if tool.Name() != "file_writer" {
		return result, err
	}
	if err != nil {
		return result, err
	}

	repoCfg, errRepo := core.NewRepoConfigFromState(ctx, f.cfg)
	if errRepo != nil {
		return result, fmt.Errorf("failed to load repo configuration: %w", errRepo)
	}

	return f.executeWriteIntercept(ctx, repoCfg, result)
}

func (f *FileWriteNotifierInterceptor) executeWriteIntercept(ctx interceptorContext, repoCfg *core.RepoConfiguration, result map[string]any) (map[string]any, error) {
	userId := ctx.UserID()
	projectId := repoCfg.ProjectId

	containPattern := userId + "/" + projectId
	regexPattern := `^.*` + regexp.QuoteMeta(userId) + `/` + regexp.QuoteMeta(projectId) + `/?`
	re, err := regexp.Compile(regexPattern)
	if err != nil {
		return result, fmt.Errorf("failed to compile regex %s: %w", regexPattern, err)
	}

	if val, ok := result["filePath"]; ok {
		if pathStr, ok := val.(string); ok {
			if strings.Contains(pathStr, containPattern) {
				result["filePath"] = re.ReplaceAllString(pathStr, "")
				state := ctx.State()
				uniqueKey := fmt.Sprintf("file-write-notification:%d-%d", time.Now().UnixNano(), atomic.AddUint64(&notificationCounter, 1))
				_ = state.Set(uniqueKey, result["filePath"])
			}
		}
	}

	return result, nil
}
