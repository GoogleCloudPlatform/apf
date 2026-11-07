# Product Requirements Document (PRD)
## Project: Foundry Showcase (On-Demand Provisioning)

**Branding**: Foundry Showcase  
**Status**: Final  
**Version**: 1.0  
**Authors**: Product Management  

---

## 1. Executive Summary
The **Foundry Showcase** feature is an onboarding accelerator designed to demonstrate the full capabilities of the Agentic Product Foundry immediately upon a fresh install. Rather than confronting users with an empty dashboard and locked workspaces, this feature allows them to provision a realistic, pre-configured sample project containing pre-populated spec files, complete database records, and a rich, multi-turn chat history. This provides immediate, interactive context, allowing users to explore how specialized agents collaborate across the Onboarding, Analysis, Planning, and Solutioning phases.

---

## 2. Business Objectives & Success Metrics
- **Onboarding Acceleration**: Reduce the time-to-value (TTV) to under 10 seconds.
- **Platform Demonstration**: Showcase a fully designed, planned, and architected workspace (completing 4 out of 5 governance gates).
- **Zero Configuration**: Users can explore the platform without connecting any external VCS (e.g., GitHub) or cloud databases.
- **Frictionless Recovery**: Provide an instant "Restore to default state" action to reset the workspace back to its clean default state.

---

## 3. Scope of Requirements

### 3.1 Backend Seeding Engine (In-Scope)
- **Deterministic Provisioning**: Handle `POST /api/projects/provision-showcase` and generate a unique, deterministic project ID (`showcase-{orgID}-{userID}`) to prevent race conditions.
- **Embedded Templates**: Use Go `embed.FS` to bundle JSON database models and Markdown specs into the API tier.
- **Firestore Seeding**: Hydrate Firestore tables (`projects`, `sessions`, `messages`, `artifacts`) for the target user.
- **Workspace Generation**: Dynamically extract and write the embedded markdown spec files to the project's local directory under `docs/spec/`.
- **Fail-Safe Cleanup**: Execute transactions or defer-catch blocks to completely wipe partially-created resources if seeding fails.

### 3.2 Reset & Restoration Engine (In-Scope)
- **Clean Purge**: Handle `POST /api/projects/{id}/reset` to delete all database sessions, messages, and artifacts associated with the showcase.
- **Filesystem Deletion**: Completely clear the project's filesystem folder.
- **Re-seeding**: Invoke the seeding engine to restore the project back to its pristine default state using the same project ID.

### 3.3 Security Guardrails (In-Scope)
- **ID Prefix Restriction**: Restrict the reset action strictly to project IDs prefixed with `showcase-`.
- **Path Traversal Shield**: Prevent traversal attacks by sanitizing project ID strings (disallowing `.`, `..`, `/`, `\`).

---

## 4. User Journeys

### 4.1 Provisioning the Showcase
1. **Discover**: A new user logs in and navigates to the empty Projects Dashboard.
2. **Access**: The user opens Settings and clicks the "Project Templates" tab.
3. **Provision**: The user clicks "Provision Foundry Showcase".
4. **Onboarding Tour**: The system shows a full-screen loading spinner while provisioning database records and writing files.
5. **Success**: A success toast notification is shown: *"Showcase project provisioned successfully!"* The Settings modal closes, and the Projects Dashboard displays the new "Foundry Showcase" project tile adorned with a "Sample Project" badge.
6. **Interaction**: The user clicks the tile, entering a fully-populated canvas with a complete chat history, active governance gates, and interactive specs.

### 4.2 Resetting the Showcase
1. **Modify**: A user plays around with the showcase, edits files, sends new chat messages, and moves artifacts.
2. **Reset**: To start over, the user returns to Settings -> Project Templates, and clicks "Restore to default state" on the showcase card.
3. **Confirm**: The system clears the modified files and database records, re-seeds the showcase, and displays: *"Showcase project restored to default state!"*
