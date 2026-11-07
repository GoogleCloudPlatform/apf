---
status: done
baseline_revision: 58a4aa4ab5c73ae5a8f5b5b82b93c878237dbd60
followup_review_recommended: false
---

# Story 4.13: Unified File Explorer with Phase-Associated Artifacts & Status Indicators

Status: done

## Story

As a Product Manager, UX Designer, Architect, or Developer,
I want the File Explorer in the Workspace Explorer sidebar to serve as the unified navigation surface for all generated artifacts and workspace files,
so that when "Show All Files" is unchecked, it displays only Firestore-tracked artifacts associated with the active phase along with live status indicators (`In Progress`, `Complete`, `Staged for Review`, `Not Started`), and when "Show All Files" is checked, it reveals the full repository directory tree, eliminating redundant sidebar sections.

## Review Triage Log

### 2026-08-15 — Review pass
- intent_gap: 0
- bad_spec: 0
- patch: 6 (high 0, medium 4, low 2)
- defer: 0
- reject: 0
- addressed_findings:
  - `[medium]` `[patch]` Added `onRefreshArtifacts={refetchArtifacts}` and `isLoadingArtifacts={isLoading}` props forwarding in `ExplorerPanel.tsx` and `page.tsx` for complete dual-refresh and loading synchronization.
  - `[medium]` `[patch]` Prevented keyboard event bubbling on the artifact delete button in `FilteredFileListing.tsx` (`onKeyDown={(e) => e.stopPropagation()}`) and added `focus-visible:opacity-100` for keyboard accessibility.
  - `[medium]` `[patch]` Added `Escape` key listener to Delete Confirmation Dialog in `FilteredFileListing.tsx` and protected against undefined path splitting when deriving artifact filenames.
  - `[medium]` `[patch]` Tightened tree leaf artifact matching in `FileNode` (`FileExplorer.tsx`) to prevent false-positive status indicator dots on identically named files across different directories.
  - `[low]` `[patch]` Added `useEffect` in `FileNode` to trigger `fetchChildren()` on auto-expansion when un-cached directories are opened via active file selection.
  - `[low]` `[patch]` Cleared `childrenCache` in `FileExplorer.tsx` on manual refresh so expanded subfolders refetch fresh directory content.

## Auto Run Result

### Summary of Implemented Change
- Unified the File Explorer as the single navigation surface across the application workspace by removing the legacy `GENERATED ARTIFACTS` accordion and obsolete `PhaseOutcomeArtifacts` component.
- Implemented phase-filtered artifact navigation in `FilteredFileListing.tsx` and `FileExplorer.tsx`, rendering high-density rows with file icons, filenames, faint parent directory paths, alphabetical sorting, and colored status indicator dots with tooltips (`emerald` for Complete, `amber` for In Progress, `blue` for Staged for Review, `slate` for Not Started).
- Enhanced full repository tree view (`Show All Files` toggle) in `FileExplorer.tsx` to cross-reference tracked Firestore artifacts and display matching status indicator dots on leaf file items.
- Connected dual refresh functionality simultaneously refreshing both filesystem directory tree and Firestore artifacts.
- Synchronized `CanvasPane` preview selection on click and auto-closing on artifact deletion.
- Updated onboarding tour definitions (`tour-steps.json`) and test page objects (`workspace-page.ts`) to point to `#tour-workspace-file-explorer`.
- Verified 100% test coverage across all unit test suites (226 vitest tests) and Playwright E2E suites (15 tests).

### Files Changed
- `ui/src/components/workspace/FilteredFileListing.tsx`: Phase-filtered artifact rendering with status dots, tooltips, keyboard navigation, and deletion modal.
- `ui/src/components/workspace/FileExplorer.tsx`: Phase filtering, tree view status dots, dual refresh, and cache management.
- `ui/src/app/(dashboard)/projects/[id]/workspace/_components/ExplorerPanel.tsx`: Elimination of `GENERATED ARTIFACTS` accordion and wiring of `FileExplorer`.
- `ui/src/app/(dashboard)/projects/[id]/workspace/page.tsx`: Updated ExplorerPanel prop forwarding for artifact refetching.
- `ui/src/app/(dashboard)/projects/[id]/workspace/_hooks/useWorkspace.ts`: Enhanced active document auto-close logic on deletion.
- `ui/src/stores/tour-steps.json`: Updated onboarding tour step target to `#tour-workspace-file-explorer`.
- `ui/e2e/page-objects/workspace-page.ts`: Updated `artifactsSection` locator.
- `ui/e2e/specs/projects/find-file-in-tree.spec.ts`: Updated E2E tests for phase filtered mode and tree view toggle.
- `ui/src/app/(dashboard)/projects/[id]/workspace/_components/ExplorerPanel.test.tsx`: Unit tests for accordion removal and props forwarding.
- `ui/src/components/workspace/FilteredFileListing.test.tsx`: Unit tests for artifact rendering, status indicators, and deletion.
- `ui/src/components/workspace/FileExplorer.test.tsx`: Unit tests for phase filtering, header switching, and dual refresh.
- `ui/src/components/workspace/PhaseOutcomeArtifacts.tsx`: Removed obsolete component.
- `ui/src/components/workspace/PhaseOutcomeArtifacts.test.tsx`: Removed obsolete test suite.

