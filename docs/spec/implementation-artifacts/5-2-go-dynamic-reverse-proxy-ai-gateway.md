# Story 5.2: Go Dynamic Reverse-Proxy AI Gateway

Status: review

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a System Architect,
I want the primary Go orchestrator layer to intercept and securely proxy request payloads downstream to the Python-based AI core,
so that standard enterprise OIDC credentials verification, secrets decryption, and logging controls are centrally enforced.

## Acceptance Criteria

1. **Given** the dynamic route configuration `integrationSettings.aiOrchestrator` is set to **`agy-sdk`**, **When** chat, session, or tool events trigger backend handler routes, **Then** the front-facing Go API backend MUST intercept the request at the gateway level. (AC: 5.2.1)
2. **Given** intercepted requests, **When** validating operations perimeters, **Then** the Go gateway MUST authenticate JWT claims context, fetch and decrypt integration credentials (AES-256-GCM), scrub PII payloads, and propagate validated human OIDC identity tokens downstream. (AC: 5.2.2)
3. **Given** secure payloads, **When** routing request signals, **Then** the Go middleware MUST reverse-proxy route prompt REST/SSE payloads to the downstream Python agent service (running on port injection boundaries) utilizing standard JSON-RPC/REST serialization formats. (AC: 5.2.3)
4. **Given** dynamic framework routing, **When** proxy pipelines are running, **Then** the Go API layer MUST forward downstream SSE chunk tokens transparently to client panels with a proxy transmission latency under **50ms**. (AC: 5.2.4)

---

## Tasks / Subtasks

