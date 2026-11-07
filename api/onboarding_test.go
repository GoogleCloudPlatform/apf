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
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"google.com/apf/api/core"
	"google.com/apf/api/db"
)

type mockOnboardingProjectStore struct {
	db.ProjectStore
	project *db.Project
}

func (m *mockOnboardingProjectStore) GetProject(ctx context.Context, organizationID string, id string) (*db.Project, error) {
	if m.project == nil || m.project.ID != id {
		return nil, fmt.Errorf("project not found")
	}
	return m.project, nil
}

func (m *mockOnboardingProjectStore) UpdateProject(ctx context.Context, organizationID string, p *db.Project) error {
	m.project = p
	return nil
}

func TestGetOnboardingReportHandler(t *testing.T) {
	emulatorHost := os.Getenv("FIRESTORE_EMULATOR_HOST")
	if emulatorHost == "" {
		t.Skip("Skipping Firestore integration test: FIRESTORE_EMULATOR_HOST is not set")
	}

	ctx := context.Background()
	client, err := firestore.NewClient(ctx, "agenticproductfoundry-489717")
	if err != nil {
		t.Fatalf("Failed to connect to Firestore emulator: %v", err)
	}
	defer client.Close()
	FirestoreClient = client

	projectId := "project-report-test-456"
	reportRef := client.Collection("projects").Doc(projectId).Collection("onboarding").Doc("discovery")
	
	// Pre-seed mock report document
	mockReport := DiscoveryReport{
		Status:   "completed",
		ScanDate: time.Now().Truncate(time.Second),
		RuntimeStack: map[string]interface{}{
			"languages": []interface{}{
				map[string]interface{}{"name": "Go", "confidence": "high"},
			},
			"frameworks": []interface{}{
				map[string]interface{}{"name": "Gin", "confidence": "high"},
			},
		},
		LegacySpecRegistry: []interface{}{
			map[string]interface{}{"path": "docs/prd.md", "title": "Requirements"},
		},
	}
	_, err = reportRef.Set(ctx, &mockReport)
	if err != nil {
		t.Fatalf("Failed to seed mock report: %v", err)
	}
	defer func() {
		_, _ = reportRef.Delete(ctx)
	}()

	// Mock project details for access middleware
	projName := "Report Test Project"
	status := "Onboarding"
	owner := "user-1"
	proj := &db.Project{
		ID:      projectId,
		Name:    &projName,
		Status:  &status,
		OwnerId: &owner,
	}
	store = &mockOnboardingProjectStore{project: proj}

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/projects/%s/onboard/report", projectId), nil)
	req.SetPathValue("id", projectId)
	claims := &core.UserClaims{
		Subject: "user-1",
		Roles:   []string{"member"},
	}
	req = withTestCtx(req, claims)

	w := httptest.NewRecorder()
	handler := projectAccessMiddleware(getOnboardingReportHandler)
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}

	var fetched DiscoveryReport
	if err := json.NewDecoder(w.Body).Decode(&fetched); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if fetched.Status != "completed" {
		t.Errorf("Expected fetched report status to be 'completed', got '%s'", fetched.Status)
	}

	langs, ok := fetched.RuntimeStack["languages"].([]interface{})
	if !ok || len(langs) == 0 {
		t.Error("Expected languages in runtime stack to be parsed")
	} else {
		firstLang := langs[0].(map[string]interface{})
		if firstLang["name"] != "Go" {
			t.Errorf("Expected first language to be 'Go', got '%v'", firstLang["name"])
		}
	}
}
