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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"google.com/apf/api/agent"
	"google.com/apf/api/core"
	"google.com/apf/api/db"
	"google.com/apf/api/integrations"
)

type mockGitArtifactStore struct {
	db.ArtifactStore
	artifacts []db.Artifact
	updated   *db.Artifact
}

func (m *mockGitArtifactStore) ListArtifactsForProject(ctx context.Context, organizationID string, projectId string) ([]db.Artifact, error) {
	return m.artifacts, nil
}

func (m *mockGitArtifactStore) UpdateArtifact(ctx context.Context, organizationID string, a *db.Artifact) error {
	m.updated = a
	return nil
}

type mockGitProjectStore struct {
	db.ProjectStore
	project *db.Project
	updated *db.Project
}

func (m *mockGitProjectStore) GetProject(ctx context.Context, organizationID string, id string) (*db.Project, error) {
	return m.project, nil
}

func (m *mockGitProjectStore) UpdateProject(ctx context.Context, organizationID string, p *db.Project) error {
	m.updated = p
	return nil
}

type mockGitUserProjectStore struct {
	db.UserProjectStore
	setting *db.UserProject
}

func (m *mockGitUserProjectStore) GetProjectAppSetting(ctx context.Context, organizationID string, userId, projectId, appId string) (*db.UserProject, error) {
	return m.setting, nil
}

type mockGitSessionStore struct {
	db.SessionStore
	sessions []db.Session
}

func (m *mockGitSessionStore) ListSessionsForProject(ctx context.Context, organizationID string, projectId, userId, phase string) ([]db.Session, error) {
	return m.sessions, nil
}

type mockGitIntegrationStore struct {
	db.IntegrationStore
	integration *db.Integration
	err         error
}

func (m *mockGitIntegrationStore) GetIntegration(ctx context.Context, userId, name string) (*db.Integration, error) {
	return m.integration, m.err
}

type mockGitClient struct {
	pullErr        error
	commitErr      error
	stageErr       error
	pushErr        error
	initErr        error
	calledPull     bool
	calledCommit   bool
	calledStage    bool
	calledPush     bool
	calledInit     bool
	lastRepoCfg    *GitRepoConfig
	lastFilePath   string
	lastCommitMsg  string
}

func (m *mockGitClient) Pull(ctx context.Context, organizationID string, repoCfg *GitRepoConfig) error {
	m.calledPull = true
	m.lastRepoCfg = repoCfg
	return m.pullErr
}

func (m *mockGitClient) LocalCommit(ctx context.Context, organizationID string, repoCfg *GitRepoConfig, filePath string, message string, authorName string, authorEmail string) error {
	m.calledCommit = true
	m.lastRepoCfg = repoCfg
	m.lastFilePath = filePath
	m.lastCommitMsg = message
	return m.commitErr
}

func (m *mockGitClient) StagePath(ctx context.Context, organizationID string, repoCfg *GitRepoConfig, relPath string) error {
	m.calledStage = true
	m.lastRepoCfg = repoCfg
	m.lastFilePath = relPath
	return m.stageErr
}

func (m *mockGitClient) Push(ctx context.Context, organizationID string, repoCfg *GitRepoConfig) error {
	m.calledPush = true
	m.lastRepoCfg = repoCfg
	return m.pushErr
}

func (m *mockGitClient) EnsureRepoInitialized(ctx context.Context, organizationID string, repoCfg *GitRepoConfig) error {
	m.calledInit = true
	m.lastRepoCfg = repoCfg
	return m.initErr
}

func (m *mockGitClient) CreateSnapshot(ctx context.Context, organizationID string, repoCfg *GitRepoConfig, description string) (*GitSnapshot, error) {
	return &GitSnapshot{
		ID:          "refs/foundry/backups/test-snapshot",
		RefName:     "refs/foundry/backups/test-snapshot",
		Timestamp:   time.Now(),
		CommitHash:  "1234567",
		Description: description,
	}, nil
}

func (m *mockGitClient) RestoreSnapshot(ctx context.Context, organizationID string, repoCfg *GitRepoConfig, snapshotID string) error {
	return nil
}

func (m *mockGitClient) ListSnapshots(ctx context.Context, organizationID string, repoCfg *GitRepoConfig) ([]*GitSnapshot, error) {
	return []*GitSnapshot{
		{
			ID:          "refs/foundry/backups/test-snapshot",
			RefName:     "refs/foundry/backups/test-snapshot",
			Timestamp:   time.Now(),
			CommitHash:  "1234567",
			Description: "Test snapshot",
		},
	}, nil
}

