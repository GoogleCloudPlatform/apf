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

	"google.com/apf/api/db"
)

func seedFeatures(ctx context.Context) {
	features := []db.Feature{
		{
			ID:      "project_list",
			Enabled: true,
			Properties: map[string]interface{}{
				"enable_share_projects": false,
			},
		},
		{
			ID:      "workflows",
			Enabled: true,
			Properties: map[string]interface{}{
				"enable_workflow_run": false,
			},
		},
		{
			ID:      "app_banners",
			Enabled: true,
			Properties: map[string]interface{}{
				"show_alpha_notice": true,
				"show_version":      true,
			},
		},
	}

	for _, feature := range features {
		existing, err := featureStore.GetFeature(ctx, feature.ID)
		if err == nil && existing != nil {
			log.Printf("Feature '%s' already seeded, skipping", feature.ID)
			continue
		}
		if err := featureStore.SaveFeature(ctx, &feature); err != nil {
			log.Printf("Failed to seed feature '%s': %v", feature.ID, err)
		} else {
			log.Printf("Successfully seeded feature '%s'", feature.ID)
		}
	}
}
