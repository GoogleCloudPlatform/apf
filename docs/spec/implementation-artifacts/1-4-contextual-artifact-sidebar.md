# Story 1.4: Contextual Artifact Sidebar

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a User (PM/UX/Engineer),
I want a collapsible right-hand pop-out menu for artifact navigation,
So that I can easily browse artifacts related to my active persona and reference artifacts from other disciplines without cluttering the screen.

## Acceptance Criteria

1. **Given** the user is in the APF workspace, **When** they click a collapsed persona/action icon on the right rail, **Then** the panel dynamically expands to show the contextual navigation area. (AC: 1.4.1)
2. **Given** the expanded panel, **When** they view the active persona outcomes tab, **Then** it MUST display a list of all required and custom artifacts collected by the specific active persona. (AC: 1.4.2)
3. **Given** the expanded panel, **When** they view the references tab, **Then** it MUST allow the user to view other disciplines' (UX, Solutioning, Analysis, etc.) artifact collections for reference. (AC: 1.4.3)

---

## Tasks / Subtasks

- [x] **Task 1: Design and Implement the Collapsible Sidebar Layout & Right Rail (AC: 1.4.1)**
  - [x] Implement the vertical menu rail (`_buildRightRail()` in [workspace.dart](ui/lib/screens/workspace/workspace.dart)) containing key buttons: Methods (`Icons.build`), Status (`Icons.assignment`), and References (`Icons.auto_stories`).
  - [x] Add control states (`_isSidebarOpen`, `_selectedSidebarTab`) to manage smooth dynamic layout changes (Flex 2 for sidebar, Flex 8 for chat pane/workspace views).
- [x] **Task 2: Build the Phase Outcome Artifacts Panel & Status Tracking (AC: 1.4.2)**
  - [x] Implement the `PhaseOutcomeArtifacts` widget inside [artifacts.dart](ui/lib/screens/workspace/widgets/artifacts.dart) to show all matching/required files.
  - [x] Add dynamic folder and directory nesting with expand/collapse logic, complete with progress colors (Not Started: Grey, In Progress: Orange, Complete: Green).
- [x] **Task 3: Build Cross-Discipline Reference Artifacts & Markdown Preview (AC: 1.4.3)**
  - [x] Implement reference list builder (`_buildReferencesContent(...)`) mapping and rendering available artifacts from other non-active phase channels.
  - [x] Wire dynamic callbacks to support rich markdown previews (`_showMarkdownPreview()`) using secure `nextjs_markdown` overlays.

---

## Dev Notes

### 🏗️ 1. Architectural Guardrails Complied
*   **Collapsible Responsive Workspace Layout:** The UI adheres to high-performance responsive layout strategies, utilizing Flex split grids (Flex 2 for contextual panel, Flex 8 for primary viewport) to ensure clean screen use on multi-resolution setups.
*   **Stateful Synchronization:** Changes to active workspace projects immediately rebuild sidebar parameters via the standard Firestore stream and `WorkspaceData` future listeners.

---

## 🛠️ Project Structure & Source Maps

### 1. Finalized Files Matrix

| File Path | Status | Operational Role |
| :--- | :--- | :--- |
| [ui/lib/screens/workspace/workspace.dart](ui/lib/screens/workspace/workspace.dart) | `COMPLETED` | Main page view container, managing the collapsible state, right-hand navigation rail, and markdown previews. |
| [ui/lib/screens/workspace/widgets/artifacts.dart](ui/lib/screens/workspace/widgets/artifacts.dart) | `COMPLETED` | Implementation of `PhaseOutcomeArtifacts` and `ReferenceCard` components for nested folder rendering and interactive actions. |

---

## Dev Agent Record

### Agent Model Used

Gemini Next

### Debug Log References

- Verified all collapsible states and rail navigation callbacks.
- Confirmed that non-active phase artifacts display under the References tab and launch the preview dialog successfully.

### Completion Notes List

- ✅ Collapsible sidebar and right action rail layout built.
- ✅ Target state-based `PhaseOutcomeArtifacts` directory and status map complete.
- ✅ Discipline references list with Markdown preview overlays operational.

### File List

- `ui/lib/screens/workspace/workspace.dart`
- `ui/lib/screens/workspace/widgets/artifacts.dart`
