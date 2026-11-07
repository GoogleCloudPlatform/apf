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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.com/apf/api/core"
	"google.com/apf/api/db"
	"google.com/apf/api/integrations"
)

type mockWorkflowStore struct {
	db.WorkflowStore
	workflows map[string]db.WorkflowTemplate
}

func (m *mockWorkflowStore) GetWorkflow(ctx context.Context, id string) (*db.WorkflowTemplate, error) {
	w, ok := m.workflows[id]
	if !ok {
		return nil, fmt.Errorf("workflow not found")
	}
	return &w, nil
}

func (m *mockWorkflowStore) CreateWorkflow(ctx context.Context, w *db.WorkflowTemplate) error {
	if w.ID == "" {
		w.ID = fmt.Sprintf("wf-%d", len(m.workflows)+1)
	}
	m.workflows[w.ID] = *w
	return nil
}

func (m *mockWorkflowStore) DeleteWorkflow(ctx context.Context, id string) error {
	if _, ok := m.workflows[id]; !ok {
		return fmt.Errorf("workflow not found")
	}
	delete(m.workflows, id)
	return nil
}

func (m *mockWorkflowStore) ListWorkflows(ctx context.Context) ([]db.WorkflowTemplate, error) {
	var list []db.WorkflowTemplate
	for _, w := range m.workflows {
		list = append(list, w)
	}
	return list, nil
}

type mockIntegrationStore struct {
	db.IntegrationStore
	integrations map[string]*db.Integration
}

func (m *mockIntegrationStore) GetIntegration(ctx context.Context, userId, name string) (*db.Integration, error) {
	if m.integrations == nil {
		return nil, fmt.Errorf("not found")
	}
	docID := fmt.Sprintf("%s_%s", userId, name)
	i, ok := m.integrations[docID]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return i, nil
}

func (m *mockIntegrationStore) SaveIntegration(ctx context.Context, integration *db.Integration) error {
	if m.integrations == nil {
		m.integrations = make(map[string]*db.Integration)
	}
	docID := fmt.Sprintf("%s_%s", integration.UserID, integration.Name)
	m.integrations[docID] = integration
	return nil
}

func TestGetWorkflowHandler(t *testing.T) {
	mockStore := &mockWorkflowStore{workflows: make(map[string]db.WorkflowTemplate)}
	Store = mockStore

	// Seed mock workflow
	mockWf := db.WorkflowTemplate{
		ID:        "wf-123",
		Name:      "Test ADK Flow",
		Framework: "ADK",
		Nodes: []db.AdkNode{
			{BaseNodeFields: db.BaseNodeFields{ID: "start", Name: "START", Type: "start"}},
			{
				BaseNodeFields: db.BaseNodeFields{ID: "agent-1", Name: "agent_pm", Type: "agent"},
				AgentFields:    db.AgentFields{Model: "gemini-2.5-flash"},
			},
		},
		Edges: []db.AdkEdge{
			{ID: "edge-1", FromNode: "start", ToNode: "agent-1"},
		},
	}
	mockStore.workflows[mockWf.ID] = mockWf

	t.Run("Success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows/wf-123", nil)
		req.SetPathValue("id", "wf-123")
		w := httptest.NewRecorder()

		GetWorkflowHandler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200 OK, got %d", w.Code)
		}

		var fetched db.WorkflowTemplate
		if err := json.NewDecoder(w.Body).Decode(&fetched); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if fetched.Name != "Test ADK Flow" || len(fetched.Nodes) != 2 {
			t.Errorf("Fetched workflow content mismatch: %+v", fetched)
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows/non-existent", nil)
		req.SetPathValue("id", "non-existent")
		w := httptest.NewRecorder()

		GetWorkflowHandler(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404 Not Found, got %d", w.Code)
		}
	})
}

