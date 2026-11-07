# Story 11.9: Multi-Tenant Identity Verification and Fail-Safe Security Context Propagation

Status: done

## Story

As a Security Engineer,  
I want to implement strict multi-tenant validation, fail-safe environment configuration audits on startup, and zero-trust identity context propagation in all downstream services,  
So that organization boundaries are cryptographically enforced and the application cannot run in a misconfigured, insecure state in production.

## Acceptance Criteria

1. **Given** the backend API receives a request to a protected, organization-scoped endpoint in production mode (`BYPASS_IAP_VALIDATION=false`)
   **When** the auth middleware resolves the user's claims
   **Then** the middleware MUST verify that a valid, non-empty `OrganizationID` is present in the extracted claims (mapped from `firebase.tenant`)
   **And** MUST reject the request with an HTTP `403 Forbidden` if the user's token is not scoped to a valid tenant, preventing unauthorized data-plane access.
2. **Given** the backend application starts up in a production environment
   **When** the configuration module is initialized
   **Then** it MUST perform a startup self-test to verify that `IAP_AUDIENCE` and `JWT_SECRET` are configured with valid, non-empty values
   **And** MUST panic and prevent the application from starting if they are missing or if `BYPASS_IAP_VALIDATION` is set to `true` in a production environment.
3. **Given** a backend developer implements a new endpoint or database operation
   **When** they retrieve the active user's identity or organization scope
   **Then** they MUST obtain this information exclusively from the Go `context` (via `core.UserClaimsKey`)
   **And** any direct references to cleartext headers like `x-goog-authenticated-user-email` or `X-Organization-Id` in downstream handlers or repositories MUST be prohibited (enforced via static analysis/code-review).
4. **Given** the backend runs in local development mode with `BYPASS_IAP_VALIDATION=true`
   **When** a request with simulated headers is processed
   **Then** the middleware MUST only trust mock/cleartext headers if the environment is explicitly configured as a non-production profile (e.g., `local` or `test`), otherwise rejecting the request to prevent production unauthorized access.

## Tasks / Subtasks

### Phase 1: Configuration Struct & Startup Self-Test

- [x] **Extend Configuration Struct** (AC: 2)
  - Add fields to the `Configuration` struct in [api/core/configuration.go](api/core/configuration.go):
    - `Environment string`: loaded from `ENVIRONMENT` environment variable, defaulting to `"production"`.
    - `JwtSecret string`: loaded from `JWT_SECRET` environment variable.
- [x] **Implement Fail-Safe Startup Validation** (AC: 2)
  - In `NewConfig()`, implement the following startup self-test rules:
    - Determine `isProduction := config.Environment != "local" && config.Environment != "test"`.
    - If `isProduction` is true:
      - Verify `config.IapAudience` is non-empty. If empty, panic: `"Configuration error: IAP_AUDIENCE must be set in production mode"`.
      - Verify `config.JwtSecret` is non-empty. If empty, panic: `"Configuration error: JWT_SECRET must be set in production mode"`.
      - Verify `config.BypassIAPValidation` is false. If true, panic: `"Configuration error: BYPASS_IAP_VALIDATION cannot be true in production mode"`.

### Phase 2: Authentication & Middleware Hardening

- [x] **Harden `resolveClaims` in `api/middleware.go`** (AC: 4)
  - Update `resolveClaims` to retrieve the JWT secret from `appConfig.JwtSecret` rather than direct `os.Getenv("JWT_SECRET")` calls.
  - In `resolveClaims`, if `bypassIAP` is true:
    - Check `appConfig.Environment`.
    - If it is not explicitly `"local"` or `"test"`, return a validation error: `"security violation: IAP bypass is only allowed in local or test environments"`. This prevents unauthorized access in production environments.
- [x] **Harden `orgScopeMiddleware` in `api/middleware.go`** (AC: 1)
  - Extract `bypassIAP := appConfig.BypassIAPValidation`.
  - In `orgScopeMiddleware`, if `!bypassIAP` (Strict Production Mode):
    - Verify that `claims.OrganizationID` is non-empty.
    - If it is empty (or missing), immediately reject the request with HTTP `403 Forbidden` (`"Forbidden: user token is not scoped to a valid tenant"`). This guarantees tenant isolation boundaries at the data plane.

### Phase 3: Downstream Context Enforcement & Guardrails

- [x] **Verify Context-Only Downstream Scope Extraction** (AC: 3)
  - Document and enforce that all API handlers and repository layers MUST retrieve the active user's identity and organization scope exclusively from the Go request `context` (via `core.UserClaimsKey` or `core.GetOrganizationID`).
  - Strictly prohibit direct header lookups for `X-Organization-Id`, `x-goog-authenticated-user-email`, or `x-goog-authenticated-user-id` in any downstream code (e.g. `api/projects.go`, `api/workspace.go`, `api/chat.go`), ensuring all context is properly validated and sanitised by the middleware.

### Phase 4: Testing & Integration

