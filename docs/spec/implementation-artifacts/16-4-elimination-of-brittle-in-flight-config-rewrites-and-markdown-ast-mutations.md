---
title: 'Story 16.4: Elimination of Brittle In-Flight Config Rewrites and Markdown AST Mutations'
type: 'refactoring'
created: '2026-08-15'
status: 'ready-for-dev'
baseline_commit: '23b9c83f3428d854ba51161336cbff49e5124d02'
review_loop_iteration: 0
context: []
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** `ConfigRewriterInterceptor` in `ai/plugins/bmad/config_rewriter.go` currently intercepts every single `file_reader` call and evaluates 6 regular expressions. When a skill file (`SKILL.md`), YAML config (`config.yaml`), or TOML file is read, it executes fragile string mutations—including `fixSkillFile`, `fixBmadPrdSkillFile`, `injectBmmConfiguration`, `injectBmadCoreConfiguration`, and a 150-line custom bracket parser in `addStringItemToTomlArray`. Whenever BMad updates skill wording or workflow structure, these regexes silently fail or mangle the instructions.

**Approach:** Now that dynamic configuration is seeded into `_bmad/custom/config.user.toml` (Story 16.3) and BMad scripts (`resolve_config.py`, `resolve_customization.py`) execute natively via `uv_runner`, eliminate in-flight string mutations from `ConfigRewriterInterceptor`. Deprecate and remove markdown regex transformations and YAML injectors, simplifying `ConfigRewriterInterceptor` or removing it from the interceptor pipeline entirely, resulting in dramatically improved performance and zero maintenance overhead during BMad upstream upgrades.

## Boundaries & Constraints

**Always:**
- Allow skills and agents to read unmodified `SKILL.md` and configuration files directly from the filesystem.
- Rely on native `resolve_config.py` and `resolve_customization.py` runs via `uv_run` for all dynamic resolution.
- Keep `ConfigRewriterInterceptor` minimal or retire it cleanly while maintaining backwards compatibility for existing tests.

**Never:**
- Do not perform regex replacements on markdown instruction files (`SKILL.md`).
- Do not maintain custom character-by-character parsers for TOML/YAML files.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
| :--- | :--- | :--- | :--- |
| **Read SKILL.md** | Agent reads `skills/bmad-brainstorming/SKILL.md` via `file_reader` | Returns exact, unmodified content of `SKILL.md` | Content is clean and unaltered; agent executes Step 1 & Step 3 natively via `uv_run` |
| **Read BMad Config** | Agent reads `_bmad/config.toml` | Returns exact file content | Native `resolve_config.py` merges layers accurately |
| **Legacy YAML Read** | Agent or legacy code reads `_bmad/core/config.yaml` | Returns file content as is without regex manipulation | File is read directly from disk |

</frozen-after-approval>

---

## Story

As a Core Platform Maintainer,
I want to eliminate in-flight regex rewrites and string mutations on markdown and configuration files,
So that BMad framework updates can be ingested cleanly without breaking interceptors or corrupting skill instructions.

---

## Acceptance Criteria

1. **Given** `ConfigRewriterInterceptor` in `ai/plugins/bmad/config_rewriter.go`, **When** a `file_reader` tool call completes for any `SKILL.md` file, **Then** the interceptor MUST NOT perform regex replacements or mutate the file content, returning the raw content directly. (AC: 16.4.1)
2. **Given** `ConfigRewriterInterceptor`, **When** reading configuration files (`_bmad/core/config.yaml`, `_bmad/bmm/config.yaml`, `_bmad/config.toml`, `customize.toml`), **Then** the interceptor MUST NOT invoke `injectBmmConfiguration`, `injectBmadCoreConfiguration`, `injectTomlConfiguration`, or `addStringItemToTomlArray`. (AC: 16.4.2)
3. **Given** `ai/plugins/bmad/interceptor.go`, **When** configuring `BmadInterceptorPlugin`, **Then** the interceptor pipeline MUST be streamlined to eliminate redundant interceptor passes. (AC: 16.4.3)
4. **Given** test cases in `ai/plugins/bmad/interceptor_test.go`, **When** running unit tests, **Then** all tests MUST be updated to verify clean pass-through of file reads without regex mutation. (AC: 16.4.4)

---

## Tasks / Subtasks

- [ ] **Task 1: Deprecate Brittle Mutation Helpers in config_rewriter.go (AC: 16.4.1, 16.4.2)**
  - [ ] Remove `fixSkillFile` and `fixBmadPrdSkillFile` from `ConfigRewriterInterceptor`.
  - [ ] Remove `injectBmmConfiguration`, `injectBmadCoreConfiguration`, and `injectTomlConfiguration`.
  - [ ] Remove `addStringItemToTomlArray` custom character parser.
- [ ] **Task 2: Simplify or Retire ConfigRewriterInterceptor (AC: 16.4.2, 16.4.3)**
  - [ ] Update `ConfigRewriterInterceptor.After()` to act as a lightweight pass-through or retire the interceptor from `ensureInterceptors()` in `ai/plugins/bmad/interceptor.go`.
- [ ] **Task 3: Update Unit Tests in interceptor_test.go (AC: 16.4.4)**
  - [ ] Refactor tests that assert on regex string replacement to assert on native file content preservation.