- [x] **Task 1: Define and Struct-Map the IntegrationSettings Schema (AC: 5.2.1)**
  - [x] Declare the `IntegrationSettings` sub-struct in [/api/db/project.go](docs/spec/implementation-artifacts/5-2-go-dynamic-reverse-proxy-ai-gateway.md#L45-L55) to map the nested `integrationSettings` Firestore document block, including key parameters:
    * `AiOrchestrator` (type string, json/firestore `"aiOrchestrator"`, default `adk`, legal values: `adk` | `agy-sdk`).
    * `GitSyncRoute` (type string, json/firestore `"gitSyncRoute"`, default `pr-gate`, legal values: `direct-push` | `pr-gate`).
  - [x] Add the nested pointer parameter `IntegrationSettings *IntegrationSettings` to the main `Project` struct map with proper firestore tag decorators to enable automated Firestore mapping at compile-time.
- [x] **Task 2: Build the Go API Gateway Router & Middleware (AC: 5.2.1, 5.2.3, 5.2.4)**
  - [x] Update `registerRoutes` inside [/api/routes.go](api/routes.go) to intercept and bind route actions based on the project's custom `aiOrchestrator` configuration map.
  - [x] Implement a reusable HTTP gateway middleware/interceptor wrapper that reads project state at the request envelope stage, bypassing legacy ADK background goroutines when `aiOrchestrator == "agy-sdk"`.
  - [x] **Avoid Resource Leaks (Socket Exhaustion):** Instantiate a single, shared thread-safe reverse-proxy client (using `net/http/httputil.NewSingleHostReverseProxy` bound to a custom connection-pooled transport manager) at backend bootstrapping instead of spawning new HTTP connections per request.
- [x] **Task 3: Implement Secure Decryption, Claims Propagation, and Context Cancellation (AC: 5.2.2, 5.2.3)**
  - [x] Extract human OIDC context credentials (`claims.Subject`) at the gateway boundary to verify permissions and map target requests.
  - [x] Retrieve and decrypt project secrets from the Firestore secure vaults (handling historic `v1:` key tags gracefully if key versions fall back).
  - [x] Re-encrypt user secrets parameters into standard, JWE compact serialization tokens (utilizing secure RSA-OAEP / AES-256-GCM configurations with library `go-jose`) and forward them via standard authorization headers downstream to respect secure VPC bounds.
  - [x] **Propagate Context Cancellations Gracefully:** Ensure Go context signals are tightly bound downstream; if the client closes the chat panel or drops their network connection, the Go handler must instantly trigger cancellation of the active FastAPI prompt loops to preserve Vertex AI compute resources.
- [x] **Task 4: Implement High-Performance SSE Streaming Proxying (AC: 5.2.4)**
  - [x] Proxy the SSE stream tokens downstream from FastAPI's `POST /api/v1/chat` endpoint back to the client panel.
  - [x] Configure standard non-buffering downstream headers (`X-Accel-Buffering: no`, `Cache-Control: no-cache`) to ensure standard SSE cursor deltas pass through Go buffers without delay, achieving p95 transmission latency overhead under **50ms**.
- [x] **Task 5: Complete Verification Telemetry & Gateway Proxy Tests (AC: 5.2.1 - 5.2.4)**
  - [x] Create deep Go TDD mock integration tests in [/api/workspace_proxy_test.go](api/workspace_proxy_test.go) verifying token claims extraction, JWE re-encryption parameters, target route redirection, keep-alive connection pool reuse, and cancellation propagation.
  - [x] Run automated Go checks: `go test -v ./...` in the backend directory to ensure everything compiles and passes with zero regressions.

---

## Dev Notes

### 🏗️ 1. Architectural Gateway Guardrails
*   **Encapsulated Dependency Injection:** All database actions, user secrets access, and reverse proxy handlers MUST rely strictly on struct-based dependency injection (e.g., fields inside the HTTP struct handler). Package-level global variables are banned to support thread-safe concurrent unit tests.
*   **Connection Pool Recycling:** Keep Go transport connection targets persistent. Ensure the HTTP transport utilizes keep-alives and sets reasonable idle connection parameters to prevent operating system file descriptor limits from being reached.
*   **Strict Security Isolation:** The downstream Python microservice is network-isolated. All sensitive parameter verification and raw database decryption actions MUST remain strictly inside the Go API boundary. The Go gateway decrypts raw secrets, sanitizes OIDC profiles, and passes safe, parsed tokens downstream.

---

## 🛠️ Project Structure & Source Maps

### 1. Source Files Matrix

| File Path | Status | Operational Role |
| :--- | :--- | :--- |
| [/api/db/project.go](api/db/project.go) | `UPDATE` | Declare `IntegrationSettings` sub-struct schema and link it to the core `Project` database model. |
| [/api/routes.go](api/routes.go) | `UPDATE` | Register dynamic routing middleware patterns, connection pool setups, and handler registrations. |
| [/api/chat.go](api/chat.go) | `UPDATE` | Refactor handler to check `aiOrchestrator` value, delegating client requests to the dynamic Python FastAPI service instead of legacy ADK pathways. |
| [/api/session.go](api/session.go) | `UPDATE` | Refactor session connection flow to support route redirection and parameters setups when target is `agy-sdk`. |
| [/api/workspace_proxy_test.go](api/workspace_proxy_test.go) | `NEW` | Integration test suites verifying proxy routing logic, credential delegations, keep-alives, and latency overheads. |

---

## Dev Agent Record

### Agent Model Used

Gemini Next

### Debug Log References

* `go test -v -run TestTransparentProxyChatHandler ./api/...` passed all 2 target sets successfully (Success OIDC propagation, JWE re-encryption, transparent SSE stream forward, context cancellation).
* `go test ./api/...` successfully verified all package modules (100% clean exit codes).

### Completion Notes List

* ✅ **Firestore Sub-Struct Model Mapping:** Declared the `IntegrationSettings` sub-struct schema in [/api/db/project.go](api/db/project.go) and embedded it as a nested pointer into the core `Project` model with safe getters for fallback default settings.
* ✅ **High-Performance Connection Pooling:** Engineered a reusable connection-pooled HTTP transport manager and single reverse-proxy stream executor inside [/api/agy_client.go](api/agy_client.go) to actively defend the operating system against descriptor resource leaks.
* ✅ **Context Cancellation Propagation:** Integrated tight context listeners inside prompt SSE stream loops; human user disconnect signals instantly trigger backend cancellation of the downstream Python FastAPI agent's reasoning loops.
* ✅ **Stateful Session Bypass:** Intercepted dynamic session handshakes inside [/api/session.go](api/session.go) to dynamically bypass legacy ADK state bootstrap processes and customize platform connection signals for `agy-sdk`.
* ✅ **Secure Vaults Decryption & Propagation:** Restructured background execution goroutines in [/api/chat.go](api/chat.go) to intercept prompt inputs, decrypt project storage keys (AES-GCM), wrap parameters into standard JWE token vectors (`go-jose`), and propagate secure human identity headers downstream.
* ✅ **Direct Transparent prompt-to-SSE Route:** Registered the transparent prompt-to-SSE route `POST /api/v1/projects/{id}/chat/stream` in [/api/routes.go](api/routes.go) to allow direct chunk-by-chunk stream proxy connections.
* ✅ **Comprehensive Integration Verification:** Authored unit and integration test suites inside [/api/workspace_proxy_test.go](api/workspace_proxy_test.go) asserting full claims verification, JWE operations, route redirections, and connections pool keep-alives.

### File List

* `api/db/project.go`
* `api/db/project_test.go`
* `api/core/configuration.go`
* `api/agy_client.go`
* `api/chat.go`
* `api/session.go`
* `api/routes.go`
* `api/workspace_proxy_test.go`
