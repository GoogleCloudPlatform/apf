# Story 10.2: Organization-Aware Non-Administrative (Data Plane) APIs

Status: done

## Story

As an Enterprise Administrator,
I want all non-administrative (data plane) APIs in the middleware to be organization-aware,
So that cross-organization data access attempts are rejected before reaching orchestration enclaves.

## Acceptance Criteria

1. **Given** an incoming request to a non-administrative (data plane) API endpoint
   **When** the request reaches the middleware
   **Then** the middleware MUST extract and validate the `organizationId` scope from the request context
   **And** return a `403 Forbidden` error if the context does not match the requested workspace organization scope

## Tasks / Subtasks

- [x] Implement `orgScopeMiddleware` in `api/middleware.go` (AC: 1)
  - [x] Extract `X-Organization-Id` header from the request
  - [x] Retrieve `UserClaims` from the request context (injected by `authMiddleware`)
  - [x] Validate that the requested organization ID (from header) is valid for the user:
    - [x] If `claims.OrganizationID` is set (org-scoped token), it must match the header
    - [x] If `claims.OrganizationID` is not set (primary token), the header must be present in `claims.Organizations`
  - [x] Inject the validated organization ID into the request context using `core.OrganizationIDKey`
  - [x] Return `403 Forbidden` if validation fails (mismatch, not a member, or header missing)
- [x] Apply `orgScopeMiddleware` to all non-administrative (data plane) APIs in `api/routes.go` (AC: 1)
  - [x] Wrap data plane endpoints (e.g., projects, messages, sessions, git, artifacts, activities) with `orgScopeMiddleware`
  - [x] Ensure `orgScopeMiddleware` runs after `authMiddleware` (or `optionalAuthMiddleware` if applicable, but verify if they should be fully authenticated)
  - [x] Update `GET /api/v1/projects` to use `authMiddleware` instead of `optionalAuthMiddleware` to ensure claims are present for org validation
- [x] Write unit tests for `orgScopeMiddleware` in `api/middleware_test.go` (AC: 1)
  - [x] Test success with matching org-scoped token
  - [x] Test success with primary token and allowed org in header
  - [x] Test failure with mismatched org ID
  - [x] Test failure with org ID not in allowed list
  - [x] Test failure with missing header
- [x] Verify existing tests pass and update if necessary

### Review Findings

- [x] [Review][Patch] meHandler fails with 500 when called with primary token [api/auth_handlers.go:19]
- [x] [Review][Patch] Unwrapped User and Integration endpoints break primary token access [api/routes.go:28-37, 121-143]
- [x] [Review][Patch] Global State Mutation and Race Conditions in TestRoutesMiddlewareIntegration [api/routes_test.go:332]
- [x] [Review][Patch] Hardcoded Header String "X-Organization-Id" [api/middleware.go:18]
- [x] [Review][Patch] Missing Header Format Validation for X-Organization-Id [api/middleware.go:18]
- [x] [Review][Patch] Leaked Domain Logic in Middleware [api/middleware.go:25]
- [x] [Review][Patch] Asymmetric Context API (Missing Context Setter Helper) [api/middleware.go:45]
- [x] [Review][Patch] Insufficient Test Coverage for Empty Claims in orgScopeMiddleware [api/middleware_test.go:78]
- [x] [Review][Patch] Superficial Integration Testing in TestRoutesMiddlewareIntegration [api/routes_test.go:332]
- [x] [Review][Defer] Likely Copy-Paste Bug in Route Registration (Approve handler using gitCommitHandler is pre-existing) [api/routes.go:97] — deferred, pre-existing

## Dev Notes

### Middleware Design
The `orgScopeMiddleware` will act as a guard for all data plane APIs. It ensures that every request is explicitly scoped to an organization that the user is authorized to access.

```go
func orgScopeMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
		if !ok || claims == nil {
			http.Error(w, "Unauthorized: missing user claims", http.StatusUnauthorized)
			return
		}

		reqOrgID := r.Header.Get("X-Organization-Id")
		if reqOrgID == "" {
			http.Error(w, "Forbidden: X-Organization-Id header is required", http.StatusForbidden)
			return
		}

		// Validate organization access
		if claims.OrganizationID != "" {
			if claims.OrganizationID != reqOrgID {
				http.Error(w, "Forbidden: Organization mismatch", http.StatusForbidden)
				return
			}
		} else {
			authorized := false
			for _, org := range claims.Organizations {
				if org == reqOrgID {
					authorized = true
					break
				}
			}
			if !authorized {
				http.Error(w, "Forbidden: User is not a member of this organization", http.StatusForbidden)
				return
			}
		}

		// Inject validated organization ID into context
		ctx := context.WithValue(r.Context(), core.OrganizationIDKey, reqOrgID)
		next(w, r.WithContext(ctx))
	}
}
```

### Affected Endpoints
All endpoints under `/api/v1/projects/...` (projects, messages, sessions, git, artifacts, activities) are considered data plane and must be protected by this middleware.

### References

- Multi-Tenancy Isolation Requirements: [docs/spec/planning-artifacts/prd.md#FR-8.1](docs/spec/planning-artifacts/prd.md)
- Architecture boundaries: [docs/spec/planning-artifacts/architecture.md#Data-&-Resource-Architecture](docs/spec/planning-artifacts/architecture.md)
- Epics breakdown: [docs/spec/planning-artifacts/epics.md#Story-10.2](docs/spec/planning-artifacts/epics.md)
- Previous Story (10.1): [10-1-data-partitioning-by-organization-id-across-all-storage-mediums.md](docs/spec/implementation-artifacts/10-1-data-partitioning-by-organization-id-across-all-storage-mediums.md)

## Dev Agent Record

### Agent Model Used

Gemini Next

### Debug Log References

### Completion Notes List
- Implemented `orgScopeMiddleware` in `api/middleware.go` to validate `X-Organization-Id` header against user claims (both org-scoped and primary tokens).
- Injected validated organization ID into request context.
- Wrapped all data plane endpoints under `/api/v1/projects` with `orgScopeMiddleware` in `api/routes.go`.
- Updated `GET /api/v1/projects` to use `authMiddleware` (required for claims) and `orgScopeMiddleware`.
- Added unit tests for `orgScopeMiddleware` in `api/middleware_test.go` covering success and failure cases.
- Created `api/routes_test.go` to verify route-level middleware integration.

### File List
- `api/middleware.go`
- `api/routes.go`
- `api/middleware_test.go`
- `api/routes_test.go`