- [x] **Add Startup Self-Test Unit Tests** (AC: 2)
  - In [api/core/configuration_test.go](api/core/configuration_test.go), add tests verifying that `NewConfig()` panics under a production environment setup if `IAP_AUDIENCE` is empty, `JWT_SECRET` is empty, or `BYPASS_IAP_VALIDATION` is true.
- [x] **Add Middleware Security Integration Tests** (AC: 1, 4)
  - In [api/middleware_test.go](api/middleware_test.go), add tests verifying:
    - `orgScopeMiddleware` in production mode rejects requests with `403 Forbidden` if the user's claims lack a valid tenant scope (`OrganizationID`).
    - `resolveClaims` rejects request bypass attempts when the environment is configured as `"production"`.

### Review Findings

- [x] [Review][Patch] Environmental Divergence in Tenant Isolation & Local Dev Blocking [api/middleware.go:271-301]
- [x] [Review][Patch] Empty JWT Secret Handling in Dev/Test Modes [api/middleware.go:117-123]
- [x] [Review][Patch] Case-Sensitivity in Environment Detection [api/core/configuration.go:101-115]
- [x] [Review][Patch] Missing JWT Secret Strength Check in Production [api/core/configuration.go]
- [x] [Review][Defer] Redundant appConfig Nil-Pointer Checks [api/middleware.go] — deferred, pre-existing

## Dev Notes

### Fail-Secure Environment Profile Design
To prevent unathorized access in production environments, the platform adopts a **fail-secure** environment design.
- The `ENVIRONMENT` variable explicitly defines the active profile.
- Any environment profile that is not explicitly `"local"` or `"test"` is treated as a **production environment** by default.
- In production profiles, IAP validation bypasses are completely rejected at both application start time (via startup panic) and request validation time.

### Zero-Trust Context Propagation
Downstream handlers must treat the request headers as untrusted. The only source of truth for identity and tenant scoping is the Go `context`. The middleware validates the cryptographic signatures of incoming assertions, maps the claims, and injects them into the context. Downstream code merely consumes these pre-validated contexts.

## References

- [api/core/configuration.go](api/core/configuration.go)
- [api/middleware.go](api/middleware.go)
- [api/core/types.go](api/core/types.go)
- [docs/spec/planning-artifacts/epics.md](docs/spec/planning-artifacts/epics.md#L995-L1018)

## Dev Agent Record

### Agent Model Used

Gemini Next

### Debug Log References

- Workspace test suite successfully executed via: `ENVIRONMENT=test go test ./ai/... ./api/... ./runners/...` (All tests passed, zero regressions).

### Completion Notes List

- **Fail-Safe Startup Audits:** Extended `Configuration` struct with `Environment` and `JwtSecret` fields. Implemented startup self-test in `NewConfig()` that immediately panics if the application is started in a production-profile environment with missing `IAP_AUDIENCE`, missing `JWT_SECRET`, or with `BYPASS_IAP_VALIDATION` set to true.
- **Middleware Hardening:** Hardened `resolveClaims` to strictly reject IAP bypass attempts if the active environment is not explicitly `"local"` or `"test"`, preventing unauthorized access. Updated the JWT token verification to dynamically retrieve the secret from `appConfig.JwtSecret`.
- **Tenant Isolation Enforcement:** Hardened `orgScopeMiddleware` to immediately reject requests with HTTP `403 Forbidden` if the user's OIDC claims lack a valid, non-empty `OrganizationID` when running in strict production mode, guaranteeing absolute tenant isolation at the data-plane.
- **Downstream Context Verification:** Executed exhaustive codebase searches verifying that no downstream handlers or repositories perform direct header lookups for identity or organization scoping, ensuring complete reliance on secure Go request contexts.
- **Architectural Documentation:** Documented the Zero-Trust context propagation and fail-secure profile design in the core project architecture specification (`docs/spec/planning-artifacts/architecture.md`).
- **Robust Security Testing:** Updated existing tests to align with the new security model. Added comprehensive unit and integration tests to verify all startup panic conditions, bypass rejections, and tenant isolation failures, with all tests passing 100%.

### File List

- [api/core/configuration.go](api/core/configuration.go) (Modified)
- [api/core/configuration_test.go](api/core/configuration_test.go) (Modified)
- [api/middleware.go](api/middleware.go) (Modified)
- [api/middleware_test.go](api/middleware_test.go) (Modified)
- [api/routes_test.go](api/routes_test.go) (Modified)
- [docs/spec/planning-artifacts/architecture.md](docs/spec/planning-artifacts/architecture.md) (Modified)
- [docs/spec/implementation-artifacts/sprint-status.yaml](docs/spec/implementation-artifacts/sprint-status.yaml) (Modified)
- [docs/spec/implementation-artifacts/11-9-multi-tenant-identity-verification-and-fail-safe-security-context-propagation.md](docs/spec/implementation-artifacts/11-9-multi-tenant-identity-verification-and-fail-safe-security-context-propagation.md) (Modified)
