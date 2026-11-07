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
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"google.com/apf/ai/core"
	aitesting "google.com/apf/ai/testing"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/session"
	"google.golang.org/adk/tool"
	"google.golang.org/genai"
)

func TestExecuteWriteIntercept(t *testing.T) {
	cfg := &core.Configuration{
		RepoCfg: core.RepoConfiguration{
			ProjectId: "myproject",
		},
	}
	plugin := &BmadInterceptorPlugin{cfg: cfg}

	mockState := &aitesting.MockState{Data: make(map[string]interface{})}
	mockCtx := &aitesting.MockCallbackContext{
		UserId:    "myuser",
		SessionId: "mysession",
		StateObj:  mockState,
	}

	result := map[string]any{
		"filePath": "some/path/myuser/myproject/file.txt",
	}

	got, err := plugin.executeWriteIntercept(mockCtx, &cfg.RepoCfg, result)
	if err != nil {
		t.Fatalf("executeWriteIntercept failed: %v", err)
	}

	expectedPath := "file.txt"
	if got["filePath"] != expectedPath {
		t.Errorf("executeWriteIntercept got filePath %q, want %q", got["filePath"], expectedPath)
	}

	var notification string
	for k, v := range mockState.Data {
		if strings.HasPrefix(k, "file-write-notification:") {
			if str, ok := v.(string); ok {
				notification = str
				break
			}
		}
	}
	if notification == "" {
		t.Fatalf("failed to get unique file-write-notification key from mock state")
	}
	if notification != expectedPath {
		t.Errorf("notification got %q, want %q", notification, expectedPath)
	}
}

type mockTool struct {
	name string
}

func (m *mockTool) Name() string        { return m.name }
func (m *mockTool) Description() string { return "" }
func (m *mockTool) IsLongRunning() bool { return false }

var _ tool.Tool = (*mockTool)(nil)

func TestBeforeToolCallback_FileWriterSandboxPathCorrection(t *testing.T) {
	cfg := &core.Configuration{
		WorkspaceMountPath: "/mount",
		BmadCfg: core.BmadConfiguration{
			BmadOutputPath: "specs",
		},
	}
	plugin := &BmadInterceptorPlugin{cfg: cfg}

	mockState := &aitesting.MockState{Data: make(map[string]interface{})}
	mockState.Data["projectId"] = "myproject"
	mockState.Data["repoUrl"] = "https://github.com/example/repo.git"
	mockState.Data["specLocation"] = "specs"

	mockCtx := &aitesting.MockCallbackContext{
		UserId:    "myuser",
		SessionId: "mysession",
		StateObj:  mockState,
	}

	tWriter := &mockTool{name: "file_writer"}

	testCases := []struct {
		name         string
		inputPath    string
		expectedPath string
	}{
		{
			name:         "Outside mount path with standard spec pattern",
			inputPath:    "/tmp/outside/default-organization/myuser/myproject/specs/subfolder/file.txt",
			expectedPath: "/mount/default-organization/myuser/myproject/specs/subfolder/file.txt",
		},
		{
			name:         "Clean relative path anchoring",
			inputPath:    "specs/subfolder/file.txt",
			expectedPath: "/mount/default-organization/myuser/myproject/specs/subfolder/file.txt",
		},
		{
			name:         "Already inside mount path",
			inputPath:    "/mount/default-organization/myuser/myproject/specs/subfolder/file.txt",
			expectedPath: "/mount/default-organization/myuser/myproject/specs/subfolder/file.txt",
		},
		{
			name:         "Does not contain spec pattern - forced to specs root",
			inputPath:    "/var/outside/myuser/myproject/other/file.txt",
			expectedPath: "/mount/default-organization/myuser/myproject/specs/file.txt",
		},
		{
			name:         "Relative path at root - forced to specs folder",
			inputPath:    "PRD_Template.md",
			expectedPath: "/mount/default-organization/myuser/myproject/specs/PRD_Template.md",
		},
		{
			name:         "Relative path in other subfolder - forced under specs",
			inputPath:    "other/somefile.txt",
			expectedPath: "/mount/default-organization/myuser/myproject/specs/other/somefile.txt",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			args := map[string]any{
				"filePath": tc.inputPath,
			}
			toolCtx := &aitesting.MockToolContext{MockCallbackContext: mockCtx}
			res, err := plugin.BeforeToolCallback(toolCtx, tWriter, args)
			if err != nil {
				t.Fatalf("BeforeToolCallback failed: %v", err)
			}
			if res != nil {
				t.Errorf("Expected result to be nil, got %v", res)
			}
			if args["filePath"] != tc.expectedPath {
				t.Errorf("BeforeToolCallback filePath got %q, want %q", args["filePath"], tc.expectedPath)
			}
		})
	}
}

