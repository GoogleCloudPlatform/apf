# Story 4.2: Tabbed Workspace Explorer: Artifacts and File Explorer Views

Status: done

## Story

As a Developer/User,
I want a tabbed navigation in the workspace explorer section with "Artifacts View" and "File Explorer View" tabs,
so that I can switch between viewing high-level generated specs/artifacts and the underlying project directory tree.

## Acceptance Criteria

1. **Given** the user is in the project workspace details view (e.g., `/projects/[id]/workspace`)
   **When** the workspace Explorer section (left sidebar) renders
   **Then** it displays two distinct tabs: "Artifacts" and "File Explorer".
2. **And** the "Artifacts" tab MUST be selected by default, displaying the registered artifacts list/rail (which includes Chat Sessions, Quick Actions, and Generated Artifacts/Workflows as currently rendered in `ExplorerPanel`).
3. **When** the user clicks on the "File Explorer" tab
   **Then** the UI switches the explorer view to render the `FileExplorer` component showing the project filesystem.
4. **And** the tab switching MUST be instantaneous (latency < 100ms) without triggering a full-page or full-workspace reload.
5. **And** both tabs MUST maintain their scroll position and expansion state (e.g., which sections are expanded in "Artifacts", and which folders are expanded in "File Explorer") when switching back and forth during the active session.

## Tasks / Subtasks

