---
title: 'Story 12.12: Super Admin On-Demand Job Execution API & Admin Portal Trigger UI'
type: 'feature'
created: '2026-08-22'
status: 'done'
baseline_revision: 'cba39050318896036dad9e0a6aedd8e0c1be8218'
review_loop_iteration: 0
followup_review_recommended: false
context: []
warnings: []
deferred: []
---

<intent-contract>

## Intent

**Problem:** Platform and system administrators need the ability to trigger long-running Cloud Run housekeeping jobs (such as NFS storage audits and orphaned workspace cleanups) immediately on-demand from the System Admin Portal without waiting for the nightly cron schedule, while enforcing strict Super Admin access controls and observing asynchronous execution lifecycle status.

**Approach:** Expose a secure `POST /api/v1/admin/jobs/{jobName}/run` endpoint requiring Parent-Project Super Admin authentication (`is_super_admin: true`), invoking the Google Cloud Run v2 API (`projects.locations.jobs.run`) with argument overrides (including `--dry-run` flag support) and returning HTTP `202 Accepted` with execution metadata (`status: "PENDING"`, `executionName`, `jobName`). Integrate trigger actions into the System Admin Portal (`/admin/nfs`) featuring progress spinners during in-flight dispatch and success notifications upon completion.

## Boundaries & Constraints

**Always:**
- Require `is_super_admin: true` from validated user claims context for `POST /api/v1/admin/jobs/{jobName}/run`. Reject non-super-admins and standard tenant users with HTTP `403 Forbidden`.
- Accept optional JSON payload `{ "dryRun": true }`.
- Translate `{jobName}` (e.g., `nfs-audit`, `hanging-cleanup`) and `dryRun` flag into Cloud Run Job container execution arguments: `[jobName]` or `[jobName, "--dry-run"]`.
- Return HTTP `202 Accepted` with JSON payload `{ "status": "PENDING", "executionName": "<execution-name>", "jobName": "{jobName}" }` on successful trigger initiation.
- Implement `CloudRunJobRunner` interface in the API tier with a Google Cloud Run v2 SDK implementation and a mockable test adapter.
- Update System Admin Portal (`ui/src/app/(dashboard)/admin/nfs/page.tsx` and related components) to provide on-demand trigger buttons ("Run Storage Audit Now" / "Run Cleanup"), active execution status badges with spinners while in progress, and toast/alert success notifications.
- Write comprehensive unit tests in `api/admin_job_handlers_test.go` and UI tests in `ui/src/app/(dashboard)/admin/nfs/page.test.tsx`.
- Ensure all Go tests (`go test ./...`) and UI tests (`npm test -- --run`) pass with zero errors.

**Block If:**
- Upstream requirements require modifying GCP IAM roles outside of Cloud Run Job invoker permissions or breaking existing auth middleware contracts.

**Never:**
- Never modify or write `sprint-status.yaml`.
- Never bypass Super Admin authentication checks for administrative job execution endpoints.
- Never block synchronous HTTP request handling on long-running batch job completion; always return HTTP 202 Accepted asynchronously.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| Super Admin Triggers Audit Job | `POST /api/v1/admin/jobs/nfs-audit/run` with `is_super_admin: true`, body `{ "dryRun": false }` | HTTP 202 Accepted, `{ "status": "PENDING", "executionName": "projects/.../locations/.../jobs/.../executions/...", "jobName": "nfs-audit" }` | HTTP 500 if Cloud Run SDK call fails |
| Super Admin Triggers Cleanup Job with Dry Run | `POST /api/v1/admin/jobs/hanging-cleanup/run` with `is_super_admin: true`, body `{ "dryRun": true }` | HTTP 202 Accepted with container args `["hanging-cleanup", "--dry-run"]` | HTTP 500 on Cloud Run client error |
| Non-Super-Admin Trigger Attempt | `POST /api/v1/admin/jobs/nfs-audit/run` with `is_super_admin: false` or missing claims | HTTP 403 Forbidden, `{ "status": "ERROR", "message": "Forbidden: Super Administrator access required" }` | Returns 403 immediately without calling Cloud Run API |
| Unauthenticated Trigger Request | `POST /api/v1/admin/jobs/nfs-audit/run` with missing or invalid auth token | HTTP 401 Unauthorized / HTTP 403 Forbidden | Handled by auth middleware |
| Empty or Invalid Job Name | `POST /api/v1/admin/jobs//run` or invalid path | HTTP 400 Bad Request / HTTP 404 Not Found | Returns error response |
| Malformed JSON Body | `POST /api/v1/admin/jobs/nfs-audit/run` with invalid JSON body | HTTP 400 Bad Request, `{ "status": "ERROR", "message": "Invalid request payload" }` | Returns 400 |
| Cloud Run API SDK Error | Cloud Run v2 API returns gRPC/HTTP error (e.g. Job not found, Quota exceeded) | HTTP 500 Internal Server Error, `{ "status": "ERROR", "message": "..." }` | Returns 500 with descriptive error message |
| UI On-Demand Trigger Execution | Super admin clicks "Run Storage Audit Now" / "Run Cleanup" in `/admin/nfs` | Sends POST request, shows spinner and active execution status badge while pending/running, displays success toast and updates data | Shows error alert if API returns error status |

