---
title: 'Story 18.6: Image Artifact Preview & Media Viewer (Frontend)'
type: 'feature'
created: '2026-08-27'
status: 'done'
baseline_revision: 'f41a1875cfbaf491e488e968aea1ab83b04f3afe'
review_loop_iteration: 1
followup_review_recommended: false
context:
  - docs/spec/implementation-artifacts/epic-18-context.md
warnings: []
deferred: []
---

<intent-contract>

## Intent

**Problem:** When users click on image artifacts (`.png`, `.jpg`, `.jpeg`, `.gif`, `.webp`, `.svg`) in the Session Artifacts dropdown or Artifact Sidebar, the preview pane attempts to load them in the Monaco code editor or text viewer, displaying garbled binary text, unnecessary line numbers, and inappropriate specification "Approve"/"Edit" buttons instead of a dedicated image viewer.

**Approach:** Build a dedicated `ImageViewer` component with zoom controls, pan support, responsive aspect ratio fitting, and transparency checkerboard background; integrate it into `CanvasPane` to render image artifacts with an image header showing filename, dimensions, formatted file size, and a download action button, while hiding Monaco editor toolbars, line numbers, syntax tools, and specification approval buttons.

## Boundaries & Constraints

**Always:**
- Detect image artifacts by checking file extension (`.png`, `.jpg`, `.jpeg`, `.gif`, `.webp`, `.svg`, `.bmp`, `.ico`, `.avif`), MIME type (`image/*`), or `artifactType` (`image`).
- Mount the `ImageViewer` component in the preview pane whenever an image artifact is active.
- Display zoom controls in `ImageViewer`: Zoom In (`+`), Zoom Out (`-`), Reset Zoom / Fit to Screen, and zoom percentage indicator.
- Support mouse/pointer drag panning when zoomed in (or when dragging on canvas).
- Support responsive aspect ratio fitting so images scale smoothly within the container viewport without distortion.
- In the preview pane header for image artifacts, display:
  1. Image filename (`displayTitle`).
  2. Image dimensions in pixels (e.g. `1920 × 1080 px`), dynamically measured upon image load or from metadata.
  3. Formatted file size (e.g. `2.4 MB`, `15.0 KB`) using `formatFileSize`.
  4. A "Download" action button that triggers downloading the raw image file with its original filename.
- When previewing an image artifact, HIDE the Monaco editor, text line numbers, syntax highlighting toolbar, Copy code button, and specification "Approve" / "Push" buttons.
- For non-image artifacts (e.g. markdown specifications or code files), ensure standard Monaco/Markdown rendering with specification review and approval controls continues to function normally.
- In backend `GetArtifactContentHandler`, ensure Content-Type detection fallback uses `mime.TypeByExtension` for unindexed workspace files.

**Block If:**
- External third-party credentials or human-only administrative actions outside the repository are required.

**Never:**
- Break Monaco code editor or Markdown preview capabilities for non-image files and specs.
- Mutate `sprint-status.yaml` or any orchestrator-owned state.
- Render binary/garbage text in Monaco when an image artifact is selected.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| Preview PNG/JPG/WebP/GIF/SVG artifact | User clicks image artifact from Session Artifacts dropdown or Artifact Sidebar | `CanvasPane` mounts `ImageViewer`; Monaco editor & Approve buttons are hidden; header shows filename, dimensions, size, and Download button | Show loading spinner during fetch, error state on failure |
| Preview non-image file (.md, .ts, .go) | User clicks markdown spec or code file | `CanvasPane` mounts `VSCodeArtifactViewer` with Monaco editor / Markdown preview and Approve dropdown | Standard error handling |
| Image Zoom In / Zoom Out / Reset | User clicks `+` / `-` / Reset buttons in `ImageViewer` | Scale updates by ±25% (range 25% - 500%), percentage displays accurately, Reset returns to 100% and centered (0,0) pan | Bounded scale limits |
| Image Pan on Drag | User drags mouse while zoomed in | Image translates smoothly following cursor displacement with grab/grabbing cursor | Clamped / smooth coordinate updates |
| Image Dimension Measurement | Image finishes loading in DOM (`onLoad`) | Measures `naturalWidth` and `naturalHeight`, displays `W × H px` badge in header | Gracefully omit badge if dimensions are unavailable or load fails |
| Download Image | User clicks "Download" button in header | Triggers browser download of the image blob / file with original filename | Toast error if download creation fails |
| Broken / 404 Image URL | Server returns 404 or corrupted image stream | `ImageViewer` renders error state with retry button | Does not crash application |

