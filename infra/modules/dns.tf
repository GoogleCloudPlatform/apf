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

resource "google_dns_managed_zone" "private_zone" {
  name        = "${var.prefix}-private-zone"
  dns_name    = "run.app."
  description = "Private DNS zone for internal Cloud Run communication"
  visibility  = "private"

  private_visibility_config {
    networks {
      network_url = module.vpc.network_self_link
    }
  }
}

resource "google_dns_record_set" "cname" {
  name         = "*.run.app."
  managed_zone = google_dns_managed_zone.private_zone.name
  type         = "CNAME"
  ttl          = 300
  rrdatas      = ["restricted.googleapis.com."]
}

resource "google_dns_record_set" "soa" {
  name         = "run.app."
  managed_zone = google_dns_managed_zone.private_zone.name
  type         = "SOA"
  ttl          = 21600
  rrdatas      = ["ns-gcp-private.googledomains.com. cloud-dns-hostmaster.google.com. 1 21600 3600 1209600 300"]
}

resource "google_dns_record_set" "ns" {
  name         = "run.app."
  managed_zone = google_dns_managed_zone.private_zone.name
  type         = "NS"
  ttl          = 21600
  rrdatas      = ["ns-gcp-private.googledomains.com."]
}

resource "google_dns_managed_zone" "googleapis_zone" {
  name        = "${var.prefix}-googleapis-zone"
  dns_name    = "googleapis.com."
  description = "Private DNS zone for googleapis.com"
  visibility  = "private"

  private_visibility_config {
    networks {
      network_url = module.vpc.network_self_link
    }
  }
}

resource "google_dns_record_set" "restricted_a" {
  name         = "restricted.googleapis.com."
  managed_zone = google_dns_managed_zone.googleapis_zone.name
  type         = "A"
  ttl          = 300
  rrdatas = [
    "199.36.153.4",
    "199.36.153.5",
    "199.36.153.6",
    "199.36.153.7"
  ]
}

resource "google_dns_record_set" "private_a" {
  name         = "private.googleapis.com."
  managed_zone = google_dns_managed_zone.googleapis_zone.name
  type         = "A"
  ttl          = 300
  rrdatas = [
    "199.36.153.8",
    "199.36.153.9",
    "199.36.153.10",
    "199.36.153.11"
  ]
}

resource "google_dns_record_set" "googleapis_cname" {
  name         = "*.googleapis.com."
  managed_zone = google_dns_managed_zone.googleapis_zone.name
  type         = "CNAME"
  ttl          = 300
  rrdatas      = ["restricted.googleapis.com."]
}

# Bypass Restricted VIP for Gemini AI Studio API endpoint (not supported on restricted.googleapis.com)
resource "google_dns_record_set" "generativelanguage_cname" {
  name         = "generativelanguage.googleapis.com."
  managed_zone = google_dns_managed_zone.googleapis_zone.name
  type         = "CNAME"
  ttl          = 300
  rrdatas      = ["private.googleapis.com."]
}

# ------------------------------------------------------------------------------
# Public DNS Zone (Managed conditionally, e.g., in Prod)
# ------------------------------------------------------------------------------

resource "google_dns_managed_zone" "public_zone" {
  count = var.manage_public_dns_zone ? 1 : 0

  name        = var.public_dns_zone_name
  dns_name    = "${var.domain_name}."
  description = "DNS zone for domain: ${var.domain_name}"
  visibility  = "public"

  dnssec_config {
    state = "on"

    # Matching existing config
    default_key_specs {
      algorithm  = "rsasha256"
      key_length = 2048
      key_type   = "keySigning"
    }
    default_key_specs {
      algorithm  = "rsasha256"
      key_length = 1024
      key_type   = "zoneSigning"
    }
  }
}

# Apex A record for the managed public zone (e.g., deltaproductfoundry.app -> Prod LB IP)
resource "google_dns_record_set" "apex_a" {
  count = var.manage_public_dns_zone ? 1 : 0

  project      = var.project_id
  managed_zone = google_dns_managed_zone.public_zone[0].name
  name         = "${var.domain_name}."
  type         = "A"
  ttl          = 300
  rrdatas      = [google_compute_global_address.default.address]
}

# ------------------------------------------------------------------------------
# Public DNS Records in an Externally Managed Zone (e.g., Staging registering in Prod's zone)
# ------------------------------------------------------------------------------

data "google_dns_managed_zone" "referenced_public_zone" {
  count = (var.public_dns_zone_name_to_use != "" && !var.manage_public_dns_zone) ? 1 : 0

  name    = var.public_dns_zone_name_to_use
  project = var.project_id
}

# Subdomain A record in the referenced public zone (e.g., staging.deltaproductfoundry.app -> Staging LB IP)
resource "google_dns_record_set" "subdomain_a" {
  count = (var.public_dns_zone_name_to_use != "" && !var.manage_public_dns_zone) ? 1 : 0

  project      = var.project_id
  managed_zone = data.google_dns_managed_zone.referenced_public_zone[0].name
  name         = "${var.domain_name}."
  type         = "A"
  ttl          = 300
  rrdatas      = [google_compute_global_address.default.address]
}
