---
title: 'Story 18.5: Chat Session Collapsible Chevron & Artifacts Dropdown (Frontend)'
type: 'feature'
created: '2026-08-27'
status: 'done'
baseline_revision: 'eb183747711caa135114b2e7116fe0f92c2a7328'
review_loop_iteration: 1
followup_review_recommended: false
context:
  - docs/spec/implementation-artifacts/epic-18-context.md
warnings: []
deferred: []
---

<intent-contract>

## Intent

**Problem:** Users cannot view or access artifacts attached to past or active chat sessions in the session history sidebar, making it difficult to browse, verify, or preview attachments across conversations.

**Approach:** In the Workspace Explorer Chat Sessions list, detect sessions with registered artifacts, render an expandable chevron indicator on the right side of those session items, toggle expand/collapse state on click with smooth CSS transitions, and render a nested sub-list of session artifacts displaying file type icons/thumbnails, filenames, formatted file sizes, and action triggers to open/preview artifacts in the preview pane.

## Boundaries & Constraints

**Always:**
- Filter `liveArtifacts` by `sessionId === session.id` to identify associated artifacts for each chat session.
- Render an expandable chevron indicator on the right of any session item that has one or more associated artifacts.
- Never render a chevron indicator on session items that have zero associated artifacts.
- When the user clicks the session item or clicks the chevron indicator, toggle the session's expand/collapse state with a smooth CSS transition.
- When expanded, render a dropdown / nested sub-list containing all artifacts associated with that session.
- For each rendered artifact item in the dropdown/sub-list, display:
  1. File type icon (or image icon/thumbnail for image files: `.png`, `.jpg`, `.jpeg`, `.webp`, `.gif`, `.svg` or MIME starting with `image/`).
  2. Filename (`art.name || art.title`).
  3. Formatted file size (e.g. `(2.4 MB)`, `15.0 KB`) using shared `formatFileSize` utility.
  4. An action trigger / button (and row click) to open/preview the artifact in the workspace preview pane via `setActiveArtifactTitle`.
- Maintain full compatibility with existing session switching and deletion workflows.
- Export and share `formatFileSize` utility in `ui/src/lib/utils.ts`.
- Ensure backend `GetArtifactContentHandler` seamlessly serves content for session artifacts when requested by artifact ID or storage path.

**Block If:**
- External third-party credentials or human-only administrative actions outside the repository are required.

**Never:**
- Break existing session switching (`handleSwitchSession`) or deletion (`setSessionToDelete`) behavior.
- Show an expandable chevron or empty dropdown container for sessions without artifacts.
- Mutate `sprint-status.yaml` or any orchestrator-owned state.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| Session with 0 artifacts | Session item in list, `liveArtifacts` has no entries with `sessionId === s.id` | Session item rendered without chevron icon; clicking switches session | No error |
| Session with 2 artifacts (collapsed) | Session item in list, `liveArtifacts` has 2 matching entries | Session item renders chevron icon on right; sub-list is collapsed/hidden | No error |
| Click chevron or session with artifacts | User clicks chevron button or session row | Expands sub-list showing the 2 artifacts with smooth CSS transition | No error |
| Expanded artifact item display | Sub-list rendered with text and image artifacts | Text file displays file icon, filename, size; Image file displays image icon/thumbnail, filename, size | Fallback gracefully if size is missing |
| Click artifact preview action | User clicks artifact item or preview action trigger | Calls `setActiveArtifactTitle(art.name || art.title || art.path || art.id)` to open preview pane | No error |
| Click delete session button | User clicks trash icon on session item | Opens session deletion confirmation dialog; does not toggle artifact dropdown | Prevent event propagation (`e.stopPropagation()`) |
| Toggle collapse on expanded session | User clicks chevron or session again | Collapses sub-list smoothly | No error |

</intent-contract>

## Code Map

- `ui/src/lib/utils.ts` -- Add and export `formatFileSize(bytes: number): string` utility function.
- `ui/src/components/workspace/ChatPane.tsx` -- Update to import and reuse `formatFileSize` from `@/lib/utils`.
- `ui/src/components/workspace/CanvasPane.tsx` -- Ensure `targetArtifact` resolution checks `a.name === activeArtifactTitle` alongside title, id, and path.
- `ui/src/app/(dashboard)/projects/[id]/workspace/_components/ExplorerPanel.tsx` -- Add collapsible chevron indicator, expand/collapse state tracking, smooth dropdown animation, and session artifacts nested sub-list with file icons, filenames, formatted sizes, and preview triggers.
- `ui/src/app/(dashboard)/projects/[id]/workspace/_components/ExplorerPanel.test.tsx` -- Unit tests verifying chevron rendering on sessions with artifacts, omission on sessions without artifacts, expand/collapse toggling, artifact items rendering, and preview trigger execution.
- `ui/src/components/workspace/CanvasPane.test.tsx` -- Unit test for session artifact matching and rendering by name and storagePath.
- `api/workspace/workspace.go` -- Ensure `GetArtifactContentHandler` supports session artifacts with `StoragePath` and `StorageType` via `ReadSessionArtifact`.
- `api/workspace/workspace_save_test.go` -- Unit tests for session artifact content retrieval handler.

## Tasks & Acceptance

