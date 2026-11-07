---
workflowType: 'prd'
workflow: 'edit'
status: 'final'
updated: '2026-06-08'
classification:
  domain: 'Developer Tools'
  projectType: 'Enterprise Platform'
  complexity: 'High'
inputDocuments: ['design-artifacts/E-PRD/Original_APF_PRD.md', 'docs/planning-artifacts/PRD_Edit_Analysis.md', 'docs/planning-artifacts/sprint-change-proposal-2026-05-14.md', 'docs/planning-artifacts/sprint-change-proposal-2026-05-30.md']
stepsCompleted: ['step-e-01-discovery', 'step-e-02-review', 'step-e-03-edit']
lastEdited: '2026-06-08'
editHistory:
  - date: '2026-06-08'
    changes: 'Added FR-4.4 Centralized Version Control Architecture and updated FR-4.2 to specify that all Git operations (commit, push, pull) are managed strictly within the backend API tier.'
  - date: '2026-05-30'
    changes: 'Added Section 5.3 User Journeys and Section 6 FR-7 Functional Requirements defining the Automated Onboarding and Project Discovery workflow, establishing the New -> Onboarding -> Onboarded lifecycle, interactive tree pickers for legacy ingestion, and Dashboard tile action buttons.'
  - date: '2026-05-30'
    changes: 'Updated repository synchronization functional requirement FR-4.2 under Story 4.3 PR Handoff, removing Direct Sync option to enforce gated Pull Request reviews exclusively and migrating the Audit Trail Event Capture requirement under a newly established Epic 6 Compliance.'
  - date: '2026-05-21'
    changes: 'Updated FR-1.3 to refine user secrets management requirements under Story 2.3 Backend Remediation. Mandated JWE compact serialization, key version headers, user lockout recovery, and strict dependency injection, resolving implementation detail leakage in alignment with DEC-004.'
  - date: '2026-05-14'
    changes: 'Embedded explicit measurement methods (APM, WebSockets, load testing) and business context into NFRs 1.1, 1.2, and 3.1 during PRD validation simple fixes.'
  - date: '2026-05-14'
    changes: 'Removed implementation leakage from FR-1.2.2 (isHistorical flag) and FR-1.3.1 (AES-256-GCM/Firestore) during PRD validation simple fixes.'
  - date: '2026-05-14'
    changes: 'Expanded FR-1.3 User Configuration Settings and Security & Compliance with explicit application-level secret encryption, OIDC token propagation, and key validation lifecycle actions based on Story 2.3 code review intent gaps.'
  - date: '2026-05-13'
    changes: 'Expanded FR-1.2 Activity Stream with explicit broadcast hooks, historical metadata tagging, and reconnection history preservation based on Story 2.2 adversarial review.'
  - date: '2026-03-18'
    changes: 'Reimagined as an enterprise wrapper for open-source agent frameworks (BMAD).'
---

# **Product Requirements Document: Agentic Product Foundry (APF)**

**Branding:** Agentic Foundry
**Status:** Final
**Version:** 1.0
**Authors:** [Eric Maxwell](mailto:ericmaxwell@google.com), [Sumeet Singh](mailto:sumeetsing@google.com)
**Target Audience:** Enterprise Customers (deployed via GCC / IDP pipelines)

---

## **1. Executive Summary**

The **Agentic Product Foundry (APF)** is an Enterprise Enablement Platform designed to democratize and scale AI-assisted software delivery. Moving away from custom agent chains, APF acts as a **governed wrapper around open-source agent frameworks** (starting with the BMAD methodology).

APF extends agent capabilities beyond the developer's IDE into a **centralized GUI Workspace**. It empowers Product Managers, UX Designers, and Architects to interact collaboratively with specialized agents (e.g., `bmad-pm`, `bmad-ux`, `bmad-architect`) in a unified canvas environment. Crucially, all generated artifacts are stored directly in the **Git repository**, ensuring a continuous, reviewable context chain that seamlessly hydrades developer workspaces like AntiGravity.

---

## **2. Problem Statement**

