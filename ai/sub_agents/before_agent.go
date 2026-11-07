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

package subagents

import (
	"fmt"
	"log"
	"log/slog"

	"google.com/apf/ai/core"
	"google.golang.org/adk/agent"
	"google.golang.org/genai"
)

func OnBeforeAgent(ctx agent.CallbackContext) (*genai.Content, error) {
	agentName := ctx.AgentName()
	log.Printf("[Callback] Entering agent: %s", agentName)
	if skip, _ := ctx.State().Get("skip_llm_agent"); skip == true {
		log.Printf("[Callback] State condition met: Skipping agent %s", agentName)
		return genai.NewContentFromText(
				fmt.Sprintf("Agent %s skipped by before_agent_callback.", agentName),
				genai.RoleModel,
			),
			nil
	}

	if val, err := ctx.State().Get("workspace_ready"); err == nil && val == "true" {
		slog.Debug("Skipping workspace initialization as workspace is already ready")
	} else {
		cfg := core.NewConfig()
		_, err := core.NewWorkspace(ctx, cfg)

		if err != nil {
			return nil, fmt.Errorf("workspace initialization failed: %w", err)
		}
	}

	return nil, nil
}
