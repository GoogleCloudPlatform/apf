# Story 4.11: Structured Git Sync Response & GitHub/GitLab Integration

Status: ready-for-dev

## Story

As an API Consumer or Frontend Client,
I want the workspace Git sync handlers (`POST /api/v1/projects/{id}/git/pull` and `POST /api/v1/projects/{id}/git/sync`) to return a structured checklist response containing dynamic branch names, step progress, and GitHub PR resolution links,
so that the user interface can display progress and guide users to resolve conflicts via GitHub/GitLab PRs or local IDE workflows.

## Acceptance Criteria

1. **Given** a call to `POST /api/v1/projects/{id}/git/pull` or `/git/sync`, **When** the operation finishes, **Then** the handler MUST return HTTP `200 OK` with a structured `GitSyncChecklistResponse` payload. (AC: 4.11.1)
2. **Given** a `GitSyncChecklistResponse` payload, **When** serialized, **Then** it MUST contain `status` (`SUCCESS` | `CONFLICT_DETECTED` | `ERROR`), `message`, `dynamicBranches` (`baseBranch`, `specBranch`, `backupRef`, `shadowBranch`), and an array of `steps` (`id`, `label`, `status`, `detail`). (AC: 4.11.2)
3. **Given** a merge conflict occurs (`status: "CONFLICT_DETECTED"`), **When** a GitHub/GitLab integration is configured for the project, **Then** the system MUST ensure local commits are pushed to the spec branch and populate `conflictInfo.pullRequestUrl` with the direct GitHub/GitLab PR URL. (AC: 4.11.3)
4. **Given** a merge conflict occurs, **When** constructing `conflictInfo`, **Then** the system MUST include `conflictingFiles`, `cliInstructions`, and a clear `nextStepRecommendation` string. (AC: 4.11.4)
5. **Given** a call to `GET /api/v1/projects/{id}/git/local-changes`, **When** calculating changes to be staged or synced, **Then** `GetWorkspaceStatus` MUST first perform a `git pull` / synchronization on the `main` (`baseBranch`) branch so that `main` is synchronized with the remote, and MUST return BOTH uncommitted local worktree modifications (from `git status --porcelain=v1`) AND files modified in committed commits on `specBranch` that are not yet in `baseBranch` (from `git diff --name-status refs/heads/<baseBranch>...refs/heads/<specBranch>`), deduplicating by file path. (AC: 4.11.5)

---

## Tasks / Subtasks

- [ ] **Task 1: Define `GitSyncChecklistResponse` Structs in `api/workspace/git_client.go` or `workspace.go` (AC: 4.11.1, 4.11.2)**
  - [ ] **Subtask 1.1:** Create `GitSyncChecklistStep` struct (`ID`, `Label`, `Status`, `Detail`).
  - [ ] **Subtask 1.2:** Create `DynamicBranchInfo` struct (`BaseBranch`, `SpecBranch`, `BackupRef`, `ShadowBranch`).
  - [ ] **Subtask 1.3:** Create `ConflictResolution` struct (`ConflictingFiles`, `PullRequestURL`, `CLIInstructions`, `NextStepRecommendation`).
  - [ ] **Subtask 1.4:** Create `GitSyncChecklistResponse` top-level response struct.
- [ ] **Task 2: Build GitHub/GitLab PR URL Resolution Helper in `api/integrations/github.go` (AC: 4.11.3)**
  - [ ] **Subtask 2.1:** Implement `GetOrOpenPullRequestURL(ctx, orgID, userId, projectId, specBranch, baseBranch)`.
  - [ ] **Subtask 2.2:** Pushes spec branch if necessary and returns web URL of existing or newly opened PR/MR.
- [ ] **Task 3: Refactor `GitPullHandler` and `GitSyncHandler` in `api/workspace/workspace.go` (AC: 4.11.1, 4.11.4)**
  - [ ] **Subtask 3.1:** Map internal 3-stage execution steps to `GitSyncChecklistStep` items.
  - [ ] **Subtask 3.2:** Handle `*MergeConflictError` gracefully, returning HTTP `200 OK` with `status: "CONFLICT_DETECTED"` and populated `conflictInfo`.
  - [ ] **Subtask 3.3:** Ensure artifact database re-indexing (`SyncProjectArtifactsFromDisk`) is only called on `status: "SUCCESS"`.
- [ ] **Task 4: Integration Testing & Verification (AC: 4.11.1 - 4.11.4)**
  - [ ] **Subtask 4.1:** Update tests in `api/workspace/workspace_sync_test.go` to assert new JSON response structure for both clean and conflicting scenarios.
- [ ] **Task 5: Include Committed Branch Changes in `GetWorkspaceStatus` (AC: 4.11.5)**
  - [x] **Subtask 5.1:** Perform a `git pull` / remote fetch and update on the `main` (`baseBranch`) branch prior to comparing branch commits so that `main` is synchronized with remote.
  - [ ] **Subtask 5.2:** Update `GetWorkspaceStatus` in `api/workspace/git_client.go` to perform triple-dot branch diff (`git diff --name-status <baseRef>...<specRef>`) against the synchronized base branch when `baseBranch != specBranch`.
  - [ ] **Subtask 5.3:** Merge and deduplicate uncommitted worktree changes with committed branch modifications into `[]GitFileStatusItem`.
  - [ ] **Subtask 5.4:** Add unit and integration tests in `api/workspace/workspace_git_test.go` verifying that `main` is pulled and synchronized with remote before comparing and that committed changes on `specBranch` are returned by `GET /api/v1/projects/{id}/git/local-changes`.

---

## Dev Notes

### 🏗️ Architectural Guardrails
- **HTTP 200 OK for Conflict Payload**: Conflict is an expected business state, not an unhandled server error. Returning HTTP `200 OK` allows standard JSON parsing on the frontend.
- **Base Branch Remote Synchronization**: Before calculating committed branch differences between `baseBranch` and `specBranch`, the system MUST pull/sync `main` (`baseBranch`) with the remote `origin` to prevent comparing against stale base branch commits.
- **Deduplicated Local Changes**: When merging uncommitted worktree changes and committed branch diffs, uncommitted worktree status takes precedence for working copy attributes, while ensuring no file is listed twice.

## 🛠️ Target Files Matrix

| File Path | Status | Operational Role |
| :--- | :--- | :--- |
| `api/workspace/git_client.go` | `UPDATE` | Add checklist response structs. |
| `api/workspace/workspace.go` | `UPDATE` | Refactor handlers to populate and return `GitSyncChecklistResponse`. |
| `api/integrations/github.go` | `UPDATE` | Implement `GetOrOpenPullRequestURL` helper. |
| `api/workspace/workspace_sync_test.go` | `UPDATE` | Add test coverage for structured checklist response. |

---

## Dev Agent Record

### Agent Model Used
Gemini Next

### Debug Log References
### Completion Notes List
### File List