func (m *mockGitClient) GetWorkspaceStatus(ctx context.Context, organizationID string, repoCfg *GitRepoConfig) ([]GitFileStatusItem, error) {
	return []GitFileStatusItem{
		{
			Path:      "docs/spec/prd.md",
			Staged:    "modified",
			Worktree:  "modified",
			IsChecked: true,
		},
	}, nil
}

func (m *mockGitClient) FastForwardBaseToSpec(ctx context.Context, organizationID string, repoCfg *GitRepoConfig) error {
	return nil
}

func (m *mockGitClient) CheckBranchMergeability(ctx context.Context, organizationID string, repoCfg *GitRepoConfig, baseBranch, specBranch string) (bool, []string, error) {
	return true, nil, nil
}

func TestGitSyncHandler_Success(t *testing.T) {
	// Mock stores
	origArtifactStore := ArtifactStore
	origStore := db.ProjectStoreInstance
	origUserProjectStore := UserProjectStore
	origSessionStore := SessionStore
	origMessageStore := agent.Store
	origIntegrationStore := integrations.Store
	origGitClient := gitClient
	defer func() {
		ArtifactStore = origArtifactStore
		db.ProjectStoreInstance = origStore
		UserProjectStore = origUserProjectStore
		SessionStore = origSessionStore
		agent.Store = origMessageStore
		integrations.Store = origIntegrationStore
		gitClient = origGitClient
	}()

	projectId := "proj123"
	userId := "user@google.com"
	repoURL := "https://github.com/test-owner/test-repo"

	proj := &db.Project{
		ID: projectId,
		AppIntegrations: []db.ProjectIntegration{
			{
				IntegrationID: "github",
				Enabled:       true,
				Settings: map[string]interface{}{
					"repoUrl": repoURL,
				},
			},
		},
	}

	art := db.Artifact{
		ID:        "art123",
		ProjectId: projectId,
		Path:      "docs/spec/prd.md",
		Title:     "prd.md",
		Status:    db.ArtifactStatusInProgress,
	}

	mArtStore := &mockGitArtifactStore{artifacts: []db.Artifact{art}}
	mProjStore := &mockGitProjectStore{project: proj}
	mUserProjStore := &mockGitUserProjectStore{setting: &db.UserProject{IsReadOnly: false}}
	mSessionStore := &mockGitSessionStore{sessions: []db.Session{
		{
			ID:             "sess123",
			ProjectId:      projectId,
			UserId:         userId,
			SpecBranchName: "feature-branch",
		},
	}}
	mIntegrationStore := &mockGitIntegrationStore{
		integration: &db.Integration{
			UserID: userId,
			Name:   "github",
			Settings: map[string]interface{}{
				"accessToken": "mock-oauth-token",
			},
		},
	}

	ArtifactStore = mArtStore
	db.ProjectStoreInstance = mProjStore
	UserProjectStore = mUserProjStore
	SessionStore = mSessionStore
	agent.Store = &safeDummyMessageStore{}
	integrations.Store = mIntegrationStore

	mGitClient := &mockGitClient{}
	gitClient = mGitClient

	// Setup Mock GitHub API
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "GET" && r.URL.Path == "/repos/test-owner/test-repo/branches/feature-branch" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"name": "feature-branch"}`))
			return
		}
		if r.Method == "POST" && r.URL.Path == "/repos/test-owner/test-repo/pulls" {
			var payload integrations.PullRequestPayload
			json.NewDecoder(r.Body).Decode(&payload)
			if payload.Head != "feature-branch" || payload.Base != "main" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(integrations.PullRequestResponse{
				HTMLURL: "https://github.com/test-owner/test-repo/pull/1",
				Number:  1,
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	integrations.SetGitHubAPIURLForTesting(ts.URL)
	defer integrations.SetGitHubAPIURLForTesting("https://api.github.com")

	reqBody := GitSyncRequest{
		FilePath:   "docs/spec/prd.md",
		ArtifactID: "art123",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/proj123/git/sync", bytes.NewReader(bodyBytes))
	req.SetPathValue("id", projectId)

	claims := &core.UserClaims{Subject: userId, OrganizationID: "test-organization"}
	req = req.WithContext(context.WithValue(req.Context(), core.UserClaimsKey, claims))

	w := httptest.NewRecorder()
	GitSyncHandler(w, req)

	if !mGitClient.calledCommit {
		t.Error("Expected gitClient.LocalCommit to be called")
	}

	if !mGitClient.calledPush {
		t.Error("Expected gitClient.Push to be called")
	}

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "SUCCESS" {
		t.Errorf("Expected status SUCCESS, got %q", resp["status"])
	}
	if resp["branchName"] != "feature-branch" {
		t.Errorf("Expected branchName 'feature-branch', got %q", resp["branchName"])
	}

	// Verify artifact status updated to Staged for Review
	if mArtStore.updated == nil {
		t.Error("Expected artifact to be updated")
	} else if mArtStore.updated.Status != db.ArtifactStatusStagedForReview {
		t.Errorf("Expected artifact status Staged for Review, got %q", mArtStore.updated.Status)
	}
}

func TestGitMergeHandler_Success(t *testing.T) {
	// Mock stores
	origArtifactStore := ArtifactStore
	origStore := db.ProjectStoreInstance
	origUserProjectStore := UserProjectStore
	origSessionStore := SessionStore
	origMessageStore := agent.Store
	origIntegrationStore := integrations.Store
	origGitClient := gitClient
	defer func() {
		ArtifactStore = origArtifactStore
		db.ProjectStoreInstance = origStore
		UserProjectStore = origUserProjectStore
		SessionStore = origSessionStore
		agent.Store = origMessageStore
		integrations.Store = origIntegrationStore
		gitClient = origGitClient
	}()

	projectId := "proj123"
	userId := "user@google.com"
	repoURL := "https://github.com/test-owner/test-repo"

	proj := &db.Project{
		ID: projectId,
		AppIntegrations: []db.ProjectIntegration{
			{
				IntegrationID: "github",
				Enabled:       true,
				Settings: map[string]interface{}{
					"repoUrl": repoURL,
				},
			},
		},
	}

	art := db.Artifact{
		ID:        "art123",
		ProjectId: projectId,
		Path:      "docs/spec/prd.md",
		Title:     "prd.md",
		Status:    db.ArtifactStatusInProgress,
	}

	mArtStore := &mockGitArtifactStore{artifacts: []db.Artifact{art}}
	mProjStore := &mockGitProjectStore{project: proj}
	mUserProjStore := &mockGitUserProjectStore{setting: &db.UserProject{IsReadOnly: false}}
	mSessionStore := &mockGitSessionStore{sessions: []db.Session{
		{
			ID:             "sess123",
			ProjectId:      projectId,
			UserId:         userId,
			SpecBranchName: "feature-branch",
		},
	}}
	mIntegrationStore := &mockGitIntegrationStore{
		integration: &db.Integration{
			UserID: userId,
			Name:   "github",
			Settings: map[string]interface{}{
				"accessToken": "mock-oauth-token",
			},
		},
	}

	ArtifactStore = mArtStore
	db.ProjectStoreInstance = mProjStore
	UserProjectStore = mUserProjStore
	SessionStore = mSessionStore
	agent.Store = &safeDummyMessageStore{}
	integrations.Store = mIntegrationStore

	mGitClient := &mockGitClient{}
	gitClient = mGitClient

	// Setup Mock GitHub API
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "GET" && r.URL.Path == "/repos/test-owner/test-repo/branches/feature-branch" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"name": "feature-branch"}`))
			return
		}
		if r.Method == "POST" && r.URL.Path == "/repos/test-owner/test-repo/merges" {
			var payload map[string]string
			json.NewDecoder(r.Body).Decode(&payload)
			if payload["head"] != "feature-branch" || payload["base"] != "main" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusCreated)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	integrations.SetGitHubAPIURLForTesting(ts.URL)
	defer integrations.SetGitHubAPIURLForTesting("https://api.github.com")

	reqBody := GitSyncRequest{
		FilePath:   "docs/spec/prd.md",
		ArtifactID: "art123",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/proj123/git/merge", bytes.NewReader(bodyBytes))
	req.SetPathValue("id", projectId)

	claims := &core.UserClaims{Subject: userId, OrganizationID: "test-organization"}
	req = req.WithContext(context.WithValue(req.Context(), core.UserClaimsKey, claims))

	w := httptest.NewRecorder()
	GitMergeHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "SUCCESS" {
		t.Errorf("Expected status SUCCESS, got %q", resp["status"])
	}

	// Verify artifact status updated to Complete
	if mArtStore.updated == nil {
		t.Error("Expected artifact to be updated")
	} else if mArtStore.updated.Status != db.ArtifactStatusComplete {
		t.Errorf("Expected artifact status Complete, got %q", mArtStore.updated.Status)
	}
}
func TestGitSyncHandler_PermissionError(t *testing.T) {
	// Mock stores
	origArtifactStore := ArtifactStore
	origStore := db.ProjectStoreInstance
	origUserProjectStore := UserProjectStore
	origSessionStore := SessionStore
	origMessageStore := agent.Store
	origIntegrationStore := integrations.Store
	origGitClient := gitClient
	defer func() {
		ArtifactStore = origArtifactStore
		db.ProjectStoreInstance = origStore
		UserProjectStore = origUserProjectStore
		SessionStore = origSessionStore
		agent.Store = origMessageStore
		integrations.Store = origIntegrationStore
		gitClient = origGitClient
	}()

	projectId := "proj123"
	userId := "user@google.com"
	repoURL := "https://github.com/test-owner/test-repo"

	proj := &db.Project{
		ID: projectId,
		AppIntegrations: []db.ProjectIntegration{
			{
				IntegrationID: "github",
				Enabled:       true,
				Settings: map[string]interface{}{
					"repoUrl": repoURL,
				},
			},
		},
	}

	art := db.Artifact{
		ID:        "art123",
		ProjectId: projectId,
		Path:      "docs/spec/prd.md",
		Title:     "prd.md",
		Status:    db.ArtifactStatusInProgress,
	}

	mArtStore := &mockGitArtifactStore{artifacts: []db.Artifact{art}}
	mProjStore := &mockGitProjectStore{project: proj}
	mUserProjStore := &mockGitUserProjectStore{setting: &db.UserProject{IsReadOnly: false}}
	mSessionStore := &mockGitSessionStore{sessions: []db.Session{
		{
			ID:             "sess123",
			ProjectId:      projectId,
			UserId:         userId,
			SpecBranchName: "feature-branch",
		},
	}}
	mIntegrationStore := &mockGitIntegrationStore{
		integration: &db.Integration{
			UserID: userId,
			Name:   "github",
			Settings: map[string]interface{}{
				"accessToken": "mock-oauth-token",
			},
		},
	}

	ArtifactStore = mArtStore
	db.ProjectStoreInstance = mProjStore
	UserProjectStore = mUserProjStore
	SessionStore = mSessionStore
	agent.Store = &safeDummyMessageStore{}
	integrations.Store = mIntegrationStore

	mGitClient := &mockGitClient{pushErr: ErrWriteAccessDenied}
	gitClient = mGitClient

	reqBody := GitSyncRequest{
		FilePath:   "docs/spec/prd.md",
		ArtifactID: "art123",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/proj123/git/sync", bytes.NewReader(bodyBytes))
	req.SetPathValue("id", projectId)

	claims := &core.UserClaims{Subject: userId, OrganizationID: "test-organization"}
	req = req.WithContext(context.WithValue(req.Context(), core.UserClaimsKey, claims))

	w := httptest.NewRecorder()
	GitSyncHandler(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403 Forbidden, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp Response
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Status != "ERROR" {
		t.Errorf("Expected status ERROR, got %q", resp.Status)
	}
	if resp.Title != "Access Denied" {
		t.Errorf("Expected title 'Access Denied', got %q", resp.Title)
	}
	if resp.Message != "Failed to push to remote repository: write access not granted or token expired." {
		t.Errorf("Expected message 'Failed to push to remote repository: write access not granted or token expired.', got %q", resp.Message)
	}
}

func TestGitStatusHandler_Success(t *testing.T) {
	origArtifactStore := ArtifactStore
	origStore := db.ProjectStoreInstance
	origUserProjectStore := UserProjectStore
	origSessionStore := SessionStore
	origGitClient := gitClient
	defer func() {
		ArtifactStore = origArtifactStore
		db.ProjectStoreInstance = origStore
		UserProjectStore = origUserProjectStore
		SessionStore = origSessionStore
		gitClient = origGitClient
	}()

	projectId := "proj123"
	userId := "user@google.com"

	mArtifactStore := &mockGitArtifactStore{}
	mProjectStore := &mockGitProjectStore{
		project: &db.Project{ID: projectId},
	}
	mUserProjectStore := &mockGitUserProjectStore{
		setting: &db.UserProject{IsReadOnly: false},
	}
	mSessionStore := &mockGitSessionStore{}

	ArtifactStore = mArtifactStore
	db.ProjectStoreInstance = mProjectStore
	UserProjectStore = mUserProjectStore
	SessionStore = mSessionStore

	mGitClient := &mockGitClient{}
	gitClient = mGitClient

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/proj123/git/local-changes", nil)
	req.SetPathValue("id", projectId)

	claims := &core.UserClaims{Subject: userId, OrganizationID: "test-organization"}
	req = req.WithContext(context.WithValue(req.Context(), core.UserClaimsKey, claims))

	w := httptest.NewRecorder()
	GitStatusHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}

	var statusItems []GitFileStatusItem
	if err := json.NewDecoder(w.Body).Decode(&statusItems); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(statusItems) != 1 {
		t.Fatalf("Expected 1 status item, got %d", len(statusItems))
	}
	if statusItems[0].Path != "docs/spec/prd.md" {
		t.Errorf("Expected path 'docs/spec/prd.md', got %q", statusItems[0].Path)
	}
}

func TestGitSyncHandler_PullMode_ConflictReturns409(t *testing.T) {
	origArtifactStore := ArtifactStore
	origStore := db.ProjectStoreInstance
	origUserProjectStore := UserProjectStore
	origSessionStore := SessionStore
	origIntegrationStore := integrations.Store
	origGitClient := gitClient
	defer func() {
		ArtifactStore = origArtifactStore
		db.ProjectStoreInstance = origStore
		UserProjectStore = origUserProjectStore
		SessionStore = origSessionStore
		integrations.Store = origIntegrationStore
		gitClient = origGitClient
	}()

	projectId := "proj123"
	userId := "user@google.com"
	repoURL := "https://github.com/test-owner/test-repo"

	mArtifactStore := &mockGitArtifactStore{}
	mProjectStore := &mockGitProjectStore{
		project: &db.Project{
			ID: projectId,
			AppIntegrations: []db.ProjectIntegration{
				{
					IntegrationID: "github",
					Enabled:       true,
					Settings: map[string]interface{}{
						"repoUrl": repoURL,
					},
				},
			},
		},
	}
	mUserProjectStore := &mockGitUserProjectStore{
		setting: &db.UserProject{IsReadOnly: false},
	}
	mSessionStore := &mockGitSessionStore{
		sessions: []db.Session{
			{SpecBranchName: "specs/user-branch"},
		},
	}
	mIntegrationStore := &mockGitIntegrationStore{
		integration: &db.Integration{
			UserID: userId,
			Name:   "github",
			Settings: map[string]interface{}{
				"accessToken": "mock-oauth-token",
			},
		},
	}

	ArtifactStore = mArtifactStore
	db.ProjectStoreInstance = mProjectStore
	UserProjectStore = mUserProjectStore
	SessionStore = mSessionStore
	integrations.Store = mIntegrationStore

	mGitClient := &mockGitClient{
		pullErr: fmt.Errorf("rebase failed due to conflict: merge conflict in file docs/spec/prd.md"),
	}
	gitClient = mGitClient

	bodyBytes, _ := json.Marshal(GitSyncRequest{
		Action: "pull",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/proj123/git/sync", bytes.NewReader(bodyBytes))
	req.SetPathValue("id", projectId)

	claims := &core.UserClaims{Subject: userId, OrganizationID: "test-organization"}
	req = req.WithContext(context.WithValue(req.Context(), core.UserClaimsKey, claims))

	w := httptest.NewRecorder()
	GitSyncHandler(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("Expected status 409 Conflict, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestGitSyncHandler_PushMode_ConflictFallbackToPR(t *testing.T) {
	origArtifactStore := ArtifactStore
	origStore := db.ProjectStoreInstance
	origUserProjectStore := UserProjectStore
	origSessionStore := SessionStore
	origIntegrationStore := integrations.Store
	origGitClient := gitClient
	defer func() {
		ArtifactStore = origArtifactStore
		db.ProjectStoreInstance = origStore
		UserProjectStore = origUserProjectStore
		SessionStore = origSessionStore
		integrations.Store = origIntegrationStore
		gitClient = origGitClient
	}()

	projectId := "proj123"
	userId := "user@google.com"
	repoURL := "https://github.com/test-owner/test-repo"

	mArtifactStore := &mockGitArtifactStore{}
	mProjectStore := &mockGitProjectStore{
		project: &db.Project{
			ID: projectId,
			AppIntegrations: []db.ProjectIntegration{
				{
					IntegrationID: "github",
					Enabled:       true,
					Settings: map[string]interface{}{
						"repoUrl": repoURL,
					},
				},
			},
		},
	}
	mUserProjectStore := &mockGitUserProjectStore{
		setting: &db.UserProject{IsReadOnly: false},
	}
	mSessionStore := &mockGitSessionStore{
		sessions: []db.Session{
			{SpecBranchName: "specs/user-branch"},
		},
	}
	mIntegrationStore := &mockGitIntegrationStore{
		integration: &db.Integration{
			UserID: userId,
			Name:   "github",
			Settings: map[string]interface{}{
				"accessToken": "mock-oauth-token",
			},
		},
	}

	ArtifactStore = mArtifactStore
	db.ProjectStoreInstance = mProjectStore
	UserProjectStore = mUserProjectStore
	SessionStore = mSessionStore
	integrations.Store = mIntegrationStore

	mGitClient := &mockGitClient{
		pullErr: fmt.Errorf("rebase conflict during cherry-pick"),
	}
	gitClient = mGitClient

	bodyBytes, _ := json.Marshal(GitSyncRequest{
		Action: "push",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/proj123/git/sync", bytes.NewReader(bodyBytes))
	req.SetPathValue("id", projectId)

	claims := &core.UserClaims{Subject: userId, OrganizationID: "test-organization"}
	req = req.WithContext(context.WithValue(req.Context(), core.UserClaimsKey, claims))

	w := httptest.NewRecorder()
	GitSyncHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}
	if !mGitClient.calledPush {
		t.Errorf("Expected GitClient.Push to be called")
	}
}

func TestGitSyncHandler_ReviewMode_Success(t *testing.T) {
	origArtifactStore := ArtifactStore
	origStore := db.ProjectStoreInstance
	origUserProjectStore := UserProjectStore
	origSessionStore := SessionStore
	origIntegrationStore := integrations.Store
	origGitClient := gitClient
	defer func() {
		ArtifactStore = origArtifactStore
		db.ProjectStoreInstance = origStore
		UserProjectStore = origUserProjectStore
		SessionStore = origSessionStore
		integrations.Store = origIntegrationStore
		gitClient = origGitClient
	}()

	projectId := "proj123"
	userId := "user@google.com"
	repoURL := "https://github.com/test-owner/test-repo"

	mArtifactStore := &mockGitArtifactStore{}
	mProjectStore := &mockGitProjectStore{
		project: &db.Project{
			ID: projectId,
			AppIntegrations: []db.ProjectIntegration{
				{
					IntegrationID: "github",
					Enabled:       true,
					Settings: map[string]interface{}{
						"repoUrl": repoURL,
					},
				},
			},
		},
	}
	mUserProjectStore := &mockGitUserProjectStore{
		setting: &db.UserProject{IsReadOnly: false},
	}
	mSessionStore := &mockGitSessionStore{
		sessions: []db.Session{
			{SpecBranchName: "specs/user-branch"},
		},
	}
	mIntegrationStore := &mockGitIntegrationStore{
		integration: &db.Integration{
			UserID: userId,
			Name:   "github",
			Settings: map[string]interface{}{
				"accessToken": "mock-oauth-token",
			},
		},
	}

	ArtifactStore = mArtifactStore
	db.ProjectStoreInstance = mProjectStore
	UserProjectStore = mUserProjectStore
	SessionStore = mSessionStore
	integrations.Store = mIntegrationStore

	mGitClient := &mockGitClient{}
	gitClient = mGitClient

	bodyBytes, _ := json.Marshal(GitSyncRequest{
		Action: "request_review",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/proj123/git/sync", bytes.NewReader(bodyBytes))
	req.SetPathValue("id", projectId)

	claims := &core.UserClaims{Subject: userId, OrganizationID: "test-organization"}
	req = req.WithContext(context.WithValue(req.Context(), core.UserClaimsKey, claims))

	w := httptest.NewRecorder()
	GitSyncHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}
	if !mGitClient.calledPull {
		t.Errorf("Expected GitClient.Pull to be called before push")
	}
	if !mGitClient.calledPush {
		t.Errorf("Expected GitClient.Push to be called")
	}
}
