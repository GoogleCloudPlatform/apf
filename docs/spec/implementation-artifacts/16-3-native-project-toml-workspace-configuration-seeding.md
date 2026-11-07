---
title: 'Story 16.3: Native Project TOML Workspace Configuration Seeding'
type: 'feature'
created: '2026-08-15'
status: 'ready-for-dev'
baseline_commit: '23b9c83f3428d854ba51161336cbff49e5124d02'
review_loop_iteration: 0
context: []
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** In APF, dynamic session parameters (`user_name`, `project_name`, `output_folder`, `planning_artifacts`, `implementation_artifacts`) are injected on-the-fly during `file_reader` tool calls by intercepting and regex-rewriting files like `_bmad/core/config.yaml`, `_bmad/bmm/config.yaml`, and `customize.toml`. This creates an unmaintainable dual-config paradigm where legacy YAML files conflict with BMad's native TOML configuration engine (`_bmad/config.toml`, `_bmad/custom/config.user.toml`).

**Approach:** Seed and maintain a clean, native `_bmad/custom/config.user.toml` configuration file in the project workspace mount during workspace initialization (`ai/core/workspace.go` and API project initialization). This TOML file will contain the exact user- and project-specific overrides, allowing BMad's standard `resolve_config.py` and `resolve_customization.py` scripts to resolve all configuration values natively via standard TOML layer merging, without requiring any in-flight text manipulation.

## Boundaries & Constraints

**Always:**
- Write/update `_bmad/custom/config.user.toml` in the project workspace during workspace initialization or session configuration updates.
- Structure `config.user.toml` with standard BMad TOML tables: `[core]`, `[modules.bmm]`, etc.
- Populate `user_name`, `project_name`, `output_folder`, `planning_artifacts`, and `implementation_artifacts` based on active GORM session state and `RepoConfiguration`.
- Ensure `_bmad/custom/` directory is created if it does not already exist.

**Never:**
- Do not overwrite custom user configurations outside the managed keys.
- Do not store unescaped path variables or invalid TOML syntax.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
| :--- | :--- | :--- | :--- |
| **Fresh Workspace Init** | User enters project workspace for first time | Creates `_bmad/custom/config.user.toml` with `[core]` and `[modules.bmm]` populated | Handles missing directories with `os.MkdirAll` |
| **Existing Custom Config** | User has already created custom settings in `config.user.toml` | Merges or updates managed fields while preserving existing user keys | Uses TOML parser/formatter to avoid data loss |
| **Default Fallbacks** | `user_name` or `project_name` missing in session state | Falls back cleanly to configuration defaults (`Foundry User`, `projectId`) | Non-blocking initialization |

</frozen-after-approval>

---

## Story

As a Developer and AI Agent Framework,
I want the project workspace to automatically maintain an up-to-date `_bmad/custom/config.user.toml` file,
So that BMad's native TOML configuration resolution scripts (`resolve_config.py`, `resolve_customization.py`) execute accurately with live project context without needing in-flight regex rewrites.

---

## Acceptance Criteria

1. **Given** a project workspace initialization in `ai/core/workspace.go`, **When** `NewWorkspace()` or workspace preparation is executed, **Then** the system MUST ensure the directory `<projectRoot>/_bmad/custom/` exists. (AC: 16.3.1)
2. **Given** active session parameters (`userId`, `projectName`, `specLocation`, `planningArtifactsLocation`, `implementationArtifactsLocation`), **When** seeding the project configuration, **Then** the system MUST write or update `<projectRoot>/_bmad/custom/config.user.toml` containing:
   ```toml
   [core]
   user_name = "<user_name>"
   project_name = "<project_name>"
   output_folder = "{project-root}/<specLocation>"

   [modules.bmm]
   planning_artifacts = "{project-root}/<planningArtifactsLocation>"
   implementation_artifacts = "{project-root}/<implementationArtifactsLocation>"
   project_knowledge = "{project-root}/<specLocation>"
   ```
   (AC: 16.3.2)
3. **Given** `_bmad/scripts/resolve_config.py` executed via `uv_run`, **When** resolving `--key core`, **Then** the script MUST output the merged JSON configuration reflecting the exact values seeded into `config.user.toml`. (AC: 16.3.3)
4. **Given** automated unit tests in `ai/core/workspace_unit_test.go`, **When** testing workspace seeding, **Then** tests MUST verify correct TOML formatting, token placement, and idempotent updates. (AC: 16.3.4)

---

## Tasks / Subtasks

- [ ] **Task 1: Implement TOML Configuration Seeder (AC: 16.3.1, 16.3.2)**
  - [ ] Create helper `SyncWorkspaceBmadConfig(projectRoot string, repoCfg *RepoConfiguration, bmadCfg *BmadConfiguration, userName string) error` in `ai/core/workspace.go` or `ai/core/bmad_configuration.go`.
  - [ ] Generate structured TOML string targeting `<projectRoot>/_bmad/custom/config.user.toml`.
  - [ ] Write file atomically with `0644` permissions.
- [ ] **Task 2: Hook Seeder into Workspace Initialization (AC: 16.3.1)**
  - [ ] Call `SyncWorkspaceBmadConfig` inside `NewWorkspace()` in `ai/core/workspace.go`.
  - [ ] Call `SyncWorkspaceBmadConfig` when session state changes relevant parameters.
- [ ] **Task 3: Add Unit & Integration Tests (AC: 16.3.4)**
  - [ ] Add unit test in `ai/core/workspace_unit_test.go` verifying file creation, content, and idempotency.
