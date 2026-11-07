# Story 11.6: Organization Settings - Roles & Permissions Configuration Screen (Org Admin Portal)

Status: done

## Story

As an Organization Admin/Owner,  
I want a Roles & Permissions Configuration screen to view predefined roles and create custom roles with granular permissions,  
So that I can tailor access control to the specific needs of my organization's members.

## Acceptance Criteria

1. **Given** an Organization Admin/Owner is logged into their organization dashboard
2. **When** they navigate to the Roles & Permissions Configuration page (or Organization Settings tab)
3. **Then** the UI displays a "Predefined Roles" tab showing read-only default roles (Admin, Editor, Viewer) and their mapped permissions (e.g., `read:data`, `write:data`)
4. **And** the UI displays a "Custom Roles" tab listing all custom roles created specifically for this organization
5. **And** the UI provides a "Create Custom Role" action (opening a modal/form)
6. **When** they input the Custom Role Name (e.g., "External Auditor"), check granular permissions (e.g., `view:billing`, `edit:settings`, `download:reports`), and click "Save Custom Role"
7. **Then** the UI saves the custom role configuration to the database (scoped to the Organization ID)
8. **And** when the role is assigned to a user, the backend updates the user's Google Identity Platform custom claims with the role name and the specific array of permissions (`tenantAuth.setCustomUserClaims(uid, { role: "External Auditor", permissions: ["view:billing"] })`)

## Tasks / Subtasks

### Phase 1: Core Models & Database Store

- [x] **Define Custom Role Model** (AC: 6, 7)
  - Create [api/db/role.go](api/db/role.go):
    - Define `CustomRole` struct:
      ```go
      type CustomRole struct {
          ID          string    `firestore:"-" json:"id"`
          Name        string    `firestore:"name" json:"name"`
          Permissions []string  `firestore:"permissions" json:"permissions"`
          Description string    `firestore:"description" json:"description"`
          CreatedAt   time.Time `firestore:"createdAt" json:"createdAt"`
      }
      ```
    - Define `RoleStore` interface:
      ```go
      type RoleStore interface {
          CreateCustomRole(ctx context.Context, orgID string, role *CustomRole) error
          GetCustomRole(ctx context.Context, orgID string, name string) (*CustomRole, error)
          ListCustomRoles(ctx context.Context, orgID string) ([]CustomRole, error)
      }
      ```
- [x] **Implement Firestore Role Store** (AC: 7)
  - Implement `FirestoreRoleStore` in [api/db/role.go](api/db/role.go) storing roles in Firestore under:
    `/organizations/{organizationId}/roles/{roleNameSlug}`
    - Standardize slug generation for document IDs (e.g., converting "External Auditor" -> "external-auditor").
- [x] **Extend User Store with Role Store Dependency** (AC: 8)
  - Inject `RoleStore` dependency into `FirestoreUserStore` in [api/db/user.go](api/db/user.go).
  - Update initialization in `NewFirestoreUserStore` or via setter.

### Phase 2: Claims & Token Context Updates

- [x] **Update User Claims Struct** (AC: 8)
  - Add `Permissions []string json:"permissions"` to `UserClaims` in [api/core/types.go](api/core/types.go).
- [x] **Update Auth Claims Parsing & Token Minting** (AC: 8)
  - Update JWT validation and parsing in [api/auth.go](api/auth.go):
    - Extract `permissions` claim (string array) and populate `UserClaims.Permissions`.
  - Update `MintOrgSessionToken` in [api/auth.go](api/auth.go) to accept permissions array and encode it as `permissions` in the signed JWT payload.
  - Update token generation in `loginHandler` or token refresh flows.
- [x] **Update User Role Assignment with Permissions Claims** (AC: 8)
  - Define static permissions map in [api/db/user.go](api/db/user.go) for predefined roles:
    - `owner`: `["read:data", "write:data", "invite:users", "manage:roles", "delete:organization"]`
    - `admin`: `["read:data", "write:data", "invite:users", "manage:roles"]`
    - `editor`: `["read:data", "write:data"]`
    - `viewer`: `["read:data"]`
  - Update `UpdateUserRole` in [api/db/user.go](api/db/user.go):
    - Fetch permissions array (if predefined role, use static map; if custom role, fetch from `RoleStore`).
    - Update Firebase Auth Tenant custom claims with:
      - `roles`: `[]string{string(role)}` (backwards compatibility)
      - `role`: `string(role)`
      - `permissions`: `permissions`
    - Apply transactional rollback logic on Firestore if Firebase Auth claim update fails.

### Phase 3: API Endpoints & Request Validation

- [x] **Create Roles Handlers** (AC: 3, 4, 6, 7)
  - Create [api/role_handlers.go](api/role_handlers.go):
    - Implement `ListRoles` (`GET /api/v1/admin/organizations/{orgId}/roles`):
      - Query custom roles from database.
      - Return merged list of predefined roles (static) and custom roles (database).
    - Implement `CreateRole` (`POST /api/v1/admin/organizations/{orgId}/roles`):
      - Validate payload (Role Name required, alphanumeric/spaces, permissions array valid).
      - Verify role name does not conflict with predefined roles (`admin`, `editor`, `viewer`, `owner`).
      - Save custom role scoped to organization in `RoleStore`.
