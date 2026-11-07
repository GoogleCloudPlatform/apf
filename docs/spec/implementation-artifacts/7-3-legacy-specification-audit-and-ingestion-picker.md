# Story 7.3: Legacy Specification Audit & Ingestion Picker

Status: ready-for-dev

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a PM,
I want an interactive directory tree picker displaying discovered legacy markdown specs,
so that I can check and select which requirements to ingest into active BMad agent contexts.

## Acceptance Criteria

1. **Given** the automated repository scan finishes successfully, **When** displaying the workspace's Onboarding Tab, **Then** the system MUST render the **Automated Repository Analysis Report** displaying the classified runtime stack (languages/frameworks) with modern styled chips. (AC: 7.3.1)
2. **Given** the onboarding workspace has discovered legacy specifications (`.md` and `.html` files), **When** displaying the Ingestion view, **Then** the UI MUST render an **Interactive Directory Tree Picker** that parses a flat list of paths into a nested tree structure of folders and files. (AC: 7.3.2)
3. **Given** the Ingestion Tree Picker, **When** rendering files, **Then**:
   - Each node MUST display with appropriate icons (Folder/File icons).
   - Each file MUST have a checkbox for active BMad agent context ingestion.
   - Parent folders MUST support recursive multi-select: checking a folder selects all nested children, and unchecking it deselects all children. (AC: 7.3.3)
4. **Given** discovered legacy files in the repository scan, **When** a file matches pre-existing BMad spec naming conventions (e.g. `prd.md`, `architecture.md`, `ux-design-specification.md`), **Then** the UI MUST:
   - Flag the file with a prominent emerald badge labeled `Existing BMad Spec`.
   - Automatically pre-select the file checkbox (set to Checked). (AC: 7.3.4)
5. **Given** a pre-existing BMad spec file is selected, **When** rendering the file item row in the tree picker, **Then** the UI MUST render a policy toggle or segment selector offering two explicit choices:
   - `Keep Existing Spec (Read-Only)` (value: `"keep"`)
   - `Re-Reconcile & Overwrite` (value: `"overwrite"`)
   - The policy MUST default to `Keep Existing Spec (Read-Only)` upon initial scan load. Non-BMad legacy specifications MUST NOT display policy toggles. (AC: 7.3.5)
6. **Given** the user selects files and sets write policies, **When** they trigger a save (e.g., clicking "Save Selections" or proceeding to the final stage), **Then** the UI MUST:
   - Make a PUT request to `/api/v1/projects/{id}/onboard/ingestion` to save the configurations as a draft state.
   - Display a success Snackbar/Toast notification indicating: `"Ingestion selections saved successfully!"`.
   - On workspace reloading, retrieve the draft selections via a GET request to `/api/v1/projects/{id}/onboard/ingestion` and correctly re-hydrate the tree checkboxes and policy states. (AC: 7.3.6)

## Tasks / Subtasks

- [ ] **Task 1: Backend Data Models & Store Interfaces for Ingestion (AC: 7.3.6)**
  - [ ] Create a new file [api/db/onboarding.go](api/db/onboarding.go) to define database models:
    - `IngestionFileSelection` struct containing `Path` (string), `Ingest` (bool), and `Policy` (string - legal values: `"keep"` | `"overwrite"`).
    - `OnboardingIngestion` struct containing `ProjectID` (string), `Files` ([]IngestionFileSelection), and `LastUpdated` (time.Time).
  - [ ] In [api/db/onboarding.go](api/db/onboarding.go), define the `OnboardingStore` interface:
    ```go
    type OnboardingStore interface {
        GetIngestionMapping(ctx context.Context, projectId string) (*OnboardingIngestion, error)
        SaveIngestionMapping(ctx context.Context, mapping *OnboardingIngestion) error
    }
    ```
  - [ ] Implement the `FirestoreOnboardingStore` struct that implements `OnboardingStore` using standard Firestore queries under the path: `projects/{projectId}/onboarding/ingestion` (saving the configuration as a single document structure for simple retrieval).
  - [ ] Instantiate the new store in the backend engine server setup (e.g., in [api/projects.go](api/projects.go) or main endpoint context) to make it available for dependency injection.

