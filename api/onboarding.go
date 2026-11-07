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
	"fmt"
	"net/http"
	"strings"
	"time"
)

type DiscoveryReport struct {
	Status             string                 `json:"status" firestore:"status"` // completed, failed
	ErrorDetails       string                 `json:"errorDetails,omitempty" firestore:"errorDetails,omitempty"`
	RuntimeStack       map[string]interface{} `json:"runtime_stack" firestore:"runtime_stack"`
	LegacySpecRegistry []interface{}          `json:"legacy_spec_registry" firestore:"legacy_spec_registry"`
	ScanDate           time.Time              `json:"scanDate" firestore:"scanDate"`
}

func getOnboardingReportHandler(w http.ResponseWriter, r *http.Request) {
	projectId := r.PathValue("id")

	// Fetch onboarding discovery document from subcollection
	docRef := FirestoreClient.Collection("projects").Doc(projectId).Collection("onboarding").Doc("discovery")
	docSnap, err := docRef.Get(r.Context())
	if err != nil {
		if strings.Contains(err.Error(), "NotFound") || strings.Contains(err.Error(), "not found") {
			sendJSON(w, http.StatusNotFound, Response{Message: "No discovery scan report found for this project. Please start the onboarding scan first.", Status: "ERROR", Title: "Onboarding Scan Required"})
			return
		}
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to fetch discovery report: %v", err), Status: "ERROR", Title: "Error Fetching Report"})
		return
	}

	var report DiscoveryReport
	if err := docSnap.DataTo(&report); err != nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to parse report data: %v", err), Status: "ERROR", Title: "Error Parsing Report"})
		return
	}

	sendJSON(w, http.StatusOK, report)
}
