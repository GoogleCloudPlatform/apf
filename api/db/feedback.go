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
	"fmt"
	"sync"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/google/uuid"
)

var ErrInvalidFeedback = errors.New("invalid feedback specification")

// Feedback represents a user feedback document stored in Firestore.
type Feedback struct {
	ID             string    `json:"id" firestore:"id"`
	Type           string    `json:"type" firestore:"type"`                     // "bug" | "feature_request" | "general"
	Subject        string    `json:"subject" firestore:"subject"`               // short summary
	Message        string    `json:"message" firestore:"message"`               // detailed description
	UserEmail      string    `json:"userEmail" firestore:"userEmail"`           // submitter email
	OrganizationID string    `json:"organizationId,omitempty" firestore:"organizationId,omitempty"` // optional org ID
	ProjectID      string    `json:"projectId,omitempty" firestore:"projectId,omitempty"`         // optional project ID
	CreatedAt      time.Time `json:"createdAt" firestore:"createdAt"`
}

// FeedbackStore defines storage operations for user feedback.
type FeedbackStore interface {
	CreateFeedback(ctx context.Context, feedback *Feedback) (*Feedback, error)
	GetFeedback(ctx context.Context, feedbackID string) (*Feedback, error)
}

// FirestoreFeedbackStore implements FeedbackStore backed by Firestore.
type FirestoreFeedbackStore struct {
	client *firestore.Client
}

// NewFirestoreFeedbackStore creates a new FirestoreFeedbackStore.
func NewFirestoreFeedbackStore(client *firestore.Client) *FirestoreFeedbackStore {
	return &FirestoreFeedbackStore{client: client}
}

// CreateFeedback persists a new Feedback document to the root "feedback" collection.
func (s *FirestoreFeedbackStore) CreateFeedback(ctx context.Context, feedback *Feedback) (*Feedback, error) {
	if feedback == nil || feedback.Type == "" || feedback.Subject == "" || feedback.Message == "" {
		return nil, ErrInvalidFeedback
	}
	if s == nil || s.client == nil {
		return nil, errors.New("firestore store or client is nil")
	}

	fbCopy := *feedback
	if fbCopy.ID == "" {
		fbCopy.ID = uuid.New().String()
	}
	if fbCopy.CreatedAt.IsZero() {
		fbCopy.CreatedAt = time.Now()
	}

	docRef := s.client.Collection("feedback").Doc(fbCopy.ID)
	_, err := docRef.Set(ctx, map[string]interface{}{
		"id":             fbCopy.ID,
		"type":           fbCopy.Type,
		"subject":        fbCopy.Subject,
		"message":        fbCopy.Message,
		"userEmail":      fbCopy.UserEmail,
		"organizationId": fbCopy.OrganizationID,
		"projectId":      fbCopy.ProjectID,
		"createdAt":      fbCopy.CreatedAt,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to save feedback %s: %w", fbCopy.ID, err)
	}

	return &fbCopy, nil
}

// GetFeedback fetches a feedback document by ID.
func (s *FirestoreFeedbackStore) GetFeedback(ctx context.Context, feedbackID string) (*Feedback, error) {
	if feedbackID == "" {
		return nil, errors.New("feedback ID is required")
	}
	if s.client == nil {
		return nil, errors.New("firestore client is nil")
	}

	doc, err := s.client.Collection("feedback").Doc(feedbackID).Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get feedback %s: %w", feedbackID, err)
	}

	var fb Feedback
	if err := doc.DataTo(&fb); err != nil {
		return nil, fmt.Errorf("failed to parse feedback %s: %w", feedbackID, err)
	}
	fb.ID = doc.Ref.ID
	return &fb, nil
}

// MockFeedbackStore provides an in-memory implementation of FeedbackStore for unit testing.
type MockFeedbackStore struct {
	mu       sync.RWMutex
	feedback map[string]*Feedback
}

// NewMockFeedbackStore creates a new in-memory MockFeedbackStore.
func NewMockFeedbackStore() *MockFeedbackStore {
	return &MockFeedbackStore{
		feedback: make(map[string]*Feedback),
	}
}

func (m *MockFeedbackStore) CreateFeedback(ctx context.Context, feedback *Feedback) (*Feedback, error) {
	if feedback == nil || feedback.Type == "" || feedback.Subject == "" || feedback.Message == "" {
		return nil, ErrInvalidFeedback
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	fbCopy := *feedback
	if fbCopy.ID == "" {
		fbCopy.ID = fmt.Sprintf("fb-%s", uuid.New().String()[:8])
	}
	if fbCopy.CreatedAt.IsZero() {
		fbCopy.CreatedAt = time.Now()
	}

	m.feedback[fbCopy.ID] = &fbCopy
	return &fbCopy, nil
}

func (m *MockFeedbackStore) GetFeedback(ctx context.Context, feedbackID string) (*Feedback, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	fb, exists := m.feedback[feedbackID]
	if !exists {
		return nil, errors.New("feedback not found")
	}
	fbCopy := *fb
	return &fbCopy, nil
}
