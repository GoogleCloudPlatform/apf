# Story 1.1: Interactive Node Grid Layout

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a PM/UX Designer,
I want to view and interact with my project workflows in an interactive node grid layout,
so that I can visualize decision points and status triggers in real-time.

## Acceptance Criteria

1. **Given** the user navigates to the APF workspace, **When** the canvas module loads, **Then** the UI MUST display the interactive node workflow grid representing the project's state progression within **2.0 seconds** (p95 latency baseline). (AC: 1.1)
2. **Given** the active node workflow canvas, **When** rendering workflow layouts, **Then** the system MUST dynamically compile and display visual decision points (represented by Material 3 Diamond/Rhombus shapes) and state-machine status triggers. (AC: 1.2)
3. **Given** active user interactions on the canvas (e.g. clicking, expanding, or mutating workflow nodes), **When** visual states are updated, **Then** the node grid must render local state transitions in under **500ms** (p95 interaction baseline) without blocking the primary Dart main thread. (AC: 1.3)
4. **Given** multiple active viewers in a collaborative workspace session, **When** layout mutations or node status triggers are committed, **Then** visual updates MUST synchronize across all active viewers with synchronous latency under **100ms** (p95 WebSocket transmission baseline). (AC: 1.4)
5. **Given** the visual canvas popup stages, **When** a user clicks the primary **"Approve & Handoff"** action CTA, **Then** the platform MUST natively initialize a backend repository synchronization block and auto-create a GitHub Pull Request to unblock the developer, cleanly displaying confirmation via Snackbars. (AC: 1.5)

---

## Tasks / Subtasks

