---
title: 'Story 18.2: Chat File & Multimodal Image Upload UI (Frontend)'
type: 'feature'
created: '2026-08-27'
status: 'done'
baseline_revision: '5125df3e17d238636c44b83a6c5ce4efb1fe930c'
review_loop_iteration: 0
followup_review_recommended: true
context:
  - docs/spec/implementation-artifacts/epic-18-context.md
warnings: []
deferred: []
---

<intent-contract>

## Intent

**Problem:** Users cannot attach documents, code snippets, or multimodal images to prompt turns in the chat interface, preventing direct provision of visual references and contextual files to active AI agents.

**Approach:** Enable the chat composer attachment button and drag-and-drop file ingestion, stage files with removable thumbnail/icon badge chips, enforce 10MB limits with toast validation, upload attachments to the session upload API, and attach the resulting artifact metadata to the chat turn message payload.

## Boundaries & Constraints

**Always:**
- Enable the file attachment button (paperclip icon) in both standard bottom and centered empty-state chat composers.
- Support file picker and drag-and-drop staging for text/code formats (`.txt`, `.md`, `.pdf`, `.csv`, `.json`, `.yaml`, `.yml`, `.py`, `.go`, `.ts`, `.tsx`, `.js`, `.html`) and image formats (`.png`, `.jpg`, `.jpeg`, `.webp`, `.gif`, `.svg`).
- Enforce strict 10MB maximum file size limit per file with immediate toast validation on violation via `sonner` (`toast.error`).
- Render removable badge chips for staged files displaying thumbnail (for images) or file icon (for text/code), filename, formatted size, and an `X` remove button.
- On message submission with staged attachments, upload files via `POST /api/v1/projects/{projectId}/sessions/{sessionId}/artifacts/upload` before dispatching the message turn.
- Attach uploaded artifact metadata (`id`, `name`, `mimeType`, `size`, `storageType`, `storagePath`) and `artifactIds` array to the chat turn message payload.
- In `api.ts`, ensure `ApiClient` properly supports `FormData` payloads without forcing `Content-Type: application/json` or stringifying the form data.
- Update `Message` types in frontend and backend to support optional `artifacts` and `artifactIds`.
- Clean up object URLs on component unmount and file removal to prevent browser memory leaks.

**Block If:**
- External third-party credentials or human-only administrative actions outside the repository are required.

**Never:**
- Allow staging or uploading of files exceeding 10MB.
- Crash or block text-only chat messages when no files are attached.
- Modify or delete existing session chat history or unaffected workspace components.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| Attach valid image via button | User clicks paperclip and selects `diagram.png` (2MB) | Staged chip rendered with image thumbnail preview, filename, `2.0 MB`, and remove button | No error |
| Attach valid code file via drag-and-drop | User drops `main.go` (15KB) onto chat input area | Staged chip rendered with Go file icon badge, `main.go`, `15.0 KB`, and remove button | No error |
| Attach file exceeding 10MB | User selects `large_video.mp4` (15MB) | File is not staged; immediate toast error displayed ("File exceeds 10MB limit") | `toast.error` shown |
| Remove staged file | User clicks `X` on staged chip `diagram.png` | Chip removed from staged list; object URL revoked; input text preserved | No error |
| Send message with staged files | Message "Explain this diagram" with 1 staged image | Files uploaded to `/api/v1/projects/{p}/sessions/{s}/artifacts/upload`, message payload dispatched with `artifacts` and `artifactIds` | If upload fails, error toast shown, message send aborted, staged files preserved |
| Send message without attachments | User types text and clicks Send | Message dispatched normally with standard payload | Standard error handling |
| Multiple files selection | User selects `doc.md` (5KB) and `arch.png` (500KB) | Both chips rendered in staged list with correct icons and sizes | Non-conforming files rejected individually with toast |

</intent-contract>

## Code Map

