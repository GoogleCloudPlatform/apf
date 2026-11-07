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

output "perf_testing_service_account_email" {
  description = "Email of the Performance Testing Service Account"
  value       = google_service_account.perf_testing.email
}

output "maintenance_job_name" {
  description = "Name of the Cloud Run v2 Maintenance Job"
  value       = google_cloud_run_v2_job.maintenance.name
}

output "maintenance_job_id" {
  description = "ID of the Cloud Run v2 Maintenance Job"
  value       = google_cloud_run_v2_job.maintenance.id
}

output "housekeeping_job_name" {
  description = "Name of the Cloud Run v2 Maintenance Job (legacy alias)"
  value       = google_cloud_run_v2_job.maintenance.name
}

output "housekeeping_job_id" {
  description = "ID of the Cloud Run v2 Maintenance Job (legacy alias)"
  value       = google_cloud_run_v2_job.maintenance.id
}

output "job_scheduler_service_account_email" {
  description = "Email of the Job Scheduler Service Account"
  value       = google_service_account.job_scheduler.email
}

