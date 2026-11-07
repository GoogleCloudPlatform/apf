---
title: 'Story 16.2: UVRunnerTool Binary Command Guardrails & Tool-Redirect Guidance'
type: 'feature'
created: '2026-08-15'
status: 'ready-for-dev'
baseline_commit: '23b9c83f3428d854ba51161336cbff49e5124d02'
review_loop_iteration: 0
context: []
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** When an LLM agent attempts to invoke shell utilities (such as `find`, `ls`, `grep`, `bash`, `python3`) via the `uv_run` tool, `UVRunnerTool` attempts to resolve the binary name as a file relative to `projectRoot` (e.g. `filepath.Join(projectRoot, "find")`), passes it to `uv run`, and produces an unhelpful subprocess failure (`exitCode: 2`, `durationMs: 16ms`) without explaining why the command failed or guiding the agent toward the correct ADK tool.

**Approach:** Introduce strict script path validation and intelligent error redirection in `ai/tools/uv_runner.go`. When an agent supplies a system binary name, inline interpreter invocation, or non-Python path to `uv_run`, the tool will reject the execution immediately with a structured, agent-friendly error message pointing to the proper specialized tool (e.g. `directory_reader` for file searches/listings, `file_reader` for file contents).

## Boundaries & Constraints

**Always:**
- Validate `scriptPath` before executing `uv run`.
- Allow valid Python scripts (`.py`), Python script paths, and paths with `{project-root}` or `{skill-root}` tokens resolving to `.py` files.
- Return structured `UVRunnerOutput` with `Success: false`, `ExitCode: 1`, and actionable guidance in `Stderr` when non-script commands (`find`, `ls`, `grep`, `cat`, `bash`, `sh`, `python3`, `python`) are passed.
- Maintain zero-trust path boundary checking for resolved script paths.
- Ensure the rejection happens fast (< 1ms) without acquiring concurrency semaphores or launching OS subprocesses.

**Never:**
- Do not execute arbitrary shell commands via `uv_run`.
- Do not panic or return raw nil error structures that crash the ADK runner.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
| :--- | :--- | :--- | :--- |
| **System Utility Rejection** | `uv_run(scriptPath: "find", args: [".", "-name", "*.md"])` | Immediate rejection without spawning subprocess | Returns `Success: false`, `Stderr: "Invalid scriptPath 'find': uv_run only executes Python (.py) scripts. To search for files or list directories, use the 'directory_reader' tool."` |
| **Python Binary Rejection** | `uv_run(scriptPath: "python3", args: ["-c", "..."])` | Immediate rejection with tool redirect | Returns `Success: false`, `Stderr: "Invalid scriptPath 'python3': uv_run only executes standalone script files (.py). For directory listings use 'directory_reader', or save code to a .py script."` |
| **Valid Python Script** | `uv_run(scriptPath: "_bmad/scripts/memlog.py", args: ["init", ...])` | Passes validation; executes `uv run ...` | Standard execution output |
| **Token-Prefixed Script** | `uv_run(scriptPath: "{project-root}/_bmad/scripts/resolve_config.py", args: [...])` | Resolves tokens and executes valid `.py` script | Standard execution output |

</frozen-after-approval>

---

## Story

As an AI Subagent (e.g. `APF Brainstorm`, `APF Dev`, `APF Architect`),
I want clear, immediate validation and guidance when invoking `uv_run`,
So that if I erroneously attempt to invoke system commands (`find`, `ls`, `python3`), I am immediately directed to use the correct ADK tool (`directory_reader`, `file_reader`) rather than wasting execution cycles on failing subprocesses.

---

## Acceptance Criteria

1. **Given** an invocation of `uv_run` where `scriptPath` is a common shell or interpreter command (`find`, `ls`, `grep`, `cat`, `bash`, `sh`, `python`, `python3`), **When** `UVRunnerTool.Execute()` validates the input, **Then** it MUST immediately return `&UVRunnerOutput{Success: false, ExitCode: 1, Stderr: "..."}` containing explicit guidance recommending `directory_reader` or `file_reader` without launching a subprocess or holding a semaphore. (AC: 16.2.1)
2. **Given** an invocation of `uv_run` where `scriptPath` does not end with `.py` and does not resolve to an existing executable script, **When** validating the path, **Then** it MUST return a structured validation error explaining that `uv_run` requires a `.py` Python script. (AC: 16.2.2)
3. **Given** valid Python scripts located within `projectRoot` or `bmadRoot` (e.g. `_bmad/scripts/memlog.py`, `_bmad/scripts/resolve_config.py`), **When** validating, **Then** validation MUST pass and execute the script normally. (AC: 16.2.3)
4. **Given** unit tests in `ai/tools/uv_runner_test.go`, **When** running tests, **Then** test cases MUST verify rejection of `find`, `ls`, and `python3` with appropriate error messages and zero subprocess executions. (AC: 16.2.4)

---

## Tasks / Subtasks

- [ ] **Task 1: Add Script Path Validation to UVRunnerTool (AC: 16.2.1, 16.2.2)**
  - [ ] In `ai/tools/uv_runner.go`, add validation helper `validateScriptPath(scriptPath string) error`.
  - [ ] Check for forbidden binary command aliases (`find`, `ls`, `grep`, `cat`, `bash`, `sh`, `python`, `python3`, `echo`, `touch`, `mkdir`).
  - [ ] If matched, return a descriptive error pointing to `directory_reader` / `file_reader` / `file_writer`.
  - [ ] Verify script file extension is `.py` or a valid script in project or BMad root.
- [ ] **Task 2: Update Error Response Handling (AC: 16.2.1)**
  - [ ] Return `&UVRunnerOutput{Success: false, ExitCode: 1, Stderr: err.Error()}` on validation errors before acquiring semaphore.
- [ ] **Task 3: Add Unit Tests in uv_runner_test.go (AC: 16.2.4)**
  - [ ] Test `TestUVRunner_RejectsBinaryCommands` for `find`, `ls`, `python3`.
  - [ ] Verify no subprocess calls were made to `CommandExecutor`.
