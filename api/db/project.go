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
	"encoding/json"
	"fmt"
	"log"
	"path"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)



type Project struct {
	ID                              string               `json:"id" firestore:"-"`
	Name                            *string              `json:"name" firestore:"name"`
	Desc                            *string              `json:"description" firestore:"description"`
	Status                          *string              `json:"status" firestore:"status"`
	OwnerId                         *string              `json:"ownerId" firestore:"ownerId"`
	Visibility                      *string              `json:"visibility" firestore:"visibility"` // public, private, hidden
	BaseBranch                      *string              `json:"baseBranch" firestore:"baseBranch"`
	Icon                            *string              `json:"icon" firestore:"icon"`
	Members                         *map[string]string   `json:"members" firestore:"members"`                 // user_id -> role
	GovernanceGates                 *map[string]string   `json:"governanceGates" firestore:"governanceGates"` // Phase -> Status
	IntegrationSettings             *IntegrationSettings `json:"integrationSettings" firestore:"integrationSettings"`
	AppIntegrations                 []ProjectIntegration `json:"app_integrations" firestore:"app_integrations"`
	BmadConfig                       *BmadConfig          `json:"bmadConfig" firestore:"bmadConfig"`
	LinkedWorkflows                 []string             `json:"linkedWorkflows,omitempty" firestore:"linkedWorkflows,omitempty"`
	Artifacts                       []Artifact           `json:"artifacts" firestore:"-"`
	IsReadOnly                      bool                 `json:"isReadOnly" firestore:"-"`
	CreatedAt                       time.Time            `json:"createdAt" firestore:"createdAt"`
}

type BmadConfig struct {
	SpecFolder                      string `json:"specFolder" firestore:"specFolder"`
	ImplementationArtifactsPath     string `json:"implementationArtifactsPath" firestore:"implementationArtifactsPath"`
	PlanningArtifactsPath           string `json:"planningArtifactsPath" firestore:"planningArtifactsPath"`
	TestingArtifactsPath            string `json:"testingArtifactsPath" firestore:"testingArtifactsPath"`
}

type ProjectIntegration struct {
	IntegrationID string                 `json:"integration_id" firestore:"integration_id"`
	Enabled       bool                   `json:"enabled" firestore:"enabled"`
	Settings      map[string]interface{} `json:"settings" firestore:"settings"`
}

type IntegrationSettings struct {
	AiOrchestrator string `json:"aiOrchestrator" firestore:"aiOrchestrator"`
}

func (p *Project) GetAiOrchestrator() string {
	if p.IntegrationSettings != nil && p.IntegrationSettings.AiOrchestrator != "" {
		return p.IntegrationSettings.AiOrchestrator
	}
	return "adk"
}

func (p *Project) GetGitSyncRoute() string {
	return "pr-gate"
}

func (p *Project) GetGitHubSetting(key string) string {
	for _, integration := range p.AppIntegrations {
		if integration.IntegrationID == "github" && integration.Enabled {
			if val, ok := integration.Settings[key].(string); ok {
				return val
			}
		}
	}
	return ""
}

func (p *Project) GetRepoUrl() string {
	return p.GetGitHubSetting("repoUrl")
}

func (p *Project) GetSpecLocation() string {
	if p.BmadConfig != nil && p.BmadConfig.SpecFolder != "" {
		return p.BmadConfig.SpecFolder
	}
	val := p.GetGitHubSetting("specLocation")
	if val != "" {
		return val
	}
	return "docs/spec"
}

func (p *Project) GetSpecBranchPrefix() string {
	val := p.GetGitHubSetting("specBranchPrefix")
	if val == "" {
		return "specs"
	}
	return val
}

func (p *Project) GetGitHubEmail() string {
	return p.GetGitHubSetting("email")
}

func cleanRelativeSubpath(specFolder, subpath string) string {
	cleanSpec := path.Clean(specFolder)
	cleanSub := path.Clean(subpath)
	if strings.HasPrefix(cleanSub, cleanSpec+"/") {
		return strings.TrimPrefix(cleanSub, cleanSpec+"/")
	}
	if cleanSub == cleanSpec {
		return ""
	}
	return subpath
}

