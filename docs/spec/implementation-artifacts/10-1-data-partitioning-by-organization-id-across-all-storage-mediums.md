# Story 10.1: Data Partitioning by Organization-ID Across All Storage Mediums

Status: ready-for-dev

## Story

As a Platform Engineer,
I want the system to have the capability to partition data by organizationId across all data storage mediums including Firebase, Cloud Storage, and NFS,
so that customer teams operate within strict data isolation boundaries without cross-organization context bleed.

## Acceptance Criteria

1. **Given** a multi-organization workspace environment
   **When** a read or write operation occurs across Firebase, Cloud Storage, or NFS
   **Then** the underlying adapter MUST partition the data path or query exclusively by `organizationId`
   **And** prevent any cross-organization access across all storage boundaries

## Tasks / Subtasks

- [x] Define Organization ID context key and retrieval helpers (AC: 1)
  - [x] Add `OrganizationIDKey` to `api/core/types.go`
  - [x] Add `OrganizationID` field to `core.UserClaims`
  - [x] Implement `core.GetOrganizationID(ctx context.Context) (string, error)` helper
  - [x] Update auth token verification in `api/auth.go` to extract `organizationId` from claims and populate `UserClaims`
- [x] Refactor Firestore Stores to use organization-partitioned paths (AC: 1)
  - [x] Update `FirestoreProjectStore` to nest collections: `/organizations/{organizationId}/projects`
  - [x] Update `FirestoreSessionStore` to nest collections: `/organizations/{organizationId}/sessions`
  - [x] Update `FirestoreArtifactStore` to nest collections: `/organizations/{organizationId}/artifacts`
  - [x] Update `FirestoreAccessRequestStore` to nest collections: `/organizations/{organizationId}/access_requests`
  - [x] Update `FirestoreUserProjectStore` to nest collections: `/organizations/{organizationId}/user_project`
  - [x] Update `FirestoreIntegrationStore` to nest collections: `/organizations/{organizationId}/integrations`
  - [x] Update `FirestoreMessageStore` to nest collections: `/organizations/{organizationId}/sessions/{sessionId}/messages` (where sessions are also partitioned)
  - [x] Update `FirestoreUserStore` to nest collections: `/organizations/{organizationId}/users`
- [x] Refactor GCS Artifact Service to partition paths by organization (AC: 1)
  - [x] Update `GcsArtifactService` in `api/core/artifact_service.go` to use path: `organizations/{organizationId}/workspaces/{workspaceID}/docs/{path}`
  - [x] Retrieve `organizationId` from `ctx` in GCS operations
- [x] Refactor FileSystem Artifact Service to partition paths by organization (AC: 1)
  - [x] Update `FileSystemArtifactService` in `api/core/artifact_service.go` to use path: `{basePath}/organizations/{organizationId}/workspaces/{workspaceID}/{path}`
  - [x] Retrieve `organizationId` from `ctx` in filesystem operations
- [x] Refactor Git Client to partition workspaces on NFS by organization (AC: 1)
  - [x] Update `GitClient.getClonePath` in `api/git_client.go` to include `organizations/{organizationId}/` prefix: `{WorkspaceMountPath}/organizations/{organizationId}/workspaces/{userId}/{projectId}`
  - [x] Update `GitClient` lock paths to be organization-isolated
  - [x] Pass `ctx` to `getClonePath` and extract `organizationId`
- [x] Update and fix tests (AC: 1)
  - [x] Update database and integration tests to inject mock `organizationId` into context for all operations

## Dev Notes

### Firestore Nesting Structure
To enforce strict boundaries, all organization-specific collections are nested under the root `/organizations/{organizationId}` document.
- `/organizations/{organizationId}/projects/{projectId}`
- `/organizations/{organizationId}/sessions/{sessionId}`
- `/organizations/{organizationId}/artifacts/{artifactId}`
- `/organizations/{organizationId}/access_requests/{requestId}`
- `/organizations/{organizationId}/user_project/{userProjectId}`
- `/organizations/{organizationId}/integrations/{integrationId}`
- `/organizations/{organizationId}/users/{userId}`

Global collections (like `configs` and templates `workflows`) remain at the root level but are read-only for standard organization users.

### Cloud Storage (GCS) Partitioning
Paths in GCS MUST be prefixed with the organization ID:
`organizations/{organizationId}/workspaces/{workspaceID}/docs/{path}`

### Local Filesystem / NFS Partitioning
Local workspace directories (clones) MUST be separated by organization:
`{WorkspaceMountPath}/organizations/{organizationId}/workspaces/{userId}/{projectId}`

### References

- Multi-Tenancy Isolation Requirements: [docs/spec/planning-artifacts/prd.md#FR-8.1](docs/spec/planning-artifacts/prd.md)
- Architecture boundaries: [docs/spec/planning-artifacts/architecture.md#Data-&-Resource-Architecture](docs/spec/planning-artifacts/architecture.md)
- Epics breakdown: [docs/spec/planning-artifacts/epics.md#Story-10.1](docs/spec/planning-artifacts/epics.md)

## Dev Agent Record

### Agent Model Used

Gemini Next

### Debug Log References

### Completion Notes List
- Defined OrganizationIDKey in `api/core/types.go`
- Added OrganizationID to `core.UserClaims`
- Implemented `core.GetOrganizationID(ctx context.Context) (string, error)` helper
- Updated simulated token validation in `api/auth.go` (both Mock and Firebase authenticators) to parse `-organization-[organizationId]` suffix and populate OrganizationID
- Added unit tests in `api/core/types_test.go` and `api/auth_test.go`
- Created `api/db/organization_helper.go` with helpers to fetch organization-nested collections and queries
- Refactored Firestore stores to use organization-partitioned paths (`/organizations/{organizationId}/...`): Project, AccessRequest, Session, Artifact, UserProject, Integration, Message, User
- Created `api/db/test_helper_test.go` and updated database integration tests (`project_test.go`, `user_test.go`, `message_test.go`) to run under organization context and verify isolation
- Refactored `FileSystemArtifactService` and `GcsArtifactService` in `api/core/artifact_service.go` to partition paths by organization: `organizations/{organizationId}/workspaces/{workspaceID}/...`
- Refactored `GitClient` in `api/git_client.go` to partition workspaces on NFS by organization: `{WorkspaceMountPath}/organizations/{organizationId}/workspaces/{userId}/{projectId}` and isolate lock files
- Created `api/test_helper_test.go` and updated API integration tests (`git_client_integration_test.go`, `workspace_sync_test.go`, `workspace_delete_test.go`, `workspace_git_test.go`) to inject mock organization ID to satisfy partitioning requirements

### File List
- `api/core/types.go`
- `api/core/types_test.go`
- `api/auth.go`
- `api/auth_test.go`
- `api/db/organization_helper.go`
- `api/db/test_helper_test.go`
- `api/db/project.go`
- `api/db/project_test.go`
- `api/db/session.go`
- `api/db/artifact.go`
- `api/db/user.go`
- `api/db/user_test.go`
- `api/db/integration.go`
- `api/db/message.go`
- `api/db/message_test.go`
- `api/core/artifact_service.go`
- `api/git_client.go`
- `api/git_client_integration_test.go`
- `api/workspace_sync_test.go`
- `api/workspace_delete_test.go`
- `api/workspace_git_test.go`
- `api/test_helper_test.go`
