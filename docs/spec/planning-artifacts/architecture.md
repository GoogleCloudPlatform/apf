---
stepsCompleted: [1, 2, 3, 4, 5, 6, 7, 8]
inputDocuments: [
  "docs/spec/planning-artifacts/prd.md",
  "docs/specs//planning-artifacts/ux-design-specification.md",
  "docs/specs//planning-artifacts/research/market-agent-foundry-research.md"
]
workflowType: 'architecture'
lastStep: 8
status: 'complete'
completedAt: '2026-06-13'
project_name: 'Agentic Product Foundry'
user_name: 'User'
date: '2026-03-23'
---

# Architecture Decision Document

_This document builds collaboratively through step-by-step discovery. Sections are appended as we work through each architectural decision together._

## Project Context Analysis

### Requirements Overview

**Functional Requirements:**
*   **Agentic Orchestration Engine:** Multi-agent pipeline requiring persistent state management.
*   **Synchronous Dual-Pane Handoff:** Low-latency bidirectional communication for real-time artifact compilation.
*   **Three-in-a-Box Governance:** Robust data modeling to track document verification states before unlocking developer handoff.
*   **Hybrid Storage Implementation:** Routing ephemeral iterative work to GCS and gating final artifact promotion via Source Control MCP endpoints.

**Non-Functional Requirements:**
*   **Enterprise-Grade Extensibility (Pluggable Architecture):** The system must be designed from Day 1 using a highly modular pattern (e.g., Ports and Adapters). While the MVP standardizes on GitHub and the BMAD framework, the core orchestrator must proceed decoupled.
*   **Comprehensive Observability & Audit Trails:** Full, immutable audit trails of every inference, user interaction, and state mutation, standardizing on OpenTelemetry (OTel).
*   **Google First-Party Native:** Architecture must effortlessly integrate with Google Cloud standard practices (Application Default Credentials (ADC), Cloud Logging, Cloud Trace).
*   **Enterprise-Grade GCP Security:** Operation strictly within live VPC perimeters, utilizing IAM Service Accounts, restricted Role-Based Access Control (RBAC).
*   **LLM Provider Agnosticism:** Core execution engine utilizing a Model Router abstraction to decouple agent prompt logic from underlying LLM provider, guaranteeing compliance with data residency laws.

**Scale & Complexity:**
*   **Primary domain:** Web-based Desktop Enterprise Application (Full-Stack + AI GenKit Platform)
*   **Complexity level:** Enterprise High-Complexity

### Technical Constraints & Dependencies
*   **Modern React / Next.js UI:** Must render agent tool outputs and interactive canvas components via a modern **Next.js (App Router)** frontend over WebSockets/SSE.
*   **Concurrency & State Mutex:** Single-player write-lock mutexes on workspaces to prevent simultaneous LLM generation collisions.
*   **Agent Execution Topology:** Asynchronous Task Pipelines (Cloud Tasks / PubSub) with detached serverless execution to strictly prevent 60-second HTTP timeout limits on massive GenAI inferences.
*   **Quota & Token Governance Proxy:** Smart Token Governance layer decoupled from the raw LLM using a lightweight, fast Redis-backed token bucket algorithm and budgeting queue per Cost-Center. This provides absolute prevention of `429 Too Many Requests` API limit exhaustion without the heavyweight infrastructure overhead of an enterprise API Gateway.
*   **Active DLP Interception Pipeline (Model Armor):** Google Cloud Model Armor deployed in conjunction with Sensitive Data Protection (DLP). Model Armor acts as the AI runtime security layer (integrated natively into the Go orchestrator), leveraging the underlying DLP engine to actively detect and scrub out PII, JWTs, and secure keys from human prompts and copied logs prior to RAG ingestion.
*   **Organization-Isolated Context Boundaries:** Hard multitenancy at the Vector DB/RAG layer partitioned exclusively by `Workspace-ID` to mathematically prevent context bleed across corporate teams.

