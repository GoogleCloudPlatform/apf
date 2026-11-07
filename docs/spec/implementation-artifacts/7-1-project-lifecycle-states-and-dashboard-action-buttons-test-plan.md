# Test Plan: Story 7.1 Project Lifecycle States & Dashboard Action Buttons

This document establishes the comprehensive test plan for validating the linear status flow gating, dashboard button states, and workspace phase navigation controls introduced in Story 7.1.

---

## 🎯 1. Goal & Scope

The goal of this test plan is to guarantee 100% operational reliability for the initial project lifecycle state transitions (`New` -> `Onboarding` -> `Onboarded`) and the associated navigation gates. Testing spans backend API unit tests, frontend Vitest component tests, and rigorous manual browser verification scripts.

### Acceptance Criteria (AC) Mapping
- **AC 7.1.1 (Initial Status)**: Verify that newly initialized projects in Firestore default to the `New` state.
- **AC 7.1.2 (Dynamic Dashboard Tile Button)**: Verify that projects in `New`/`Onboarding` states render an `"Onboard"` button, and `Onboarded` projects render `"Open Workspace"`.
- **AC 7.1.3 (New -> Onboarding State & Route Transition)**: Verify that clicking `"Onboard"` on a `New` project card updates its status to `"Onboarding"` in the database and redirects to `/projects/{id}/workspace?phase=ONBOARDING`.
- **AC 7.1.4 (Onboarding Session Resumption)**: Verify that clicking `"Onboard"` on an `Onboarding` project redirects directly to `/projects/{id}/workspace?phase=ONBOARDING` without redundant API writes.
- **AC 7.1.5 (Workspace Phase Gating & Locks)**: Verify that non-onboarded projects have all standard workspace tabs (Analysis, Planning, Solutioning, Implementation) locked, displaying the exclusive Onboarding Panel and hiding the Sidebar Chat and Canvas.

---

## 🧪 2. Automated Test Suites

### 2.1 Backend Go API Tests (`api/projects_test.go`)

The backend Go tests validate Firestore database operations and HTTP handler access controls. Execute these tests using:
```bash
go test -v -run "TestCreateProjectDefaultStatusNew|TestUpdateProjectStatusToOnboarding|TestUpdateProjectStatusForbidden"
```

| Test Case Name | Target AC | Objective / Assertions |
| :--- | :--- | :--- |
| `TestCreateProjectDefaultStatusNew` | AC 7.1.1 | **Given** a `POST /api/v1/projects` payload is submitted by an authorized user,<br>**When** the project record is created,<br>**Then** verify that the returned project status is explicitly set to `"New"`. |
| `TestUpdateProjectStatusToOnboarding` | AC 7.1.3 | **Given** a project in `"New"` status,<br>**When** a `PUT /api/v1/projects/{id}` payload with `{"status": "Onboarding"}` is sent by a project member,<br>**Then** verify that the HTTP response returns `200 OK` and the project's Firestore status field is successfully updated to `"Onboarding"`. |
| `TestUpdateProjectStatusForbidden` | AC 7.1.3 | **Given** a project in `"New"` status,<br>**When** a `PUT` status update payload is submitted by an unauthorized/non-member user,<br>**Then** verify that the HTTP response is blocked with `403 Forbidden` and the database state remains unchanged. |

---

### 2.2 Frontend Vitest Tests (`ui/src/components/project/ProjectCard.test.tsx`)

The frontend unit tests validate client-side routing actions and card button labeling. Execute these tests using:
```bash
npm run test -- ui/src/components/project/ProjectCard.test.tsx
```

