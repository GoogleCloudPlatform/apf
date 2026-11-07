---
title: 'Story 5.6: Cloud Run Container Image Packaging & Infrastructure Provisioning for uv Python Toolchain'
type: 'feature'
created: '2026-08-15'
status: 'done'
baseline_commit: '05df01fcbfa8c74ac2218819810fae85141fc9eb'
review_loop_iteration: 0
context:
  - file: docs/spec/planning-artifacts/epics.md
  - file: docs/spec/planning-artifacts/architecture.md
  - file: docs/spec/implementation-artifacts/5-5-sandboxed-uv-run-tool-execution-engine-for-bmad-scripts-and-python-utilities.md
  - file: ai/Dockerfile
  - file: infra/cicd/manifests/ai.yaml
  - file: infra/cicd/clouddeploy.yaml
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** The `apf-ai` Cloud Run runtime stage currently uses `gcr.io/distroless/static-debian12`, which lacks Python 3 and the `uv` toolchain. When agents invoke the `uv_run` tool in Cloud Run staging and production environments, executions fail with `exec: "uv": executable file not found in $PATH`. Furthermore, downloading Python libraries at runtime introduces latency spikes, egress bottlenecks, and failure risks during LLM interaction turns.

**Approach:**
1. Upgrade the runtime stage in `ai/Dockerfile` from `distroless` to `debian:bookworm-slim`, install system `python3`, `python3-pip`, and `python3-venv`, copy `/bin/uv` and `/bin/uvx` from `ghcr.io/astral-sh/uv:latest`, pre-install standard BMad Python dependencies (`pyyaml`, `tomli`, `jinja2`) via `pip install --no-cache-dir --break-system-packages`, and configure default environment variables (`UV_SYSTEM_PYTHON=1`, `UV_CACHE_DIR=/tmp/uv-cache`, `HOME=/tmp`, `PYTHONUNBUFFERED=1`).
2. Parameterize `container_concurrency: "80"` in `infra/cicd/clouddeploy.yaml` for both `apf-staging` and `apf-production` targets, and wire `${container_concurrency}` into `infra/cicd/manifests/ai.yaml` for `containerConcurrency` and `CONTAINER_CONCURRENCY` env variable.
3. Validate container build, entrypoint execution, and dependency importing with zero network latency.

## Boundaries & Constraints

**Always:**
- Use a lean `debian:bookworm-slim` base runtime image to run the Go application binary while providing a complete Debian glibc and Python 3 execution environment.
- Copy `/uv` and `/uvx` from official `ghcr.io/astral-sh/uv:latest` into `/bin/`.
- Pre-install BMad Python dependencies (`pyyaml`, `tomli`, `jinja2`) via `pip install --no-cache-dir --break-system-packages pyyaml tomli jinja2` during the Docker image build.
- Define default container environment variables `UV_SYSTEM_PYTHON=1`, `UV_CACHE_DIR=/tmp/uv-cache`, `HOME=/tmp`, and `PYTHONUNBUFFERED=1`.
- Rely on Cloud Run gen2 in-memory tmpfs `/tmp` (within the 8Gi RAM limit) for `UV_CACHE_DIR=/tmp/uv-cache`.
- Parameterize `container_concurrency` via `infra/cicd/clouddeploy.yaml` and reference it in `infra/cicd/manifests/ai.yaml`.
- Maintain existing non-root or security guidelines and ensure `/apf` binary and `/config/bmad` directories remain accessible with execution permissions (`chmod 0755`).

