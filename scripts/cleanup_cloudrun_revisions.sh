#!/bin/bash
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

# scripts/cleanup_cloudrun_revisions.sh
# Removes Cloud Run revisions older than the last 5 versions for APF services.

set -euo pipefail

PROJECT_ID="${GOOGLE_CLOUD_PROJECT:-agenticproductfoundry-489717}"
REGION="${GOOGLE_CLOUD_LOCATION:-us-central1}"
KEEP_COUNT="${KEEP_COUNT:-5}"
DRY_RUN="${DRY_RUN:-false}"
TARGET_ENV=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --dry-run|-d)
      DRY_RUN=true
      shift
      ;;
    --keep|-k)
      KEEP_COUNT="$2"
      shift 2
      ;;
    --keep=*)
      KEEP_COUNT="${1#*=}"
      shift
      ;;
    -*)
      echo "Unknown flag: $1" >&2
      exit 1
      ;;
    *)
      if [ -z "$TARGET_ENV" ]; then
        TARGET_ENV="$1"
      fi
      shift
      ;;
  esac
done

TARGET_ENV="${TARGET_ENV:-staging}" # staging, prod, all

SERVICES=()
case "$TARGET_ENV" in
  staging|stg)
    SERVICES=("apf-stg-ui" "apf-stg-ai" "apf-stg-api")
    ;;
  production|prod)
    SERVICES=("apf-prod-ui" "apf-prod-ai" "apf-prod-api")
    ;;
  all|both)
    SERVICES=("apf-stg-ui" "apf-stg-ai" "apf-stg-api" "apf-prod-ui" "apf-prod-ai" "apf-prod-api")
    ;;
  *)
    # Custom single service
    SERVICES=("$TARGET_ENV")
    ;;
esac

echo "================================================================="
echo "Cloud Run Revisions Cleanup"
echo "Project:   $PROJECT_ID"
echo "Region:    $REGION"
echo "Services:  ${SERVICES[*]}"
echo "Keep Last: $KEEP_COUNT revisions per service"
echo "Dry Run:   $DRY_RUN"
echo "================================================================="

for svc in "${SERVICES[@]}"; do
  echo ""
  echo ">>> Processing Cloud Run Service: $svc"

  if [ "$DRY_RUN" = "true" ] || [ "$DRY_RUN" = "1" ]; then
    echo "[DRY RUN] Stale revisions to delete (older than last $KEEP_COUNT):"
    gcloud run revisions list --project="$PROJECT_ID" --region="$REGION" --service="$svc" --sort-by="~CREATE_TIME" --format="value(name)" 2>/dev/null | awk "NR>$KEEP_COUNT" || true
  else
    # Single-liner awk + xargs deletion of revisions older than last KEEP_COUNT versions
    gcloud run revisions list --project="$PROJECT_ID" --region="$REGION" --service="$svc" --sort-by="~CREATE_TIME" --format="value(name)" 2>/dev/null | awk "NR>$KEEP_COUNT" | xargs -r -n 1 gcloud run revisions delete --region="$REGION" --project="$PROJECT_ID" --quiet
  fi
done

echo ""
echo "Cloud Run revisions cleanup completed."