func TestCreateWorkflowHandler(t *testing.T) {
	mockStore := &mockWorkflowStore{workflows: make(map[string]db.WorkflowTemplate)}
	Store = mockStore

	t.Run("Success", func(t *testing.T) {
		newWf := db.WorkflowTemplate{
			Name:      "Created Flow",
			Framework: "ADK",
			Nodes: []db.AdkNode{
				{BaseNodeFields: db.BaseNodeFields{ID: "start", Name: "START", Type: "start"}},
			},
		}
		body, _ := json.Marshal(newWf)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows", bytes.NewReader(body))
		w := httptest.NewRecorder()

		CreateWorkflowHandler(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("Expected status 201 Created, got %d. Body: %s", w.Code, w.Body.String())
		}

		var created db.WorkflowTemplate
		if err := json.NewDecoder(w.Body).Decode(&created); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if created.ID == "" || created.Name != "Created Flow" {
			t.Errorf("Created workflow invalid: %+v", created)
		}

		// Verify saved in store
		if _, ok := mockStore.workflows[created.ID]; !ok {
			t.Error("Workflow was not saved to the store")
		}
	})

	t.Run("MissingName", func(t *testing.T) {
		invalidWf := db.WorkflowTemplate{
			Framework: "ADK",
		}
		body, _ := json.Marshal(invalidWf)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows", bytes.NewReader(body))
		w := httptest.NewRecorder()

		CreateWorkflowHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 Bad Request, got %d", w.Code)
		}
	})
}

func TestUpdateWorkflowHandler(t *testing.T) {
	mockStore := &mockWorkflowStore{workflows: make(map[string]db.WorkflowTemplate)}
	Store = mockStore

	// Seed mock workflow
	mockWf := db.WorkflowTemplate{
		ID:        "wf-1",
		Name:      "Original Flow",
		Framework: "ADK",
	}
	mockStore.workflows[mockWf.ID] = mockWf

	t.Run("Success", func(t *testing.T) {
		updatedWf := db.WorkflowTemplate{
			Name:      "Updated Flow",
			Framework: "ADK",
			Nodes: []db.AdkNode{
				{BaseNodeFields: db.BaseNodeFields{ID: "start", Name: "START", Type: "start"}},
			},
		}
		body, _ := json.Marshal(updatedWf)
		req := httptest.NewRequest(http.MethodPut, "/api/v1/workflows/wf-1", bytes.NewReader(body))
		req.SetPathValue("id", "wf-1")
		w := httptest.NewRecorder()

		UpdateWorkflowHandler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200 OK, got %d", w.Code)
		}

		var fetched db.WorkflowTemplate
		if err := json.NewDecoder(w.Body).Decode(&fetched); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if fetched.Name != "Updated Flow" || fetched.ID != "wf-1" {
			t.Errorf("Updated workflow content mismatch: %+v", fetched)
		}

		// Verify saved in store
		saved := mockStore.workflows["wf-1"]
		if saved.Name != "Updated Flow" || len(saved.Nodes) != 1 {
			t.Error("Workflow updates were not saved to store")
		}
	})
}

func TestDeleteWorkflowHandler(t *testing.T) {
	mockStore := &mockWorkflowStore{workflows: make(map[string]db.WorkflowTemplate)}
	Store = mockStore

	// Seed mock workflow
	mockWf := db.WorkflowTemplate{
		ID:   "wf-delete-me",
		Name: "Trash Flow",
	}
	mockStore.workflows[mockWf.ID] = mockWf

	t.Run("Success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/workflows/wf-delete-me", nil)
		req.SetPathValue("id", "wf-delete-me")
		w := httptest.NewRecorder()

		DeleteWorkflowHandler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200 OK, got %d", w.Code)
		}

		if _, ok := mockStore.workflows["wf-delete-me"]; ok {
			t.Error("Workflow was not removed from the store")
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/workflows/non-existent", nil)
		req.SetPathValue("id", "non-existent")
		w := httptest.NewRecorder()

		DeleteWorkflowHandler(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500 Internal Server Error, got %d", w.Code)
		}
	})
}

