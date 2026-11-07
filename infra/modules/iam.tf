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

resource "google_iap_web_backend_service_iam_member" "ui_iap_user" {
  for_each            = toset(var.iap_allowed_members)
  project             = var.project_id
  web_backend_service = google_compute_backend_service.ui.name
  role                = "roles/iap.httpsResourceAccessor"
  member              = each.value
}

resource "google_storage_bucket_iam_member" "workspaces_object_user" {
  bucket = "${var.prefix}-workspaces"
  role   = "roles/storage.objectUser"
  member = "serviceAccount:${google_service_account.compute.email}"
}

resource "google_storage_bucket_iam_member" "config_object_user" {
  bucket = "${var.prefix}-config"
  role   = "roles/storage.objectUser"
  member = "serviceAccount:${google_service_account.compute.email}"
}

resource "google_storage_bucket_iam_member" "artifacts_object_user" {
  bucket = "${var.prefix}-artifacts"
  role   = "roles/storage.objectUser"
  member = "serviceAccount:${google_service_account.compute.email}"
}

resource "google_storage_bucket_iam_member" "workspaces_bucket_reader" {
  bucket = "${var.prefix}-workspaces"
  role   = "roles/storage.legacyBucketReader"
  member = "serviceAccount:${google_service_account.compute.email}"
}

resource "google_storage_bucket_iam_member" "config_bucket_reader" {
  bucket = "${var.prefix}-config"
  role   = "roles/storage.legacyBucketReader"
  member = "serviceAccount:${google_service_account.compute.email}"
}

resource "google_storage_bucket_iam_member" "artifacts_bucket_reader" {
  bucket = "${var.prefix}-artifacts"
  role   = "roles/storage.legacyBucketReader"
  member = "serviceAccount:${google_service_account.compute.email}"
}

resource "google_iap_web_backend_service_iam_member" "api_iap_user" {
  for_each            = toset(var.iap_allowed_members)
  project             = var.project_id
  web_backend_service = google_compute_backend_service.api.name
  role                = "roles/iap.httpsResourceAccessor"
  member              = each.value
}




