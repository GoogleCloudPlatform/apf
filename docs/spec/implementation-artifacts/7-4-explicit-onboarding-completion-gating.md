# Story 7.4: Explicit Onboarding Completion Gating

Status: in-progress

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a PM,
I want an explicit validation gate to complete the onboarding phase,
so that my workspace tools are unlocked only when the project is fully calibrated and versioned.

## Acceptance Criteria

1. **Given** the PM has finalized their file ingestion selections in the Onboarding Tab, **When** they click the primary action button `"Complete Onboarding"`, **Then** the system MUST transition the project status to `Onboarded` in the Firestore database. (AC: 7.4.1)
2. **Given** the onboarding completion trigger, **When** running, **Then** the backend orchestrator MUST automatically generate a canonical `project-context.md` file outlining:
   - Discovered core technical runtime stacks (e.g., Go, Node) and frameworks.
   - List of mapped legacy requirements and specification file paths chosen for ingestion. (AC: 7.4.2)
3. **Given** the generated `project-context.md`, **When** saved, **Then** the backend MUST write the file to the project's workspace filesystem and commit/push it directly to the repository's tracking Git branch (using the authenticated human user's OAuth/OIDC credentials). (AC: 7.4.3)
4. **Given** a project status transition to `Onboarded`, **When** returning to the Projects Dashboard, **Then** the tile action button for this project MUST transition from `"Onboard"` to `"Open Workspace"` (or `"Launch"` / standard workspace launch CTA). (AC: 7.4.4)
5. **Given** the workspace view, **When** the project status transitions to `Onboarded`, **Then** the system MUST immediately unlock all active visual canvas tabs, right-hand persona menus, and sidebar chat tools, permitting standard active sprint operations. (AC: 7.4.5)

## Tasks / Subtasks

- [ ] **Task 1: Backend Onboarding Complete API Endpoint (AC: 7.4.1, 7.4.2, 7.4.3)**
  - [ ] Define the completion endpoint route: `POST /api/v1/projects/{id}/onboard/complete` in `api/routes.go` (protected by `authMiddleware`).
  - [ ] Implement `completeOnboardingHandler` in `api/onboarding.go`.
  - [ ] In `completeOnboardingHandler`:
    - Validate human user credentials and OIDC token.
    - Fetch active project record from Firestore using `store.GetProject`.
    - Retrieve active onboarding selections and stack discovery details from Firestore (e.g., `projects/{id}/onboarding/discovery`).
    - Generate the markdown payload for `project-context.md` using a standard Go template (`text/template`).
    - Save `project-context.md` to the project workspace (using `core.ArtifactService`).
    - Register a new database `db.Artifact` entry for `project-context.md` with status = `"Complete"`, type = `"project_context"`, phase = `"ANALYSIS"`.
    - Commit and push `project-context.md` to the remote repository branch using the user's OAuth credentials (similar to the git commit sync flow in `gitSyncHandler` using `go-git/v6`).
    - Update `project.Status = "Onboarded"` and save changes via `store.UpdateProject`.
    - Broadcast a global project activity stream message: `"Project onboarding completed successfully! All active workspace tools are now unlocked."`
    - Respond with `200 OK` containing the updated project profile.

- [ ] **Task 2: Frontend "Complete Onboarding" Trigger & GUI State Transitions (AC: 7.4.1, 7.4.5)**
  - [ ] In `OnboardingPanel.tsx`:
    - Add a click handler to the `"Complete Onboarding"` primary button.
    - Upon click, fire `POST /api/v1/projects/[id]/onboard/complete`.
    - Show a smooth loading spinner while the backend generates specs, commits to Git, and transitions status.
    - Once successful, fire a workspace state refresh or reload so `project.status` is updated to `"Onboarded"`.
  - [ ] Update `useWorkspace.ts` and `WorkspaceHeader.tsx`:
    - Verify that once `project.status === "Onboarded"`, the onboarding tab is disabled/hidden, and the active phase dynamically transitions to the first active phase (e.g. `"ANALYSIS"`).
    - Verify that all visual canvas panels, sidebar chat, and collapsed persona rails dynamically unlock and become fully interactive.

- [ ] **Task 3: Dashboard CTA Transition (AC: 7.4.4)**
  - [ ] Update `ProjectCard.tsx` to verify that if `project.status === "Onboarded"`, the card CTA button renders `"Open Workspace"` (or `"Launch"`) and routes directly to standard workspace navigation without gating to the onboarding view.

- [ ] **Task 4: End-to-End Verification & Unit Tests (AC: 7.4.1 - 7.4.5)**
  - [ ] Add integration tests in `api/onboarding_test.go`:
    - Assert that `POST /api/v1/projects/{id}/onboard/complete` transitions project status to `"Onboarded"`, correctly generates `project-context.md`, and successfully triggers git commit logic.
    - Verify error scenarios (e.g. repository write permissions errors, Firestore database failures) are handled gracefully.
  - [ ] Add Vitest tests for the React Next.js frontend:
    - Assert that `ProjectCard.tsx` correctly transitions CTA display labels based on the `"Onboarded"` state.
    - Assert that the onboarding panel fires the correct POST request and reloads the workspace layout upon success.

## Dev Notes

### 🏗️ 1. Architectural Guardrails
*   **Centralized Status Mutex:** Never allow the client to bypass onboarding completion. Gating MUST be centrally enforced on both frontend routing hooks and backend API handlers by validating project status before permitting standard chat/canvas mutations.
*   **OAuth/OIDC Token Propagation:** To adhere to Zero-Trust architectures, git commits and push operations MUST propagate the caller's OIDC identity context, mathematically proving the platform has no write privileges beyond user bounds.
*   **Template Generation Safety:** Ensure `project-context.md` generation is robust. If stack discovery was skipped or failed, the template should gracefully output a minimal configuration structure without panicking.

### 🛠️ Project Structure & Target Files Matrix

| File Path | Status | Operational Role |
| :--- | :--- | :--- |
| [api/routes.go](api/routes.go) | `UPDATE` | Register `POST /api/v1/projects/{id}/onboard/complete` completion route. |
| [api/onboarding.go](api/onboarding.go) | `UPDATE` | Implement `completeOnboardingHandler` defining template compilation, Git staging, commits, and Firestore status transitions. |
| [ui/src/components/project/ProjectCard.tsx](ui/src/components/project/ProjectCard.tsx) | `UPDATE` | Render `"Open Workspace"` instead of `"Onboard"` once status is `"Onboarded"`. |
| [ui/src/app/(dashboard)/projects/[id]/workspace/_components/OnboardingPanel.tsx](ui/src/app/(dashboard)/projects/[id]/workspace/_components/OnboardingPanel.tsx) | `UPDATE` | Bind click handlers to the `"Complete Onboarding"` CTA and reload active workspace states. |
| [ui/src/app/(dashboard)/projects/[id]/workspace/_hooks/useWorkspace.ts](ui/src/app/(dashboard)/projects/[id]/workspace/_hooks/useWorkspace.ts) | `UPDATE` | Unlock all phases and standard tools when `project.status === "Onboarded"`. |

### References

- [PRD Onboarding Completion Gating](docs/spec/planning-artifacts/prd.md#L105-L106)
- [PRD FR-7.5 Explicit Onboarding Completion Gating](docs/spec/planning-artifacts/prd.md#L154-L156)
- [Epics Story 7.4 Acceptance Criteria](docs/spec/planning-artifacts/epics.md#L441-L456)
- [Go-Git Commit Syncer Flow Reference](api/workspace.go#L412-L565)

## Dev Agent Record

### Agent Model Used

Gemini Next

### Debug Log References

### Completion Notes List

### File List
