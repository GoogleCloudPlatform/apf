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

package workspace

import (
	"context"
	"errors"
	"testing"

	"google.com/apf/api/agent"
	"google.com/apf/api/db"
)

type mockGateArtifactStore struct {
	db.ArtifactStore
	artifacts        []db.Artifact
	updatedArtifacts []db.Artifact
	createdArtifacts []db.Artifact
	err              error
}

func (m *mockGateArtifactStore) ListArtifactsForProject(ctx context.Context, organizationID string, projectId string) ([]db.Artifact, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.artifacts, nil
}

func (m *mockGateArtifactStore) UpdateArtifact(ctx context.Context, organizationID string, a *db.Artifact) error {
	if m.err != nil {
		return m.err
	}
	m.updatedArtifacts = append(m.updatedArtifacts, *a)
	for i := range m.artifacts {
		if m.artifacts[i].ID == a.ID {
			m.artifacts[i] = *a
			break
		}
	}
	return nil
}

func (m *mockGateArtifactStore) CreateArtifact(ctx context.Context, organizationID string, a *db.Artifact) error {
	if m.err != nil {
		return m.err
	}
	m.createdArtifacts = append(m.createdArtifacts, *a)
	m.artifacts = append(m.artifacts, *a)
	return nil
}

type mockGateProjectStore struct {
	db.ProjectStore
	project *db.Project
	updated *db.Project
	err     error
}

func (m *mockGateProjectStore) GetProject(ctx context.Context, organizationID string, id string) (*db.Project, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.project == nil {
		return nil, errors.New("not found")
	}
	return m.project, nil
}

func (m *mockGateProjectStore) UpdateProject(ctx context.Context, organizationID string, p *db.Project) error {
	if m.err != nil {
		return m.err
	}
	m.updated = p
	return nil
}

type mockGateMessageStore struct {
	db.MessageStore
	createdMessages []*db.Message
}

func (m *mockGateMessageStore) CreateMessage(ctx context.Context, organizationID string, projectId string, userId string, msg *db.Message) (*db.Message, error) {
	m.createdMessages = append(m.createdMessages, msg)
	return msg, nil
}

func TestRecalculateGates(t *testing.T) {
	// Set up original stores to restore later
	origArtifactStore := ArtifactStore
	origStore := db.ProjectStoreInstance
	origMessageStore := agent.Store
	defer func() {
		ArtifactStore = origArtifactStore
		db.ProjectStoreInstance = origStore
		agent.Store = origMessageStore
	}()

	// Define some target mock artifacts: one phase ANALYSIS (no required), planning with required
	mockArts := []db.Artifact{
		{
			ID:        "art1",
			ProjectId: "proj123",
			Phase:     "ANALYSIS",
			Status:    db.ArtifactStatusComplete,
			Required:  false,
		},
		{
			ID:        "art2",
			ProjectId: "proj123",
			Phase:     "PLANNING",
			Status:    db.ArtifactStatusStagedForReview,
			Required:  true,
		},
		{
			ID:        "art3",
			ProjectId: "proj123",
			Phase:     "SOLUTIONING",
			Status:    db.ArtifactStatusInProgress,
			Required:  true,
		},
	}

	mockProj := &db.Project{
		ID: "proj123",
	}

	mArtifactStore := &mockGateArtifactStore{artifacts: mockArts}
	mProjectStore := &mockGateProjectStore{project: mockProj}
	mMessageStore := &mockGateMessageStore{}

	ArtifactStore = mArtifactStore
	db.ProjectStoreInstance = mProjectStore
	agent.Store = mMessageStore

	err := recalculateGates(context.Background(), "test-organization", "proj123")
	if err != nil {
		t.Fatalf("recalculateGates failed: %v", err)
	}

	// Verify that the project gates were updated correctly
	if mProjectStore.updated == nil {
		t.Fatal("Expected project to be updated, but it was not")
	}

	gates := *mProjectStore.updated.GovernanceGates
	if gates["ANALYSIS"] != "Complete" {
		t.Errorf("Expected ANALYSIS to be Complete, got %s", gates["ANALYSIS"])
	}

	// Since PLANNING has a required artifact with status "Staged for Review", it should be "Complete" now!
	if gates["PLANNING"] != "Complete" {
		t.Errorf("Expected PLANNING to be Complete, got %s", gates["PLANNING"])
	}

	// Since SOLUTIONING only has an "In Progress" artifact and no Complete/Staged ones, it should be "In Progress"
	if gates["SOLUTIONING"] != "In Progress" {
		t.Errorf("Expected SOLUTIONING to be In Progress, got %s", gates["SOLUTIONING"])
	}

	// Verify a notification message was broadcasted
	if len(mMessageStore.createdMessages) != 1 {
		t.Errorf("Expected 1 notification message, got %d", len(mMessageStore.createdMessages))
	} else {
		msg := mMessageStore.createdMessages[0]
		if msg.NotificationDetails == nil || msg.NotificationDetails.Name != "governance-update" {
			t.Errorf("Expected notification 'governance-update', got details: %v", msg.NotificationDetails)
		}
	}
}

