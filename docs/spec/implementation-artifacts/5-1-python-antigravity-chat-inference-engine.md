# Story 5.1: Python Antigravity Chat Inference Engine

Status: review

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a User,
I want a Python-based execution core utilizing the advanced `google-antigravity` SDK,
so that chat inferences are processed in parallel with baseline pipelines, delivering high-fidelity streaming responses.

## Acceptance Criteria

1. **Given** downstream REST/SSE requests forwarded from the Go middleware gateway, **When** executing conversational prompts under the `/agy` service context, **Then** the Python engine MUST process prompts dynamically using the native **`google-antigravity` SDK (version >= 0.1.0)**. (AC: 5.1.1)
2. **Given** active chat sessions, **When** generating agent responses, **Then** the Python engine MUST stream responses utilizing chunked Server-Sent Events (SSE) back to the Go proxy gateway in under a baseline p95 latency of **2.0 seconds**. (AC: 5.1.2)
3. **Given** local development and execution runtime perimeters, **When** deploying or testing the service, **Then** all python package dependencies MUST be managed strictly via the **`uv` package manager** under standard Python `>=3.13` guidelines, passing validation checks. (AC: 5.1.3)

---

## Tasks / Subtasks

- [x] **Task 1: Bootstrap the Python agy Agent Service Core (AC: 5.1.1, 5.1.3)**
  - [x] **Subtask 1.1:** Configure `agy/pyproject.toml` to run Python `>=3.13` and lock dependencies on `google-antigravity>=0.1.0` and a lightweight Python web frame (specifically **FastAPI** and **uvicorn**).
  - [x] **Subtask 1.2:** Implement the central runtime execution server inside [agy/main.py](agy/main.py) utilizing the `uv` toolchain.
- [x] **Task 2: Implement SSE Streaming Inferences API Endpoint (AC: 5.1.1, 5.1.2)**
  - [x] **Subtask 2.1:** Implement `POST /api/v1/chat` endpoint in `agy/main.py` accepting the proxy-forwarded prompts JSON blocks.
  - [x] **Subtask 2.2:** Integrate the `google-antigravity` SDK chat sessions: initialize agent engines conversationally, and stream responses dynamically utilizing FastAPI `StreamingResponse` helpers.
  - [x] **Subtask 2.3:** Implement standard context cancels: ensure downstream prompt inference loops are terminated instantly if the caller drops.
- [x] **Task 3: Complete Verification Telemetry & Unit Tests (AC: 5.1.1 - 5.1.3)**
  - [x] **Subtask 3.1:** Write Python backend mock unit tests in `agy/tests/test_main.py` verifying request structures, SSE streams, and context drops handlers.
  - [x] **Subtask 3.2:** Run python formatters and lints using standard formatters: run `ruff check` or `uv run ruff format` ensuring clean scores.

---

## Dev Notes

### 🏗️ 1. Architectural Python Service Guardrails
*   **uv Toolchain Compliance:** The python agent backend MUST rely exclusively on `uv` package management. Do NOT introduce raw `pip` dependencies lists or virtualenv configuration files.
*   **FastAPI & Streaming SSE:** Inbound payloads are managed via pydantic models. Conversational tokens MUST stream asynchronously to minimize p95 latencies under the required 2.0s limit.

---

## 🛠️ Project Structure & Source Maps

### 1. Source Files Matrix

| File Path | Status | Operational Role |
| :--- | :--- | :--- |
| [agy/pyproject.toml](agy/pyproject.toml) | `UPDATE` | Declare FastAPI, uvicorn, and `google-antigravity` target dependency pins. |
| [agy/main.py](agy/main.py) | `UPDATE` | Implement the FastAPI REST server, prompt models, `google-antigravity` chat stream connectors. |
| [agy/tests/test_main.py](agy/tests/test_main.py) | `NEW` | Python unit test suites verifying stream processing, prompt routing, and session cancellations. |

---

## Dev Agent Record

### Agent Model Used

Gemini Next

### Debug Log References

- Mock testing executions `uv run pytest` executed with exit code 0, passing all 3 unit tests.
- Py_compile diagnostic check `python3 -m py_compile main.py tests/test_main.py` completed with exit code 0.

### Completion Notes List

- ✅ FastAPI microservice application scaffolding, CORS mappings, and REST health check endpoints integrated in `agy/main.py`.
- ✅ Asynchronous context manager sessions (`async with Agent(...)`) streaming prompts cursor deltas natively using the `google-antigravity` SDK in `main.py`.
- ✅ Isolated TestClient TDD mock units testing framework configured in `agy/tests/test_main.py`.
- ✅ `fastapi`, `uvicorn`, `pytest`, and `httpx` dependencies locked inside `agy/pyproject.toml`.

### File List

- `agy/pyproject.toml`
- `agy/main.py`
- `agy/tests/test_main.py`
- `agy/uv.lock`
