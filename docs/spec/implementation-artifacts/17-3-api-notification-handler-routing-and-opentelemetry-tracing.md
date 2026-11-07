# Story 17.3: API Notification Handler, Routing & OpenTelemetry Tracing

Status: ready-for-dev

## Story

As an API Client and Developer,
I want a RESTful endpoint `POST /api/v1/notifications/email` and administrative discovery endpoint `GET /api/v1/notifications/providers`,
so that authenticated organization members can dispatch email notifications with strict payload validation, context timeout management, fail-fast error handling, and OpenTelemetry observability.

## User Request Context & Specifics

1. Create `NotificationHandler` in `api/notification_handlers.go`.
2. Strictly require constructor dependency injection of `EmailNotificationProvider` (`NewNotificationHandler(provider, timeout)`).
3. Enforce 1MB request body limit via `http.MaxBytesReader(w, r.Body, 1<<20)`.
4. Validate `toEmail` (valid RFC 5322 regex), `subject`, and `htmlContent` (**mandatory**).
5. Route through global default provider without allowing client-specified provider overrides.
6. Create OpenTelemetry spans (`notifications.send_email`) with attributes and status tracking.
7. Implement `GET /api/v1/notifications/providers` restricted to Org Admin, Org Owner, and SuperAdmin roles.
8. Wire routes in `api/routes.go` and initialize handler in `api/main.go`.
9. Document endpoints in `api/openapi.yaml`.

## Verified Design Architecture & Elicited Decisions

- **Strict Constructor Dependency Injection**: `NewNotificationHandler(provider notifications.EmailNotificationProvider, timeout time.Duration)` accepts the provider explicitly.
- **Fail-Fast Error Handling**: Upstream provider failures return HTTP `502 Bad Gateway` (or `500 Internal Server Error`) immediately.
- **Context Timeout & Cancellation**: Timeouts are derived from `r.Context()` using `context.WithTimeout(r.Context(), h.timeout)`.
- **OpenTelemetry Observability**: Emits span `notifications.send_email` with attributes: `notification.provider`, `notification.recipient_domain`, `notification.subject_length`, and `notification.status`.
- **Role-Based Access Control**: `POST /api/v1/notifications/email` is accessible to all authenticated org members; `GET /api/v1/notifications/providers` is restricted to Org Admins, Org Owners, and SuperAdmins.

## Acceptance Criteria

1. **Given** the `NotificationHandler` in `api/notification_handlers.go`, **When** instantiated, **Then** its constructor MUST strictly require dependency injection: `NewNotificationHandler(provider notifications.EmailNotificationProvider, timeout time.Duration)`.
2. **Given** an incoming request to `POST /api/v1/notifications/email`, **When** reading the request body, **Then** the handler MUST enforce a 1MB payload size limit via `http.MaxBytesReader(w, r.Body, 1<<20)`, returning `400 Bad Request` if the body exceeds 1MB.
3. **Given** request payload validation, **When** validating fields, **Then** the handler MUST enforce:
   - `toEmail`: Required valid RFC 5322 email string (single recipient only).
   - `subject`: Required non-empty string.
   - `htmlContent`: **Mandatory** non-empty HTML string.
   - Rejecting or returning `400 Bad Request` with structured error messages for missing or invalid fields.
4. **Given** request execution, **When** dispatching via the injected provider, **Then** the handler MUST strictly use the configured provider (disallowing client-specified overrides) and derive a timeout context via `context.WithTimeout(r.Context(), h.timeout)`.
5. **Given** an active notification dispatch, **When** processing the request, **Then** the handler MUST create an OpenTelemetry span `notifications.send_email` recording attributes `notification.provider`, `notification.recipient_domain`, `notification.subject_length`, and `notification.status`.
6. **Given** upstream provider errors or timeouts, **When** dispatch fails, **Then** the handler MUST fail fast, record the error in the active OTel span, and return `502 Bad Gateway` (or `500 Internal Server Error`) with structured JSON error details.
7. **Given** a successful or duplicate dispatch, **When** provider returns `SendResult`, **Then** the handler MUST return `200 OK` with JSON `{ "status": "SUCCESS" | "DUPLICATE", "message": "Email notification processed", "result": result }`.
8. **Given** a request to `GET /api/v1/notifications/providers`, **When** the caller is authenticated, **Then** the handler MUST verify that the user is an Org Admin, Org Owner, or SuperAdmin (`RoleOrgAdmin`, `RoleOrgOwner`, SuperAdmin), returning `403 Forbidden` for regular members, and `200 OK` with active provider name and registered providers list for authorized roles.
9. **Given** route registration in `api/routes.go`, **When** registering routes, **Then** `POST /api/v1/notifications/email` and `GET /api/v1/notifications/providers` MUST be wrapped with `authMiddleware` and `orgScopeMiddleware`, with CORS preflight `OPTIONS` registered.
10. **Given** API documentation in `api/openapi.yaml`, **When** inspecting endpoint definitions, **Then** `/api/v1/notifications/email` and `/api/v1/notifications/providers` MUST be fully documented with request/response schemas and authentication requirements.
11. **Given** unit tests in `api/notification_handlers_test.go`, **When** running `go test -v -run TestNotificationHandler ./api/...`, **Then** 100% of handler tests (validation, auth/orgScope gating, provider dispatch, error handling) MUST pass cleanly.

## Tasks / Subtasks

- [ ] Task 1: Notification Handler Implementation (AC: #1, #2, #3, #4, #5, #6, #7, #8)
  - [ ] Implement `NotificationHandler` in `api/notification_handlers.go`.
  - [ ] Implement `SendEmailNotification` with 1MB body reader limit, mandatory `htmlContent`, regex validation, OTel spans, and error handling.
  - [ ] Implement `ListProviders` with role check (Org Admin, Org Owner, SuperAdmin).
- [ ] Task 2: Configuration and Server Boot Wiring (AC: #1, #9)
  - [ ] Add `NOTIFICATIONS_*` fields to `api/core/configuration.go`.
  - [ ] Instantiate provider in `api/main.go` and inject into `NewNotificationHandler`.
  - [ ] Register routes in `api/routes.go`.
- [ ] Task 3: OpenAPI Documentation (AC: #10)
  - [ ] Add `/api/v1/notifications/email` and `/api/v1/notifications/providers` paths and schemas to `api/openapi.yaml`.
- [ ] Task 4: Handler Unit Test Suite (AC: #11)
  - [ ] Create `api/notification_handlers_test.go` verifying 200 OK with mock provider, 400 Bad Request on missing fields/large body, 502 Bad Gateway on provider failure, and 403 Forbidden on provider listing.
