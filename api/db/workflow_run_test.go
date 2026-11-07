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
	"os"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
)

func TestWorkflowRunStore(t *testing.T) {
	emulatorHost := os.Getenv("FIRESTORE_EMULATOR_HOST")
	if emulatorHost == "" {
		t.Skip("Skipping Firestore integration test: FIRESTORE_EMULATOR_HOST is not set")
	}

	ctx := testCtx()
	client, err := firestore.NewClient(ctx, "agenticproductfoundry-489717")
	if err != nil {
		t.Fatalf("Failed to create Firestore client: %v", err)
	}
	defer client.Close()

	runStore := NewFirestoreWorkflowRunStore(client)
	orgID := "test-org-runs"

	run1 := &WorkflowRun{
		UserID:     "user-1",
		ProjectID:  "project-1",
		WorkflowID: "workflow-1",
		StartDate:  time.Now(),
	}

	err = runStore.CreateWorkflowRun(ctx, orgID, run1)
	if err != nil {
		t.Fatalf("Failed to create workflow run: %v", err)
	}

	if run1.ID == "" {
		t.Errorf("Expected run ID to be populated, got empty")
	}

	run2 := &WorkflowRun{
		UserID:     "user-2",
		ProjectID:  "project-2",
		WorkflowID: "workflow-2",
		StartDate:  time.Now().Add(1 * time.Minute),
	}

	err = runStore.CreateWorkflowRun(ctx, orgID, run2)
	if err != nil {
		t.Fatalf("Failed to create second workflow run: %v", err)
	}

	// List runs and verify
	runs, err := runStore.ListWorkflowRuns(ctx, orgID)
	if err != nil {
		t.Fatalf("Failed to list workflow runs: %v", err)
	}

	if len(runs) != 2 {
		t.Errorf("Expected 2 runs, got %d", len(runs))
	}

	// Since we order by startDate desc, run2 should be first
	if runs[0].ID != run2.ID {
		t.Errorf("Expected first run in list to be run2 (%s), got %s", run2.ID, runs[0].ID)
	}
	if runs[1].ID != run1.ID {
		t.Errorf("Expected second run in list to be run1 (%s), got %s", run1.ID, runs[1].ID)
	}

	// Clean up
	col := organizationCollection(orgID, client, "workflow_runs")
	_, _ = col.Doc(run1.ID).Delete(ctx)
	_, _ = col.Doc(run2.ID).Delete(ctx)
}
