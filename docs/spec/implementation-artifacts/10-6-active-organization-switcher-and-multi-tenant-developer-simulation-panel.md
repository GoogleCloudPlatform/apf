# Story 10.6: Active Organization Switcher & Multi-Tenant Developer Simulation Panel

Status: ready-for-dev

## Story

As a User,
I want to be able to switch my active organization directly from the UI header and have developer tools to simulate tenant context switching,
So that I can quickly transition between authorized workspaces and developers can test isolation boundaries effectively.

## Acceptance Criteria

1. **Given** an authenticated user is in the workspace
   **When** the user selects a different organization from the Workspace Selector dropdown in the navigation header
   **Then** the UI MUST update the active organization ID in the store
   **And** clear the local state/cache of the active workspace (projects, chat history, audit logs)
   **And** fetch and load only the data matching the new organization scope

2. **Given** a local development environment is running with mock auth enabled
   **When** a developer interacts with the Simulation Panel in the footer
   **Then** the developer MUST be able to select and simulate different pre-configured mock organization contexts
   **And** the UI client MUST update the authorization token and `X-Organization-Id` header sent in all subsequent API requests

## Tasks / Subtasks

- [ ] Update state store (`ui/src/stores/auth.ts`) to manage accessible organizations and active organization ID (AC: 1, 2)
  - [ ] Add `activeOrganizationId` and `myOrganizations` state fields
  - [ ] Add `selectOrganization(orgId: string)` action updating state and triggering workspace/project state invalidation
- [ ] Update API client (`ui/src/lib/api.ts`) to inject active organization header (AC: 1, 2)
  - [ ] Intercept requests and append `X-Organization-Id` if `activeOrganizationId` is set in store
- [ ] Create Layout Navigation Selector (`ui/src/components/navigation/WorkspaceSelector.tsx`) (AC: 1)
  - [ ] Implement UI dropdown component in the navigation header listing `myOrganizations`
  - [ ] Bind dropdown selection to `selectOrganization` action
- [ ] Implement Mock Authentication Updates in the Backend (`api/auth.go`) (AC: 2)
  - [ ] Add or expand endpoints/token-helpers to allow simulated login tokens mapping to different organizations (e.g. `simulated-jwt-token-for-user1-org-orgA`)
- [ ] Create Local Development Simulation Panel in Footer (AC: 2)
  - [ ] Implement a footer simulation overlay visible only in dev mode
  - [ ] Allow switching mock tokens/orgs on the fly and updating state store
- [ ] Verify Tenant Bleed Isolation (AC: 1, 2)
  - [ ] Ensure switching organizations clears old project/session data from current view and does not allow cached leaks

## Dev Notes

Refer to the multi-tenant login UI implementation plan for details on store layout, header configuration, and test scenarios.

### References
- Multi-Tenant Login UI Plan: [multi_tenant_login_ui_plan.md](multi_tenant_login_ui_plan.md)
- Epics breakdown: [docs/spec/planning-artifacts/epics.md#Story-10.6](docs/spec/planning-artifacts/epics.md)
