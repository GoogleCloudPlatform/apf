# APF Foundry - PRD Addendum

This document preserves the comprehensive background research, structural gap analyses, rejected alternative architectures, and forensic security defect mappings created during the evolution of the **Agentic Product Foundry (APF)** platform. 

Dividing this downstream context keeps our primary [prd.md](prd.md) requirements clean, implementation-agnostic, and audit-ready, while locking in a powerful, durable blueprint to guide future maintenance developers and system architects.

---

## 🗺️ Part A: Core Concept Shift & Strategic Pivot Trace

The strategic pivot executed on **2026-03-18** (formally registered under prior decision [DEC-001](.decision-log.md#dec-001-reimagined-as-an-enterprise-wrapper-for-open-source-frameworks-bmad)) completely re-imagined the platform wrapper from a proprietary, IDE-locked consulting asset to a collaborative, open-source enterprise wrapper built on top of the BMAD methodology.

### 1. Conceptual Shift Matrix

| Dimension | Obsolete Blueprint (v0.2) | Enterprise GUI Canvas Wrapper (v1.0) |
| :--- | :--- | :--- |
| **Core Engine** | Custom Modular Agent Chains (Bespoke Scaffold, RFC, Security, and Ops Agents maintained in isolation) | **Open-Source Agent Frameworks** (Starting with the validated **BMAD** agent scripts tree: `bmad-pm`, `bmad-ux`, etc.) |
| **Execution Context** | Provisioned workspace inside a local IDE and gated via backend IDP servers. | **Centralized GUI Console** operating natively on Web/Desktop *outside* developer workstation setups. |
| **User Interface** | API-first, terminal commands with custom text triggers for human gates and Discovery stages. | **Visual Canvas Grid** displaying progress pipeline bars, interactive chat sidebars, and live model previews. |
| **Artifact Storage** | Stored inside detached database volumes managed strictly by the IDP state machine engine. | Stored directly inside the code branch **Git Repository**, locking in continuous traceability. |
| **IDP Role** | Acting as the active operational state machine and enforcement gateway for all runtime steps. | Serving as a **Context Provider** (hydrating agents with historical codebase data, architecture configurations, and VPC keys). |

### 2. High-Level Feature Pivot Details

*   **Democratized GUI Console:** Resolves the *"IDE Lock-in"* problem where non-technical product owners and UX designers were blocked from collaborating with AI agents. Injects progress tracking pipelines, multi-persona workspace channels, and live grid rendering of diagrams.
*   **Repository-Centric Continuity:** Ensures intermediate specs (PRD markdown, user journey flows, and topography files) are automatically checked-in to Git. This prevents context rot, makes designs reviewable under standard PR flows, and automatically hydrates local anti-gravity packages on clone.
*   **Model Context Protocol (MCP) Connectors:** Replaces custom fragile API bridges with the industry-standard Model Context Protocol. Sandboxes interactive design grids (Draw.io, Figma, Lucidchart) within secure web iframes, propagating identities without database-layer plaintext storage exposure.
*   **Intelligent Context Calibration:** Calibrates agent context with pre-existing architectures, security logs, and live repository patterns under five minutes, eliminating "hallucinated boilerplate" generation failures.

---

## 🛡️ Part B: Story 2.3 Secrets Management Defect Analysis Map

This forensic analysis captures the critical security vulnerabilities and code defects identified during the adversarial review of the Story 2.3 secrets settings backend. 

*Maintaining this technical map prevents platform engineers from introducing security or architectural regressions during subsequent refactoring, security reviews, or database schema migrations of the `api` package.*

### 1. The 4 "Bad Spec" Security Flaws Resolved

#### 🚫 Bad Spec 1: Discarded Ephemeral Private Keys (Dev Fallback Cryptographic Black-Hole)
*   **Vulnerability:** In `api/agent/agent_proxy.go`, the initialization function `init()` generated a fallback temporary RSA-2048 public-private keypair in memory when environment configuration public keys were absent (common in detached dev/test environments). However, it only persisted the public key to `proxyPublicKey`, discarding the private key.
*   **Exploit Vector:** Because the private key was instantly purged from RAM, the orchestrator could never decrypt the sensitive user secrets transmitted to the agent proxy in dev runtimes, causing 100% cryptographic decryption failures and locking testing capabilities.
*   **Remediation:** Refactored the proxy manager to utilize a fixed, pre-shared public-private keypair constant configured strictly in local test scopes, or deploy symmetric token exchanges.

#### 🚫 Bad Spec 2: Superficial Dependency Injection (Global State Bypass)
*   **Vulnerability:** Handlers in `api/chat.go`, `api/session.go`, and `api/secrets.go` performed actions on database records by calling package-level global pointers: `var ChatSecretsStore` and `var SessionSecretsStore`. These variables were assigned to the central instance at initial bootstrapping in `main.go`.
*   **Exploit Vector:** Package-level globals bypass encapsulation, exposing internal data-access interfaces to other threads and components. They introduced severe concurrency race conditions and completely blocked our ability to run parallel unit tests under continuous integration.
*   **Remediation:** Completely refactored all HTTP handlers to operate as struct receivers encapsulating store dependency variables (e.g. `type SecretsHandler struct { store db.UserSecretsStore }`), eliminating global states.

#### 🚫 Bad Spec 3: Non-Standard Proprietary Ciphertext Serialization
*   **Vulnerability:** Asymmetric secret serialization in `EncryptSecretsForProxy` used a homegrown string format mapping the payload as `jwe-encrypted-enclave-token:<cek>.<nonce+ciphertext>`.
*   **Exploit Vector:** Homegrown token formatting blocks integration compatibility. Standard enterprise decrypters and JWE (JSON Web Encryption) clients could not parse these keys, preventing third-party integrations and agent execution in isolated client VPCs.
*   **Remediation:** Refactored encryption procedures to strictly construct and yield standard, compliant JWE compact serialization strings starting with `eyJ...` utilising the standard cryptographic enclave library `go-jose`.

#### 🚫 Bad Spec 4: Database Decryption Lockout under Key Rotation
*   **Vulnerability:** Ciphertext saved to Firestore database documents lacked version tags (e.g. key rotation indexes). Decryption failures on retrieval returned a direct API error that blocked human users from updating or revoking their keys.
*   **Exploit Vector:** If the master database encryption key was rotated (as mandated by enterprise security guidelines), all historic keys became un-decryptable, creating a database lockout where the user was permanently blocked from resetting their credentials via the settings UI.
*   **Remediation:** Prefixed all Firestore ciphertext with explicit key version tags (`v1:`). Refactored retrieval APIs to implement a graceful overwrite fallback: if decryption fails due to a key mismatch, the backend returns a localized masked string with a warning flag, enabling the user to reset or overwrite the corrupted keys without a complete lock.

### 2. The 12 Code Defect Patches Applied

*   **Patch 1: Organization Boundary Leakage in Session Routing:** Explicitly scrubbed a hardcoded `"user"` organization path inside proxy session initialization URLs, fully routing variables to multi-organization workspace partitions.
*   **Patch 2: Global State Mutex Bottlenecks in Proxy Key:** Removed package global storage mutations on `proxyPublicKey`, encapsulating keys within isolated, thread-safe instance scopes.
*   **Patch 3: Silenced Credentials Propagation Failures:** Enforced explicit authentication propagation error bubbling in backend setup blocks, terminating sessions with strict errors when key validations failed.
*   **Patch 4: Nil Context Map Panics:** Injected strict guards against `targetMap == nil` crashes inside token assembly pipelines.
*   **Patch 5: Uncancellable Handlers Context Leaks:** Replaced global `context.WithoutCancel` hooks with localized, cancellable context timeouts (`context.WithTimeout`) inside chat and session loops, preventing deadlocked worker processes.
*   **Patch 6: Overly Restrictive Token Validation Regex:** Expanded base64 validation validators to support modern extended JWT and OIDC keys (`^[A-Za-z0-9_\.\-\+\/=]{10,2048}$`).
*   **Patch 7: Validation Crashes on JSON Arrays:** Refactored diagnostic preflight API checks to cleanly support third-party vendors returning JSON array bodies upon success.
*   **Patch 8: Bearer Token Leakage on HTTP Redirects:** Configured custom HTTP client redirect rules to actively strip standard `Authorization` headers on cross-domain requests.
*   **Patch 9: Synchronous Preflight Denial of Service (DoS):** Refactored preflight diagnostic execution. Moved long-running (up to 5s) vendor endpoint triggers off the main HTTP request-response thread using background Go channels and context controls.
*   **Patch 10: Rigid Masking Heuristics:** Decoupled frontend token masking assertions from strict, brittle 32-bullet character length matchings.
*   **Patch 11: Secret Update Concurrency Race Conditions:** Wrapped user configuration saves and revocations inside strict database Transactions (`tx.Get`, `tx.Set`) to prevent lost updates under double-submit canvas requests.
*   **Patch 12: Plaintext Storage Testing Mockups:** Upgraded the testing suites to ensure simulated mocks replicate real encryption-decryption transforms, preventing regressions in secure storage logic.

---

## 🧭 Part C: Strategic Compliance, Git Sync Alignment, and Rejected Architectures

This section records the downstream architectural mappings, codebase audit details, and strategic rationale supporting the governance refinements executed on **2026-05-30** (formally logged under [DEC-008](.decision-log.md#dec-008-modular-compliance-isolation-under-epic-6), [DEC-009](.decision-log.md#dec-009-complete-removal-of-direct-sync-route-requirement), and [DEC-010](.decision-log.md#dec-010-ai-tier-git-push-alignment-and-api-tier-streamlining)).

### 1. Rejected Alternative: The "Direct Sync Route" (Bypassing PR Gates)

*   **Vulnerability of Direct Sync:** The initial blueprint for repository synchronization supported a "Direct Sync Route" that allowed automated agents or human workspace operators to push and auto-merge directly to the primary branch (`main`) of the target repository.
*   **Security Audit Finding:** Allowing non-gated programmatic writes to primary code branches violates basic principles of supply-chain security. Compromised user tokens or unverified agent mutations could inject malicious artifacts, overwrite production code, or bypass CI/CD tests.
*   **Resolution:** Enforce gated PR synchronization exclusively. All sync triggers must stage changes to a dedicated feature branch and auto-create a standard Pull Request, guaranteeing a mandatory **"human-in-the-loop" peer review gate**.

### 2. Modular Compliance Isolation & Strategic Shift

To maintain strict compliance boundaries, the platform separated **VCS Transport Synchronization** from **Regulatory Logging and Compliance Gating**.

| Dimension | Obsolete Transport-Bound logging | Decoupled Compliance Architecture (Epic 6) |
| :--- | :--- | :--- |
| **Audit Feed Isolation** | Log event outputs coupled directly to git synchronization threads. | **Independent Compliance Enclave** running inside organization-isolated logging contexts (`FR-6.1`). |
| **Operational Responsibility** | Git synchronization engine handles log collection, creating performance bottlenecks. | Dedicated logging middleware and hooks capture activity states asynchronously. |
| **Data Retention Profile** | Tied to Git history longevity rules (subject to squash/prune actions). | Strictly enforced **12-month persistent audit log retention** independent of VCS states. |

### 3. Codebase Audit: AI-Tier Git Push Alignment

*   **The Duplicate Git Push Problem:** In the initial implementation plan for Story 4.3 (PR Handoff), both the AI-tier file-writing interceptor and the API-tier orchestration handler were tasked with executing git push commands, causing severe concurrency lockups and branch conflicts.
*   **Audit Discovery:** A codebase audit revealed that the local git commit and remote branch push are already performed automatically inside the **AI tier via the file-writing interceptor plugin (Story 4.1)**. 
*   **Remediation & API-Tier Streamlining:**
    *   **AI Tier (Story 4.1):** Continues to handle raw file creation, staging, commit, and push as part of its write-interceptor lifecycle.
    *   **API Tier (Story 4.3):** Streamlined to strictly perform branch verification against the remote VCS, handling pending push states gracefully with user Snackbars, and initializing the Pull Request via GitHub API.