func TestWorkflowValidation(t *testing.T) {
	t.Run("ValidADKFlow", func(t *testing.T) {
		wf := db.WorkflowTemplate{
			Name:      "Valid Flow",
			Framework: "ADK",
			Nodes: []db.AdkNode{
				{BaseNodeFields: db.BaseNodeFields{ID: "start", Name: "START", Type: "start"}},
				{BaseNodeFields: db.BaseNodeFields{ID: "node1", Name: "agent_pm", Type: "agent"}},
				{BaseNodeFields: db.BaseNodeFields{ID: "node2", Name: "_helper_func", Type: "function"}},
			},
		}
		if err := wf.Validate(); err != nil {
			t.Errorf("Expected valid workflow to pass validation, got: %v", err)
		}
	})

	t.Run("InvalidPythonIdentifierSpace", func(t *testing.T) {
		wf := db.WorkflowTemplate{
			Name:      "Invalid Flow",
			Framework: "ADK",
			Nodes: []db.AdkNode{
				{BaseNodeFields: db.BaseNodeFields{ID: "node1", Name: "agent pm", Type: "agent"}},
			},
		}
		if err := wf.Validate(); err == nil {
			t.Error("Expected validation error for spaces in node name, got nil")
		}
	})

	t.Run("InvalidPythonIdentifierDash", func(t *testing.T) {
		wf := db.WorkflowTemplate{
			Name:      "Invalid Flow",
			Framework: "ADK",
			Nodes: []db.AdkNode{
				{BaseNodeFields: db.BaseNodeFields{ID: "node1", Name: "agent-pm", Type: "agent"}},
			},
		}
		if err := wf.Validate(); err == nil {
			t.Error("Expected validation error for dashes in node name, got nil")
		}
	})

	t.Run("InvalidPythonIdentifierStartDigit", func(t *testing.T) {
		wf := db.WorkflowTemplate{
			Name:      "Invalid Flow",
			Framework: "ADK",
			Nodes: []db.AdkNode{
				{BaseNodeFields: db.BaseNodeFields{ID: "node1", Name: "1agent", Type: "agent"}},
			},
		}
		if err := wf.Validate(); err == nil {
			t.Error("Expected validation error for node name starting with a digit, got nil")
		}
	})

	t.Run("InvalidPythonIdentifierKeyword", func(t *testing.T) {
		wf := db.WorkflowTemplate{
			Name:      "Invalid Flow",
			Framework: "ADK",
			Nodes: []db.AdkNode{
				{BaseNodeFields: db.BaseNodeFields{ID: "node1", Name: "class", Type: "agent"}},
			},
		}
		if err := wf.Validate(); err == nil {
			t.Error("Expected validation error for node name being a Python keyword, got nil")
		}
	})

	t.Run("DuplicateNodeNames", func(t *testing.T) {
		wf := db.WorkflowTemplate{
			Name:      "Invalid Flow",
			Framework: "ADK",
			Nodes: []db.AdkNode{
				{BaseNodeFields: db.BaseNodeFields{ID: "node1", Name: "agent_pm", Type: "agent"}},
				{BaseNodeFields: db.BaseNodeFields{ID: "node2", Name: "agent_pm", Type: "agent"}},
			},
		}
		if err := wf.Validate(); err == nil {
			t.Error("Expected validation error for duplicate node names, got nil")
		}
	})
}

type mockWorkflowRunStore struct {
	db.WorkflowRunStore
	runs []db.WorkflowRun
}

func (m *mockWorkflowRunStore) CreateWorkflowRun(ctx context.Context, organizationID string, run *db.WorkflowRun) error {
	run.ID = fmt.Sprintf("run-%d", len(m.runs)+1)
	m.runs = append(m.runs, *run)
	return nil
}

func (m *mockWorkflowRunStore) GetWorkflowRun(ctx context.Context, organizationID string, id string) (*db.WorkflowRun, error) {
	for _, r := range m.runs {
		if r.ID == id {
			runCopy := r
			return &runCopy, nil
		}
	}
	return nil, fmt.Errorf("not found")
}