- [ ] **Task 2: Backend Ingestion Mapping REST HTTP API Endpoints (AC: 7.3.6)**
  - [ ] In the onboarding controller [api/onboarding.go](api/onboarding.go) (created in Story 7.2), implement the route handlers:
    - `getIngestionMappingHandler(w http.ResponseWriter, r *http.Request)`:
      - Authenticate the caller and parse the OIDC principal.
      - Fetch and return the `OnboardingIngestion` draft record from Firestore for the given project. Return empty shell payload if no draft exists yet.
    - `saveIngestionMappingHandler(w http.ResponseWriter, r *http.Request)`:
      - Authenticate the user and decode the `OnboardingIngestion` payload.
      - Validate payload fields (e.g., paths must match files listed in the discovery scan report, and `Policy` must be empty, `"keep"`, or `"overwrite"`).
      - Persist the selection mapping to Firestore. Return `200 OK` with the saved payload.
  - [ ] Register these new routes in [api/routes.go](api/routes.go):
    - `GET /api/v1/projects/{id}/onboard/ingestion` &rarr; `getIngestionMappingHandler`
    - `PUT /api/v1/projects/{id}/onboard/ingestion` &rarr; `saveIngestionMappingHandler`

- [ ] **Task 3: Interactive Ingestion Tree Picker Frontend Component (AC: 7.3.2, 7.3.3, 7.3.4, 7.3.5)**
  - [ ] Create a new file [ui/src/app/(dashboard)/projects/[id]/workspace/_components/IngestionTreePicker.tsx](ui/src/app/(dashboard)/projects/[id]/workspace/_components/IngestionTreePicker.tsx):
    - Define typescript structures for node levels (`FileNode` vs `FolderNode` in a standard hierarchical layout).
    - Write a utility function that parses flat string paths (e.g., `["docs/prd.md", "docs/guides/setup.md", "README.md"]`) into a nested `FolderNode` root hierarchy.
    - Implement standard state management using a flat key-value state for checking: `selectedPaths: Record<string, boolean>` and policies: `writePolicies: Record<string, "keep" | "overwrite">`.
    - Render collapsible sub-folders recursively, matching Tailwind styling guidelines (using modern Lucide icons: `Folder`, `FolderOpen`, `FileText`, `Settings`).
    - Bind checkbox interaction handlers supporting recursive folder triggers: when parent is clicked, select all descendants, and toggle indeterminate visual states if only some elements are checked.
    - Render a prominent green badge (`"Existing BMad Spec"`) next to files matching BMad structures (e.g., `prd.md`, `architecture.md`, `ux-design-specification.md`).
    - For badged specifications, render a segment/toggle selector supporting `Keep Existing Spec (Read-Only)` vs `Re-Reconcile & Overwrite`, defaulting to `"keep"`.

- [ ] **Task 4: Onboarding Panel Ingestion State & Report Screen Integration (AC: 7.3.1, 7.3.6)**
  - [ ] Update [OnboardingPanel.tsx](ui/src/app/(dashboard)/projects/[id]/workspace/_components/OnboardingPanel.tsx):
    - Retrieve scan results from `/api/v1/projects/${projectId}/onboard/report` when a scan completes successfully or is loaded in dynamic restoration mode.
    - Introduce a state machine transition showing the "Ingestion Configuration" screen once scan data is loaded.
    - In the Ingestion Configuration view:
      - Render the **Automated Repository Analysis Report** detailing classified stack technologies (e.g., Go, Node, Next.js, Firestore) using color-coded badge components.
      - Mount the newly created `<IngestionTreePicker />` forwarding the list of detected legacy documents and current draft selections.
      - Fetch existing draft selections on load via `/api/v1/projects/${projectId}/onboard/ingestion` and initialize the tree picker state with those values.
      - Add a `"Save Draft"` action button (which writes current checkboxes/policies to the PUT endpoint) and a primary step CTA button `"Continue to Verification"` which saves draft state and proceeds to the final gating stage (Story 7.4).