func (p *Project) GetPlanningArtifactsLocation() string {
	if p.BmadConfig != nil && p.BmadConfig.PlanningArtifactsPath != "" {
		return path.Join(p.GetSpecLocation(), cleanRelativeSubpath(p.GetSpecLocation(), p.BmadConfig.PlanningArtifactsPath))
	}
	val := p.GetGitHubSetting("planningArtifactsLocation")
	if val != "" {
		return val
	}
	return path.Join(p.GetSpecLocation(), "plans")
}

func (p *Project) GetImplementationArtifactsLocation() string {
	if p.BmadConfig != nil && p.BmadConfig.ImplementationArtifactsPath != "" {
		return path.Join(p.GetSpecLocation(), cleanRelativeSubpath(p.GetSpecLocation(), p.BmadConfig.ImplementationArtifactsPath))
	}
	val := p.GetGitHubSetting("implementationArtifactsLocation")
	if val != "" {
		return val
	}
	return path.Join(p.GetSpecLocation(), "implementation")
}

func (p *Project) GetTestingArtifactsLocation() string {
	if p.BmadConfig != nil && p.BmadConfig.TestingArtifactsPath != "" {
		return path.Join(p.GetSpecLocation(), cleanRelativeSubpath(p.GetSpecLocation(), p.BmadConfig.TestingArtifactsPath))
	}
	val := p.GetGitHubSetting("testingArtifactsLocation")
	if val != "" {
		return val
	}
	return path.Join(p.GetSpecLocation(), "test")
}

func (p *Project) MarshalJSON() ([]byte, error) {
	type Alias Project
	return json.Marshal(&struct {
		*Alias
		SpecLocation                    string `json:"specLocation"`
		PlanningArtifactsLocation       string `json:"planningArtifactsLocation"`
		ImplementationArtifactsLocation string `json:"implementationArtifactsLocation"`
		TestingArtifactsLocation        string `json:"testingArtifactsLocation"`
	}{
		Alias:                           (*Alias)(p),
		SpecLocation:                    p.GetSpecLocation(),
		PlanningArtifactsLocation:       p.GetPlanningArtifactsLocation(),
		ImplementationArtifactsLocation: p.GetImplementationArtifactsLocation(),
		TestingArtifactsLocation:        p.GetTestingArtifactsLocation(),
	})
}

type AccessRequest struct {
	ID            string `json:"id" firestore:"-"`
	ProjectId     string `json:"projectId" firestore:"projectId"`
	UserId        string `json:"userId" firestore:"userId"`
	UserName      string `json:"userName" firestore:"userName"`
	RequestDate   string `json:"requestDate" firestore:"requestDate"` // Or timestamp
	Status        string `json:"status" firestore:"status"`           // pending, approved, rejected
	RequestedRole string `json:"requestedRole" firestore:"requestedRole"`
}

// Global store instances to be initialized by the main application package.
var (
	ProjectStoreInstance ProjectStore
)

type ProjectStore interface {
	ListProjects(ctx context.Context, organizationID string) ([]Project, error)
	ListProjectsByUser(ctx context.Context, organizationID string, userId string) (myProjects []Project, browseProjects []Project, err error)
	GetProject(ctx context.Context, organizationID string, id string) (*Project, error)
	CreateProject(ctx context.Context, organizationID string, p *Project) error
	UpdateProject(ctx context.Context, organizationID string, p *Project) error
	DeleteProject(ctx context.Context, organizationID string, id string) error
	ReconcileMemberID(ctx context.Context, organizationID string, oldID string, newID string) error
}

type AccessRequestStore interface {
	CreateRequest(ctx context.Context, organizationID string, r *AccessRequest) error
	UpdateRequest(ctx context.Context, organizationID string, r *AccessRequest) error
	ListRequestsForProject(ctx context.Context, organizationID string, projectId string) ([]AccessRequest, error)
}

type FirestoreProjectStore struct {
	client *firestore.Client
}

func NewFirestoreProjectStore(client *firestore.Client) *FirestoreProjectStore {
	return &FirestoreProjectStore{client: client}
}

func (s *FirestoreProjectStore) ListProjects(ctx context.Context, organizationID string) ([]Project, error) {
	var projects []Project
	col := organizationCollection(organizationID, s.client, "projects")
	iter := col.Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var p Project
		if err := doc.DataTo(&p); err != nil {
			return nil, err
		}
		p.ID = doc.Ref.ID
		projects = append(projects, p)
	}
	return projects, nil
}