**Never:**
- Do not install heavyweight build toolchains (e.g. gcc, g++, make) in the final runtime stage.
- Do not allow runtime Python package downloads during user chat turns.
- Do not hardcode organization or project-specific paths into the container image.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
| :--- | :--- | :--- | :--- |
| **Container Startup & Binary Check** | Container starts in Cloud Run or Docker | `uv --version` and `python3 --version` succeed; `/apf` entrypoint initializes | If `uv` missing, fail image build immediately |
| **Pre-installed Dependency Import** | `uv run python3 -c "import yaml, tomli, jinja2; print('OK')"` | Outputs `OK` in < 50ms without network access | If import fails, build fails during docker verification |
| **Tmpfs Cache Directory** | Script execution with `UV_CACHE_DIR=/tmp/uv-cache` | Fast in-memory cache operations on Cloud Run tmpfs; no disk contention | Cleaned up on container termination |
| **Cloud Run High Concurrency** | 80 concurrent agent requests on single instance | Cloud Run instance concurrency limit (${container_concurrency} = 80) aligns with tool semaphore | No container throttling or semaphore starvation |

</frozen-after-approval>

---

## Story

As a Platform Engineer and DevOps Specialist,
I want the `apf-ai` Cloud Run container image and deployment manifests to package the `uv` toolchain, system Python 3 runtime, and pre-installed BMad dependencies,
So that `uv_run` script executions run with sub-second startup times, zero network dependency fetching, and guaranteed container-level concurrency alignment in staging and production environments.

---

## Acceptance Criteria

1. **Given** the `ai/Dockerfile` build pipeline, **When** the runtime container image is assembled, **Then** the runtime stage MUST use a `debian:bookworm-slim` base image with `python3`, `python3-pip`, and `python3-venv` installed, containing `/bin/uv` and `/bin/uvx` copied from `ghcr.io/astral-sh/uv:latest` on the system `PATH`. (AC: 5.6.1)
2. **Given** container image assembly, **When** building the image, **Then** the image MUST pre-install standard BMad script dependencies (`pyyaml`, `tomli`, `jinja2`) into the system Python environment via `pip install --no-cache-dir --break-system-packages`. (AC: 5.6.2)
3. **Given** container runtime execution, **When** the container starts, **Then** default environment variables `UV_SYSTEM_PYTHON=1`, `UV_CACHE_DIR=/tmp/uv-cache`, `HOME=/tmp`, and `PYTHONUNBUFFERED=1` MUST be configured in the container image and/or manifest. (AC: 5.6.3)
4. **Given** Cloud Deploy and Cloud Run deployment configuration, **When** deploying `infra/cicd/manifests/ai.yaml` via `infra/cicd/clouddeploy.yaml`, **Then** `clouddeploy.yaml` MUST parameterize `container_concurrency: "80"` for `apf-staging` and `apf-production` targets, and `ai.yaml` MUST specify `containerConcurrency: 80 # from-param: ${container_concurrency}` under `spec.template.spec` and inject `CONTAINER_CONCURRENCY` into the container `env` block. (AC: 5.6.4)
5. **Given** the compiled container image, **When** testing container sanity, **Then** `uv --version`, `python3 --version`, and `uv run python3 -c "import yaml, tomli, jinja2; print('ok')"` MUST execute successfully with exit code 0. (AC: 5.6.5)

---

## Developer Context & Technical Guardrails

### 1. `ai/Dockerfile` Multi-Stage Architecture

The `ai/Dockerfile` currently has two stages:
1. `builder`: `golang:1.26.1-alpine` compiling `/app/ai/apf` and running `/app/third_party/bmad/package.go --install-dir /config/bmad --catalog-file ../api/seeding/templates/showcase/bmad_catalog.json`.
2. `runtime`: `debian:bookworm-slim`.

**Required Changes to Runtime Stage:**
```dockerfile
# Runtime stage
FROM debian:bookworm-slim

WORKDIR /

# Install Python 3, pip, venv, and ca-certificates
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        python3 \
        python3-pip \
        python3-venv \
        ca-certificates && \
    rm -rf /var/lib/apt/lists/*

# Install uv binary from official multi-arch Astral image
COPY --from=ghcr.io/astral-sh/uv:latest /uv /uvx /bin/

# Pre-install standard BMad Python dependencies into system Python
RUN pip install --no-cache-dir --break-system-packages pyyaml tomli jinja2

# Set Cloud Run runtime performance and caching environment variables
ENV UV_SYSTEM_PYTHON=1 \
    UV_CACHE_DIR=/tmp/uv-cache \
    HOME=/tmp \
    PYTHONUNBUFFERED=1

# Copy compiled Go application binary and BMad artifacts from builder
COPY --from=builder /app/ai/apf /apf
COPY --from=builder /app/ai/sub_agents/scaffolder/service-catalog /sub_agents/scaffolder/service-catalog
COPY --from=builder /config/bmad /config/bmad

ARG COMMIT_SHA=""
ENV COMMIT_SHA=${COMMIT_SHA}

EXPOSE 8080

ENTRYPOINT ["/apf", "web", "-read-timeout", "5m", "-write-timeout", "5m", "api", "a2a", "webui"]
```

