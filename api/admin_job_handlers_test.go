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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	run "google.golang.org/api/run/v2"

	"google.com/apf/api/core"
)

type mockCloudRunJobRunner struct {
	lastJobName      string
	lastDryRun       bool
	mockResp         *JobRunResponse
	mockErr          error
	callCount        int
	lastPageSize     int
	lastPageToken    string
	mockListExecResp *JobExecutionsListResponse
	mockListExecErr  error
	listExecCount    int
}

func (m *mockCloudRunJobRunner) RunJob(ctx context.Context, jobName string, dryRun bool) (*JobRunResponse, error) {
	m.callCount++
	m.lastJobName = jobName
	m.lastDryRun = dryRun
	if m.mockErr != nil {
		return nil, m.mockErr
	}
	if m.mockResp != nil {
		return m.mockResp, nil
	}
	return &JobRunResponse{
		Status:        "PENDING",
		ExecutionName: fmt.Sprintf("projects/test-proj/locations/us-central1/jobs/test-job/executions/%s-mock-123", jobName),
		JobName:       jobName,
	}, nil
}

func (m *mockCloudRunJobRunner) ListExecutions(ctx context.Context, pageSize int, pageToken string) (*JobExecutionsListResponse, error) {
	m.listExecCount++
	m.lastPageSize = pageSize
	m.lastPageToken = pageToken
	if m.mockListExecErr != nil {
		return nil, m.mockListExecErr
	}
	if m.mockListExecResp != nil {
		return m.mockListExecResp, nil
	}
	return &JobExecutionsListResponse{
		Executions: []*JobExecutionItem{
			{
				ID:                "exec-123",
				Name:              "projects/test-proj/locations/us-central1/jobs/test-job/executions/exec-123",
				JobName:           "nfs-audit",
				DryRun:            false,
				Status:            "SUCCEEDED",
				DurationSeconds:   12,
				DurationFormatted: "12s",
			},
		},
		NextPageToken: "",
	}, nil
}