### Cross-Cutting Concerns Identified
*   **Deterministic Golden Path Scaffolding:** GenKit agents are expressly forbidden from hallucinating infrastructure setups. The orchestrator must deterministically clone/merge from Platform Engineering-owned, CI-approved repository templates and cleanly inject AI context.
*   **Semantic Dependency Graphing:** Dependencies between generated artifacts must be strictly typed as a metadata graph. GCS changes to upstream components inherently flag downstream artifacts as "stale."

## Starter Template Evaluation

### Primary Technology Domain

**Enterprise AI Full-Stack** (Web-Based Desktop GUI + Serverless LLM Orchestration + Declarative Cloud Infrastructure)

### Selected Starters: APF Multi-Tier Scaffold (ADK-Go, Next.js-React, Terraform-Foundation)

**Rationale for Selection:**
This enterprise combination ensures high performance backend execution (Go), robust declarative cloud infrastructure (Terraform Modules), and a highly modern, responsive web application presentation layer (Next.js App Router).

**Architectural Decisions Provided by Starters:**

**Language & Runtime:**
*   Backend: Go 1.22+ leveraging Google's Agent Development Kit (ADK) deployed statelessly to Cloud Run. ADK provides out-of-the-box infrastructure for routing, specialized prompt templates, memory, and native observability.
*   Frontend: TypeScript / React via Next.js (App Router).
*   Infrastructure: HashiCorp Configuration Language (HCL).

**Styling Solution:**
*   Frontend strictly utilizes Tailwind CSS and Shadcn UI / Radix primitives for modern semantic enterprise styling.

**Build Tooling:**
*   Backend: Standard `go build` compiled to static binaries (perfect for ultra-small Docker images on Cloud Run).
*   Frontend: Standard Next.js production build (`next build`).

**Testing Framework:**
*   Backend: Native Go `testing` toolchain with GenKit evaluation datasets.
*   Frontend: Vitest and React Testing Library (RTL).

**Code Organization:**
*   Strict Poly-Directory structure: `/orchestrator` (Go), `/client` (Next.js), and `/infrastructure` (HCL).
*   Terraform state physically isolated in a GCS Remote Backend bucket (configured in `backend.tf`).

**Development Experience:**
*   Local testing enabled via the GenKit Developer UI (`genkit start`).
*   Next.js Fast Refresh for frontend component iteration (`next dev`).
*   Terraform `plan` explicitly enforced before any `apply`.

**Note:** Project initialization using these commands should be the first implementation story.

## Core Architectural Decisions

### Decision Priority Analysis

**Critical Decisions (Block Implementation):**
*   **Operational Database:** Google Cloud Firestore (NoSQL).
*   **Authentication Provider:** Google Cloud Identity / Identity-Aware Proxy (IAP).
*   **APF Deployment CI/CD:** Google Cloud Build.

**Important Decisions (Shape Architecture):**
*   **Client-Server API:** Standard REST (JSON) over HTTP for non-generative operations.
*   **Next.js State Management:** Zustand and React Query (TanStack Query).

**Deferred Decisions (Post-MVP):**
*   **CRDT Multiplayer Canvas Editing:** Deferred in favor of strict Mutex Single-Player locks for MVP.

