---
title: 'Story 12.13: Super Admin Jobs Catalog & Cloud Run Execution History Backend API'
type: 'feature'
created: '2026-08-31'
status: 'done'
baseline_revision: 'a403c9da24e1517d842a7438dfb7a0a4cc0423a3'
review_loop_iteration: 0
followup_review_recommended: false
context: []
warnings: []
deferred: []
---

<intent-contract>

## Intent

**Problem:** Platform and system administrators need backend API endpoints to inspect the catalog of registered batch housekeeping jobs and query real-time execution history directly from Google Cloud Run v2 without requiring separate database storage, so operational tooling and admin screens can discover available jobs and monitor execution states.

**Approach:** Implement `GET /api/v1/admin/jobs` to expose the registered job catalog with parameter descriptions and `GET /api/v1/admin/jobs/executions` to proxy and parse Google Cloud Run v2 executions with pagination, timing calculations, status mapping, error extraction, and logging deep-links, guarded by Super Admin RBAC.

## Boundaries & Constraints

**Always:**
- Require `is_super_admin: true` from validated user claims context for both `GET /api/v1/admin/jobs` and `GET /api/v1/admin/jobs/executions`. Reject non-super-admins and standard tenant users with HTTP `403 Forbidden`.
- `GET /api/v1/admin/jobs` must return HTTP `200 OK` with all registered jobs in `jobs.DefaultRegistry`, including name, description, and supported parameter flags (such as `dryRun`).
- `GET /api/v1/admin/jobs/executions` must support optional query parameters `pageSize` and `pageToken`.
- Parse Cloud Run v2 Executions into structured records containing `id`, `name`, `jobName` (extracted from container args or fallback), `dryRun` (boolean from args), `status` (`RUNNING`, `SUCCEEDED`, `FAILED`, `CANCELLED`), `createTime`, `startTime`, `completionTime`, `durationSeconds`, `durationFormatted` (e.g. `"1m 14s"`), `errorMessage`, and `logUri`.
- Provide `SimulatedCloudRunJobRunner` implementation for `ListExecutions` to support local development and unit tests without GCP credentials.
- Register endpoints in `api/routes.go` with `authMiddleware` and `superAdminAuthMiddleware` alongside OPTIONS preflight handlers.
- Ensure all Go tests pass with zero errors via `go test ./...`.

**Block If:**
- Modifying IAM security architecture outside of Cloud Run Job permissions or breaking existing auth middleware contracts.

**Never:**
- Never modify or write `sprint-status.yaml`.
- Never bypass Super Admin authentication checks for administrative job endpoints.
- Never require persistent database tables or Firestore collections for execution history; query Cloud Run v2 API directly.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| Super Admin Lists Job Catalog | `GET /api/v1/admin/jobs` with `is_super_admin: true` | HTTP 200 OK, JSON array of job catalog items (`name`, `description`, `parameters`) | No error expected |
| Non-Super-Admin Lists Jobs | `GET /api/v1/admin/jobs` with `is_super_admin: false` | HTTP 403 Forbidden, `{ "status": "ERROR", "message": "Forbidden: Super Administrator access required" }` | Returns 403 immediately |
| Super Admin Queries Executions (Default) | `GET /api/v1/admin/jobs/executions` with `is_super_admin: true` | HTTP 200 OK, `{ "executions": [...], "nextPageToken": "..." }` parsed from Cloud Run v2 | Returns HTTP 500 if Cloud Run API fails |
| Super Admin Queries Executions (Paginated) | `GET /api/v1/admin/jobs/executions?pageSize=10&pageToken=abc` | HTTP 200 OK with pageSize=10 and pageToken passed to Cloud Run SDK | Returns HTTP 500 on SDK error |
| Non-Super-Admin Queries Executions | `GET /api/v1/admin/jobs/executions` with `is_super_admin: false` | HTTP 403 Forbidden | Returns 403 immediately |
| Missing User Claims | `GET /api/v1/admin/jobs` or `GET /api/v1/admin/jobs/executions` without claims | HTTP 403 Forbidden | Returns 403 |
| Local / Simulated Runner Mode | `GET /api/v1/admin/jobs/executions` in dev mode | HTTP 200 OK with simulated execution entries, statuses, durations, and log URIs | No error expected |
| Cloud Run SDK Execution Parsing | Cloud Run v2 execution with args `["nfs-audit", "--dry-run"]`, failed condition | Structured item with `jobName: "nfs-audit"`, `dryRun: true`, `status: "FAILED"`, formatted duration, and condition error | Error details captured in `errorMessage` |

