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
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	run "google.golang.org/api/run/v2"

	"google.com/apf/api/core"
	"google.com/apf/api/jobs"
)

// JobRunRequest holds optional parameters for triggering an on-demand job execution.
type JobRunRequest struct {
	DryRun bool `json:"dryRun"`
}

// JobRunResponse represents the result of triggering an on-demand job execution.
type JobRunResponse struct {
	Status        string `json:"status"`
	ExecutionName string `json:"executionName"`
	JobName       string `json:"jobName"`
	Message       string `json:"message,omitempty"`
}

// JobParameterInfo describes a configurable parameter for a housekeeping job.
type JobParameterInfo struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	Description  string `json:"description"`
	Required     bool   `json:"required"`
	DefaultValue any    `json:"defaultValue,omitempty"`
}

// JobCatalogItem describes a registered housekeeping job and its parameters.
type JobCatalogItem struct {
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Parameters  []JobParameterInfo `json:"parameters"`
}

// JobExecutionItem represents a structured execution record parsed from Cloud Run v2.
type JobExecutionItem struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	JobName           string `json:"jobName"`
	DryRun            bool   `json:"dryRun"`
	Status            string `json:"status"`
	CreateTime        string `json:"createTime,omitempty"`
	StartTime         string `json:"startTime,omitempty"`
	CompletionTime    string `json:"completionTime,omitempty"`
	DurationSeconds   int64  `json:"durationSeconds"`
	DurationFormatted string `json:"durationFormatted"`
	ErrorMessage      string `json:"errorMessage,omitempty"`
	LogURI            string `json:"logUri,omitempty"`
}

// JobExecutionsListResponse represents the paginated response containing job executions.
type JobExecutionsListResponse struct {
	Executions    []*JobExecutionItem `json:"executions"`
	NextPageToken string              `json:"nextPageToken,omitempty"`
}

// CloudRunJobRunner defines the interface for triggering and inspecting Cloud Run jobs.
type CloudRunJobRunner interface {
	RunJob(ctx context.Context, jobName string, dryRun bool) (*JobRunResponse, error)
	ListExecutions(ctx context.Context, pageSize int, pageToken string) (*JobExecutionsListResponse, error)
}

// GCPCloudRunJobRunner implements CloudRunJobRunner using the Google Cloud Run v2 SDK.
type GCPCloudRunJobRunner struct {
	Project  string
	Location string
	JobName  string
	Service  *run.Service
}

// NewGCPCloudRunJobRunner creates a new GCP Cloud Run job runner.
func NewGCPCloudRunJobRunner(ctx context.Context, cfg *core.Configuration) (*GCPCloudRunJobRunner, error) {
	svc, err := run.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create Cloud Run v2 service: %w", err)
	}

	project := "agenticproductfoundry-489717"
	location := "us-central1"
	if cfg != nil {
		if cfg.GoogleCloudProject != "" {
			project = cfg.GoogleCloudProject
		}
		if cfg.GoogleCloudLocation != "" {
			location = cfg.GoogleCloudLocation
		}
	}

	jobName := os.Getenv("MAINTENANCE_JOB_NAME")
	if jobName == "" {
		if cfg != nil && (cfg.Environment == "production" || cfg.Environment == "prod") {
			jobName = "apf-prod-jobs"
		} else {
			jobName = "apf-stg-jobs"
		}
	}

	return &GCPCloudRunJobRunner{
		Project:  project,
		Location: location,
		JobName:  jobName,
		Service:  svc,
	}, nil
}

