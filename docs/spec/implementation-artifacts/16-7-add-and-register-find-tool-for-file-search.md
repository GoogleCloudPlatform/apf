---
title: 'Story 16.7: Add and Register Find Tool for File Search Across All Agent Swarms'
type: 'feature'
created: '2026-08-15'
status: 'done'
baseline_commit: '4d61d85c88b0a991ee787e35a1ffaa1f6230ee8e'
baseline_revision: '4d61d85e942d5026c6dd9b3b869beeee4e7e8245'
review_loop_iteration: 0
followup_review_recommended: false
deferred:
  - summary: >-
      Add search result count limit / pagination parameter to FindInput for large queries.
    evidence: |-
      Large codebases or unbounded searches could produce excessive output without limit parameter.
    location: >-
      ai/tools/find.go
    severity: low
context: []
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** When LLM agents across swarms need to discover files, locate nested assets by pattern or name, or recursively scan project directories (such as finding `*.go` files, `SKILL.md`, `.memlog.md`, `project-context.md`, or configuration files), they currently only have `directory_reader` (which performs shallow 2-3 level directory listing) or `file_reader` (which reads single known file paths). Because there is no dedicated recursive file-finding tool, LLMs frequently attempt to run `find` or `python3` via `uv_run` or fail when searching deeply nested directory structures, resulting in command hallucination and broken workflows.

**Approach:** Implement a dedicated, performant `FindTool` (`find_files`) in `ai/tools/find.go` that supports recursive file search with pattern matching (glob/substring), path exclusions, depth control, and type filtering ("file", "directory", "any"). Expose `GetFindTool()` via `ai/sub_agents/tools.go` and register the tool in the `Tools` slice of all agent swarms (`Analysis`, `Help`, `Planning`, `Solutioning`, `Implementation`, `Overwatch`) for both root agents and catalog subagents. Add comprehensive unit tests in `ai/tools/find_test.go` and update agent loader test suites.

## Boundaries & Constraints

**Always:**
- Implement `FindTool` in `ai/tools/find.go` with tool name `find_files` and input schema `{ directoryPath, pattern, maxDepth, type, excludes }`.
- Default `directoryPath` to the current project/workspace root (`"."`) if empty or unspecified.
- Support glob matching (e.g. `*.go`, `*test*`, `**/*.md`) and case-insensitive substring matching.
- Exclude standard ignored directories by default (e.g. `.git`, `node_modules`, `vendor`) unless explicitly requested.
- Provide safe traversal within project boundaries; never panic or return nil errors on non-existent directories.
- Export `GetFindTool() (tool.Tool, error)` in `ai/sub_agents/tools.go`.
- Register `findTool` across every catalog subagent and root agent in `analysis`, `help`, `planning`, `solutioning`, `implementation`, and `overwatch`.
- Maintain existing tools (`fileReaderTool`, `fileWriterTool`, `directoryReaderTool`, `uvRunnerTool`, `suggestPhaseSwitchTool`) across all agents.

**Never:**
- Do not return file contents in `FindTool` (only return matched file paths and count) to preserve LLM token context.
- Do not allow filesystem traversal outside the authorized workspace sandbox.
- Do not block or panic on unreadable subdirectories or symlink loops.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
| :--- | :--- | :--- | :--- |
| **Recursive Extension Search** | `find_files(directoryPath: "ai", pattern: "*.go")` | Returns all Go source files nested under `ai/` recursively | Returns empty list if no files match pattern |
| **Filename Search** | `find_files(pattern: "SKILL.md")` | Returns all `SKILL.md` paths across the workspace | Returns empty list if not found |
| **Depth-Limited Search** | `find_files(directoryPath: ".", pattern: "*", maxDepth: 1)` | Returns only top-level items without recursing deeper | Respects `maxDepth` boundary |
| **Type Filter (Directories Only)** | `find_files(directoryPath: "docs", type: "directory")` | Returns only directory paths under `docs/` | Excludes regular files |
| **Exclusion Filtering** | `find_files(pattern: "*.json", excludes: ["node_modules", ".git"])` | Matches JSON files while skipping excluded paths | Skips walking excluded subtrees |
| **Non-Existent Directory** | `find_files(directoryPath: "non/existent/path", pattern: "*")` | Returns `&FindOutput{FilesPaths: [], TotalFound: 0}` with warning log | Clean return with zero exit/panic |

