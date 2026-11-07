---
title: 'Story 11.11: Admin Portal AI Token Usage & Cost Analytics Dashboard Screen'
type: 'feature'
created: '2026-08-10'
status: 'done'
baseline_commit: '575c9b7801571acdf1141a83ddf22e95e7a678da'
review_loop_iteration: 0
context: []
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** System Administrators and Organization Admins lack visual operational insights into AI token consumption, model distribution, turn counts, and cost metrics aggregated across users, projects, organizations, and sessions. Furthermore, token indicators must remain centralized in the Admin Portal and absent from user-facing workspace chat interfaces.

**Approach:** Build a dedicated Next.js Admin Portal dashboard page (`ui/src/app/(dashboard)/admin/ai-usage/page.tsx`) and navigation tab (`LeftBar.tsx`) that queries the backend telemetry API (`GET /api/v1/telemetry/usage`). The screen presents top-level KPI metrics cards (Total Tokens, Cost USD, Active Turns, Total Users/Projects), interactive date range selectors (Today, 7D, 30D default, Custom), multi-tenant filters (Organization, Project, User, Session), a tabbed breakdown section ("By User", "By Project", "By Session") with paginated tables, a header "Export CSV" stream trigger (`?format=csv`), and ensures workspace chat panes remain clean.

## Boundaries & Constraints

**Always:**
- Access telemetry strictly via `GET /api/v1/telemetry/usage` with valid Bearer token authentication and organization context.
- Restrict screen access to System Admins (`isSuperAdmin`) and Org Admins (`hasRole("org:admin")`), rendering `UnauthorizedScreen` for unauthorized roles.
- Display cost USD values formatted as `$0.0000` (4 decimal places) alongside integer token counts (`promptTokens`, `completionTokens`, `totalTokens`).
- Provide date window controls (Today, Last 7 Days, Last 30 Days default, and Custom Date Range picker).
- Include export controls triggering direct CSV download (`GET /api/v1/telemetry/usage?format=csv`).
- Organize breakdown views into a Tabbed Layout ("By User", "By Project", "By Session") with top KPI metric cards.
- Guarantee that workspace chat bubbles, sidebar chat panes, and user-facing screens remain completely free of token usage or cost badges.

**Ask First:**
- Modifying global navigation routes outside `/admin`.

**Never:**
- Do not display token usage or cost indicators inside user workspace chat panes.
- Do not query Firestore counter collections directly from the frontend UI.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| Admin Telemetry Load | Navigating to `/admin/ai-usage` as Org Admin | Fetches telemetry metrics and renders top KPI cards and breakdown tables | Render error alert banner if fetch fails |
| Filter Date Window | Selecting "Last 7 Days" or custom date range | Triggers API request with `startDate` and `endDate` parameters and refreshes dashboard | Reset pagination offset to 0 |
| Export CSV Report | Clicking "Export CSV" button | Triggers backend download stream `telemetry-usage.csv` (`format=csv`) | Toast error message if CSV download fails |
| Cross-Tenant Protection | Non-admin user navigates to `/admin/ai-usage` | Displays `UnauthorizedScreen` access denied warning | Prevent API telemetry calls |
| Empty Data State | Organization has zero logged LLM events | Displays empty state placeholders ("0 Total Tokens", "$0.0000 Cost") | Render friendly empty state message |

</frozen-after-approval>

## Code Map

- `ui/src/components/shared/LeftBar.tsx` -- Add "AI Usage & Cost" menu item (ID 18, path `/admin/ai-usage`, icon `Cpu`) under admin navigation section.
- `ui/src/app/(dashboard)/admin/ai-usage/page.tsx` -- `NEW` Next.js React page component for AI Token Usage & Cost Analytics Dashboard, handling RBAC verification, KPI summary cards, filter bars (date window, project/user/session filters), tabbed User/Project/Session breakdown tables, and CSV export streaming.
- `ui/src/app/(dashboard)/admin/ai-usage/page.test.tsx` -- `NEW` React Testing Library unit tests verifying admin rendering, RBAC authorization boundaries, filter updates, tab switching, and CSV export trigger.

## Tasks & Acceptance

**Execution:**
- [x] `ui/src/components/shared/LeftBar.tsx` -- Add "AI Usage & Cost" tab under Admin navigation section -- Enable navigation to `/admin/ai-usage`.
- [x] `ui/src/app/(dashboard)/admin/ai-usage/page.tsx` -- Build `AIUsageAnalyticsPage` component fetching `/api/v1/telemetry/usage` with KPI summary cards, date range selectors, tabbed breakdown tables ("By User", "By Project", "By Session"), and CSV download button -- Present complete telemetry analytics to admins.
- [x] `ui/src/app/(dashboard)/admin/ai-usage/page.test.tsx` -- Write UI unit and integration tests verifying rendering, RBAC checks, filter changes, tab switching, and CSV export -- Ensure 100% test coverage for admin telemetry dashboard.

**Acceptance Criteria:**
- Given an authorized Administrator visiting `/admin/ai-usage`, when the page loads, then KPI summary cards, token rollups, cost USD ($0.0000), and breakdown tables render successfully.
- Given interactive filter changes (Date range, Project, User), when applied, then the page updates parameters and re-fetches telemetry metrics.
- Given the "Export CSV" button is clicked, then a CSV file containing the aggregated report is downloaded from backend API.
- Given a non-admin user visiting `/admin/ai-usage`, then `UnauthorizedScreen` is displayed and no backend calls are made.

## Design Notes

The page queries `/api/v1/telemetry/usage` with standard React `useEffect` and `useState` hooks, using `useAuthStore` for Bearer token and `useAppStore` for setting the header page title ("Admin: AI Usage & Cost").

## Verification

**Commands:**
- `npm test -- ui/src/app/\(dashboard\)/admin/ai-usage/page.test.tsx` -- expected: PASS all UI component unit tests.

## Suggested Review Order

**Sidebar Navigation & Permissions**

- Admin mode navigation menu item registration and active selection index mapping
  [`LeftBar.tsx:126`](../../../ui/src/components/shared/LeftBar.tsx#L126)

**Admin Telemetry Analytics Dashboard Component**

- AI Usage Analytics Page React component implementing RBAC checks, KPI summary cards, filter bars, tabbed breakdown tables, and CSV export stream trigger
  [`page.tsx:115`](../../../ui/src/app/(dashboard)/admin/ai-usage/page.tsx#L115)

**Verification & Tests**

- React Testing Library unit tests verifying rendering, RBAC authorization, filter updates, tab switching, null guards, and CSV download stream
  [`page.test.tsx:1`](../../../ui/src/app/(dashboard)/admin/ai-usage/page.test.tsx#L1)
