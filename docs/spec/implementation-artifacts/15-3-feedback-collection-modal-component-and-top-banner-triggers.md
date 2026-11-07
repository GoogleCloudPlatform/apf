# Story 15.3: Feedback Collection Modal Component & Top Banner Triggers

Status: done

## Story

As a user,
I want to click on "Learn More" or "File a Bug" links in the top notice banner to open a lightweight feedback modal dialog,
So that I can easily submit feedback, report issues, or provide product suggestions without leaving my active workspace page.

## Acceptance Criteria

1. **Given** a user interacting with the top banner navigation/notice bar
   **When** the user clicks on the "Learn More" or "File a Bug" buttons/links
   **Then** a lightweight feedback collection modal dialog opens over the current view with the appropriate feedback type pre-selected (`"general"` for Learn More, `"bug"` for File a Bug).

2. **Given** the feedback collection modal dialog is open
   **When** rendered to the user
   **Then** it displays minimal, clean UI elements (feedback type selector/dropdown for `bug`, `feature_request`, `general`; subject input field; feedback detail text area; pre-filled or editable user email field; and Submit / Cancel action buttons).

3. **Given** a user submitting feedback
   **When** the user fills out the required feedback details (type, subject, message) and clicks "Submit"
   **Then** the feedback payload is validated and submitted via `POST /api/v1/feedback` to be stored in the root-level `feedback` Firestore collection, a success toast notification is displayed, and the modal dialog closes gracefully.

4. **Given** a user deciding to cancel or dismiss the feedback modal
   **When** the user clicks the "Cancel" button, close icon (X), or clicks outside the modal backdrop
   **Then** the modal closes without submitting data or losing active workspace state.

## Tasks / Subtasks