While Internal Developer Platforms (IDPs) standardize environment setups, the **upstream planning and design phases** remain heavily manual, siloed, and IDE-locked:

*   **The IDE Sill:** Advanced methodologies (like BMAD) are often bound to developer terminals or IDEs, excluding product and design personas from real-time agent collaboration.
*   **Stale Context Engine:** AI assistants lack deep integration with local repository history and enterprise architectures, leading to unaligned code generation.
*   **Hand-off Fragmentation:** Artifacts (PRDs, wireframes, TDRs) scattered across wikis and drives fail to move with the code branch, leading to knowledge rot.

---

## **3. Success Criteria & Goals**

### **3.1 Primary Goals**
1.  **Democratize Agent Access:** Provide interactive canvases for PM and UX workflows running *outside* terminal setups.
2.  **Repository-Centric Continuity:** Standardize artifact buildup directly inside the `.git` tree.
3.  **Extensible Ecosystem Connectivity:** Enable plug-and-play connections into Golden Path catalogs and Google's **Application Design Center**.

### **3.2 Success Metrics (KPIs)**
*   **Lead Time Reduction:** Reduce time from Idea gate to scaffolded Repository by **30%** (< 2 hours for standard setups).
*   **Artifact Traceability:** Maintain **100%** compliance of Functional Requirements mapped to Gherkin scenarios or topiary schemas.
*   **Context Value:** Increase architectural gating pass rates by **25%** utilizing historical context calibration to prevent design anti-patterns.
*   **Counter-Metrics (Quality Safeguards):**
    *   **Post-Scaffold Build Failure Rate:** Maintain **0%** build failures on newly scaffolded repositories, ensuring speed does not introduce compiler or configuration errors.
    *   **Artifact Rejection Rate:** Enforce that less than **5%** of agent-generated PRs are rejected by human reviewers during remote synchronization gating.

---

## **4. Product Scope**

### **In-Scope**
*   Centralized Web GUI embedding Enterprise Visual Modeling tools (e.g. Draw.io) and rapid UI builders (e.g. Google Stitch).
*   Live discussions with BMAD agents natively powered by Google's Agent Development Kit (ADK) infrastructure.
*   Extensible Adapter API connecting internal developer platform states.

### **Out-of-Scope**
*   Building native Large Language Models (LLM components).
*   Replacing developer workstations (AntiGravity integration focus instead).

---

## **5. User Journeys**

### **5.1 Persona: Product Manager (PM)**
*   **UJ-1.1 Project Initiation:** PM accesses the platform console and inputs high-level project intent to scaffold a new workspace.
*   **UJ-1.2 Live Agent Collaboration:** PM enters the GUI Workspace and drafts the product requirements iteratively with the `bmad-pm` agent via real-time visual sidebar chat.
*   **UJ-1.3 Gated Remote Synchronization:** PM approves the specification, triggering the workspace orchestrator to verify branch states and initialize a secure GitHub Pull Request.

### **5.2 Persona: UX Designer**
*   **UJ-2.1 Visual Flow Canvas:** UX Designer ingests the approved PRD specification and triggers the `bmad-ux` agent to render and arrange node diagrams on the visual canvas grid.
*   **UJ-2.2 Style Guide Calibration:** UX Designer imports style rules from VCS local repository branches to calibrate the design components under Google Stitch templates.

