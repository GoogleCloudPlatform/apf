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

package db

import (
	"testing"
)

func TestDetermineArtifactType(t *testing.T) {
	tests := []struct {
		filePath string
		expected ArtifactType
	}{
		{"docs/brainstorm.md", ArtifactTypeBrainstorm},
		{"docs/research-doc.md", ArtifactTypeResearch},
		{"docs/recon/domain-research.md", ArtifactTypeResearch},
		{"docs/brief.md", ArtifactTypeProductBrief},
		{"docs/spec/plans/briefs/brief-apf-2026-08-22/brief.md", ArtifactTypeProductBrief},
		{"docs/PRD.md", ArtifactTypePrd},
		{"docs/spec/plans/prds/prd-my-app-2026/prd.md", ArtifactTypePrd},
		{"docs/ux-design.md", ArtifactTypeUxDesign},
		{"docs/spec/plans/ux/ux-spec/ux-design.md", ArtifactTypeUxDesign},
		{"docs/architecture.md", ArtifactTypeArchitecture},
		{"docs/spec/plans/architecture/arch-v1/architecture.md", ArtifactTypeArchitecture},
		{"docs/spec/plans/adrs/adr-001-database.md", ArtifactTypeArchitecture},
		{"docs/epics.md", ArtifactTypeEpicsStories},
		{"docs/epics_and_stories.md", ArtifactTypeEpicsStories},
		{"docs/epic-1.md", ArtifactTypeEpicsStories},
		{"docs/spec/plans/epics/epic-1-auth.md", ArtifactTypeEpicsStories},
		{"docs/story-1.1.md", ArtifactTypeStory},
		{"docs/stories.md", ArtifactTypeStory},
		{"docs/spec/implementation/stories/1-1-login.md", ArtifactTypeStory},
		{"docs/sprint-plan.md", ArtifactTypeSprintPlan},
		{"docs/readiness.md", ArtifactTypeReadiness},
		{"docs/project-context.md", ArtifactTypeProjectContext},
		{"AGENTS.md", ArtifactTypeProjectContext},
		{"docs/AGENTS.md", ArtifactTypeProjectContext},
		{"docs/project-status.md", ArtifactTypeProjectStatus},
		{"docs/prfaq-agentic-foundry.md", ArtifactTypePrfaq},
		{"docs/epic-1-retro-2026-07-11.md", ArtifactTypeRetrospective},
		{"docs/spec/implementation/retrospectives/retro-epic-1.md", ArtifactTypeRetrospective},
		{"docs/investigations/bug-investigation.md", ArtifactTypeInvestigation},
		{"docs/case-123.md", ArtifactTypeInvestigation},
		{"docs/test-summary.md", ArtifactTypeTestSummary},
		{"docs/traceability-matrix.md", ArtifactTypeTestSummary},
		{"docs/1-2-user-login.md", ArtifactTypeStory},
		{"docs/1-1-login.yaml", ArtifactTypeStory},
		{"docs/1-1-login.json", ArtifactTypeStory},
		{"docs/epic-1-story-2.md", ArtifactTypeStory},
		{"docs/spec/plans/prds/prd-checkout-flow-2026.md", ArtifactTypePrd},
		{"docs/brief-apf-2026-08-22.md", ArtifactTypeProductBrief},
		{"docs/epic-1-retro-2026-07-11.md", ArtifactTypeRetrospective},
		{"docs/spec/plans/architecture/v1/sub/architecture-2026.md", ArtifactTypeArchitecture},
		{"docs/sprint-status.yaml", ArtifactTypeSprintPlan},
		{"sprint-status.yaml", ArtifactTypeSprintPlan},
		{"docs/epics.yaml", ArtifactTypeEpicsStories},
		{"reconcile-brainstorming.md", ArtifactTypePrd},
		{"docs/prd/reconcile-brainstorming.md", ArtifactTypePrd},
		{"planning-artifacts/prd/reconcile-brainstorming.md", ArtifactTypePrd},
		{"docs/unknown.md", ""},
	}

	for _, tt := range tests {
		t.Run(tt.filePath, func(t *testing.T) {
			actual := DetermineArtifactType(tt.filePath)
			if actual != tt.expected {
				t.Errorf("DetermineArtifactType(%q) = %q, expected %q", tt.filePath, actual, tt.expected)
			}
		})
	}
}

