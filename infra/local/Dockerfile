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

# Use the official lightweight PostgreSQL 15 image
FROM postgres:15-alpine

# Set default environment variables for local development
# These are fallback/default configurations and can be overridden at runtime
# (e.g., using docker run -e POSTGRES_PASSWORD=...)
ENV POSTGRES_DB=apf-local-session-db
ENV POSTGRES_USER=apf-local-adk-user
ENV POSTGRES_PASSWORD=apf-local-adk-password

# Copy initialization script to create multiple databases
COPY --chown=postgres:postgres init-user-db.sh /docker-entrypoint-initdb.d/
RUN chmod 755 /docker-entrypoint-initdb.d/init-user-db.sh

# Expose default PostgreSQL port
EXPOSE 5432

# Health check to ensure the database is ready to accept connections before other services start
HEALTHCHECK --interval=5s --timeout=5s --retries=5 \
  CMD pg_isready -U "${POSTGRES_USER}" -d "${POSTGRES_DB}"

# Optimize for local development & testing:
# - fsync=off: Disables flushing data to physical disk on every transaction (massive write performance speedup for tests/dev, data safe in container lifecycle but not crash resilient)
# - synchronous_commit=off: Transactions return success before logs are flushed to disk (improves latency)
# - full_page_writes=off: Disables page writes during checkpoints (further optimization for temporary/dev databases)
CMD ["postgres", "-c", "fsync=off", "-c", "synchronous_commit=off", "-c", "full_page_writes=off"]
