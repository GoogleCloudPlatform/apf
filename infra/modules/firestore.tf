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

resource "google_firestore_database" "default" {
  project     = var.project_id
  name        = "${var.prefix}-firestore"
  type        = "FIRESTORE_NATIVE"
  location_id = var.region
}

resource "google_firestore_index" "messages_activity_index" {
  project     = var.project_id
  database    = google_firestore_database.default.name
  collection  = "messages"
  query_scope = "COLLECTION_GROUP"

  fields {
    field_path = "isNotification"
    order      = "ASCENDING"
  }

  fields {
    field_path = "projectId"
    order      = "ASCENDING"
  }

  fields {
    field_path = "timestamp"
    order      = "ASCENDING"
  }
}

resource "google_firestore_index" "messages_session_index" {
  project     = var.project_id
  database    = google_firestore_database.default.name
  collection  = "messages"
  query_scope = "COLLECTION"

  fields {
    field_path = "isTransient"
    order      = "ASCENDING"
  }

  fields {
    field_path = "timestamp"
    order      = "ASCENDING"
  }
}

resource "google_firestore_field" "users_userid_index" {
  project    = var.project_id
  database   = google_firestore_database.default.name
  collection = "users"
  field      = "userId"

  index_config {
    indexes {
      order       = "ASCENDING"
      query_scope = "COLLECTION_GROUP"
    }
    indexes {
      order       = "DESCENDING"
      query_scope = "COLLECTION_GROUP"
    }
    indexes {
      order       = "ASCENDING"
      query_scope = "COLLECTION"
    }
    indexes {
      order       = "DESCENDING"
      query_scope = "COLLECTION"
    }
  }
}


