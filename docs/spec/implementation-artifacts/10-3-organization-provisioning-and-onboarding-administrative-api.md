# Story 10.3: Organization Provisioning & Onboarding (Administrative API)

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a System Administrator,
I want to be able to create a new organization in the system,
so that enterprise customers can be securely provisioned with dedicated workspace and storage boundaries.

## Acceptance Criteria

1. **Given** an authenticated System Administrator session
2. **When** the administrator submits a new organization onboarding request
3. **Then** the administrative API MUST initialize a new organization record in Firestore under a central `organizations` collection
4. **And** allocate the corresponding organization storage bucket prefix and isolation boundaries

## Tasks / Subtasks

- [x] Define API Route and Handler for Organization Creation (AC: 1, 2, 3)
  - [x] Add route `POST /api/v1/organizations` in `api/routes.go` protected by `authMiddleware`.
  - [x] Create `api/organization_handlers.go` to house the handler.
  - [x] Implement `createOrganizationHandler` to decode `CreateOrganizationRequest` (containing `name`).
  - [x] Enforce role check: verify the caller has the `admin` role in their claims. Return `403 Forbidden` if not.
  - [x] Enforce name uniqueness: check if an organization with the same name already exists in Firestore, and return `409 Conflict` if it does.
- [x] Implement DB layer for Organization Creation (AC: 3)
  - [x] Call `db.OrganizationStoreInstance.CreateOrganization` to save the new organization.
  - [x] Ensure the created organization ID is returned in the response (`CreateOrganizationResponse`).
- [x] Verify Storage and Isolation Boundaries (AC: 4)
  - [x] Confirm that the created organization ID works with the partitioning logic implemented in Story 10.1 (Firestore nested collections `/organizations/{organizationId}/...`, GCS paths, and NFS workspace paths).
- [x] Implement Tests (AC: 1, 2, 3, 4)
  - [x] Create `api/organization_handlers_test.go` to test the new endpoint.
  - [x] Test successful organization creation by a user with the `admin` role.
  - [x] Test that a user without the `admin` role (e.g., standard `user` or `viewer`) is rejected with `403 Forbidden`.
  - [x] Test validation: empty organization name should return `400 Bad Request`.
  - [x] Test duplicate handling: attempting to create an organization with an existing name should return `409 Conflict`.
  - [x] Test integration: verify the organization document is actually created in the Firestore emulator.

## Dev Notes

- Reuses `db.OrganizationStoreInstance` which is already initialized in `api/main.go` and backed by `FirestoreOrganizationStore` in `api/db/organization.go`.
- Auth checks should leverage the existing `core.UserClaims` from the request context, populated by `authMiddleware`.
- GCS and NFS isolation boundaries are virtual and path-based (implemented in 10.1), so creating the Firestore record is sufficient for "allocation".

### Project Structure Notes

- New handler file `api/organization_handlers.go` aligns with existing patterns like `api/user_handlers.go`.
- New test file `api/organization_handlers_test.go` aligns with existing handler tests.

### References

- Multi-Tenancy Isolation Requirements: [docs/spec/planning-artifacts/prd.md#FR-8.1](docs/spec/planning-artifacts/prd.md)
- Epics breakdown: [docs/spec/planning-artifacts/epics.md#Story-10.3](docs/spec/planning-artifacts/epics.md#L768-L779)
- Previous Story 10.1 context: [docs/spec/implementation-artifacts/10-1-data-partitioning-by-organization-id-across-all-storage-mediums.md](docs/spec/implementation-artifacts/10-1-data-partitioning-by-organization-id-across-all-storage-mediums.md)

## Dev Agent Record

### Agent Model Used

Gemini Next

### Debug Log References

### Completion Notes List
- Implemented `POST /api/v1/organizations` endpoint to provision a new organization.
- Enforced admin-only access using JWT roles checking (`claims.Roles`).
- Handled organization name uniqueness constraints by checking Firestore with the new `GetOrganizationByName` database helper.
- Responded with `201 Created` containing the generated organization ID and name.
- Created `api/organization_handlers_test.go` verifying success flows, role authorization, validation errors, and Firestore integration.

### File List
- `api/db/organization.go`
- `api/routes.go`
- `api/organization_handlers.go`
- `api/organization_handlers_test.go`

### Review Findings

- [x] [Review][Patch] Enforce case-insensitive organization name uniqueness [api/organization_handlers.go:55]
- [x] [Review][Patch] Race Condition in Organization Creation [api/organization_handlers.go:55]
- [x] [Review][Patch] Lax Input Validation on Organization Name (Whitespace) [api/organization_handlers.go:49]
- [x] [Review][Patch] Request Body Size Limit Missing [api/organization_handlers.go:44]
- [x] [Review][Patch] Information Disclosure and Missing Logging in Handler [api/organization_handlers.go:61]
- [x] [Review][Patch] Resource Leak in Integration Test (Cleanup) [api/organization_handlers_test.go:260]
- [x] [Review][Patch] Global State Pollution in Tests [api/organization_handlers_test.go:50]
- [x] [Review][Patch] Missing test coverage for invalid JSON request payloads [api/organization_handlers_test.go]
- [x] [Review][Defer] System Administrator cannot select organization due to missing role bypass [api/auth.go:381] — deferred, pre-existing

