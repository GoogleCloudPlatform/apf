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
	"encoding/json"
	"os"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

func TestDeleteProject_Cascading(t *testing.T) {
	emulatorHost := os.Getenv("FIRESTORE_EMULATOR_HOST")
	if emulatorHost == "" {
		t.Skip("Skipping Firestore integration test: FIRESTORE_EMULATOR_HOST is not set")
	}

	ctx := testCtx()

	// Connect to the emulator using a mock project ID
	client, err := firestore.NewClient(ctx, "agenticproductfoundry-489717")
	if err != nil {
		t.Fatalf("Failed to create Firestore client: %v", err)
	}
	defer client.Close()

	store := NewFirestoreProjectStore(client)

	// 1. Setup/Seed Mock Data for a Project
	projName := "Cascading Test Project"
	projDesc := "Testing recursive deletions"
	projStatus := "In Progress"
	projOwner := "test-owner"
	projVisibility := "private"

	project := &Project{
		Name:       &projName,
		Desc:       &projDesc,
		Status:     &projStatus,
		OwnerId:    &projOwner,
		Visibility: &projVisibility,
	}

	err = store.CreateProject(ctx, "test-organization", project)
	if err != nil {
		t.Fatalf("Failed to create test project: %v", err)
	}
	projectID := project.ID
	if projectID == "" {
		t.Fatal("Expected project ID to be generated, got empty string")
	}

	// 2. Seed associated access_requests
	reqStore := NewFirestoreAccessRequestStore(client)
	req1 := &AccessRequest{
		ProjectId:     projectID,
		UserId:        "user-1",
		UserName:      "User One",
		RequestedRole: "member",
		Status:        "pending",
		RequestDate:   time.Now().Format(time.RFC3339),
	}
	req2 := &AccessRequest{
		ProjectId:     projectID,
		UserId:        "user-2",
		UserName:      "User Two",
		RequestedRole: "admin",
		Status:        "approved",
		RequestDate:   time.Now().Format(time.RFC3339),
	}
	if err := reqStore.CreateRequest(ctx, "test-organization", req1); err != nil {
		t.Fatalf("Failed to create access request 1: %v", err)
	}
	if err := reqStore.CreateRequest(ctx, "test-organization", req2); err != nil {
		t.Fatalf("Failed to create access request 2: %v", err)
	}

	// 3. Seed associated artifacts
	artStore := NewFirestoreArtifactStore(client)
	art1 := &Artifact{
		ProjectId:    projectID,
		WorkflowID:   "bmad",
		Title:        "PRD.md",
		Path:         "docs/PRD.md",
		Phase:        "PLANNING",
		Status:       ArtifactStatusComplete,
		ArtifactType: DetermineArtifactType("docs/PRD.md"),
	}
	art2 := &Artifact{
		ProjectId:    projectID,
		WorkflowID:   "bmad",
		Title:        "architecture.md",
		Path:         "docs/architecture.md",
		Phase:        "SOLUTIONING",
		Status:       ArtifactStatusInProgress,
		ArtifactType: DetermineArtifactType("docs/architecture.md"),
	}
	if err := artStore.CreateArtifact(ctx, "test-organization", art1); err != nil {
		t.Fatalf("Failed to create artifact 1: %v", err)
	}
	if err := artStore.CreateArtifact(ctx, "test-organization", art2); err != nil {
		t.Fatalf("Failed to create artifact 2: %v", err)
	}

	// 4. Seed sessions and nested messages
	sessionStore := NewFirestoreSessionStore(client)
	sess1 := &Session{
		UserId:    "user-1",
		ProjectId: projectID,
		Phase:     "PLANNING",
		CreatedAt: time.Now(),
		Status:    "active",
	}
	sess2 := &Session{
		UserId:    "user-2",
		ProjectId: projectID,
		Phase:     "SOLUTIONING",
		CreatedAt: time.Now(),
		Status:    "inactive",
	}
	if err := sessionStore.CreateSession(ctx, "test-organization", sess1); err != nil {
		t.Fatalf("Failed to create session 1: %v", err)
	}
	if err := sessionStore.CreateSession(ctx, "test-organization", sess2); err != nil {
		t.Fatalf("Failed to create session 2: %v", err)
	}

	// Seed messages for session 1
	_, _, err = organizationCollectionRef(client, "sessions").Doc(sess1.ID).Collection("messages").Add(ctx, map[string]interface{}{
		"sender":    "user",
		"text":      "Hello agent",
		"timestamp": time.Now(),
	})
	if err != nil {
		t.Fatalf("Failed to seed message for session 1: %v", err)
	}

	// Seed messages for session 2
	_, _, err = organizationCollectionRef(client, "sessions").Doc(sess2.ID).Collection("messages").Add(ctx, map[string]interface{}{
		"sender":    "agent",
		"text":      "Hello user",
		"timestamp": time.Now(),
	})
	if err != nil {
		t.Fatalf("Failed to seed message for session 2: %v", err)
	}

	// 5. Seed associated user_project and nested app_credentials
	upStore := NewFirestoreUserProjectStore(client)
	upSetting := &UserProject{
		UserID:       "user-1",
		ProjectID:    projectID,
		AppID:        "figma",
		Enabled:      true,
		CredentialID: "figma-global-credential-ref",
	}
	if err := upStore.SaveProjectAppSetting(ctx, "test-organization", upSetting); err != nil {
		t.Fatalf("Failed to create user project setting: %v", err)
	}

	// 6. Execute delete operation
	err = store.DeleteProject(ctx, "test-organization", projectID)
	if err != nil {
		t.Fatalf("DeleteProject failed: %v", err)
	}

	// 7. Verify Deletions

	// Check project document
	_, err = organizationCollectionRef(client, "projects").Doc(projectID).Get(ctx)
	if err == nil {
		t.Errorf("Expected project document %s to be deleted, but it still exists", projectID)
	}

	// Check access_requests
	reqsIter := organizationCollectionRef(client, "access_requests").Where("projectId", "==", projectID).Documents(ctx)
	for {
		doc, err := reqsIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			t.Fatalf("Error iterating access requests: %v", err)
		}
		t.Errorf("Expected access request %s to be deleted, but it still exists", doc.Ref.ID)
	}

	// Check artifacts
	artsIter := organizationCollectionRef(client, "artifacts").Where("projectId", "==", projectID).Documents(ctx)
	for {
		doc, err := artsIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			t.Fatalf("Error iterating artifacts: %v", err)
		}
		t.Errorf("Expected artifact %s to be deleted, but it still exists", doc.Ref.ID)
	}

	// Check sessions and messages
	sessIter := organizationCollectionRef(client, "sessions").Where("projectId", "==", projectID).Documents(ctx)
	for {
		doc, err := sessIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			t.Fatalf("Error iterating sessions: %v", err)
		}
		t.Errorf("Expected session %s to be deleted, but it still exists", doc.Ref.ID)

		// Check nested messages in case the session document was somehow deleted but messages remained (orphaned subcollection)
		msgsIter := doc.Ref.Collection("messages").Documents(ctx)
		for {
			msgDoc, err := msgsIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				t.Fatalf("Error iterating session messages: %v", err)
			}
			t.Errorf("Expected nested message %s to be deleted, but it still exists", msgDoc.Ref.ID)
		}
	}

	// Check user_project and app_credentials
	parentDocID := "user-1_" + projectID
	_, err = organizationCollectionRef(client, "user_project").Doc(parentDocID).Get(ctx)
	if err == nil {
		t.Errorf("Expected user_project parent document %s to be deleted, but it still exists", parentDocID)
	}
	// Also check that all app_credentials under it are gone
	upCredsIter := organizationCollectionRef(client, "user_project").Doc(parentDocID).Collection("app_credentials").Documents(ctx)
	for {
		doc, err := upCredsIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			t.Fatalf("Error iterating user project app credentials: %v", err)
		}
		t.Errorf("Expected user project app credential %s to be deleted, but it still exists", doc.Ref.ID)
	}
}

