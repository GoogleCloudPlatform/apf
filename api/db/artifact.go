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
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

type ArtifactType string

const (
	ArtifactTypeBrainstorm     ArtifactType = "brainstorming-session-doc"
	ArtifactTypePrd            ArtifactType = "product-req-doc"
	ArtifactTypeUxDesign       ArtifactType = "ux-design-doc"
	ArtifactTypeResearch       ArtifactType = "research-doc"
	ArtifactTypeProductBrief   ArtifactType = "product-brief-doc"
	ArtifactTypePrfaq          ArtifactType = "prfaq-doc"
	ArtifactTypeArchitecture   ArtifactType = "architecture-doc"
	ArtifactTypeEpicsStories   ArtifactType = "epics-stories-doc"
	ArtifactTypeReadiness      ArtifactType = "readiness-report-doc"
	ArtifactTypeSprintPlan     ArtifactType = "sprint-plan-doc"
	ArtifactTypeStory          ArtifactType = "story-doc"
	ArtifactTypeRetrospective  ArtifactType = "retrospective-doc"
	ArtifactTypeInvestigation  ArtifactType = "investigation-doc"
	ArtifactTypeTestSummary    ArtifactType = "test-summary-doc"
	ArtifactTypeProjectContext ArtifactType = "project-context-doc"
	ArtifactTypeProjectStatus  ArtifactType = "project-status-doc"
)

var storyFilePattern = regexp.MustCompile(`^[0-9]+-[0-9]+-.*\.(md|markdown|yaml|yml|json|txt|html)$`)

// DetermineArtifactType returns the ArtifactType based on path substrings.
func DetermineArtifactType(filePath string) ArtifactType {
	pathLower := filepath.ToSlash(strings.ToLower(filepath.Clean(filePath)))
	baseName := strings.ToLower(filepath.Base(pathLower))

	if strings.Contains(pathLower, "project-context") || strings.Contains(pathLower, "agents.") || baseName == "agents" {
		return ArtifactTypeProjectContext
	}
	if strings.Contains(pathLower, "project-status") {
		return ArtifactTypeProjectStatus
	}
	if strings.Contains(pathLower, "reconcile-brain") ||
		strings.Contains(pathLower, "/prd/") ||
		strings.HasPrefix(pathLower, "prd/") ||
		strings.Contains(pathLower, "\\prd\\") {
		return ArtifactTypePrd
	}
	if strings.Contains(pathLower, "brief") {
		return ArtifactTypeProductBrief
	}
	if strings.Contains(pathLower, "brainstorm") {
		return ArtifactTypeBrainstorm
	}
	if strings.Contains(pathLower, "research") || strings.Contains(pathLower, "recon") {
		return ArtifactTypeResearch
	}
	if strings.Contains(pathLower, "prfaq") {
		return ArtifactTypePrfaq
	}
	if strings.Contains(pathLower, "prd") {
		return ArtifactTypePrd
	}
	if strings.Contains(pathLower, "ux") || strings.Contains(pathLower, "design") {
		return ArtifactTypeUxDesign
	}
	if strings.Contains(pathLower, "architecture") ||
		strings.Contains(pathLower, "/adr/") ||
		strings.Contains(pathLower, "/adrs/") ||
		strings.HasPrefix(pathLower, "adr/") ||
		strings.HasPrefix(pathLower, "adrs/") ||
		strings.HasPrefix(baseName, "adr-") {
		return ArtifactTypeArchitecture
	}
	if strings.Contains(pathLower, "readiness") {
		return ArtifactTypeReadiness
	}
	if strings.Contains(pathLower, "retro") {
		return ArtifactTypeRetrospective
	}
	if strings.Contains(pathLower, "investigation") || strings.Contains(baseName, "case-") {
		return ArtifactTypeInvestigation
	}
	if strings.Contains(pathLower, "test-summary") ||
		strings.Contains(pathLower, "test-review") ||
		strings.Contains(pathLower, "traceability") ||
		strings.Contains(pathLower, "nfr-audit") {
		return ArtifactTypeTestSummary
	}
	if strings.Contains(pathLower, "epics-and-stories") ||
		strings.Contains(pathLower, "epics_and_stories") ||
		baseName == "epics.md" ||
		strings.HasPrefix(baseName, "epics.") ||
		(strings.Contains(pathLower, "epic") && !strings.Contains(pathLower, "story") && !strings.Contains(pathLower, "stories") && !strings.Contains(pathLower, "retro")) {
		return ArtifactTypeEpicsStories
	}
	if strings.Contains(pathLower, "stories") || strings.Contains(pathLower, "story") || storyFilePattern.MatchString(baseName) {
		return ArtifactTypeStory
	}
	if strings.Contains(pathLower, "sprint") {
		return ArtifactTypeSprintPlan
	}
	return ""
}

