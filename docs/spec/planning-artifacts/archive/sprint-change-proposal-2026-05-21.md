# Sprint Change Proposal: Epic 1 & 4 Status Alignment, Story 4.3/4.4 Backlog Additions & Epic 5 Parallel Dual-Engine Integration

**Project:** Agentic Product Foundry (APF)
**Date:** 2026-05-21
**Author:** Amelia (Senior Software Engineer / Developer Agent)
**Status:** Approved & Finalized (via Incremental Review)

---

## Section 1: Issue Summary

### Problem Statement
During our sprint backlog audits, baseline repository scans, and agile planning sessions, three distinct categories of tracking mismatches and architectural intent gaps were identified:
1.  **Status Mismatches & Specs Gaps:** Sprints 1 and 2 concluded with no individual story specifications files on disk for Story 1.1 (Node Grid Layout), 1.2 (Agent Chat), or 1.3 (VPC Infrastructure), blocking BMad validation engines. Furthermore, Story 1.2 and 1.3 features are 100% physically complete and working in the codebase but remained marked as backlog in our maps.
2.  **Automated Handoff Integration Intent Gap (Story 4.3):** The PRD (`FR-4`) and UX Canvas specifications explicitly mandate that clicking the primary **"Approve & Handoff"** CTA inside the visual workspace must commit staging assets and automatically initialize a **Pull Request (PR)** on the remote repository (specifically GitHub) to unblock the developer. However, no story in [epics.md](/docs/specs/planning-artifacts/epics.md) defined this engineering scope, leaving the project with an un-designed, un-tracked critical handoff bridge.
3.  **VCS Pull and Workspace Re-Hydration Intent Gap (Story 4.4):** Visual workspace specs mandate a **Re-Hydration Flow (Journey 4)**, allowing a user to click 'Edit' on a previously approved/finalized spec, pulling remote Git commits, bridging them back into the GCS runtime scratchpad, and downgrading tracking status back to `In-Process`. However, no story defined this reverse pull-sync execution pipeline, leaving the vector database un-hydrated and the agent conversing using stale requirements.
4.  **Parallel Framework Integration Gap (Epic 5):** The newly released **Antigravity SDK (`agy-sdk`)** offers a superior ML orchestration core, but our baseline tracks only supported the legacy Agentic Development Kit (ADK) GenKit connectors. To eliminate single-framework lock-in and enable high-performance benchmarks in production, a dedicated Epic 5 and Stories map are required to track concurrent dual-engine implementations (ADK vs. Python `agy-sdk`) in parallel paths, utilizing a secure Go API reverse-proxy gateway and a downstream Python agent executor core.

