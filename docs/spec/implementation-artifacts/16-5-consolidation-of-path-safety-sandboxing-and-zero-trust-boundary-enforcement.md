---
title: 'Story 16.5: Consolidation of Path Safety, Sandboxing & Zero-Trust Boundary Enforcement'
type: 'refactoring'
created: '2026-08-15'
status: 'ready-for-dev'
baseline_commit: '23b9c83f3428d854ba51161336cbff49e5124d02'
review_loop_iteration: 0
context: []
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** Currently, file path handling undergoes a multi-pass "ping-pong" across 3 separate interceptors: `PathSafetyInterceptor` (converts relative paths to absolute, enforces `docs/spec` write jail), `ContentSanitizerInterceptor` (converts absolute paths back to relative and sanitizes content), and `FileWriteNotifierInterceptor` (strips paths a second time and registers GORM state). This multi-stage circular path mutation is fragile, hard to debug, and requires complex regexes to scrub file paths.

**Approach:** Consolidate path safety, boundary enforcement, and path sanitization into a single, unified security architecture. Standardize on project-relative paths as the clean agent-facing interface. Centralize path resolution so that tools (`file_reader`, `file_writer`, `directory_reader`, `uv_runner`) resolve paths internally within the zero-trust sandbox (`projectRoot` and `bmadRoot`), while interceptors enforce boundaries and state notifications in a single, coherent pass without circular path mutation.

## Boundaries & Constraints

**Always:**
- Enforce Zero-Trust perimeter boundaries: reads must stay within `projectRoot` or `bmadRoot`; writes must stay within `projectRoot/docs/spec` (or root instruction files `AGENTS.md` / `project-context.md`).
- Present clean, relative paths to the LLM agent and UI (e.g. `docs/spec/plans/prd.md`, not `/mount/org/user/proj/docs/spec/plans/prd.md`).
- Ensure GORM state keys `file-write-notification:<uuid>` are emitted reliably on file writes for Git sync and UI reactivity.
- Keep path checking robust against directory traversal attacks (`../`, symlinks).

**Never:**
- Do not expose internal server paths (`/mount/...`, `/usr/local/...`) to the agent context.
- Do not perform redundant string replacements across multiple interceptors when a single resolution step suffices.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
| :--- | :--- | :--- | :--- |
| **Relative File Write** | Agent writes to `plans/overview.md` | Path is securely placed into `projectRoot/docs/spec/plans/overview.md`; returns clean relative path `docs/spec/plans/overview.md` | Path jail enforced automatically |
| **Direct Specs Write** | Agent writes to `docs/spec/plans/overview.md` | Writes to `projectRoot/docs/spec/plans/overview.md` | Success |
| **Root Instruction Write** | Agent writes to `AGENTS.md` | Allowed in `projectRoot/AGENTS.md` | Success |
| **Directory Traversal** | Agent passes `../../etc/passwd` | Interceptor blocks traversal attempt | Returns zero-trust perimeter error |
| **Read BMad Framework File** | Agent reads `skills/bmad-brainstorming/SKILL.md` | Resolves path within `bmadRoot`; returns file content | Read allowed |

</frozen-after-approval>

---

## Story

As a Security Architect and Developer,
I want a consolidated, zero-trust path safety and boundary enforcement layer,
So that path resolution, sandbox isolation, and content sanitization operate predictably in a single pass without circular path transformations.

---

## Acceptance Criteria

1. **Given** a tool call to `file_writer`, **When** `PathSafetyInterceptor.Before()` runs, **Then** it MUST validate that the target file path resides within `projectRoot/<specLocation>` or is an allowed root instruction file (`AGENTS.md`, `project-context.md`), preventing any out-of-sandbox write. (AC: 16.5.1)
2. **Given** tool calls to `file_reader` or `directory_reader`, **When** `PathSafetyInterceptor.Before()` runs, **Then** it MUST verify the target path resides strictly within `projectRoot` or `bmadRoot`, blocking unauthorized system paths. (AC: 16.5.2)
3. **Given** tool execution outputs, **When** `ContentSanitizerInterceptor` or unified path interceptor processes returned file paths, **Then** all paths MUST be sanitized to project-relative paths (or `bmadRoot`-relative paths) without leaking internal filesystem mount prefixes (`/mount/orgId/userId/projectId/`). (AC: 16.5.3)
4. **Given** a successful `file_writer` execution, **When** registering the write event, **Then** `FileWriteNotifierInterceptor` MUST record a unique `file-write-notification:<uuid>` entry in `ctx.State()` containing the clean project-relative file path. (AC: 16.5.4)
5. **Given** the test suite in `ai/plugins/bmad/interceptor_test.go`, **When** running tests, **Then** all path safety, sandbox jail, and sanitization tests MUST pass with 100% reliability. (AC: 16.5.5)

---

## Tasks / Subtasks

- [ ] **Task 1: Unify Path Resolution Logic (AC: 16.5.1, 16.5.2)**
  - [ ] Consolidate path normalization helpers in `ai/plugins/bmad/path_safety.go`.
  - [ ] Implement clean relative-to-sandbox resolver: `ResolveProjectOrBmadPath(rawPath, projectRoot, bmadRoot)`.
  - [ ] Maintain zero-trust write guardrails for `docs/spec` and root instruction files.
- [ ] **Task 2: Streamline Content and Output Path Sanitization (AC: 16.5.3)**
  - [ ] In `ai/plugins/bmad/content_sanitizer.go`, streamline `sanitizePath()` and `sanitizeContent()` to operate predictably on relative paths.
  - [ ] Ensure error messages from tools are cleanly sanitized.
- [ ] **Task 3: Consolidate State Notification Key Format (AC: 16.5.4)**
  - [ ] Standardize `file-write-notification` keys to use UUIDs matching `uv_runner.go` convention.
- [ ] **Task 4: Update Unit & Boundary Tests (AC: 16.5.5)**
  - [ ] Add comprehensive test cases in `interceptor_test.go` covering edge cases: relative paths, absolute paths, traversal attempts, spec jails, and root instructions.
