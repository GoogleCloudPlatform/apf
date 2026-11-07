# Environment Setup

To get started with the AI project locally, copy the `env.template` file into a `.env` file inside this directory:

```bash
cp env.template .env
```

Open the `.env` file to fill in your values. Most baseline configurations are already pre-filled and can be left as-is. `GEMINI_API_KEY` in `.env` is referenced by `local.go` when seeding Application credentials into the database for local development (and is not used directly to initialize the `Configuration` class).

# Configurations

This section details the configuration parameters used by the agents.

| Name | Description | Default Value |
| :--- | :--- | :--- |
| `GCS_ARTIFACT_BUCKET` | GCS bucket for storing product configurations. This is the primary location for hosting steering instructions like bmad or other frameworks (from `configuration.go`) | *Empty* |
| `GEMINI_API_KEY` | Gemini API key (referenced in `local.go` for seeding local app credentials) | *Empty* |
| `DEFAULT_LLM_MODEL` | Default LLM model to use (from `configuration.go`) | `gemini-3.7-flash` |
| `SERVICE_NAME` | Name of the service. This is the name used in cloud run for publishing (from `configuration.go`) | `apf-ai` |
| `APP_URL` | This url is a full path to where ADK api is available. Used by ADK web as well as external callers invoking agents as API (from `configuration.go`) | `http://localhost:8080` |
| `AGENT_LOADER_NAME` | Name of the agent loader is used load specific ADK agent in a deployment. Using super makes all the agents available under the same service runtime (from `configuration.go`) | `super` |
| `GITHUB_ACCESS_TOKEN` | GitHub Access Token used for scaffolding golden paths. (from `configuration.go`) | *Empty* |
| `GITHUB_MCP_URL` | GitHub MCP URL used to interact with github to scaffold golden paths. (from `configuration.go`) | `https://api.githubcopilot.com/mcp/` |
| `ARTIFACT_SERVICE_TYPE` | Type of artifact service to use. GCS forces Google cloud storage based artifact service. any other value casuses a fallback to InMemory Artifact service. (from `configuration.go`) | `GCS` |
| `WORKSPACE_MOUNT_PATH` | Workspace mount path is the location of the mouting path in Cloud run, where the workspaces GCS bucket is mounted. (from `configuration.go`) | *Empty* |
| `LOG_LEVEL` | Logging level in the agents and associated tools (from `configuration.go`) | `INFO` |
| `BMAD_ROOT_PATH` | Root path for BMAD is a file system path which is mounted into the agent runtime to perform work. (from `bmad_configuration.go`) | `workspaces/config/bmad` |
| `BMAD_PLANNING_ARTIFACTS_PATH` | Path for BMAD planning artifacts is a location within BMAD_ROOT_PATH where planning artifacts are stored. (from `bmad_configuration.go`) | `docs/planning-artifacts` |
| `BMAD_IMPLEMENTATION_ARTIFACTS_PATH` | Path for BMAD implementation artifacts is a location within BMAD_ROOT_PATH where planning artifacts are stored. (from `bmad_configuration.go`) | `docs/planning-artifacts` |
