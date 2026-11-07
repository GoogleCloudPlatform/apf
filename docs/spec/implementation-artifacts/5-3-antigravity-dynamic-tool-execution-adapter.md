# Story 5.3: Antigravity Dynamic Tool Execution Adapter

Status: ready-for-dev

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a Product Owner,
I want visual tools synchronization adapters (Figma, Stitch, Draw.io) to support execution via the Python `google-antigravity` SDK core,
so that tool payloads edits are resolved with 100% functional parity in parallel with GenKit MPC systems.

## Acceptance Criteria

1. **Given** visual canvas mutations trigger tool sync handlers under the **`agy-sdk`** core engine, **When** executing adapter commands, **Then** the Python engine MUST parse, validate, and execute visual tool bindings natively using the `google-antigravity` SDK. (AC: 5.3.1)
2. **Given** dynamic tool execution, **When** mutating visual specifications (Draw.io XML/JSON, Stitch component trees), **Then** the agy-sdk tool bindings MUST achieve **100% functional parity** with the ADK baseline implementations. (AC: 5.3.2)
3. **Given** parallel execution routes, **When** a user performs visual edit integrations, **Then** tool execution state logs MUST serialize into the central Firestore databases using identical object mapping profiles. (AC: 5.3.3)

---

## Tasks / Subtasks

- [ ] **Task 1: Map the agy-sdk Model Context Protocol (MCP) Client Bindings (AC: 5.3.1)**
  - [ ] Implement robust MCP client bindings inside the Python `agy` service utilizing the native `google-antigravity` SDK.
  - [ ] Implement network handshakes and payload encryptions (JWE) to securely exchange user setting keys with local/remote MCP tool servers.
- [ ] **Task 2: Build the Dynamic Tool Execution Adapters (AC: 5.3.1, 5.3.2)**
  - [ ] Implement the Draw.io adapter under `agy`: parse visual diagram XML into agent semantic models, programmatically alter diagram blocks, and push changes back to active canvas enclaves.
  - [ ] Implement the Stitch UI builder adapter under `agy`: parse user requirements blueprints, and output functional frontend component trees.
  - [ ] Ensure that dynamic tool JSON inputs/outputs conversions achieve 100% functional parity with GenKit ADK counterparts.
- [ ] **Task 3: Resolve Parity Serialization & Audit Logs (AC: 5.3.3)**
  - [ ] Bind all `agy` tool execution states to standard telemetry streams.
  - [ ] Ensure that tool traces, completion parameters, and audit records map identically to standard OpenTelemetry (OTel) objects persisted in Google Cloud logs.
- [ ] **Task 4: Complete Verification Testing (AC: 5.3.1 - 5.3.3)**
  - [ ] Write Python integration tests in `agy/tests/test_adapters.py` verifying tool handshakes, XML/JSON parsing boundaries, and serializations.
  - [ ] Run python compilation and lint checks: run `uv run ruff check` in the `agy` folder.

---

## Dev Notes

### 🏗️ 1. Architectural Tooling Guardrails
*   **Decoupled Schema Contracts:** Python and Go tool handlers MUST enforce identical request/response schemas. Avoid introducing any framework-specific custom types that could corrupt joint Firestore databases.
*   **PII & Token Scrubbing:** Any user credentials or API tokens mapped by the python tool execution engine MUST undergo application-layer encryption and sanitization using validated OIDC credentials tracing.

---

## 🛠️ Project Structure & Source Maps

### 1. Source Files Matrix

| File Path | Status | Operational Role |
| :--- | :--- | :--- |
| `agy/main.py` | `UPDATE` | Expose dynamic REST endpoints to intercept, map, and invoke custom tool execution adapters. |
| `agy/integrations/drawio.py` | `NEW` | Python-based Draw.io sync adapter; XML parsing, visual node mapping, and JWE exchanges. |
| `agy/integrations/stitch.py` | `NEW` | Python-based Google Stitch UI builder; component tree compiler and mock schemas sync. |
| `agy/tests/test_adapters.py` | `NEW` | Python integration test suites verifying adapter executions, schema parity, and trace serializations. |

---

## Dev Agent Record

### Agent Model Used

Gemini Next

### Debug Log References

### Completion Notes List

### File List