func TestBeforeToolCallback_FileWriterZeroTrustGuardrail(t *testing.T) {
	cfg := &core.Configuration{
		WorkspaceMountPath: "/mount",
	}
	plugin := &BmadInterceptorPlugin{cfg: cfg}

	mockState := &aitesting.MockState{Data: make(map[string]interface{})}
	mockState.Data["projectId"] = "myproject"
	mockState.Data["repoUrl"] = "https://github.com/example/repo.git"
	mockState.Data["specLocation"] = "docs/specs"

	mockCtx := &aitesting.MockCallbackContext{
		UserId:    "myuser",
		SessionId: "mysession",
		StateObj:  mockState,
	}

	tWriter := &mockTool{name: "file_writer"}

	testCases := []struct {
		name         string
		inputPath    string
		expectedPath string
	}{
		{
			name:         "Relative path at root - forced to docs/specs",
			inputPath:    "PRD_Template.md",
			expectedPath: "/mount/default-organization/myuser/myproject/docs/specs/PRD_Template.md",
		},
		{
			name:         "Relative path starting with overlapping docs - forced to docs/specs/plans/prd.md",
			inputPath:    "docs/plans/prd.md",
			expectedPath: "/mount/default-organization/myuser/myproject/docs/specs/plans/prd.md",
		},
		{
			name:         "Relative path completely outside - forced to docs/specs/other/file.txt",
			inputPath:    "other/file.txt",
			expectedPath: "/mount/default-organization/myuser/myproject/docs/specs/other/file.txt",
		},
		{
			name:         "Absolute path outside - fallback to base name under docs/specs",
			inputPath:    "/var/other/file.txt",
			expectedPath: "/mount/default-organization/myuser/myproject/docs/specs/file.txt",
		},
		{
			name:         "Already inside docs/specs - remains unchanged",
			inputPath:    "docs/specs/plans/prds/prd.md",
			expectedPath: "/mount/default-organization/myuser/myproject/docs/specs/plans/prds/prd.md",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			args := map[string]any{
				"filePath": tc.inputPath,
			}
			toolCtx := &aitesting.MockToolContext{MockCallbackContext: mockCtx}
			_, err := plugin.BeforeToolCallback(toolCtx, tWriter, args)
			if err != nil {
				t.Fatalf("BeforeToolCallback failed: %v", err)
			}
			if args["filePath"] != tc.expectedPath {
				t.Errorf("BeforeToolCallback filePath got %q, want %q", args["filePath"], tc.expectedPath)
			}
		})
	}
}

func TestBeforeToolCallback_FileReaderZeroTrustGuardrail(t *testing.T) {
	cfg := &core.Configuration{
		WorkspaceMountPath: "/mount",
		BmadCfg: core.BmadConfiguration{
			BmadRootPath: "workspaces/config/bmad",
		},
	}
	plugin := &BmadInterceptorPlugin{cfg: cfg}

	mockState := &aitesting.MockState{Data: make(map[string]interface{})}
	mockState.Data["projectId"] = "myproject"
	mockState.Data["repoUrl"] = "https://github.com/example/repo.git"
	mockState.Data["specLocation"] = "docs/specs"

	mockCtx := &aitesting.MockCallbackContext{
		UserId:    "myuser",
		SessionId: "mysession",
		StateObj:  mockState,
	}

	tReader := &mockTool{name: "file_reader"}

	testCases := []struct {
		name          string
		inputPath     string
		expectedPath  string
		expectedError bool
	}{
		{
			name:         "Relative path for skill inside workspace - anchors to user unique workspace path",
			inputPath:    ".agents/skills/bmad-prd/SKILL.md",
			expectedPath: "/mount/default-organization/myuser/myproject/.agents/skills/bmad-prd/SKILL.md",
		},
		{
			name:         "Relative path inside workspace - anchors to user unique workspace path",
			inputPath:    "docs/specs/plans/prd.md",
			expectedPath: "/mount/default-organization/myuser/myproject/docs/specs/plans/prd.md",
		},
		{
			name:         "Already absolute path inside workspace - remains unchanged",
			inputPath:    "/mount/default-organization/myuser/myproject/docs/specs/plans/prd.md",
			expectedPath: "/mount/default-organization/myuser/myproject/docs/specs/plans/prd.md",
		},
		{
			name:          "Path outside workspace - raises perimeter breach error",
			inputPath:     "/tmp/sensitive_file.txt",
			expectedError: true,
		},
		{
			name:          "Path to external system directory - raises perimeter breach error",
			inputPath:     "/config/bmad/skills/bmad-prd/SKILL.md",
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			args := map[string]any{
				"filePath": tc.inputPath,
			}
			toolCtx := &aitesting.MockToolContext{MockCallbackContext: mockCtx}
			_, err := plugin.BeforeToolCallback(toolCtx, tReader, args)

			if tc.expectedError {
				if err == nil {
					t.Fatalf("Expected safety perimeter error, but got nil")
				}
				if !strings.Contains(err.Error(), "zero-trust perimeter breach") {
					t.Errorf("Expected error to mention 'zero-trust perimeter breach', got %q", err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("BeforeToolCallback failed: %v", err)
				}
				if args["filePath"] != tc.expectedPath {
					t.Errorf("BeforeToolCallback filePath got %q, want %q", args["filePath"], tc.expectedPath)
				}
			}
		})
	}
}

