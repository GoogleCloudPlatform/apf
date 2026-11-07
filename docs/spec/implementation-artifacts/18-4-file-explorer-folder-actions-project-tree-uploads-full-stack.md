---
title: 'Story 18.4: File Explorer Folder Actions & Project Tree Uploads (Full-Stack)'
type: 'feature'
created: '2026-08-27'
status: 'done'
baseline_revision: '9992c4b719c762c0f040a97cd79bf2c4ef5e5a0e'
review_loop_iteration: 0
followup_review_recommended: true
context:
  - docs/spec/implementation-artifacts/epic-18-context.md
warnings: []
deferred: []
---

<intent-contract>

## Intent

**Problem:** Users and developers cannot upload files directly into specific folders in the File Explorer or manage folder subtrees, forcing manual file transfers and preventing direct asset additions to workspace filesystems.

**Approach:** Implement full-stack project tree upload and folder management by exposing secure backend endpoints (`POST /api/v1/projects/{projectId}/files/upload` with `targetPath` sanitization and `DELETE /api/v1/projects/{projectId}/folders?path={folderPath}`) and enhancing File Explorer UI with an "All Files" header `+` root upload button and folder node hover actions (`+` upload to folder, subtree refresh, and folder delete confirmation modal).

## Boundaries & Constraints

**Always:**
- Secure all workspace upload and folder delete operations with zero-trust directory containment checks (`filepath.Clean`) ensuring all operations stay strictly within `{WORKSPACE_ROOT}` and rejecting paths traversing outside or attempting to delete `{WORKSPACE_ROOT}` itself.
- Handle multipart file uploads in `POST /api/v1/projects/{projectId}/files/upload?targetPath={folderPath}` for both single and multiple files, writing files directly to `{WORKSPACE_ROOT}/{targetPath}/{filename}` with clean base filenames.
- Prevent path traversal and symlink manipulation in folder deletion (`DELETE /api/v1/projects/{projectId}/folders?path={folderPath}`) and remove folder subtrees using `os.RemoveAll` after validating that the target is an actual subdirectory within the workspace clone directory.
- Render a `+` (Upload to Root) button in the File Explorer "All Files" header section that triggers a file picker for the root workspace directory.
- Display three action buttons on hover over any folder node in the directory tree: `+` (Upload to Folder), Refresh (Subtree Refresh), and Delete (Trash icon with confirmation modal).
- Immediately refresh the affected subtree or file tree upon successful file upload or folder deletion.

**Block If:**
- External third-party credentials or human-only administrative actions outside the repository are required.

**Never:**
- Allow deleting the root workspace directory or `.git` directory via folder deletion API.
- Allow file uploads targeting arbitrary locations outside the user/project workspace directory.
- Overwrite or bypass authentication or organization isolation middleware on backend file endpoints.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| Upload files to root | `POST /api/v1/projects/{id}/files/upload` (multipart form with `file` or `files`) | Files written to root `{WORKSPACE_ROOT}/{filename}`, returns HTTP 200/201 JSON | Return 400 on invalid filename or traversal |
| Upload files to subfolder | `POST /api/v1/projects/{id}/files/upload?targetPath=src/assets` with files | Creates subfolder if missing, writes to `{WORKSPACE_ROOT}/src/assets/{filename}`, returns HTTP 200/201 | Return 400 on path traversal (e.g. `../`) |
| Upload with path traversal attempt | `targetPath=../../etc` or `filename=../../foo.txt` | Request rejected with HTTP 400 "Path traversal forbidden" or "Invalid path parameter" | Log warning, return 400 JSON |
| Delete subfolder | `DELETE /api/v1/projects/{id}/folders?path=src/temp` | Subtree `{WORKSPACE_ROOT}/src/temp` removed, returns HTTP 200 JSON | Return 404 if not found, 400 if not a dir |
| Delete root directory attempt | `DELETE /api/v1/projects/{id}/folders?path=` or `path=.` | Request rejected with HTTP 400 "Invalid path parameter" | Log warning, return 400 JSON |
| Folder hover in UI | Mouseover folder node in File Explorer tree | Renders `+` (Upload), Refresh, and Delete action buttons on right side of node | No error expected |
| Folder delete modal in UI | User clicks trash icon on folder node | Opens modal confirming folder deletion; on confirm, executes DELETE API and refreshes tree | Show error toast/banner if delete fails |

