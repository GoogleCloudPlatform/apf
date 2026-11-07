# Story 12.7: NFS Volume and Folder-Level Metadata Management (System Admin)

Status: done

## Story

As a System Administrator (Super Admin),  
I want an administrative API and management interface for NFS volumes and folder-level workspace metadata (including storage quota enforcement, folder permission/lock flags, organization/tenant workspace bindings, retention policies, and volume health/capacity monitoring),  
so that system administrators can maintain storage governance, prevent runaway NFS volume consumption, isolate folder-level tenant metadata, and monitor volume health across all organization workspaces.

## Acceptance Criteria

1. **System Admin NFS Volume & Folder Metadata API**  
   **Given** a System Administrator authenticated at the Parent Project level with super-admin authorization (`is_super_admin === true`)  
   **When** they send a `GET` request to `/api/v1/admin/nfs/volumes` or `/api/v1/admin/nfs/folders`  
   **Then** the backend MUST return real-time NFS volume status (total capacity, available space, mount health) and an array of folder-level metadata records (organization ID, project ID, workspace path, quota limit, used bytes, file count, read-only flag, and retention policy).

2. **Folder-Level Quota & Access Control Configuration**  
   **Given** a System Administrator on the Admin Portal or sending a `PUT` request to `/api/v1/admin/nfs/folders/{folderId}`  
   **When** they update folder metadata attributes (e.g. setting `quotaSizeBytes`, toggling `isReadOnly`, or modifying `retentionPolicy`)  
   **Then** the backend MUST update and persist the folder metadata in Firestore under `/admin_nfs_metadata/folders/{folderId}`  
   **And** immediately return the updated metadata object with HTTP `200 OK`.

3. **Quota & Read-Only Write Interception on NFS**  
   **Given** an organization workspace on NFS whose folder metadata has `isReadOnly: true` or where disk usage exceeds `quotaSizeBytes`  
   **When** an AI agent, API handler (`FileSystemArtifactService`), or Git client (`GitClient`) attempts a file write, create, or commit operation on that workspace folder  
   **Then** the storage service / Git client MUST intercept the operation and reject it with an HTTP `403 Forbidden` / `StorageQuotaExceededError` before allocating disk space or modifying files.

4. **Background Storage Audit & Metadata Indexing Worker**  
   **Given** workspace directories configured under `{WorkspaceMountPath}/organizations/{organizationId}/workspaces/{userId}/{projectId}`  
   **When** the background NFS audit worker runs (periodically or on-demand via `POST /api/v1/admin/nfs/audit`)  
   **Then** it MUST scan workspace directory sizes, compute total bytes used and file counts per tenant workspace, and update the corresponding folder metadata records in Firestore with the calculated metrics and `lastAuditedAt` timestamp.

5. **System Admin Portal NFS Management UI**  
   **Given** a System Administrator accessing the System Admin Portal at `/admin/nfs`  
   **When** the page loads  
   **Then** the UI MUST render:  
   - An NFS Volume Health & Capacity overview card (capacity progress bar, mount status, total tenant folders).  
   - A searchable, filterable Folder Metadata Table (Org ID, Workspace Path, Quota, Usage %, Status, Retention).  
   - An Edit Quota & Lock Modal allowing admins to adjust storage limits, toggle read-only lock, or trigger a manual storage audit.

## Tasks / Subtasks

- [x] Task 1: Define NFS Volume & Folder Metadata Domain Models (`api/core/types.go` & `api/admin/nfs_types.go`) (AC: 1, 2)
  - [x] Subtask 1.1: Add `NFSVolumeStats` struct (`TotalBytes`, `AvailableBytes`, `UsedBytes`, `MountPath`, `IsHealthy`) to `api/core/types.go`.
  - [x] Subtask 1.2: Add `NFSFolderMetadata` struct (`FolderID`, `OrganizationID`, `ProjectID`, `WorkspacePath`, `QuotaSizeBytes`, `UsedSizeBytes`, `FileCount`, `IsReadOnly`, `RetentionPolicy`, `LastAuditedAt`) to `api/core/types.go`.
  - [x] Subtask 1.3: Define Firestore collection path constants (`/admin_nfs_metadata/folders`) and error types (`ErrNFSQuotaExceeded`, `ErrNFSFolderLocked`).