func (s *FirestoreProjectStore) ListProjectsByUser(ctx context.Context, organizationID string, userId string) ([]Project, []Project, error) {
	if userId == "" {
		return nil, nil, fmt.Errorf("userId cannot be empty")
	}

	var myProjects []Project
	var browseProjects []Project

	col := organizationCollection(organizationID, s.client, "projects")

	// 1. Query projects where the user is a member
	myIter := col.WherePath(firestore.FieldPath{"members", userId}, "!=", nil).Documents(ctx)
	for {
		doc, err := myIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, nil, err
		}
		var p Project
		if err := doc.DataTo(&p); err != nil {
			return nil, nil, err
		}
		p.ID = doc.Ref.ID
		myProjects = append(myProjects, p)
	}

	sort.Slice(myProjects, func(i, j int) bool {
		return myProjects[i].CreatedAt.After(myProjects[j].CreatedAt)
	})

	// 2. Query public projects
	browseIter := col.Where("visibility", "==", "public").Documents(ctx)
	for {
		doc, err := browseIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, nil, err
		}
		var p Project
		if err := doc.DataTo(&p); err != nil {
			return nil, nil, err
		}
		p.ID = doc.Ref.ID

		// Filter out if the user is already a member
		isMember := false
		if p.Members != nil && (*p.Members)[userId] != "" {
			isMember = true
		}

		if !isMember {
			browseProjects = append(browseProjects, p)
		}
	}

	sort.Slice(browseProjects, func(i, j int) bool {
		return browseProjects[i].CreatedAt.After(browseProjects[j].CreatedAt)
	})

	return myProjects, browseProjects, nil
}

func (s *FirestoreProjectStore) GetProject(ctx context.Context, organizationID string, id string) (*Project, error) {
	col := organizationCollection(organizationID, s.client, "projects")
	doc, err := col.Doc(id).Get(ctx)
	if err != nil {
		return nil, err
	}
	var p Project
	if err := doc.DataTo(&p); err != nil {
		return nil, err
	}
	p.ID = doc.Ref.ID
	return &p, nil
}

func (s *FirestoreProjectStore) CreateProject(ctx context.Context, organizationID string, p *Project) error {
	if p.CreatedAt.IsZero() {
		p.CreatedAt = time.Now()
	}
	col := organizationCollection(organizationID, s.client, "projects")
	var ref *firestore.DocumentRef
	var err error
	if p.ID == "" {
		ref, _, err = col.Add(ctx, p)
	} else {
		ref = col.Doc(p.ID)
		_, err = ref.Set(ctx, p)
	}
	if err != nil {
		return err
	}
	p.ID = ref.ID
	return nil
}

func (s *FirestoreProjectStore) UpdateProject(ctx context.Context, organizationID string, p *Project) error {
	if p.ID == "" {
		return fmt.Errorf("project ID is required for update")
	}
	col := organizationCollection(organizationID, s.client, "projects")
	_, err := col.Doc(p.ID).Set(ctx, p)
	return err
}