### 2. `infra/cicd/clouddeploy.yaml` & `infra/cicd/manifests/ai.yaml` Configuration

In `infra/cicd/clouddeploy.yaml`, add `container_concurrency: "80"` to both staging and production targets:
```yaml
deployParameters:
  ...
  container_concurrency: "80"
```

In `infra/cicd/manifests/ai.yaml`, add:
```yaml
spec:
  template:
    metadata:
      ...
    spec:
      containerConcurrency: 80 # from-param: ${container_concurrency}
      serviceAccountName: ...
      containers:
        - image: apf-ai
          ports:
            - containerPort: 8080
          resources:
            limits:
              cpu: "2"
              memory: 8Gi
          env:
            - name: CONTAINER_CONCURRENCY
              value: "80" # from-param: ${container_concurrency}
            - name: UV_SYSTEM_PYTHON
              value: "1"
            - name: UV_CACHE_DIR
              value: "/tmp/uv-cache"
```

### 3. Previous Story Intelligence (Story 5.5)

* In Story 5.5, `ai/core/configuration.go` was updated to read `CONTAINER_CONCURRENCY` / `CONCURRENCY` (defaulting to 80), initializing the `UVRunnerTool` semaphore with capacity matching Cloud Run concurrency.
* `ai/tools/uv_runner.go` passes `UV_SYSTEM_PYTHON=1`, `UV_CACHE_DIR=/tmp/uv-cache`, and `HOME=/tmp` to all subprocesses.
* Having Python and `uv` pre-packaged in the container ensures `TestUVRunner_RealUV` and production agent turns execute seamlessly with sub-second execution latency.

---

## Tasks / Subtasks

- [x] **Task 1: Update `ai/Dockerfile` Runtime Stage (AC: 5.6.1, 5.6.2, 5.6.3)**
  - [x] Switch runtime stage from `gcr.io/distroless/static-debian12` to `debian:bookworm-slim`.
  - [x] Install `python3`, `python3-pip`, `python3-venv`, and `ca-certificates`.
  - [x] Copy `uv` and `uvx` binaries from `ghcr.io/astral-sh/uv:latest` into `/bin/`.
  - [x] Pre-install `pyyaml`, `tomli`, `jinja2` via `pip install --no-cache-dir --break-system-packages`.
  - [x] Set environment variables: `ENV UV_SYSTEM_PYTHON=1 UV_CACHE_DIR=/tmp/uv-cache HOME=/tmp PYTHONUNBUFFERED=1`.
  - [x] Copy `/app/ai/apf`, `/sub_agents/scaffolder/service-catalog`, and `/config/bmad` into the runtime container.
- [x] **Task 2: Update Cloud Deploy & Cloud Run Deployment Manifests (AC: 5.6.4)**
  - [x] In `infra/cicd/clouddeploy.yaml`, add `container_concurrency: "80"` to `apf-staging` and `apf-production` deploy parameters.
  - [x] In `infra/cicd/manifests/ai.yaml`, add `containerConcurrency: 80 # from-param: ${container_concurrency}` under `spec.template.spec`.
  - [x] In `infra/cicd/manifests/ai.yaml`, add `CONTAINER_CONCURRENCY`, `UV_SYSTEM_PYTHON`, and `UV_CACHE_DIR` to `env`.