// DetermineArtifactPhaseForProject determines the phase for a file path.
// Files in the implementation folder return "IMPLEMENTATION". Other files (including those in the planning folder)
// are categorized by document type (e.g. brief/research/brainstorm/prfaq/analysis -> ANALYSIS, prd/ux/spec/test-plan -> PLANNING, architecture/epics -> SOLUTIONING).
func DetermineArtifactPhaseForProject(filePath string, project *Project) string {
	if project != nil {
		filePathClean := filepath.ToSlash(strings.ToLower(filepath.Clean(filePath)))

		if implLoc := project.GetImplementationArtifactsLocation(); implLoc != "" {
			implClean := filepath.ToSlash(strings.ToLower(filepath.Clean(implLoc)))
			if strings.Contains(filePathClean, implClean) {
				return "IMPLEMENTATION"
			}
		}
	}

	return DetermineArtifactPhase(filePath)
}

// DetermineArtifactPhase infers the lifecycle Phase ("ONBOARDING", "ANALYSIS", "PLANNING", "SOLUTIONING", "IMPLEMENTATION") based on file path substrings and tokens.
func DetermineArtifactPhase(filePath string) string {
	filePathLower := filepath.ToSlash(strings.ToLower(filepath.Clean(filePath)))
	baseName := filepath.Base(filePathLower)

	// 1. ONBOARDING
	if strings.Contains(filePathLower, "project-context") ||
		strings.Contains(filePathLower, "project-status") ||
		strings.Contains(filePathLower, "agents.") ||
		baseName == "agents" {
		return "ONBOARDING"
	}

	// 2. Specific PRD / Reconcile overrides (e.g. reconcile-brainstorming.md, /prd/...)
	if strings.Contains(filePathLower, "reconcile-brain") ||
		strings.Contains(filePathLower, "/prd/") ||
		strings.HasPrefix(filePathLower, "prd/") ||
		strings.Contains(filePathLower, "\\prd\\") {
		return "PLANNING"
	}

	// 3. ANALYSIS
	if strings.Contains(filePathLower, "brief") ||
		strings.Contains(filePathLower, "brainstorm") ||
		strings.Contains(filePathLower, "research") ||
		strings.Contains(filePathLower, "recon") ||
		strings.Contains(filePathLower, "prfaq") ||
		strings.Contains(filePathLower, "forged-idea") ||
		strings.Contains(filePathLower, "analysis") {
		return "ANALYSIS"
	}

	// 4. SOLUTIONING
	if strings.Contains(filePathLower, "architecture") ||
		strings.Contains(filePathLower, "/adr/") ||
		strings.Contains(filePathLower, "/adrs/") ||
		strings.HasPrefix(filePathLower, "adr/") ||
		strings.HasPrefix(filePathLower, "adrs/") ||
		strings.HasPrefix(baseName, "adr-") ||
		strings.Contains(filePathLower, "epics-and-stories") ||
		strings.Contains(filePathLower, "epics_and_stories") ||
		(strings.Contains(filePathLower, "epic") && !strings.Contains(filePathLower, "story") && !strings.Contains(filePathLower, "stories") && !strings.Contains(filePathLower, "retro")) ||
		strings.Contains(filePathLower, "readiness") ||
		strings.Contains(filePathLower, "solutioning") {
		return "SOLUTIONING"
	}

	// 5. IMPLEMENTATION
	if strings.Contains(filePathLower, "implementation") ||
		strings.Contains(filePathLower, "sprint") ||
		strings.Contains(filePathLower, "retro") ||
		strings.Contains(filePathLower, "investigation") ||
		strings.Contains(baseName, "case-") ||
		strings.Contains(filePathLower, "stories") ||
		strings.Contains(filePathLower, "story") ||
		storyFilePattern.MatchString(baseName) ||
		strings.Contains(filePathLower, "test-strategy") ||
		strings.Contains(filePathLower, "test-review") ||
		strings.Contains(filePathLower, "traceability") ||
		strings.Contains(filePathLower, "nfr-audit") ||
		strings.Contains(filePathLower, "code-reviews") {
		return "IMPLEMENTATION"
	}

	// 6. PLANNING
	if strings.Contains(filePathLower, "prd") ||
		strings.Contains(filePathLower, "ux") ||
		strings.Contains(filePathLower, "design") ||
		strings.Contains(filePathLower, "spec") ||
		strings.Contains(filePathLower, "test-plan") {
		return "PLANNING"
	}

	return "PLANNING"
}

