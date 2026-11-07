# Sprint Change Proposal: Epic 6 Compliance Creation, Story 4.2 Migration, Direct Sync Route Deletion, and AI-Tier Push Architectural Alignment

**Project:** Agentic Product Foundry (APF)
**Date:** 2026-05-30
**Author:** Antigravity (AI Coding Assistant)
**Status:** Approved & Finalized

---

## Section 1: Issue Summary

### Problem Statement & Scope Reduction Decisions
During backlog validation audits and code reviews:
1. **Modular Compliance Isolation:** Mixing compliance logging constraints (Story 4.2 Audit Trail Event Capture) with transport-level repository synchronization logic (Epic 4) diluted focus. A new dedicated Epic is needed.
2. **Direct Sync Route Removal:** During the code review of Story 4.3 (Pull Request Handoff), the **"Direct Sync Route"** (which commits and pushes directly to `main` without peer review) was evaluated. Under strict enterprise governance, allowing automated agents to bypass PR review gates and merge directly into the primary branch introduces unacceptable security risks. Therefore, a business decision was made to **completely remove the Direct Sync Route requirement** from the system, enforcing gated PR synchronization exclusively.
3. **AI-Tier Push Alignment:** In the original Story 4.3 spec, it was assumed that the API tier must perform the `git push` operation. However, a physical codebase audit and socratic alignment revealed that the local Git commit and remote push sequence already occurs automatically in the **AI tier via the file-writing interceptor plugin (Story 4.1)**. Implementing duplicate git push logic in the API tier is redundant. Instead, the API tier's role in Story 4.3 is streamlined to strictly verify remote branch existence and fail gracefully if the AI-tier sync is pending.

---

## Section 2: Impact Analysis

### Epic Impact
* **Epic 4 (Repository Synchronization):** Scope is simplified to strictly target remote branch verification and gated Pull Request handoffs.
* **Epic 6 (Compliance):** A new epic is established, dedicated to governance, logging, and security compliance.

### Story Impact
* **Story 4.2 (Audit Trail Event Capture):** Migrated to Epic 6 and renumbered as **Story 6.1**.
* **Story 4.3 (PR Handoff):** Streamlined to focus strictly on dynamic feature branch verification and GitHub PR initialization, removing API-tier git push and Direct Sync Route configurations.

### Artifact Conflicts
* **APF_Foundry_PRD.md:** Updated `FR-4.2` to enforce "Remote Synchronization Review Gates" only, removing the Direct Sync option.
* **epics.md:** Updated Epic definitions, Story 4.3 acceptance criteria, and the Sprint Plan.
* **sprint-status.yaml:** Updated tracking status mapping for Epic 4, Epic 6, and Story 6.1.

---

## Section 3: Recommended Approach

### Selected Path Forward: Direct Adjustments & Spec Alignment
We proceed with direct code and specification adjustments, aligning the API-tier role with the existing AI-tier auto-commit plugin and deleting all direct-sync paths in both front-end/back-end specs.

---

## Section 4: Detailed Change Proposals

### 1. PRD Requirement Updates (`APF_Foundry_PRD.md`)
* **Path:** [prd.md](prd.md)
* **Changes Made:** Removed `Direct Sync Route` from `FR-4.2` and mandated gated Pull Request flows exclusively.

### 2. Epic Catalog & Sprint Plan Updates (`epics.md`)
* **Path:** [epics.md](file:///usr/local/google/home/sumeetsing/workspace/agentic-foundry/docs/spec/planning-artifacts/epics.md)
* **Changes Made:**
  * Removed Direct Sync Route from Story 4.3's BDD criteria.
  * Created `Epic 6: Compliance` and moved Story 4.2 under it as `Story 6.1`.

### 3. Story Specification Refactoring (`4-3-automated-pull-request-handoff-and-remote-vcs-integration.md`)
* **Path:** [4-3-automated-pull-request-handoff-and-remote-vcs-integration.md](file:///usr/local/google/home/sumeetsing/workspace/agentic-foundry/docs/spec/implementation-artifacts/4-3-automated-pull-request-handoff-and-remote-vcs-integration.md)
* **Changes Made:** Removed direct sync references, redirected Task 2 to perform branch verification and graceful failures in the API tier, and updated all AC references.

---

## Section 5: Implementation Handoff

### Scope Classification
* **Scope:** Minor (Planning/Tracking alignment and scope reduction).

### Success Criteria
* Direct Sync logic is completely removed from specs.
* API tier strictly verifies remote branch presence and handles pending states gracefully.
