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
echo "🚀 Starting APF Application Processes (Part 2: Ai, Api, ADK, Ui)"
echo "================================================================="

ROOT_DIR=$(git rev-parse --show-toplevel 2>/dev/null || pwd)
mkdir -p "$ROOT_DIR/logs"

HOT_RELOAD=${HOT_RELOAD:-true}
DEBUG=${DEBUG:-false}

kill_ui() {
    echo "Killing UI service (port 8082) and its parents..."
    port=8082
    pid=$(lsof -t -sTCP:LISTEN -i:$port 2>/dev/null || ss -lptn "sport = :$port" 2>/dev/null | grep -oP 'pid=\K\d+' | head -n 1)
    if [ -n "$pid" ]; then
      ppid=$(ps -o ppid= -p $pid | tr -d ' ')
      if [ -n "$ppid" ] && [ "$ppid" -ne 1 ]; then
        gppid=$(ps -o ppid= -p $ppid | tr -d ' ')
        if [ -n "$gppid" ] && [ "$gppid" -ne 1 ]; then
          kill -9 $gppid 2>/dev/null || true
        fi
        kill -9 $ppid 2>/dev/null || true
      fi
      kill -9 $pid 2>/dev/null || true
    fi
    pkill -9 -f '[n]ext dev' 2>/dev/null || true
}

stop_apps() {
    set +e
    echo "Stopping application processes (Ai, Api, ADK, Ui)..."
    [ -n "$AI_PID" ] && kill -9 $AI_PID 2>/dev/null || true
    [ -n "$API_PID" ] && kill -9 $API_PID 2>/dev/null || true
    [ -n "$ADK_PID" ] && kill -9 $ADK_PID 2>/dev/null || true
    [ -n "$RUNNER_PID" ] && kill -9 $RUNNER_PID 2>/dev/null || true
    [ -n "$UI_PID" ] && kill -9 $UI_PID 2>/dev/null || true
    kill -9 $(lsof -t -sTCP:LISTEN -i:8080 -i:8081 -i:8086 -i:8087) 2>/dev/null || true
    kill_ui
}

cleanup() {
    set +e
    stty echo 2>/dev/null || true
    stop_apps
}
trap cleanup EXIT
trap 'exit 0' SIGINT SIGTERM

