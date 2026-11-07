---
title: 'Story 12.14: Super Admin Jobs Dashboard & Execution History Portal UI'
type: 'feature'
created: '2026-08-31'
status: 'done'
baseline_revision: '1b24e8896a77fed13c1da9246a368505b9038bfa'
review_loop_iteration: 0
followup_review_recommended: false
context: []
warnings: []
deferred: []
---

<intent-contract>

## Intent

**Problem:** Platform and system administrators lack a dedicated, centralized screen in the System Admin Portal to discover available batch housekeeping jobs, trigger them on-demand with configurable parameters (such as dry-run mode), and monitor both active in-flight and historical job executions with outcome statuses, duration metrics, error details, and log links without having to access the GCP Cloud Console or execute manual CLI commands.

**Approach:** Build a dedicated `/admin/jobs` dashboard in the System Admin Portal (`ui/src/app/(dashboard)/admin/jobs/page.tsx`) that:
1. Displays the registered **Job Catalog** dynamically fetched from `GET /api/v1/admin/jobs`.
2. Provides an interactive **Run Job Modal** allowing Super Admins to trigger any registered job on-demand with a dry-run checkbox and safety confirmation.
3. Renders a comprehensive **Job Execution History & Live Monitoring Table** dynamically queried from `GET /api/v1/admin/jobs/executions` with automatic polling during active runs, status badges (`RUNNING`, `SUCCEEDED`, `FAILED`, `CANCELLED`), formatted duration, error messages, and direct links to Cloud Logging.
4. Integrates the **"Jobs"** section seamlessly into the Super Admin navigation bar and dashboard overview.

## Boundaries & Constraints

**Always:**
- Guard `/admin/jobs` with Super Admin authorization (`is_super_admin: true`). Render a clear Access Denied state if a standard user or non-super-admin navigates to the page.
- Dynamically fetch registered jobs from `GET /api/v1/admin/jobs` to populate the catalog table/cards with job names, descriptions, and parameter capabilities.
- Dynamically fetch historical and active job executions from `GET /api/v1/admin/jobs/executions` with support for manual refresh and pagination (`pageSize`, `pageToken`).
- Automatically poll `GET /api/v1/admin/jobs/executions` (every 5 seconds) while any execution is in `RUNNING` or `PENDING` state, automatically stopping polling once all executions reach a terminal state (`SUCCEEDED`, `FAILED`, `CANCELLED`).
- Trigger jobs via `POST /api/v1/admin/jobs/{jobName}/run` with payload `{ "dryRun": true | false }`, displaying immediate toast notifications and updating the active execution list.
- Display status badges with consistent styling:
  - `RUNNING` / `PENDING`: Blue/Amber with an animated spinner.
  - `SUCCEEDED`: Green badge with checkmark icon.
  - `FAILED`: Red badge with alert icon and expandable/hoverable error details.
  - `CANCELLED`: Gray badge with slash/ban icon.
- Provide direct Cloud Logging deep-links (`logUri`) that open the execution logs in Google Cloud Console in a new browser tab (`target="_blank" rel="noopener noreferrer"`).
- Add the "Jobs" tab/link to the Admin Navigation sidebar (`AdminSidebar.tsx` / `admin/page.tsx` tabs).
- Author comprehensive unit and integration tests in `ui/src/app/(dashboard)/admin/jobs/page.test.tsx` verifying catalog rendering, modal triggers, dry-run toggling, execution table rendering, auto-polling lifecycle, and error state handling.
- Ensure all TypeScript checks (`tsc --noEmit`) and UI tests (`npm test`) pass with 100% success.

**Block If:**
- Upstream changes require storing custom execution tracking state in Firebase/Firestore; all data must be queried dynamically via API endpoints.

**Never:**
- Never modify or write `sprint-status.yaml`.
- Never store execution logs or states in local browser storage; always rely on the backend API and GCP Cloud Run v2 as the single source of truth.
- Never allow non-super-admins to view or trigger job execution controls.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| Super Admin Loads Jobs Page | Navigates to `/admin/jobs` as Super Admin | Fetches catalog (`GET /api/v1/admin/jobs`) and executions (`GET /api/v1/admin/jobs/executions`), renders catalog cards and history table | Displays error alert banner if API calls fail |
| Non-Super-Admin Access Attempt | Navigates to `/admin/jobs` without `is_super_admin: true` | Renders "Access Denied: Super Administrator access required" component | Blocks all API requests and hides trigger controls |
| On-Demand Job Execution (Dry Run) | Clicks "Run" on `global-workspace-provisioning`, selects "Dry Run", confirms modal | Sends `POST /api/v1/admin/jobs/global-workspace-provisioning/run` with `{ "dryRun": true }`, displays success toast, adds pending execution, starts polling | Displays error toast if backend returns error |
| On-Demand Job Execution (Live) | Clicks "Run" on `nfs-audit`, leaves Dry Run unchecked, confirms modal | Sends `POST /api/v1/admin/jobs/nfs-audit/run` with `{ "dryRun": false }`, displays success toast, starts polling | Displays error toast on failure |
| Active Job Polling | One or more executions have status `RUNNING` or `PENDING` | Automatically polls `GET /api/v1/admin/jobs/executions` every 5 seconds until all executions finish | Stops polling if page unmounts or all jobs complete |
| Failed Job Error Inspection | Historical execution has status `FAILED` with `errorMessage` | Displays red `FAILED` badge and renders error message tooltip/accordion with error details | Handles empty error message gracefully |
| External Log Deep-Link | User clicks "View Logs" button for an execution with `logUri` | Opens Cloud Logging URL in a new browser tab with `rel="noopener noreferrer"` | Disables button with tooltip if `logUri` is absent |
| Empty State (No Executions) | `executions` array is empty | Renders clean empty-state illustration: "No job executions found. Trigger a job from the catalog above to get started." | Clean UI without broken table rows |
| Manual Refresh | User clicks "Refresh" button | Triggers immediate fetch of executions table with spinning refresh icon | Resets loading state on completion |

