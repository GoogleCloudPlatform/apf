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

resource "google_artifact_registry_repository" "default" {
  location      = var.region
  repository_id = "${var.prefix}-repo"
  description   = "Artifact Registry for Agentic Foundry"
  format        = "DOCKER"

  project = var.project_id

  # Enable active deletions (set to true if you want to preview deletions in logs first)
  cleanup_policy_dry_run = false

  # Keep policy to explicitly protect the 10 most recent versions
  cleanup_policies {
    id     = "keep-minimum-10"
    action = "KEEP"
    most_recent_versions {
      keep_count = 10
    }
  }

  # Keep policy to explicitly protect images tagged as 'latest'
  cleanup_policies {
    id     = "keep-latest"
    action = "KEEP"
    condition {
      tag_state    = "TAGGED"
      tag_prefixes = ["latest"]
    }
  }

  # Policy to delete versions older than 2 months (60 days / 5184000s)
  cleanup_policies {
    id     = "delete-older-than-2-months"
    action = "DELETE"
    condition {
      tag_state  = "ANY"
      older_than = "5184000s"
    }
  }

  # Policy to automatically delete any image/cache versions that do not have any tags
  cleanup_policies {
    id     = "delete-untagged"
    action = "DELETE"
    condition {
      tag_state = "UNTAGGED"
    }
  }

  depends_on = [
    google_project_service.apis
  ]
}