// RunJob triggers a Cloud Run job execution with optional container override arguments.
func (r *GCPCloudRunJobRunner) RunJob(ctx context.Context, jobName string, dryRun bool) (*JobRunResponse, error) {
	if r.Service == nil {
		return nil, errors.New("cloud run service client is not initialized")
	}

	var args []string
	if dryRun {
		args = []string{jobName, "--dry-run"}
	} else {
		args = []string{jobName}
	}

	req := &run.GoogleCloudRunV2RunJobRequest{
		Overrides: &run.GoogleCloudRunV2Overrides{
			ContainerOverrides: []*run.GoogleCloudRunV2ContainerOverride{
				{
					Args: args,
				},
			},
		},
	}

	parent := fmt.Sprintf("projects/%s/locations/%s/jobs/%s", r.Project, r.Location, r.JobName)
	op, err := r.Service.Projects.Locations.Jobs.Run(parent, req).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to run cloud run job: %w", err)
	}

	executionName := op.Name
	if len(op.Response) > 0 {
		var exec run.GoogleCloudRunV2Execution
		if err := json.Unmarshal(op.Response, &exec); err == nil && exec.Name != "" {
			executionName = exec.Name
		}
	}

	return &JobRunResponse{
		Status:        "PENDING",
		ExecutionName: executionName,
		JobName:       jobName,
	}, nil
}

// ListExecutions retrieves and parses executions from Cloud Run v2 API.
func (r *GCPCloudRunJobRunner) ListExecutions(ctx context.Context, pageSize int, pageToken string) (*JobExecutionsListResponse, error) {
	if r.Service == nil {
		return nil, errors.New("cloud run service client is not initialized")
	}

	parent := fmt.Sprintf("projects/%s/locations/%s/jobs/%s", r.Project, r.Location, r.JobName)
	call := r.Service.Projects.Locations.Jobs.Executions.List(parent).Context(ctx)
	if pageSize > 0 {
		call = call.PageSize(int64(pageSize))
	}
	if pageToken != "" {
		call = call.PageToken(pageToken)
	}

	resp, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list cloud run executions: %w", err)
	}

	executions := make([]*JobExecutionItem, 0, len(resp.Executions))
	for _, exec := range resp.Executions {
		executions = append(executions, parseCloudRunExecution(exec, r.Project, r.Location, r.JobName))
	}

	return &JobExecutionsListResponse{
		Executions:    executions,
		NextPageToken: resp.NextPageToken,
	}, nil
}

// formatDuration formats a duration into a human-readable string (e.g. "1h 2m 3s", "1m 14s", "42s", "0s").
func formatDuration(d time.Duration) string {
	if d <= 0 {
		return "0s"
	}
	totalSec := int64(d.Seconds())
	h := totalSec / 3600
	m := (totalSec % 3600) / 60
	s := totalSec % 60

	if h > 0 {
		if m > 0 && s > 0 {
			return fmt.Sprintf("%dh %dm %ds", h, m, s)
		} else if m > 0 {
			return fmt.Sprintf("%dh %dm", h, m)
		} else if s > 0 {
			return fmt.Sprintf("%dh %ds", h, s)
		}
		return fmt.Sprintf("%dh", h)
	}
	if m > 0 {
		if s > 0 {
			return fmt.Sprintf("%dm %ds", m, s)
		}
		return fmt.Sprintf("%dm", m)
	}
	return fmt.Sprintf("%ds", s)
}

