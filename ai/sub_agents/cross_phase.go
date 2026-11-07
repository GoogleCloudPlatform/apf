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

import "fmt"

// GetGlobalCrossPhaseInstructions returns the canonical phase-to-skill mapping guidance
// to inject into all Phase Root Agents so they can detect cross-phase skill requests.
func GetGlobalCrossPhaseInstructions(currentPhase string) string {
	return fmt.Sprintf(`
### CROSS-PHASE SKILL DETECTION & REDIRECTION GUIDANCE
You are currently operating in the "%s" phase tab of the project workspace.
Every workspace tab hosts an isolated agent specialist and specific skills:

1. ONBOARDING (APF Overwatch Agent): Project Context ('bmad-project-context'), Project Status ('project-status-tracker').
2. ANALYSIS (APF Analysis Agent / Mary): Business Analyst ('bmad-agent-analyst'), Brainstorming ('bmad-brainstorming'), Deep Recon ('bmad-deep-recon'), Forge Idea ('bmad-forge-idea'), PRFAQ Challenge ('bmad-prfaq'), Product Brief ('bmad-product-brief').
3. PLANNING (APF Planning Agent / John PM & Sally UX): Product Manager ('bmad-agent-pm'), UX Designer ('bmad-agent-ux-designer'), Create PRD ('bmad-prd'), Spec Distillation ('bmad-spec'), UX Design ('bmad-ux'), Test Design ('bmad-testarch-test-design'), Review ('bmad-review').
4. SOLUTIONING (APF Solutioning Agent / Winston Architect): System Architect ('bmad-agent-architect'), Create Architecture ('bmad-architecture'), Create Epics and Stories ('bmad-create-epics-and-stories'), Review ('bmad-review').
5. IMPLEMENTATION (APF Implementation Agent / Murat TEA): Sprint Planning ('bmad-sprint-planning'), Correct Course ('bmad-correct-course'), Test Architect ('bmad-tea'), Retrospective ('bmad-retrospective'), Review ('bmad-review').

UNIVERSAL SKILLS:
- BMad Help ('bmad-help') is available in EVERY phase tab. Do NOT redirect or suggest a phase switch when the user asks for BMad help, next steps, or guidance; delegate directly to your local BMad Help sub-agent.

IMPORTANT RULE:
If the user asks to execute a skill or task that explicitly belongs to another phase (for example, asking to "Create a PRD" while in SOLUTIONING, or asking to "Brainstorm" while in SOLUTIONING):
- DO NOT attempt to run the task yourself or pretend you have that skill.
- MUST call the 'suggest_phase_switch' tool with targetPhase (e.g. "PLANNING"), skillId (e.g. "create-prd"), and reason explaining which specialist handles that task in that phase.
- Inform the user politely in your response that the task belongs to that phase and that they can click the switch card to continue.

### TEMPORARY EXECUTION SCRIPTS & SCRATCH DIRECTORY
When authoring or executing temporary helper scripts, batch scripts (such as batch memlog population scripts), or scratch utilities:
- Store temporary execution scripts in {project-root}/.tmp/
- Execute them using the 'uv_run' tool with scriptPath set to '{project-root}/.tmp/<script_name>.py'.
`, currentPhase)
}

// GetGlobalScratchInstructions returns universal instructions for temporary execution scripts.
func GetGlobalScratchInstructions() string {
	return `
### TEMPORARY EXECUTION SCRIPTS & SCRATCH DIRECTORY
When authoring or executing temporary helper scripts, batch scripts (such as batch memlog population scripts), or scratch utilities:
- Store temporary execution scripts in {project-root}/.tmp/
- Execute them using the 'uv_run' tool with scriptPath set to '{project-root}/.tmp/<script_name>.py'.
`
}
