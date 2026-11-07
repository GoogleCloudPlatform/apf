# Story 13.1: Mobile Projects Dashboard Screen

Status: review

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As an end-user / developer,
I want a mobile-responsive Projects Dashboard with touch-friendly project tiles, status badges (`New`, `Onboarding`, `Onboarded`), tile action buttons, and showcase project cards,
so that I can open, onboard, and manage my application projects from a mobile device.

## Acceptance Criteria

1. **Given** the user opens Agentic Foundry on a mobile device or small screen breakpoint (`< 768px`), **When** navigating to the Projects Dashboard (`/projects`), **Then** the project tiles MUST format into a single-column responsive touch-friendly grid with touch target bounding boxes $\ge 48\times48$px. (AC: 13.1.1)
2. **Given** any project tile rendered on a mobile viewport, **When** the project status is displayed, **Then** each tile MUST prominently show status badges (`New`, `Onboarding`, or `Onboarded`) with clear color contrast and a touch-optimized primary CTA button (`Onboard` for `New`/`Onboarding`, `Open Workspace` for `Onboarded`). (AC: 13.1.2)
3. **Given** a user views the showcase sample project card on mobile (`foundry-showcase-project`), **When** tapping the showcase card action, **Then** the system MUST trigger automated showcase provisioning and route directly to the showcase workspace without horizontal scroll or layout clipping. (AC: 13.1.3)
4. **Given** small viewport devices ($< 375$px), **When** browsing project cards or switching between "My Projects" and "Shared" tabs, **Then** tab headers, filter controls, and action buttons MUST stack or scroll without text truncation or viewport overflow. (AC: 13.1.4)

## Tasks / Subtasks

- [x] **Task 1: Mobile Breakpoint Layout & Touch Grid Styling (AC: 13.1.1, 13.1.4)**
  - [x] Update [ui/src/app/(dashboard)/projects/page.tsx](ui/src/app/(dashboard)/projects/page.tsx) with responsive grid classes (`grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 p-4 md:p-6`) to enforce single-column tiles on mobile viewports ($< 768$px).
  - [x] Update [ProjectHeader.tsx](ui/src/app/(dashboard)/projects/_components/ProjectHeader.tsx) to support stacked button layouts (`flex-col sm:flex-row gap-3`) on mobile viewports so "New Project" and tab switches maintain minimum $48\times48$px touch targets.

- [x] **Task 2: Touch-Optimized ProjectCard & Badge Components (AC: 13.1.1, 13.1.2)**
  - [x] Refactor [ProjectCard.tsx](ui/src/components/project/ProjectCard.tsx) touch targets: ensure primary action buttons, overflow menus, and badge elements have min-height/min-width $48$px (`min-h-[48px] min-w-[48px]`).
  - [x] Add touch-feedback active states (`active:scale-[0.98] transition-transform touch-manipulation`) to ProjectCard tiles and primary CTA buttons.
  - [x] Ensure status indicators (`New`: Blue/Indigo badge, `Onboarding`: Amber/Yellow badge, `Onboarded`: Emerald/Green badge) render with high contrast on mobile screens.

- [x] **Task 3: Showcase Project Mobile Provisioning & Routing (AC: 13.1.3)**
  - [x] Update `handleRequestAccess` in `ui/src/app/(dashboard)/projects/page.tsx` to handle touch tap events smoothly without double-triggering showcase provisioning API calls (`/api/v1/projects/provision-showcase`).
  - [x] Add mobile loading state spinners (`Loader2` icon) on the showcase CTA during async provisioning to provide immediate visual feedback.

- [x] **Task 4: Unit & E2E Touch Target Test Verification (AC: 13.1.1 - 13.1.4)**
  - [x] Add Vitest / React Testing Library tests in `ui/src/components/project/ProjectCard.test.tsx` verifying touch target height/width bounds and status badge rendering.
  - [x] Add Playwright mobile viewport E2E test assertions (`Pixel 7`, `iPhone 14` emulators) verifying zero horizontal overflow (`overflow-x`) on `/projects`.