func TestBeforeToolCallback_FindFilesZeroTrustGuardrail(t *testing.T) {
	cfg := &core.Configuration{
		WorkspaceMountPath: "/mount",
		BmadCfg: core.BmadConfiguration{
			BmadRootPath: "workspaces/config/bmad",
		},
	}
	plugin := &BmadInterceptorPlugin{cfg: cfg}

	mockState := &aitesting.MockState{Data: make(map[string]interface{})}
	mockState.Data["projectId"] = "myproject"
	mockState.Data["repoUrl"] = "https://github.com/example/repo.git"
	mockState.Data["specLocation"] = "docs/specs"

	mockCtx := &aitesting.MockCallbackContext{
		UserId:    "myuser",
		SessionId: "mysession",
		StateObj:  mockState,
	}

	tFind := &mockTool{name: "find_files"}

	testCases := []struct {
		name          string
		inputPath     string
		expectedPath  string
		expectedError bool
	}{
		{
			name:         "Empty path - defaults to project workspace root",
			inputPath:    "",
			expectedPath: "/mount/default-organization/myuser/myproject",
		},
		{
			name:         "Relative path inside workspace - anchors to user unique workspace path",
			inputPath:    "docs/specs",
			expectedPath: "/mount/default-organization/myuser/myproject/docs/specs",
		},
		{
			name:         "Already absolute path inside workspace - remains unchanged",
			inputPath:    "/mount/default-organization/myuser/myproject/docs/specs",
			expectedPath: "/mount/default-organization/myuser/myproject/docs/specs",
		},
		{
			name:         "Project root placeholder {project-root} alone",
			inputPath:    "{project-root}",
			expectedPath: "/mount/default-organization/myuser/myproject",
		},
		{
			name:         "Project root placeholder with subpath {project-root}/docs/specs",
			inputPath:    "{project-root}/docs/specs",
			expectedPath: "/mount/default-organization/myuser/myproject/docs/specs",
		},
		{
			name:         "Skill root placeholder with subpath {skill-root}/bmad-brainstorming",
			inputPath:    "{skill-root}/bmad-brainstorming",
			expectedPath: "/mount/default-organization/myuser/myproject/.agents/skills/bmad-brainstorming",
		},
		{
			name:         "file: URI prefix with {project-root} placeholder",
			inputPath:    "file:{project-root}/docs/specs",
			expectedPath: "/mount/default-organization/myuser/myproject/docs/specs",
		},
		{
			name:          "Path outside workspace - raises perimeter breach error",
			inputPath:     "/etc",
			expectedError: true,
		},
		{
			name:          "Path to global bmad root - raises perimeter breach error",
			inputPath:     "/config/bmad/skills/bmad-brainstorming",
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			args := map[string]any{}
			if tc.inputPath != "" {
				args["directoryPath"] = tc.inputPath
			}
			toolCtx := &aitesting.MockToolContext{MockCallbackContext: mockCtx}
			_, err := plugin.BeforeToolCallback(toolCtx, tFind, args)

			if tc.expectedError {
				if err == nil {
					t.Fatalf("Expected safety perimeter error, but got nil")
				}
				if !strings.Contains(err.Error(), "zero-trust perimeter breach") {
					t.Errorf("Expected error to mention 'zero-trust perimeter breach', got %q", err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("BeforeToolCallback failed: %v", err)
				}
				if args["directoryPath"] != tc.expectedPath {
					t.Errorf("BeforeToolCallback directoryPath got %q, want %q", args["directoryPath"], tc.expectedPath)
				}
			}
		})
	}
}