### Data & Resource Architecture
*   **Primary DB:** Google Cloud Firestore (NoSQL).
*   **Project Context Boundary:** The core architectural abstraction is the `Project-ID`. Every document, generated artifact, and iterative chat history strictly exists within a specific `Project-ID` boundary.
*   **Collaboration & RBAC:** Firestore structures must contain authorization sub-trees (e.g., `owners`, `editors`) at the Project level. Any CRUD operation or agent invocation first validates whether the human user's OIDC context intersects with the active project's authorization list. 
*   **Agent Contextual Awareness:** When executing GenKit logic, the orchestrator injects the active `Project-ID` into the agent's memory. The agent's tools are explicitly constrained to only interact with GCS scratchpad URIs belonging to that project (e.g., `gs://apf-scratchpad/projects/{project-id}/...`), preventing cross-project hallucination.
*   **Rationale:** Firestore provides instant serverless compatibility with our Cloud Run executors ($0 idle cost). Establishing `Project-ID` as the absolute root entity enables immediate multi-user collaboration and safe, bounded AI interaction against the GCS data store.
*   **Parameterized Git Repository Synchronization & Gating:** To satisfy dynamic enterprise governance policies, remote repository sync pipelines MUST support two distinct technical execution routes, dynamically parameterizable via a dedicated project integration setting inside Firestore (`integrationSettings.gitSyncRoute`):
    *   **Direct Sync Pipeline (High-Velocity):** The backend Go orchestrator utilizes the standard Git adapter (`go-git/v6`) to stage, commit, and push the newly generated specifications (under `design-artifacts/` or `docs/spec/`) directly to the remote repository's primary tracking branch (e.g., `main`), propagating the human user's authenticated OAuth context.
    *   **Pull Request Integration (High-Governance):** The orchestrator stages and pushes the workspace specifications to a dynamically created remote feature branch (naming schema: `artifact-sync-{projectId}-{timestamp}`). Upon a successful remote branch sync, the backend orchestrator MUST immediately invoke standard remote VCS APIs (specifically **GitHub REST API v3** wrappers) to initialize a standard Pull Request against the target primary branch. To secure authorization, the push and PR initialization MUST propagate the user's OAuth/OIDC token, mathematically proving that the platform cannot commit or open branches beyond the user's specific repository write boundaries.
*   **VCS Git Pull & Staging Workspace Re-Hydration:** To support the fluid editing of committed and approved specifications, the platform MUST implement an automated Git-to-Workspace pull-sync pipeline. When a finalized/approved artifact is opened for active revision, the Go backend orchestrator MUST utilize the standard Git adapter (`go-git/v6`) to pull the latest file commit from the remote repository's branch, staging and decoding the raw specification assets directly into the user's ephemeral workspace environment (writing to GCS paths matching the schema `gs://apf-scratchpad/projects/{projectId}/...` and refreshing the active Firestore tracking status of the artifact back to `In Progress`). To maintain strict Zero-Trust perimeters, the pull operations and branch retrievals MUST propagate the human user's validated OIDC identity claims token, verifying read/write permissions at the remote VCS interface.

### Authentication & Security
*   **Provider:** Google Cloud Identity via IAP (Identity-Aware Proxy) with **Strict Identity Propagation**.
*   **Rationale:** Enforces strict "Zero Trust" boundaries at the GCP Load Balancer level. Cryptographically authenticates users before executing ADK orchestrator logic.
*   **Agent Identity Constraint:** Agents **do not** have standalone service accounts or identities. The orchestrator strictly propagates the human user's OIDC/OAuth token (Identity Propagation) down to all tools and MCP endpoints. This mathematically guarantees the agent cannot perform any action the human user is not authorized to do.
*   **User-Level Secrets Management, Compliant JWE Transmission & Key Rotation:** Individuals must configure personal API keys (e.g., Figma, Draw.io) for external MCP connections via a dedicated Settings Screen. These tokens are stored securely per-user bound to their authenticated IAP identity profile in Firestore. To guarantee defense-in-depth beyond storage-level encryption, all secrets persisted in database records MUST undergo application-level encryption specifically utilizing the **AES-256-GCM** algorithm. Stored ciphertext MUST include a key version header (e.g., `v1:`) to support seamless key rotation and a graceful fallback mechanism to prevent permanent user lockout upon decryption failure. When the orchestrator communicates with detached agent proxy runtimes or remote MCP servers, sensitive user secrets MUST NOT be transmitted in plaintext. The orchestrator MUST employ End-to-End Asymmetric Encryption adhering strictly to the standard JWE compact serialization specification (`eyJ...`) using standard JWE libraries, or short-lived secure token exchange (STS). For dev/test fallback scenarios lacking an external public key, the system MUST utilize a pre-shared dev key pair or symmetric token exchange rather than ephemeral in-memory generation that discards private keys. Furthermore, to enable clean encapsulation and independent parallel unit testing, the backend orchestrator mandates strict struct-based dependency injection for all HTTP handlers and data access layers (e.g., refactoring chat, secrets, and session handlers into struct receivers with explicit `db.UserSecretsStore` dependencies), completely prohibiting package-level global state.
*   **Auditability:** Standardizes on OpenTelemetry (OTel) and Cloud Audit Logs. Every action performed by an agent is tagged with a composite identity trace (e.g., `Principal: User-[ID]`, `Actor: Agent-[Name]`) to clearly distinguish autonomous background actions from direct human UI interactions while maintaining a single security principal.
*   **Zero-Trust Context Propagation & Downstream Enforcement:** Downstream API handlers and repository layers MUST retrieve the active user's identity and organization scope exclusively from the Go request `context` (injected by the authentication middleware). Direct header lookups for `X-Organization-Id` or `x-goog-authenticated-user-*` are strictly prohibited in downstream code. The platform adopts a fail-secure design where any profile that is not explicitly `"local"` or `"test"` is treated as a production environment, completely disabling IAP bypass mechanisms at both application startup (via fail-safe configuration audits) and during request validation.