func (m *mockWorkflowRunStore) DeleteWorkflowRun(ctx context.Context, organizationID string, id string) error {
	for i, r := range m.runs {
		if r.ID == id {
			m.runs = append(m.runs[:i], m.runs[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("not found")
}

func (m *mockWorkflowRunStore) ListWorkflowRuns(ctx context.Context, organizationID string) ([]db.WorkflowRun, error) {
	return m.runs, nil
}

func (m *mockWorkflowRunStore) UpdateWorkflowRunStatus(ctx context.Context, organizationID string, id string, status string) error {
	for i, r := range m.runs {
		if r.ID == id {
			m.runs[i].Status = status
			return nil
		}
	}
	return nil
}

func TestRunWorkflowBufferedHandler(t *testing.T) {
	t.Setenv("WORKFLOW_AGENT_BASE_URL", "http://localhost:9999")
	t.Setenv("WORKSPACE_MOUNT_PATH", "/nonexistent/path/for/test")
	origStore := integrations.Store
	defer func() { integrations.Store = origStore }()
	integrations.Store = &mockIntegrationStore{}

	mockRunStore := &mockWorkflowRunStore{}
	RunStore = mockRunStore

	claims := &core.UserClaims{
		Subject:        "test-user-uid",
		OrganizationID: "test-org-id",
		Organizations:  []string{"test-org-id"},
	}
	ctx := context.WithValue(context.Background(), core.UserClaimsKey, claims)
	ctx = context.WithValue(ctx, core.OrganizationIDKey, "test-org-id")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/wf-123/run?projectId=proj-123", nil)
	req = req.WithContext(ctx)
	req.SetPathValue("id", "wf-123")
	w := httptest.NewRecorder()

	// This will try to spawn the Python process and fail because of directory resolution in test,
	// but it should successfully log the run first!
	RunWorkflowBufferedHandler(w, req)

	// Verify that the run was logged despite the subsequent file system/execution failure
	if len(mockRunStore.runs) != 1 {
		t.Fatalf("Expected 1 workflow run to be logged, got %d", len(mockRunStore.runs))
	}

	loggedRun := mockRunStore.runs[0]
	if loggedRun.UserID != "test-user-uid" {
		t.Errorf("Expected UserID 'test-user-uid', got '%s'", loggedRun.UserID)
	}
	if loggedRun.ProjectID != "proj-123" {
		t.Errorf("Expected ProjectID 'proj-123', got '%s'", loggedRun.ProjectID)
	}
	if loggedRun.WorkflowID != "wf-123" {
		t.Errorf("Expected WorkflowID 'wf-123', got '%s'", loggedRun.WorkflowID)
	}
	if loggedRun.StartDate.IsZero() {
		t.Error("Expected StartDate to be set, got zero time")
	}

	// The handler should have returned 500 because it couldn't resolve/access the project workspace in test
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500 Internal Server Error, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestDeleteWorkflowRunHandler(t *testing.T) {
	mockRunStore := &mockWorkflowRunStore{
		runs: []db.WorkflowRun{
			{ID: "run-1", ProjectID: "proj-100", WorkflowID: "wf-1"},
			{ID: "run-2", ProjectID: "proj-200", WorkflowID: "wf-2"},
		},
	}
	RunStore = mockRunStore

	claims := &core.UserClaims{
		Subject:        "test-user-uid",
		OrganizationID: "test-org-id",
		Organizations:  []string{"test-org-id"},
	}
	ctx := context.WithValue(context.Background(), core.UserClaimsKey, claims)
	ctx = context.WithValue(ctx, core.OrganizationIDKey, "test-org-id")

	t.Run("Success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/proj-100/workflow-runs/run-1", nil)
		req = req.WithContext(ctx)
		req.SetPathValue("id", "proj-100")
		req.SetPathValue("runId", "run-1")
		w := httptest.NewRecorder()

		DeleteWorkflowRunHandler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200 OK, got %d. Body: %s", w.Code, w.Body.String())
		}

		if len(mockRunStore.runs) != 1 || mockRunStore.runs[0].ID != "run-2" {
			t.Errorf("Expected only run-2 left, got %+v", mockRunStore.runs)
		}
	})

	t.Run("ForbiddenDifferentProject", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/proj-100/workflow-runs/run-2", nil)
		req = req.WithContext(ctx)
		req.SetPathValue("id", "proj-100")
		req.SetPathValue("runId", "run-2")
		w := httptest.NewRecorder()

		DeleteWorkflowRunHandler(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected status 403 Forbidden, got %d", w.Code)
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/proj-100/workflow-runs/run-404", nil)
		req = req.WithContext(ctx)
		req.SetPathValue("id", "proj-100")
		req.SetPathValue("runId", "run-404")
		w := httptest.NewRecorder()

		DeleteWorkflowRunHandler(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404 Not Found, got %d", w.Code)
		}
	})
}

func TestListWorkflowRunsHandler_AwaitingInput(t *testing.T) {
	mockRunStore := &mockWorkflowRunStore{
		runs: []db.WorkflowRun{
			{ID: "run-waiting", ProjectID: "proj-1", WorkflowID: "wf-1", Status: "AWAITING_INPUT", AwaitingInput: true},
			{ID: "run-done", ProjectID: "proj-1", WorkflowID: "wf-1", Status: "COMPLETED", AwaitingInput: false},
		},
	}
	RunStore = mockRunStore
	Store = &mockWorkflowStore{
		workflows: map[string]db.WorkflowTemplate{
			"wf-1": {ID: "wf-1", Name: "Test Workflow"},
		},
	}

	claims := &core.UserClaims{
		Subject:        "test-user-uid",
		OrganizationID: "test-org-id",
		Organizations:  []string{"test-org-id"},
	}
	ctx := context.WithValue(context.Background(), core.UserClaimsKey, claims)
	ctx = context.WithValue(ctx, core.OrganizationIDKey, "test-org-id")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/proj-1/workflow-runs", nil)
	req = req.WithContext(ctx)
	req.SetPathValue("id", "proj-1")
	w := httptest.NewRecorder()

	ListWorkflowRunsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}

	var runs []map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&runs); err != nil {
		t.Fatalf("Failed to decode json: %v", err)
	}
	if len(runs) != 2 {
		t.Fatalf("Expected 2 runs, got %d", len(runs))
	}

	if runs[0]["status"] != "AWAITING_INPUT" || runs[0]["awaitingInput"] != true {
		t.Errorf("Expected run-waiting to have status AWAITING_INPUT and awaitingInput true, got %+v", runs[0])
	}
	if runs[1]["status"] != "COMPLETED" || runs[1]["awaitingInput"] == true {
		t.Errorf("Expected run-done to have status COMPLETED and awaitingInput false, got %+v", runs[1])
	}
}

