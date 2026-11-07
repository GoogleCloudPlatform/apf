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

package workflow

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"google.com/apf/api/agent"
	"google.com/apf/api/core"
	"google.com/apf/api/db"
)

var Store db.WorkflowStore
var RunStore db.WorkflowRunStore

func sendJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, x-goog-authenticated-user-email, x-goog-authenticated-user-id, X-Organization-Id")
	w.WriteHeader(statusCode)
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			slog.Error("Error encoding response", "error", err)
		}
	}
}

// GetWorkflowHandler handles fetching a single workflow by ID.
func GetWorkflowHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing workflow ID"})
		return
	}
	wf, err := Store.GetWorkflow(r.Context(), id)
	if err != nil {
		sendJSON(w, http.StatusNotFound, map[string]string{"error": "Workflow not found"})
		return
	}
	sendJSON(w, http.StatusOK, wf)
}

// CreateWorkflowHandler handles creating a new workflow.
func CreateWorkflowHandler(w http.ResponseWriter, r *http.Request) {
	var wf db.WorkflowTemplate
	if err := json.NewDecoder(r.Body).Decode(&wf); err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}
	if err := wf.Validate(); err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := Store.CreateWorkflow(r.Context(), &wf); err != nil {
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	sendJSON(w, http.StatusCreated, wf)
}

// UpdateWorkflowHandler handles updating an existing workflow.
func UpdateWorkflowHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing workflow ID"})
		return
	}
	var wf db.WorkflowTemplate
	if err := json.NewDecoder(r.Body).Decode(&wf); err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}
	wf.ID = id
	if err := wf.Validate(); err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := Store.CreateWorkflow(r.Context(), &wf); err != nil {
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	sendJSON(w, http.StatusOK, wf)
}

// DeleteWorkflowHandler handles deleting a workflow.
func DeleteWorkflowHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing workflow ID"})
		return
	}
	if err := Store.DeleteWorkflow(r.Context(), id); err != nil {
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	sendJSON(w, http.StatusOK, map[string]string{"message": "Workflow deleted successfully"})
}

