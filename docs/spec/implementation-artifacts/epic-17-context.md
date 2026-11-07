# Epic 17 Context: Multi-Provider Email Notification Platform & Custom Notification Provider (CNP) Integration

<!-- Generated from planning artifacts. Regenerate with compile-epic-context if planning docs change. -->

## Goal

Establish a modular, provider-agnostic notification adapter subsystem in `api/notifications/` supporting Custom Notification Provider (CNP), local Console, and Mock providers, enabling secure, authenticated, and observable email notification dispatch with automated infrastructure provisioning.

## Stories

- Story 17.1: Generic Email Notification Adapter Subsystem & Provider Registry
- Story 17.2: Custom Notification Provider (CNP) Provider Implementation & AuthN/AuthZ Impersonation
- Story 17.3: API Notification Handler, Routing & OpenTelemetry Tracing
- Story 17.4: Notifications Infrastructure Provisioning, IAM Roles & Cloud Deploy Pipeline

## Requirements & Constraints

- Generic adapter subsystem with `EmailNotificationProvider` interface, thread-safe registry (`sync.RWMutex`), and normalized request/response structs (`EmailNotification`, `SendResult`).
- CNP Admit API provider supporting direct API-rendered HTML/text messages, Google ADC IAM token creator impersonation cached in memory via `oauth2.ReuseTokenSource` for `NOTIFICATIONS_SENDER_SERVICE_ACCOUNT`, UUID v4 `request_id` deduplication with idempotent `ALREADY_EXISTS` (409) handling, and offline simulation (`NOTIFICATIONS_ENVIRONMENT=local`).
- API endpoint `POST /api/v1/notifications/email` with 1MB body limit (`http.MaxBytesReader`), mandatory `htmlContent` validation, single-recipient atomic dispatch, and fail-fast 502/500 errors.
- Administration endpoint `GET /api/v1/notifications/providers` restricted to Org Admin, Org Owner, and SuperAdmin roles.
- OpenTelemetry span `notifications.send_email` recording provider, recipient domain, subject length, and status.
- Terraform provisioning of dedicated `google_service_account.notifications_sender` and IAM Token Creator bindings to compute SA.

## Technical Decisions

- **Strict Dependency Injection**: All handlers and consumers receive `EmailNotificationProvider` explicitly via constructor parameters.
- **Explicit Lifecycle Registration**: Providers are initialized and registered via `notifications.InitializeProviders(cfg)` during server startup in `main.go`.
- **Zero Firestore Persistence**: Dispatches are in-memory operations logged with structured `slog` and traced with OpenTelemetry.
- **Legacy Cleanup**: Completely eliminate SendGrid SDK and replace `core.EmailService` with an adapter wrapping the default notification provider.

## Cross-Story Dependencies

- **Story 17.1** provides the base `EmailNotificationProvider` interface, registry, `ConsoleEmailProvider`, and `MockEmailProvider`.
- **Story 17.2** implements `CNPEmailProvider` on top of Story 17.1's abstraction.
- **Story 17.3** builds the HTTP handler and routing injecting the provider from Story 17.1 & 17.2.
- **Story 17.4** defines Terraform IAM bindings and deploy parameters for cloud environments.
