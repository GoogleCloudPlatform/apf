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

resource "google_service_account" "ops" {
  account_id   = "${var.prefix}-ops"
  display_name = "${var.prefix} Ops Service Account"
  project      = var.project_id
}

resource "google_service_account" "compute" {
  account_id   = "${var.prefix}-compute"
  display_name = "${var.prefix} Compute Service Account"
  project      = var.project_id
}

# Permissions for apf-ops
resource "google_project_iam_member" "ops_artifactregistry_admin" {
  project = var.project_id
  role    = "roles/artifactregistry.admin"
  member  = "serviceAccount:${google_service_account.ops.email}"
}

resource "google_project_iam_member" "ops_storage_admin" {
  project = var.project_id
  role    = "roles/storage.admin"
  member  = "serviceAccount:${google_service_account.ops.email}"
}

resource "google_project_iam_member" "ops_run_admin" {
  project = var.project_id
  role    = "roles/run.admin"
  member  = "serviceAccount:${google_service_account.ops.email}"
}

resource "google_project_iam_member" "ops_vpcaccess_admin" {
  project = var.project_id
  role    = "roles/vpcaccess.admin"
  member  = "serviceAccount:${google_service_account.ops.email}"
}

resource "google_project_iam_member" "ops_secretmanager_admin" {
  project = var.project_id
  role    = "roles/secretmanager.admin"
  member  = "serviceAccount:${google_service_account.ops.email}"
}

resource "google_project_iam_member" "ops_logging_logwriter" {
  project = var.project_id
  role    = "roles/logging.logWriter"
  member  = "serviceAccount:${google_service_account.ops.email}"
}

resource "google_service_account_iam_member" "ops_act_as_compute" {
  service_account_id = google_service_account.compute.name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${google_service_account.ops.email}"
}

# Permissions for apf-compute
resource "google_project_iam_member" "compute_datastore_user" {
  project = var.project_id
  role    = "roles/datastore.user"
  member  = "serviceAccount:${google_service_account.compute.email}"
}

resource "google_project_iam_member" "compute_storage_object_admin" {
  project = var.project_id
  role    = "roles/storage.objectAdmin"
  member  = "serviceAccount:${google_service_account.compute.email}"
}

resource "google_project_iam_member" "compute_aiplatform_user" {
  project = var.project_id
  role    = "roles/aiplatform.user"
  member  = "serviceAccount:${google_service_account.compute.email}"
}

resource "google_project_iam_member" "compute_secretmanager_accessor" {
  project = var.project_id
  role    = "roles/secretmanager.secretAccessor"
  member  = "serviceAccount:${google_service_account.compute.email}"
}

resource "google_project_iam_member" "compute_cloudtrace_agent" {
  project = var.project_id
  role    = "roles/cloudtrace.agent"
  member  = "serviceAccount:${google_service_account.compute.email}"
}

resource "google_project_iam_member" "compute_cloudsql_client" {
  project = var.project_id
  role    = "roles/cloudsql.client"
  member  = "serviceAccount:${google_service_account.compute.email}"
}

resource "google_project_iam_member" "compute_cloudsql_user" {
  project = var.project_id
  role    = "roles/cloudsql.instanceUser"
  member  = "serviceAccount:${google_service_account.compute.email}"
}

resource "google_project_iam_member" "compute_identitytoolkit_admin" {
  project = var.project_id
  role    = "roles/identitytoolkit.admin"
  member  = "serviceAccount:${google_service_account.compute.email}"
}

resource "google_project_iam_member" "compute_run_developer" {
  project = var.project_id
  role    = "roles/run.developer"
  member  = "serviceAccount:${google_service_account.compute.email}"
}

resource "google_service_account_iam_member" "compute_act_as_compute" {
  service_account_id = google_service_account.compute.name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${google_service_account.compute.email}"
}

# Service Account for Performance Testing & k6 Load Harness
resource "google_service_account" "perf_testing" {
  account_id   = "${var.prefix}-perf-testing"
  display_name = "${var.prefix} Performance Testing Service Account"
  project      = var.project_id
}

# Grant IAP Secured Web App User permission to access IAP-protected endpoints
resource "google_project_iam_member" "perf_testing_iap_user" {
  project = var.project_id
  role    = "roles/iap.httpsResourceAccessor"
  member  = "serviceAccount:${google_service_account.perf_testing.email}"
}