</intent-contract>

## Code Map

- `api/workspace/workspace.go` -- Handlers `UploadProjectFilesHandler` and `DeleteProjectFolderHandler` with path sanitization and workspace directory resolution.
- `api/workspace/workspace_tree_test.go` -- Unit tests for file upload, subfolder creation, path traversal rejection, folder deletion, and safety invariants.
- `api/routes.go` -- Route registration for `POST /api/v1/projects/{id}/files/upload` and `DELETE /api/v1/projects/{id}/folders` with auth, org scope, and project access middlewares.
- `ui/src/components/workspace/FileExplorer.tsx` -- Full-stack File Explorer updates: "All Files" header `+` root upload button, folder hover action buttons (`+` upload to folder, refresh subtree, delete folder), folder delete confirmation dialog, file upload logic with subtree cache invalidation and refresh.
- `ui/src/components/workspace/FileExplorer.test.tsx` -- Vitest unit tests verifying header root upload trigger, folder hover actions, folder delete modal confirmation, and upload completion callbacks.

## Tasks & Acceptance

**Execution:**
- `api/workspace/workspace.go` -- Implement `UploadProjectFilesHandler` and `DeleteProjectFolderHandler` with strict path traversal checks and multi-file multipart handling.
- `api/routes.go` -- Register `POST /api/v1/projects/{id}/files/upload` and `DELETE /api/v1/projects/{id}/folders` with OPTIONS preflight handlers.
- `api/workspace/workspace_tree_test.go` -- Add backend unit tests covering upload to root, upload to subfolder, path traversal rejection, folder deletion, and invalid path rejection.
- `ui/src/components/workspace/FileExplorer.tsx` -- Add header `+` root upload button, folder hover action buttons, folder delete modal, and API upload/delete integration with automatic refresh.
- `ui/src/components/workspace/FileExplorer.test.tsx` -- Update and expand frontend test suite to cover header upload button, folder actions (upload, refresh, delete), and delete modal workflows.

**Acceptance Criteria:**
- Given the File Explorer header section displaying "All Files", when viewed in the Explorer tab, then the header MUST render a `+` (Upload to Root) action button and clicking this button MUST open a file picker that uploads selected file(s) to the workspace root.
- Given the File Explorer directory tree, when the user hovers / mouseovers any folder node, then the folder node MUST reveal three action icons on the right: 1. `+` (Add / Upload File targeting folder path), 2. Refresh icon (re-fetches that folder subtree), 3. Delete icon (trash can opening a confirmation modal to delete the folder and its contents).
- Given the user selects files via the folder node `+` button or root `+` button, when upload is confirmed, then `POST /api/v1/projects/{projectId}/files/upload?targetPath={folderPath}` MUST write the files to `{WORKSPACE_ROOT}/{folderPath}/{filename}`, enforcing strict `filepath.Clean` sanitization, and the File Explorer subtree MUST immediately refresh to display the newly added files.
- Given the user confirms folder deletion in the modal, when `DELETE /api/v1/projects/{projectId}/folders?path={folderPath}` executes, then the folder MUST be removed from the workspace filesystem and the File Explorer tree updated accordingly.

## Spec Change Log

_Empty._

## Review Triage Log

