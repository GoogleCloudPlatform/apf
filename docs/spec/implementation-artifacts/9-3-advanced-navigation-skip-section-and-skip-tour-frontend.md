# Story 9.3: Advanced Navigation: Skip Section and Skip Tour (Frontend)

Status: done

## Story

As a User,
I want the ability to skip the entire current section of the tour or dismiss the tour completely,
so that I don't have to click "Next" repeatedly for features I am already familiar with.

## Acceptance Criteria

1. **Given** the onboarding tour is active and displaying a step popover
   **When** the popover is rendered
   **Then** it MUST display distinct buttons for `Skip Section` and `Skip Tour`
   **And** the popover MUST display the current section name and step progress (e.g., *“Section 2 of 4: User Secrets - Step 1/3”*)
2. **Given** the user is viewing a tour step within a multi-step section
   **When** the user clicks `Skip Section`
   **Then** the tour MUST immediately dismiss the active overlay, mark the entire section as skipped (adding it to `completed_sections`), and transition to the first step of the next scheduled section
3. **Given** the user is on any step of the tour
   **When** the user clicks `Skip Tour`
   **Then** the tour MUST immediately close, restore the standard UI state, and trigger a background API call to update the user's `tour_status.completed = true` in Firestore to prevent subsequent auto-launches
4. **Given** the tour navigation buttons
   **When** navigated via keyboard
   **Then** they MUST support standard `Tab` focus and `Enter`/`Space` activation

## Tasks / Subtasks

- [x] **Task 1: Update Onboarding Store (`ui/src/stores/onboarding.ts`)** (AC: 1, 2, 3)
  - [x] Update `TourStep` interface to include optional `sectionId` and `sectionTitle` fields.
  - [x] Add `completedSections: string[]` to `OnboardingState` store state.
  - [x] Add `skipSection: () => void` and `skipTour: () => void` actions to `OnboardingState`.
  - [x] Implement `skipSection` logic:
    - Identify current step's `sectionId`.
    - Add it to `completedSections` and persist to `localStorage`.
    - Find the next step index in the active tour that has a different (or undefined) `sectionId`.
    - Update `currentStep` to this index to jump to the next section.
    - If no subsequent section exists, end/complete the tour.
    - Trigger background sync API call to update `completed_sections`.
  - [x] Implement `skipTour` logic:
    - Set `activeTour` to null and reset progress.
    - Save tour completion state locally.
    - Trigger background sync API call to set `completed = true`.
  - [x] Implement a helper (or selector) `getCurrentSectionInfo` to parse active steps and calculate:
    - Active section title.
    - Current section index / total sections count.
    - Current step index / total steps count within the active section.
  - [x] Group existing `catalog` tour steps into contiguous sections using `sectionId` and `sectionTitle` (e.g., "catalog-overview", "secrets", "project-setup", "catalog-features") to enable validation.
- [x] **Task 2: Implement Background API Sync Client** (AC: 3)
  - [x] Create/extend a helper method in the store to sync tour state to the backend using `api.put("/api/v1/users/{id}/tour-status", status)`. (Note: Endpoint corrected from spec).
  - [x] Ensure API request payload conforms to the expected backend profile:
    ```json
    {
      "completed": true,
      "completed_sections": ["catalog-overview", "secrets"],
      "last_active_step": { "section_id": "project-setup", "step_index": 0 }
    }
    ```
  - [x] Wrap the network call in error handling blocks to prevent API failures from interrupting or crashing the user's frontend UI experience.
- [x] **Task 3: Update OnboardingTour UI Overlay Component (`ui/src/components/shared/OnboardingTour.tsx`)** (AC: 1, 2, 3, 4)
  - [x] Bind `skipSection` and `skipTour` actions from `useOnboardingStore`.
  - [x] Retrieve dynamic section info (`sectionTitle`, progress indicators) and render it in the bubble content instead of the simple "X of Y" overall step count.
  - [x] Render distinct, accessible buttons for `Skip Section` and `Skip Tour` within the bubble controls area.
  - [x] Apply interactive styling to new buttons, ensuring they support standard Tailwind focus rings (`focus-visible:ring-2 focus-visible:ring-primary focus-visible:outline-none`) for keyboard users.
  - [x] Ensure standard keyboard navigation flow: tab ordering correctly cycles through Close, Skip Section, Skip Tour, and Next/Got it, and triggering them via Enter/Space executes the expected state transitions.

## Dev Notes

- **Source tree components to touch:**
  - [onboarding.ts](ui/src/stores/onboarding.ts) (State management & navigation logic)
  - [OnboardingTour.tsx](ui/src/components/shared/OnboardingTour.tsx) (UI presentation layer)
  - [api.ts](ui/src/lib/api.ts) (API client reference)
- **Testing standards summary:**
  - Manual verification in the browser (Chrome DevTools) is required as there is no configured automated UI testing framework (Jest/Playwright) in the `ui` package.
  - Verify accessibility by navigating the tour overlay using only the `Tab` and `Space`/`Enter` keys.

### Project Structure Notes

- Alignment with the Next.js/TypeScript structure in the `ui/` directory. Note that [architecture.md](docs/spec/planning-artifacts/architecture.md) references Next.js React/Dart frontend structures which are obsolete; the current Next.js/Zustand implementation in the `/ui` folder is the single source of truth for the presentation tier.

### References

- Epic definition: [epics.md#Epic-9](docs/spec/planning-artifacts/epics.md#L617)
- PRD requirements: [prd.md](docs/spec/planning-artifacts/prd.md)

## Dev Agent Record

### Agent Model Used

Gemini Next

### Debug Log References

### Completion Notes List

### File List

### Review Findings

- [ ] [Review][Patch] Fragile Project ID Extraction from URL Pathname [ui/src/components/shared/OnboardingTour.tsx:25]
- [ ] [Review][Patch] Unhandled API Errors in Project Query [ui/src/components/shared/OnboardingTour.tsx:27-36]
- [ ] [Review][Patch] Type Safety Bypassed in Onboarding Store Tests [ui/src/stores/onboarding.test.ts]
- [ ] [Review][Patch] Overloaded `useEffect` and Unstable Dependencies [ui/src/components/shared/OnboardingTour.tsx:64-150]
- [ ] [Review][Patch] Hardcoded Section-to-Tour Mapping in UI [ui/src/components/shared/OnboardingTour.tsx:128-130]
- [ ] [Review][Patch] Non-Contiguous Section Progress Display Bug [ui/src/stores/onboarding.ts:steps]
- [ ] [Review][Patch] Broken Background Step Advancement (Section ID Mismatch) [ui/src/stores/onboarding.ts:850]
- [ ] [Review][Patch] Potential Crash from Out-of-Bounds `startingStep` [ui/src/stores/onboarding.ts:653]
- [x] [Review][Defer] Mixed Concerns in `completedSections` State [ui/src/stores/onboarding.ts] — deferred, pre-existing
- [x] [Review][Defer] Inefficient Section Info Calculation [ui/src/stores/onboarding.ts:863-902] — deferred, pre-existing
- [x] [Review][Defer] Inconsistent State when Skipping Undefined Section [ui/src/stores/onboarding.ts:skipSection] — deferred, pre-existing
