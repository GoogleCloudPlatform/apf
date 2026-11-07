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

output "ui_url" {
  value       = module.agentic_foundry.ui_url
  description = "URL of the UI Frontend"
}

output "api_url" {
  value       = module.agentic_foundry.api_url
  description = "URL of the API Backend"
}

output "ai_url" {
  value       = module.agentic_foundry.ai_url
  description = "URL of the AI Agent Engine"
}

output "load_balancer_ip" {
  value       = module.agentic_foundry.load_balancer_ip
  description = "External Load Balancer IP Address"
}