func TestBeforeToolCallback_FileWriterContentSanitized(t *testing.T) {
	cfg := &core.Configuration{
		WorkspaceMountPath: "/mount",
		BmadCfg: core.BmadConfiguration{
			BmadOutputPath: "specs",
		},
	}
	plugin := &BmadInterceptorPlugin{cfg: cfg}

	mockState := &aitesting.MockState{Data: make(map[string]interface{})}
	mockState.Data["projectId"] = "myproject"
	mockState.Data["repoUrl"] = "https://github.com/example/repo.git"
	mockState.Data["specLocation"] = "specs"

	mockCtx := &aitesting.MockCallbackContext{
		UserId:    "myuser",
		SessionId: "mysession",
		StateObj:  mockState,
	}

	tWriter := &mockTool{name: "file_writer"}

	testCases := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "Standard absolute path in content",
			content:  "Ref: /usr/local/google/home/myuser/workspace/myproject/myuser/myproject/specs/todo.md is here",
			expected: "Ref: specs/todo.md is here",
		},
		{
			name:     "Markdown link with file scheme in content",
			content:  "Please check [todo](file:///usr/local/google/home/myuser/workspace/myproject/myuser/myproject/specs/todo.md)",
			expected: "Please check [todo](file:///specs/todo.md)",
		},
		{
			name:     "Relative path should not be modified",
			content:  "Check myuser/myproject/specs/todo.md",
			expected: "Check myuser/myproject/specs/todo.md",
		},
		{
			name:     "Multiple absolute paths in content",
			content:  "First /mount/myuser/myproject/specs/a.md and Second /mount/myuser/myproject/specs/b.md",
			expected: "First specs/a.md and Second specs/b.md",
		},
		{
			name:     "No match in content",
			content:  "No absolute spec path: /usr/local/google/home/myuser/workspace/other/file.txt",
			expected: "No absolute spec path: /usr/local/google/home/myuser/workspace/other/file.txt",
		},
		{
			name:     "Absolute path with leading space inside quote",
			content:  `Ref: " /mount/myuser/myproject/specs/todo.md"`,
			expected: `Ref: "specs/todo.md"`,
		},
		{
			name:     "Absolute path with leading space at start of content",
			content:  ` /mount/myuser/myproject/specs/todo.md is top priority`,
			expected: `specs/todo.md is top priority`,
		},
		{
			name:     "Absolute path with leading space inside bracket",
			content:  `Ref: [todo]( /mount/myuser/myproject/specs/todo.md)`,
			expected: `Ref: [todo](specs/todo.md)`,
		},
		{
			name:     "Absolute path with leading space after newline",
			content:  "Ref:\n /mount/myuser/myproject/specs/todo.md",
			expected: "Ref:\nspecs/todo.md",
		},
		{
			name:     "Absolute path with double space after colon",
			content:  `Ref:  /mount/myuser/myproject/specs/todo.md`,
			expected: `Ref: specs/todo.md`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			args := map[string]any{
				"filePath": "/mount/myuser/myproject/specs/test.md",
				"content":  tc.content,
			}
			toolCtx := &aitesting.MockToolContext{MockCallbackContext: mockCtx}
			res, err := plugin.BeforeToolCallback(toolCtx, tWriter, args)
			if err != nil {
				t.Fatalf("BeforeToolCallback failed: %v", err)
			}
			if res != nil {
				t.Errorf("Expected result to be nil, got %v", res)
			}
			gotContent, ok := args["content"].(string)
			if !ok {
				t.Fatalf("Expected content to be string, got %T", args["content"])
			}
			if gotContent != tc.expected {
				t.Errorf("BeforeToolCallback content got %q, want %q", gotContent, tc.expected)
			}
		})
	}
}

type MockSession struct {
	session.Session
	UserIdVal    string
	SessionIdVal string
	StateObj     *aitesting.MockState
}

func (m *MockSession) ID() string {
	return m.SessionIdVal
}

func (m *MockSession) AppName() string {
	return "test-app"
}

func (m *MockSession) UserID() string {
	return m.UserIdVal
}

func (m *MockSession) State() session.State {
	return m.StateObj
}

type MockInvocationContext struct {
	context.Context
	InvocationIDVal string
	SessionVal      session.Session
}

