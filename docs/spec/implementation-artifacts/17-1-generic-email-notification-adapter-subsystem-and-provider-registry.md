---
status: done
baseline_revision: 9504a8cca941676a46ccaffffc98b39ace10b8c5
followup_review_recommended: false
context:
  - docs/spec/planning-artifacts/epics.md
  - docs/spec/implementation-artifacts/epic-17-context.md
  - api/core/email_service.go
  - api/core/configuration.go
---

# Story 17.1: Generic Email Notification Adapter Subsystem & Provider Registry

## Story

As a Developer and System Architect,
I want a generic email notification adapter interface (`EmailNotificationProvider`) and thread-safe registry in `api/notifications/`,
so that the platform can swap out and route email notifications to different underlying transports (CNP, Console, Mock) cleanly without coupling high-level business logic to concrete provider implementations.

## User Request Context & Specifics

1. Establish a modular, provider-agnostic notification adapter subsystem under `api/notifications/`.
2. Implement normalized `EmailNotification` request and `SendResult` response structures.
3. Provide a thread-safe provider registry supporting registration, dynamic provider lookup, setting/retrieving the default provider, and a `ResetRegistry()` helper for test isolation.
4. Implement `ConsoleEmailProvider` for local structured logging (`slog`) with metadata attributes and safe payload truncation.
5. Implement `MockEmailProvider` with sent message history tracking and simulated error injection for test suites.
6. Connect `core.EmailService` and `core.EmailServiceInstance` as an adapter on top of the provider registry for backwards compatibility.
7. Remove legacy SendGrid implementation (`SendGridEmailService`) and dependency (`github.com/sendgrid/sendgrid-go`) cleanly from `api/core/email_service.go`.

## Verified Design Architecture & Elicited Decisions

- **Strict Dependency Injection**: All handlers and consumers must receive `EmailNotificationProvider` via constructor dependency injection.
- **Explicit Lifecycle Registration**: Providers are explicitly registered and the default provider configured via `notifications.InitializeProviders(cfg)` during server startup in `main.go`.
- **Name Normalization**: Provider names are automatically normalized to lower-case (`strings.ToLower(strings.TrimSpace(name))`) on registration and lookup (`"cnp"`, `"console"`, `"mock"`).
- **Thread-Safe Registry & Error Handling**: The registry uses `sync.RWMutex`. `GetProvider(name)` returns an explicit `ErrProviderNotFound` error when an unknown provider is queried, and falls back to `ConsoleEmailProvider` as a safe default if no default provider is set. `RegisterProvider` overwrites existing registrations idempotently. `ResetRegistry()` resets registry state between tests.
- **Typed `SendStatus` Enum**: Defines `type SendStatus string` with typed constants:
  - `SendStatusSuccess SendStatus = "SUCCESS"`
  - `SendStatusDuplicate SendStatus = "DUPLICATE"`
  - `SendStatusError SendStatus = "ERROR"`
- **Structured Slog Logging**: `ConsoleEmailProvider` logs structured key-value attributes (`to`, `toName`, `subject`, `requestId`, `htmlContentLength`, `textContentLength`) plus a 200-char preview of `textContent` to prevent console log flooding.
- **Mock Provider Capabilities**: Minimal test spy exposing `SentNotifications() []*EmailNotification`, `Clear()`, and `SetError(err error)`.
- **Zero Firestore Persistence for Dispatch**: Dispatches are lightweight in-memory operations without database writes.
- **Legacy Cleanup**: Completely eliminate `SendGridEmailService` and its Twilio SendGrid SDK dependency.

## Acceptance Criteria

1. **Given** the `api/notifications/` package, **When** defining the notification provider abstraction in `provider.go`, **Then** it MUST declare:
   - `type SendStatus string` with constants `SendStatusSuccess`, `SendStatusDuplicate`, `SendStatusError`.
   - `var ErrProviderNotFound = errors.New("notification provider not found")`.
   - `type EmailNotificationProvider interface` with methods:
     - `Name() string`
     - `Send(ctx context.Context, notif *EmailNotification) (*SendResult, error)`
     - `ValidateConfig() error`
2. **Given** a notification dispatch request, **When** creating an `EmailNotification` struct, **Then** it MUST support normalized fields: `ToEmail` (string), `ToName` (string), `Subject` (string), `HTMLContent` (string), `TextContent` (string), `RequestID` (string), and `Metadata` (map[string]interface{}).
3. **Given** a completed notification dispatch, **When** receiving a `SendResult` struct, **Then** it MUST contain normalized fields: `Provider` (string), `MessageID` (string), `Status` (`SendStatus`), `SenderServiceAccount` (string), `Timestamp` (time.Time), and `Details` (map[string]interface{}).
4. **Given** the provider registry in `api/notifications/provider.go`, **When** registering and resolving providers, **Then** functions `RegisterProvider(provider)`, `GetProvider(name)`, `SetDefaultProvider(provider)`, `GetDefaultProvider()`, `ListRegisteredProviders()`, and `ResetRegistry()` MUST operate thread-safely with `sync.RWMutex`. Provider names MUST be normalized via `strings.ToLower(strings.TrimSpace(name))`. If an unknown provider is queried, `GetProvider(name)` returns `ErrProviderNotFound`. If no default provider is configured, `GetDefaultProvider()` falls back to `ConsoleEmailProvider`.
5. **Given** local development or testing without active cloud credentials, **When** initializing the `ConsoleEmailProvider` (`console_provider.go`), **Then** it MUST log structured key-value attributes (`to`, `toName`, `subject`, `requestId`, `htmlContentLength`, `textContentLength`, `preview`) to `slog.Info` and return a successful `SendResult` with `Status: SendStatusSuccess`.
6. **Given** unit and integration testing environments, **When** initializing `MockEmailProvider` (`mock_provider.go`), **Then** it MUST capture sent notifications in-memory in a thread-safe slice, support inspecting sent messages via `SentNotifications()`, clearing history via `Clear()`, and injecting simulated errors via `SetError(err)`.
7. **Given** legacy email callers using `core.EmailService`, **When** `core.InitializeEmailService()` runs, **Then** `core.EmailServiceInstance` MUST wrap and delegate to the active default `EmailNotificationProvider`, and `SendGridEmailService` / `github.com/sendgrid/sendgrid-go` MUST be removed.
8. **Given** automated unit tests in `api/notifications/provider_test.go`, **When** running `go test -v ./api/notifications/...`, **Then** 100% of registry, console provider, and mock provider unit tests MUST pass cleanly.