func TestDetermineArtifactPhase(t *testing.T) {
	tests := []struct {
		filePath string
		expected string
	}{
		{"docs/project-context.md", "ONBOARDING"},
		{"AGENTS.md", "ONBOARDING"},
		{"docs/AGENTS.md", "ONBOARDING"},
		{"docs/project-status.md", "ONBOARDING"},
		{"docs/brief.md", "ANALYSIS"},
		{"docs/brainstorm.md", "ANALYSIS"},
		{"docs/research-doc.md", "ANALYSIS"},
		{"docs/recon/technical-research.md", "ANALYSIS"},
		{"docs/forged-idea.md", "ANALYSIS"},
		{"docs/prfaq-agentic-foundry.md", "ANALYSIS"},
		{"docs/spec/plans/briefs/brief-apf-2026-08-22/brief.md", "ANALYSIS"},
		{"docs/prd.md", "PLANNING"},
		{"docs/ux-design.md", "PLANNING"},
		{"docs/spec/plans/ux/ux-spec/ux-design.md", "PLANNING"},
		{"docs/spec/plans/specs/kernel.spec.md", "PLANNING"},
		{"docs/spec/plans/test-plans/phase-1-test-plan.md", "PLANNING"},
		{"reconcile-brainstorming.md", "PLANNING"},
		{"docs/prd/reconcile-brainstorming.md", "PLANNING"},
		{"planning-artifacts/prd/reconcile-brainstorming.md", "PLANNING"},
		{"docs/architecture.md", "SOLUTIONING"},
		{"docs/spec/plans/architecture/arch-v1/architecture.md", "SOLUTIONING"},
		{"docs/spec/plans/adrs/adr-001-database.md", "SOLUTIONING"},
		{"docs/epics.md", "SOLUTIONING"},
		{"docs/spec/plans/epics/epics-and-stories.md", "SOLUTIONING"},
		{"docs/spec/plans/epics/epic-1-auth.md", "SOLUTIONING"},
		{"docs/implementation-readiness-report.md", "SOLUTIONING"},
		{"docs/sprint-status.yaml", "IMPLEMENTATION"},
		{"docs/spec/implementation/stories/1-1-login.md", "IMPLEMENTATION"},
		{"docs/spec/implementation/stories/epic-1-story-2.md", "IMPLEMENTATION"},
		{"docs/spec/implementation/retrospectives/retro-epic-1.md", "IMPLEMENTATION"},
		{"docs/spec/implementation/code-reviews/cr-123-review.md", "IMPLEMENTATION"},
		{"docs/spec/implementation/traceability-matrix.md", "IMPLEMENTATION"},
		{"docs/epic-1-retro-2026-07-11.md", "IMPLEMENTATION"},
		{"docs/investigations/bug.md", "IMPLEMENTATION"},
		{"docs/story-1.1.md", "IMPLEMENTATION"},
		{"docs/unknown.md", "PLANNING"},
	}

	for _, tt := range tests {
		t.Run(tt.filePath, func(t *testing.T) {
			actual := DetermineArtifactPhase(tt.filePath)
			if actual != tt.expected {
				t.Errorf("DetermineArtifactPhase(%q) = %q, expected %q", tt.filePath, actual, tt.expected)
			}
		})
	}
}

