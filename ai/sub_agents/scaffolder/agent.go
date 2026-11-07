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

package scaffolder

import (
	"context"
	"log"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
	"google.golang.org/adk/tool/mcptoolset"

	"google.com/apf/ai/core"
	"google.com/apf/ai/sub_agents/scaffolder/tools"
	mcp "google.com/apf/ai/tools/mcp"
)

// NewRootAgent creates and returns the root agent with all configured tools.
func NewAgentLoader(ctx context.Context, cfg *core.Configuration) (agent.Loader, error) {
	model := core.NewDynamicModel(cfg, cfg.DefaultLLMModel)

	getGoldenPathTool, err := functiontool.New(functiontool.Config{
		Name:        "get_golden_path",
		Description: "Get the golden path for a given golden path ID.",
	}, tools.GetGoldenPath)
	if err != nil {
		return nil, err
	}

	getGoldenPathTemplateTool, err := functiontool.New(functiontool.Config{
		Name:        "get_golden_path_template",
		Description: "Get the golden path template for a given golden path ID.",
	}, tools.GetGoldenPathTemplate)
	if err != nil {
		return nil, err
	}

	transport := mcp.GithubMCPTransport(ctx, cfg)
	mcpToolSet, err := mcptoolset.New(mcptoolset.Config{
		Transport: transport,
	})
	if err != nil {
		log.Fatalf("Failed to create MCP tool set: %v", err)
	}

	sourceCodeScaffolderAgent, err := llmagent.New(llmagent.Config{
		Name:        "SourceCodeScaffolder",
		Model:       model,
		Description: "Agent responsible for scaffolding source code.",
		Instruction: `You are responsible for scaffolding the source code.
			1. Read the local archetype files:
				a. Use the tool 'get_golden_path_template' with 'golden_path' to get the content of all files in the archetype.
				b. Print to the logs the files/folders that you are about to process.
			2. Process the files:
				a. STRICTLY ONLY modify the content of the files returned by the tool.
				b. DO NOT create any new files or folders that were not present in the archetype.
				c. Replace the 'PROJECT_NAME' placeholder with 'project_name' (assuming deterministic naming) in all configuration files (e.g., cloudbuild.yaml, README.md) and other source files.
			3. Push to GitHub:
				a. Extract the owner from '[source_code_github]' (e.g., if 'https://github.com/hader-google', target 'hader-google').
				b. Create a new repository named after 'project_name'.
				- **CRITICAL**: If the owner ('[source_code_github]') is a **User** and not an Organization, do NOT try to create it "in an organization". Just create the repository for that user.
				c. Push the modified archetype files (with replaced placeholders) to the new repository called 'project_name'.
		`,
		Tools: []tool.Tool{getGoldenPathTemplateTool},
		Toolsets: []tool.Toolset{
			mcpToolSet,
		},
	})
	if err != nil {
		return nil, err
	}

	rootAgent, err := llmagent.New(llmagent.Config{
		Name:        "ScaffolderOrchestrator",
		Model:       model,
		Description: "Orchestrator that plans the scaffolding based on a golden path.",
		Instruction: `You are the orchestrator. 
		1. Receive 'project_name' and 'golden_path_id'. 
		2. Use 'GetGoldenPath' to find template details. 
		3. If found, call 'SourceCodeScaffolder' 
		with the enriched context (project_name, golden_path_id).`,
		Tools:     []tool.Tool{getGoldenPathTool},
		SubAgents: []agent.Agent{sourceCodeScaffolderAgent},
	})
	if err != nil {
		return nil, err
	}

	return agent.NewSingleLoader(rootAgent), nil
}
