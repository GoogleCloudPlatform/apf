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

package registry

import (
	"context"
	"fmt"

	"google.com/apf/ai/core"
	"google.com/apf/ai/sub_agents/scaffolder"
	"google.com/apf/ai/sub_agents/analysis"
	"google.com/apf/ai/sub_agents/help"
	"google.com/apf/ai/sub_agents/implementation"
	"google.com/apf/ai/sub_agents/overwatch"
	"google.com/apf/ai/sub_agents/planning"
	"google.com/apf/ai/sub_agents/solutioning"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
)

type AgentLoaderFunc func(context.Context, *core.Configuration) (agent.Loader, error)

var agentLoaders = make(map[string]AgentLoaderFunc)

func RegisterAllAgents() {
	agentLoaders["scaffolder"] = scaffolder.NewAgentLoader
	agentLoaders["analysis"] = analysis.NewAgentLoader
	agentLoaders["planning"] = planning.NewAgentLoader
	agentLoaders["overwatch"] = overwatch.NewAgentLoader
	agentLoaders["solutioning"] = solutioning.NewAgentLoader
	agentLoaders["implementation"] = implementation.NewAgentLoader
	agentLoaders["help"] = help.NewAgentLoader
}

func GetAgentLoader(name string) AgentLoaderFunc {
	loader, ok := agentLoaders[name]
	if !ok {
		return func(ctx context.Context, cfg *core.Configuration) (agent.Loader, error) {
			return nil, fmt.Errorf("agent loader not found: %s", name)
		}
	}
	return loader
}

func GetSuperLoader(ctx context.Context, cfg *core.Configuration) (agent.Loader, error) {

	var agents []agent.Agent

	model := core.NewDynamicModel(cfg, cfg.DefaultLLMModel)

	root, err := llmagent.New(llmagent.Config{
		Name:        "root_agent",
		Model:       model,
		Description: "Root agent for Multiloader",
		Instruction: "You are the orchestrator for multiple agents. Your job is to orchestrate the work across different agents to build an agentic workflow of agents who can help build a complete product.",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create super agent: %w", err)
	}

	for _, loader := range agentLoaders {
		agentLoader, err := loader(ctx, cfg)
		if err != nil {
			return nil, err
		}
		agents = append(agents, agentLoader.RootAgent())
	}

	return agent.NewMultiLoader(root, agents...)
}
