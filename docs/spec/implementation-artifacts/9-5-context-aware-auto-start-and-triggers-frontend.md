# Story 9.5: Context-Aware Auto-Start & Triggers (Frontend)

Status: done

## Story

As a User,
I want the tour to automatically start when I access a new section for the first time,
so that I am guided through the features of that section exactly when I need them.

## Acceptance Criteria

1. **Given** a user logs in for the first time (`tour_status.completed == false` and `completed_sections` is empty)
   **When** they land on the Project Dashboard / Overview (`/projects`)
   **Then** the frontend MUST automatically trigger the Dashboard tour section (`catalog-overview` section of the `catalog` tour)
2. **Given** a user clicks the "Replay Tour" (Sparkles icon) action in the header
   **When** triggered
   **Then** the frontend MUST launch the tour from the first step of the section corresponding to the current page/context (e.g., Dashboard section if on `/projects`, workspace overview if on workspace, settings if on `/settings`) regardless of completion status
3. **Given** the tour is in-progress (not fully completed)
   **When** the user navigates to a new screen or tab (e.g., the Onboarding Tab in Workspace)
   **Then** the system MUST verify if the tour section corresponding to this screen is in `completed_sections`
   **And** if it is NOT in `completed_sections`, the frontend MUST automatically start the tour from the first step of that screen's section
4. **Given** the user navigates to a screen
   **When** the corresponding tour section is already marked as completed or skipped in `completed_sections`
   **Then** the tour MUST NOT launch automatically

## Tasks / Subtasks

- [x] **Task 1: Update Onboarding Store (`ui/src/stores/onboarding.ts`)** (AC: 1, 3, 4)
  - [x] Add `tourStatus` state to `OnboardingState` matching backend structure (including `completed` boolean).
  - [x] Implement helper `getSectionFirstStepIndex(tour: TourType, sectionId: string): number` to locate the first step index for a given section.
  - [x] Ensure `startTour` action supports passing a starting step index and resets `activeTour` and `currentStep` state appropriately.
  - [x] Load `tourStatus` from backend (via profile endpoint) or fallback to local storage on init.
- [x] **Task 2: Implement Context-Aware Trigger Logic in UI (`ui/src/components/shared/OnboardingTour.tsx`)** (AC: 1, 3, 4)
  - [x] Import `useSearchParams` from `next/navigation` to react to query parameter changes (specifically `phase` in workspace).
  - [x] Extract `projectId` from pathname to dynamically query project details using React Query `useQuery` (caches and shares data with Workspace page).
  - [x] Refactor the auto-trigger `useEffect` to:
    - [x] Return early if `tourStatus.completed` is true.
    - [x] Return early if `activeTour` is already set (running).
    - [x] Map route and query parameter states to tour sections:
      - `/projects` -> `catalog-overview` section of `catalog` tour.
      - `/settings` -> `settings-overview` section of `settings` tour.
      - `/projects/[id]/workspace` with `phase=ONBOARDING` -> `onboarding-tab` section of `workspace` tour.
      - `/projects/[id]/workspace` with other phases -> `workspace-overview` section of `workspace` tour.
    - [x] Check if the determined section is in `completedSections`.
    - [x] If not completed, trigger `startTour` at the calculated section starting step.
- [x] **Task 3: Refine Replay Button Trigger (`ui/src/components/shared/TopBar.tsx`)** (AC: 2)
  - [x] Verify/refine the "Quick Tour" button in the `TopBar` to ensure clicking it calls `startTour` for the current page context without being blocked by `completedSections` or `tourStatus.completed`.


### Review Findings