type ArtifactStatus string

const (
	ArtifactStatusNotStarted      ArtifactStatus = "Not Started"
	ArtifactStatusInProgress      ArtifactStatus = "In Progress"
	ArtifactStatusStagedForReview ArtifactStatus = "Staged for Review"
	ArtifactStatusComplete        ArtifactStatus = "Complete"
)

func (s ArtifactStatus) Validate() error {
	switch s {
	case ArtifactStatusNotStarted, ArtifactStatusInProgress, ArtifactStatusStagedForReview, ArtifactStatusComplete:
		return nil
	default:
		return fmt.Errorf("invalid artifact status: %q", string(s))
	}
}

func (s *ArtifactStatus) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	status := ArtifactStatus(str)
	if err := status.Validate(); err != nil {
		return err
	}
	*s = status
	return nil
}

func (s *ArtifactStatus) UnmarshalText(text []byte) error {
	status := ArtifactStatus(text)
	if err := status.Validate(); err != nil {
		return err
	}
	*s = status
	return nil
}

// Artifact represents a workspace file instance.
type Artifact struct {
	ID           string         `json:"id" firestore:"-"`
	ProjectId    string         `json:"projectId" firestore:"projectId"`
	WorkflowID   string         `json:"workflowId,omitempty" firestore:"workflowId,omitempty"`
	Title        string         `json:"title,omitempty" firestore:"title,omitempty"`
	Path         string         `json:"path,omitempty" firestore:"path,omitempty"`
	Phase        string         `json:"phase,omitempty" firestore:"phase,omitempty"`
	Status       ArtifactStatus `json:"status,omitempty" firestore:"status,omitempty"`
	Owner        string         `json:"owner,omitempty" firestore:"owner,omitempty"`
	Content      string         `json:"content,omitempty" firestore:"content,omitempty"`
	Required     bool           `json:"required,omitempty" firestore:"required,omitempty"`
	IsCustom     bool           `json:"isCustom,omitempty" firestore:"isCustom,omitempty"`
	ArtifactType ArtifactType   `json:"artifactType,omitempty" firestore:"artifactType,omitempty"`

	// Session artifact fields (Story 18.1)
	SessionId   string    `json:"sessionId,omitempty" firestore:"sessionId,omitempty"`
	Name        string    `json:"name,omitempty" firestore:"name,omitempty"`
	MimeType    string    `json:"mimeType,omitempty" firestore:"mimeType,omitempty"`
	Size        int64     `json:"size,omitempty" firestore:"size,omitempty"`
	StorageType string    `json:"storageType,omitempty" firestore:"storageType,omitempty"`
	StoragePath string    `json:"storagePath,omitempty" firestore:"storagePath,omitempty"`
	CreatedAt   time.Time `json:"createdAt,omitempty" firestore:"createdAt,omitempty"`
}

// ArtifactStore interface for artifact data operations.
type ArtifactStore interface {
	ListArtifactsForProject(ctx context.Context, organizationID string, projectId string) ([]Artifact, error)
	UpdateArtifact(ctx context.Context, organizationID string, a *Artifact) error
	CreateArtifact(ctx context.Context, organizationID string, a *Artifact) error
	DeleteArtifact(ctx context.Context, organizationID string, artifactId string) error
	SyncArtifacts(ctx context.Context, organizationID string, projectId string, discovered []Artifact) (addedCount int, deletedCount int, err error)
}

type FirestoreArtifactStore struct {
	client *firestore.Client
}

func NewFirestoreArtifactStore(client *firestore.Client) *FirestoreArtifactStore {
	return &FirestoreArtifactStore{client: client}
}

func (s *FirestoreArtifactStore) ListArtifactsForProject(ctx context.Context, organizationID string, projectId string) ([]Artifact, error) {
	artifacts := []Artifact{}
	col := organizationCollection(organizationID, s.client, "artifacts")
	iter := col.Where("projectId", "==", projectId).Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var a Artifact
		if err := doc.DataTo(&a); err != nil {
			return nil, err
		}
		a.ID = doc.Ref.ID
		artifacts = append(artifacts, a)
	}
	return artifacts, nil
}

func (s *FirestoreArtifactStore) UpdateArtifact(ctx context.Context, organizationID string, a *Artifact) error {
	if a.ID == "" {
		return fmt.Errorf("artifact ID is required for update")
	}
	if a.Status != "" {
		if err := a.Status.Validate(); err != nil {
			return err
		}
	}
	col := organizationCollection(organizationID, s.client, "artifacts")
	_, err := col.Doc(a.ID).Set(ctx, a)
	return err
}