### **5.3 Automated Onboarding & Discovery (Persona: PM / Architect)**
*   **UJ-3.1 Project Onboarding Discovery Launch:** The PM connects a remote VCS repository and creates a new project. The system initializes the project in the database with a status of `New`, and the Projects Dashboard tile displays a prominent `Onboard` action button which routes the PM directly into the workspace's dedicated Onboarding Tab.
*   **UJ-3.2 Background Scan and Progress Tracking:** The PM clicks "Start Onboarding" (or "Onboard" on the tile). The system transitions the project status to `Onboarding` (the tile continues to display `Onboard`), launching the background Discovery Agent to clone and analyze the codebase while publishing real-time progress logs to the Activity Stream.
*   **UJ-3.3 Interactive Analysis Report Audit:** Once the scan completes, the Onboarding Tab renders the Automated Repository Analysis Report. The PM and Architect review stack discoveries, framework mappings, and cataloged legacy specifications. If the scan fails, a warning card is displayed, but the PM is not blocked from completing the onboarding.
*   **UJ-3.4 Gated Ingestion Selection:** The PM uses an interactive directory tree picker to select discovered legacy markdown specifications (e.g., `docs/legacy-requirements.md`) to ingest and map to active BMad agent contexts. Pre-existing BMad spec files are flagged as `Existing BMad Spec` and pre-selected, allowing the PM to toggle between keeping them read-only or overwriting them.
*   **UJ-3.5 Explicit Onboarding Completion:** The PM explicitly clicks "Complete Onboarding". The system updates the project status to `Onboarded` in the database, auto-generates the canonical `project-context.md`, commits it to Git, and unlocks all visual canvas tools for standard active sprint operations.

---


## **6. Functional Requirements**

### **FR-1: Central Control Console & Dashboard**
*   **FR-1.1 Pipeline Tracking:** The system SHALL display a visual Idea-to-Production pipeline tracking bar representing continuous state progression from Ideation through Architecture.
*   **FR-1.2 Activity Stream:** The system SHALL provide a live collaboration activity feed broadcasting cross-agent validation requests and status updates.
    *   **FR-1.2.1 Lifecycle Broadcast Hooks:** All platform workflow execution endpoints and lifecycle state transitions MUST explicitly broadcast internal platform activity events to the activity stream store.
    *   **FR-1.2.2 Historical Event Tagging & Stream Filtering:** The backend MUST tag historical activity stream messages with an explicit metadata flag or deliver them via a dedicated REST fetch endpoint. Client stream filtering MUST rely on this server-provided metadata rather than client-side time comparison heuristics.
    *   **FR-1.2.3 Reconnection History Preservation:** Automatic client reconnections following transient network disruptions MUST preserve existing activity history in the notification drawer without purging previously fetched items.
*   **FR-1.3 User Configuration Settings:** The system SHALL provide a dedicated user-level settings screen allowing individuals to securely input and manage their personal API keys and authorization tokens required for third-party MCP integrations (e.g., Figma, Lucidchart, Stitch).
    *   **FR-1.3.1 Application-Level Secret Encryption & Key Rotation:** The backend MUST perform strong application-level encryption on all user-configured API keys before persisting them to database records, ensuring defense-in-depth beyond default storage-level encryption. Stored ciphertext MUST include explicit key versioning headers (e.g., `v1:`) to support seamless key rotation. If decryption fails due to key rotation or data corruption, the backend MUST NOT permanently lock the user out; it MUST provide a graceful error recovery mechanism allowing the user to overwrite or reset un-decryptable secrets.
    *   **FR-1.3.2 OIDC/OAuth Token Propagation & Compliant JWE Transmission:** The orchestrator MUST explicitly extract the authenticated human user's OIDC/OAuth identity token from the request context (`claims.Token` or identity headers) and propagate it alongside their configured third-party API keys into the MCP runtime execution environment. Crucially, the transmission of user secrets to detached agent proxy runtimes MUST utilize End-to-End Asymmetric Encryption adhering strictly to the standard JWE compact serialization format (`eyJ...`) or secure symmetric token exchange (STS). Dev/test fallback environments lacking an external public key MUST utilize a pre-shared dev key pair or symmetric token exchange rather than discarding ephemeral private keys in memory.
    *   **FR-1.3.3 Key Validation, Lifecycle Actions & Clean Dependency Injection:** The user settings interface and backend MUST provide explicit, secure lifecycle actions to update, validate, and revoke individual third-party integration keys. All backend HTTP handlers managing secrets, chat, and sessions MUST be refactored to utilize clean, decoupled dependency injection, completely eliminating reliance on package-level global variables. Preflight diagnostic validation MUST verify keys against official vendor endpoints, supporting both JSON object and JSON array success responses, while enforcing strict HTTP redirect policies to prevent Bearer token leakage.