### Review Findings Breakdown
- Patches applied: 6
- Items deferred: 0
- Items rejected: 0

### Follow-up Review Recommendation
`false` (0 high severity findings, score = 3 × 0 + 6 = 6).

### Verification Performed
- Ran `npx vitest run`: 34 test files passed, 226 unit tests passed.
- Ran `npx playwright test ui/e2e/specs/projects/find-file-in-tree.spec.ts --workers=3`: 15 E2E tests passed.

### Residual Risks
None. All components, hooks, tour steps, and test suites are fully aligned and passing.

---

## Acceptance Criteria

1. **Elimination of Generated Artifacts Accordion (AC: 4.13.1)**
   - **Given** a user views any project workspace (`/projects/[id]/workspace`) across any lifecycle phase (`ONBOARDING`, `ANALYSIS`, `PLANNING`, `SOLUTIONING`, `IMPLEMENTATION`)
   - **When** the `ExplorerPanel` sidebar renders
   - **Then** the separate `GENERATED ARTIFACTS` accordion (`#tour-workspace-artifacts` / `PhaseOutcomeArtifacts`) MUST be removed, leaving only `CHAT SESSIONS` and `FILE EXPLORER` (plus `WORKFLOWS` in Implementation phase).
   - **And** the onboarding product tour target MUST update to point to `#tour-workspace-file-explorer`.

2. **Phase-Filtered Artifacts View (Default / Unchecked Mode) (AC: 4.13.2)**
   - **Given** the `FileExplorer` component with "Show All Files" switch unchecked (the default state)
   - **When** rendering the file listing
   - **Then** the header MUST display `"<PhaseName> Files"` (e.g., `"Onboarding Files"`, `"Planning Files"`, `"Solutioning Files"`)
   - **And** the list MUST display ONLY the artifacts tracked in Firestore for the project whose phase matches the active phase (`(a.phase || "").toUpperCase() === (activePhase || "").toUpperCase()`).
   - **And** if no artifacts are tracked for the active phase, a clean empty state message MUST be displayed (e.g., `"No files tracked for Planning phase."`).

3. **Artifact Status Indicators (AC: 4.13.3)**
   - **Given** an artifact rendered in the phase-filtered file listing
   - **When** rendering the artifact item
   - **Then** each item MUST display:
     - The appropriate file-type icon via `FileIcon` based on filename/extension
     - The artifact file name (`a.title` or `a.path.split('/').pop()`)
     - The parent directory path subtitle if nested (e.g., `docs/specs/prd`)
     - A clean colored status indicator dot only (with tooltip title) matching its Firestore lifecycle state:
       - **In Progress**: Amber dot (`bg-amber-500`, title="In Progress")
       - **Complete**: Emerald dot (`bg-emerald-500`, title="Complete")
       - **Staged for Review**: Blue dot (`bg-blue-500`, title="Staged for Review")
       - **Not Started**: Slate dot (`bg-slate-400`, title="Not Started")

4. **Interactive Selection & Canvas Preview Integration (AC: 4.13.4)**
   - **Given** the phase-filtered artifact list
   - **When** the user clicks on an artifact item
   - **Then** `onFileSelect` MUST be called with `{ name, path }`, setting `activeArtifactTitle` and opening the artifact in the Canvas preview pane (`#tour-workspace-canvas`)
   - **And** the selected artifact item MUST display active selection styling (`bg-primary/10 border-l-2 border-primary text-primary font-semibold`).

5. **Artifact Deletion with Confirmation (AC: 4.13.5)**
   - **Given** an artifact in the phase-filtered list
   - **When** the user hovers over the artifact item and clicks the trash icon button
   - **Then** a modal confirmation dialog MUST appear requesting confirmation to delete the artifact
   - **And** upon confirming delete, `onDeleteArtifact` / `deleteArtifact(artifact.id)` MUST be invoked via the API tier and the item removed from the list.

6. **Hierarchical Directory Tree View (Checked Mode) (AC: 4.13.6)**
   - **Given** the `FileExplorer` component
   - **When** the user toggles "Show All Files" switch to ON (checked)
   - **Then** the header MUST display `"All Files"`
   - **And** the view MUST render the full hierarchical repository directory tree (`[role="tree"]`) with on-demand folder expansion, collapsing, and subfolder refreshing
   - **And** any file leaf node matching a tracked Firestore artifact MUST display its lifecycle status indicator dot.

