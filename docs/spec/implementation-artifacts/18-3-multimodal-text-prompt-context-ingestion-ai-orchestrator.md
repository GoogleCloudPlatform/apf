---
title: 'Story 18.3: Multimodal & Text Prompt Context Ingestion (AI Orchestrator)'
type: 'feature'
created: '2026-08-27'
status: 'done'
baseline_revision: '3bad6af9a040c8952cc6beb2d8059e9c4b75e290'
review_loop_iteration: 0
followup_review_recommended: true
context:
  - docs/spec/implementation-artifacts/epic-18-context.md
warnings: []
deferred: []
---

<intent-contract>

## Intent

**Problem:** The AI orchestrator only sends plain text strings to the ADK agent, ignoring attached session artifacts and preventing the agent from analyzing uploaded text/code files and multimodal visual diagrams.

**Approach:** Ingest attached session artifacts via `ArtifactService`, inject text/code documents into the prompt context within demarcation blocks (`[Attached File: <name>]`), construct multimodal `InlineData` image parts for Gemini vision calls, and append structured warnings on corrupted/unsupported files without interrupting the stream.

## Boundaries & Constraints

**Always:**
- Extract text and code artifacts (`.txt`, `.md`, `.json`, `.yaml`, `.yml`, `.py`, `.go`, `.ts`, `.tsx`, `.js`, `.html`, `.css`, `.csv`, `.sql`, `.sh`, `.xml`, `.proto`, etc., or `text/*` / JSON / YAML MIME types) and format them into demarcation blocks: `[Attached File: <filename>]\n<content>`.
- Ingest image artifacts (`.png`, `.jpg`, `.jpeg`, `.webp`, `.gif`, etc., or `image/*` MIME types) as multimodal `ProxyMessagePart` objects containing `InlineData` with detected MIME type and base64-encoded payload.
- On corrupted files, missing storage paths, or unsupported non-text binary files, append a structured warning (e.g. `[Warning: Could not read attached file "<filename>": <reason>]` or `[Warning: Unsupported binary file "<filename>"]`) to the prompt context and log the issue without crashing or dropping the agent execution stream.
- Support multi-tier reading of session artifacts from both local filesystem (`local_file`) and Google Cloud Storage (`gcs`) via `ArtifactService`.
- Pass both the prompt text part and all multimodal image parts to `/api/run_sse` in the `newMessage.parts` array.

**Block If:**
- External third-party credentials or human-only administrative actions outside the repository are required.

**Never:**
- Crash or terminate the agent turn on a corrupted or unsupported file.
- Discard or truncate the user's primary prompt text when appending attached file demarcation blocks.
- Hardcode local storage paths or assume local filesystem when running in cloud environments.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| Message with text artifact | User prompt "Analyze this" + attached `notes.md` | Text part contains prompt and `[Attached File: notes.md]\n...` block | No error expected |
| Message with image artifact | User prompt "Explain diagram" + attached `arch.png` | `newMessage.parts` contains text part and `inlineData` part (`image/png`, base64) | No error expected |
| Message with multiple artifacts (text + image) | User prompt with `schema.sql` and `flow.jpg` | Text part contains prompt + `schema.sql` demarcation block; `parts` contains text and image parts | No error expected |
| Unreadable / missing artifact | Artifact metadata points to non-existent file path | Warning `[Warning: Could not read attached file "missing.txt": ...]` injected into prompt; stream continues | Log warning, proceed with turn |
| Corrupted / 0-byte image | Image file has 0 bytes or corrupted data | Warning `[Warning: Could not load attached file "...": empty file]` injected into prompt; stream continues | Log warning, proceed with turn |
| Unsupported binary file | Attached `.bin` or `.exe` with non-UTF8 bytes | Warning `[Warning: Unsupported binary file format for "..."]` injected into prompt; stream continues | Log warning, proceed with turn |
| Empty prompt with image | Empty text prompt + attached `screenshot.png` | Ingests image part with empty or placeholder text part; stream executes cleanly | No error expected |

</intent-contract>

## Code Map

