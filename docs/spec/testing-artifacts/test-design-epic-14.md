---
workflowStatus: 'completed'
totalSteps: 5
stepsCompleted: ['step-01-detect-mode', 'step-02-load-context', 'step-03-risk-and-testability', 'step-04-coverage-plan', 'step-05-generate-output']
lastStep: 'step-05-generate-output'
nextStep: ''
lastSaved: '2026-07-26T00:03:30Z'
inputDocuments: ['docs/spec/planning-artifacts/prd.md', 'docs/spec/planning-artifacts/architecture.md', 'docs/spec/planning-artifacts/epics.md', 'docs/spec/implementation-artifacts/14-1-k6-performance-test-suite-scaffolding-and-directory-structure.md']
---

# Test Design: Epic 14 - Performance & Real-Time Latency Testing Infrastructure

**Date:** 2026-07-26  
**Author:** Master Test Architect (Sumeet Singh)  
**Status:** Approved  

---

## Executive Summary

**Scope:** Epic-level test design for **Epic 14: Performance & Real-Time Latency Testing Infrastructure** (incorporating associated operational monitoring in Story 11.10).

**Risk Summary:**
- Total risks identified: 5
- High-priority risks (≥6): 4 (PERF, SEC, OPS)
- Critical categories: PERF (Performance/Scalability), SEC (Security/Auth), OPS (CI/CD Pipeline)

**Coverage Summary:**
- P0 scenarios: 3 (~15 hours)
- P1 scenarios: 3 (~12 hours)
- P2/P3 scenarios: 3 (~8 hours)
- **Total effort**: ~35 hours (~5 engineering days)

---

## Not in Scope

| Item | Reasoning | Mitigation |
| :--- | :--- | :--- |
| **MCP Server Load Testing** | Explicitly excluded by stakeholder direction for the initial performance epic scope. | Will be addressed in a future dedicated MCP testing epic if required. |
| **Production Environment Chaos Injection** | Load testing will run in local and isolated Staging environments only. | VPC and tenant isolation guarantee production safety. |

---

## Risk Assessment

### High-Priority Risks (Score ≥6)

| Risk ID | Category | Description | Probability | Impact | Score | Mitigation | Owner | Timeline |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **R-001** | PERF | 200 concurrent user load causes Firestore connection throttling or Cloud Run container CPU/memory exhaustion. | 3 | 3 | **9** | Perform multi-tenant synthetic data seeding (Story 14.2) + tiered VU ramp-up (5 -> 50 -> 200 VUs) in isolated staging cluster. | Lead QA / DevOps | Sprint 9 |
| **R-002** | PERF | Multi-viewer WebSocket canvas node sync latency exceeds 100ms SLA under 200 VU load. | 3 | 3 | **9** | Implement custom k6 WebSocket trend collector (`ws_frame_latency_ms`) + SSE chunk buffer optimization. | Perf Engineer | Sprint 9 |
| **R-003** | SEC | Staging synthetic performance test authentication leaks test tokens or exposes tenant authorization boundaries. | 2 | 3 | **6** | Implement dual-mode auth helper in `perf/utils/auth.js` strictly scoped to `tenant-perf-staging`. | Security / QA | Sprint 9 |
| **R-004** | OPS | CI/CD automated performance gates fail builds due to transient network noise rather than true latency regressions. | 2 | 3 | **6** | Configure k6 threshold assertions with retry tolerance and standard deviation margins before triggering build failure. | DevOps | Sprint 9 |

### Medium-Priority Risks (Score 3-4)

| Risk ID | Category | Description | Probability | Impact | Score | Mitigation | Owner |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **R-005** | DATA | Performance data seeding pollutes shared staging collections or leaves orphaned storage objects. | 2 | 2 | **4** | Enforce dataset prefixing (`perf_test_*`) and auto-cleanup teardown scripts. | Perf Engineer |

### Risk Category Legend
- **TECH**: Technical/Architecture (flaws, integration, scalability)
- **SEC**: Security (access controls, auth, data exposure)
- **PERF**: Performance (SLA violations, degradation, resource limits)
- **DATA**: Data Integrity (loss, corruption, inconsistency)
- **OPS**: Operations (deployment, config, monitoring)

