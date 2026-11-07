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

package help

import (
	_ "embed"

	"context"
	"encoding/json"
	"fmt"

	"google.com/apf/ai/core"
	subagents "google.com/apf/ai/sub_agents"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/tool"
)

//go:embed sub_agents.json
var subAgents []byte

func NewAgentLoader(ctx context.Context, cfg *core.Configuration) (agent.Loader, error) {
	model := core.NewDynamicModel(cfg, cfg.DefaultLLMModel)

	fileReaderTool, err := subagents.GetFileReaderTool()
	if err != nil {
		return nil, fmt.Errorf("failed to create file tool: %w", err)
	}

	fileWriterTool, err := subagents.GetFileWriterTool()
	if err != nil {
		return nil, fmt.Errorf("failed to create file tool: %w", err)
	}

	directoryReaderTool, err := subagents.GetDirectoryReaderTool()
	if err != nil {
		return nil, fmt.Errorf("failed to create directory reader tool: %w", err)
	}

	uvRunnerTool, err := subagents.GetUVRunnerTool(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create uv runner tool: %w", err)
	}

	findTool, err := subagents.GetFindTool()
	if err != nil {
		return nil, fmt.Errorf("failed to create find tool: %w", err)
	}

	suggestPhaseSwitchTool, err := subagents.GetSuggestPhaseSwitchTool()
	if err != nil {
		return nil, fmt.Errorf("failed to create suggest phase switch tool: %w", err)
	}

	// Create agents from catalog
	var catalog subagents.SubAgentList
	err = json.Unmarshal(subAgents, &catalog)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal sub agents: %w", err)
	}

	var agents []agent.Agent

	for _, agentCfg := range catalog.Agents {

		agentInstructions := agentCfg.Instructions + "\n" + subagents.GetGlobalScratchInstructions()

		agent, err := llmagent.New(llmagent.Config{
			Name:        agentCfg.Name,
			Model:       model,
			Description: agentCfg.Description,
			Instruction: agentInstructions,
			Tools: []tool.Tool{
				fileReaderTool,
				fileWriterTool,
				directoryReaderTool,
				findTool,
				uvRunnerTool,
			},
			BeforeAgentCallbacks: []agent.BeforeAgentCallback{subagents.OnBeforeAgent},
		})

		if err != nil {
			return nil, fmt.Errorf("failed to create agent %s: %w", agentCfg.Name, err)
		}

		agents = append(agents, agent)
	}

	instructions := "Greet the user and wait for instructions. Confirm understanding and then delegate to the corresponding sub agent.\n" +
		subagents.GetGlobalCrossPhaseInstructions("HELP")

	rootAgent, err := llmagent.New(llmagent.Config{
		Name:        "APF Help Agent",
		Model:       model,
		Description: "A general help subagent swarm that assists the user with BMad questions, recommendations, and guidance.",
		Instruction: instructions,
		SubAgents:   agents,
		Tools: []tool.Tool{
			fileReaderTool,
			fileWriterTool,
			directoryReaderTool,
			findTool,
			uvRunnerTool,
			suggestPhaseSwitchTool,
		},
		BeforeAgentCallbacks: []agent.BeforeAgentCallback{subagents.OnBeforeAgent},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create root agent: %w", err)
	}

	return agent.NewMultiLoader(rootAgent)
}
