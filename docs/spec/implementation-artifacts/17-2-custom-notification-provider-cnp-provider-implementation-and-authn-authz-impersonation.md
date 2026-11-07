# Story 17.2: Custom Notification Provider (CNP) Provider Implementation & AuthN/AuthZ Impersonation

Status: ready-for-dev

## Story

As a System Architect and Backend Engineer,
I want a specialized `CNPEmailProvider` for Custom Notification Provider (CNP) Admit API with IAM service account impersonation, in-memory token caching, and deduplication handling,
so that the API tier can securely deliver enterprise email campaigns through CNP within Google Cloud infrastructure.

## User Request Context & Specifics

1. Implement `CNPEmailProvider` in `api/notifications/cnp_provider.go`.
2. Direct API-rendered HTML/plain-text email messages (`rendering_options.rendered_message.email_message`).
3. Google Application Default Credentials (ADC) with IAM token impersonation for `NOTIFICATIONS_SENDER_SERVICE_ACCOUNT`.
4. In-memory token caching and automatic refresh with `oauth2.ReuseTokenSource`.
5. Unique `request_id` generation (UUID v4) and mapping of CNP `ALREADY_EXISTS` (409) to idempotent success (`SendResult{Status: "DUPLICATE"}`).
6. Offline simulated mode when `NOTIFICATIONS_ENVIRONMENT=local`.

## Verified Design Architecture & Elicited Decisions

- **Direct API Rendered Content**: Serializes `subject`, `html_body`, and `text_body` directly into `rendering_options.rendered_message.email_message`.
- **Recipient Targeting**: Sets `recipient.email_user.email_address` with the normalized `ToEmail`.
- **AuthN / AuthZ Token Generation**: Generates OAuth2 access token for the configured sender SA using IAM Credentials API and caches tokens using `oauth2.ReuseTokenSource`.
- **Idempotent Deduplication**: Returns `200 OK` with `status: "DUPLICATE"` on `ALREADY_EXISTS` (409).
- **Simulated Mode**: `NOTIFICATIONS_ENVIRONMENT=local` verifies parameters and payload offline without outbound network calls.

## Acceptance Criteria

1. **Given** the `CNPEmailProvider` in `api/notifications/cnp_provider.go`, **When** validating configuration via `ValidateConfig()`, **Then** it MUST verify that `ProjectID` and `SenderServiceAccount` are non-empty and well-formed.
2. **Given** a valid `EmailNotification` with direct API-rendered content (`Subject`, `HTMLContent`, optional `TextContent`), **When** `CNPEmailProvider.Send()` serializes the CNP Admit payload, **Then** it MUST construct a `CNPAdmitRequest` JSON payload matching the CNP proto schema:
   - `campaign_identifiers.project_id` = `config.ProjectID`
   - `campaign_identifiers.campaign_id` = `config.DefaultCampaign`
   - `recipient.email_user.email_address` = `notif.ToEmail`
   - `rendering_options.rendered_message.email_message.subject` = `notif.Subject`
   - `rendering_options.rendered_message.email_message.html_body` = `notif.HTMLContent`
   - `rendering_options.rendered_message.email_message.text_body` = `notif.TextContent`
3. **Given** CNP service account authentication on Cloud Run, **When** generating OAuth2 access tokens for `NOTIFICATIONS_SENDER_SERVICE_ACCOUNT`, **Then** the provider MUST use Google Application Default Credentials (ADC) with IAM token generation and wrap the token source in `oauth2.ReuseTokenSource` for in-memory token reuse and automatic pre-expiry refresh.
4. **Given** a dispatch without an explicit `RequestID`, **When** sending via CNP, **Then** the provider MUST auto-generate a unique UUID v4 `request_id` for deduplication.
5. **Given** an upstream CNP response returning `ALREADY_EXISTS` (HTTP 409 status code), **When** handling the response, **Then** the provider MUST treat it as an idempotent success and return `&SendResult{Status: "DUPLICATE", MessageID: reqID}` without raising an error.
6. **Given** an upstream CNP response returning `PERMISSION_DENIED` (HTTP 403) or `UNAUTHENTICATED` (HTTP 401), **When** handling the response, **Then** the provider MUST return a descriptive error detailing the authorization or token validation failure.
7. **Given** `NOTIFICATIONS_ENVIRONMENT=local`, **When** `CNPEmailProvider.Send()` runs, **Then** it MUST execute in offline simulated mode, validating sender service account format and message structure without making network calls.
8. **Given** unit tests in `api/notifications/cnp_provider_test.go`, **When** running `go test -v ./api/notifications/...`, **Then** tests MUST verify payload serialization, AuthN bearer headers, AuthZ failure handling, deduplication idempotency, and simulated mode.

## Tasks / Subtasks

- [ ] Task 1: CNP Configuration Struct & Provider Constructor (AC: #1, #3)
  - [ ] Define `CNPConfig` struct with `ProjectID`, `DefaultCampaign`, `SenderServiceAccount`, `Endpoint`, `Environment`, `Timeout`, `TokenSource`, and `HTTPClient`.
  - [ ] Implement `NewCNPEmailProvider(cfg)` and `ValidateConfig()`.
  - [ ] Configure `oauth2.ReuseTokenSource` around the token source.
- [ ] Task 2: CNP Admit Request Payload Modeling & Dispatch (AC: #2, #4, #5, #6)
  - [ ] Define `CNPAdmitRequest`, `CNPRenderingOptions`, and `CNPRenderedMessage` structs.
  - [ ] Implement `Send(ctx, notif)` building the HTTP request to the CNP Admit endpoint.
  - [ ] Implement response status code handling (200 OK -> SUCCESS, 409 -> DUPLICATE, 401/403/500 -> descriptive error).
- [ ] Task 3: Offline Simulation Mode (AC: #7)
  - [ ] Implement local simulation path when `config.Environment == "local"`.
- [ ] Task 4: CNP Unit & Integration Test Suite (AC: #8)
  - [ ] Create `api/notifications/cnp_provider_test.go` with mock HTTP server verifying JSON payload, bearer headers, 409 handling, and local mode.