</intent-contract>

## Code Map

- `api/admin_job_handlers.go` -- Defines catalog data structures (`JobCatalogItem`, `JobParameterInfo`, `JobExecutionItem`, `JobExecutionsListResponse`), adds `ListExecutions` method to `CloudRunJobRunner` interface and its GCP / Simulated implementations, implements `ListJobCatalog` and `ListJobExecutions` HTTP handlers, and provides duration formatting and execution parsing helpers.
- `api/admin_job_handlers_test.go` -- Unit tests verifying super admin RBAC, non-super-admin 403 rejection, job catalog response accuracy, Cloud Run execution parsing (args, status, duration, error messages), pagination handling, and simulated execution listing.
- `api/admin_handlers.go` -- `AdminHandler` wrapping `CloudRunJobRunner` and providing access to job registry.
- `api/routes.go` -- Registers `GET /api/v1/admin/jobs`, `GET /api/v1/admin/jobs/executions`, and OPTIONS preflight handlers under `superAdminAuthMiddleware`.

## Tasks & Acceptance

**Execution:**
- `api/admin_job_handlers.go` -- Implement `ListJobCatalog` handler, `ListExecutions` method on `CloudRunJobRunner` and `GCPCloudRunJobRunner` / `SimulatedCloudRunJobRunner`, execution parsing logic, duration formatting, and `ListJobExecutions` handler -- Exposes catalog discovery and execution history APIs.
- `api/routes.go` -- Register `GET /api/v1/admin/jobs` and `GET /api/v1/admin/jobs/executions` with `superAdminAuthMiddleware` and OPTIONS preflights -- Connects endpoints to router.
- `api/admin_job_handlers_test.go` -- Author comprehensive unit tests covering catalog listing, execution querying, RBAC, execution parser edge cases, pagination, and error responses -- Verifies API behavior and security boundaries.

**Acceptance Criteria:**
- Given an authenticated Super Administrator (`is_super_admin: true`), when they send `GET /api/v1/admin/jobs`, then the backend returns HTTP 200 OK with a JSON list of all registered jobs in `jobs.DefaultRegistry`.
- Given a non-super-admin user attempting to access `GET /api/v1/admin/jobs` or `GET /api/v1/admin/jobs/executions`, when requested, then the backend rejects the request with HTTP 403 Forbidden.
- Given an authenticated Super Administrator (`is_super_admin: true`), when they send `GET /api/v1/admin/jobs/executions`, then the backend queries Google Cloud Run v2 API and returns HTTP 200 OK with parsed execution records and `nextPageToken`.
- Given unit tests in `api/admin_job_handlers_test.go`, when executed, then all tests pass with zero errors.

## Spec Change Log

## Review Triage Log

### 2026-08-31 — Review pass 1
- intent_gap: 0
- bad_spec: 0
- patch: 4 (low 4)
- defer: 0
- reject: 10
- addressed_findings:
  - `[low]` `[patch]` Added safety nil checks for containers and conditions in `parseCloudRunExecution`.
  - `[low]` `[patch]` Applied `url.QueryEscape` to query and project parameters in fallback `logURI` generation.
  - `[low]` `[patch]` Clamped `pageSize` query parameter to 100 in `ListJobExecutions`.
  - `[low]` `[patch]` Added page token support to `SimulatedCloudRunJobRunner.ListExecutions` and added unit test coverage.

