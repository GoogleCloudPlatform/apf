---
title: 'Story 12.11: Cloud Run v2 Job & Cloud Scheduler Terraform Infrastructure'
type: 'feature'
created: '2026-08-22'
status: 'done'
baseline_revision: '167e91a285347117e3fea69dbfd6179e2b93db27'
review_loop_iteration: 0
followup_review_recommended: false
context: []
warnings: []
deferred: []
---

<intent-contract>

## Intent

**Problem:** Automated housekeeping routines (such as storage audits and orphaned workspace cleanup) must execute reliably on a nightly schedule without competing with or degrading the interactive API service container, requiring dedicated Cloud Run v2 Job and Cloud Scheduler infrastructure provisioned via Terraform with VPC networking and NFS volume mounts.

**Approach:** Author `infra/modules/housekeeping_jobs.tf` to provision a dedicated job scheduler service account with `roles/run.invoker` IAM role, a `google_cloud_run_v2_job` configured with VPC access, NFS `/workspaces` mount, `/apf-jobs` command, and secret environment variables, along with a `google_cloud_scheduler_job` configured for nightly execution (`0 2 * * *`) targeting the Cloud Run Job run endpoint via OAuth/OIDC authentication.

## Boundaries & Constraints

**Always:**
- Create `infra/modules/housekeeping_jobs.tf` provisioning `google_service_account.job_scheduler`, `google_project_iam_member.job_scheduler_run_invoker`, `google_cloud_run_v2_job.housekeeping`, and `google_cloud_scheduler_job.housekeeping_scheduler`.
- Configure `google_service_account.job_scheduler` with account ID `${var.prefix}-job-scheduler` and grant `roles/run.invoker` IAM binding on the project.
- Configure `google_cloud_run_v2_job` with `max_retries = 2`, `timeout = "1800s"`, `service_account = google_service_account.compute.email`.
- Configure direct VPC egress / VPC Access Connector with `egress = "ALL_TRAFFIC"` referencing `google_vpc_access_connector.connector.id`.
- Configure NFS volume mount for `/workspaces` referencing the Filestore NFS instance (`google_filestore_instance.nfs_instance.networks[0].ip_addresses[0]` and path `/workspaces`).
- Set container image to `${var.region}-docker.pkg.dev/${var.project_id}/${var.prefix}-repo/apf-api:latest`, command to `["/apf-jobs"]`, default args to `["nfs-audit"]`.
- Set environment variables for `GOOGLE_CLOUD_PROJECT`, `WORKSPACE_MOUNT_PATH`, `FIRESTORE_DB_NAME`, and CloudSQL password secret from Secret Manager (`google_secret_manager_secret.db_password.secret_id`).
- Configure `google_cloud_scheduler_job` to target `https://${var.region}-run.googleapis.com/v2/projects/${var.project_id}/locations/${var.region}/jobs/${google_cloud_run_v2_job.housekeeping.name}:run` on schedule `0 2 * * *` (nightly at 2:00 AM UTC) with OAuth token for the scheduler service account.
- Add `cloudscheduler.googleapis.com` to the default `apis` variable list in `infra/modules/variables.tf`.
- Ensure `terraform validate` passes with zero errors in `infra/staging` and `infra/prod`.

**Block If:**
- Upstream requirements require breaking changes to VPC CIDR ranges or decommissioning existing Compute service accounts.

**Never:**
- Never modify `sprint-status.yaml`.
- Never hardcode sensitive secrets in plaintext within Terraform definitions.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|---|---|---|---|
| Terraform Module Validation | `terraform validate` in `infra/staging` and `infra/prod` | Validates configuration with exit code 0 | Halts on syntax error, invalid provider schema, or missing reference |
| Cloud Run v2 Job Provisioning | `infra/modules/housekeeping_jobs.tf` applied | Provisions `${var.prefix}-housekeeping` Cloud Run Job with 1800s timeout, max 2 retries, compute service account, VPC egress ALL_TRAFFIC, NFS mount, and env vars | Rejects invalid container command or unsupported volume configuration |
| Cloud Scheduler Job Trigger | Scheduled cron trigger at `0 2 * * *` UTC | Sends authenticated HTTP POST request to Cloud Run v2 Job run endpoint using scheduler SA | Cloud Scheduler retries on transient network failures per retry config |
| Scheduler IAM Authorization | Cloud Scheduler calls Cloud Run v2 Job API | `roles/run.invoker` allows the scheduler service account to execute Cloud Run jobs | HTTP 403 Forbidden if IAM binding is missing or incorrect |