### AI Orchestration Core Architecture
*   **Engine:** Go GenKit Agent Development Kit (ADK) Pipeline.
*   **Execution Runtime:** The system SHALL execute chat prompt inferences, RAG vector retrievals, memory persistence, and local tool execution enclaves natively in Go using the standard GenKit SDK framework statelessly on Cloud Run.
*   **Rationale:** Standardizing entirely on Go 1.22+ and GenKit ADK ensures single-binary execution speeds, minimal container footprint on Cloud Run, robust WebSocket support for Server-Driven UI, and native integration with Google Cloud observability.

### API & Communication Patterns
*   **Pattern:** Standard REST (JSON) for standard UI operations; Server-Sent Events (SSE) strictly reserved for asynchronous GenKit UI streaming.
*   **Rationale:** Decouples standard dashboard data-fetching from the heavy LLM orchestration layer while avoiding the massive configuration overhead of gRPC-Web through IAP proxies.

### Frontend Architecture
*   **Framework:** Next.js (App Router) utilizing React Server Components (RSC) and Client Components.
*   **State Management:** Zustand (for global UI sync) and React Query (for asynchronous server state).
*   **Visual Canvas & UI Builder Integration (MCP):** The platform strictly avoids building custom graph rendering engines or UI canvas tools. Instead, the right-hand pane acts as a unified iframe wrapper for enterprise-standard visual tools (specifically **Draw.io/diagrams.net** and **Lucidchart**) and rapid full-stack application builders (specifically **Google Stitch**).
*   **Visual Context Sync:** The APF Backend utilizes the **Model Context Protocol (MCP)** to communicate with standard modeling and UI building MCP servers. This enables ADK agents to natively parse diagram XML/JSON for deep architectural context, programmatically alter visual topologies, and output functional frontend mockups directly into the Stitch workspace context.
*   **Rationale:** Next.js provides modern web rendering speeds, robust Server-Sent Events (SSE) support for Generative UI streaming, and an industry-standard React component ecosystem.

### Infrastructure & Deployment
*   **Pipeline Engine:** Google Cloud Build.
*   **Rationale:** Keeps the deployment of the Agentic Product Foundry completely self-contained within native GCP infrastructure. Prevents enforcing third-party CI/CD vendor lock-in (like GitHub Actions) on the Enterprise installation itself, while allowing the generated scaffolding downstream to support any customer Git provider.

### Decision Impact Analysis

**Implementation Sequence:**
1. Provision GCP Foundation via Terraform (VPC, Firestore, Cloud Run, Cloud Build).
2. Establish IAP Authentication proxy wrapper.
3. Scaffold Go GenKit backend orchestrator with REST endpoints and SSE streams.
4. Scaffold Next.js React client with Zustand and bind to SSE streaming.
5. Implement Pluggable MCP Git integration layers.

**Cross-Component Dependencies:**
*   The Go orchestrator is heavily dependent on the chosen Auth Provider (IAP) passing validated JWTs in headers to secure its REST endpoints and execute authenticated Git adapter commits.

## Project Structure & Boundaries

### Complete Project Directory Structure

