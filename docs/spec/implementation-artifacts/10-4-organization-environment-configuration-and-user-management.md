# Story 10.4: Organization Environment Configuration & User Management

Status: review

## Story

As a Organization Administrator,
I want to be able to configure environment settings for the organization environment and add users to it,
so that my organization can customize workspace integrations and govern user access within our isolated organization scope.

## Acceptance Criteria

1. **Given** an authenticated Organization Administrator session
   **When** the organization administrator submits environment configuration settings or adds a user email
   **Then** the administrative API MUST update the organization's configuration profile and authorization sub-tree (`users`) in Firestore
   **And** grant the invited user access restricted strictly to that organization's workspace and storage boundaries

## Tasks / Subtasks

- [x] Update Database Models (AC: 1)
  - [x] Add `Role` and `Email` fields to `db.User` struct in `api/db/user.go`
  - [x] Add `Config` (type `OrganizationConfig`) field to `db.Organization` struct in `api/db/organization.go`
  - [x] Define `OrganizationConfig`, `BrandingConfig`, `SSOConfig` structs in `api/db/organization.go`
- [x] Implement Database Store Methods (AC: 1)
  - [x] Add and implement `InviteUser(ctx context.Context, organizationID string, email string, role string) error` in `db.UserStore` and `db.FirestoreUserStore`
  - [x] Add and implement `UpdateUserRole(ctx context.Context, organizationID string, id string, role string) error` in `db.UserStore` and `db.FirestoreUserStore`
  - [x] Add and implement `UpdateOrganizationConfig(ctx context.Context, id string, config *db.OrganizationConfig) error` in `db.OrganizationStore` and `db.FirestoreOrganizationStore`
  - [x] Update `GetOrCreateUser` in `db.FirestoreUserStore` to reconcile email-based invitations on first login (migrate email doc to UID doc)
- [x] Implement Admin Middleware (AC: 1)
  - [x] Implement `orgAdminAccessMiddleware(next http.HandlerFunc) http.HandlerFunc` in `api/middleware.go` to verify user has `admin` role in the active organization matching the request path
- [x] Implement Administrative API Handlers (AC: 1)
  - [x] Create `api/admin_handlers.go`
  - [x] Implement `updateOrganizationConfigHandler` (`PUT /api/v1/admin/organizations/{orgId}/config`)
  - [x] Implement `inviteUserHandler` (`POST /api/v1/admin/organizations/{orgId}/users`)
  - [x] Implement `listOrganizationUsersHandler` (`GET /api/v1/admin/organizations/{orgId}/users`)
  - [x] Implement `removeUserHandler` (`DELETE /api/v1/admin/organizations/{orgId}/users/{userId}`)
- [x] Register Routes (AC: 1)
  - [x] Register new admin routes in `api/routes.go` protected by `authMiddleware` and `orgAdminAccessMiddleware`
- [x] Seeding & Testing (AC: 1)
  - [x] Update `seedLocalData` in `api/seeding/local.go` to assign "admin" role to the simulated admin user
  - [x] Add unit tests in `api/db/user_test.go` for invitation reconciliation and role updates
  - [x] Add integration tests in `api/admin_handlers_test.go` (new file) verifying all admin endpoints and RBAC gating

### Review Findings

- [x] [Review][Decision] Remove Last Admin / Self-Deletion Safety — Should we block removing the last admin or self-deletion in `removeUserHandler`?
- [x] [Review][Decision] HTTP Semantics Violation (Reconciliation on GET) — Should we remove reconciliation logic from `orgAdminAccessMiddleware` to avoid DB writes on GET requests, relying solely on `selectOrganizationHandler`?
- [x] [Review][Decision] JWT Role Trust vs DB Check — Should `orgAdminAccessMiddleware` trust the JWT claims for the admin role, or should it always check the database to support immediate revocation?
- [x] [Review][Patch] Reconciliation Email Fallback [api/db/user.go:112]
- [x] [Review][Patch] Reconciliation when UID is Email [api/db/user.go:104]
- [x] [Review][Patch] Duplicate Invitations [api/admin_handlers.go:75]
- [x] [Review][Patch] SSO Config Overwrite [api/admin_handlers.go:18]
- [x] [Review][Patch] Unsafe DB Write in UpdateUserRole [api/db/user.go:767]
- [x] [Review][Patch] Unsafe DB Write in UpdateOrganizationConfig [api/db/organization.go:538]
- [x] [Review][Patch] Duplicated Reconciliation Logic [api/db/user.go:112]
- [x] [Review][Patch] Role Type Safety [api/db/user.go:652]
- [x] [Review][Patch] Ignored Deletion Error [api/db/user.go:714]
- [x] [Review][Patch] Type Assertion in Tests [api/admin_handlers_test.go:296]
- [x] [Review][Patch] Misleading Email Fallback [api/admin_handlers.go:135]
- [x] [Review][Defer] Pagination on List Users [api/db/user.go:789] — deferred, pre-existing

