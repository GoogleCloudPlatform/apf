---
title: 'Story 5.4: ADK PostgreSQL Session DB Telemetry Aggregation Engine & GORM ORM Integration'
type: 'feature'
created: '2026-08-10'
status: 'done'
baseline_commit: '3c8899d64e4330cbf1c86d67eba7303f6a5553bb'
review_loop_iteration: 0
context: []
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** Standard Firestore message documents omit or mark intermediate reasoning steps, tool retries, and scratchpad turns as transient (`isTransient: true`), causing token telemetry queries to under-count LLM resource consumption. Additionally, streaming updates to Firestore counter documents on every message turn cause database lock contention and latency degradation.

**Approach:** Introduce a dedicated GORM ORM integration (`gorm.io/driver/postgres`) in the Go API server (`api/`) that queries the ADK PostgreSQL Session Database directly to compute complete, multi-dimensional usage aggregations (Cost/Consumption per User, Project, Organization, and Session specific & turn averages). Cost calculations dynamically apply model rate cards from the Firestore `configs` collection with baseline fallbacks, API queries default to a trailing 30-day window with 5-minute in-memory caching and `singleflight` request coalescing, user/project breakdown queries support `?limit=` (capped at 500) and `?offset=` pagination, backend CSV export is supported via `?format=csv`, cost is returned as floating-point USD rounded to 4 decimal places (`costUSD`), error responses follow standard APF JSON error formats, `/health` endpoints remain lightweight, and RBAC is strictly enforced for System Admins and Org Admins.

## Boundaries & Constraints

**Always:**
- Query the ADK PostgreSQL Session Database (`adk_sessions` and `adk_events` tables) directly for aggregate token counts and event history.
- Extract `organization_id` and `project_id` tenant boundaries directly from existing session and event metadata stored in PostgreSQL.
- Retrieve model pricing rate cards dynamically from the Firestore `configs` collection (`configs/global`). If a model ID is unknown, fallback to default baseline rates and log a warning.
- Format cost calculations as floating-point USD rounded to 4 decimal places (`costUSD`) alongside raw integer token counts (`promptTokens`, `completionTokens`, `totalTokens`).
- Enforce strict RBAC: System Admins have platform-wide telemetry access; Organization Admins are strictly scoped to their `organizationId`.
- Default query date range to the trailing 30 days when `startDate`/`endDate` parameters are omitted.
- Support pagination parameters (`?limit=50` default, capped at max `500`; `?offset=0`) for user and project breakdown listings.
- Support `?format=csv` query parameter to stream downloadable CSV export files directly from the backend API.
- Return standardized APF JSON error response payloads (`{"error": {"code": string, "message": string, "status": int}}`) on client and server errors.
- Apply a 5-minute in-memory TTL cache with `golang.org/x/sync/singleflight` request coalescing for `/api/v1/telemetry/usage` API responses, bypassing cache when `?refresh=true` is supplied.
- Keep `/health` and `/api/v1/health` endpoints lightweight (HTTP server check only).

**Ask First:**
- Modifying default SQL indices on ADK PostgreSQL tables.

**Never:**
- Do not perform blocking multi-document update transactions in Firestore during LLM response streaming.
- Do not display per-message token or cost badges in the user-facing workspace chat pane UI.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| Query Org Usage | `GET /api/v1/telemetry/usage?organizationId=org123` | Returns JSON rollup containing total prompt/completion tokens, calculated `costUSD` (4 decimal places via Firestore rate card), and paginated user/project breakdowns for the last 30 days | Return HTTP 404 (`ORG_NOT_FOUND`) if org not found; 403 (`FORBIDDEN`) if unauthorized |
| Stream CSV Export | `GET /api/v1/telemetry/usage?organizationId=org123&format=csv` | Returns `text/csv` stream with `Content-Disposition: attachment; filename="telemetry-org123.csv"` | Return HTTP 403 (`FORBIDDEN`) if unauthorized |
| Exceed Max Limit | Request with `?limit=1000` | Caps pagination limit to 500 records | Log warning; proceed with limit=500 |
| Concurrent Refresh | Multiple admins request `?refresh=true` simultaneously | Coalesces queries via `singleflight` group so only 1 SQL query executes against PostgreSQL | Share query result across concurrent callers |
| Org Admin Cross-Tenant Query | Org Admin for `org123` requests `organizationId=org456` | Rejects unauthorized cross-tenant request | Return HTTP 403 (`FORBIDDEN`) with standard APF error JSON |
| Query Session Telemetry | `GET /api/v1/telemetry/usage?sessionId=sess456` | Returns total session tokens, turn count, and calculated average token consumption per turn | Return empty stats object if session has no events |
| Unknown Model ID | Event contains unrecognized model `gemini-experimental` | Calculates cost using baseline model rate and logs warning | Log warning metric; complete response |
| Force Refresh Cache | `GET /api/v1/telemetry/usage?organizationId=org123&refresh=true` | Bypasses 5-minute in-memory cache and re-executes SQL aggregations directly against PostgreSQL | Log cache bypass event |
| Database Disconnection | PostgreSQL database unreachable | Telemetry endpoint handles database timeout gracefully | Return HTTP 503 (`DATABASE_UNAVAILABLE`) with APF JSON error |

</frozen-after-approval>

## Code Map

