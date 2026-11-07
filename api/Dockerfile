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

# Copy dependency files for caching
COPY api/go.mod api/go.sum ./api/

# Download dependencies
WORKDIR /app/api
ENV GOWORK=off
RUN go mod download

# Explicitly copy ONLY what is needed: API source code and BMAD library/skills
WORKDIR /app
COPY api/ ./api/
COPY third_party/bmad/ ./third_party/bmad/

# Build the API binary and Jobs CLI
WORKDIR /app/api
RUN CGO_ENABLED=0 GOOS=linux go build -o apf-api .
RUN CGO_ENABLED=0 GOOS=linux go build -o apf-jobs ./cmd/jobs

# Run bmad packaging script to generate prepackaged skills and configuration
RUN go run ../third_party/bmad/package.go --install-dir /config/bmad --catalog-file ./seeding/templates/showcase/bmad_catalog.json

# Runtime stage
FROM alpine:3.20

RUN apk add --no-cache git ca-certificates openssh-client tzdata && \
    git config --system protocol.version 2 && \
    git config --system fetch.negotiationAlgorithm skipping && \
    git config --system fetch.writeCommitGraph true && \
    git config --system core.commitGraph true && \
    git config --system gc.writeCommitGraph true && \
    git config --system remote.origin.tagOpt --no-tags

ENV GIT_PROTOCOL_FROM_USER=2

WORKDIR /
COPY --from=builder /app/api/apf-api /apf-api
COPY --from=builder /app/api/apf-jobs /apf-jobs
COPY --from=builder /app/api/version.json /version.json
COPY --from=builder /config/bmad /config/bmad

RUN chmod +x /apf-api /apf-jobs

EXPOSE 8081

ENTRYPOINT ["/apf-api"]

