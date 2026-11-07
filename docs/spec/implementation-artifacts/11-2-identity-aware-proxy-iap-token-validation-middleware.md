# Story 11.2: Identity-Aware Proxy (IAP) Token Validation Middleware

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a Security Engineer,  
I want the backend API to intercept and validate Google Cloud Identity-Aware Proxy (IAP) JWT assertions,  
so that all non-public API endpoints are protected and user identity claims are securely extracted into the request context.

## Acceptance Criteria

1. **Given** an unauthenticated request attempts to access a protected backend API endpoint,  
   **When** the token validation middleware intercepts the request,  
   **Then** the request MUST be rejected with an HTTP `401 Unauthorized` status if the `X-Goog-IAP-JWT-Assertion` header is missing, malformed, or has an invalid signature.
2. **Given** an authenticated request is made through Google Cloud IAP with a valid JWT assertion header,  
   **When** the validation middleware successfully verifies the signature and audience claims of the JWT,  
   **Then** the middleware MUST extract the user's email, subject (unique user ID), and organization/tenant identifier from the claims,  
   **And** inject the extracted user identity profile (`core.UserClaims`) securely into the request context for subsequent handlers to consume.
3. **Given** local development or testing environments,  
   **When** a custom bypass configuration (`BYPASS_IAP_VALIDATION=true`) is active,  
   **Then** the middleware MUST fall back to a mock validation mechanism (allowing mock headers or mock JWTs) without attempting to reach Google's public key endpoint, ensuring offline testability and local development velocity.

## Tasks / Subtasks

- [x] **Task 1: Extend Configuration for IAP (AC: 2, 3)**
  - [x] Add `IapAudience string` and `BypassIAPValidation bool` fields to `core.Configuration` in [configuration.go](api/core/configuration.go).
  - [x] Map these fields to environment variables `IAP_AUDIENCE` (e.g., `/projects/PROJECT_NUMBER/global/backendServices/SERVICE_ID`) and `BYPASS_IAP_VALIDATION` (boolean, defaults to `false` in production, `true` in local envs).
- [x] **Task 2: Introduce Testable Token Validation Interface (AC: 1, 2, 3)**
  - [x] In [auth.go](api/auth.go) (or a new `api/iap_validator.go`), define a `TokenValidator` interface to enable clean mocking:
    ```go
    type TokenValidator interface {
        Validate(ctx context.Context, token string, audience string) (*idtoken.Payload, error)
    }
    ```
  - [x] Implement `GoogleTokenValidator` which wraps the standard `google.golang.org/api/idtoken.Validate` function.
  - [x] Implement `MockTokenValidator` which allows injecting pre-configured payloads or errors for testing.
  - [x] Introduce a package-level or struct-injected instance of `TokenValidator`, defaulting to `GoogleTokenValidator` but overridable in tests.
- [x] **Task 3: Implement Rigorous IAP Token Validation in Middleware (AC: 1, 2, 3)**
  - [x] Update `resolveClaims` in [middleware.go](api/middleware.go):
    - [x] Retrieve the assertion from header `X-Goog-IAP-JWT-Assertion` (case-insensitive key).
    - [x] If `BypassIAPValidation` is `true` (local dev):
      - [x] If the header is missing, check if `x-goog-authenticated-user-email` is present as a fallback, and map it to a mock claim.
      - [x] If the header starts with `simulated-jwt-`, parse it using `MockAuthenticator` or a mock handler.
      - [x] Bypass real cryptographic validation and proceed.
    - [x] If `BypassIAPValidation` is `false` (production/strict mode):
      - [x] If the header is missing, return an explicit unauthorized error ("missing IAP assertion").
      - [x] Validate the token using the injected `TokenValidator.Validate(ctx, token, expectedAudience)`.
      - [x] If validation fails, return the signature/audience verification error.
      - [x] Extract claims from the payload:
        - `Subject` (from `payload.Subject`)
        - `Email` (from `payload.Claims["email"]`)
        - `OrganizationID` (from nested claim `firebase.tenant` within `payload.Claims["firebase"]`)
        - Inject these into a fresh `core.UserClaims` struct and assign it to the request context.
- [x] **Task 4: Secure Org Session JWT Validation and Identity Matching (AC: 2)**
  - [x] In `resolveClaims`, verify that if both a Bearer token (Org Session JWT) and an IAP header are present, their subjects/emails match exactly. If there is a mismatch, reject with a `401 Unauthorized` status (preventing token reuse/session hijacking).
- [x] **Task 5: Write Exhaustive Unit Tests (AC: 1, 2, 3)**
  - [x] Create/update tests in [middleware_test.go](api/middleware_test.go):
    - [x] Test missing `X-Goog-IAP-JWT-Assertion` header under strict mode -> expect `401 Unauthorized`.
    - [x] Test invalid/malformed JWT -> expect `401 Unauthorized`.
    - [x] Test wrong audience -> expect `401 Unauthorized`.
    - [x] Test successful validation -> expect `200 OK` and verify correct user, email, and organization/tenant ID are in the request context.
    - [x] Test bypass mode -> verify mock token/email works without network calls.