</frozen-after-approval>

---

## Story

As an AI Subagent in any APF agent swarm (e.g. `APF Brainstorm`, `APF Dev`, `APF Architect`, `APF Help`),
I want a dedicated, recursive file-finding tool (`find_files`),
So that I can quickly and reliably locate files across the project workspace by name, extension, or glob pattern without resorting to unsupported shell commands.

---

## Acceptance Criteria

1. **Given** the `ai/tools` package, **When** `FindTool.Execute()` is called with a directory path and pattern, **Then** it MUST recursively search the filesystem, filter matches by pattern, type, and exclusions, and return `*FindOutput` containing `FilesPaths` and `TotalFound`. (AC: 16.7.1)
2. **Given** `ai/sub_agents/tools.go`, **When** initializing swarm tools, **Then** `subagents.GetFindTool()` MUST return an ADK `tool.Tool` instance wrapping `FindTool`. (AC: 16.7.2)
3. **Given** agent loader factories in `analysis`, `help`, `planning`, `solutioning`, `implementation`, and `overwatch`, **When** calling `NewAgentLoader()`, **Then** the loader MUST instantiate `findTool` and register it in the `Tools` slice for all root agents and catalog subagents. (AC: 16.7.3)
4. **Given** unit tests in `ai/tools/find_test.go` and `ai/sub_agents/*_test.go`, **When** running `go test ./ai/...`, **Then** all tests MUST pass and verify pattern matching, exclusions, non-existent directory handling, and tool registration across all agents. (AC: 16.7.4)

---

## Tasks / Subtasks

- [x] **Task 1: Implement FindTool in `ai/tools/find.go` (AC: 16.7.1)**
  - [x] Define `FindInput` struct (`DirectoryPath`, `Pattern`, `MaxDepth`, `Type`, `Excludes`).
  - [x] Define `FindOutput` struct (`FilesPaths []string`, `TotalFound int`).
  - [x] Implement `FindFiles(ctx tool.Context, input *FindInput) (*FindOutput, error)` with recursive directory walking, glob pattern matching, and exclusion filtering.
  - [x] Implement `FindTool` struct with `Name() string` returning `"find_files"` and descriptive `Description()`.
- [x] **Task 2: Expose GetFindTool in `ai/sub_agents/tools.go` (AC: 16.7.2)**
  - [x] Add `func GetFindTool() (tool.Tool, error)` to `ai/sub_agents/tools.go`.
- [x] **Task 3: Register FindTool across All Agent Swarms (AC: 16.7.3)**
  - [x] Update `ai/sub_agents/analysis/agent.go` to instantiate and register `findTool` on root agent and all subagents.
  - [x] Update `ai/sub_agents/help/agent.go` to instantiate and register `findTool` on root agent and all subagents.
  - [x] Update `ai/sub_agents/planning/agent.go` to instantiate and register `findTool` on root agent and all subagents.
  - [x] Update `ai/sub_agents/solutioning/agent.go` to instantiate and register `findTool` on root agent and all subagents.
  - [x] Update `ai/sub_agents/implementation/agent.go` to instantiate and register `findTool` on root agent and all subagents.
  - [x] Update `ai/sub_agents/overwatch/agent.go` to instantiate and register `findTool` on root agent and all subagents.