---

## NFR Planning

**Purpose:** Capture epic-specific NFR thresholds, planned validation, and evidence expected for later `nfr-assess`.

| NFR Category | Requirement / Threshold | Risk Link | Planned Validation | Evidence Needed |
| :--- | :--- | :--- | :--- | :--- |
| **Scalability** | 200 concurrent active users per workspace cluster without 5xx errors (NFR-3.1) | R-001 | k6 load test harness (`scenarios/load_test.js`) | `reports/summary.json` & `reports/summary.html` |
| **Performance (WS)** | Canvas WebSocket node sync latency < 100ms at p95 (NFR-1.2) | R-002 | k6 WS latency suite (`scenarios/latency_test.js`) | k6 `ws_frame_latency_ms` metric trend |
| **Performance (SSE)** | SSE token delivery latency < 50ms per chunk (NFR-1.5) | R-002 | k6 SSE streaming response collector | k6 SSE summary metrics |
| **Performance (Cancel)** | Execution cancellation termination < 1s (NFR-1.4) | R-002 | API cancellation benchmark scenario | k6 API response timing |
| **Performance (Nav)** | Directory tree fetch latency < 150ms (NFR-1.7) | R-001 | HTTP benchmark scenario | k6 API response timing |
| **Security** | Isolated test tenant token validation without boundary bleed | R-003 | Dual-mode auth helper verification | Auth audit log / HTTP 403 test reports |

---

## Entry Criteria

- [x] Epic 14 requirements and stories fully defined in `docs/spec/planning-artifacts/epics.md`
- [x] Story 14.1 specification created and context engine initialized (`ready-for-dev`)
- [ ] Isolated staging performance testing cluster provisioned
- [ ] Grafana k6 runner installed in local/CI environment (`mise install` / `k6`)

## Exit Criteria

- [ ] All P0 performance test scenarios passing (100% pass rate)
- [ ] 200 concurrent user load test executed with 0% 5xx errors and API p95 < 500ms
- [ ] WebSocket frame sync latency confirmed < 100ms at p95
- [ ] SSE token delivery latency confirmed < 50ms
- [ ] Automated k6 regression quality gate integrated into CI pipeline

---

## Test Coverage Plan

### P0 (Critical) - Run on every performance test suite execution

**Criteria**: Core scalability + High risk (≥6) + SLA validation

| Requirement | Test Level | Risk Link | Test Scenario | Owner | Notes |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **NFR-3.1** (200 Users) | Performance / Load | R-001 | 200 VU Ramp & Sustained Load Harness (`scenarios/load_test.js`) | Perf QA | Validates 200 concurrent active users |
| **NFR-1.2** (WS Sync) | Performance / Latency | R-002 | WebSocket Multi-Viewer Frame Sync Latency (< 100ms) | Perf QA | Measures frame broadcast timing |
| **NFR-1.5** (SSE Stream)| Performance / Latency | R-002 | SSE Chat Response Token Delivery Latency (< 50ms) | Perf QA | Measures streaming chunk delivery |

**Total P0**: 3 scenarios, ~15 hours

### P1 (High) - Run on PR & Staging Release Deployments

**Criteria**: Important scaffolding + Infrastructure + Admin Telemetry

| Requirement | Test Level | Risk Link | Test Scenario | Owner | Notes |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **AR-PERF-1** (k6 Setup) | API / Integration | R-004 | Baseline Smoke Test & Dual Report Generation (`scenarios/smoke_test.js`) | Lead QA | Verifies health, auth, reports |
| **AR-PERF-2** (Data Seed)| CLI / Integration | R-005 | Synthetic Multi-Tenant Data Seeding & Teardown Verification | Dev / QA | Verifies 200 VU data seeding |
| **FR-14.1** (Admin UI) | E2E / UI (Playwright) | R-004 | Admin Console Performance Dashboard Telemetry Render (Story 11.10) | Frontend | Verifies APM dashboard widgets |

**Total P1**: 3 scenarios, ~12 hours

### P2/P3 (Medium/Low) - Run Nightly / On-Demand