func TestOnFileWriteHook_TransitionToInProgress(t *testing.T) {
	origArtifactStore := ArtifactStore
	origStore := db.ProjectStoreInstance
	origMessageStore := agent.Store
	defer func() {
		ArtifactStore = origArtifactStore
		db.ProjectStoreInstance = origStore
		agent.Store = origMessageStore
	}()

	mockArts := []db.Artifact{
		{
			ID:        "art_analysis_1",
			ProjectId: "proj123",
			Phase:     "ANALYSIS",
			Title:     "analysis-doc.md",
			Path:      "docs/analysis-doc.md",
			Status:    db.ArtifactStatusNotStarted,
		},
	}

	mockProj := &db.Project{
		ID: "proj123",
	}

	mArtifactStore := &mockGateArtifactStore{artifacts: mockArts}
	mProjectStore := &mockGateProjectStore{project: mockProj}
	mMessageStore := &mockGateMessageStore{}

	ArtifactStore = mArtifactStore
	db.ProjectStoreInstance = mProjectStore
	agent.Store = mMessageStore

	// Trigger OnFileWriteHook for a non-onboarding phase
	agent.OnFileWriteHook(context.Background(), "proj123", "test-organization", "ANALYSIS", "docs/analysis-doc.md")

	// 1. Verify the artifact status was transitioned to In Progress
	if len(mArtifactStore.updatedArtifacts) != 1 {
		t.Fatalf("Expected 1 artifact to be updated, got %d", len(mArtifactStore.updatedArtifacts))
	}
	updatedArt := mArtifactStore.updatedArtifacts[0]
	if updatedArt.Status != db.ArtifactStatusInProgress {
		t.Errorf("Expected artifact status to be In Progress, got %s", updatedArt.Status)
	}

	// 2. Verify that gates were recalculated (ANALYSIS transitioned to In Progress via the hook, so ANALYSIS gate should be In Progress)
	if mProjectStore.updated == nil {
		t.Fatal("Expected project gates to be recalculated/updated, but they were not")
	}
	gates := *mProjectStore.updated.GovernanceGates
	if gates["ANALYSIS"] != "In Progress" {
		t.Errorf("Expected ANALYSIS gate to be In Progress, got %s", gates["ANALYSIS"])
	}
}

func TestOnFileWriteHook_CreatesMissingArtifact(t *testing.T) {
	origArtifactStore := ArtifactStore
	origStore := db.ProjectStoreInstance
	origMessageStore := agent.Store
	defer func() {
		ArtifactStore = origArtifactStore
		db.ProjectStoreInstance = origStore
		agent.Store = origMessageStore
	}()

	mockProj := &db.Project{
		ID: "proj123",
	}

	mArtifactStore := &mockGateArtifactStore{artifacts: []db.Artifact{}}
	mProjectStore := &mockGateProjectStore{project: mockProj}
	mMessageStore := &mockGateMessageStore{}

	ArtifactStore = mArtifactStore
	db.ProjectStoreInstance = mProjectStore
	agent.Store = mMessageStore

	agent.OnFileWriteHook(context.Background(), "proj123", "test-organization", "PLANNING", "docs/spec/prd.md")

	if len(mArtifactStore.createdArtifacts) != 1 {
		t.Fatalf("Expected 1 artifact to be created, got %d", len(mArtifactStore.createdArtifacts))
	}

	createdArt := mArtifactStore.createdArtifacts[0]
	if createdArt.ID != "proj123_prd.md" {
		t.Errorf("Expected ID proj123_prd.md, got %s", createdArt.ID)
	}
	if createdArt.Title != "prd.md" {
		t.Errorf("Expected Title prd.md, got %s", createdArt.Title)
	}
	if createdArt.Path != "docs/spec/prd.md" {
		t.Errorf("Expected Path docs/spec/prd.md, got %s", createdArt.Path)
	}
	if createdArt.Phase != "PLANNING" {
		t.Errorf("Expected Phase PLANNING, got %s", createdArt.Phase)
	}
	if createdArt.Status != db.ArtifactStatusInProgress {
		t.Errorf("Expected Status In Progress, got %s", createdArt.Status)
	}
	if createdArt.ArtifactType != db.ArtifactTypePrd {
		t.Errorf("Expected ArtifactType %s, got %s", db.ArtifactTypePrd, createdArt.ArtifactType)
	}
}

func TestOnFileWriteHook_SkipsTmpDirectory(t *testing.T) {
	origArtifactStore := ArtifactStore
	origStore := db.ProjectStoreInstance
	origMessageStore := agent.Store
	defer func() {
		ArtifactStore = origArtifactStore
		db.ProjectStoreInstance = origStore
		agent.Store = origMessageStore
	}()

	mockProj := &db.Project{
		ID: "proj123",
	}

	mArtifactStore := &mockGateArtifactStore{artifacts: []db.Artifact{}}
	mProjectStore := &mockGateProjectStore{project: mockProj}
	mMessageStore := &mockGateMessageStore{}

	ArtifactStore = mArtifactStore
	db.ProjectStoreInstance = mProjectStore
	agent.Store = mMessageStore

	// File in .tmp folder should be completely ignored (no artifacts created, no project updates)
	agent.OnFileWriteHook(context.Background(), "proj123", "test-organization", "ANALYSIS", ".tmp/populate_brainstorm.py")

	if len(mArtifactStore.createdArtifacts) != 0 {
		t.Errorf("Expected 0 artifacts created for .tmp file, got %d", len(mArtifactStore.createdArtifacts))
	}
	if len(mArtifactStore.updatedArtifacts) != 0 {
		t.Errorf("Expected 0 artifacts updated for .tmp file, got %d", len(mArtifactStore.updatedArtifacts))
	}
	if mProjectStore.updated != nil {
		t.Errorf("Expected project store not to be updated for .tmp file, but was updated")
	}
}