- [ ] **Task 5: Verification & Automated Test Suites (AC: 7.3.1 - 7.3.6)**
  - [ ] Create Go unit/integration test cases in `api/onboarding_test.go` and `api/db/onboarding_test.go`:
    - Verify that `GET /api/v1/projects/{id}/onboard/ingestion` correctly returns draft models or empty shell structures when Firestore is uninitialized.
    - Verify that `PUT /api/v1/projects/{id}/onboard/ingestion` correctly validates file boundaries and write policies, rejecting illegal path references, and persists data securely under OIDC organization contexts.
  - [ ] Add Next.js / Vitest test files for `IngestionTreePicker.tsx`:
    - Test that parsing a flat path array correctly builds hierarchical tree nodes.
    - Test that recursive checkboxes select and deselect all sub-folder descendants properly.
    - Test that known specifications (e.g., `docs/spec/planning-artifacts/prd.md`) are pre-selected with the default `"keep"` write policy.
  - [ ] Test the integrated flow in `OnboardingPanel.tsx` asserting that save calls fire correct request payloads and show snackbar alerts.

## Dev Notes

### 🏗️ 1. Architectural Guardrails

*   **Tree Struct Complexity & Memoization:** Building a recursive DOM tree structure can trigger heavy rendering delays on large folders. Use React `useMemo` to cache the hierarchical tree construction from flat list data and memoize list items to keep DOM updates responsive.
*   **Validation of Filesystem Bounds:** In the backend `saveIngestionMappingHandler`, do not blindly accept arbitrary paths. Ensure all paths submitted:
    1. Are safe relative paths (no directory traversal `../` escapes outside the repository tree).
    2. Are registered in the project's background discovery report as existing specifications, preventing the user from injecting unauthorized random paths.
*   **Linear Lifecycle Consistency:** Gating logic must remain consistent. Standard workspace modules remain locked during this step. Selections must be saved as draft state, allowing multi-session persistence until the final onboarding gate is confirmed.

### 🛠️ Project Structure & Target Files Matrix

| File Path | Status | Operational Role |
| :--- | :--- | :--- |
| [api/db/onboarding.go](api/db/onboarding.go) | `NEW` | Database models for `OnboardingIngestion` and the `OnboardingStore` interface/Firestore implementation. |
| [api/onboarding.go](api/onboarding.go) | `UPDATE` | Add `getIngestionMappingHandler` and `saveIngestionMappingHandler` API routes. |
| [api/routes.go](api/routes.go) | `UPDATE` | Register GET and PUT endpoints for the onboarding ingestion selections. |
| [ui/src/app/(dashboard)/projects/[id]/workspace/_components/IngestionTreePicker.tsx](ui/src/app/(dashboard)/projects/[id]/workspace/_components/IngestionTreePicker.tsx) | `NEW` | Interactive collapsible directory tree picker React component with checkboxes, badges, and write-policy toggles. |
| [ui/src/app/(dashboard)/projects/[id]/workspace/_components/OnboardingPanel.tsx](ui/src/app/(dashboard)/projects/[id]/workspace/_components/OnboardingPanel.tsx) | `UPDATE` | Integrate the stack report presentation, draft retrieval/loading, mounting `IngestionTreePicker`, and save configurations. |

### References

- [PRD - Project Onboarding & Discovery Journeys](docs/spec/planning-artifacts/prd.md#L101-L106)
- [PRD - FR-7.4 Ingestion Tree Gating & Pre-existing Specifications](docs/spec/planning-artifacts/prd.md#L154-L155)
- [Epics - Story 7.3: Legacy Specification Audit & Ingestion Picker Context](docs/spec/planning-artifacts/epics.md#L426-L440)
- [Previous Story 7.2 Spec & Context](docs/spec/implementation-artifacts/7-2-background-discovery-agent-stack-analysis.md)

## Dev Agent Record

### Agent Model Used

Gemini Next

### Debug Log References

### Completion Notes List

### File List
