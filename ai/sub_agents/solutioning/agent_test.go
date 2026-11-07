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

package solutioning

import (
	"context"
	"reflect"
	"testing"
	"unsafe"

	"google.com/apf/ai/core"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/tool"
)

func TestSolutioningAgentCreation(t *testing.T) {
	ctx := context.Background()
	cfg := core.NewConfig()

	// Ensure we have a dummy API key if not set, to avoid failure in NewModel
	if cfg.GeminiAPIKey == "" {
		cfg.GeminiAPIKey = "dummy-key"
	}

	loader, err := NewAgentLoader(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to create agent loader: %v", err)
	}

	rootAgent := loader.RootAgent()
	if rootAgent == nil {
		t.Fatal("Root agent is nil")
	}

	if rootAgent.Name() != "APF Solutioning Agent" {
		t.Errorf("Expected agent name 'APF Solutioning Agent', got '%s'", rootAgent.Name())
	}

	rootTools := getToolNames(rootAgent)
	expectedRootTools := []string{"file_reader", "file_writer", "directory_reader", "find_files", "uv_run", "suggest_phase_switch"}
	for _, expected := range expectedRootTools {
		if !contains(rootTools, expected) {
			t.Errorf("Root agent missing expected tool '%s'. Present tools: %v", expected, rootTools)
		}
	}

	subAgents := rootAgent.SubAgents()
	if len(subAgents) == 0 {
		t.Fatal("Expected subagents on root agent, got 0")
	}

	expectedSubTools := []string{"file_reader", "file_writer", "directory_reader", "find_files", "uv_run"}
	for _, subAgent := range subAgents {
		subTools := getToolNames(subAgent)
		for _, expected := range expectedSubTools {
			if !contains(subTools, expected) {
				t.Errorf("Subagent '%s' missing expected tool '%s'. Present tools: %v", subAgent.Name(), expected, subTools)
			}
		}
	}

	t.Logf("Successfully created agent: %s with subagents and verified tools", rootAgent.Name())
}

func getToolNames(ag agent.Agent) []string {
	if ag == nil {
		return nil
	}

	val := reflect.ValueOf(ag)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return nil
	}

	return findToolsInValue(val)
}

func findToolsInValue(val reflect.Value) []string {
	var toolNames []string

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := val.Type().Field(i)

		if fieldType.Name == "Tools" && field.Kind() == reflect.Slice {
			for j := 0; j < field.Len(); j++ {
				elem := field.Index(j)
				if !elem.IsNil() {
					if elem.CanInterface() {
						if t, ok := elem.Interface().(tool.Tool); ok {
							toolNames = append(toolNames, t.Name())
						}
					} else {
						// Read unexported field safely via unsafe
						elemPtr := reflect.NewAt(elem.Type(), unsafe.Pointer(elem.UnsafeAddr())).Elem()
						if t, ok := elemPtr.Interface().(tool.Tool); ok {
							toolNames = append(toolNames, t.Name())
						}
					}
				}
			}
			if len(toolNames) > 0 {
				return toolNames
			}
		}

		if field.Kind() == reflect.Struct {
			subTools := findToolsInValue(field)
			if len(subTools) > 0 {
				return subTools
			}
		}
	}

	return toolNames
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
