# Story 14.1: k6 Performance Test Suite Scaffolding & Directory Structure

Status: complete

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a Lead QA Engineer / Performance Engineer,  
I want to establish a structured performance testing directory at `{project-root}/perf/`,  
so that our performance tests follow modular best practices with clean separation of scenarios, utilities, configurations, and reports.

## Acceptance Criteria

1. **Given** the performance testing repository workspace
2. **When** initializing the `{project-root}/perf/` directory
3. **Then** the folder structure is scaffolded with `scenarios/`, `utils/`, `config/`, and `reports/` subdirectories
4. **And** modular k6 ES6 scripts are established for HTTP data plane and WebSocket streaming connections
5. **And** environment configuration profiles (local, staging, prod) and execution helper scripts are committed to the codebase.

## Tasks / Subtasks

- [x] Create `{project-root}/perf/` folder hierarchy (`scenarios/`, `utils/`, `config/`, `reports/`) with `perf/reports/.gitkeep` and update `.gitignore` (AC: 3)
- [x] Implement k6 configuration profile loader in `perf/config/profiles.js` supporting target URL resolution (`http://localhost:8081` default, `K6_TARGET_URL` / `TARGET_ENV` overrides), SLA thresholds, and tiered VU ramps (AC: 5)
- [x] Implement dual-mode authentication helper in `perf/utils/auth.js` (OIDC/Firebase REST API fetch for staging/prod, mock JWT/header fallback for local dev) (AC: 4)
- [x] Implement custom HTTP and WebSocket helpers in `perf/utils/http_client.js` and `perf/utils/ws_client.js` with custom k6 metrics (`ws_connecting_errors`, `ws_frame_latency_ms`) and graceful disconnect handling (AC: 4)
- [x] Implement initial baseline smoke test scenario in `perf/scenarios/smoke_test.js` with `handleSummary` exporting `reports/summary.json` and `reports/summary.html` (AC: 4, 5)
- [x] Add `perf/package.json` with npm scripts and add `perf:smoke`, `perf:load`, and `perf:latency` task aliases to root `mise.toml` (AC: 5)

## Dev Notes

### Architecture & Standards
- **Folder Structure**:
  ```
  perf/
  ├── config/
  │   └── profiles.js         # Target URL resolution (default http://localhost:8081, K6_TARGET_URL/TARGET_ENV overrides), thresholds (p95 < 500ms, error < 1% + K6_P95_THRESHOLD_MS env override), and tiered vUser ramp profiles (local: 5 VUs/30s, staging: 50 VUs/5m, stress: 200 VUs/10m)
  ├── utils/
  │   ├── auth.js             # Dual-mode auth helper (OIDC/Firebase REST & Local mock JWT)
  │   ├── http_client.js      # Custom k6 HTTP wrapper with tagging
  │   └── ws_client.js        # k6/ws WebSocket wrapper, graceful disconnects & custom metrics (ws_connecting_errors, ws_frame_latency_ms)
  ├── scenarios/
  │   ├── smoke_test.js       # Baseline API sanity check scenario (/api/v1/health, /api/v1/user/profile, /api/v1/projects) with dual HTML/JSON report export
  │   ├── load_test.js        # 200 concurrent user load test scenario
  │   └── latency_test.js     # Real-time WebSocket/SSE latency benchmarking scenario
  ├── reports/                # Output directory for summary JSON/HTML reports (.gitkeep tracked; *.json & *.html ignored in .gitignore)
  └── package.json            # k6 script execution dependencies and npm scripts
  ```
- **Technology Stack**: Grafana k6 (Native ES6 JS runtime), `k6/http`, `k6/ws`, `k6/metrics`, `k6-reporter` (HTML report generation).
- **CLI / Task Integration**: Root `mise.toml` updated with `perf:smoke`, `perf:load`, and `perf:latency` tasks pointing to `k6 run` scenarios.

### Source Tree Components to Create / Modify
- [perf/package.json](perf/package.json)
- [perf/config/profiles.js](perf/config/profiles.js)
- [perf/utils/auth.js](perf/utils/auth.js)
- [perf/utils/http_client.js](perf/utils/http_client.js)
- [perf/utils/ws_client.js](perf/utils/ws_client.js)
- [perf/scenarios/smoke_test.js](perf/scenarios/smoke_test.js)
- [mise.toml](mise.toml)
- [.gitignore](.gitignore)

### References
- [epics.md#Epic 14: Performance & Real-Time Latency Testing Infrastructure](docs/spec/planning-artifacts/epics.md)
- [sprint-status.yaml](docs/spec/implementation-artifacts/sprint-status.yaml)

## Dev Agent Record

### Agent Model Used

Gemini Next

### Debug Log References

N/A

### Completion Notes List

- Story file created and context engine initialized.
- `sprint-status.yaml` updated with `14-1-k6-performance-test-suite-scaffolding-and-directory-structure: done`.
- Created `{project-root}/perf/` directory structure with `scenarios/`, `utils/`, `config/`, and `reports/` (.gitkeep).
- Updated `.gitignore` with ignores for `perf/reports/*.json` and `perf/reports/*.html`.
- Implemented `perf/config/profiles.js` supporting dynamic target URL resolution, thresholds, and tiered VU ramps.
- Implemented `perf/utils/auth.js`, `perf/utils/http_client.js`, and `perf/utils/ws_client.js` with custom k6 metrics.
- Implemented baseline smoke test scenario in `perf/scenarios/smoke_test.js` with dual HTML/JSON report export.
- Created `perf/package.json` and added `perf:smoke`, `perf:load`, `perf:latency` task aliases to root `mise.toml`.

### File List

- `docs/spec/implementation-artifacts/14-1-k6-performance-test-suite-scaffolding-and-directory-structure.md`
- `docs/spec/implementation-artifacts/sprint-status.yaml`
- `perf/config/profiles.js`
- `perf/utils/auth.js`
- `perf/utils/http_client.js`
- `perf/utils/ws_client.js`
- `perf/scenarios/smoke_test.js`
- `perf/package.json`
- `perf/reports/.gitkeep`
- `perf/scenarios/.gitkeep`
- `perf/utils/.gitkeep`
- `perf/config/.gitkeep`
- `mise.toml`
- `.gitignore`

