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

resource "google_monitoring_dashboard" "tracing" {
  project        = var.project_id
  dashboard_json = <<EOF
{
  "displayName": "${var.display_name_prefix} Trace Monitor",
  "gridLayout": {
    "columns": "2",
    "widgets": [
      {
        "title": "UI Latency Heatmap",
        "xyChart": {
          "dataSets": [
            {
              "timeSeriesQuery": {
                "timeSeriesFilter": {
                  "filter": "metric.type=\"run.googleapis.com/request_latencies\" resource.type=\"cloud_run_revision\" AND resource.labels.service_name=\"${var.cloud_run_ui_service_name}\""
                }
              },
              "plotType": "HEATMAP"
            }
          ]
        }
      },
      {
        "title": "API Latency Heatmap",
        "xyChart": {
          "dataSets": [
            {
              "timeSeriesQuery": {
                "timeSeriesFilter": {
                  "filter": "metric.type=\"run.googleapis.com/request_latencies\" resource.type=\"cloud_run_revision\" AND resource.labels.service_name=\"${var.cloud_run_api_service_name}\""
                }
              },
              "plotType": "HEATMAP"
            }
          ]
        }
      },
      {
        "title": "AI Latency Heatmap",
        "xyChart": {
          "dataSets": [
            {
              "timeSeriesQuery": {
                "timeSeriesFilter": {
                  "filter": "metric.type=\"run.googleapis.com/request_latencies\" resource.type=\"cloud_run_revision\" AND resource.labels.service_name=\"${var.cloud_run_ai_service_name}\""
                }
              },
              "plotType": "HEATMAP"
            }
          ]
        }
      },
      {
        "title": "p99 Latency by Service",
        "xyChart": {
          "dataSets": [
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
              "legendTemplate": "UI"
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
              "legendTemplate": "API"
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
              "legendTemplate": "AI"
            }
          ]
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