- [ ] **Task 1: Build the High-Fidelity Custom Node Grid Visualizer (AC: 1.1, 1.2, 1.3)**
  - [ ] **Subtask 1.1:** Create `ui/lib/screens/workspace/widgets/node_grid_layout.dart` to implement the core interactive node grid widget using standard Material 3 styles, CustomPainter, and gesture detectors.
  - [ ] **Subtask 1.2:** Map and display distinct node categories based on [ux-design-specification.md:L175-183](docs/spec/planning-artifacts/ux-design-specification.md#L175-183) (Pending: slate grey, In-Development: pulsing Electric Blue, Approved/Complete: Clean Mint Green).
  - [ ] **Subtask 1.3:** Implement smooth custom zoom, pan, and viewport centering logic inside `NodeGridLayout` using keyboard/mouse gesture triggers.
  - [ ] **Subtask 1.4:** Enforce a strict p95 performance threshold of < 500ms for node rendering under complex layouts, using localized repaint boundaries.
- [ ] **Task 2: Integrate Canvas Pane to Dynamically Support Node Grid Layouts (AC: 1.1, 1.2)**
  - [ ] **Subtask 2.1:** Update `CanvasPane` in [ui/lib/screens/workspace/canvas_pane.dart](ui/lib/screens/workspace/canvas_pane.dart) to detect if the active artifact represents a visual layout (e.g., diagram `.drawio` or JSON workflow files).
  - [ ] **Subtask 2.2:** If a visual artifact is detected, dynamically mount and render `NodeGridLayout` with the resolved workflow details, rather than falling back to standard Markdown rendering.
  - [ ] **Subtask 2.3:** Safeguard rendering. Ensure if backends return un-decryptable or missing assets, the canvas triggers a graceful mock fallback panel displaying active instructions.
- [ ] **Task 3: Wire the Primary "Approve & Handoff" Git Sync PR Loop (AC: 1.5)**
  - [ ] **Subtask 3.1:** In [ui/lib/screens/workspace/workspace.dart](ui/lib/screens/workspace/workspace.dart#L450), wire the primary "Approve & Handoff" elevated button action callback (currently empty) to invoke the project's repository synchronization endpoints.
  - [ ] **Subtask 3.2:** Execute backend PR creation: trigger `POST /api/v1/projects/{id}/git/sync` to compile GCS assets, commit them to Git branch, and open the corresponding Pull Request on GitHub.
  - [ ] **Subtask 3.3:** Gracefully handle synchronization failures (network drops, invalid auth headers), capturing raw diagnostic error events and presenting recovery retry Snackbars.
- [ ] **Task 4: High-Performance Real-Time Synchronization via WebSockets/SSE (AC: 1.4)**
  - [ ] **Subtask 4.1:** Bind `NodeGridLayout` state updates to the real-time activity stream using `ActivityStreamService` SSE endpoints.
  - [ ] **Subtask 4.2:** Ensure that active layout changes, validation states, and node tracking values synchronize in under 100ms using server-propagated notification logs.
- [ ] **Task 5: Complete Verification Testing (AC: 1.1 - 1.5)**
  - [ ] **Subtask 5.1:** Write unit/widget tests in `ui/test/screens/workspace/node_grid_layout_test.dart` to verify node mapping, shape parameters, and touch controls.
  - [ ] **Subtask 5.2:** Verify clean compilation and analyze and style boundaries: run `nextjs analyze` ensuring 0 warnings.

---

## Dev Notes

### 🏗️ 1. Relevant Architectural Design System Guardrails
*   **Material 3 Compliance:** Custom nodes, cards, and text inputs must strictly utilize default Material 3 visual definitions (`Card`, `TextButton`, `IconButton`) as detailed in [ux-design-specification.md:L127-143](docs/spec/planning-artifacts/ux-design-specification.md#L127-143). Avoid manual custom shadow borders and flat non-accessible graphics.
*   **Desktop Deep Work Focus:** The infinite canvas controls must prioritize deep-focus desktop layouts (e.g. supporting mouse wheel panning, precise double-click triggers, and keyboard shortcuts) in accordance with the platform's desktop-first strategy. 
*   **Telemetry Verification SLAs:** Ensure the canvas interactive loading threshold is strictly under **2.0s** at the 95th percentile, and local mutations reflect under **500ms** at the p95 level, matching [prd.md:NFR-1.1](../planning-artifacts/prd.md#L141-L142).

### 🚀 2. Forensic Learnings & Best Practice Warnings (disaster-prevention)
*   ** positional casting is banned:** In accordance with **Story 2.1 learnings**, never retrieve lists of features using dynamic future positioning. Always wrap async futures in clear, robust data structures (specifically **`WorkspaceData`** in [ui/lib/screens/models.dart](ui/lib/screens/models.dart)) to guarantee compile-time typing and avoid runtime casting crashes.
*   **Gating checks encapsulation:** Visual gating state checks (such as active locking transitions) MUST utilize the central models extension: **`WorkflowTemplateExtension`** (with `.isPhaseSatisfied()`), preventing custom validation duplication in visual components.
*   **SSE Packet Fragmentation:** In accordance with **Story 2.2 learnings**, network streams are vulnerable to TCP buffer splitting. All stream integrations must utilize standard Dart **`LineSplitter()`** framing to buffer packet fragments before trying to decode raw JSON arrays, preventing silent message loss.
*   **Memory leak prevention:** Any WebSocket or SSE stream listeners initiated by the node grid widgets must be completely terminated and unsubscribed inside widget **`dispose()`** overrides to prevent memory leaks and singleton deadlocks on page navigations.

---

## 🛠️ Project Structure & Source Maps

The developer must strictly touch, configure, or create the following files under these designations:

### 1. Unified Files Matrix

| File Path | Status | Operational Role |
| :--- | :--- | :--- |
| [ui/lib/screens/workspace/canvas_pane.dart](ui/lib/screens/workspace/canvas_pane.dart) | `UPDATE` | Intercept outcome active artifact loading; route to `NodeGridLayout` on diagram mappings; fallback on error. |
| [ui/lib/screens/workspace/workspace.dart](ui/lib/screens/workspace/workspace.dart) | `UPDATE` | Wire the visual stage's primary "Approve & Handoff" elevated button action to the backend PR sync endpoint. |
| [ui/lib/screens/workspace/widgets/node_grid_layout.dart](ui/lib/screens/workspace/widgets/node_grid_layout.dart) | `NEW` | Core visual interactive node grid map widget; gesture bounds, diamond deciders, and loading telemetry. |
| [ui/test/screens/workspace/node_grid_layout_test.dart](ui/test/screens/workspace/node_grid_layout_test.dart) | `NEW` | Comprehensive widget verification test suites for pan/zoom, shapes, and color metrics. |

---

## 📚 References & Traceability Links

*   **Requirements Origin:** [prd.md: FR-2 Multi-Persona Canvas](../planning-artifacts/prd.md#FR-2), [prd.md: FR-2.2 Embedded Modeling Tools](../planning-artifacts/prd.md#FR-2.2), and [NFR-1.1 / 1.2 Performance Metrics](../planning-artifacts/prd.md#NFR-1.1).
*   **Design Blueprints:** [ux-design-specification.md: Split-Pane Minimalist Choice](docs/spec/planning-artifacts/ux-design-specification.md#Design-Direction-Decision), [Custom Component Tracker](docs/spec/planning-artifacts/ux-design-specification.md#2-governance-gate-tracker), and [UX visual mocks html source](docs/spec/planning-artifacts/ux-design-directions.html).
*   **Technical Design:** [architecture.md: Technology Stack & canvaskit](docs/spec/planning-artifacts/architecture.md#Starter-Template-Evaluation) and [Visual Context sync MCP Decision](docs/spec/planning-artifacts/architecture.md#Visual-Canvas-&-UI-Builder-Integration-(MCP)).
*   **Historical Learing:** [walkthrough-story-2.1.md: Models extensions encapsulation](docs/spec/planning-artifacts/walkthrough-story-2.1.md#Models.dart), [2-2-live-cross-agent-activity-feed.md: SSE buffering and dispose overrides](docs/spec/implementation-artifacts/2-2-live-cross-agent-activity-feed.md#Dev-Notes), and [2-3-user-level-settings-screen.md: struct-based injection baseline](docs/spec/implementation-artifacts/2-3-user-level-settings-screen.md#Tasks-Subtasks).

---

## Dev Agent Record

### Agent Model Used

Gemini Next

### Debug Log References

### Completion Notes List

### File List