- [x] [Review][Patch] Auto-trigger tour loop when tour completed or skipped [OnboardingTour.tsx:140]
- [x] [Review][Patch] Skipped sections are lost on page reload [onboarding.ts:1083]
- [x] [Review][Patch] `advanceStep` fails to match background progress [onboarding.ts:1238]
- [x] [Review][Patch] `TopBar` Quick Tour action defaults to wrong tour for un-onboarded workspaces [TopBar.tsx:255-261]
- [x] [Review][Patch] Fragile hardcoded step index for project creation replay [TopBar.tsx:266]
- [x] [Review][Patch] Ignored HTTP Error Status in `syncTourStateWithBackend` [onboarding.ts:1201]
- [x] [Review][Patch] Race Condition in `syncTourStateWithBackend` [onboarding.ts:1177-1221]
- [x] [Review][Patch] Missing Error Handling for Project Query in `OnboardingTour` [OnboardingTour.tsx:27-36]
- [x] [Review][Patch] Fragile `projectId` Extraction from Pathname [OnboardingTour.tsx:25]
- [x] [Review][Patch] Granular section ID stored/cast as TourType breaks resume logic [ui/src/stores/onboarding.ts:508]
- [x] [Review][Patch] Auto-trigger Loop for Naturally Completed Sections [ui/src/stores/onboarding.ts:417]
- [x] [Review][Patch] Potential crash if localStorage has non-array value for completed sections [ui/src/stores/onboarding.ts:81]
- [x] [Review][Patch] Missing guard for empty userId in loadTourStatusFromBackend [ui/src/stores/onboarding.ts:502]
- [x] [Review][Patch] Race condition in optimistic update rollback [ui/src/stores/onboarding.ts:596]
- [x] [Review][Patch] Fragile projectId extraction using useParams [ui/src/components/shared/OnboardingTour.tsx:20]
- [x] [Review][Defer] Duplicate project details query [ui/src/components/shared/OnboardingTour.tsx:24] — deferred, pre-existing
- [x] [Review][Defer] Hardcoded 1500ms timeout for rendering [ui/src/components/shared/OnboardingTour.tsx:40] — deferred, pre-existing
- [x] [Review][Defer] Too many dependencies in auto-trigger useEffect [ui/src/components/shared/OnboardingTour.tsx:40] — deferred, pre-existing
- [x] [Review][Defer] Inconsistent tour completion check ignoring workspace_onboarding [ui/src/stores/onboarding.ts:102] — deferred, pre-existing
- [x] [Review][Defer] Inefficient grouping of steps in getCurrentSectionInfo [ui/src/stores/onboarding.ts:630] — deferred, pre-existing
- [x] [Review][Defer] Error toast spam on query retries [ui/src/components/shared/OnboardingTour.tsx:40] — deferred, pre-existing


## Dev Notes

- **Source tree components to touch:**
  - [onboarding.ts](ui/src/stores/onboarding.ts) (Store states & helper actions)
  - [OnboardingTour.tsx](ui/src/components/shared/OnboardingTour.tsx) (Dynamic trigger logic hooks)
  - [TopBar.tsx](ui/src/components/shared/TopBar.tsx) (Manual trigger button action)
- **Testing standards summary:**
  - Manual verification via Chrome DevTools.
  - Verify auto-trigger by clearing localStorage or resetting Firestore `tour_status` and navigating between pages (Dashboard, settings, workspace).
  - Verify manual trigger by clicking Sparkles icon in header on different pages and ensuring it starts the correct tour.

### Project Structure Notes

- Uses standard Next.js navigation hooks (`usePathname`, `useSearchParams`) and React Query for project state query cache.

### References

- Epic definition: [epics.md#Epic-9](docs/spec/planning-artifacts/epics.md#L617)
- PRD requirements: [prd.md](docs/spec/planning-artifacts/prd.md)

## Dev Agent Record

### Agent Model Used

Gemini Next

### Debug Log References

- None.

### Completion Notes List

- Added `tourStatus` state to `OnboardingState` matching Firestore schema.
- Implemented `getSectionFirstStepIndex` helper to locate starting steps for sections.
- Assigned granular `sectionId` and `sectionTitle` to all steps across tours in the store config.
- Refactored `useEffect` auto-trigger in `OnboardingTour` to support section-level completion checks and context-aware page-matching resume using granular section IDs.
- Incorporated React Query `useQuery` cache sharing to retrieve project status for correct workspace tour phase fallback when phase query param is missing.
- Updated `TopBar` replay button to support page/phase-specific replay context (triggers `workspace_onboarding` if in ONBOARDING phase).
- Extended test coverage in `onboarding.test.ts` with assertions verifying the new `tourStatus` state and `getSectionFirstStepIndex` helper logic.
- Configured path alias resolution for `vitest` tests.
- Fixed `api` mock in `onboarding.test.ts` to prevent `TypeError` during test runs when sync actions are triggered.

### File List

- `ui/src/stores/onboarding.ts`
- `ui/src/components/shared/OnboardingTour.tsx`
- `ui/src/components/shared/TopBar.tsx`
- `ui/src/stores/onboarding.test.ts`
- `ui/src/components/shared/OnboardingTour.test.tsx`
- `ui/vitest.config.ts`