- `api/core/artifact_service.go` -- `ArtifactService` interface and implementations (`FileSystemArtifactService`, `GcsArtifactService`) with `ReadSessionArtifact(ctx context.Context, storageType string, storagePath string) ([]byte, error)`.
- `api/core/artifact_service_test.go` -- Unit tests for `ReadSessionArtifact` across local filesystem and GCS storage drivers.
- `api/agent/prompt_ingestion.go` -- Ingestion engine with `BuildPromptParts(ctx context.Context, prompt string, artifacts []db.Artifact, svc core.ArtifactService) ([]ProxyMessagePart, error)` implementing text injection, image base64 encoding, and structured warning fallback.
- `api/agent/prompt_ingestion_test.go` -- Comprehensive unit test suite covering text injection, multimodal image parts, multiple attachments, unreadable files, and unsupported binary formats.
- `api/agent/agent_proxy.go` -- Added `InlineData *ProxyInlineData` to `ProxyMessagePart`, and implemented `RunAgentStreamingWithParts` method for streaming multipart messages to `/api/run_sse`.
- `api/agent/agent_proxy_test.go` -- Unit tests for multipart payload serialization and streaming execution.
- `api/chat.go` -- `UserMessageHandler` updated to resolve artifacts, construct prompt parts via `BuildPromptParts`, and dispatch to `proxy.RunAgentStreamingWithParts`.
- `api/chat_test.go` -- Integration tests verifying `UserMessageHandler` processes attached artifacts and dispatches multipart payloads.

## Tasks & Acceptance

**Execution:**
- `api/core/artifact_service.go` -- Add `ReadSessionArtifact(ctx context.Context, storageType string, storagePath string) ([]byte, error)` to `ArtifactService` interface, `FileSystemArtifactService`, and `GcsArtifactService`.
- `api/core/artifact_service_test.go` -- Add unit tests for `ReadSessionArtifact` validating reading from local filesystem and GCS paths.
- `api/agent/agent_proxy.go` -- Update `ProxyMessagePart` with `InlineData *ProxyInlineData` struct and add `RunAgentStreamingWithParts(ctx context.Context, appName, sessionId, projectId, userId string, parts []ProxyMessagePart) (io.ReadCloser, error)`.
- `api/agent/prompt_ingestion.go` -- Create prompt ingestion engine with file classification, demarcation block formatting for text/code, base64 multimodal packaging for images, and structured warning resilience for unreadable or unsupported files.
- `api/agent/prompt_ingestion_test.go` -- Add unit tests for `BuildPromptParts` covering text injection, image parts, mixed attachments, empty prompts, and error warnings.
- `api/chat.go` -- Update `UserMessageHandler` to load `core.ArtifactService`, build prompt parts with attached artifacts, and invoke `RunAgentStreamingWithParts`.
- `api/chat_test.go` -- Add tests verifying `UserMessageHandler` handles messages with text and image attachments.

**Acceptance Criteria:**
- Given a chat message with attached text or code artifacts (`.txt`, `.md`, `.json`, `.yaml`, `.py`, `.go`, etc.), when the AI orchestrator receives the chat turn request, then the backend MUST read the text content and inject it into the prompt context under formatted demarcation blocks (e.g., `[Attached File: filename.md]\n...content...`).
- Given a chat message with attached image artifacts (`.png`, `.jpg`, `.jpeg`, `.webp`, `.gif`), when the AI orchestrator prepares the model payload for Gemini, then the system MUST construct multimodal `Part` objects with `InlineData` (base64-encoded bytes and MIME type) and pass the image parts directly to the Gemini model call alongside the text prompt.
- Given an unsupported binary file or corrupted image, when content extraction fails, then the system MUST return a structured warning to the user turn without crashing the agent execution stream.

## Spec Change Log

## Review Triage Log