## Design Notes

- Duration calculation computes `completionTime - startTime` if completed, or `now - startTime` if running.
- Duration formatting outputs human-readable strings like `"1h 2m 3s"`, `"1m 14s"`, `"42s"`, `"0s"`.
- Execution status is derived from Cloud Run v2 conditions and task counters: `CANCELLED` if cancelled, `SUCCEEDED` if succeeded, `FAILED` if failed or error conditions present, `RUNNING` if reconciling or running count > 0.
- `jobName` and `dryRun` are extracted from `exec.Template.Containers[0].Args` (first arg is jobName, presence of `--dry-run` sets dryRun to true).
- `logUri` uses `exec.LogUri` if populated or generates a deep link to GCP Cloud Logging.

## Verification

**Commands:**
- `cd api && go test -v -run TestAdminJobHandlers ./...` -- expected: PASS
- `cd api && go test ./...` -- expected: ok google.com/apf/api ...

## Auto Run Result

Status: done

### Summary of Implemented Change
- Implemented `GET /api/v1/admin/jobs` endpoint in `api/admin_job_handlers.go`, exposing the catalog of registered housekeeping jobs (`jobs.DefaultRegistry`) with parameter schema (`dryRun` boolean flag), guarded by Super Admin RBAC (`is_super_admin: true`).
- Extended `CloudRunJobRunner` interface with `ListExecutions(ctx context.Context, pageSize int, pageToken string) (*JobExecutionsListResponse, error)` and implemented it on `GCPCloudRunJobRunner` (querying Google Cloud Run v2 API) and `SimulatedCloudRunJobRunner` (supporting local dev / tests).
- Implemented `GET /api/v1/admin/jobs/executions` endpoint supporting `pageSize` (clamped to 100) and `pageToken` query parameters, guarded by `superAdminAuthMiddleware`.
- Implemented Cloud Run execution parser `parseCloudRunExecution` extracting `id`, `jobName`, and `dryRun` from container args, evaluating condition status (`RUNNING`, `SUCCEEDED`, `FAILED`, `CANCELLED`), computing runtime duration (`durationSeconds` and `durationFormatted`), aggregating error details, and constructing Google Cloud Logging deep links.
- Registered endpoints and OPTIONS preflights in `api/routes.go`.
- Wrote extensive unit tests in `api/admin_job_handlers_test.go` covering RBAC, catalog listing, execution querying, edge case parsing (nil containers/conditions), duration formatting, simulated pagination, error handling, and router integration.

### Files Changed
- `api/admin_job_handlers.go` -- Added catalog & execution data types, `ListExecutions` runner methods, execution parser, duration formatter, and `ListJobCatalog` & `ListJobExecutions` HTTP handlers.
- `api/admin_job_handlers_test.go` -- Added unit tests for catalog discovery, execution querying, RBAC, execution parser edge cases, duration formatting, simulated pagination, and router integration.
- `api/routes.go` -- Registered `GET /api/v1/admin/jobs`, `GET /api/v1/admin/jobs/executions`, and OPTIONS preflights.
- `docs/spec/implementation-artifacts/12-13-super-admin-jobs-catalog-cloud-run-execution-history-backend.md` -- Spec and execution record.

### Review Findings Breakdown
- Pass 1: 4 patches applied, 0 items deferred, 10 items rejected
- Total Patches Applied: 4
- Deferred Items: 0

### Follow-up Review Recommendation
- Patched counts by severity (Pass 1): high: 0, medium: 0, low: 4
- Formula score: 3 × 0 + 1 × 4 = 4 (< 5)
- Recommendation: `false`

### Verification Performed
- `cd api && go test -v -run TestAdminJobHandlers ./...` -- PASS (all 18 unit tests passed)
- `cd api && go test ./...` -- PASS (all Go packages passed with 0 errors)

### Residual Risks
- None. All endpoints adhere to Super Admin RBAC and interface abstractions enable seamless local simulation and GCP Cloud Run execution.