// parseCloudRunExecution parses a Cloud Run v2 Execution into a structured JobExecutionItem.
func parseCloudRunExecution(exec *run.GoogleCloudRunV2Execution, project, location, defaultJobName string) *JobExecutionItem {
	if exec == nil {
		return &JobExecutionItem{
			Status:            "UNKNOWN",
			DurationFormatted: "0s",
		}
	}

	id := exec.Name
	if idx := strings.LastIndex(exec.Name, "/"); idx != -1 && idx < len(exec.Name)-1 {
		id = exec.Name[idx+1:]
	} else if exec.Uid != "" {
		id = exec.Uid
	}

	jobName := ""
	dryRun := false
	if exec.Template != nil && len(exec.Template.Containers) > 0 {
		container := exec.Template.Containers[0]
		if container != nil {
			if len(container.Args) > 0 {
				jobName = container.Args[0]
			}
			for _, arg := range container.Args {
				if arg == "--dry-run" || arg == "-dry-run" || arg == "--dryRun" {
					dryRun = true
				}
			}
		}
	}

	if jobName == "" {
		if defaultJobName != "" {
			jobName = defaultJobName
		} else {
			jobName = "unknown"
		}
	}

	// Status determination
	status := "RUNNING"
	if exec.CancelledCount > 0 {
		status = "CANCELLED"
	} else if exec.FailedCount > 0 {
		status = "FAILED"
	} else if exec.SucceededCount > 0 && exec.RunningCount == 0 {
		status = "SUCCEEDED"
	} else if exec.RunningCount > 0 || exec.Reconciling {
		status = "RUNNING"
	}

	var errorMessages []string
	for _, cond := range exec.Conditions {
		if cond == nil {
			continue
		}
		condType := strings.ToLower(cond.Type)
		condReason := strings.ToLower(cond.Reason)

		if strings.Contains(condType, "cancel") || strings.Contains(condReason, "cancel") {
			status = "CANCELLED"
		} else if condType == "completed" || condType == "ready" {
			if cond.State == "CONDITION_SUCCEEDED" || cond.State == "STATE_TRUE" || cond.State == "True" {
				status = "SUCCEEDED"
			} else if cond.State == "CONDITION_FAILED" || cond.State == "STATE_FALSE" || cond.State == "False" {
				status = "FAILED"
			}
		}

		if cond.State == "CONDITION_FAILED" || cond.State == "STATE_FALSE" || cond.State == "False" || cond.Severity == "ERROR" {
			if cond.Message != "" {
				errorMessages = append(errorMessages, cond.Message)
			} else if cond.Reason != "" {
				errorMessages = append(errorMessages, cond.Reason)
			}
		}
	}

	if status == "RUNNING" && exec.CompletionTime != "" {
		if exec.FailedCount > 0 || len(errorMessages) > 0 {
			status = "FAILED"
		} else {
			status = "SUCCEEDED"
		}
	}

	// Duration calculation
	var durationSeconds int64
	durationFormatted := "0s"
	if exec.StartTime != "" {
		startTime, err1 := time.Parse(time.RFC3339Nano, exec.StartTime)
		if err1 != nil {
			startTime, err1 = time.Parse(time.RFC3339, exec.StartTime)
		}
		if err1 == nil {
			var dur time.Duration
			if exec.CompletionTime != "" {
				completionTime, err2 := time.Parse(time.RFC3339Nano, exec.CompletionTime)
				if err2 != nil {
					completionTime, err2 = time.Parse(time.RFC3339, exec.CompletionTime)
				}
				if err2 == nil {
					dur = completionTime.Sub(startTime)
				}
			} else {
				dur = time.Since(startTime)
			}
			if dur > 0 {
				durationSeconds = int64(dur.Seconds())
				durationFormatted = formatDuration(dur)
			}
		}
	}

	logURI := exec.LogUri
	if logURI == "" {
		queryFilter := fmt.Sprintf("resource.type=\"cloud_run_job\" labels.\"run.googleapis.com/execution_name\"=\"%s\"", id)
		logURI = fmt.Sprintf("https://console.cloud.google.com/logs/query;query=%s?project=%s", url.QueryEscape(queryFilter), url.QueryEscape(project))
	}

	var errorMessage string
	if len(errorMessages) > 0 {
		errorMessage = strings.Join(errorMessages, "; ")
	}

	return &JobExecutionItem{
		ID:                id,
		Name:              exec.Name,
		JobName:           jobName,
		DryRun:            dryRun,
		Status:            status,
		CreateTime:        exec.CreateTime,
		StartTime:         exec.StartTime,
		CompletionTime:    exec.CompletionTime,
		DurationSeconds:   durationSeconds,
		DurationFormatted: durationFormatted,
		ErrorMessage:      errorMessage,
		LogURI:            logURI,
	}
}

// SimulatedCloudRunJobRunner provides a simulated or mock fallback implementation of CloudRunJobRunner.
type SimulatedCloudRunJobRunner struct {
	Project  string
	Location string
	JobName  string
}

// NewSimulatedCloudRunJobRunner creates a new SimulatedCloudRunJobRunner.
func NewSimulatedCloudRunJobRunner(cfg *core.Configuration) *SimulatedCloudRunJobRunner {
	project := "agenticproductfoundry-489717"
	location := "us-central1"
	if cfg != nil {
		if cfg.GoogleCloudProject != "" {
			project = cfg.GoogleCloudProject
		}
		if cfg.GoogleCloudLocation != "" {
			location = cfg.GoogleCloudLocation
		}
	}

	jobName := os.Getenv("MAINTENANCE_JOB_NAME")
	if jobName == "" {
		if cfg != nil && (cfg.Environment == "production" || cfg.Environment == "prod") {
			jobName = "apf-prod-jobs"
		} else {
			jobName = "apf-stg-jobs"
		}
	}

	return &SimulatedCloudRunJobRunner{
		Project:  project,
		Location: location,
		JobName:  jobName,
	}
}

