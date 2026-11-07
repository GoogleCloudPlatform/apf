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
	"regexp"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

// BaseNodeFields contains the common fields shared by all ADK nodes.
type BaseNodeFields struct {
	ID            string                 `json:"id" firestore:"id"`
	Name          string                 `json:"name" firestore:"name"`
	DisplayName   string                 `json:"displayName,omitempty" firestore:"displayName,omitempty"`
	Type          string                 `json:"type" firestore:"type"` // "agent", "function", "join", "custom", "trigger"
	Description   string                 `json:"description,omitempty" firestore:"description,omitempty"`
	RerunOnResume bool                   `json:"rerunOnResume,omitempty" firestore:"rerunOnResume,omitempty"`
	WaitForOutput bool                   `json:"waitForOutput,omitempty" firestore:"waitForOutput,omitempty"`
	Timeout       float64                `json:"timeout,omitempty" firestore:"timeout,omitempty"`
	RetryConfig   *RetryConfig           `json:"retryConfig,omitempty" firestore:"retryConfig,omitempty"`
	InputSchema   map[string]interface{} `json:"inputSchema,omitempty" firestore:"inputSchema,omitempty"`
	OutputSchema  map[string]interface{} `json:"outputSchema,omitempty" firestore:"outputSchema,omitempty"`
	StateSchema   map[string]interface{} `json:"stateSchema,omitempty" firestore:"stateSchema,omitempty"`
	Input         map[string]interface{} `json:"input,omitempty" firestore:"input,omitempty"`
	Output        map[string]interface{} `json:"output,omitempty" firestore:"output,omitempty"`
}

// Skill represents a skill assigned to an Agent node.
type Skill struct {
	DisplayName string `json:"displayName,omitempty" firestore:"displayName,omitempty"`
	Name        string `json:"name,omitempty" firestore:"name,omitempty"`
	Description string `json:"description,omitempty" firestore:"description,omitempty"`
}

// Tool represents a tool assigned to an Agent node.
type Tool struct {
	DisplayName string `json:"displayName,omitempty" firestore:"displayName,omitempty"`
	Name        string `json:"name,omitempty" firestore:"name,omitempty"`
	Description string `json:"description,omitempty" firestore:"description,omitempty"`
}

// AgentFields contains fields specific to LLM Agent nodes.
type AgentFields struct {
	Model       string  `json:"model,omitempty" firestore:"model,omitempty"`
	Instruction string  `json:"instruction,omitempty" firestore:"instruction,omitempty"`
	AgentID     string  `json:"agentId,omitempty" firestore:"agentId,omitempty"`
	Skills      []Skill `json:"skills,omitempty" firestore:"skills,omitempty"`
	Tools       []Tool  `json:"tools,omitempty" firestore:"tools,omitempty"`
}

// FunctionFields contains fields specific to deterministic Python function nodes.
type FunctionFields struct {
	ParameterBinding string                 `json:"parameterBinding,omitempty" firestore:"parameterBinding,omitempty"` // "state" | "node_input"
	AuthConfig       map[string]interface{} `json:"authConfig,omitempty" firestore:"authConfig,omitempty"`
}

// CustomFields contains fields specific to custom/legacy node types.
type CustomFields struct {
	ParallelWorker bool `json:"parallelWorker,omitempty" firestore:"parallelWorker,omitempty"`
}

// TriggerFields contains fields specific to Trigger nodes (e.g., ScheduledTrigger).
type TriggerFields struct {
	TriggerInterval     string `json:"triggerInterval,omitempty" firestore:"triggerInterval,omitempty"`
	IntervalValue       int    `json:"intervalValue,omitempty" firestore:"intervalValue,omitempty"`
	TriggerAtMinute     int    `json:"triggerAtMinute,omitempty" firestore:"triggerAtMinute,omitempty"`
	TriggerAtHour       int    `json:"triggerAtHour,omitempty" firestore:"triggerAtHour,omitempty"`
	TriggerOnWeekdays   []int  `json:"triggerOnWeekdays,omitempty" firestore:"triggerOnWeekdays,omitempty"`
	TriggerAtDayOfMonth int    `json:"triggerAtDayOfMonth,omitempty" firestore:"triggerAtDayOfMonth,omitempty"`
	CronExpression      string `json:"cronExpression,omitempty" firestore:"cronExpression,omitempty"`
}

