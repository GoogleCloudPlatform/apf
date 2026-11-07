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
	"sync"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AutojoinRule represents an automatic organization mapping rule.
type AutojoinRule struct {
	ID             string     `json:"id" firestore:"-"`
	Domain         string     `json:"domain" firestore:"domain"`                   // e.g., "google.com"
	OrganizationID string     `json:"organization_id" firestore:"organization_id"` // target organization ID
	DefaultRoles   []UserRole `json:"default_roles" firestore:"default_roles"`     // default roles assigned to new users
	IsActive       bool       `json:"is_active" firestore:"is_active"`             // toggle to enable/disable rule
	CreatedAt      time.Time  `json:"created_at" firestore:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" firestore:"updated_at"`
}

// AutojoinStore defines database operations for autojoin configurations.
type AutojoinStore interface {
	ListRules(ctx context.Context) ([]AutojoinRule, error)
	GetRule(ctx context.Context, id string) (*AutojoinRule, error)
	CreateRule(ctx context.Context, rule *AutojoinRule) (string, error)
	UpdateRule(ctx context.Context, id string, rule *AutojoinRule) error
	DeleteRule(ctx context.Context, id string) error
	FindActiveRuleByDomain(ctx context.Context, domain string) (*AutojoinRule, error)
}

// FirestoreAutojoinStore implements AutojoinStore using Google Cloud Firestore.
type FirestoreAutojoinStore struct {
	client      *firestore.Client
	cacheMu     sync.RWMutex
	activeCache map[string]*AutojoinRule
	cacheLoaded bool
}

// NewFirestoreAutojoinStore creates a new FirestoreAutojoinStore.
func NewFirestoreAutojoinStore(client *firestore.Client) *FirestoreAutojoinStore {
	return &FirestoreAutojoinStore{
		client:      client,
		activeCache: make(map[string]*AutojoinRule),
	}
}

// ListRules retrieves all autojoin rules chronologically.
func (s *FirestoreAutojoinStore) ListRules(ctx context.Context) ([]AutojoinRule, error) {
	docs, err := s.client.Collection("autojoin_rules").OrderBy("created_at", firestore.Asc).Documents(ctx).GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to list autojoin rules: %w", err)
	}

	rules := make([]AutojoinRule, 0, len(docs))
	for _, doc := range docs {
		var rule AutojoinRule
		if err := doc.DataTo(&rule); err != nil {
			return nil, fmt.Errorf("failed to decode autojoin rule: %w", err)
		}
		rule.ID = doc.Ref.ID
		rules = append(rules, rule)
	}
	return rules, nil
}

// GetRule retrieves a specific autojoin rule by document ID.
func (s *FirestoreAutojoinStore) GetRule(ctx context.Context, id string) (*AutojoinRule, error) {
	if id == "" {
		return nil, fmt.Errorf("rule ID cannot be empty")
	}

	doc, err := s.client.Collection("autojoin_rules").Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get autojoin rule: %w", err)
	}

	var rule AutojoinRule
	if err := doc.DataTo(&rule); err != nil {
		return nil, fmt.Errorf("failed to decode autojoin rule: %w", err)
	}
	rule.ID = doc.Ref.ID
	return &rule, nil
}

// CreateRule inserts a new autojoin rule into Firestore.
func (s *FirestoreAutojoinStore) CreateRule(ctx context.Context, rule *AutojoinRule) (string, error) {
	if rule.Domain == "" {
		return "", fmt.Errorf("domain cannot be empty")
	}
	if rule.OrganizationID == "" {
		return "", fmt.Errorf("organization_id cannot be empty")
	}
	if len(rule.DefaultRoles) == 0 {
		return "", fmt.Errorf("default_roles cannot be empty")
	}

	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()

	docRef, _, err := s.client.Collection("autojoin_rules").Add(ctx, rule)
	if err != nil {
		return "", fmt.Errorf("failed to create autojoin rule: %w", err)
	}

	// Invalidate cache
	s.cacheMu.Lock()
	s.cacheLoaded = false
	s.cacheMu.Unlock()

	return docRef.ID, nil
}

