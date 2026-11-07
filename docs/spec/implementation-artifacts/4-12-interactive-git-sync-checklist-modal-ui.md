# Story 4.12: Interactive Git Sync Checklist Modal UI & Conflict Resolution

Status: ready-for-dev

## Story

As a Product Manager, UX Designer, or Developer,
I want an interactive checklist modal UI to display during remote Git synchronization,
so that I can see the complete step progress, dynamic branch/ref names created behind the scenes, where the process got stuck, and receive clear guidance on how to resolve merge conflicts.

## Acceptance Criteria

1. **Given** a user triggers Git sync or pull in the UI, **When** the operation finishes or encounters a conflict, **Then** the UI MUST display the `GitSyncModal` checklist modal. (AC: 4.12.1)
2. **Given** the `GitSyncModal` component, **When** rendering branch metadata, **Then** it MUST display badges for `Base Branch`, `Spec Branch`, `Backup Ref` (e.g. `refs/foundry/backups/...`), and `Shadow Branch` (e.g. `refs/foundry/shadow-rebase-...`). (AC: 4.12.2)
3. **Given** the step checklist in `GitSyncModal`, **When** rendering execution steps, **Then** each step MUST display a status icon: `completed` (green checkmark), `failed` (red cross), `skipped` (gray dash), or `in_progress` (spinner). (AC: 4.12.3)
4. **Given** a rebase conflict (`status: "CONFLICT_DETECTED"`), **When** `GitSyncModal` renders the conflict callout box, **Then** it MUST highlight the stuck step, display the list of conflicting files, present a prominent `[ Resolve Conflict on GitHub PR ]` primary button, and provide a collapsible block with copy-pasteable CLI commands. (AC: 4.12.4)

---

## Tasks / Subtasks

- [ ] **Task 1: Build `GitSyncModal.tsx` Component (`ui/src/components/project/GitSyncModal.tsx`) (AC: 4.12.1 - 4.12.4)**
  - [ ] **Subtask 1.1:** Create modal dialog container using Radix Dialog / Tailwind UI primitives.
  - [ ] **Subtask 1.2:** Build Branch Context Chips section (`baseBranch`, `specBranch`, `backupRef`, `shadowBranch`).
  - [ ] **Subtask 1.3:** Build Step Checklist list renderer with status icons and detail text.
  - [ ] **Subtask 1.4:** Build Conflict Resolution Alert Box with `pullRequestUrl` button and collapsible CLI commands accordion.
- [ ] **Task 2: Integrate `GitSyncModal` in Housekeeping Page (`ui/src/app/(dashboard)/projects/[id]/housekeeping/housekeeping.tsx`) (AC: 4.12.1)**
  - [ ] **Subtask 2.1:** Add modal state management (`isSyncModalOpen`, `syncChecklistData`).
  - [ ] **Subtask 2.2:** Connect `syncWorkspaceMutation` response to open `GitSyncModal` and pass checklist payload.
- [ ] **Task 3: Visual Polish & Accessibility (AC: 4.12.3, 4.12.4)**
  - [ ] **Subtask 3.1:** Ensure dark/light mode compatibility and clear visual hierarchy for failed vs completed steps.
  - [ ] **Subtask 3.2:** Verify keyboard navigation and focus management inside the modal.
- [ ] **Task 4: Testing & Verification (AC: 4.12.1 - 4.12.4)**
  - [ ] **Subtask 4.1:** Write Playwright / React Testing Library tests for `GitSyncModal` verifying clean sync state and conflict state rendering.

---

## Dev Notes

### 🏗️ Architectural Guardrails
- **Non-Streaming Modal**: Displays complete checklist upon completion without requiring WebSocket streaming.
- **Clear Call to Action**: Highlighted orange/red alert banner with direct GitHub PR link for 1-click navigation.

---

## 🛠️ Target Files Matrix

| File Path | Status | Operational Role |
| :--- | :--- | :--- |
| `ui/src/components/project/GitSyncModal.tsx` | `NEW` | Interactive Git Sync Checklist Modal UI component. |
| `ui/src/app/(dashboard)/projects/[id]/housekeeping/housekeeping.tsx` | `UPDATE` | Wire sync button and mutation to trigger `GitSyncModal`. |

---

## Dev Agent Record

### Agent Model Used
Gemini Next

### Debug Log References
### Completion Notes List
### File List
