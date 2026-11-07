# Story 11.8: Parent Project Super Admin Authentication & Global Resource Authorization

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a Super Administrator,  
I want to log in via a dedicated portal at the Parent Project level and gain global resource management authorization,  
So that I can securely administer all tenant organizations without being bound to any single customer tenant silo.

## Acceptance Criteria

1. **Given** a Super Administrator accesses the dedicated administrative portal (e.g., `admin.myapp.com`)  
   **When** they log in  
   **Then** the frontend login SDK is initialized without setting a `tenantId` (authenticating them against the root Parent Project)  
   **And** they are successfully authenticated at the parent project level.
2. **Given** the Cloud Run backend receives an API request from the administrative portal with the Super Admin's JWT token  
   **When** the backend verifies the ID token (`admin.auth().verifyIdToken(token)`)  
   **Then** the backend checks that the token **does not** contain a `firebase.tenant` claim  
   **And** verifies that the custom claim `is_super_admin === true` is present  
   **And** grants the request access to global administrative operations (e.g., provisioning organizations, viewing all billing, etc.).

## Tasks / Subtasks

- [x] **Task 1: Update Claims Model to Support Super Admin Organization Bypass (AC: 2)**
  - [x] In [types.go](api/core/types.go), update the `HasAccessToOrg` method on `UserClaims` to return `true` immediately if `IsSuperAdmin` is `true`.
- [x] **Task 2: Enable Organization Admin Authorization Bypass (AC: 2)**
  - [x] In [middleware.go](api/middleware.go), update `orgAdminAccessMiddleware` to check if `claims.IsSuperAdmin` is `true`. If so, skip the database lookup in `userStore.GetUser`, inject the target `orgId` into the request context using `core.OrganizationIDKey`, and proceed immediately.
- [x] **Task 3: Enable Project Level Access Bypass (AC: 2)**
  - [x] In [middleware.go](api/middleware.go), update `projectAccessMiddleware` to check if `claims.IsSuperAdmin` is `true`. If so, bypass the project ownership/membership database checks and proceed immediately.
- [x] **Task 4: Add Comprehensive Backend Unit and Integration Tests (AC: 2)**
  - [x] In [middleware_test.go](api/middleware_test.go), add a test verifying that `orgScopeMiddleware` successfully permits and propagates organization scope for a Super Admin (since `HasAccessToOrg` returns true).
  - [x] In [admin_handlers_test.go](api/admin_handlers_test.go), add a test in `TestAdminMiddleware_Gating` verifying that a Super Admin (with no record in the organization database) is successfully granted organization admin privileges by `orgAdminAccessMiddleware`.
  - [x] In [projects_test.go](api/projects_test.go), add a test verifying that `projectAccessMiddleware` successfully permits project access to a Super Admin even if they are not listed as an owner or member in the project document.

## Dev Notes

### Architecture & Security Constraints

- **Root Parent Project Authentication:** A Super Admin has a JWT token minted against the root Firebase project itself, meaning it has no `firebase.tenant` claim. In production, Google Cloud IAP or Firebase Auth SDK validates this against the root project certificates.
- **IsSuperAdmin Claim Propagation:** The backend authentication middleware (`resolveClaims` in [middleware.go](api/middleware.go)) already extracts `is_super_admin` from the validated IAP token payload when `tenantID` is empty. It maps it to `core.UserClaims.IsSuperAdmin`.
- **Zero-Trust Tenant Boundaries vs. Super Admin Power:** While organization isolation is strictly enforced for normal tenant users, Super Administrators must be dynamically elevated to access and administer any organization. Bypassing database membership checks in middlewares specifically for `IsSuperAdmin == true` is the clean, secure, and designated way to implement this without corrupting tenant databases with dummy admin users.

### Source Tree Components to Touch

- [api/core/types.go](api/core/types.go) — Allow global access bypass in `HasAccessToOrg`.
- [api/middleware.go](api/middleware.go) — Bypass database membership checks for Super Admins in `orgAdminAccessMiddleware` and `projectAccessMiddleware`.
- [api/middleware_test.go](api/middleware_test.go) — Verify `orgScopeMiddleware` behavior for Super Admins.
- [api/admin_handlers_test.go](api/admin_handlers_test.go) — Verify `orgAdminAccessMiddleware` bypass logic.
- [api/projects_test.go](api/projects_test.go) — Verify `projectAccessMiddleware` bypass logic.

### Testing Standards Summary

- Mock authentication using `MockTokenValidator` and `MockAuthenticator` where necessary.
- Inject Super Admin claims (e.g. `Subject: "super-admin-uid", IsSuperAdmin: true`) into the test context and assert that handlers return `200 OK` rather than `403 Forbidden` or `401 Unauthorized`.
- Ensure all tests run cleanly via `go test ./api/...`.

### Project Structure Notes

- Frontend routing gating is already implemented in [RouteGuard.tsx](ui/src/components/shared/RouteGuard.tsx#L71-L76). It detects `isSuperAdmin` and redirects them to the global administration screens (e.g. `/admin/provision`) instead of forcing organization selection. No frontend changes are needed.

### References

- [Source: docs/spec/planning-artifacts/epics.md#Line856-Line1086](docs/spec/planning-artifacts/epics.md#L856-L1086) - Identity and Access Management Epic requirements and breakdown.
- [Source: api/middleware.go](api/middleware.go) - Authentication and Authorization Middleware.
- [Source: ui/src/components/shared/RouteGuard.tsx](ui/src/components/shared/RouteGuard.tsx) - Frontend routing gating.

## Dev Agent Record

### Agent Model Used

Gemini Next

### Debug Log References

- Verified all changes against the full API test suite via `go test ./api/...` which completed successfully.

### Completion Notes List

- Updated the claims model `core.UserClaims.HasAccessToOrg` to immediately authorize Super Admins (`IsSuperAdmin = true`) for any requested organization.
- Updated `orgAdminAccessMiddleware` to bypass organization user database lookups for Super Admins, immediately injecting organization context and permitting access.
- Updated `projectAccessMiddleware` to bypass project membership/ownership database checks for Super Admins, immediately permitting project access.
- Added comprehensive unit/integration test cases to `api/middleware_test.go`, `api/admin_handlers_test.go`, and `api/projects_test.go` to validate Super Admin bypass functionality across all middlewares.

### File List

- `api/core/types.go`
- `api/middleware.go`
- `api/middleware_test.go`
- `api/admin_handlers_test.go`
- `api/projects_test.go`

### Review Findings

- [x] [Review][Patch] Super Admin Status Lost on Claim Merging [api/middleware.go:635]
- [x] [Review][Patch] Refine projectAccessMiddleware to Bypassing Authorization only, not Existence [api/middleware.go:293]
- [x] [Review][Patch] Security Vulnerability: Internal JWT Error Leakage [api/middleware.go:684]
- [x] [Review][Patch] Inconsistent Local Dev Simulation: Super Admin Role & Project Creation in UI [ui/src/stores/auth.ts:39]
- [x] [Review][Patch] Session Hijacking Bypass due to Empty Email Match in Dev Mode [api/middleware.go:660]
- [x] [Review][Patch] Trailing Whitespace in types.go [api/core/types.go:62]
- [x] [Review][Defer] Pre-existing: Misleading "Super Admin" Simulation in Admin Handlers Test [api/admin_handlers_test.go:470] — deferred, pre-existing
- [x] [Review][Defer] Pre-existing: Code Cleanliness and decoupling issues in Admin Handlers Tests and Middleware [api/admin_handlers_test.go:79] — deferred, pre-existing

