# Story 9.2: Tour Integration for Newly Created Sections (Frontend)

Status: done

## Story

As a PM/UX Designer,
I want the onboarding tour to include steps for the newly created sections (Secrets, Project Discovery, File Upload),
so that users are introduced to these new features during the tour.

## Acceptance Criteria

1. **User Secrets Tour Step (Catalog / Project Overview)**:
   - When the catalog tour is active and guides the user to configure secrets:
     - The tour MUST highlight the "User Secrets" settings button (key icon, target: `#tour-user-secrets-button`) in the Project Overview / Quick Actions.
     - The tooltip MUST explain that they need to configure their personal API keys.
     - This step MUST be an action step (`isAction: true`), requiring the user to click the button to proceed.
   - When the Secrets Modal is opened:
     - The tour MUST automatically advance to the next step.
     - The tour MUST highlight the token input fields (target: `#tour-secrets-modal-input`) inside the modal.
     - The tooltip MUST explain that these tokens are stored securely per-user, encrypted at rest, and masked in the UI.

2. **Project Discovery Tour Steps (Dashboard & Onboarding Tab)**:
   - When the tour is active on the Dashboard and a project is in `New` or `Onboarding` state:
     - The tour MUST highlight the `Onboard` button on that project's card (target: `#tour-project-launch-button`).
     - The tooltip MUST explain that clicking this button will launch the repository discovery process.
     - This step MUST be an action step (`isAction: true`).
   - When the user clicks `Onboard` and is redirected to the workspace (which opens in `ONBOARDING` phase):
     - The workspace onboarding tour (`workspace_onboarding`) MUST start automatically.
     - **Step 1 (Phases)**: Highlight the Onboarding Phase tab (target: `#tour-workspace-phase-onboarding`). Explain the goal of this phase (scan and ingest).
     - **Step 2 (Start Scan)**: Highlight the "Start Onboarding" button (target: `#tour-onboarding-start-scan-button`). Explain that this triggers the background discovery agent. Action step (`isAction: true`).
     - **Step 3 (Progress Log)**: Once scan starts, highlight the Activity Stream / Progress Log area (target: `#tour-onboarding-activity-stream`). Explain that it shows live progress.
     - **Step 4 (Tree Picker)**: Once scan completes (detected by the appearance of the tree picker), auto-advance to highlight the Interactive Directory Tree Picker (target: `#tour-onboarding-tree-picker`). Explain how to select files and use the "Keep Existing" vs "Overwrite" toggles.
     - **Step 5 (Complete Onboarding)**: Highlight the "Complete Onboarding" button (target: `#tour-onboarding-complete-button`). Explain that this finalizes onboarding, commits `project-context.md`, and unlocks other tools. Action step (`isAction: true`).

3. **Chat File Upload Tour Step (Workspace Chat)**:
   - In the general `workspace` tour (active when project is `Onboarded`), after explaining the chat panel:
     - The tour MUST highlight the file attachment button (clip icon, target: `#tour-chat-file-upload-button`).
     - The tooltip MUST explain that they can upload text/code files (up to 10MB) to inject them as context for the agent.

## Tasks / Subtasks

- [ ] **Store Updates (`ui/src/stores/onboarding.ts`)**:
  - [ ] Add `workspace_onboarding` to `TourType` type definition.
  - [ ] Define steps for `workspace_onboarding` in `steps` record.
  - [ ] Insert Secrets button and modal steps into `catalog` tour steps.
  - [ ] Insert File Upload step into `workspace` tour steps.
- [ ] **Runner Updates (`ui/src/components/shared/OnboardingTour.tsx`)**:
  - [ ] Update auto-trigger effect to detect `phase=ONBOARDING` in search parameters and start `workspace_onboarding` tour.
  - [ ] Implement auto-advancing from Step 3 (Progress Log) to Step 4 (Tree Picker) in `workspace_onboarding` by observing when `#tour-onboarding-tree-picker` becomes present in the DOM.
  - [ ] Update `handleNext` to handle routing/actions for:
    - Secrets button click (wait for modal).
    - `Onboard` button click (routes to `/projects/${id}/workspace?phase=ONBOARDING`).
    - `Start Onboarding` button click.
    - `Complete Onboarding` button click.
