# Epic & Story Breakdown
## Project: Foundry Showcase (On-Demand Provisioning)

**Status**: Final  
**Version**: 1.0  
**Authors**: Product Management & System Architecture  

---

## 1. Requirements Inventory

### 1.1 Functional Requirements (FR)
- **FR-9.8.1**: The system SHALL provide a backend endpoint `POST /api/projects/provision-showcase` to seed a pre-configured showcase project deterministically.
- **FR-9.8.2**: The backend seeder SHALL utilize Go `embed.FS` to load project metadata, session history, chat messages, and workspace files without disk/network overhead.
- **FR-9.8.3**: The backend seeder SHALL copy the workspace Markdown files onto the project's local directory path under `docs/spec/`.
- **FR-9.8.4**: The system SHALL provide a backend endpoint `POST /api/projects/:id/reset` to wipe a showcase project and re-seed it from templates.
- **FR-9.8.5**: The reset handler MUST restrict operations strictly to project IDs prefixed with `showcase-` and reject any path traversal strings (`..`, `/`, `\`) for security.
- **FR-9.8.6**: The frontend SHALL display a "Project Templates" tab in Settings containing the showcase card with "Provision" or "Restore" buttons depending on state.
- **FR-9.8.7**: The frontend Dashboard SHALL render the provisioned showcase tile with a "Sample Project" badge.

---

## 2. Epic Breakdown

### Epic 9: Onboarding & Tours
This epic focuses on accelerating the user onboarding flow, guiding new users through the platform features, and seeding rich reference materials.

#### Story 9.6: Seeded Showcase Project (Frontend & Backend Implementation)
**Complexity**: Medium  
**Status**: Ready  
**Description**:  
As a new user, I want to provision a pre-configured showcase project on-demand from the Settings modal so that I can immediately explore the platform's capabilities with a fully-populated, realistic workspace.

**Acceptance Criteria**:
1. **Given** the user is on the Settings dialog, **When** they click "Project Templates" and click "Provision Foundry Showcase", **Then** the system MUST display a full-screen loading spinner.
2. **Given** the backend API receives `POST /api/projects/provision-showcase`, **When** it seeds the project, **Then** it MUST write Firestore collections (project, sessions, messages, artifacts) and write the 5 markdown files into the project workspace directory on disk.
3. **Given** the seeding succeeds, **When** the frontend closes the modal, **Then** it MUST display a success snackbar: *"Showcase project provisioned successfully!"* and display the "Foundry Showcase" tile on the Dashboard with a "Sample Project" badge.
4. **Given** the user modifies the showcase workspace, **When** they click "Restore to default state" in Settings, **Then** the backend MUST delete all Firestore sessions/messages/artifacts for the showcase, purge the workspace folder, re-seed from templates, and display a success snackbar: *"Showcase project restored to default state!"*

---

## 3. Verification & Acceptance Gate
- **Gating Policy**: The Solutioning phase is satisfied when `project-context.md`, `prd.md`, `architecture.md`, `ux-design-specification.md`, and `epics.md` are marked as **Complete**.
- **Implementation Readiness**: With all 5 design artifacts complete and Epic 9/Story 9.6 fully spec'd, the project is ready to transition to the Implementation phase.
