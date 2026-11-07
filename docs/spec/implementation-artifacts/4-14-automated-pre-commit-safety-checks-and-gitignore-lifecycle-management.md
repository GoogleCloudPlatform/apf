# Story 4.14: Automated Pre-Commit Safety Checks & Gitignore Lifecycle Management

Status: ready-for-dev

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a Developer or Platform Engineer,
I want the system to execute an automated pre-commit check and enforce `.gitignore` lifecycle rules prior to staging and committing changes,
So that transient files and scratch directories (such as `.agents/.tmp/`) are never committed to the source tree, while ensuring necessary agent configurations (`.agents/`) remain tracked and any existing `.gitignore` contents are preserved.

## Acceptance Criteria

1. **Pre-Commit File Path Validation & Scratch Exclusions**:
   - **Given** a Git staging or commit operation is initiated on the API tier (via `LocalCommit`, `StagePath`, or `gitSyncHandler`),
   - **When** the system validates paths scheduled for staging or commit,
   - **Then** the system MUST reject any file paths located in `.agents/.tmp/` or temporary scratch directories, returning a zero-trust perimeter breach error. (AC: 4.14.1)
2. **Automatic `.gitignore` Initialization**:
   - **Given** a workspace repository clone where `.gitignore` is not present at the root,
   - **When** `enforceGitignore` executes prior to staging/committing,
   - **Then** the system MUST automatically create a new root `.gitignore` file with standard default exclusions, including `.tmp/` and `.agents/.tmp/`. (AC: 4.14.2)
3. **Preservation of Existing `.gitignore` Content**:
   - **Given** a workspace repository clone where `.gitignore` already exists at the root,
   - **When** `enforceGitignore` inspects the file,
   - **Then** the system MUST preserve all existing lines, comments, custom exclusions, and formatting intact without truncating or overwriting existing entries. (AC: 4.14.3)
4. **Idempotent Appending of Missing Exclusions**:
   - **Given** an existing root `.gitignore` missing `.agents/.tmp/` or `.tmp/`,
   - **When** `enforceGitignore` evaluates the file content,
   - **Then** the system MUST append the missing exclusion pattern(s) (`.agents/.tmp/` and `.tmp/`) to the end of the file and save the updated `.gitignore`. If the patterns are already present in any valid glob format (`.agents/.tmp`, `.agents/.tmp/`, `/.agents/.tmp/`), no duplicate lines MUST be added. (AC: 4.14.4)
5. **Enforce `.agents` Parent Directory Tracking**:
   - **Given** an existing root `.gitignore` containing a broad pattern ignoring the entire `.agents` directory (e.g., `.agents` or `.agents/`),
   - **When** `enforceGitignore` sanitizes the file,
   - **Then** the system MUST remove the blocking root `.agents` exclusion line to ensure custom agent skills and configurations remain version-controlled, while preserving the nested `.agents/.tmp/` ignore rule. (AC: 4.14.5)
6. **Comprehensive Unit & Integration Test Coverage**:
   - **Given** the new `.gitignore` enforcement and pre-commit checks,
   - **When** running Go test suites in `api/workspace/`,
   - **Then** unit tests in `git_client_sanitize_test.go` and integration tests in `git_client_integration_test.go` MUST cover `.gitignore` creation, preservation, pattern appending, and `.agents/.tmp` exclusion validation. (AC: 4.14.6)

## Tasks / Subtasks

- [ ] **Task 1: Update Pre-Commit Path Sanitization in `api/workspace/git_client.go` (AC: 4.14.1)**
  - [ ] Audit `isAllowedAgentsPath` to explicitly disallow `.agents/.tmp` and `.agents/.tmp/*`.
  - [ ] Ensure `sanitizeArtifacts` rejects any files within `.agents/.tmp/` with an explicit safety breach error.
- [ ] **Task 2: Refactor `enforceGitignore` in `api/workspace/git_client.go` (AC: 4.14.2, 4.14.3, 4.14.4, 4.14.5)**
  - [ ] Handle missing `.gitignore` by creating a new `.gitignore` file containing `.tmp/` and `.agents/.tmp/`.
  - [ ] When `.gitignore` exists, parse existing lines line-by-line while preserving comments and whitespace.
  - [ ] Check for presence of `.agents/.tmp/` pattern (matching `.agents/.tmp`, `.agents/.tmp/`, `/.agents/.tmp/`, `**/.agents/.tmp/`).
  - [ ] Check for presence of `.tmp/` pattern (matching `.tmp`, `.tmp/`, `/.tmp/`).
  - [ ] Strip any whole-directory ignore lines matching `.agents` or `.agents/`.
  - [ ] Append missing required patterns (`.agents/.tmp/`, `.tmp/`) to the end of `newLines`.
  - [ ] Write back to `.gitignore` only if modifications occurred (`modified == true`), with permissions `0644`.
- [ ] **Task 3: Add Unit Tests in `api/workspace/git_client_sanitize_test.go` (AC: 4.14.1, 4.14.2, 4.14.3, 4.14.4, 4.14.5)**
  - [ ] Add test case in `TestGitClient_SanitizeArtifacts` verifying `.agents/.tmp/file.txt` and `.agents/.tmp/cache.json` are rejected.
  - [ ] Add new test function `TestGitClient_EnforceGitignore` testing:
    - Creation when `.gitignore` does not exist.
    - Preservation of existing custom entries and comments.
    - Appending `.agents/.tmp/` when missing.
    - Idempotency when `.agents/.tmp/` already exists.
    - Removal of blocking `.agents/` line while adding `.agents/.tmp/`.
- [ ] **Task 4: Add Integration Tests in `api/workspace/git_client_integration_test.go` (AC: 4.14.6)**
  - [ ] Add integration test verifying `LocalCommit` and `StagePath` successfully trigger `enforceGitignore` and create/update `.gitignore` in a real git repository worktree.

## Dev Notes

- **Language & Framework**: Go 1.22+ using `github.com/go-git/go-git/v6` and standard `os`/`filepath`/`strings` libraries in the backend API tier (`api/workspace`).
- **File Location**: All changes are localized to `api/workspace/git_client.go`, `api/workspace/git_client_sanitize_test.go`, and `api/workspace/git_client_integration_test.go`.
- **Zero-Trust Safety & Isolation**:
  - `isAllowedAgentsPath` allows `.agents/_bmad/*` and `.agents/skills/*` but MUST explicitly deny `.agents/.tmp/*`.
  - Staged files violating path boundaries return descriptive errors preventing unauthorized commits.
- **Gitignore Line Formatting**:
  - Use clean POSIX newlines (`\n`) and ensure trailing newline when appending.
  - Trim whitespace when evaluating lines, but preserve original lines in the output buffer.

### Project Structure Notes

- Alignment with existing `api/workspace` directory structure.
- No modifications to external dependencies or protobufs needed.

### References

- [Source: docs/spec/planning-artifacts/epics.md#Story-4.14]
- [Source: docs/spec/planning-artifacts/architecture.md]
- [Source: docs/spec/planning-artifacts/prd.md#FR-4.2]
- [Source: api/workspace/git_client.go#L1180-L1254]

## Dev Agent Record

### Agent Model Used

Gemini 2.5 Pro

### Debug Log References

### Completion Notes List

### File List

- `api/workspace/git_client.go`
- `api/workspace/git_client_sanitize_test.go`
- `api/workspace/git_client_integration_test.go`
