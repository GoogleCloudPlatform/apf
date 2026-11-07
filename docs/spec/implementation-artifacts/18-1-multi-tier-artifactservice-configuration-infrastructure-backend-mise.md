---
title: 'Story 18.1: Multi-Tier ArtifactService & Configuration Infrastructure (Backend & Mise)'
type: 'feature'
created: '2026-08-27'
status: 'done'
baseline_revision: '0d745418691bc2cb4cce8763cfbc5e5485073d14'
review_loop_iteration: 0
followup_review_recommended: false
context:
  - docs/spec/implementation-artifacts/epic-18-context.md
warnings: []
deferred: []
---

<intent-contract>

## Intent

**Problem:** The backend lacks multi-tier session artifact storage for chat uploads across cloud (GCS) and local development (local filesystem) environments, as well as the upload endpoint and metadata persistence in Firestore.

**Approach:** Extend `ArtifactService` and configuration to support multi-tier storage (`GCS_ARTIFACT_BUCKET` and `LOCAL_ARTIFACT_PATH`), configure `mise.toml` and env setup scripts, and implement `POST /api/v1/projects/{projectId}/sessions/{sessionId}/artifacts/upload` with Firestore metadata registration under the `artifacts` collection.

## Boundaries & Constraints

**Always:**
- Strict path sanitization on all uploaded filenames using `filepath.Base` to prevent path traversal.
- Enforce 10MB maximum file size limit with immediate `400 Bad Request` on violation.
- In Staging/Production cloud environments, persist artifacts to GCS bucket at `projects/{projectId}/sessions/{sessionId}/{filename}`.
- In Local mode, persist artifacts to `{LOCAL_ARTIFACT_PATH}/{projectId}/{sessionId}/{filename}` creating parent directories as needed.
- Record metadata in Firestore under `artifacts` collection with fields: `id` (UUID string), `projectId` (string), `sessionId` (string), `name` (original filename), `mimeType` (detected MIME string), `size` (int64 bytes), `storageType` (`"gcs"` or `"local_file"`), `storagePath` (GCS URI or local path), and `createdAt` (ISO 8601 timestamp).
- Return HTTP 201 Created with the artifact metadata JSON object upon successful upload.

**Block If:**
- External third-party credentials or human-only administrative actions outside the repository are required.

**Never:**
- Allow unsanitized file paths that escape the local artifact directory or root workspace.
- Hardcode storage bucket names or local directory paths.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| Upload valid text file (Local) | Multipart POST to `/api/v1/projects/p1/sessions/s1/artifacts/upload` with file `notes.md` | HTTP 201 Created, file written to `{LOCAL_ARTIFACT_PATH}/p1/s1/notes.md`, Firestore doc created with storageType `local_file` | No error |
| Upload valid image file (GCS) | Multipart POST with image `diagram.png` in cloud environment | HTTP 201 Created, file streamed to `gs://{bucket}/projects/p1/sessions/s1/diagram.png`, Firestore doc created with storageType `gcs` | No error |
| File exceeds 10MB limit | File payload > 10MB | HTTP 400 Bad Request with size error | Reject before write |
| Missing file part | Multipart request without `file` form field | HTTP 400 Bad Request: "File is required" | Reject with 400 |
| Path traversal attack in filename | Filename `../../etc/shadow` or `..\boot.ini` | Sanitized to `shadow` or `boot.ini`, written safely inside session folder | Path sanitized |
| GCS write error | GCS bucket unreachable or permission denied | HTTP 500 Internal Server Error | Log error and return 500 |

</intent-contract>

## Code Map

