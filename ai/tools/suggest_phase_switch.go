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

	"google.golang.org/adk/tool"
)

// SuggestPhaseSwitchInput defines the arguments for the SuggestPhaseSwitchTool.
type SuggestPhaseSwitchInput struct {
	TargetPhase string `json:"targetPhase"` // e.g., "ONBOARDING", "ANALYSIS", "PLANNING", "SOLUTIONING", "IMPLEMENTATION"
	SkillID     string `json:"skillId"`     // e.g., "create-prd", "brainstorming"
	Reason      string `json:"reason"`      // Human-readable explanation for why to switch phases
}

// SuggestPhaseSwitchOutput defines the return values for the SuggestPhaseSwitchTool.
type SuggestPhaseSwitchOutput struct {
	TargetPhase string `json:"targetPhase"`
	SkillID     string `json:"skillId"`
	Reason      string `json:"reason"`
	Status      string `json:"status"`
}

// SuggestPhaseSwitchTool is an ADK tool for suggesting a user switch to another workspace phase tab.
type SuggestPhaseSwitchTool struct{}

// Name returns the name of the tool.
func (t *SuggestPhaseSwitchTool) Name() string {
	return "suggest_phase_switch"
}

// Description returns a description of the tool.
func (t *SuggestPhaseSwitchTool) Description() string {
	return "Suggests switching the user's workspace tab to a different project phase (ONBOARDING, ANALYSIS, PLANNING, SOLUTIONING, IMPLEMENTATION) when the user asks for a task or skill that belongs to another phase."
}

// Execute runs the tool with the given arguments.
func (t *SuggestPhaseSwitchTool) Execute(ctx tool.Context, input *SuggestPhaseSwitchInput) (*SuggestPhaseSwitchOutput, error) {
	slog.Info("Cross-phase switch suggested by agent",
		"targetPhase", input.TargetPhase,
		"skillId", input.SkillID,
		"reason", input.Reason,
	)

	return &SuggestPhaseSwitchOutput{
		TargetPhase: input.TargetPhase,
		SkillID:     input.SkillID,
		Reason:      input.Reason,
		Status:      "SUGGESTED",
	}, nil
}