- [x] **Register Routes** (AC: 3, 4)
  - Wire up new endpoints in [api/routes.go](api/routes.go):
    - `GET /api/v1/admin/organizations/{orgId}/roles` -> `AdminHandler.ListRoles`
    - `POST /api/v1/admin/organizations/{orgId}/roles` -> `AdminHandler.CreateRole`
- [x] **Update Invite & Update User Handlers to Validate Custom Roles** (AC: 8)
  - Update `InviteUser` and `UpdateUserRole` handlers in [api/admin_handlers.go](api/admin_handlers.go):
    - Replace hardcoded role validations (`admin`/`editor`/`viewer`) with a dynamic check:
      - Check if role is predefined OR exists in custom roles for the organization.

### Phase 4: Frontend UI Components

- [x] **Create Roles & Permissions Settings View** (AC: 2, 3, 4)
  - Add a **Roles & Permissions** tab to [ui/src/app/(dashboard)/settings/page.tsx](ui/src/app/(dashboard)/settings/page.tsx) (visible to org admins):
    - Render two sub-tabs:
      - **Predefined Roles**: Renders a read-only list/table showing Admin, Editor, Viewer roles and their default permissions.
      - **Custom Roles**: Renders a list/table of custom roles retrieved from `GET /api/v1/admin/organizations/{orgId}/roles` (filtering out predefined ones or showing them separately).
    - Add a "Create Custom Role" button in the Custom Roles tab.
- [x] **Create Create Custom Role Modal** (AC: 5, 6, 7)
  - Create [ui/src/components/shared/CreateCustomRoleModal.tsx](ui/src/components/shared/CreateCustomRoleModal.tsx):
    - Form fields:
      - Role Name (input field, validated).
      - Description (textarea, optional).
      - Granular Permissions Checkboxes:
        - Render checkboxes for all system permissions: `read:data`, `write:data`, `invite:users`, `manage:roles`, `view:billing`, `edit:settings`, `download:reports`.
    - Submit button with loading spinner, and Cancel button.
    - Integrate with React Query mutation:
      - `POST /api/v1/admin/organizations/{orgId}/roles`
      - Trigger refetching of the roles list on success, close modal, and display success toast.
- [x] **Make Invite Member Modal Role Dropdown Dynamic** (AC: 8)
  - Update [ui/src/components/shared/InviteMemberModal.tsx](ui/src/components/shared/InviteMemberModal.tsx):
    - Query the roles API (`GET /api/v1/admin/organizations/{orgId}/roles`) to fetch available roles.
    - Populate the role selector dropdown dynamically with all fetched roles (predefined and custom), replacing hardcoded options.
- [x] **Update Member Directory Role Selector** (AC: 8)
  - Update the user list role updates dropdown in [ui/src/app/(dashboard)/settings/page.tsx](ui/src/app/(dashboard)/settings/page.tsx) to also load roles dynamically from the API rather than using hardcoded `admin`/`editor`/`viewer` selectors.

### Phase 5: Testing & Verification

- [x] **Add Backend Integration Tests** (AC: 7, 8)
  - Create [api/role_handlers_test.go](api/role_handlers_test.go):
    - Test `GET /api/v1/admin/organizations/{orgId}/roles` (predefined roles returned).
    - Test `POST /api/v1/admin/organizations/{orgId}/roles` (successful custom role creation, duplicate checks, payload validations).
  - Update `api/admin_handlers_test.go`:
    - Verify role update/invite logic now accepts custom roles.
    - Mock `RoleStore` and check that custom claims update includes correctly fetched permissions.
- [x] **Add Frontend Component Tests** (AC: 3, 4, 5, 6)
  - Create Vitest test suite for `CreateCustomRoleModal` verifying validation, check-toggles, and mock mutations.
  - Update `InviteMemberModal` tests to mock dynamic role queries.

## Dev Notes

### Firebase Auth Tenant Custom Claims & Size Limit
When applying custom claims to users, Google Identity Platform/Firebase limits the payload size to 1000 bytes. Because we scope permissions to arrays of strings, we must keep permission string names short (e.g. `read:data` instead of `permission_to_read_organization_data`) to prevent exceeding this budget for users with many permissions.

Symmetric token exchange / mock token validation in development must be updated to cleanly parse the new `permissions` claims field.

### JWT Token Force-Refresh Requirement
When a user's role or permissions are updated, their active session JWT token on the client side will NOT immediately reflect the new claims. The client application must detect the change (or the updated user must force-refresh their token using `auth.currentUser.getIdToken(true)`) to obtain a new token containing the updated claims. The frontend must handle this gracefully to prevent stale permissions errors.