</intent-contract>

## Code Map

- `infra/modules/housekeeping_jobs.tf` -- New Terraform configuration defining the Cloud Run v2 Job, Cloud Scheduler Job, and scheduler service account with invoker permissions.
- `infra/modules/variables.tf` -- Includes `cloudscheduler.googleapis.com` in default GCP API list.
- `infra/modules/outputs.tf` -- Exports housekeeping job name, ID, and scheduler service account email.
- `infra/modules/filestore.tf` -- NFS Filestore instance providing `/workspaces` mount.
- `infra/modules/service_accounts.tf` -- Compute service account used as the runtime identity for the Cloud Run Job.
- `infra/modules/vpc_access_connector.tf` -- Serverless VPC access connector enabling Cloud Run Job to access internal VPC and Filestore.
- `infra/staging/main.tf` -- Staging root module consuming `infra/modules`.
- `infra/prod/main.tf` -- Production root module consuming `infra/modules`.

## Tasks & Acceptance

**Execution:**
- `infra/modules/variables.tf` -- Add `cloudscheduler.googleapis.com` to `apis` default list -- Enables Cloud Scheduler API in GCP projects.
- `infra/modules/housekeeping_jobs.tf` -- Create Terraform resource declarations for `google_service_account.job_scheduler`, `google_project_iam_member.job_scheduler_run_invoker`, `google_cloud_run_v2_job.housekeeping`, and `google_cloud_scheduler_job.housekeeping_scheduler` -- Provisions serverless housekeeping job execution and cron triggering.
- `infra/modules/outputs.tf` -- Add module outputs for `housekeeping_job_name`, `housekeeping_job_id`, and `job_scheduler_service_account_email` -- Exposes housekeeping job metadata for root modules and CI/CD pipelines.

**Acceptance Criteria:**
- Given `infra/modules/housekeeping_jobs.tf`, when applied, then it creates a `google_service_account` for the job scheduler with `roles/run.invoker` IAM binding on the project.
- Given the `google_cloud_run_v2_job` resource definition, when deployed, then it is configured with `max_retries = 2`, `timeout = "1800s"`, `service_account = google_service_account.compute.email`, VPC Access Connector / direct VPC egress set to `ALL_TRAFFIC`, NFS volume mount for `/workspaces`, image set to `${var.region}-docker.pkg.dev/${var.project_id}/${var.prefix}-repo/apf-api:latest`, command set to `["/apf-jobs"]`, default args `["nfs-audit"]`, environment variables for `GOOGLE_CLOUD_PROJECT`, `WORKSPACE_MOUNT_PATH`, and `FIRESTORE_DB_NAME`, and CloudSQL password secret from Secret Manager.
- Given the `google_cloud_scheduler_job` resource, when deployed, then it targets the Cloud Run v2 Job HTTP run endpoint (`https://${var.region}-run.googleapis.com/v2/projects/${var.project_id}/locations/${var.region}/jobs/${job_name}:run`) on schedule `0 2 * * *` (nightly at 2:00 AM UTC) with OIDC/OAuth authentication using the scheduler service account.
- Given `infra/staging` and `infra/prod`, when `terraform validate` is executed, then the configuration validates with zero syntax or provider schema errors.

## Spec Change Log

## Review Triage Log

### 2026-08-22 — Review pass 1
- intent_gap: 0
- bad_spec: 0
- patch: 0
- defer: 0
- reject: 14
- addressed_findings:
  - none

