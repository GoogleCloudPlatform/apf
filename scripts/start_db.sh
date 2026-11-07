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
echo "🗄️  Starting APF Local Databases & Data Seeding (Part 1)"
echo "================================================================="

ROOT_DIR=$(git rev-parse --show-toplevel 2>/dev/null || pwd)
mkdir -p "$ROOT_DIR/logs"

# 1. Clean up lingering background daemons on target DB/emulator ports
echo "Cleaning up lingering background daemons on target DB/emulator ports (8084, 8085, 4000, 4400, 4500)..."
kill -9 $(lsof -t -sTCP:LISTEN -i:8084 -i:8085 -i:4000 -i:4400 -i:4500) 2>/dev/null || true

# 2. Start PostgreSQL container
echo "Starting local PostgreSQL session store container..."
"$ROOT_DIR/scripts/manage_db.sh" start

# 3. Start Firebase Emulators (Firestore: 8084, Auth: 8085) in detached/background mode
echo "Starting Firebase Emulators (Firestore: 8084, Auth: 8085)... -> logging to logs/firestore.log"
cd "$ROOT_DIR"
firebase emulators:start < /dev/null > "$ROOT_DIR/logs/firestore.log" 2>&1 &
FB_PID=$!
disown $FB_PID 2>/dev/null || true

echo "Waiting for Firebase emulators to initialize..."
timeout=30

# Poll Firestore Emulator (8084)
count=0
until curl -s http://127.0.0.1:8084/ > /dev/null || [ $count -eq $timeout ]; do
    sleep 1
    count=$((count + 1))
done
if [ $count -eq $timeout ]; then
    echo "❌ Timeout waiting for Firestore emulator on port 8084"
    exit 1
fi

# Poll Auth Emulator (8085)
count=0
until curl -s http://127.0.0.1:8085/ > /dev/null || [ $count -eq $timeout ]; do
    sleep 1
    count=$((count + 1))
done
if [ $count -eq $timeout ]; then
    echo "❌ Timeout waiting for Firebase Auth emulator on port 8085"
    exit 1
fi

echo "✅ Firebase emulators are ready!"

# 4. Seed Firestore
echo "Seeding Firestore database... -> logging to logs/seed.log"
cd "$ROOT_DIR/api"
if [ -f .env ]; then
    set -a
    source .env
    set +a
elif [ -f "$ROOT_DIR/config.env" ]; then
    set -a
    source "$ROOT_DIR/config.env"
    set +a
fi
export FIRESTORE_EMULATOR_HOST="${FIRESTORE_EMULATOR_HOST:-localhost:8084}"
export FIREBASE_AUTH_EMULATOR_HOST="${FIREBASE_AUTH_EMULATOR_HOST:-localhost:8085}"
go run ./seeding -env local > "$ROOT_DIR/logs/seed.log" 2>&1

# 5. Seed PostgreSQL session database
echo "Seeding PostgreSQL session store database... -> logging to logs/pg-seed.log"
cd "$ROOT_DIR/ai"
if [ -f .env ]; then
    set -a
    source .env
    set +a
elif [ -f "$ROOT_DIR/config.env" ]; then
    set -a
    source "$ROOT_DIR/config.env"
    set +a
fi
go run seed/main.go > "$ROOT_DIR/logs/pg-seed.log" 2>&1

echo "================================================================="
echo "✅ Part 1 Complete: Local databases (Postgres & Firestore emulators)"
echo "   have been started, seeded, and are running detached in background!"
echo "   - Postgres DB:      localhost:5432"
echo "   - pgAdmin UI:       http://localhost:8083 (admin@example.com / admin)"
echo "   - Firestore API:    http://127.0.0.1:8084"
echo "   - Auth API:         http://127.0.0.1:8085"
echo "   - Firestore UI:     http://127.0.0.1:4000/firestore"
echo "================================================================="
