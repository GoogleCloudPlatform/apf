<!-- omit in toc -->
# Contributing to Agentic Product Foundry

# How to contribute

We'd love to accept your patches and contributions to this project.

## Before you begin

### Sign our Contributor License Agreement

Contributions to this project must be accompanied by a
[Contributor License Agreement](https://cla.developers.google.com/about) (CLA).
You (or your employer) retain the copyright to your contribution; this simply
gives us permission to use and redistribute your contributions as part of the
project.

If you or your current employer have already signed the Google CLA (even if it
was for a different project), you probably don't need to do it again.

Visit <https://cla.developers.google.com/> to see your current agreements or to
sign a new one.

### Review our community guidelines

This project follows
[Google's Open Source Community Guidelines](https://opensource.google/conduct/).

## Contribution process

All types of contributions are encouraged and valued. See the [Table of Contents](#table-of-contents) for different ways to help and details about how this project handles them. Please read the relevant sections before making your contribution to ensure a smooth onboarding and review process. 🎉

### Code reviews

All submissions, including submissions by project members, require review. We
use GitHub pull requests for this purpose. Consult
[GitHub Help](https://help.github.com/articles/about-pull-requests/) for more
information on using pull requests.

<!-- omit in toc -->
## Table of Contents

- 🚣 **[Start Committing Code](#start-committing-code)**
  - [Host Prerequisites](#host-prerequisites)
  - [Step 1: Install the Mise CLI](#step-1-install-the-mise-cli)
  - [Step 2: Automated Environment Setup (`mise run env-setup`)](#step-2-automated-environment-setup-mise-run-env-setup)
  - [Step 3: Spin Up the Local Stack (`mise run local`)](#step-3-spin-up-the-local-stack-mise-run-local)
  - [Local Simulation Login Credentials](#-local-simulation-login-credentials)
  - [Daily Maintenance & Cleanup Workflows](#-daily-maintenance--cleanup-workflows)
  - [IDE Launch Parity (VS Code / Antigravity)](#-ide-launch-parity-vs-code--antigravity)
  - [Troubleshooting](#-troubleshooting)
- 🏗️ **[Repository Architecture](#️-repository-architecture)**
- 🔀 **[Your First Code Contribution (PR Workflow)](#-your-first-code-contribution-pr-workflow)**
- 🧪 **[Testing & Quality Assurance](#-testing--quality-assurance)**
  - [1. Backend Go Tests (`ai/` & `api/`)](#1-backend-go-tests-ai--api)
  - [2. Frontend Tests (`ui/`)](#2-frontend-tests-ui)
  - [3. End-to-End Browser Tests (Playwright)](#3-end-to-end-browser-tests-playwright)
  - [4. Performance & Smoke Tests (k6)](#4-performance--smoke-tests-k6)
  - [5. Mise Orchestration Verification](#5-mise-orchestration-verification)
- 🗄️ **[Database Emulators & Cloud SQL Proxy](#️-database-emulators--cloud-sql-proxy)**
  - [Local Firestore Emulator & Seeding](#local-firestore-emulator--seeding)
  - [Local PostgreSQL Session Database](#local-postgresql-session-database)
  - [Cloud SQL Proxy (Optional Staging Access)](#cloud-sql-proxy-optional-staging-access)
- 📜 **[Styleguides & Conventions](#-styleguides--conventions)**
  - [Commit Messages (Conventional Commits)](#commit-messages-conventional-commits)
  - [Go Coding Standards](#go-coding-standards)
  - [TypeScript & React Standards](#typescript--react-standards)
  - [Documentation Guidelines](#documentation-guidelines)
- 🐞 **[Reporting Bugs & Suggesting Enhancements](#-reporting-bugs--suggesting-enhancements)**
  - [Reporting Bugs](#reporting-bugs)
  - [Suggesting Enhancements](#suggesting-enhancements)
- 🔒 **[Security Vulnerability Reporting](#-security-vulnerability-reporting)**
- 🤝 **[Code of Conduct & Community](#-code-of-conduct--community)**

---

## 🚣 Start Committing Code

To make your contribution experience as smooth and frictionless as possible, we use `mise` as our single source of truth for toolchain management and task orchestration. This provides instant, sandboxed onboarding without polluting your host OS.

### Host Prerequisites
Before setting up the project, ensure you have the following installed on your host machine:
- **Docker & Docker Compose**: Required for running the local PostgreSQL session database container.
- **gcloud CLI**: Required for GCP service authentication and accessing cloud resources.
- **Git**: For cloning the repository and version control.

---

### Step 1: Install the Mise CLI
If you do not have `mise` installed on your machine yet, open your terminal and run the official installer:
```bash
# 1. Download and execute the mise installer
curl https://mise.run | sh

# 2. Add mise to your active shell profile (e.g., ~/.bashrc or ~/.zshrc)
echo 'eval "$(~/.local/bin/mise activate bash)"' >> ~/.bashrc
source ~/.bashrc

# 3. Verify installation
mise --version

# 4. Trust the repository configuration
mise trust
```

> [!NOTE]
> **Mise Trust**: If you encounter warnings or permission errors regarding untrusted configuration files when entering the repository or running tasks, run `mise trust` in your terminal. This marks the repository's `mise.toml` configuration as trusted by your local machine.

---

### Step 2: Automated Environment Setup (`mise run env-setup`)
Once `mise` is active, navigate to the repository root and execute the master onboarding task:
```bash
mise run env-setup
```

This task automatically:
1. Installs all required sandboxed runtimes and tools via Mise (**Go 1.26.1, Python 3.13, Node 20, Java 21, Terraform, jq, air, Cloud SQL Proxy, Skaffold, k6**).
2. Verifies and installs the Firebase CLI (`firebase-tools`) via npm if missing.
3. Creates a local `config.env` file from the template (`config.env.template`) and prompts you securely in the terminal for required environment variables.
4. Generates and synchronizes individual service environment files (`ai/.env`, `api/.env`, and `ui/.env.local`).
5. Packages and installs optimized BMad skills into `./ai/workspaces/config/bmad` (`mise run bmad:package ./ai/workspaces/config/bmad`).

> [!IMPORTANT]
> These generated credentials and key files are strictly gitignored to guarantee your sensitive keys are never accidentally committed to the repository.

---

### Step 3: Spin Up the Local Stack (`mise run local`)
To launch the entire application stack locally, simply run:
```bash
mise run local
```

The local development orchestration is split into two modular parts (`scripts/start_all.sh` runs Part 1 followed by Part 2):

#### Part 1: Database Containers & Data Seeding (`mise run dev-db`)
Can also be run standalone via:
```bash
mise run dev-db   # Aliases: mise run db:local, mise run db:setup-local
```
1. **Local Postgres DB**: Launches a local Docker container (`apf-local-session-db`) on port `5432` and `pgadmin` on port `8083`.
2. **Firestore & Auth Emulators**: Starts local Firestore emulator on port `8084` and Auth emulator on port `8085` (`http://localhost:4000/firestore`).
3. **Data Seeding**: Waits for emulators and databases to initialize, seeds Firestore test data (`api/seeding -env local`) and Postgres session database (`ai/seed/main.go`).
*Part 1 runs detached and stays running in the background after startup completes.*

#### Part 2: Application Processes (`mise run dev-apps`)
Can also be run standalone via:
```bash
mise run dev-apps   # Alias: mise run local-apps
```
Starts all 3 core application processes:
1. **AI Service**: Starts on `http://localhost:8080` (`logs/ai.log`).
2. **API Service**: Starts on `http://localhost:8081` (`logs/api.log`).
3. **UI Frontend**: Starts Next.js frontend on `http://localhost:8082` (`logs/ui.log`).

When running application processes, press `Ctrl+R` at any time to cleanly restart application services (`Ai, Api, Ui`), or press `Ctrl+C` to cleanly stop application services without terminating your background database containers.

---

### 🔑 Local Simulation Login Credentials

The local environment uses a **Simulation Authenticator** for speed and convenience. You can log in using your `SIMULATED_USER_ID` or the following simulated test roles:

| Role | Username | Password (Simulation) | Description |
| :--- | :--- | :--- | :--- |
| 👑 **Global Super Admin** | `super-admin` | `super-admin` *(Strict)* | Bypasses all organization-level checks, accesses the Organization Provisioning Portal. |
| 🏢 **Enterprise Admin** | `admin` | `admin` *(Strict)* | Full administrative permissions within their selected organization. |
| 🛠️ **Project Creator** | `user1` | *Any password* (e.g. `user123`) | Can create and configure projects inside their organization. |
| 👤 **Standard User** | `user2` | *Any password* (e.g. `user123`) | General member access within their organization. |
| 👁️ **Standard Viewer** | `viewer` | *Any password* (e.g. `user123`) | Read-only viewer access within their organization. |

---

### 💡 Daily Maintenance & Cleanup Workflows

To synchronize environment configurations and update tool versions when pulled from upstream, run:
```bash
# Sync env variables and update dependencies
mise run daily
```

If you ever encounter port collision errors (`address already in use`) or need to clean up lingering background processes running on target ports (`8080`, `8081`, `8082`, `8084`, `8085`, `4000`, `4400`, `4500`), run:
```bash
# Force-kill all background daemons and stop local DB
mise run clean
```

---

### 🖥️ IDE Launch Parity (VS Code / Antigravity)

If you prefer VS Code or Antigravity workflows, you can trigger these tasks from the IDE:
- Open the Command Palette (`Ctrl+Shift+P` / `Cmd+Shift+P`), select **Tasks: Run Task**, and choose:
  - `🚀 Launch All Services (Mise)`
  - `🗄️ Launch Local Databases & Seed (Part 1)`
  - `🚀 Launch Application Processes (Part 2)`
  - `🛠️ Setup Developer Environment (Mise)`
  - `🛑 Stop All Services (Mise)`

You can also use the native `.vscode/launch.json` debug configs (**Launch All**, **Launch Backends**, **Ui**, **Ai**, **Api**, **Package BMAD**). Under the hood, these configs execute matching `mise run` commands to ensure absolute parity between IDE and CLI.

---

### 🛠️ Troubleshooting

#### 1. The UI White Screen of Death (Caching)
If `http://localhost:8082/` shows the loading bar but stays blank, your browser is likely caching a stale JS bundle.
- **Fix**: Open the page in Incognito or perform a hard refresh (`Cmd+Shift+R` or `Ctrl+F5`).

#### 2. Address Already in Use
If Go or Next.js servers fail to start because of port conflicts:
- **Fix**: Run `mise run clean` to force-kill lingering background processes on target ports.

#### 3. `invalid_rapt` / `invalid_grant` Authentication Errors
If you encounter re-authentication errors when the local backend communicates with Google Cloud services:
- **Fix**: Run `gcloud auth application-default login` to refresh your local Application Default Credentials (ADC).

---

## 🏗️ Repository Architecture

Agentic Product Foundry is organized as a multi-service monorepo:

| Directory | Component | Stack | Description |
| :--- | :--- | :--- | :--- |
| **`ai/`** | AI Engine & Agent Runner | Go, Gemini ADK | Executes AI agent workflows, manages prompt interactions, and tools. |
| **`api/`** | Backend REST API | Go, Firestore SDK | Handles project management, workspace operations, and org auth. |
| **`ui/`** | Visual Frontend Web App | Next.js 16, React 19, Tailwind, Monaco | Interactive visual canvas, real-time agent chat, and workspace management. |
| **`third_party/bmad/`** | BMAD Skills Pack | Go, Markdown, YAML | Definitions and packaging compiler for BMAD domain agent skills. |
| **`infra/`** | Infrastructure as Code | Terraform, Docker | Local Docker Compose environments and cloud deployment targets. |
| **`perf/`** | Performance & Load Testing | k6, Playwright | Baseline smoke tests, stress testing, and concurrency benchmarks. |
| **`scripts/`** | Developer Automation | Bash, Mise | Orchestration scripts for environment configuration, startup, and DB seeding. |
| **`docs/`** | Documentation & Specs | Markdown | System architecture, PRDs, database schemas, and provisioning guides. |

---

## 🔀 Your First Code Contribution (PR Workflow)

We welcome contributions from everyone! Follow this structured workflow to submit changes:

### 1. Fork & Clone
Fork the repository on GitHub and clone your fork locally:
```bash
git clone https://github.com/<your-username>/agentic-foundry.git
cd agentic-foundry
mise trust
mise run env-setup
```

### 2. Create a Feature Branch
Create a descriptive branch based off `main`:
```bash
git checkout -b feat/my-new-feature
# or
git checkout -b fix/issue-description
```

### 3. Develop & Test
Make your changes following the [Styleguides & Conventions](#-styleguides--conventions). Verify that all local tests and linter checks pass:
```bash
# Run backend tests
cd api && go test -v ./...
cd ../ai && go test -v ./...

# Run frontend tests & linting
cd ../ui && npm run test:run && npm run lint

# Validate Mise orchestration
cd .. && ./scripts/test_mise_orchestration.sh
```

### 4. Commit Your Changes
Format your commit messages following [Conventional Commits](#commit-messages-conventional-commits):
```bash
git commit -m "feat(ui): add visual zoom controls to canvas pane"
```

### 5. Push & Open a Pull Request
Push your branch to your GitHub fork and open a Pull Request against `main`:
```bash
git push -u origin feat/my-new-feature
```

- Fill out the [Pull Request Template](.github/PR_TEMPLATE/pr_template.md) completely.
- Reference any relevant issues (e.g., `Fixes #42` or `Closes #105`).
- Attach screenshots or recordings for any UI changes.

### 6. Review & CI Verification
- Automated GitHub Actions CI checks will run against your pull request.
- A project maintainer will review your code, request any necessary adjustments, and approve the PR for merging.

---

## 🧪 Testing & Quality Assurance

We enforce automated quality gates across all services. Always run relevant test suites before opening a PR.

### 1. Backend Go Tests (`ai/` & `api/`)
Run unit and integration tests across Go services:
```bash
# Run API backend tests
cd api && go test -v ./...

# Run AI service tests
cd ai && go test -v ./...
```

### 2. Frontend Tests (`ui/`)
Run component unit tests, linter, and type checks:
```bash
# Run Vitest unit tests
cd ui && npm run test:run

# Run linter
cd ui && npm run lint

# Run strict TypeScript type check
cd ui && npx tsc --noEmit
```

### 3. End-to-End Browser Tests (Playwright)
Run end-to-end UI flows using Playwright:
```bash
# Interactive UI mode
mise run test-playwright

# Headless CLI execution
cd ui && npx playwright test
```

### 4. Performance & Smoke Tests (k6)
Run performance benchmarks and load testing:
```bash
# Baseline k6 smoke test
mise run perf:smoke

# Load test scenario
mise run perf:load

# Real-time latency benchmark
mise run perf:latency
```

### 5. Mise Orchestration Verification
Run the automated environment and task orchestration verification suite:
```bash
./scripts/test_mise_orchestration.sh
```

---

## 🗄️ Database Emulators & Cloud SQL Proxy

### Local Firestore Emulator & Seeding
The Firestore database stores workspaces, session histories, and metadata. Local development runs against the Firestore Emulator by default to avoid cloud charges.

#### Running the Emulator & Seeder Standalone
The emulators start automatically with `mise run local` or `mise run dev-db`. To start only the Firebase emulators:
```bash
mise run db:emulators
```
This starts:
- Firestore Emulator on port `8084`
- Auth Emulator on port `8085`
- Emulator Suite UI at `http://localhost:4000/firestore` and `http://localhost:4000/auth`

---

### Local PostgreSQL Session Database
A local PostgreSQL instance runs inside Docker container `apf-local-session-db` to back the agent session storage.

- **Container Management**:
  ```bash
  mise run db:session-start   # Start container
  mise run db:session-stop    # Stop container
  mise run db:session-status  # Check status
  mise run db:session-clean   # Remove container and volumes
  ```
- **Connection**: `localhost:5432` | DB: `apf-local-session-db` | User: `apf-local-adk-user` | Password: `apf-local-adk-password`
- **pgAdmin UI**: `http://localhost:8083` (`admin@example.com` / `admin`)

---

### Cloud SQL Proxy (Optional Staging Access)
If you need to connect to an external or staging Cloud SQL PostgreSQL database:

```bash
mise run db:proxy
```
The proxy listens on `127.0.0.1:5432` and connects to the instance defined in your `CLOUDSQL_CONNECTION_NAME` environment variable.

---

## 📜 Styleguides & Conventions

### Commit Messages (Conventional Commits)
We follow the [Conventional Commits](https://www.conventionalcommits.org/) specification:

```text
<type>(<optional scope>): <description>

[optional body]

[optional footer(s)]
```

#### Allowed Types:
- `feat`: A new feature or capability.
- `fix`: A bug fix.
- `docs`: Documentation updates (e.g., `README.md`, `CONTRIBUTING.md`, architectural specs).
- `style`: Formatting, missing semicolons, or whitespace changes (no code behavior change).
- `refactor`: Code refactoring without fixing bugs or adding features.
- `perf`: Performance optimizations.
- `test`: Adding or correcting tests.
- `build`: Changes that affect the build system or external dependencies.
- `ci`: Changes to CI/CD configuration files and deployment scripts.
- `chore`: Maintenance tasks, toolchain updates, or repo hygiene.

#### Examples:
- `feat(ui): add visual minimap to interactive canvas`
- `fix(api): handle missing project ID in session initialization`
- `docs: update onboarding steps in CONTRIBUTING.md`

---

### Go Coding Standards
- **Formatting**: Always format code using `gofmt` (`go fmt ./...`).
- **Error Handling**: Return errors explicitly rather than using panics. Wrap errors with contextual messages (e.g., `fmt.Errorf("failed to load session %s: %w", id, err)`).
- **Context Propagation**: Pass `context.Context` as the first argument in functions performing I/O or network requests.
- **Testing**: Place unit tests in `_test.go` files adjacent to source files and utilize table-driven tests.

---

### TypeScript & React Standards
- **Framework**: Next.js 16 App Router with React 19.
- **Type Safety**: Write strict TypeScript. Avoid `any` — use explicit interfaces or generics.
- **UI Components**: Build accessible components using Radix UI primitives and Tailwind CSS utility classes.
- **State Management**: Use React hooks for local state and Zustand for cross-component shared state.

---

### Documentation Guidelines
- Maintain all links as relative repository paths (e.g. `[README](README.md)` or `[Architecture](docs/spec/planning-artifacts/architecture.md)`).
- Use standard GitHub alert callouts (`[!NOTE]`, `[!TIP]`, `[!IMPORTANT]`, `[!WARNING]`) to highlight important context.
- Keep technical architecture specs in `docs/` in sync whenever modifying system invariants or data models.

---

## 🐞 Reporting Bugs & Suggesting Enhancements

### Reporting Bugs
Before filing a bug report, search existing [GitHub Issues](https://github.com/GoogleCloudPlatform/apf/issues) to verify if the issue has already been reported.

To submit a bug report:
1. Open a new issue using our [Bug Report Template](https://github.com/GoogleCloudPlatform/apf/issues/new?template=bug_report.md).
2. Provide a clear description, reproduction steps, expected vs. actual behavior, and relevant logs.
3. Specify your environment (OS, browser, Docker/Node/Go versions).

### Suggesting Enhancements
Feature suggestions and enhancements are tracked as [GitHub Issues](https://github.com/GoogleCloudPlatform/apf/issues).
1. Open a new issue using our [Feature Request Template](https://github.com/GoogleCloudPlatform/apf/issues/new?template=feature_request.md).
2. Describe the feature motivation, use case, and proposed user experience.
3. Provide diagrams or mockups if applicable.

---

## 🔒 Security Vulnerability Reporting

We take the security of Agentic Product Foundry seriously.

> [!CAUTION]
> **Do NOT report security vulnerabilities through public GitHub issues or discussions.**

To report a security vulnerability privately:
1. Navigate to the repository's **Security** tab and click **[Report a vulnerability](https://github.com/GoogleCloudPlatform/apf/security/advisories/new)** (GitHub Private Vulnerability Reporting).
2. Provide a detailed summary, severity assessment, proof-of-concept steps, and affected versions.
3. Maintainers will review the report, collaborate with you on a patch in a private repository fork, and coordinate a responsible disclosure timeline.

---

## 🤝 Code of Conduct & Community

This project and everyone participating in it is governed by the [Agentic Product Foundry Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.

### Community & Maintainers
- **Becoming a Maintainer**: Community members who consistently contribute high-quality PRs, helpful reviews, and active issue triage may be invited to join the project maintainer team.


