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

resource "google_filestore_instance" "nfs_instance" {
  name     = "${var.prefix}-nfs"
  location = var.zone
  tier     = "BASIC_HDD"

  file_shares {
    capacity_gb = 1024 # Minimum allowed for BASIC_HDD is 1TB
    name        = "workspaces"
  }

  networks {
    network = module.vpc.network_name
    modes   = ["MODE_IPV4"]
  }
}

# Private DNS zone for resolving the Filestore IP internally via a static hostname
resource "google_dns_managed_zone" "internal_zone" {
  name        = "${var.prefix}-internal-zone"
  dns_name    = "apf.internal."
  description = "Private DNS zone for internal services"
  visibility  = "private"

  private_visibility_config {
    networks {
      network_url = module.vpc.network_self_link
    }
  }
}

# A record pointing nfs.apf.internal to the Filestore IP
resource "google_dns_record_set" "nfs_dns" {
  name         = "nfs.apf.internal."
  managed_zone = google_dns_managed_zone.internal_zone.name
  type         = "A"
  ttl          = 300
  rrdatas      = [google_filestore_instance.nfs_instance.networks[0].ip_addresses[0]]
}

# Firewall rule to allow NFS traffic within the VPC
resource "google_compute_firewall" "allow_nfs" {
  name    = "${var.prefix}-allow-nfs"
  network = module.vpc.network_name

  allow {
    protocol = "tcp"
    ports    = ["111", "2049", "20048"]
  }

  allow {
    protocol = "udp"
    ports    = ["111", "2049", "20048"]
  }

  source_ranges = [
    var.subnet_range,
    var.vpc_connector_range
  ]
}
