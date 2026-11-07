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

# Secrets managed by Terraform (containers only, versions managed externally)

resource "google_secret_manager_secret" "iap_client_secret" {
  count = var.create_iap_secret ? 1 : 0

  secret_id = "iap-client-secret"
  project   = var.project_id

  replication {
    auto {}
  }
}

resource "google_secret_manager_secret" "github_client_secret" {
  secret_id = "${var.prefix}-github-client-secret"
  project   = var.project_id

  replication {
    auto {}
  }
}

resource "google_secret_manager_secret" "secrets_encryption_key" {
  secret_id = "${var.prefix}-secrets-encryption-key"
  project   = var.project_id

  replication {
    auto {}
  }
}

resource "google_secret_manager_secret" "jwt_secret" {
  secret_id = "${var.prefix}-jwt-secret"
  project   = var.project_id

  replication {
    auto {}
  }
}

resource "google_secret_manager_secret" "super_admin_emails" {
  secret_id = "${var.prefix}-super-admin-emails"
  project   = var.project_id

  replication {
    auto {}
  }
}

resource "google_secret_manager_secret" "perf_test_key" {
  secret_id = "${var.prefix}-perf-test-key"
  project   = var.project_id

  replication {
    auto {}
  }
}

resource "google_secret_manager_secret" "secret_encryption_public_key" {
  secret_id = "${var.prefix}-secret-encryption-public-key"
  project   = var.project_id

  replication {
    auto {}
  }
}

resource "google_secret_manager_secret" "secret_encryption_private_key" {
  secret_id = "${var.prefix}-secret-encryption-private-key"
  project   = var.project_id

  replication {
    auto {}
  }
}

