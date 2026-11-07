# Story 4.3: Automated Pull Request Handoff and Remote VCS Integration

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a Product Manager / Stakeholder,
I want the system to dynamically sync local specifications branches with the remote VCS repository and auto-create a Pull Request,
so that approved design artifacts and requirements are cleanly staged for development under standard peer-review governance.

## Acceptance Criteria

1. **Given** a user triggers the "Approve & Handoff" action on the visual canvas popup, **When** the repository synchronization endpoint is queried, **Then** the backend MUST push the staged local specifications (under `design-artifacts/` or `docs/spec/`) to the remote Git repository. (AC: 4.3.1)
2. **Given** the remote synchronization executes, **When** preparing remote staging, **Then** the orchestrator MUST dynamically create and check out a dedicated remote feature branch matching the schema `artifact-sync-{projectId}-{timestamp}` and push staged changes to it. (AC: 4.3.2)
3. **Given** a successful feature branch push, **When** the branch is remote, **Then** the orchestrator MUST automatically trigger the remote VCS API (specifically **GitHub REST API v3** wrappers) to initialize a standard Pull Request against the target primary branch. (AC: 4.3.3)
4. **Given** the remote Git push and Pull Request operations, **When** authenticating against the remote repository and VCS APIs, **Then** the system MUST strictly propagate the authenticated human user's OIDC/OAuth identity token from the request context to prove write authorizations, blocking mutations beyond their specific repository access boundaries. (AC: 4.3.5)
5. **Given** the user triggers the handoff visual flow in the UI, **When** the synchronization is running, **Then** the platform MUST cleanly render non-intrusive progressive loading indicators, and display connection confirmation overlays or detailed error diagnostics upon completion via Snackbars. (AC: 4.3.6)

---

## Tasks / Subtasks

- [ ] **Task 1: Implement Dynamic Git Sync REST API Route in Go (AC: 4.3.1, 4.3.5)**
  - [ ] **Subtask 1.1:** Register `POST /api/v1/projects/{id}/git/sync` and its preflight options handler in [api/routes.go](api/routes.go).
  - [ ] **Subtask 1.2:** Extract request body JSON `{"filePath": "..."}` and verify user identity claims from context.
  - [ ] **Subtask 1.3:** Build hierarchical credential resolver (Project PAT $\rightarrow$ user global OAuth $\rightarrow$ system environment fallback) to select the user's active access token.
  - [ ] **Subtask 1.4:** Parse owner/repo from `project.RepoUrl` with strict validation enforcing HTTPS-only `github.com` hostnames.
- [ ] **Task 2: Verify Remote Feature Branch & Graceful Errors (AC: 4.3.2, 4.3.5)**
  - [ ] **Subtask 2.1:** Integrate checks within `gitSyncHandler` utilizing standard VCS client calls or lightweight HTTP fetches to verify the target feature branch exists in the remote GitHub repository.
  - [ ] **Subtask 2.2:** If the remote branch is not yet created or pushed, fail gracefully by returning a JSON payload with status `400 Bad Request` and a detailed diagnostic message informing the user that the AI agent's branch push is pending.
  - [ ] **Subtask 2.3:** Ensure all REST and SSE response streams are handled safely without bleeding raw network exceptions.
- [ ] **Task 3: Automate remote GitHub Pull Request Initialization (AC: 4.3.3, 4.3.5)**
  - [ ] **Subtask 3.1:** Create HTTP client wrapper `GitHubPRClient` in `api/integrations/github_pr.go` to invoke `POST /repos/{owner}/{repo}/pulls` with user auth headers.
  - [ ] **Subtask 3.2:** Dynamically resolve the target base branch (project planning branch $\rightarrow$ default config $\rightarrow$ `"main"` fallback).
  - [ ] **Subtask 3.3:** Compose standard PR Title (`spec(sync): approve and stage {fileName}`) and rich markdown description reporting user, timestamp, file paths, and automated instructions.
  - [ ] **Subtask 3.4:** Look up target database `Artifact` record and update status to `'Staged for Review'` upon successful PR creation, returning the link details to the client.
