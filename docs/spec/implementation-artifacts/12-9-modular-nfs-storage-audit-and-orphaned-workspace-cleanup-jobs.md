---
title: 'Story 12.9: Modular Housekeeping Jobs: NFS Storage Audit & Orphaned Workspace Cleanup'
type: 'feature'
created: '2026-08-22'
status: 'done'
baseline_revision: '2b2cd30494a33f1048c5106c7338cbd06d97781d'
review_loop_iteration: 0
followup_review_recommended: true
context: []
warnings: []
deferred:
  - summary: >-
      Firestore collection group queries in DiscoverHangingProjects load all documents into memory via GetAll() without pagination.
    evidence: |-
      api/jobs/hanging_cleanup.go calls firestoreClient.CollectionGroup("projects").Documents(ctx).GetAll() and firestoreClient.CollectionGroup("user_project").Documents(ctx).GetAll(), matching pre-existing repository patterns.
    location: >-
      api/jobs/hanging_cleanup.go:313
    severity: low
---

<intent-contract>

## Intent

**Problem:** Storage governance scans and orphaned workspace deletion routines are currently embedded within administrative HTTP handlers and background workers rather than modular, self-contained `Job` implementations with CLI dispatcher and dry-run support.

**Approach:** Implement `NFSAuditJob` in `api/jobs/nfs_audit.go` and `HangingWorkspaceCleanupJob` in `api/jobs/hanging_cleanup.go` with CLI `--dry-run` flag handling, structured `slog` metrics logging, resilient directory traversal, and Firestore metadata synchronization.

## Boundaries & Constraints

**Always:**
- Implement `NFSAuditJob` with name `"nfs-audit"` satisfying the `Job` interface (`Name()`, `Description()`, `Run(ctx, cfg)`).
- Implement `HangingWorkspaceCleanupJob` with name `"hanging-cleanup"` satisfying the `Job` interface (`Name()`, `Description()`, `Run(ctx, cfg)`).
- Automatically register both jobs in `api/jobs/DefaultRegistry` during package initialization.
- Support `--dry-run` execution for destructive workspace cleanup operations via CLI flag, context (`jobs.IsDryRun(ctx)`), struct field, and environment variable `DRY_RUN`.
- In dry-run mode, scan and log candidate orphaned folders and their byte sizes without invoking `os.RemoveAll`.
- In non-dry-run mode, remove candidate dangling project folders and associated lock files (`.lock`), logging freed byte counts and deleted item counts.
- Log structured warnings via `slog.Warn` and continue directory traversal when encountering unreadable directories, broken symlinks, or permission errors.
- Ensure all API routes in `api/routes.go` and existing tests continue to compile and pass.

**Block If:**
- Upstream requirements mandate altering multi-tenant directory path partitioning or replacing Firestore metadata schemas.

**Never:**
- Never delete active project directories registered in Firestore / database.
- Never modify `sprint-status.yaml`.
- Never abort the entire cleanup or audit job prematurely due to a single folder permission error or broken symlink.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|---|---|---|---|
| NFS Storage Audit Run | `apf-jobs nfs-audit` | Scans mounted workspace path, updates Firestore `/admin_nfs_metadata/folders`, logs scanned bytes and file counts; exits 0 | If Firestore fails, returns error and exits 1 |
| Hanging Cleanup Dry-Run | `apf-jobs hanging-cleanup --dry-run` | Scans workspace directories, identifies folders missing in DB, logs candidate paths and sizes without disk deletion; exits 0 | Logs warnings on permission errors and continues |
| Hanging Cleanup Live Run | `apf-jobs hanging-cleanup` | Discovers dangling folders and lock files absent in DB, removes them via `os.RemoveAll`, logs freed bytes and deleted counts; exits 0 | Logs warnings on permission errors and continues |
| Unreadable directory / Broken symlink | Corrupted / restricted folder during audit or cleanup | Logs structured warning via `slog.Warn`, continues scanning remaining directories, returns nil error if scan completes | Non-fatal error logged; scan proceeds |
| Missing Workspace Mount Path | Workspace folder absent or empty | Gracefully handles non-existent path, returns nil or empty list without panicking | Logs info/debug, returns nil error |
| Positional vs Flag CLI args | `apf-jobs hanging-cleanup --dry-run` and `apf-jobs --dry-run hanging-cleanup` | Both flag orders correctly recognize `hanging-cleanup` job and activate dry-run mode | Unrecognized flags return error and exit 1 |

</intent-contract>

## Code Map

