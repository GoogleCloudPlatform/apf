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

module "vpc" {
  source  = "terraform-google-modules/network/google"
  version = "~> 18.1"


  project_id   = var.project_id
  network_name = "${var.prefix}-vpc"
  routing_mode = "GLOBAL"

  subnets = [
    {
      subnet_name           = "${var.prefix}-subnet-us-central"
      subnet_ip             = var.subnet_range
      subnet_region         = var.region
      subnet_private_access = "true"
      subnet_flow_logs      = "true"
      description           = "Subnet for central (default and closest region)"
    }
  ]

  secondary_ranges = {
    "${var.prefix}-subnet-us-central" = [
      {
        range_name    = "${var.prefix}-subnet-us-central-secondary"
        ip_cidr_range = var.secondary_subnet_range
      },
    ]
  }
}