### 2026-08-27 — Review pass
- intent_gap: 0
- bad_spec: 0
- patch: 5: (high 0, medium 0, low 5)
- defer: 0
- reject: 8
- addressed_findings:
  - `[low]` `[patch]` Added uninitialized client guard and URI validation to `GcsArtifactService.ReadSessionArtifact`
  - `[low]` `[patch]` Added empty parts slice guard in `AgentProxy.RunAgentStreamingWithParts`
  - `[low]` `[patch]` Enhanced `UserMessageHandler` to merge missing artifact IDs from `ArtifactStore` when resolving session artifacts
  - `[low]` `[patch]` Added warning logging when listing artifacts for project fails during prompt ingestion
  - `[low]` `[patch]` Added fallback to GCS storage driver when configured in `UserMessageHandler`

### 2026-08-27 — Review pass 2
- intent_gap: 0
- bad_spec: 0
- patch: 5: (high 0, medium 0, low 5)
- defer: 0
- reject: 7
- addressed_findings:
  - `[low]` `[patch]` Added path traversal validation and storageType verification to `FileSystemArtifactService.ReadSessionArtifact`
  - `[low]` `[patch]` Added storageType verification to `GcsArtifactService.ReadSessionArtifact`
  - `[low]` `[patch]` Reclassified SVG files as text/code artifacts and added `.bmp`, `.tiff`, `.ico` support to `resolveImageMimeType`
  - `[low]` `[patch]` Optimized prompt parts construction to omit redundant empty text part when only images are provided
  - `[low]` `[patch]` Used cancellation-aware request context for artifact resolution in `UserMessageHandler`

## Verification

**Commands:**
- `cd api && go test ./...` -- expected: All unit and integration tests pass without error
- `cd api && go vet ./...` -- expected: No vet errors

## Auto Run Result

### Summary of Implemented Change
- Implemented `ReadSessionArtifact` on `ArtifactService` interface, `FileSystemArtifactService`, and `GcsArtifactService` to support reading session artifacts from both local disk and Google Cloud Storage.
- Implemented `BuildPromptParts` prompt ingestion engine to format text/code documents into formatted demarcation blocks (`[Attached File: <filename>]\n<content>`), encode images into base64 `InlineData` parts, and append structured non-fatal warnings on missing or unsupported files.
- Extended `ProxyMessagePart` with `InlineData *ProxyInlineData` and implemented `RunAgentStreamingWithParts` in `AgentProxy` to transmit multipart message payloads to `/api/run_sse`.
- Updated `UserMessageHandler` to resolve session artifacts, build prompt parts, and stream multipart payloads to the ADK agent.
- Added comprehensive unit tests in `prompt_ingestion_test.go`, `artifact_service_test.go`, `agent_proxy_test.go`, and `chat_test.go`.

### Files Changed
- `api/agent/agent_proxy.go` -- Added `ProxyInlineData`, `InlineData` part field, and `RunAgentStreamingWithParts` method.
- `api/agent/agent_proxy_test.go` -- Unit tests for multipart payload serialization and streaming.
- `api/agent/prompt_ingestion.go` -- Ingestion engine for text demarcation, multimodal image parts, and warning handling.
- `api/agent/prompt_ingestion_test.go` -- Comprehensive unit test suite for prompt parts assembly.
- `api/artifact_upload_test.go` -- Updated test mocks with `ReadSessionArtifact`.
- `api/chat.go` -- Updated `UserMessageHandler` with artifact resolution and multipart streaming.
- `api/chat_test.go` -- Integration tests for message handler artifact processing.
- `api/core/artifact_service.go` -- Added `ReadSessionArtifact` across filesystem and GCS implementations.
- `api/core/artifact_service_test.go` -- Unit tests for reading session artifacts.
- `api/main.go` -- Wired `ArtifactStore` and `AppConfig` into `ChatHandler`.

### Review Findings Breakdown
- Pass 1: 5 patches applied, 0 deferred, 8 rejected.
- Pass 2: 5 patches applied, 0 deferred, 7 rejected.

### Follow-up Review Recommendation
- High: 0, Medium: 0, Low: 5.
- Score: `(3 * 0) + (1 * 5) = 5`.
- `followup_review_recommended: true` (score >= 5).

### Verification Performed
- `cd api && go test ./...` -> PASS (all packages passed).
- `cd api && go vet ./...` -> PASS (clean, 0 warnings).

### Residual Risks
- None identified. Downstream vision capabilities depend on Gemini model support for the supplied image MIME types.