## Dev Notes

### Firestore Structure for Invitations
Invitations are stored as user documents using the email as the document ID:
`/organizations/{organizationId}/users/{email}`
Fields:
- `email`: `user@example.com`
- `role`: `viewer` (or specified role)

Upon first login, the `GetOrCreateUser` method will check for an existing document with the user's email, copy the role to a new document with the user's UID as ID, and delete the email-based document.

### Organization Configuration
The organization document at `/organizations/{organizationId}` will support a `config` map containing branding and SSO settings:
```json
{
  "name": "Organization Name",
  "config": {
    "branding": {
      "logo_url": "https://example.com/logo.png",
      "theme": "dark"
    },
    "sso": {
      "enabled": true,
      "provider_url": "https://sso.example.com"
    },
    "env_settings": {
      "some_key": "some_value"
    }
  }
}
```

### API Contracts

#### PUT /api/v1/admin/organizations/{orgId}/config
Request Payload:
```json
{
  "branding": {
    "logo_url": "string",
    "theme": "string"
  },
  "sso": {
    "enabled": true,
    "provider_url": "string"
  },
  "env_settings": {
    "key": "value"
  }
}
```
Response: `200 OK` with updated organization object, or error.

#### POST /api/v1/admin/organizations/{orgId}/users
Request Payload:
```json
{
  "email": "user@example.com",
  "role": "admin|viewer"
}
```
Response: `201 Created` with created user invitation object, or error.

#### GET /api/v1/admin/organizations/{orgId}/users
Response: `200 OK` with list of users:
```json
{
  "users": [
    {
      "id": "uid_or_email",
      "userId": "uid_if_registered_else_empty",
      "email": "user@example.com",
      "role": "admin|viewer",
      "status": "active|invited"
    }
  ]
}
```

### Files to Touch
- `api/db/user.go`: Update `User` struct, `UserStore` interface, `FirestoreUserStore` implementation.
- `api/db/organization.go`: Update `Organization` struct, `OrganizationStore` interface, `FirestoreOrganizationStore` implementation.
- `api/middleware.go`: Implement `orgAdminAccessMiddleware`.
- `api/routes.go`: Register admin endpoints.
- `api/admin_handlers.go`: New file for admin handlers.
- `api/seeding/local.go`: Update seeding data.
- `api/db/user_test.go`: Add store tests.
- `api/admin_handlers_test.go`: New file for handler integration tests.

### Testing Standards
- Store tests should use Firestore emulator (verify via `test_helper_test.go` pattern).
- Admin API tests should mock auth and verify both success (admin access) and failure (non-admin 403, unauthenticated 401) cases.

## Dev Agent Record

### Agent Model Used

Gemini Next

### Debug Log References
- All tests passing in local emulator environment.

### Completion Notes List
- Updated db.User and db.Organization models with new configuration and role fields.
- Implemented db.UserStore and db.OrganizationStore methods for invitations, role updates, and configuration management.
- Implemented orgAdminAccessMiddleware to restrict access to admin endpoints to users with "admin" role in the active organization.
- Implemented administrative API handlers for configuration updates, user invitations, listing, and removal.
- Integrated invitation reconciliation on first login (during select-organization or admin access).
- Added unit and integration tests covering all new features and RBAC gating.

### File List
- [api/db/user.go](api/db/user.go)
- [api/db/organization.go](api/db/organization.go)
- [api/db/user_test.go](api/db/user_test.go)
- [api/db/organization_test.go](api/db/organization_test.go) (new)
- [api/middleware.go](api/middleware.go)
- [api/routes.go](api/routes.go)
- [api/admin_handlers.go](api/admin_handlers.go) (new)
- [api/admin_handlers_test.go](api/admin_handlers_test.go) (new)
- [api/user_handlers_test.go](api/user_handlers_test.go)
- [api/seeding/local.go](api/seeding/local.go)

