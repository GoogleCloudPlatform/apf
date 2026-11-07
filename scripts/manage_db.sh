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

# scripts/manage_db.sh

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
ROOT_DIR="$DIR/.."
COMPOSE_FILE="$ROOT_DIR/infra/local/docker-compose.yml"

start_db() {
    echo "Starting local stack via Docker Compose..."
    docker compose -f "$COMPOSE_FILE" up --build -d
}

stop_db() {
    echo "Stopping local stack via Docker Compose..."
    docker compose -f "$COMPOSE_FILE" stop
}

clean_db() {
    echo "Removing local stack via Docker Compose..."
    docker compose -f "$COMPOSE_FILE" down -v
}

status_db() {
    echo "Checking status via Docker Compose..."
    docker compose -f "$COMPOSE_FILE" ps
}

case "$1" in
    start)
        start_db
        ;;
    stop)
        stop_db
        ;;
    clean)
        clean_db
        ;;
    status)
        status_db
        ;;
    *)
        echo "Usage: $0 {start|stop|clean|status}"
        exit 1
        ;;
esac

