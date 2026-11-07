# Deferred Work Tracker
This document tracks deferred architecture modifications, postponed refactorings, and secondary issues discovered during active code reviews.

---

## Deferred from: code review of story-4.4 (2026-05-21)
- **Submodule Credentials Delegation Gaps Blocking Local Builds**
  *   **Location**: [ai/core/workspace.go:207](ai/core/workspace.go#L207)
  *   **Detail**: If a repository leverages relative or SSH submodule endpoints, cloning with `RecurseSubmodules` fails due to lack of credential propagation for internal paths. This has no immediate impact as the active workspace contains no submodules, but is a potential future block.

## Deferred from: code review of 9-1-onboarding-tour-state-and-persistence-backend-firestore.md (2026-06-10)
- **Global State Mutation of userStore in Tests**
  *   **Location**: [api/user_handlers_test.go:21](api/user_handlers_test.go#L21)
  *   **Detail**: Handler tests mutate package-level global `userStore`, preventing parallel test execution and risking flakes. Pre-existing pattern.
- **Hardcoded Project ID in user_test.go**
  *   **Location**: [api/db/user_test.go:276](api/db/user_test.go#L276)
  *   **Detail**: Firestore client initialized with hardcoded project ID instead of environment configuration. Pre-existing pattern.

## Deferred from: code review of 9-2-tour-integration-for-newly-created-sections-frontend.md (2026-06-10)
- **Uncontrolled/unbound checkboxes in Ingestion Picker**
  *   **Location**: [OnboardingPanel.tsx:158](ui/src/app/(dashboard)/projects/[id]/workspace/_components/OnboardingPanel.tsx#L158)
  *   **Detail**: Checkboxes for "Ingest" and "Overwrite" are not bound to React state and have no `onChange` handlers, meaning the selection is ignored when calling the completion API. Deferred because OnboardingPanel is a temporary stub.
- **Fragile tour progression using hardcoded step indices**
  *   **Location**: Multiple files
  *   **Detail**: Tour progression triggers use hardcoded numbers (e.g. `currentStep === 17`). If steps in `onboarding.ts` are reordered, this breaks. Pre-existing technical debt.

## Deferred from: code review of 9-3-advanced-navigation-skip-section-and-skip-tour-frontend.md (2026-06-10)
- **Mixed Concerns in `completedSections` State**
  *   **Location**: [onboarding.ts](ui/src/stores/onboarding.ts)
  *   **Detail**: The store mixes concerns by storing both granular `sectionId` strings (e.g. `"catalog-overview"`) and high-level `tourId` strings (e.g. `"catalog"`) in the same `completedSections` array.
- **Inefficient Section Info Calculation**
  *   **Location**: [onboarding.ts](ui/src/stores/onboarding.ts#L863-L902)
  *   **Detail**: `getCurrentSectionInfo` is called on every render of `OnboardingTour` and loops through all steps of the active tour to reconstruct section groupings.
- **Inconsistent State when Skipping Undefined Section**
  *   **Location**: [onboarding.ts](ui/src/stores/onboarding.ts#L695-L728)
  *   **Detail**: If a step has no `sectionId` (falls back to `"Overview"`), "Skip Section" is still available but clicking it doesn't persist completion because `currentSectionId` is undefined.

## Deferred from: code review of 9-4-tour-resiliency-and-action-gating-frontend-backend.md (2026-06-10)
- **Potential state drift if `syncTourStateWithBackend` fails during completion**
  *   **Location**: [ui/src/stores/onboarding.ts:447](ui/src/stores/onboarding.ts#L447)
  *   **Detail**: Client state is updated and saved to localStorage before calling the backend. If the backend call fails, the client thinks the tour is complete but the backend does not, causing drift on reload.
- **Flaky 1500ms timeout for tour trigger/resume**
  *   **Location**: [ui/src/components/shared/OnboardingTour.tsx:113](ui/src/components/shared/OnboardingTour.tsx#L113)
  *   **Detail**: Using `setTimeout` with 1500ms to wait for render is fragile. Can fail on slow devices or feel laggy on fast ones. Pre-existing.
- **Missing loading/error state in `OnboardingTour` project query**
  *   **Location**: [ui/src/components/shared/OnboardingTour.tsx:22](ui/src/components/shared/OnboardingTour.tsx#L22)
  *   **Detail**: `useQuery` for project has no loading/error handling, so if it fails, the tour phase logic silently resolves to null and skips the tour. Pre-existing.
- **Hardcoded tour types violating DRY**
  *   **Location**: [ui/src/stores/onboarding.ts](ui/src/stores/onboarding.ts)
  *   **Detail**: Tour keys are repeated in multiple places. It should be refactored to use constants or types. Pre-existing.
- **Unvalidated API response in `loadTourStatusFromBackend`**
  *   **Location**: [ui/src/stores/onboarding.ts:502](ui/src/stores/onboarding.ts#L502)
  *   **Detail**: The API response is parsed and used directly without validation, which could cause crashes if the backend response structure changes. Pre-existing.
- **Fragile grouping for undefined section IDs in `getCurrentSectionInfo`**
  *   **Location**: [ui/src/stores/onboarding.ts:646](ui/src/stores/onboarding.ts#L646)
  *   **Detail**: Steps with `undefined` section IDs are grouped under `"Overview"`. If adjacent steps have different titles but both have `undefined` section ID, they are incorrectly grouped together. Pre-existing.
- **Disabling type safety in tests with `// @ts-nocheck`**
  *   **Location**: Test files
  *   **Detail**: Disabling TypeScript in tests can hide API signature changes and typing errors. Pre-existing.

## Deferred from: code review of 9-5-context-aware-auto-start-and-triggers-frontend.md (2026-06-10)
- **Duplicate Project Details Query**
  *   **Location**: [ui/src/components/shared/OnboardingTour.tsx:24](ui/src/components/shared/OnboardingTour.tsx#L24)
  *   **Detail**: Duplicate `useQuery` block for project details in `OnboardingTour.tsx` and `TopBar.tsx`. Pre-existing pattern.
- **Hardcoded 1500ms Timeout for Rendering**
  *   **Location**: [ui/src/components/shared/OnboardingTour.tsx:40](ui/src/components/shared/OnboardingTour.tsx#L40)
  *   **Detail**: Using `setTimeout` to wait for render before starting a tour. Pre-existing.
- **Too Many Dependencies in auto-trigger useEffect**
  *   **Location**: [ui/src/components/shared/OnboardingTour.tsx:40](ui/src/components/shared/OnboardingTour.tsx#L40)
  *   **Detail**: 13 dependencies in `useEffect` can lead to infinite render loops or erratic behavior. Pre-existing.
- **Inconsistent Tour Completion Check ignoring workspace_onboarding**
  *   **Location**: [ui/src/stores/onboarding.ts:102](ui/src/stores/onboarding.ts#L102)
  *   **Detail**: `getInitialTourStatus` completion check ignores `workspace_onboarding` tour status. Pre-existing.
- **Inefficient Grouping of Steps in getCurrentSectionInfo**
  *   **Location**: [ui/src/stores/onboarding.ts:630](ui/src/stores/onboarding.ts#L630)
  *   **Detail**: Grouping calculation runs on every render instead of being memoized. Pre-existing.
- **Error Toast Spam on Query Retries**
  *   **Location**: [ui/src/components/shared/OnboardingTour.tsx:40](ui/src/components/shared/OnboardingTour.tsx#L40)
  *   **Detail**: `toast.error` triggers on every query error state transition, causing multiple toasts if queries retry. Pre-existing.

## Deferred from: code review of 1-5-cancel-chat-message.md (2026-06-13)
- **Missing enforcement of 1-second termination limit for GenAI inference and running tools**
  *   **Location**: [api/chat.go:320](api/chat.go#L320)
  *   **Detail**: When cancellation occurs, HTTP request context is cancelled on API tier, but remote inference service termination limits depend on downstream Genkit/Antigravity transport disconnect propagation.

## Deferred from: code review of 10-2-organization-aware-non-administrative-data-plane-apis.md (2026-06-18)
- **Likely Copy-Paste Bug in Route Registration (Approve handler)**
  *   **Location**: [api/routes.go:97](api/routes.go#L97)
  *   **Detail**: The route `/api/v1/projects/{id}/git/approve` is registered using `gitCommitHandler` instead of an approve-specific handler. This is a suspicious reuse of the commit handler for an approval action. Pre-existing.

## Deferred from: code review of 10-3-organization-provisioning-and-onboarding-administrative-api.md (2026-06-18)
- **System Administrator cannot select organization due to missing role bypass**
  *   **Location**: [api/auth.go:381](api/auth.go#L381)
  *   **Detail**: In `selectOrganizationHandler`, there is no bypass for `admin` role. Since admins are not automatically members of new orgs, they can't select them. This is a pre-existing issue.

## Deferred from: code review of 10-4-organization-environment-configuration-and-user-management.md (2026-06-18)
- **Missing Pagination on List Users**
  *   **Location**: [api/db/user.go:789](api/db/user.go#L789)
  *   **Detail**: `ListUsers` retrieves all organization users at once using `GetAll()`. This does not scale and should be paginated.

## Deferred from: code review of 11-3-system-administration-organization-provisioning-screen-system-admin-portal.md (2026-06-19)
- **Insecure Storage of Sensitive Tokens in LocalStorage**
  *   **Location**: [ui/src/stores/auth.ts:169](ui/src/stores/auth.ts#L169)
  *   **Detail**: Storing sensitive session tokens in `localStorage` makes them vulnerable to XSS. They should be stored in secure, `HttpOnly` cookies. Pre-existing architectural pattern.
- **Silent Failures in Rollback Logic**
  *   **Location**: [api/organization_handlers.go:84](api/organization_handlers.go#L84)
  *   **Detail**: Rollback failures during organization provisioning are only logged, potentially leaving orphaned resources in Firestore or Firebase Auth without alerting or automatic retry.

## Deferred from: code review of 11-4-organization-settings-member-directory-and-user-management-screen-org-admin-portal.md (2026-06-19)
- **Scalability Bottleneck: Lack of Pagination in User Listing**
  *   **Location**: [api/db/user.go:580](api/db/user.go#L580)
  *   **Detail**: The `ListUsers` API fetches all users from Firebase Tenant and Firestore into memory. Deferred to prioritize MVP core features for story 11.4; scalability will be addressed in a future performance cycle.

## Deferred from: code review of 11-5-organization-settings-invite-member-modal-and-invitation-flow.md (2026-06-19)
- **Missing Production Mailer**
  *   **Location**: [api/mail.go:334](api/mail.go#L334)
  *   **Detail**: Only `LogMailer` is implemented. Pluggable interface exists but real implementation is deferred.
- **Sensitive Tokens in Logs**
  *   **Location**: [api/mail.go:325-329](api/mail.go#L325-L329)
  *   **Detail**: `LogMailer` logs full link with tokens. Deferred until production mailer is implemented and log mailer is disabled in prod.
- **Global State Mutation in Tests**
  *   **Location**: [api/admin_handlers_test.go:60-68](api/admin_handlers_test.go#L60-L68)
  *   **Detail**: Tests mutate global variables, preventing parallel execution. Refactoring to DI is deferred.
- **Hardcoded Roles in Frontend**
  *   **Location**: [ui/src/components/shared/InviteMemberModal.tsx:1099-1101](ui/src/components/shared/InviteMemberModal.tsx#L1099-L1101)
  *   **Detail**: Roles are hardcoded in the dropdown. Moving to config/constants is deferred.

## Deferred from: code review of 11-6-organization-settings-roles-and-permissions-configuration-screen-org-admin-portal.md (2026-06-19)
- **Weak JWT Secret Fallback in Production**
  *   **Location**: [api/auth.go:325-327](api/auth.go#L325-L327)
  *   **Detail**: Silent fallback to a hardcoded development secret key if `JWT_SECRET` is missing. Pre-existing architectural pattern.
- **Stored XSS via Custom Role Description**
  *   **Location**: [api/role_handlers.go:954-959](api/role_handlers.go#L954-L959)
  *   **Detail**: Role description is stored without sanitization. React escapes by default, but it's a potential risk if rendering methods change. Pre-existing pattern.
- **Stale Permissions in Firebase Custom Claims on Role Update**
  *   **Location**: [api/db/user.go:721, 735](api/db/user.go#L721)
  *   **Detail**: Changing a custom role's permissions doesn't update existing users' claims until they are re-assigned. Out of scope for this story (no role edit feature).

## Deferred from: code review of 11-8-parent-project-super-admin-authentication-and-global-resource-authorization.md (2026-06-19)
- **Misleading "Super Admin" Simulation in Admin Handlers Test**
  *   **Location**: [api/admin_handlers_test.go:470](api/admin_handlers_test.go#L470)
  *   **Detail**: Test attempts to simulate a Super Admin caller by setting the subject to `"super-admin"` but fails to set `IsSuperAdmin: true` in the claims. Pre-existing.
- **Code Cleanliness and decoupling issues in Admin Handlers Tests and Middleware**
  *   **Location**: [api/admin_handlers_test.go:79](api/admin_handlers_test.go#L79)
  *   **Detail**: Pre-existing patterns of: initializing `AdminHandler` with `nil` stores and then manually assigning them later, suppressing `json.Marshal` errors using the blank identifier, and hardcoded cryptographic fallback secrets.

## Deferred from: code review of 11-9-multi-tenant-identity-verification-and-fail-safe-security-context-propagation.md (2026-06-19)
- **Redundant appConfig Nil-Pointer Checks**
  *   **Location**: [api/middleware.go](api/middleware.go)
  *   **Detail**: The middleware contains numerous `if appConfig != nil` checks on the request hot path. If the configuration fails to initialize, the application should panic at startup rather than serving requests in an uninitialized state. These redundant checks increase cognitive load. Deferred, pre-existing pattern to prioritize critical security patches.

## Deferred from: code review of 9-6-seeded-showcase-project-foundry-showcase-frontend-backend.md (2026-06-25)
- **Coupled Storage Dependencies (Package-Level Global Variables)**
  *   **Location**: [api/showcase_seeder.go](api/showcase_seeder.go)
  *   **Detail**: The showcase seeder relies heavily on package-level global variables (`store`, `Store`, `sessionStore`, `artifactStore`) for database and message persistence. This tight coupling to global state is a pre-existing codebase-wide pattern that limits testability, but refactoring it to dependency injection is deferred as it is out of scope for this story.

## Deferred from: code review of 4-2-tabbed-workspace-explorer-artifacts-and-file-explorer-views.md (2026-06-25)
- **State Reset on Parent Unmount**
  *   **Location**: [ui/src/components/workspace/FileExplorer.tsx](ui/src/components/workspace/FileExplorer.tsx)
  *   **Detail**: The folder expansion state (`isOpen`) in `FileExplorer.tsx` is stored locally within each `FileNode` component using `useState`. If the workspace view or the panel itself is ever unmounted during navigation, all expansion states are lost. Hoisting this state to a context or a parent explorer state would make the state truly persistent during the active session.

## Deferred from: code review of 10-7-super-admin-autojoin-configuration-and-auto-provisioning.md (2026-06-26)
- [x] **Performance: Frequent Transactional Reconciliation on List API**
  *   **Location**: [api/auth_handlers.go:215](api/auth_handlers.go#L215)
  *   **Detail**: `ReconcileAutojoinRules` is invoked on every OIDC organization list retrieval, which previously triggered a Firestore query and transaction.
  *   **Resolution**: Implemented a thread-safe read-through memory cache on `FirestoreAutojoinStore` with auto-invalidation on Create, Update, and Delete. Active domain queries now read from memory, reducing Firestore reads to zero during active user handshakes.
- [x] **Hardcoded Platform Roles in Frontend**
  *   **Location**: [ui/src/app/admin/autojoin/page.tsx:972](ui/src/app/admin/autojoin/page.tsx#L972)
  *   **Detail**: Predefined platform roles and descriptions were hardcoded in the client, risking synchronization drift.
  *   **Resolution**: Exposed a secure global endpoint `GET /api/v1/admin/roles` to retrieve predefined platform roles directly from the Go backend, and updated the Next.js page to fetch and render roles dynamically.
- [x] **Poor UX: Use of Native Browser Dialogs**
  *   **Location**: [ui/src/app/admin/autojoin/page.tsx:1058](ui/src/app/admin/autojoin/page.tsx#L1058)
  *   **Detail**: Native window `alert()` and `confirm()` prompts were used for errors and deletion confirmation.
  *   **Resolution**: Replaced native calls with dynamic page/modal error banners and a gorgeous custom React-based Delete Confirmation Modal overlay.

### DW-1: Firestore collection group queries in DiscoverHangingProjects load all documents into memory via GetAll() without pagination.
origin: spec-deferred 6254a5d89c17
location: api/jobs/hanging_cleanup.go:313
source_spec: `12-9-modular-nfs-storage-audit-and-orphaned-workspace-cleanup-jobs.md`
severity: low
reason: api/jobs/hanging_cleanup.go calls firestoreClient.CollectionGroup("projects").Documents(ctx).GetAll() and firestoreClient.CollectionGroup("user_project").Documents(ctx).GetAll(), matching pre-existing repository patterns.
status: open

### DW-2: Follow-up review still recommended for 12-9-modular-nfs-storage-audit-and-orphaned-workspace-cleanup-jobs after the damping cap was spent
origin: review-budget-followup
location: n/a
source_spec: `12-9-modular-nfs-storage-audit-and-orphaned-workspace-cleanup-jobs.md`
severity: low
reason: The follow-up-review damping cap (limits.max_followup_reviews = 1) was spent with the story finalized (status: done, verify green) while the review pass still recommended an independent follow-up. The work was committed by bmad-loop run 20260822-182136-6e5b; this entry preserves the lingering recommendation for a deliberate later review.
status: open

### DW-3: Follow-up review still recommended for 18-2-chat-file-multimodal-image-upload-ui-frontend after the damping cap was spent
origin: review-budget-followup
location: n/a
source_spec: `18-2-chat-file-multimodal-image-upload-ui-frontend.md`
severity: low
reason: The follow-up-review damping cap (limits.max_followup_reviews = 1) was spent with the story finalized (status: done, verify green) while the review pass still recommended an independent follow-up. The work was committed by bmad-loop run 20260827-023736-11bf; this entry preserves the lingering recommendation for a deliberate later review.
status: closed

### DW-4: Follow-up review still recommended for 18-4-file-explorer-folder-actions-project-tree-uploads-full-stack after the damping cap was spent
origin: review-budget-followup
location: n/a
source_spec: `18-4-file-explorer-folder-actions-project-tree-uploads-full-stack.md`
severity: low
reason: The follow-up-review damping cap (limits.max_followup_reviews = 1) was spent with the story finalized (status: done, verify green) while the review pass still recommended an independent follow-up. The work was committed by bmad-loop run 20260827-023736-11bf; this entry preserves the lingering recommendation for a deliberate later review.
status: open
