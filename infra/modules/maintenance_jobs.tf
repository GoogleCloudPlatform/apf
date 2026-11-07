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

# Service account dedicated to triggering scheduled Cloud Run jobs
resource "google_service_account" "job_scheduler" {
  account_id   = "${var.prefix}-job-scheduler"
  display_name = "${var.prefix} Job Scheduler Service Account"
  project      = var.project_id
}

# Grant run.invoker role to the scheduler service account on the project
resource "google_project_iam_member" "job_scheduler_run_invoker" {
  project = var.project_id
  role    = "roles/run.invoker"
  member  = "serviceAccount:${google_service_account.job_scheduler.email}"
}

# Cloud Run v2 Job for maintenance operations (storage audit, workspace cleanup)
resource "google_cloud_run_v2_job" "maintenance" {
  name                = "${var.prefix}-maintenance"
  location            = var.region
  project             = var.project_id
  deletion_protection = false

  template {
    template {
      max_retries     = 2
      timeout         = "1800s"
      service_account = google_service_account.compute.email

      vpc_access {
        connector = google_vpc_access_connector.connector.id
        egress    = "ALL_TRAFFIC"
      }

      volumes {
        name = "workspaces"
        nfs {
          server = google_filestore_instance.nfs_instance.networks[0].ip_addresses[0]
          path   = "/workspaces"
        }
      }

      containers {
        image   = "${var.region}-docker.pkg.dev/${var.project_id}/${var.prefix}-repo/apf-api:latest"
        command = ["/apf-jobs"]
        args    = ["nfs-audit"]

        volume_mounts {
          name       = "workspaces"
          mount_path = "/workspaces"
        }

        env {
          name  = "GOOGLE_CLOUD_PROJECT"
          value = var.project_id
        }

        env {
          name  = "WORKSPACE_MOUNT_PATH"
          value = "/workspaces"
        }

        env {
          name  = "FIRESTORE_DB_NAME"
          value = google_firestore_database.default.name
        }

        env {
          name = "SECRETS_ENCRYPTION_KEY"
          value_source {
            secret_key_ref {
              secret  = google_secret_manager_secret.secrets_encryption_key.secret_id
              version = "latest"
            }
          }
        }

        env {
          name = "CLOUDSQL_PASSWORD"
          value_source {
            secret_key_ref {
              secret  = google_secret_manager_secret.db_password.secret_id
              version = "latest"
            }
          }
        }
      }
    }
  }
}

# Cloud Scheduler job to trigger the Cloud Run v2 Job nightly at 2:00 AM UTC
resource "google_cloud_scheduler_job" "maintenance_scheduler" {
  name        = "${var.prefix}-maintenance-scheduler"
  description = "Nightly trigger for APF maintenance Cloud Run job"
  schedule    = "0 2 * * *"
  time_zone   = "UTC"
  region      = var.region
  project     = var.project_id

  http_target {
    http_method = "POST"
    uri         = "https://${var.region}-run.googleapis.com/v2/projects/${var.project_id}/locations/${var.region}/jobs/${google_cloud_run_v2_job.maintenance.name}:run"

    oauth_token {
      service_account_email = google_service_account.job_scheduler.email
    }
  }
}