- [x] **Task 3: Sanity Test Container Image Build & Script Execution (AC: 5.6.5)**
  - [x] Validate Dockerfile syntax and build container image.
  - [x] Run container smoke test verifying `uv`, `python3`, and standard package imports (`yaml`, `tomli`, `jinja2`).
  - [x] Run all Go test suites across `./ai/...` and `./api/...`.

---

## Code Map

- `ai/Dockerfile` -- `UPDATE` Upgrade runtime stage to `debian:bookworm-slim`, install python3/pip/venv, copy `uv` binary, pre-install python dependencies, and set `UV_*` environment variables.
- `infra/cicd/clouddeploy.yaml` -- `UPDATE` Add `container_concurrency: "80"` to staging and production targets.
- `infra/cicd/manifests/ai.yaml` -- `UPDATE` Set `containerConcurrency` from parameter and pass `CONTAINER_CONCURRENCY`, `UV_SYSTEM_PYTHON`, and `UV_CACHE_DIR` environment variables.

---

## Verification Plan

### Automated / Container Tests
1. Verify Dockerfile build and container entrypoints:
   ```bash
   docker build -t apf-ai-test -f ai/Dockerfile .
   docker run --rm --entrypoint uv apf-ai-test --version
   docker run --rm --entrypoint python3 apf-ai-test --version
   docker run --rm --entrypoint python3 apf-ai-test -c "import yaml, tomli, jinja2; print('python imports ok')"
   docker run --rm --entrypoint uv apf-ai-test run python3 -c "import yaml, tomli, jinja2; print('uv run ok')"
   ```
2. Verify Go tests pass:
   ```bash
   go test -count=1 ./ai/... ./api/...
   ```

---

## Review Triage Log

### 2026-08-15 — Review pass
- intent_gap: 0
- bad_spec: 0
- patch: 0
- defer: 0
- reject: 0
- addressed_findings:
  - none

---

## Auto Run Result

Status: done
Summary: Upgraded `ai/Dockerfile` to use `debian:bookworm-slim` with `python3`, `uv` multi-stage binary packaging, and pre-installed `pyyaml`, `tomli`, and `jinja2` packages. Parameterized Cloud Deploy targets with `container_concurrency: "80"` and mapped Cloud Run manifests in `infra/cicd/manifests/ai.yaml`. Verified container image build, `uv` command execution, offline python package imports, and Go test suites.

### Files Changed:
- `ai/Dockerfile`: Upgraded runtime stage to `debian:bookworm-slim`, installed Python 3, copied `/bin/uv` and `/bin/uvx` from `ghcr.io/astral-sh/uv:latest`, pre-installed `pyyaml tomli jinja2`, and configured `UV_SYSTEM_PYTHON=1`, `UV_CACHE_DIR=/tmp/uv-cache`, and `HOME=/tmp`.
- `infra/cicd/clouddeploy.yaml`: Added `container_concurrency: "80"` to `apf-staging` and `apf-production` deployment targets.
- `infra/cicd/manifests/ai.yaml`: Added `containerConcurrency: 80 # from-param: ${container_concurrency}` under `spec.template.spec` and injected `CONTAINER_CONCURRENCY`, `UV_SYSTEM_PYTHON`, and `UV_CACHE_DIR` into container `env`.

### Verification Performed:
- `docker build -t apf-ai-test -f ai/Dockerfile .`: PASS (built image successfully in 78.9s).
- `docker run --rm --entrypoint uv apf-ai-test --version`: PASS (`uv 0.12.5 (x86_64-unknown-linux-musl)`).
- `docker run --rm --entrypoint python3 apf-ai-test --version`: PASS (`Python 3.11.2`).
- `docker run --rm --entrypoint python3 apf-ai-test -c "import yaml, tomli, jinja2; print('python imports ok')"`: PASS (`python imports ok`).
- `docker run --rm --entrypoint uv apf-ai-test run python3 -c "import yaml, tomli, jinja2; print('uv run ok')"`: PASS (`uv run ok`).
- `go test -count=1 ./ai/... ./api/...`: PASS (all repository Go tests passed).

