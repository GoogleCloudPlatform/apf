# Agentic Foundry API Server

This directory contains the backend API server for Agentic Foundry, implemented in Go.

---

## Project Seeding & Extraction Utility

We have a CLI utility designed for **project seeding** (importing templates and database records into a project workspace) and **project extraction** (exporting active workspaces back into shareable templates).

The utility lives in `cmd/seed_project`.

### How to Use

#### 1. Project Seeding (`-mode seed`)

Import files and register database artifacts for an existing project:

```bash
go run ./cmd/seed_project \
  -mode "seed" \
  -project-id "your-project-id" \
  -user-id "your-user-id" \
  -source "/path/to/source/template/folder" \
  -destination "/path/to/destination/workspace/folder" \
  -artifacts-json "/path/to/artifacts.json"
```

*During seeding, the utility automatically determines the artifact types matching your file paths if they are not pre-specified in the JSON.*

#### 2. Project Extraction (`-mode extract`)

Extract files and database artifacts from an existing project workspace to create a reusable template:

```bash
go run ./cmd/seed_project \
  -mode "extract" \
  -project-id "your-project-id" \
  -source "/path/to/destination/workspace/folder" \
  -destination "/path/to/target/template/folder" \
  -artifacts-json "/path/to/target/template/folder/artifacts.json"
```

*Note: If `-artifacts-json` is omitted in `extract` mode, it will automatically export to `artifacts.json` inside your target `-destination` folder.*

---

### CLI Flags Reference

| Flag | Type | Required | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `-mode` | string | No | `seed` | Operation mode: `seed` (to import) or `extract` (to export). |
| `-project-id` | string | **Yes** | - | The ID of the target project in Firestore. |
| `-user-id` | string | *Conditional* | - | The user ID owning the seeded data (required for `seed` mode). |
| `-source` | string | **Yes** | - | Path to the input source folder. |
| `-destination` | string | **Yes** | - | Target path where output files will be copied. |
| `-artifacts-json` | string | *Conditional* | - | Path to the artifacts list JSON file. |
