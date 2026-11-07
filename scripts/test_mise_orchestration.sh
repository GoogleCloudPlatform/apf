#!/bin/bash
# Copyright 2026 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -e

echo "================================================================="
echo "🧪 Running Automated Verification Suite: Mise Orchestration"
echo "================================================================="

ROOT_DIR=$(git rev-parse --show-toplevel 2>/dev/null || pwd)
FAILURES=0

# Helper function for reporting
assert_success() {
    if [ $? -eq 0 ]; then
        echo "✅ PASS: $1"
    else
        echo "❌ FAIL: $1"
        FAILURES=$((FAILURES + 1))
    fi
}

# Helper function for checking file existence
assert_file_exists() {
    if [ -f "$1" ]; then
        echo "✅ PASS: File exists -> $2"
    else
        echo "❌ FAIL: Missing file -> $2 ($1)"
        FAILURES=$((FAILURES + 1))
    fi
}

echo ""
echo "--- Assertion 1: Structural Integrity of Configuration Files ---"
assert_file_exists "$ROOT_DIR/mise.toml" "Root mise.toml"
assert_file_exists "$ROOT_DIR/config.env.template" "Root config.env.template"
assert_file_exists "$ROOT_DIR/api/mise.toml" "api/mise.toml"
assert_file_exists "$ROOT_DIR/ai/mise.toml" "ai/mise.toml"
assert_file_exists "$ROOT_DIR/ui/package.json" "ui/package.json"
assert_file_exists "$ROOT_DIR/infra/mise.toml" "infra/mise.toml"
assert_file_exists "$ROOT_DIR/scripts/start_db.sh" "scripts/start_db.sh"
assert_file_exists "$ROOT_DIR/scripts/start_apps.sh" "scripts/start_apps.sh"
assert_file_exists "$ROOT_DIR/scripts/start_all.sh" "scripts/start_all.sh"

echo ""
echo "--- Assertion 2: Validating tasks.json Proxy Mappings ---"
if grep -q "mise run dev-api" "$ROOT_DIR/.vscode/tasks.json" && \
   grep -q "mise run dev-ai" "$ROOT_DIR/.vscode/tasks.json" && \
   grep -q "mise run kill-all" "$ROOT_DIR/.vscode/tasks.json" && \
   grep -q "mise run db:proxy" "$ROOT_DIR/.vscode/tasks.json" && \
   grep -q "mise run dev-db" "$ROOT_DIR/.vscode/tasks.json" && \
   grep -q "mise run dev-apps" "$ROOT_DIR/.vscode/tasks.json" && \
   grep -q "mise run db:emulators" "$ROOT_DIR/.vscode/tasks.json"; then
    echo "✅ PASS: .vscode/tasks.json correctly proxies to mise run tasks"
else
    echo "❌ FAIL: .vscode/tasks.json is missing required mise proxy mappings"
    FAILURES=$((FAILURES + 1))
fi

echo ""
echo "--- Assertion 3: Validating Root mise.toml Task Definitions ---"
if grep -q "tasks.env-setup" "$ROOT_DIR/mise.toml" && \
   grep -q "bmad:package ./ai/workspaces/config/bmad" "$ROOT_DIR/mise.toml" && \
   grep -q "tasks.dev-all" "$ROOT_DIR/mise.toml" && \
   grep -q "tasks.dev-db" "$ROOT_DIR/mise.toml" && \
   grep -q "tasks.dev-apps" "$ROOT_DIR/mise.toml" && \
   grep -q "tasks.kill-all" "$ROOT_DIR/mise.toml" && \
   grep -q "tasks.kill-ai" "$ROOT_DIR/mise.toml"; then
    echo "✅ PASS: Root mise.toml contains all required global task definitions"
else
    echo "❌ FAIL: Root mise.toml is missing required task definitions"
    FAILURES=$((FAILURES + 1))
fi

echo ""
echo "--- Assertion 4: Testing Standalone Environment Script Execution ---"
if [ -f "$ROOT_DIR/scripts/setup_env.sh" ]; then
    echo "Executing scripts/setup_env.sh..."
    # Provide mock/non-interactive env to avoid blocking prompt during automated tests
    GEMINI_API_KEY="mock-key-for-automated-testing" bash "$ROOT_DIR/scripts/setup_env.sh" >/dev/null 2>&1
    assert_success "scripts/setup_env.sh executed successfully"
    
    assert_file_exists "$ROOT_DIR/ai/.env" "Generated ai/.env"
    assert_file_exists "$ROOT_DIR/api/.env" "Generated api/.env"
    assert_file_exists "$ROOT_DIR/ui/.env.local" "Generated ui/.env.local"
else
    echo "❌ FAIL: scripts/setup_env.sh not found"
    FAILURES=$((FAILURES + 1))
fi

echo ""
echo "--- Assertion 5: Testing Process Termination & Port Cleanup ---"
echo "Executing mise run kill-all..."
mise run kill-all >/dev/null 2>&1 || true
# Check if ports 8080, 8081, 8082, 8084 are free
if lsof -t -i:8080 -i:8081 -i:8082 -i:8084 >/dev/null 2>&1; then
    echo "❌ FAIL: Lingering processes detected on target ports after kill-all"
    FAILURES=$((FAILURES + 1))
else
    echo "✅ PASS: Ports 8080, 8081, 8082, and 8084 are successfully verified free"
fi

echo ""
echo "================================================================="
if [ $FAILURES -eq 0 ]; then
    echo "🎉 ALL AUTOMATED VERIFICATION CHECKS PASSED SUCCESSFULLY!"
    echo "================================================================="
    exit 0
else
    echo "⚠️ VERIFICATION SUITE FAILED WITH $FAILURES ERRORS."
    echo "================================================================="
    exit 1
fi
