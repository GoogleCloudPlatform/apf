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

resource "google_identity_platform_config" "default" {
  project = var.project_id

  # Enable Email/Password provider as a basic default
  sign_in {
    allow_duplicate_emails = false

    email {
      enabled           = true
      password_required = true
    }
  }

  # Enable Multi-Tenancy
  multi_tenant {
    allow_tenants = true
  }

  # Ensure the APIs are enabled first
  depends_on = [
    google_project_service.apis
  ]
}

resource "google_identity_platform_default_supported_idp_config" "google" {
  count = var.iap_client_id != "" ? 1 : 0

  project       = var.project_id
  idp_id        = "google.com"
  client_id     = var.iap_client_id
  client_secret = data.google_secret_manager_secret_version.iap_secret.secret_data
  enabled       = true

  depends_on = [
    google_identity_platform_config.default
  ]
}