- `api/core/configuration.go` -- `Configuration` struct, `GCS_ARTIFACT_BUCKET` and `LOCAL_ARTIFACT_PATH` fields, `GetLocalArtifactPath()`
- `api/core/configuration_test.go` -- Tests for configuration loading and defaults
- `api/core/artifact_service.go` -- `ArtifactService` interface and implementations (`FileSystemArtifactService`, `GcsArtifactService`) with `SaveSessionArtifact`
- `api/core/artifact_service_test.go` -- Unit tests for `SaveSessionArtifact` under local and GCS backends
- `api/db/artifact.go` -- `Artifact` struct with session artifact metadata fields and Firestore store methods
- `api/session.go` -- `uploadSessionArtifactHandler` method on `SessionHandler` handling multipart stream, MIME detection, and metadata persistence
- `api/routes.go` -- Route registration for `POST /api/v1/projects/{id}/sessions/{sessionId}/artifacts/upload`
- `mise.toml` -- Local development environment variables initialization for `LOCAL_ARTIFACT_PATH`
- `scripts/setup_env.sh` -- Script propagating `LOCAL_ARTIFACT_PATH` and `GCS_ARTIFACT_BUCKET` to service `.env` files
- `config.env.template` -- Master environment template with `LOCAL_ARTIFACT_PATH` default

## Tasks & Acceptance

**Execution:**
- `api/core/configuration.go` -- Add `GcsArtifactBucket` and `LocalArtifactPath` to `Configuration`, with `GetLocalArtifactPath()` defaulting to `.foundry/artifacts`.
- `mise.toml` -- Set `LOCAL_ARTIFACT_PATH = ".foundry/artifacts"` in environment and task configurations.
- `scripts/setup_env.sh` & `config.env.template` -- Ensure `LOCAL_ARTIFACT_PATH` is defined and generated into `api/.env` and `ai/.env`.
- `api/core/artifact_service.go` -- Add `SaveSessionArtifact(ctx context.Context, projectId, sessionId, filename string, r io.Reader) (storageType string, storagePath string, bytesWritten int64, err error)` to `ArtifactService` and implement for `FileSystemArtifactService` and `GcsArtifactService`.
- `api/db/artifact.go` -- Add session artifact fields (`SessionId`, `Name`, `MimeType`, `Size`, `StorageType`, `StoragePath`, `CreatedAt`) to `Artifact` struct and helper methods on `FirestoreArtifactStore` / `ArtifactStore`.
- `api/session.go` -- Implement `uploadSessionArtifactHandler` on `SessionHandler` using `ArtifactService` and `ArtifactStore`.
- `api/routes.go` -- Register `POST /api/v1/projects/{id}/sessions/{sessionId}/artifacts/upload` and OPTIONS preflight with auth and org scope middleware.
- `api/artifact_upload_test.go` -- Add end-to-end tests for artifact upload endpoint, MIME detection, 10MB limit, and Firestore metadata recording.

**Acceptance Criteria:**
- Given the application configuration in `api/core/configuration.go`, when environment variables are loaded, then the system reads `GCS_ARTIFACT_BUCKET` falling back gracefully if unset, and in local mode reads `LOCAL_ARTIFACT_PATH` defaulting to `.foundry/artifacts`.
- Given the local developer setup in `mise.toml`, when local setup tasks execute, then the environment configuration initializes and exports `LOCAL_ARTIFACT_PATH` along with `WORKSPACE_MOUNT_PATH`.
- Given an authenticated user uploading a file via `POST /api/v1/projects/{projectId}/sessions/{sessionId}/artifacts/upload`, when running in Staging or Production, then the `ArtifactService` streams and persists the file to the configured GCS bucket under `projects/{projectId}/sessions/{sessionId}/{filename}`, and when running in Local mode, writes to `{LOCAL_ARTIFACT_PATH}/{projectId}/{sessionId}/{filename}`.
- Given a successful file upload to the `ArtifactService`, when the write completes, then the backend records an artifact metadata document in Firestore under `artifacts` collection containing `id`, `projectId`, `sessionId`, `name`, `mimeType`, `size`, `storageType`, `storagePath`, and `createdAt`, and the API returns HTTP 201 with the created artifact metadata object.

## Spec Change Log

## Review Triage Log