*   **FR-1.4 Contextual Artifact Sidebar:** The system SHALL provide a collapsible right-hand pop-out menu for artifact navigation. When expanded via a persona icon click, it SHALL display a list of artifacts collected by the specific active persona, and allow users to view collections of artifacts from other personas (e.g., UX, Engineering) for reference.
*   **FR-1.5 Project-Level App Integration Dynamic Settings:** The system SHALL provide project-level application integration settings that dynamically render a custom configuration form (e.g., specification directory path, base git branch, spec branch prefix for GitHub) and a searchable repository selector, persisting these parameters under an array of app_integrations inside the project database collection.


### **FR-2: Multi-Persona Interactive Canvas**
*   **FR-2.1 Canvas chat:** The sidebar interaction SHALL support instructing agents alongside the visual modeling tools with response latency < 2 seconds.
*   **FR-2.2 Embedded Modeling Tools:** The canvas grid SHALL dynamically mount and render the native editors for Draw.io or Lucidchart via platform embed APIs, mapping agent modifications generated via MCP sync to visual state updates within < 500ms.

### **FR-3: Extensible Adapter Service**
*   **FR-3.1 Golden paths triggers:** The system SHALL enable mapping golden path service catalogs to template configurations brokers via standard adapter hooks.
*   **FR-3.2 Context ingest calibration:** The plug-in system SHALL absorb previous codebase architectures and patterns and update agent contexts within < 5 minutes of ingest pull triggers.

### **FR-4: Repository Continuity Sync**
*   **FR-4.1 State Sync tracker:** The system SHALL automatically check-in generated specifications (e.g., `design-artifacts/`) to the local git branch upon state save anchors.
*   **FR-4.2 Remote Synchronization Review Gates:** The system SHALL support dynamic, secure synchronization of generated workspace artifacts with the remote Version Control System (VCS). When a user triggers sync, the API tier SHALL commit, push the branch to the remote repository, and auto-create a standard Pull Request (PR) against the primary branch, enforcing secure human-in-the-loop review gates.
*   **FR-4.3 Git-to-Workspace Spec Re-Hydration & Edit Sync:** The system SHALL support dynamic, automated reverse-synchronization when a user triggers an "Edit" action on a previously approved/finalized specification. The backend orchestrator (API tier) MUST pull the latest commit from the remote VCS repository, staging and decoding the spec contents back into the active, ephemeral workspace runtime environment, while transitioning the specification's status indicator back to "In-Progress".
*   **FR-4.4 Centralized Version Control Architecture:** The system SHALL centralize all Git operations (clone, pull, commit, push, merge), OIDC authentication routing, and repository synchronization concerns strictly in the backend API orchestrator tier. The downstream AI agent inference tier (ADK or agy-sdk) SHALL be completely decoupled from version control libraries and repository management tasks.

### **FR-5: Parallel Dual-Engine AI Orchestration**
*   **FR-5.1 Parameterized Engine Routing:** The system SHALL support parameterized, concurrent execution pathways for all AI orchestration and inference operations. The backend orchestrator MUST dynamically route chat interactions, event streams, and background agent execution loops through either the baseline Agentic Development Kit (ADK) core or the high-performance Antigravity SDK (agy-sdk) core based strictly on the active project-level integration configuration, with absolute functional transparent handoffs for the human stakeholder.
*   **FR-5.2 Dynamic Tool Adapters Parity:** The visual canvas modeling tools and external Model Context Protocol (MCP) servers sync triggers MUST integrate cleanly under both execution paths. Tool inputs, outputs, and JSON transformations executed by either the ADK engine or the agy-sdk engine MUST achieve 100% functional parity.
*   **FR-5.3 Schema & Telemetry Isolation:** Operational telemetry (OpenTelemetry logs, audit trails, and status dial markers) generated by either active core engine MUST serialize into database tables using identical schema profiles, mathematically preventing data bleed, corruption, or trace loss during framework-switching events.