- `ui/src/lib/api.ts` -- `ApiClient.request` handling for `FormData` bodies (avoid JSON stringification and default header override), plus `uploadSessionArtifact` helper function.
- `ui/src/lib/types.ts` -- `Message` interface definition updated to include `artifacts?: Artifact[]` and `artifactIds?: string[]`.
- `ui/src/hooks/useChatStream.ts` -- `useChatStream` hook `sendMessage` method updated to accept optional artifact metadata and pass `artifacts` / `artifactIds` in payload.
- `ui/src/components/workspace/ChatPane.tsx` -- Main chat component: paperclip button click handler, hidden file input, drag-and-drop dropzones, staged file chips list with thumbnails/icons and remove buttons, 10MB validation with `toast.error`, file upload orchestrator on submit, and message bubble attachment rendering.
- `ui/src/components/workspace/ChatPane.test.tsx` -- Unit tests for file attachment button, file picker selection, 10MB validation toast, staged chip removal, drag-and-drop, and message submission with attachments.
- `api/db/message.go` -- Go `Message` struct updated with `ArtifactIDs` and `Artifacts` fields for Firestore persistence.

## Tasks & Acceptance

**Execution:**
- `ui/src/lib/api.ts` -- Update `ApiClient` to properly handle `FormData` (bypass JSON stringification and allow browser boundary header) and export `uploadSessionArtifact(projectId, sessionId, file)`.
- `ui/src/lib/types.ts` -- Add `artifacts?: Artifact[]` and `artifactIds?: string[]` to the `Message` interface.
- `ui/src/hooks/useChatStream.ts` -- Update `sendMessage` to accept optional `artifacts?: Artifact[]` and attach them to the outgoing message payload and optimistic message.
- `api/db/message.go` -- Add `ArtifactIDs []string` and `Artifacts []Artifact` to `Message` struct with JSON and Firestore tags.
- `ui/src/components/workspace/ChatPane.tsx` -- Implement file upload trigger, drag-and-drop handler, 10MB size validation with toast errors, staged attachment chips with preview thumbnails/icons and remove buttons, file upload on send, and attachment display in message bubbles.
- `ui/src/components/workspace/ChatPane.test.tsx` -- Add comprehensive test coverage verifying attachment button rendering, file staging, file size limits, removal, upload dispatch, and error handling.

**Acceptance Criteria:**
- Given the chat panel in the GUI Workspace, when the user looks at the chat input area, then the UI displays a file attachment button (clip/paperclip icon) next to the message input.
- Given the user clicks the attachment button or drags-and-drops files onto the chat input area, when selecting files from their local device, then the file picker accepts text/code formats and image formats, and enforces a maximum file size limit of 10MB per file with immediate toast validation if exceeded.
- Given valid files are attached, when staged prior to sending, then the chat bar renders removable badge chips for each attached file displaying file icon / thumbnail, file name and formatted size, and an `X` remove button.
- Given attached files in the chat bar, when the user clicks Send, then the UI uploads the files to the session upload endpoint and attaches the resulting artifact IDs / metadata to the message turn payload.

## Spec Change Log

## Review Triage Log

### 2026-08-27 — Review pass
- intent_gap: 0
- bad_spec: 0
- patch: 5: (high 0, medium 0, low 5)
- defer: 0
- reject: 8
- addressed_findings:
  - `[low]` `[patch]` Added URL parameter encoding (`encodeURIComponent`) on project and session IDs in `uploadSessionArtifact`
  - `[low]` `[patch]` Sanitized and filtered `artifactIds` array mapping in `useChatStream.ts`
  - `[low]` `[patch]` Guarded `formatFileSize` against invalid inputs and normalized file extensions to lowercase for case-insensitive matching
  - `[low]` `[patch]` Added unit test coverage for `ApiClient` with `FormData` bodies and `uploadSessionArtifact` error handling
  - `[low]` `[patch]` Added backend unit test verifying `UserMessageHandler` deserializes `ArtifactIDs` and `Artifacts`