</intent-contract>

## Code Map

- `ui/src/lib/utils.ts` -- Add `isImageFile(filenameOrPath?: string | null, mimeType?: string | null, artifactType?: string | null): boolean` helper and export it.
- `ui/src/components/workspace/ImageViewer.tsx` -- New component: dedicated image viewer with zoom controls (zoom in, zoom out, reset, zoom level badge), pan support (drag to pan), responsive aspect ratio fitting, transparent checkered background, dimension detection, loading spinner, and error state.
- `ui/src/components/workspace/ImageViewer.test.tsx` -- Unit tests for `ImageViewer`: rendering, zoom in/out/reset interactions, panning, dimension reporting, and error handling.
- `ui/src/components/workspace/CanvasPane.tsx` -- Integrate `ImageViewer` when `isImage` is true; render image header with filename, dimensions, file size, and Download button; hide Monaco editor, Copy button, and Approve dropdown; maintain full Monaco/Markdown behavior for non-image artifacts.
- `ui/src/components/workspace/CanvasPane.test.tsx` -- Unit tests verifying image artifact mounting, image header metadata display, Monaco/Approve button suppression, Download button action, and non-image fallback preservation.
- `api/workspace/workspace.go` -- Enhance `GetArtifactContentHandler` with `mime.TypeByExtension` fallback for direct workspace files.

## Tasks & Acceptance

**Execution:**
- `ui/src/lib/utils.ts` -- Add and export `isImageFile` helper function.
- `ui/src/components/workspace/ImageViewer.tsx` -- Create `ImageViewer` component with zoom controls, pan interactions, aspect ratio fitting, dimension callback, and checkered background.
- `ui/src/components/workspace/ImageViewer.test.tsx` -- Write unit tests for `ImageViewer` component features and edge cases.
- `ui/src/components/workspace/CanvasPane.tsx` -- Integrate `ImageViewer`, dynamic dimension detection, image header tray with dimensions, file size, Download button, and conditional suppression of Monaco editor and Approve buttons.
- `ui/src/components/workspace/CanvasPane.test.tsx` -- Add unit tests verifying `ImageViewer` integration, image header metadata, Monaco suppression, download handler, and non-image rendering.
- `api/workspace/workspace.go` -- Ensure `GetArtifactContentHandler` provides accurate `Content-Type` headers for image files.

**Acceptance Criteria:**
- Given an image artifact (`.png`, `.jpg`, `.jpeg`, `.gif`, `.webp`, `.svg`) is clicked from the Session Artifacts dropdown or Artifact Sidebar, when the preview pane opens, then the system MUST mount an `ImageViewer` component displaying the image with zoom controls, pan support, and responsive aspect ratio fitting.
- Given the preview pane is rendering an image artifact, when displayed to the user, then the Monaco code editor, text line numbers, syntax highlighting toolbar, and specification "Approve" / "Edit" buttons MUST BE HIDDEN, and the header MUST display the image filename, dimensions (if available), file size, and a "Download" action button.
- Given non-image artifacts (e.g., markdown specifications or code files), when clicked, then standard Monaco/Markdown rendering with specification review and approval controls MUST continue to function as normal.

## Spec Change Log

_Empty._