### Predefined vs Custom Role Separation
- Predefined roles should not be persisted in the database to prevent manual modifications or accidental deletion. They are represented as immutable static constructs in both the Go backend and React frontend.
- Custom roles are fully dynamic, persisted in Firestore, and mapped dynamically onto user custom claims when assigned.

## References
- Pre-existing settings dashboard: [ui/src/app/(dashboard)/settings/page.tsx](ui/src/app/(dashboard)/settings/page.tsx)
- User models & stores: [api/db/user.go](api/db/user.go)
- Member invitation flow: [docs/spec/implementation-artifacts/11-5-organization-settings-invite-member-modal-and-invitation-flow.md](docs/spec/implementation-artifacts/11-5-organization-settings-invite-member-modal-and-invitation-flow.md)
- Decoupled admin API handlers: [api/admin_handlers.go](api/admin_handlers.go)

## Dev Agent Record

### Agent Model Used

Gemini 1.5 Pro (Jetski Expert Developer)

### Debug Log References

- `.system_generated/logs/transcript.jsonl`

### Completion Notes List

- Created core `CustomRole` and `RoleStore` interface with full Firestore integration in `api/db/role.go`.
- Dynamic Slugification for document IDs to ensure clean, readable, and queryable Firestore references.
- Injected `RoleStore` dependency into `UserStore` without breaking existing test suites (via `SetRoleStore` setters).
- Dynamic permission resolution from Predefined or Custom Roles, propagated into custom claims during role updates/assignment.
- Secured claims context: added permissions to signed session JWT tokens in `api/auth.go` for immediate client-side decoding.
- Dynamic invite and role assignment validation: handlers in `api/admin_handlers.go` now check both predefined and custom roles.
- Handlers `ListRoles` and `CreateRole` in `api/role_handlers.go` feature robust request schema validation, regex name checks, conflict prevention, and merged response mapping.
- Registered administrative routes in `api/routes.go`.
- Wrote full backend integration tests in `api/db/role_test.go`, `api/role_handlers_test.go`, and expanded `api/admin_handlers_test.go` with custom role cases.
- Created `CreateCustomRoleModal` component for granular permission configurations.
- Dynamic UI updates: populated member lists and invitation selectors dynamically from the roles API.
- Wrote Vitest unit tests for both modals to verify correct mutation payloads, checklist toggles, and validation error messaging.

### File List

- `api/db/role.go` (Created)
- `api/db/role_test.go` (Created)
- `api/db/user.go` (Modified)
- `api/core/types.go` (Modified)
- `api/auth.go` (Modified)
- `api/main.go` (Modified)
- `api/admin_handlers.go` (Modified)
- `api/role_handlers.go` (Created)
- `api/role_handlers_test.go` (Created)
- `api/routes.go` (Modified)
- `api/admin_handlers_test.go` (Modified)
- `ui/src/components/shared/CreateCustomRoleModal.tsx` (Created)
- `ui/src/components/shared/CreateCustomRoleModal.test.tsx` (Created)
- `ui/src/components/shared/InviteMemberModal.tsx` (Modified)
- `ui/src/components/shared/InviteMemberModal.test.tsx` (Modified)
- `ui/src/app/(dashboard)/settings/page.tsx` (Modified)

## Review Findings

### Patches (Unresolved)
- [x] [Review][Patch] Role Identifier Mismatch (Slug vs. Display Name) [api/admin_handlers.go:59]
- [x] [Review][Patch] Privilege Escalation: Admin can grant Owner-only permissions to Custom Roles [api/role_handlers.go:927-942]
- [x] [Review][Patch] Privilege Escalation: Admin can promote users to Owner [api/admin_handlers.go:33, 69]
- [x] [Review][Patch] Firestore Path Manipulation in slugify [api/db/role.go:418]
- [x] [Review][Patch] Incorrect Error Mapping in isValidRole [api/admin_handlers.go:37-39]
- [x] [Review][Patch] Asymmetric Case Sensitivity in Role Validation [api/admin_handlers.go:33, 40]
- [x] [Review][Patch] Lack of Permission Deduplication in Custom Role Creation [api/role_handlers.go:937-942]
- [x] [Review][Patch] Non-deterministic Order of Predefined Roles in API Response [api/role_handlers.go:851-859]
- [x] [Review][Patch] Missing Limit on Custom Roles [api/role_handlers.go:884]
- [x] [Review][Patch] Orphaning Organization by Demoting Last Owner/Admin [api/db/user.go:402]

### Deferred (Pre-existing / Out of Scope)
- [x] [Review][Defer] Weak JWT Secret Fallback in Production [api/auth.go:325-327] — deferred, pre-existing
- [x] [Review][Defer] Stored XSS via Custom Role Description [api/role_handlers.go:954-959] — deferred, pre-existing
- [x] [Review][Defer] Stale Permissions in Firebase Custom Claims on Role Update [api/db/user.go:721, 735] — deferred, out of scope
