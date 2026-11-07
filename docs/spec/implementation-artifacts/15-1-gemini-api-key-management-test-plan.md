---
workflowStatus: 'completed'
totalSteps: 5
stepsCompleted: ['step-01-detect-mode', 'step-02-load-context', 'step-03-risk-and-testability', 'step-04-coverage-plan', 'step-05-generate-output']
lastStep: 'step-05-generate-output'
nextStep: ''
lastSaved: '2026-07-26T04:42:39Z'
inputDocuments:
  - 'docs/spec/implementation-artifacts/15-1-gemini-api-key-management.md'
  - 'docs/spec/planning-artifacts/epics.md'
  - 'api/agent/agent_proxy.go'
  - 'api/db/credentials.go'
  - 'api/db/user.go'
  - 'api/integrations/github.go'
---

# Test Design: Story 15.1 - Gemini API Key Management

**Date:** 2026-07-26
**Author:** Sumeet / Master Test Architect
**Status:** Approved

---

## Executive Summary

**Scope:** Epic-Level / Story-Level Test Design for **Story 15.1: Gemini API Key Management** in **Epic 15 (External Integrations Management)**.

**Risk Summary:**
- Total risks identified: 6
- High-priority risks (Score ≥6): 4
- Critical categories: **SEC** (Security), **OPS** (Operations), **DATA** (Data Integrity)

**Coverage Summary:**
- P0 scenarios: 5 (~12 hours)
- P1 scenarios: 4 (~10 hours)
- P2/P3 scenarios: 2 (~4 hours)
- **Total effort**: ~26 hours (~3.2 days)

---

## Elicited Test Architecture Decisions

1. **Modular Codebase Test Locations**:
   - Backend Go unit and API integration tests placed alongside package code (`api/agent/agent_proxy_test.go`, `api/user_handlers_test.go`, `api/cmd/migrate_gemini_integration/main_test.go`).
   - Frontend E2E specs placed in `ui/e2e/specs/settings/gemini-key.spec.ts`.
2. **Mock HTTP Server for Validation**:
   - Synchronous Gemini connection test calls during settings save (`TC-P1-2`) are stubbed using `httptest.NewServer` or mock interface adapters to guarantee fast, deterministic offline execution without outbound network dependency or quota consumption.
3. **Manual CLI Migration Execution with Dry-Run**:
   - The one-time migration script `cmd/migrate_gemini_integration` supports `--dry-run` for manual CLI execution by developers/operators prior to live execution in staging/prod. Automated CI pipeline dry-run gating is not required.

---

## Not in Scope

| Item | Reasoning | Mitigation |
| :--- | :--- | :--- |
| **Server Environment Variable (`GEMINI_API_KEY`) Fallback** | Explicitly forbidden in design. System operates strictly database-only across all environments. | Gateway blocks requests missing DB key credentials with HTTP 400 (`GEMINI_API_KEY_REQUIRED`). |
| **Automated CI Pipeline Migration Gating** | User explicitly requested manual execution for migration script in deployment workflows. | Script CLI provides `--dry-run` flag for operator safety checks before executing live upserts. |
| **Third-Party Model Providers (OpenAI, Anthropic)** | Scope restricted to Gemini API Key integration management for Story 15.1. | Extensible `AppIntegration` catalog schema supports future providers without breaking 15.1. |

---

## Risk Assessment

### High-Priority Risks (Score ≥6)

| Risk ID | Category | Description | Probability | Impact | Score | Mitigation | Owner | Timeline |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **R-001** | **SEC** | Unencrypted or plaintext Gemini API key exposure in logs, GET API payloads, or Firestore database collections. | 2 | 3 | **6** | Mandatory AES-GCM encryption (`SECRETS_ENCRYPTION_KEY`) on write; GET API endpoints return only masked previews (`••••••••a1b2`); sanitize log output formatters. | Security / Dev Lead | Sprint 10 |
| **R-002** | **SEC** | Gateway bypass where AI inference calls execute without a valid Gemini API key document configured in the database. | 2 | 3 | **6** | `AgentProxy` gateway gating checks key document presence in database and returns HTTP 400 (`GEMINI_API_KEY_REQUIRED`) with structured error JSON before calling AI tier. | Dev Lead | Sprint 10 |
| **R-003** | **SEC** | Cross-user or cross-project credential leakage when resolving project-level vs user-level keys. | 2 | 3 | **6** | Scoped `UserProject` (`appId: "gemini"`, `userId`, `projectId`) queries; strict multi-tenant isolation via `organizationCollection(orgID, ...)`. | Dev Lead | Sprint 10 |
| **R-004** | **OPS** | Migration script failure or partial catalog update in staging/prod during deployment. | 2 | 3 | **6** | Executable CLI tool `cmd/migrate_gemini_integration` supports `--dry-run` and uses idempotent bulk upserts with ADC credentials for manual operator runs. | DevOps Lead | Sprint 10 |

