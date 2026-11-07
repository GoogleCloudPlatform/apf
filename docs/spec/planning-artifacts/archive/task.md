# Task List: Story 2.3 Backend Remediation & Course Correction

## 1. Specification Amendments (`bmad-sm` / `bmad-pm`)
- `[x]` Update `APF_Foundry_PRD.md` to mandate standard JWE compact serialization (`eyJ...`), key versioning headers (`v1:`), and struct-based dependency injection without package globals.
- `[x]` Update `architecture.md` under `Authentication & Security` to specify standard JWE compact serialization and explicit key versioning/rotation strategies.
- `[x]` Update `epics.md` Story 2.3 acceptance criteria to enforce key versioning headers, compliant JWE serialization, clean dependency injection, pre-shared dev keys, and Firestore transactions.
- `[x]` Update `2-3-user-level-settings-screen.md` to detail all 16 backend refactoring and patch items.

## 2. Backend Remediation (`bmad-dev` / Amelia)

### 2.1 Bad Spec Resolutions
- `[x]` **Bad Spec 1 (Dev Fallback Keys):** Refactor `init()` in `api/agent/agent_proxy.go` to use a pre-shared dev key pair or symmetric AES-GCM token exchange for dev/test fallback rather than discarding ephemeral private keys in memory.
- `[x]` **Bad Spec 2 (Clean Dependency Injection):** Refactor `api/chat.go`, `api/session.go`, and `api/secrets.go` to define struct receivers (`ChatHandler`, `SessionHandler`, `SecretsHandler`) encapsulating `db.UserSecretsStore`, completely removing `var ChatSecretsStore` and `var SessionSecretsStore` package globals.
- `[x]` **Bad Spec 3 (JWE Compliance):** Rewrite `EncryptSecretsForProxy` in `api/agent/agent_proxy.go` to construct standard JWE compact serialization strings (`eyJ...`) using a compliant JWE library or standard headers.
- `[x]` **Bad Spec 4 (Key Rotation & Recovery):** Update `Encrypt` and `Decrypt` in `api/db/secrets.go` to prepend/verify `v1:` key version headers, and update `GetUserSecrets` to support graceful error recovery allowing users to overwrite corrupted/un-decryptable keys without permanent lockout.

### 2.2 Code Defect Remediation (Patch Items)
- `[x]` **Patch 1 (Hardcoded Organization URLs):** Fix hardcoded `"user"` organization in `CreateSession` and `GetSession` (`api/agent/agent_proxy.go`).
- `[x]` **Patch 2 (Global proxyPublicKey State):** Remove package-level global `proxyPublicKey` state; encapsulate within `AgentProxy` struct instance.
- `[x]` **Patch 3 (Silenced Auth Errors):** Explicitly return and handle authentication propagation errors in `populateAuthAndSecrets`.
- `[x]` **Patch 4 (Nil targetMap Guard):** Add explicit guard against `targetMap == nil` in `populateAuthAndSecrets`.
- `[x]` **Patch 5 (Uncancellable Contexts):** Replace `context.WithoutCancel` with cancellable timeouts (`context.WithTimeout`) in `UserMessageHandler` (`api/chat.go`) and `connectSessionHandler` (`api/session.go`).
- `[x]` **Patch 6 (Restrictive Token Regex):** Expand `tokenRegex` in `api/secrets.go` to allow base64/JWT characters (`^[A-Za-z0-9_\.\-\+\/=]{10,2048}$`).
- `[x]` **Patch 7 (JSON Array Diagnostic Bodies):** Update `preflightCheckFunc` in `api/secrets.go` to support external APIs returning JSON array bodies upon successful authentication.
- `[x]` **Patch 8 (Auth Header Leakage):** Implement a custom HTTP client redirect policy in `preflightCheckFunc` to strip `Authorization` headers on cross-domain redirects.
- `[x]` **Patch 9 (Synchronous Preflight DoS):** Move synchronous 5s preflight HTTP calls out of the main HTTP handler thread (e.g., execute via goroutine/channel or background worker).
- `[x]` **Patch 10 (Brittle Masking Strings):** Decouple `isMasked` validation in `api/secrets.go` from exact 32-bullet string checks.
- `[x]` **Patch 11 (Race Conditions & Lost Updates):** Wrap `updateUserSecretsHandler` and `revokeUserSecretHandler` logic in Firestore transactions (`tx.Get`, `tx.Set`).
- `[x]` **Patch 12 (Blind Mock Implementation):** Update `mockUserSecretsStore` in `api/secrets_test.go` to simulate encryption/decryption transformations.

## 3. Verification
- `[x]` Run Go unit tests (`blaze test //api/...`).
- `[x]` Verify all 16 remediation items pass `bmad-code-review`.
