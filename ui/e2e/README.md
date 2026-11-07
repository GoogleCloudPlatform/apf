# Test Architecture & E2E Framework Guide

This document outlines the **Test Architecture Enterprise (TEA)** framework setup for **Agentic Foundry**.

---

## 🏗️ Architecture Overview

The repository utilizes a **Fullstack Test Architecture**:
- **Frontend / E2E Testing:** Playwright + TypeScript (`ui/playwright.config.ts`, `ui/e2e/`)
- **Frontend Unit / Component Testing:** Vitest + React Testing Library (`ui/src/**/*.test.tsx`)
- **Backend Service / Integration Testing:** Go `testing` + `testify` (`api/*_test.go`)

```
agentic-foundry/
├── api/                             # Go Backend
│   ├── *_test.go                    # Go unit & integration tests
│   └── db/                          # Database unit tests
└── ui/                              # Next.js Frontend
    ├── playwright.config.ts         # Playwright E2E configuration
    └── e2e/                         # E2E Test Suite
        ├── fixtures/                # Composable fixtures (Login, Project, API)
        ├── page-objects/            # Page Object Models
        └── specs/                   # Test Spec Files (P0, P1, P2)
```

---

## 🚀 Running Tests

### 1. E2E Browser & API Integration Tests (Playwright)

From the `ui/` directory:

```bash
# Run all E2E tests in headless mode
npm run test:e2e

# Run tests with interactive UI mode / headed mode
npx playwright test --ui

# Debug a specific spec file
npx playwright test e2e/specs/login.spec.ts --debug

# View HTML test execution & trace report
npx playwright show-report
```

### 2. Frontend Unit & Component Tests (Vitest)

```bash
# Run unit test suite in watch mode
npm run test

# Single pass run
npm run test:run
```

### 3. Backend Tests (Go)

From the root directory:

```bash
# Run all Go package tests
go test -v ./...

# Run specific test with race detection
go test -v -race ./api/...
```

---

## 🧪 TEA Quality Standards & Best Practices

1. **Deterministic Assertions (No Flakiness):**
   - Never use hard waits (`page.waitForTimeout(5000)` or `sleep`).
   - Use auto-waiting Playwright locators and explicit state assertions (`expect(locator).toBeVisible()`).

2. **Resilient Selectors:**
   - Prefer role-based and test-ID selectors (`page.getByRole('button', { name: 'Onboard' })`, `page.getByTestId('project-card')`).
   - Avoid brittle CSS paths or index-based locators.

3. **Fixture Composition:**
   - Extend base Playwright `test` using `mergeTests` for modular, reusable setup (`authFixtures`, `projectFixtures`).
   - Ensure fixtures clean up seeded data automatically upon test completion.

4. **Network-First Interception:**
   - Intercept and mock dynamic routes (`page.route('/api/v1/projects', ...)`) *before* triggering UI click actions to prevent race conditions.

---

## ⚙️ CI/CD Quality Gate Thresholds

- **P0 Critical Suite:** **100% Pass Rate required** on all pull requests.
- **P1 High-Priority Suite:** **$\ge 95\%$ Pass Rate required** before merging to main.
- **Artifact Retention:** Traces and screenshots automatically retained on first retry failure.
