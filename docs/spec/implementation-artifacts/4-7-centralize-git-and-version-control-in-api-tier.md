# Story 4.7: Centralize Git and Version Control in API Tier

Status: ready-for-dev

## Story

As a System Architect,
I want to migrate and centralize all Git operations (pull, commit, push, merge) from the AI tier to the API tier,
So that the downstream agent inference services are completely decoupled from version control and repository management libraries.

## Acceptance Criteria

1. **Given** a workspace session is initialized or explicitly re-hydrated,
   **When** the user opens the workspace,
   **Then** the API tier MUST execute `git pull` on the shared NFS mount (`gitPullHandler`) to ensure the local files are updated with any remote commits. (AC: 4.7.1)
2. **Given** the agent writes or modifies a file,
   **When** the file is saved under spec-approved locations (e.g., `docs/spec/` or `design-artifacts/`),
   **Then** the API tier's `OnFileWriteHook` MUST automatically perform a local git commit for that file (without pushing to the remote repository), using the custom signature compiled from the user's OIDC identity context (`claims.Subject`). (AC: 4.7.2)
3. **Given** the user triggers the "Sync" action in the UI,
   **When** the repository synchronization endpoint runs on the API tier,
   **Then** the API tier MUST push all local branch changes to the remote VCS feature branch (propagating the user's OIDC/OAuth credentials resolved via `ResolveGitHubToken`) and auto-create the Pull Request. (AC: 4.7.3)
4. **Given** the new centralized architecture,
   **When** agents are running,
   **Then** the AI tier MUST NOT perform any git commands, and the `GitSyncTracker` interceptor and any other git-related plugins in the AI tier must be completely disabled or removed. (AC: 4.7.4)
5. **Given** Git commit or staging operations,
   **When** running on the API tier,
   **Then** the system MUST check that code is only being added to spec-approved locations (e.g., `docs/spec/`) and verify/enforce root `.gitignore` boundaries (adding `.agents` if missing). (AC: 4.7.5)
6. **Given** Git operations running on the API tier,
   **When** modifying the workspace,
   **Then** the API tier MUST acquire a file-based lock (flock) on the workspace directory to prevent corruption from parallel executions. (AC: 4.7.6)
7. **Given** branch checkout operations,
   **When** checking out a target specBranch,
   **Then** the API tier's Git client MUST automatically handle checkout: if it exists locally, check it out; if it only exists on remote, track it; if it does not exist, create it branching off the baseBranch. (AC: 4.7.7)
8. **Given** Git failure scenarios on the API tier,
   **When** executing operations,
   **Then** the system MUST fail-hard but execute the following targeted recovery actions:
   - **Dirty Worktree on Pull**: Auto-commit dirty changes locally before executing pull to avoid merge blocks. (AC: 4.7.8)
   - **Auth Error (401/403)**: Return an HTTP 403/Forbidden with a payload instructing the user to reconnect credentials in Settings. (AC: 4.7.9)
   - **Network Timeout / Connection Drops**: Retry the operation up to 3 times using exponential backoff. (AC: 4.7.10)
   - **Corrupted or Missing .git Directory**: Automatically re-initialize Git (`git init`) directly in the existing directory, configure the remote origin, fetch, and align tracking branches, preserving existing files. (AC: 4.7.11)
   - **Push Rejected (Non-Fast-Forward)**: Automatically fetch the latest changes, rebase/merge, and retry the remote push. (AC: 4.7.12)
   - **Stale Git Index Lock (.git/index.lock)**: Detect and clean up stale lock files older than 5 minutes before executing the operation. (AC: 4.7.13)

## Tasks / Subtasks

- [x] **Task 1: Disable AI Tier Git Operations & Dependencies (AC: 4.7.4)**
  - [x] Disable or remove `GitSyncTrackerInterceptor` from the plugin registry in the AI tier (`ai/plugins/bmad/git_sync_tracker.go` and `ai/main.go`).
  - [x] Remove `git_commit` tool from the ADK tools layer in the AI tier (`ai/tools/git.go`).
  - [x] Completely remove `github.com/go-git/go-git/v6` and `github.com/go-git/go-billy/v6` imports from the AI tier codebase and `ai/go.mod`.
- [x] **Task 2: Implement Git Pull, Commit, and Push Logic in API Tier (AC: 4.7.1, 4.7.2, 4.7.3, 4.7.6, 4.7.7, 4.7.8, 4.7.9, 4.7.10, 4.7.11, 4.7.12, 4.7.13)**
  - [x] Integrate `github.com/go-git/go-git/v6` directly in the API tier by creating a new `api/git_client.go` helper module.
  - [x] Implement `Pull`, `LocalCommit`, and `Push` methods in the API tier using the shared NFS workspace path (`{WorkspaceMountPath}/{userId}/{projectId}`).
  - [x] Implement file-based lock (flock) mechanisms before Git operations are run.
  - [x] Implement branch checkout logic resolving dynamic branch creation, local/remote checkouts, and fallback tracks.
  - [x] Implement safety check checks and error handlers for dirty worktrees (auto-committing before pull), credentials mismatch, and transient retries.
  - [x] Implement automatic Git re-initialization (`git init` + config remote origin + fetch) upon `.git` directory corruption or absence, preserving existing files.
  - [x] Implement auto-fetch, rebase, and push-retry on non-fast-forward push rejections.
  - [x] Implement stale `.git/index.lock` detection and deletion logic.
  - [x] Ensure user OAuth tokens resolved via `integrations.ResolveGitHubToken(ctx, userId, projectId)` propagate for authorization.
- [x] **Task 3: Integrate Git Actions to Workspace Lifecycle (AC: 4.7.1, 4.7.2, 4.7.3)**
  - [x] In `api/workspace.go`, implement `gitPullHandler` to perform the physical `git pull` operation on workspace open.
  - [x] In `api/workspace.go`'s `OnFileWriteHook`, invoke `LocalCommit` for the written file if it resides in an approved specification directory.
  - [x] In `api/workspace.go`'s `gitSyncHandler`, perform the `git push` to the remote branch before opening the GitHub PR.
- [x] **Task 4: Implement Commit Safety Guardrails in API Tier (AC: 4.7.5)**
  - [x] Port the `.gitignore` verification and path check guardrails from the AI tier to the API tier's commit process.
- [x] **Task 5: Write Integration Tests (AC: 4.7.1 - 4.7.13)**
  - [x] Add unit and integration tests in `api/workspace_git_test.go` and `api/git_client_integration_test.go` to verify the new API-tier pull, local commit, push, locking, checkout, safety checks, and recovery loops.

## Dev Notes

- **NFS Share**: Both API and AI pods access the same `/workspaces` path, meaning local git operations by the API tier will directly update the files seen by the AI tier.
- **Git User Identity**: Use the authenticated user's OIDC email (from `claims.Subject`) and parsed name for the commit signature so that blame/history is accurate to the active user.
- **Fail-Hard with Recovery**: Local commits, pulls, and pushes should report clean stack errors and fail-hard upon final failure after recovery actions have been exhausted.

## Project Structure & Source Maps

### 1. Affected Files Matrix

| File Path | Status | Operational Role |
| :--- | :--- | :--- |
| [ai/plugins/bmad/git_sync_tracker.go](ai/plugins/bmad/git_sync_tracker.go) | `UPDATE` | Disable/remove git sync hooks in the AI tier. |
| [ai/tools/git.go](ai/tools/git.go) | `UPDATE` | Remove git commit tool from agent tools. |
| [ai/go.mod](ai/go.mod) | `UPDATE` | Remove go-git/go-billy dependencies. |
| [api/git_client.go](api/git_client.go) | `NEW` | Dedicated Git client using go-git for pull, local commit, push, locking, and safety checks. |
| [api/workspace.go](api/workspace.go) | `UPDATE` | Refactor workspace handlers to call the new Git client routines (on pull, on file write, and on sync). |
| [api/workspace_git_test.go](api/workspace_git_test.go) | `UPDATE` | Add tests verifying centralized git operations. |