- [x] **Task 4: Implement Unit Tests & Verification (AC: 16.7.4)**
  - [x] Create `ai/tools/find_test.go` covering recursive search, glob matching, exclusions, max depth limits, and non-existent paths.
  - [x] Update `ai/sub_agents/analysis/agent_test.go`, `help/agent_test.go`, `planning/agent_test.go`, `solutioning/agent_test.go`, `implementation/agent_test.go`, and `overwatch/agent_test.go` to assert presence of `find_files` on root and subagents.
  - [x] Run `go test ./ai/...` and verify clean execution.

## Review Triage Log

### 2026-08-15 — Review pass
- intent_gap: 0
- bad_spec: 0
- patch: 2: (high 0, medium 0, low 2)
- defer: 1: (high 0, medium 0, low 1)
- reject: 10
- addressed_findings:
  - `[low]` `[patch]` Precompiled glob regular expression once prior to directory traversal and wired context cancellation checks in `walkFindDir`.
  - `[low]` `[patch]` Initialized `results` slice to prevent nil JSON serialization on single-file non-matches.

## Auto Run Result

### Summary of Implemented Change
Implemented a dedicated, recursive file finding tool `FindTool` (`find_files`) in `ai/tools/find.go` supporting glob pattern matching, case-insensitive substring search, path exclusions (with `.git`, `node_modules`, `vendor` default exclusions), max depth limiting, and type filtering (`"file"`, `"directory"`, `"any"`). Exported `GetFindTool()` via `ai/sub_agents/tools.go` and registered the tool across all six APF agent swarms (`Analysis`, `Help`, `Planning`, `Solutioning`, `Implementation`, `Overwatch`) for both root agents and catalog subagents.

### Files Changed
- `ai/tools/find.go`: New tool implementation for `FindTool` (`find_files`).
- `ai/tools/find_test.go`: Comprehensive unit tests covering all search options, patterns, exclusions, depth, and edge cases.
- `ai/sub_agents/tools.go`: Exported `GetFindTool() (tool.Tool, error)`.
- `ai/sub_agents/analysis/agent.go`: Registered `findTool` on root agent and all subagents.
- `ai/sub_agents/analysis/agent_test.go`: Verified `find_files` presence on root and subagents.
- `ai/sub_agents/help/agent.go`: Registered `findTool` on root agent and all subagents.
- `ai/sub_agents/help/agent_test.go`: Verified `find_files` presence on root and subagents.
- `ai/sub_agents/planning/agent.go`: Registered `findTool` on root agent and all subagents.
- `ai/sub_agents/planning/agent_test.go`: Verified `find_files` presence on root and subagents.
- `ai/sub_agents/solutioning/agent.go`: Registered `findTool` on root agent and all subagents.
- `ai/sub_agents/solutioning/agent_test.go`: Verified `find_files` presence on root and subagents.
- `ai/sub_agents/implementation/agent.go`: Registered `findTool` on root agent and all subagents.
- `ai/sub_agents/implementation/agent_test.go`: Verified `find_files` presence on root and subagents.
- `ai/sub_agents/overwatch/agent.go`: Registered `findTool` on root agent and all subagents.
- `ai/sub_agents/overwatch/agent_test.go`: Verified `find_files` presence on root and subagents.
- `docs/spec/implementation-artifacts/16-7-add-and-register-find-tool-for-file-search.md`: Story specification, tasks, review log, and results.

### Review Findings Breakdown
- Patches applied: 2 (low severity: regex precompilation & context cancellation support, non-nil slice initialization)
- Items deferred: 1 (limit/maxResults parameter on FindInput)
- Items rejected: 10 (noise / duplicate / out-of-scope SDK questions)

### Follow-up Review Recommendation
- Patched count: 2 (low 2, medium 0, high 0)
- Score: 2 (`3 × 0 + 1 × 2 = 2`)
- Follow-up review recommended: `false`

### Verification Performed
- `go test -count=1 -v ./ai/tools/... ./ai/sub_agents/...` passed with 100% success across 11 test suites.
- `go test -count=1 ./ai/... ./api/... ./runners/tmux/...` passed the full repository test suite.

### Residual Risks
- None. All agent loaders and tool registrations are verified with zero test failures or regressions.

