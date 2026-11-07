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

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
ROOT_DIR="$DIR/.."

echo "================================================================="
echo "🚀 Starting APF Local Stack (2-Part Orchestration)"
echo "   Part 1: Database containers & data seeding (detached/background)"
echo "   Part 2: Application processes (Ai, Api, ADK, Ui)"
echo "================================================================="

# Part 1: Start Database Containers & Seed Data (runs detached and goes to background)
"$ROOT_DIR/scripts/start_db.sh"

# Part 2: Start Application Processes (Ai, Api, ADK, Ui)
exec "$ROOT_DIR/scripts/start_apps.sh"