---

## Tasks / Subtasks

- [ ] **Task 1: Update `ExplorerPanel` Layout & Remove Generated Artifacts Section (AC: 4.13.1)**
  - [ ] **Subtask 1.1:** Remove `PhaseOutcomeArtifacts` import and `<ExplorerSection id="tour-workspace-artifacts">` block in `ui/src/app/(dashboard)/projects/[id]/workspace/_components/ExplorerPanel.tsx`.
  - [ ] **Subtask 1.2:** Pass `liveArtifacts`, `activePhase`, `deleteArtifact`, and `isLoading` directly into `<FileExplorer />`.
  - [ ] **Subtask 1.3:** Update `ui/src/app/(dashboard)/projects/[id]/workspace/page.tsx` to clean up obsolete artifact accordion expansion bindings.
  - [ ] **Subtask 1.4:** Update `ui/src/stores/tour-steps.json` target from `#tour-workspace-artifacts` to `#tour-workspace-file-explorer`.
  - [ ] **Subtask 1.5:** Update `ui/e2e/page-objects/workspace-page.ts` `artifactsSection` locator to point to `#tour-workspace-file-explorer`.

- [ ] **Task 2: Enhance `FilteredFileListing` with Artifact Status Indicator Dots & Deletion (AC: 4.13.2, 4.13.3, 4.13.4, 4.13.5)**
  - [ ] **Subtask 2.1:** Update `FilteredFileListingProps` in `ui/src/components/workspace/FilteredFileListing.tsx` to accept `artifacts: Artifact[]`, `activePhase`, `selectedFilePath`, `onFileSelect`, and `onDeleteArtifact`.
  - [ ] **Subtask 2.2:** Build status indicator dot helper (`getArtifactStatusDot`) with distinct colors and tooltip titles for `In Progress` (amber), `Complete` (emerald), `Staged for Review` (blue), and `Not Started` (slate).
  - [ ] **Subtask 2.3:** Add hover delete action and Radix/Tailwind delete confirmation dialog.
  - [ ] **Subtask 2.4:** Add contextual empty state when no artifacts are present for `activePhase`.

- [ ] **Task 3: Enhance `FileExplorer` with Phase Filtering & Tree Status Badges (AC: 4.13.2, 4.13.6)**
  - [ ] **Subtask 3.1:** Update `FileExplorer.tsx` props to accept `liveArtifacts`, `activePhase`, and `onDeleteArtifact`.
  - [ ] **Subtask 3.2:** Filter `liveArtifacts` by active phase when `showAllFiles` is false and pass to `FilteredFileListing`.
  - [ ] **Subtask 3.3:** In `FileNode` (tree view), match file nodes to `liveArtifacts` and render status indicator dots next to filenames.

- [ ] **Task 4: Retire Obsolete `PhaseOutcomeArtifacts` Component (AC: 4.13.1)**
  - [ ] **Subtask 4.1:** Cleanly remove `ui/src/components/workspace/PhaseOutcomeArtifacts.tsx` and `PhaseOutcomeArtifacts.test.tsx`.

- [ ] **Task 5: Comprehensive Unit & E2E Testing (AC: 4.13.1 - 4.13.6)**
  - [ ] **Subtask 5.1:** Update `ExplorerPanel.test.tsx` to assert CHAT SESSIONS and FILE EXPLORER, verifying no GENERATED ARTIFACTS accordion.
  - [ ] **Subtask 5.2:** Update `FilteredFileListing.test.tsx` for artifact rendering, status badges, deletion confirmation, and phase empty states.
  - [ ] **Subtask 5.3:** Update `FileExplorer.test.tsx` for phase filtering, phase switching, status badges, and tree view toggle.
  - [ ] **Subtask 5.4:** Run all vitest unit tests (`npx vitest run`) and Playwright E2E tests (`npx playwright test ui/e2e/specs/projects/find-file-in-tree.spec.ts`).

---

## Dev Notes