func (m *MockInvocationContext) InvocationID() string {
	return m.InvocationIDVal
}

func (m *MockInvocationContext) Session() session.Session {
	return m.SessionVal
}

func (m *MockInvocationContext) Agent() agent.Agent          { return nil }
func (m *MockInvocationContext) Artifacts() agent.Artifacts  { return nil }
func (m *MockInvocationContext) Memory() agent.Memory        { return nil }
func (m *MockInvocationContext) Branch() string              { return "main" }
func (m *MockInvocationContext) UserContent() *genai.Content { return nil }
func (m *MockInvocationContext) RunConfig() *agent.RunConfig { return nil }
func (m *MockInvocationContext) EndInvocation()              {}
func (m *MockInvocationContext) Ended() bool                 { return false }
func (m *MockInvocationContext) WithContext(ctx context.Context) agent.InvocationContext {
	return m
}



func TestBeforeToolCallback_DirectoryReaderZeroTrustGuardrail(t *testing.T) {
	cfg := &core.Configuration{
		WorkspaceMountPath: "/mount",
		BmadCfg: core.BmadConfiguration{
			BmadRootPath: "workspaces/config/bmad",
		},
	}
	plugin := &BmadInterceptorPlugin{cfg: cfg}

	mockState := &aitesting.MockState{Data: make(map[string]interface{})}
	mockState.Data["projectId"] = "myproject"
	mockState.Data["repoUrl"] = "https://github.com/example/repo.git"

	mockCtx := &aitesting.MockCallbackContext{
		UserId:    "myuser",
		SessionId: "mysession",
		StateObj:  mockState,
	}

	tReader := &mockTool{name: "directory_reader"}

	testCases := []struct {
		name          string
		inputPath     string
		expectedPath  string
		expectedError bool
	}{
		{
			name:         "Relative path inside workspace - anchors to user unique workspace path",
			inputPath:    "docs/specs",
			expectedPath: "/mount/default-organization/myuser/myproject/docs/specs",
		},
		{
			name:         "Relative path for skill inside workspace - anchors to user unique workspace path",
			inputPath:    ".agents/skills/bmad-prd",
			expectedPath: "/mount/default-organization/myuser/myproject/.agents/skills/bmad-prd",
		},
		{
			name:         "Already absolute path inside workspace - remains unchanged",
			inputPath:    "/mount/default-organization/myuser/myproject/docs/specs",
			expectedPath: "/mount/default-organization/myuser/myproject/docs/specs",
		},
		{
			name:          "Path outside workspace - raises perimeter breach error",
			inputPath:     "/etc",
			expectedError: true,
		},
		{
			name:          "Path to global bmad root - raises perimeter breach error",
			inputPath:     "/config/bmad/skills/bmad-prd",
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			args := map[string]any{
				"directoryPathOrGlob": tc.inputPath,
			}
			toolCtx := &aitesting.MockToolContext{MockCallbackContext: mockCtx}
			_, err := plugin.BeforeToolCallback(toolCtx, tReader, args)

			if tc.expectedError {
				if err == nil {
					t.Fatalf("Expected safety perimeter error, but got nil")
				}
				if !strings.Contains(err.Error(), "zero-trust perimeter breach") {
					t.Errorf("Expected error to mention 'zero-trust perimeter breach', got %q", err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("BeforeToolCallback failed: %v", err)
				}
				if args["directoryPathOrGlob"] != tc.expectedPath {
					t.Errorf("BeforeToolCallback directoryPathOrGlob got %q, want %q", args["directoryPathOrGlob"], tc.expectedPath)
				}
			}
		})
	}
}

