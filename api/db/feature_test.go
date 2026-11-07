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

package db

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestMockFeatureStore(t *testing.T) {
	ctx := context.Background()
	store := NewMockFeatureStore()

	// 1. Get non-existent feature -> return ErrFeatureNotFound
	_, err := store.GetFeature(ctx, "canvas_v2")
	if !errors.Is(err, ErrFeatureNotFound) {
		t.Fatalf("expected ErrFeatureNotFound, got: %v", err)
	}

	// 2. Save feature
	now := time.Now()
	feat := &Feature{
		ID:      "canvas_v2",
		Enabled: true,
		Properties: map[string]interface{}{
			"show_experimental_badge": true,
			"max_retry_count":         float64(3),
		},
		UpdatedAt: now,
	}
	if err := store.SaveFeature(ctx, feat); err != nil {
		t.Fatalf("failed to save feature: %v", err)
	}

	// 3. Get feature -> verify contents
	saved, err := store.GetFeature(ctx, "canvas_v2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if saved.ID != "canvas_v2" || !saved.Enabled {
		t.Errorf("unexpected feature state: %+v", saved)
	}
	if saved.Properties["show_experimental_badge"] != true {
		t.Errorf("expected show_experimental_badge property to be true")
	}

	// 4. List features
	list, err := store.ListFeatures(ctx)
	if err != nil {
		t.Fatalf("unexpected list error: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 feature, got %d", len(list))
	}

	// 5. Delete feature
	if err := store.DeleteFeature(ctx, "canvas_v2"); err != nil {
		t.Fatalf("failed to delete feature: %v", err)
	}
	_, err = store.GetFeature(ctx, "canvas_v2")
	if !errors.Is(err, ErrFeatureNotFound) {
		t.Errorf("expected ErrFeatureNotFound after deletion, got: %v", err)
	}
}