- [x] Task 1: Backend Data Model & Firestore Feedback Store (AC: #3)
  - [x] Define `Feedback` struct (`id`, `type`, `subject`, `message`, `userEmail`, `organizationId`, `projectId`, `createdAt`) in `api/db/feedback.go`.
  - [x] Implement `FeedbackStore` interface and `FirestoreFeedbackStore` struct supporting `CreateFeedback(ctx, feedback)` persisting to root `feedback` Firestore collection.
  - [x] Implement `MockFeedbackStore` for unit testing.
- [x] Task 2: Backend REST API Endpoint (`POST /api/v1/feedback`) (AC: #3)
  - [x] Create `api/feedback_handlers.go` containing `CreateFeedbackHandler`.
  - [x] Validate payload fields: `type` (must be `"bug"`, `"feature_request"`, or `"general"`), `subject` (non-empty, max 200 chars), `message` (non-empty, max 4000 chars).
  - [x] Extract `userEmail` and `organizationId` from request context / auth claims, allowing client payload overrides if provided.
  - [x] Register `POST /api/v1/feedback` and preflight endpoints in `api/routes.go` with standard authentication middleware.
  - [x] Return 201 Created JSON payload `{ "id": "...", "status": "success", "message": "Feedback submitted successfully" }` on success, or 400 Bad Request on invalid payload.
- [x] Task 3: Frontend Feedback Modal Component (`FeedbackModal.tsx`) (AC: #1, #2, #4)
  - [x] Create `ui/src/components/shared/FeedbackModal.tsx` dialog component.
  - [x] Implement form inputs: Feedback Type select (`bug` | `feature_request` | `general`), Subject text input, Message textarea, and User Email input (pre-filled from `useAuthStore`).
  - [x] Add Form validation handling (required fields, clear error state display).
  - [x] Add backdrop click, ESC key, close icon (X), and Cancel button handlers for modal dismissal without data loss.
- [x] Task 4: Integration with TopBar Banner Triggers (AC: #1)
  - [x] Update `ui/src/components/shared/TopBar.tsx` banner "Learn More" and "File a Bug" link click handlers to trigger `FeedbackModal`.
  - [x] Pass initial feedback type (`"general"` when clicking "Learn More", `"bug"` when clicking "File a Bug") to modal state.
  - [x] Render `FeedbackModal` when open state is active.
- [x] Task 5: API Client Integration & Toast Feedback (AC: #3)
  - [x] Add `submitFeedback(payload)` method to `ui/src/lib/api.ts`.
  - [x] Handle submission loading state (disabled buttons, spinner).
  - [x] Display success toast notification upon successful submission and close modal.
- [x] Task 6: Unit & Integration Test Suite (AC: #1, #2, #3, #4)
  - [x] Write Go unit tests in `api/feedback_handlers_test.go` verifying payload validation, Firestore persistence, error codes, and HTTP handler behavior.
  - [x] Write Vitest frontend unit tests in `ui/src/components/shared/FeedbackModal.test.tsx` and `ui/src/components/shared/TopBar.test.tsx` verifying render, trigger opening, modal closing, form validation, and submit actions.

### Review Findings

- [x] [Review][Patch] Include `id` in Firestore Document Persistence & Model Tag [`api/db/feedback.go`:18]
- [x] [Review][Patch] Multi-byte Character Validation with `utf8.RuneCountInString` [`api/feedback_handlers.go`:85]
- [x] [Review][Patch] Scope Escape Key Listener to Open State in `FeedbackModal` [`ui/src/components/shared/FeedbackModal.tsx`:42]
- [x] [Review][Patch] Enforce Unbounded Request Body Limit with `http.MaxBytesReader` [`api/feedback_handlers.go`:34]
- [x] [Review][Patch] Non-POST 405 Response Compliance (`Allow` header & JSON error format) [`api/feedback_handlers.go`:29]
- [x] [Review][Patch] Handler Nil Guarding in `CreateFeedbackHandler` [`api/feedback_handlers.go`:100]
- [x] [Review][Patch] Defensive Copying in `MockFeedbackStore` [`api/db/feedback.go`:120]
- [x] [Review][Patch] Trim Whitespace for ProjectID and OrgID [`api/feedback_handlers.go`:121]

## Dev Notes

### Architecture & Data Patterns

- **Firestore Schema**:
  - Collection: `feedback` (root-level)
  - Document ID: auto-generated UUID / Firestore document ID
  - Fields:
    - `id`: string (UUID)
    - `type`: string (`"bug"` | `"feature_request"` | `"general"`)
    - `subject`: string (short summary)
    - `message`: string (detailed description)
    - `userEmail`: string (submitter email address)
    - `organizationId`: string (optional/active tenant org ID)
    - `projectId`: string (optional/active project ID if submitted from workspace)
    - `createdAt`: timestamp (submission RFC3339 datetime)

- **API Endpoint Specification**:
  - Route: `POST /api/v1/feedback`
  - Request Body:
    ```json
    {
      "type": "bug",
      "subject": "Canvas rendering lag on zoom",
      "message": "When zooming in past 150%, canvas nodes take over 2 seconds to re-render.",
      "userEmail": "user@example.com",
      "organizationId": "org-123",
      "projectId": "proj-456"
    }
    ```
  - Response 201 Created:
    ```json
    {
      "id": "fb-uuid-12345",
      "status": "success",
      "message": "Feedback submitted successfully"
    }
    ```
  - Response 400 Bad Request:
    ```json
    {
      "code": "INVALID_FEEDBACK_PAYLOAD",
      "error": "Subject and message are required fields"
    }
    ```

### Project Structure & Component Locations

- **Backend API Tier**:
  - `api/db/feedback.go`: Firestore `FeedbackStore` interface, `FirestoreFeedbackStore`, `MockFeedbackStore`, and data structs.
  - `api/feedback_handlers.go`: `CreateFeedbackHandler` logic and payload validation.
  - `api/feedback_handlers_test.go`: Unit tests for feedback handlers.
  - `api/routes.go`: Route registration for `/api/v1/feedback`.
- **Frontend UI Tier**:
  - `ui/src/components/shared/FeedbackModal.tsx`: Modal component for entering and submitting feedback.
  - `ui/src/components/shared/TopBar.tsx`: Notice banner triggers ("Learn More", "File a Bug").
  - `ui/src/lib/api.ts`: API client helper for feedback submission.
  - `ui/src/lib/types.ts`: TypeScript interfaces for `FeedbackPayload` and `FeedbackResponse`.

### References

- Epics Document: [epics.md#Story-15.3](docs/spec/planning-artifacts/epics.md#L1467-L1495)
- Architecture Document: [architecture.md](docs/spec/planning-artifacts/architecture.md)
- Existing TopBar: [TopBar.tsx](ui/src/components/shared/TopBar.tsx#L80-L97)
- Previous Story: [15-2-application-level-feature-flags-and-on-demand-feature-document-loading-api.md](docs/spec/implementation-artifacts/15-2-application-level-feature-flags-and-on-demand-feature-document-loading-api.md)

## Dev Agent Record

### Agent Model Used

- Gemini / Jetski Coding Agent

### Debug Log References

- Go backend tests run clean (`go test ./...` passed 100%).
- Vitest frontend tests run clean (`npx vitest run src/components/shared` passed 100%, 31 tests passed).

### Completion Notes List

- Implemented `Feedback` model and `FirestoreFeedbackStore`/`MockFeedbackStore` in `api/db/feedback.go`.
- Implemented `CreateFeedbackHandler` in `api/feedback_handlers.go` with payload validation and context defaults extraction.
- Registered `POST /api/v1/feedback` and preflight endpoints in `api/routes.go` and wired `feedbackHandler` in `api/main.go`.
- Created `FeedbackModal.tsx` component with type selector, inputs, form validation, backdrop/ESC dismissal, and submit handling.
- Integrated `FeedbackModal` into `TopBar.tsx` connecting banner links ("Learn More" -> "general", "file a bug" -> "bug").
- Added `submitFeedback` helper in `ui/src/lib/api.ts` and type definitions in `ui/src/lib/types.ts`.
- Added comprehensive unit test suites in `api/db/feedback_test.go`, `api/feedback_handlers_test.go`, `ui/src/components/shared/FeedbackModal.test.tsx`, and updated `ui/src/components/shared/TopBar.test.tsx`.

### File List

- `api/db/feedback.go`
- `api/db/feedback_test.go`
- `api/feedback_handlers.go`
- `api/feedback_handlers_test.go`
- `api/routes.go`
- `api/routes_test.go`
- `api/main.go`
- `ui/src/lib/types.ts`
- `ui/src/lib/api.ts`
- `ui/src/components/shared/FeedbackModal.tsx`
- `ui/src/components/shared/FeedbackModal.test.tsx`
- `ui/src/components/shared/TopBar.tsx`
- `ui/src/components/shared/TopBar.test.tsx`
- `docs/spec/implementation-artifacts/15-3-feedback-collection-modal-component-and-top-banner-triggers.md`
- `docs/spec/implementation-artifacts/sprint-status.yaml`

### Change Log

- Date: 2026-08-02 — Completed implementation of Story 15.3: Feedback Collection Modal Component & Top Banner Triggers. All tests passing and ready for code review.