func (s *FirestoreProjectStore) DeleteProject(ctx context.Context, organizationID string, id string) error {
	batch := s.client.Batch()
	count := 0

	commitIfThreshold := func() error {
		if count >= 400 {
			_, err := batch.Commit(ctx)
			if err != nil {
				return err
			}
			batch = s.client.Batch()
			count = 0
		}
		return nil
	}

	// 1. Query and delete access_requests
	accessReqsCol := organizationCollection(organizationID, s.client, "access_requests")
	accessReqsIter := accessReqsCol.Where("projectId", "==", id).Documents(ctx)
	for {
		doc, err := accessReqsIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		batch.Delete(doc.Ref)
		count++
		if err := commitIfThreshold(); err != nil {
			return err
		}
	}

	// 2. Query and delete artifacts
	artifactsCol := organizationCollection(organizationID, s.client, "artifacts")
	artifactsIter := artifactsCol.Where("projectId", "==", id).Documents(ctx)
	for {
		doc, err := artifactsIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		batch.Delete(doc.Ref)
		count++
		if err := commitIfThreshold(); err != nil {
			return err
		}
	}

	// 3. Query and delete sessions and their subcollection messages
	sessionsCol := organizationCollection(organizationID, s.client, "sessions")
	sessionsIter := sessionsCol.Where("projectId", "==", id).Documents(ctx)
	for {
		sessDoc, err := sessionsIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}

		// Query and delete messages under this session
		messagesIter := sessDoc.Ref.Collection("messages").Documents(ctx)
		for {
			msgDoc, err := messagesIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return err
			}
			batch.Delete(msgDoc.Ref)
			count++
			if err := commitIfThreshold(); err != nil {
				return err
			}
		}

		// Delete the session itself
		batch.Delete(sessDoc.Ref)
		count++
		if err := commitIfThreshold(); err != nil {
			return err
		}
	}

	// 4. Query and delete user_project and their subcollection app_credentials
	userProjectsCol := organizationCollection(organizationID, s.client, "user_project")
	userProjectsIter := userProjectsCol.Where("projectId", "==", id).Documents(ctx)
	for {
		upDoc, err := userProjectsIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}

		// Query and delete all app_credentials under this user_project
		appCredsIter := upDoc.Ref.Collection("app_credentials").Documents(ctx)
		for {
			credDoc, err := appCredsIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return err
			}
			batch.Delete(credDoc.Ref)
			count++
			if err := commitIfThreshold(); err != nil {
				return err
			}
		}

		// Delete the user_project parent document itself
		batch.Delete(upDoc.Ref)
		count++
		if err := commitIfThreshold(); err != nil {
			return err
		}
	}

	// 5. Delete the project itself
	projectsCol := organizationCollection(organizationID, s.client, "projects")
	batch.Delete(projectsCol.Doc(id))
	count++

	// 6. Commit any remaining deletes
	if count > 0 {
		_, err := batch.Commit(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

// ReconcileMemberID replaces a temporary member key (email) with the permanent userID after they log in.
func (s *FirestoreProjectStore) ReconcileMemberID(ctx context.Context, organizationID string, oldID string, newID string) error {
	col := organizationCollection(organizationID, s.client, "projects")
	iter := col.WherePath(firestore.FieldPath{"members", oldID}, "!=", nil).Documents(ctx)
	defer iter.Stop()

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}

		var p Project
		if err := doc.DataTo(&p); err != nil {
			return err
		}
		p.ID = doc.Ref.ID

		if p.Members != nil {
			role, ok := (*p.Members)[oldID]
			if ok {
				(*p.Members)[newID] = role
				delete(*p.Members, oldID)

				_, err := doc.Ref.Set(ctx, p)
				if err != nil {
					return err
				}
				log.Printf("INFO: Reconciled project membership for project %s: replaced %s with %s (role: %s)", p.ID, oldID, newID, role)
			}
		}
	}
	return nil
}

type FirestoreAccessRequestStore struct {
	client *firestore.Client
}

func NewFirestoreAccessRequestStore(client *firestore.Client) *FirestoreAccessRequestStore {
	return &FirestoreAccessRequestStore{client: client}
}

func (s *FirestoreAccessRequestStore) CreateRequest(ctx context.Context, organizationID string, r *AccessRequest) error {
	col := organizationCollection(organizationID, s.client, "access_requests")
	ref, _, err := col.Add(ctx, r)
	if err != nil {
		return err
	}
	r.ID = ref.ID
	return nil
}

func (s *FirestoreAccessRequestStore) UpdateRequest(ctx context.Context, organizationID string, r *AccessRequest) error {
	if r.ID == "" {
		return fmt.Errorf("request ID is required")
	}
	col := organizationCollection(organizationID, s.client, "access_requests")
	_, err := col.Doc(r.ID).Set(ctx, r)
	return err
}

func (s *FirestoreAccessRequestStore) ListRequestsForProject(ctx context.Context, organizationID string, projectId string) ([]AccessRequest, error) {
	var requests []AccessRequest
	col := organizationCollection(organizationID, s.client, "access_requests")
	iter := col.Where("projectId", "==", projectId).Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var r AccessRequest
		if err := doc.DataTo(&r); err != nil {
			return nil, err
		}
		r.ID = doc.Ref.ID
		requests = append(requests, r)
	}
	return requests, nil
}
