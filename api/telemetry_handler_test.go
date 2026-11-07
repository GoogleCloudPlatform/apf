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
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"google.com/apf/api/core"
	"google.com/apf/api/db"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type mockOrgStore struct {
	db.OrganizationStore
	orgs map[string]string
}

func (m *mockOrgStore) GetOrganization(ctx context.Context, id string) (*db.Organization, error) {
	if name, ok := m.orgs[id]; ok {
		return &db.Organization{ID: id, Name: name}, nil
	}
	return nil, fmt.Errorf("organization not found")
}

type TestEvent struct {
	ID             string    `gorm:"primaryKey;column:id"`
	SessionID      string    `gorm:"column:session_id"`
	UserID         string    `gorm:"column:user_id"`
	Timestamp      time.Time `gorm:"column:timestamp"`
	UsageMetadata  string    `gorm:"column:usage_metadata"`
	CustomMetadata string    `gorm:"column:custom_metadata"`
}

func (TestEvent) TableName() string {
	return "events"
}

type TestSession struct {
	ID     string `gorm:"primaryKey;column:id"`
	UserID string `gorm:"column:user_id"`
	State  string `gorm:"column:state"`
}

func (TestSession) TableName() string {
	return "sessions"
}

func setupTestDB(t *testing.T) *gorm.DB {
	dbName := fmt.Sprintf("file:memdb_%d?mode=memory&cache=shared", time.Now().UnixNano())
	gormDB, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open sqlite memory DB: %v", err)
	}

	err = gormDB.AutoMigrate(&TestSession{}, &TestEvent{})
	if err != nil {
		t.Fatalf("Failed to auto migrate test schema: %v", err)
	}

	return gormDB
}

func seedTestEvent(gormDB *gorm.DB, id, sessionID, userID, projectID, orgID, model string, prompt, completion, total int64, ts time.Time) {
	stateJSON, _ := json.Marshal(map[string]string{
		"projectId":      projectID,
		"organizationId": orgID,
	})
	sess := TestSession{
		ID:     sessionID,
		UserID: userID,
		State:  string(stateJSON),
	}
	gormDB.FirstOrCreate(&sess, TestSession{ID: sessionID})

	usageJSON, _ := json.Marshal(map[string]int64{
		"promptTokenCount":     prompt,
		"candidatesTokenCount": completion,
		"totalTokenCount":      total,
	})
	customJSON, _ := json.Marshal(map[string]string{
		"model": model,
	})

	evt := TestEvent{
		ID:             id,
		SessionID:      sessionID,
		UserID:         userID,
		Timestamp:      ts,
		UsageMetadata:  string(usageJSON),
		CustomMetadata: string(customJSON),
	}
	gormDB.Create(&evt)
}

func TestTelemetryHandler_RBAC(t *testing.T) {
	gormDB := setupTestDB(t)
	orgStore := &mockOrgStore{orgs: map[string]string{"org1": "Org One", "org2": "Org Two"}}
	handler := NewTelemetryHandler(gormDB, nil, orgStore)

	tests := []struct {
		name           string
		claims         *core.UserClaims
		url            string
		expectedStatus int
		expectedCode   string
	}{
		{
			name:           "Unauthenticated request returns 401 UNAUTHORIZED",
			claims:         nil,
			url:            "/api/v1/telemetry/usage",
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   "UNAUTHORIZED",
		},
		{
			name: "Standard user without SuperAdmin role returns 403 FORBIDDEN",
			claims: &core.UserClaims{
				Subject:        "user123",
				OrganizationID: "org1",
				IsSuperAdmin:   false,
			},
			url:            "/api/v1/telemetry/usage",
			expectedStatus: http.StatusForbidden,
			expectedCode:   "FORBIDDEN",
		},
		{
			name: "OrgAdmin without SuperAdmin role returns 403 FORBIDDEN",
			claims: &core.UserClaims{
				Subject:        "orgadmin1",
				OrganizationID: "org1",
				Roles:          []string{string(core.RoleOrgAdmin)},
				IsSuperAdmin:   false,
			},
			url:            "/api/v1/telemetry/usage?organizationId=org1",
			expectedStatus: http.StatusForbidden,
			expectedCode:   "FORBIDDEN",
		},
		{
			name: "SuperAdmin user allowed to query any organization",
			claims: &core.UserClaims{
				Subject:        "superadmin1",
				OrganizationID: "org1",
				IsSuperAdmin:   true,
			},
			url:            "/api/v1/telemetry/usage?organizationId=org2",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tc.url, nil)
			if tc.claims != nil {
				req = req.WithContext(context.WithValue(req.Context(), core.UserClaimsKey, tc.claims))
			}
			rr := httptest.NewRecorder()

			handler.GetTelemetryUsage(rr, req)

			if rr.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tc.expectedStatus, rr.Code, rr.Body.String())
			}

			if tc.expectedCode != "" {
				var errResp APFErrorPayload
				if err := json.Unmarshal(rr.Body.Bytes(), &errResp); err != nil {
					t.Fatalf("Failed to parse APF error payload: %v", err)
				}
				if errResp.Error.Code != tc.expectedCode {
					t.Errorf("Expected error code %s, got %s", tc.expectedCode, errResp.Error.Code)
				}
			}
		})
	}
}

