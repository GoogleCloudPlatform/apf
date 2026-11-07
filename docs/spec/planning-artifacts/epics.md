---
stepsCompleted: ["step-01-validate-prerequisites", "step-02-design-epics", "step-03-create-stories", "step-04-final-validation"]
inputDocuments: ["docs/spec/planning-artifacts/prd.md", "docs/spec/planning-artifacts/architecture.md", "docs/spec/planning-artifacts/ux-design-specification.md"]
---

# agentic-foundry - Epic Breakdown

## Overview

This document provides the complete epic and story breakdown for agentic-foundry, decomposing the requirements from the PRD, UX Design if it exists, and Architecture requirements into implementable stories.

## Requirements Inventory

### Functional Requirements

- FR-1.1: The system SHALL display a visual Idea-to-Production pipeline tracking bar representing continuous state progression from Ideation through Architecture.
- FR-1.2: The system SHALL provide a live collaboration activity feed broadcasting cross-agent validation requests and status updates.
- FR-1.3: The system SHALL provide a dedicated user-level settings screen allowing individuals to securely input and manage their personal API keys and authorization tokens required for third-party MCP integrations (e.g., Figma, Lucidchart, Stitch).
- FR-1.4: The system SHALL provide a collapsible right-hand pop-out menu for artifact navigation. When expanded via a persona icon click, it SHALL display a list of artifacts collected by the specific active persona, and allow users to view collections of artifacts from other personas (e.g., UX, Engineering) for reference.
- FR-1.5: The system SHALL provide project-level application integration settings that dynamically render a custom configuration form (e.g., specification directory path, base git branch, spec branch prefix for GitHub) and a searchable repository selector, persisting these parameters under an array of app_integrations inside the project database collection.
- FR-2.1: The sidebar interaction SHALL support instructing agents visually or narratively with response latency < 2 seconds.
- FR-2.2: The grid model SHALL render node workflow layouts displaying decision points and status triggers with < 500ms render state updates.
- FR-2.3: The system SHALL support uploading files (up to 10MB) through the chat interface.
- FR-2.4: The system SHALL extract text content from uploaded files (supporting standard text, markdown, pdf, and code files like `.py`, `.go`, `.json`, `.yaml`) to be used as context.
- FR-2.5: The AI orchestrator SHALL inject the extracted file content into the agent's context window for subsequent prompts in the active session.
- FR-2.6: The system SHALL allow a user to cancel an in-progress chat message, immediately terminating the active agent execution and stopping further processing for that message.
- FR-3.1: The system SHALL enable mapping golden path service catalogs to template configurations brokers via standard adapter hooks.
- FR-3.2: The plug-in system SHALL absorb previous codebase architectures and patterns and update agent contexts within < 5 minutes of ingest pull triggers.
- FR-4.1: The system SHALL automatically check-in generated specifications (e.g., `design-artifacts/`) to the local git branch upon state save anchors.
- FR-4.2: The system SHALL enforce commit safety guardrails, verifying that files are only added to spec-approved locations, `.gitignore` exists in the root, and ensuring the `.agents` directory is NOT ignored in `.gitignore` so that customizations can be tracked and committed.
- FR-4.3: The system SHALL pull the latest remote Git commits back into the workspace to re-hydrate and edit finalized specifications dynamically via the API tier.
- FR-4.4: The system SHALL centralize all Git and version control operations in the API tier, completely decoupling downstream agent inference services from repository management.
- FR-4.5: The workspace Explorer section SHALL provide a tabbed interface containing two tabs: Artifacts View and File Explorer View.
- FR-4.6: The system SHALL provide a recursive File Explorer component and an API to fetch and display the project directory tree on-demand (one level at a time) using file-type-specific colorful icons.
- FR-7.1: The system SHALL support a managed project-level lifecycle. Newly created projects MUST initialize in the database with a status of `New`.
- FR-7.2: The dashboard workspace SHALL dynamically check project status. Projects in the `New` or `Onboarding` state MUST render an action button labeled `Onboard` on their Projects Dashboard tile.
- FR-7.3: When the PM triggers onboarding, the system SHALL transition the project status to `Onboarding` and launch an autonomous background agent to analyze the target repository.
- FR-7.4: The Onboarding Tab SHALL present an interactive tree picker displaying discovered legacy files for PM ingestion selection.
- FR-7.5: The Onboarding Tab SHALL provide an explicit "Complete Onboarding" action. Upon completion, the system SHALL transition the project status to `Onboarded`, auto-generate `project-context.md`, initialize and package the `.agents` directory with default templates in the codebase root, and commit both to the Git repository. Active workspace tools SHALL NOT be locked or gated pending this completion.
- FR-9.1: The system SHALL automatically trigger the Product Onboarding Tour when a user logs in for the first time.
- FR-9.2: The system SHALL provide a clear and accessible option (e.g., a "Help" menu item or a dedicated "Tour" button in the header) for users to manually restart the tour at any time.
- FR-9.3: The tour MUST be updated to include the newly introduced sections of the GUI Workspace: User Secrets Settings Modal, Onboarding Tab & Project Discovery, and Chat File Upload.
- FR-9.4: The tour interface SHALL provide Next/Back buttons for step navigation, a "Skip Section" action to bypass the entire active section, and a "Skip Tour" action to dismiss the tour.
- FR-9.5: The system SHALL persist the user's tour progress (completed sections, last active step, and overall completion status) in Firestore.
- FR-9.6: The system SHALL support state recovery and interruption resiliency, allowing the user to resume from the last saved step or automatically advancing if they complete the required action independently.
- FR-9.7: If a tour session was interrupted and is in-progress, and the user navigates to a new screen or section they have not yet toured, the tour SHALL automatically resume/start from the first step of that specific screen or section.
- FR-9.8: The system SHALL seed exactly one pre-configured showcase project named 'Foundry Showcase' using an embedded data folder deployed with the API tier, providing users with immediate reference materials on a fresh install.
- FR-8.1: System should support multitenancy.
- FR-11.1: The system SHALL support provisioning a new Google Identity Platform Tenant (Organization) via an administrative backend API.
- FR-11.2: The system SHALL support configuring user authentication, including Email/Password and Google OAuth IDP providers, at both the project and tenant level.
- FR-11.3: The system SHALL intercept and validate Google Cloud Identity-Aware Proxy (IAP) OIDC JWT assertions at the backend API tier.
- FR-11.4: The system SHALL support Parent Project level authentication and authorization for Super Administrators, allowing global resource management.
- FR-11.5: The system SHALL support tenant-level organization owners and administrators, retrieving user lists and managing users using custom claims.
- FR-12.1: The system SHALL provide a modular Job registry in the API tier and a standalone CLI entrypoint (`api/cmd/jobs/main.go`) capable of executing individual batch housekeeping jobs with argument parsing, structured logging, and graceful signal cancellation.
- FR-12.2: The system SHALL implement modular housekeeping jobs for NFS storage usage audits and orphaned workspace cleanup with dry-run support and Firestore metadata sync.
- FR-12.3: The system SHALL compile both the HTTP web server (`/apf-api`) and the Jobs CLI runner (`/apf-jobs`) into a single container image in the multi-stage Docker build pipeline.
- FR-12.4: The infrastructure SHALL provision Cloud Run v2 Jobs and Cloud Scheduler cron triggers in Terraform with VPC Direct Egress, NFS mount, and Secret Manager bindings.
- FR-12.5: The backend API SHALL provide an administrative endpoint (`POST /api/v1/admin/jobs/{jobName}/run`) allowing Super Administrators to trigger Cloud Run Job executions on-demand from the Web UI.
- FR-13.1: The system SHALL provide a responsive, touch-optimized mobile experience (Mobile Web/PWA and Flutter Mobile target) enabling developers to create, inspect, and build applications from mobile devices.
- FR-13.2: The mobile workspace SHALL provide a touch-first chat interface with voice input capabilities, quick action prompt chips, and instant agent response streaming.
- FR-13.3: The mobile viewport SHALL render adaptive tabbed views (Canvas Grid, File Explorer, Artifact Viewer) with pinch-to-zoom, gesture navigation, and touch-optimized node layout controls.
- FR-13.4: The system SHALL send push notifications for agent task completions/approvals and support transient offline editing state recovery.
- FR-15.1: The system SHALL provide configuration management for external integrations, enabling users to input and securely store their Gemini API key for use in Gemini model calls.
- FR-15.2: The system SHALL store global feature flag configurations at the application level in a Firestore collection named 'features', where top-level documents represent overall features and document properties represent sub-features and UI behavior settings. The system SHALL provide an API to fetch feature flag documents on-demand per screen/UI route rather than loading all features at application startup.
- FR-15.3: The system SHALL provide a Feedback Collection Modal Component and top notice banner triggers, persisting submitted feedback to the root-level `feedback` Firestore collection.
- FR-16.1: The system SHALL provision the complete standard toolset (`file_reader`, `file_writer`, `directory_reader`, and `uv_run`) across all agent swarms and specialist catalogs, including the Analysis and Help swarms.
- FR-16.2: The `uv_run` tool SHALL enforce strict script path validation and reject non-Python shell binaries (`find`, `ls`, `grep`, `python3`) immediately with structured guidance redirecting the agent to `directory_reader` or `file_reader`.
- FR-16.3: The system SHALL automatically generate and maintain an up-to-date `_bmad/custom/config.user.toml` in the project workspace mount during workspace initialization, seeding dynamic session and repo context (`user_name`, `project_name`, `output_folder`, `planning_artifacts`, `implementation_artifacts`).
- FR-16.4: The system SHALL eliminate in-flight regex string mutations on markdown instructions (`SKILL.md`) and configuration files, relying on BMad's native TOML resolver scripts (`resolve_config.py`, `resolve_customization.py`) executed via `uv_runner`.
- FR-16.5: The system SHALL enforce zero-trust boundary sandboxing and path normalization in a unified single-pass interceptor pipeline without circular relative/absolute path transformations.
- FR-16.6: The system SHALL provide an automated end-to-end regression test harness validating brainstorm activation, directory globbing, and multi-layer TOML configuration resolution.
- FR-17.1: The system SHALL provide a generic email notification adapter subsystem with an `EmailNotificationProvider` interface and thread-safe registry in `api/notifications/` supporting CNP, Console, and Mock implementations, strictly injectable into API handlers.
- FR-17.2: The system SHALL implement a `CNPEmailProvider` for Custom Notification Provider (CNP) Admit API supporting direct API-rendered HTML/text emails, IAM token impersonation with in-memory caching via `oauth2.ReuseTokenSource` for `NOTIFICATIONS_SENDER_SERVICE_ACCOUNT`, and `ALREADY_EXISTS` (409) deduplication as idempotent success.
- FR-17.3: The system SHALL provide an API endpoint `POST /api/v1/notifications/email` with 1MB payload limits (`http.MaxBytesReader`), mandatory `htmlContent` validation, single-recipient atomic dispatch, and org-scoped authentication (`authMiddleware` + `orgScopeMiddleware`).
- FR-17.4: The system SHALL provide an administrative endpoint `GET /api/v1/notifications/providers` restricted to Org Admins, Org Owners, and SuperAdmins returning registered providers and the active default.
- FR-17.5: The infrastructure SHALL provision a dedicated notifications sender service account in Terraform, configure IAM token creator bindings for the Cloud Run compute SA, inject `NOTIFICATIONS_*` environment parameters via Cloud Deploy, and export the sender SA for CNP `api-senders` group membership.
- FR-18.1: The system SHALL process chat-uploaded files through the platform's `ArtifactService` and register them in the artifacts collection with metadata linking them to the active `projectId` and `sessionId`.
- FR-18.2: The system SHALL support a multi-tier `ArtifactService` storage strategy: storing chat-uploaded artifacts in Google Cloud Storage (GCS) using the configured `GCS_ARTIFACT_BUCKET` in Staging/Production cloud environments, and storing artifacts in the configured local filesystem path (`LOCAL_ARTIFACT_PATH`) in local development mode (initialized via `mise.toml` / environment setup).
- FR-18.3: The system SHALL extract text and document content from chat-uploaded artifacts and inject the content into prompt context for active agent turns.
- FR-18.4: When an uploaded file is an image (`.png`, `.jpg`, `.jpeg`, `.webp`, `.gif`) or non-text media, the AI orchestrator SHALL package the image as multipart content (`InlineData` / multimodal parts) in the LLM prompt payload, allowing vision-capable models to analyze images alongside text prompts.
- FR-18.5: When files are uploaded directly into the project tree (via File Explorer upload actions or drag-and-drop), the system SHALL save the files to the designated folder path in the workspace filesystem (mounted NFS/local) and trigger an immediate File Explorer tree refresh.
- FR-18.6: In the Chat session history/list, chat sessions containing associated artifacts SHALL display an expandable chevron indicator on the right; clicking a session with artifacts SHALL toggle a dropdown/sub-list displaying all artifacts uploaded in that session.
- FR-18.7: When an image-type artifact (`.png`, `.jpg`, `.jpeg`, `.gif`, `.webp`, `.svg`) is clicked, the preview pane SHALL render the image in a dedicated Image Viewer with zoom, pan, and scaling, while suppressing Monaco code editor toolbars, text editing controls, and specification "Approve" buttons.
- FR-18.8: The File Explorer SHALL render interactive action icons on the right of folder nodes on hover (`+` to upload files to that specific folder, `Refresh` to reload that folder's subtree, and `Delete` to delete the folder with confirmation), and render a `+` icon in the "All Files" header to upload files directly into the project root.
- FR-19.1: The Implementation Tab UI SHALL render a dedicated "Build Sandbox" action button positioned beneath the chat input textbox.
- FR-19.2: When the user clicks "Build Sandbox", the system SHALL invoke the Sandbox Builder skill/agent to scan project context (`project-context.md`, `AGENTS.md`, `architecture.md`, `prd.md`, and specification artifacts) to discover runtime languages, frameworks, system libraries, and required runtime environments.
- FR-19.3: The UI SHALL render an interactive Sandbox Stack Configuration screen/table displaying discovered runtime environments, dependencies, and packages in a structured table, allowing users to add new items, edit parameters, or remove items before confirming build execution.
- FR-19.4: The Sandbox Builder Agent SHALL generate a deterministic `Dockerfile` (and runtime sandbox specification plan) containing all required runtimes, system dependencies, and package configurations matching the confirmed stack list.
- FR-19.5: The system SHALL provide an automated sandbox validation and test harness capable of building the generated `Dockerfile` image in a secure, isolated container execution environment.
- FR-19.6: The test harness SHALL execute container verification sweeps and health tests, validating binary availability, toolchain versions, runtime dependencies, and exposed port/environment configurations.
- FR-19.7: The system SHALL support ephemeral deployment and smoke testing of the built sandbox image, streaming build logs and diagnostic verification results back to the user workspace.

### NonFunctional Requirements

- NFR-1.1: The dashboard and canvas components SHALL achieve interactive load time in < 2 seconds.
- NFR-1.2: Canvas node updates and layout triggers SHALL synchronize across active viewers with synchronous latency < 100ms.
- NFR-1.3: File upload and content extraction latency SHALL be less than 3 seconds for files up to 5MB.
- NFR-1.4: Cancellation Latency: The system SHALL terminate active agent processing within 1 second of the user triggering the cancellation.
- NFR-1.5: SSE token delivery latency SHALL be < 50ms per chunk during live agent execution.
- NFR-1.7: On-demand directory tree fetch latency SHALL be < 150ms.
- NFR-2.1: The APF Console Platform wrapper SHALL maintain 99.9% uptime availability (measured monthly).
- NFR-2.2: File storage reliability SHALL ensure no data loss for active session attachments.
- NFR-3.1: The canvas module and workspace cluster SHALL support at least 200 concurrent active users per workspace cluster session without degradation, verified via automated distributed load testing suites.
- NFR-9.1: Storing and retrieving the tour state from Firestore SHALL not introduce visible latency (p95 < 200ms).
- NFR-9.2: The tour navigation (including "Skip Section") MUST be fully keyboard-navigable and screen-reader accessible.
- NFR-13.1: Touch interaction latency SHALL be < 100ms and initial mobile viewport render time SHALL be < 500ms on standard mobile devices.
- NFR-13.2: Mobile UI components SHALL adhere to touch target guidelines (minimum 48x48px) and support seamless orientation switching.
- NFR-16.1: Command path validation and redirection in `uv_runner` SHALL execute in < 1ms without spawning operating system subprocesses or acquiring semaphore locks.
- NFR-16.2: Native BMad TOML configuration resolution via `resolve_config.py` in `uv_runner` SHALL complete in < 200ms per agent turn.
- NFR-17.1: The email notification handler SHALL fail fast on upstream provider errors (returning 502/500) and cancel in-flight provider calls upon HTTP client disconnect.
- NFR-17.2: The email notification handler SHALL record custom OpenTelemetry spans (`notifications.send_email`) with provider, recipient domain, and status attributes to Cloud Trace.
- NFR-17.3: The system SHALL support offline local simulation (`NOTIFICATIONS_ENVIRONMENT=local`) for the CNP provider verifying sender service account formatting and message structure without network calls.
- NFR-18.1: Uploading and persisting artifacts to GCS (in Cloud) or local filesystem (in local mode) SHALL complete in < 2.0s for files up to 10MB.
- NFR-18.2: Expanding a chat session's artifact dropdown SHALL query and render the associated artifact list in < 100ms.
- NFR-18.3: All file uploads and workspace storage operations MUST enforce zero-trust path sanitization (`filepath.Clean`) and MIME-type validation, preventing directory traversal attacks or unauthorized file overwrites.
- NFR-19.1: Sandbox specification analysis and stack discovery SHALL complete in < 10 seconds.
- NFR-19.2: The sandbox test harness SHALL enforce a strict build and test timeout (maximum 5 minutes) and execute within isolated container sandboxes to prevent host system contamination.
- NFR-19.3: Ephemeral sandbox test deployments SHALL be automatically garbage collected and torn down upon test completion or cancellation within < 10 seconds.

### Additional Requirements

- Architecture Requirement (Adapters): Integration adapters SHALL support Figma sync utilizing Figma Web API v1+ with sync completion < 10 seconds for standard files.
- Architecture Requirement (Connectors): Plug endpoints SHALL fetch IDP schema metadata structures utilizing HTTPS/TLS 1.3 connections.
- Architecture Requirement (Endpoints): Endpoint configuration configurations SHALL feed AntiGravity workspace packages compliant with JSON-RPC standard endpoints.
- Architecture Requirement (Storage): Uploaded files SHALL be stored in a project-specific, organization-isolated GCS bucket path (e.g., `gs://apf-scratchpad/projects/{projectId}/uploads/`).
- Security Requirement: All operations SHALL run strictly within organization VPC bounds with complete network isolation from public endpoints.
- Security Requirement: The system SHALL capture and retain audit logs for all agent-human interactions with 99.99% capture reliability and review retention specs of 12 months.
- Security Requirement (DLP): Uploaded files MUST be routed through the DLP/Model Armor pipeline to scrub sensitive data (PII, credentials) before being sent to the LLM.
- Architecture Requirement (Cancellation): The backend API and AI tiers MUST support execution cancellation propagation (via Go contexts or task cancellation APIs) to immediately abort active GenAI inferences and tool executions.
- Architecture Requirement (Tour Storage): The user profile schema in Firestore MUST be updated to include a `tour_status` object (e.g., `{ completed: boolean, last_step: number }`).
- Architecture Requirement (Tour Frontend): The frontend SHALL use a robust React tour library (e.g., react-joyride) or a custom overlay-based tour wizard.
- Security Requirement (IAM-Claims): Custom roles and granular permissions (e.g., `view:billing`, `edit:settings`) SHALL be mapped to Google Identity Platform custom claims using `tenantAuth.setCustomUserClaims()`.
- Architecture Requirement (k6 Testing): Establish a dedicated performance testing directory at `{project-root}/perf` configured with k6 load testing tools following modular best practices (`perf/scenarios/`, `perf/utils/`, `perf/config/`, `perf/reports/`).
- Architecture Requirement (Data Seeding): Provide automated CLI scripts and data fixture generators to seed realistic multi-tenant workspace data, project file trees, active session histories, and synthetic user accounts into Firestore and GCS storage enclaves.

### UX Design Requirements

- UX-DR1: The chat input area SHALL feature a file attachment button (clip icon).
- UX-DR2: Attached files SHALL be displayed as removable chips/tokens below the text input before sending.
- UX-DR3: The chat history SHALL display file attachments associated with each message, allowing users to download or view them.
- UX-DR4: The UI SHALL display an upload progress indicator and error state if the upload fails (e.g., file too large, unsupported format).
- UX-DR5: The chat interface SHALL display a visible 'Cancel' action button (e.g., a stop icon) next to or within the message bubble of an actively processing message. The button SHALL disappear once processing is complete.
- UX-DR-9.1: The tour popover card MUST display both "Skip Step" and "Skip Section" controls clearly.
- UX-DR-9.2: The popover MUST show a section-level progress indicator (e.g., "Section 2 of 4: User Secrets" and "Step 1/3").
- UX-DR-9.3: Visual cue (e.g., pulsing highlight) on the next target element if the user manually performs the required action.
- UX-DR-11.1: The System Admin Portal SHALL provide an Organization Provisioning form to input Org Name, designated Owner Email, and a "Provision Organization" action.
- UX-DR-11.2: The Org Admin dashboard SHALL provide a Member Directory screen featuring a member list table (Name, Email, Role, Status), an "Invite Member" action, and role edit/revoke actions.
- UX-DR-11.3: The Member Directory SHALL provide an "Invite Member" modal/form requesting Email, Role selection dropdown, and a "Send Invite" action.
- UX-DR-11.4: The Org Admin dashboard SHALL provide a Roles & Permissions Configuration screen showing a read-only list of predefined roles and a Custom Roles tab with a "Create Custom Role" checklist.
- UX-DR-11.5: The platform SHALL provide a tenant-scoped login page and an Invitation Acceptance landing page configured dynamically with the tenant ID.
- UX-DR-11.6: The Admin Console Performance Dashboard SHALL display real-time APM telemetry metrics, latency distribution charts (WebSocket frame latency, SSE token delivery rate, directory fetch timing), active user concurrency gauges, and k6 benchmark run history reports.
- UX-DR-13.1: The mobile interface SHALL feature a persistent bottom navigation bar for quick tab switching (Canvas, Chat, Explorer, Settings).
- UX-DR-13.2: Sidebars SHALL automatically convert to collapsible slide-over drawers with touch swipe dismiss gestures on mobile viewports.
- UX-DR-18.1: The chat input area SHALL display a file attachment button (clip icon) supporting text, markdown, PDF, code files, and images, rendering removable badge chips with thumbnails and size tags for attached items.
- UX-DR-18.2: Chat session items with attached artifacts SHALL render an expandable chevron indicator that reveals a dropdown of uploaded session artifacts with preview and reference triggers.
- UX-DR-18.3: The artifact preview pane SHALL render a dedicated Image Viewer without Monaco action bars, text editing controls, or specification approval buttons when viewing image-type artifacts.
- UX-DR-18.4: File Explorer folder nodes SHALL reveal `+` (Add File), Refresh, and Delete action buttons on hover, and the "All Files" header SHALL feature a `+` button to upload files to the project root.
- UX-DR-19.1: The Implementation Tab chat area SHALL render a primary action button labeled "Build Sandbox" positioned under the chat input textbox.
- UX-DR-19.2: Clicking "Build Sandbox" SHALL open an interactive modal/panel displaying an editable table of discovered runtime stack components (Category, Package/Tool, Version, Status, Actions) with "Add Dependency", "Edit", "Remove", and "Confirm & Generate Dockerfile" actions.
- UX-DR-19.3: The Sandbox Build modal SHALL provide live streaming build logs, verification progress indicators (checking binaries, running health sweeps), and ephemeral container status.

### FR Coverage Map

FR-1.1: Epic 2 - Pipeline tracking bar
FR-1.2: Epic 2 - Collaboration activity feed
FR-1.3: Epic 2 - User-level secrets management
FR-1.5: Epic 2 - Project-level app integration settings
FR-2.1: Epic 1 - Canvas sidebar chat
FR-2.2: Epic 1 - Node grid layout
FR-2.3: Epic 8 - File upload for chat context
FR-2.4: Epic 8 - File upload for chat context
FR-2.5: Epic 8 - File upload for chat context
FR-2.6: Epic 1 - Cancel chat message
FR-3.1: Epic 3 - Golden path triggers via adapters
FR-3.2: Epic 3 - Context ingest calibration
FR-4.1: Epic 4 - Git state sync tracker
FR-4.2: Epic 4 - Remote synchronization review gates, pre-commit checks, and .gitignore lifecycle management
FR-4.3: Epic 4 - Git-to-Workspace spec re-hydration & edit sync
FR-4.4: Epic 4 - Centralized version control architecture
FR-4.5: Epic 4 - Tabbed workspace explorer
FR-4.6: Epic 4 - On-demand recursive file explorer
FR-7.1: Epic 7 - Project lifecycle state machine
FR-7.2: Epic 7 - GUI Dashboard gating & tile actions
FR-7.3: Epic 7 - Autonomous background discovery agent
FR-7.4: Epic 7 - Ingestion tree gating & pre-existing spec toggles
FR-7.5: Epic 7 - Explicit completion gating
FR-9.1: Epic 9 - First-time login trigger
FR-9.2: Epic 9 - Manual trigger/replay
FR-9.3: Epic 9 - Expanded target sections (Secrets, Onboarding, File Upload)
FR-9.4: Epic 9 - Skip controls (Step, Section, Tour)
FR-9.5: Epic 9 - Firestore state persistence
FR-9.6: Epic 9 - State recovery & resiliency
FR-9.7: Epic 9 - Context-aware auto-start on navigation
FR-9.8: Epic 9 - Seeded showcase project
FR-8.1: Epic 10 - Multi-Tenancy Isolation & Management
FR-11.1: Epic 11 - Identity & Access Platform Configuration
FR-11.2: Epic 11 - Identity & Access Platform Configuration
FR-11.3: Epic 11 - Identity & Access Middleware
FR-11.4: Epic 11 - Super Admin Authentication & Authorization
FR-11.5: Epic 11 - Org Admin Portal & User Management
FR-12.1: Epic 12 - Housekeeping Job Framework, Registry & CLI Dispatcher
FR-12.2: Epic 12 - Modular NFS Storage Audit & Orphaned Workspace Cleanup Jobs
FR-12.3: Epic 12 - Multi-Binary Container Packaging in Dockerfile & CI/CD
FR-12.4: Epic 12 - Cloud Run v2 Job & Cloud Scheduler Terraform Infrastructure
FR-12.5: Epic 12 - Super Admin On-Demand Job Execution API & Admin Portal Trigger UI
FR-12.6: Epic 12 - Jobs Catalog Discovery API
FR-12.7: Epic 12 - Cloud Run Job Execution History & Live Status API
FR-12.8: Epic 12 - Dedicated Super Admin Jobs Dashboard & Live Execution Monitor UI
FR-12.9: Epic 12 - System Admin Portal Navigation Integration
FR-13.1: Epic 13 - Mobile application core workspace
FR-13.2: Epic 13 - Mobile agent chat interface
FR-13.3: Epic 13 - Mobile touch canvas & gesture node explorer
FR-13.4: Epic 13 - Mobile push notifications & real-time sync
FR-14.1: Epic 11 - Application Admin Performance Dashboard UI
FR-15.1: Epic 15 - Gemini API key input and integration management
FR-15.2: Epic 15 - Application-level feature flags and on-demand feature document loading API
FR-15.3: Epic 15 - Feedback Collection Modal Component & Top Banner Triggers
FR-16.1: Epic 16 - Unified toolset provisioning across all agent swarms
FR-16.2: Epic 16 - UVRunnerTool binary command guardrails & tool-redirect guidance
FR-16.3: Epic 16 - Native project TOML workspace configuration seeding
FR-16.4: Epic 16 - Elimination of brittle in-flight config rewrites and markdown AST mutations
FR-16.5: Epic 16 - Consolidation of path safety, sandboxing & zero-trust boundary enforcement
FR-16.6: Epic 16 - End-to-end regression test suite for brainstorming activation & TOML resolution
FR-17.1: Epic 17 - Generic email notification adapter subsystem & provider registry
FR-17.2: Epic 17 - Custom Notification Provider (CNP) Admit provider
FR-17.3: Epic 17 - API email notification dispatch endpoint
FR-17.4: Epic 17 - Notification provider introspection & administration endpoint
FR-17.5: Epic 17 - Notifications infrastructure & Terraform provisioning
FR-18.1: Epic 18 - Chat upload integration with ArtifactService
FR-18.2: Epic 18 - Multi-tier storage strategy (GCS in Cloud / Local FS in dev via mise)
FR-18.3: Epic 18 - Text document extraction and prompt context injection
FR-18.4: Epic 18 - Multimodal image ingestion as multipart content
FR-18.5: Epic 18 - Project tree direct file upload & File Explorer refresh
FR-18.6: Epic 18 - Chat session expandable chevron and artifact dropdown
FR-18.7: Epic 18 - Image artifact preview & media viewer without Monaco controls
FR-18.8: Epic 18 - File Explorer folder hover actions and header root upload
FR-19.1: Epic 19 - Implementation Tab "Build Sandbox" trigger button
FR-19.2: Epic 19 - Specification analysis & runtime stack discovery skill/agent
FR-19.3: Epic 19 - Interactive Sandbox Stack Configuration table UI
FR-19.4: Epic 19 - Autonomous Dockerfile synthesis & runtime plan generation
FR-19.5: Epic 19 - Isolated container build engine & validation harness
FR-19.6: Epic 19 - Container runtime health checks, dependency & toolchain verification sweeps
FR-19.7: Epic 19 - Ephemeral sandbox deployment, smoke testing & log streaming UI

## Epic List

### Epic 1: Interactive Agent Canvas
Empower product and design personas to instruct agents and visualize workflows in real-time through a centralized workspace.
**FRs covered:** FR-2.1, FR-2.2, FR-2.6

### Epic 2: Project Collaboration & Pipeline Tracking
Provide visibility into the end-to-end delivery pipeline and live multi-agent collaboration activity.
**FRs covered:** FR-1.1, FR-1.2, FR-1.3, FR-1.5

### Epic 3: Enterprise Context Integration
Connect the workspace to internal developer platforms and historical codebases to ensure generated designs align with enterprise standards.
**FRs covered:** FR-3.1, FR-3.2

### Epic 4: Repository Synchronization
Eliminate hand-off fragmentation by directly checking all generated artifacts into the target git repository.
**FRs covered:** FR-4.1, FR-4.2, FR-4.3, FR-4.4, FR-4.5, FR-4.6

### Epic 5: Parallel Dual-Engine AI Orchestration
Eliminate single-framework risk by establishing concurrent execution paths inside backend middleware, proxying inferences to either legacy GenKit ADK or a high-performance downstream Python Antigravity SDK service.
**FRs covered:** FR-5.1, FR-5.2, FR-5.3

### Epic 6: Compliance
Ensure complete platform auditability, end-to-end data logging compliance, and secure network isolation boundaries.
**FRs covered:** Security Requirement (Audit Trail)

### Epic 7: Automated Onboarding & Project Discovery
Empower teams to seamlessly onboard existing (brownfield) codebases into APF by autonomously scanning stacks, auditing legacy specs, and providing a gated interactive ingestion selection interface.
**FRs covered:** FR-7.1, FR-7.2, FR-7.3, FR-7.4, FR-7.5

### Epic 8: File Upload & Context Integration (MVP)
Allow users to upload local text files to `docs/spec/context/` to provide additional context to BMAD agents during chat interactions.
**FRs covered:** FR-2.3, FR-2.4, FR-2.5

### Epic 9: Interactive Product Onboarding Tour
Enhance the existing onboarding tour to cover newly created sections (Secrets, Project Discovery, File Upload) and implement resilient, context-aware navigation controls (Skip Section, Auto-Resume, State Recovery).
**FRs covered:** FR-9.1, FR-9.2, FR-9.3, FR-9.4, FR-9.5, FR-9.6, FR-9.7, FR-9.8

### Epic 10: Multi-Tenancy Isolation & Management
Enable secure multi-organization workspaces where customer teams operate independently without context or data leakage across organizational boundaries.
**FRs covered:** FR-8.1

### Epic 11: Identity and Access Management
Secure the platform infrastructure, API endpoints, and portals by implementing Google Identity Platform, validating IAP/OIDC tokens, and providing administrative portals for user management and organization bootstrapping.
**FRs covered:** FR-11.1, FR-11.2, FR-11.3, FR-11.4, FR-11.5, FR-14.1

### Epic 12: Administrative NFS Storage & Infrastructure Governance / Operations
Empower platform and system administrators to maintain storage governance, automate long-running workspace housekeeping, prevent runaway NFS volume consumption, isolate multi-tenant workspace quotas, and execute batch maintenance tasks reliably via dedicated Cloud Run Jobs and Cloud Scheduler without degrading API server performance.
**FRs covered:** FR-12.1, FR-12.2, FR-12.3, FR-12.4, FR-12.5, FR-12.6, FR-12.7, FR-12.8, FR-12.9

### Epic 13: Mobile Application Development Experience
Deliver a responsive, touch-optimized mobile experience enabling developers and product managers to interact with BMAD agents, view and edit specs, inspect visual canvas layouts, and trigger builds using mobile devices.
**FRs covered:** FR-13.1, FR-13.2, FR-13.3, FR-13.4

### Epic 14: Performance & Real-Time Latency Testing Infrastructure
Establish automated k6 load testing scripts in `{project-root}/perf/`, synthetic multi-tenant test data seeding fixtures, distributed load harnesses simulating 200 concurrent active users, real-time WebSocket/SSE latency benchmarking, and CI/CD performance regression gates.
**FRs covered:** NFR-1.1, NFR-1.2, NFR-1.4, NFR-1.5, NFR-1.7, NFR-3.1, AR-PERF-1, AR-PERF-2

### Epic 15: External Integrations & Feature Flag Management
Provide a centralized interface and secure configuration management for external service integrations, AI model providers (including Gemini API key entry), third-party developer tool connectors, and application-level dynamic feature flags.
**FRs covered:** FR-15.1, FR-15.2, FR-15.3

### Epic 16: Zero-Trust Path Resolution, Tool Parity & Native BMad TOML Configuration Architecture
Unify agent toolsets, eliminate fragile in-flight regex config/markdown mutations in favor of native BMad TOML layer resolution, provide script validation guardrails in `uv_runner`, and consolidate path safety into a single-pass zero-trust boundary architecture.
**FRs covered:** FR-16.1, FR-16.2, FR-16.3, FR-16.4, FR-16.5, FR-16.6, NFR-16.1, NFR-16.2

### Epic 17: Multi-Provider Email Notification Platform & Custom Notification Provider (CNP) Integration
Establish a modular, provider-agnostic notification adapter subsystem supporting Custom Notification Provider (CNP), local Console, and Mock providers, enabling secure, authenticated, and observable email notification dispatch with automated infrastructure provisioning.
**FRs covered:** FR-17.1, FR-17.2, FR-17.3, FR-17.4, FR-17.5, NFR-17.1, NFR-17.2, NFR-17.3

### Epic 18: Unified File Upload, Multi-Environment Artifact Management, Multimodal Chat Context & File Explorer Management
Enable users to upload text files, documents, and images via chat prompts and the File Explorer project tree. Chat uploads utilize a pluggable ArtifactService (GCS in Cloud via GCS_ARTIFACT_BUCKET, Local FS in dev via mise.toml) with text and multimodal multipart prompt ingestion. In chat view, sessions with artifacts display an expandable chevron and artifact dropdown. Image artifacts preview in a dedicated image viewer without Monaco/approval buttons. File Explorer folder nodes reveal `+` (upload to folder), Refresh, and Delete on hover, with a `+` root upload button in the header.
**FRs covered:** FR-18.1, FR-18.2, FR-18.3, FR-18.4, FR-18.5, FR-18.6, FR-18.7, FR-18.8

### Epic 19: Development Sandbox Builder & Runtime Verification Harness
Empower developers and autonomous agents to trigger an interactive sandbox build workflow from the Implementation Tab, dynamically synthesize deterministic containerized runtime specifications (Dockerfile) by scanning project specifications and context files (project-context.md, AGENTS.md, architecture.md), review/edit detected runtime dependencies in a structured table, and automatically build, validate, test, and ephemerally deploy the resulting sandbox container to verify runtime dependencies, toolchains, and environment health.
**FRs covered:** FR-19.1, FR-19.2, FR-19.3, FR-19.4, FR-19.5, FR-19.6, FR-19.7, NFR-19.1, NFR-19.2, NFR-19.3, UX-DR-19.1, UX-DR-19.2, UX-DR-19.3


## Epic 1: Interactive Agent Canvas

Empower product and design personas to instruct agents and visualize workflows in real-time through a centralized workspace.

### Story 1.1: Interactive Node Grid Layout

As a PM/UX Designer,
I want to view and interact with my project workflows in an interactive node grid layout,
So that I can visualize decision points and status triggers in real-time.

**Acceptance Criteria:**

**Given** the user navigates to the APF workspace
**When** the canvas module loads
**Then** the UI displays an interactive node workflow layout within 2 seconds
**And** the layout renders decision points and status triggers
**And** state updates reflect in under 500ms
**And** layout triggers synchronize across active viewers within 100ms

### Story 1.2: Agent Sidebar Chat

As a PM/UX Designer,
I want a sidebar chat interface to instruct agents visually or narratively,
So that I can collaborate directly with AI assistants during my workflow.

**Acceptance Criteria:**

**Given** the user is in the APF workspace
**When** they open the sidebar chat and submit an instruction
**Then** the system captures the interaction in the audit log
**And** the agent responds with a latency of less than 2 seconds

### Story 1.3: Secure Canvas Workspace Infrastructure

As a System Architect,
I want the canvas and chat operations to run within strict organization VPC bounds and support concurrent usage,
So that enterprise data remains secure, isolated, and highly available.

**Acceptance Criteria:**

**Given** a user initializes a workspace session
**When** the canvas and chat modules start
**Then** all operations are strictly confined within the organization VPC bounds with complete network isolation
**And** the system supports at least 50 concurrent users per active workspace cluster without degradation

### Story 1.4: Contextual Artifact Sidebar

As a User (PM/UX/Engineer),
I want a collapsible right-hand pop-out menu for artifact navigation,
So that I can easily browse artifacts related to my active persona and reference artifacts from other disciplines without cluttering the screen.

**Acceptance Criteria:**

**Given** the user is in the APF workspace
**When** they click a collapsed persona icon on the right edge
**Then** the panel dynamically expands
**And** it displays a list of artifacts collected by the specific active persona
**And** it allows the user to view collections of artifacts from other personas (UX, Engineering) for reference.

### Story 1.5: Cancel Chat Message

As a User (PM/UX/Architect),
I want to be able to cancel a chat message that is currently being processed by an agent,
So that I can stop the agent from continuing to process that message if I realize it was incorrect or no longer needed.

**Acceptance Criteria:**

**Given** the user has sent a chat message and the agent is currently processing it (typing indicator/thought stream is active)
**When** the user clicks the "Cancel" (stop) button associated with the active message
**Then** the client MUST immediately send a cancellation request to the backend API
**And** the backend orchestrator MUST propagate the cancellation to the active execution context (e.g., cancelling the Go `context.Context` or aborting the running background task)
**And** the active GenAI inference and any running tool executions for that message MUST be terminated within 1 second
**And** the UI MUST transition the message state to "Cancelled" and stop all loading/typing animations
**And** the system MUST log the cancellation event in the audit trail (containing Principal, Action: Cancel Message, and Timestamp)
**And** the chat interface MUST remain fully responsive, allowing the user to submit new instructions immediately.

## Epic 2: Project Collaboration & Pipeline Tracking

Provide visibility into the end-to-end delivery pipeline and live multi-agent collaboration activity.

### Story 2.1: Idea-to-Production Pipeline Tracking Bar

As a Product Manager,
I want a visual pipeline tracking bar on my dashboard,
So that I can see the continuous state progression of my project from Ideation through Architecture.

**Acceptance Criteria:**

**Given** the user is viewing a project dashboard
**When** the project state advances (e.g., Ideation -> Architecture)
**Then** the UI displays a visual pipeline tracking bar
**And** the tracking bar accurately reflects the current state across the pipeline
**And** the dashboard components maintain interactive load times

### Story 2.2: Live Cross-Agent Activity Feed

As a Team Member,
I want a real-time collaboration activity feed in my workspace,
So that I can monitor cross-agent validation requests and live status updates.

**Acceptance Criteria:**

**Given** the user is observing the project workspace
**When** active agents perform validation or issue status updates
**Then** the activity feed broadcasts these events live
**And** the platform maintains 99.9% uptime to support continuous monitoring

### Story 2.3: User-Level Secrets Management Settings Screen

As a User (PM, UX Designer, Architect, Developer),
I want a dedicated user-level settings screen to securely input and manage my personal API keys and authorization tokens,
So that I can authorize third-party MCP integrations (e.g., Figma, Lucidchart, Stitch, Draw.io) without exposing my credentials globally or violating organization isolation.

**Acceptance Criteria:**

**Given** a user accesses the settings screen
**When** they input personal API keys or tokens (Figma, Lucidchart, Stitch, Draw.io)
**Then** the backend MUST store these tokens securely per-user bound to their authenticated IAP identity profile (`claims.Subject`) with explicit key versioning headers (`v1:`) to support key rotation and prevent decryption lockout
**And** the orchestrator MUST dynamically inject them into the MCP execution context at runtime using standard JWE compact serialization (`eyJ...`) or secure token exchange (utilizing pre-shared dev key pairs for dev/test fallback)
**And** all backend HTTP handlers MUST utilize clean struct-based dependency injection without package-level global variables
**And** the UI MUST mask sensitive token strings server-side (`••••••••••••••••••••••••••••••••`) without plaintext reveal toggles
**And** the UI MUST provide explicit actions to update, validate (via secure preflight diagnostic API calls supporting JSON array bodies and strict redirect policies), and revoke keys using Firestore transactions to prevent lost updates
**And** the settings modal MUST be accessible via a dedicated `Icons.key` "User Secrets" button consistently across both the Project Overview quick actions bar and the Workspace Detail right rail

### Story 2.4: Project-Level App Integration Dynamic Settings

As a User (PM, Architect, Developer),
I want a project-level integration settings section that dynamically loads a custom configuration form once an app integration is enabled,
So that I can configure app-specific properties (such as repository connections and path definitions) and save them directly to the project's persistent storage.

**Acceptance Criteria:**

**Given** the user is on the Project Detail/Overview screen,
**When** an integration (such as GitHub) is configured and enabled,
**Then** the UI MUST dynamically load and inject a custom configuration form specific to that integration from a dedicated component folder (`ui/lib/screens/project/integrations/`),
**And** the dynamic form for **GitHub** MUST render input text fields for "Specification Directory Path", "Base Git Branch", and "Spec Branch Prefix",
**And** the GitHub form MUST render a searchable dropdown list displaying all remote repositories the user has access to based on their authenticated integration credentials, including a responsive text search input to filter repositories,
**And** clicking the "Save" button on the dynamic form MUST trigger a Firestore transaction/batch update on the backend to record these settings under the `app_integrations` array inside the project collection,
**And** all integration forms MUST adhere strictly to Material 3 design patterns with responsive, accessible form layouts.

## Epic 3: Enterprise Context Integration

Connect the workspace to internal developer platforms and historical codebases to ensure generated designs align with enterprise standards.

### Story 3.1: Figma Sync Adapter

As a UX Designer,
I want to sync design tokens and mockups via the Figma Web API,
So that my visual requirements are fetched directly into the APF workspace for agent context.

**Acceptance Criteria:**

**Given** the user triggers a golden path sync for design files
**When** the Figma integration adapter runs
**Then** the system connects utilizing Figma Web API v1+
**And** standard files complete synchronization in under 10 seconds
**And** the imported data calibrates the UX agent context within 5 minutes

### Story 3.2: Native IDP Schema Connector

As a System Architect,
I want the APF workspace to securely fetch schema metadata from our Internal Developer Platform,
So that generated code and architecture designs conform to existing enterprise standards.

**Acceptance Criteria:**

**Given** the workspace needs to hydrate architectural context
**When** the connector triggers a metadata pull
**Then** the plug endpoints fetch IDP schema metadata
**And** the fetch utilizes secure HTTPS/TLS 1.3 connections
**And** the context is successfully updated for agent usage within 5 minutes

### Story 3.3: AntiGravity JSON-RPC Hydration Outpost

As a Platform Engineer,
I want the configuration to feed AntiGravity workspace packages,
So that agent context is properly aligned natively in developer environments.

**Acceptance Criteria:**

**Given** the APF has generated workspace structures and schemas
**When** the hydration outpost distributes to AntiGravity
**Then** the configuration feeds are packaged compliantly using JSON-RPC standard endpoints
**And** the operation runs securely without exposing information publicly

## Epic 4: Repository Synchronization

Eliminate hand-off fragmentation by directly checking all generated artifacts into the target git repository.

### Story 4.1: Git State Sync Tracker

As a Product Manager,
I want the system to automatically check-in generated specifications (e.g., `design-artifacts/`) to the local git branch via the API tier,
So that artifact generation is perfectly synchronized with the codebase upon state save anchors.

**Acceptance Criteria:**

**Given** the user generates or updates specifications (e.g., in `design-artifacts/`)
**When** a state save anchor is triggered (e.g., when the API tier receives a file write notification)
**Then** the API tier automatically adds the file, commits the changes to the local git branch, and pushes them to the remote git branch
**And** all generated specifications are successfully versioned

### Story 4.2: Tabbed Workspace Explorer: Artifacts and File Explorer Views

As a Developer/User,
I want a tabbed navigation in the workspace explorer section with "Artifacts View" and "File Explorer View" tabs,
So that I can switch between viewing high-level generated specs/artifacts and the underlying project directory tree.

**Acceptance Criteria:**

**Given** the user is in the project workspace details view
**When** the workspace Explorer section renders
**Then** it displays two distinct tabs: "Artifacts" and "File Explorer"
**And** the "Artifacts" tab is selected by default, displaying the registered artifacts list/rail
**When** the user clicks on the "File Explorer" tab
**Then** the UI switches the explorer view to render the FileExplorer component showing the project filesystem
**And** tab switching is instantaneous (latency < 100ms) without triggering full-page or full-workspace reload
**And** both tabs maintain their scroll and expansion state when switching back and forth during the active session.

### Story 4.3: Automated Pull Request Handoff and Remote VCS Integration

As a Product Manager / Stakeholder,
I want the system to dynamically sync local specifications branches with the remote VCS repository and auto-create a Pull Request via the API tier,
So that approved design artifacts and requirements are cleanly staged for development under standard peer-review governance.

**Acceptance Criteria:**

**Given** a user triggers the "Approve & Handoff" action on the visual canvas popup,
**When** the repository synchronization endpoint runs on the API tier,
**Then** the API tier MUST push the local branch changes to the remote VCS feature branch using the authenticated human user's OIDC/OAuth credentials,
**And** the API tier MUST automatically create a standard Pull Request (PR) on GitHub against the primary branch,
**And** the system MUST enforce a **Pull Request Review Gate** for secure human-in-the-loop synchronization,
**And** the UI MUST cleanly display progress indicators and connection confirmations via Snackbars.

### Story 4.4: Git-to-Workspace Spec Re-Hydration and Edit Sync

As a User (PM, UX Designer, Architect, Developer),
I want to re-hydrate and edit a finalized specification by dynamically pulling the latest Git commits back into my workspace via the API tier,
So that I can fluidly iterate and refine approved requirements through natural language agent dialogues without manual copy-paste steps.

**Acceptance Criteria:**

**Given** a user clicks "Edit" on a finalized specification in the visual workspace,
**When** the edit session is initialized,
**Then** the API tier MUST pull the latest commit from the remote VCS branch using the caller's validated OIDC/OAuth credentials to ensure access authorization,
**And** the API tier MUST automatically stage and decode these raw assets directly into the ephemeral workspace runtime environment (Google Cloud Storage bucket paths and database maps),
**And** the system MUST transition the artifact's status indicator back to `In Progress` in the tracking log,
**And** the canvas modal and chat pane MUST immediately reload, reflecting the newly pulled and staged spec state dynamically.

### Story 4.5: Git Commit Safety Guardrails

As System,
I want to prevent the introduction of extraneous files that might get generated on the fly while running the agents,
So that the repository remains clean and only contains authorized specifications.

**Acceptance Criteria:**

**Given** a git commit operation is triggered on the API tier
**When** staging and committing files
**Then** the API tier MUST check that code is only being added to spec-approved locations (e.g., `docs/spec/` or `.agents/`)
**And** the API tier MUST verify that a `.gitignore` file is present in the root of the codebase prior to committing
**And** if `.gitignore` is missing in the root, it MUST be automatically created
**And** the API tier MUST ensure that the `.agents` directory is NOT ignored in the `.gitignore` file, automatically removing any blocking `.agents` ignore patterns if present

### Story 4.6: File-to-Database Artifact Synchronization

As a User,
I want to trigger a force synchronization of artifacts recorded by the system in the database to match the files currently present on the workspace file system,
So that the database and UI accurately reflect the true state of the files, avoiding stale listings, double loads, or crashes when files are modified or deleted directly on disk.

**Acceptance Criteria:**

**Given** the user is in the project workspace
**When** they trigger the "Force Sync Artifacts" action
**Then** the backend MUST scan the designated workspace directories (`docs/spec/` and `design-artifacts/`) on the local filesystem
**And** the system MUST compare the filesystem assets with the artifact records stored in the Firestore database
**And** any stale database records representing files that no longer exist on disk MUST be removed
**And** any new files found on disk that are not registered in the database MUST be created as new artifact entries with correct metadata (filename, path, timestamp)
**And** any files whose hash or last modified time differs from the database records MUST have their database metadata updated
**And** the operation MUST run inside a database transaction or batch to ensure atomicity
**And** the UI MUST display a success Snackbar and immediately refresh the artifact navigation rail.

### Story 4.7: Centralize Git and Version Control in API Tier

As a System Architect,
I want to migrate and centralize all Git operations (pull, commit, push, merge) from the AI tier to the API tier,
So that the downstream agent inference services are completely decoupled from version control and repository management libraries.

**Acceptance Criteria:**

**Given** the new centralized architecture
**When** agents are running
**Then** the AI tier MUST NOT perform any git commands, and its file-saving git-commit interceptor plugins MUST be disabled
**And** the API tier MUST trigger `git pull` on the shared NFS mount before launching agents to ensure workspaces are up-to-date
**And** the API tier MUST trigger `git commit` and `git push` on the shared NFS mount at the end of agent runs for any created/modified specifications.

### Story 4.8: On-Demand Recursive File Explorer Component and API

As a Developer/User,
I want a rich, interactive FileExplorer component that fetches and renders a recursive project directory tree on-demand (one level at a time) with file-type-specific colorful icons,
So that I can easily browse and navigate the workspace filesystem structure efficiently without overloading the network or frontend memory.

**Acceptance Criteria:**

**Given** the user has selected the "File Explorer" tab
**When** the FileExplorer component is rendered
**Then** the UI calls the backend directory-listing API to fetch the root-level files and folders of the project workspace
**And** the component renders the root directory level, displaying folders first followed by files in alphabetical order
**And** folders are displayed with a folder icon and an expand/collapse arrow indicator, and files are displayed with colorful icons representing their specific file extension (e.g. orange Python icon for `.py`, blue Go icon for `.go`, yellow JS/TS icon, red PDF icon, etc.)
**When** the user clicks on a folder node to expand it
**Then** the UI dynamically calls the backend directory-listing API passing the folder's absolute/relative path to fetch its immediate children (1 level deep)
**And** displays a loading spinner/indicator inline next to the folder name while the request is in flight
**And** renders the child nodes nested under that folder once loaded, updating the arrow to an expanded state
**When** the user clicks an already expanded folder
**Then** the UI collapses the folder node and hides its children from the view without making any backend API requests
**And** the backend directory-listing API MUST restrict directory traversal to the user's workspace path bounds, returning a `403 Forbidden` error if any path outside the workspace is requested
**And** the backend API responses are cached or optimized to achieve P95 response latency < 150ms.

### Story 4.9: Transactional Git Workspace Snapshotting & Local State Preservation

As a Developer or System User,
I want the Git client engine to capture local workspace snapshots (stashing uncommitted files and creating immutable pre-sync backup refs) before executing remote synchronization,
So that my local working tree and HEAD commit state are 100% guaranteed to be preserved and restored if any downstream rebase or fetch step fails.

**Acceptance Criteria:**

**Given** a remote sync operation is initiated on a local repository clone
**When** uncommitted local changes exist in the working directory
**Then** the engine MUST automatically stash uncommitted changes into an isolated stash reference (`refs/foundry/stash/<userId>`) before mutating refs
**And** the engine MUST create an immutable backup reference pointing to the local HEAD commit SHA (`refs/foundry/backups/<branch>-<timestamp>`)
**And** if any rebase/fetch error occurs, the engine MUST execute `rollbackLocalState` performing a hard reset to the pre-sync backup ref and restoring stashed dirty changes
**And** on success, the engine MUST pop the stashed changes back into the worktree and clean up the temporary backup ref.

### Story 4.10: Dry-Run Rebase Engine & Collision Detection

As a Developer or System User,
I want the Git client engine to test rebase operations on an isolated temporary shadow branch (`refs/foundry/shadow-rebase-<uuid>`),
So that merge conflicts are detected without mutating the active working branch or leaving conflict markers (`<<<<<<< HEAD`) in workspace files.

**Acceptance Criteria:**

**Given** local HEAD has diverged from remote tracking commit
**When** executing rebase check
**Then** the engine MUST perform the cherry-pick sequence in an isolated shadow branch (`refs/foundry/shadow-rebase-<uuid>`)
**And** if a merge conflict occurs during shadow cherry-pick, `dryRunRebase` MUST catch the conflict immediately, abort the shadow rebase, delete the shadow branch, and return a `MergeConflictError` containing the list of conflicting files
**And** the active local working tree MUST remain completely untouched at its pre-sync HEAD commit without writing conflict markers (`<<<<<<< HEAD`) to disk
**And** if the shadow rebase succeeds with zero conflicts, the engine MUST fast-forward the main local branch to the rebased HEAD commit and delete the shadow branch.

### Story 4.11: Structured Git Sync Response & GitHub/GitLab Integration

As an API Consumer or Frontend Client,
I want the workspace Git sync handlers (`POST /api/v1/projects/{id}/git/pull` and `POST /api/v1/projects/{id}/git/sync`) to return a structured checklist response containing dynamic branch names, step progress, and GitHub PR resolution links,
So that the user interface can display progress and guide users to resolve conflicts via GitHub/GitLab PRs or local IDE workflows.

**Acceptance Criteria:**

**Given** a call to `POST /api/v1/projects/{id}/git/pull` or `/git/sync`
**When** the operation finishes
**Then** the handler MUST return HTTP `200 OK` with a structured `GitSyncChecklistResponse` payload
**And** the JSON response MUST contain `status` (`SUCCESS` | `CONFLICT_DETECTED` | `ERROR`), `message`, `dynamicBranches` (`baseBranch`, `specBranch`, `backupRef`, `shadowBranch`), and an array of `steps` (`id`, `label`, `status`, `detail`)
**And** if a merge conflict occurs (`status: "CONFLICT_DETECTED"`), the system MUST ensure local commits are pushed to the spec branch and populate `conflictInfo.pullRequestUrl` with the direct GitHub/GitLab PR URL
**And** the payload MUST include `conflictingFiles`, `cliInstructions`, and a clear `nextStepRecommendation` string
**When** calling `GET /api/v1/projects/{id}/git/local-changes` to calculate files to stage/sync, the API tier MUST first perform a `git pull` / remote sync on the `main` (`baseBranch`) branch so that `main` is synchronized with the remote, and MUST return BOTH uncommitted local worktree modifications (`git status`) AND committed changes on `specBranch` that are not yet in `baseBranch` (`git diff <base>...<spec>`), deduplicating by file path.

### Story 4.12: Interactive Git Sync Checklist Modal UI & Conflict Resolution

As a Product Manager, UX Designer, or Developer,
I want an interactive checklist modal UI to display during remote Git synchronization,
So that I can see the complete step progress, dynamic branch/ref names created behind the scenes, where the process got stuck, and receive clear guidance on how to resolve merge conflicts.

**Acceptance Criteria:**

**Given** a user triggers Git sync or pull in the UI
**When** the operation finishes or encounters a conflict
**Then** the UI MUST display the `GitSyncModal` checklist modal
**And** it MUST render dynamic branch badges for `Base Branch`, `Spec Branch`, `Backup Ref`, and `Shadow Branch`
**And** each execution step MUST display a status icon: `completed` (green checkmark), `failed` (red cross), `skipped` (gray dash), or `in_progress` (spinner)
**And** on `CONFLICT_DETECTED`, the modal MUST highlight the stuck step, display conflicting files, present a prominent `[ Resolve Conflict on GitHub PR ]` primary button, and provide a collapsible block with copy-pasteable CLI commands.

### Story 4.13: Unified File Explorer with Phase-Associated Artifacts & Status Indicators

As a Product Manager, UX Designer, Architect, or Developer,
I want the File Explorer to serve as the unified navigation surface for all generated artifacts and workspace files, automatically filtering to only Firestore-tracked artifacts associated with the active phase when "Show All Files" is unchecked while displaying live status indicators (`In Progress`, `Complete`, etc.), and revealing the full repository directory tree when "Show All Files" is checked,
So that I can eliminate redundant sidebar accordions, clearly see what artifacts belong to my current workflow phase along with their lifecycle state, and seamlessly transition between high-level phase deliverables and raw codebase inspection.

**Acceptance Criteria:**

**Given** the user is viewing a project workspace in any lifecycle phase (`ONBOARDING`, `ANALYSIS`, `PLANNING`, `SOLUTIONING`, `IMPLEMENTATION`)
**When** the Workspace Explorer sidebar renders
**Then** the standalone `GENERATED ARTIFACTS` section MUST be removed, leaving `CHAT SESSIONS` and `FILE EXPLORER` (plus `WORKFLOWS` in Implementation)
**And** the `FileExplorer` section MUST have a "Show All Files" switch that is unchecked (OFF) by default
**When** "Show All Files" is unchecked (default mode)
**Then** the header MUST display `"<PhaseName> Files"` (e.g. `"Planning Files"`, `"Solutioning Files"`)
**And** the list MUST display ONLY the artifacts tracked in Firestore for the project that match the active phase (`a.phase === activePhase`)
**And** each artifact item MUST display its file-type icon, file name, parent path subtitle (if nested), and a visible status indicator badge/dot matching its Firestore status:
  - `In Progress`: Amber badge (`text-amber-600 bg-amber-500/10 border-amber-500/30`)
  - `Complete`: Emerald badge (`text-emerald-600 bg-emerald-500/10 border-emerald-500/30`)
  - `Staged for Review`: Blue badge (`text-blue-600 bg-blue-500/10 border-blue-500/30`)
  - `Not Started`: Slate badge (`text-slate-500 bg-slate-500/10 border-slate-500/30`)
**And** if no artifacts exist for the active phase, a clean contextual empty state message MUST be rendered (e.g., `"No files tracked for Planning phase."`)
**When** the user clicks on any artifact in the phase-filtered list
**Then** the Canvas preview pane (`#tour-workspace-canvas`) MUST open displaying the artifact content
**And** the selected artifact item MUST receive active visual selection styling
**When** the user hovers over an artifact item and clicks the delete action (if permitted)
**Then** a confirmation dialog MUST display, and upon confirmation, the artifact MUST be deleted via the API and removed from the list
**When** the user toggles "Show All Files" ON
**Then** the header MUST display `"All Files"`
**And** the File Explorer MUST switch to the hierarchical directory tree view showing all workspace filesystem folders and files
**And** any file leaf node matching a tracked Firestore artifact MUST display its lifecycle status indicator dot.

### Story 4.14: Automated Pre-Commit Safety Checks & Gitignore Lifecycle Management

As a Developer or Platform Engineer,
I want the system to execute an automated pre-commit check and enforce `.gitignore` lifecycle rules prior to staging and committing changes,
So that transient files and scratch directories (such as `.agents/.tmp/`) are never committed to the source tree, while ensuring necessary agent configurations (`.agents/`) remain tracked and any existing `.gitignore` contents are preserved.

**Acceptance Criteria:**

**Given** a Git commit or staging operation is initiated (via git hooks, API tier commit pipeline, or workspace sync handlers)
**When** validating workspace files before creating a commit
**Then** the system MUST execute a pre-commit validation check verifying that files scheduled for commit comply with repository safety policies and do not include restricted or temporary files
**And** the system MUST verify whether a `.gitignore` file exists in the root of the project repository
**And** if `.gitignore` is not present in the root, the system MUST automatically create a new `.gitignore` file populated with standard project exclusions and `.agents/.tmp/`
**And** if `.gitignore` is already present in the root, the system MUST inspect its contents to determine whether the required exclusion patterns (including `.agents/.tmp/`) are already present
**And** if any required exclusion pattern (specifically `.agents/.tmp/`) is missing from the existing `.gitignore`, the system MUST append the missing pattern(s) to the end of the file
**And** the system MUST strictly preserve all existing rules, comments, whitespace, and custom configurations in the existing `.gitignore` file without overwriting or truncating it
**And** the system MUST ensure that the parent `.agents/` directory is NOT excluded in `.gitignore`, ensuring agent skills and configurations remain version-controlled while `.agents/.tmp/` is ignored
**And** if any file in the staged change set violates exclusion rules, the pre-commit check MUST abort the commit and return a clear, descriptive error indicating the offending file(s).

## Epic 5: Parallel Dual-Engine AI Orchestration

Eliminate single-framework risk by establishing concurrent execution paths inside backend middleware, proxying inferences to either legacy GenKit ADK or a high-performance downstream Python Antigravity SDK service.

### Story 5.1: Python Antigravity Chat Inference Engine

As a User,
I want a Python-based execution core utilizing the advanced `google-antigravity` SDK,
So that chat inferences are processed in parallel with baseline pipelines, delivering high-fidelity streaming responses.

**Acceptance Criteria:**

**Given** downstream requests from the Go gateway proxy,
**When** executing prompts,
**Then** the Python service (under `/agy`) MUST process the conversational dialogue using native `google-antigravity` inference libraries,
**And** the agent response stream MUST route back through the Go gateway to client panels within a p95 latency under **2.0 seconds**.

### Story 5.2: Go Dynamic Reverse-Proxy AI Gateway

As a System Architect,
I want the primary Go orchestrator layer to intercept and securely proxy request payloads downstream to the Python-based AI core,
So that standard enterprise OIDC credentials verification, secrets decryption, and logging controls are centrally enforced.

**Acceptance Criteria:**

**Given** the dynamic route config `integrationSettings.aiOrchestrator` is set to `agy-sdk`,
**When** chat, session, or tool REST/SSE events are triggered,
**Then** the front-facing Go API backend MUST verify OIDC identity tokens, pull and decrypt secrets (AES-256-GCM), and scrub JWT/PII enclaves,
**And** the orchestrator MUST reverse-proxy route the request downstream to the Python service using standard JSON-RPC serialization,
**And** response streams MUST cleanly return to the client panel with latency overhead under **50ms**.

### Story 5.3: Antigravity Dynamic Tool Execution Adapter

As a Product Owner,
I want visual tools synchronization adapters (Figma, Stitch, Draw.io) to support execution via the Python `google-antigravity` SDK core,
So that tool payloads edits are resolved with 100% functional parity in parallel with GenKit MPC systems.

**Acceptance Criteria:**

**Given** visual canvas mutations trigger tool sync handlers under `agy-sdk` configuration,
**When** executing tool parameters,
**Then** the Python backend MUST parse, validate, and execute adapter bindings natively,
**And** JSON transformations and database updates MUST achieve 100% functional parity with the ADK baseline.

### Story 5.4: ADK PostgreSQL Session DB Telemetry Aggregation Engine & GORM ORM Integration

As a System Architect,
I want the telemetry engine to query the ADK PostgreSQL Session Database directly (capturing all unabridged agent events, including transient tool executions and scratchpad turns), using a dedicated GORM ORM layer in the backend API service,
So that aggregate cost and consumption statistics (per user, per project, per organization, and specific/average per session) account for 100% of LLM token activity without information loss.

**Acceptance Criteria:**

**Given** an AI execution session running through the ADK runner or Python inference engine
**When** agent events (including transient turns, tool calls, and model completions) are persisted to the ADK PostgreSQL Session Database
**Then** the ADK runner event handler MUST record `prompt_tokens`, `completion_tokens`, `total_tokens`, and calculated cost on every event record in PostgreSQL
**And** the API tier MUST introduce a GORM PostgreSQL connection pool (`gorm.io/driver/postgres`) and ORM model mappings (`ADKSession`, `ADKEvent`) targeting the ADK Session Database
**And** the telemetry reporting API (`/api/v1/telemetry/usage`) MUST query the PostgreSQL database directly using optimized SQL/GORM aggregation queries to return:
  - Cost/Consumption per User (`userId`)
  - Cost/Consumption per Project (`projectId`)
  - Cost/Consumption per Organization (`organizationId`)
  - Specific and Average Cost/Consumption per Session (`sessionId`)
**And** transient Firestore chat messages MUST NOT be relied upon for token totals, ensuring 100% complete accounting across all intermediate agent invocations.

### Story 5.5: Sandboxed uv_run Tool Execution Engine for BMad Scripts & Python Utilities

As a Developer and AI Subagent (Analysis, Planning, Solutioning, Implementation),
I want a dedicated `uv_run` ADK tool that securely executes Python utility scripts (e.g. `memlog.py`, `brain.py`, `resolve_party.py`) in the project workspace sandbox,
So that BMad workflows and Python scripts execute natively during agent turns without generating intermediate `.sh` shell scripts or violating zero-trust container boundaries.

**Acceptance Criteria:**

**Given** an active AI agent session in any phase (Analysis, Planning, Solutioning, Implementation)
**When** the agent invokes the `uv_run` tool with `scriptPath` and optional `args`
**Then** the tool MUST dynamically substitute `{project-root}` with `projectRoot` and `{skill-root}` with `bmadRoot` across all path arguments
**And** the tool MUST enforce zero-trust security validation, confirming the resolved script path resides within either the user's project workspace mount or the configured BMad framework directory (`~/.apf/config/bmad/`)
**And** the tool MUST anchor execution in `projectRoot` (`cmd.Dir = projectRoot`) with Cloud Run optimized environment variables (`UV_SYSTEM_PYTHON=1`, `UV_CACHE_DIR=/tmp/uv-cache`, `HOME=/tmp`)
**And** the tool MUST enforce container-level concurrency control via a semaphore whose capacity matches the configured Cloud Run container concurrency (`CONTAINER_CONCURRENCY` / `CONCURRENCY`, defaulting to `80`)
**And** the tool MUST enforce a 30-second context execution timeout and truncate `stdout`/`stderr` exceeding 64KB
**And** on non-zero exit codes, the tool MUST return a structured `UVRunnerOutput` (`Success: false`, `ExitCode`, `Stderr`) without crashing the ADK tool runner, enabling agent self-healing
**And** the tool MUST detect newly created or modified files in `docs/spec` and register `file-write-notification:<uuid>` state keys in `ctx.State()` to trigger Firestore status transitions and local Git auto-commits
**And** the tool MUST emit structured `slog.Info` / `slog.Warn` audit logs capturing `scriptPath`, `exitCode`, `durationMs`, `userId`, `projectId`, and `createdFilesCount`
**And** the tool MUST utilize a mockable `CommandExecutor` interface supporting 100% deterministic unit testing in CI.

### Story 5.6: Cloud Run Container Image Packaging & Infrastructure Provisioning for uv Python Toolchain

As a Platform Engineer and DevOps Specialist,
I want the `apf-ai` Cloud Run container image and deployment manifests to package the `uv` toolchain, system Python 3 runtime, and pre-installed BMad dependencies,
So that `uv_run` script executions run with sub-second startup times, zero network dependency fetching, and guaranteed container-level concurrency alignment in staging and production environments.

**Acceptance Criteria:**

**Given** the `ai/Dockerfile` build pipeline
**When** the runtime container image is assembled
**Then** the base image MUST include a Debian-compatible Python 3 runtime (e.g. `python:3.12-slim-bookworm`), with `/bin/uv` and `/bin/uvx` copied from `ghcr.io/astral-sh/uv:latest` on the system `PATH`
**And** the container image MUST pre-install standard BMad script dependencies (`pyyaml`, `tomli`, `jinja2`) into the system Python environment
**And** the container image MUST define default environment variables `UV_SYSTEM_PYTHON=1`, `UV_CACHE_DIR=/tmp/uv-cache`, `HOME=/tmp`, and `PYTHONUNBUFFERED=1`
**And** the Cloud Run service manifest (`infra/cicd/manifests/ai.yaml`) MUST configure `containerConcurrency: 80` (or parameterized concurrency) under `spec`
**And** the Cloud Run service manifest MUST pass `CONTAINER_CONCURRENCY="80"`, `UV_SYSTEM_PYTHON="1"`, `UV_CACHE_DIR="/tmp/uv-cache"`, and `HOME="/tmp"` in the container `env` block
**And** building and running the container image locally with `docker build` MUST verify that `uv --version`, `python3 --version`, and `uv run python3 -c "import yaml, tomli; print('ok')"` exit with code 0.

## Epic 6: Compliance

Ensure complete platform auditability, end-to-end data logging compliance, and secure network isolation boundaries.

### Story 6.1: Audit Trail Event Capture

As a Compliance Officer,
I want all agent-workspace interactions to be rigorously captured and retained,
So that we have a complete audit log of system behavior and decision-making for at least 12 months.

**Acceptance Criteria:**

**Given** agent-human interactions occur in the workspace
**When** the events are logged to the audit module
**Then** the system captures the audit logs with 99.99% capture reliability
**And** the log entries meet the review retention specifications of precisely 12 months

## Epic 7: Automated Onboarding & Project Discovery

Empower teams to seamlessly onboard existing (brownfield) codebases into APF by autonomously scanning stacks, auditing legacy specs, and providing a gated interactive ingestion selection interface.

### Story 7.1: Project Lifecycle States & Dashboard Action Buttons

As a PM,
I want a project level status set to "New" upon project creation, showing an "Onboard" action button on its Projects Dashboard tile,
So that I am guided directly to the onboarding workflow when launching a brand new workspace.

**Acceptance Criteria:**

**Given** a new project is created in the database
**When** the record is initialized
**Then** the project status MUST be set to `New`
**And** the Projects Dashboard tile for this project MUST render a primary action button labeled `Onboard`
**And** clicking the `Onboard` button on a `New` project MUST route the user directly to the workspace's dedicated Onboarding Tab and transition the project status to `Onboarding`
**And** clicking `Onboard` on a project in the `Onboarding` state MUST resume the active onboarding session state dynamically.

### Story 7.2: Background Discovery Agent Stack Analysis

As a PM / System Architect,
I want an autonomous background agent to analyze my connected legacy repository,
So that my core language stacks and project frameworks are automatically classified.

**Acceptance Criteria:**

**Given** the PM clicks "Start Onboarding" in the Onboarding Tab
**When** the scan triggers
**Then** the orchestrator MUST spin up the autonomous `Discovery Agent` in the background
**And** the agent MUST automatically parse configuration files (e.g., `go.mod`, `package.json`, `pom.xml`, `Dockerfile`, `.tf`)
**And** the agent MUST classify the core runtime stack and environment frameworks
**And** the UI MUST display a live progress bar and publish real-time scan progress events to the Activity Stream
**And** if the background scan fails or is interrupted, the system MUST display a warning card but allow the PM to explicitly click "Complete Onboarding" to complete the onboarding as a minimal/new project.

### Story 7.3: Legacy Specification Audit & Ingestion Picker

As a PM,
I want an interactive directory tree picker displaying discovered legacy markdown specs,
So that I can check and select which requirements to ingest into active BMad agent contexts.

**Acceptance Criteria:**

**Given** the automated scan completes successfully
**When** displaying the Onboarding Tab
**Then** the UI MUST render the scannable Automated Repository Analysis Report listing discovered stacks and detected legacy documentation (`.md`/`.html` files)
**And** the UI MUST render an interactive directory tree picker allowing the PM to check checkboxes next to files to map them for active BMad agent ingestion
**And** any pre-existing BMad specifications (`prd.md`, `architecture.md`) MUST be flagged as `Existing BMad Spec` and pre-selected
**And** the tree picker MUST provide options toggles: `Keep Existing Spec (Read-Only)` and `Re-Reconcile & Overwrite` to govern file writing outcomes.

### Story 7.4: Explicit Onboarding Completion

As a PM,
I want an explicit action to complete the onboarding phase,
So that my project context is generated, agent customizations are packaged, and both are committed to the repository.

**Acceptance Criteria:**

**Given** the PM has finalized their file ingestion selections
**When** they click "Complete Onboarding"
**Then** the system MUST transition the project status to `Onboarded` in the database
**And** the system MUST auto-generate a canonical `project-context.md` file outlining stack discoveries and mapped legacy ingestion pointers
**And** the system MUST initialize the `.agents` directory in the repository root with default template configurations (skills, rules, workflows)
**And** the system MUST commit and push both the `project-context.md` and the `.agents` directory to the remote repository
**And** the Projects Dashboard tile action button MUST transition to show a standard `Open Workspace` trigger
**And** active workspace tools (Canvas, Chat) MUST remain accessible throughout the process and NOT be locked or gated by onboarding status


## Epic 8: File Upload & Context Integration (MVP)

Allow users to upload local text files to `docs/spec/context/` to provide additional context to BMAD agents during chat interactions.

### Story 8.1: Chat File Upload UI (MVP)

As a User,
I want to select and attach text files in the chat panel,
So that I can prepare them to be sent as context to the agent.

**Acceptance Criteria:**

**Given** the user is in the GUI Workspace chat panel
**When** they click the attachment button (clip icon)
**Then** a file picker opens allowing selection of local files
**And** the UI restricts selection to plain text extensions (e.g., `.txt`, `.md`, `.py`, `.go`, `.json`, `.yaml`, `.csv`)
**And** selected files are displayed as removable chips in the chat input area
**And** the UI displays an error warning if a selected file exceeds 10MB or is of an unsupported type.

### Story 8.2: Backend File Upload API (MVP)

As a Developer,
I want an API endpoint to upload files to the local workspace,
So that they are stored on disk for the agent to access.

**Acceptance Criteria:**

**Given** the user sends a message with an attached file
**When** the upload API is called with the file payload, `projectId`, and `sessionId`
**Then** the backend writes the file to the local workspace filesystem under `docs/spec/context/{filename}`
**And** the file is staged locally (not committed to Git immediately)
**And** the API returns a success response with the resolved file path.

### Story 8.3: Contextual Artifact Sidebar & DB Tracking (MVP)

As a User,
I want to see my uploaded context files in the sidebar,
So that I can verify which files are currently active in my workspace.

**Acceptance Criteria:**

**Given** a file is successfully uploaded to the workspace
**When** the upload completes
**Then** the backend registers the file in the Firestore database as a `context` type artifact associated with the project and session
**And** the Contextual Artifact Sidebar dynamically refreshes to display the file in a new "Context Files" section
**And** these context files do NOT affect or block the pipeline tracker gating logic.

### Story 8.4: Agent Context Ingestion (MVP)

As a Developer,
I want the agent to automatically load files from the context directory,
So that the agent can use them as context for its responses.

**Acceptance Criteria:**

**Given** there are files present in the `docs/spec/context/` directory in the workspace
**When** the agent starts a run/session
**Then** the agent automatically scans the `docs/spec/context/` directory
**And** the agent reads the text content of all files in that directory
**And** the agent appends this content to its active context window/memory for the duration of the run.

## Epic 9: Interactive Product Onboarding Tour

Enhance the existing onboarding tour to cover newly created sections (Secrets, Project Discovery, File Upload) and implement resilient, context-aware navigation controls (Skip Section, Auto-Resume, State Recovery).

### Story 9.1: Onboarding Tour State & Persistence (Backend/Firestore)

As a Developer,
I want to store and retrieve the user's onboarding tour progress in Firestore via API endpoints,
So that the user's tour state is persisted across sessions and devices.

**Acceptance Criteria:**

**Given** an authenticated user (via OIDC/IAP)
**When** they request their user profile
**Then** the API tier MUST return the user's `tour_status` state from Firestore
**Given** an authenticated user
**When** they update their tour progress (e.g., completing a step or section)
**Then** the API tier MUST update the user's profile document in Firestore under a `tour_status` map containing:
  - `completed` (boolean): Indicates if the entire tour is finished
  - `completed_sections` (list of strings): IDs of sections the user has completed or skipped
  - `last_active_step` (map): `{ "section_id": String, "step_index": Integer }`
**Given** a request to update `tour_status`
**When** the database write operation fails
**Then** the API MUST return a standard error response (e.g., `500 Internal Server Error`) and preserve the existing database state
**Given** the Firestore read/write operations for `tour_status`
**When** executed in the backend
**Then** the API response latency MUST be less than 200ms for the 95th percentile

### Story 9.2: Tour Integration for Newly Created Sections (Frontend)

As a PM/UX Designer,
I want the onboarding tour to include steps for the newly created sections (Secrets, Project Discovery, File Upload),
So that users are introduced to these new features during the tour.

**Acceptance Criteria:**

**Given** the onboarding tour is active
**When** it guides the user to configure integration secrets
**Then** the UI MUST highlight the User Secrets settings button (key icon) in the Project Overview
**And** when the Secrets modal is opened, the tour MUST highlight the token input fields and explain the application-level encryption and masking behavior
**Given** the onboarding tour is active on the Dashboard
**When** a project is in the `New` or `Onboarding` state
**Then** the tour MUST highlight the `Onboard` button on the project tile and explain its role in launching repository discovery
**Given** the onboarding tour is active in the Onboarding Tab
**When** a discovery scan is triggered
**Then** the tour MUST point to the live progress log / Activity Stream
**And** upon scan completion, it MUST guide the user through the interactive directory tree picker (explaining the "Keep Existing" vs. "Overwrite" toggles) and the explicit "Complete Onboarding" button
**Given** the onboarding tour is active in the GUI Workspace chat panel
**When** explaining context integration
**Then** the tour MUST highlight the file attachment (clip icon) button and explain the supported file types and size limits (10MB)

### Story 9.3: Advanced Navigation: Skip Section and Skip Tour (Frontend)

As a User,
I want the ability to skip the entire current section of the tour or dismiss the tour completely,
So that I don't have to click "Next" repeatedly for features I am already familiar with.

**Acceptance Criteria:**

**Given** the onboarding tour is active and displaying a step popover
**When** the popover is rendered
**Then** it MUST display distinct buttons for `Skip Section` and `Skip Tour`
**And** the popover MUST display the current section name and step progress (e.g., *“Section 2 of 4: User Secrets - Step 1/3”*)
**Given** the user is viewing a tour step within a multi-step section
**When** the user clicks `Skip Section`
**Then** the tour MUST immediately dismiss the active overlay, mark the entire section as skipped (adding it to `completed_sections`), and transition to the first step of the next scheduled section
**Given** the user is on any step of the tour
**When** the user clicks `Skip Tour`
**Then** the tour MUST immediately close, restore the standard UI state, and trigger a background API call to update the user's `tour_status.completed = true` in Firestore to prevent subsequent auto-launches
**Given** the tour navigation buttons
**When** navigated via keyboard
**Then** they MUST support standard `Tab` focus and `Enter`/`Space` activation

### Story 9.4: Tour Resiliency & Action Gating (Frontend/Backend)

As a User,
I want the onboarding tour to persist my progress and resume from where I left off if it is interrupted,
So that I don't lose my place in the tour due to accidental navigation or browser refreshes.

**Acceptance Criteria:**

**Given** the user is in the middle of the tour
**When** they transition to a new step
**Then** the frontend MUST send a background update to the API tier to record the `last_active_step` in Firestore (p95 latency < 200ms)
**Given** the tour session was interrupted (e.g., browser refresh, logout, or navigation away)
**When** the user logs back in or returns to the page
**Then** the system MUST query Firestore for `last_active_step` and resume the tour overlay from that exact step
**Given** a tour step that gates progression on a specific user action (e.g., "Click the Key icon to open User Secrets")
**When** the user performs the required action (even if the tour overlay was closed/crashed)
**Then** the system MUST detect the action (via UI state listeners) and automatically advance the internal tour state to the next logical step
**And** the UI MUST display a pulsing highlight cue on the active target element to encourage completion

### Story 9.5: Context-Aware Auto-Start & Triggers (Frontend)

As a User,
I want the tour to automatically start when I access a new section for the first time,
So that I am guided through the features of that section exactly when I need them.

**Acceptance Criteria:**

**Given** a user logs in for the first time (`tour_status.completed == false` and `completed_sections` is empty)
**When** they land on the Project Dashboard / Overview
**Then** the frontend MUST automatically trigger the Dashboard tour section
**Given** a user clicks the "Replay Tour" or "Help" action in the header
**When** triggered
**Then** the frontend MUST launch the tour from the first step of the Dashboard section
**Given** the tour is in-progress (not fully completed)
**When** the user navigates to a new screen or tab (e.g., the Onboarding Tab)
**Then** the system MUST verify if the tour section corresponding to this screen is in `completed_sections`
**And** if it is NOT in `completed_sections`, the frontend MUST automatically start the tour from the first step of that screen's section
**Given** the user navigates to a screen
**When** the corresponding tour section is already marked as completed or skipped
**Then** the tour MUST NOT launch automatically

### Story 9.6: Seeded Showcase Project (Foundry Showcase) (Frontend/Backend)

As a User,
I want to be able to provision a pre-configured showcase project named "Foundry Showcase" on-demand from the Settings screen,
so that I can explore features and reference a real-world design setup scoped to my active organization.

**Acceptance Criteria:**

**Given** the user is on the Settings screen
**When** they view the left-hand navigation
**Then** the UI MUST display a new menu item labeled "Project Templates"
**Given** the user selects the "Project Templates" menu item
**When** the panel loads
**Then** the right-hand panel MUST display a section for the showcase project with a primary action button labeled "Provision Foundry Showcase"
**Given** the user clicks "Provision Foundry Showcase"
**When** the request is sent to the backend API (`POST /api/projects/provision-showcase`)
**Then** the UI MUST display a loading spinner and disable the button to prevent double submissions
**Given** the backend receives the provisioning request
**When** the seeder executes
**Then** it MUST read pre-configured project data and specs from an embedded directory (using Go `//go:embed`) deployed with the API tier
**And** it MUST seed the project, sessions, messages, and artifacts records into Firestore, dynamically scoping them to the user's active `organizationId` and generating a unique `projectId`
**And** it MUST extract and write the workspace spec files onto the project's filesystem path (e.g., `/projects/{showcaseProjectId}/docs/spec/`)
**Given** the backend returns a successful response
**When** the client receives the success status
**Then** the UI MUST show a success Snackbar notification, close the Settings modal, and refresh the Projects Dashboard to show the new "Foundry Showcase" project tile
**Given** the "Foundry Showcase" project tile is rendered on the Projects Dashboard
**When** the user views it
**Then** it MUST display correct metadata (status: `Onboarded`, tag: `Sample Project`)
**Given** the user opens the "Foundry Showcase" workspace
**When** they interact with the workspace
**Then** clicking on the tile MUST launch the GUI Workspace immediately, loading the seeded interactive canvas, sidebar chat history, and specs, allowing the user to fully edit, run agents on, and synchronize the project as an active, write-enabled workspace
**And** the provisioned showcase project MUST NOT be linked to any remote Git repository by default
**Given** the showcase project has already been provisioned
**When** the user views the "Project Templates" screen under Settings
**Then** the UI MUST display a "Restore to default state" link
**And** clicking this link MUST trigger a backend reset that deletes all modifications, overwriting Firestore records and workspace files to return the project to its initial seeded state.


## Epic 10: Multi-Tenancy Isolation & Management

### Story 10.1: Data Partitioning by Organization-ID Across All Storage Mediums

As a Platform Engineer,
I want the system to have the capability to partition data by organizationId across all data storage mediums including Firebase, Cloud Storage, and NFS,
So that customer teams operate within strict data isolation boundaries without cross-organization context bleed.

**Acceptance Criteria:**

**Given** a multi-organization workspace environment
**When** a read or write operation occurs across Firebase, Cloud Storage, or NFS
**Then** the underlying adapter MUST partition the data path or query exclusively by `organizationId`
**And** prevent any cross-organization access across all storage boundaries

### Story 10.2: Organization-Aware Non-Administrative (Data Plane) APIs

As an Enterprise Administrator,
I want all non-administrative (data plane) APIs in the middleware to be organization-aware,
So that cross-organization data access attempts are rejected before reaching orchestration enclaves.

**Acceptance Criteria:**

**Given** an incoming request to a non-administrative (data plane) API endpoint
**When** the request reaches the middleware
**Then** the middleware MUST extract and validate the `organizationId` scope from the request context
**And** return a `403 Forbidden` error if the context does not match the requested workspace organization scope


### Story 10.3: Organization Provisioning & Onboarding (Administrative API)

As a System Administrator,
I want to be able to create a new organization in the system,
So that enterprise customers can be securely provisioned with dedicated workspace and storage boundaries.

**Acceptance Criteria:**

**Given** an authenticated System Administrator session
**When** the administrator submits a new organization onboarding request
**Then** the administrative API MUST initialize a new organization record in Firestore under a central `organizations` collection
**And** allocate the corresponding organization storage bucket prefix and isolation boundaries
### Story 10.4: Organization Environment Configuration & User Management

As a Organization Administrator,
I want to be able to configure environment settings for the organization environment and add users to it,
So that my organization can customize workspace integrations and govern user access within our isolated organization scope.

**Acceptance Criteria:**

**Given** an authenticated Organization Administrator session
**When** the organization administrator submits environment configuration settings or adds a user email
**Then** the administrative API MUST update the organization's configuration profile and authorization sub-tree (`users`) in Firestore
**And** grant the invited user access restricted strictly to that organization's workspace and storage boundaries

### Story 10.5: Multi-Tenant Login & Organization Selection UI Flow

As a User,
I want the UI to dynamically resolve my organization when I log in and allow me to switch between multiple organizations I belong to,
So that I can access isolated workspaces and operate within correct security boundaries.

**Acceptance Criteria:**

**Given** a user is on the login page and enters their email
**When** the user clicks "Next" or submits the email
**Then** the UI MUST call the backend resolution API to check the user's organization configuration
**And** if the organization has SAML/SSO enabled, redirect the user to the configured Identity Provider (IdP)
**And** if the organization uses credentials, show the password field and customize the logo/branding for that organization

**Given** a user is successfully authenticated and belongs to multiple organizations
**When** the login process finishes
**Then** the UI MUST display an Organization Selection screen to choose the active organization
**And** store the chosen organization ID in the session/cookies

**Given** a user has selected an active organization
**When** the UI makes any API request
**Then** the UI client MUST automatically append the active organization ID to the request headers (using the `X-Organization-Id` header)

### Story 10.6: Active Organization Switcher & Multi-Tenant Developer Simulation Panel

As a User,
I want to be able to switch my active organization directly from the UI header and have developer tools to simulate tenant context switching,
So that I can quickly transition between authorized workspaces and developers can test isolation boundaries effectively.

**Acceptance Criteria:**

**Given** an authenticated user is in the workspace
**When** the user selects a different organization from the Workspace Selector dropdown in the navigation header
**Then** the UI MUST update the active organization ID in the store
**And** clear the local state/cache of the active workspace (projects, chat history, audit logs)
**And** fetch and load only the data matching the new organization scope

**Given** a local development environment is running with mock auth enabled
**When** a developer interacts with the Simulation Panel in the footer
**Then** the developer MUST be able to select and simulate different pre-configured mock organization contexts
**And** the UI client MUST update the authorization token and `X-Organization-Id` header sent in all subsequent API requests

### Story 10.7: Super Admin Autojoin Configuration and Auto-Provisioning

As a Super Administrator,
I want to configure automatic organization join rules based on authenticated user claims (specifically email domains),
So that new users matching these domains are automatically added to the correct organization with pre-selected roles upon their first login, eliminating manual invitation overhead.

**Acceptance Criteria:**

**Given** the user is logged in as a Super Administrator
**When** they navigate to the System Administration portal's "Autojoin Configurations" tab
**Then** the UI MUST display a list of all active autojoin mappings (showing Email Domain, Target Organization, Assigned Roles, and Status)

**Given** the user is on the Autojoin Configurations screen
**When** they click "Create Mapping" and input an email domain (e.g., `google.com`), select a target organization, select default roles (e.g., `[Developer]`), and click "Save"
**Then** the backend MUST validate that the domain is well-formed, the organization exists, and the roles are valid
**And** the mapping MUST be saved to Firestore under the `autojoin_rules` collection
**And** the UI MUST list the new rule immediately in the table

**Given** an autojoin mapping exists for `google.com`
**When** the Super Admin modifies the default roles or deletes the mapping
**Then** the Firestore rule MUST be updated or deleted in real-time
**And** the UI MUST reflect the changes immediately

**Given** an autojoin mapping exists: `google.com` domain maps to organization `Google Cloud` with default roles `[Developer]`
**When** a new user `user@google.com` logs in for the very first time (after passing IAP/JWT verification)
**Then** the backend authentication interceptor MUST check their email domain `google.com` against the autojoin rules
**And** automatically provision their user profile and associate them with organization `Google Cloud` with the `[Developer]` roles in Firestore
**And** automatically route them directly to the `Google Cloud` project dashboard instead of a blank/onboarding setup page

**Given** no autojoin mapping exists for `external.com`
**When** a new user `user@external.com` logs in for the first time
**Then** they MUST NOT be associated with any organization
**And** they MUST be prompted to create a new organization or wait for a manual invitation


## Epic 11: Identity and Access Management

Secure the platform infrastructure, API endpoints, and portals by implementing Google Identity Platform, validating IAP/OIDC tokens, and providing administrative portals for user management and organization bootstrapping.

### Story 11.1: Google Identity Platform Configuration & Infrastructure Provisioning

As a Systems Administrator / Platform Engineer,  
I want to provision and configure the Google Identity Platform using Infrastructure-as-Code (IaC),  
So that user authentication, email/password sign-in, Google IDP integration, and multi-tenant capabilities are enabled at the project level.

**Acceptance Criteria:**

**Given** the Google Cloud project environment is initialized and APIs are enabled
**When** the Terraform module runs and applies the configuration
**Then** the Google Identity Platform is configured for the project
**And** the Email/Password sign-in provider is enabled (`password_required = true`) with duplicate emails disallowed
**And** Multi-tenancy is enabled on the Identity Platform (`allow_tenants = true`)
**And** if Google OAuth client credentials are provided, the default supported IDP configuration is successfully created and enabled for `google.com`

### Story 11.2: Identity-Aware Proxy (IAP) Token Validation Middleware

As a Security Engineer,  
I want the backend API to intercept and validate Google Cloud Identity-Aware Proxy (IAP) JWT assertions,  
So that all non-public API endpoints are protected and user identity claims are securely extracted into the request context.

**Acceptance Criteria:**

**Given** an unauthenticated request attempts to access a protected backend API endpoint
**When** the token validation middleware intercepts the request
**Then** the request is rejected with an HTTP `401 Unauthorized` status if the `X-Goog-IAP-JWT-Assertion` header is missing, malformed, or has an invalid signature
**Given** an authenticated request is made through Google Cloud IAP with a valid JWT assertion header
**When** the validation middleware successfully verifies the signature and audience claims of the JWT
**Then** the middleware extracts the user's email, subject (unique user ID), and organization/tenant identifier from the claims
**And** injects the extracted user identity profile securely into the request context for subsequent handlers to consume

### Story 11.3: System Administration - Organization Provisioning Screen (System Admin Portal)

As a Super Administrator,  
I want a System Admin Portal to provision a new Organization and assign its initial Owner,  
So that I can bootstrap new customer environments and set up their Identity Platform Tenant boundaries.

**Acceptance Criteria:**

**Given** a Super Administrator is authenticated and authorized to access the System Admin Portal
**When** they fill out the Organization Provisioning form with the Organization Name and designated Owner's email, and click the "Provision Organization" button
**Then** the UI submits the request to the administrative backend API
**And** the backend calls the Google Identity Platform APIs to provision a new Identity Platform Tenant for that Organization
**And** the backend creates the initial owner's user account inside that specific tenant
**And** sets the custom claim `{ role: "owner" }` on their profile
**And** the UI displays a success status and the new tenant configuration details once complete

### Story 11.4: Organization Settings - Member Directory & User Management Screen (Org Admin Portal)

As an Organization Admin/Owner,  
I want a Member Directory screen to manage my organization's members, statuses, and roles,  
So that I can control who has access to our organization's workspace and configure their roles.

**Acceptance Criteria:**

**Given** an Organization Admin/Owner is logged into their organization dashboard
**When** they navigate to the Member Directory page
**Then** the UI requests the member list from the backend for that active organization tenant
**And** the backend queries the Google Identity Platform for that tenant (`tenantAuth.listUsers()`) to retrieve user profiles, statuses, and custom claims
**And** the UI renders a table listing all members including their Name, Email, Role (from `customClaims`), and Status (Active, Invited, Pending)
**And** the UI provides an "Invite Member" action button
**And** the UI provides actions next to each member to update their role (e.g., promoting/demoting) or revoke their access (deleting the user from the tenant)
**And** these role change or deletion actions update the user's custom claims or status in the Identity Platform Tenant via a secure backend API call

### Story 11.5: Organization Settings - Invite Member Modal & Invitation Flow

As an Organization Admin/Owner,  
I want to invite new members (Admins, Editors, Viewers) to my organization via a secure email invitation link,  
So that they can join the organization with their pre-configured role.

**Acceptance Criteria:**

**Given** an Organization Admin/Owner is on the Member Directory screen
**When** they click the "Invite Member" button and the Invite Modal opens
**And** they input the invitee's Email, select a Role (Admin, Editor, Viewer), and click "Send Invite"
**Then** the UI submits the invitation details to the backend API
**And** the backend generates a secure, tenant-scoped Email Link sign-in URL using the Google Identity Platform Admin SDK (`tenantAuth.generateSignInWithEmailLink(email, actionCodeSettings)`)
**And** the backend persists the pending invite details (including the designated role to be set as a custom claim)
**And** triggers an email containing the secure invitation link to the invitee
**And** the UI displays a confirmation that the invitation was successfully sent

### Story 11.6: Organization Settings - Roles & Permissions Configuration Screen (Org Admin Portal)

As an Organization Admin/Owner,  
I want a Roles & Permissions Configuration screen to view predefined roles and create custom roles with granular permissions,  
So that I can tailor access control to the specific needs of my organization's members.

**Acceptance Criteria:**

**Given** an Organization Admin/Owner is logged into their organization dashboard
**When** they navigate to the Roles & Permissions Configuration page
**Then** the UI displays a "Predefined Roles" tab showing read-only default roles (Admin, Editor, Viewer) and their mapped permissions (e.g., `read:data`, `write:data`)
**And** the UI displays a "Custom Roles" tab listing all custom roles created specifically for this organization
**And** the UI provides a "Create Custom Role" action (opening a modal/form)
**When** they input the Custom Role Name (e.g., "External Auditor"), check granular permissions (e.g., `view:billing`, `edit:settings`, `download:reports`), and click "Save Custom Role"
**Then** the UI saves the custom role configuration to the database (scoped to the Organization ID)
**And** when the role is assigned to a user, the backend updates the user's Google Identity Platform custom claims with the role name and the specific array of permissions (`tenantAuth.setCustomUserClaims(uid, { role: "External Auditor", permissions: ["view:billing"] })`)

### Story 11.7: Invited User Experience - Tenant-Scoped Login & Invitation Acceptance Page

As an Invited User,  
I want a tenant-scoped landing page to set up my credentials and log into my organization's workspace,  
So that my session and identity are securely bound to the correct tenant boundaries.

**Acceptance Criteria:**

**Given** an invited user accesses the platform using their unique invitation email link
**When** they land on the Invitation Acceptance page
**Then** the page initializes the Identity Platform Client SDK configured with their specific Tenant ID (`auth.tenantId = tenantId`)
**And** the UI displays a password setup form (or a "Sign in with Google" button) to complete account creation
**When** they submit their password or complete Google OAuth
**Then** they are successfully authenticated within that specific tenant
**And** they are redirected to their organization's dashboard with the active tenant session token containing their custom claims/roles
**Given** an existing user accesses the login page
**When** they log in
**Then** the client SDK uses their organization's tenant ID to authenticate them within the correct silo

### Story 11.8: Parent Project Super Admin Authentication & Global Resource Authorization

As a Super Administrator,  
I want to log in via a dedicated portal at the Parent Project level and gain global resource management authorization,  
So that I can securely administer all tenant organizations without being bound to any single customer tenant silo.

**Acceptance Criteria:**

**Given** a Super Administrator accesses the dedicated administrative portal (e.g., `admin.myapp.com`)
**When** they log in
**Then** the frontend login SDK is initialized without setting a `tenantId` (authenticating them against the root Parent Project)
**And** they are successfully authenticated at the parent project level
**Given** the Cloud Run backend receives an API request from the administrative portal with the Super Admin's JWT token
**When** the backend verifies the ID token (`admin.auth().verifyIdToken(token)`)
**Then** the backend checks that the token **does not** contain a `firebase.tenant` claim
**And** verifies that the custom claim `is_super_admin === true` is present
**And** grants the request access to global administrative operations (e.g., provisioning organizations, viewing all billing, etc.)

### Story 11.9: Multi-Tenant Identity Verification and Fail-Safe Security Context Propagation

As a Security Engineer,  
I want to implement strict multi-tenant validation, fail-safe environment configuration audits on startup, and zero-trust identity context propagation in all downstream services,  
So that organization boundaries are cryptographically enforced and the application cannot run in a misconfigured, insecure state in production.

**Acceptance Criteria:**

**Given** the backend API receives a request to a protected, organization-scoped endpoint in production mode (`BYPASS_IAP_VALIDATION=false`)
**When** the auth middleware resolves the user's claims
**Then** the middleware MUST verify that a valid, non-empty `OrganizationID` is present in the extracted claims (mapped from `firebase.tenant`)
**And** MUST reject the request with an HTTP `403 Forbidden` if the user's token is not scoped to a valid tenant, preventing unauthorized data-plane access.
**Given** the backend application starts up in a production environment
**When** the configuration module is initialized
**Then** it MUST perform a startup self-test to verify that `IAP_AUDIENCE` and `JWT_SECRET` are configured with valid, non-empty values
**And** MUST panic and prevent the application from starting if they are missing or if `BYPASS_IAP_VALIDATION` is set to `true` in a production environment.
**Given** a backend developer implements a new endpoint or database operation
**When** they retrieve the active user's identity or organization scope
**Then** they MUST obtain this information exclusively from the Go `context` (via `core.UserClaimsKey`)
**And** any direct references to cleartext headers like `x-goog-authenticated-user-email` or `X-Organization-Id` in downstream handlers or repositories MUST be prohibited (enforced via static analysis/code-review).
**Given** the backend runs in local development mode with `BYPASS_IAP_VALIDATION=true`
**When** a request with simulated headers is processed
**Then** the middleware MUST only trust mock/cleartext headers if the environment is explicitly configured as a non-production profile (e.g., `local` or `test`), otherwise rejecting the request to prevent perf test mode activation in production.

### Story 11.10: Application Admin Performance & Operational Telemetry Dashboard UI

As a System Administrator / Operations Engineer,
I want an operational Performance Dashboard within the System Admin Portal,
So that I can monitor real-time APM telemetry, latency distribution charts, concurrent active workspace user load, and historical load testing benchmark runs.

**Acceptance Criteria:**

**Given** a System Administrator accesses the Administration Portal
**When** navigating to the "Performance & Operations" tab
**Then** the dashboard displays live telemetry widgets for P95 latency (WebSocket frame sync < 100ms, SSE token delivery < 50ms, directory tree fetch < 150ms)
**And** shows an interactive gauge of active concurrent workspace sessions (benchmarked against 200 users capacity)
**And** provides a historical log viewer displaying past k6 load test run summaries and regression reports with filterable date controls.

### Story 11.11: Admin Portal AI Token Usage & Cost Analytics Dashboard Screen

As a System Administrator / Organization Admin,
I want a dedicated "AI Usage & Cost Analytics" dashboard screen within the Admin Section of the Administration Portal,
So that I can monitor aggregate AI token consumption and cost statistics (per user, per project, per organization, and specific/average per session) from the ADK PostgreSQL Session Database without displaying usage metrics in the workspace chat UI.

**Acceptance Criteria:**

**Given** an authorized Administrator accesses the Administration Portal (`/admin`)
**When** they navigate to the new **"AI Usage & Cost Analytics"** dashboard tab
**Then** the screen renders an interactive analytics dashboard querying the backend telemetry API endpoint (`/api/v1/telemetry/usage`)
**And** the UI displays dedicated analytical widgets and data tables for:
  - 👤 **Cost & Consumption per User:** Filterable member directory table displaying total prompt/completion tokens, total turns, and calculated cost per user
  - 📁 **Cost & Consumption per Project:** Project usage breakdown charts and telemetry table
  - 🏢 **Cost & Consumption per Organization:** Tenant-wide aggregate token consumption, monthly cost rollups, and model distribution
  - 💬 **Specific & Average Cost/Consumption per Session:** Session telemetry log showing total session tokens, turn count, and calculated average consumption per turn
**And** the dashboard provides interactive time-range selectors (Today, Last 7 Days, Last 30 Days, Custom Date Range) and export controls (CSV/JSON data download)
**And** user-facing workspace chat bubbles and sidebar chat panes MUST remain completely clean and free of token/cost indicators.

## Epic 12: Administrative NFS Storage & Infrastructure Governance / Operations

Empower platform and system administrators to maintain storage governance, automate long-running workspace housekeeping, prevent runaway NFS volume consumption, isolate multi-tenant workspace quotas, and execute batch maintenance tasks reliably via dedicated Cloud Run Jobs and Cloud Scheduler without degrading API server performance.

### Story 12.7: NFS Volume and Folder-Level Metadata Management (System Admin)

As a System Administrator (Super Admin),  
I want an administrative API and management interface for NFS volumes and folder-level workspace metadata (including storage quota enforcement, folder permission/lock flags, organization/tenant workspace bindings, retention policies, and volume health/capacity monitoring),  
so that system administrators can maintain storage governance, prevent runaway NFS volume consumption, isolate folder-level tenant metadata, and monitor volume health across all organization workspaces.

**Acceptance Criteria:**

**Given** a System Administrator authenticated at the Parent Project level with super-admin authorization (`is_super_admin === true`)  
**When** they send a `GET` request to `/api/v1/admin/nfs/volumes` or `/api/v1/admin/nfs/folders`  
**Then** the backend MUST return real-time NFS volume status (total capacity, available space, mount health) and an array of folder-level metadata records (organization ID, project ID, workspace path, quota limit, used bytes, file count, read-only flag, and retention policy).  
**Given** a System Administrator on the Admin Portal or sending a `PUT` request to `/api/v1/admin/nfs/folders/{folderId}`  
**When** they update folder metadata attributes (e.g. setting `quotaSizeBytes`, toggling `isReadOnly`, or modifying `retentionPolicy`)  
**Then** the backend MUST update and persist the folder metadata in Firestore under `/admin_nfs_metadata/folders/{folderId}`  
**And** immediately return the updated metadata object with HTTP `200 OK`.  
**Given** an organization workspace on NFS whose folder metadata has `isReadOnly: true` or where disk usage exceeds `quotaSizeBytes`  
**When** an AI agent, API handler (`FileSystemArtifactService`), or Git client (`GitClient`) attempts a file write, create, or commit operation on that workspace folder  
**Then** the storage service / Git client MUST intercept the operation and reject it with an HTTP `403 Forbidden` / `StorageQuotaExceededError` before allocating disk space or modifying files.  
**Given** workspace directories configured under `{WorkspaceMountPath}/organizations/{organizationId}/workspaces/{userId}/{projectId}`  
**When** the background NFS audit worker runs (periodically or on-demand via `POST /api/v1/admin/nfs/audit`)  
**Then** it MUST scan workspace directory sizes, compute total bytes used and file counts per tenant workspace, and update the corresponding folder metadata records in Firestore with the calculated metrics and `lastAuditedAt` timestamp.  
**Given** a System Administrator accessing the System Admin Portal at `/admin/nfs`  
**When** the page loads  
**Then** the UI MUST render an NFS Volume Health & Capacity overview card, a searchable/filterable Folder Metadata Table, and an Edit Quota & Lock Modal.

### Story 12.8: Housekeeping Job Framework, Registry & CLI Dispatcher

As a System Developer / Platform Engineer,  
I want a unified, modular Go `Job` interface and thread-safe registry in `api/jobs/` and a standalone CLI entrypoint in `api/cmd/jobs/main.go`,  
So that housekeeping tasks can be executed as standalone batch processes with clear argument parsing, structured logging, and signal handling.

**Acceptance Criteria:**

**Given** the `api/jobs/` package  
**When** imported  
**Then** it defines a standard `Job` interface:
```go
type Job interface {
    Name() string
    Description() string
    Run(ctx context.Context, cfg *core.Configuration) error
}
```
and a thread-safe `Registry` struct supporting `Register(Job)`, `Get(name string) (Job, bool)`, `List() []Job`, and `Execute(ctx context.Context, name string, cfg *core.Configuration) error`.  
**Given** the CLI command `go run ./api/cmd/jobs --list` or `apf-jobs --list`  
**When** executed  
**Then** it outputs a formatted list of all registered housekeeping jobs with their names and descriptions and exits with code 0.  
**Given** the CLI command `go run ./api/cmd/jobs <job-name>` or `go run ./api/cmd/jobs --name <job-name>`  
**When** executed with a valid registered job name  
**Then** it initializes `core.Configuration` from environment variables, executes the job with an interrupt-aware context (`signal.NotifyContext(SIGINT, SIGTERM)`), logs start/finish metrics using `log/slog`, and exits with code 0 on success or code 1 on error.  
**Given** an invalid job name provided to the CLI  
**When** executed  
**Then** it prints a helpful error message listing valid available jobs and exits with code 1.  
**Given** unit tests in `api/jobs/job_test.go` and `api/cmd/jobs/main_test.go`  
**When** `go test ./api/jobs/...` is run  
**Then** all tests pass validating registration, execution, and error handling.

### Story 12.9: Modular Housekeeping Jobs: NFS Storage Audit & Orphaned Workspace Cleanup

As a System Administrator,  
I want the existing NFS audit scan and hanging workspace discovery/deletion routines ported into modular `Job` implementations (`NFSAuditJob` and `HangingWorkspaceCleanupJob`),  
So that storage governance and workspace disk cleanup can run reliably as batch operations with audit persistence and dry-run safety.

**Acceptance Criteria:**

**Given** `NFSAuditJob` in `api/jobs/nfs_audit.go`  
**When** executed via `apf-jobs nfs-audit`  
**Then** it initializes `db.NFSStore`, invokes `RunNFSAuditScan`, updates organization folder metadata in Firestore under `/admin_nfs_metadata/folders`, logs scanned byte counts and file counts via `slog`, and returns nil error upon successful completion.  
**Given** `HangingWorkspaceCleanupJob` in `api/jobs/hanging_cleanup.go`  
**When** executed via `apf-jobs hanging-cleanup`  
**Then** it calls `DiscoverHangingProjects` against the mounted workspace path, removes dangling project directories and lock files for projects absent in the database, and records freed disk space and deleted counts in logs.  
**Given** the flag `--dry-run` passed to `apf-jobs hanging-cleanup --dry-run`  
**When** executed  
**Then** it scans and reports all candidate orphaned folders and their sizes without performing any `os.RemoveAll` disk deletions.  
**Given** unreadable directories, broken symlinks, or permission errors on individual folders during scanning  
**When** encountered  
**Then** the job logs a structured warning and continues scanning remaining directories rather than aborting prematurely.  
**Given** unit & integration tests in `api/jobs/nfs_audit_test.go` and `api/jobs/hanging_cleanup_test.go`  
**When** tests are run  
**Then** all tests pass with high branch coverage.

### Story 12.10: Multi-Binary Container Packaging in `api/Dockerfile` & CI/CD Pipeline

As a DevOps / Release Engineer,  
I want the `api/Dockerfile` to compile both `/apf-api` and `/apf-jobs` into the single API container image,  
So that Cloud Run Jobs can reuse the standard `apf-api` container image without requiring a separate Docker build pipeline or duplicate artifact registry repositories.

**Acceptance Criteria:**

**Given** `api/Dockerfile`  
**When** the builder stage executes  
**Then** it compiles the API web server to `/out/apf-api` (`go build -o /out/apf-api .`) and the Jobs CLI to `/out/apf-jobs` (`go build -o /out/apf-jobs ./cmd/jobs`).  
**Given** the final runtime stage of `api/Dockerfile`  
**When** assembled  
**Then** both `/apf-api` and `/apf-jobs` are copied into the root directory with executable permissions (`chmod +x`).  
**Given** the container default `ENTRYPOINT ["/apf-api"]`  
**When** run without arguments  
**Then** it starts the HTTP web server on port 8081 as normal.  
**Given** the container run with entrypoint/args overridden (`docker run <image> /apf-jobs --list`)  
**When** executed  
**Then** it executes the CLI runner and lists available jobs.  
**Given** `infra/cicd/cloudbuild.yaml`  
**When** Kaniko builds the API image  
**Then** the build succeeds and the resulting image in Artifact Registry contains both binaries.

### Story 12.11: Cloud Run v2 Job & Cloud Scheduler Terraform Infrastructure

As a Cloud Infrastructure Engineer,  
I want Terraform configurations in `infra/modules/` to provision the `google_cloud_run_v2_job` and `google_cloud_scheduler_job` resources,  
So that housekeeping tasks run in isolated compute environments on automated cron schedules with VPC access and NFS volume mounts.

**Acceptance Criteria:**

**Given** `infra/modules/housekeeping_jobs.tf`  
**When** applied  
**Then** it creates a `google_service_account` for the job scheduler with `roles/run.invoker` IAM binding on the project.  
**Given** the `google_cloud_run_v2_job` resource definition  
**When** deployed  
**Then** it is configured with `max_retries = 2`, `timeout = "1800s"`, `service_account = google_service_account.compute.email`, VPC Access Connector / direct VPC egress set to `ALL_TRAFFIC`, NFS volume mount for `/workspaces`, image set to `${var.region}-docker.pkg.dev/${var.project_id}/${var.prefix}-repo/apf-api:latest`, command set to `["/apf-jobs"]`, default args `["nfs-audit"]`, environment variables for `GOOGLE_CLOUD_PROJECT`, `WORKSPACE_MOUNT_PATH`, and `FIRESTORE_DB_NAME`, and CloudSQL password secret from Secret Manager.  
**Given** the `google_cloud_scheduler_job` resource  
**When** deployed  
**Then** it targets the Cloud Run v2 Job HTTP run endpoint (`https://${var.region}-run.googleapis.com/v2/projects/${var.project_id}/locations/${var.region}/jobs/${job_name}:run`) on schedule `0 2 * * *` (nightly at 2:00 AM UTC) with OIDC/OAuth authentication using the scheduler service account.  
**Given** `infra/staging` and `infra/prod`  
**When** `terraform validate` and `terraform plan` are executed  
**Then** the configuration validates with zero syntax or provider schema errors.

### Story 12.12: Super Admin On-Demand Job Execution API & Admin Portal Trigger UI

As a Super Administrator,  
I want to trigger Cloud Run housekeeping jobs on-demand from the System Admin Portal and monitor their execution status,  
So that I can force immediate storage audits or workspace cleanups without waiting for the nightly cron schedule.

**Acceptance Criteria:**

**Given** an authenticated Super Administrator (`is_super_admin: true`)  
**When** they send `POST /api/v1/admin/jobs/{jobName}/run` (with optional payload `{ "dryRun": true }`)  
**Then** the API handler calls the Google Cloud Run v2 API (`projects.locations.jobs.run`) to instantiate a new job execution with overridden args.  
**Given** a non-super-admin user attempting to call the endpoint  
**When** requested  
**Then** the backend rejects the request with HTTP `403 Forbidden`.  
**Given** a successful trigger from the backend  
**When** the execution starts  
**Then** the endpoint returns HTTP `202 Accepted` with `{ "status": "PENDING", "executionName": "projects/.../executions/...", "jobName": "{jobName}" }`.  
**Given** the System Admin Portal (`ui/src/app/(dashboard)/admin/nfs/page.tsx` or `/admin/jobs`)  
**When** viewed  
**Then** it provides an action button to "Run Storage Audit Now" / "Run Cleanup", displays an active execution status badge with spinner while in progress, and displays a success notification upon completion.  
**Given** unit tests in `api/admin_job_handlers_test.go`  
**When** executed  
**Then** tests pass verifying Super Admin access control and Cloud Run SDK request formation.

### Story 12.13: Super Admin Jobs Catalog & Cloud Run Execution History Backend API

As a Super Administrator,  
I want secure backend API endpoints to retrieve the catalog of registered system jobs and query real-time execution history directly from the Google Cloud Run v2 API without requiring database storage,  
So that the admin frontend and operational tooling can inspect available maintenance tasks, track running batch jobs, and review historical outcomes, durations, and logs.

**Acceptance Criteria:**

**Given** an authenticated Super Administrator (`is_super_admin: true`)  
**When** they send `GET /api/v1/admin/jobs`  
**Then** the backend returns HTTP `200 OK` with a JSON list of all registered jobs in `jobs.DefaultRegistry` (`name`, `description`, and supported parameter flags).  
**Given** a non-super-admin user attempting to access `GET /api/v1/admin/jobs` or `GET /api/v1/admin/jobs/executions`  
**When** requested  
**Then** the backend rejects the request with HTTP `403 Forbidden`.  
**Given** an authenticated Super Administrator (`is_super_admin: true`)  
**When** they send `GET /api/v1/admin/jobs/executions` (with optional query parameters `pageSize` and `pageToken`)  
**Then** the API proxies the Google Cloud Run v2 API (`Projects.Locations.Jobs.Executions.List`) for the configured housekeeping job (`MAINTENANCE_JOB_NAME` or `apf-stg-jobs` / `apf-prod-jobs`),  
**And** parses each execution into a structured response containing:
- `id`: Execution identifier (e.g. `apf-stg-housekeeping-j4tql`)
- `name`: Full GCP resource name (`projects/.../locations/.../jobs/.../executions/...`)
- `jobName`: Sub-job extracted from container execution arguments (e.g., `"global-workspace-provisioning"`, `"nfs-audit"`, `"hanging-cleanup"`)
- `dryRun`: Boolean indicating whether `--dry-run` flag was passed in args
- `status`: Execution status (`RUNNING`, `SUCCEEDED`, `FAILED`, `CANCELLED`) derived from Cloud Run conditions
- `createTime`: Timestamp when execution was created (RFC3339)
- `startTime`: Timestamp when container task began running (RFC3339)
- `completionTime`: Timestamp when execution finished (or null if in progress)
- `durationSeconds`: Total runtime in seconds (computed from `startTime` and `completionTime` or current time if running)
- `durationFormatted`: Human-readable duration string (e.g. `"1m 14s"`, `"42s"`)
- `errorMessage`: Detailed error description or exit code if status is `FAILED`
- `logUri`: Direct deep-link URL to Google Cloud Logging for that specific execution  
**And** returns HTTP `200 OK` with the executions array and `nextPageToken`.  
**Given** `SimulatedCloudRunJobRunner` or local development mode without GCP credentials  
**When** `GET /api/v1/admin/jobs/executions` is called  
**Then** it returns simulated execution entries and HTTP `200 OK` for local verification without crashing.  
**Given** unit tests in `api/admin_job_handlers_test.go`  
**When** executed  
**Then** tests pass verifying RBAC, catalog listing, argument parsing, execution status mapping, error extraction, and pagination.

### Story 12.14: Super Admin Jobs Dashboard & Execution History Portal UI (`/admin/jobs`)

As a Super Administrator,  
I want a dedicated System Admin Portal screen at `/admin/jobs` with an available jobs catalog and a live/historical executions monitor,  
So that I can trigger system housekeeping jobs on-demand (with live status updates) and inspect past execution outcomes, durations, and log links directly from the UI.

**Acceptance Criteria:**

**Given** a Super Administrator accessing the System Admin Portal  
**When** they navigate to `/admin/jobs` or click "Jobs" in the admin navigation sidebar/header tabs  
**Then** the UI renders the Jobs Management & Execution History page containing:
- **Registered Jobs Catalog Table**: Displays available system jobs (`global-workspace-provisioning`, `nfs-audit`, `hanging-cleanup`) with their descriptions and "Run Job" action buttons.
- **Active & Historical Executions Table**: Displays past and in-flight executions with columns for *Execution ID*, *Job Name*, *Dry Run*, *Status*, *Started At*, *Duration*, *Outcome Details*, and *Actions*.  
**Given** the Registered Jobs Catalog Table  
**When** the user clicks "Run Job" on any job row  
**Then** the UI displays an execution modal allowing the user to toggle "Dry Run" mode and confirm execution.  
**Given** the user confirms execution in the modal  
**When** submitted  
**Then** the UI sends `POST /api/v1/admin/jobs/{jobName}/run` with `{ "dryRun": true/false }`, displays an active progress notification, adds the new pending execution to the top of the executions table, and initiates periodic status polling.  
**Given** an execution row in the Executions Table  
**When** rendered  
**Then** it displays appropriate status badges:
- 🔵 `RUNNING` (with animated pulse spinner)
- 🟢 `SUCCEEDED`
- 🔴 `FAILED` (with hoverable error tooltip or expandable details)
- ⚪ `CANCELLED`  
**And** includes a "View Logs" external link opening the GCP Cloud Logging URL in a new tab.  
**Given** the Executions Table  
**When** the user clicks the "Refresh" button or while an execution is `RUNNING`  
**Then** the UI polls `GET /api/v1/admin/jobs/executions` every 5 seconds until active executions reach a terminal state (`SUCCEEDED`, `FAILED`, `CANCELLED`).  
**Given** component and integration tests in `ui/src/app/(dashboard)/admin/jobs/page.test.tsx`  
**When** executed via `npm test`  
**Then** all tests pass validating job listing, on-demand execution trigger, live status polling, error handling, and log link rendering.

## Epic 13: Mobile Application Development Experience

Deliver a responsive, touch-optimized mobile experience enabling developers and product managers to interact with BMAD agents, view and edit specs, inspect visual canvas layouts, and trigger builds using mobile devices.

### Story 13.1: Mobile Projects Dashboard Screen

As an end-user / developer,
I want a mobile-responsive Projects Dashboard with touch-friendly project tiles, status badges (`New`, `Onboarding`, `Onboarded`), tile action buttons, and showcase project cards,
So that I can open, onboard, and manage my application projects from a mobile device.

**Acceptance Criteria:**

**Given** the user opens Agentic Foundry on a mobile device or small screen breakpoint (< 768px)
**When** navigating to the Projects Dashboard
**Then** the project tiles format into a single responsive touch-friendly grid with touch targets ≥ 48x48px
**And** each tile displays status indicators (`New`, `Onboarding`, `Onboarded`) and a primary action button (e.g., `Onboard` or `Open Workspace`)
**And** tapping the showcase project card initializes the sample workspace directly on mobile.

### Story 13.2: Mobile Agent Chat Workspace Screen

As a developer / PM,
I want a mobile-optimized agent chat interface with persona switching (`bmad-pm`, `bmad-ux`, `bmad-dev`), quick prompt chips, collapsible context drawers, and response cancellation,
So that I can interact with AI agents and review generated specs effectively on mobile devices.

**Acceptance Criteria:**

**Given** a user is active in the Mobile Chat view
**When** interacting with an active agent persona
**Then** the message stream renders fluidly without horizontal overflow or viewport shifting
**And** quick-action prompt chips (e.g., "Scaffold", "Run Code Review", "Approve Spec") are rendered above the keyboard input
**And** touch triggers for file attachments, photo captures, and message cancellations function smoothly on iOS and Android viewports.

### Story 13.3: Mobile Touch Canvas & Node Workflow Screen

As a PM / UX Designer,
I want to view and interact with visual node workflow canvas diagrams using multi-touch gestures (pinch-to-zoom, drag-pan, tap-to-inspect),
So that I can review architecture diagrams and workflow decision trees from my mobile device.

**Acceptance Criteria:**

**Given** a user opens the Canvas tab on a mobile screen
**When** inspecting workflow diagrams and decision trees
**Then** touch gestures support smooth multi-touch pinch-to-zoom and drag-pan interactions with < 100ms response latency
**And** tapping a node opens a mobile bottom sheet previewing node details, status triggers, and associated spec artifacts
**And** double-tap or long-press options enable quick node action menus.

### Story 13.4: Mobile Workspace Explorer & Spec Viewer Screen

As a developer,
I want a mobile tabbed workspace explorer with a touch file tree picker and a rich Markdown spec viewer,
So that I can navigate project directories and review generated specifications and code diffs on mobile.

**Acceptance Criteria:**

**Given** a user opens the Workspace Explorer tab on a mobile device
**When** navigating the project directory tree
**Then** the file tree component supports touch expand/collapse gestures with colorful file-type icons
**And** tapping a spec file (e.g., `prd.md`, `architecture.md`) opens a readable mobile Markdown viewer with copy and section navigation controls
**And** file diffs render within screen bounds with touch-scroll capability.

### Story 13.5: Mobile User Settings & API Secrets Screen

As an end-user,
I want a dedicated mobile user settings screen to securely manage personal API keys, MCP integration tokens, and persona preferences,
So that I can configure my integrations and user settings from a mobile device.

**Acceptance Criteria:**

**Given** the user opens the User Settings screen on a mobile device
**When** entering or updating secret API keys (e.g., Figma token, GitHub token)
**Then** the secret inputs use secure password fields with visibility toggle buttons and clear touch targets
**And** submitting changes triggers encrypted persistence with immediate visual confirmation snackbars
**And** user persona defaults can be toggled via mobile radio cards.

### Story 13.6: Mobile Product Onboarding Tour Screen & Overlays

As a new mobile user,
I want touch-friendly product tour popover cards with section progress indicators and swipe controls,
So that I can learn how to use the platform workspace on mobile viewports.

**Acceptance Criteria:**

**Given** a new user logs into the mobile workspace or manually triggers the Product Tour
**When** the tour starts
**Then** tour popover cards position dynamically relative to target elements without covering primary mobile navigation
**And** popovers display section progress (e.g., "Section 2 of 4: User Secrets") and touch controls ("Next", "Skip Section", "Skip Tour")
**And** swiping or tapping "Next" advances the tour smoothly with state saved in Firestore.

### Story 13.7: Mobile Push Notifications & Task Sync Engine

As a mobile developer / PM,
I want to receive push notifications and real-time alerts on my mobile device when agent tasks complete, PRs are created, or reviews require approval,
So that I can monitor ongoing development workflows on mobile.

**Acceptance Criteria:**

**Given** an agent task or build job is initiated in a project workspace
**When** the task completes or reaches a required human approval gate
**Then** a push notification (Web Push / FCM) is delivered to registered mobile devices
**And** tapping the notification opens the app directly to the relevant agent chat or artifact approval screen.


## Epic 14: Performance & Real-Time Latency Testing Infrastructure

Establish automated k6 load testing scripts in `{project-root}/perf/`, synthetic multi-tenant test data seeding fixtures, distributed load harnesses simulating 200 concurrent active users, real-time WebSocket/SSE latency benchmarking, and CI/CD performance regression gates.

### Story 14.1: k6 Performance Test Suite Scaffolding & Directory Structure

As a Lead QA Engineer / Performance Engineer,
I want to establish a structured performance testing directory at `{project-root}/perf/`,
So that our performance tests follow modular best practices with clean separation of scenarios, utilities, configurations, and reports.

**Acceptance Criteria:**

**Given** the performance testing repository workspace
**When** initializing the `{project-root}/perf/` directory
**Then** the folder structure is scaffolded with `scenarios/`, `utils/`, `config/`, and `reports/` subdirectories
**And** modular k6 ES6 scripts are established for HTTP data plane and WebSocket streaming connections
**And** environment configuration profiles (local, staging, prod) and npm/make execution scripts are committed to the codebase.

### Story 14.2: Performance Test Data Seeding & Synthetic Environment Fixtures

As a Performance Engineer,
I want CLI tooling and fixture generators to seed synthetic multi-tenant workspace data,
So that reproducible load testing can be executed against populated Firestore databases and GCS storage enclaves.

**Acceptance Criteria:**

**Given** an isolated target performance testing environment
**When** running the performance data seeding script (`go run ./scripts/perf-seed/main.go`)
**Then** the script generates synthetic multi-tenant organization boundaries, showcase workspaces, project file trees, and active session histories
**And** populates Firestore collections and GCS bucket paths without interfering with production tenant data
**And** outputs environment credential profiles and session tokens for k6 virtual users.

### Story 14.3: Distributed Multi-Cluster Load Test Harness (200 Concurrent Users)

As a Performance Engineer / DevOps Engineer,
I want a k6 load testing scenario simulating 200 concurrent active users across workspace cluster sessions,
So that we can measure platform scalability and verify stability under multi-tenant load without degradation.

**Acceptance Criteria:**

**Given** the target application cluster initialized with performance seed data
**When** running the k6 distributed load scenario targeting 200 concurrent virtual users over a 10-minute sustained window
**Then** the cluster maintains operations with 0% unhandled 5xx API errors
**And** API response latency p95 remains below 500ms under load (NFR-3.1).

### Story 14.4: Real-Time WebSocket & SSE Streaming Latency Benchmarking Suite

As a Performance Engineer,
I want k6 metric collectors to profile real-time WebSocket canvas sync latency and SSE chat response streaming rates,
So that we can empirically verify real-time user experience standards under stress.

**Acceptance Criteria:**

**Given** active k6 virtual user sessions connected to workspace streaming endpoints
**When** generating concurrent canvas node mutation events and live agent execution prompts
**Then** WebSocket frame broadcast synchronization latency across multi-viewer sessions measures < 100ms at p95 (NFR-1.2)
**And** SSE token delivery streaming latency measures < 50ms per chunk (NFR-1.5)
**And** execution cancellation signals propagate and abort active backend context processing within < 1s (NFR-1.4).

### Story 14.5: CI/CD Performance Regression Quality Gates & Telemetry Alarming

As a DevOps / Release Engineer,
I want automated k6 performance regression gates in our Cloud Build / CI pipelines,
So that build deployments fail automatically if latency or throughput thresholds regress.

**Acceptance Criteria:**

**Given** a release candidate Pull Request or staging build deployment
**When** the automated performance pipeline step executes k6 benchmark scenarios
**Then** k6 evaluates defined threshold assertions (WebSocket latency < 100ms, API error rate < 0.1%, p95 response < 500ms)
**And** automatically fails the pipeline build job if any performance threshold is breached, posting a summary performance report to the build logs.


## Epic 15: External Integrations & Feature Flag Management

Provide a centralized interface and secure configuration management for external service integrations, AI model providers (including Gemini API key entry), third-party developer tool connectors, and application-level dynamic feature flags.

### Story 15.1: Gemini API Key Management

As a user,
I should be able to input my Gemini API key,
So that it can be used in Gemini model calls.

**Acceptance Criteria:**

**Given** a user accessing the external integrations management settings page
**When** the user inputs a valid Gemini API key and submits the form
**Then** the Gemini API key is securely encrypted and stored in the user's secret configuration
**And** subsequent Gemini model calls dynamically load and authenticate using the configured Gemini API key
**And** the UI displays a masked key representation or status indicator for security verification.

### Story 15.2: Application-Level Feature Flags & On-Demand Feature Document Loading API

As a developer/administrator,
I want to manage experimental feature flags in an application-level `features` collection in Firestore and fetch individual feature flag documents on-demand when navigating to specific UI screens,
So that experimental UI features and sub-features dynamically adjust their behavior based on target feature configuration without degrading application startup performance by loading all flags upfront.

**Acceptance Criteria:**

**Given** global application-level feature flag configurations in Firestore
**When** the administrator configures the `features` collection
**Then** top-level documents in `features` represent overall features (e.g., `canvas_v2`, `agent_copilot`), and properties within each document define sub-features and UI behavior settings (e.g., `{ enabled: true, show_experimental_badge: true, max_retry_count: 3 }`).

**Given** a backend REST API endpoint `GET /api/v1/features/:featureId`
**When** a client sends a request for a specific feature document ID
**Then** the API fetches and returns only the requested feature document from the `features` collection with a sub-200ms latency
**And** returns a 404 status with an explicit error structure if the specified feature document does not exist.

**Given** a user navigating to a specific UI screen or component with an experimental feature
**When** the screen component mounts or the route transition occurs
**Then** the client application issues an on-demand API request to load the corresponding feature document for that screen/feature rather than fetching all system feature flags at application startup
**And** the UI dynamically updates component rendering and sub-feature behavior according to the returned sub-properties.

**Given** a network failure or uninitialized feature flag state during screen navigation
**When** the on-demand feature document request fails or returns default disabled states
**Then** the UI gracefully falls back to default component behavior without crashing or blocking screen navigation.

### Story 15.3: Feedback Collection Modal Component & Top Banner Triggers

As a user,
I want to click on "Learn More" or "File a Bug" links in the top notice banner to open a lightweight feedback modal dialog,
So that I can easily submit feedback, report issues, or provide product suggestions without leaving my active workspace page.

**Implementation Details:**
- **Firestore Storage**: Submitted feedback items are persisted to the root-level `feedback` collection in Firestore.
- **Data Model**: Documents in `feedback` collection contain fields: `id`, `type` (e.g. `"bug"`, `"feature_request"`, `"general"`), `subject`, `message`, `userEmail`, `organizationId`, `projectId`, and `createdAt` timestamp.
- **Backend API**: `POST /api/v1/feedback` endpoint handles validation, claims parsing, and document persistence to `feedback` collection.

**Acceptance Criteria:**

**Given** a user interacting with the top banner navigation/notice bar
**When** the user clicks on the "Learn More" or "File a Bug" buttons/links
**Then** a lightweight feedback collection modal dialog opens over the current view.

**Given** the feedback collection modal dialog is open
**When** rendered to the user
**Then** it displays minimal, clean UI elements (feedback type selector/subject, feedback detail text area, optional email field, and Submit/Cancel action buttons).

**Given** a user submitting feedback
**When** the user fills out the required feedback details and clicks "Submit"
**Then** the feedback payload is validated and submitted via `POST /api/v1/feedback` to be stored in the root-level `feedback` Firestore collection, a success toast notification is displayed, and the modal dialog closes gracefully.

**Given** a user deciding to cancel or dismiss the feedback modal
**When** the user clicks the "Cancel" button, close icon (X), or clicks outside the modal backdrop
**Then** the modal closes without submitting data or losing active workspace state.


## Epic 16: Zero-Trust Path Resolution, Tool Parity & Native BMad TOML Configuration Architecture

Unify agent toolsets, eliminate fragile in-flight regex config/markdown mutations in favor of native BMad TOML layer resolution, provide script validation guardrails in `uv_runner`, and consolidate path safety into a single-pass zero-trust boundary architecture.

### Story 16.1: Unified Toolset Provisioning Across All Agent Swarms

As an AI Subagent in the Analysis or Help swarm (e.g. `APF Brainstorm`, `APF Product Brief`, `APF Help`),
I want full access to the standard toolset (`file_reader`, `file_writer`, `directory_reader`, and `uv_run`),
So that I can scan directories, evaluate glob patterns, and execute sandboxed BMad Python scripts without falling back to invalid shell command invocations.

**Acceptance Criteria:**

**Given** the `ai/sub_agents/analysis/agent.go` agent factory,
**When** initializing subagents and the root `APF Analysis Agent`,
**Then** the loader MUST instantiate `directoryReaderTool` via `subagents.GetDirectoryReaderTool()` and register it in the `Tools` slice for all subagents and the root agent.

**Given** the `ai/sub_agents/help/agent.go` agent factory,
**When** initializing subagents and the root `APF Help Agent`,
**Then** the loader MUST instantiate `directoryReaderTool` and `uvRunnerTool` via `subagents.GetDirectoryReaderTool()` and `subagents.GetUVRunnerTool(cfg)`, registering both in the `Tools` slice for all subagents and the root agent.

**Given** an active `APF Brainstorm` session executing `SKILL.md`,
**When** the agent reaches step 5 (`Glob {workflow.output_dir}/*/.memlog.md`),
**Then** the agent MUST execute the `directory_reader` tool to discover existing memlogs rather than calling `uv_run` with `find` or `python3`.

**Given** unit tests in `ai/sub_agents/analysis/agent_test.go` and `ai/sub_agents/help/agent_test.go`,
**When** running `go test ./ai/sub_agents/...`,
**Then** all tests MUST pass and verify the presence of `directory_reader` and `uv_run` tools across all configured agents.

### Story 16.2: UVRunnerTool Binary Command Guardrails & Tool-Redirect Guidance

As an AI Subagent (e.g. `APF Brainstorm`, `APF Dev`, `APF Architect`),
I want clear, immediate validation and guidance when invoking `uv_run`,
So that if I erroneously attempt to invoke system commands (`find`, `ls`, `python3`), I am immediately directed to use the correct ADK tool (`directory_reader`, `file_reader`) rather than wasting execution cycles on failing subprocesses.

**Acceptance Criteria:**

**Given** an invocation of `uv_run` where `scriptPath` is a common shell or interpreter command (`find`, `ls`, `grep`, `cat`, `bash`, `sh`, `python`, `python3`),
**When** `UVRunnerTool.Execute()` validates the input,
**Then** it MUST immediately return `&UVRunnerOutput{Success: false, ExitCode: 1, Stderr: "..."}` containing explicit guidance recommending `directory_reader` or `file_reader` without launching a subprocess or holding a semaphore.

**Given** an invocation of `uv_run` where `scriptPath` does not end with `.py` and does not resolve to an existing executable script,
**When** validating the path,
**Then** it MUST return a structured validation error explaining that `uv_run` requires a `.py` Python script.

**Given** valid Python scripts located within `projectRoot` or `bmadRoot` (e.g. `_bmad/scripts/memlog.py`, `_bmad/scripts/resolve_config.py`),
**When** validating,
**Then** validation MUST pass and execute the script normally.

**Given** unit tests in `ai/tools/uv_runner_test.go`,
**When** running tests,
**Then** test cases MUST verify rejection of `find`, `ls`, and `python3` with appropriate error messages and zero subprocess executions.

### Story 16.3: Native Project TOML Workspace Configuration Seeding

As a Developer and AI Agent Framework,
I want the project workspace to automatically maintain an up-to-date `_bmad/custom/config.user.toml` file,
So that BMad's native TOML configuration resolution scripts (`resolve_config.py`, `resolve_customization.py`) execute accurately with live project context without needing in-flight regex rewrites.

**Acceptance Criteria:**

**Given** a project workspace initialization in `ai/core/workspace.go`,
**When** `NewWorkspace()` or workspace preparation is executed,
**Then** the system MUST ensure the directory `<projectRoot>/_bmad/custom/` exists.

**Given** active session parameters (`userId`, `projectName`, `specLocation`, `planningArtifactsLocation`, `implementationArtifactsLocation`),
**When** seeding the project configuration,
**Then** the system MUST write or update `<projectRoot>/_bmad/custom/config.user.toml` containing:
```toml
[core]
user_name = "<user_name>"
project_name = "<project_name>"
output_folder = "{project-root}/<specLocation>"

[modules.bmm]
planning_artifacts = "{project-root}/<planningArtifactsLocation>"
implementation_artifacts = "{project-root}/<implementationArtifactsLocation>"
project_knowledge = "{project-root}/<specLocation>"
```

**Given** `_bmad/scripts/resolve_config.py` executed via `uv_run`,
**When** resolving `--key core`,
**Then** the script MUST output the merged JSON configuration reflecting the exact values seeded into `config.user.toml`.

**Given** automated unit tests in `ai/core/workspace_unit_test.go`,
**When** testing workspace seeding,
**Then** tests MUST verify correct TOML formatting, token placement, and idempotent updates.

### Story 16.4: Elimination of Brittle In-Flight Config Rewrites and Markdown AST Mutations

As a Core Platform Maintainer,
I want to eliminate in-flight regex rewrites and string mutations on markdown and configuration files,
So that BMad framework updates can be ingested cleanly without breaking interceptors or corrupting skill instructions.

**Acceptance Criteria:**

**Given** `ConfigRewriterInterceptor` in `ai/plugins/bmad/config_rewriter.go`,
**When** a `file_reader` tool call completes for any `SKILL.md` file,
**Then** the interceptor MUST NOT perform regex replacements or mutate the file content, returning the raw content directly.

**Given** `ConfigRewriterInterceptor`,
**When** reading configuration files (`_bmad/core/config.yaml`, `_bmad/bmm/config.yaml`, `_bmad/config.toml`, `customize.toml`),
**Then** the interceptor MUST NOT invoke `injectBmmConfiguration`, `injectBmadCoreConfiguration`, `injectTomlConfiguration`, or `addStringItemToTomlArray`.

**Given** `ai/plugins/bmad/interceptor.go`,
**When** configuring `BmadInterceptorPlugin`,
**Then** the interceptor pipeline MUST be streamlined to eliminate redundant interceptor passes.

**Given** test cases in `ai/plugins/bmad/interceptor_test.go`,
**When** running unit tests,
**Then** all tests MUST be updated to verify clean pass-through of file reads without regex mutation.

### Story 16.5: Consolidation of Path Safety, Sandboxing & Zero-Trust Boundary Enforcement

As a Security Architect and Developer,
I want a consolidated, zero-trust path safety and boundary enforcement layer,
So that path resolution, sandbox isolation, and content sanitization operate predictably in a single pass without circular path transformations.

**Acceptance Criteria:**

**Given** a tool call to `file_writer`,
**When** `PathSafetyInterceptor.Before()` runs,
**Then** it MUST validate that the target file path resides within `projectRoot/<specLocation>` or is an allowed root instruction file (`AGENTS.md`, `project-context.md`), preventing any out-of-sandbox write.

**Given** tool calls to `file_reader` or `directory_reader`,
**When** `PathSafetyInterceptor.Before()` runs,
**Then** it MUST verify the target path resides strictly within `projectRoot` or `bmadRoot`, blocking unauthorized system paths.

**Given** tool execution outputs,
**When** `ContentSanitizerInterceptor` or unified path interceptor processes returned file paths,
**Then** all paths MUST be sanitized to project-relative paths (or `bmadRoot`-relative paths) without leaking internal filesystem mount prefixes (`/mount/orgId/userId/projectId/`).

**Given** a successful `file_writer` execution,
**When** registering the write event,
**Then** `FileWriteNotifierInterceptor` MUST record a unique `file-write-notification:<uuid>` entry in `ctx.State()` containing the clean project-relative file path.

**Given** the test suite in `ai/plugins/bmad/interceptor_test.go`,
**When** running tests,
**Then** all path safety, sandbox jail, and sanitization tests MUST pass with 100% reliability.

### Story 16.6: End-to-End Regression Test Suite for Brainstorming Activation & TOML Resolution

As a Quality Engineer and Developer,
I want an end-to-end regression test suite verifying Brainstorm activation and native TOML configuration resolution,
So that all subagents execute BMad skills and Python utilities reliably without regression.

**Acceptance Criteria:**

**Given** a mock project workspace with `_bmad/` configuration and `skills/bmad-brainstorming/`,
**When** simulating `APF Brainstorm` activation,
**Then** the test MUST verify that `resolve_customization.py` and `resolve_config.py` execute successfully via `UVRunnerTool` and return valid merged configuration JSON.

**Given** a test workspace containing `.memlog.md` files in `docs/spec/`,
**When** the agent invokes `DirectoryReaderTool` with `docs/spec/*/.memlog.md`,
**Then** the tool MUST return all matching file paths accurately.

**Given** `UVRunnerTool`,
**When** passed invalid command binary names (`find`, `ls`, `python3`),
**Then** the test MUST verify immediate rejection with clear guidance without launching subprocesses.

**Given** the entire AI package test suite (`go test -v ./ai/...`),
**When** running the automated tests,
**Then** 100% of tests MUST pass cleanly.


## Epic 17: Multi-Provider Email Notification Platform & Custom Notification Provider (CNP) Integration

Establish a modular, provider-agnostic notification adapter subsystem supporting Custom Notification Provider (CNP), local Console, and Mock providers, enabling secure, authenticated, and observable email notification dispatch with automated infrastructure provisioning.

### Story 17.1: Generic Email Notification Adapter Subsystem & Provider Registry

As a Developer and System Architect,
I want a generic email notification adapter interface (`EmailNotificationProvider`) and thread-safe registry in `api/notifications/`,
So that the platform can swap out and route email notifications to different underlying transports (CNP, Console, Mock) cleanly without coupling high-level business logic to concrete provider implementations.

**Acceptance Criteria:**

**Given** the `api/notifications/` package,
**When** defining the notification provider abstraction in `provider.go`,
**Then** it MUST declare:
- `type SendStatus string` with constants `SendStatusSuccess`, `SendStatusDuplicate`, `SendStatusError`.
- `var ErrProviderNotFound = errors.New("notification provider not found")`.
- `type EmailNotificationProvider interface` with methods:
  - `Name() string`
  - `Send(ctx context.Context, notif *EmailNotification) (*SendResult, error)`
  - `ValidateConfig() error`

**Given** a notification dispatch request,
**When** creating an `EmailNotification` struct,
**Then** it MUST support normalized fields: `ToEmail` (string), `ToName` (string), `Subject` (string), `HTMLContent` (string), `TextContent` (string), `RequestID` (string), and `Metadata` (map[string]interface{}).

**Given** a completed notification dispatch,
**When** receiving a `SendResult` struct,
**Then** it MUST contain normalized fields: `Provider` (string), `MessageID` (string), `Status` (`SendStatus`), `SenderServiceAccount` (string), `Timestamp` (time.Time), and `Details` (map[string]interface{}).

**Given** the provider registry in `api/notifications/provider.go`,
**When** registering and resolving providers,
**Then** functions `RegisterProvider(provider)`, `GetProvider(name)`, `SetDefaultProvider(provider)`, `GetDefaultProvider()`, `ListRegisteredProviders()`, and `ResetRegistry()` MUST operate thread-safely with `sync.RWMutex`. Provider names MUST be normalized via `strings.ToLower(strings.TrimSpace(name))`. If an unknown provider is queried, `GetProvider(name)` returns `ErrProviderNotFound`. If no default provider is configured, `GetDefaultProvider()` falls back to `ConsoleEmailProvider`.

**Given** local development or testing without active cloud credentials,
**When** initializing the `ConsoleEmailProvider` (`console_provider.go`),
**Then** it MUST log structured key-value attributes (`to`, `toName`, `subject`, `requestId`, `htmlContentLength`, `textContentLength`, `preview`) to `slog.Info` and return a successful `SendResult` with `Status: SendStatusSuccess`.

**Given** unit and integration testing environments,
**When** initializing `MockEmailProvider` (`mock_provider.go`),
**Then** it MUST capture sent notifications in-memory in a thread-safe slice, support inspecting sent messages via `SentNotifications()`, clearing history via `Clear()`, and injecting simulated errors via `SetError(err)`.

**Given** legacy email callers using `core.EmailService`,
**When** `core.InitializeEmailService()` runs,
**Then** `core.EmailServiceInstance` MUST wrap and delegate to the active default `EmailNotificationProvider`, and `SendGridEmailService` / `github.com/sendgrid/sendgrid-go` MUST be removed.

**Given** automated unit tests in `api/notifications/provider_test.go`,
**When** running `go test -v ./api/notifications/...`,
**Then** 100% of registry, console provider, and mock provider unit tests MUST pass cleanly.

### Story 17.2: Custom Notification Provider (CNP) Provider Implementation & AuthN/AuthZ Impersonation

As a System Architect and Backend Engineer,
I want a specialized `CNPEmailProvider` for Custom Notification Provider (CNP) Admit API with IAM service account impersonation, in-memory token caching, and deduplication handling,
So that the API tier can securely deliver enterprise email campaigns through CNP within Google Cloud infrastructure.

**Acceptance Criteria:**

**Given** the `CNPEmailProvider` in `api/notifications/cnp_provider.go`,
**When** validating configuration via `ValidateConfig()`,
**Then** it MUST verify that `ProjectID` and `SenderServiceAccount` are non-empty and well-formed.

**Given** a valid `EmailNotification` with direct API-rendered content (`Subject`, `HTMLContent`, optional `TextContent`),
**When** `CNPEmailProvider.Send()` serializes the CNP Admit payload,
**Then** it MUST construct a `CNPAdmitRequest` JSON payload matching the CNP proto schema:
- `campaign_identifiers.project_id` = `config.ProjectID`
- `campaign_identifiers.campaign_id` = `config.DefaultCampaign`
- `recipient.email_user.email_address` = `notif.ToEmail`
- `rendering_options.rendered_message.email_message.subject` = `notif.Subject`
- `rendering_options.rendered_message.email_message.html_body` = `notif.HTMLContent`
- `rendering_options.rendered_message.email_message.text_body` = `notif.TextContent`

**Given** CNP service account authentication on Cloud Run,
**When** generating OAuth2 access tokens for `NOTIFICATIONS_SENDER_SERVICE_ACCOUNT`,
**Then** the provider MUST use Google Application Default Credentials (ADC) with IAM token generation and wrap the token source in `oauth2.ReuseTokenSource` for in-memory token reuse and automatic pre-expiry refresh.

**Given** a dispatch without an explicit `RequestID`,
**When** sending via CNP,
**Then** the provider MUST auto-generate a unique UUID v4 `request_id` for deduplication.

**Given** an upstream CNP response returning `ALREADY_EXISTS` (HTTP 409 status code),
**When** handling the response,
**Then** the provider MUST treat it as an idempotent success and return `&SendResult{Status: "DUPLICATE", MessageID: reqID}` without raising an error.

**Given** an upstream CNP response returning `PERMISSION_DENIED` (HTTP 403) or `UNAUTHENTICATED` (HTTP 401),
**When** handling the response,
**Then** the provider MUST return a descriptive error detailing the authorization or token validation failure.

**Given** `NOTIFICATIONS_ENVIRONMENT=local`,
**When** `CNPEmailProvider.Send()` runs,
**Then** it MUST execute in offline simulated mode, validating sender service account format and message structure without making network calls.

**Given** unit tests in `api/notifications/cnp_provider_test.go`,
**When** running `go test -v ./api/notifications/...`,
**Then** tests MUST verify payload serialization, AuthN bearer headers, AuthZ failure handling, deduplication idempotency, and simulated mode.

### Story 17.3: API Notification Handler, Routing & OpenTelemetry Tracing

As an API Client and Developer,
I want a RESTful endpoint `POST /api/v1/notifications/email` and administrative discovery endpoint `GET /api/v1/notifications/providers`,
So that authenticated organization members can dispatch email notifications with strict payload validation, context timeout management, fail-fast error handling, and OpenTelemetry observability.

**Acceptance Criteria:**

**Given** the `NotificationHandler` in `api/notification_handlers.go`,
**When** instantiated,
**Then** its constructor MUST strictly require dependency injection: `NewNotificationHandler(provider notifications.EmailNotificationProvider, timeout time.Duration)`.

**Given** an incoming request to `POST /api/v1/notifications/email`,
**When** reading the request body,
**Then** the handler MUST enforce a 1MB payload size limit via `http.MaxBytesReader(w, r.Body, 1<<20)`, returning `400 Bad Request` if the body exceeds 1MB.

**Given** request payload validation,
**When** validating fields,
**Then** the handler MUST enforce:
- `toEmail`: Required valid RFC 5322 email string (single recipient only).
- `subject`: Required non-empty string.
- `htmlContent`: **Mandatory** non-empty HTML string.
- Rejecting or returning `400 Bad Request` with structured error messages for missing or invalid fields.

**Given** request execution,
**When** dispatching via the injected provider,
**Then** the handler MUST strictly use the configured provider (disallowing client-specified overrides) and derive a timeout context via `context.WithTimeout(r.Context(), h.timeout)`.

**Given** an active notification dispatch,
**When** processing the request,
**Then** the handler MUST create an OpenTelemetry span `notifications.send_email` recording attributes `notification.provider`, `notification.recipient_domain`, `notification.subject_length`, and `notification.status`.

**Given** upstream provider errors or timeouts,
**When** dispatch fails,
**Then** the handler MUST fail fast, record the error in the active OTel span, and return `502 Bad Gateway` (or `500 Internal Server Error`) with structured JSON error details.

**Given** a successful or duplicate dispatch,
**When** provider returns `SendResult`,
**Then** the handler MUST return `200 OK` with JSON `{ "status": "SUCCESS" | "DUPLICATE", "message": "Email notification processed", "result": result }`.

**Given** a request to `GET /api/v1/notifications/providers`,
**When** the caller is authenticated,
**Then** the handler MUST verify that the user is an Org Admin, Org Owner, or SuperAdmin (`RoleOrgAdmin`, `RoleOrgOwner`, SuperAdmin), returning `403 Forbidden` for regular members, and `200 OK` with active provider name and registered providers list for authorized roles.

**Given** route registration in `api/routes.go`,
**When** registering routes,
**Then** `POST /api/v1/notifications/email` and `GET /api/v1/notifications/providers` MUST be wrapped with `authMiddleware` and `orgScopeMiddleware`, with CORS preflight `OPTIONS` registered.

**Given** API documentation in `api/openapi.yaml`,
**When** inspecting endpoint definitions,
**Then** `/api/v1/notifications/email` and `/api/v1/notifications/providers` MUST be fully documented with request/response schemas and authentication requirements.

**Given** unit tests in `api/notification_handlers_test.go`,
**When** running `go test -v -run TestNotificationHandler ./api/...`,
**Then** 100% of handler tests (validation, auth/orgScope gating, provider dispatch, error handling) MUST pass cleanly.

### Story 17.4: Notifications Infrastructure Provisioning, IAM Roles & Cloud Deploy Pipeline

As a DevOps and Infrastructure Engineer,
I want Terraform infrastructure definitions, IAM role bindings, and Cloud Deploy manifests for notification services,
So that the dedicated notification sender service account and environment configurations are securely provisioned and deployed across staging and production environments.

**Acceptance Criteria:**

**Given** `infra/modules/service_accounts.tf`,
**When** applying Terraform,
**Then** it MUST provision a dedicated service account `google_service_account.notifications_sender` with `account_id = "${var.prefix}-notifications-sender"`.

**Given** `infra/modules/service_accounts.tf`,
**When** configuring IAM permissions,
**Then** it MUST bind `roles/iam.serviceAccountTokenCreator` on `google_service_account.notifications_sender` to `serviceAccount:${google_service_account.compute.email}`.

**Given** `infra/modules/variables.tf` and `infra/modules/outputs.tf`,
**When** configuring module interfaces,
**Then** it MUST declare variables `notifications_project_id`, `notifications_campaign_id`, `notifications_service_endpoint`, `email_service_type`, and export output `notifications_sender_service_account_email`.

**Given** `infra/cicd/manifests/api.yaml`,
**When** deploying the API Cloud Run service,
**Then** container environment variables MUST include:
- `EMAIL_SERVICE_TYPE`: `placeholder-email-service-type` (`${email_service_type}`)
- `NOTIFICATIONS_PROJECT_ID`: `placeholder-notifications-project-id` (`${notifications_project_id}`)
- `NOTIFICATIONS_CAMPAIGN_ID`: `placeholder-notifications-campaign-id` (`${notifications_campaign_id}`)
- `NOTIFICATIONS_SENDER_SERVICE_ACCOUNT`: `placeholder-notifications-sender-sa` (`${notifications_sender_service_account}`)
- `NOTIFICATIONS_SERVICE_ENDPOINT`: `placeholder-notifications-endpoint` (`${notifications_service_endpoint}`)
- `NOTIFICATIONS_ENVIRONMENT`: `placeholder-notifications-env` (`${notifications_environment}`)

**Given** `infra/cicd/clouddeploy.yaml`,
**When** configuring delivery targets,
**Then** `apf-staging` and `apf-production` targets MUST include deploy parameters for `email_service_type`, `notifications_project_id`, `notifications_campaign_id`, `notifications_sender_service_account`, `notifications_service_endpoint`, and `notifications_environment`.

**Given** environment templates in `config.env.template` and `api/.env`,
**When** configuring local environments,
**Then** template variables for `EMAIL_SERVICE_TYPE=CONSOLE`, `NOTIFICATIONS_PROJECT_ID`, `NOTIFICATIONS_SENDER_SERVICE_ACCOUNT`, `NOTIFICATIONS_SERVICE_ENDPOINT`, and `NOTIFICATIONS_ENVIRONMENT=local` MUST be present.

**Given** CNP MDB group configuration,
**When** onboarding the project in CNP,
**Then** documentation MUST specify adding `${prefix}-notifications-sender@${project_id}.iam.gserviceaccount.com` as `child_user` in `cnp-<notifications_project_id>-api-senders.prod`.


## Epic 18: Unified File Upload, Multi-Environment Artifact Management, Multimodal Chat Context & File Explorer Management

Empower users to upload text files, documents, and images through both the chat interface and the project tree. Chat uploads leverage a pluggable ArtifactService (persisting to GCS in Staging/Production cloud environments and local filesystem for local development, initialized via mise.toml). Text documents are extracted into prompt text, while non-text files/images (.png, .jpg, .jpeg, .webp, .gif) are fed directly into the model as multipart / multimodal content parts. In the chat view, sessions with attached artifacts display an expandable chevron indicator and a dropdown displaying all session artifacts. Clicking an image artifact opens a dedicated Image Viewer in the preview pane that renders the image cleanly while suppressing code editor (Monaco) controls, edit tools, and specification approval buttons. The File Explorer provides intuitive folder actions: on mouseover, folder nodes reveal `+` (Upload file to folder), Refresh, and Delete icons, while the "All Files" header includes a `+` icon to upload directly to the project root.

### Story 18.1: Multi-Tier `ArtifactService` & Configuration Infrastructure (Backend & Mise)

As a Developer / DevOps Engineer,
I want the backend API and environment configuration to support multi-tier artifact persistence across cloud and local environments,
So that chat attachments are stored in GCS in cloud deployments and on the local filesystem in local development.

**Acceptance Criteria:**

- **Given** the application configuration in `api/core/configuration.go`
  **When** environment variables are loaded
  **Then** the system MUST read `GCS_ARTIFACT_BUCKET` (reintroduced from previous cleanup) and fallback gracefully if unset
  **And** in local mode (`APP_ENV=local` or unset), the system MUST read `LOCAL_ARTIFACT_PATH` defaulting to `.foundry/artifacts` or the mounted workspace path.
- **Given** the local developer setup in `mise.toml`
  **When** `mise run dev` or local setup tasks execute
  **Then** the environment configuration MUST initialize and export `LOCAL_ARTIFACT_PATH` along with `WORKSPACE_MOUNT_PATH`.
- **Given** an authenticated user uploading a file via `POST /api/v1/projects/{projectId}/sessions/{sessionId}/artifacts/upload`
  **When** running in Staging or Production
  **Then** the `ArtifactService` MUST stream and persist the file to the configured GCS bucket under `projects/{projectId}/sessions/{sessionId}/{filename}`
  **And** when running in Local mode, the `ArtifactService` MUST write the file to `{LOCAL_ARTIFACT_PATH}/{projectId}/{sessionId}/{filename}`.
- **Given** a successful file upload to the `ArtifactService`
  **When** the write completes
  **Then** the backend MUST record an artifact metadata document in Firestore under `artifacts` collection containing `id` (UUID string), `projectId` (string), `sessionId` (string), `name` (original filename), `mimeType` (detected MIME string, e.g. `image/png`, `application/pdf`, `text/markdown`), `size` (int64 bytes), `storageType` (`"gcs"` or `"local_file"`), `storagePath` (GCS URI or local path), and `createdAt` (ISO 8601 timestamp)
  **And** the API MUST return HTTP 201 with the created artifact metadata object.

### Story 18.2: Chat File & Multimodal Image Upload UI (Frontend)

As a User (PM, Designer, Developer),
I want to attach documents, code snippets, and images in the chat prompt input bar,
So that I can supply immediate context and visual references to the active AI agent.

**Acceptance Criteria:**

- **Given** the chat panel in the GUI Workspace
  **When** the user looks at the chat input area
  **Then** the UI MUST display a file attachment button (clip/paperclip icon) next to the message input.
- **Given** the user clicks the attachment button or drags-and-drops files onto the chat input area
  **When** selecting files from their local device
  **Then** the file picker MUST accept text/code formats (`.txt`, `.md`, `.pdf`, `.csv`, `.json`, `.yaml`, `.py`, `.go`, `.ts`, `.tsx`, `.js`, `.html`) and image formats (`.png`, `.jpg`, `.jpeg`, `.webp`, `.gif`, `.svg`)
  **And** enforce a maximum file size limit of 10MB per file with immediate toast validation if exceeded.
- **Given** valid files are attached
  **When** staged prior to sending
  **Then** the chat bar MUST render removable badge chips for each attached file displaying:
    - File icon (or thumbnail for images)
    - File name and formatted size (e.g. `diagram.png (1.2 MB)`)
    - An `X` remove button to detach the file before dispatch.
- **Given** attached files in the chat bar
  **When** the user clicks Send
  **Then** the UI MUST upload the files to the session upload endpoint and attach the resulting artifact IDs / metadata to the message turn payload.

### Story 18.3: Multimodal & Text Prompt Context Ingestion (AI Orchestrator)

As an AI Agent / Orchestrator,
I want to receive uploaded files as either extracted text or multimodal content parts,
So that I can reason over both textual documentation and visual diagrams in my responses.

**Acceptance Criteria:**

- **Given** a chat message with attached text or code artifacts (`.txt`, `.md`, `.json`, `.yaml`, `.py`, `.go`, etc.)
  **When** the AI orchestrator receives the chat turn request
  **Then** the backend MUST read the text content and inject it into the prompt context under formatted demarcation blocks (e.g., `[Attached File: filename.md]\n...content...`).
- **Given** a chat message with attached image artifacts (`.png`, `.jpg`, `.jpeg`, `.webp`, `.gif`)
  **When** the AI orchestrator prepares the model payload for Gemini
  **Then** the system MUST construct multimodal `Part` objects with `InlineData` (base64-encoded bytes and MIME type) or authenticated storage references
  **And** pass the image parts directly to the Gemini model call alongside the text prompt.
- **Given** an unsupported binary file or corrupted image
  **When** content extraction fails
  **Then** the system MUST return a structured warning to the user turn without crashing the agent execution stream.

### Story 18.4: File Explorer Folder Actions & Project Tree Uploads (Full-Stack)

As a Developer / User,
I want to upload files directly into specific folders in the File Explorer and manage folder subtrees,
So that I can add assets directly to the workspace filesystem and commit them with code.

**Acceptance Criteria:**

- **Given** the File Explorer header section displaying "All Files"
  **When** viewed in the Explorer tab
  **Then** the header MUST render a `+` (Upload to Root) action button.
  **And** clicking this button MUST open a file picker that uploads the selected file(s) directly to the root directory of the workspace project tree.
- **Given** the File Explorer directory tree
  **When** the user hovers / mouseovers any folder node
  **Then** the folder node MUST reveal three action icons on the right:
    1. `+` (Add / Upload File): Opens file upload dialog targeting this specific folder path.
    2. Refresh icon: Re-fetches and re-renders only that specific folder subtree.
    3. Delete icon (trash can): Opens a confirmation modal to delete the folder and its contents.
- **Given** the user selects files via the folder node `+` button or drags files onto a folder node
  **When** upload is confirmed
  **Then** `POST /api/v1/projects/{projectId}/files/upload?targetPath={folderPath}` MUST write the files to `{WORKSPACE_ROOT}/{folderPath}/{filename}`
  **And** the backend MUST enforce strict path traversal sanitization (`filepath.Clean`) rejecting any paths attempting to escape `{WORKSPACE_ROOT}`
  **And** the File Explorer subtree MUST immediately refresh to display the newly added files.
- **Given** the user confirms folder deletion
  **When** `DELETE /api/v1/projects/{projectId}/folders?path={folderPath}` executes
  **Then** the folder MUST be removed from the workspace filesystem and the File Explorer tree updated accordingly.

### Story 18.5: Chat Session Collapsible Chevron & Artifacts Dropdown (Frontend)

As a User,
I want to see which chat sessions contain uploaded artifacts via an expandable chevron and dropdown,
So that I can quickly browse, view, and reuse files associated with any conversation.

**Acceptance Criteria:**

- **Given** the Chat history / Session list sidebar
  **When** a chat session has one or more associated artifacts registered in Firestore
  **Then** the session item MUST render an expandable chevron indicator on the right.
  **And** sessions without any artifacts MUST NOT render an expandable chevron.
- **Given** a session with artifacts
  **When** the user clicks the session or clicks the chevron indicator
  **Then** the session item MUST toggle expand/collapse state with a smooth CSS transition
  **And** when expanded, render a dropdown / nested sub-list of all artifacts associated with that session.
- **Given** the expanded session artifact dropdown
  **When** rendered
  **Then** each artifact item MUST display:
    - File type icon (or image thumbnail for image files)
    - File name
    - File size (formatted)
    - An action trigger to open/preview the artifact in the workspace preview pane.

### Story 18.6: Image Artifact Preview & Media Viewer (Frontend)

As a User,
I want to view image-type artifacts in a dedicated viewer in the preview pane without code editor controls,
So that I can inspect images clearly without irrelevant editor buttons.

**Acceptance Criteria:**

- **Given** an image artifact (`.png`, `.jpg`, `.jpeg`, `.gif`, `.webp`, `.svg`) is clicked from the Session Artifacts dropdown or Artifact Sidebar
  **When** the preview pane opens
  **Then** the system MUST mount an `ImageViewer` component displaying the image with zoom controls, pan support, and responsive aspect ratio fitting.
- **Given** the preview pane is rendering an image artifact
  **When** displayed to the user
  **Then** the Monaco code editor, text line numbers, syntax highlighting toolbar, and specification "Approve" / "Edit" buttons MUST BE HIDDEN
  **And** the header MUST display the image filename, dimensions (if available), file size, and a "Download" action button.
- **Given** non-image artifacts (e.g., markdown specifications or code files)
  **When** clicked
  **Then** standard Monaco/Markdown rendering with specification review and approval controls MUST continue to function as normal.

## Epic 19: Development Sandbox Builder & Runtime Verification Harness

Empower developers and autonomous agents to trigger an interactive sandbox build workflow directly from the Implementation Tab, dynamically synthesize deterministic containerized runtime specifications (`Dockerfile`) by scanning project specifications and context files (`project-context.md`, `AGENTS.md`, `architecture.md`, etc.), review and customize detected runtime dependencies in an interactive table UI, and automatically build, validate, test, and ephemerally deploy the resulting sandbox container to verify runtime dependencies, toolchains, and environment health.

### Story 19.1: Implementation Tab "Build Sandbox" Action & Stack Discovery Skill Invocation

As a developer using the Implementation Tab,
I want a dedicated "Build Sandbox" action button positioned beneath the chat input textbox,
So that I can trigger the sandbox builder workflow to analyze my project stack without manual CLI commands.

**Acceptance Criteria:**

- **Given** a user is on the Implementation Tab workspace
  **When** viewing the chat prompt interface
  **Then** a primary action button labeled "Build Sandbox" (with a container/box icon) MUST be rendered directly beneath the chat message input textarea.
- **Given** an active chat message or tool execution is in progress
  **When** the user views the "Build Sandbox" button
  **Then** it MUST be disabled with a tooltip indicating "Agent execution in progress".
- **Given** the workspace is idle
  **When** the user clicks "Build Sandbox"
  **Then** the UI MUST initiate the `bmad-sandbox-builder` skill/agent execution via the platform's backend orchestrator API (`POST /api/v1/sessions/{sessionId}/sandbox/discover` or chat turn dispatch), displaying an active scanning state with a progress spinner.
- **Given** the skill scan completes
  **When** the discovery payload is returned
  **Then** the UI MUST automatically open the Sandbox Stack Configuration modal populated with discovered dependencies.

### Story 19.2: Autonomous Specification Scanning & Runtime Stack Discovery Agent (`bmad-sandbox-builder`)

As an autonomous Sandbox Builder Agent (`bmad-sandbox-builder`),
I want to parse and inspect accumulated project specifications, architecture files, and root instructions (`project-context.md`, `AGENTS.md`, `architecture.md`, `prd.md`, package manifests),
So that I can identify the complete runtime stack, language toolchains, system packages, and required execution environments for the project.

**Acceptance Criteria:**

- **Given** the `bmad-sandbox-builder` agent/skill is invoked
  **When** it executes its discovery routine
  **Then** it MUST read and parse `project-context.md`, `AGENTS.md`, `docs/spec/planning-artifacts/architecture.md`, `docs/spec/planning-artifacts/prd.md`, and any existing dependency manifests (`go.mod`, `package.json`, `pyproject.toml`, `requirements.txt`).
- **Given** extracted project context
  **When** analyzing technical invariants
  **Then** the agent MUST categorize runtime requirements into:
    1. Primary Language Runtime & SDK (e.g., Go 1.22+, Node.js 20+, Python 3.11+)
    2. Build & Package Managers (e.g., `uv`, `pnpm`, `npm`, `go`, `pip`)
    3. System Dependencies & OS Libraries (e.g., `curl`, `git`, `build-essential`, `ca-certificates`)
    4. Service Dependencies / Mocks (e.g., local Firestore emulator, in-memory databases, Redis)
    5. Environment Variables & Default Ports (e.g., `PORT=8080`, `NODE_ENV=development`)
- **Given** ambiguous or unmentioned versions in specifications
  **When** building the stack manifest
  **Then** the agent MUST assign stable LTS versions and flag them with a status of `"detected_default"` with confidence notes.
- **Given** completion of the scan
  **Then** the agent MUST return a structured JSON stack manifest payload conforming to the Sandbox Stack Schema (`category`, `name`, `version`, `sourceDoc`, `isOptional`, `installCommand`).

### Story 19.3: Interactive Sandbox Stack Configuration Table UI (Add / Edit / Remove Items)

As a developer configuring my project sandbox,
I want an interactive modal displaying discovered runtime libraries and tools in a structured table where I can add, edit, or remove items,
So that I can tailor and approve the runtime configuration before the Dockerfile and container are built.

**Acceptance Criteria:**

- **Given** the stack discovery completes
  **When** the Sandbox Stack Configuration modal renders
  **Then** it MUST display an interactive table with columns: Category, Component/Package, Version/Tag, Source (e.g. `architecture.md:L42` or `detected`), and Actions.
- **Given** the table is rendered
  **When** the user clicks "Add Component"
  **Then** an inline row or modal form MUST appear allowing entry of Category (Dropdown: Runtime, Package Manager, System Lib, Service), Name, Version, and Optional Install Command.
- **Given** any existing row
  **When** the user clicks the "Edit" action
  **Then** the version and parameters for that item MUST become editable inline with validation checks.
- **Given** any row
  **When** the user clicks the "Remove" (trash icon) action
  **Then** that component MUST be deleted from the sandbox stack manifest with an undo snackbar notice.
- **Given** the user is satisfied with the stack inventory
  **When** clicking "Confirm & Build Sandbox"
  **Then** the finalized manifest MUST be submitted to the backend API (`POST /api/v1/sessions/{sessionId}/sandbox/build`) to initiate Dockerfile generation and test harness execution.

### Story 19.4: Autonomous Dockerfile Synthesis & Sandbox Plan Generation

As an autonomous Sandbox Builder Agent,
I want to synthesize an optimized, multi-stage `Dockerfile` and sandbox runtime execution plan from the confirmed stack manifest,
So that the container image encapsulates all required runtimes, build tools, system packages, and working directory layouts deterministically.

**Acceptance Criteria:**

- **Given** the confirmed sandbox stack manifest
  **When** the agent generates the `Dockerfile`
  **Then** it MUST construct a clean multi-stage or layered `Dockerfile` using official minimal base images (e.g., `alpine`, `distroless`, `debian-slim`, or `cgr.dev/chainguard`).
- **Given** multi-runtime projects (e.g. Go backend + Next.js frontend + Python agents)
  **When** generating build layers
  **Then** the Dockerfile MUST install all verified language toolchains, package managers, and system dependencies with non-root security principles (`USER apf`) where appropriate.
- **Given** the generation process
  **When** writing files
  **Then** the agent MUST save the synthesized `Dockerfile` and accompanying `.dockerignore` to the project's sandbox directory (`docs/spec/sandbox/Dockerfile` or workspace root) and register them as workspace artifacts.
- **Given** generation completion
  **Then** the agent MUST produce a structured `sandbox-plan.json` listing all verification assertions (binary names, `--version` commands, test endpoints) to be executed by the test harness.

### Story 19.5: Isolated Container Build Engine & Verification Test Harness

As the platform backend API,
I want a secure, isolated container build engine and verification test harness,
So that I can build the generated `Dockerfile` hermetically, capture build telemetry, and enforce execution guardrails without compromising host security.

**Acceptance Criteria:**

- **Given** a request to build a sandbox image
  **When** the backend build engine runs (`POST /api/v1/sessions/{sessionId}/sandbox/build`)
  **Then** it MUST execute `docker build` / container builder inside an isolated sandbox context with memory and CPU quota limits.
- **Given** active image building
  **When** stdout and stderr chunks are produced
  **Then** the build engine MUST stream the raw log output in real time over Server-Sent Events (SSE) or WebSocket to the client.
- **Given** a build duration exceeding the configured timeout (`NFR-19.2`, default 5 minutes)
  **When** timeout triggers
  **Then** the build harness MUST cancel the process gracefully, terminate container workers, and return a structured `BUILD_TIMEOUT` error.
- **Given** a successful image build
  **Then** the harness MUST tag the image with a deterministic local identifier (`apf-sandbox-{projectId}-{buildId}:latest`) and transition to verification testing.

### Story 19.6: Container Runtime Health Sweeps, Ephemeral Deployment & Live Diagnostic Log Streaming UI

As a developer and verification harness,
I want to execute automated health checks and runtime assertions against the built sandbox container in an ephemeral deployment and view live streaming diagnostics in the UI,
So that I have verified proof that all requested tools and runtime environments are operating correctly in the container.

**Acceptance Criteria:**

- **Given** a built sandbox container image
  **When** the test harness initiates runtime verification
  **Then** it MUST execute every test assertion defined in `sandbox-plan.json` (e.g., `node --version`, `go version`, `python3 --version`, `uv --version`, library import tests).
- **Given** verification assertions running
  **When** any required binary or toolchain fails or returns non-zero
  **Then** the test harness MUST report a structured failure diagnostics report identifying the missing component and suggested remediation.
- **Given** binary assertions pass
  **When** the ephemeral container is launched for smoke testing
  **Then** the harness MUST start the container with mock environment variables, test healthcheck endpoints, and capture stdout/stderr runtime logs.
- **Given** the frontend Sandbox Build modal
  **When** tests execute
  **Then** the UI MUST display real-time checkmarks/spinners for each runtime dependency and stream live container execution logs in an embedded terminal viewer.
- **Given** test completion (or user modal dismissal / cancellation)
  **Then** the backend harness MUST stop and remove the ephemeral container within < 10 seconds (`NFR-19.3`), leaving no orphaned processes or dangling containers.


## Sprint Plan

This section maps the stories defined above into execution Sprints based on the agreed-upon phased approach.

### Sprint 1: Core Workspace (The MVP)
- **Story 7.1: Project Lifecycle States & Dashboard Action Buttons** (Epic 7) - *Note: Initial project state and UI onboarding cues.*
- **Story 7.2: Background Discovery Agent Stack Analysis** (Epic 7) - *Note: Core discovery scan agent backend.*
- **Story 1.2: Agent Sidebar Chat** (Epic 1)
- **Story 1.3: Secure Canvas Workspace Infrastructure** (Epic 1) - *Note: This is foundational infrastructure.*

### Sprint 2: Governance & Complexity
- **Story 7.3: Legacy Specification Audit & Ingestion Picker** (Epic 7) - *Note: User selections picker for detected specs.*
- **Story 7.4: Explicit Onboarding Completion Gating** (Epic 7) - *Note: Concluding onboarding and status transition.*
- **Story 1.1: Interactive Node Grid Layout** (Epic 1)
- **Story 2.1: Idea-to-Production Pipeline Tracking Bar** (Epic 2)
- **Story 2.2: Live Cross-Agent Activity Feed** (Epic 2)
- **Story 2.3: User-Level Secrets Management Settings Screen** (Epic 2)
- **Story 2.4: Project-Level App Integration Dynamic Settings** (Epic 2)
- **Story 6.1: Audit Trail Event Capture** (Epic 6) - *Note: Required for governance.*
- **Story 4.3: Automated Pull Request Handoff and Remote VCS Integration** (Epic 4) - *Note: Critical dependency for Story 1.1 Handoff CTA.*
- **Story 4.4: Git-to-Workspace Spec Re-Hydration and Edit Sync** (Epic 4) - *Note: Required for spec-level revisions.*
- **Story 4.5: Git Commit Safety Guardrails** (Epic 4) - *Note: Core governance guardrails on commit.*
- **Story 4.7: Centralize Git and Version Control in API Tier** (Epic 4) - *Note: Migrate and centralize all git operations to the API tier.*
- **Story 5.1: Python Antigravity Chat Inference Engine** (Epic 5) - *Note: Dynamic python chat core.*
- **Story 5.2: Go Dynamic Reverse-Proxy AI Gateway** (Epic 5) - *Note: Essential for dynamic proxy gateway routing.*
- **Story 1.5: Cancel Chat Message** (Epic 1) - *Note: Allow users to cancel active chat processing.*

### Sprint 3: Polish & Integrations
- **Story 1.4: Contextual Artifact Sidebar** (Epic 1)
- **Story 3.2: Native IDP Schema Connector** (Epic 3)
- **Story 3.3: AntiGravity JSON-RPC Hydration Outpost** (Epic 3)
- **Story 4.1: Git State Sync Tracker** (Epic 4)
- **Story 4.2: Tabbed Workspace Explorer: Artifacts and File Explorer Views** (Epic 4)
- **Story 4.8: On-Demand Recursive File Explorer Component and API** (Epic 4)
- **Story 4.6: File-to-Database Artifact Synchronization** (Epic 4)
- **Story 5.3: Antigravity Dynamic Tool Execution Adapter** (Epic 5) - *Note: Reverse tool adapters parity under agy-sdk.*

### Sprint 4: Context & File Upload (MVP)
- **Story 8.1: Chat File Upload UI (MVP)** (Epic 8)
- **Story 8.2: Backend File Upload API (MVP)** (Epic 8)
- **Story 8.3: Contextual Artifact Sidebar & DB Tracking (MVP)** (Epic 8)
- **Story 8.4: Agent Context Ingestion (MVP)** (Epic 8)

### Sprint 5: Enhanced Onboarding Tour
- **Story 9.1: Onboarding Tour State & Persistence (Backend/Firestore)** (Epic 9)
- **Story 9.2: Tour Integration for Newly Created Sections (Frontend)** (Epic 9)
- **Story 9.3: Advanced Navigation: Skip Section and Skip Tour (Frontend)** (Epic 9)
- **Story 9.4: Tour Resiliency & Action Gating (Frontend/Backend)** (Epic 9)
- **Story 9.5: Context-Aware Auto-Start & Triggers (Frontend)** (Epic 9)
- **Story 9.6: Seeded Sample Projects for Reference (Frontend/Backend)** (Epic 9)

### Sprint 6: Multi-Tenancy Hard Boundaries
- **Story 10.1: Data Partitioning by Organization-ID Across All Storage Mediums** (Epic 10)
- **Story 10.2: Organization-Aware Non-Administrative (Data Plane) APIs** (Epic 10)
- **Story 10.3: Organization Provisioning & Onboarding (Administrative API)** (Epic 10)
- **Story 10.4: Organization Environment Configuration & User Management** (Epic 10)
- **Story 10.5: Multi-Tenant Login & Organization Selection UI Flow** (Epic 10)
- **Story 10.6: Active Organization Switcher & Multi-Tenant Developer Simulation Panel** (Epic 10)
- **Story 10.7: Super Admin Autojoin Configuration and Auto-Provisioning** (Epic 10)

### Sprint 7: Enterprise Identity & Access Control
- **Story 11.1: Google Identity Platform Configuration & Infrastructure Provisioning** (Epic 11)
- **Story 11.2: Identity-Aware Proxy (IAP) Token Validation Middleware** (Epic 11)
- **Story 11.3: System Administration - Organization Provisioning Screen (System Admin Portal)** (Epic 11)
- **Story 11.4: Organization Settings - Member Directory & User Management Screen (Org Admin Portal)** (Epic 11)
- **Story 11.5: Organization Settings - Invite Member Modal & Invitation Flow** (Epic 11)
- **Story 11.6: Organization Settings - Roles & Permissions Configuration Screen (Org Admin Portal)** (Epic 11)
- **Story 11.7: Invited User Experience - Tenant-Scoped Login & Invitation Acceptance Page** (Epic 11)
- **Story 11.8: Parent Project Super Admin Authentication & Global Resource Authorization** (Epic 11)
- **Story 11.9: Multi-Tenant Identity Verification and Fail-Safe Security Context Propagation** (Epic 11) - *Note: Addresses security and validation risks in production deployment.*
- **Story 11.10: Application Admin Performance & Operational Telemetry Dashboard UI** (Epic 11) - *Note: Admin console performance monitoring screen.*

### Sprint 8: Mobile Application Experience
- **Story 13.1: Mobile Projects Dashboard Screen** (Epic 13)
- **Story 13.2: Mobile Agent Chat Workspace Screen** (Epic 13)
- **Story 13.3: Mobile Touch Canvas & Node Workflow Screen** (Epic 13)
- **Story 13.4: Mobile Workspace Explorer & Spec Viewer Screen** (Epic 13)
- **Story 13.5: Mobile User Settings & API Secrets Screen** (Epic 13)
- **Story 13.6: Mobile Product Onboarding Tour Screen & Overlays** (Epic 13)
- **Story 13.7: Mobile Push Notifications & Task Sync Engine** (Epic 13)

### Sprint 9: Performance Testing & Operational Telemetry
- **Story 14.1: k6 Performance Test Suite Scaffolding & Directory Structure** (Epic 14)
- **Story 14.2: Performance Test Data Seeding & Synthetic Environment Fixtures** (Epic 14)
- **Story 14.3: Distributed Multi-Cluster Load Test Harness (200 Concurrent Users)** (Epic 14)
- **Story 14.4: Real-Time WebSocket & SSE Streaming Latency Benchmarking Suite** (Epic 14)
- **Story 14.5: CI/CD Performance Regression Quality Gates & Telemetry Alarming** (Epic 14)

### Sprint 10: External Integrations & Feature Flag Management
- **Story 15.1: Gemini API Key Management** (Epic 15)
- **Story 15.2: Application-Level Feature Flags & On-Demand Feature Document Loading API** (Epic 15)
- **Story 15.3: Feedback Collection Modal Component & Top Banner Triggers** (Epic 15)

### Sprint 11: Zero-Trust Path Resolution, Tool Parity & Native Configuration Architecture
- **Story 16.1: Unified Toolset Provisioning Across All Agent Swarms** (Epic 16)
- **Story 16.2: UVRunnerTool Binary Command Guardrails & Tool-Redirect Guidance** (Epic 16)
- **Story 16.3: Native Project TOML Workspace Configuration Seeding** (Epic 16)
- **Story 16.4: Elimination of Brittle In-Flight Config Rewrites and Markdown AST Mutations** (Epic 16)
- **Story 16.5: Consolidation of Path Safety, Sandboxing & Zero-Trust Boundary Enforcement** (Epic 16)
- **Story 16.6: End-to-End Regression Test Suite for Brainstorming Activation & TOML Resolution** (Epic 16)

### Sprint 12: Multi-Provider Email Notification Platform
- **Story 17.1: Generic Email Notification Adapter Subsystem & Provider Registry** (Epic 17)
- **Story 17.2: Custom Notification Provider (CNP) Provider Implementation & AuthN/AuthZ Impersonation** (Epic 17)
- **Story 17.3: API Notification Handler, Routing & OpenTelemetry Tracing** (Epic 17)
- **Story 17.4: Notifications Infrastructure Provisioning, IAM Roles & Cloud Deploy Pipeline** (Epic 17)

### Sprint 13: Unified File Upload & Multi-Environment Artifact Management
- **Story 18.1: Multi-Tier `ArtifactService` & Configuration Infrastructure (Backend & Mise)** (Epic 18)
- **Story 18.2: Chat File & Multimodal Image Upload UI (Frontend)** (Epic 18)
- **Story 18.3: Multimodal & Text Prompt Context Ingestion (AI Orchestrator)** (Epic 18)
- **Story 18.4: File Explorer Folder Actions & Project Tree Uploads (Full-Stack)** (Epic 18)
- **Story 18.5: Chat Session Collapsible Chevron & Artifacts Dropdown (Frontend)** (Epic 18)
- **Story 18.6: Image Artifact Preview & Media Viewer (Frontend)** (Epic 18)

### Sprint 14: Development Sandbox Builder & Runtime Verification Harness
- **Story 19.1: Implementation Tab "Build Sandbox" Action & Stack Discovery Skill Invocation** (Epic 19)
- **Story 19.2: Autonomous Specification Scanning & Runtime Stack Discovery Agent (`bmad-sandbox-builder`)** (Epic 19)
- **Story 19.3: Interactive Sandbox Stack Configuration Table UI (Add / Edit / Remove Items)** (Epic 19)
- **Story 19.4: Autonomous Dockerfile Synthesis & Sandbox Plan Generation** (Epic 19)
- **Story 19.5: Isolated Container Build Engine & Verification Test Harness** (Epic 19)
- **Story 19.6: Container Runtime Health Sweeps, Ephemeral Deployment & Live Diagnostic Log Streaming UI** (Epic 19)






