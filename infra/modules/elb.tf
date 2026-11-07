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

data "google_secret_manager_secret_version" "iap_secret" {
  project = var.project_id
  secret  = var.create_iap_secret ? google_secret_manager_secret.iap_client_secret[0].secret_id : "iap-client-secret"
  version = "latest"
}

resource "google_compute_global_address" "default" {
  name = "${var.prefix}-lb-ip"
}

resource "google_compute_url_map" "default" {
  name            = "${var.prefix}-url-map"
  default_service = google_compute_backend_service.ui.id

  host_rule {
    hosts        = ["*"]
    path_matcher = "allpaths"
  }

  path_matcher {
    name            = "allpaths"
    default_service = google_compute_backend_service.ui.id

    path_rule {
      paths   = ["/api/*", "/health"]
      service = google_compute_backend_service.api.id
    }
  }
}

resource "random_id" "cert" {
  byte_length = 4
  keepers = {
    domain = var.domain_name
  }
}

resource "google_compute_managed_ssl_certificate" "default" {
  name = "${var.prefix}-cert-${random_id.cert.hex}"

  managed {
    domains = [var.domain_name]
  }

  lifecycle {
    create_before_destroy = true
  }
}

resource "google_compute_target_https_proxy" "default" {
  name             = "${var.prefix}-target-https-proxy"
  url_map          = google_compute_url_map.default.id
  ssl_certificates = [google_compute_managed_ssl_certificate.default.id]
}

resource "google_compute_global_forwarding_rule" "https" {
  name                  = "${var.prefix}-forwarding-rule-https"
  ip_protocol           = "TCP"
  load_balancing_scheme = "EXTERNAL"
  port_range            = "443"
  target                = google_compute_target_https_proxy.default.id
  ip_address            = google_compute_global_address.default.id
}

resource "google_compute_url_map" "redirect" {
  name = "${var.prefix}-redirect-url-map"

  default_url_redirect {
    https_redirect         = true
    redirect_response_code = "MOVED_PERMANENTLY_DEFAULT"
    strip_query            = false
  }
}

resource "google_compute_target_http_proxy" "default" {
  name    = "${var.prefix}-target-proxy"
  url_map = google_compute_url_map.redirect.id
}

resource "google_compute_global_forwarding_rule" "default" {
  name                  = "${var.prefix}-forwarding-rule"
  ip_protocol           = "TCP"
  load_balancing_scheme = "EXTERNAL"
  port_range            = "80"
  target                = google_compute_target_http_proxy.default.id
  ip_address            = google_compute_global_address.default.id
}

resource "google_compute_backend_service" "ui" {
  name                  = "${var.prefix}-ui-backend-service"
  load_balancing_scheme = "EXTERNAL"
  protocol              = "HTTP"

  backend {
    group = google_compute_region_network_endpoint_group.ui_neg.id
  }

  iap {
    enabled              = true
    oauth2_client_id     = var.iap_client_id
    oauth2_client_secret = data.google_secret_manager_secret_version.iap_secret.secret_data
  }
}

resource "google_compute_region_network_endpoint_group" "ui_neg" {
  name                  = "${var.prefix}-ui-neg"
  network_endpoint_type = "SERVERLESS"
  region                = var.region

  cloud_run {
    service = var.cloud_run_ui_service_name
  }
}

resource "google_compute_backend_service" "api" {
  name                  = "${var.prefix}-api-backend-service"
  load_balancing_scheme = "EXTERNAL"
  protocol              = "HTTP"

  backend {
    group = google_compute_region_network_endpoint_group.api_neg.id
  }

  iap {
    enabled              = true
    oauth2_client_id     = var.iap_client_id
    oauth2_client_secret = data.google_secret_manager_secret_version.iap_secret.secret_data
  }
}

resource "google_compute_region_network_endpoint_group" "api_neg" {
  name                  = "${var.prefix}-api-neg"
  network_endpoint_type = "SERVERLESS"
  region                = var.region

  cloud_run {
    service = var.cloud_run_api_service_name
  }
}

output "api_iap_audience" {
  description = "The IAP audience string for the API backend service"
  value       = "/projects/${var.project_number}/global/backendServices/${google_compute_backend_service.api.generated_id}"
}