func TestTelemetryHandler_AggregationAndCost(t *testing.T) {
	gormDB := setupTestDB(t)
	now := time.Now()

	seedTestEvent(gormDB, "evt1", "sess1", "user1", "proj1", "org1", "gemini-1.5-pro", 1000, 500, 1500, now.Add(-90*time.Minute))
	seedTestEvent(gormDB, "evt2", "sess1", "user1", "proj1", "org1", "gemini-1.5-pro", 2000, 1000, 3000, now.Add(-80*time.Minute))
	seedTestEvent(gormDB, "evt3", "sess2", "user2", "proj1", "org1", "gemini-1.5-flash", 5000, 2000, 7000, now.Add(-30*time.Minute))

	handler := NewTelemetryHandler(gormDB, nil, nil)
	claims := &core.UserClaims{
		Subject:        "admin1",
		OrganizationID: "org1",
		IsSuperAdmin:   true,
	}

	req := httptest.NewRequest("GET", "/api/v1/telemetry/usage?organizationId=org1", nil)
	req = req.WithContext(context.WithValue(req.Context(), core.UserClaimsKey, claims))
	rr := httptest.NewRecorder()

	handler.GetTelemetryUsage(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	var resp db.TelemetryUsageResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.PromptTokens != 8000 {
		t.Errorf("Expected 8000 prompt tokens, got %d", resp.PromptTokens)
	}
	if resp.CompletionTokens != 3500 {
		t.Errorf("Expected 3500 completion tokens, got %d", resp.CompletionTokens)
	}
	if resp.TotalTokens != 11500 {
		t.Errorf("Expected 11500 total tokens, got %d", resp.TotalTokens)
	}
	if resp.CostUSD <= 0 {
		t.Errorf("Expected positive costUSD, got %f", resp.CostUSD)
	}
	if len(resp.UserBreakdown) != 2 {
		t.Errorf("Expected 2 users in breakdown, got %d", len(resp.UserBreakdown))
	}
}

func TestTelemetryHandler_UnknownModelFallback(t *testing.T) {
	gormDB := setupTestDB(t)
	now := time.Now()

	seedTestEvent(gormDB, "evt_unk", "sess_unk", "user_unk", "proj1", "org1", "gemini-experimental", 10000, 5000, 15000, now)

	handler := NewTelemetryHandler(gormDB, nil, nil)
	claims := &core.UserClaims{
		Subject:        "admin1",
		OrganizationID: "org1",
		IsSuperAdmin:   true,
	}

	req := httptest.NewRequest("GET", "/api/v1/telemetry/usage?organizationId=org1", nil)
	req = req.WithContext(context.WithValue(req.Context(), core.UserClaimsKey, claims))
	rr := httptest.NewRecorder()

	handler.GetTelemetryUsage(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	var resp db.TelemetryUsageResponse
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if resp.TotalTokens != 15000 {
		t.Errorf("Expected 15000 total tokens, got %d", resp.TotalTokens)
	}
	if resp.CostUSD != 0.0031 {
		t.Errorf("Expected fallback baseline costUSD 0.0031, got %f", resp.CostUSD)
	}
}

func TestTelemetryHandler_SessionTelemetry(t *testing.T) {
	gormDB := setupTestDB(t)
	now := time.Now()

	seedTestEvent(gormDB, "e1", "sess_specific", "user1", "proj1", "org1", "gemini-1.5-pro", 1000, 500, 1500, now)
	seedTestEvent(gormDB, "e2", "sess_specific", "user1", "proj1", "org1", "gemini-1.5-pro", 3000, 1500, 4500, now)

	handler := NewTelemetryHandler(gormDB, nil, nil)
	claims := &core.UserClaims{
		Subject:        "admin1",
		OrganizationID: "org1",
		IsSuperAdmin:   true,
	}

	req := httptest.NewRequest("GET", "/api/v1/telemetry/usage?sessionId=sess_specific", nil)
	req = req.WithContext(context.WithValue(req.Context(), core.UserClaimsKey, claims))
	rr := httptest.NewRecorder()

	handler.GetTelemetryUsage(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	var resp db.TelemetryUsageResponse
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if resp.TurnCount != 2 {
		t.Errorf("Expected 2 turns, got %d", resp.TurnCount)
	}
	if resp.TotalTokens != 6000 {
		t.Errorf("Expected 6000 total tokens, got %d", resp.TotalTokens)
	}
	if resp.AverageTokensPerTurn != 3000 {
		t.Errorf("Expected average tokens per turn 3000, got %f", resp.AverageTokensPerTurn)
	}
}

func TestTelemetryHandler_PaginationAndCSV(t *testing.T) {
	gormDB := setupTestDB(t)
	now := time.Now()

	for i := 1; i <= 10; i++ {
		uid := fmt.Sprintf("user_%d", i)
		sid := fmt.Sprintf("sess_%d", i)
		seedTestEvent(gormDB, fmt.Sprintf("evt_%d", i), sid, uid, "proj1", "org1", "gemini-1.5-pro", 1000, 500, 1500, now)
	}

	handler := NewTelemetryHandler(gormDB, nil, nil)
	claims := &core.UserClaims{
		Subject:        "admin1",
		OrganizationID: "org1",
		IsSuperAdmin:   true,
	}

	req := httptest.NewRequest("GET", "/api/v1/telemetry/usage?organizationId=org1&limit=3&offset=0", nil)
	req = req.WithContext(context.WithValue(req.Context(), core.UserClaimsKey, claims))
	rr := httptest.NewRecorder()

	handler.GetTelemetryUsage(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", rr.Code)
	}

	var resp db.TelemetryUsageResponse
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if len(resp.UserBreakdown) != 3 {
		t.Errorf("Expected 3 users in breakdown page, got %d", len(resp.UserBreakdown))
	}
	if resp.TotalUsers != 10 {
		t.Errorf("Expected totalUsers=10, got %d", resp.TotalUsers)
	}

	// Test max cap limit > 500
	reqCap := httptest.NewRequest("GET", "/api/v1/telemetry/usage?organizationId=org1&limit=1000", nil)
	reqCap = reqCap.WithContext(context.WithValue(reqCap.Context(), core.UserClaimsKey, claims))
	rrCap := httptest.NewRecorder()

	handler.GetTelemetryUsage(rrCap, reqCap)
	var respCap db.TelemetryUsageResponse
	json.Unmarshal(rrCap.Body.Bytes(), &respCap)

	if respCap.Limit != 500 {
		t.Errorf("Expected limit capped at 500, got %d", respCap.Limit)
	}

	// Test CSV export format
	reqCSV := httptest.NewRequest("GET", "/api/v1/telemetry/usage?organizationId=org1&format=csv", nil)
	reqCSV = reqCSV.WithContext(context.WithValue(reqCSV.Context(), core.UserClaimsKey, claims))
	rrCSV := httptest.NewRecorder()

	handler.GetTelemetryUsage(rrCSV, reqCSV)

	if rrCSV.Code != http.StatusOK {
		t.Fatalf("Expected status 200 for CSV, got %d", rrCSV.Code)
	}
	ct := rrCSV.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/csv") {
		t.Errorf("Expected Content-Type text/csv, got %s", ct)
	}
	cd := rrCSV.Header().Get("Content-Disposition")
	if !strings.Contains(cd, "attachment; filename=") {
		t.Errorf("Expected Content-Disposition header with filename, got %s", cd)
	}

	r := csv.NewReader(strings.NewReader(rrCSV.Body.String()))
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("Failed to parse CSV response: %v", err)
	}
	if len(records) < 2 {
		t.Fatalf("Expected CSV header and rows, got %d lines", len(records))
	}
}

func TestTelemetryHandler_CachingAndSingleflight(t *testing.T) {
	gormDB := setupTestDB(t)
	now := time.Now()

	seedTestEvent(gormDB, "e_cache", "s_cache", "u_cache", "p_cache", "org1", "gemini-1.5-pro", 2000, 1000, 3000, now)

	handler := NewTelemetryHandler(gormDB, nil, nil)
	claims := &core.UserClaims{
		Subject:        "admin1",
		OrganizationID: "org1",
		IsSuperAdmin:   true,
	}

	req1 := httptest.NewRequest("GET", "/api/v1/telemetry/usage?organizationId=org1", nil)
	req1 = req1.WithContext(context.WithValue(req1.Context(), core.UserClaimsKey, claims))
	rr1 := httptest.NewRecorder()
	handler.GetTelemetryUsage(rr1, req1)

	if rr1.Code != http.StatusOK {
		t.Fatalf("Initial request failed with status %d", rr1.Code)
	}

	seedTestEvent(gormDB, "e_cache2", "s_cache", "u_cache", "p_cache", "org1", "gemini-1.5-pro", 5000, 5000, 10000, now)

	req2 := httptest.NewRequest("GET", "/api/v1/telemetry/usage?organizationId=org1", nil)
	req2 = req2.WithContext(context.WithValue(req2.Context(), core.UserClaimsKey, claims))
	rr2 := httptest.NewRecorder()
	handler.GetTelemetryUsage(rr2, req2)

	var respCached db.TelemetryUsageResponse
	json.Unmarshal(rr2.Body.Bytes(), &respCached)
	if respCached.TotalTokens != 3000 {
		t.Errorf("Expected cached totalTokens=3000, got %d", respCached.TotalTokens)
	}

	reqRefresh := httptest.NewRequest("GET", "/api/v1/telemetry/usage?organizationId=org1&refresh=true", nil)
	reqRefresh = reqRefresh.WithContext(context.WithValue(reqRefresh.Context(), core.UserClaimsKey, claims))
	rrRefresh := httptest.NewRecorder()
	handler.GetTelemetryUsage(rrRefresh, reqRefresh)

	var respRefreshed db.TelemetryUsageResponse
	json.Unmarshal(rrRefresh.Body.Bytes(), &respRefreshed)
	if respRefreshed.TotalTokens != 13000 {
		t.Errorf("Expected refreshed totalTokens=13000, got %d", respRefreshed.TotalTokens)
	}

	var wg sync.WaitGroup
	var callCount int32
	concurrentRequests := 10

	for i := 0; i < concurrentRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			reqC := httptest.NewRequest("GET", "/api/v1/telemetry/usage?organizationId=org1&refresh=true", nil)
			reqC = reqC.WithContext(context.WithValue(reqC.Context(), core.UserClaimsKey, claims))
			rrC := httptest.NewRecorder()
			handler.GetTelemetryUsage(rrC, reqC)
			if rrC.Code == http.StatusOK {
				atomic.AddInt32(&callCount, 1)
			}
		}()
	}
	wg.Wait()

	if callCount != int32(concurrentRequests) {
		t.Errorf("Expected all %d concurrent requests to succeed, got %d", concurrentRequests, callCount)
	}
}

