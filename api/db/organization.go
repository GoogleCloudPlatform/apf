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
	"strings"

	"cloud.google.com/go/firestore"
)

var ErrConflict = errors.New("organization already exists")

type BrandingConfig struct {
	LogoURL *string `firestore:"logo_url,omitempty" json:"logo_url,omitempty"`
	Theme   *string `firestore:"theme,omitempty" json:"theme,omitempty"`
}

type SSOConfig struct {
	Enabled     *bool   `firestore:"enabled,omitempty" json:"enabled,omitempty"`
	ProviderURL *string `firestore:"provider_url,omitempty" json:"provider_url,omitempty"`
}

type OrganizationConfig struct {
	Branding    *BrandingConfig    `firestore:"branding,omitempty" json:"branding,omitempty"`
	SSO         *SSOConfig         `firestore:"sso,omitempty" json:"sso,omitempty"`
	EnvSettings map[string]string `firestore:"env_settings,omitempty" json:"env_settings,omitempty"`
}

type Organization struct {
	ID        string              `json:"id" firestore:"-"`
	TenantID  string              `json:"tenant_id" firestore:"tenant_id"`
	Name      string              `json:"name" firestore:"name"`
	UserCount int                 `json:"user_count" firestore:"-"`
	Config    *OrganizationConfig `json:"config,omitempty" firestore:"config,omitempty"`
}

type OrganizationStore interface {
	GetOrganization(ctx context.Context, id string) (*Organization, error)
	GetOrganizationByName(ctx context.Context, name string) (*Organization, error)
	CreateOrganization(ctx context.Context, org *Organization) (string, error)
	CreateOrganizationWithID(ctx context.Context, id string, org *Organization) error
	ListOrganizations(ctx context.Context) ([]Organization, error)
	UpdateOrganizationConfig(ctx context.Context, id string, config *OrganizationConfig) error
	DeleteOrganization(ctx context.Context, id string) error
	UpdateOrganizationName(ctx context.Context, id string, name string) error
}

type FirestoreOrganizationStore struct {
	client *firestore.Client
}

func NewFirestoreOrganizationStore(client *firestore.Client) *FirestoreOrganizationStore {
	return &FirestoreOrganizationStore{client: client}
}

var OrganizationStoreInstance OrganizationStore

func (s *FirestoreOrganizationStore) GetOrganization(ctx context.Context, id string) (*Organization, error) {
	if id == "" {
		return nil, fmt.Errorf("organization ID cannot be empty")
	}
	doc, err := s.client.Collection("organizations").Doc(id).Get(ctx)
	if err != nil {
		return nil, err
	}
	var org Organization
	if err := doc.DataTo(&org); err != nil {
		return nil, err
	}
	org.ID = doc.Ref.ID
	return &org, nil
}

func (s *FirestoreOrganizationStore) GetOrganizationByName(ctx context.Context, name string) (*Organization, error) {
	if name == "" {
		return nil, fmt.Errorf("organization name cannot be empty")
	}
	nameLower := strings.ToLower(name)
	docs, err := s.client.Collection("organizations").Where("name_lowercase", "==", nameLower).Limit(1).Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}
	if len(docs) == 0 {
		return nil, ErrNotFound
	}
	var org Organization
	if err := docs[0].DataTo(&org); err != nil {
		return nil, err
	}
	org.ID = docs[0].Ref.ID
	return &org, nil
}

func (s *FirestoreOrganizationStore) CreateOrganization(ctx context.Context, org *Organization) (string, error) {
	var orgID string
	err := s.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		nameLower := strings.ToLower(org.Name)
		query := s.client.Collection("organizations").Where("name_lowercase", "==", nameLower).Limit(1)
		iter := tx.Documents(query)
		docs, err := iter.GetAll()
		if err != nil {
			return err
		}
		if len(docs) > 0 {
			return ErrConflict
		}

		newDocRef := s.client.Collection("organizations").NewDoc()
		orgID = newDocRef.ID
		return tx.Create(newDocRef, map[string]interface{}{
			"name":           org.Name,
			"name_lowercase": nameLower,
			"tenant_id":      org.TenantID,
		})
	})
	if err != nil {
		return "", err
	}
	return orgID, nil
}

func (s *FirestoreOrganizationStore) CreateOrganizationWithID(ctx context.Context, id string, org *Organization) error {
	if id == "" {
		return fmt.Errorf("organization ID cannot be empty")
	}
	if org == nil {
		return fmt.Errorf("organization cannot be nil")
	}
	return s.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		nameLower := strings.ToLower(org.Name)
		query := s.client.Collection("organizations").Where("name_lowercase", "==", nameLower).Limit(1)
		iter := tx.Documents(query)
		docs, err := iter.GetAll()
		if err != nil {
			return err
		}
		if len(docs) > 0 {
			return ErrConflict
		}

		docRef := s.client.Collection("organizations").Doc(id)
		return tx.Create(docRef, map[string]interface{}{
			"name":           org.Name,
			"name_lowercase": nameLower,
			"tenant_id":      org.TenantID,
		})
	})
}

func (s *FirestoreOrganizationStore) ListOrganizations(ctx context.Context) ([]Organization, error) {
	docs, err := s.client.Collection("organizations").Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}
	var orgs []Organization
	for _, doc := range docs {
		var org Organization
		if err := doc.DataTo(&org); err != nil {
			return nil, err
		}
		org.ID = doc.Ref.ID
		orgs = append(orgs, org)
	}
	return orgs, nil
}

func (s *FirestoreOrganizationStore) UpdateOrganizationConfig(ctx context.Context, id string, config *OrganizationConfig) error {
	if id == "" {
		return fmt.Errorf("organization ID cannot be empty")
	}
	_, err := s.client.Collection("organizations").Doc(id).Update(ctx, []firestore.Update{
		{Path: "config", Value: config},
	})
	return err
}

func (s *FirestoreOrganizationStore) DeleteOrganization(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("organization ID cannot be empty")
	}
	_, err := s.client.Collection("organizations").Doc(id).Delete(ctx)
	return err
}

func (s *FirestoreOrganizationStore) UpdateOrganizationName(ctx context.Context, id string, name string) error {
	if id == "" {
		return fmt.Errorf("organization ID cannot be empty")
	}
	if name == "" {
		return fmt.Errorf("organization name cannot be empty")
	}
	nameLower := strings.ToLower(name)
	_, err := s.client.Collection("organizations").Doc(id).Update(ctx, []firestore.Update{
		{Path: "name", Value: name},
		{Path: "name_lowercase", Value: nameLower},
	})
	return err
}