// RunJob generates a simulated execution response for local testing.
func (s *SimulatedCloudRunJobRunner) RunJob(ctx context.Context, jobName string, dryRun bool) (*JobRunResponse, error) {
	execID := fmt.Sprintf("simulated-%s-%d", jobName, time.Now().UnixNano())
	executionName := fmt.Sprintf("projects/%s/locations/%s/jobs/%s/executions/%s", s.Project, s.Location, s.JobName, execID)
	return &JobRunResponse{
		Status:        "PENDING",
		ExecutionName: executionName,
		JobName:       jobName,
	}, nil
}

// ListExecutions returns simulated execution records for local testing and dev mode.
func (s *SimulatedCloudRunJobRunner) ListExecutions(ctx context.Context, pageSize int, pageToken string) (*JobExecutionsListResponse, error) {
	now := time.Now().UTC()
	t1Start := now.Add(-10 * time.Minute)
	t1End := t1Start.Add(74 * time.Second)
	t2Start := now.Add(-1 * time.Hour)
	t2End := t2Start.Add(42 * time.Second)
	t3Start := now.Add(-2 * time.Minute)

	simulatedExecutions := []*JobExecutionItem{
		{
			ID:                "simulated-nfs-audit-1",
			Name:              fmt.Sprintf("projects/%s/locations/%s/jobs/%s/executions/simulated-nfs-audit-1", s.Project, s.Location, s.JobName),
			JobName:           "nfs-audit",
			DryRun:            true,
			Status:            "SUCCEEDED",
			CreateTime:        t1Start.Format(time.RFC3339),
			StartTime:         t1Start.Format(time.RFC3339),
			CompletionTime:    t1End.Format(time.RFC3339),
			DurationSeconds:   74,
			DurationFormatted: "1m 14s",
			LogURI:            fmt.Sprintf("https://console.cloud.google.com/logs/query;query=resource.type%%3D%%22cloud_run_job%%22?project=%s", s.Project),
		},
		{
			ID:                "simulated-hanging-cleanup-2",
			Name:              fmt.Sprintf("projects/%s/locations/%s/jobs/%s/executions/simulated-hanging-cleanup-2", s.Project, s.Location, s.JobName),
			JobName:           "hanging-cleanup",
			DryRun:            false,
			Status:            "FAILED",
			CreateTime:        t2Start.Format(time.RFC3339),
			StartTime:         t2Start.Format(time.RFC3339),
			CompletionTime:    t2End.Format(time.RFC3339),
			DurationSeconds:   42,
			DurationFormatted: "42s",
			ErrorMessage:      "PermissionDenied: unable to delete directory /mnt/nfs/hanging-workspace",
			LogURI:            fmt.Sprintf("https://console.cloud.google.com/logs/query;query=resource.type%%3D%%22cloud_run_job%%22?project=%s", s.Project),
		},
		{
			ID:                "simulated-global-provisioning-3",
			Name:              fmt.Sprintf("projects/%s/locations/%s/jobs/%s/executions/simulated-global-provisioning-3", s.Project, s.Location, s.JobName),
			JobName:           "global-workspace-provisioning",
			DryRun:            false,
			Status:            "RUNNING",
			CreateTime:        t3Start.Format(time.RFC3339),
			StartTime:         t3Start.Format(time.RFC3339),
			DurationSeconds:   int64(now.Sub(t3Start).Seconds()),
			DurationFormatted: formatDuration(now.Sub(t3Start)),
			LogURI:            fmt.Sprintf("https://console.cloud.google.com/logs/query;query=resource.type%%3D%%22cloud_run_job%%22?project=%s", s.Project),
		},
	}

	if pageToken != "" {
		if pageToken == "simulated-next-token" && pageSize > 0 && pageSize < len(simulatedExecutions) {
			return &JobExecutionsListResponse{
				Executions: simulatedExecutions[pageSize:],
			}, nil
		}
		return &JobExecutionsListResponse{
			Executions: []*JobExecutionItem{},
		}, nil
	}

	if pageSize > 0 && pageSize < len(simulatedExecutions) {
		return &JobExecutionsListResponse{
			Executions:    simulatedExecutions[:pageSize],
			NextPageToken: "simulated-next-token",
		}, nil
	}

	return &JobExecutionsListResponse{
		Executions: simulatedExecutions,
	}, nil
}