- `api/go.mod` -- Dependency declarations for `gorm.io/driver/postgres`, `gorm.io/gorm`, and `github.com/jackc/pgx/v5`.
- `api/db/adk_postgres.go` -- `NEW` GORM PostgreSQL connection pool manager and database connection factory for ADK Session DB.
- `api/db/adk_telemetry.go` -- `NEW` Struct models (`ADKSession`, `ADKEvent`), rate card lookup from Firestore `configs` with baseline fallbacks, SQL aggregation queries, and CSV formatting logic.
- `api/telemetry_handler.go` -- `NEW` HTTP handler for `/api/v1/telemetry/usage` implementing RBAC checks, 30-day default date windowing, pagination (`limit`/`offset` capped at 500), CSV streaming (`format=csv`), 5-minute TTL caching with `singleflight` coalescing, APF JSON error responses, and `?refresh=true` support.
- `api/routes.go` -- Register `/api/v1/telemetry/usage` handler in HTTP mux.
- `api/telemetry_handler_test.go` -- `NEW` Unit & integration tests for telemetry aggregation queries, RBAC scoping, rate card baseline fallbacks, CSV formatting, singleflight coalescing, pagination caps, and cache bypass.

## Tasks & Acceptance

**Execution:**
- [x] `api/go.mod` -- Add GORM PostgreSQL dependencies -- Enable GORM PostgreSQL connection pool in API server.
- [x] `api/db/adk_postgres.go` -- Build GORM connection manager for ADK Session Database -- Connect API server to PostgreSQL database.
- [x] `api/db/adk_telemetry.go` -- Declare `ADKSession` and `ADKEvent` GORM models, Firestore rate card fetcher with baseline fallback, aggregation queries, and CSV formatter -- Provide SQL queries and export formatting for User, Project, Org, and Session metrics.
- [x] `api/telemetry_handler.go` -- Implement `TelemetryHandler` for `/api/v1/telemetry/usage` with RBAC enforcement, 30-day default window, pagination (`limit`/`offset` capped at 500), CSV export streaming (`format=csv`), 5-min TTL cache, singleflight coalescing, and APF JSON error formatting -- Serve cached multi-dimensional usage analytics and downloadable reports.
- [x] `api/routes.go` -- Register `GET /api/v1/telemetry/usage` endpoint -- Expose telemetry route to Admin Portal.
- [x] `api/telemetry_handler_test.go` -- Write TDD unit tests for telemetry handler, RBAC rules, CSV export streaming, singleflight coalescing, pagination caps, rate card fallbacks, and cache bypass -- Verify 100% test coverage for telemetry calculations.

**Acceptance Criteria:**
- Given a System/Org Admin requesting `/api/v1/telemetry/usage`, when supplying `organizationId`, `projectId`, `userId`, or `sessionId`, then the API verifies RBAC context, queries ADK PostgreSQL, applies the rate card (with baseline fallback for unknown models), and returns complete token consumption and calculated `costUSD` (4 decimal places).
- Given transient tool executions or intermediate scratchpad events in an agent session, when telemetry is computed, then 100% of LLM token activity from PostgreSQL event logs is included in total and average calculations.
- Given no date filter parameters, when querying telemetry, then results default to a trailing 30-day window.
- Given `format=csv`, when requesting telemetry, then the API streams a downloadable `text/csv` file containing the aggregated report.
- Given `?limit=` exceeding 500, when querying telemetry, then the backend caps the page limit to 500 records.
- Given client or server errors, when executing requests, then standard APF JSON error objects (`{"error": {"code": ..., "message": ...}}`) are returned.
- Given repeated telemetry requests within 5 minutes, when `?refresh=true` is omitted, then cached metrics are returned in < 10ms.
- Given multiple concurrent requests with `?refresh=true`, when executed simultaneously, then `singleflight` coalesces the requests to execute only 1 SQL query against PostgreSQL.

## Design Notes

The telemetry service queries `adk_events` joined with `adk_sessions` using SQL aggregate functions (`SUM(total_tokens)`, `AVG(total_tokens)`, `SUM(cost)`) grouped by `user_id`, `project_id`, `organization_id`, or `session_id`. Tenant context (`organization_id`, `project_id`) is parsed directly from PostgreSQL session metadata.

## Verification

**Commands:**
- `go test -v ./api/...` -- expected: PASS all telemetry handler and SQL aggregation unit tests.

## Suggested Review Order

**Database Models & Aggregation Engine**

- GORM PostgreSQL connection pool manager initialization for ADK Session DB
  [`adk_postgres.go:128`](../../../api/db/adk_postgres.go#L128)

- GORM models and multi-dimensional SQL aggregation queries with dynamic rate card calculation
  [`adk_telemetry.go:210`](../../../api/db/adk_telemetry.go#L210)

- Downloadable CSV report generator and formula injection sanitizer
  [`adk_telemetry.go:470`](../../../api/db/adk_telemetry.go#L470)

**API Gateway Router & Telemetry Handler**

- Telemetry Usage HTTP handler with RBAC, 30-day default date windowing, pagination, singleflight coalescing, and APF JSON error responses
  [`telemetry_handler.go:80`](../../../api/telemetry_handler.go#L80)

- Route registration for GET /api/v1/telemetry/usage endpoint
  [`routes.go:284`](../../../api/routes.go#L284)

**Verification & Tests**

- Unit and integration tests covering RBAC, rate cards, CSV streaming, singleflight coalescing, and filtering
  [`telemetry_handler_test.go:1`](../../../api/telemetry_handler_test.go#L1)