func TestArtifactsMatch(t *testing.T) {
	tests := []struct {
		name     string
		a, b     Artifact
		expected bool
	}{
		{
			name:     "exact ID match",
			a:        Artifact{ID: "APF_prd.md", Title: "PRD"},
			b:        Artifact{ID: "APF_prd.md", Title: "Different Title"},
			expected: true,
		},
		{
			name:     "exact clean path match",
			a:        Artifact{Path: "docs/specs/prd.md"},
			b:        Artifact{Path: "docs/specs/prd.md"},
			expected: true,
		},
		{
			name:     "title and filename match",
			a:        Artifact{Title: "prd.md", Path: ""},
			b:        Artifact{Path: "docs/specs/prd.md"},
			expected: true,
		},
		{
			name:     "generic filename index.md without path match",
			a:        Artifact{Path: "folderA/index.md"},
			b:        Artifact{Path: "folderB/index.md"},
			expected: false,
		},
		{
			name:     "unrelated files",
			a:        Artifact{Path: "docs/specs/prd.md"},
			b:        Artifact{Path: "docs/specs/ux.md"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := ArtifactsMatch(tt.a, tt.b)
			if actual != tt.expected {
				t.Errorf("ArtifactsMatch() = %v, expected %v", actual, tt.expected)
			}
		})
	}
}

func TestDetermineArtifactPhaseForProject(t *testing.T) {
	proj := &Project{
		ID: "p1",
		BmadConfig: &BmadConfig{
			SpecFolder:                  "docs/spec",
			PlanningArtifactsPath:       "plans",
			ImplementationArtifactsPath: "implementation",
			TestingArtifactsPath:        "test",
		},
	}

	tests := []struct {
		filePath string
		expected string
	}{
		{"docs/spec/plans/brief_notes.md", "ANALYSIS"},
		{"docs/spec/plans/research.md", "ANALYSIS"},
		{"docs/spec/plans/brainstorming.md", "ANALYSIS"},
		{"docs/spec/plans/briefs/brief-apf-2026-08-22/brief.md", "ANALYSIS"},
		{"docs/spec/plans/prd.md", "PLANNING"},
		{"docs/spec/plans/prds/prd-v1/prd.md", "PLANNING"},
		{"docs/spec/plans/ux-design.md", "PLANNING"},
		{"docs/spec/plans/ux/ux-spec/ux-design.md", "PLANNING"},
		{"docs/spec/plans/specs/kernel.spec.md", "PLANNING"},
		{"docs/spec/plans/test-plans/phase-1-test-plan.md", "PLANNING"},
		{"docs/spec/plans/architecture.md", "SOLUTIONING"},
		{"docs/spec/plans/architecture/arch-v1/architecture.md", "SOLUTIONING"},
		{"docs/spec/plans/adrs/adr-001-database.md", "SOLUTIONING"},
		{"docs/spec/plans/readiness-report.md", "SOLUTIONING"},
		{"docs/spec/plans/epics.md", "SOLUTIONING"},
		{"docs/spec/plans/epics/epics-and-stories.md", "SOLUTIONING"},
		{"docs/spec/plans/epics/epic-1-auth.md", "SOLUTIONING"},
		{"docs/spec/implementation/prd_summary.md", "IMPLEMENTATION"},
		{"docs/spec/implementation/stories/1-1-login.md", "IMPLEMENTATION"},
		{"docs/spec/implementation/stories/epic-1-story-2.md", "IMPLEMENTATION"},
		{"docs/spec/implementation/sprint-status.yaml", "IMPLEMENTATION"},
		{"docs/spec/implementation/retrospectives/retro-epic-1.md", "IMPLEMENTATION"},
		{"docs/spec/brief.md", "ANALYSIS"},
		{"docs/spec/analysis.md", "ANALYSIS"},
		{"docs/spec/prd.md", "PLANNING"},
		{"docs/spec/plans/prd/reconcile-brainstorming.md", "PLANNING"},
		{"docs/prd/reconcile-brainstorming.md", "PLANNING"},
	}

	for _, tt := range tests {
		t.Run(tt.filePath, func(t *testing.T) {
			actual := DetermineArtifactPhaseForProject(tt.filePath, proj)
			if actual != tt.expected {
				t.Errorf("DetermineArtifactPhaseForProject(%q) = %q, expected %q", tt.filePath, actual, tt.expected)
			}
		})
	}
}
