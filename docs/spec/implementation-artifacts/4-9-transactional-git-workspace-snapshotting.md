# Story 4.9: Transactional Git Workspace Snapshotting & Local State Preservation

Status: ready-for-dev

## Story

As a Developer or System User,
I want the Git client engine to capture local workspace snapshots (stashing uncommitted files and creating immutable pre-sync backup refs) before executing remote synchronization,
so that my local working tree and HEAD commit state are 100% guaranteed to be preserved and restored if any downstream rebase or fetch step fails.

## Acceptance Criteria

1. **Given** a remote sync operation is initiated on a local repository clone, **When** uncommitted local changes exist in the working directory, **Then** the engine MUST automatically stash uncommitted changes into an isolated stash reference (`refs/foundry/stash/<userId>`) before mutating refs. (AC: 4.9.1)
2. **Given** any remote sync operation, **When** snapshotting local state, **Then** the engine MUST create an immutable backup reference pointing to the local HEAD commit SHA (`refs/foundry/backups/<branch>-<timestamp>`). (AC: 4.9.2)
3. **Given** a failed or aborted sync operation, **When** rolling back, **Then** `rollbackLocalState` MUST perform a hard reset to the pre-sync backup reference and restore the stashed dirty changes, returning the workspace tree to its exact pre-sync state. (AC: 4.9.3)
4. **Given** a successful sync operation, **When** finalizing, **Then** the engine MUST pop the stashed changes back into the worktree and clean up the temporary backup reference. (AC: 4.9.4)

---

## Tasks / Subtasks

- [ ] **Task 1: Implement `createLocalBackup` in `api/workspace/git_client.go` (AC: 4.9.1, 4.9.2)**
  - [ ] **Subtask 1.1:** Inspect `Worktree.Status()`. If dirty, stage and stash uncommitted files into `refs/foundry/stash/<userId>`.
  - [ ] **Subtask 1.2:** Store current `HEAD` commit hash into a new Git reference `refs/foundry/backups/<branch>-<timestamp>`.
  - [ ] **Subtask 1.3:** Return `backupRef` and `stashRef` to caller.
- [ ] **Task 2: Implement `rollbackLocalState` in `api/workspace/git_client.go` (AC: 4.9.3)**
  - [ ] **Subtask 2.1:** Perform `w.Reset(&git.ResetOptions{Commit: backupCommitHash, Mode: git.HardReset})`.
  - [ ] **Subtask 2.2:** Apply and pop stashed dirty changes from `stashRef`.
  - [ ] **Subtask 2.3:** Log rollback details for auditability.
- [ ] **Task 3: Implement Cleanup & Defer Safeguards (AC: 4.9.4)**
  - [ ] **Subtask 3.1:** On success, delete temporary backup reference and restore stash onto the fast-forwarded worktree.
  - [ ] **Subtask 3.2:** Ensure `defer` blocks guarantee `rollbackLocalState` execution if panic or unexpected error occurs during sync.
- [ ] **Task 4: Unit Testing & Verification (AC: 4.9.1 - 4.9.4)**
  - [ ] **Subtask 4.1:** Write unit test `TestGitClient_SnapshotAndRollback` in `api/workspace/git_client_integration_test.go` verifying dirty file preservation and HEAD restoration.

---

## Dev Notes

### 🏗️ Architectural Guardrails
- **Zero Data Loss Guarantee**: Local state snapshot MUST happen under lock before any fetch/rebase call.
- **Native `go-git/v6`**: Use `go-git/v6` plumbing references (`Storer.SetReference`) and stash operations. Avoid raw shell commands.

---

## 🛠️ Target Files Matrix

| File Path | Status | Operational Role |
| :--- | :--- | :--- |
| `api/workspace/git_client.go` | `UPDATE` | Implement `createLocalBackup`, `rollbackLocalState`, and cleanup logic. |
| `api/workspace/git_client_integration_test.go` | `UPDATE` | Add unit tests for snapshotting and rollback. |

---

## Dev Agent Record

### Agent Model Used
Gemini Next

### Debug Log References
### Completion Notes List
### File List
