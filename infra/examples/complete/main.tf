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

terraform {
  required_version = ">= 1.5.0"
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
    google-beta = {
      source  = "hashicorp/google-beta"
      version = "~> 5.0"
    }
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}

provider "google-beta" {
  project = var.project_id
  region  = var.region
}

module "agentic_foundry" {
  source = "../../modules"

  project_id                  = var.project_id
  project_number              = var.project_number
  region                      = var.region
  zone                        = var.zone
  prefix                      = var.prefix
  display_name_prefix         = var.display_name_prefix
  domain_name                 = var.domain_name
  cloud_run_ui_service_name   = var.cloud_run_ui_service_name
  cloud_run_api_service_name  = var.cloud_run_api_service_name
  cloud_run_ai_service_name   = var.cloud_run_ai_service_name
  iap_client_id               = var.iap_client_id
  iap_allowed_members         = var.iap_allowed_members
  subnet_range                = var.subnet_range
  secondary_subnet_range      = var.secondary_subnet_range
  vpc_connector_range         = var.vpc_connector_range
  manage_public_dns_zone      = var.manage_public_dns_zone
  public_dns_zone_name        = var.public_dns_zone_name
  public_dns_zone_name_to_use = var.public_dns_zone_name_to_use
}