**Criteria**: Edge latency + Report styling

| Requirement | Test Level | Test Scenario | Owner | Notes |
| :--- | :--- | :--- | :--- | :--- |
| **NFR-1.4** (Cancellation)| API Benchmark | Execution Cancellation Abort Latency (< 1s) under load | QA | Tests context cancellation |
| **NFR-1.7** (Directory) | API Benchmark | Recursive Directory Tree Lazy-Load Fetch Latency (< 150ms) | QA | Tests directory tree fetch |
| **AR-PERF-1** (Reports) | File / Formatting | `reports/summary.html` and `reports/summary.json` layout audit | QA | Validates report formats |

**Total P2/P3**: 3 scenarios, ~8 hours

---

## Execution Order

### 1. Smoke Tests (< 1 min)
- [ ] Run `mise run perf:smoke` (`perf/scenarios/smoke_test.js`) against target server
- [ ] Verify health (`/api/v1/health`), auth profile (`/api/v1/user/profile`), and project list (`/api/v1/projects`)

### 2. Data Seeding & Setup (< 3 min)
- [ ] Execute `go run ./scripts/perf-seed/main.go --tenants=1 --users=200`
- [ ] Verify synthetic workspace fixtures in Firestore and GCS

### 3. Real-Time Latency Suite (< 5 min)
- [ ] Run `mise run perf:latency` (`perf/scenarios/latency_test.js`)
- [ ] Assert WebSocket frame sync < 100ms and SSE token delivery < 50ms

### 4. 200 Concurrent User Load Test (< 12 min)
- [ ] Run `mise run perf:load` (`perf/scenarios/load_test.js`)
- [ ] Validate 200 VU ramp (0->50 VUs 2m, 50->200 VUs 3m, hold 10m) with 0% 5xx errors and API p95 < 500ms

---

## Resource Estimates

| Priority | Count | Hours/Scenario | Total Hours | Notes |
| :--- | :--- | :--- | :--- | :--- |
| **P0** | 3 | 5.0 | 15.0 | Complex WebSocket & 200 VU load script creation |
| **P1** | 3 | 4.0 | 12.0 | Seed CLI, k6 scaffolding, Admin UI test |
| **P2/P3**| 3 | 2.6 | 8.0 | Cancellation timing & report verification |
| **Total**| **9** | **-** | **~35.0** | **~5 engineering days** |

---

## Quality Gate Criteria

### Pass/Fail Thresholds
- **P0 pass rate**: 100% (no exceptions)
- **P1 pass rate**: ≥ 95%
- **200 VU Load Test**: 0% unhandled 5xx errors; HTTP API p95 < 500ms
- **WebSocket Frame Sync**: p95 < 100ms
- **SSE Token Delivery**: p95 < 500ms per chunk

### Non-Negotiable Requirements
- [ ] All P0 performance tests pass
- [ ] High risks R-001, R-002, R-003, R-004 mitigated
- [ ] k6 performance quality gate integrated into CI pipeline with regression alarming

---

## Related Documents

- PRD: [prd.md](file:///usr/local/google/home/sumeetsing/workspace/agentic-foundry/docs/spec/planning-artifacts/prd.md)
- Epics: [epics.md](file:///usr/local/google/home/sumeetsing/workspace/agentic-foundry/docs/spec/planning-artifacts/epics.md#Epic%2014:%20Performance%20&%20Real-Time%20Latency%20Testing%20Infrastructure)
- Story 14.1: [14-1-k6-performance-test-suite-scaffolding-and-directory-structure.md](file:///usr/local/google/home/sumeetsing/workspace/agentic-foundry/docs/spec/implementation-artifacts/14-1-k6-performance-test-suite-scaffolding-and-directory-structure.md)
- Sprint Status: [sprint-status.yaml](file:///usr/local/google/home/sumeetsing/workspace/agentic-foundry/docs/spec/implementation-artifacts/sprint-status.yaml)

---

**Generated by**: BMad TEA Agent - Test Architect Module  
**Workflow**: `bmad-testarch-test-design`  
**Version**: 4.0 (BMad v6)  
