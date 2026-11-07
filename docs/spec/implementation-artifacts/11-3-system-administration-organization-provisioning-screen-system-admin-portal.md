# Story 11.3: System Administration - Organization Provisioning Screen (System Admin Portal)

Status: done

## Story

As a Super Administrator,
I want a System Admin Portal to provision a new Organization and assign its initial Owner,
so that new tenants can be onboarded to the platform with isolated resources.

## Acceptance Criteria

1. **Access Control**: Only authenticated users with the `is_super_admin` claim (Super Administrators) can access the System Admin Portal (`/admin/provision`). Regular users or organization admins must be blocked (403 Forbidden on API, redirected on UI).
2. **Super Admin Authentication**: Super Admins authenticate at the parent project level (root), meaning their token does not contain a `firebase.tenant` claim and they possess the custom claim `is_super_admin === true`.
3. **UI Form**: The `/admin/provision` page must render a form capturing:
   - **Organization Name** (Required, text, validated for non-empty).
   - **Owner Email** (Required, email format validated).
4. **Provisioning Action**: Clicking "Provision Organization" must:
   - Validate inputs on the client side.
   - Show a loading spinner and disable form inputs.
   - Submit a `POST /api/v1/organizations` request to the backend with payload `{ name: "Org Name", ownerEmail: "owner@email.com" }`.
5. **Backend Provisioning Flow**: The backend handler must execute the following steps atomically or with rollback:
   - Verify the caller is a Super Admin (`claims.IsSuperAdmin == true`).
   - Call Firebase Admin SDK `TenantManager` to create a new Tenant with the organization name.
   - Save the Organization in Firestore using the Firebase Tenant ID as the Firestore Document ID under `/organizations/{tenantID}`.
   - Call Firebase Admin SDK Tenant Client to create the owner user in that tenant and set custom claim `{ role: "owner" }`.
   - Create the owner user record in Firestore under `/organizations/{tenantID}/users/{ownerUID}` with `role: "owner"`.
   - **Error Rollback**: If database registration or owner creation fails after tenant creation, the backend must invoke `TenantManager.DeleteTenant` to prevent orphaned resources.
6. **UI Feedback**:
   - On success: Show a clear success message with the new Organization ID and Owner UID, and clear the form.
   - On failure: Show a prominent error message (e.g., "Organization name already exists" or "Failed to provision organization") and keep form inputs populated.

## Tasks / Subtasks

### Backend Tasks

- [ ] **Core Types & Authentication** (AC: 1, 2)
  - [ ] Add `IsSuperAdmin bool` to `core.UserClaims` struct in [api/core/types.go](api/core/types.go) with `json:"is_super_admin"` tag.
  - [ ] Update `MockAuthenticator.VerifyToken` in [api/auth.go](api/auth.go) to support "super-admin" username (setting `IsSuperAdmin: true` and `Roles: []string{"admin"}`).
  - [ ] Update `FirebaseAuthAuthenticator.VerifyToken` in [api/auth.go](api/auth.go) to check `is_super_admin` claim in Firebase token and set `IsSuperAdmin: true` (only if organization is empty).
  - [ ] Define `OrganizationProvisioner` interface in [api/auth.go](api/auth.go):
    ```go
    type OrganizationProvisioner interface {
        CreateOrganization(ctx context.Context, name string) (string, error)
        CreateOrganizationOwner(ctx context.Context, orgID string, email string) (string, error)
        DeleteOrganization(ctx context.Context, orgID string) error
    }
    ```
  - [ ] Implement `MockOrganizationProvisioner` in [api/auth.go](api/auth.go) for local development (returns mock IDs, no-op on delete).
  - [ ] Implement `FirebaseOrganizationProvisioner` in [api/auth.go](api/auth.go) using Firebase Admin SDK (`Client.TenantManager`).
  - [ ] Declare `globalOrganizationProvisioner` in [api/auth.go](api/auth.go) (defaults to `MockOrganizationProvisioner`) and initialize it with `FirebaseOrganizationProvisioner` in [api/main.go](api/main.go) when Firebase Auth client is initialized.

