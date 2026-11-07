# Story 9.6: Seeded Showcase Project (Foundry Showcase)

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a User,
I want to be able to provision a pre-configured showcase project named "Foundry Showcase" on-demand from the Settings screen,
and be able to reset it to its initial state at any time,
so that I can explore features and reference a real-world design setup scoped to my active organization.

## Acceptance Criteria

1. **Given** the user is on the Settings screen
   **When** they view the left-hand navigation
   **Then** the UI MUST display a new menu item labeled **"Project Templates"**.
2. **Given** the user selects the "Project Templates" menu item
   **When** the panel loads
   **Then** the right-hand panel MUST display a section for the showcase project with a primary action button labeled **"Provision Foundry Showcase"**.
3. **Given** the user clicks "Provision Foundry Showcase"
   **When** the request is sent to the backend API (`POST /api/projects/provision-showcase`)
   **Then** the UI MUST display a loading spinner and disable the button to prevent double submissions.
4. **Given** the backend receives the provisioning request
   **When** the seeder executes
   **Then** it MUST read pre-configured project data and specs from an embedded directory under **`api/seeding/templates/showcase`** (using Go `//go:embed`) deployed with the API tier
   **And** it MUST seed the project, sessions, messages, and artifacts records into Firestore, dynamically scoping them to the user's active `organizationId` and generating a unique `projectId`
   **And** it MUST extract and write the workspace spec files (e.g., `prd.md`, `architecture.md`, `ux-design-specification.md`) onto the project's filesystem path (e.g., `/projects/{showcaseProjectId}/docs/spec/`).
5. **Given** the backend returns a successful response
   **When** the client receives the success status
   **Then** the UI MUST show a success Snackbar notification (e.g., *"Showcase project provisioned successfully!"*), close the Settings modal, and refresh the Projects Dashboard to show the new "Foundry Showcase" project tile.
6. **Given** the "Foundry Showcase" project tile is rendered on the Projects Dashboard
   **When** the user views it
   **Then** it MUST display correct metadata (status: `Onboarded`, tag: `Sample Project`).
7. **Given** the user opens the "Foundry Showcase" workspace
   **When** they interact with the workspace
   **Then** clicking on the tile MUST launch the GUI Workspace immediately, loading the seeded interactive canvas, sidebar chat history, and specs
   **And** the user MUST be able to fully edit, run agents on, and synchronize the project as a standard active, write-enabled workspace without any read-only constraints.
8. **And** the provisioned showcase project MUST NOT be linked to any remote Git repository by default (all integration and VCS settings initialized as empty).
9. **Given** the showcase project has already been provisioned
   **When** the user views the "Project Templates" tab in the Settings screen
   **Then** the UI MUST display a link or secondary action labeled **"Restore to default state"**.
10. **Given** the user clicks the "Restore to default state" link
    **When** the request is sent to the backend API
    **Then** the backend MUST delete all existing sessions, messages, and artifacts for the showcase project in Firestore, overwrite the project record with the default seed template, delete all modified files in the project workspace filesystem, and re-extract the original seed spec files, returning the workspace to its initial seeded state.

## Tasks / Subtasks

- [x] Task 1: Backend Seeding Engine & API Reset Implementation (Go API Tier) (AC: 4, 8, 10)
  - [x] Create seed data directory **`api/seeding/templates/showcase/`** containing Firestore JSON templates for the showcase project, sessions, messages, and artifacts.
  - [x] Ensure that the seeded `project.json` or seeder code initializes the project's integrations, remote repository URLs, and Git synchronization configurations as empty/unlinked.
  - [x] Add sample specs/artifacts in the seed directory (`api/seeding/templates/showcase/workspace/prd.md`, `api/seeding/templates/showcase/workspace/architecture.md`, `api/seeding/templates/showcase/workspace/ux-design-specification.md`) representing a completed real-world project.
  - [x] Implement `api/seeding/seeder.go` (or in a new `seeding` package) utilizing the standard Go `embed` package (`//go:embed`) to bundle the showcase project seed data under **`api/seeding/templates/showcase`** into the Go binary.
  - [x] Implement a seed executor function that:
    *   Generates a new unique `projectId` for the showcase.
    *   Replaces placeholder values in the JSON templates with the caller's active `organizationId` and the generated `projectId`.
    *   Writes all records (project, sessions, messages, artifacts) to Firestore.
    *   Extracts the embedded spec files and writes them to the project's workspace directory on the filesystem.
  - [x] Implement a reset function that accepts a `projectId`, deletes all associated sub-collections (sessions, messages, artifacts) in Firestore, overwrites the core project document, clears the project workspace filesystem directory, and re-extracts the embedded files.
  - [x] Register a new endpoint `POST /api/projects/provision-showcase` in `api/routes.go` protected by `authMiddleware` and `orgScopeMiddleware`. Support an optional query parameter `?reset=true` (or a separate `POST /api/projects/{projectId}/reset` endpoint) to trigger the reset flow.
