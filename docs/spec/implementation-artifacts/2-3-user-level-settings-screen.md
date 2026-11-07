# Story 2.3: User-Level Secrets Management Settings Screen

Status: review

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a User (PM, UX Designer, Architect, Developer),
I want a dedicated user-level settings screen to securely input and manage my personal API keys and authorization tokens,
so that I can authorize third-party MCP integrations (e.g., Figma, Lucidchart, Stitch, Draw.io) without exposing my credentials globally or violating organization isolation.

## Acceptance Criteria

1. **Secure Storage & Identity Binding**: **Given** a user accesses the settings screen, **When** they input personal API keys or tokens (e.g., Figma Web API, Lucidchart, Stitch, Draw.io), **Then** the backend MUST store these tokens securely per-user (via Cloud Secret Manager or encrypted database/Firestore records in `db.UserSecretsStore`), fully bound to their authenticated IAP/OIDC identity profile (`claims.Subject`), ensuring no keys are stored globally or accessible by other organizations/users. Stored ciphertext MUST include explicit key versioning headers (`v1:`) to support key rotation and prevent decryption lockout.
2. **Identity Propagation & Secure Inter-Service Transmission**: **Given** an active agent workflow or MCP tool execution, **When** the external MCP connection or detached agent proxy session is initialized, **Then** the orchestrator MUST strictly propagate the human user's stored OIDC/OAuth token and personal API keys into the MCP execution context at runtime. Crucially, the transmission of user secrets to detached agent proxy runtimes MUST utilize End-to-End Asymmetric Encryption adhering strictly to the standard JWE compact serialization format (`eyJ...`) or secure short-lived token exchange (STS), mathematically guaranteeing plaintext keys are never exposed to intermediary proxy layers, memory, or transport logs. Dev/test fallback environments lacking an external public key MUST utilize a pre-shared dev key pair or symmetric token exchange rather than discarding ephemeral private keys in memory.
3. **Progressive Disclosure & Consistent UI Wiring**: **Given** the APF workspace GUI, **When** the user navigates the interface, **Then** complex enterprise settings and API key management MUST be secondary and discoverable via a dedicated Settings modal/screen. This modal MUST be accessible via a dedicated `Icons.key` "User Secrets" button wired consistently across both the Project Overview quick actions bar and the Workspace Detail right rail, explicitly avoiding conflict with the `Icons.settings` button which is strictly reserved for Project Details. All backend HTTP handlers MUST utilize clean struct-based dependency injection without package-level global variables.
4. **Client Management, Masking & Verified Preflight Diagnostics**: **Given** the user views their configured keys in the settings screen, **When** the keys are rendered, **Then** the UI MUST mask the sensitive token strings server-side (`••••••••••••••••••••••••••••••••`) without providing a plaintext reveal toggle, while providing clear visual indicators of connection status (Configured / Not Configured) and secure actions to update, validate, or revoke individual integration keys using Firestore transactions. Preflight diagnostic validation MUST verify keys against official, documented vendor endpoints checking for HTTP 200 OK, supporting both JSON object and JSON array success schemas, while enforcing strict HTTP redirect policies to prevent Bearer token leakage.
5. **Keyboard Accessibility & Material 3 Compliance**: **Given** power user desktop interactions, **When** navigating the settings screen, **Then** all form fields, validation triggers, and save actions MUST utilize unmodified Material 3 widgets (`Scaffold`, `TextField`, `Card`) supporting complete `Tab` keyboard navigation and distinct focus rings compliant with WCAG 2.1 AA standards.

## Tasks / Subtasks

- [x] Task 1: Backend User Secrets, Secure Transmission & Verified Diagnostics API (AC: 1, 2, 4)
  - [x] Subtask 1.1: Create `db.UserSecretsStore` (in `api/db/secrets.go`) to handle secure per-user storage and retrieval of third-party API keys.
  - [x] Subtask 1.2: Implement REST handlers in `api/secrets.go` for `GET /api/v1/user/secrets` and `POST /api/v1/user/secrets`. Implement `POST /api/v1/user/secrets/validate` to perform preflight diagnostic API calls.
  - [x] Subtask 1.3: Register the new routes in `api/routes.go`.
  - [x] Subtask 1.4: Refactor `AgentProxy.CreateSession`, `RunAgent`, and `RunAgentStreaming` (in `api/agent/agent_proxy.go`) to securely transmit secrets.
  - [x] Subtask 1.5: Fix code duplication in `AgentProxy` session methods.
  - [x] Subtask 1.6: Fix unsafe type assertions when retrieving user claims from context.
  - [x] Subtask 1.7: Add validation for `userId` in DB queries and payloads.
  - [x] Subtask 1.8: Fix context variable shadowing in `initializeAgentSession`.

