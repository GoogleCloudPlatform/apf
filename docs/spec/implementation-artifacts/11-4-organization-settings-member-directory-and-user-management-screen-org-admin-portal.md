# Story 11.4: Organization Settings - Member Directory & User Management Screen (Org Admin Portal)

Status: done

## Story

As an Organization Admin/Owner,
I want a Member Directory screen to manage my organization's members, statuses, and roles,
so that I can control who has access to our organization's workspace and configure their roles.

## Acceptance Criteria

1. **Given** an Organization Admin/Owner is logged into their organization dashboard
   **When** they navigate to the Member Directory page
   **Then** the UI requests the member list from the backend for that active organization tenant (AC: 1)
2. **And** the backend queries the Google Identity Platform for that tenant (`tenantAuth.listUsers()`) to retrieve user profiles, statuses, and custom claims (AC: 2)
3. **And** the UI renders a table listing all members including their Name, Email, Role (from `customClaims`), and Status (Active, Invited, Pending) (AC: 3)
4. **And** the UI provides an "Invite Member" action button (AC: 4)
5. **And** the UI provides actions next to each member to update their role (e.g., promoting/demoting) or revoke their access (deleting the user from the tenant) (AC: 5)
6. **And** these role change or deletion actions update the user's custom claims or status in the Identity Platform Tenant via a secure backend API call (AC: 6)

## Tasks / Subtasks

### Phase 1: Backend - Firebase Tenant Auth Integration

- [ ] **Extend UserStore for Firebase Auth Tenant Support** (AC: 2, 6)
  - [ ] Update `db.UserStore` and `db.FirestoreUserStore` in `api/db/user.go` to accept the Firebase Auth Client.
  - [ ] Implement Tenant Client retrieval helper in `FirestoreUserStore` using `authClient.TenantManager().AuthForTenant(tenantID)`.
  - [ ] Update `ListUsers(ctx, orgID)` to query Firebase Auth Tenant if configured.
    - [ ] Retrieve all users from the tenant.
    - [ ] Retrieve pending invites from Firestore (docs where `UserID` is empty or status is invited).
    - [ ] Merge lists: map Firebase users to "active" status and extract roles from custom claims (`roles` array); map Firestore docs to "invited" status.
  - [ ] Update `UpdateUserRole(ctx, orgID, userID, role)` to update both Firestore and Firebase Auth Tenant custom claims.
    - [ ] Set `roles` claim as `[]string{string(role)}` in the tenant.
  - [ ] Update `DeleteUser(ctx, orgID, userID)` to delete from Firestore AND delete the user from the Firebase Auth Tenant if they have a UID.
  - [ ] Ensure robust Mock Mode fallback: if Firebase Auth is not configured or in local dev without emulator, fallback gracefully to pure Firestore operations (preserving 10.4 behavior).

- [ ] **Implement Role Update HTTP Endpoint** (AC: 5, 6)
  - [ ] Add `PUT /api/v1/admin/organizations/{orgId}/users/{userId}/role` route in `api/routes.go`, protected by `authMiddleware` and `orgAdminAccessMiddleware`.
  - [ ] Implement `updateUserRoleHandler` in `api/admin_handlers.go`:
    - [ ] Decode role payload (verify it is "admin" or "viewer").
    - [ ] Prevent updating own role (self-demotion safety check).
    - [ ] Call `userStore.UpdateUserRole`.

- [ ] **Refactor Admin Handlers to use Dependency Injection** (Architecture Alignment)
  - [ ] Create `AdminHandler` struct in `api/admin_handlers.go` holding `UserStore` and `OrganizationStore`.
  - [ ] Refactor `updateOrganizationConfigHandler`, `inviteUserHandler`, `listOrganizationUsersHandler`, `removeUserHandler`, and the new `updateUserRoleHandler` as methods on `AdminHandler`.
  - [ ] Update `api/main.go` to initialize `AdminHandler` and `api/routes.go` to register its methods.

- [ ] **Unit & Integration Testing** (AC: 2, 5, 6)
  - [ ] Update `api/admin_handlers_test.go` to test the new role update endpoint (success, self-demotion block, invalid role).
  - [ ] Add tests for `ListUsers` merging logic (mocking Firebase tenant auth if possible, or verifying mock fallback).

### Phase 2: Frontend - Member Directory UI