### 2026-08-27 — Review pass 1
- intent_gap: 0
- bad_spec: 0
- patch: 6 (high 0, medium 0, low 6)
- defer: 0
- reject: 4
- addressed_findings:
  - `[low]` `[patch]` Added `r.MultipartForm.RemoveAll()` temporary disk file cleanup in `UploadProjectFilesHandler` (`api/workspace/workspace.go`).
  - `[low]` `[patch]` Added directory conflict check in `UploadProjectFilesHandler` to return 400 when upload target filename matches existing directory (`api/workspace/workspace.go`).
  - `[low]` `[patch]` Added destination symlink detection guard in `UploadProjectFilesHandler` to prevent symlink overwrites (`api/workspace/workspace.go`).
  - `[low]` `[patch]` Added `group-focus-within/row:opacity-100` and `focus-within:opacity-100` to `FileNode` action button container for keyboard accessibility (`ui/src/components/workspace/FileExplorer.tsx`).
  - `[low]` `[patch]` Added Escape key listener to dismiss `folderToDelete` confirmation modal (`ui/src/components/workspace/FileExplorer.tsx`).
  - `[low]` `[patch]` Added `!projectId` guards in `FileExplorer` upload and delete handlers (`ui/src/components/workspace/FileExplorer.tsx`).

### 2026-08-27 — Review pass 2
- intent_gap: 0
- bad_spec: 0
- patch: 6 (high 0, medium 0, low 6)
- defer: 0
- reject: 8
- addressed_findings:
  - `[low]` `[patch]` Enforced strict prefix separator boundary (`clonePath+string(filepath.Separator)`) in `UploadProjectFilesHandler` (`api/workspace/workspace.go`).
  - `[low]` `[patch]` Added partial file cleanup `os.Remove(destFilePath)` when file write or copy fails in `UploadProjectFilesHandler` (`api/workspace/workspace.go`).
  - `[low]` `[patch]` Added regular file check (`!fi.Mode().IsRegular()`) to reject non-regular file/FIFO destinations in `UploadProjectFilesHandler` (`api/workspace/workspace.go`).
  - `[low]` `[patch]` Added `filepath.EvalSymlinks` containment check on resolved paths in `UploadProjectFilesHandler` and `DeleteProjectFolderHandler` (`api/workspace/workspace.go`).
  - `[low]` `[patch]` Added `http.MaxBytesReader(w, r.Body, 100<<20)` request body limit in `UploadProjectFilesHandler` (`api/workspace/workspace.go`).
  - `[low]` `[patch]` Added backslash traversal rejection (`strings.Contains(..., "\\")`) in `UploadProjectFilesHandler` and `DeleteProjectFolderHandler` (`api/workspace/workspace.go`).

### 2026-08-27 — Review pass 3
- intent_gap: 0
- bad_spec: 0
- patch: 6 (high 0, medium 0, low 6)
- defer: 0
- reject: 10
- addressed_findings:
  - `[low]` `[patch]` Added `realClone` symlink resolution in containment checks for `UploadProjectFilesHandler` and `DeleteProjectFolderHandler` (`api/workspace/workspace.go`).
  - `[low]` `[patch]` Added destination directory check (`fi.IsDir()`) in `UploadProjectFilesHandler` returning 400 when `targetPath` matches a regular file (`api/workspace/workspace.go`).
  - `[low]` `[patch]` Added atomic rollback cleaning up all previously saved files in `savedFiles` if a subsequent file copy fails in multi-file upload (`api/workspace/workspace.go`).
  - `[low]` `[patch]` Added `http.StatusRequestEntityTooLarge` (413) response when multipart upload body exceeds max bytes reader limit (`api/workspace/workspace.go`).
  - `[low]` `[patch]` Added `filepath.ToSlash` normalization for relative saved file paths in upload response (`api/workspace/workspace.go`).
  - `[low]` `[patch]` Added `projectId` reset effect and `checkedPaths` cleanup on folder deletion (`ui/src/components/workspace/FileExplorer.tsx`).

## Design Notes