- [x] Task 2: Implement Firestore NFS Metadata Store (`api/db/nfs_store.go`) (AC: 1, 2, 4)
  - [x] Subtask 2.1: Implement `GetFolderMetadata(ctx context.Context, folderID string) (*core.NFSFolderMetadata, error)`.
  - [x] Subtask 2.2: Implement `UpdateFolderMetadata(ctx context.Context, folderID string, updates core.NFSFolderMetadataUpdate) error`.
  - [x] Subtask 2.3: Implement `ListFolderMetadata(ctx context.Context, filter core.NFSFolderFilter) ([]core.NFSFolderMetadata, error)`.
  - [x] Subtask 2.4: Implement `UpsertFolderAuditStats(ctx context.Context, folderID string, usedBytes int64, fileCount int) error`.

- [x] Task 3: System Admin NFS API Handlers & Super Admin Guard (`api/admin_nfs_handlers.go` & `api/routes.go`) (AC: 1, 2, 4)
  - [x] Subtask 3.1: Create `superAdminAuthMiddleware` in `api/auth.go` verifying parent-project token and `is_super_admin: true` claim.
  - [x] Subtask 3.2: Implement `GetNFSVolumeStatusHandler` (`GET /api/v1/admin/nfs/volumes`).
  - [x] Subtask 3.3: Implement `ListNFSFolderMetadataHandler` (`GET /api/v1/admin/nfs/folders`).
  - [x] Subtask 3.4: Implement `UpdateNFSFolderMetadataHandler` (`PUT /api/v1/admin/nfs/folders/{folderId}`).
  - [x] Subtask 3.5: Implement `TriggerNFSAuditHandler` (`POST /api/v1/admin/nfs/audit`).
  - [x] Subtask 3.6: Register new admin routes in `api/routes.go`.

- [x] Task 4: NFS Quota & Lock Enforcement in Storage Services (AC: 3)
  - [x] Subtask 4.1: Add `CheckFolderWritePermitted(ctx context.Context, workspacePath string)` helper in `api/db/nfs_store.go`.
  - [x] Subtask 4.2: Update `FileSystemArtifactService.WriteArtifact` (`api/core/artifact_service.go`) to invoke `CheckFolderWritePermitted` prior to writing files.
  - [x] Subtask 4.3: Update `GitClient` (`api/workspace/git_client.go`) methods (`Pull`, `Commit`, `Clone`) to verify folder quota and lock status before executing Git operations on NFS.

- [x] Task 5: Background Storage Audit Worker (`api/admin_nfs_audit_worker.go`) (AC: 4)
  - [x] Subtask 5.1: Implement recursive directory scanner that computes directory size in bytes and file count for `{WorkspaceMountPath}/organizations/{orgId}/workspaces/{userId}/{projectId}`.
  - [x] Subtask 5.2: Implement background audit runner that iterates over registered tenant workspace folders and updates Firestore metadata.

- [x] Task 6: System Admin NFS Management UI (AC: 5)
  - [x] Subtask 6.1: Create Admin NFS Page at `ui/src/app/(dashboard)/admin/nfs/page.tsx`.
  - [x] Subtask 6.2: Create `NFSVolumeStatsCard.tsx` component (`ui/src/components/admin/NFSVolumeStatsCard.tsx`).
  - [x] Subtask 6.3: Create `FolderMetadataTable.tsx` component (`ui/src/components/admin/FolderMetadataTable.tsx`).
  - [x] Subtask 6.4: Create `EditQuotaModal.tsx` component (`ui/src/components/admin/EditQuotaModal.tsx`) with quota input, read-only toggle, and submit handler.

- [x] Task 7: Unit & Integration Tests (AC: 1, 2, 3, 4, 5)
  - [x] Subtask 7.1: Add unit tests in `api/db/nfs_store_test.go` for CRUD operations on folder metadata.
  - [x] Subtask 7.2: Add API integration tests in `api/admin_nfs_handlers_test.go` validating super-admin access vs non-admin rejection.
  - [x] Subtask 7.3: Add storage enforcement tests in `api/core/artifact_service_nfs_test.go` verifying write blockage when quota is exceeded or folder is locked.

## Dev Notes

### Architecture & Multi-Tenancy Alignment

Workspace directories on NFS follow the multi-tenant partitioning schema established in Story 10.1:
`{WorkspaceMountPath}/organizations/{organizationId}/workspaces/{userId}/{projectId}`

