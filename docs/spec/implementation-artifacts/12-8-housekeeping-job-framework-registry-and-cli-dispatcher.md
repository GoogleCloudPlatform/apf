---
title: 'Story 12.8: Housekeeping Job Framework, Registry & CLI Dispatcher'
type: 'feature'
created: '2026-08-22'
status: 'done'
baseline_revision: 'cc663a2cb72bf0a37732bcb560b66d7fd1bcd5b9'
review_loop_iteration: 0
followup_review_recommended: false
context: []
warnings: []
deferred: []
---

<intent-contract>

## Intent

**Problem:** Housekeeping tasks (e.g., NFS storage audits, orphaned workspace cleanup, retention sweeps) currently lack a unified Go job interface, thread-safe registry, and standalone CLI dispatcher, making it difficult to execute maintenance tasks as independent batch processes outside HTTP server requests.

**Approach:** Implement a standard `Job` interface and thread-safe `Registry` in `api/jobs/job.go`, and a standalone CLI entrypoint in `api/cmd/jobs/main.go` supporting `--list`, positional or `--name` job execution, `signal.NotifyContext` interrupt handling, and structured `log/slog` logging.

## Boundaries & Constraints

**Always:**
- Define `Job` interface with `Name() string`, `Description() string`, and `Run(ctx context.Context, cfg *core.Configuration) error`.
- Ensure `Registry` operations (`Register`, `Get`, `List`, `Execute`) are protected by `sync.RWMutex` for concurrent safety.
- Provide a default package-level registry (`DefaultRegistry`) and top-level helper functions in `api/jobs/`.
- Handle OS interrupt signals (`SIGINT`, `SIGTERM`) in `api/cmd/jobs/main.go` using `signal.NotifyContext`.
- Bootstrap `core.Configuration` via `core.NewConfig()` when executing jobs via the CLI dispatcher.
- Emit structured logs via `log/slog` with job name, execution duration, and outcome.
- Exit with code 0 on success or `--list`, and exit with code 1 on unknown jobs, invalid arguments, or job execution errors.

**Block If:**
- Core application architecture requires replacing Go standard library CLI and concurrency primitives with an incompatible external framework.

**Never:**
- Never modify or interfere with HTTP routes in `api/routes.go` or web server lifecycle in `api/main.go`.
- Never modify `sprint-status.yaml`.
- Never execute destructive operations without dry-run protections and explicit job parameters.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|---|---|---|---|
| List jobs flag | `apf-jobs --list` | Formatted list of all registered jobs with names and descriptions; exits 0 | No error expected |
| Positional job execution | `apf-jobs <valid-job>` | Runs job with `signal.NotifyContext` and `core.Configuration`; logs start/finish metrics; exits 0 | If job returns error, logs error and exits 1 |
| Flag job execution | `apf-jobs --name <valid-job>` | Runs job with `signal.NotifyContext` and `core.Configuration`; logs start/finish metrics; exits 0 | If job returns error, logs error and exits 1 |
| Unknown job name | `apf-jobs unknown-job` | Prints error message listing available registered jobs; exits 1 | Logs error and returns exit code 1 |
| Missing arguments | `apf-jobs` (no flags or args) | Prints usage instructions and list of available jobs; exits 1 | Logs error/usage and returns exit code 1 |
| Concurrent registry access | Goroutines concurrently calling `Register`, `Get`, `List`, `Execute` | Safe concurrent execution without race conditions | Mutex synchronization protects internal map and slice |
| OS Interrupt Signal | `SIGINT`/`SIGTERM` received during job execution | `signal.NotifyContext` cancels context; job receives cancellation; exits with error code 1 | Job error logged |

</intent-contract>

## Code Map

- `api/jobs/job.go` -- Defines `Job` interface, thread-safe `Registry`, `DefaultRegistry`, and package helper functions (`Register`, `Get`, `List`, `Execute`).
- `api/jobs/job_test.go` -- Unit tests validating job registration, retrieval, listing, execution, concurrency safety (`-race`), and error handling.
- `api/cmd/jobs/main.go` -- CLI dispatcher entrypoint for `apf-jobs` supporting `--list`, positional or `--name` job execution, `signal.NotifyContext`, structured `slog` logging, and exit code handling.
- `api/cmd/jobs/main_test.go` -- Tests for CLI dispatcher argument parsing, `--list`, job execution, missing args, unknown job errors, signal handling, and exit codes.
- `api/core/configuration.go` -- `core.Configuration` and `core.NewConfig()` used by the CLI to bootstrap configuration from environment variables.

## Tasks & Acceptance

**Execution:**
- `api/jobs/job.go` -- Create `Job` interface, `Registry` struct, `DefaultRegistry`, and error definitions -- Provides modular job abstraction and thread-safe registry.
- `api/jobs/job_test.go` -- Implement unit tests for `Registry` methods and concurrency safety -- Validates registration, lookup, execution, and race condition freedom.
- `api/cmd/jobs/main.go` -- Implement CLI entrypoint supporting `--list`, positional/flag dispatch, signal handling, and structured logging -- Provides standalone CLI tool.
- `api/cmd/jobs/main_test.go` -- Implement unit and integration tests for CLI dispatcher -- Verifies flag parsing, job execution, error messaging, and exit codes.

