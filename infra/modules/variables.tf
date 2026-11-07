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

variable "project_id" {
  type        = string
  description = "Project ID"
}

variable "zone" {
  type        = string
  description = "Zone for zonal resources (like Filestore)"
}


variable "project_number" {
  type        = string
  description = "Project Number"
}

variable "region" {
  type        = string
  description = "Region for this infrastructure"
}

variable "prefix" {
  type        = string
  description = "Name for this infrastructure"
}

variable "display_name_prefix" {
  type        = string
  description = "display name for readable text in UI (e.g. APF Staging)"
}


variable "cloud_run_ui_service_name" {
  type        = string
  description = "Name of the Cloud Run UI service"
}

variable "subnet_range" {
  type        = string
  description = "CIDR range for the subnet"
}

variable "secondary_subnet_range" {
  type        = string
  description = "secondary CIDR range for the subnet"
}

variable "cloud_run_api_service_name" {
  type        = string
  description = "Name of the Cloud Run API service"
}

variable "iap_client_id" {
  type        = string
  description = "OAuth 2.0 Client ID for IAP"
}
variable "cloud_run_ai_service_name" {
  type        = string
  description = "Name of the Cloud Run AI service"
}

variable "vpc_connector_range" {
  type        = string
  description = "CIDR range for the VPC access connector"
}

variable "domain_name" {
  type        = string
  description = "Custom domain name for the environment"
}

variable "apis" {
  type        = list(string)
  description = "List of GCP APIs to enable in the project"
  default = [
    "compute.googleapis.com",
    "servicenetworking.googleapis.com",
    "aiplatform.googleapis.com",         # Vertex AI
    "generativelanguage.googleapis.com", # Gemini
    "discoveryengine.googleapis.com",    # Agent Builder
    "connectors.googleapis.com",
    "storage-component.googleapis.com", # Cloud Storage
    "servicedirectory.googleapis.com",
    "dns.googleapis.com",
    "cloudkms.googleapis.com",
    "run.googleapis.com",
    "cloudbuild.googleapis.com",
    "cloudresourcemanager.googleapis.com",
    "iap.googleapis.com",
    "cloudtrace.googleapis.com",
    "artifactregistry.googleapis.com",
    "file.googleapis.com",
    "identitytoolkit.googleapis.com",
    "cloudscheduler.googleapis.com"
  ]
}

variable "manage_public_dns_zone" {
  type        = bool
  default     = false
  description = "Whether to manage the public DNS zone for the domain in this environment"
}

variable "public_dns_zone_name" {
  type        = string
  default     = ""
  description = "The name of the public DNS zone in GCP (required if manage_public_dns_zone is true or if public_dns_zone_name_to_use is set)"
}

variable "public_dns_zone_name_to_use" {
  type        = string
  default     = ""
  description = "The name of the existing public DNS zone to use for registering records (used when manage_public_dns_zone is false)"
}

variable "create_iap_secret" {
  type        = bool
  default     = false
  description = "Whether to create and manage the iap-client-secret Secret Manager resource"
}

variable "iap_allowed_members" {
  type        = list(string)
  default     = []
  description = "List of IAM members/domains authorized to access the IAP protected services (e.g. ['domain:example.com', 'user:admin@example.com'])"
}