## Design Notes

- Cloud Run v2 Jobs require `template.task_spec` to configure `max_retries`, `timeout`, `service_account`, `vpc_access`, `containers`, and `volumes`.
- The NFS volume mount in Cloud Run v2 maps the Filestore IP (`google_filestore_instance.nfs_instance.networks[0].ip_addresses[0]`) to `/workspaces`.
- Cloud Scheduler invokes the Google Cloud Run v2 Jobs API endpoint `https://${var.region}-run.googleapis.com/v2/projects/${var.project_id}/locations/${var.region}/jobs/${job_name}:run` using HTTP POST with an OAuth token generated for the scheduler service account.

## Verification

**Commands:**
- `cd infra/staging && terraform validate` -- expected: Success! The configuration is valid.
- `cd infra/prod && terraform validate` -- expected: Success! The configuration is valid.

## Auto Run Result

Status: done

### Summary of Implemented Change
- Enabled `cloudscheduler.googleapis.com` in `infra/modules/variables.tf` default API list.
- Created `infra/modules/housekeeping_jobs.tf` provisioning:
  - Dedicated job scheduler service account (`${var.prefix}-job-scheduler`) with project-level `roles/run.invoker` IAM role binding.
  - Serverless `google_cloud_run_v2_job` named `${var.prefix}-housekeeping` configured with `max_retries = 2`, `timeout = "1800s"`, `service_account = google_service_account.compute.email`, VPC Access Connector with `egress = "ALL_TRAFFIC"`, NFS volume mount for `/workspaces` backed by Filestore, container image `${var.region}-docker.pkg.dev/${var.project_id}/${var.prefix}-repo/apf-api:latest`, command `["/apf-jobs"]`, default args `["nfs-audit"]`, environment variables (`GOOGLE_CLOUD_PROJECT`, `WORKSPACE_MOUNT_PATH`, `FIRESTORE_DB_NAME`), and `CLOUDSQL_PASSWORD` Secret Manager reference.
  - Cron trigger `google_cloud_scheduler_job` named `${var.prefix}-housekeeping-scheduler` running nightly at `0 2 * * *` UTC targeting the Cloud Run Job REST run endpoint with OAuth token authentication using the scheduler service account.
- Exported `housekeeping_job_name`, `housekeeping_job_id`, and `job_scheduler_service_account_email` in `infra/modules/outputs.tf`.
- Validated Terraform syntax and provider schema across both staging (`infra/staging`) and production (`infra/prod`) environments with zero errors.

### Files Changed
- `infra/modules/variables.tf` -- Added `cloudscheduler.googleapis.com` to `apis` default variable list.
- `infra/modules/housekeeping_jobs.tf` -- Created Terraform resources for Cloud Run v2 Job, Cloud Scheduler Job, and scheduler service account with IAM binding.
- `infra/modules/outputs.tf` -- Exported housekeeping job name, ID, and scheduler service account email.
- `docs/spec/implementation-artifacts/12-11-cloud-run-v2-job-and-cloud-scheduler-terraform-infrastructure.md` -- Spec and review execution records.

### Review Findings Breakdown
- Pass 1: 0 patches applied, 0 items deferred, 14 items rejected (noise / out-of-scope hardening suggestions / contradicted ACs)
- Total Patches Applied: 0
- Deferred Items: 0

### Follow-up Review Recommendation
- Patched counts by severity (Pass 1): high: 0, medium: 0, low: 0
- Formula score: 3 × 0 + 1 × 0 = 0 (< 5)
- Recommendation: `false`

### Verification Performed
- `cd infra/staging && terraform validate` -- Passed (`Success! The configuration is valid`)
- `cd infra/prod && terraform validate` -- Passed (`Success! The configuration is valid`)
- `cd infra && terraform fmt -check -diff modules/ staging/ prod/` -- Passed (zero diffs)

### Residual Risks
- None. Declarative infrastructure satisfies all ACs without regressions to existing VPC, Filestore, or Cloud SQL resources.