// ListJobCatalog handles GET /api/v1/admin/jobs
func (h *AdminHandler) ListJobCatalog(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || claims == nil || !claims.IsSuperAdmin {
		sendJSON(w, http.StatusForbidden, Response{
			Message: "Forbidden: Super Administrator access required",
			Status:  "ERROR",
		})
		return
	}

	allJobs := jobs.List()
	catalog := make([]JobCatalogItem, 0, len(allJobs))

	for _, j := range allJobs {
		item := JobCatalogItem{
			Name:        j.Name(),
			Description: j.Description(),
			Parameters: []JobParameterInfo{
				{
					Name:         "dryRun",
					Type:         "boolean",
					Description:  "Runs the job in simulation mode without committing destructive changes",
					Required:     false,
					DefaultValue: false,
				},
			},
		}
		catalog = append(catalog, item)
	}

	sendJSON(w, http.StatusOK, catalog)
}

// ListJobExecutions handles GET /api/v1/admin/jobs/executions
func (h *AdminHandler) ListJobExecutions(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || claims == nil || !claims.IsSuperAdmin {
		sendJSON(w, http.StatusForbidden, Response{
			Message: "Forbidden: Super Administrator access required",
			Status:  "ERROR",
		})
		return
	}

	var pageSize int
	if ps := strings.TrimSpace(r.URL.Query().Get("pageSize")); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 {
			pageSize = parsed
			if pageSize > 100 {
				pageSize = 100
			}
		}
	}
	pageToken := strings.TrimSpace(r.URL.Query().Get("pageToken"))

	runner := h.JobRunner
	if runner == nil {
		runner = NewSimulatedCloudRunJobRunner(h.Config)
	}

	resp, err := runner.ListExecutions(r.Context(), pageSize, pageToken)
	if err != nil {
		slog.Error("Failed to list Cloud Run job executions", "error", err)
		sendJSON(w, http.StatusInternalServerError, Response{
			Message: fmt.Sprintf("Failed to list job executions: %v", err),
			Status:  "ERROR",
		})
		return
	}

	sendJSON(w, http.StatusOK, resp)
}

// RunCloudRunJob handles POST /api/v1/admin/jobs/{jobName}/run
func (h *AdminHandler) RunCloudRunJob(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || claims == nil || !claims.IsSuperAdmin {
		sendJSON(w, http.StatusForbidden, Response{
			Message: "Forbidden: Super Administrator access required",
			Status:  "ERROR",
		})
		return
	}

	jobName := strings.TrimSpace(r.PathValue("jobName"))
	if jobName == "" {
		sendJSON(w, http.StatusBadRequest, Response{
			Message: "Job name is required",
			Status:  "ERROR",
		})
		return
	}

	var req JobRunRequest
	if r.Body != nil && r.Body != http.NoBody && r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			sendJSON(w, http.StatusBadRequest, Response{
				Message: "Invalid request payload",
				Status:  "ERROR",
			})
			return
		}
	}

	runner := h.JobRunner
	if runner == nil {
		runner = NewSimulatedCloudRunJobRunner(h.Config)
	}

	resp, err := runner.RunJob(r.Context(), jobName, req.DryRun)
	if err != nil {
		slog.Error("Failed to trigger Cloud Run job execution", "jobName", jobName, "error", err)
		sendJSON(w, http.StatusInternalServerError, Response{
			Message: fmt.Sprintf("Failed to trigger job execution: %v", err),
			Status:  "ERROR",
		})
		return
	}

	sendJSON(w, http.StatusAccepted, resp)
}