</intent-contract>

## Code Map

- `ui/src/app/(dashboard)/admin/jobs/page.tsx` -- Main Super Admin Jobs Dashboard page handling state management, catalog fetching, execution polling, and modal orchestration.
- `ui/src/components/admin/JobsCatalogCard.tsx` -- Card/table component rendering the list of registered system jobs (`global-workspace-provisioning`, `nfs-audit`, `hanging-cleanup`) with trigger action buttons.
- `ui/src/components/admin/JobRunModal.tsx` -- Interactive confirmation dialog for launching a job with parameter selection (dry-run toggle) and safety disclaimer.
- `ui/src/components/admin/JobExecutionsTable.tsx` -- Searchable and filterable table displaying execution history, status badges, dry-run tags, duration, timestamps, and Cloud Logging links.
- `ui/src/app/(dashboard)/admin/page.tsx` & `ui/src/components/shared/LeftBar.tsx` -- Updates admin portal navigation to include the "Jobs" route.
- `ui/src/app/(dashboard)/admin/jobs/page.test.tsx` -- Comprehensive React Testing Library test suite verifying rendering, RBAC gating, modal interactions, polling behavior, and error handling.

## Tasks & Acceptance

**Execution:**
- [x] `ui/src/components/admin/JobsCatalogCard.tsx` -- Create catalog component listing registered jobs with "Run" and "Dry Run" action triggers.
- [x] `ui/src/components/admin/JobRunModal.tsx` -- Create modal component for parameter selection (dry-run checkbox) and confirmation before dispatch.
- [x] `ui/src/components/admin/JobExecutionsTable.tsx` -- Create executions table with live status badges, duration formatting, error popovers, and Cloud Logging links.
- [x] `ui/src/app/(dashboard)/admin/jobs/page.tsx` -- Implement the `/admin/jobs` dashboard page with RBAC gating, data fetching, active run polling, and toast notifications.
- [x] `ui/src/components/shared/LeftBar.tsx` & navigation -- Add Jobs link to admin navigation.
- [x] `ui/src/app/(dashboard)/admin/jobs/page.test.tsx` -- Author unit test suite covering full user journeys and edge cases.

**Acceptance Criteria:**
- Given a Super Administrator accessing `/admin/jobs`, when the page loads, then it renders the Registered Jobs Catalog and Historical Executions Table with live data from `GET /api/v1/admin/jobs` and `GET /api/v1/admin/jobs/executions`.
- Given a non-super-admin user, when they attempt to view `/admin/jobs`, then the UI renders an Access Denied state.
- Given the Job Run Modal, when a Super Admin toggles "Dry Run" and confirms execution, then the frontend dispatches `POST /api/v1/admin/jobs/{jobName}/run` with `{ "dryRun": true }` and displays an active status badge.
- Given an active in-flight execution (`RUNNING`), when displayed in the table, then the page automatically polls the executions endpoint every 5 seconds until the status updates to `SUCCEEDED` or `FAILED`.
- Given a completed execution with a `logUri`, when the user clicks "View Logs", then the browser opens the Google Cloud Logging console for that execution in a new tab.
- Given the test suite in `ui/src/app/(dashboard)/admin/jobs/page.test.tsx`, when run via `npm test`, then all tests pass.

## Design Notes

- Uses standard APF Admin UI design patterns with Tailwind CSS, Lucide icons (`Play`, `Terminal`, `CheckCircle2`, `AlertTriangle`, `Clock`, `ExternalLink`, `RotateCw`), and accessible dialog components.
- Directly leverages the zero-DB backend APIs created in Story 12.13 (`GET /api/v1/admin/jobs` and `GET /api/v1/admin/jobs/executions`).

## Suggested Review Order

**Dashboard Entry Point & State Orchestration**

- Gated admin screen with KPI metrics, auto-polling lifecycle, and data fetching
  [`page.tsx:28`](../../../ui/src/app/(dashboard)/admin/jobs/page.tsx#L28)

**Catalog & On-Demand Dispatch Modal**

- Interactive modal with dry-run toggle and safety confirmation
  [`JobRunModal.tsx:43`](../../../ui/src/components/admin/JobRunModal.tsx#L43)

- Catalog grid component rendering registered batch tasks
  [`JobsCatalogCard.tsx:26`](../../../ui/src/components/admin/JobsCatalogCard.tsx#L26)

**Live Monitoring & Execution History Table**

- Real-time execution table with status badges, search/filters, duration, and Cloud Logging links
  [`JobExecutionsTable.tsx:44`](../../../ui/src/components/admin/JobExecutionsTable.tsx#L44)

**Navigation & Peripherals**

- LeftBar navigation tab integration under admin section
  [`LeftBar.tsx:103`](../../../ui/src/components/shared/LeftBar.tsx#L103)

- Comprehensive React Testing Library test suite covering RBAC, triggers, search, and polling
  [`page.test.tsx:50`](../../../ui/src/app/(dashboard)/admin/jobs/page.test.tsx#L50)