- [ ] **UI Component Targets**:
  - [ ] Add `id="tour-user-secrets-button"` to the key icon button in Project Overview/Workspace.
  - [ ] Add `id="tour-secrets-modal-input"` (or similar) to token inputs in Secrets Modal.
  - [ ] Add `id="tour-onboarding-start-scan-button"` to the start scan button in Onboarding view.
  - [ ] Add `id="tour-onboarding-activity-stream"` to the progress log/activity stream container.
  - [ ] Add `id="tour-onboarding-tree-picker"` to the tree picker container.
  - [ ] Add `id="tour-onboarding-complete-button"` to the complete onboarding button.
  - [ ] Add `id="tour-chat-file-upload-button"` to the clip icon button in `ChatPane.tsx`.
  - [ ] Fix mismatch in `ProjectCard.tsx`: change class `tour-project-launch-button` to `id="tour-project-launch-button"` (or update store to use class selector `.tour-project-launch-button`).

### Review Findings

- [x] [Review][Decision] Performance bottleneck: MutationObserver vs Reactive state transition — The tour uses a MutationObserver on `document.body` to auto-advance when `#tour-onboarding-tree-picker` renders. Because logs stream frequently during the scan, this observer fires repeatedly, executing query selectors on every mutation. We need to decide if we should replace it with a reactive `useEffect` inside `OnboardingPanel` (which increases coupling but performs better) or keep the observer and optimize it to watch a narrower target.
- [x] [Review][Patch] State updates on unmounted component in OnboardingPanel [ui/src/app/(dashboard)/projects/[id]/workspace/_components/OnboardingPanel.tsx:36]
- [x] [Review][Patch] Silent fallback to success on API failure in production [ui/src/app/(dashboard)/projects/[id]/workspace/_components/OnboardingPanel.tsx:64]
- [x] [Review][Patch] Potential invalid routing to `/projects/null/workspace` [ui/src/components/shared/OnboardingTour.tsx:434]
- [x] [Review][Patch] Missing accessible labels for checkboxes in OnboardingPanel [ui/src/app/(dashboard)/projects/[id]/workspace/_components/OnboardingPanel.tsx:159]
- [x] [Review][Patch] Race condition allowing double-trigger of scan [ui/src/app/(dashboard)/projects/[id]/workspace/_components/OnboardingPanel.tsx:27]
- [x] [Review][Patch] Missing security explanation in secrets tooltip in onboarding.ts [ui/src/stores/onboarding.ts:100]
- [x] [Review][Defer] Uncontrolled/unbound checkboxes in Ingestion Picker [ui/src/app/(dashboard)/projects/[id]/workspace/_components/OnboardingPanel.tsx:158] — deferred, pre-existing
- [x] [Review][Defer] Fragile tour progression using hardcoded step indices [multiple files] — deferred, pre-existing

## Dev Notes

- **Architecture/Patterns**:
  - The tour system is built using Zustand (`useOnboardingStore`) and a custom overlay component (`OnboardingTour.tsx`).
  - Target selectors MUST be unique CSS selectors (IDs preferred: `#tour-...`).
  - Use `isAction: true` for steps that require the user to perform an action to advance.
- **Coordination**:
  - This story must coordinate with the implementation of Epic 2 (Secrets), Epic 7 (Discovery), and Epic 8 (File Upload) to ensure selectors match.
  - If those features are not yet implemented, the developer implementing them must ensure the IDs defined here are added to the corresponding elements.

### Project Structure Notes

- Fits into the existing `ui/src/stores/onboarding.ts` and `ui/src/components/shared/OnboardingTour.tsx`.
- No new files are required, only modifications to existing frontend files and adding IDs to components.

### References

- Tour Store: [onboarding.ts](ui/src/stores/onboarding.ts)
- Tour Runner: [OnboardingTour.tsx](ui/src/components/shared/OnboardingTour.tsx)
- Epic 9 Specs: [epics.md#Epic 9: Interactive Product Onboarding Tour](docs/spec/planning-artifacts/epics.md#L617)

## Dev Agent Record

### Agent Model Used

Gemini Next

### Debug Log References

### Completion Notes List

### File List
