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

package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"log"

	"google.com/apf/api/db"
)

//go:embed templates/showcase/auto_dev_loop_workflow.json
var showcaseWorkflowJSON []byte

func seedWorkflows(ctx context.Context) {
	seedBmadWorkflow(ctx)
	//seedAdkWorkflow(ctx)
	seedShowcaseWorkflow(ctx)
}

func seedBmadWorkflow(ctx context.Context) {
	// Seed BMAD Workflow
	bmadWorkflow := db.WorkflowTemplate{
		Name:      "Standard BMAD Workflow",
		Framework: "BMAD",
		Phases: map[string][]db.PhaseAction{
			"ONBOARDING": {
				{ID: "bmad-project-context", Name: "Project Context", DisplayPrompt: "Generate project context by setting up, refreshing, or auditing repository instructions (AGENTS.md).", Prompt: "/bmad-project-context"},
				{ID: "bmad-help", Name: "Help", DisplayPrompt: "Analyze current state and recommend the next steps or skills in the BMad workflow.", Prompt: "/bmad-help"},
			},
			"ANALYSIS": {
				{ID: "bmad-agent-analyst", Name: "Talk to Business Analyst (Mary)", DisplayPrompt: "Strategic business analyst persona (Mary) expert in requirements discovery, domain research, competitive analysis, and business modeling.", Prompt: "/bmad-agent-analyst"},
				{ID: "bmad-brainstorming", Name: "Brainstorming", DisplayPrompt: "Facilitates creative brainstorming sessions using diverse ideation techniques to explore innovative product concepts.", Prompt: "/bmad-brainstorming"},
				{ID: "bmad-deep-recon", Name: "Deep Recon", DisplayPrompt: "Conduct decision-grade research across market, domain, technical, competitive, user-voice, and academic literature.", Prompt: "/bmad-deep-recon"},
				{ID: "bmad-forge-idea", Name: "Forge Idea", DisplayPrompt: "Pressure-tests an idea through persona-driven interrogation until it hardens, proves out, or dies cheaply.", Prompt: "/bmad-forge-idea"},
				{ID: "bmad-prfaq", Name: "PRFAQ Challenge", DisplayPrompt: "Working Backwards PRFAQ challenge that stress-tests a product concept customer-first via press releases and customer/internal FAQs.", Prompt: "/bmad-prfaq"},
				{ID: "bmad-product-brief", Name: "Product Brief", DisplayPrompt: "Creates, updates, or validates a concise Product Brief summarizing goals, scope, and key success metrics.", Prompt: "/bmad-product-brief"},
				{ID: "bmad-help", Name: "Help", DisplayPrompt: "Analyze current state and recommend the next steps or skills in the BMad workflow.", Prompt: "/bmad-help"},
			},
			"PLANNING": {
				{ID: "bmad-agent-pm", Name: "Talk to Product Manager (John)", DisplayPrompt: "Product manager persona (John) for PRD creation, requirements discovery, PRFAQs, customer journeys, and feature prioritization.", Prompt: "/bmad-agent-pm"},
				{ID: "bmad-agent-ux-designer", Name: "Talk to UX Designer (Sally)", DisplayPrompt: "UX designer and UI specialist persona (Sally) responsible for wireframes, interaction workflows, design systems, and usability guidelines.", Prompt: "/bmad-agent-ux-designer"},
				{ID: "bmad-prd", Name: "Create PRD", DisplayPrompt: "Creates, updates, or validates a comprehensive Product Requirements Document (PRD).", Prompt: "/bmad-prd"},
				{ID: "bmad-spec", Name: "Spec Distillation", DisplayPrompt: "Distills intent into the SPEC kernel and companion documents: the canonical, preservation-validated contract for downstream development.", Prompt: "/bmad-spec"},
				{ID: "bmad-ux", Name: "UX Design", DisplayPrompt: "Plans UX interaction patterns, UI components, layout structures, and design specifications.", Prompt: "/bmad-ux"},
				{ID: "bmad-testarch-test-design", Name: "Test Design", DisplayPrompt: "Creates comprehensive system-level or epic-level test plans, test strategies, and risk matrices.", Prompt: "/bmad-testarch-test-design"},
				{ID: "bmad-review", Name: "Review", DisplayPrompt: "Multi-lens review engine over any diff, spec, doc, or artifact (adversarial, edge-case, verification-gap, structure, prose).", Prompt: "/bmad-review"},
				{ID: "bmad-help", Name: "Help", DisplayPrompt: "Analyze current state and recommend the next steps or skills in the BMad workflow.", Prompt: "/bmad-help"},
			},
			"SOLUTIONING": {
				{ID: "bmad-agent-architect", Name: "Talk to System Architect (Winston)", DisplayPrompt: "System architect and technical design leader persona (Winston) focusing on system invariants, data schemas, API boundaries, non-functional requirements, and architectural spikes.", Prompt: "/bmad-agent-architect"},
				{ID: "bmad-architecture", Name: "Architecture", DisplayPrompt: "Produces the system architecture spine of invariants, data models, API boundaries, and technical decisions that guide development.", Prompt: "/bmad-architecture"},
				{ID: "bmad-review", Name: "Review", DisplayPrompt: "Multi-lens review engine over any diff, spec, doc, or artifact (adversarial, edge-case, verification-gap, structure, prose).", Prompt: "/bmad-review"},
				{ID: "bmad-help", Name: "Help", DisplayPrompt: "Analyze current state and recommend the next steps or skills in the BMad workflow.", Prompt: "/bmad-help"},
			},
			"IMPLEMENTATION": {
				{ID: "bmad-create-epics-and-stories", Name: "Create Epics and Stories", DisplayPrompt: "Breaks product requirements and architecture into epics and actionable user stories with explicit acceptance criteria.", Prompt: "/bmad-create-epics-and-stories"},
				{ID: "bmad-sprint-planning", Name: "Sprint Planning", DisplayPrompt: "Gates planning readiness, generates sprint tracking, summarizes sprint progress, and repairs tracking state.", Prompt: "/bmad-sprint-planning"},
				{ID: "bmad-correct-course", Name: "Correct Course", DisplayPrompt: "Manages significant changes during sprint execution, realigning story scope and updating sprint tracking.", Prompt: "/bmad-correct-course"},
				{ID: "bmad-tea", Name: "Talk to Test Architect (Murat)", DisplayPrompt: "Test Architect and Quality Advisor persona (Murat) providing test architecture strategy, ATDD design, NFR evaluation, and test suite reviews.", Prompt: "/bmad-tea"},
				{ID: "bmad-retrospective", Name: "Retrospective", DisplayPrompt: "Runs evidence-based epic retrospectives, verifying deliverables against sources and rendering acceptance verdicts.", Prompt: "/bmad-retrospective"},
				{ID: "bmad-review", Name: "Review", DisplayPrompt: "Multi-lens review engine over any diff, spec, doc, or artifact (adversarial, edge-case, verification-gap, structure, prose).", Prompt: "/bmad-review"},
				{ID: "bmad-help", Name: "Help", DisplayPrompt: "Analyze current state and recommend the next steps or skills in the BMad workflow.", Prompt: "/bmad-help"},
			},
		},
		Artifacts: map[string][]db.ArtifactTemplate{
			"ONBOARDING": {
				{ID: "project-context-doc", Title: "Project Context", Required: false},
				{ID: "project-status-doc", Title: "Project Status", Required: false},
			},
			"ANALYSIS": {
				{ID: "brainstorming-session-doc", Title: "Brainstorming Sessions", Required: false},
				{ID: "research-doc", Title: "Research Documents", Required: false},
				{ID: "product-brief-doc", Title: "Product Briefs", Required: false},
			},
			"PLANNING": {
				{ID: "product-req-doc", Title: "PRDs", Required: true},
				{ID: "ux-design-doc", Title: "UX Designs", Required: false},
			},
			"SOLUTIONING": {
				{ID: "architecture-doc", Title: "Architecture Documents", Required: true},
				{ID: "epics-stories-doc", Title: "Epics and Stories", Required: true},
				{ID: "readiness-report-doc", Title: "Readiness Reports", Required: true},
			},
			"IMPLEMENTATION": {
				{ID: "sprint-plan-doc", Title: "Sprint Plan", Required: true},
				{ID: "story-doc", Title: "Story", Required: true},
			},
		},
		Dependencies: map[string][]string{
			"ONBOARDING":     {},
			"ANALYSIS":       {"ONBOARDING"},
			"PLANNING":       {"ONBOARDING"},
			"SOLUTIONING":    {"PLANNING"},
			"IMPLEMENTATION": {"SOLUTIONING"},
		},
	}

	// Idempotency check
	workflows, err := workflowStore.ListWorkflows(ctx)
	var existingWorkflow *db.WorkflowTemplate
	if err == nil {
		for _, w := range workflows {
			if w.Framework == "BMAD" {
				existingWorkflow = &w
				break
			}
		}
	}

	if existingWorkflow != nil {
		bmadWorkflow.ID = existingWorkflow.ID
		log.Printf("Found existing BMAD workflow, updating content for ID: %s", bmadWorkflow.ID)
	}

	err = workflowStore.CreateWorkflow(ctx, &bmadWorkflow)
	if err != nil {
		log.Printf("Failed to seed BMAD workflow: %v", err)
	} else {
		log.Printf("Successfully seeded BMAD workflow with ID: %s", bmadWorkflow.ID)
	}
}

func seedShowcaseWorkflow(ctx context.Context) {
	var wf db.WorkflowTemplate
	if err := json.Unmarshal(showcaseWorkflowJSON, &wf); err != nil {
		log.Printf("Failed to unmarshal showcase workflow: %v", err)
		return
	}

	wf.ID = "showcase-workflow-dev-loop"

	// Idempotency check
	workflows, err := workflowStore.ListWorkflows(ctx)
	var existingWorkflow *db.WorkflowTemplate
	if err == nil {
		for _, w := range workflows {
			if w.ID == wf.ID {
				existingWorkflow = &w
				break
			}
		}
	}

	if existingWorkflow != nil {
		log.Printf("Found existing showcase workflow, updating content for ID: %s", wf.ID)
	}

	err = workflowStore.CreateWorkflow(ctx, &wf)
	if err != nil {
		log.Printf("Failed to seed showcase workflow: %v", err)
	} else {
		log.Printf("Successfully seeded showcase workflow with ID: %s", wf.ID)
	}
}