- [x] **Task 1: Implement Tabbed Layout in `ExplorerPanel.tsx` (AC: 1, 2, 3, 4)**
  - [x] Add state for `activeTab` (`"artifacts" | "file-explorer"`, defaulting to `"artifacts"`).
  - [x] Render a sleek, minimalist tab header (tabs: "Artifacts", "File Explorer") below the "WORKSPACE EXPLORER" title section, matching the Material 3/Tailwind styling.
  - [x] Wrap the existing explorer sections (Chat Sessions, Quick Actions, Generated Artifacts, Workflows) in an "Artifacts" container.
  - [x] Render both the "Artifacts" container and the new `FileExplorer` component side-by-side (hidden/shown conditionally).
  - [x] **CRITICAL**: Use CSS-based visibility toggling (e.g., Tailwind's `hidden` class: `className={cn(activeTab !== 'artifacts' && 'hidden')}`) instead of conditional rendering (`{activeTab === 'artifacts' && ...}`) to ensure that both views remain mounted in the DOM. This is required to preserve their scroll positions and internal states (like expanded folders/sections) when switching tabs.
- [x] **Task 2: Create Placeholder `FileExplorer` Component (AC: 3, 5)**
  - [x] Create a new component `FileExplorer.tsx` in `ui/src/components/workspace/`.
  - [x] Implement it as a static/mock placeholder that shows a mock folder tree structure (e.g., `api/`, `ai/`, `ui/`, `docs/`) with folder/file icons to verify rendering, tab switching, and state preservation.
  - [x] Include a clear developer note in the code indicating that the real recursive, on-demand filesystem fetching and API integration will be implemented in Story 4.8.
- [x] **Task 3: Verify State Preservation and Performance (AC: 4, 5)**
  - [x] Verify that expanding a section in "Artifacts" (e.g., "CHAT SESSIONS"), scrolling down, switching to "File Explorer", and switching back preserves the expanded state and scroll position.
  - [x] Verify that tab switching does not trigger any network requests or parent re-renders that would reset state.

### Review Findings

- [x] [Review][Patch] Sidebar Collapse Causes Component Unmounting and State Loss [ui/src/app/(dashboard)/projects/[id]/workspace/_components/ExplorerPanel.tsx:96]
- [x] [Review][Patch] Double Scrollbars in File Explorer [ui/src/components/workspace/FileExplorer.tsx:118]
- [x] [Review][Patch] Hardcoded Magic Height in File Explorer [ui/src/components/workspace/FileExplorer.tsx:118]
- [x] [Review][Patch] Use of @ts-nocheck in Tests [ui/src/app/(dashboard)/projects/[id]/workspace/_components/ExplorerPanel.test.tsx:1]
- [x] [Review][Patch] Brittle CSS Class Testing in Unit Tests [ui/src/app/(dashboard)/projects/[id]/workspace/_components/ExplorerPanel.test.tsx]
- [x] [Review][Patch] Use of Array Index as React Keys [ui/src/components/workspace/FileExplorer.tsx:801]
- [x] [Review][Patch] Complete Lack of Keyboard Accessibility [ui/src/components/workspace/FileExplorer.tsx]
- [x] [Review][Patch] Missing ARIA Semantics for Tabs and Tree View [ui/src/app/(dashboard)/projects/[id]/workspace/_components/ExplorerPanel.tsx]
- [x] [Review][Patch] Misleading Cursor Style for Non-Clickable Files [ui/src/components/workspace/FileExplorer.tsx]
- [x] [Review][Patch] Poor Prop Mocking in Tests [ui/src/app/(dashboard)/projects/[id]/workspace/_components/ExplorerPanel.test.tsx]
- [x] [Review][Patch] Optional onLinkWorkflows called without check [ui/src/app/(dashboard)/projects/[id]/workspace/_components/ExplorerPanel.tsx:312]
- [x] [Review][Defer] State Reset on Parent Unmount [ui/src/components/workspace/FileExplorer.tsx] — deferred, pre-existing

## Dev Notes

- **Technology Stack**: React, TypeScript, Next.js (App Router), Tailwind CSS, Lucide React (for icons).
- **State Preservation Pattern**: Do NOT unmount the tab contents. Use:
  ```tsx
  <div className={cn("flex-1 overflow-y-auto", activeTab !== "artifacts" && "hidden")}>
    {/* Existing ExplorerPanel content */}
  </div>
  <div className={cn("flex-1 overflow-y-auto", activeTab !== "file-explorer" && "hidden")}>
    <FileExplorer ... />
  </div>
  ```
  This ensures the DOM nodes persist, preserving scroll position and local React state (like expanded folders in `FileExplorer` once implemented).
- **File Locations**:
  - Modify: `ui/src/app/(dashboard)/projects/[id]/workspace/_components/ExplorerPanel.tsx`
  - Create: `ui/src/components/workspace/FileExplorer.tsx` (placeholder)
- **Design Alignment**: The tabs should look clean and integrated, using the theme's border and text colors. The active tab should have a subtle highlight (e.g., border-b-2 border-primary or similar).

### Project Structure Notes

- **Frontend components**: `ui/src/app/(dashboard)/projects/[id]/workspace/_components/` for page-specific layout, and `ui/src/components/workspace/` for reusable components like `FileExplorer`.
- **Dependency**: Story 4.8 will implement the actual recursive file explorer API and component logic. This story (4.2) focuses purely on the container, tab navigation, and state preservation.

### References

- [Source: ui/src/app/(dashboard)/projects/[id]/workspace/_components/ExplorerPanel.tsx](/ui/src/app/(dashboard)/projects/[id]/workspace/_components/ExplorerPanel.tsx)
- [Source: docs/spec/planning-artifacts/epics.md#Story 4.2](docs/spec/planning-artifacts/epics.md#L385-L401)
- [Source: docs/spec/planning-artifacts/ux-design-specification.md#Split-Pane Minimalist with Unified Persona Workspaces](docs/spec/planning-artifacts/ux-design-specification.md#L211-L221)

## Dev Agent Record

### Agent Model Used

Gemini Next

### Debug Log References

- None.

### Completion Notes List

- Implemented the Tabbed Workspace Explorer using a clean, Material 3/Tailwind inspired tab header.
- Integrated the tabs "Artifacts" and "File Explorer" with stateful switching, defaulting to "Artifacts".
- Wrapped original explorer items in an Artifacts container, and created a new interactive placeholder `FileExplorer` component representing a project workspace folder structure.
- Used CSS-based visibility toggling (Tailwind `hidden` class) with independent scroll areas to ensure that both views remain fully mounted in the DOM. This successfully preserves their scroll position and folder expansion states.
- Authored a comprehensive unit test suite in `ExplorerPanel.test.tsx` verifying default states, tab switching, and DOM-preservation.
- Ran the full vitest test suite; all 48 tests passed without regression.

### File List

- `ui/src/app/(dashboard)/projects/[id]/workspace/_components/ExplorerPanel.tsx` (Modified)
- `ui/src/components/workspace/FileExplorer.tsx` (Created)
- `ui/src/app/(dashboard)/projects/[id]/workspace/_components/ExplorerPanel.test.tsx` (Created)
