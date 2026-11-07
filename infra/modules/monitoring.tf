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

resource "google_monitoring_dashboard" "end_to_end_logging" {
  project        = var.project_id
  dashboard_json = <<EOF
{
  "displayName": "${var.display_name_prefix} Log Monitor",
  "mosaicLayout": {
    "columns": "10",
    "tiles": [
      {
        "xPos": 0,
        "yPos": 0,
        "width": 8,
        "height": 4,
        "widget": {
          "title": "AI Service Logs",
          "logsPanel": {
            "filter": "resource.type=\"cloud_run_revision\" AND resource.labels.service_name=\"${var.cloud_run_ai_service_name}\"",
            "resourceNames": [
              "projects/${var.project_id}"
            ]
          }
        }
      },
      {
        "xPos": 8,
        "yPos": 0,
        "width": 2,
        "height": 4,
        "widget": {
          "title": "AI Metrics (Req & Latency)",
          "xyChart": {
            "dataSets": [
              {
                "timeSeriesQuery": {
                  "timeSeriesFilter": {
                    "filter": "metric.type=\"run.googleapis.com/request_count\" resource.type=\"cloud_run_revision\" AND resource.labels.service_name=\"${var.cloud_run_ai_service_name}\"",
                    "aggregation": {
                      "alignmentPeriod": "60s",
                      "perSeriesAligner": "ALIGN_RATE"
                    }
                  }
                },
                "plotType": "LINE",
                "legendTemplate": "Requests"
              },
              {
                "timeSeriesQuery": {
                  "timeSeriesFilter": {
                    "filter": "metric.type=\"run.googleapis.com/request_latencies\" resource.type=\"cloud_run_revision\" AND resource.labels.service_name=\"${var.cloud_run_ai_service_name}\"",
                    "aggregation": {
                      "alignmentPeriod": "60s",
                      "perSeriesAligner": "ALIGN_PERCENTILE_99"
                    }
                  }
                },
                "plotType": "LINE",
                "legendTemplate": "Latency (p99)",
                "targetAxis": "Y2"
              }
            ]
          }
        }
      },
      {
        "xPos": 0,
        "yPos": 4,
        "width": 8,
        "height": 4,
        "widget": {
          "title": "API Service Logs",
          "logsPanel": {
            "filter": "resource.type=\"cloud_run_revision\" AND resource.labels.service_name=\"${var.cloud_run_api_service_name}\"",
            "resourceNames": [
              "projects/${var.project_id}"
            ]
          }
        }
      },
      {
        "xPos": 8,
        "yPos": 4,
        "width": 2,
        "height": 4,
        "widget": {
          "title": "API Metrics (Req & Latency)",
          "xyChart": {
            "dataSets": [
              {
                "timeSeriesQuery": {
                  "timeSeriesFilter": {
                    "filter": "metric.type=\"run.googleapis.com/request_count\" resource.type=\"cloud_run_revision\" AND resource.labels.service_name=\"${var.cloud_run_api_service_name}\"",
                    "aggregation": {
                      "alignmentPeriod": "60s",
                      "perSeriesAligner": "ALIGN_RATE"
                    }
                  }
                },
                "plotType": "LINE",
                "legendTemplate": "Requests"
              },
              {
                "timeSeriesQuery": {
                  "timeSeriesFilter": {
                    "filter": "metric.type=\"run.googleapis.com/request_latencies\" resource.type=\"cloud_run_revision\" AND resource.labels.service_name=\"${var.cloud_run_api_service_name}\"",
                    "aggregation": {
                      "alignmentPeriod": "60s",
                      "perSeriesAligner": "ALIGN_PERCENTILE_99"
                    }
                  }
                },
                "plotType": "LINE",
                "legendTemplate": "Latency (p99)",
                "targetAxis": "Y2"
              }
            ]
          }
        }
      },
      {
        "xPos": 0,
        "yPos": 8,
        "width": 8,
        "height": 4,
        "widget": {
          "title": "UI Service Logs",
          "logsPanel": {
            "filter": "resource.type=\"cloud_run_revision\" AND resource.labels.service_name=\"${var.cloud_run_ui_service_name}\"",
            "resourceNames": [
              "projects/${var.project_id}"
            ]
          }
        }
      },
      {
        "xPos": 8,
        "yPos": 8,
        "width": 2,
        "height": 4,
        "widget": {
          "title": "UI Metrics (Req & Latency)",
          "xyChart": {
            "dataSets": [
              {
                "timeSeriesQuery": {
                  "timeSeriesFilter": {
                    "filter": "metric.type=\"run.googleapis.com/request_count\" resource.type=\"cloud_run_revision\" AND resource.labels.service_name=\"${var.cloud_run_ui_service_name}\"",
                    "aggregation": {
                      "alignmentPeriod": "60s",
                      "perSeriesAligner": "ALIGN_RATE"
                    }
                  }
                },
                "plotType": "LINE",
                "legendTemplate": "Requests"
              },
              {
                "timeSeriesQuery": {
                  "timeSeriesFilter": {
                    "filter": "metric.type=\"run.googleapis.com/request_latencies\" resource.type=\"cloud_run_revision\" AND resource.labels.service_name=\"${var.cloud_run_ui_service_name}\"",
                    "aggregation": {
                      "alignmentPeriod": "60s",
                      "perSeriesAligner": "ALIGN_PERCENTILE_99"
                    }
                  }
                },
                "plotType": "LINE",
                "legendTemplate": "Latency (p99)",
                "targetAxis": "Y2"
              }
            ]
          }
        }
      }
    ]
  }
}
EOF
  lifecycle {
    ignore_changes = [
      dashboard_json,
    ]
  }
}
