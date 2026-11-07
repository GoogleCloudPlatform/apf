# Firestore Database Schema

This document describes the Firestore database schema used in the Agentic Product Foundry (APF) to support BMAD artifact governance, projects, and workspaces.

## Overview

Firestore is used as the primary operational database. Collections are created dynamically when documents are added. Default templates are seeded using the `api/seed.go` script.

## Collections

### 1. `projects`

Stores metadata for projects, access control, and governance gate status.

| Field | Type | Description |
| :--- | :--- | :--- |
| `name` | String | The name of the project. |
| `description` | String | Brief description of the project. |
| `status` | String | High-level status (e.g., "Not Started", "Active"). |
| `ownerId` | String | User ID (Firebase UID) of the project owner. |
| `visibility` | String | "public" or "private". |
| `icon` | String | Icon representing the project. |
| `repoUrl` | String | Git repository URL. |
| `repoAccessToken` | String | Personal access token for git operations. |
| `planningArtifactsLocation` | String | Location within the repo to store planning artifacts. |
| `members` | Map (String -> String) | Map of User ID to Role (e.g., "owner", "editor", "contributor", "viewer"). |
| `governanceGates` | Map (String -> String) | Map of Phase (e.g., "ANALYSIS") to Status ("Not Started", "In Progress", "Complete"). |

### 2. `artifacts`

Stores metadata for files/documents associated with a project.

| Field | Type | Description |
| :--- | :--- | :--- |
| `projectId` | String | ID of the parent project. |
| `workflowId` | String | ID of the workflow template it was created from. |
| `title` | String | Human-readable title of the artifact. |
| `phase` | String | The BMAD phase it belongs to (e.g., "PLANNING"). |
| `status` | String | Status of the artifact (e.g., "Not Started", "In Progress", "Complete"). |
| `owner` | String | The owner or persona responsible. |
| `content` | String | The actual content (markdown). *Note: Future plans may move large content to GCS.* |
| `required` | Boolean | Whether this artifact is required to pass the gate for its phase. |
| `isCustom` | Boolean | Whether it was added manually by a user (not from template). |

### 3. `workflows`

Stores templates for different methodologies (e.g., BMAD).

| Field | Type | Description |
| :--- | :--- | :--- |
| `name` | String | Name of the template (e.g., "Standard BMAD Workflow"). |
| `framework` | String | The framework name (e.g., "BMAD"). |
| `phases` | Map | Lists actions per phase. |
| `artifacts` | Map | Lists default artifacts per phase. |
| `dependencies` | Map | Lists prerequisite phases for each phase (Phase Name -> Array of Prerequisite Phase Names). |

### 4. `configs`

Stores global configuration settings for the application.

| Field | Type | Description |
| :--- | :--- | :--- |
| `document_id` | Identifier | Use `global` for system-wide settings. |
| `gcsWorkspaceBucket` | String | Base GCS bucket URI for artifact workspace storage. |

## Governance Gates

Gate status is calculated asynchronously whenever an artifact is saved.
- **Complete:** All required artifacts in the phase are marked as "Complete".
- **In Progress:** At least one artifact is updated, but not all required are complete.
- **Not Started:** All artifacts in the phase are untouched.

### Phase Access Control (Gating)

Access to a phase in the UI is controlled by dependencies defined in the `workflows` collection. A phase is enabled only if all its listed prerequisite phases have a status of "Complete" in the project's `governanceGates`.

## Seeding

Run the following command to seed the default BMAD template:
```bash
go run . -seed
```
*(Requires valid Google Cloud credentials accessible by Application Default Credentials).*
