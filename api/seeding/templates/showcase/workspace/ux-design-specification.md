# UX Design Specification
## Project: Foundry Showcase (On-Demand Provisioning)

**Branding**: Foundry Showcase  
**Status**: Final  
**Version**: 1.0  
**Authors**: UX Design  

---

## 1. Interaction Points & Entry Areas

### 1.1 Settings Screen - Project Templates Tab
A new navigation tab named **Project Templates** is added to the Settings dialog. This tab hosts the cards for pre-configured projects.
- **Showcase Card UI**:
  - Displays the project icon (🚀), title ("Foundry Showcase"), and a description.
  - **State A (Not Provisioned)**: Displays a primary action button labeled **Provision Foundry Showcase**.
  - **State B (Provisioned)**: Displays a secondary action button labeled **Restore to default state** and an indicator showing "Status: Provisioned".

### 1.2 Dashboard Workspace Integration
Once provisioned, the showcase project appears on the user's Projects Dashboard.
- **Tile Styling**:
  - The tile features a distinctive **Sample Project** badge (rendered in a subtle theme color, e.g., emerald or indigo) to clearly distinguish it from active production workspaces.
  - Clicking the tile opens the workspace directly.

---

## 2. Dynamic Loading & Status Feedback

To ensure a highly polished, responsive user experience:
- **Full-Screen Loading Overlay**: Provisioning and resetting trigger a modal, semi-transparent loading screen displaying a spinner and a description (e.g., *"Provisioning Showcase Project..."* or *"Restoring workspace to default state..."*).
- **Toast / Snackbar Rules**:
  - Upon successful provisioning: Close the Settings modal, refresh the Dashboard, and show: *"Showcase project provisioned successfully!"*
  - Upon successful reset: Refresh the active workspace screen and show: *"Showcase project restored to default state!"*
  - If an error occurs: Keep the modal open and display a prominent error snackbar: *"Failed to provision showcase project. Please try again."*

---

## 3. Gated Tab Navigation Rules
Within the Showcase workspace:
- The pipeline tracking bar and workspace tabs are dynamically gated based on artifact completion rules:
  - **Onboarding**, **Planning**, and **Solutioning** tabs are fully unlocked, and their respective pizza trackers display a **Complete** status pill because their required artifacts (`project-context.md`, `prd.md`, `architecture.md`, `ux-design-specification.md`, `epics.md`) are marked as `Complete`.
  - The **Implementation** tab remains locked (or gated) because no implementation artifacts exist yet, and its pipeline tracker displays **Not Started** (🔘).