// UpdateRule modifies an existing autojoin rule document.
func (s *FirestoreAutojoinStore) UpdateRule(ctx context.Context, id string, rule *AutojoinRule) error {
	if id == "" {
		return fmt.Errorf("rule ID cannot be empty")
	}
	if rule.Domain == "" {
		return fmt.Errorf("domain cannot be empty")
	}
	if rule.OrganizationID == "" {
		return fmt.Errorf("organization_id cannot be empty")
	}
	if len(rule.DefaultRoles) == 0 {
		return fmt.Errorf("default_roles cannot be empty")
	}

	docRef := s.client.Collection("autojoin_rules").Doc(id)

	// Retrieve the existing document to verify existence and preserve CreatedAt
	doc, err := docRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return ErrNotFound
		}
		return fmt.Errorf("failed to retrieve autojoin rule: %w", err)
	}

	var existing AutojoinRule
	if err := doc.DataTo(&existing); err != nil {
		return fmt.Errorf("failed to decode existing autojoin rule: %w", err)
	}

	// Preserve CreatedAt
	rule.CreatedAt = existing.CreatedAt
	rule.UpdatedAt = time.Now()

	_, err = docRef.Set(ctx, rule)
	if err != nil {
		return fmt.Errorf("failed to update autojoin rule: %w", err)
	}

	// Invalidate cache
	s.cacheMu.Lock()
	s.cacheLoaded = false
	s.cacheMu.Unlock()

	return nil
}

// DeleteRule removes a rule from Firestore.
func (s *FirestoreAutojoinStore) DeleteRule(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("rule ID cannot be empty")
	}

	_, err := s.client.Collection("autojoin_rules").Doc(id).Delete(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete autojoin rule: %w", err)
	}

	// Invalidate cache
	s.cacheMu.Lock()
	s.cacheLoaded = false
	s.cacheMu.Unlock()

	return nil
}

// FindActiveRuleByDomain queries for an enabled rule matching the exact domain (utilizes in-memory cache).
func (s *FirestoreAutojoinStore) FindActiveRuleByDomain(ctx context.Context, domain string) (*AutojoinRule, error) {
	if domain == "" {
		return nil, fmt.Errorf("domain cannot be empty")
	}

	if err := s.loadCache(ctx); err != nil {
		// Fallback to direct Firestore query if cache fails to load
		return s.findActiveRuleByDomainDirect(ctx, domain)
	}

	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()

	rule, ok := s.activeCache[domain]
	if !ok {
		return nil, ErrNotFound
	}

	// Return a copy to prevent external mutation of cached structures
	copyRule := *rule
	return &copyRule, nil
}

func (s *FirestoreAutojoinStore) findActiveRuleByDomainDirect(ctx context.Context, domain string) (*AutojoinRule, error) {
	query := s.client.Collection("autojoin_rules").
		Where("domain", "==", domain).
		Where("is_active", "==", true).
		Limit(1)

	iter := query.Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if err == iterator.Done {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query active autojoin rule: %w", err)
	}

	var rule AutojoinRule
	if err := doc.DataTo(&rule); err != nil {
		return nil, fmt.Errorf("failed to decode active autojoin rule: %w", err)
	}
	rule.ID = doc.Ref.ID
	return &rule, nil
}

func (s *FirestoreAutojoinStore) loadCache(ctx context.Context) error {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	if s.cacheLoaded {
		return nil
	}

	docs, err := s.client.Collection("autojoin_rules").Where("is_active", "==", true).Documents(ctx).GetAll()
	if err != nil {
		return err
	}

	s.activeCache = make(map[string]*AutojoinRule)
	for _, doc := range docs {
		var rule AutojoinRule
		if err := doc.DataTo(&rule); err != nil {
			return err
		}
		rule.ID = doc.Ref.ID
		s.activeCache[rule.Domain] = &rule
	}
	s.cacheLoaded = true
	return nil
}
