# Epic 18 Context: Unified File Upload, Multi-Environment Artifact Management, Multimodal Chat Context & File Explorer Management

<!-- Compiled from planning artifacts. Edit freely. Regenerate with compile-epic-context if planning docs change. -->

## Goal

Empower users to upload text files, documents, and images through both the chat interface and the project tree. Chat uploads leverage a pluggable ArtifactService (persisting to GCS in Staging/Production cloud environments and local filesystem for local development, initialized via mise.toml). Text documents are extracted into prompt context, while non-text files and images (.png, .jpg, .jpeg, .webp, .gif) are fed directly into the model as multipart/multimodal content parts. In the chat view, sessions with attached artifacts display an expandable chevron indicator and a dropdown displaying all session artifacts. Clicking an image artifact opens a dedicated Image Viewer in the preview pane that renders the image cleanly while suppressing code editor (Monaco) controls, edit tools, and specification approval buttons. The File Explorer provides intuitive folder actions: on mouseover, folder nodes reveal `+` (Upload file to folder), Refresh, and Delete icons, while the "All Files" header includes a `+` icon to upload directly to the project root.

## Stories

- Story 18.1: Multi-Tier `ArtifactService` & Configuration Infrastructure (Backend & Mise)
- Story 18.2: Chat File & Multimodal Image Upload UI (Frontend)
- Story 18.3: Multimodal & Text Prompt Context Ingestion (AI Orchestrator)
- Story 18.4: File Explorer Folder Actions & Project Tree Uploads (Full-Stack)
- Story 18.5: Chat Session Collapsible Chevron & Artifacts Dropdown (Frontend)
- Story 18.6: Image Artifact Preview & Media Viewer (Frontend)

## Requirements & Constraints

- **Multi-Tier Artifact Storage**: Store session chat uploads in Google Cloud Storage (`GCS_ARTIFACT_BUCKET`) for Staging/Production cloud deployments, and in the local filesystem (`LOCAL_ARTIFACT_PATH`, defaulting to `.foundry/artifacts` or mounted workspace) in local development mode (`APP_ENV=local`).
- **Metadata Persistence**: Record artifact records in Firestore (`artifacts` collection) upon upload completion with fields: `id` (UUID string), `projectId`, `sessionId`, `name` (original filename), `mimeType`, `size` (int64 bytes), `storageType` (`"gcs"` or `"local_file"`), `storagePath` (URI or local path), and `createdAt` (ISO 8601 timestamp).
- **Session Upload Endpoint**: `POST /api/v1/projects/{projectId}/sessions/{sessionId}/artifacts/upload` returns HTTP 201 with artifact metadata object.
- **Workspace Project Tree Upload & Delete**:
  - `POST /api/v1/projects/{projectId}/files/upload?targetPath={folderPath}` writes files directly to `{WORKSPACE_ROOT}/{folderPath}/{filename}` with strict path traversal sanitization (`filepath.Clean`) ensuring operations cannot escape `WORKSPACE_ROOT`.
  - `DELETE /api/v1/projects/{projectId}/folders?path={folderPath}` safely removes the folder subtree from the workspace filesystem.
- **File Type & Size Constraints**: File upload picker supports text/code files (`.txt`, `.md`, `.pdf`, `.csv`, `.json`, `.yaml`, `.py`, `.go`, `.ts`, `.tsx`, `.js`, `.html`) and image formats (`.png`, `.jpg`, `.jpeg`, `.webp`, `.gif`, `.svg`) with a 10MB per-file limit and client toast validation.
- **Prompt Context Ingestion**:
  - Text and code documents are extracted and injected into prompt context within formatted demarcation blocks (e.g., `[Attached File: filename.md]\n...content...`).
  - Image artifacts are formatted as multipart `Part` objects with `InlineData` (base64 and MIME type) or authenticated storage references, passed directly to vision-capable Gemini model calls.
  - Corrupted or unreadable files produce structured warnings in the turn payload without crashing the agent execution stream.
- **Latency & Reliability**: File upload and content extraction latency target < 3s for files up to 5MB, with zero data loss for active session attachments.

## Technical Decisions

- **Storage Backend Abstraction**: Pluggable `ArtifactService` decoupling cloud blob persistence (GCS client) from local development filesystem persistence (`LOCAL_ARTIFACT_PATH` scaffolded via `mise.toml`).
- **Dual Upload Paths**:
  - *Session Artifacts* (Chat): Tracked in Firestore and stored in scoped bucket/local artifact directories (`projects/{projectId}/sessions/{sessionId}/`).
  - *Workspace Files* (File Explorer): Directly written to the mounted workspace filesystem (`WORKSPACE_ROOT`) and immediately reflected in git/workspace explorer state.
- **Zero-Trust Workspace Path Security**: All directory and file upload endpoints strictly enforce path containment within `WORKSPACE_ROOT` via `filepath.Clean` and canonical path checks to prevent directory traversal attacks.
- **Multimodal Payload Ingestion**: AI orchestrator dynamically branches processing based on MIME type: text extraction for documentation vs. inline/multimodal `Part` generation for Gemini vision calls.

## UX & Interaction Patterns

- **Chat Attachment Staging**: Clip/paperclip icon and drag-and-drop support on the chat input area. Staged files render removable badge chips with thumbnails/icons, filenames, and formatted sizes before dispatch.
- **Session List Chevron & Dropdown**: Sessions in the history list containing attached artifacts display an expandable chevron indicator; toggling expands an accordion sub-list displaying all session artifacts with icons, filenames, sizes, and preview triggers.
- **Dedicated Image Preview Viewer**: Clicking image artifacts opens an `ImageViewer` component with zoom, pan, and responsive fitting, while hiding Monaco code editor toolbars, line numbers, syntax tools, and specification "Approve"/"Edit" buttons. Header displays filename, dimensions, size, and a download action.
- **File Explorer Hover & Header Controls**: Folder nodes reveal action icons on hover (`+` upload to folder, `Refresh` subtree, `Delete` trash can with confirmation modal). The "All Files" header includes a `+` button to upload directly to the workspace root.

## Cross-Story Dependencies

- **Story 18.1** provides the backend `ArtifactService` storage abstraction, configuration loading, Firestore metadata persistence, and session upload endpoint used by Story 18.2, Story 18.3, and Story 18.5.
- **Story 18.2** stages attachments in the chat UI, uploads them via Story 18.1's endpoint, and attaches artifact metadata to the chat turn payload for Story 18.3.
- **Story 18.3** uses artifact metadata and storage paths from Story 18.1/18.2 to extract text context or construct multimodal vision parts for Gemini.
- **Story 18.4** implements workspace project tree file/folder upload and delete APIs with File Explorer hover and header action controls.
- **Story 18.5** queries session artifact metadata registered in Story 18.1 to render the session list chevron and dropdown sub-list.
- **Story 18.6** consumes artifact URLs/storage paths from Story 18.1/18.5 to mount the dedicated `ImageViewer` preview pane for image artifacts.
