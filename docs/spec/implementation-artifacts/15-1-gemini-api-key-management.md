# Story 15.1: Gemini API Key Management

Status: ready-for-dev

## Story

As a user,
I should be able to input my Gemini API key in user settings or override it at a project level,
so that it can be used by the AI tier to make all Gemini model calls.

## User Request Context & Specifics

1. Enable the Gemini integration in the seed data.
2. Create a one-time executable script to update this configuration in staging and prod.
3. Similar to GitHub integration, user should be able to provide Gemini API Key in settings, or override at a project level.
4. This API key will be used by the AI tier to make all Gemini model calls.
5. **Database-Seeded Dev/Test Credentials**: For dev/test environment, the Gemini API key MUST be seeded into and loaded exclusively from the database (similar to the simulated GitHub access token `ghu_simulated_access_token_...`) and MUST NOT load or fall back to environment variables (`GEMINI_API_KEY`).
6. **Manual Migration CLI Execution**: The one-time migration script `cmd/migrate_gemini_integration` is executed manually by the engineer/operator, using `--dry-run` to preview changes prior to live execution (CI pipeline automated gating is not required).

## Verified Design Architecture & Elicited Decisions

- **Strict Database-Only Key Requirement & HTTP 400 Error**: AI execution requests MUST be blocked at the API gateway level if neither a user-level secret nor a project-level key override is configured in the database for the user/project. The system returns HTTP 400 Bad Request with structured JSON `{ "code": "GEMINI_API_KEY_REQUIRED", "error": "Gemini API key is required. Please configure your key in User Secrets Settings or Project Settings." }`.
- **Database-Seeded Dev/Test Credentials (No Env Var Fallback)**: In local development and automated testing environments (similar to GitHub's simulated access token), a seed credential (`gemini_simulated_api_key_12345`) is pre-populated in the seed database. Runtime key resolution ALWAYS reads exclusively from the database via `db.ResolveAppCredential`. Reading or falling back to environment variables (`GEMINI_API_KEY` or `cfg.GeminiAPIKey`) is strictly forbidden across all environments.
- **Immediate Hard Deletion on Revocation**: When a user deletes or removes their Gemini API key, the key document (`AppCredential` or project `UserProject` setting) is instantly hard-deleted from Firestore. Subsequent AI execution requests for that user/project immediately fail at the gateway with HTTP 400 `GEMINI_API_KEY_REQUIRED`.
- **Inherit Global Key by Default**: Enabling Gemini for a project automatically inherits the user's global Gemini API key unless the user explicitly toggles "Override for this project" and inputs a project-specific key (`UserProject` setting with `isOverridden: true`).
- **User-Scoped Project Override**: Project-level Gemini API key overrides are configured per-user per-project (`appId: "gemini"`). Each user maintains isolated keys for a project.
- **Synchronous Validation on Save**: API key submissions perform non-empty format validation and a lightweight connection check against Gemini API endpoints on submit, returning immediate validation feedback in the UI.
- **Masked Preview & Status Metadata**: GET endpoints return a masked key representation (e.g., `••••••••a1b2`), `configured: true` status flag, and `lastTestedAt` ISO timestamp.
- **Dedicated UI Integration Card**: The UI renders a dedicated Gemini AI Model card in User Secrets Settings and Project Integration Settings featuring status badges ("Configured" / "Not Configured"), masked key display, a "Test Connection" button, and Save action.
- **Manual Executable Migration Script with ADC & Dry-Run**: The migration script `api/cmd/migrate_gemini_integration/main.go` uses Application Default Credentials (ADC) or Firestore emulator host with a `--dry-run` flag to perform an idempotent bulk upsert of the `"gemini"` integration document into the global `app_integrations` catalog and every organization's `app_integrations` collection in Firestore. It is run manually by operators in staging and prod environments.
- **Encrypted JWE Transmission**: The resolved `GeminiApiKey` is packaged inside the encrypted JWE payload (`userSecretsEncrypted`) forwarded to the AI tier agent proxy, enabling request-scoped GenAI client instantiation without global state caching.

## Acceptance Criteria

1. **Given** seed data initialization or manual staging/prod migration script execution, **When** seeding or running the migration script, **Then** the Gemini integration (`id: "gemini"`) is enabled and registered in the `app_integrations` catalog with `authType: "apikey"` and field `gemini_api_key` (`isSecret: true`), and a dev/test seed credential (`gemini_simulated_api_key_12345`) is pre-populated in seed database fixtures.
2. **Given** any execution environment (dev, test, staging, prod), **When** `AgentProxy` resolves the Gemini API Key, **Then** it fetches the key exclusively from the database via `db.ResolveAppCredential(..., "gemini")` and NEVER reads or falls back to environment variables (`GEMINI_API_KEY`).
3. **Given** a user in the User Secrets Settings screen, **When** they enter a global Gemini API Key and click Save, **Then** the system validates the key synchronously against Gemini API endpoints, encrypts it using `SECRETS_ENCRYPTION_KEY`, and stores it as an `AppCredential` (`integrationId: "gemini"`).
4. **Given** a user in User Secrets or Project Settings, **When** they delete/remove their Gemini API key, **Then** the system hard-deletes the key document from Firestore, and subsequent AI execution requests fail with HTTP 400 `GEMINI_API_KEY_REQUIRED`.
5. **Given** a user in the Project Integration Settings screen, **When** they enable Gemini integration, **Then** it inherits the global key by default unless "Override for this project" is toggled, in which case it validates and saves a project-specific `UserProject` setting (`appId: "gemini"`, `isOverridden: true`).
6. **Given** Settings UI rendering or GET credential status API queries, **Then** Gemini integration is presented as a dedicated card displaying a status badge ("Configured" / "Not Configured"), masked key representation (`••••••••a1b2`), `lastTestedAt` timestamp, "Test Connection" button, Delete action, and Save/Edit controls.
7. **Given** an AI execution request, **When** `AgentProxy` populates authentication and secrets for a session, **Then** it resolves the Gemini API key via `db.ResolveAppCredential(..., "gemini")` (checking project-level override -> user global secret). If no key document exists in database, it aborts execution at gateway level with HTTP 400 and code `GEMINI_API_KEY_REQUIRED`.
8. **Given** a resolved Gemini API Key, **When** calling the AI tier, **Then** `AgentProxy` packages `GeminiApiKey` into the JWE `userSecretsEncrypted` payload, and the AI service instantiates a request-scoped Gemini model client.
9. **Given** manual execution by an operator in staging or production, **When** running `cmd/migrate_gemini_integration` with `--dry-run`, **Then** the script previews all catalog changes without mutating Firestore, and when run live, idempotently provisions/updates the `"gemini"` integration document across all target organizations and the global catalog.

## Tasks / Subtasks

- [ ] Task 1: Seed Data & Integration Catalog Definition (AC: #1)
  - [ ] Add `"gemini"` integration definition and simulated dev/test seed credential (`gemini_simulated_api_key_12345`) to `api/seeding/templates/showcase/project.json` and seed data fixtures.
  - [ ] Define `gemini` integration metadata (`id: "gemini"`, `name: "Gemini AI Model"`, `authType: "apikey"`, `icon: "gemini"`, `isSecret: true`).
- [ ] Task 2: Executable Manual Migration Script with ADC & Dry-Run (AC: #9)
  - [ ] Create `api/cmd/migrate_gemini_integration/main.go` supporting `--dry-run`, `--project-id`, and ADC authentication for manual CLI execution by operators.
  - [ ] Implement idempotent Firestore upsert logic targeting `app_integrations` collection for global & org-scoped configurations across all organizations.
- [ ] Task 3: User & Project Secrets Integration API, Database-Only Resolution, & Gateway Gating (AC: #2, #3, #4, #5, #7)
  - [ ] Update `api/agent/agent_proxy.go` (`userSecretsBridge`) to include `GeminiApiKey` and `GeminiConfigured`.
  - [ ] Update `populateAuthAndSecrets` in `api/agent/agent_proxy.go` to resolve `"gemini"` credentials exclusively via `db.ResolveAppCredential` from database (no environment variable reading), enforcing strict key presence (returning HTTP 400 `GEMINI_API_KEY_REQUIRED` if missing in database).
  - [ ] Add API endpoint / handler for validating Gemini API key synchronously against Gemini API endpoint on save.
  - [ ] Add hard deletion handler on key removal in User & Project settings.
  - [ ] Include masked key preview (`••••••••a1b2`) and `lastTestedAt` timestamp in GET status responses.
- [ ] Task 4: UI Dedicated Integration Card Component (AC: #6)
  - [ ] Create/update User Secrets and Project Settings UI components to render a dedicated Gemini AI integration card.
  - [ ] Implement status badges ("Configured"/"Not Configured"), key masking, "Test Connection" button, Delete action, and project override toggle.
- [ ] Task 5: AI Tier Integration & Request-Scoped Model Client (AC: #8)
  - [ ] Update `api/agent/agent_proxy.go` to include `GeminiApiKey` in the JWE `userSecretsEncrypted` payload.
  - [ ] In AI tier / ADK agent proxy, decrypt `userSecretsEncrypted` payload and instantiate request-scoped GenAI model client using the provided key.
- [ ] Task 6: Automated & Integration Tests (AC: #1, #2, #3, #4, #5, #6, #7, #8, #9)
  - [ ] Add unit tests in `api/agent/agent_proxy_test.go` verifying `GeminiApiKey` database resolution, HTTP 400 gateway gating when missing in DB, dev/test database seed key loading, deletion revocation, and JWE payload encryption.
  - [ ] Add unit tests for key validation endpoint and migration script (`--dry-run` and live execution).

## Dev Notes

### Architecture & Code Patterns

- **Database-Only Secrets Resolution Hierarchy**:
  ```
  1. Project-level override in DB (UserProject setting for projectId + appId "gemini", if isOverridden == true and enabled == true)
  2. User-level global credential in DB (AppCredential for userId + integrationId "gemini")
  3. Dev/Test Environment: Pre-populated with seed credential in DB ("gemini_simulated_api_key_12345") loaded via db.ResolveAppCredential.
  4. Requirement: MUST NOT read or fall back to environment variables (GEMINI_API_KEY). If key document is absent from DB, return HTTP 400 GEMINI_API_KEY_REQUIRED.
  ```
- **Encryption & Revocation**: Secrets stored in Firestore MUST be encrypted using `db.EncryptSecret` / `db.DecryptSecret` via `SECRETS_ENCRYPTION_KEY`. On key deletion, documents are hard-deleted from Firestore immediately.
- **Agent Proxy Bridge**: `userSecretsBridge` in `api/agent/agent_proxy.go` updated:
  ```go
  type userSecretsBridge struct {
  	UserID               string `json:"userId"`
  	FigmaConfigured      bool   `json:"figmaConfigured"`
  	LucidchartConfigured bool   `json:"lucidchartConfigured"`
  	StitchConfigured     bool   `json:"stitchConfigured"`
  	DrawioConfigured     bool   `json:"drawioConfigured"`
  	GeminiConfigured     bool   `json:"geminiConfigured"`
  	FigmaApiKey          string `json:"figmaApiKey"`
  	LucidchartApiKey     string `json:"lucidchartApiKey"`
  	StitchApiKey         string `json:"stitchApiKey"`
  	DrawioApiKey         string `json:"drawioApiKey"`
  	GeminiApiKey         string `json:"geminiApiKey"`
  }
  ```
- **Migration CLI Script**: Placed in `api/cmd/migrate_gemini_integration/main.go`, utilizing `core.NewConfig()`, Google Cloud ADC, and supporting `--dry-run` for manual execution.

### Project Structure & Component Locations

- **Backend API Tier**:
  - `api/agent/agent_proxy.go`: Agent run context resolution, database-only key gating (HTTP 400 when missing in DB), and JWE secrets payload population.
  - `api/db/credentials.go`: `ResolveAppCredential` credential hierarchy resolution.
  - `api/db/config.go`: `AppIntegration` definitions and Firestore `app_integrations` store.
  - `api/cmd/migrate_gemini_integration/main.go` (NEW): One-time migration script.
  - `api/seeding/templates/showcase/project.json`: Showcase seed data template (contains `gemini_simulated_api_key_12345`).
- **Frontend UI Tier**:
  - `ui/`: Settings modal & Project Integration card components.
- **AI Tier**:
  - `ai/core/configuration.go`: Baseline configuration.
  - `ai/registry/registry.go`: Request-scoped Gemini model client instantiation from JWE payload.

### References

- [db/user.go](api/db/user.go#L78-L100)
- [agent/agent_proxy.go](api/agent/agent_proxy.go#L52-L62)
- [integrations/github.go](api/integrations/github.go#L620-L630)
- [epics.md](docs/spec/planning-artifacts/epics.md#L1359-L1376)

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### Review Findings

- [x] [Review][Patch] populateAuthAndSecrets & GetJWESecretsForUser bypass resolution when projectId is empty [api/agent/agent_proxy.go:201,265]
- [x] [Review][Patch] Migration CLI main.go silently logs warning and exits cleanly on organization iteration failure [api/cmd/migrate_gemini_integration/main.go:83]
- [x] [Review][Patch] Add GeminiIntegration validation implementation to integrations/implementations.go & registry.go [api/integrations/implementations.go]
- [x] [Review][Patch] Add Gemini integration definition to showcase project seed template [api/seeding/templates/showcase/project.json:14]
- [x] [Review][Patch] Add data-testid and UI card support for Playwright E2E tests [ui/src/components/shared/AppIntegrationsList.tsx]




### File List