### 2026-08-27 — Follow-up Review pass 2
- intent_gap: 0
- bad_spec: 0
- patch: 6: (high 0, medium 0, low 6)
- defer: 0
- reject: 8
- addressed_findings:
  - `[low]` `[patch]` Added `isMountedRef` lifecycle guard in `ChatPane.tsx` to prevent state updates after unmount during file upload
  - `[low]` `[patch]` Guarded `isAllowedFileExtension` against filenames with trailing dots and no extension
  - `[low]` `[patch]` Added project ID and session ID validation before initiating session artifact uploads
  - `[low]` `[patch]` Added fallback title `"Attachment"` for artifact chips in message bubbles when both name and title are absent
  - `[low]` `[patch]` Guarded `isArtifactRelatedToTemplate` against empty/whitespace titles and trimmed string inputs
  - `[low]` `[patch]` Added unit test suite `ui/src/hooks/useChatStream.test.ts` verifying `sendMessage` attaches `artifacts` and `artifactIds` to outgoing payloads

## Design Notes

- **Image Thumbnails vs Code Icons**: Staged image files generate a transient object URL (`URL.createObjectURL(file)`) rendered in a miniature `<img>` tag with `rounded object-cover`, while non-image files use `FileIcon` component with file extension badges.
- **Drag and Drop UX**: Drag event listeners on the chat form provide clear visual feedback (e.g. `border-dashed border-primary/60 bg-primary/5`) while dragging files over the composer.
- **Upload Resilience**: Uploads are performed via `Promise.all` targeting `POST /api/v1/projects/{projectId}/sessions/{sessionId}/artifacts/upload`. If any upload fails, an informative toast is presented and staged files are retained so the user does not lose their staged context.

## Verification

**Commands:**
- `npm run test:run` in `ui/` -- expected: All UI unit tests pass including ChatPane attachment tests
- `cd api && go test ./...` -- expected: All backend unit tests pass

## Auto Run Result

### Summary of Implemented Change
- Implemented file attachment and multimodal image upload in the chat interface for Story 18.2.
- Enabled paperclip attachment button in both bottom and centered chat composers.
- Added drag-and-drop file staging with visual dropzone styling.
- Enforced 10MB per-file limit with immediate toast validation on oversized files.
- Added staged file badge chips with thumbnail previews (images) or file icons (code/text), formatted file sizes, and remove buttons with object URL cleanup.
- Integrated upload on submit to `POST /api/v1/projects/{projectId}/sessions/{sessionId}/artifacts/upload` with error toast handling and staged file retention on failure.
- Updated `ApiClient` in `ui/src/lib/api.ts` to support `FormData` payloads without forcing JSON headers.
- Updated `useChatStream` hook and Go backend `Message` model to support `artifacts` and `artifactIds`.
- Added attachment bubble rendering in message history.

### Files Changed
- `ui/src/lib/api.ts` — Handled `FormData` requests and added `uploadSessionArtifact` helper.
- `ui/src/lib/types.ts` — Updated `Message` and `Artifact` models to support attachments and hardened template matching.
- `ui/src/hooks/useChatStream.ts` — Updated `sendMessage` to support `artifacts` / `artifactIds` payload parameters.
- `ui/src/components/workspace/ChatPane.tsx` — Added paperclip button, hidden file input, drag-and-drop staging, 10MB limit enforcement, staged chips with object URL lifecycle management, and message bubble artifact chips.
- `ui/src/components/workspace/ChatPane.test.tsx` — Added comprehensive unit test coverage for file attachment UX.
- `ui/src/hooks/useChatStream.test.ts` — Added unit test coverage for hook payload serialization with artifacts.
- `ui/src/lib/api.test.ts` — Added unit tests for `ApiClient` with `FormData` and `uploadSessionArtifact`.
- `api/db/message.go` — Added `ArtifactIDs` and `Artifacts` fields to Go `Message` struct.
- `api/chat_test.go` — Added backend test verifying serialization/deserialization of message artifacts.

### Review Findings Breakdown
- Patches applied: 6 (high 0, medium 0, low 6)
- Items deferred: 0
- Items rejected: 8

### Follow-up Review Recommendation
- Recommended: `true` (Score: 6 from 6 low severity patches; formula: `3 × 0 + 1 × 6 = 6 >= 5`)

### Verification Performed
- `npm run test:run` in `ui/`: Passed (40 test files, 332 tests passed).
- `cd api && go test ./...`: Passed (all packages passed).

### Residual Risks
- None identified. Client-side and server-side tests pass with zero regressions.