```text
agentic-foundry/
├── README.md
├── CONTRIBUTING.md
├── Agent.md
├── firebase.json
├── go.work
├── mise.toml
├── .gitignore
├── .agents/
│   ├── rules/
│   ├── skills/
│   └── workflows/
├── ai/
│   ├── go.mod
│   ├── main.go
│   ├── plugins/
│   ├── sub_agents/
│   └── tools/
├── api/
│   ├── go.mod
│   ├── main.go
│   ├── routes.go
│   ├── chat.go
│   ├── workspace.go
│   ├── git_client.go
│   ├── agent/
│   ├── core/
│   └── db/
├── ui/
│   ├── package.json
│   ├── tailwind.config.ts
│   ├── next.config.mjs
│   ├── src/
│   └── public/
├── infra/
│   ├── local/
│   ├── modules/
│   ├── staging/
│   └── prod/
├── third_party/
│   └── bmad/
├── runners/
│   └── tmux/
├── scripts/
└── docs/
    └── spec/
        └── planning-artifacts/
            ├── prd.md
            ├── architecture.md
            ├── epics.md
            └── ux-design-specification.md
```

### Architectural Boundaries
*   **API (`api/`)**: REST & SSE streaming endpoints, Git VCS synchronization, and workspace gating.
*   **AI (`ai/`)**: Detached GenKit agent inference, RAG embeddings, and tool execution enclaves.
*   **UI (`ui/`)**: Next.js (App Router) presentation layer using Zustand for real-time canvas synchronization.
*   **Data & Isolation**: Hard multi-tenancy isolation by `Workspace-ID` / `Organization-ID` inside organization VPC enclaves.

## Architecture Validation Results

### Coherence Validation ✅

**Decision Compatibility:**
All decisions across the Go GenKit ADK backend (`api/`), Next.js App Router frontend (`ui/`), GenKit Agent execution enclaves (`ai/`), and Terraform deployment infrastructure (`infra/`) operate compatibly over standard REST/SSE boundaries.

**Pattern Consistency:**
Implementation patterns enforcing `snake_case` in Firestore/database models and strict `camelCase` JSON payload serialization align seamlessly.

**Structure Alignment:**
The physical directory structure fully matches the active repository topography.

### Requirements Coverage Validation ✅

**Epic/Feature Coverage:**
100% coverage mapped across Epic 1 (Interactive Canvas), Epic 2 (Collaboration Tracking), Epic 4 (Repository Sync), and Epic 8 (Multi-Tenancy Isolation).

**Functional Requirements Coverage:**
All FRs and NFRs are architecturally supported.

### Implementation Readiness Validation ✅

**Decision Completeness:**
All critical decisions documented with versions.

**Structure Completeness:**
Complete directory structure established.

**Pattern Completeness:**
Naming, communication, and error recovery patterns defined.

### Gap Analysis Results
No critical or blocking gaps remaining.

### Validation Issues Addressed
Consolidated dual-engine AI orchestration under Go GenKit ADK and replaced Flutter Web entirely with Modern Next.js (App Router).

### Architecture Completeness Checklist

**Requirements Analysis**
- [x] Project context thoroughly analyzed
- [x] Scale and complexity assessed
- [x] Technical constraints identified
- [x] Cross-cutting concerns mapped

**Architectural Decisions**
- [x] Critical decisions documented with versions
- [x] Technology stack fully specified
- [x] Integration patterns defined
- [x] Performance considerations addressed

**Implementation Patterns**
- [x] Naming conventions established
- [x] Structure patterns defined
- [x] Communication patterns specified
- [x] Process patterns documented

**Project Structure**
- [x] Complete directory structure defined
- [x] Component boundaries established
- [x] Integration points mapped
- [x] Requirements to structure mapping complete

### Architecture Readiness Assessment

**Overall Status:** READY FOR IMPLEMENTATION

**Confidence Level:** High

**Key Strengths:**
*   Single orchestration API gateway (Go GenKit ADK).
*   Modern web rendering speeds (Next.js App Router).
*   Hard multi-tenancy isolation at vector embedding and storage layers.

**Areas for Future Enhancement:**
*   CRDT real-time multiplayer editing engines post-MVP.

### Implementation Handoff

**AI Agent Guidelines:**
- Follow all architectural decisions exactly as documented.
- Use implementation patterns consistently across all components.
- Respect project structure and boundaries.
