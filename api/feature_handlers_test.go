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
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"google.com/apf/api/db"
)

func TestGetFeatureHandler(t *testing.T) {
	mockStore := db.NewMockFeatureStore()
	handler := NewFeatureHandler(mockStore)

	// 1. Test 404 Feature Not Found
	req := httptest.NewRequest("GET", "/api/v1/features/canvas_v2", nil)
	req.SetPathValue("featureId", "canvas_v2")
	rr := httptest.NewRecorder()

	handler.GetFeatureHandler(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 Not Found, got: %d", rr.Code)
	}

	var errResp FeatureErrorResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if errResp.Code != "FEATURE_NOT_FOUND" {
		t.Errorf("expected error code FEATURE_NOT_FOUND, got: %s", errResp.Code)
	}

	// 2. Save feature and test 200 OK
	now := time.Now().Truncate(time.Second)
	mockStore.SaveFeature(req.Context(), &db.Feature{
		ID:      "canvas_v2",
		Enabled: true,
		Properties: map[string]interface{}{
			"show_experimental_badge": true,
			"max_retry_count":         float64(3),
		},
		UpdatedAt: now,
	})

	req2 := httptest.NewRequest("GET", "/api/v1/features/canvas_v2", nil)
	req2.SetPathValue("featureId", "canvas_v2")
	rr2 := httptest.NewRecorder()

	handler.GetFeatureHandler(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK, got: %d", rr2.Code)
	}

	var feat db.Feature
	if err := json.Unmarshal(rr2.Body.Bytes(), &feat); err != nil {
		t.Fatalf("failed to decode feature response: %v", err)
	}
	if feat.ID != "canvas_v2" || !feat.Enabled {
		t.Errorf("unexpected feature state: %+v", feat)
	}
	if feat.Properties["show_experimental_badge"] != true {
		t.Errorf("expected property show_experimental_badge to be true")
	}
}

func TestAdminFeatureHandlers(t *testing.T) {
	mockStore := db.NewMockFeatureStore()
	handler := NewFeatureHandler(mockStore)

	// 1. Test SaveFeatureHandler (PUT)
	bodyPayload := `{"enabled": true, "properties": {"show_badge": true}}`
	req := httptest.NewRequest("PUT", "/api/v1/admin/features/agent_copilot", bytes.NewBufferString(bodyPayload))
	req.SetPathValue("featureId", "agent_copilot")
	rr := httptest.NewRecorder()

	handler.SaveFeatureHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK, got: %d", rr.Code)
	}

	var saved db.Feature
	if err := json.Unmarshal(rr.Body.Bytes(), &saved); err != nil {
		t.Fatalf("failed to decode saved feature: %v", err)
	}
	if saved.ID != "agent_copilot" || !saved.Enabled {
		t.Errorf("unexpected saved feature: %+v", saved)
	}

	// 2. Test ListFeaturesHandler (GET)
	reqList := httptest.NewRequest("GET", "/api/v1/admin/features", nil)
	rrList := httptest.NewRecorder()

	handler.ListFeaturesHandler(rrList, reqList)

	if rrList.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK, got: %d", rrList.Code)
	}

	var list []db.Feature
	if err := json.Unmarshal(rrList.Body.Bytes(), &list); err != nil {
		t.Fatalf("failed to decode list response: %v", err)
	}
	if len(list) != 1 || list[0].ID != "agent_copilot" {
		t.Errorf("unexpected list response: %+v", list)
	}
}