### **FR-6: Compliance & Governance**
*   **FR-6.1 Audit Trail Event Capture:** The system SHALL capture and retain comprehensive, immutable audit logs for all agent-human interactions and workspace mutation events with a p99 capture reliability of 99.99%, preserving files and databases in structured event enclaves for review and retention specifications of exactly 12 months.

### **FR-7: Automated Onboarding & Project Discovery**
*   **FR-7.1 Project Lifecycle State Machine:** The system SHALL support a managed project-level lifecycle. Newly created projects MUST initialize in the database with a status of `New`.
*   **FR-7.2 GUI Dashboard Gating & Tile Actions:** The dashboard workspace SHALL dynamically check project status. Projects in the `New` or `Onboarding` state MUST render an action button labeled `Onboard` on their Projects Dashboard tile.
    *   **FR-7.2.1 Session Resumption:** Clicking the `Onboard` action button on an `Onboarding` project MUST seamlessly restore the active workspace session, loading either the active background scan progress bar or the generated analysis report.
*   **FR-7.3 Autonomous Background Discovery Agent:** When the PM triggers onboarding, the system SHALL transition the project status to `Onboarding` and launch an autonomous background agent to analyze the target repository.
    *   **FR-7.3.1 Technical Stack Analysis:** The discovery agent SHALL automatically parse configuration files (e.g., `go.mod`, `package.json`, `pom.xml`, `Dockerfile`, `.tf`) to classify core runtimes and project frameworks.
    *   **FR-7.3.2 Legacy Specification Audit:** The discovery agent SHALL scan filesystem trees to catalog existing product briefs, design documents, and other legacy requirements (`.md`/`.html` files).
    *   **FR-7.3.3 Scan Interruption Recovery:** If the background scan fails or is interrupted, the system SHALL display a warning card but MUST allow the user to proceed and mark onboarding as complete, onboarding the project as a minimal/new setup.
*   **FR-7.4 Ingestion Tree Gating & Pre-existing Specifications:** The Onboarding Tab SHALL present an interactive tree picker displaying discovered legacy files for PM ingestion selection.
    *   **FR-7.4.1 Pre-existing Spec Toggles:** Discovered existing BMad specifications (`prd.md`, `architecture.md`) MUST be flagged as `Existing BMad Spec` and pre-selected. The UI SHALL provide an options toggle: `Keep Existing Spec (Read-Only)` or `Re-Reconcile & Overwrite` to govern file-writing outcomes.
*   **FR-7.5 Explicit Onboarding Completion Gating:** The Onboarding Tab SHALL enforce an explicit user completion gate. The project status SHALL NOT transition to `Onboarded` until the user explicitly clicks the `Complete Onboarding` button. Upon completion, the system SHALL generate a canonical `project-context.md` outlining stack discoveries, commit it to the repository, and unlock all active canvas tools.

---


## **7. Interface Requirements**

To maintain governance standards, strict adapter brokers govern platform extensions:

*   **Visual Canvas Adapters (MCP):** The Interactive Canvas pane SHALL embed Enterprise visual modeling tools (specifically **Draw.io** and **Lucidchart**) and rapid UI application builders (specifically **Google Stitch**) via native web iframes.
*   **Agent to Visual Sync:** The APF Backend SHALL implement **Model Context Protocol (MCP)** adapters to connect to the Draw.io, Lucidchart, and Stitch MCP servers. The system SHALL support bidirectional sync: parsing diagram XML/JSON into agent context, outputting agent-generated modifications back to the embedded `.drawio` file dynamically, and generating functional UI component trees via Stitch.
*   **Workspace Adapters:** Integration adapters SHALL support Figma sync utilizing Figma Web API v1+ with sync completion < 10 seconds for standard files.
*   **Context Connectors:** Plug endpoints SHALL fetch IDP schema metadata structures utilizing HTTPS/TLS 1.3 connections.
*   **Hydration Outpost:** Endpoint configuration configurations SHALL feed AntiGravity workspace packages compliant with `JSON-RPC` standard endpoints.

