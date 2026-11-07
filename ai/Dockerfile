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

# Build stage
FROM golang:1.26.1-alpine AS builder

WORKDIR /app

# Copy workspace definition and dependency files for caching
COPY ai/go.mod ai/go.sum ./ai/

# Download dependencies
WORKDIR /app/ai
ENV GOWORK=off
RUN go mod download

# Copy the entire repository source code
WORKDIR /app
COPY . ./

# Build the binary
WORKDIR /app/ai
RUN CGO_ENABLED=0 GOOS=linux go build -o apf

# Run bmad packaging script to generate prepackaged skills and configuration
RUN go run ../third_party/bmad/package.go --install-dir /config/bmad --catalog-file ../api/seeding/templates/showcase/bmad_catalog.json

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

COPY --from=builder /app/ai/apf /apf
COPY --from=builder /app/ai/sub_agents/scaffolder/service-catalog /sub_agents/scaffolder/service-catalog
COPY --from=builder /config/bmad /config/bmad

ARG COMMIT_SHA=""
ENV COMMIT_SHA=${COMMIT_SHA}

EXPOSE 8080

# Cloud Run sets PORT env var which ADK launcher reads automatically
# Supports REST API, A2A protocol, and Web UI
# API endpoints: /api/run_sse, /api/apps/...
# A2A endpoint: /a2a/invoke (JSON-RPC)
# Agent card: /.well-known/agent-card.json
# Web UI: /ui/ (requires IAP for authenticated access)
ENTRYPOINT ["/apf", "web", "-read-timeout", "5m", "-write-timeout", "5m", "api", "a2a", "webui"]