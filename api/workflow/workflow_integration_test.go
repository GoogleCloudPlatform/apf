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
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"google.com/apf/api/agent"
	core "google.com/apf/api/core"
	"google.com/apf/api/db"
	"google.com/apf/api/integrations"
)

func TestWorkflowRunnerIntegration(t *testing.T) {
	// Only run this test if explicitly requested (e.g., via go test -run TestWorkflowRunnerIntegration)
	// to prevent it from running in standard CI/CD tests since it requires the local stack running.
	if os.Getenv("RUN_INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test; set RUN_INTEGRATION_TEST=true to run")
	}

	ctx := context.Background()

	// 1. Force Emulator Environment
	os.Setenv("FIRESTORE_EMULATOR_HOST", "localhost:8084")
	os.Setenv("FIRESTORE_DB_NAME", "apf-local-firestore")
	os.Setenv("GOOGLE_CLOUD_PROJECT", "agenticproductfoundry-489717")

	cfg := core.NewConfig()
	
	// 2. Initialize real Firestore client pointing to emulator
	client, err := firestore.NewClientWithDatabase(ctx, cfg.GoogleCloudProject, cfg.FirestoreDbName)
	if err != nil {
		t.Fatalf("Failed to create Firestore client: %v", err)
	}
	defer client.Close()

	// 3. Initialize global stores
	sessionStore := db.NewFirestoreSessionStore(client)
	RunStore = db.NewFirestoreWorkflowRunStore(client)
	Store = db.NewFirestoreWorkflowStore(client)
	integrations.Store = db.NewFirestoreIntegrationStore(client)
	
	agent.Store = db.NewFirestoreMessageStore(client, sessionStore)
	agent.SetConfig(cfg)

	// 4. Construct HTTP request
	workflowID := "wBcWWbTFveltFm45C9vp"
	projectID := "foundry-showcase-project"
	userID := "test-user@example.com"
	orgID := "62NBs2UptUZVXWDvV2gr7gOEyLBF"

	req, err := http.NewRequest("POST", fmt.Sprintf("/api/v1/workflows/%s/run?projectId=%s", workflowID, projectID), nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.SetPathValue("id", workflowID)

	// 5. Inject Context Claims (simulating authMiddleware and orgScopeMiddleware)
	claims := &core.UserClaims{
		Subject:  userID,
		Username: userID,
	}
	
	reqCtx := req.Context()
	reqCtx = context.WithValue(reqCtx, core.UserClaimsKey, claims)
	
	// Inject organization ID claims
	orgCtxKey := core.ContextKey("organizationId")
	reqCtx = context.WithValue(reqCtx, orgCtxKey, orgID)
	
	req = req.WithContext(reqCtx)

	// 6. Record response
	w := httptest.NewRecorder()

	t.Log("=========================================================")
	t.Logf("🚀 Triggering integration run for workflow: %s", workflowID)
	t.Logf("   Project: %s, User: %s, Org: %s", projectID, userID, orgID)
	t.Log("=========================================================")

	start := time.Now()
	RunWorkflowBufferedHandler(w, req)
	duration := time.Since(start)

	t.Log("=========================================================")
	t.Logf("📊 Run finished in %v", duration)
	t.Logf("   Response Status: %d", w.Code)
	t.Log("=========================================================")

	if w.Code != http.StatusOK {
		t.Fatalf("Workflow run failed with status %d. Response: %s", w.Code, w.Body.String())
	}

	t.Logf("Response Output Payload:\n%s", w.Body.String())
}
