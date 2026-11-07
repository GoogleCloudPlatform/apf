# Epic 12 Context: Administrative NFS Storage & Infrastructure Governance / Operations

<!-- Generated from planning artifacts. Regenerate with compile-epic-context if planning docs change. -->

## Goal

Empower platform and system administrators to maintain storage governance, automate long-running workspace housekeeping, prevent runaway NFS volume consumption, isolate multi-tenant workspace quotas, and execute batch maintenance tasks reliably via dedicated Cloud Run Jobs and Cloud Scheduler without degrading API server performance.

## Stories

- Story 12.7: NFS Volume and Folder-Level Metadata Management (System Admin)
- Story 12.8: Housekeeping Job Framework, Registry & CLI Dispatcher
- Story 12.9: Modular Housekeeping Jobs: NFS Storage Audit & Orphaned Workspace Cleanup
- Story 12.10: Multi-Binary Container Packaging in `api/Dockerfile` & CI/CD Pipeline
- Story 12.11: Cloud Run v2 Job & Cloud Scheduler Terraform Infrastructure
- Story 12.12: Super Admin On-Demand Job Execution API & Admin Portal Trigger UI
- Story 12.13: Super Admin Jobs Catalog & Cloud Run Execution History Backend API
- Story 12.14: Super Admin Jobs Dashboard & Execution History Portal UI (`/admin/jobs`)

## Requirements & Constraints

- **Super Admin Access Control**: Administrative NFS management endpoints and manual job execution APIs require Parent-Project Super Admin authentication (`is_super_admin: true`). Standard users and tenant org admins must be rejected with HTTP 403 Forbidden.
- **NFS Quota & Write Enforcement**: Workspace writes and Git operations from API services, background workers, or AI agents must be intercepted and rejected with HTTP 403 / `StorageQuotaExceededError` if the folder metadata has `isReadOnly: true` or disk usage exceeds `quotaSizeBytes`.
- **Multi-Tenant Workspace Directory Layout**: Workspace directories are structured under `{WorkspaceMountPath}/organizations/{organizationId}/workspaces/{userId}/{projectId}`.
- **Metadata Persistence**: Volume metrics and tenant folder metadata (quota limit, used bytes, file count, read-only flag, retention policy, last audited timestamp) are persisted in Firestore under `/admin_nfs_metadata/folders/{folderId}`.
- **Modular Batch CLI Dispatcher**: Housekeeping routines must execute as standalone Go CLI commands (`apf-jobs <job-name>`) with structured `slog` logging, interrupt handling (`SIGINT`, `SIGTERM`), and clean exit codes without competing with HTTP request handlers.
- **Dry-Run & Scan Resilience**: Destructive cleanup jobs must support a `--dry-run` mode to report candidate deletions without altering disk state. Permission issues, unreadable folders, or broken symlinks encountered during directory traversals must log warnings and continue scanning.
- **Automated Infrastructure**: Nightly scheduled runs execute via Cloud Scheduler targeting Cloud Run v2 Jobs over OIDC authentication with VPC access and NFS volume mounts.

## Technical Decisions

- **Job Interface & Registry**: Standard `Job` interface (`Name()`, `Description()`, `Run(ctx, cfg) error`) and thread-safe registry (`api/jobs/`) for registering and invoking batch operations.
- **Multi-Binary Container Packaging**: Multi-stage `api/Dockerfile` builds both the web server (`/apf-api`) and the job CLI (`/apf-jobs`) into a single container image, allowing Cloud Run Jobs to reuse the same deployed image.
- **Cloud Run v2 Job Infrastructure**: Terraform modules (`infra/modules/housekeeping_jobs.tf`) configure `google_cloud_run_v2_job` with 1800s timeout, direct VPC egress, NFS mounts for `/workspaces`, and `google_cloud_scheduler_job` triggers.
- **On-Demand Job Dispatching**: API handler calls Google Cloud Run v2 API (`projects.locations.jobs.run`) to trigger on-demand executions with parameter overrides, returning HTTP 202 Accepted.

## UX & Interaction Patterns

- **Admin NFS & Housekeeping Management UI (`/admin/nfs` or `/admin/jobs`)**:
  - Capacity overview card with mount health, total capacity, and storage consumption progress bars.
  - Searchable and filterable folder metadata table displaying tenant organizations, project paths, quota allocations, current usage, and lock states.
  - Edit Quota & Lock modal allowing Super Admins to adjust size limits, toggle read-only status, and save changes.
  - Trigger action buttons to run storage audits or workspace cleanups on demand, accompanied by progress spinners and completion alerts.

## Cross-Story Dependencies

- **Story 12.7** establishes the Firestore folder metadata repository, domain models, and storage write enforcement hooks.
- **Story 12.8** defines the `Job` interface, thread-safe registry, and standalone CLI dispatcher (`apf-jobs`).
- **Story 12.9** implements the modular audit and orphaned workspace cleanup jobs on top of Story 12.8's framework, updating Story 12.7's metadata store.
- **Story 12.10** packages `/apf-api` and `/apf-jobs` into the unified container image for deployment.
- **Story 12.11** provisions Terraform resources for Cloud Run v2 Jobs and Cloud Scheduler using the container image from Story 12.10.
- **Story 12.12** exposes the Super Admin on-demand run API and UI controls to trigger the Cloud Run Jobs provisioned in Story 12.11.
- **Story 12.13** implements `GET /api/v1/admin/jobs` (catalog discovery) and `GET /api/v1/admin/jobs/executions` (Cloud Run v2 SDK executions querying).
- **Story 12.14** builds the dedicated `/admin/jobs` System Admin screen for job catalog inspection, on-demand execution triggering, live status monitoring, and historical execution log viewing.
