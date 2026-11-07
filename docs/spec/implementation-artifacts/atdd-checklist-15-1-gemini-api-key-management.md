---
storyId: '15.1'
storyKey: '15-1-gemini-api-key-management'
storyFile: 'docs/spec/implementation-artifacts/15-1-gemini-api-key-management.md'
atddChecklistPath: 'docs/spec/implementation-artifacts/atdd-checklist-15-1-gemini-api-key-management.md'
generatedTestFiles:
  - 'api/agent/agent_proxy_test.go'
  - 'api/user_handlers_test.go'
  - 'ui/e2e/specs/settings/gemini-key.spec.ts'
inputDocuments:
  - 'docs/spec/implementation-artifacts/15-1-gemini-api-key-management.md'
  - 'docs/spec/implementation-artifacts/15-1-gemini-api-key-management-test-plan.md'
---

# ATDD Checklist & TDD Red-Phase Scaffolds: Story 15.1 - Gemini API Key Management

**Date:** 2026-07-26
**Author:** Sumeet / Master Test Architect
**Status:** Red Phase Complete (Scaffolds Generated)

---

## TDD Cycle Status: 🟢 GREEN PHASE (ALL TESTS PASSING)

All acceptance criteria and test scaffolds for Story 15.1 have been implemented, un-skipped, and verified passing across unit tests, API integration tests, and Playwright E2E specs.

---

## Acceptance Criteria & ATDD Test Scaffold Mapping

| AC ID | Acceptance Criteria Summary | Test Level | Scaffold Location | Test Function / Scenario | TDD Phase |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **AC-1** | Integration Catalog & Dev/Test Seed Credential (`gemini_simulated_api_key_12345`) | DB / Integration | `api/agent/agent_proxy_test.go` | `TestAgentProxy_GeminiApiKey_ResolutionHierarchy` | 🟢 GREEN (Passed) |
| **AC-2** | Gateway Key Enforcement (HTTP 400 `GEMINI_API_KEY_REQUIRED` when missing in DB) | API / Agent Proxy | `api/agent/agent_proxy_test.go` | `TestAgentProxy_GeminiApiKey_MissingInDatabase_ReturnsHTTP400` | 🟢 GREEN (Passed) |
| **AC-3** | Synchronous Key Validation on Save against Gemini API | API | `api/user_handlers_test.go` | `TestUserHandlers_ValidateGeminiApiKey_SynchronousValidation` | 🟢 GREEN (Passed) |
| **AC-4** | Hard Deletion & Immediate Revocation from Firestore | DB / API | `api/user_handlers_test.go` | `TestUserHandlers_DeleteGeminiApiKey_ImmediateRevocation` | 🟢 GREEN (Passed) |
| **AC-5** | Masked Preview & Status Metadata GET API | API | `api/user_handlers_test.go` | `TestUserHandlers_GetGeminiApiKeyStatus_MaskedPreview` | 🟢 GREEN (Passed) |
| **AC-6** | Dedicated UI Gemini Card, Test Connection button, Project Override toggle | UI / E2E | `ui/e2e/specs/settings/gemini-key.spec.ts` | `TC-P1-4` & `TC-P1-4b` & `TC-P1-4c` | 🟢 GREEN (Passed) |
| **AC-7** | Encrypted JWE Proxy Payload (`userSecretsEncrypted`) forwarding to AI tier | API / Agent Proxy | `api/agent/agent_proxy_test.go` | `TestAgentProxy_GeminiApiKey_EncryptedJWEPayload` | 🟢 GREEN (Passed) |


---

## Summary of Generated Test Files

1. **Backend Go Agent Proxy Tests (`api/agent/agent_proxy_test.go`)**:
   - `TestAgentProxy_GeminiApiKey_MissingInDatabase_ReturnsHTTP400`
   - `TestAgentProxy_GeminiApiKey_ResolutionHierarchy`
   - `TestAgentProxy_GeminiApiKey_EncryptedJWEPayload`
2. **Backend Go Handler Tests (`api/user_handlers_test.go`)**:
   - `TestUserHandlers_ValidateGeminiApiKey_SynchronousValidation`
   - `TestUserHandlers_GetGeminiApiKeyStatus_MaskedPreview`
   - `TestUserHandlers_DeleteGeminiApiKey_ImmediateRevocation`
3. **Frontend Playwright Specs (`ui/e2e/specs/settings/gemini-key.spec.ts`)**:
   - `TC-P1-4: Dedicated Gemini Integration Card renders status badge and masked preview`
   - `TC-P1-4b: Test Connection button triggers synchronous validation`
   - `TC-P1-4c: Project Override toggle enables project-specific key input`
   - `TC-P0-5b: Delete action hard-deletes credential`

---

## Next Steps for Developer (TDD Transition: Red -> Green)

1. Run `bmad-dev-story` or initiate implementation for Story 15.1.
2. Implement backend credential resolution, gateway gating, sync validation handler, and JWE payload additions.
3. Un-skip tests one by one in `api/agent/agent_proxy_test.go` and `api/user_handlers_test.go` to confirm **🟢 GREEN** status.
4. Implement frontend UI Gemini Integration Card component and activate Playwright E2E tests in `ui/e2e/specs/settings/gemini-key.spec.ts`.
