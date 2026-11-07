#!/usr/bin/env bash
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

# Helper script to mint an IAP OIDC ID Token for Staging performance testing
# Uses gcloud access token to request an IAP OIDC ID token for apf-stg-perf-testing SA

SA_EMAIL="${STAGING_PERF_SA_EMAIL:-${PERF_SA_EMAIL}}"
IAP_CLIENT_ID="${IAP_CLIENT_ID}"

if [ -z "$SA_EMAIL" ] || [ -z "$IAP_CLIENT_ID" ]; then
  echo "Error: STAGING_PERF_SA_EMAIL and IAP_CLIENT_ID environment variables must be set." >&2
  exit 1
fi

ACCESS_TOKEN=$(gcloud auth print-access-token 2>/dev/null || gcloud auth application-default print-access-token 2>/dev/null)

if [ -z "$ACCESS_TOKEN" ]; then
  echo "Error: Failed to obtain gcloud access token. Please run 'gcloud auth application-default login'." >&2
  exit 1
fi

TOKEN=$(curl -s -X POST \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"audience\": \"${IAP_CLIENT_ID}\", \"includeEmail\": true}" \
  "https://iamcredentials.googleapis.com/v1/projects/-/serviceAccounts/${SA_EMAIL}:generateIdToken" | jq -r '.token // empty')

echo "$TOKEN"
