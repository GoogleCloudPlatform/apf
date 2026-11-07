# Story 1.2: Agent Sidebar Chat

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a PM/UX Designer,
I want a sidebar chat interface to instruct agents visually or narratively,
so that I can collaborate directly with AI assistants during my workflow.

## Acceptance Criteria

1. **Given** the user is in the APF workspace, **When** they open the sidebar chat pane and submit an interactive prompt, **Then** the system MUST route the prompt via standard Server-Sent Events (SSE) to active GenKit backend models. (AC: 1.2.1)
2. **Given** the chat invocation, **When** the agent initiates structured inferences, **Then** the backend MUST capture and write the complete event block into the central project audit log with absolute compliance. (AC: 1.2.2)
3. **Given** direct network connections, **When** the human submits a message prompt, **Then** the agent's interactive response MUST stream and trigger rendering UI actions with a baseline latency under **2.0 seconds** (p95 interaction SLA). (AC: 1.2.3)

---

## Tasks / Subtasks

- [x] **Task 1: Build the Chat Pane Interface & Input Fields (AC: 1.2.1, 1.2.3)**
  - [x] Create `ui/lib/screens/workspace/chat_pane.dart` to implement the primary orchestration panel, utilizing Riverpod state listeners.
  - [x] Wire dynamic text inputs, scrolling list views, and streaming text blocks.
- [x] **Task 2: Implement Backend SSE Activity Broadcasts & Observability (AC: 1.2.1, 1.2.2)**
  - [x] Implement the central Go chat route endpoints in `api/chat.go` utilizing standard ADK SDK libraries.
  - [x] Integrate standard open-telemetry (OTel) metrics trackers to capture all inferences and log them to standard Google Cloud logs per-session.
- [x] **Task 3: Resolve Code Review Integrity Constraints (AC: 1.2.1 - 1.2.3)**
  - [x] Implement standard Dart `LineSplitter()` fragment buffers to safeguard parsing against TCP packet splits.
  - [x] Implement dynamic un-subscriptions on page dispose to eliminate worker memory leaks.

---

## Dev Notes

### 🏗️ 1. Architectural Guardrails Complied
*   **Server-Driven UI (SDUI):** Frontend chat consumes Server-Sent Events (SSE) dynamically via the unified `ApiClient` stream.
*   **Persona Avatars Integration:** Streamed bubbles are cleanly prefixed with the specialized agent icon (e.g. `📋` John, `🎨` Sally) and color tokens matching [ux-design-specification.md:L175-183](docs/spec/planning-artifacts/ux-design-specification.md#L175-183) definitions to preserve distinct corporate personas visually.

---

## 🛠️ Project Structure & Source Maps

### 1. Finalized Files Matrix

| File Path | Status | Operational Role |
| :--- | :--- | :--- |
| [ui/lib/screens/workspace/chat_pane.dart](ui/lib/screens/workspace/chat_pane.dart) | `COMPLETED` | Core conversational chat interface widget; Riverpod states, ListView streaming, and action buttons. |
| [api/chat.go](api/chat.go) | `COMPLETED` | Backend Go chat route endpoints; context mappings, GenKit pipelines, and OTel traces logging hooks. |

---

## Dev Agent Record

### Agent Model Used

Gemini Next

### Debug Log References

- Baseline chat endpoints verified successfully via integration tests.
- Reconnect backoff and dispose listeners verified cleanly under network drops tests.

### Completion Notes List

- ✅ Unified chat pane and SSE streams integrated.
- ✅ Structured OpenTelemetry compliance audit logs configured.
- ✅ Memory leakage dispose overrides verified cleanly.

### File List

- `ui/lib/screens/workspace/chat_pane.dart`
- `api/chat.go`