- [ ] **Task 4: Integrate Next.js "Approve & Sync" Visual Sync Loop (AC: 4.3.6)**
  - [ ] **Subtask 4.1:** Refactor `CanvasPane.tsx` inside [ui/src/components/workspace/CanvasPane.tsx](ui/src/components/workspace/CanvasPane.tsx) to correctly trigger `gitSyncMutation` when the "Approve & Sync" button is clicked.
  - [ ] **Subtask 4.2:** Track syncing states and display the Loader2 animation and inline linear progress bar loader overlay when `isSyncing` is true.
  - [ ] **Subtask 4.3:** Connect green success toast messages utilizing the `toast.success` action block from `sonner`, presenting a dynamic "Open PR" button linked to the returned `prUrl`.
  - [ ] **Subtask 4.4:** Enforce dynamic redirection: if sync returns status `401` or `403`, display credentials validation error toast with a custom callback redirect button pointing to the project details settings screen (`/projects/${projectId}/details`).
- [ ] **Task 5: Complete Verification Testing (AC: 4.3.1 - 4.3.6)**
  - [ ] **Subtask 5.1:** Write unit and integration tests in `api/workspace_sync_test.go` checking Git direct push paths, gated PR-switching branches, defer checkout recovery states, and mock PR calls.
  - [ ] **Subtask 5.2:** Run standard linting and formatting suites: verify TypeScript compiling and run workspace validation checks.

### Review Findings

- [x] [Review][Patch] Verify Remote Branch existence and Fail Gracefully in `gitSyncHandler` [api/workspace.go:412] — Verify if the target remote branch has been created and pushed by the AI tier, returning a clear graceful error response if the branch is missing rather than allowing a raw API error.

---

## Dev Notes

### 🏗️ 1. Architectural Guardrails & Socratic Agreements
*   **Scoped Path Operations:** Actions are strictly target-scoped to the active file path relative to the clone directory, rather than bulk project path additions.
*   **Hierarchy-Driven Security Context:** Authentication token resolution follows a user-first strict hierarchy. Sensitive pushes and API events must utilize project-level PAT or global OAuth user-authenticated structures, preventing system default privileges from performing unauthorized user Git operations.
*   **VCS Format Constraints:** URL checks strictly allow only standard HTTPS `github.com` repository endpoints (SSH host patterns are blocked to guarantee parse predictability).
*   **State Switch-Back Resilience:** The local workspace replica checkout MUST wrap branch transitions inside target Go `defer` blocks to keep the local working directories in a safe, known tracking state at all times.
*   **Actionable Toast Redirections:** UI error messaging for credential validation blocks maps directly to custom navigation actions that close visual popups and open credentials configurations automatically.

---

## 🛠️ Project Structure & Source Maps

### 1. Source Files Matrix

| File Path | Status | Operational Role |
| :--- | :--- | :--- |
| [api/workspace.go](api/workspace.go) | `UPDATE` | Implement `gitSyncHandler` with dynamic git-push, dynamic signatures, and defer checkout recovery logic. |
| [api/routes.go](api/routes.go) | `UPDATE` | Register sync REST endpoint path (`POST /api/v1/projects/{id}/git/sync`) and preflight rules. |
| [api/integrations/github_pr.go](api/integrations/github_pr.go) | `UPDATE` | Lightweight GitHub REST API client wrapper creating Pull Requests with native Go HTTP structures. |
| [ui/src/components/workspace/CanvasPane.tsx](ui/src/components/workspace/CanvasPane.tsx) | `UPDATE` | Update dialog state, linear progressive overlays, action spinners, "Open PR" toast triggers, and settings redirections. |
| `api/workspace_sync_test.go` | `NEW` | Suite verifying direct checkouts vs gated branch pushes, mock REST APIs, and defer worktree state resets. |

---

## Dev Agent Record

### Agent Model Used

Gemini Next

### Completion Notes List

*   Refactored `PhaseOutcomeArtifacts` in `ui/lib/screens/workspace/widgets/artifacts.dart` to fully support themes and high-density compact line-item layouts matching Chat Sessions height and text sizing (12 font, 8.0/4.0 vertical paddings, primary folder accents). Completed with 0 warnings.

