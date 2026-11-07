---
title: 'Story 12.10: Multi-Binary Container Packaging in api/Dockerfile & CI/CD Pipeline'
type: 'feature'
created: '2026-08-22'
status: 'done'
baseline_revision: 'f5c785c6018c070eca92a116389570e67e74aa43'
review_loop_iteration: 0
followup_review_recommended: false
context: []
warnings: []
deferred: []
---

<intent-contract>

## Intent

**Problem:** The `api/Dockerfile` currently only compiles `/apf-api` (the web server), preventing Cloud Run Jobs and batch execution environments from invoking the `apf-jobs` CLI without separate container build pipelines or duplicate container image registries.

**Approach:** Update `api/Dockerfile` to compile both `apf-api` and `apf-jobs` binaries during the builder stage, copy both binaries with executable permissions to the root directory in the runtime stage, preserve `/apf-api` as the default `ENTRYPOINT`, and verify CI/CD packaging compatibility with `infra/cicd/cloudbuild.yaml`.

## Boundaries & Constraints

**Always:**
- Compile the API web server to `/apf-api` using `CGO_ENABLED=0 GOOS=linux go build -o apf-api .` in the builder stage of `api/Dockerfile`.
- Compile the Jobs CLI to `/apf-jobs` using `CGO_ENABLED=0 GOOS=linux go build -o apf-jobs ./cmd/jobs` in the builder stage of `api/Dockerfile`.
- Copy both binaries (`/apf-api` and `/apf-jobs`) into the runtime container image root.
- Ensure executable permissions (`chmod +x`) on both `/apf-api` and `/apf-jobs` in the runtime stage.
- Preserve `ENTRYPOINT ["/apf-api"]` as the container default entrypoint.
- Ensure compatibility with `infra/cicd/cloudbuild.yaml` Kaniko build step.

**Block If:**
- Upstream requirements require splitting into separate container image repositories or altering the base Go/Alpine container runtime images.

**Never:**
- Never modify `sprint-status.yaml`.
- Never alter or break the default web server startup behavior when the container is run without command overrides.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|---|---|---|---|
| Container Web Server Run (Default) | `docker run <image>` | Starts HTTP web server on port 8081 via `/apf-api` entrypoint; listens for HTTP traffic | Exits with error code if port binding or configuration fails |
| Container Jobs Runner Run (CLI Override) | `docker run <image> /apf-jobs --list` | Executes `/apf-jobs` CLI dispatcher and prints available housekeeping jobs list to stdout; exits 0 | If invalid job name or flags passed, logs error and exits 1 |
| Container Jobs Runner Job Execution | `docker run <image> /apf-jobs nfs-audit` | Executes `nfs-audit` housekeeping job within container runtime environment | Exits 0 on job success, 1 on failure |
| Kaniko Container Build | `cloudbuild.yaml` build-api step with `api/Dockerfile` | Builds single multi-binary image containing both `/apf-api` and `/apf-jobs` without build failure | Build fails if any compilation or copy step errors |

</intent-contract>

## Code Map

- `api/Dockerfile` -- Multi-stage Dockerfile building `/apf-api` and `/apf-jobs` in builder stage and copying both binaries into runtime image.
- `api/cmd/jobs/main.go` -- Jobs CLI entrypoint compiled into `/apf-jobs`.
- `infra/cicd/cloudbuild.yaml` -- CI/CD build configuration using Kaniko executor on `api/Dockerfile`.

## Tasks & Acceptance

**Execution:**
- `api/Dockerfile` -- Update builder stage to compile `apf-jobs` alongside `apf-api`, copy `apf-jobs` into the runtime image, and ensure executable permissions -- Packages both binaries into unified container image.
- `infra/cicd/cloudbuild.yaml` -- Verify Kaniko build configuration for API container matches multi-binary Dockerfile context -- Ensures CI/CD pipeline builds multi-binary image.

