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

package core

import (
	"testing"

	aitesting "google.com/apf/ai/testing"
)

func TestNewRepoConfigFromState(t *testing.T) {
	cfg := &Configuration{
		BmadCfg: BmadConfiguration{
			BmadPlanningArtifactsPath:       "default-plan-path",
			BmadImplementationArtifactsPath: "default-impl-path",
			BmadOutputPath:                  "default-spec-path",
		},
	}

	t.Run("ProjectName set in state", func(t *testing.T) {
		mockState := &aitesting.MockState{
			Data: map[string]interface{}{
				"projectId":   "proj-id-123",
				"projectName": "My Awesome Project",
				"repoUrl":     "https://github.com/my/repo",
			},
		}
		ctx := &aitesting.MockCallbackContext{
			UserId:    "test-user",
			SessionId: "session-abc",
			StateObj:  mockState,
		}

		repoCfg, err := NewRepoConfigFromState(ctx, cfg)
		if err != nil {
			t.Fatalf("NewRepoConfigFromState failed: %v", err)
		}

		if repoCfg.ProjectId != "proj-id-123" {
			t.Errorf("expected ProjectId 'proj-id-123', got %q", repoCfg.ProjectId)
		}
		if repoCfg.ProjectName != "My Awesome Project" {
			t.Errorf("expected ProjectName 'My Awesome Project', got %q", repoCfg.ProjectName)
		}
	})

	t.Run("ProjectName absent, defaults to ProjectId", func(t *testing.T) {
		mockState := &aitesting.MockState{
			Data: map[string]interface{}{
				"projectId": "proj-id-456",
				"repoUrl":   "https://github.com/my/repo",
			},
		}
		ctx := &aitesting.MockCallbackContext{
			UserId:    "test-user",
			SessionId: "session-abc",
			StateObj:  mockState,
		}

		repoCfg, err := NewRepoConfigFromState(ctx, cfg)
		if err != nil {
			t.Fatalf("NewRepoConfigFromState failed: %v", err)
		}

		if repoCfg.ProjectId != "proj-id-456" {
			t.Errorf("expected ProjectId 'proj-id-456', got %q", repoCfg.ProjectId)
		}
		if repoCfg.ProjectName != "proj-id-456" {
			t.Errorf("expected ProjectName to default to 'proj-id-456', got %q", repoCfg.ProjectName)
		}
	})
}