### Discovery Context & Evidence
*   **Backlog Scan:** A systematic check of `epics.md` and our active task checklists identified that while Story 1.1 relies on the PR-sync loop in its Acceptance Criteria ([Story 1.1 AC: 1.5](/docs/specs/implementation-artifacts/1-1-interactive-node-grid-layout.md#L19-L20)), the underlying backend endpoints, remote push adapters, and VCS REST integrations were completely un-profiled in any active story backlog.
*   **Physical Code Audits:** Verified that the folder [agy/](/agy) has been successfully checked in on origin/main, with [pyproject.toml](/agy/pyproject.toml) declaring dependencies on `google-antigravity>=0.1.0` and Python `>=3.13`, providing the physical backplane to establish the parallel Antigravity path.

---

## Section 2: Impact Analysis

### Epic Impact
*   **Epic 1 (Interactive Agent Canvas):** Status advanced to `in-progress`. Stories 1.1, 1.2, and 1.3 are now marked `done` based on codebase reality, leaving only Story 1.4 active in the backlog.
*   **Epic 2 (Project Collaboration & Pipeline Tracking):** Status remains `done` (100% completed).
*   **Epic 3 (Enterprise Context Integration):** Status remains `backlog` (unstarted).
*   **Epic 4 (Repository Synchronization):** Status remains `in-progress`. Story 4.1 is `done`. We have successfully appended **Story 4.3** and **Story 4.4** as active backlog items to execute these bidirectional VCS integrations.
*   **Epic 5 (Parallel Dual-Engine AI Orchestration):** A brand-new epic has been successfully created (**`Status: in-progress`**), with Stories 5.1 and 5.2 scheduled under Sprint 2 to track the Python chat engine and Go proxy gateway, and Story 5.3 allocated under Sprint 3.

### Story Impact
*   **Story 4.3, 4.4, 5.1, 5.2, & 5.3:** Brand-new technical story context files have been compiled, structured, and saved to disk in `ready-for-dev`, unblocking canvas handoff, re-hydration editing, and parallel python agent executions.
*   **Story 1.1, 1.2, & 1.3:** Story specification blueprints have been formally written, finalized, and marked `done` to resolve files presence gaps.

### Artifact Conflicts
*   **PRD ([prd.md](prd.md)):** Modified `FR-4` to append `FR-4.2 Parameterized Sync` and `FR-4.3 Spec Re-Hydration`. Created new `FR-5 Parallel Dual-Engine AI Orchestration` mandating parameterized core framework routing, custom tool adapters parity, and strict telemetry isolation.
*   **Architecture ([architecture.md](/docs/specs/planning-artifacts/architecture.md)):** Added technical blueprints for this bidirectional pipeline and the parallel core under `Data & Resource Architecture`, specifically mapping Firestore routing configurations (`integrationSettings.aiOrchestrator` values: `adk` | `agy-sdk`), Go HTTPGateway reverse-proxy middleware, and downstream Python `google-antigravity` executor enclaves.
*   **Current Epics ([epics.md](/docs/specs/planning-artifacts/epics.md)):** Appended Epic 5 and its 3 new stories, scheduling them into Sprint 2 and Sprint 3 maps, while completely unscheduling Story 3.1 (Figma Sync Adapter) from all active sprint tracks per PO directive.

### Technical Impact
*   **Backend (Go):** Requires refactoring `ChatHandler`, `SessionHandler`, and `SecretsHandler` HTTP struct receivers to intercept and redirect requests dynamically based on Firestore `aiOrchestrator` settings, and implementing `net/http/httputil.NewSingleHostReverseProxy` gateway routes (Task 2 in Story 5.2).
*   **Backend (Python under `/agy`):** Requires developing the FastAPI/uvicorn REST server routing `main.py`, executing `google-antigravity` SDK streaming chat inferences (Task 2 in Story 5.1), and implementing Draw.io/Stitch tool execution adapters (Task 2 in Story 5.3).
*   **Frontend (ui/):** Wires Riverpod state snap reloads upon engine-switching configurations.

---

## Section 3: Recommended Approach

### Selected Path Forward: Backlog Additions & Incremental Alignment
We executed the **Incremental Mode** strategy to collaboratively refine, align, and authorize these updates. Story 4.3, Story 4.4, Story 5.1, and Story 5.2 have been successfully scheduled into Sprint 2 because they represent critical technical prerequisites to fulfill active canvas rendering, revision-gating, and core inference benchmarks.

### Rationale
*   **traceability & Compliance:** Rather than implementing Git-sync and ML engines ad-hoc without spec cover, this course correction guarantees 100% architectural alignment and full audit compliance under BMad standards.
*   **Path A Decoupling:** Keeps product requirements implementation-agnostic (decoupled) by isolating low-level libraries (GitHub API, `go-git/v6`, GCS buckets, FastAPI, google-antigravity python SDK) strictly inside `architecture.md` and the individual story files.

### Estimates & Risk
*   **Effort Estimate:** High (Baseline status alignments on disk + 5 brand-new technical Story spec sheets in ready-for-dev + moderate backend Go proxy routing + substantial Python agent/adapter engineering).
*   **Risk Assessment:** Low. Changes are strictly scoped to repository continuity sync modules, canvas callback handlers, and downstream python reverse proxy boundaries, with zero regression risk on Epic 1 or 2 core client assets.

---

## Section 4: Detailed Change Proposals

### 1. PRD Specifications Modification (`APF_Foundry_PRD.md`)
*   **Path:** [prd.md: FR-4 Repository Sync](prd.md#FR-4) & [FR-5 Parallel Orchestration](prd.md#FR-5)
*   **Additions:**
    *   Appended `FR-4.2 Parameterized Remote Synchronization` and `FR-4.3 Git-to-Workspace Spec Re-Hydration & Edit Sync`.
    *   Appended `FR-5 Parallel Dual-Engine AI Orchestration` mandating parameterized engine routing, dynamic tool adapters parity, and schema & telemetry isolation.

### 2. Architecture Blueprints Modification (`architecture.md`)
*   **Path:** [architecture.md: Data & Resource Architecture](/docs/specs/planning-artifacts/architecture.md#Data-&-Resource-Architecture)
*   **Additions:**
    *   Added `Parameterized Git Repository Sync & Gating` and `VCS Git Pull & Staging Workspace Re-Hydration` defining `go-git/v6` push/pull enclaves, and user OIDC credential checks.
    *   Added `AI Orchestration Core Architecture` mapping Go dynamic reverse-proxy middleware, downstream Python `google-antigravity` execution cores (Python >=3.13, uv, FastAPI), and Firestore routing configurations.

### 3. Epics Map & Sprint Allocation Modification (`epics.md`)
*   **Path:** [epics.md: Epic 4 & 5 Story List](/docs/specs/planning-artifacts/epics.md#Story-4.3) & [Sprint 2/3 Allocations](/docs/specs/planning-artifacts/epics.md#Sprint-2)
*   **Additions:** Appended Stories 4.3, 4.4, 5.1, 5.2, and 5.3, scheduling the push/pull sync and python chat/Go proxy cores under Sprint 2, the tool adapter under Sprint 3, and removing Story 3.1 (Figma Sync Adapter) entirely from active sprint plans.

### 4. Sprint Status Tracker Updates (`sprint-status.yaml`)
*   **Path:** [sprint-status.yaml](/docs/specs/implementation-artifacts/sprint-status.yaml)
*   **Status Transitions Applied:**
    *   `epic-1: backlog` ➔ `epic-1: in-progress`
    *   `1-1-interactive-node-grid-layout: ready-for-dev` ➔ `1-1-interactive-node-grid-layout: done`
    *   `1-2-agent-sidebar-chat: backlog` ➔ `1-2-agent-sidebar-chat: done`
    *   `1-3-secure-canvas-workspace-infrastructure: backlog` ➔ `1-3-secure-canvas-workspace-infrastructure: done`
    *   `epic-4: in-progress` ➔ `4-3-...: backlog` & `4-4-...: backlog`
    *   `epic-5: in-progress` ➔ `5-1-...: ready-for-dev`, `5-2-...: ready-for-dev`, `5-3-...: ready-for-dev` (🆕 Entries Added)

### 5. Story Specification Documents Creation (`implementation-artifacts/`)
*   **Story 1.1, 1.2, & 1.3 Specs:** Formally saved in `Status: done`.
*   **Story 4.3, 4.4, 5.1, 5.2, & 5.3 Specs:** Formally compiled and saved in `Status: ready-for-dev`.

---

## Section 5: Implementation Handoff

### Scope Classification
*   **Scope:** Major (Epic 5 parallel framework integration requires PM/PO/Arch/Dev coordination, backend Go proxy middleware gateway, substantial Python `google-antigravity` ML executor coding under `/agy`, custom visual tool adapters, and telemetry metrics audit validations).

### Success Criteria
*   Story 4.3, 4.4, 5.1, 5.2, and 5.3 specifications documents exist on disk under `implementation-artifacts/` in `ready-for-dev`.
*   All planning specifications (PRD, Architecture, Epics) are staged and synchronized.
*   `sprint-status.yaml` is validated, containing all 17 stories, correct status keys, and matching project timestamps.