- [x] Task 2: Settings Screen UI & Templates Tab with Reset Link (Next.js UI) (AC: 1, 2, 3, 5, 9, 10)
  - [x] Update the Settings screen layout (`ui/src/components/settings/`) to add a "Project Templates" navigation item to the left pane.
  - [x] Build the right-hand panel for "Project Templates" with a description of the showcase project and a primary "Provision Foundry Showcase" action button.
  - [x] Detect if the showcase project has already been provisioned (e.g. by scanning the user's project list for the showcase tag/metadata).
  - [x] If provisioned, render a secondary link labeled **"Restore to default state"** next to or below the template card.
  - [x] Implement the API call to trigger the reset, showing a loading overlay/spinner and disabling the link to prevent concurrent requests.
  - [x] Implement post-success actions for reset: display a success Snackbar (*"Showcase project restored to default state!"*), close the Settings modal, and refresh the active workspace view or Projects Dashboard.
- [x] Task 3: Projects Dashboard & Workspace Landing (Next.js UI) (AC: 6, 7)
  - [x] Update the Projects Dashboard page (`ui/src/`) to render a "Sample Project" or "Showcase" badge on the "Foundry Showcase" tile.
  - [x] Ensure the "Foundry Showcase" tile redirects to the workspace view normally, loading the seeded canvas state, chat history, and specs successfully.

### Review Findings

- [x] [Review][Patch] Arbitrary Project Deletion and Path Traversal Vulnerability [api/projects_showcase_handler.go]
- [x] [Review][Patch] Provisioning Idempotency Race Condition (Double-Clicking) [api/projects_showcase_handler.go]
- [x] [Review][Patch] Inconsistent Showcase Identification between Frontend and Backend [api/projects_showcase_handler.go]
- [x] [Review][Patch] Orphaned Global Session Messages in Firestore on Deletion/Reset [api/db/project.go]
- [x] [Review][Patch] Lack of Seeding Atomicity (Broken State on Failure) [api/showcase_seeder.go]
- [x] [Review][Patch] Silent Ignorance of Database Errors in Duplicate Check [api/projects_showcase_handler.go]
- [x] [Review][Patch] Raw Error Information Disclosure in API Responses [api/projects_showcase_handler.go]
- [x] [Review][Patch] Duplicated Workspace Path Resolution Logic [api/showcase_seeder.go]
- [x] [Review][Patch] Hard Page Reload Instead of Client-Side Router Navigation [ui/src/app/(dashboard)/settings/page.tsx]
- [x] [Review][Patch] Incomplete UI Button Disabling and Query State Handling [ui/src/app/(dashboard)/settings/page.tsx]
- [x] [Review][Defer] Coupled Storage Dependencies (Package-Level Global Variables) [api/showcase_seeder.go] — deferred, pre-existing

## Dev Notes

*   **Go Embed Package**: Ensure `embed.FS` is used correctly to bundle the files under **`api/seeding/templates/showcase`**. The workspace extraction should write directly to the active filesystem mount, aligning with the project's standard workspace file-writing conventions.
*   **Security & Scoping**: The seeder must strictly resolve the `organizationId` from the request context claims (`core.GetOrganizationID(r.Context())`) to prevent cross-tenant leakage or parameter manipulation.
*   **Idempotent Seeding and Resetting**: The reset delete-and-re-seed operation should run atomically (using a Firestore transaction or batch) where possible to prevent leaving the database in a half-deleted or corrupt state if the request fails mid-way.
*   **No Read-Only Gating Needed**: Since the project is provisioned within the user's active tenant, they have full edit permissions. This simplifies the story as we do not need to intercept mutating API requests or disable UI inputs.
*   **Unlinked Git Repository**: Since the project is not linked to any remote repository by default, standard Git synchronization or Pull Request features will be inactive/unconfigured until the user manually links a repository in the project's settings screen.
*   **Tailwind/Shadcn**: Use standard Tailwind CSS classes and Shadcn components for the new Settings menu item, provision panel, and reset link.

### Project Structure Notes

*   Align with poly-directory structure:
    *   Backend logic: `api/db/project.go`, **`api/seeding/`**
    *   Frontend logic: `ui/src/components/settings/`
*   Maintain dependency injection standard for all HTTP and database handlers.

### References

*   [Source: docs/spec/planning-artifacts/epics.md#Story 9.6: Seeded Showcase Project (Foundry Showcase) (Frontend/Backend)]
*   [Source: docs/spec/planning-artifacts/architecture.md#Data & Resource Architecture]

## Dev Agent Record

### Agent Model Used

Gemini Next

### Debug Log References

### Completion Notes List

- Implemented on-demand showcase project provisioning and resetting.
- Created embedded templates (`api/seeding/templates/showcase/`) for project, sessions, messages, and artifacts.
- Added Go backend seeder (`api/showcase_seeder.go`) and handlers (`api/projects_showcase_handler.go`) with route registration.
- Added "Project Templates" tab to Settings screen in UI (`ui/src/app/(dashboard)/settings/page.tsx`).
- Implemented "Provision Foundry Showcase" and "Restore to default state" actions in UI with loading states.
- Added "Sample Project" badge to ProjectCard and ProjectRow components.

### File List

- `api/seeding/templates/showcase/project.json`
- `api/seeding/templates/showcase/sessions.json`
- `api/seeding/templates/showcase/messages.json`
- `api/seeding/templates/showcase/artifacts.json`
- `api/seeding/templates/showcase/workspace/prd.md`
- `api/seeding/templates/showcase/workspace/architecture.md`
- `api/seeding/templates/showcase/workspace/ux-design-specification.md`
- `api/showcase_seeder.go`
- `api/projects_showcase_handler.go`
- `api/routes.go`
- `ui/src/app/(dashboard)/settings/page.tsx`
- `ui/src/components/project/ProjectCard.tsx`
- `ui/src/components/project/ProjectRow.tsx`