</intent-contract>

## Code Map

- `api/admin_job_handlers.go` -- Defines `JobRunRequest`, `JobRunResponse`, `CloudRunJobRunner` interface, Google Cloud Run v2 SDK runner implementation, and `RunCloudRunJob` HTTP handler for `POST /api/v1/admin/jobs/{jobName}/run`.
- `api/admin_job_handlers_test.go` -- Unit tests verifying super admin RBAC, non-super-admin 403 rejection, Cloud Run SDK request formation, dry-run flag propagation, and error handling.
- `api/admin_handlers.go` -- Embeds `CloudRunJobRunner` on `AdminHandler` with `WithJobRunner` chaining helper.
- `api/routes.go` -- Registers `POST /api/v1/admin/jobs/{jobName}/run` and `OPTIONS /api/v1/admin/jobs/{jobName}/run` with `authMiddleware` and `superAdminAuthMiddleware`.
- `ui/src/app/(dashboard)/admin/nfs/page.tsx` -- System Admin Portal screen updated with on-demand job execution triggers, active status badges, loading spinners, and toast notifications.
- `ui/src/components/admin/NFSVolumeStatsCard.tsx` -- Volume health card with "Run Storage Audit Now" button and dry-run indicator.
- `ui/src/components/admin/HangingProjectsSection.tsx` -- Hanging projects section with "Run Cleanup" on-demand job execution trigger.
- `ui/src/app/(dashboard)/admin/nfs/page.test.tsx` -- Component tests verifying UI rendering, on-demand trigger button interactions, loading spinner states, and toast notifications.

## Tasks & Acceptance

**Execution:**
- `api/admin_job_handlers.go` -- Create `CloudRunJobRunner` interface, Cloud Run v2 SDK client adapter, and `RunCloudRunJob` handler -- Exposes secure on-demand job execution endpoint.
- `api/admin_handlers.go` -- Add `JobRunner` field and `WithJobRunner` method to `AdminHandler` -- Integrates job runner into administrative handler.
- `api/routes.go` -- Route `POST /api/v1/admin/jobs/{jobName}/run` and OPTIONS preflight through `superAdminAuthMiddleware` -- Connects API endpoint to routing mux.
- `api/admin_job_handlers_test.go` -- Implement unit tests covering RBAC, parameter translation, and Cloud Run API error responses -- Validates backend functionality and security boundaries.
- `ui/src/app/(dashboard)/admin/nfs/page.tsx` -- Update page with on-demand trigger controls, active job execution status indicator badge with spinner, and success notifications -- Delivers Super Admin UI experience.
- `ui/src/app/(dashboard)/admin/nfs/page.test.tsx` -- Author React Testing Library test suite for `/admin/nfs` job trigger interactions and badge display -- Validates UI state management.

**Acceptance Criteria:**
- Given an authenticated Super Administrator (`is_super_admin: true`), when they send `POST /api/v1/admin/jobs/{jobName}/run` (with optional payload `{ "dryRun": true }`), then the API handler calls the Google Cloud Run v2 API (`projects.locations.jobs.run`) to instantiate a new job execution with overridden args.
- Given a non-super-admin user attempting to call the endpoint, when requested, then the backend rejects the request with HTTP `403 Forbidden`.
- Given a successful trigger from the backend, when the execution starts, then the endpoint returns HTTP `202 Accepted` with `{ "status": "PENDING", "executionName": "projects/.../executions/...", "jobName": "{jobName}" }`.
- Given the System Admin Portal (`ui/src/app/(dashboard)/admin/nfs/page.tsx` or `/admin/jobs`), when viewed, then it provides an action button to "Run Storage Audit Now" / "Run Cleanup", displays an active execution status badge with spinner while in progress, and displays a success notification upon completion.
- Given unit tests in `api/admin_job_handlers_test.go`, when executed, then tests pass verifying Super Admin access control and Cloud Run SDK request formation.

## Spec Change Log

## Review Triage Log

### 2026-08-22 — Review pass 1
- intent_gap: 0
- bad_spec: 0
- patch: 3 (low 3)
- defer: 0
- reject: 12
- addressed_findings:
  - `[low]` `[patch]` Added `encodeURIComponent(jobName)` to `handleTriggerJob` in `page.tsx` for robust URI parameter escaping.
  - `[low]` `[patch]` Added auto-clearing timer for `activeJob` state in `page.tsx` to ensure UI controls reset gracefully.
  - `[low]` `[patch]` Added explicit test cases in `page.test.tsx` verifying dry-run checkbox toggles for both Storage Audit and Cleanup jobs.

