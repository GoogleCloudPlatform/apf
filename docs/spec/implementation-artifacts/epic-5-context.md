# Epic 5 Context: Parallel Dual-Engine AI Orchestration

<!-- Compiled from planning artifacts. Edit freely. Regenerate with compile-epic-context if planning docs change. -->

## Goal

Eliminate single-framework risk by establishing concurrent execution paths inside backend middleware, proxying inferences to either legacy GenKit ADK or a high-performance downstream Python Antigravity SDK service (`/agy`), while providing reliable multi-dimensional telemetry aggregation directly from the ADK PostgreSQL Session Database.

## Stories

- Story 5.1: Python Antigravity Chat Inference Engine
- Story 5.2: Go Dynamic Reverse-Proxy AI Gateway
- Story 5.3: Antigravity Dynamic Tool Execution Adapter
- Story 5.4: ADK PostgreSQL Session DB Telemetry Aggregation Engine & GORM ORM Integration

## Requirements & Constraints

- **Streaming Latency:** Response stream must route back through the Go gateway with p95 latency under 2.0s and reverse-proxy overhead < 50ms.
- **Data Integrity:** Telemetry and usage aggregation MUST query the ADK PostgreSQL Session Database directly to ensure 100% complete accounting of intermediate tool executions, scratchpad turns, and transient events.
- **OIDC & Security:** Go gateway must verify OIDC tokens, decrypt AES-256-GCM secrets, and propagate security context downstream.

## Technical Decisions

- **AI Core Architecture:** Dual-engine path supporting legacy GenKit ADK and native Python `google-antigravity` SDK core.
- **ORM & Database Storage:** Introduce GORM PostgreSQL connection pool (`gorm.io/driver/postgres`) and models (`ADKSession`, `ADKEvent`) in the `api/` tier to query the ADK PostgreSQL database.
- **API Endpoint:** `/api/v1/telemetry/usage` providing aggregated usage reporting (Cost/Consumption per User, Project, Organization, and Session).