func TestAfterToolCallback_ContentSanitizedAndOutputsSanitized(t *testing.T) {
	cfg := &core.Configuration{
		WorkspaceMountPath: "/mount",
		BmadCfg: core.BmadConfiguration{
			BmadRootPath: "/bmad",
		},
	}
	plugin := &BmadInterceptorPlugin{cfg: cfg}

	mockState := &aitesting.MockState{Data: make(map[string]interface{})}
	mockState.Data["projectId"] = "myproject"
	mockState.Data["repoUrl"] = "https://github.com/example/repo.git"
	mockState.Data["specLocation"] = "specs"

	mockCtx := &aitesting.MockCallbackContext{
		UserId:    "myuser",
		SessionId: "mysession",
		StateObj:  mockState,
	}
	toolCtx := &aitesting.MockToolContext{MockCallbackContext: mockCtx}

	tReader := &mockTool{name: "file_reader"}
	tDirReader := &mockTool{name: "directory_reader"}

	t.Run("file_reader output sanitization", func(t *testing.T) {
		args := map[string]any{
			"filePath": "/mount/default-organization/myuser/myproject/src/main.go",
		}
		result := map[string]any{
			"filePath": "/mount/default-organization/myuser/myproject/src/main.go",
			"content":  "Ref: /mount/default-organization/myuser/myproject/src/helper.go and file:///mount/default-organization/myuser/myproject/specs/todo.md",
		}

		got, err := plugin.AfterToolCallback(toolCtx, tReader, args, result, nil)
		if err != nil {
			t.Fatalf("AfterToolCallback failed: %v", err)
		}

		expectedFilePath := "src/main.go"
		expectedContent := "Ref: src/helper.go and file:///specs/todo.md"

		if got["filePath"] != expectedFilePath {
			t.Errorf("filePath got %q, want %q", got["filePath"], expectedFilePath)
		}
		if got["content"] != expectedContent {
			t.Errorf("content got %q, want %q", got["content"], expectedContent)
		}
	})

	t.Run("directory_reader output sanitization", func(t *testing.T) {
		args := map[string]any{
			"directoryPathOrGlob": "/mount/default-organization/myuser/myproject/src",
		}
		result := map[string]any{
			"filesPaths": []string{
				"/mount/default-organization/myuser/myproject/src/main.go",
				"/mount/default-organization/myuser/myproject/src/helper.go",
			},
		}

		got, err := plugin.AfterToolCallback(toolCtx, tDirReader, args, result, nil)
		if err != nil {
			t.Fatalf("AfterToolCallback failed: %v", err)
		}

		expectedPaths := []string{
			"src/main.go",
			"src/helper.go",
		}

		gotPaths, ok := got["filesPaths"].([]string)
		if !ok {
			// Try []any
			gotAnyPaths, ok := got["filesPaths"].([]any)
			if !ok {
				t.Fatalf("filesPaths is not []string or []any, got %T", got["filesPaths"])
			}
			gotPaths = make([]string, len(gotAnyPaths))
			for i, v := range gotAnyPaths {
				gotPaths[i] = v.(string)
			}
		}

		if len(gotPaths) != len(expectedPaths) {
			t.Fatalf("filesPaths len got %d, want %d", len(gotPaths), len(expectedPaths))
		}
		for i, p := range gotPaths {
			if p != expectedPaths[i] {
				t.Errorf("filesPaths[%d] got %q, want %q", i, p, expectedPaths[i])
			}
		}
	})

	t.Run("tool error sanitization", func(t *testing.T) {
		args := map[string]any{
			"filePath": "/mount/default-organization/myuser/myproject/src/main.go",
		}
		origErr := fmt.Errorf("failed to open /mount/default-organization/myuser/myproject/src/main.go: file:///mount/default-organization/myuser/myproject/src/main.go not found")

		_, err := plugin.AfterToolCallback(toolCtx, tReader, args, nil, origErr)
		if err == nil {
			t.Fatalf("Expected error, got nil")
		}

		expectedErrStr := "failed to open src/main.go: file:///src/main.go not found"
		if err.Error() != expectedErrStr {
			t.Errorf("error got %q, want %q", err.Error(), expectedErrStr)
		}
	})
}

func TestPathSafetyInterceptor_RootInstructionFile(t *testing.T) {
	cfg := &core.Configuration{
		WorkspaceMountPath: "/mount",
		BmadCfg: core.BmadConfiguration{
			BmadRootPath:   "/bmad",
			BmadOutputPath: "docs/specs",
		},
	}
	plugin := &BmadInterceptorPlugin{cfg: cfg}

	mockState := &aitesting.MockState{Data: make(map[string]interface{})}
	mockState.Data["projectId"] = "myproject"
	mockCtx := &aitesting.MockCallbackContext{
		UserId:    "myuser",
		SessionId: "mysession",
		StateObj:  mockState,
	}
	tWriter := &mockTool{name: "file_writer"}

	args := map[string]any{
		"filePath": "AGENTS.md",
	}
	toolCtx := &aitesting.MockToolContext{MockCallbackContext: mockCtx}

	_, err := plugin.BeforeToolCallback(toolCtx, tWriter, args)
	if err != nil {
		t.Fatalf("BeforeToolCallback failed: %v", err)
	}

	gotPath := args["filePath"].(string)
	expectedPath := "/mount/default-organization/myuser/myproject/AGENTS.md"
	if gotPath != expectedPath {
		t.Errorf("BeforeToolCallback redirected AGENTS.md to %q, want %q", gotPath, expectedPath)
	}
}