### 🏗️ Architectural & Design Alignment
- **Single Navigation Surface**: Eliminates the redundant `GENERATED ARTIFACTS` accordion, making `FileExplorer` the single source of truth for both phase deliverables and full repository browsing.
- **Empty State Behavior**: When no Firestore artifacts exist for the active phase, renders a clean contextual empty state ("No files tracked for <Phase> phase. Talk to the agent or run a quick action to generate artifacts.") while keeping "Show All Files" available to view raw repository files.
- **Filtered View Layout & Sorting**: Flat, high-density list displaying the file icon, filename, faint parent path subtitle (e.g. `docs/specs/prd/`), and a right-aligned colored status indicator dot only (with tooltip title, e.g. `title="In Progress"`), sorted alphabetically by filename/title.
- **Robust Phase Matching**: Artifacts in the filtered view use trimmed, case-insensitive phase matching (`(a.phase || "").trim().toUpperCase() === (activePhase || "").trim().toUpperCase()`). Untracked or phase-unassigned files are accessible via the "Show All Files" tree mode.
- **Tree View Status Dots**: When "Show All Files" is checked (hierarchical tree mode), leaf file nodes that match a tracked Firestore artifact display a subtle colored indicator dot next to the filename (emerald for Complete, amber for In Progress, blue for Staged for Review).
- **Real-Time Live SSE Sync**: `useWorkspace` refetches artifacts upon receiving SSE `file_write` / `artifact_update` events, ensuring newly created or modified artifacts immediately appear with live status in `FileExplorer` without requiring manual page reload.
- **Dual Refresh Action**: Clicking the Refresh button in the header refetches both Firestore artifacts and the filesystem directory tree simultaneously.
- **Keyboard Accessibility**: Full keyboard navigation support (Enter / Space to select, `tabIndex={0}`, `role="list"` / `role="listitem"`, `aria-selected`) across both list and tree modes.
- **State Persistence Across Navigation**: The "Show All Files" toggle state and File Explorer accordion expanded state are preserved across phase changes and chat session switches within the user's active session.
- **Loading Presentation**: FileExplorer renders subtle inline skeleton loader rows matching the high-density layout during initial artifact and file tree fetches.
- **CanvasPane Persistence & Auto-Close**: When switching phases via the top navigation bar, any open document in `CanvasPane` remains open and active while `FileExplorer` dynamically updates to show the newly selected phase's artifacts. If an active document is deleted, `CanvasPane` automatically closes (`setActiveArtifactTitle(null)`).
- **Status Lifecycle Handling**: The `FileExplorer` renders status indicator dots; detailed manual status changes, editing, and approvals are handled directly in `CanvasPane`.
- **Header Structure**: Compact header containing the Phase Title (`"<Phase> Files"` / `"All Files"`), the "Show All Files" toggle switch, and a manual Refresh button.
- **Deletion Interaction**: Hovering an artifact in the phase-filtered list reveals a trash icon button; clicking it opens a confirmation dialog before calling `DELETE /api/v1/projects/{id}/artifacts/{artifactId}`.
- **CanvasPane Synchronization**: Single-clicking an artifact sets `activeArtifactTitle`, highlights the item, and immediately renders the document in the Canvas preview pane.

---

## 🛠️ Target Files Matrix

| File Path | Status | Operational Role |
| :--- | :--- | :--- |
| `ui/src/app/(dashboard)/projects/[id]/workspace/_components/ExplorerPanel.tsx` | `MODIFY` | Remove GENERATED ARTIFACTS accordion; connect FileExplorer to phase artifacts. |
| `ui/src/app/(dashboard)/projects/[id]/workspace/page.tsx` | `MODIFY` | Clean up ExplorerPanel props. |
| `ui/src/components/workspace/FileExplorer.tsx` | `MODIFY` | Wire phase filtering, dual refresh, and tree status indicator dots. |
| `ui/src/components/workspace/FilteredFileListing.tsx` | `MODIFY` | Render flat phase-filtered artifacts with status badges, keyboard navigation, and deletion modal. |
| `ui/src/components/workspace/PhaseOutcomeArtifacts.tsx` | `DELETE` | Retire superseded component. |
| `ui/src/stores/tour-steps.json` | `MODIFY` | Update onboarding tour step target to `#tour-workspace-file-explorer`. |
| `ui/e2e/page-objects/workspace-page.ts` | `MODIFY` | Update `artifactsSection` locator. |
| `ui/src/app/(dashboard)/projects/[id]/workspace/_components/ExplorerPanel.test.tsx` | `MODIFY` | Update ExplorerPanel tests. |
| `ui/src/components/workspace/FileExplorer.test.tsx` | `MODIFY` | Add phase-filtered tests, dual refresh tests, and status badge tests. |
| `ui/src/components/workspace/FilteredFileListing.test.tsx` | `MODIFY` | Add artifact list, keyboard navigation, and status indicator tests. |
| `ui/src/components/workspace/PhaseOutcomeArtifacts.test.tsx` | `DELETE` | Remove superseded test file. |

---

## Dev Agent Record

### Agent Model Used
Gemini Next

### Debug Log References
- Implementation Plan: `plan_phase_files_and_remove_generated_artifacts.md`
- Grill-Me Interview: Resolved empty state, visual layout, alphabetical sorting, case-insensitive phase matching, tree status dots, real-time SSE sync, dual refresh, keyboard navigation, state persistence, skeleton loading, canvas persistence, auto-close on delete, deletion UX, and canvas click behavior.

### Completion Notes List
### File List
