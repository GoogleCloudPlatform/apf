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
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

type WorkflowRun struct {
	ID            string    `json:"id" firestore:"-"`
	UserID        string    `json:"userId" firestore:"userId"`
	ProjectID     string    `json:"projectId" firestore:"projectId"`
	WorkflowID    string    `json:"workflowId" firestore:"workflowId"`
	StartDate     time.Time `json:"startDate" firestore:"startDate"`
	Status        string    `json:"status,omitempty" firestore:"status,omitempty"`
	AwaitingInput bool      `json:"awaitingInput,omitempty" firestore:"awaitingInput,omitempty"`
}

type WorkflowRunStore interface {
	CreateWorkflowRun(ctx context.Context, organizationID string, run *WorkflowRun) error
	GetWorkflowRun(ctx context.Context, organizationID string, id string) (*WorkflowRun, error)
	ListWorkflowRuns(ctx context.Context, organizationID string) ([]WorkflowRun, error)
	UpdateWorkflowRunStatus(ctx context.Context, organizationID string, id string, status string) error
	DeleteWorkflowRun(ctx context.Context, organizationID string, id string) error
}

var WorkflowRunStoreInstance WorkflowRunStore

type FirestoreWorkflowRunStore struct {
	client *firestore.Client
}

func NewFirestoreWorkflowRunStore(client *firestore.Client) *FirestoreWorkflowRunStore {
	return &FirestoreWorkflowRunStore{client: client}
}

func (s *FirestoreWorkflowRunStore) CreateWorkflowRun(ctx context.Context, organizationID string, run *WorkflowRun) error {
	if organizationID == "" {
		return fmt.Errorf("organizationID cannot be empty")
	}
	if run == nil {
		return fmt.Errorf("workflow run cannot be nil")
	}
	if run.StartDate.IsZero() {
		run.StartDate = time.Now()
	}

	col := organizationCollection(organizationID, s.client, "workflow_runs")
	ref, _, err := col.Add(ctx, run)
	if err != nil {
		return err
	}
	run.ID = ref.ID
	return nil
}

func (s *FirestoreWorkflowRunStore) ListWorkflowRuns(ctx context.Context, organizationID string) ([]WorkflowRun, error) {
	if organizationID == "" {
		return nil, fmt.Errorf("organizationID cannot be empty")
	}

	var runs []WorkflowRun
	col := organizationCollection(organizationID, s.client, "workflow_runs")
	iter := col.OrderBy("startDate", firestore.Desc).Documents(ctx)
	defer iter.Stop()

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var run WorkflowRun
		if err := doc.DataTo(&run); err != nil {
			return nil, err
		}
		run.ID = doc.Ref.ID
		runs = append(runs, run)
	}
	return runs, nil
}

func (s *FirestoreWorkflowRunStore) GetWorkflowRun(ctx context.Context, organizationID string, id string) (*WorkflowRun, error) {
	if organizationID == "" {
		return nil, fmt.Errorf("organizationID cannot be empty")
	}
	if id == "" {
		return nil, fmt.Errorf("id cannot be empty")
	}

	col := organizationCollection(organizationID, s.client, "workflow_runs")
	doc, err := col.Doc(id).Get(ctx)
	if err != nil {
		return nil, err
	}
	var run WorkflowRun
	if err := doc.DataTo(&run); err != nil {
		return nil, err
	}
	run.ID = doc.Ref.ID
	return &run, nil
}

func (s *FirestoreWorkflowRunStore) DeleteWorkflowRun(ctx context.Context, organizationID string, id string) error {
	if organizationID == "" {
		return fmt.Errorf("organizationID cannot be empty")
	}
	if id == "" {
		return fmt.Errorf("id cannot be empty")
	}

	// First delete any associated session/messages for this workflow run ID
	sessionsCol := organizationCollection(organizationID, s.client, "sessions")
	messagesRef := sessionsCol.Doc(id).Collection("messages")
	iter := messagesRef.DocumentRefs(ctx)
	for {
		docRef, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to list messages for workflow run deletion: %w", err)
		}
		if _, err := docRef.Delete(ctx); err != nil {
			return fmt.Errorf("failed to delete message for workflow run: %w", err)
		}
	}
	// Delete any session document itself if it exists
	_, _ = sessionsCol.Doc(id).Delete(ctx)

	// Now delete the workflow run document
	col := organizationCollection(organizationID, s.client, "workflow_runs")
	_, err := col.Doc(id).Delete(ctx)
	return err
}

func (s *FirestoreWorkflowRunStore) UpdateWorkflowRunStatus(ctx context.Context, organizationID string, id string, status string) error {
	if organizationID == "" || id == "" {
		return nil
	}
	col := organizationCollection(organizationID, s.client, "workflow_runs")
	_, err := col.Doc(id).Update(ctx, []firestore.Update{
		{Path: "status", Value: status},
	})
	return err
}