start_apps() {
    echo "Cleaning up lingering application background daemons on app ports (8080, 8081, 8086, 8087, 8082)..."
    kill -9 $(lsof -t -sTCP:LISTEN -i:8080 -i:8081 -i:8086 -i:8087) 2>/dev/null || true
    kill_ui

    # 1. Start Ai (port 8080)
    if [ "$HOT_RELOAD" = "true" ] && command -v air >/dev/null 2>&1; then
        echo "Starting Ai service in HOT RELOAD mode (port 8080)... -> logging to logs/ai.log"
        cd "$ROOT_DIR/ai"
        if [ -f .env ]; then
            source .env
        fi
        if [ "$DEBUG" = "true" ]; then
            echo "Running with Delve debugger..."
            air -c .air.toml < /dev/null > "$ROOT_DIR/logs/ai.log" 2>&1 &
        else
            air -c .air.toml -build.full_bin "./tmp/ai web --port 8080 -read-timeout 5m -write-timeout 5m api webui" < /dev/null > "$ROOT_DIR/logs/ai.log" 2>&1 &
        fi
        AI_PID=$!
    else
        if [ "$HOT_RELOAD" = "true" ]; then
            echo "⚠️ air is not installed. Falling back to standard run."
        fi
        echo "Starting Ai service (port 8080)... -> logging to logs/ai.log"
        cd "$ROOT_DIR/ai"
        if [ -f .env ]; then
            source .env
        fi
        go run . web --port 8080 -read-timeout 5m -write-timeout 5m api webui < /dev/null > "$ROOT_DIR/logs/ai.log" 2>&1 &
        AI_PID=$!
    fi

    # 2. Start Api (port 8081)
    if [ "$HOT_RELOAD" = "true" ] && command -v air >/dev/null 2>&1; then
        echo "Starting Api service in HOT RELOAD mode (port 8081)... -> logging to logs/api.log"
        cd "$ROOT_DIR/api"
        if [ "$DEBUG" = "true" ]; then
            echo "Running with Delve debugger..."
            air -c .air.toml < /dev/null > "$ROOT_DIR/logs/api.log" 2>&1 &
        else
            air -c .air.toml -build.full_bin "./tmp/api" < /dev/null > "$ROOT_DIR/logs/api.log" 2>&1 &
        fi
        API_PID=$!
    else
        if [ "$HOT_RELOAD" = "true" ]; then
            echo "⚠️ air is not installed. Falling back to standard run."
        fi
        echo "Starting Api service (port 8081)... -> logging to logs/api.log"
        cd "$ROOT_DIR/api"
        go run . < /dev/null > "$ROOT_DIR/logs/api.log" 2>&1 &
        API_PID=$!
    fi

    # 3. Start ADK Web UI & Runner (if runners/adk exists)
    if [ -d "$ROOT_DIR/runners/adk" ]; then
        echo "Starting ADK Web UI (port 8086)... -> logging to logs/adk-ui.log"
        (cd "$ROOT_DIR/runners/adk" && WORKFLOW_LOAD_METHOD=FILE WORKFLOW_FILE_PATH="$ROOT_DIR/runners/adk/test_workflow.json" uv run adk web --port 8086 --reload --reload_agents . < /dev/null > "$ROOT_DIR/logs/adk-ui.log" 2>&1) &
        ADK_PID=$!

        echo "Starting ADK Programmatic Runner API (port 8087)... -> logging to logs/adk.log"
        (cd "$ROOT_DIR/runners/adk" && WORKFLOW_LOAD_METHOD=FILE WORKFLOW_FILE_PATH="$ROOT_DIR/runners/adk/test_workflow.json" PYTHONPATH="$ROOT_DIR" uv run python main.py --port 8087 < /dev/null > "$ROOT_DIR/logs/adk.log" 2>&1) &
        RUNNER_PID=$!
    fi

    echo "Waiting 3 seconds for backend services to initialize..."
    sleep 3

    # 4. Start Ui (port 8082)
    echo "Starting Next.js UI service (port 8082)... -> logging to logs/ui.log"
    cd "$ROOT_DIR/ui"
    if [ ! -d "node_modules" ]; then
        echo "⚠️ node_modules not found in ui/. Running npm install..."
        npm install --legacy-peer-deps < /dev/null
    fi
    npm run dev < /dev/null > "$ROOT_DIR/logs/ui.log" 2>&1 &
    UI_PID=$!

    echo "================================================================="
    echo "✅ Application processes (Ai, Api, Ui) are running!"
    echo ""
    echo "To view live service logs in separate terminal tabs, run:"
    echo "- Ai Logs:          tail -f logs/ai.log"
    echo "- Api Logs:         tail -f logs/api.log"
    echo "- Ui Logs:          tail -f logs/ui.log"
    echo ""
    echo "- Ai Endpoint:      http://localhost:8080"
    echo "- Api Endpoint:     http://localhost:8081"
    echo "- Ui Endpoint:      http://localhost:8082"
    echo ""
    echo "Press Ctrl+R here at any time to cleanly restart application services."
    echo "Press Ctrl+C here at any time to cleanly stop application services."
    echo "================================================================="
}

while true; do
    start_apps
    
    while true; do
        if ! read -r -s -n 1 key; then
            echo "Stdin closed. Waiting for background application processes..."
            wait $AI_PID $API_PID $UI_PID $ADK_PID $RUNNER_PID
            exit 0
        fi
        
        if [[ "$key" == $'\x12' ]]; then
            echo "🔄 Ctrl+R detected! Restarting application processes (Ai, Api, ADK, Ui)..."
            stop_apps
            break
        fi
    done
done