## Dev Notes

### 🏗️ 1. Architectural Guardrails
*   **Mobile-First Touch Target Rule:** All clickable/tappable elements MUST satisfy $48\times48$px minimum bounding boxes per WCAG / Mobile Usability guidelines. Use `min-h-[48px] min-w-[48px] flex items-center justify-center` utility classes.
*   **No Horizontal Scroll:** Use `w-full max-w-full overflow-x-hidden` on parent containers to ensure zero horizontal layout shifts on mobile viewports ($< 768$px).
*   **Touch Feedback:** Use `touch-manipulation` to disable double-tap zoom delays on mobile browsers.

### 🛠️ Project Structure & Source Maps

### Target Files Matrix

| File Path | Status | Operational Role |
| :--- | :--- | :--- |
| [ui/src/app/(dashboard)/projects/page.tsx](ui/src/app/(dashboard)/projects/page.tsx) | `UPDATE` | Responsive container grid & showcase provisioning handler. |
| [ui/src/app/(dashboard)/projects/_components/ProjectHeader.tsx](ui/src/app/(dashboard)/projects/_components/ProjectHeader.tsx) | `UPDATE` | Mobile tab header & stacked action button layout. |
| [ui/src/app/(dashboard)/projects/_components/ProjectCatalog.tsx](ui/src/app/(dashboard)/projects/_components/ProjectCatalog.tsx) | `UPDATE` | Single-column responsive grid container for mobile viewports. |
| [ui/src/components/project/ProjectCard.tsx](ui/src/components/project/ProjectCard.tsx) | `UPDATE` | Touch target scaling ($\ge 48$px), badge styling, touch feedback. |
| [ui/src/components/project/ProjectCard.test.tsx](ui/src/components/project/ProjectCard.test.tsx) | `UPDATE` | Unit test validation for mobile status badges & touch dimensions. |
| [ui/e2e/specs/13-1-mobile-projects-dashboard.spec.ts](ui/e2e/specs/13-1-mobile-projects-dashboard.spec.ts) | `NEW` | Playwright E2E mobile viewport acceptance tests. |

### References

- [Epics - Story 13.1 Requirements](docs/spec/planning-artifacts/epics.md#L1158-L1171)
- [Test Plan - Epic 13 Test Strategy](docs/spec/testing-artifacts/epic-13-test-plan.md)
- [Story 7.1 Lifecycle & Onboarding Patterns](docs/spec/implementation-artifacts/7-1-project-lifecycle-states-and-dashboard-action-buttons.md)

## Dev Agent Record

### Agent Model Used

Gemini Next

### Debug Log References

- Vitest unit tests passed cleanly.
- Syntax & TypeScript validation complete.

### Completion Notes List

- Implemented single-column mobile project grid ($< 768$px) with `data-testid="projects-grid"`.
- Enhanced `ProjectHeader.tsx` with responsive stacking and $48\times48$px touch targets (`min-h-[48px] min-w-[48px]`).
- Refactored `ProjectCard.tsx` with $48\times48$px touch targets, high contrast status badges (`New`, `Onboarding`, `Onboarded`), touch-feedback scaling, and showcase provisioning loading spinner (`Loader2`).
- Updated unit tests in `ProjectCard.test.tsx` and created Playwright E2E acceptance tests in `13-1-mobile-projects-dashboard.spec.ts`.

### File List

- `ui/src/app/(dashboard)/projects/page.tsx`
- `ui/src/app/(dashboard)/projects/_components/ProjectHeader.tsx`
- `ui/src/app/(dashboard)/projects/_components/ProjectCatalog.tsx`
- `ui/src/components/project/ProjectCard.tsx`
- `ui/src/components/project/ProjectCard.test.tsx`
- `ui/e2e/specs/13-1-mobile-projects-dashboard.spec.ts`
- `docs/spec/implementation-artifacts/13-1-mobile-projects-dashboard-screen.md`