// LoopFields contains fields specific to Loop nodes.
type LoopFields struct {
	ExitCondition map[string]interface{} `json:"exitCondition,omitempty" firestore:"exitCondition,omitempty"`
}

// AdkNode represents a node in an ADK 2.0 workflow graph, composed of structured fields.
type AdkNode struct {
	BaseNodeFields
	AgentFields
	FunctionFields
	CustomFields
	TriggerFields
	LoopFields
}

// RetryConfig represents the retry policy for a node.
type RetryConfig struct {
	MaxAttempts   int      `json:"maxAttempts,omitempty" firestore:"maxAttempts,omitempty"`
	InitialDelay  float64  `json:"initialDelay,omitempty" firestore:"initialDelay,omitempty"`
	MaxDelay      float64  `json:"maxDelay,omitempty" firestore:"maxDelay,omitempty"`
	BackoffFactor float64  `json:"backoffFactor,omitempty" firestore:"backoffFactor,omitempty"`
	Jitter        float64  `json:"jitter,omitempty" firestore:"jitter,omitempty"`
	Exceptions    []string `json:"exceptions,omitempty" firestore:"exceptions,omitempty"`
}

// AdkEdge represents a directed edge in an ADK 2.0 workflow graph.
type AdkEdge struct {
	ID       string      `json:"id" firestore:"id"`
	FromNode string      `json:"fromNode" firestore:"fromNode"`
	ToNode   string      `json:"toNode" firestore:"toNode"`
	Route    interface{} `json:"route,omitempty" firestore:"route,omitempty"` // string, bool, int, or array of these
}

// VisualNode represents a node in the visual designer canvas.
type VisualNode struct {
	ID          string                 `json:"id" firestore:"id"`
	Type        string                 `json:"type" firestore:"type"`
	Position    map[string]float64     `json:"position" firestore:"position"` // {"x": 100, "y": 200}
	Orientation string                 `json:"orientation,omitempty" firestore:"orientation,omitempty"`
	Data        map[string]interface{} `json:"data,omitempty" firestore:"data,omitempty"`
}

// VisualEdge represents an edge in the visual designer canvas.
type VisualEdge struct {
	ID           string      `json:"id" firestore:"id"`
	Source       string      `json:"source" firestore:"source"`
	Target       string      `json:"target" firestore:"target"`
	SourceHandle string      `json:"sourceHandle,omitempty" firestore:"sourceHandle,omitempty"`
	TargetHandle string      `json:"targetHandle,omitempty" firestore:"targetHandle,omitempty"`
	Route        interface{} `json:"route,omitempty" firestore:"route,omitempty"`
}

// VisualLayout represents the full designer canvas visual state.
type VisualLayout struct {
	Nodes []VisualNode `json:"nodes,omitempty" firestore:"nodes,omitempty"`
	Edges []VisualEdge `json:"edges,omitempty" firestore:"edges,omitempty"`
}

// WorkflowTemplate defines the structure of a workflow.
type WorkflowTemplate struct {
	ID           string                        `json:"id" firestore:"-"`
	Name         string                        `json:"name" firestore:"name"`
	DisplayName  string                        `json:"displayName,omitempty" firestore:"displayName,omitempty"`
	Description  string                        `json:"description,omitempty" firestore:"description,omitempty"`
	Framework    string                        `json:"framework" firestore:"framework"` // e.g., "BMAD", "ADK"
	Phases       map[string][]PhaseAction      `json:"phases,omitempty" firestore:"phases,omitempty"`
	Artifacts    map[string][]ArtifactTemplate `json:"artifacts,omitempty" firestore:"artifacts,omitempty"`
	Dependencies map[string][]string           `json:"dependencies,omitempty" firestore:"dependencies,omitempty"`

	// ADK 2.0 Graph-based fields (pure logical execution graph)
	Nodes []AdkNode `json:"nodes,omitempty" firestore:"nodes,omitempty"`
	Edges []AdkEdge `json:"edges,omitempty" firestore:"edges,omitempty"`

	// Visual designer layout (completely separated from logical graph)
	Layout *VisualLayout `json:"layout,omitempty" firestore:"layout,omitempty"`
}