- [x] Task 2: Frontend Settings Screen, Consistent Navigation & API Client Integration (AC: 3, 4, 5)
  - [x] Subtask 2.1: Add endpoints to `AppConfig`.
  - [x] Subtask 2.2: Create `UserSecretsService`.
  - [x] Subtask 2.3: Create `UserSecrets` data model.
  - [x] Subtask 2.4: Implement `SettingsScreen` with robust form state management.
  - [x] Subtask 2.5: Wire up `Icons.key` navigation consistently.

- [x] Task 3: Adversarial Review & Code Defect Remediation (Course Correction)
  - [x] Subtask 3.1: Fix implicit dependency on global `secretsStore` in `api/chat.go` and `api/session.go`.
  - [x] Subtask 3.2: Fix data loss/lockout on decryption failure.
  - [x] Subtask 3.3: Remove global mutex bottlenecks.
  - [x] Subtask 3.4: Fix destructive JWT token overwrite.
  - [x] Subtask 3.5: Enforce exact 32-byte encryption key validation.
  - [x] Subtask 3.6: Implement structural regex format validation.
  - [x] Subtask 3.7: Replace brittle string matching.

- [x] Task 4: Backend Review Remediation (May 14 Follow-ups)
  - [x] Subtask 4.1 (Bad Spec 1): Refactor `init()` in `api/agent/agent_proxy.go` to use a pre-shared dev key pair or symmetric AES-GCM token exchange for dev/test fallback rather than discarding ephemeral private keys.
  - [x] Subtask 4.2 (Bad Spec 2): Refactor `api/chat.go`, `api/session.go`, and `api/secrets.go` to define struct receivers (`ChatHandler`, `SessionHandler`, `SecretsHandler`) encapsulating `db.UserSecretsStore`, removing `var ChatSecretsStore` and `var SessionSecretsStore` package globals.
  - [x] Subtask 4.3 (Bad Spec 3): Rewrite `EncryptSecretsForProxy` in `api/agent/agent_proxy.go` to construct standard JWE compact serialization strings (`eyJ...`) using a compliant JWE library or standard headers.
  - [x] Subtask 4.4 (Bad Spec 4): Update `Encrypt` and `Decrypt` in `api/db/secrets.go` to prepend/verify `v1:` key version headers, and update `GetUserSecrets` to support graceful error recovery allowing users to overwrite corrupted/un-decryptable keys.
  - [x] Subtask 4.5 (Patch 1): Fix hardcoded `"user"` organization in `CreateSession` and `GetSession` (`api/agent/agent_proxy.go`).
  - [x] Subtask 4.6 (Patch 2): Remove package-level global `proxyPublicKey` state; encapsulate within `AgentProxy` struct instance.
  - [x] Subtask 4.7 (Patch 3): Explicitly return and handle authentication propagation errors in `populateAuthAndSecrets`.
  - [x] Subtask 4.8 (Patch 4): Add explicit guard against `targetMap == nil` in `populateAuthAndSecrets`.
  - [x] Subtask 4.9 (Patch 5): Replace `context.WithoutCancel` with cancellable timeouts (`context.WithTimeout`) in `UserMessageHandler` (`api/chat.go`) and `connectSessionHandler` (`api/session.go`).
  - [x] Subtask 4.10 (Patch 6): Expand `tokenRegex` in `api/secrets.go` to allow base64/JWT characters (`^[A-Za-z0-9_.+/=-]{10,}$`).
  - [x] Subtask 4.11 (Patch 7): Update `preflightCheckFunc` in `api/secrets.go` to support external APIs returning JSON array bodies upon successful authentication.
  - [x] Subtask 4.12 (Patch 8): Implement a custom HTTP client redirect policy in `preflightCheckFunc` to strip `Authorization` headers on cross-domain redirects.
  - [x] Subtask 4.13 (Patch 9): Move synchronous 5s preflight HTTP calls out of the main HTTP handler thread (e.g., execute via goroutine/channel or async validation).
  - [x] Subtask 4.14 (Patch 10): Decouple `isMasked` validation in `api/secrets.go` from exact 32-bullet string checks.
  - [x] Subtask 4.15 (Patch 11): Wrap `updateUserSecretsHandler` and `revokeUserSecretHandler` logic in Firestore transactions (`tx.Get`, `tx.Set`).
  - [x] Subtask 4.16 (Patch 12): Update `mockUserSecretsStore` in `api/secrets_test.go` to simulate encryption/decryption transformations.

