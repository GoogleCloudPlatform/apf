/**
 * Copyright 2026 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

export interface BmadSkill {
  id: string
  name: string
  code: string
  module: "BMad Method" | "BMad Builder" | "Core" | "Test Architecture Enterprise" | "BMAD Loop Skills"
  phase: "0-learning" | "1-ideation" | "2-planning" | "3-solutioning" | "4-implementation" | "anytime"
  phaseLabel: string
  description: string
  action?: string
  args?: string
  precededBy?: string
  followedBy?: string
  required?: boolean
  outputLocation?: string
  outputs?: string
  tags: string[]
  examples?: string[]
  apf_supported?: boolean
  requires_sandbox?: boolean
}

export const BMAD_PHASES = [
  { id: "all", label: "All Phases" },
  { id: "1-ideation", label: "Ideation & Discovery" },
  { id: "2-planning", label: "Planning & Requirements" },
  { id: "3-solutioning", label: "Solutioning & Design" },
  { id: "4-implementation", label: "Implementation & Ship" },
  { id: "0-learning", label: "Learning & Academy" },
  { id: "anytime", label: "Anytime & Tooling" },
] as const

export const BMAD_MODULES = [
  "All",
  "BMad Method",
  "BMad Builder",
  "Test Architecture Enterprise",
  "BMAD Loop Skills",
  "Core"
] as const

export const BMAD_SKILLS_CATALOG: BmadSkill[] = [
  // ================= BMAD METHOD =================
  {
    id: "bmad-brainstorming",
    apf_supported: true,
    name: "Brainstorm Project",
    code: "BP",
    module: "BMad Method",
    phase: "1-ideation",
    phaseLabel: "Ideation & Discovery",
    description: "Expert-guided facilitation through creative brainstorming techniques (first principles, SCAMPER, crazy 8s, role-storming) to unlock innovative product ideas.",
    action: "brainstorm",
    args: "{technique: optional specific technique}|{topic: initial concept}",
    followedBy: "bmad-product-brief or bmad-prfaq",
    required: false,
    outputLocation: "{output_folder}/brainstorming",
    outputs: "Brainstorming notes, idea clusters & synthesis report",
    tags: ["ideation", "creative", "brainstorm", "discovery", "phase-1"],
    examples: [
      "/bmad-brainstorming \"AI-powered onboarding workflow\"",
      "Help me brainstorm features for a collaborative canvas editor"
    ]
  },
  {
    id: "bmad-product-brief",
    apf_supported: true,
    name: "Create Product Brief",
    code: "CB",
    module: "BMad Method",
    phase: "1-ideation",
    phaseLabel: "Ideation & Discovery",
    description: "An expert-guided experience to nail down your product idea in a concise, authoritative brief. Ideal when you already have a clear concept and want a streamlined foundation.",
    action: "create-brief",
    args: "-A (automated mode)",
    followedBy: "bmad-prd",
    required: false,
    outputLocation: "planning_artifacts",
    outputs: "Product Brief document (Markdown)",
    tags: ["brief", "concept", "product-definition", "phase-1"],
    examples: [
      "/bmad-product-brief \"Real-time multi-agent activity stream\"",
      "Let's create a product brief for organization auto-provisioning"
    ]
  },
  {
    id: "bmad-prfaq",
    apf_supported: true,
    name: "Working Backwards PRFAQ Challenge",
    code: "WB",
    module: "BMad Method",
    phase: "1-ideation",
    phaseLabel: "Ideation & Discovery",
    description: "Amazon Working Backwards-style guided challenge to forge and stress-test your concept. Puts the product through the customer press release and internal/external FAQ gauntlet.",
    action: "prfaq-challenge",
    args: "-H (headless mode)",
    followedBy: "bmad-prd",
    required: false,
    outputLocation: "planning_artifacts",
    outputs: "PRFAQ document with Press Release, Customer FAQ & Stakeholder FAQ",
    tags: ["prfaq", "working-backwards", "customer-centric", "validation", "phase-1"],
    examples: [
      "/bmad-prfaq \"Automated code migration assistant\"",
      "Run the PRFAQ challenge for the new developer sandbox"
    ]
  },
  {
    id: "bmad-forge-idea",
    apf_supported: true,
    name: "Forge Idea (Stress-Test)",
    code: "FI",
    module: "Core",
    phase: "1-ideation",
    phaseLabel: "Ideation & Discovery",
    description: "Persona-driven interrogation to pressure-test an idea until it hardens, proves feasible, or fails cheaply before significant investment.",
    action: "forge",
    followedBy: "bmad-spec or bmad-build",
    required: false,
    outputLocation: "{output_folder}/forge",
    outputs: "Refined idea brief, risk matrix & recommendation",
    tags: ["pressure-test", "stress-test", "interrogation", "validation", "phase-1"],
    examples: [
      "/bmad-forge-idea \"Zero-trust microVM execution pool\"",
      "Stress test my idea for automated database sharding"
    ]
  },
  {
    id: "bmad-prd",
    apf_supported: true,
    name: "Create, Edit & Review PRD",
    code: "PRD",
    module: "BMad Method",
    phase: "2-planning",
    phaseLabel: "Planning & Requirements",
    description: "Facilitated PRD workflow to create a comprehensive PRD via coached discovery, update against new change signals, or validate against quality checklists with HTML findings reports.",
    precededBy: "bmad-product-brief or bmad-prfaq",
    followedBy: "bmad-ux or bmad-architecture",
    required: true,
    outputLocation: "planning_artifacts",
    outputs: "Product Requirements Document (PRD.md)",
    tags: ["prd", "requirements", "product-spec", "functional-requirements", "phase-2"],
    examples: [
      "/bmad-prd \"Interactive documentation screen with search\"",
      "Create the PRD for user role management and invitation flow"
    ]
  },
  {
    id: "bmad-ux",
    apf_supported: true,
    name: "UX Design & Pattern Planning",
    code: "CU",
    module: "BMad Method",
    phase: "2-planning",
    phaseLabel: "Planning & Requirements",
    description: "Guidance through realizing the plan for user experience, interaction workflows, information hierarchy, and UI component specifications.",
    precededBy: "bmad-prd",
    followedBy: "bmad-architecture",
    required: false,
    outputLocation: "planning_artifacts",
    outputs: "UX Design Specification & Component Wireframes",
    tags: ["ux", "ui", "wireframes", "interaction-design", "phase-2"],
    examples: [
      "/bmad-ux \"Two-pane split layout for docs with sticky nav\"",
      "Design the UX specification for the activity feed side drawer"
    ]
  },
  {
    id: "bmad-project-context",
    apf_supported: true,
    name: "Project Context & AGENTS.md",
    code: "PC",
    module: "BMad Method",
    phase: "2-planning",
    phaseLabel: "Planning & Requirements",
    description: "Sets up or refreshes repository agent instructions (AGENTS.md block) so AI swarms follow verified commands, style policies, architecture conventions, and recorded pitfalls.",
    outputLocation: "repo root (AGENTS.md)",
    outputs: "AGENTS.md managed instructions block & pitfall registry",
    tags: ["agents.md", "context", "repo-guidelines", "pitfalls", "policy", "phase-2"],
    examples: [
      "/bmad-project-context --refresh",
      "Generate project context instructions for the frontend package"
    ]
  },
  {
    id: "bmad-spec",
    apf_supported: true,
    name: "Distill Intent to Canonical Spec",
    code: "SPC",
    module: "BMad Method",
    phase: "2-planning",
    phaseLabel: "Planning & Requirements",
    description: "Distills any intent input (PRD, brief, notes, chat transcript) into a succinct, invariant SPEC.md contract + companions that downstream implementation derives from without ambiguity.",
    args: "[path to raw intent files]",
    outputLocation: "{output_folder}/specs/spec-{slug}",
    outputs: "SPEC.md + companion contracts & invariants",
    tags: ["spec", "contract", "invariants", "kernel", "phase-2"],
    examples: [
      "/bmad-spec docs/prd-search.md",
      "Distill this conversation into an executable SPEC"
    ]
  },
  {
    id: "bmad-architecture",
    apf_supported: true,
    name: "Technical Architecture Spine",
    code: "CA",
    module: "BMad Method",
    phase: "3-solutioning",
    phaseLabel: "Solutioning & Design",
    description: "Produces the architecture spine: the lean set of invariants, component boundaries, data schemas, API contracts, and integration models that keep features consistent across iterations.",
    precededBy: "bmad-prd (and bmad-ux)",
    followedBy: "bmad-create-epics-and-stories",
    required: true,
    outputLocation: "planning_artifacts",
    outputs: "Technical Architecture Document (Architecture.md)",
    tags: ["architecture", "technical-design", "invariants", "schemas", "apis", "phase-3"],
    examples: [
      "/bmad-architecture",
      "Produce the technical architecture for the multi-tenant IAM system"
    ]
  },
  {
    id: "bmad-testarch-test-design",
    apf_supported: true,
    name: "Risk-Based Test Design",
    code: "TD",
    module: "Test Architecture Enterprise",
    phase: "3-solutioning",
    phaseLabel: "Solutioning & Design",
    description: "Enterprise test planning: risk analysis, critical path mapping, boundary condition matrix, and end-to-end test scenarios before writing code.",
    precededBy: "bmad-architecture",
    followedBy: "bmad-testarch-framework",
    required: false,
    outputLocation: "test_artifacts",
    outputs: "Test Design Document & Risk Coverage Matrix",
    tags: ["testing", "test-plan", "risk-matrix", "tea", "phase-3"],
    examples: [
      "/bmad-testarch-test-design",
      "Design test plan for token refresh middleware and auth redirects"
    ]
  },
  {
    id: "bmad-testarch-framework",
    apf_supported: true,
    requires_sandbox: true,
    name: "Test Framework Scaffolding",
    code: "TF",
    module: "Test Architecture Enterprise",
    phase: "3-solutioning",
    phaseLabel: "Solutioning & Design",
    description: "Initializes production-grade test harness (Vitest, Jest, Playwright, Cypress) with CI reporters, mock servers, fixtures, and execution scripts.",
    precededBy: "bmad-testarch-test-design",
    followedBy: "bmad-testarch-ci",
    required: false,
    outputLocation: "test_artifacts",
    outputs: "Framework scaffold, configs & helper utilities",
    tags: ["testing", "framework", "playwright", "vitest", "tea", "phase-3"],
    examples: [
      "/bmad-testarch-framework --vitest --playwright",
      "Initialize testing harness for Next.js and Go API services"
    ]
  },
  {
    id: "bmad-testarch-ci",
    apf_supported: true,
    requires_sandbox: true,
    name: "CI/CD Quality Pipeline",
    code: "CI",
    module: "Test Architecture Enterprise",
    phase: "3-solutioning",
    phaseLabel: "Solutioning & Design",
    description: "Configures automated quality gates, linting checks, parallel test runners, coverage thresholds, and status reporting in CI workflows.",
    precededBy: "bmad-testarch-framework",
    required: false,
    outputLocation: "test_artifacts",
    outputs: "CI/CD workflow definitions (.github/workflows, etc.)",
    tags: ["ci", "cd", "quality-gates", "pipelines", "tea", "phase-3"],
    examples: [
      "/bmad-testarch-ci",
      "Set up GitHub Actions workflow with unit tests, e2e, and lint gates"
    ]
  },
  {
    id: "bmad-create-epics-and-stories",
    apf_supported: true,
    name: "Create Epics & Stories",
    code: "CE",
    module: "BMad Method",
    phase: "4-implementation",
    phaseLabel: "Implementation & Ship",
    description: "Breaks requirements and architecture down into right-sized, sequenced epics and user stories with precise acceptance criteria and verification plans.",
    precededBy: "bmad-architecture",
    followedBy: "bmad-sprint-planning",
    required: true,
    outputLocation: "planning_artifacts",
    outputs: "Epics & Stories registry (epics_and_stories.md)",
    tags: ["epics", "stories", "backlog", "breakdown", "phase-4"],
    examples: [
      "/bmad-create-epics-and-stories",
      "Break down the documentation screen feature into executable stories"
    ]
  },
  {
    id: "bmad-sprint-planning",
    apf_supported: true,
    name: "Sprint Planning & Readiness Gate",
    code: "SP",
    module: "BMad Method",
    phase: "4-implementation",
    phaseLabel: "Implementation & Ship",
    description: "Evaluates implementation readiness (PASS / CONCERNS / FAIL) across epics and creates/tracks sprint progress for automated dev iterations.",
    precededBy: "bmad-create-epics-and-stories",
    followedBy: "bmad-build",
    required: true,
    outputLocation: "implementation_artifacts",
    outputs: "Sprint status tracker (sprint_status.md)",
    tags: ["sprint-planning", "readiness-gate", "tracking", "status", "phase-4"],
    examples: [
      "/bmad-sprint-planning",
      "Check implementation readiness for Epic 3"
    ]
  },
  {
    id: "bmad-testarch-atdd",
    apf_supported: true,
    requires_sandbox: true,
    name: "Acceptance Test-Driven Scaffolding",
    code: "AT",
    module: "Test Architecture Enterprise",
    phase: "4-implementation",
    phaseLabel: "Implementation & Ship",
    description: "Generates red-phase failing acceptance test scaffolds before writing implementation code, enforcing the TDD quality cycle.",
    followedBy: "bmad-build",
    required: false,
    outputLocation: "test_artifacts",
    outputs: "Red-phase acceptance test files & ATDD checklist",
    tags: ["atdd", "tdd", "acceptance-tests", "red-green", "tea", "phase-4"],
    examples: [
      "/bmad-testarch-atdd \"Search lens documentation navigation\"",
      "Generate failing acceptance tests for user invitation endpoint"
    ]
  },
  {
    id: "bmad-build",
    apf_supported: true,
    requires_sandbox: true,
    name: "Build Loop (Intent to Code)",
    code: "BD",
    module: "BMad Method",
    phase: "4-implementation",
    phaseLabel: "Implementation & Ship",
    description: "Official implementation engine: clarifies intent, creates precise step-by-step plans, writes clean production code following repo conventions, verifies with tests, and presents structured walkthroughs.",
    precededBy: "bmad-sprint-planning or story definition",
    followedBy: "bmad-code-review",
    required: true,
    outputLocation: "implementation_artifacts",
    outputs: "Working code, unit test additions & walkthrough artifact",
    tags: ["build", "implementation", "coding", "tdd", "developer", "phase-4"],
    examples: [
      "/bmad-build \"Create docs screen with BMAD Reference\"",
      "Implement the OAuth2 callback handler with session token storage"
    ]
  },
  {
    id: "bmad-code-review",
    apf_supported: true,
    requires_sandbox: true,
    name: "Adversarial Code Review",
    code: "CR",
    module: "BMad Method",
    phase: "4-implementation",
    phaseLabel: "Implementation & Ship",
    description: "Adversarial multi-lens code review over any diff or pull request. Identifies logic flaws, concurrency hazards, security vulnerabilities, edge cases, and test gaps.",
    precededBy: "bmad-build",
    followedBy: "bmad-checkpoint-preview",
    required: false,
    outputLocation: "implementation_artifacts",
    outputs: "Code review report with severity-ranked triage list",
    tags: ["code-review", "adversarial", "security", "edge-cases", "quality", "phase-4"],
    examples: [
      "/bmad-code-review",
      "Run adversarial code review on the latest branch diff"
    ]
  },
  {
    id: "bmad-checkpoint-preview",
    apf_supported: true,
    requires_sandbox: true,
    name: "Human-in-the-Loop Checkpoint",
    code: "CK",
    module: "BMad Method",
    phase: "4-implementation",
    phaseLabel: "Implementation & Ship",
    description: "Guided walkthrough of a code change from high-level intent to granular details. Focuses human attention on critical decisions, risky paths, and live testing instructions.",
    precededBy: "bmad-code-review or bmad-build",
    required: false,
    outputLocation: "implementation_artifacts",
    outputs: "Checkpoint walkthrough summary with diff breakdown",
    tags: ["checkpoint", "preview", "human-review", "walkthrough", "phase-4"],
    examples: [
      "/bmad-checkpoint-preview",
      "Walk me through the authentication middleware rewrite"
    ]
  },
  {
    id: "bmad-qa-generate-e2e-tests",
    apf_supported: true,
    requires_sandbox: true,
    name: "QA Automated Test Suite Generation",
    code: "QA",
    module: "BMad Method",
    phase: "4-implementation",
    phaseLabel: "Implementation & Ship",
    description: "Generates robust end-to-end and integration automated tests covering user flows, failure states, network timeouts, and cross-browser rendering.",
    precededBy: "bmad-build",
    required: false,
    outputLocation: "implementation_artifacts",
    outputs: "E2E automated test suites & test run scripts",
    tags: ["qa", "e2e", "integration-tests", "playwright", "phase-4"],
    examples: [
      "/bmad-qa-generate-e2e-tests \"Docs search and navigation flow\"",
      "Create automated E2E tests for project creation and git sync"
    ]
  },
  {
    id: "bmad-retrospective",
    apf_supported: true,
    name: "Epic Retrospective & Verdict",
    code: "ER",
    module: "BMad Method",
    phase: "4-implementation",
    phaseLabel: "Implementation & Ship",
    description: "Evidence-based epic retrospective: collects completed deliverables, verifies requirements coverage against actual code, and renders an acceptance verdict.",
    precededBy: "bmad-code-review / Epic completion",
    required: false,
    outputLocation: "implementation_artifacts",
    outputs: "Retrospective report, metrics & lessons learned",
    tags: ["retrospective", "verdict", "post-mortem", "acceptance", "phase-4"],
    examples: [
      "/bmad-retrospective --epic 2",
      "Let's retro Epic 1: Multi-tenant database partitioning"
    ]
  },
  {
    id: "bmad-testarch-automate",
    apf_supported: true,
    requires_sandbox: true,
    name: "Automated Coverage Expansion",
    code: "TA",
    module: "Test Architecture Enterprise",
    phase: "4-implementation",
    phaseLabel: "Implementation & Ship",
    description: "Systematically increases test coverage across edge cases, error handlers, null checks, and boundary values to reach target quality thresholds.",
    precededBy: "bmad-testarch-atdd",
    required: false,
    outputLocation: "test_artifacts",
    outputs: "Expanded test suites & coverage delta reports",
    tags: ["test-coverage", "unit-tests", "edge-cases", "tea", "phase-4"],
    examples: [
      "/bmad-testarch-automate",
      "Expand unit test coverage for the AppIntegrationsList component"
    ]
  },
  {
    id: "bmad-testarch-nfr",
    apf_supported: true,
    requires_sandbox: true,
    name: "NFR Evidence Audit",
    code: "NR",
    module: "Test Architecture Enterprise",
    phase: "4-implementation",
    phaseLabel: "Implementation & Ship",
    description: "Audits non-functional requirements evidence including latency benchmarks, memory utilization, concurrent load handling, and security hardening.",
    required: false,
    outputLocation: "test_artifacts",
    outputs: "NFR audit report with compliance scorecard",
    tags: ["nfr", "performance", "security", "scalability", "tea", "phase-4"],
    examples: [
      "/bmad-testarch-nfr",
      "Audit latency and token consumption metrics against targets"
    ]
  },
  {
    id: "bmad-testarch-trace",
    apf_supported: true,
    requires_sandbox: true,
    name: "Traceability Matrix & Quality Gate",
    code: "TR",
    module: "Test Architecture Enterprise",
    phase: "4-implementation",
    phaseLabel: "Implementation & Ship",
    description: "Generates an end-to-end traceability matrix linking user stories, acceptance criteria, unit tests, and E2E suites with an objective release gate verdict.",
    required: false,
    outputLocation: "test_artifacts",
    outputs: "Traceability matrix (HTML/Markdown) & release gate decision",
    tags: ["traceability", "quality-gate", "matrix", "compliance", "tea", "phase-4"],
    examples: [
      "/bmad-testarch-trace",
      "Generate release traceability report for v1.2 release"
    ]
  },

  // ================= LEARNING & ACADEMY =================
  {
    id: "bmad-teach-me-testing",
    apf_supported: true,
    requires_sandbox: true,
    name: "Teach Me Testing (TEA Academy)",
    code: "TMT",
    module: "Test Architecture Enterprise",
    phase: "0-learning",
    phaseLabel: "Learning & Academy",
    description: "Interactive 7-session masterclass teaching modern software testing fundamentals, ATDD, test isolation, mocking patterns, and enterprise test strategies.",
    outputLocation: "test_artifacts",
    outputs: "Progress log, session notes & completion certificate",
    tags: ["education", "testing-academy", "tea", "learning", "mentorship", "phase-0"],
    examples: [
      "/bmad-teach-me-testing",
      "Start session 3 of the testing masterclass on ATDD"
    ]
  },

  // ================= ANYTIME & TOOLING =================
  {
    id: "bmad-party-mode",
    apf_supported: true,
    name: "Party Mode (Multi-Agent Roundtable)",
    code: "PM",
    module: "Core",
    phase: "anytime",
    phaseLabel: "Anytime & Tooling",
    description: "Orchestrates lively roundtables between installed BMAD personas (Architect, PM, Dev, TEA, UX Designer) or custom AI focus groups to evaluate proposals from multiple viewpoints.",
    outputLocation: "chat & optional summary artifact",
    outputs: "Synthesized multi-perspective consensus report",
    tags: ["party-mode", "roundtable", "personas", "multi-agent", "discussion", "anytime"],
    examples: [
      "/bmad-party-mode \"Should we migrate from REST to gRPC?\"",
      "Bring together Winston (Architect), John (PM), and Sally (UX) to debate this design"
    ]
  },
  {
    id: "bmad-deep-recon",
    apf_supported: true,
    name: "Deep Recon (Decision Research)",
    code: "RS",
    module: "Core",
    phase: "anytime",
    phaseLabel: "Anytime & Tooling",
    description: "Decision-grade research across market analysis, competitive landscapes, technical architectures, user voice, or academic literature with cited summaries and source verification.",
    args: "[type: market|domain|technical|competitive|user-voice|academic-lit]",
    outputLocation: "{planning_artifacts}/research",
    outputs: "Structured cited research briefing & executive summary",
    tags: ["research", "deep-recon", "competitive-analysis", "benchmarking", "anytime"],
    examples: [
      "/bmad-deep-recon technical \"Vector database performance under high QPS\"",
      "Run competitive research on agentic IDEs"
    ]
  },
  {
    id: "bmad-advanced-elicitation",
    apf_supported: true,
    name: "Advanced Elicitation",
    code: "AE",
    module: "Core",
    phase: "anytime",
    phaseLabel: "Anytime & Tooling",
    description: "Pushes any draft, architecture plan, or PRD beyond its initial version using structured techniques: pre-mortem, first principles, red team, or Socratic questioning.",
    outputLocation: "in-place refinement",
    outputs: "Hardened document with identified vulnerabilities addressed",
    tags: ["elicitation", "pre-mortem", "red-team", "socratic", "first-principles", "anytime"],
    examples: [
      "/bmad-advanced-elicitation pre-mortem",
      "Apply red team critique to our database migration plan"
    ]
  },
  {
    id: "bmad-review",
    apf_supported: true,
    name: "Multi-Lens Artifact Review",
    code: "RV",
    module: "Core",
    phase: "anytime",
    phaseLabel: "Anytime & Tooling",
    description: "Runs configurable review lenses over any document or diff: adversarial critique, edge-case hunting, verification gaps, structure reorganization, and prose polish.",
    args: "[path to artifact or diff]",
    outputLocation: "review report",
    outputs: "Findings JSON array & Markdown review summary",
    tags: ["review", "lenses", "critique", "structure", "prose", "anytime"],
    examples: [
      "/bmad-review docs/spec.md",
      "Run edge-case hunter lens on the authentication state machine"
    ]
  },
  {
    id: "bmad-customize",
    apf_supported: false,
    name: "BMad Customization Manager",
    code: "BC",
    module: "Core",
    phase: "anytime",
    phaseLabel: "Anytime & Tooling",
    description: "Authors and updates customization overrides for installed skills without manual TOML editing. Configures persistent facts, prompt hooks, and templates.",
    outputLocation: "{project-root}/_bmad/custom",
    outputs: "TOML override configuration files",
    tags: ["customize", "overrides", "configuration", "toml", "hooks", "anytime"],
    examples: [
      "/bmad-customize \"Add TypeScript strict mode guidelines to developer agent\"",
      "Customize the PRD template to include security review checklists"
    ]
  },
  {
    id: "bmad-agent-builder",
    apf_supported: false,
    name: "Agent Builder & Analyzer",
    code: "BA",
    module: "BMad Builder",
    phase: "anytime",
    phaseLabel: "Anytime & Tooling",
    description: "Conversational discovery engine to build new specialized AI agent skills, optimize prompt craft, analyze agent structure, and verify tool bindings.",
    action: "build-process | quality-analysis",
    args: "{-H: headless mode}|{description: agent concept}|{path: existing agent}",
    outputLocation: "bmad_builder_output_folder",
    outputs: "New Agent SKILL.md & quality scorecard",
    tags: ["agent-builder", "create-agent", "prompt-engineering", "builder", "anytime"],
    examples: [
      "/bmad-agent-builder \"Build a Database Migration Specialist Agent\"",
      "Analyze the developer agent prompt for quality improvements"
    ]
  },
  {
    id: "bmad-workflow-builder",
    apf_supported: false,
    name: "Workflow Builder & Converter",
    code: "BW",
    module: "BMad Builder",
    phase: "anytime",
    phaseLabel: "Anytime & Tooling",
    description: "Builds, edits, or analyzes automated multi-step workflows. Also converts legacy skills into BMad-compliant outcome-driven specifications with comparison reports.",
    action: "build-process | convert-process",
    outputLocation: "bmad_builder_output_folder",
    outputs: "Workflow skill definition & before/after report",
    tags: ["workflow-builder", "skills", "automation", "converter", "builder", "anytime"],
    examples: [
      "/bmad-workflow-builder \"Build an automated release tagging workflow\"",
      "Convert this custom prompt script into a BMad skill"
    ]
  },
  {
    id: "bmad-module-builder",
    apf_supported: false,
    name: "Module Builder & Packager",
    code: "MB",
    module: "BMad Builder",
    phase: "anytime",
    phaseLabel: "Anytime & Tooling",
    description: "Ideate, scaffold, and package full modular bundles of skills, prompts, and templates ready for distribution across developer teams.",
    outputLocation: "bmad_builder_output_folder",
    outputs: "Module package scaffold & registration manifests",
    tags: ["module-builder", "packaging", "distribution", "modules", "builder", "anytime"],
    examples: [
      "/bmad-module-builder",
      "Package our data engineering skills into an installable BMad module"
    ]
  },
  {
    id: "bmad-correct-course",
    apf_supported: true,
    name: "Correct Course (Sprint Re-alignment)",
    code: "CC",
    module: "BMad Method",
    phase: "anytime",
    phaseLabel: "Anytime & Tooling",
    description: "Navigate unexpected pivot points or roadblocks during execution. Evaluates impact, updates requirements, and proposes adjusted epics/stories without losing progress.",
    outputLocation: "planning_artifacts",
    outputs: "Course correction proposal & updated backlog",
    tags: ["course-correction", "pivot", "sprint-adjustment", "resilience", "anytime"],
    examples: [
      "/bmad-correct-course \"API v2 breaking change discovered in provider\"",
      "Propose sprint adjustments due to auth provider deprecation"
    ]
  },
  {
    id: "bmad-loop-sweep",
    apf_supported: true,
    requires_sandbox: true,
    name: "Loop Sweep & Deferred Work Triage",
    code: "ST",
    module: "BMAD Loop Skills",
    phase: "anytime",
    phaseLabel: "Anytime & Tooling",
    description: "Automated triage engine that scans the deferred-work ledger against the codebase to partition tasks into actionable bundles, resolved items, and human escalations.",
    outputLocation: "implementation_artifacts",
    outputs: "result.json triage plan & execution partition",
    tags: ["loop", "autonomous", "deferred-work", "sweep", "triage", "anytime"],
    examples: [
      "/bmad-loop-sweep",
      "Triage open deferred items from previous build cycles"
    ]
  }
]
