---
storyId: '13.1'
storyKey: '13-1-mobile-projects-dashboard-screen'
storyFile: 'docs/spec/implementation-artifacts/13-1-mobile-projects-dashboard-screen.md'
atddChecklistPath: 'docs/spec/testing-artifacts/atdd-checklist-13-1-mobile-projects-dashboard-screen.md'
stepsCompleted: ['step-01-preflight-and-context', 'step-02-generation-mode', 'step-03-test-strategy', 'step-04-generate-tests', 'step-05-validate-and-handoff']
lastStep: 'step-05-validate-and-handoff'
lastSaved: '2026-07-22'
generatedTestFiles:
  - 'ui/e2e/specs/13-1-mobile-projects-dashboard.spec.ts'
---

# ATDD Red-Phase Checklist — Story 13.1

**Story:** Story 13.1: Mobile Projects Dashboard Screen  
**Status:** 🔴 Red-Phase Scaffolds Generated (Ready for `dev-story` implementation)  

---

### 🔴 Acceptance Test Scaffolds Generated

| ID | Acceptance Criterion | Test Scaffold File | TDD Phase |
|---|---|---|---|
| **AC 13.1.1** | Single-column responsive grid ($<768$px) with touch target bounding box $\ge 48\times48$px | [13-1-mobile-projects-dashboard.spec.ts](file:///usr/local/google/home/sumeetsing/workspace/agentic-foundry/ui/e2e/specs/13-1-mobile-projects-dashboard.spec.ts#L13-L28) | 🔴 RED (`test.skip()`) |
| **AC 13.1.2** | Dynamic status badges (`New`, `Onboarding`, `Onboarded`) with `Onboard` / `Open Workspace` CTAs | [13-1-mobile-projects-dashboard.spec.ts](file:///usr/local/google/home/sumeetsing/workspace/agentic-foundry/ui/e2e/specs/13-1-mobile-projects-dashboard.spec.ts#L30-L41) | 🔴 RED (`test.skip()`) |
| **AC 13.1.3** | Showcase project card mobile provisioning & workspace redirection | [13-1-mobile-projects-dashboard.spec.ts](file:///usr/local/google/home/sumeetsing/workspace/agentic-foundry/ui/e2e/specs/13-1-mobile-projects-dashboard.spec.ts#L43-L57) | 🔴 RED (`test.skip()`) |
| **AC 13.1.4** | Small viewport ($375$px) header stacking with zero horizontal scroll overflow | [13-1-mobile-projects-dashboard.spec.ts](file:///usr/local/google/home/sumeetsing/workspace/agentic-foundry/ui/e2e/specs/13-1-mobile-projects-dashboard.spec.ts#L59-L67) | 🔴 RED (`test.skip()`) |

---

### 🚀 Implementation Handoff Guide for `dev-story`

When implementing Story 13.1 via `/bmad-dev-story 13-1`:
1. Unskip (`test.skip()` $\rightarrow$ `test()`) each scenario in [13-1-mobile-projects-dashboard.spec.ts](file:///usr/local/google/home/sumeetsing/workspace/agentic-foundry/ui/e2e/specs/13-1-mobile-projects-dashboard.spec.ts) sequentially.
2. Observe the test fail (Red Phase).
3. Implement the feature logic in `page.tsx`, `ProjectHeader.tsx`, and `ProjectCard.tsx` until the test passes (Green Phase).
4. Refactor clean production code (Refactor Phase).