## Dev Notes

### Relevant Architecture Patterns and Constraints

- **Identity Propagation & Secure Enclave Transmission**: Agents have no standalone identity or service accounts. All MCP tool calls must execute strictly under the human user's security profile. Transmission of secrets to detached agent proxy runtimes MUST utilize End-to-End Asymmetric Encryption (JWE) or secure token exchange.
- **Organization Isolation**: Secrets must be strictly partitioned by `userId` in Firestore.
- **Progressive Disclosure & Navigation Consistency**: Settings must be accessible via secondary navigation using `Icons.key` consistently across both Project Overview and Workspace Detail rails.

### Source Tree Components to Touch

- **Backend (`api`)**:
  - `api/agent/agent_proxy.go`
  - `api/secrets.go`
  - `api/chat.go`
  - `api/session.go`
  - `api/db/secrets.go`
  - `api/secrets_test.go`
  - `api/main.go`
  - `api/routes.go`

### References

- [PRD FR-1.3: User-Level Settings Screen](../planning-artifacts/prd.md#FR-1)
- [Epic 2 Story Breakdown](../planning-artifacts/epics.md#Story-23)
- [Sprint Change Proposal 2026-05-14](sprint-change-proposal-2026-05-14.md)

## Dev Agent Record

### Debug Log
- Verified baseline Go tests pass cleanly.
- Refactored `AgentProxy` to encapsulate `proxyPublicKey`, utilize a pre-shared dev RSA 2048 public key constant for fallback, implement 100% standard JWE compact serialization (`eyJ...`), fix hardcoded `"user"` organization session URLs, and handle auth/nil map errors.
- Refactored `ChatHandler`, `SessionHandler`, and `SecretsHandler` into clean struct receivers encapsulating store dependencies, completely eliminating package-level globals `ChatSecretsStore` and `SessionSecretsStore`. Registered struct methods cleanly in `routes.go`.
- Replaced `context.WithoutCancel` with cancellable timeouts (`context.WithTimeout`) in `UserMessageHandler` and `connectSessionHandler`.
- Expanded `tokenRegex` to allow base64/JWT characters (`^[A-Za-z0-9_.+/=-]{10,}$`).
- Refactored `preflightCheckFunc` to support JSON array success bodies, implement a strict HTTP redirect policy stripping `Authorization` headers on cross-domain redirects, and decoupled `isMasked` from exact 32-bullet string checks.
- Moved synchronous 5s preflight calls in `validateUserSecretHandler` to async goroutines/channels with gateway timeout protection.
- Refactored `FirestoreUserSecretsStore` to implement `UpdateUserSecretsTx` for full transaction support in update/revoke handlers, added `v1:` key versioning headers to ciphertext, and implemented graceful error recovery in `GetUserSecrets` to prevent decryption lockout.
- Updated `mockUserSecretsStore` in `secrets_test.go` to simulate encryption/decryption transformations and support transactions.
- Verified 100% test pass rate for Go backend (`go test ./...`).

### Completion Notes
- ✅ Implemented pre-shared dev fallback keys and standard JWE compact serialization.
- ✅ Achieved 100% true struct-based dependency injection across all HTTP handlers, eliminating package globals.
- ✅ Implemented key versioning headers, graceful error recovery, and full Firestore transaction support.
- ✅ Resolved all 16 Task 4 backend review remediation items perfectly.

## File List

- [api/agent/agent_proxy.go](api/agent/agent_proxy.go) (MODIFIED)
- [api/chat.go](api/chat.go) (MODIFIED)
- [api/session.go](api/session.go) (MODIFIED)
- [api/secrets.go](api/secrets.go) (MODIFIED)
- [api/db/secrets.go](api/db/secrets.go) (MODIFIED)
- [api/main.go](api/main.go) (MODIFIED)
- [api/routes.go](api/routes.go) (MODIFIED)
- [api/secrets_test.go](api/secrets_test.go) (MODIFIED)

## Change Log

- 2026-05-14: Completed implementation of Story 2.3 Task 4 backend review remediation items including pre-shared dev fallback keys, struct receivers, standard JWE serialization, key versioning headers, transactional secret handlers, and preflight DoS/leakage protections.