## Design Notes

- `CloudRunJobRunner` interface abstracts Google Cloud Run v2 API (`google.golang.org/api/run/v2`), accepting `ctx context.Context, jobName string, dryRun bool` and returning `(*JobRunResponse, error)`.
- When `dryRun: true` is provided, arguments passed to the container override are `[]string{jobName, "--dry-run"}`; otherwise `[]string{jobName}`.
- Parent resource name in Cloud Run v2 is `projects/{project}/locations/{location}/jobs/{cloudRunJobName}` where `cloudRunJobName` defaults to `${prefix}-jobs` (or is configured via environment variable `MAINTENANCE_JOB_NAME` or defaults to `apf-stg-jobs` / `apf-prod-jobs`).
- In local/testing environments without GCP credentials, a mock runner or simulated execution can be injected or used as a fallback to ensure local developer workflows remain functional.

## Verification

**Commands:**
- `cd api && go test -v -run TestAdminJobHandlers ./...` -- expected: PASS
- `cd api && go test ./...` -- expected: ok google.com/apf/api ...
- `cd ui && npm test -- --run` -- expected: Test Files passed, Tests passed

## Auto Run Result

Status: done

### Summary of Implemented Change
- Implemented `POST /api/v1/admin/jobs/{jobName}/run` endpoint with optional `{ "dryRun": bool }` payload in `api/admin_job_handlers.go`, guarded by `superAdminAuthMiddleware` for Super Administrator access.
- Implemented `CloudRunJobRunner` interface with `GCPCloudRunJobRunner` using Google Cloud Run v2 SDK (`projects.locations.jobs.run`) and `SimulatedCloudRunJobRunner` fallback for local/test environments.
- Registered admin job routes in `api/routes.go` and wired `CloudRunJobRunner` onto `AdminHandler` in `api/admin_handlers.go` and `api/main.go`.
- Wrote unit tests in `api/admin_job_handlers_test.go` verifying super admin authorization, non-super-admin 403 rejection, missing claims rejection, empty job name validation, malformed JSON body handling, Cloud Run SDK request formation with dry-run args, and simulated execution.
- Updated System Admin Portal UI (`ui/src/app/(dashboard)/admin/nfs/page.tsx`, `NFSVolumeStatsCard.tsx`, `HangingProjectsSection.tsx`) with on-demand job execution triggers, dry-run toggles, active execution status badges with spinners, and toast notifications.
- Authored test suite in `ui/src/app/(dashboard)/admin/nfs/page.test.tsx` verifying UI rendering, trigger actions, toggles, loading states, and toast notifications.

### Files Changed
- `api/admin_job_handlers.go` -- New file implementing Cloud Run Job runner interface, GCP SDK adapter, and HTTP handler.
- `api/admin_job_handlers_test.go` -- New file containing unit tests for admin job execution handler.
- `api/admin_handlers.go` -- Added `JobRunner` field and `WithJobRunner` builder method on `AdminHandler`.
- `api/routes.go` -- Registered `POST /api/v1/admin/jobs/{jobName}/run` and OPTIONS preflight route.
- `api/main.go` -- Initialized `CloudRunJobRunner` and attached to `adminHandler`.
- `ui/src/app/(dashboard)/admin/nfs/page.tsx` -- Added on-demand job trigger handlers, active execution badges, and toast alerts.
- `ui/src/components/admin/NFSVolumeStatsCard.tsx` -- Added "Run Storage Audit Now" action button and dry-run toggle.
- `ui/src/components/admin/HangingProjectsSection.tsx` -- Added "Run Cleanup" action button and dry-run toggle.
- `ui/src/app/(dashboard)/admin/nfs/page.test.tsx` -- New test file verifying Admin NFS page on-demand job triggers and states.
- `docs/spec/implementation-artifacts/12-12-super-admin-on-demand-job-execution-api-and-ui.md` -- Story spec and execution record.

### Review Findings Breakdown
- Pass 1: 3 patches applied, 0 items deferred, 12 items rejected
- Total Patches Applied: 3
- Deferred Items: 0

### Follow-up Review Recommendation
- Patched counts by severity (Pass 1): high: 0, medium: 0, low: 3
- Formula score: 3 × 0 + 1 × 3 = 3 (< 5)
- Recommendation: `false`

### Verification Performed
- `cd api && go test -v -run TestAdminJobHandlers ./...` -- PASS (all 8 unit test suites passed)
- `cd api && go test ./...` -- PASS (all Go packages passed with 0 errors)
- `cd ui && npm test -- --run` -- PASS (all 37 test files and 293 tests passed with 0 errors)

### Residual Risks
- None. Super Admin RBAC and non-blocking asynchronous execution contracts are strictly enforced.

