// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"log"
	"os"
	"strings"

	"google.com/apf/api/db"
)

func seedProdData(ctx context.Context) {
	bucket := os.Getenv("GCS_ARTIFACT_BUCKET")
	if bucket == "" {
		bucket = "apf-prod-workspaces"
	}
	if !strings.HasPrefix(bucket, "gs://") {
		bucket = "gs://" + bucket
	}
	globalConfig := db.GlobalConfig{
		GCSWorkspaceBucket: bucket,
	}
	if err := configStore.SetGlobalConfig(ctx, &globalConfig); err != nil {
		log.Printf("Failed to seed prod global config: %v", err)
	} else {
		log.Printf("Successfully seeded prod global configuration")
	}
}