- `api/jobs/job.go` -- Core `Job` interface, thread-safe `Registry`, and context dry-run helpers (`ContextWithDryRun`, `IsDryRun`).
- `api/jobs/nfs_audit.go` -- `NFSAuditJob` implementation executing storage audit scan, logging byte/file metrics, and updating Firestore metadata.
- `api/jobs/nfs_audit_test.go` -- Unit and integration tests for `NFSAuditJob` registration, scan execution, and error handling.
- `api/jobs/hanging_cleanup.go` -- `HangingWorkspaceCleanupJob` implementation supporting `--dry-run`, orphaned folder discovery, lock deletion, and error resilience.
- `api/jobs/hanging_cleanup_test.go` -- Unit and integration tests for `HangingWorkspaceCleanupJob` verifying dry-run vs live deletion, lock file removal, and symlink resilience.
- `api/cmd/jobs/main.go` -- CLI dispatcher updated with `--dry-run` flag support and seamless flag/argument parsing.
- `api/cmd/jobs/main_test.go` -- CLI dispatcher tests covering `--dry-run` execution, job listing, and flag ordering.
- `api/db/nfs_store.go` -- `NFSStore` interface and Firestore implementation for folder metadata persistence.
- `api/db/project.go` -- `ProjectStore` interface and domain model for verifying active project registration.

## Tasks & Acceptance

**Execution:**
- `api/jobs/job.go` -- Add dry-run context helper functions (`ContextWithDryRun`, `IsDryRun`) -- Enables context-propagated dry-run state across jobs.
- `api/jobs/nfs_audit.go` -- Implement `NFSAuditJob` with `RunNFSAuditScan` invocation, metrics logging, and auto-registration -- Provides modular storage audit job.
- `api/jobs/nfs_audit_test.go` -- Unit tests for `NFSAuditJob` execution, Firestore metadata upsert, and error resilience -- Validates audit job behavior.
- `api/jobs/hanging_cleanup.go` -- Implement `HangingWorkspaceCleanupJob` with `DiscoverHangingProjects`, `--dry-run` protection, lock cleanup, and resilience -- Provides modular orphaned workspace cleanup job.
- `api/jobs/hanging_cleanup_test.go` -- Unit tests for `HangingWorkspaceCleanupJob` verifying dry-run reporting, active folder preservation, orphan deletion, and permission error resilience -- Validates cleanup job safety.
- `api/cmd/jobs/main.go` -- Add `--dry-run` flag parsing and propagate dry-run context into job execution -- Exposes dry-run flag in CLI runner.
- `api/cmd/jobs/main_test.go` -- Update CLI dispatcher tests to verify `--dry-run` execution and job registry listing -- Ensures CLI compatibility.

**Acceptance Criteria:**
- Given `NFSAuditJob` in `api/jobs/nfs_audit.go`, when executed via `apf-jobs nfs-audit`, then it initializes `db.NFSStore`, invokes `RunNFSAuditScan`, updates organization folder metadata in Firestore under `/admin_nfs_metadata/folders`, logs scanned byte counts and file counts via `slog`, and returns nil error upon successful completion.
- Given `HangingWorkspaceCleanupJob` in `api/jobs/hanging_cleanup.go`, when executed via `apf-jobs hanging-cleanup`, then it calls `DiscoverHangingProjects` against the mounted workspace path, removes dangling project directories and lock files for projects absent in the database, and records freed disk space and deleted counts in logs.
- Given the flag `--dry-run` passed to `apf-jobs hanging-cleanup --dry-run`, when executed, then it scans and reports all candidate orphaned folders and their sizes without performing any `os.RemoveAll` disk deletions.
- Given unreadable directories, broken symlinks, or permission errors on individual folders during scanning, when encountered, then the job logs a structured warning and continues scanning remaining directories rather than aborting prematurely.
- Given unit & integration tests in `api/jobs/nfs_audit_test.go` and `api/jobs/hanging_cleanup_test.go`, when tests are run, then all tests pass with high branch coverage.

## Spec Change Log

## Review Triage Log

### 2026-08-22 — Review pass 3
- intent_gap: 0
- bad_spec: 0
- patch: 4: (high 0, medium 1, low 3)
- defer: 0
- reject: 10
- addressed_findings:
  - `[low]` `[patch]` Added circular/nested directory symlink cycle detection and root symlink skipping in `ScanWorkspaceDirectory`.
  - `[low]` `[patch]` Added symlinked user directory checking in `DiscoverHangingProjects`.
  - `[medium]` `[patch]` Enforced both `OrganizationStore` and `ProjectStore` presence when Firestore client is nil in `HangingWorkspaceCleanupJob.Run`.
  - `[low]` `[patch]` Standardized `DRY_RUN` environment variable parsing and added non-empty workspace path check in `RunNFSAuditScan`.

### 2026-08-22 — Review pass 2
- intent_gap: 0
- bad_spec: 0
- patch: 4: (high 0, medium 1, low 3)
- defer: 1: (high 0, medium 0, low 1)
- reject: 10
- addressed_findings:
  - `[medium]` `[patch]` Propagated database query errors in `DiscoverHangingProjects` to prevent false-positive orphan deletions if Firestore or store queries fail.
  - `[low]` `[patch]` Added symlinked directory resolution in `DiscoverWorkspaceFolders`, `ScanWorkspaceDirectory`, and `DiscoverHangingProjects`.
  - `[low]` `[patch]` Rejected unexpected extra positional CLI arguments in `apf-jobs`.
  - `[low]` `[patch]` Added `DryRun bool` struct field and uniform environment checking to `NFSAuditJob`.