func TestTelemetryHandler_DatabaseUnavailable(t *testing.T) {
	handler := NewTelemetryHandler(nil, nil, nil)
	claims := &core.UserClaims{
		Subject:        "admin1",
		OrganizationID: "org1",
		IsSuperAdmin:   true,
	}

	req := httptest.NewRequest("GET", "/api/v1/telemetry/usage?organizationId=org1", nil)
	req = req.WithContext(context.WithValue(req.Context(), core.UserClaimsKey, claims))
	rr := httptest.NewRecorder()

	handler.GetTelemetryUsage(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("Expected status 503 DATABASE_UNAVAILABLE, got %d", rr.Code)
	}

	var errResp APFErrorPayload
	if err := json.Unmarshal(rr.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("Failed to parse APF error response: %v", err)
	}
	if errResp.Error.Code != "DATABASE_UNAVAILABLE" {
		t.Errorf("Expected code DATABASE_UNAVAILABLE, got %s", errResp.Error.Code)
	}
}

func TestTelemetryHandler_ProjectAndUserFiltering(t *testing.T) {
	gormDB := setupTestDB(t)
	now := time.Now()

	seedTestEvent(gormDB, "evt1", "sess1", "user1", "proj1", "org1", "gemini-1.5-pro", 1000, 500, 1500, now.Add(-10*time.Minute))
	seedTestEvent(gormDB, "evt2", "sess2", "user2", "proj2", "org1", "gemini-1.5-pro", 2000, 1000, 3000, now.Add(-5*time.Minute))

	handler := NewTelemetryHandler(gormDB, nil, nil)
	claims := &core.UserClaims{
		Subject:        "admin1",
		OrganizationID: "org1",
		IsSuperAdmin:   true,
	}

	req := httptest.NewRequest("GET", "/api/v1/telemetry/usage?organizationId=org1&projectId=proj1", nil)
	req = req.WithContext(context.WithValue(req.Context(), core.UserClaimsKey, claims))
	rr := httptest.NewRecorder()

	handler.GetTelemetryUsage(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	var resp db.TelemetryUsageResponse
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if resp.TotalTokens != 1500 {
		t.Errorf("Expected 1500 total tokens for proj1, got %d", resp.TotalTokens)
	}
	if len(resp.ProjectBreakdown) != 1 || resp.ProjectBreakdown[0].ProjectID != "proj1" {
		t.Errorf("Expected projectBreakdown to only contain proj1, got %v", resp.ProjectBreakdown)
	}
}