**Acceptance Criteria:**
- Given `api/Dockerfile`, when the builder stage executes, then it compiles the API web server to `/out/apf-api` (`go build -o apf-api .`) and the Jobs CLI to `/out/apf-jobs` (`go build -o apf-jobs ./cmd/jobs`).
- Given the final runtime stage of `api/Dockerfile`, when assembled, then both `/apf-api` and `/apf-jobs` are copied into the root directory with executable permissions (`chmod +x`).
- Given the container default `ENTRYPOINT ["/apf-api"]`, when run without arguments, then it starts the HTTP web server on port 8081 as normal.
- Given the container run with entrypoint/args overridden (`docker run <image> /apf-jobs --list`), when executed, then it executes the CLI runner and lists available jobs.
- Given `infra/cicd/cloudbuild.yaml`, when Kaniko builds the API image, then the build succeeds and the resulting image in Artifact Registry contains both binaries.

## Spec Change Log

## Review Triage Log

### 2026-08-22 — Review pass 1
- intent_gap: 0
- bad_spec: 0
- patch: 0
- defer: 0
- reject: 12
- addressed_findings:
  - none

## Verification

**Commands:**
- `docker build -f api/Dockerfile -t apf-api-test .` -- expected: Docker build succeeds with both binaries compiled and copied into the image
- `docker run --rm --entrypoint /apf-jobs -e ENVIRONMENT=local -e BYPASS_IAP_VALIDATION=true -e SECRETS_ENCRYPTION_KEY=dev-32-bytes-secrets-key-apfninj apf-api-test --list` -- expected: Container executes `/apf-jobs --list` and prints available housekeeping jobs
- `cd api && CGO_ENABLED=0 GOOS=linux go build -o /tmp/apf-api . && CGO_ENABLED=0 GOOS=linux go build -o /tmp/apf-jobs ./cmd/jobs` -- expected: Both binaries build cleanly with zero compilation errors

## Auto Run Result

Status: done

### Summary of Implemented Change
- Updated `api/Dockerfile` builder stage to compile both `/apf-api` (API web server) and `/apf-jobs` (Housekeeping Jobs CLI).
- Updated `api/Dockerfile` runtime stage to copy both `/apf-api` and `/apf-jobs` binaries into the container root filesystem with explicit executable permissions (`chmod +x`).
- Preserved default `ENTRYPOINT ["/apf-api"]` so the container starts the HTTP API service on port 8081 when run without arguments, while allowing Cloud Run Jobs and operators to invoke `/apf-jobs` via command / entrypoint override.
- Verified packaging and CI/CD alignment with `infra/cicd/cloudbuild.yaml`.

### Files Changed
- `api/Dockerfile` -- Added multi-binary compilation (`apf-api` and `apf-jobs`) and root placement with executable permissions in runtime image.
- `docs/spec/implementation-artifacts/12-10-multi-binary-container-packaging-and-cicd-pipeline.md` -- Spec and review execution records.

### Review Findings Breakdown
- Pass 1: 0 patches applied, 0 items deferred, 12 items rejected (noise / out-of-scope hardening suggestions)
- Total Patches Applied: 0
- Deferred Items: 0

### Follow-up Review Recommendation
- Patched counts by severity (Pass 1): high: 0, medium: 0, low: 0
- Formula score: 3 × 0 + 1 × 0 = 0 (< 5)
- Recommendation: `false`

### Verification Performed
- `cd api && CGO_ENABLED=0 GOOS=linux go build -o /tmp/apf-api . && CGO_ENABLED=0 GOOS=linux go build -o /tmp/apf-jobs ./cmd/jobs` -- Passed (both binaries compiled cleanly with exit code 0)
- `docker build -f api/Dockerfile -t apf-api-test .` -- Passed (multi-stage Docker build completed with exit code 0)
- `docker run --rm --entrypoint /apf-jobs -e ENVIRONMENT=local -e BYPASS_IAP_VALIDATION=true -e SECRETS_ENCRYPTION_KEY=dev-32-bytes-secrets-key-apfninj apf-api-test --list` -- Passed (successfully executed CLI runner in container and listed registered jobs)
- `docker run --rm --entrypoint /apf-jobs -e ENVIRONMENT=local -e BYPASS_IAP_VALIDATION=true -e SECRETS_ENCRYPTION_KEY=dev-32-bytes-secrets-key-apfninj apf-api-test hanging-cleanup --dry-run` -- Passed (successfully executed hanging-cleanup dry-run scan inside container)
- `cd api && go test ./...` -- Passed (all unit and integration tests across api packages passed)

### Residual Risks
- None. Container image packaging preserves backward compatibility for the API service and enables Cloud Run Jobs invocation.

