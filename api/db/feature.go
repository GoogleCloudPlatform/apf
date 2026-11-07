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
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var ErrFeatureNotFound = errors.New("feature document not found")

// Feature represents an application-level feature flag document.
type Feature struct {
	ID         string                 `json:"id" firestore:"-"`
	Enabled    bool                   `json:"enabled" firestore:"enabled"`
	Properties map[string]interface{} `json:"properties" firestore:"properties"`
	UpdatedAt  time.Time              `json:"updatedAt" firestore:"updatedAt"`
}

// FeatureStore defines operations for managing feature flag configurations.
type FeatureStore interface {
	GetFeature(ctx context.Context, featureID string) (*Feature, error)
	SaveFeature(ctx context.Context, feature *Feature) error
	ListFeatures(ctx context.Context) ([]Feature, error)
	DeleteFeature(ctx context.Context, featureID string) error
}

// FirestoreFeatureStore implements FeatureStore backed by Google Cloud Firestore.
type FirestoreFeatureStore struct {
	client *firestore.Client
}

// NewFirestoreFeatureStore creates a new FirestoreFeatureStore.
func NewFirestoreFeatureStore(client *firestore.Client) *FirestoreFeatureStore {
	return &FirestoreFeatureStore{client: client}
}

// GetFeature fetches a feature document from the "features" Firestore collection.
func (s *FirestoreFeatureStore) GetFeature(ctx context.Context, featureID string) (*Feature, error) {
	if featureID == "" {
		return nil, ErrFeatureNotFound
	}
	if s.client == nil {
		return nil, errors.New("firestore client is nil")
	}
	doc, err := s.client.Collection("features").Doc(featureID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, ErrFeatureNotFound
		}
		return nil, fmt.Errorf("failed to get feature %s: %w", featureID, err)
	}

	var feature Feature
	if err := doc.DataTo(&feature); err != nil {
		return nil, fmt.Errorf("failed to parse feature %s: %w", featureID, err)
	}
	feature.ID = doc.Ref.ID
	if feature.Properties == nil {
		feature.Properties = make(map[string]interface{})
	}
	return &feature, nil
}

// SaveFeature stores or updates a feature document in the "features" collection.
func (s *FirestoreFeatureStore) SaveFeature(ctx context.Context, feature *Feature) error {
	if feature == nil || feature.ID == "" {
		return fmt.Errorf("invalid feature specification")
	}
	if s.client == nil {
		return errors.New("firestore client is nil")
	}
	feature.UpdatedAt = time.Now()
	if feature.Properties == nil {
		feature.Properties = make(map[string]interface{})
	}

	_, err := s.client.Collection("features").Doc(feature.ID).Set(ctx, map[string]interface{}{
		"enabled":    feature.Enabled,
		"properties": feature.Properties,
		"updatedAt":  feature.UpdatedAt,
	})
	if err != nil {
		return fmt.Errorf("failed to save feature %s: %w", feature.ID, err)
	}
	return nil
}

// ListFeatures retrieves all feature documents from the "features" collection.
func (s *FirestoreFeatureStore) ListFeatures(ctx context.Context) ([]Feature, error) {
	if s.client == nil {
		return nil, errors.New("firestore client is nil")
	}
	iter := s.client.Collection("features").Documents(ctx)
	defer iter.Stop()

	var features []Feature
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate features: %w", err)
		}
		var f Feature
		if err := doc.DataTo(&f); err != nil {
			continue
		}
		f.ID = doc.Ref.ID
		if f.Properties == nil {
			f.Properties = make(map[string]interface{})
		}
		features = append(features, f)
	}
	return features, nil
}

// DeleteFeature deletes a feature document by ID.
func (s *FirestoreFeatureStore) DeleteFeature(ctx context.Context, featureID string) error {
	if featureID == "" {
		return ErrFeatureNotFound
	}
	if s.client == nil {
		return errors.New("firestore client is nil")
	}
	_, err := s.client.Collection("features").Doc(featureID).Delete(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete feature %s: %w", featureID, err)
	}
	return nil
}

// MockFeatureStore provides an in-memory implementation of FeatureStore for testing.
type MockFeatureStore struct {
	mu       sync.RWMutex
	features map[string]*Feature
}

// NewMockFeatureStore creates a new in-memory MockFeatureStore.
func NewMockFeatureStore() *MockFeatureStore {
	return &MockFeatureStore{
		features: make(map[string]*Feature),
	}
}

func deepCopyMap(m map[string]interface{}) map[string]interface{} {
	if m == nil {
		return make(map[string]interface{})
	}
	res := make(map[string]interface{}, len(m))
	for k, v := range m {
		if nestedMap, ok := v.(map[string]interface{}); ok {
			res[k] = deepCopyMap(nestedMap)
		} else {
			res[k] = v
		}
	}
	return res
}

func (m *MockFeatureStore) GetFeature(ctx context.Context, featureID string) (*Feature, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	f, exists := m.features[featureID]
	if !exists {
		return nil, ErrFeatureNotFound
	}
	return &Feature{
		ID:         f.ID,
		Enabled:    f.Enabled,
		Properties: deepCopyMap(f.Properties),
		UpdatedAt:  f.UpdatedAt,
	}, nil
}

func (m *MockFeatureStore) SaveFeature(ctx context.Context, feature *Feature) error {
	if feature == nil || feature.ID == "" {
		return fmt.Errorf("invalid feature specification")
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	updatedAt := feature.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = time.Now()
	}

	m.features[feature.ID] = &Feature{
		ID:         feature.ID,
		Enabled:    feature.Enabled,
		Properties: deepCopyMap(feature.Properties),
		UpdatedAt:  updatedAt,
	}
	return nil
}

func (m *MockFeatureStore) ListFeatures(ctx context.Context) ([]Feature, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []Feature
	for _, f := range m.features {
		result = append(result, Feature{
			ID:         f.ID,
			Enabled:    f.Enabled,
			Properties: deepCopyMap(f.Properties),
			UpdatedAt:  f.UpdatedAt,
		})
	}
	return result, nil
}

func (m *MockFeatureStore) DeleteFeature(ctx context.Context, featureID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.features[featureID]; !exists {
		return ErrFeatureNotFound
	}
	delete(m.features, featureID)
	return nil
}
