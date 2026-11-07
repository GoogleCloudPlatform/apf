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

# SendGrid Integration Infrastructure
# This file manages the GCP resources needed for SendGrid.
# We use the official Google provider to store the API key securely and manage DNS.

# ------------------------------------------------------------------------------
# 1. Secret Manager for SendGrid API Key
# ------------------------------------------------------------------------------

resource "google_secret_manager_secret" "sendgrid_api_key" {
  secret_id = "${var.prefix}-sendgrid-api-key"
  project   = var.project_id

  replication {
    auto {}
  }

  labels = {
    environment = var.prefix
    service     = "sendgrid"
  }
}

# Note: We do not manage the secret version (the actual key value) in Terraform
# to prevent the sensitive API key from being stored in plain text in the
# terraform.tfvars or state files.
#
# After applying this Terraform, you should manually add the secret version:
# gcloud secrets versions add ${var.prefix}-sendgrid-api-key --data-file=path/to/api_key.txt

# ------------------------------------------------------------------------------
# 2. DNS Records for Domain Verification (Sender Authentication)
# ------------------------------------------------------------------------------
# SendGrid requires 3 CNAME records to verify your domain and enable DKIM/SPF.
#
# UNCOMMENT AND CUSTOMIZE THIS SECTION if you manage your public DNS zone
# in this Terraform configuration.
#
# Assuming you have a public managed zone defined like:
# data "google_dns_managed_zone" "public_zone" {
#   name    = "${var.prefix}-public-zone"
#   project = var.project_id
# }
#
# These subdomains (em1234, s1, s2) and their targets (u1234.wl.sendgrid.net, etc.)
# must be replaced with the exact values provided by SendGrid when you initiate
# Domain Authentication in the SendGrid Console.

/*
resource "google_dns_record_set" "sendgrid_cname_1" {
  project      = var.project_id
  managed_zone = data.google_dns_managed_zone.public_zone.name
  name         = "em1234.${var.domain_name}."
  type         = "CNAME"
  ttl          = 300
  rrdatas      = ["u1234567.wl.sendgrid.net."]
}

resource "google_dns_record_set" "sendgrid_cname_2" {
  project      = var.project_id
  managed_zone = data.google_dns_managed_zone.public_zone.name
  name         = "s1._domainkey.${var.domain_name}."
  type         = "CNAME"
  ttl          = 300
  rrdatas      = ["dkim.sendgrid.net."]
}

resource "google_dns_record_set" "sendgrid_cname_3" {
  project      = var.project_id
  managed_zone = data.google_dns_managed_zone.public_zone.name
  name         = "s2._domainkey.${var.domain_name}."
  type         = "CNAME"
  ttl          = 300
  rrdatas      = ["dkim.sendgrid.net."]
}
*/