## Review Triage Log

### 2026-08-27 — Review pass
- intent_gap: 0
- bad_spec: 0
- patch: 2: (high 0, medium 1, low 1)
- defer: 0
- reject: 12
- addressed_findings:
  - `[medium]` `[patch]` Added global mouse movement and mouse up listeners during drag panning and mouse wheel zoom support in ImageViewer.
  - `[low]` `[patch]` Added download error handling assertion test in CanvasPane.test.tsx.

## Verification

**Commands:**
- `pnpm vitest run src/components/workspace/ImageViewer.test.tsx src/components/workspace/CanvasPane.test.tsx` -- expected: All unit tests pass
- `pnpm vitest run` -- expected: Full test suite passes
- `go test -v ./workspace -run TestGetArtifactContentHandler` -- expected: Backend handler tests pass

## Auto Run Result

### Summary of implemented change
Implemented a dedicated `ImageViewer` component and integrated it into the workspace preview pane (`CanvasPane`). When users click image-type artifacts (`.png`, `.jpg`, `.jpeg`, `.gif`, `.webp`, `.svg`, `.bmp`, `.ico`, `.avif`), the preview pane mounts `ImageViewer` with zoom controls (25%-500%), pan drag interactions, mouse wheel zoom, responsive aspect ratio fitting, transparent checkered background, dynamic pixel dimension detection, loading spinner, and error/retry state. The header displays the image filename, dimensions badge, formatted file size, and a download action button, while Monaco editor, line numbers, code toolbar, and specification Approve/Push controls are cleanly suppressed. For non-image files and specifications, standard Monaco and Markdown previewing with approval workflows continue to function as normal.

### Files changed
- `ui/src/lib/utils.ts` -- Added and exported `isImageFile` helper detecting image extensions, MIME types, and image artifact types.
- `ui/src/components/workspace/ImageViewer.tsx` -- Created dedicated image viewer component with zoom controls, pan dragging, wheel zoom, aspect ratio fitting, checkered alpha background, and dynamic dimension measurement.
- `ui/src/components/workspace/ImageViewer.test.tsx` -- Unit tests verifying image mounting, dimension detection, zoom in/out/reset/wheel interactions, pan dragging, loading state, and error recovery.
- `ui/src/components/workspace/CanvasPane.tsx` -- Integrated `ImageViewer` for image artifacts, built image header with dimensions badge, file size, and Download action, and suppressed Monaco editor and Approve controls.
- `ui/src/components/workspace/CanvasPane.test.tsx` -- Unit tests verifying image viewer rendering, header metadata display, Monaco and Approve button suppression on image artifacts, Download button action, and non-image specification rendering.
- `api/workspace/workspace.go` -- Enhanced `GetArtifactContentHandler` with `mime.TypeByExtension` fallback for direct workspace files.
- `api/workspace/workspace_save_test.go` -- Unit test verifying MIME type fallback in `GetArtifactContentHandler`.
- `docs/spec/implementation-artifacts/18-6-image-artifact-preview-media-viewer-frontend.md` -- Specification, test audit, review triage, and verification results.

### Review findings breakdown
- Patches applied: 2 (global drag listener + wheel zoom enhancement in ImageViewer, download error handling test in CanvasPane.test.tsx)
- Items deferred: 0
- Items rejected: 12

### Follow-up review recommendation
- `false` (Score: 4 = 3 × 1 medium + 1 × 1 low; threshold >= 5)

### Verification performed
- `pnpm vitest run src/components/workspace/ImageViewer.test.tsx src/components/workspace/CanvasPane.test.tsx`: PASS (19/19 tests)
- `pnpm vitest run`: PASS (355/355 tests across 41 test files)
- `go test -v ./workspace -run TestGetArtifactContentHandler`: PASS (4/4 tests)
- `go test -v ./workspace/...`: PASS (All workspace backend tests)

### Residual risks
- None identified.