### Medium-Priority Risks (Score 3-4)

| Risk ID | Category | Description | Probability | Impact | Score | Mitigation | Owner |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **R-005** | **PERF** | Synchronous validation latency during API key submission in Settings UI. | 2 | 2 | **4** | Lightweight API connection ping with timeout (≤3s) before committing to Firestore. | Dev Lead |
| **R-006** | **DATA** | Orphaned credentials after user deletes key. | 2 | 2 | **4** | Hard-deletion handler instantly removes `AppCredential` or `UserProject` Firestore document, invalidating gateway checks immediately. | Dev Lead |

---

## NFR Planning

| NFR Category | Requirement / Threshold | Risk Link | Planned Validation | Evidence Needed |
| :--- | :--- | :--- | :--- | :--- |
| **Security** | 100% encryption at rest; 0% raw API key exposure in GET API responses or logs | R-001 | API & Unit tests for AES-GCM encryption and masked preview (`••••••••a1b2`) | Go unit test logs & API payload audit |
| **Security / Auth** | 100% enforcement of gateway key check; HTTP 400 `GEMINI_API_KEY_REQUIRED` when missing | R-002 | `AgentProxy` integration test suite for unauthenticated execution attempts | API test execution logs |
| **Reliability** | Idempotent deployment migration script with 0% data corruption across orgs | R-004 | CLI dry-run and live migration execution tests against test Firestore database | Migration script log output |
| **Performance** | Synchronous key validation on save latency ≤ 3.0s | R-005 | API handler benchmarking test | Benchmark report |

---

## Entry & Exit Criteria

### Entry Criteria
- [x] Story 15.1 specification ([15-1-gemini-api-key-management.md](docs/spec/implementation-artifacts/15-1-gemini-api-key-management.md)) approved and set to `ready-for-dev`.
- [x] Test environment with Firestore emulator (`FIRESTORE_EMULATOR_HOST`) and `SECRETS_ENCRYPTION_KEY` configured.
- [x] Seed template `api/seeding/templates/showcase/project.json` containing simulated dev/test key fixture (`gemini_simulated_api_key_12345`).

### Exit Criteria
- [ ] 100% pass rate on all P0 (Critical) security and gateway gating tests.
- [ ] ≥95% pass rate on P1 (High) resolution hierarchy and migration tests.
- [ ] Zero unencrypted API keys detected in Firestore storage or API response logs.
- [ ] Executable migration CLI `cmd/migrate_gemini_integration` verified with `--dry-run` and live upserts in manual execution testing.

---

## Test Coverage Plan

### P0 (Critical) - Run on every commit

**Criteria**: Security enforcement + Gateway key gating + Data protection + Hard revocation

| Scenario ID | Requirement / Description | Test Level | Risk Link | Target File Path | Owner | Expected Outcome / Notes |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **TC-P0-1** | Gateway Key Enforcement (Missing Key) | API / Integration | R-002 | `api/agent/agent_proxy_test.go` | Dev | Gateway blocks execution and returns HTTP 400 Bad Request with JSON `{ "code": "GEMINI_API_KEY_REQUIRED", ... }`. |
| **TC-P0-2** | Encryption at Rest | Unit / DB | R-001 | `api/db/user_test.go` | Dev | Gemini API key field is stored encrypted via AES-GCM in Firestore using `SECRETS_ENCRYPTION_KEY`. |
| **TC-P0-3** | Masked Preview in GET Endpoints | API | R-001 | `api/user_handlers_test.go` | Dev | GET credential status endpoints return masked key (`••••••••a1b2`), `configured: true`, and `lastTestedAt` timestamp; raw key is never returned. |
| **TC-P0-4** | Encrypted JWE Proxy Payload Forwarding | API / Agent | R-001 | `api/agent/agent_proxy_test.go` | Dev | `AgentProxy` packages resolved `GeminiApiKey` into JWE `userSecretsEncrypted` payload forwarded to AI service proxy. |
| **TC-P0-5** | Hard Deletion & Immediate Revocation | API / DB | R-006 | `api/user_handlers_test.go` | Dev | Deleting key removes document from Firestore; subsequent AI calls instantly fail with HTTP 400 `GEMINI_API_KEY_REQUIRED`. |

