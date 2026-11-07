---
title: 'Story 16.6: End-to-End Regression Test Suite for Brainstorming Activation & TOML Resolution'
type: 'testing'
created: '2026-08-15'
status: 'ready-for-dev'
baseline_commit: '23b9c83f3428d854ba51161336cbff49e5124d02'
review_loop_iteration: 0
context: []
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** Changes across tool provisioning, script validation, TOML configuration seeding, and path safety must be verified together to ensure the entire AI inference lifecycle—specifically `APF Brainstorm` activation, `memlog.py` initialization, directory globbing, and BMad multi-layer TOML resolution—functions seamlessly end-to-end without regressions or tool failures.

**Approach:** Develop an end-to-end regression testing suite in `ai/plugins/bmad/interceptor_test.go` and `ai/sub_agents/analysis/agent_test.go`. The test suite will construct a realistic workspace fixture with `_bmad/config.toml`, `_bmad/custom/config.user.toml`, and `skills/bmad-brainstorming/SKILL.md`, simulating the entire agent turn: reading skill files, resolving TOML customizations and configurations via `resolve_config.py`, globbing existing `.memlog.md` files via `directory_reader`, running `memlog.py` via `uv_run`, and asserting zero tool errors or perimeter violations.

## Boundaries & Constraints

**Always:**
- Use hermetic test fixtures created in temporary directories.
- Test the full agent toolset: `file_reader`, `file_writer`, `directory_reader`, and `uv_run`.
- Verify that `resolve_config.py` outputs expected JSON matching the seeded `config.user.toml`.
- Verify that `directory_reader` correctly returns `.memlog.md` paths when globbed.

**Never:**
- Do not rely on unmanaged external network resources or live GCS buckets in unit tests.
- Do not skip assertions on tool error logs or return codes.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
| :--- | :--- | :--- | :--- |
| **Brainstorm Full Activation** | Agent loads `bmad-brainstorming` skill in fresh workspace | Step 1 (`resolve_customization.py`) $\rightarrow$ Step 3 (`resolve_config.py`) $\rightarrow$ Step 5 (`directory_reader` glob) succeed with 0 errors | All steps pass with exit code 0 |
| **Resuming Existing Memlog** | Workspace contains existing `.memlog.md` files | `directory_reader` discovers `.memlog.md` files; `file_reader` loads frontmatter | Correct list of resume options presented |
| **TOML Override Precedence** | `_bmad/custom/config.user.toml` overrides `user_name` | `resolve_config.py` returns overridden user name | Verified in merged JSON output |

</frozen-after-approval>

---

## Story

As a Quality Engineer and Developer,
I want an end-to-end regression test suite verifying Brainstorm activation and native TOML configuration resolution,
So that all subagents execute BMad skills and Python utilities reliably without regression.

---

## Acceptance Criteria

1. **Given** a mock project workspace with `_bmad/` configuration and `skills/bmad-brainstorming/`, **When** simulating `APF Brainstorm` activation, **Then** the test MUST verify that `resolve_customization.py` and `resolve_config.py` execute successfully via `UVRunnerTool` and return valid merged configuration JSON. (AC: 16.6.1)
2. **Given** a test workspace containing `.memlog.md` files in `docs/spec/`, **When** the agent invokes `DirectoryReaderTool` with `docs/spec/*/.memlog.md`, **Then** the tool MUST return all matching file paths accurately. (AC: 16.6.2)
3. **Given** `UVRunnerTool`, **When** passed invalid command binary names (`find`, `ls`, `python3`), **Then** the test MUST verify immediate rejection with clear guidance without launching subprocesses. (AC: 16.6.3)
4. **Given** the entire AI package test suite (`go test -v ./ai/...`), **When** running the automated tests, **Then** 100% of tests MUST pass cleanly. (AC: 16.6.4)

---

## Tasks / Subtasks

- [ ] **Task 1: Create Brainstorm Activation Integration Test (AC: 16.6.1, 16.6.2)**
  - [ ] In `ai/plugins/bmad/interceptor_test.go` or a dedicated `ai/sub_agents/analysis/agent_integration_test.go`, build `TestBrainstormActivation_EndToEnd`.
  - [ ] Set up workspace directory with `_bmad/` scripts, `config.toml`, `config.user.toml`, and skill files.
  - [ ] Execute `file_reader`, `uv_run`, and `directory_reader` sequence.
- [ ] **Task 2: Test TOML Resolution & Precedence (AC: 16.6.1)**
  - [ ] Verify `resolve_config.py` merges `_bmad/config.toml` and `_bmad/custom/config.user.toml` accurately.
- [ ] **Task 3: Run Full AI Test Suite (AC: 16.6.4)**
  - [ ] Run `go test -v ./ai/...` and confirm all tests pass.