func TestWorkflowTemplateAgentSkillsJSON(t *testing.T) {
	rawJSON := `{
		"name": "test_workflow",
		"nodes": [
			{
				"id": "agent_1",
				"name": "agent_1",
				"type": "agent",
				"agentId": "agent-pm",
				"model": "gemini-2.5-flash",
				"skills": [
					{
						"name": "brainstorming",
						"displayName": "Brainstorming",
						"description": "Ideation session"
					}
				],
				"tools": [
					{
						"name": "web_search",
						"displayName": "Web Search",
						"description": "Search engine tool"
					}
				]
			}
		]
	}`

	var tmpl db.WorkflowTemplate
	if err := json.Unmarshal([]byte(rawJSON), &tmpl); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if len(tmpl.Nodes) != 1 {
		t.Fatalf("Expected 1 node, got %d", len(tmpl.Nodes))
	}
	node := tmpl.Nodes[0]
	if node.AgentID != "agent-pm" {
		t.Errorf("Expected AgentID 'agent-pm', got '%s'", node.AgentID)
	}
	if len(node.Skills) != 1 || node.Skills[0].Name != "brainstorming" {
		t.Errorf("Expected 1 skill 'brainstorming', got %+v", node.Skills)
	}
	if len(node.Tools) != 1 || node.Tools[0].Name != "web_search" {
		t.Errorf("Expected 1 tool 'web_search', got %+v", node.Tools)
	}
}

func TestListWorkflowsHandler_FilterByType(t *testing.T) {
	mockStore := &mockWorkflowStore{
		workflows: map[string]db.WorkflowTemplate{
			"wf-bmad": {ID: "wf-bmad", Name: "Standard BMAD Workflow", Framework: "BMAD"},
			"wf-adk":  {ID: "wf-adk", Name: "ADK Dev Loop", Framework: "ADK 2.0"},
		},
	}
	Store = mockStore

	claims := &core.UserClaims{
		Subject:        "test-user-uid",
		OrganizationID: "test-org-id",
	}
	ctx := context.WithValue(context.Background(), core.UserClaimsKey, claims)

	t.Run("FilterBMAD", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows?type=BMAD", nil)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		ListWorkflowsHandler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200 OK, got %d", w.Code)
		}

		var workflows []db.WorkflowTemplate
		if err := json.NewDecoder(w.Body).Decode(&workflows); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(workflows) != 1 {
			t.Fatalf("Expected 1 workflow for type=BMAD, got %d", len(workflows))
		}

		if workflows[0].ID != "wf-bmad" {
			t.Errorf("Expected BMAD workflow 'wf-bmad', got %+v", workflows[0])
		}
	})

	t.Run("FilterADK", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows?type=ADK", nil)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		ListWorkflowsHandler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200 OK, got %d", w.Code)
		}

		var workflows []db.WorkflowTemplate
		if err := json.NewDecoder(w.Body).Decode(&workflows); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(workflows) != 1 {
			t.Fatalf("Expected 1 workflow for type=ADK, got %d", len(workflows))
		}

		if workflows[0].ID != "wf-adk" {
			t.Errorf("Expected ADK workflow 'wf-adk', got %+v", workflows[0])
		}
	})

	t.Run("NoFilterParam", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows", nil)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		ListWorkflowsHandler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200 OK, got %d", w.Code)
		}

		var workflows []db.WorkflowTemplate
		if err := json.NewDecoder(w.Body).Decode(&workflows); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(workflows) != 2 {
			t.Fatalf("Expected 2 workflows without type param, got %d", len(workflows))
		}
	})
}