**Total P0**: 10 tests, ~12 hours

---

### P1 (High) - Run on PR to main

**Criteria**: Resolution hierarchy + Key validation + Migration CLI + UI card interactions

| Scenario ID | Requirement / Description | Test Level | Risk Link | Target File Path | Owner | Expected Outcome / Notes |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **TC-P1-1** | Credential Resolution Hierarchy | API / Integration | R-003 | `api/agent/agent_proxy_test.go` | Dev | `db.ResolveAppCredential` correctly resolves Project Override (`UserProject`) -> User Global Secret (`AppCredential`) -> Dev/Test DB seed credential. |
| **TC-P1-2** | Synchronous API Key Validation on Save | API / Integration | R-005 | `api/user_handlers_test.go` | Dev | Endpoint validates format and tests ping against `httptest.NewServer` stub before saving; returns 400 on invalid key. |
| **TC-P1-3** | One-Time Migration CLI Script | Integration / CLI | R-004 | `api/cmd/migrate_gemini_integration/main_test.go` | DevOps | `cmd/migrate_gemini_integration` `--dry-run` previews catalog changes; live execution idempotently provisions global & org `app_integrations`. |
| **TC-P1-4** | Settings UI Gemini Card & Project Override | UI / E2E | R-003 | `ui/e2e/specs/settings/gemini-key.spec.ts` | QA | UI renders Gemini Card, displays status badge, handles "Test Connection", toggles project override, and saves key seamlessly. |

**Total P1**: 9 tests, ~10 hours

---

### P2 (Medium) - Run nightly/weekly

**Criteria**: Dev/Test seed data + Multi-tenant boundary checks

| Scenario ID | Requirement / Description | Test Level | Risk Link | Target File Path | Owner | Expected Outcome / Notes |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **TC-P2-1** | Database Seed Credential Loading in Dev/Test | Integration | R-003 | `api/agent/agent_proxy_test.go` | Dev | Pre-populated seed credential (`gemini_simulated_api_key_12345`) loads exclusively from database without reading env variables. |
| **TC-P2-2** | Multi-Tenant Org Catalog Isolation | Integration / DB | R-003 | `api/db/config_test.go` | Dev | `ListOrgAppIntegrations` restricts `"gemini"` configuration changes to the requesting tenant organization. |

**Total P2**: 4 tests, ~4 hours

---

## Execution Strategy

```
┌────────────────────────────────────────────────────────────────────────┐
│                              PR GATES                                  │
│  - TC-P0-1 through TC-P0-5 (Security, Gateway 400, Encryption, JWE)    │
│  - TC-P1-1 through TC-P1-3 (Hierarchy, Sync Validation, CLI Unit)      │
│  - Target execution time: < 8 minutes                                  │
└────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌────────────────────────────────────────────────────────────────────────┐
│                           NIGHTLY / CI SUITE                           │
│  - TC-P1-4 (UI / E2E Dedicated Gemini Card Flow in Playwright)          │
│  - TC-P2-1 & TC-P2-2 (Dev/Test Seed Ingestion & Multi-Tenant Boundaries)│
└────────────────────────────────────────────────────────────────────────┘
```

---

## Quality Gate Criteria

### Pass/Fail Thresholds
- **P0 Pass Rate**: **100%** (no exceptions)
- **P1 Pass Rate**: **≥95%** (waivers required for any open issue)
- **P2 Pass Rate**: **≥90%**
- **High-Risk Mitigations**: **100%** complete (R-001, R-002, R-003, R-004)

### Non-Negotiable Requirements
- [ ] 0 plaintext API keys written to Firestore or log files.
- [ ] HTTP 400 `GEMINI_API_KEY_REQUIRED` verified for missing key execution attempts.
- [ ] Migration script `--dry-run` and live execution manually verified clean by operator prior to deployment.
- [ ] JWE `userSecretsEncrypted` payload payload contract verified with AI proxy.

---

## Follow-on Workflows

- Run `bmad-testarch-atdd` to generate red-phase P0 acceptance test scaffolds.
- Run `bmad-testarch-automate` after implementation to expand test suite automation.

---

**Generated by**: BMad TEA Agent - Master Test Architect
**Workflow**: `bmad-testarch-test-design`
**Target Story**: `Story 15.1 (15-1-gemini-api-key-management.md)`
