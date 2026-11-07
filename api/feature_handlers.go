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
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"

	"google.com/apf/api/db"
)

var validFeatureIDRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

type FeatureHandler struct {
	FeatureStore db.FeatureStore
}

func NewFeatureHandler(featureStore db.FeatureStore) *FeatureHandler {
	return &FeatureHandler{
		FeatureStore: featureStore,
	}
}

type FeatureErrorResponse struct {
	Code  string `json:"code"`
	Error string `json:"error"`
}

func extractFeatureID(r *http.Request) string {
	id := r.PathValue("featureId")
	if id == "" || !validFeatureIDRegex.MatchString(id) {
		return ""
	}
	return id
}

// GetFeatureHandler processes GET /api/v1/features/{featureId} requests on-demand.
func (h *FeatureHandler) GetFeatureHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(FeatureErrorResponse{
			Code:  "METHOD_NOT_ALLOWED",
			Error: "Method not allowed",
		})
		return
	}

	featureID := extractFeatureID(r)
	if featureID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(FeatureErrorResponse{
			Code:  "INVALID_FEATURE_ID",
			Error: "Valid feature ID is required",
		})
		return
	}

	feature, err := h.FeatureStore.GetFeature(r.Context(), featureID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		if errors.Is(err, db.ErrFeatureNotFound) {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(FeatureErrorResponse{
				Code:  "FEATURE_NOT_FOUND",
				Error: fmt.Sprintf("Feature document '%s' not found", featureID),
			})
			return
		}
		slog.Error("Failed to fetch feature", "featureId", featureID, "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(FeatureErrorResponse{
			Code:  "INTERNAL_ERROR",
			Error: "An internal server error occurred",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(feature)
}

// SaveFeatureHandler processes PUT /api/v1/admin/features/{featureId} requests.
func (h *FeatureHandler) SaveFeatureHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(FeatureErrorResponse{
			Code:  "METHOD_NOT_ALLOWED",
			Error: "Method not allowed",
		})
		return
	}

	featureID := extractFeatureID(r)
	if featureID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(FeatureErrorResponse{
			Code:  "INVALID_FEATURE_ID",
			Error: "Valid feature ID is required",
		})
		return
	}

	// Limit body consumption to 1MB max to prevent memory exhaust DoS
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req db.Feature
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(FeatureErrorResponse{
			Code:  "INVALID_REQUEST",
			Error: "Invalid JSON payload or body too large",
		})
		return
	}

	req.ID = featureID
	if err := h.FeatureStore.SaveFeature(r.Context(), &req); err != nil {
		slog.Error("Failed to save feature", "featureId", featureID, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(FeatureErrorResponse{
			Code:  "INTERNAL_ERROR",
			Error: "An internal server error occurred",
		})
		return
	}

	saved, err := h.FeatureStore.GetFeature(r.Context(), featureID)
	if err != nil {
		slog.Error("Failed to re-read saved feature", "featureId", featureID, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(FeatureErrorResponse{
			Code:  "INTERNAL_ERROR",
			Error: "Failed to confirm saved feature document",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(saved)
}

// ListFeaturesHandler processes GET /api/v1/admin/features requests.
func (h *FeatureHandler) ListFeaturesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(FeatureErrorResponse{
			Code:  "METHOD_NOT_ALLOWED",
			Error: "Method not allowed",
		})
		return
	}

	features, err := h.FeatureStore.ListFeatures(r.Context())
	if err != nil {
		slog.Error("Failed to list features", "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(FeatureErrorResponse{
			Code:  "INTERNAL_ERROR",
			Error: "An internal server error occurred",
		})
		return
	}
	if features == nil {
		features = []db.Feature{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(features)
}