func (s *FirestoreArtifactStore) DeleteArtifact(ctx context.Context, organizationID string, artifactId string) error {
	if artifactId == "" {
		return fmt.Errorf("artifact ID is required for delete")
	}
	col := organizationCollection(organizationID, s.client, "artifacts")
	_, err := col.Doc(artifactId).Delete(ctx)
	return err
}

func (s *FirestoreArtifactStore) CreateArtifact(ctx context.Context, organizationID string, a *Artifact) error {
	if a.Status != "" {
		if err := a.Status.Validate(); err != nil {
			return err
		}
	}
	col := organizationCollection(organizationID, s.client, "artifacts")
	var ref *firestore.DocumentRef
	var err error
	if a.ID == "" {
		ref, _, err = col.Add(ctx, a)
	} else {
		ref = col.Doc(a.ID)
		_, err = ref.Set(ctx, a)
	}
	if err != nil {
		return err
	}
	a.ID = ref.ID
	return nil
}

// ArtifactsMatch checks if two Artifact instances represent the same underlying file/specification.
func ArtifactsMatch(a, b Artifact) bool {
	if a.ID != "" && b.ID != "" && a.ID == b.ID {
		return true
	}
	if a.Path != "" && b.Path != "" && strings.EqualFold(filepath.Clean(a.Path), filepath.Clean(b.Path)) {
		return true
	}
	aBase := strings.ToLower(filepath.Base(a.Path))
	if aBase == "." || aBase == "/" || aBase == "" {
		aBase = strings.ToLower(strings.TrimSpace(a.Title))
	}
	bBase := strings.ToLower(filepath.Base(b.Path))
	if bBase == "." || bBase == "/" || bBase == "" {
		bBase = strings.ToLower(strings.TrimSpace(b.Title))
	}
	if aBase != "" && aBase == bBase && !isGenericFilename(aBase) {
		return true
	}
	return false
}

func isGenericFilename(filename string) bool {
	switch filename {
	case "index.md", "readme.md":
		return true
	}
	return false
}

func (s *FirestoreArtifactStore) SyncArtifacts(ctx context.Context, organizationID string, projectId string, discovered []Artifact) (addedCount int, deletedCount int, err error) {
	existing, err := s.ListArtifactsForProject(ctx, organizationID, projectId)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to list existing artifacts: %w", err)
	}

	col := organizationCollection(organizationID, s.client, "artifacts")
	batch := s.client.Batch()
	count := 0

	reconciled := make([]bool, len(discovered))

	// 1. Reconcile existing records against discovered artifacts
	for _, extArt := range existing {
		matchedIndex := -1
		for i, discArt := range discovered {
			if !reconciled[i] && ArtifactsMatch(extArt, discArt) {
				matchedIndex = i
				break
			}
		}

		if matchedIndex >= 0 {
			// Found a matching discovered artifact for this existing record.
			discArt := discovered[matchedIndex]
			reconciled[matchedIndex] = true

			// Update existing canonical document in-place
			updated := extArt
			if updated.Path == "" || updated.Path != discArt.Path {
				updated.Path = discArt.Path
			}
			if updated.Title == "" {
				updated.Title = discArt.Title
			}
			updated.Phase = discArt.Phase
			updated.ArtifactType = discArt.ArtifactType
			updated.IsCustom = true
			if updated.Status == "" || updated.Status == ArtifactStatusNotStarted {
				updated.Status = discArt.Status
			}

			docRef := col.Doc(extArt.ID)
			batch.Set(docRef, updated)
			count++
		} else {
			// Check if any other existing record already matched this discovered artifact (duplicate existing record!)
			alreadyMatched := false
			for i, discArt := range discovered {
				if reconciled[i] && ArtifactsMatch(extArt, discArt) {
					alreadyMatched = true
					break
				}
			}

			if alreadyMatched || extArt.IsCustom {
				docRef := col.Doc(extArt.ID)
				batch.Delete(docRef)
				count++
				deletedCount++
			}
		}
	}

	// 2. Insert any remaining discovered artifacts that had no matching existing record
	for i, discArt := range discovered {
		if !reconciled[i] {
			var docRef *firestore.DocumentRef
			if discArt.ID != "" {
				docRef = col.Doc(discArt.ID)
			} else {
				docRef = col.NewDoc()
				discArt.ID = docRef.ID
			}
			batch.Set(docRef, discArt)
			count++
			addedCount++
		}
	}

	if count > 0 {
		_, err := batch.Commit(ctx)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to commit firestore batch sync: %w", err)
		}
	}

	return addedCount, deletedCount, nil
}