func TestProject_IntegrationSettings(t *testing.T) {
	// Test default fallbacks
	pEmpty := &Project{}
	if got := pEmpty.GetAiOrchestrator(); got != "adk" {
		t.Errorf("Expected GetAiOrchestrator() = 'adk', got '%s'", got)
	}
	if got := pEmpty.GetGitSyncRoute(); got != "pr-gate" {
		t.Errorf("Expected GetGitSyncRoute() = 'pr-gate', got '%s'", got)
	}

	// Test customized values
	pCustom := &Project{
		IntegrationSettings: &IntegrationSettings{
			AiOrchestrator: "agy-sdk",
		},
	}
	if got := pCustom.GetAiOrchestrator(); got != "agy-sdk" {
		t.Errorf("Expected GetAiOrchestrator() = 'agy-sdk', got '%s'", got)
	}
	if got := pCustom.GetGitSyncRoute(); got != "pr-gate" {
		t.Errorf("Expected GetGitSyncRoute() = 'pr-gate', got '%s'", got)
	}
}

func TestProject_MarshalJSON(t *testing.T) {
	p := &Project{
		BmadConfig: &BmadConfig{},
	}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("Failed to marshal project: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Failed to unmarshal project JSON: %v", err)
	}

	if got, want := raw["specLocation"], "docs/spec"; got != want {
		t.Errorf("Expected specLocation %q, got %v", want, got)
	}
	if got, want := raw["planningArtifactsLocation"], "docs/spec/plans"; got != want {
		t.Errorf("Expected planningArtifactsLocation %q, got %v", want, got)
	}
	if got, want := raw["implementationArtifactsLocation"], "docs/spec/implementation"; got != want {
		t.Errorf("Expected implementationArtifactsLocation %q, got %v", want, got)
	}
	if got, want := raw["testingArtifactsLocation"], "docs/spec/test"; got != want {
		t.Errorf("Expected testingArtifactsLocation %q, got %v", want, got)
	}
}