// RunWorkflowBufferedHandler triggers the workflow execution directly.
func RunWorkflowBufferedHandler(w http.ResponseWriter, r *http.Request) {
	workflowID := r.PathValue("id")
	if workflowID == "" {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing workflow ID"})
		return
	}

	organizationID, err := core.GetOrganizationID(r.Context())
	if err != nil {
		sendJSON(w, http.StatusUnauthorized, map[string]string{"error": "Missing organization ID"})
		return
	}

	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || claims == nil {
		sendJSON(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized context"})
		return
	}
	userID := claims.Subject

	projectID := r.URL.Query().Get("projectId")

	// Log the workflow run
	run := &db.WorkflowRun{
		UserID:     userID,
		ProjectID:  projectID,
		WorkflowID: workflowID,
		StartDate:  time.Now(),
		Status:     "RUNNING",
	}
	if err := RunStore.CreateWorkflowRun(r.Context(), organizationID, run); err != nil {
		slog.Error("Failed to log workflow run", "error", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to log workflow run"})
		return
	}

	// 2. Read entry input from request body (defaults to "Start Execution")
	entryInput := "Start Execution"
	if r.Body != nil {
		bodyBytes, err := io.ReadAll(r.Body)
		if err == nil && len(bodyBytes) > 0 {
			entryInput = string(bodyBytes)
		}
	}

	sessionId := run.ID
	if sessionId == "" {
		sessionId = fmt.Sprintf("wf_run_%d", time.Now().Unix())
	}
	msg := agent.ProxyNewMessage{
		Role: "user",
		Parts: []agent.ProxyMessagePart{
			{Text: entryInput},
		},
	}

	output, err := HandleWorkflowExecutionStream(r.Context(), workflowID, sessionId, projectID, userID, organizationID, msg)
	if err != nil {
		slog.Error("Failed to execute workflow", "error", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	sendJSON(w, http.StatusOK, map[string]string{
		"status": "SUCCESS",
		"output": output,
	})
}

// HandleWorkflowExecutionStream is a shared helper to execute/resume a workflow and stream events to Firestore.
func HandleWorkflowExecutionStream(
	ctx context.Context,
	workflowID, sessionId, projectID, userID, organizationID string,
	newMessage agent.ProxyNewMessage,
) (string, error) {
	// Spawns the python workflow CLI runner directly in Go!
	stream, err := RunWorkflowDirect(ctx, workflowID, sessionId, projectID, userID, organizationID, newMessage)
	if err != nil {
		return "", err
	}
	defer stream.Close()

	var outputBuilder strings.Builder
	scanner := bufio.NewScanner(stream)
	const maxCapacity = 32 * 1024 * 1024 // 32 MB max line buffer for long-running workflow event streaming
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, maxCapacity)

	slog.Info("Streaming execution events from ADK Runner...", "workflowID", workflowID, "sessionId", sessionId)
	streamAwaitingInput := false
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// The python CLI outputs raw event JSON lines, not prefixed with "data: "
		eventData := line
		if strings.HasPrefix(line, "data: ") {
			eventData = strings.TrimPrefix(line, "data: ")
		}

		var event map[string]interface{}
		isServiceEvent := false
		if err := json.Unmarshal([]byte(eventData), &event); err == nil {
			if evType, ok := event["event"].(string); ok {
				slog.Info("Service event", "event", event)
				isServiceEvent = true
				switch evType {
				case "compiled":
					outputBuilder.WriteString(fmt.Sprintf("Compiling ADK Graph: %v...\nCompilation successful.\n", event["workflow_name"]))
				case "awaiting_input":
					streamAwaitingInput = true
					outputBuilder.WriteString("Execution paused awaiting user input.\n")
					if RunStore != nil && sessionId != "" {
						_ = RunStore.UpdateWorkflowRunStatus(ctx, organizationID, sessionId, "AWAITING_INPUT")
					}
				case "finished":
					if streamAwaitingInput {
						outputBuilder.WriteString("Execution paused awaiting user input.\n")
						if RunStore != nil && sessionId != "" {
							_ = RunStore.UpdateWorkflowRunStatus(ctx, organizationID, sessionId, "AWAITING_INPUT")
						}
					} else {
						outputBuilder.WriteString("Execution finished successfully.\n")
						if RunStore != nil && sessionId != "" {
							_ = RunStore.UpdateWorkflowRunStatus(ctx, organizationID, sessionId, "COMPLETED")
						}
					}
				case "error":
					outputBuilder.WriteString(fmt.Sprintf("Execution error: %v\n", event["message"]))
					if RunStore != nil && sessionId != "" {
						_ = RunStore.UpdateWorkflowRunStatus(ctx, organizationID, sessionId, "ERROR")
					}
					errMsg, _ := event["message"].(string)
					agent.ProcessAgentError(ctx, "Workflow Execution Error", "The workflow runner encountered an error.", errMsg, projectID, organizationID, userID, "WORKFLOW", sessionId)
				}
			}
		}

		if !isServiceEvent {
			var adkEvent agent.ADKEvent
			if err := json.Unmarshal([]byte(eventData), &adkEvent); err == nil && adkEvent.Author != "" {
				agent.ProcessAgentEvent(ctx, adkEvent, projectID, organizationID, userID, "WORKFLOW", sessionId)
				if adkEvent.Actions != nil {
					if rtc, ok := adkEvent.Actions["requested_tool_confirmations"]; ok && rtc != nil {
						streamAwaitingInput = true
					}
					if sd, ok := adkEvent.Actions["state_delta"].(map[string]interface{}); ok {
						if st, ok := sd["status"].(string); ok && strings.ToUpper(st) == "WAITING" {
							streamAwaitingInput = true
						}
					}
				}
				if adkEvent.Content != nil {
					for _, part := range adkEvent.Content.Parts {
						if _, ok := part["functionCall"]; ok {
							streamAwaitingInput = true
						}
						if _, ok := part["function_call"]; ok {
							streamAwaitingInput = true
						}
					}
				}
				outputBuilder.WriteString(fmt.Sprintf("  [Event] %s\n", eventData))
			} else {
				outputBuilder.WriteString(fmt.Sprintf("  [Event] %s\n", eventData))
			}
		}
	}

	if err := scanner.Err(); err != nil {
		slog.Error("Error reading stream from runner", "error", err)
		outputBuilder.WriteString(fmt.Sprintf("\n[Stream Error] Connection lost: %v\n", err))
	}

	return outputBuilder.String(), nil
}

type ResumeWorkflowRequest struct {
	InterruptID string                 `json:"interruptId"`
	Response    map[string]interface{} `json:"response"`
}

// ResumeWorkflowHandler handles resuming a paused workflow run with user input.
func ResumeWorkflowHandler(w http.ResponseWriter, r *http.Request) {
	workflowID := r.PathValue("id")
	runID := r.PathValue("runId")
	if workflowID == "" || runID == "" {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing workflow ID or run ID"})
		return
	}

	organizationID, err := core.GetOrganizationID(r.Context())
	if err != nil {
		sendJSON(w, http.StatusUnauthorized, map[string]string{"error": "Missing organization ID"})
		return
	}

	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || claims == nil {
		sendJSON(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized context"})
		return
	}
	userID := claims.Subject
	projectID := r.URL.Query().Get("projectId")

	var req ResumeWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	if req.InterruptID == "" {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing interrupt ID"})
		return
	}

	// Create and save the FunctionResponse message to Firestore so the UI detects resolution
	userMsg := &db.Message{
		Sender:         "user",
		Text:           "",
		IsUser:         true,
		IsTransient:    false,
		IsNotification: false,
		SessionId:      runID,
		Timestamp:      time.Now(),
		FunctionResponse: &db.FunctionResponse{
			Name:     "adk_request_input",
			ID:       req.InterruptID,
			Response: req.Response,
		},
	}

	if _, err := agent.Store.CreateMessage(r.Context(), organizationID, projectID, userID, userMsg); err != nil {
		slog.Error("Failed to save user response message to Firestore", "error", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to save response: " + err.Error()})
		return
	}

	if RunStore != nil && runID != "" {
		_ = RunStore.UpdateWorkflowRunStatus(r.Context(), organizationID, runID, "RUNNING")
	}

	// Construct the FunctionResponse message
	msg := agent.ProxyNewMessage{
		Role: "user",
		Parts: []agent.ProxyMessagePart{
			{
				FunctionResponse: &db.FunctionResponse{
					Name:     "adk_request_input",
					ID:       req.InterruptID,
					Response: req.Response,
				},
			},
		},
	}

	go func() {
		bgCtx := context.Background()
		if _, err := HandleWorkflowExecutionStream(bgCtx, workflowID, runID, projectID, userID, organizationID, msg); err != nil {
			slog.Error("Failed to process background workflow resume stream", "workflowID", workflowID, "runID", runID, "error", err)
		}
	}()

	sendJSON(w, http.StatusAccepted, map[string]string{
		"status":  "ACCEPTED",
		"message": "Workflow resume initiated in background",
	})
}

// ListWorkflowRunsHandler lists all workflow runs for a project.
func ListWorkflowRunsHandler(w http.ResponseWriter, r *http.Request) {
	projectId := r.PathValue("id")
	if projectId == "" {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing project ID"})
		return
	}

	organizationID, err := core.GetOrganizationID(r.Context())
	if err != nil {
		sendJSON(w, http.StatusUnauthorized, map[string]string{"error": "Missing organization ID"})
		return
	}

	allRuns, err := RunStore.ListWorkflowRuns(r.Context(), organizationID)
	if err != nil {
		slog.Error("Failed to list workflow runs", "error", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to list workflow runs"})
		return
	}

	// Fetch all workflows to map IDs to names
	workflows, err := Store.ListWorkflows(r.Context())
	workflowMap := make(map[string]string)
	if err == nil {
		for _, wf := range workflows {
			name := wf.DisplayName
			if name == "" {
				name = wf.Name
			}
			workflowMap[wf.ID] = name
		}
	} else {
		slog.Warn("Failed to list workflows for mapping, will show IDs instead", "error", err)
	}

	type WorkflowRunResponse struct {
		db.WorkflowRun
		WorkflowName string `json:"workflowName"`
	}

	// Filter by projectId and build response
	var projectRuns []WorkflowRunResponse
	for _, run := range allRuns {
		if run.ProjectID == projectId {
			wfName := workflowMap[run.WorkflowID]
			if wfName == "" {
				wfName = run.WorkflowID // fallback to ID
			}
			awaiting := run.Status == "AWAITING_INPUT" || run.AwaitingInput
			if !awaiting && run.Status != "COMPLETED" && run.Status != "ERROR" {
				msgs, err := agent.Store.ListMessages(r.Context(), organizationID, projectId, "", run.ID, true)
				if err == nil && len(msgs) > 0 {
					resolvedIDs := make(map[string]bool)
					for _, m := range msgs {
						if m.FunctionResponse != nil && m.FunctionResponse.ID != "" {
							resolvedIDs[m.FunctionResponse.ID] = true
						}
					}
					for _, m := range msgs {
						if m.AwaitingInput {
							interruptID := ""
							if m.Event != nil && m.Event.Content != nil && len(m.Event.Content.Parts) > 0 {
								if fc, ok := m.Event.Content.Parts[0][agent.KeyFunctionCall].(map[string]interface{}); ok {
									if id, ok := fc["id"].(string); ok {
										interruptID = id
									}
								}
							}
							if interruptID != "" && !resolvedIDs[interruptID] {
								awaiting = true
								run.Status = "AWAITING_INPUT"
								break
							}
						}
					}
				}
			}
			run.AwaitingInput = awaiting
			if run.Status == "" {
				if awaiting {
					run.Status = "AWAITING_INPUT"
				} else {
					run.Status = "COMPLETED"
				}
			}
			projectRuns = append(projectRuns, WorkflowRunResponse{
				WorkflowRun:  run,
				WorkflowName: wfName,
			})
		}
	}

	sendJSON(w, http.StatusOK, projectRuns)
}

// ListWorkflowsHandler lists all workflows.
func ListWorkflowsHandler(w http.ResponseWriter, r *http.Request) {
	_, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok {
		http.Error(w, "Unauthorized context", http.StatusUnauthorized)
		return
	}

	workflows, err := Store.ListWorkflows(r.Context())
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to list workflows: %v", err)})
		return
	}

	workflowType := r.URL.Query().Get("type")
	filteredWorkflows := make([]db.WorkflowTemplate, 0, len(workflows))
	for _, wf := range workflows {
		if workflowType != "" {
			if strings.EqualFold(wf.Framework, workflowType) || strings.HasPrefix(strings.ToUpper(wf.Framework), strings.ToUpper(workflowType)) {
				filteredWorkflows = append(filteredWorkflows, wf)
			}
		} else {
			filteredWorkflows = append(filteredWorkflows, wf)
		}
	}

	sendJSON(w, http.StatusOK, filteredWorkflows)
}

// DeleteWorkflowRunHandler deletes a specific workflow run for a project.
func DeleteWorkflowRunHandler(w http.ResponseWriter, r *http.Request) {
	projectId := r.PathValue("id")
	runId := r.PathValue("runId")
	if projectId == "" || runId == "" {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing project ID or run ID"})
		return
	}

	organizationID, err := core.GetOrganizationID(r.Context())
	if err != nil {
		sendJSON(w, http.StatusUnauthorized, map[string]string{"error": "Missing organization ID"})
		return
	}

	run, err := RunStore.GetWorkflowRun(r.Context(), organizationID, runId)
	if err != nil {
		sendJSON(w, http.StatusNotFound, map[string]string{"error": "Workflow run not found"})
		return
	}
	if run.ProjectID != projectId {
		sendJSON(w, http.StatusForbidden, map[string]string{"error": "Workflow run does not belong to this project"})
		return
	}

	if err := RunStore.DeleteWorkflowRun(r.Context(), organizationID, runId); err != nil {
		slog.Error("Failed to delete workflow run", "error", err, "runId", runId)
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to delete workflow run"})
		return
	}

	sendJSON(w, http.StatusOK, map[string]string{"message": "Workflow run deleted successfully"})
}

// DeleteWorkflowRunByWorkflowHandler deletes a specific workflow run scoped to a workflow ID.
func DeleteWorkflowRunByWorkflowHandler(w http.ResponseWriter, r *http.Request) {
	workflowId := r.PathValue("id")
	runId := r.PathValue("runId")
	if workflowId == "" || runId == "" {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing workflow ID or run ID"})
		return
	}

	organizationID, err := core.GetOrganizationID(r.Context())
	if err != nil {
		sendJSON(w, http.StatusUnauthorized, map[string]string{"error": "Missing organization ID"})
		return
	}

	run, err := RunStore.GetWorkflowRun(r.Context(), organizationID, runId)
	if err != nil {
		sendJSON(w, http.StatusNotFound, map[string]string{"error": "Workflow run not found"})
		return
	}
	if run.WorkflowID != workflowId {
		sendJSON(w, http.StatusForbidden, map[string]string{"error": "Workflow run does not belong to this workflow"})
		return
	}

	if err := RunStore.DeleteWorkflowRun(r.Context(), organizationID, runId); err != nil {
		slog.Error("Failed to delete workflow run", "error", err, "runId", runId)
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to delete workflow run"})
		return
	}

	sendJSON(w, http.StatusOK, map[string]string{"message": "Workflow run deleted successfully"})
}