## Tasks / Subtasks

- [x] Task 1: Provider Interface, Types & Registry Data Structures (AC: #1, #2, #3, #4)
  - [x] Create `api/notifications/provider.go` with `SendStatus` enum, `EmailNotification`, `SendResult`, `EmailNotificationProvider` interface, `ErrProviderNotFound`.
  - [x] Implement thread-safe registry methods: `RegisterProvider`, `GetProvider`, `SetDefaultProvider`, `GetDefaultProvider`, `ListRegisteredProviders`, `ResetRegistry`.
  - [x] Implement `InitializeProviders(cfg)` helper for application startup.
- [x] Task 2: Console & Mock Providers (AC: #5, #6)
  - [x] Create `api/notifications/console_provider.go` implementing `EmailNotificationProvider` with structured `slog.Info` logging and text preview truncation.
  - [x] Create `api/notifications/mock_provider.go` with thread-safe `SentNotifications()`, `Clear()`, and `SetError(err)` methods.
- [x] Task 3: Core EmailService Cleanup & Adapter Integration (AC: #7)
  - [x] Remove `SendGridEmailService` and SendGrid imports from `api/core/email_service.go`.
  - [x] Update `InitializeEmailService(cfg)` to delegate to `notifications.GetDefaultProvider()`.
- [x] Task 4: Unit Test Suite (AC: #8)
  - [x] Create `api/notifications/provider_test.go` verifying registry concurrency, name normalization, overwrite behavior, fallback defaults, console provider formatting, and mock provider assertions.

## Review Triage Log

### 2026-08-17 — Review pass
- intent_gap: 0
- bad_spec: 0
- patch: 5 (low 5)
- defer: 0
- reject: 9
- addressed_findings:
  - `[low]` `[patch]` Guard against empty or whitespace provider names in `RegisterProvider` and `SetDefaultProvider` (`api/notifications/provider.go`)
  - `[low]` `[patch]` Guard against nil receiver in `ProviderEmailServiceAdapter.SendEmail` (`api/core/email_service.go`)
  - `[low]` `[patch]` Guard against nil receiver across all `MockEmailProvider` methods (`api/notifications/mock_provider.go`)
  - `[low]` `[patch]` Deduplicate provider initialization in `api/main.go`
  - `[low]` `[patch]` Add fallback preview label `(HTML content only)` in `ConsoleEmailProvider` when plain text content is empty (`api/notifications/console_provider.go`)

## Auto Run Result

### Summary of Implemented Change
- Created generic email notification adapter subsystem in `api/notifications/` with `EmailNotificationProvider` interface, normalized `EmailNotification` request and `SendResult` response structures, and typed `SendStatus` enum.
- Implemented thread-safe `sync.RWMutex` provider registry supporting case-insensitive lookups, registration overwriting, dynamic fallback defaults, and `ResetRegistry()` for test isolation.
- Implemented `ConsoleEmailProvider` with structured `slog.Info` output and 200-rune text preview truncation.
- Implemented `MockEmailProvider` with thread-safe notification tracking and simulated error injection.
- Removed legacy SendGrid dependencies (`github.com/sendgrid/sendgrid-go`) from `api/core/email_service.go` and `go.mod`.
- Implemented `ProviderEmailServiceAdapter` in `api/core/email_service.go` wrapping the default notification provider for seamless backwards compatibility with legacy callers.
- Wired startup initialization in `api/main.go` and authored comprehensive unit test suites with 100% pass rate.

### Files Changed
- `api/notifications/provider.go`: Interface definition, typed enums, normalized structs, and thread-safe registry implementation.
- `api/notifications/console_provider.go`: Console provider with structured slog logging and preview truncation.
- `api/notifications/mock_provider.go`: In-memory mock provider with thread-safe inspection and error injection.
- `api/notifications/provider_test.go`: Unit tests for provider registry, console provider, and mock provider.
- `api/core/email_service.go`: Legacy adapter wrapping `notifications.GetDefaultProvider()` and SendGrid cleanup.
- `api/core/email_service_test.go`: Unit tests for legacy email adapter delegation and provider resolution.
- `api/main.go`: Server startup initialization wiring.
- `api/go.mod` & `api/go.sum`: Dependency cleanup removing SendGrid SDK.

### Review Findings Breakdown
- Patches applied: 5 (defensive nil guards, empty name validation, log preview fallback, initialization deduplication)
- Items deferred: 0
- Items rejected: 9 (SendGrid regression out of scope, CNP provider in Story 17.2, handlers in Story 17.3, health check scoping decided in grill-me)

### Follow-up Review Recommendation
- `followup_review_recommended`: `false` (all 5 patched findings were `low` severity defensive refinements).

### Verification
- `go test -v -race ./api/notifications/... ./api/core/...`: All tests passed cleanly with race detector enabled.
- `go test ./api/...`: 100% pass across all packages in the API workspace.

### Residual Risks
- None. CNP provider implementation and direct API handler injection are cleanly scheduled for Stories 17.2 and 17.3.
