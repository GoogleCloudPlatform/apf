# Project Context: Foundry Showcase

## 1. Project Overview
Foundry Showcase is a core onboarding feature of the **Agentic Product Foundry (APF)**. It enables users to provision a fully-configured, high-fidelity sample project workspace on-demand. This allows them to experience the platform's collaboration, design, and planning features immediately without manual setup.

## 2. Technical Stack
- **Backend Services**: Go 1.22+ (stateless, deployed on Cloud Run)
- **Frontend Presentation**: Next.js Client & Next.js Dashboard
- **Database Storage**: Google Cloud Firestore (NoSQL, hard-multitenancy)
- **Workspace Hosting**: Local mounted directory (`~/workspaces`) sync'd to local file systems

## 3. Directory Layout
- `api/` - Go-based REST API handlers, seeding mechanisms, and route definitions.
- `ui/` - Next.js-based frontend screens, modals, and canvas components.
- `docs/spec/` - Specification, planning, and design documents.
  - `project-context.md` - (This document) Project identity and guidelines.
  - `prd.md` - Product requirements and user journeys.
  - `architecture.md` - System diagrams and database schemas.
  - `ux-design-specification.md` - UI layouts, modals, and screen navigation rules.
  - `epics.md` - Epic and story breakdown.

## 4. Engineering Guidelines & Rules
1. **Multitenancy**: Every database write or read MUST resolve and enforce `organizationId` from the request context to prevent tenant bleed.
2. **Idempotency**: Seeding must be strictly idempotent. A unique deterministic project ID (`showcase-{orgID}-{userID}`) must be used to prevent race conditions.
3. **Robust Cleanup**: In case of seeding failures, the seeder must execute a defer function to purge Firestore records and wipe the local workspace directory cleanly, preventing orphaned resources.