func TestAdminJobHandlers_SuperAdminTriggerAuditJob(t *testing.T) {
	runner := &mockCloudRunJobRunner{}
	handler := NewAdminHandler(nil, nil, &core.Configuration{
		GoogleCloudProject:  "test-project",
		GoogleCloudLocation: "us-central1",
	}).WithJobRunner(runner)

	bodyBytes, _ := json.Marshal(JobRunRequest{DryRun: false})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/jobs/nfs-audit/run", bytes.NewReader(bodyBytes))
	req.SetPathValue("jobName", "nfs-audit")

	// Set super-admin claims
	claims := &core.UserClaims{
		Subject:      "admin-user-1",
		Username:     "admin@google.com",
		IsSuperAdmin: true,
	}
	ctx := context.WithValue(req.Context(), core.UserClaimsKey, claims)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.RunCloudRunJob(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected status 202 Accepted, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp JobRunResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Status != "PENDING" {
		t.Errorf("expected status 'PENDING', got %s", resp.Status)
	}
	if resp.JobName != "nfs-audit" {
		t.Errorf("expected jobName 'nfs-audit', got %s", resp.JobName)
	}
	if resp.ExecutionName == "" {
		t.Errorf("expected non-empty executionName")
	}

	if runner.lastJobName != "nfs-audit" {
		t.Errorf("expected runner lastJobName 'nfs-audit', got %s", runner.lastJobName)
	}
	if runner.lastDryRun != false {
		t.Errorf("expected runner lastDryRun false, got %v", runner.lastDryRun)
	}
}

func TestAdminJobHandlers_SuperAdminTriggerCleanupJobWithDryRun(t *testing.T) {
	runner := &mockCloudRunJobRunner{}
	handler := NewAdminHandler(nil, nil, &core.Configuration{
		GoogleCloudProject:  "test-project",
		GoogleCloudLocation: "us-central1",
	}).WithJobRunner(runner)

	bodyBytes, _ := json.Marshal(JobRunRequest{DryRun: true})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/jobs/hanging-cleanup/run", bytes.NewReader(bodyBytes))
	req.SetPathValue("jobName", "hanging-cleanup")

	claims := &core.UserClaims{
		Subject:      "admin-user-1",
		Username:     "admin@google.com",
		IsSuperAdmin: true,
	}
	ctx := context.WithValue(req.Context(), core.UserClaimsKey, claims)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.RunCloudRunJob(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected status 202 Accepted, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp JobRunResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Status != "PENDING" {
		t.Errorf("expected status 'PENDING', got %s", resp.Status)
	}
	if resp.JobName != "hanging-cleanup" {
		t.Errorf("expected jobName 'hanging-cleanup', got %s", resp.JobName)
	}

	if runner.lastJobName != "hanging-cleanup" {
		t.Errorf("expected runner lastJobName 'hanging-cleanup', got %s", runner.lastJobName)
	}
	if runner.lastDryRun != true {
		t.Errorf("expected runner lastDryRun true, got %v", runner.lastDryRun)
	}
}

func TestAdminJobHandlers_NonSuperAdminRejection(t *testing.T) {
	runner := &mockCloudRunJobRunner{}
	handler := NewAdminHandler(nil, nil, nil).WithJobRunner(runner)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/jobs/nfs-audit/run", nil)
	req.SetPathValue("jobName", "nfs-audit")

	// Standard tenant user (not super admin)
	claims := &core.UserClaims{
		Subject:      "user-1",
		Username:     "user@tenant.com",
		IsSuperAdmin: false,
	}
	ctx := context.WithValue(req.Context(), core.UserClaimsKey, claims)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.RunCloudRunJob(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403 Forbidden, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp Response
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if resp.Status != "ERROR" {
		t.Errorf("expected status 'ERROR', got %s", resp.Status)
	}
	if resp.Message != "Forbidden: Super Administrator access required" {
		t.Errorf("expected forbidden message, got %s", resp.Message)
	}

	if runner.callCount != 0 {
		t.Errorf("expected runner not to be called, got callCount %d", runner.callCount)
	}
}

func TestAdminJobHandlers_MissingClaims(t *testing.T) {
	runner := &mockCloudRunJobRunner{}
	handler := NewAdminHandler(nil, nil, nil).WithJobRunner(runner)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/jobs/nfs-audit/run", nil)
	req.SetPathValue("jobName", "nfs-audit")

	rec := httptest.NewRecorder()
	handler.RunCloudRunJob(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403 Forbidden, got %d", rec.Code)
	}

	if runner.callCount != 0 {
		t.Errorf("expected runner not to be called")
	}
}

func TestAdminJobHandlers_EmptyJobName(t *testing.T) {
	runner := &mockCloudRunJobRunner{}
	handler := NewAdminHandler(nil, nil, nil).WithJobRunner(runner)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/jobs//run", nil)
	req.SetPathValue("jobName", "")

	claims := &core.UserClaims{
		Subject:      "admin-user-1",
		IsSuperAdmin: true,
	}
	ctx := context.WithValue(req.Context(), core.UserClaimsKey, claims)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.RunCloudRunJob(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 Bad Request, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAdminJobHandlers_MalformedJSONPayload(t *testing.T) {
	runner := &mockCloudRunJobRunner{}
	handler := NewAdminHandler(nil, nil, nil).WithJobRunner(runner)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/jobs/nfs-audit/run", bytes.NewReader([]byte("{invalid-json")))
	req.SetPathValue("jobName", "nfs-audit")

	claims := &core.UserClaims{
		Subject:      "admin-user-1",
		IsSuperAdmin: true,
	}
	ctx := context.WithValue(req.Context(), core.UserClaimsKey, claims)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.RunCloudRunJob(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 Bad Request, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp Response
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Message != "Invalid request payload" {
		t.Errorf("expected 'Invalid request payload', got %s", resp.Message)
	}
}

func TestAdminJobHandlers_CloudRunSDKError(t *testing.T) {
	runner := &mockCloudRunJobRunner{
		mockErr: errors.New("gRPC quota exceeded: rateLimitExceeded"),
	}
	handler := NewAdminHandler(nil, nil, nil).WithJobRunner(runner)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/jobs/nfs-audit/run", nil)
	req.SetPathValue("jobName", "nfs-audit")

	claims := &core.UserClaims{
		Subject:      "admin-user-1",
		IsSuperAdmin: true,
	}
	ctx := context.WithValue(req.Context(), core.UserClaimsKey, claims)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.RunCloudRunJob(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500 Internal Server Error, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp Response
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Status != "ERROR" {
		t.Errorf("expected status 'ERROR', got %s", resp.Status)
	}
	if !bytes.Contains([]byte(resp.Message), []byte("rateLimitExceeded")) {
		t.Errorf("expected error message to contain details, got %s", resp.Message)
	}
}

func TestAdminJobHandlers_SimulatedRunner(t *testing.T) {
	sim := NewSimulatedCloudRunJobRunner(&core.Configuration{
		GoogleCloudProject:  "my-proj",
		GoogleCloudLocation: "us-central1",
	})
	resp, err := sim.RunJob(context.Background(), "nfs-audit", true)
	if err != nil {
		t.Fatalf("expected no error from simulated runner, got: %v", err)
	}
	if resp.Status != "PENDING" {
		t.Errorf("expected status 'PENDING', got %s", resp.Status)
	}
	if resp.JobName != "nfs-audit" {
		t.Errorf("expected jobName 'nfs-audit', got %s", resp.JobName)
	}
	if !bytes.Contains([]byte(resp.ExecutionName), []byte("my-proj")) {
		t.Errorf("expected executionName to contain project name, got %s", resp.ExecutionName)
	}

	listResp, err := sim.ListExecutions(context.Background(), 0, "")
	if err != nil {
		t.Fatalf("expected no error from simulated ListExecutions, got: %v", err)
	}
	if len(listResp.Executions) == 0 {
		t.Fatalf("expected simulated executions, got none")
	}

	// Test pagination in simulated runner
	pagedResp, err := sim.ListExecutions(context.Background(), 2, "")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(pagedResp.Executions) != 2 {
		t.Errorf("expected 2 paged executions, got %d", len(pagedResp.Executions))
	}
	if pagedResp.NextPageToken != "simulated-next-token" {
		t.Errorf("expected next token 'simulated-next-token', got %s", pagedResp.NextPageToken)
	}
}

func TestAdminJobHandlers_ListJobCatalog_SuperAdmin(t *testing.T) {
	handler := NewAdminHandler(nil, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/jobs", nil)
	claims := &core.UserClaims{
		Subject:      "admin-user",
		IsSuperAdmin: true,
	}
	req = req.WithContext(context.WithValue(req.Context(), core.UserClaimsKey, claims))
	rec := httptest.NewRecorder()

	handler.ListJobCatalog(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	var catalog []JobCatalogItem
	if err := json.NewDecoder(rec.Body).Decode(&catalog); err != nil {
		t.Fatalf("failed to decode catalog JSON: %v", err)
	}

	if len(catalog) == 0 {
		t.Fatalf("expected non-empty job catalog")
	}

	foundNFSAudit := false
	for _, item := range catalog {
		if item.Name == "nfs-audit" {
			foundNFSAudit = true
			if item.Description == "" {
				t.Errorf("expected non-empty description for nfs-audit")
			}
			if len(item.Parameters) == 0 {
				t.Errorf("expected parameters for nfs-audit")
			} else {
				hasDryRun := false
				for _, p := range item.Parameters {
					if p.Name == "dryRun" && p.Type == "boolean" {
						hasDryRun = true
					}
				}
				if !hasDryRun {
					t.Errorf("expected dryRun parameter on nfs-audit")
				}
			}
		}
	}

	if !foundNFSAudit {
		t.Errorf("expected nfs-audit to be present in job catalog")
	}
}

func TestAdminJobHandlers_ListJobCatalog_NonSuperAdmin(t *testing.T) {
	handler := NewAdminHandler(nil, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/jobs", nil)
	claims := &core.UserClaims{
		Subject:      "standard-user",
		IsSuperAdmin: false,
	}
	req = req.WithContext(context.WithValue(req.Context(), core.UserClaimsKey, claims))
	rec := httptest.NewRecorder()

	handler.ListJobCatalog(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403 Forbidden, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAdminJobHandlers_ListJobCatalog_MissingClaims(t *testing.T) {
	handler := NewAdminHandler(nil, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/jobs", nil)
	rec := httptest.NewRecorder()

	handler.ListJobCatalog(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403 Forbidden, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAdminJobHandlers_ListJobExecutions_SuperAdmin(t *testing.T) {
	runner := &mockCloudRunJobRunner{
		mockListExecResp: &JobExecutionsListResponse{
			Executions: []*JobExecutionItem{
				{
					ID:                "exec-1",
					Name:              "projects/p/locations/l/jobs/j/executions/exec-1",
					JobName:           "nfs-audit",
					DryRun:            true,
					Status:            "SUCCEEDED",
					CreateTime:        "2026-08-31T12:00:00Z",
					StartTime:         "2026-08-31T12:00:01Z",
					CompletionTime:    "2026-08-31T12:01:15Z",
					DurationSeconds:   74,
					DurationFormatted: "1m 14s",
					LogURI:            "https://cloudlogging.example.com",
				},
			},
			NextPageToken: "token-next-456",
		},
	}
	handler := NewAdminHandler(nil, nil, nil).WithJobRunner(runner)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/jobs/executions?pageSize=10&pageToken=token-prev-123", nil)
	claims := &core.UserClaims{
		Subject:      "admin-user",
		IsSuperAdmin: true,
	}
	req = req.WithContext(context.WithValue(req.Context(), core.UserClaimsKey, claims))
	rec := httptest.NewRecorder()

	handler.ListJobExecutions(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	if runner.lastPageSize != 10 {
		t.Errorf("expected pageSize 10, got %d", runner.lastPageSize)
	}
	if runner.lastPageToken != "token-prev-123" {
		t.Errorf("expected pageToken 'token-prev-123', got %s", runner.lastPageToken)
	}

	var resp JobExecutionsListResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode executions response: %v", err)
	}

	if len(resp.Executions) != 1 {
		t.Fatalf("expected 1 execution, got %d", len(resp.Executions))
	}
	item := resp.Executions[0]
	if item.JobName != "nfs-audit" || item.DryRun != true || item.Status != "SUCCEEDED" || item.DurationFormatted != "1m 14s" {
		t.Errorf("unexpected execution item values: %+v", item)
	}
	if resp.NextPageToken != "token-next-456" {
		t.Errorf("expected nextPageToken 'token-next-456', got %s", resp.NextPageToken)
	}
}

func TestAdminJobHandlers_ListJobExecutions_NonSuperAdmin(t *testing.T) {
	runner := &mockCloudRunJobRunner{}
	handler := NewAdminHandler(nil, nil, nil).WithJobRunner(runner)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/jobs/executions", nil)
	claims := &core.UserClaims{
		Subject:      "user-1",
		IsSuperAdmin: false,
	}
	req = req.WithContext(context.WithValue(req.Context(), core.UserClaimsKey, claims))
	rec := httptest.NewRecorder()

	handler.ListJobExecutions(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403 Forbidden, got %d: %s", rec.Code, rec.Body.String())
	}
	if runner.listExecCount != 0 {
		t.Errorf("runner should not be called")
	}
}

func TestAdminJobHandlers_ListJobExecutions_MissingClaims(t *testing.T) {
	runner := &mockCloudRunJobRunner{}
	handler := NewAdminHandler(nil, nil, nil).WithJobRunner(runner)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/jobs/executions", nil)
	rec := httptest.NewRecorder()

	handler.ListJobExecutions(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403 Forbidden, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAdminJobHandlers_ListJobExecutions_RunnerError(t *testing.T) {
	runner := &mockCloudRunJobRunner{
		mockListExecErr: errors.New("cloud run API unavailable"),
	}
	handler := NewAdminHandler(nil, nil, nil).WithJobRunner(runner)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/jobs/executions", nil)
	claims := &core.UserClaims{
		Subject:      "admin-user",
		IsSuperAdmin: true,
	}
	req = req.WithContext(context.WithValue(req.Context(), core.UserClaimsKey, claims))
	rec := httptest.NewRecorder()

	handler.ListJobExecutions(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAdminJobHandlers_ParseCloudRunExecution(t *testing.T) {
	t.Run("Failed execution with container args and error conditions", func(t *testing.T) {
		exec := &run.GoogleCloudRunV2Execution{
			Name:           "projects/test-proj/locations/us-central1/jobs/apf-housekeeping/executions/exec-abc-123",
			CreateTime:     "2026-08-31T10:00:00Z",
			StartTime:      "2026-08-31T10:00:02Z",
			CompletionTime: "2026-08-31T10:00:44Z",
			FailedCount:    1,
			Template: &run.GoogleCloudRunV2TaskTemplate{
				Containers: []*run.GoogleCloudRunV2Container{
					{
						Args: []string{"nfs-audit", "--dry-run"},
					},
				},
			},
			Conditions: []*run.GoogleCloudRunV2Condition{
				{
					Type:    "Completed",
					State:   "CONDITION_FAILED",
					Message: "Task 0 failed with exit code 1",
				},
			},
		}

		item := parseCloudRunExecution(exec, "test-proj", "us-central1", "apf-housekeeping")
		if item.ID != "exec-abc-123" {
			t.Errorf("expected ID 'exec-abc-123', got %s", item.ID)
		}
		if item.JobName != "nfs-audit" {
			t.Errorf("expected jobName 'nfs-audit', got %s", item.JobName)
		}
		if !item.DryRun {
			t.Errorf("expected dryRun true")
		}
		if item.Status != "FAILED" {
			t.Errorf("expected status 'FAILED', got %s", item.Status)
		}
		if item.DurationSeconds != 42 {
			t.Errorf("expected durationSeconds 42, got %d", item.DurationSeconds)
		}
		if item.DurationFormatted != "42s" {
			t.Errorf("expected durationFormatted '42s', got %s", item.DurationFormatted)
		}
		if item.ErrorMessage != "Task 0 failed with exit code 1" {
			t.Errorf("expected error message, got %s", item.ErrorMessage)
		}
		if !bytes.Contains([]byte(item.LogURI), []byte("exec-abc-123")) {
			t.Errorf("expected logURI to contain execution ID, got %s", item.LogURI)
		}
	})

	t.Run("Succeeded execution with custom LogUri", func(t *testing.T) {
		exec := &run.GoogleCloudRunV2Execution{
			Name:           "projects/test-proj/locations/us-central1/jobs/apf-housekeeping/executions/exec-xyz-456",
			CreateTime:     "2026-08-31T10:00:00Z",
			StartTime:      "2026-08-31T10:00:00Z",
			CompletionTime: "2026-08-31T10:01:14Z",
			SucceededCount: 1,
			LogUri:         "https://console.cloud.google.com/logs/custom-link",
			Template: &run.GoogleCloudRunV2TaskTemplate{
				Containers: []*run.GoogleCloudRunV2Container{
					{
						Args: []string{"hanging-cleanup"},
					},
				},
			},
			Conditions: []*run.GoogleCloudRunV2Condition{
				{
					Type:  "Completed",
					State: "CONDITION_SUCCEEDED",
				},
			},
		}

		item := parseCloudRunExecution(exec, "test-proj", "us-central1", "apf-housekeeping")
		if item.JobName != "hanging-cleanup" {
			t.Errorf("expected jobName 'hanging-cleanup', got %s", item.JobName)
		}
		if item.DryRun {
			t.Errorf("expected dryRun false")
		}
		if item.Status != "SUCCEEDED" {
			t.Errorf("expected status 'SUCCEEDED', got %s", item.Status)
		}
		if item.DurationSeconds != 74 {
			t.Errorf("expected durationSeconds 74, got %d", item.DurationSeconds)
		}
		if item.DurationFormatted != "1m 14s" {
			t.Errorf("expected durationFormatted '1m 14s', got %s", item.DurationFormatted)
		}
		if item.LogURI != "https://console.cloud.google.com/logs/custom-link" {
			t.Errorf("expected custom LogUri, got %s", item.LogURI)
		}
	})

	t.Run("Cancelled execution", func(t *testing.T) {
		exec := &run.GoogleCloudRunV2Execution{
			Name:           "projects/test-proj/locations/us-central1/jobs/apf-housekeeping/executions/exec-cancelled-789",
			CancelledCount: 1,
			StartTime:      "2026-08-31T10:00:00Z",
		}

		item := parseCloudRunExecution(exec, "test-proj", "us-central1", "apf-housekeeping")
		if item.Status != "CANCELLED" {
			t.Errorf("expected status 'CANCELLED', got %s", item.Status)
		}
	})

	t.Run("Nil execution handling", func(t *testing.T) {
		item := parseCloudRunExecution(nil, "p", "l", "j")
		if item == nil || item.Status != "UNKNOWN" {
			t.Errorf("expected UNKNOWN status for nil execution, got %+v", item)
		}
	})

	t.Run("Nil container and nil condition slice elements", func(t *testing.T) {
		exec := &run.GoogleCloudRunV2Execution{
			Name: "projects/test-proj/locations/us-central1/jobs/apf-housekeeping/executions/exec-nil-elements",
			Template: &run.GoogleCloudRunV2TaskTemplate{
				Containers: []*run.GoogleCloudRunV2Container{nil},
			},
			Conditions: []*run.GoogleCloudRunV2Condition{nil},
		}

		item := parseCloudRunExecution(exec, "test-proj", "us-central1", "apf-housekeeping")
		if item == nil {
			t.Fatalf("expected non-nil item")
		}
		if item.JobName != "apf-housekeeping" {
			t.Errorf("expected defaultJobName fallback, got %s", item.JobName)
		}
	})
}

func TestAdminJobHandlers_SimulatedRunnerPagination(t *testing.T) {
	sim := NewSimulatedCloudRunJobRunner(&core.Configuration{
		GoogleCloudProject:  "test-proj",
		GoogleCloudLocation: "us-central1",
	})

	// First page with pageSize=1
	page1, err := sim.ListExecutions(context.Background(), 1, "")
	if err != nil {
		t.Fatalf("expected no error on page 1, got %v", err)
	}
	if len(page1.Executions) != 1 {
		t.Fatalf("expected 1 execution on page 1, got %d", len(page1.Executions))
	}
	if page1.NextPageToken != "simulated-next-token" {
		t.Errorf("expected NextPageToken 'simulated-next-token', got %s", page1.NextPageToken)
	}

	// Second page using token
	page2, err := sim.ListExecutions(context.Background(), 1, page1.NextPageToken)
	if err != nil {
		t.Fatalf("expected no error on page 2, got %v", err)
	}
	if len(page2.Executions) < 1 {
		t.Fatalf("expected at least 1 execution on page 2, got %d", len(page2.Executions))
	}
}

func TestAdminJobHandlers_FormatDuration(t *testing.T) {
	tests := []struct {
		d        time.Duration
		expected string
	}{
		{d: 0, expected: "0s"},
		{d: -5 * time.Second, expected: "0s"},
		{d: 42 * time.Second, expected: "42s"},
		{d: 60 * time.Second, expected: "1m"},
		{d: 74 * time.Second, expected: "1m 14s"},
		{d: 3600 * time.Second, expected: "1h"},
		{d: 3605 * time.Second, expected: "1h 5s"},
		{d: 3660 * time.Second, expected: "1h 1m"},
		{d: 3723 * time.Second, expected: "1h 2m 3s"},
	}

	for _, tt := range tests {
		got := formatDuration(tt.d)
		if got != tt.expected {
			t.Errorf("formatDuration(%v) = %q, expected %q", tt.d, got, tt.expected)
		}
	}
}

func TestAdminJobHandlers_MiddlewareAndRouteIntegration(t *testing.T) {
	runner := &mockCloudRunJobRunner{}
	handler := NewAdminHandler(nil, nil, nil).WithJobRunner(runner)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/admin/jobs", superAdminAuthMiddleware(handler.ListJobCatalog))
	mux.HandleFunc("GET /api/v1/admin/jobs/executions", superAdminAuthMiddleware(handler.ListJobExecutions))
	mux.HandleFunc("POST /api/v1/admin/jobs/{jobName}/run", superAdminAuthMiddleware(handler.RunCloudRunJob))

	// 1. GET /api/v1/admin/jobs - Forbidden when unauthenticated
	req1 := httptest.NewRequest(http.MethodGet, "/api/v1/admin/jobs", nil)
	rec1 := httptest.NewRecorder()
	mux.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for missing claims, got %d", rec1.Code)
	}

	// 2. GET /api/v1/admin/jobs - OK for super admin
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/admin/jobs", nil)
	claims := &core.UserClaims{
		Subject:      "super-admin",
		IsSuperAdmin: true,
	}
	req2 = req2.WithContext(context.WithValue(req2.Context(), core.UserClaimsKey, claims))
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Errorf("expected 200 OK for super admin catalog, got %d: %s", rec2.Code, rec2.Body.String())
	}

	// 3. GET /api/v1/admin/jobs/executions - Forbidden when unauthenticated
	req3 := httptest.NewRequest(http.MethodGet, "/api/v1/admin/jobs/executions", nil)
	rec3 := httptest.NewRecorder()
	mux.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for executions missing claims, got %d", rec3.Code)
	}

	// 4. GET /api/v1/admin/jobs/executions - OK for super admin
	req4 := httptest.NewRequest(http.MethodGet, "/api/v1/admin/jobs/executions", nil)
	req4 = req4.WithContext(context.WithValue(req4.Context(), core.UserClaimsKey, claims))
	rec4 := httptest.NewRecorder()
	mux.ServeHTTP(rec4, req4)
	if rec4.Code != http.StatusOK {
		t.Errorf("expected 200 OK for super admin executions, got %d: %s", rec4.Code, rec4.Body.String())
	}

	// 5. POST /api/v1/admin/jobs/{jobName}/run - OK for super admin
	req5 := httptest.NewRequest(http.MethodPost, "/api/v1/admin/jobs/nfs-audit/run", bytes.NewReader([]byte(`{"dryRun":true}`)))
	req5 = req5.WithContext(context.WithValue(req5.Context(), core.UserClaimsKey, claims))
	rec5 := httptest.NewRecorder()
	mux.ServeHTTP(rec5, req5)
	if rec5.Code != http.StatusAccepted {
		t.Errorf("expected 202 Accepted for super admin job run, got %d: %s", rec5.Code, rec5.Body.String())
	}
}
