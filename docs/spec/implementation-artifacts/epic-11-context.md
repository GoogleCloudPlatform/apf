# Epic 11 Context: Enterprise Identity, Governance, Security, & System Administration Portal

<!-- Compiled from planning artifacts. Edit freely. Regenerate with compile-epic-context if planning docs change. -->

## Goal

Provide enterprise-grade identity, security, organization governance, and administrative oversight capabilities for Agentic Foundry. This includes multi-tenant tenant boundary isolation, IAP authentication middleware, organization provisioning, user management, RBAC enforcement, and centralized administrative telemetry analytics dashboards.

## Stories

- Story 11.1: Identity Platform Multi-Tenant Provisioning & Email/Password IDP Setup
- Story 11.2: Identity-Aware Proxy (IAP) Token Validation Middleware
- Story 11.3: System Administration - Organization Provisioning Screen (System Admin Portal)
- Story 11.4: Organization Settings - Member Directory & User Management Screen (Org Admin Portal)
- Story 11.5: Security Context & Tenant Boundaries Configuration
- Story 11.6: Organization RBAC Policy Engine & Dynamic Role Scoping
- Story 11.7: Workspace Access Control & User Assignment Matrix
- Story 11.8: Super Admin Tenant-Bypass Claims & System Governance Mode
- Story 11.9: Multi-Tenant Identity Verification & Fail-Safe Security Context Propagation
- Story 11.10: Application Admin Performance & Operational Telemetry Dashboard UI
- Story 11.11: Admin Portal AI Token Usage & Cost Analytics Dashboard Screen

## Requirements & Constraints

- **RBAC Enforcement:** System Admins have platform-wide administrative privileges; Organization Admins are strictly scoped to their tenant (`organizationId`). Standard users/viewers have no access to administrative dashboards or management screens.
- **Tenant Isolation:** All admin screens must scope queries and data modifications to the authenticated user's organization scope (`claims.OrganizationID`), unless the user is a Super Admin (`claims.IsSuperAdmin === true`).
- **Telemetry Display Boundary:** AI Token usage and cost metrics MUST be visible strictly inside the Admin Portal AI Usage & Cost Analytics Dashboard (`/admin`), and MUST NOT be displayed in individual workspace chat bubbles or sidebar chat panes.

## Technical Decisions

- **Admin UI Navigation:** The Admin Portal uses React Router under `/admin` with tab-based navigation (e.g. Overview, Members, Settings, Performance, and AI Usage & Cost Analytics).
- **Backend API Scoping:** Endpoints rely on `authMiddleware` extracting `core.UserClaims` from Go `context`. Multi-tenant requests are validated using `claims.OrganizationID` and `claims.Roles`.
- **Telemetry Query Source:** AI Usage statistics query `GET /api/v1/telemetry/usage` directly, which aggregates token metrics from the ADK PostgreSQL Session Database (`adk_sessions`, `adk_events`).

## UX & Interaction Patterns

- **Admin Analytics Screen:** Tab view inside `/admin` rendering top-level KPI cards (Total Tokens, Estimated Cost USD, Active Turns, Total Users), interactive filters (Organization, Project, Date Window: Today, 7D, 30D, Custom), breakdown tables (Per User, Per Project, Per Session), and Export CSV controls.
- **Clean Chat Pane Guarantee:** Workspace chat interfaces must remain clutter-free with zero token badges or cost metrics displayed to end users.