| Test Case Description | Target AC | Mocking & Assertions |
| :--- | :--- | :--- |
| **renders Onboard label when project status is New** | AC 7.1.2 | **Given** a project with status `"New"`,<br>**When** the `ProjectCard` renders,<br>**Then** assert that the primary button label matches precisely `"Onboard"`. |
| **renders Onboard label when project status is Onboarding** | AC 7.1.2 | **Given** a project with status `"Onboarding"`,<br>**When** the `ProjectCard` renders,<br>**Then** assert that the primary button label matches precisely `"Onboard"`. |
| **renders Open Workspace label when project status is Onboarded** | AC 7.1.2 | **Given** a project with status `"Onboarded"`,<br>**When** the `ProjectCard` renders,<br>**Then** assert that the primary button label matches precisely `"Open Workspace"`. |
| **performs API transition to Onboarding and redirects on New click** | AC 7.1.3 | **Given** a project with status `"New"`,<br>**When** the `"Onboard"` button is clicked,<br>**Then** assert that `api.put` is fired with `/api/v1/projects/project-123` and payload `{ status: "Onboarding" }`, and `router.push` is called with `/projects/project-123/workspace?phase=ONBOARDING`. |
| **redirects directly to Onboarding workspace on Onboarding click** | AC 7.1.4 | **Given** a project with status `"Onboarding"`,<br>**When** the `"Onboard"` button is clicked,<br>**Then** assert that `api.put` is **not** called, and `router.push` redirects directly to `/projects/project-123/workspace?phase=ONBOARDING`. |
| **redirects to standard workspace on Onboarded click** | AC 7.1.2 | **Given** a project with status `"Onboarded"`,<br>**When** the `"Open Workspace"` button is clicked,<br>**Then** assert that the client redirects directly to `/projects/project-123/workspace` with unlocked phases. |

---

## 🕹️ 3. Manual Verification Scenarios (Browser QA Scripts)

### Scenario A: New Project Initialization (AC 7.1.1)
1. Access the APF GUI Console and click **"Create Project"**.
2. Enter a project name (e.g., `Onboarding Sandbox`) and hit Save.
3. Access your database client (or Firestore console) and check the newly created project document.
4. **Expected Result**: The project document has a `status` field populated with `"New"`.

### Scenario B: Dashboard Button Render & Redirection (AC 7.1.2, AC 7.1.3)
1. Navigate to the Projects Dashboard (`/projects`).
2. Locate the `Onboarding Sandbox` tile.
3. **Verify**: The primary CTA button on the tile is labeled `"Onboard"`.
4. Click the `"Onboard"` button.
5. **Expected Result**:
   - A network call `PUT /api/v1/projects/{id}` with `{"status": "Onboarding"}` is successfully completed.
   - The browser url transitions immediately to `/projects/{id}/workspace?phase=ONBOARDING`.
   - The database state transitions to `"Onboarding"`.

### Scenario C: Onboarding Session Resumption (AC 7.1.4)
1. Navigate back to the Projects Dashboard (`/projects`).
2. Locate the `Onboarding Sandbox` tile (which is now in the `Onboarding` status).
3. **Verify**: The CTA button continues to render labeled `"Onboard"`.
4. Open Chrome Developer Tools, switch to the **Network Tab**, and click `"Onboard"`.
5. **Expected Result**:
   - The browser routes directly to `/projects/{id}/workspace?phase=ONBOARDING`.
   - **Verify**: No outgoing `PUT` or status-updating API network calls are triggered, ensuring seamless stateless session resumption.

### Scenario D: Deep-Link Navigation Lock & Interface Gating (AC 7.1.5)
1. With the project in `"Onboarding"` status, manually edit the browser address bar to navigate to a standard phase tab, e.g., `/projects/{id}/workspace?phase=ANALYSIS`.
2. Hit Enter to force navigate.
3. **Expected Result**:
   - The page loads but immediately intercepts the URL phase parameter.
   - The UI locks the active phase strictly to `"ONBOARDING"`, resetting the URL back to `?phase=ONBOARDING`.
   - **Verify**: The Workspace Header renders *only* the `"Onboarding"` phase tab; standard tabs (`Analysis`, `Planning`, `Solutioning`, `Implementation`) are completely hidden or disabled.
   - **Verify**: The Sidebar Chat pane on the left and the Visual Canvas on the right are completely hidden. The workspace area exclusively mounts the `OnboardingPanel` component.

### Scenario E: Final Workspace Unlocking (AC 7.1.5 Transition)
1. In the active `OnboardingPanel` workspace view, locate the simulation action button labeled **"Complete Onboarding (Simulate)"**.
2. Open Chrome Developer Tools Network tab and click **"Complete Onboarding (Simulate)"**.
3. **Expected Result**:
   - An API request `PUT /api/v1/projects/{id}` with `{"status": "Onboarded"}` completes with `200 OK`.
   - The workspace view dynamically updates:
     - The exclusive `"Onboarding"` tab disappears from the header.
     - Standard tabs (`Analysis`, `Planning`, `Solutioning`, `Implementation`) immediately unlock and render in the header.
     - The layout unlocks: the Left Sidebar Chat panel and standard Visual Canvas are mounted and visible.
4. Navigate back to the Projects Dashboard (`/projects`).
5. **Expected Result**: The project card tile's CTA button now displays `"Open Workspace"`.