- [ ] **Implement Member Directory UI Component** (AC: 1, 3, 4, 5)
  - [ ] Create or update `ui/src/app/(dashboard)/settings/page.tsx` (or a sub-page `ui/src/app/(dashboard)/settings/members/page.tsx`) to render the Member Directory.
  - [ ] Add "Members" tab/section in Settings navigation.
  - [ ] Render a high-density table listing members: Name (display name or email prefix), Email, Role, Status (with distinct badges: Active [Mint Green], Invited [Amber]).
  - [ ] Add "Invite Member" button (can open a stub modal or simple form for now, as full invite flow is Story 11.5).
  - [ ] Add Action Menu (or inline buttons) next to each member:
    - [ ] "Change Role" dropdown (Admin <-> Viewer).
    - [ ] "Revoke Access" (Delete) button.

- [ ] **Integrate UI with Backend APIs** (AC: 1, 5, 6)
  - [ ] Use `@tanstack/react-query` to fetch member list from `GET /api/v1/admin/organizations/{orgId}/users`.
  - [ ] Implement `useMutation` for Role Update calling `PUT /api/v1/admin/organizations/{orgId}/users/{userId}/role`.
  - [ ] Implement `useMutation` for User Removal calling `DELETE /api/v1/admin/organizations/{orgId}/users/{userId}`.
  - [ ] Add confirmation dialogs for destructive actions (Revoke Access) and self-action prevention.
  - [ ] Add loading states and error toast notifications.

### Review Findings

#### Patches
- [x] [Review][Patch] Missing "Name" Field in Member Directory (AC 3 Violation) [api/admin_handlers.go:147]
- [x] [Review][Patch] Missing "Pending" Status in Member Directory (AC 3 Violation) [api/admin_handlers.go:141]
- [x] [Review][Patch] Lack of Transactional Integrity in Role Updates (Firestore vs Firebase Auth) [api/db/user.go:478]
- [x] [Review][Patch] Lack of Transactional Integrity in User Deletion (Firestore vs Firebase Auth) [api/db/user.go:517]
- [x] [Review][Patch] Concurrency Race Condition in Last Admin Protection [api/admin_handlers.go:123]
- [x] [Review][Patch] Robustness Issues in User Listing (Silent Omission & All-or-Nothing Errors) [api/db/user.go:580]
- [x] [Review][Patch] Hard Crash on Firebase Init Failure [api/main.go:700]
- [x] [Review][Patch] Missing Validation for Role in Invite API [api/admin_handlers.go:77]
- [x] [Review][Patch] Race Condition in Concurrent Invites [api/admin_handlers.go:68]
- [x] [Review][Patch] Silent Success when Deleting Non-existent User [api/admin_handlers.go:126]
- [x] [Review][Patch] Fragile `isSelf` Check in UI [ui/src/app/(dashboard)/settings/page.tsx:1081]
- [x] [Review][Patch] Hardcoded Role Strings ("admin", "viewer") [api/admin_handlers.go:189]
- [x] [Review][Patch] Poor Type Safety in UI (member: any) [ui/src/app/(dashboard)/settings/page.tsx:1080]
- [x] [Review][Patch] Suboptimal UX for Revoke Access (window.confirm) [ui/src/app/(dashboard)/settings/page.tsx:1131]

#### Deferred
- [x] [Review][Defer] Scalability Bottleneck: Lack of Pagination in User Listing [api/db/user.go:580] — deferred: Prioritizing MVP core features.

## Dev Notes

- **Firebase Multi-Tenancy:** Multi-tenancy must be enabled on the Firebase project. The `orgID` in our system corresponds to the Firebase Tenant ID.
- **Custom Claims:** The user's role is stored in custom claims under the `roles` key as an array of strings (e.g., `{"roles": ["admin"]}`). This aligns with the verification logic in `api/auth.go`.
- **Mock Mode:** Local development relies on `MockAuthenticator`. The backend must seamlessly fallback to Firestore-only user management when Firebase Auth is not fully configured, allowing frontend iteration without complex GCP setup.
- **UX Styling:** Follow the Clean Light Mode aesthetic. Use Tailwind CSS and Lucide icons (e.g., `Shield` for Admin, `User` for Viewer, `UserX` for Revoke).

### Project Structure Notes

- Backend changes are localized to `api/db/user.go`, `api/admin_handlers.go`, and `api/routes.go`.
- Frontend changes are localized to `ui/src/app/(dashboard)/settings/` and related components/hooks.

### References

- [Firebase Auth Multi-tenancy Go SDK](https://pkg.go.dev/firebase.google.com/go/v4/auth#TenantManager)
- [Source: api/auth.go#FirebaseAuthAuthenticator]
- [Source: api/admin_handlers.go]
- [Source: ui/src/app/(dashboard)/settings/page.tsx]
- [Source: docs/spec/planning-artifacts/epics.md#Story 11.4]

## Dev Agent Record

### Agent Model Used

Gemini Next

### Debug Log References

### Completion Notes List

### File List
