# Story 4.4: Git-to-Workspace Spec Re-Hydration and Edit Sync

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a User (PM, UX Designer, Architect, Developer),
I want to re-hydrate and edit a finalized specification by dynamically pulling the latest Git commits back into my workspace,
so that I can fluidly iterate and refine approved requirements through natural language agent dialogues without manual copy-paste steps.

## Acceptance Criteria

1. **Given** a user clicks "Edit" on a finalized specification in the visual workspace dashboard, **When** the edit session is initialized, **Then** the backend MUST pull the latest commit from the remote VCS branch. (AC: 4.4.1)
2. **Given** the remote Git pull operation, **When** authenticating against the remote VCS, **Then** the system MUST strictly propagate the authenticated human user's OIDC/OAuth identity claims context to authorize read operations, blocking context retrievals beyond their specific repository access boundaries. (AC: 4.4.2)
3. **Given** a successful branch pull, **When** raw assets (Markdown specs under `docs/spec/` or visual diagrams JSON) are retrieved, **Then** the orchestrator MUST automatically stage and decode these files directly into the project's ephemeral GCS scratchpad path (matching the schema `gs://apf-scratchpad/projects/{projectId}/...`) and vector database context. (AC: 4.4.3)
4. **Given** the raw assets are successfully staged in GCS, **When** the re-hydration completes, **Then** the orchestrator MUST dynamically transition the target artifact's tracking status back to `In Progress` inside our Firestore databases. (AC: 4.4.4)
5. **Given** the workspace status transition, **When** re-hydration completes successfully, **Then** the React client UI MUST automatically reload active TanStack Query state models, refresh the adjacent `CanvasPane` and `ChatPane` panels dynamically, and display success toast messages in under **1.0 second**. (AC: 4.4.5)

---

## Tasks / Subtasks

- [ ] **Task 1: Implement the Git Pull/Re-Hydration REST Endpoint in Go (AC: 4.4.1, 4.4.2)**
  - [ ] **Subtask 1.1:** Create `POST /api/v1/projects/{id}/git/pull` route handler inside [api/projects.go](api/projects.go) or `api/workspace.go`.
  - [ ] **Subtask 1.2:** Register the new dynamic Pull route and its preflight configurations in [api/routes.go](api/routes.go).
  - [ ] **Subtask 1.3:** Enforce strict OAuth/OIDC claims validation, bubbling up invalid identity errors cleanly.
- [ ] **Task 2: Build Remote Branch Pulls & Staging Re-Hydration via `go-git/v6` (AC: 4.4.1, 4.4.3)**
  - [ ] **Subtask 2.1:** Configure `PullBranch` enclaves in [api/workspace.go](api/workspace.go) utilising go-git/v6 client APIs.
  - [ ] **Subtask 2.2:** Stage and copy pulled raw markdown/JSON specifications into the project GCS bucket path (schema: `gs://apf-scratchpad/projects/{projectId}/...`).
  - [ ] **Subtask 2.3:** Enforce context scrub protocols using Model Armor DLP handlers on all pulled specs before indexing.
- [ ] **Task 3: Dynamic State Transition & Semantic Memory Vector DB Re-Indexing (AC: 4.4.4)**
  - [ ] **Subtask 3.1:** Update the active Firestore tracking state: downgrade the target artifact's `status` to `In Progress` within active databases.
  - [ ] **Subtask 3.2:** Execute Vector DB re-hydration: trigger backend GenKit indexers to re-absorb the newly pulled spec baseline, refreshing the agent's semantic context under 5 minutes.
- [ ] **Task 4: Integrate Next.js React Staging Reload & Layout Visualizers (AC: 4.4.5)**
  - [ ] **Subtask 4.1:** Hook dynamic workspace edit triggers in [ui/src/app/(dashboard)/projects/[id]/workspace/_hooks/useWorkspace.ts](/ui/src/app/(dashboard)/projects/[id]/workspace/_hooks/useWorkspace.ts) to target `POST /api/v1/projects/{projectId}/git/pull`.
  - [ ] **Subtask 4.2:** Integrate progressive loading banners and trigger instant, synchronous client reloads of both `CanvasPane` and `ChatPane` in under 1.0 second.
- [ ] **Task 5: Complete Verification Testing (AC: 4.4.1 - 4.4.5)**
  - [ ] **Subtask 5.1:** Write integration test suites in `api/workspace_pull_test.go` verifying mock branch pulls, decryption resets, and Vector DB ingest triggers.
  - [ ] **Subtask 5.2:** Verify TypeScript compiles and run workspace validation checks.

---

## Dev Notes

### 🏗️ 1. Architectural Guardrails Complied
*   **ADC & propagated User Token Delegation:** Pull pipelines MUST strictly propagate the authenticated human user's OIDC context to authorize VCS pulls, ensuring a zero-trust access perimeter. Standalone agent git-pull keys are strictly prohibited.
*   **go-git/v6 and Zero Raw Shell Commands:** All branch retrievals and pulls MUST use native `go-git/v6` enclaves. Raw shell commands (e.g. `exec("git pull")`) are banned under baseline governance guidelines.
*   **RAG Context Recalibration:** Upon pulling files, backend indexers must update semantic database embeddings immediately. If RAG is not re-calibrated, the active agent will converse using stale, pre-edit requirements context, violating operational truth.

---

## 🛠️ Project Structure & Source Maps

### 1. Target Files Matrix

| File Path | Status | Operational Role |
| :--- | :--- | :--- |
| [api/workspace.go](api/workspace.go) | `UPDATE` | Implement pull-sync routines using go-git/v6; stage files into target GCS paths. |
| [api/routes.go](api/routes.go) | `UPDATE` | Register the Git pull-sync / Re-Hydration REST API endpoints. |
| [ui/src/app/(dashboard)/projects/[id]/workspace/_hooks/useWorkspace.ts](/ui/src/app/(dashboard)/projects/[id]/workspace/_hooks/useWorkspace.ts) | `UPDATE` | Bind the visual "Edit Approved Spec" CTA to query backend pull routes, staging reloading spinners, and reloading state snapshots. |
| `api/workspace_pull_test.go` | `NEW` | Go backend integration tests verifying pull routines, decrypted staging, and Firestore transitions. |

---

## 📚 References & Traceability Links

*   **Requirements Origin:** [prd.md: FR-4.3 Git pull re-hydration](../planning-artifacts/prd.md#FR-4.3).
*   **Design Blueprints:** [ux-design-specification.md: Re-Hydration Flow Journey 4](../planning-artifacts/ux-design-specification.md#Journey-4:-Re-Hydration-(Editing-Approved-Artifacts)).
*   **Technical Design:** [architecture.md: Git Pull & Staging re-hydration decisions](../planning-artifacts/architecture.md#VCS-Git-Pull-&-Staging-Workspace-Re-Hydration).

---

## Dev Agent Record

### Agent Model Used

Gemini Next

### Debug Log References

### Completion Notes List

### File List