**Execution:**
- `ui/src/lib/utils.ts` -- Export `formatFileSize` utility function for consistent size formatting.
- `ui/src/components/workspace/ChatPane.tsx` -- Import `formatFileSize` from `@/lib/utils`.
- `ui/src/components/workspace/CanvasPane.tsx` -- Update `targetArtifact` matcher to match on `a.name` as well.
- `api/workspace/workspace.go` -- Support `StoragePath` and `ReadSessionArtifact` in `GetArtifactContentHandler` for session artifact content retrieval.
- `ui/src/app/(dashboard)/projects/[id]/workspace/_components/ExplorerPanel.tsx` -- Implement session artifact matching, expandable chevron icon with rotation animation, smooth dropdown sub-list rendering, artifact type icons/thumbnails, formatted sizes, and preview trigger handlers.
- `ui/src/app/(dashboard)/projects/[id]/workspace/_components/ExplorerPanel.test.tsx` -- Add unit tests covering all acceptance criteria and edge cases.
- `ui/src/components/workspace/CanvasPane.test.tsx` -- Add test covering session artifact preview resolution.

**Acceptance Criteria:**
- Given the Chat history / Session list sidebar, when a chat session has one or more associated artifacts registered in Firestore (matched via `sessionId`), then the session item MUST render an expandable chevron indicator on the right, and sessions without any artifacts MUST NOT render an expandable chevron.
- Given a session with artifacts, when the user clicks the session or clicks the chevron indicator, then the session item MUST toggle expand/collapse state with a smooth CSS transition and render a dropdown / nested sub-list of all artifacts associated with that session.
- Given the expanded session artifact dropdown, when rendered, then each artifact item MUST display:
  - File type icon (or image thumbnail for image files)
  - File name
  - File size (formatted)
  - An action trigger to open/preview the artifact in the workspace preview pane.

## Spec Change Log

_Empty._

## Review Triage Log

### 2026-08-27 — Iteration 1

**Reviewers:** Blind Reviewer, Edge Case Hunter, Verification Gap Reviewer, Intent Alignment Auditor

**Findings & Triage:**
- `patch`: In `api/workspace/workspace.go:552`, ensured `targetArt.StorageType == "gcs"` and `targetArt.StorageType == "local_file"` correctly route storage backend regardless of environment defaults. (Applied and verified)
- `patch`: In `ui/src/app/(dashboard)/projects/[id]/workspace/_components/ExplorerPanel.tsx:305`, updated `art.size` condition to `art.size !== undefined && art.size !== null` to properly render size badge for 0-byte files. (Applied and verified)
- `patch`: In `ui/src/app/(dashboard)/projects/[id]/workspace/_components/ExplorerPanel.tsx:236`, added `aria-expanded={isExpanded}` attribute on collapsible chevron button for accessibility. (Applied and verified)
- `patch`: In `ui/src/components/workspace/CanvasPane.test.tsx`, added unit test asserting `CanvasPane` properly resolves and renders session artifacts by `name` and `storagePath`. (Applied and verified)
- `reject`: Artifact name collision across sessions in `GetArtifactContentHandler` — handler queries by artifact ID primarily; resolution order is consistent.
- `reject`: Out of scope / minor stylistic suggestions (e.g. gigabyte scale in `formatFileSize`, server internal path label display).

**Triage Summary:**
- intent_gap: 0
- bad_spec: 0
- patch: 4 (high: 0, medium: 0, low: 4)
- defer: 0
- reject: 10

## Verification

**Commands:**
- `npm --prefix ui test -- --run src/app/\(dashboard\)/projects/\[id\]/workspace/_components/ExplorerPanel.test.tsx` -- expected: PASS all tests
- `npm --prefix ui test -- --run src/components/workspace/CanvasPane.test.tsx` -- expected: PASS all tests
- `npm --prefix ui test -- --run src/components/workspace/ChatPane.test.tsx` -- expected: PASS all tests
- `go test ./api/workspace/...` -- expected: PASS all backend tests
- `npm --prefix ui run build` -- expected: UI build succeeds with zero type or bundling errors

## Auto Run Result

- **Story**: `18-5-chat-session-collapsible-chevron-artifacts-dropdown-frontend`
- **Status**: `done`
- **Followup Review Recommended**: `false`
- **Changes Summary**:
  - Implemented collapsible chevron toggle on session rows with artifacts in `ExplorerPanel.tsx`.
  - Added smooth animated expandable dropdown sub-list displaying file type icons/thumbnails, filenames, formatted file sizes (`formatFileSize`), and preview trigger actions (`setActiveArtifactTitle`).
  - Added unit test suite in `ExplorerPanel.test.tsx` (17/17 tests passing).
  - Enhanced `CanvasPane.tsx` artifact matcher to match session artifacts by `name` and `storagePath`, with unit tests in `CanvasPane.test.tsx` (9/9 tests passing).
  - Enhanced backend `GetArtifactContentHandler` in `api/workspace/workspace.go` to support session artifacts with `StoragePath` via `ReadSessionArtifact`.
  - Reused `formatFileSize` from `ui/src/lib/utils.ts` in `ChatPane.tsx` (49/49 tests passing).
- **Verification Summary**:
  - `npm --prefix ui test -- --run src/app/\(dashboard\)/projects/\[id\]/workspace/_components/ExplorerPanel.test.tsx`: PASS (17 tests)
  - `npm --prefix ui test -- --run src/components/workspace/CanvasPane.test.tsx`: PASS (9 tests)
  - `npm --prefix ui test -- --run src/components/workspace/ChatPane.test.tsx`: PASS (49 tests)
  - `go test ./api/workspace/...`: PASS
  - `npm --prefix ui run build`: PASS (0 errors)