---

## **8. Security & Compliance**

*   **Organization VPC Scope:** All operations SHALL run strictly within organization VPC bounds with complete network isolation from public endpoints.
*   **Audit Trail Log:** The system SHALL capture and retain audit logs for all agent-human interactions with 99.99% capture reliability and review retention specs of 12 months.
*   **Application-Level Secret Encryption:** All sensitive user configuration secrets (e.g., third-party API keys) SHALL undergo application-level encryption (AES-256-GCM) before database persistence to guarantee defense-in-depth.

---

## **9. Non-Functional Requirements**

### **9.1 Performance & Latency**
*   **NFR-1.1 GUI Load Time:** To prevent user abandonment during workflow transitions, the dashboard and canvas components SHALL achieve interactive load time in < 2 seconds for the 95th percentile as measured by APM monitoring.
*   **NFR-1.2 Sync Latency:** To enable seamless real-time multi-persona collaboration, canvas node updates and layout triggers SHALL synchronize across active viewers with synchronous latency < 100ms for the 95th percentile as measured by WebSocket frame timing metrics.

### **9.2 Availability & Reliability**
*   **NFR-2.1 Uptime:** The APF Console Platform wrapper SHALL maintain 99.9% uptime availability (measured monthly).

### **9.3 Scalability**
*   **NFR-3.1 Concurrent Users:** To support enterprise-scale deployment clusters, the canvas module SHALL support at least 50 concurrent users per active workspace cluster session without degradation as measured by distributed load testing suites.

---

## **10. Glossary of Terms**

To prevent downstream design and engineering ambiguity, the following canonical names represent primary system components and workspaces:

*   **GUI Workspace (or Console Workspace):** The centralized web browser dashboard and editing frame that embeds the interactive canvas grid, sidebar chat, and collapsible artifact rails.
*   **Interactive Canvas:** The primary grid modeling area that dynamically renders visual node workflows, decision points, and status triggers, powered by Draw.io, Figma, or Stitch integrations.
*   **Activity Stream:** The live collaboration feed broadcasting real-time cross-persona agent validation requests, system updates, and connection status reports.
*   **Context Connector:** The secure, TLS 1.3-compliant data adapter responsible for pulling IDP metadata schemas and codebase patterns into the active agent runtime context.
*   **Asymmetric Secret Enclave:** The secure application-layer encryption module that serializes user integration keys into standard JWE compact strings before persistence.
*   **Discovery Agent:** An autonomous background BMad agent responsible for scanning, analyzing, and classifying a brownfield repository's languages, dependencies, structures, and existing documentation files.
*   **Onboarding Tab:** The dedicated interface section within the GUI Workspace where automated discovery scan triggers, real-time analysis reports, directory scoping picks, and explicit completion gates are managed.
*   **Project Lifecycle States:**
    *   **New:** Initial state when a project is created, displaying the `Onboard` action button on the Projects Dashboard tile.
    *   **Onboarding:** In-progress state while the discovery scan runs or waits for PM validation/selection. The Projects Dashboard tile continues to display `Onboard` and resumes the active tab session.
    *   **Onboarded:** Final active state triggered when the PM explicitly clicks "Complete Onboarding", fully unlocking active canvas tools.


---

## **11. Assumptions Index**

The product requirements are constructed under the following strategic and technical assumptions:

1.  **`[ASSUMPTION-001]` OIDC Token Extraction:** It is assumed that enterprise Identity-Aware Proxy (IAP) systems successfully propagate authenticated OIDC/OAuth user identity headers under standard request context fields.
2.  **`[ASSUMPTION-002]` Downstream Integration Support:** It is assumed that third-party model context protocol (MCP) servers (Lucidchart, Stitch, Figma) are physically capable of absorbing JWE compact enclaves natively without requiring proprietary decryption layers.
3.  **`[ASSUMPTION-003]` AntiGravity Interface Parity:** It is assumed that AntiGravity developer workspaces comply natively with standard JSON-RPC endpoints to enable direct workspace hydration.