## Dev Notes

### Architecture & Security Constraints
- **Zero-Trust Boundary:** The backend MUST NOT rely on raw, unencrypted headers like `x-goog-authenticated-user-email` in production. These headers can be spoofed if an attacker bypasses the external load balancer. Cryptographic verification of `X-Goog-IAP-JWT-Assertion` is mandatory.
- **Strict Identity Propagation:** The extracted claims are wrapped in `core.UserClaims` and passed down. The subject must remain the source-of-truth identity throughout the request lifetime.
- **GCIP Tenant Extraction:** Multi-tenancy scopes users to tenants. In GCIP-integrated IAP, the tenant ID is nested inside the `firebase` claim under `tenant`. The middleware must safely navigate this map:
  ```go
  var tenantID string
  if firebaseClaim, ok := payload.Claims["firebase"].(map[string]interface{}); ok {
      tenantID, _ = firebaseClaim["tenant"].(string)
  }
  ```

### Source Tree Components to Touch
- [api/core/configuration.go](api/core/configuration.go) — Extend config for IAP audience and bypass settings.
- [api/auth.go](api/auth.go) — Define `TokenValidator` interface and implement real/mock validators.
- [api/middleware.go](api/middleware.go) — Update `resolveClaims` with strict validation logic.
- [api/middleware_test.go](api/middleware_test.go) — Write comprehensive unit tests utilizing mock validator.

### References
- [IAP Signed Headers How-To](https://cloud.google.com/iap/docs/signed-headers-howto)
- [Go idtoken.Validate Documentation](https://pkg.go.dev/google.golang.org/api/idtoken)
- [api/middleware.go](api/middleware.go#L17-L84)
- [api/auth.go](api/auth.go#L68-L164)

## Dev Agent Record

### Agent Model Used

Gemini Next

### Debug Log References

N/A

### Completion Notes List
- Implemented `TokenValidator` interface to support clean mocking of Google Cloud IAP tokens in testing.
- Added `IapAudience` and `BypassIAPValidation` to `core.Configuration`, allowing environment-driven configuration of IAP verification.
- Updated `resolveClaims` middleware to perform strict cryptographic validation of `X-Goog-IAP-JWT-Assertion` headers using standard Google verification libraries, while supporting a clean, secure mock-bypass mode for local development velocity.
- Integrated a secure identity matching validation that compares the subject/email of the Bearer Org Session JWT with the IAP assertions to prevent session hijacking and token reuse.
- Added exhaustive unit and integration tests covering missing headers, invalid signatures, wrong audiences, successful context extraction, bypass mode fallbacks, simulated tokens, and session hijacking prevention.

### File List
- [api/core/configuration.go](api/core/configuration.go)
- [api/core/configuration_test.go](api/core/configuration_test.go)
- [api/auth.go](api/auth.go)
- [api/auth_test.go](api/auth_test.go)
- [api/middleware.go](api/middleware.go)
- [api/middleware_test.go](api/middleware_test.go)

### Change Log
- **2026-06-19**: Initial implementation of IAP Token Validation Middleware, including configuration extension, interface design, middleware enforcement, hijacking protection, and exhaustive test coverage. (Status: ready-for-dev -> review)

### Review Findings

- [x] [Review][Patch] Fail-Open Bypass IAP Configuration Default [api/core/configuration.go:96]
- [x] [Review][Patch] Zero-Trust Boundary Violation via Session Cookie Fallback [api/middleware.go:186-194]
- [x] [Review][Patch] Subject Identity Inconsistency in Merged Claims [api/middleware.go:127-134]
- [x] [Review][Patch] Spoofable Cleartext Header Trust in Bypass Mode [api/middleware.go:58-70]
- [x] [Review][Dismissed] Standard ID Token Validation Used Instead of IAP Verification [api/auth.go:179] — false positive (Go idtoken package automatically routes to IAP certs for ES256 tokens)
- [x] [Review][Patch] Ensure PERF_TEST_MODE is not active in production [api/middleware.go:61, 91]
- [x] [Review][Patch] Insecure Fallback JWT Secret in Production [api/middleware.go:33]
- [x] [Review][Patch] Missing Startup Validation for IAP Audience [api/core/configuration.go:95-96]
- [x] [Review][Patch] Nil Payload Panic Risk in Middleware [api/middleware.go:82]
- [x] [Review][Patch] Global State Mutation in Tests [api/auth_test.go, api/middleware_test.go]
- [x] [Review][Patch] Weak Session Hijacking Validation on Empty Claims [api/middleware.go:117]
- [x] [Review][Patch] Mock Validator Ignores Audience [api/auth.go:188]
- [x] [Review][Patch] Error Variable Shadowing in Middleware [api/middleware.go]
- [x] [Review][Patch] Incomplete Test Coverage for Strict Mode [api/middleware_test.go]