- [ ] **Database Stores Update** (AC: 5)
  - [ ] Add `RoleOwner UserRole = "owner"` to constants in [api/db/user.go](api/db/user.go).
  - [ ] Add `CreateUser(ctx context.Context, organizationID string, user *User) error` to `UserStore` interface in [api/db/user.go](api/db/user.go) and implement it in `FirestoreUserStore` (creating doc at `/organizations/{orgID}/users/{userID}`).
  - [ ] Update `mockUserStore` in [api/user_handlers_test.go](api/user_handlers_test.go) to implement `CreateUser`.
  - [ ] Add `CreateOrganizationWithID(ctx context.Context, id string, org *Organization) error` to `OrganizationStore` interface in [api/db/organization.go](api/db/organization.go) and implement it in `FirestoreOrganizationStore` (creating doc at `/organizations/{id}` using transaction to check duplicate lowercase name).
  - [ ] Update `mockOrganizationStore` in [api/organization_handlers_test.go](api/organization_handlers_test.go) to implement `CreateOrganizationWithID`.

- [ ] **API Handler Implementation** (AC: 1, 3, 5)
  - [ ] Update `CreateOrganizationRequest` struct in [api/organization_handlers.go](api/organization_handlers.go) to include `OwnerEmail string json:"ownerEmail"`.
  - [ ] Update `createOrganizationHandler` in [api/organization_handlers.go](api/organization_handlers.go):
    - [ ] Enforce `claims.IsSuperAdmin == true` (return 403 Forbidden if not).
    - [ ] Validate `Name` and `OwnerEmail` are non-empty and valid (return 400 Bad Request if not).
    - [ ] Run fast check for duplicate name in Firestore using `GetOrganizationByName` (return 409 Conflict if exists).
    - [ ] Call `globalOrganizationProvisioner.CreateOrganization` -> get `orgID`.
    - [ ] Call `db.OrganizationStoreInstance.CreateOrganizationWithID` with `orgID` -> if fails, delete organization and return error.
    - [ ] Call `globalOrganizationProvisioner.CreateOrganizationOwner` with `orgID` and `OwnerEmail` -> get `ownerUID`. If fails, delete organization and return error.
    - [ ] Call `userStore.CreateUser` to register owner in Firestore under `/organizations/{orgID}/users/{ownerUID}` with `role: "owner"`. If fails, delete organization and return error.

- [ ] **Tests Verification**
  - [ ] Update [api/organization_handlers_test.go](api/organization_handlers_test.go):
    - [ ] Mock `userStore` in success tests.
    - [ ] Update all test cases to use `IsSuperAdmin: true` claims.
    - [ ] Pass `OwnerEmail` in request payloads.
    - [ ] Add verification that owner user was created in the database.
    - [ ] Rename `TestCreateOrganizationNotAdmin` to `TestCreateOrganizationNotSuperAdmin` and assert 403 when `IsSuperAdmin` is false.
  - [ ] Run `go test ./...` and ensure all tests pass.

### Frontend Tasks

- [ ] **Auth Store Update** (AC: 1, 2)
  - [ ] Update `AuthState` interface in [ui/src/stores/auth.ts](ui/src/stores/auth.ts) to include `isSuperAdmin: boolean`.
  - [ ] Update `getDerivedState` in [ui/src/stores/auth.ts](ui/src/stores/auth.ts) to set `isSuperAdmin: true` if token is `simulated-jwt-token-for-super-admin`.
  - [ ] Update `_checkAuthStatus` in [ui/src/stores/auth.ts](ui/src/stores/auth.ts) to read `is_super_admin` from `/api/v1/auth/me` response and set it in store state.
  - [ ] Update `login` in [ui/src/stores/auth.ts](ui/src/stores/auth.ts) to support `super-admin` / `super-admin` credentials, setting token `simulated-jwt-token-for-super-admin` in local storage.