### 2026-08-22 — Review pass 1
- intent_gap: 0
- bad_spec: 0
- patch: 4: (high 0, medium 1, low 3)
- defer: 0
- reject: 7
- addressed_findings:
  - `[low]` `[patch]` Added `nil` context guard in `jobs.ContextWithDryRun` defaulting to `context.Background()`.
  - `[medium]` `[patch]` Added database store validation and error handling in `HangingWorkspaceCleanupJob.Run` to prevent running cleanup without database records.
  - `[low]` `[patch]` Added dry-run support in `NFSAuditJob` and `RunNFSAuditScan` to skip metadata writes and record deletions when `--dry-run` is active.
  - `[low]` `[patch]` Integrated `core.ExpandPath` for workspace mount path resolution across jobs.

## Verification

**Commands:**
- `cd api && go test -v -race ./jobs/... ./cmd/jobs/...` -- expected: All job and CLI dispatcher tests pass with zero race conditions
- `cd api && go run ./cmd/jobs --list` -- expected: Displays `nfs-audit` and `hanging-cleanup` in available jobs list
- `cd api && go test ./...` -- expected: All unit and integration tests pass across the entire API module

## Auto Run Result

### Summary of Implemented Change
- Implemented modular `NFSAuditJob` in `api/jobs/nfs_audit.go` and `HangingWorkspaceCleanupJob` in `api/jobs/hanging_cleanup.go` conforming to the `Job` interface with auto-registration in `jobs.DefaultRegistry`.
- Added dry-run execution support across CLI flag (`--dry-run`), Context (`jobs.ContextWithDryRun`), struct fields, and environment variables (`DRY_RUN`).
- Enhanced `api/cmd/jobs/main.go` with multi-pass flag parsing to accept `--dry-run` and job names in any flag/positional ordering while rejecting unexpected extra arguments.
- Refactored `api/admin_nfs_audit_worker.go` and `api/admin_nfs_handlers.go` into backward-compatible shims delegating to the modular job logic.
- Implemented resilient directory scanning and discovery with symlink resolution, cycle detection, error-tolerant traversal, and safety boundaries preventing non-project deletions.
- Comprehensive unit and integration test coverage across `api/jobs/nfs_audit_test.go`, `api/jobs/hanging_cleanup_test.go`, `api/jobs/job_test.go`, and `api/cmd/jobs/main_test.go`.

### Files Changed
- `api/jobs/job.go` -- Core `Job` interface, thread-safe `Registry`, and context dry-run helpers (`ContextWithDryRun`, `IsDryRun`).
- `api/jobs/job_test.go` -- Unit tests verifying dry-run context getters and setters.
- `api/jobs/nfs_audit.go` -- `NFSAuditJob` implementation executing storage audit scan, logging byte/file metrics, and updating Firestore metadata with cycle detection and dry-run safety.
- `api/jobs/nfs_audit_test.go` -- Unit and integration tests for `NFSAuditJob` registration, scan execution, upserting, dry-run mode, legacy cleanup, and symlink handling.
- `api/jobs/hanging_cleanup.go` -- `HangingWorkspaceCleanupJob` implementation supporting `--dry-run`, orphaned folder discovery, lock deletion, and error resilience.
- `api/jobs/hanging_cleanup_test.go` -- Unit and integration tests for `HangingWorkspaceCleanupJob` verifying dry-run vs live deletion, lock file removal, and symlink resilience.
- `api/cmd/jobs/main.go` -- CLI dispatcher updated with `--dry-run` flag support and flexible flag/positional argument parsing.
- `api/cmd/jobs/main_test.go` -- CLI dispatcher tests covering `--dry-run` execution, job listing, and flag ordering.
- `api/admin_nfs_audit_worker.go` -- Thin delegation shim forwarding to `jobs.ScanWorkspaceDirectory`, `jobs.DiscoverWorkspaceFolders`, and `jobs.RunNFSAuditScan`.
- `api/admin_nfs_handlers.go` -- Backward-compatible shims delegating to `jobs.DiscoverHangingProjects` and `jobs.HangingProjectItem`.
- `docs/spec/implementation-artifacts/12-9-modular-nfs-storage-audit-and-orphaned-workspace-cleanup-jobs.md` -- Spec and review records.

### Review Findings Breakdown
- Pass 3: 4 patches applied, 0 items deferred, 10 items rejected
- Total Patches Applied across all passes: 12
- Deferred Items: 1 (pre-existing Firestore CollectionGroup pagination tracked in frontmatter and ledger)

### Follow-up Review Recommendation
- Patched counts by severity (Pass 3): high: 0, medium: 1, low: 3
- Formula score: 3 × 1 + 1 × 3 = 6 (>= 5)
- Recommendation: `true` (follow-up review recommended)

### Verification Performed
- `cd api && go test -v -race ./jobs/... ./cmd/jobs/...` -- Passed (all job and CLI tests passed with zero data races)
- `cd api && go run ./cmd/jobs --list` -- Passed (displayed `nfs-audit` and `hanging-cleanup` jobs)
- `cd api && go test ./...` -- Passed (all unit and integration tests across the entire API module passed)

### Residual Risks
- Firestore collection group queries in `DiscoverHangingProjects` load documents in memory without pagination (tracked in `deferred` list).