### 2026-08-27 — Review pass
- intent_gap: 0
- bad_spec: 0
- patch: 3: (high 0, medium 0, low 3)
- defer: 0
- reject: 8
- addressed_findings:
  - `[low]` `[patch]` Added empty file guard (`header.Size == 0`) returning HTTP 400 Bad Request
  - `[low]` `[patch]` Enhanced MIME type detection with extension mapping before content sniffing
  - `[low]` `[patch]` Added comprehensive unit tests covering empty file rejection and GCS mock persistence

## Verification

**Commands:**
- `cd api && go test ./...` -- expected: All unit and integration tests pass without error
- `cd api && go vet ./...` -- expected: No vet errors

## Auto Run Result

Status: done

### Summary of Implemented Change
Implemented multi-tier artifact storage and configuration infrastructure for Story 18.1:
1. **Configuration**: Added `GCS_ARTIFACT_BUCKET` and `LOCAL_ARTIFACT_PATH` with default resolution (`.foundry/artifacts`) in `api/core/configuration.go` and corresponding tests.
2. **Environment & Mise**: Updated `mise.toml`, `config.env.template`, and `scripts/setup_env.sh` to export and initialize `LOCAL_ARTIFACT_PATH`.
3. **Pluggable Storage Drivers**: Extended `ArtifactService` interface with `SaveSessionArtifact` supporting streaming GCS object persistence and sanitized local filesystem persistence.
4. **Data Model**: Updated `db.Artifact` with session artifact metadata fields (`SessionId`, `Name`, `MimeType`, `Size`, `StorageType`, `StoragePath`, `CreatedAt`).
5. **API Upload Handler & Routing**: Implemented `uploadSessionArtifactHandler` on `SessionHandler` and registered `POST /api/v1/projects/{id}/sessions/{sessionId}/artifacts/upload` with auth and project scope middleware, enforcing 10MB limits, empty file rejection, path sanitization, and Firestore registration.
6. **Testing**: Added comprehensive unit and integration tests in `api/artifact_upload_test.go` and `api/core/configuration_test.go`.

### Files Changed
- `api/core/configuration.go` -- Added `GcsArtifactBucket` and `LocalArtifactPath` configuration fields and `GetLocalArtifactPath()` helper.
- `api/core/configuration_test.go` -- Added unit tests for artifact path configuration and defaults.
- `api/core/artifact_service.go` -- Added `SaveSessionArtifact` and `WithLocalArtifactPath` to `FileSystemArtifactService` and `GcsArtifactService`.
- `api/core/artifact_service_test.go` -- Added unit tests for session artifact persistence.
- `api/db/artifact.go` -- Added session metadata fields to `Artifact` struct.
- `api/session.go` -- Implemented `uploadSessionArtifactHandler` with 10MB size limits, MIME detection, and Firestore persistence.
- `api/routes.go` -- Registered `POST /api/v1/projects/{id}/sessions/{sessionId}/artifacts/upload` and OPTIONS preflight.
- `api/artifact_upload_test.go` -- Added unit test suite for upload endpoint.
- `mise.toml` -- Configured `LOCAL_ARTIFACT_PATH` in environment and task configs.
- `config.env.template` -- Added `LOCAL_ARTIFACT_PATH` default.
- `scripts/setup_env.sh` -- Added `LOCAL_ARTIFACT_PATH` setup and propagation to service `.env` files.
- `docs/spec/implementation-artifacts/epic-18-context.md` -- Compiled Epic 18 context documentation.
- `docs/spec/implementation-artifacts/18-1-multi-tier-artifactservice-configuration-infrastructure-backend-mise.md` -- Story specification and results.

### Review Findings Breakdown
- Patches applied: 3 (Empty file validation, MIME type extension resolution, test coverage expansion)
- Items deferred: 0
- Items rejected: 8 (Features outside Story 18.1 scope: future stories 18.3, 18.4, 18.5 handle prompt ingestion, project tree explorer uploads, and session list dropdowns)

### Follow-up Review Recommendation
- Recommendation: `false` (3 low-severity patches applied, score 3 < 5).

### Verification
- `cd api && go test ./...`: PASS (All packages passed successfully).
- `cd api && go vet ./...`: PASS (No vet issues detected).

### Residual Risks
- None.