- [ ] **Route Guard Update** (AC: 1)
  - [ ] Update `RouteGuard` in [ui/src/components/shared/RouteGuard.tsx](ui/src/components/shared/RouteGuard.tsx):
    - [ ] Import `isSuperAdmin` from `useAuthStore`.
    - [ ] If `isAuthenticated` and `isSuperAdmin` is true:
      - [ ] If current path does not start with `/admin`, redirect to `/admin/provision`.
      - [ ] Bypass the `selectedOrgId` check (do NOT redirect to `/select-org`).

- [ ] **Provisioning Screen UI** (AC: 3, 4, 6)
  - [ ] Create a new page at `ui/src/app/admin/provision/page.tsx`.
  - [ ] Render a card layout containing:
    - [ ] Header: "Provision New Organization" with subtitle "Create a new isolated organization and assign its initial owner."
    - [ ] Input field for **Organization Name** (placeholder: "e.g. Acme Corp").
    - [ ] Input field for **Owner Email** (placeholder: "e.g. admin@acme.com").
    - [ ] Submit button: "Provision Organization".
  - [ ] Implement form state:
    - [ ] Client-side validation (non-empty name, valid email format).
    - [ ] Loading state: disable inputs and show spinner on button during submission.
    - [ ] Success state: show green banner with "Organization provisioned successfully!" and display the generated Organization ID. Clear form inputs.
    - [ ] Error state: show red banner with the error message returned by the API (e.g. "Organization name already exists"). Keep inputs populated.
  - [ ] Ensure the UI is fully responsive, dark-mode compatible, and matches the Agentic Product Foundry design system (using tailwind colors and icons from `lucide-react`).

## Dev Notes

- **Firebase Multi-Tenancy**: The Firebase Admin SDK TenantManager is used to create isolated organizations. The organization ID is used as the Firestore Organization Document ID, ensuring perfect synchronization between auth and database.
- **Super Admin Definition**: Super admins are defined at the root project level (no organization ID in token) and must have `is_super_admin: true` custom claim.
- **Rollback Robustness**: Since Firestore and Firebase Auth are separate systems, the backend must clean up the created Firebase organization if subsequent Firestore writes fail.

### Project Structure Notes

- Backend changes are confined to the `api` module, utilizing existing store patterns and interfaces.
- Frontend changes introduce a new administrative route `/admin/provision` and update the centralized `RouteGuard` and `auth` store.

### References

- Technology override rationale: [architecture.md#L70-L85](docs/spec/planning-artifacts/architecture.md#L70-L85)
- Firestore project boundaries: [architecture.md#L112-L122](docs/spec/planning-artifacts/architecture.md#L112-L122)
- Super Admin & Organization Owner definition: [epics.md#Story 11.8 & 11.3](docs/spec/planning-artifacts/epics.md#L856-L906)

## Dev Agent Record

### Agent Model Used

gemini-1.5-pro-002 (Jetski Expert Developer Pair)

### Debug Log References

- Mock tests execution: `go test -v -run TestCreateOrganization`
- Full suite compilation check: `go test ./...`

### Review Findings

- [x] [Review][Patch] RouteGuard gating regular user access and null-check pathname [ui/src/components/shared/RouteGuard.tsx:73]
- [x] [Review][Patch] Delete Firestore owner user doc on provisioning rollback [api/organization_handlers.go:91]
- [x] [Review][Patch] Use bounded timeout context for rollback operations [api/organization_handlers.go:84]
- [x] [Review][Patch] Gate hardcoded mock super-admin credentials with development check [ui/src/stores/auth.ts:178]
- [x] [Review][Patch] Gracefully handle non-JSON server errors in frontend [ui/src/app/admin/provision/page.tsx:56]
- [x] [Review][Patch] Nil pointer guard in CreateOrganizationWithID [api/db/organization.go:119]
- [x] [Review][Patch] Nil pointer and empty ID guards in CreateUser [api/db/user.go:241]
- [x] [Review][Patch] Handle DB conflict error with 409 Conflict in handler [api/organization_handlers.go:112]
- [x] [Review][Defer] Insecure storage of sensitive tokens in localStorage [ui/src/stores/auth.ts:169] — deferred, pre-existing
- [x] [Review][Defer] Silent failures in rollback logic (logging only, no DLQ) [api/organization_handlers.go:84] — deferred, pre-existing