func TestPathSafetyInterceptor_TmpDirectory(t *testing.T) {
	cfg := &core.Configuration{
		WorkspaceMountPath: "/mount",
		BmadCfg: core.BmadConfiguration{
			BmadRootPath:   "/bmad",
			BmadOutputPath: "docs/specs",
		},
	}
	plugin := &BmadInterceptorPlugin{cfg: cfg}

	mockState := &aitesting.MockState{Data: make(map[string]interface{})}
	mockState.Data["projectId"] = "myproject"
	mockCtx := &aitesting.MockCallbackContext{
		UserId:    "myuser",
		SessionId: "mysession",
		StateObj:  mockState,
	}
	tWriter := &mockTool{name: "file_writer"}
	toolCtx := &aitesting.MockToolContext{MockCallbackContext: mockCtx}

	t.Run("allows relative path under .tmp", func(t *testing.T) {
		args := map[string]any{
			"filePath": ".tmp/populate_brainstorm.py",
		}
		_, err := plugin.BeforeToolCallback(toolCtx, tWriter, args)
		if err != nil {
			t.Fatalf("BeforeToolCallback failed: %v", err)
		}
		gotPath := args["filePath"].(string)
		expectedPath := "/mount/default-organization/myuser/myproject/.tmp/populate_brainstorm.py"
		if gotPath != expectedPath {
			t.Errorf("BeforeToolCallback got %q, want %q", gotPath, expectedPath)
		}
	})

	t.Run("redirects /tmp path outside workspace to .tmp folder in project root", func(t *testing.T) {
		args := map[string]any{
			"filePath": "/tmp/populate_brainstorm.py",
		}
		_, err := plugin.BeforeToolCallback(toolCtx, tWriter, args)
		if err != nil {
			t.Fatalf("BeforeToolCallback failed: %v", err)
		}
		gotPath := args["filePath"].(string)
		expectedPath := "/mount/default-organization/myuser/myproject/.tmp/populate_brainstorm.py"
		if gotPath != expectedPath {
			t.Errorf("BeforeToolCallback got %q, want %q", gotPath, expectedPath)
		}
	})

	t.Run("redirects non-tmp out of sandbox path to specs directory", func(t *testing.T) {
		args := map[string]any{
			"filePath": "/var/log/my_spec.md",
		}
		_, err := plugin.BeforeToolCallback(toolCtx, tWriter, args)
		if err != nil {
			t.Fatalf("BeforeToolCallback failed: %v", err)
		}
		gotPath := args["filePath"].(string)
		expectedPath := "/mount/default-organization/myuser/myproject/docs/specs/my_spec.md"
		if gotPath != expectedPath {
			t.Errorf("BeforeToolCallback got %q, want %q", gotPath, expectedPath)
		}
	})
}