**Acceptance Criteria:**
- Given the `api/jobs/` package, when imported, then it defines a standard `Job` interface:
  ```go
  type Job interface {
      Name() string
      Description() string
      Run(ctx context.Context, cfg *core.Configuration) error
  }
  ```
  and a thread-safe `Registry` struct supporting `Register(Job)`, `Get(name string) (Job, bool)`, `List() []Job`, and `Execute(ctx context.Context, name string, cfg *core.Configuration) error`.
- Given the CLI command `go run ./api/cmd/jobs --list` or `apf-jobs --list`, when executed, then it outputs a formatted list of all registered housekeeping jobs with their names and descriptions and exits with code 0.
- Given the CLI command `go run ./api/cmd/jobs <job-name>` or `go run ./api/cmd/jobs --name <job-name>`, when executed with a valid registered job name, then it initializes `core.Configuration` from environment variables, executes the job with an interrupt-aware context (`signal.NotifyContext(SIGINT, SIGTERM)`), logs start/finish metrics using `log/slog`, and exits with code 0 on success or code 1 on error.
- Given an invalid job name provided to the CLI, when executed, then it prints a helpful error message listing valid available jobs and exits with code 1.
- Given unit tests in `api/jobs/job_test.go` and `api/cmd/jobs/main_test.go`, when `go test ./api/jobs/...` and `go test ./api/cmd/jobs/...` are run, then all tests pass validating registration, execution, and error handling.

## Spec Change Log

## Review Triage Log

### 2026-08-22 — Review pass
- intent_gap: 0
- bad_spec: 0
- patch: 4: (high 0, medium 0, low 4)
- defer: 0
- reject: 11
- addressed_findings:
  - `[low]` `[patch]` Added zero-value safety for `Registry` struct instances initialized without `NewRegistry()`.
  - `[low]` `[patch]` Normalized whitespace trimming in `Registry.Get` and `Registry.Execute` to match `Registry.Register`.
  - `[low]` `[patch]` Added nil writer guard (`io.Discard`) and `flag.ErrHelp` exit code 0 handling in CLI runner.
  - `[low]` `[patch]` Added dynamic column formatting for `printJobsList` in CLI dispatcher.

## Verification

**Commands:**
- `cd api && go test -v -race ./jobs/... ./cmd/jobs/...` -- expected: All unit tests pass with zero race conditions
- `cd api && go run ./cmd/jobs --list` -- expected: Outputs formatted list of registered jobs and exits 0
- `cd api && go test ./...` -- expected: All tests pass across the entire API module

## Auto Run Result

### Summary of Implemented Change
Implemented the housekeeping job framework, registry, and CLI dispatcher for Story 12.8:
- Defined the standard Go `Job` interface (`Name() string`, `Description() string`, `Run(ctx context.Context, cfg *core.Configuration) error`) and thread-safe `Registry` in `api/jobs/job.go`.
- Implemented the standalone CLI entrypoint in `api/cmd/jobs/main.go` supporting `--list`, positional or `--name` argument parsing, `signal.NotifyContext` interrupt handling, `log/slog` structured logging, and exit code propagation.
- Comprehensive unit and concurrency tests in `api/jobs/job_test.go` and CLI tests in `api/cmd/jobs/main_test.go`.

### Files Changed
- `api/jobs/job.go`: Core `Job` interface, thread-safe `Registry` with mutex locks, and `DefaultRegistry` helper functions.
- `api/jobs/job_test.go`: Unit tests for job registration, lookup, listing, execution, zero-value safety, and 50-worker concurrency tests under `-race`.
- `api/cmd/jobs/main.go`: Standalone CLI dispatcher for housekeeping jobs with signal trapping and structured logging.
- `api/cmd/jobs/main_test.go`: Unit and integration tests for CLI flag parsing, execution, error cases, and signal handling.
- `docs/spec/implementation-artifacts/12-8-housekeeping-job-framework-registry-and-cli-dispatcher.md`: Story specification, triage log, and auto run result.
- `docs/spec/implementation-artifacts/epic-12-context.md`: Compiled Epic 12 context.

### Review Findings Breakdown
- Patches applied: 4 (0 high, 0 medium, 4 low)
- Items deferred: 0
- Items rejected: 11 (out of scope for 12.8, handled in downstream stories 12.9/12.10)

### Follow-up Review Recommendation
- `followup_review_recommended: false` (Score: 4 = 3 * 0 + 1 * 4 < 5, with 0 high severity findings).

### Verification
- `cd api && go test -v -race ./jobs/... ./cmd/jobs/...`: PASS (All 17 tests passed with zero race conditions).
- `cd api && go run ./cmd/jobs --list`: PASS (Exited 0 with formatted job list).
- `cd api && go test ./...`: PASS (All API package tests passed).

### Residual Risks
- None.