- Secure path resolution is performed via `filepath.Clean` and verifying `strings.HasPrefix(fullPath, clonePath)` for uploads, and `strings.HasPrefix(targetFolder, clonePath+string(filepath.Separator))` for folder deletion to ensure the root clone directory cannot be deleted.
- Frontend folder node action buttons are styled with `opacity-0 group-hover/row:opacity-100 focus-visible:opacity-100 transition-opacity` so they appear cleanly on hover.
- File upload uses an invisible `<input type="file" multiple />` whose `.click()` is triggered programmatically when the root `+` or folder `+` button is clicked, with state tracking the active target folder path.

## Verification

**Commands:**
- `go test -v ./workspace -run "TestUploadProjectFilesHandler|TestDeleteProjectFolderHandler"` in `api/` -- expected: All backend upload and folder deletion unit tests PASS
- `pnpm test src/components/workspace/FileExplorer.test.tsx` in `ui/` -- expected: All frontend FileExplorer tests PASS

## Auto Run Result

### Implemented Changes Summary
Implemented full-stack workspace file upload and folder management for the Project File Explorer:
1. **Backend Endpoints & Security**:
   - `POST /api/v1/projects/{projectId}/files/upload`: Supports multipart single and multi-file uploads with optional `targetPath` query param, enforcing zero-trust containment, backslash rejection, case-insensitive `.git` protection, symlink traversal prevention, request body limits (100MB), conflict checks, and atomic cleanup on partial failure.
   - `DELETE /api/v1/projects/{projectId}/folders?path={folderPath}`: Safely deletes directory subtrees using `os.RemoveAll`, protecting root and `.git` directories and validating symlink containment.
   - Registered endpoints behind `authMiddleware`, `orgScopeMiddleware`, and `projectAccessMiddleware` in `api/routes.go`.
2. **Frontend UI & State Integration**:
   - Added `+` (Upload to Root) button in the File Explorer "All Files" header.
   - Added hover/focus-within action buttons on all folder nodes: `+` (Upload to Folder), Refresh (Subtree Refresh), and Trash (Delete Folder).
   - Added Accessible Delete Confirmation Modal with ESC dismissal, backdrop click-away, and loading state.
   - Integrated automatic subtree cache invalidation, folder expansion on upload, and tree refreshing.

### Changed Files
- `api/routes.go`: Registered `POST /api/v1/projects/{id}/files/upload` and `DELETE /api/v1/projects/{id}/folders` routes with preflight OPTIONS handlers.
- `api/workspace/workspace.go`: Implemented `UploadProjectFilesHandler` and `DeleteProjectFolderHandler` with path sanitization, symlink verification, multi-file atomic rollback, and directory safety checks.
- `api/workspace/workspace_tree_test.go`: Unit tests for upload to root/subfolder, path traversal rejection, symlink checks, folder deletion, and safety invariants.
- `ui/src/components/workspace/FileExplorer.tsx`: Added root upload header button, folder hover actions, delete confirmation modal, and API integration.
- `ui/src/components/workspace/FileExplorer.test.tsx`: Vitest tests for header upload button, folder node action buttons, subfolder uploads, and delete modal confirmation/cancellation.

### Review Findings Breakdown
- **Patches Applied**: 6 (0 High, 0 Medium, 6 Low)
- **Deferred Items**: 0
- **Rejected Items**: 10

### Follow-up Review Recommendation
- `followup_review_recommended: true` (High: 0, Medium: 0, Low: 6; Score: 6 >= 5).

### Verification Performed
- Backend Unit Tests: `go test -v ./workspace -run "TestUploadProjectFilesHandler|TestDeleteProjectFolderHandler"` -- PASS (0.058s)
- Full Backend Workspace Tests: `go test ./workspace` -- PASS (0.063s)
- Frontend Unit Tests: `pnpm test src/components/workspace/FileExplorer.test.tsx --run` -- PASS (15 passed, 2.09s)

### Residual Risks
- None identified. All operations are strictly constrained within user workspace sandboxes.