Key configuration parameter:
- `WorkspaceMountPath`: Configured in `api/core/configuration.go` (`WORKSPACE_MOUNT_PATH`, default `~/workspaces`).

### Authentication & Authorization

All System Admin NFS endpoints MUST use Super Admin authentication as established in Story 11.8:
- Token MUST be issued at the Parent Project level (no `firebase.tenant` claim or root project level).
- Token MUST contain custom claim `is_super_admin: true`.
- Handlers MUST reject non-super-admin requests with `403 Forbidden`.

### Storage Enforcement Flow

```
User/AI Agent Write Request -> FileSystemArtifactService / GitClient
  -> CheckFolderWritePermitted(ctx, workspacePath)
    -> Lookup NFSFolderMetadata in Firestore by FolderID (or org/project path hash)
    -> If IsReadOnly == true -> Return ErrNFSFolderLocked (403 Forbidden)
    -> If UsedSizeBytes >= QuotaSizeBytes -> Return ErrNFSQuotaExceeded (403 Forbidden)
  -> Proceed with File Write / Git Operation
```

### Source Tree References

- Configuration: [api/core/configuration.go](api/core/configuration.go#L32)
- Routes Registration: [api/routes.go](api/routes.go#L33-L36)
- FileSystem Artifact Service: [api/core/artifact_service.go](api/core/artifact_service.go#L37)
- Git Client: [api/workspace/git_client.go](api/workspace/git_client.go#L68)
- Data Partitioning Reference: [docs/spec/implementation-artifacts/10-1-data-partitioning-by-organization-id-across-all-storage-mediums.md](docs/spec/implementation-artifacts/10-1-data-partitioning-by-organization-id-across-all-storage-mediums.md)
- Super Admin Auth Reference: [docs/spec/implementation-artifacts/11-8-parent-project-super-admin-authentication-and-global-resource-authorization.md](docs/spec/implementation-artifacts/11-8-parent-project-super-admin-authentication-and-global-resource-authorization.md)

## Dev Agent Record

### Agent Model Used

Gemini Next

### Debug Log References

### Completion Notes List

- Defined NFSVolumeStats, NFSFolderMetadata, NFSFolderFilter, NFSFolderMetadataUpdate structs, ErrNFSQuotaExceeded and ErrNFSFolderLocked in api/core/types.go.
- Implemented FirestoreNFSStore in api/db/nfs_store.go with GetFolderMetadata, UpdateFolderMetadata, ListFolderMetadata, UpsertFolderAuditStats, and CheckFolderWritePermitted methods.
- Implemented System Admin NFS API handlers (GetNFSVolumeStatus, ListNFSFolderMetadata, UpdateNFSFolderMetadata, TriggerNFSAudit) in api/admin_nfs_handlers.go and registered routes in api/routes.go.
- Implemented background NFS storage audit worker in api/admin_nfs_audit_worker.go to scan workspace directories under {WorkspaceMountPath}/organizations/... and update Firestore metrics.
- Enforced NFS quota and read-only lock in FileSystemArtifactService.WriteArtifact (api/core/artifact_service.go) and GitClient (api/workspace/git_client.go).
- Implemented System Admin NFS UI in ui/src/app/(dashboard)/admin/nfs/page.tsx, with NFSVolumeStatsCard, FolderMetadataTable, and EditQuotaModal components.
- Added and verified unit & integration tests in api/db/nfs_store_test.go, api/admin_nfs_handlers_test.go, api/core/artifact_service_nfs_test.go, and api/workspace/git_client_nfs_test.go.

### File List

- `api/core/types.go`
- `api/db/nfs_store.go`
- `api/db/nfs_store_test.go`
- `api/admin_nfs_handlers.go`
- `api/admin_nfs_handlers_test.go`
- `api/admin_nfs_audit_worker.go`
- `api/admin_handlers.go`
- `api/routes.go`
- `api/main.go`
- `api/core/artifact_service.go`
- `api/core/artifact_service_nfs_test.go`
- `api/workspace/git_client.go`
- `api/workspace/git_client_nfs_test.go`
- `ui/src/components/admin/NFSVolumeStatsCard.tsx`
- `ui/src/components/admin/EditQuotaModal.tsx`
- `ui/src/components/admin/FolderMetadataTable.tsx`
- `ui/src/app/(dashboard)/admin/nfs/page.tsx`
- `docs/spec/implementation-artifacts/sprint-status.yaml`
- `docs/spec/implementation-artifacts/12-7-nfs-volume-and-folder-level-metadata-management-system-admin.md`


## Story

As a System Administrator (Super Admin),  
I want an administrative API and management interface for NFS volumes and folder-level workspace metadata (including storage quota enforcement, folder permission/lock flags, organization/tenant workspace bindings, retention policies, and volume health/capacity monitoring),  
so that system administrators can maintain storage governance, prevent runaway NFS volume consumption, isolate folder-level tenant metadata, and monitor volume health across all organization workspaces.

## Acceptance Criteria

1. **System Admin NFS Volume & Folder Metadata API**  
   **Given** a System Administrator authenticated at the Parent Project level with super-admin authorization (`is_super_admin === true`)  
   **When** they send a `GET` request to `/api/v1/admin/nfs/volumes` or `/api/v1/admin/nfs/folders`  
   **Then** the backend MUST return real-time NFS volume status (total capacity, available space, mount health) and an array of folder-level metadata records (organization ID, project ID, workspace path, quota limit, used bytes, file count, read-only flag, and retention policy).

2. **Folder-Level Quota & Access Control Configuration**  
   **Given** a System Administrator on the Admin Portal or sending a `PUT` request to `/api/v1/admin/nfs/folders/{folderId}`  
   **When** they update folder metadata attributes (e.g. setting `quotaSizeBytes`, toggling `isReadOnly`, or modifying `retentionPolicy`)  
   **Then** the backend MUST update and persist the folder metadata in Firestore under `/admin_nfs_metadata/folders/{folderId}`  
   **And** immediately return the updated metadata object with HTTP `200 OK`.

3. **Quota & Read-Only Write Interception on NFS**  
   **Given** an organization workspace on NFS whose folder metadata has `isReadOnly: true` or where disk usage exceeds `quotaSizeBytes`  
   **When** an AI agent, API handler (`FileSystemArtifactService`), or Git client (`GitClient`) attempts a file write, create, or commit operation on that workspace folder  
   **Then** the storage service / Git client MUST intercept the operation and reject it with an HTTP `403 Forbidden` / `StorageQuotaExceededError` before allocating disk space or modifying files.

4. **Background Storage Audit & Metadata Indexing Worker**  
   **Given** workspace directories configured under `{WorkspaceMountPath}/organizations/{organizationId}/workspaces/{userId}/{projectId}`  
   **When** the background NFS audit worker runs (periodically or on-demand via `POST /api/v1/admin/nfs/audit`)  
   **Then** it MUST scan workspace directory sizes, compute total bytes used and file counts per tenant workspace, and update the corresponding folder metadata records in Firestore with the calculated metrics and `lastAuditedAt` timestamp.

5. **System Admin Portal NFS Management UI**  
   **Given** a System Administrator accessing the System Admin Portal at `/admin/nfs`  
   **When** the page loads  
   **Then** the UI MUST render:  
   - An NFS Volume Health & Capacity overview card (capacity progress bar, mount status, total tenant folders).  
   - A searchable, filterable Folder Metadata Table (Org ID, Workspace Path, Quota, Usage %, Status, Retention).  
   - An Edit Quota & Lock Modal allowing admins to adjust storage limits, toggle read-only lock, or trigger a manual storage audit.

## Tasks / Subtasks

- [ ] Task 1: Define NFS Volume & Folder Metadata Domain Models (`api/core/types.go` & `api/admin/nfs_types.go`) (AC: 1, 2)
  - [ ] Subtask 1.1: Add `NFSVolumeStats` struct (`TotalBytes`, `AvailableBytes`, `UsedBytes`, `MountPath`, `IsHealthy`) to `api/core/types.go`.
  - [ ] Subtask 1.2: Add `NFSFolderMetadata` struct (`FolderID`, `OrganizationID`, `ProjectID`, `WorkspacePath`, `QuotaSizeBytes`, `UsedSizeBytes`, `FileCount`, `IsReadOnly`, `RetentionPolicy`, `LastAuditedAt`) to `api/core/types.go`.
  - [ ] Subtask 1.3: Define Firestore collection path constants (`/admin_nfs_metadata/folders`) and error types (`ErrNFSQuotaExceeded`, `ErrNFSFolderLocked`).

- [ ] Task 2: Implement Firestore NFS Metadata Store (`api/admin/nfs_store.go`) (AC: 1, 2, 4)
  - [ ] Subtask 2.1: Implement `GetFolderMetadata(ctx context.Context, folderID string) (*core.NFSFolderMetadata, error)`.
  - [ ] Subtask 2.2: Implement `UpdateFolderMetadata(ctx context.Context, folderID string, updates core.NFSFolderMetadataUpdate) error`.
  - [ ] Subtask 2.3: Implement `ListFolderMetadata(ctx context.Context, filter core.NFSFolderFilter) ([]core.NFSFolderMetadata, error)`.
  - [ ] Subtask 2.4: Implement `UpsertFolderAuditStats(ctx context.Context, folderID string, usedBytes int64, fileCount int) error`.

- [ ] Task 3: System Admin NFS API Handlers & Super Admin Guard (`api/admin_nfs_handlers.go` & `api/routes.go`) (AC: 1, 2, 4)
  - [ ] Subtask 3.1: Create `superAdminAuthMiddleware` in `api/auth.go` verifying parent-project token and `is_super_admin: true` claim.
  - [ ] Subtask 3.2: Implement `GetNFSVolumeStatusHandler` (`GET /api/v1/admin/nfs/volumes`).
  - [ ] Subtask 3.3: Implement `ListNFSFolderMetadataHandler` (`GET /api/v1/admin/nfs/folders`).
  - [ ] Subtask 3.4: Implement `UpdateNFSFolderMetadataHandler` (`PUT /api/v1/admin/nfs/folders/{folderId}`).
  - [ ] Subtask 3.5: Implement `TriggerNFSAuditHandler` (`POST /api/v1/admin/nfs/audit`).
  - [ ] Subtask 3.6: Register new admin routes in `api/routes.go`.

- [ ] Task 4: NFS Quota & Lock Enforcement in Storage Services (AC: 3)
  - [ ] Subtask 4.1: Add `CheckFolderWritePermitted(ctx context.Context, workspacePath string)` helper in `api/admin/nfs_store.go`.
  - [ ] Subtask 4.2: Update `FileSystemArtifactService.WriteArtifact` (`api/core/artifact_service.go`) to invoke `CheckFolderWritePermitted` prior to writing files.
  - [ ] Subtask 4.3: Update `GitClient` (`api/workspace/git_client.go`) methods (`Pull`, `Commit`, `Clone`) to verify folder quota and lock status before executing Git operations on NFS.

- [ ] Task 5: Background Storage Audit Worker (`api/admin/nfs_audit_worker.go`) (AC: 4)
  - [ ] Subtask 5.1: Implement recursive directory scanner that computes directory size in bytes and file count for `{WorkspaceMountPath}/organizations/{orgId}/workspaces/{userId}/{projectId}`.
  - [ ] Subtask 5.2: Implement background audit runner that iterates over registered tenant workspace folders and updates Firestore metadata.

- [ ] Task 6: System Admin NFS Management UI (AC: 5)
  - [ ] Subtask 6.1: Create Admin NFS Page at `ui/src/app/(admin)/admin/nfs/page.tsx`.
  - [ ] Subtask 6.2: Create `NFSVolumeStatsCard.tsx` component (`ui/src/components/admin/NFSVolumeStatsCard.tsx`).
  - [ ] Subtask 6.3: Create `FolderMetadataTable.tsx` component (`ui/src/components/admin/FolderMetadataTable.tsx`).
  - [ ] Subtask 6.4: Create `EditQuotaModal.tsx` component (`ui/src/components/admin/EditQuotaModal.tsx`) with quota input, read-only toggle, and submit handler.

- [x] Task 7: Unit & Integration Tests (AC: 1, 2, 3, 4, 5)
  - [x] Subtask 7.1: Add unit tests in `api/admin/nfs_store_test.go` for CRUD operations on folder metadata.
  - [x] Subtask 7.2: Add API integration tests in `api/admin_nfs_handlers_test.go` validating super-admin access vs non-admin rejection.
  - [x] Subtask 7.3: Add storage enforcement tests in `api/core/artifact_service_nfs_test.go` verifying write blockage when quota is exceeded or folder is locked.

### Review Findings

- [x] [Review][Patch] `nfsStore` not attached to `FileSystemArtifactService` or `GitClient` in server startup [`api/main.go`:150]
- [x] [Review][Patch] `superAdminAuthMiddleware` defined but omitted from `routes.go` route registration [`api/admin_nfs_handlers.go`:20]
- [x] [Review][Patch] Firestore collection path uses nested `/admin_nfs_metadata/folders/folders/{folderId}` instead of `/admin_nfs_metadata/folders/{folderId}` [`api/db/nfs_store.go`:20]
- [x] [Review][Patch] Un-normalized workspace paths cause lookup failures in `CheckFolderWritePermitted` [`api/db/nfs_store.go`:81]
- [x] [Review][Patch] `CheckFolderWritePermitted` only checks existing usage against quota, allowing writes to overshoot limit [`api/db/nfs_store.go`:288]
- [x] [Review][Patch] `RunNFSAuditScan` logs but ignores `DiscoverWorkspaceFolders` errors, returning false HTTP 200 SUCCESS [`api/admin_nfs_audit_worker.go`:136]
- [x] [Review][Patch] `GetNFSVolumeStats` swallows `syscall.Statfs` error, returning uninformative zeroed volume stats [`api/db/nfs_store.go`:245]
- [x] [Review][Patch] `ScanWorkspaceDirectory` increments `fileCount` even when `d.Info()` fails on unreadable files/symlinks [`api/admin_nfs_audit_worker.go`:36]
- [x] [Review][Patch] `DeleteArtifact` in `FileSystemArtifactService` does not check NFS read-only lock [`api/core/artifact_service.go`:68]


## Dev Notes

### Architecture & Multi-Tenancy Alignment

Workspace directories on NFS follow the multi-tenant partitioning schema established in Story 10.1:
`{WorkspaceMountPath}/organizations/{organizationId}/workspaces/{userId}/{projectId}`

Key configuration parameter:
- `WorkspaceMountPath`: Configured in `api/core/configuration.go` (`WORKSPACE_MOUNT_PATH`, default `~/workspaces`).

### Authentication & Authorization

All System Admin NFS endpoints MUST use Super Admin authentication as established in Story 11.8:
- Token MUST be issued at the Parent Project level (no `firebase.tenant` claim or root project level).
- Token MUST contain custom claim `is_super_admin: true`.
- Handlers MUST reject non-super-admin requests with `403 Forbidden`.

### Storage Enforcement Flow

```
User/AI Agent Write Request -> FileSystemArtifactService / GitClient
  -> CheckFolderWritePermitted(ctx, workspacePath)
    -> Lookup NFSFolderMetadata in Firestore by FolderID (or org/project path hash)
    -> If IsReadOnly == true -> Return ErrNFSFolderLocked (403 Forbidden)
    -> If UsedSizeBytes >= QuotaSizeBytes -> Return ErrNFSQuotaExceeded (403 Forbidden)
  -> Proceed with File Write / Git Operation
```

### Source Tree References

- Configuration: [api/core/configuration.go](api/core/configuration.go#L32)
- Routes Registration: [api/routes.go](api/routes.go#L33-L36)
- FileSystem Artifact Service: [api/core/artifact_service.go](api/core/artifact_service.go#L37)
- Git Client: [api/workspace/git_client.go](api/workspace/git_client.go#L68)
- Data Partitioning Reference: [docs/spec/implementation-artifacts/10-1-data-partitioning-by-organization-id-across-all-storage-mediums.md](docs/spec/implementation-artifacts/10-1-data-partitioning-by-organization-id-across-all-storage-mediums.md)
- Super Admin Auth Reference: [docs/spec/implementation-artifacts/11-8-parent-project-super-admin-authentication-and-global-resource-authorization.md](docs/spec/implementation-artifacts/11-8-parent-project-super-admin-authentication-and-global-resource-authorization.md)

## Dev Agent Record

### Agent Model Used

Gemini Next

### Debug Log References

### Completion Notes List

- Story context created for 12-7 NFS Volume and Folder-Level Metadata Management (System Admin)

### File List

- `docs/spec/implementation-artifacts/12-7-nfs-volume-and-folder-level-metadata-management-system-admin.md`
