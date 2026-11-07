# Story 2.1 (Revised): Idea-to-Production Pipeline Tracking Bar via Required Artifact Gating

**Epic**: Epic 2 - Project Collaboration & Pipeline Tracking
**Complexity**: Medium
**Status**: 📝 Planning

---

## 🎯 Goal

Refactor the Governance Gates and Pipeline Tracking UI to dynamically calculate phase completion based on the completion status of required artifacts in each phase, rather than relying on static database gate fields or external API state updates.

---

## User Review Required

> [!IMPORTANT]
> This shifts gating governance from explicit state mutations (`PUT /api/v1/projects/{id}/gates`) to policy-driven artifact requirements defined in the BMAD workflow template. To ensure correct behavior across both navigation and status visualization, the system distinguishes between two evaluation models:
> 1. **Dependency Gating Satisfaction**: For unlocking downstream workspace tabs, a phase is satisfied if all its `required: true` artifacts are complete, or if the phase has no required artifacts (e.g., `ANALYSIS`).
> 2. **Visual Status Tracking**: For UI status pills and pizza trackers, optional phases (like `ANALYSIS`) must evaluate the actual progress of their optional artifacts rather than short-circuiting to `Complete`. If optional artifacts are unstarted or in progress, the visual state must render as `Not Started` (🔘) or `In Progress` (🔵), and the tracker must not auto-advance past them.

---

## Proposed Changes

### Frontend Client (`ui`)

#### [MODIFY] [workspace.dart](ui/lib/screens/workspace/workspace.dart)
- Update `FutureBuilder` to await `_artifactsFuture` alongside `_projectFuture` and `_workflowsFuture`.
- Add helper method `_isPhaseSatisfied(String phase, List<Artifact> artifacts, WorkflowTemplate workflow)` to evaluate if required artifacts are complete (used for dependency gating).
- Update tab gating logic in `_buildPhaseTab` to use `_isPhaseSatisfied`.

#### [MODIFY] [project_overview.dart](ui/lib/screens/project/project_overview.dart)
- Add `_isPhaseComplete` and `_isPhaseInProgress` helper methods that evaluate visual status tracking (evaluating optional artifact progress for optional phases).
- Update `_buildPizzaTracker` to evaluate active tracker index dynamically using visual status tracking, ensuring it does not auto-advance past uncompleted optional phases.

#### [MODIFY] [project_card.dart](ui/lib/screens/project/project_card.dart)
- Add `_isPhaseComplete` and `_isPhaseInProgress` helper methods aligned with visual status tracking.
- Update mini-progress indicator rendering to correctly display `Not Started`, `In Progress`, and `Complete` states based on actual artifact activity.

---

## Verification Plan

### Automated Tests
- Next.js React widget / unit tests if applicable.

### Manual Verification
1. Launch the UI and browse Dashboard, Overview, and Workspace screens.
2. Verify that for a new project, `ANALYSIS` and `PLANNING` tabs are unlocked, while `SOLUTIONING` and `IMPLEMENTATION` are locked.
3. Mark the required `PRD` artifact in `PLANNING` as `Complete` or `In Progress`.
4. Verify that marking `PRD` complete immediately unlocks the `SOLUTIONING` tab and advances the pipeline tracker.