// PhaseAction represents a method or action in a phase.
type PhaseAction struct {
	ID            string `json:"id" firestore:"id"`
	Name          string `json:"name" firestore:"name"`
	Prompt        string `json:"prompt" firestore:"prompt"`
	DisplayPrompt string `json:"displayPrompt,omitempty" firestore:"displayPrompt,omitempty"`
}

// ArtifactTemplate represents a required artifact definition.
type ArtifactTemplate struct {
	ID       string `json:"id" firestore:"id"`
	Title    string `json:"title" firestore:"title"`
	Required bool   `json:"required" firestore:"required"`
}

type WorkflowStore interface {
	ListWorkflows(ctx context.Context) ([]WorkflowTemplate, error)
	GetWorkflow(ctx context.Context, id string) (*WorkflowTemplate, error)
	CreateWorkflow(ctx context.Context, w *WorkflowTemplate) error
	DeleteWorkflow(ctx context.Context, id string) error
}

type FirestoreWorkflowStore struct {
	client *firestore.Client
}

func NewFirestoreWorkflowStore(client *firestore.Client) *FirestoreWorkflowStore {
	return &FirestoreWorkflowStore{client: client}
}

func (s *FirestoreWorkflowStore) ListWorkflows(ctx context.Context) ([]WorkflowTemplate, error) {
	var workflows []WorkflowTemplate
	iter := s.client.Collection("workflows").Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var w WorkflowTemplate
		if err := doc.DataTo(&w); err != nil {
			return nil, err
		}
		w.ID = doc.Ref.ID
		workflows = append(workflows, w)
	}
	return workflows, nil
}

func (s *FirestoreWorkflowStore) DeleteWorkflow(ctx context.Context, id string) error {
	_, err := s.client.Collection("workflows").Doc(id).Delete(ctx)
	return err
}

func (s *FirestoreWorkflowStore) GetWorkflow(ctx context.Context, id string) (*WorkflowTemplate, error) {
	doc, err := s.client.Collection("workflows").Doc(id).Get(ctx)
	if err != nil {
		return nil, err
	}
	var w WorkflowTemplate
	if err := doc.DataTo(&w); err != nil {
		return nil, err
	}
	w.ID = doc.Ref.ID
	return &w, nil
}

func (s *FirestoreWorkflowStore) CreateWorkflow(ctx context.Context, w *WorkflowTemplate) error {
	var ref *firestore.DocumentRef
	var err error
	if w.ID == "" {
		ref, _, err = s.client.Collection("workflows").Add(ctx, w)
	} else {
		ref = s.client.Collection("workflows").Doc(w.ID)
		_, err = ref.Set(ctx, w)
	}
	if err != nil {
		return err
	}
	w.ID = ref.ID
	return nil
}

var (
	pythonIdentifierRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
	pythonKeywords        = map[string]bool{
		"False": true, "None": true, "True": true, "and": true, "as": true, "assert": true, "async": true, "await": true, "break": true,
		"class": true, "continue": true, "def": true, "del": true, "elif": true, "else": true, "except": true, "finally": true,
		"for": true, "from": true, "global": true, "if": true, "import": true, "in": true, "is": true, "lambda": true, "nonlocal": true,
		"not": true, "or": true, "pass": true, "raise": true, "return": true, "try": true, "while": true, "with": true, "yield": true,
	}
)

func IsValidPythonIdentifier(name string) bool {
	if !pythonIdentifierRegex.MatchString(name) {
		return false
	}
	return !pythonKeywords[name]
}

func (w *WorkflowTemplate) Validate() error {
	if w.Name == "" {
		return errors.New("workflow name is required")
	}

	if w.Framework != "BMAD" {
		names := make(map[string]bool)
		for _, node := range w.Nodes {
			if node.ID == "start" {
				continue
			}
			if node.Name == "" {
				return errors.New("node name is required for all execution nodes")
			}
			if !IsValidPythonIdentifier(node.Name) {
				return errors.New("node name '" + node.Name + "' is invalid: must be a valid Python identifier (alphanumeric/underscores only, cannot start with a digit, and cannot be a Python keyword)")
			}
			if names[node.Name] {
				return errors.New("duplicate node name detected: '" + node.Name + "'")
			}
			names[node.Name] = true
		}
	}

	return nil
}
