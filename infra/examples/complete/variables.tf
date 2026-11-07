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
  description = "Google Cloud Project ID"
}

variable "project_number" {
  type        = string
  description = "Google Cloud Project Number"
}

variable "region" {
  type        = string
  default     = "us-central1"
  description = "GCP Region"
}

variable "zone" {
  type        = string
  default     = "us-central1-c"
  description = "GCP Zone for zonal resources"
}

variable "prefix" {
  type        = string
  default     = "apf"
  description = "Resource naming prefix"
}

variable "display_name_prefix" {
  type        = string
  default     = "Agentic Foundry"
  description = "Display name prefix"
}

variable "domain_name" {
  type        = string
  description = "Custom domain name (e.g. foundry.example.com)"
}

variable "cloud_run_ui_service_name" {
  type        = string
  default     = "apf-ui"
}

variable "cloud_run_api_service_name" {
  type        = string
  default     = "apf-api"
}

variable "cloud_run_ai_service_name" {
  type        = string
  default     = "apf-ai"
}

variable "iap_client_id" {
  type        = string
  description = "OAuth 2.0 Client ID for Identity-Aware Proxy (IAP)"
}

variable "iap_allowed_members" {
  type        = list(string)
  default     = []
  description = "List of IAM principals/domains allowed to access IAP (e.g. ['domain:example.com'])"
}

variable "subnet_range" {
  type        = string
  default     = "10.10.0.0/24"
}

variable "secondary_subnet_range" {
  type        = string
  default     = "192.168.0.0/24"
}

variable "vpc_connector_range" {
  type        = string
  default     = "10.10.1.0/28"
}

variable "manage_public_dns_zone" {
  type        = bool
  default     = false
}

variable "public_dns_zone_name" {
  type        = string
  default     = ""
}

variable "public_dns_zone_name_to_use" {
  type        = string
  default     = ""
}