func TestPathSafetyInterceptor_BmadScripts_LocalResolution(t *testing.T) {
	tempDir := t.TempDir()
	mountDir := filepath.Join(tempDir, "mount")
	projDir := filepath.Join(mountDir, "default-organization", "myuser", "myproject")

	cfg := &core.Configuration{
		WorkspaceMountPath: mountDir,
		BmadCfg: core.BmadConfiguration{
			BmadOutputPath: "docs/specs",
		},
	}
	plugin := &BmadInterceptorPlugin{cfg: cfg}

	mockState := &aitesting.MockState{Data: make(map[string]interface{})}
	mockState.Data["projectId"] = "myproject"
	mockCtx := &aitesting.MockCallbackContext{
		UserId:    "myuser",
		SessionId: "mysession",
		StateObj:  mockState,
	}
	tReader := &mockTool{name: "file_reader"}
	toolCtx := &aitesting.MockToolContext{MockCallbackContext: mockCtx}

	t.Run("resolves _bmad/scripts/memlog.py when in project .agents/_bmad", func(t *testing.T) {
		projAgentsScriptDir := filepath.Join(projDir, ".agents", "_bmad", "scripts")
		if err := os.MkdirAll(projAgentsScriptDir, 0755); err != nil {
			t.Fatalf("Failed to create proj .agents script dir: %v", err)
		}
		memlogAgentsProj := filepath.Join(projAgentsScriptDir, "memlog.py")
		if err := os.WriteFile(memlogAgentsProj, []byte("#!/usr/bin/env python3\nprint('custom agents memlog')"), 0755); err != nil {
			t.Fatalf("Failed to write proj agents memlog: %v", err)
		}
		defer os.RemoveAll(filepath.Join(projDir, ".agents"))

		args := map[string]any{
			"filePath": "_bmad/scripts/memlog.py",
		}
		_, err := plugin.BeforeToolCallback(toolCtx, tReader, args)
		if err != nil {
			t.Fatalf("BeforeToolCallback failed: %v", err)
		}
		gotPath := args["filePath"].(string)
		if gotPath != memlogAgentsProj {
			t.Errorf("BeforeToolCallback got %q, want %q", gotPath, memlogAgentsProj)
		}
	})

	t.Run("resolves _bmad/scripts/memlog.py when in project _bmad", func(t *testing.T) {
		projScriptDir := filepath.Join(projDir, "_bmad", "scripts")
		if err := os.MkdirAll(projScriptDir, 0755); err != nil {
			t.Fatalf("Failed to create proj script dir: %v", err)
		}
		memlogProj := filepath.Join(projScriptDir, "memlog.py")
		if err := os.WriteFile(memlogProj, []byte("#!/usr/bin/env python3\nprint('custom memlog')"), 0755); err != nil {
			t.Fatalf("Failed to write proj memlog: %v", err)
		}
		defer os.RemoveAll(filepath.Join(projDir, "_bmad"))

		args := map[string]any{
			"filePath": "_bmad/scripts/memlog.py",
		}
		_, err := plugin.BeforeToolCallback(toolCtx, tReader, args)
		if err != nil {
			t.Fatalf("BeforeToolCallback failed: %v", err)
		}
		gotPath := args["filePath"].(string)
		if gotPath != memlogProj {
			t.Errorf("BeforeToolCallback got %q, want %q", gotPath, memlogProj)
		}
	})

	t.Run("rejects reading from external /config/bmad directory as perimeter breach", func(t *testing.T) {
		args := map[string]any{
			"filePath": "/config/bmad/_bmad/scripts/memlog.py",
		}
		_, err := plugin.BeforeToolCallback(toolCtx, tReader, args)
		if err == nil {
			t.Errorf("Expected perimeter breach error reading /config/bmad, got nil")
		}
	})
}

func TestPathSafetyInterceptor_FileWriter_ScriptsRedirectToTmp(t *testing.T) {
	cfg := &core.Configuration{
		WorkspaceMountPath: "/mount",
		BmadCfg: core.BmadConfiguration{
			BmadRootPath:   "/bmad",
			BmadOutputPath: "docs/specs",
		},
	}
	plugin := &BmadInterceptorPlugin{cfg: cfg}

	mockState := &aitesting.MockState{Data: make(map[string]interface{})}
	mockState.Data["projectId"] = "myproject"
	mockCtx := &aitesting.MockCallbackContext{
		UserId:    "myuser",
		SessionId: "mysession",
		StateObj:  mockState,
	}
	tWriter := &mockTool{name: "file_writer"}
	toolCtx := &aitesting.MockToolContext{MockCallbackContext: mockCtx}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "redirects _bmad/scripts/memlog.py to .tmp/memlog.py",
			input:    "_bmad/scripts/memlog.py",
			expected: "/mount/default-organization/myuser/myproject/.tmp/memlog.py",
		},
		{
			name:     "redirects custom_script.py to .tmp/custom_script.py",
			input:    "custom_script.py",
			expected: "/mount/default-organization/myuser/myproject/.tmp/custom_script.py",
		},
		{
			name:     "redirects scripts/tool.sh to .tmp/tool.sh",
			input:    "scripts/tool.sh",
			expected: "/mount/default-organization/myuser/myproject/.tmp/tool.sh",
		},
		{
			name:     "forces spec markdown file to docs/specs",
			input:    "my_spec.md",
			expected: "/mount/default-organization/myuser/myproject/docs/specs/my_spec.md",
		},
		{
			name:     "keeps spec file inside docs/specs",
			input:    "docs/specs/prd.md",
			expected: "/mount/default-organization/myuser/myproject/docs/specs/prd.md",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			args := map[string]any{
				"filePath": tc.input,
			}
			_, err := plugin.BeforeToolCallback(toolCtx, tWriter, args)
			if err != nil {
				t.Fatalf("BeforeToolCallback failed: %v", err)
			}
			gotPath := args["filePath"].(string)
			if gotPath != tc.expected {
				t.Errorf("BeforeToolCallback got %q, want %q", gotPath, tc.expected)
			}
		})
	}
}


