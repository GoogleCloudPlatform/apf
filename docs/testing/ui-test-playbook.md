# UI Test Playbook & Specification

This document provides complete human and agentic guidelines for running and verifying UI tests across the Agentic Product Foundry workspace.

---

## 1. Environment & Setup Checklist

*   **Target Domain**: `http://localhost:8082` (Next.js React Web dev server)
*   **Next.js React Semantics Flag**: Always append `?enable-semantics=true` to the URL path during testing (e.g., `http://localhost:8082/?enable-semantics=true`). This forces the Next.js React engine to render standard HTML accessibility tags (`<flt-semantics>`), enabling robust role-based locators (like `button`, `textfield`, `heading`) instead of custom canvas hits.
*   **Authentication Flow (Local/Automated Testing)**:
    *   Manual credential entry is required for local/automated testing.
    *   Use the standard development credentials:
        *   **Username**: `admin`
        *   **Password**: `admin`
    *   Enter these credentials on the login screen and click **LOG IN** to redirect to the projects page.
*   **Active Services Required**:
    *   AI Server (`port 8080`)
    *   Firestore Emulator & Seed Configuration (`port 8084`)
    *   API Server (`port 8081`)
    *   UI Web Server (`port 8082`)

---

## 2. Target Core UI Screens & Mappings

### 2.1 Login Screen & Authentication (`/login`)
The login screen is the primary gateway route for local/automated testing.
*   **Standard Login Flow**:
    *   When the browser launches and is redirected to `/login`, enter the username `admin` and password `admin`.
    *   Click the **LOG IN** button.
*   **Success Criteria**: Login succeeds and redirects the routing context to the projects dashboard (`/projects`).

### 2.2 Project List Screen (`/projects`)
Displays all active projects assigned to the current user in a card-based grid layout.
*   **Elements**:
    *   A grid of card components representing loaded workspace units.
    *   A seeded core unit named `"Foundry Showcase"`.
*   **Actions**:
    *   Clicking the card or the open button on a project redirects the routing table to the workspace dashboard page (`/projects/:id/workspace`).

### 2.3 Settings & Credentials Panel (`/settings`)
Provides control interfaces for system colors/theme state and managing global secure platform integrations.
*   **Structure**:
    *   **Left Sidebar Split Navigation**: Contains two actionable options:
        1.  `"Visual Preferences"` (index 0) - dynamic dark/light theme triggers.
        2.  `"Application Credentials"` (index 1) - secure keys panel.
    *   **Right Scrolling Viewport**: Dynamically moves focus to the selected section on target clicks.
*   **Visual Preferences Section**:
    *   Contains radio options for `"Premium Dark Mode"` and `"Sleek Light Mode"`.
    *   Selecting an option triggers an immediate theme state change (background palette change, toggle state highlight).
*   **Application Credentials Section**:
    *   Contains lists of integration cards for platforms: `"github"`, `"figma"`, `"lucidchart"`, `"stitch"`, and `"drawio"`.
    *   Each card has fields:
        *   **Toggle Expansion**: Expands to input key forms.
        *   **Fields**: Text field inputs for `username`, `password`, `accessToken`, or `pat` depending on the integration type.
        *   **Validation Button**: Triggers standard endpoint verification. Displays an AlertDialog with `"Credentials Valid"` or `"Validation Failed"` depending on keys.

### 2.4 Workspace Dashboard & Handoff IDE Layout (`/projects/:id/workspace`)
The central developer work space featuring active state panes: Chat, Canvas, and Activity logs.
*   **Core Interactions**:
    *   **Approve & Handoff Action**: A header action button dedicated to repository sync hooks.
    *   **Actions**:
        1. Click `"Approve & Handoff"`.
        2. Wait for background sync transaction calls (`/api/workspace/sync`).
        3. Verify state transition and the green confirmation success logo banner.

---

## 3. UI Assertion Guidelines for AI Agents

When executing scenarios via the `browser_subagent` tool:
1.  **Check Redirection First**: Always confirm that the browser's URL hash pattern updates as expected after key interactions (e.g. from `#/login` to `#/projects`).
2.  **Wait for Next.js React Semantics**: Next.js React Web renders modern semantics elements progressively. Allow a small render delay (`2-3 seconds`) after route changes before scanning the DOM tree.
3.  **Explicit Authentication Steps**: Since each browser session starts clean, always perform the explicit credentials login first: enter `admin` / `admin` on the login screen, click **LOG IN**, and wait for the redirect to `/projects` before executing target page steps.
4.  **Confirm Visual States via Screenshots**: Always terminate your browser tasks by capturing a snapshot at target destinations. This provides perfect visual telemetry inside your test logs.
