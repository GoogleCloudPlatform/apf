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

# scripts/build_containers.sh

set -e

# Determine repository root directory
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
ROOT_DIR="$DIR/.."

echo "Building ADK 2.0 Runner container (apf-adk2-runner)..."

BUILD_ARGS=""
if [ -n "$UV_EXTRA_INDEX_URL" ]; then
  echo "Using configured UV_EXTRA_INDEX_URL from environment..."
  export UV_EXTRA_INDEX_URL
  BUILD_ARGS="--secret id=UV_EXTRA_INDEX_URL"
fi

# Enable BuildKit for secrets support
export DOCKER_BUILDKIT=1

docker build $BUILD_ARGS -t apf-adk2-runner "$ROOT_DIR/runners/adk"

echo "Containers built successfully."
