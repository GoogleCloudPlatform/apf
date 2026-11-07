# Story 4.10: Dry-Run Rebase Engine & Collision Detection

Status: ready-for-dev

## Story

As a Developer or System User,
I want the Git client engine to test rebase operations on an isolated temporary shadow branch (`refs/foundry/shadow-rebase-<uuid>`),
so that merge conflicts are detected without mutating the active working branch or leaving conflict markers (`<<<<<<< HEAD`) in workspace files.

## Acceptance Criteria

1. **Given** local HEAD has diverged from remote tracking commit, **When** executing rebase check, **Then** the engine MUST perform the cherry-pick sequence in an isolated shadow branch (`refs/foundry/shadow-rebase-<uuid>`). (AC: 4.10.1)
2. **Given** a merge conflict/collision during the shadow cherry-pick, **When** a conflicting file is encountered, **Then** `dryRunRebase` MUST catch the conflict immediately, abort the shadow rebase, delete the shadow branch, and return a `MergeConflictError` containing the list of conflicting files. (AC: 4.10.2)
3. **Given** a detected conflict, **When** returning from `dryRunRebase`, **Then** the active local working tree MUST remain completely untouched at its pre-sync HEAD commit without writing conflict markers (`<<<<<<< HEAD`) to disk. (AC: 4.10.3)
4. **Given** a clean shadow rebase (no conflicts), **When** `dryRunRebase` finishes, **Then** the engine MUST fast-forward the main local branch to the rebased HEAD commit and delete the temporary shadow branch ref. (AC: 4.10.4)

---

## Tasks / Subtasks

- [ ] **Task 1: Implement `dryRunRebase` in `api/workspace/git_client.go` (AC: 4.10.1, 4.10.2)**
  - [ ] **Subtask 1.1:** Create temporary shadow reference `refs/foundry/shadow-rebase-<uuid>` starting at `remoteCommit`.
  - [ ] **Subtask 1.2:** Traverse unique local commits between `mergeBase` and local `HEAD`.
  - [ ] **Subtask 1.3:** Attempt cherry-picking each local commit onto the shadow ref.
  - [ ] **Subtask 1.4:** Collect conflicting file paths if cherry-pick fails due to conflict.
- [ ] **Task 2: Implement Non-Destructive Conflict Error & Abort (AC: 4.10.2, 4.10.3)**
  - [ ] **Subtask 2.1:** Construct `MergeConflictError` with `ConflictingFiles`, `LocalCommit`, `RemoteCommit`, `MergeBaseCommit`.
  - [ ] **Subtask 2.2:** Automatically delete shadow branch reference.
  - [ ] **Subtask 2.3:** Trigger `rollbackLocalState` to ensure working tree is clean.
- [ ] **Task 3: Implement Clean Fast-Forward Application (AC: 4.10.4)**
  - [ ] **Subtask 3.1:** If shadow rebase succeeds with zero conflicts, point local branch ref to the rebased shadow HEAD.
  - [ ] **Subtask 3.2:** Delete shadow reference and return clean status.
- [ ] **Task 4: Unit Testing & Verification (AC: 4.10.1 - 4.10.4)**
  - [ ] **Subtask 4.1:** Add unit test `TestGitClient_DryRunRebase_Clean` verifying successful rebase application.
  - [ ] **Subtask 4.2:** Add unit test `TestGitClient_DryRunRebase_Conflict` verifying conflict detection, error return, and clean worktree state.

---

## Dev Notes

### 🏗️ Architectural Guardrails
- **Shadow Branch Isolation**: Rebase operations MUST never occur directly on the active working branch reference.
- **Zero Conflict Markers**: Files on disk MUST NEVER contain `<<<<<<< HEAD` markers under any circumstance.

---

## 🛠️ Target Files Matrix

| File Path | Status | Operational Role |
| :--- | :--- | :--- |
| `api/workspace/git_client.go` | `UPDATE` | Implement `dryRunRebase` and `MergeConflictError`. |
| `api/workspace/git_client_integration_test.go` | `UPDATE` | Add unit tests for clean and conflicting dry-run rebases. |

---

## Dev Agent Record

### Agent Model Used
Gemini Next

### Debug Log References
### Completion Notes List
### File List
