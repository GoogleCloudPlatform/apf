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

	"cloud.google.com/go/firestore"
	"google.com/apf/api/core"
	"google.golang.org/api/iterator"
)

type GlobalConfig struct {
	DocumentID         string `firestore:"-" json:"id"`
	GCSWorkspaceBucket string `firestore:"gcsWorkspaceBucket" json:"gcsWorkspaceBucket"`
}

type CredentialAttribute struct {
	Name        string `firestore:"name" json:"name"`               // e.g. "username", "password", "accessToken", "refreshToken"
	DisplayName string `firestore:"displayName" json:"displayName"` // e.g. "Personal Access Token"
	Enabled     bool   `firestore:"enabled" json:"enabled"`         // Enable/disable flag
	Required    bool   `firestore:"required" json:"required"`
	HelpText    string `firestore:"helpText" json:"helpText"`
	IsSecret    bool   `firestore:"isSecret" json:"isSecret"`
	Value       string `firestore:"value,omitempty" json:"value,omitempty"`
}

type AppIntegration struct {
	ID           string                `firestore:"-" json:"id"` // e.g. "github", "figma"
	Name         string                `firestore:"name" json:"name"`
	Description  string                `firestore:"description" json:"description"`
	AuthType     string                `firestore:"authType" json:"authType"` // "oauth", "apikey", "userpass", "personal_access_token"
	Icon         string                `firestore:"icon" json:"icon"`
	Fields       []CredentialAttribute `firestore:"fields" json:"fields"`
	IsEnabled    bool                  `firestore:"isEnabled" json:"isEnabled"`
	DisplayOrder int                   `firestore:"displayOrder" json:"displayOrder"`
	Labels       []string              `firestore:"labels" json:"labels"`
}

type ConfigStore interface {
	GetGlobalConfig(ctx context.Context) (*GlobalConfig, error)
	SetGlobalConfig(ctx context.Context, cfg *GlobalConfig) error
	
	// Global catalog (master list)
	ListGlobalAppIntegrations(ctx context.Context) ([]AppIntegration, error)
	SaveGlobalAppIntegration(ctx context.Context, integration *AppIntegration) error
	DeleteGlobalAppIntegration(ctx context.Context, id string) error

	// Organization-scoped integrations (enabled/configured by org)
	ListOrgAppIntegrations(ctx context.Context, orgID string) ([]AppIntegration, error)
	SaveOrgAppIntegration(ctx context.Context, orgID string, integration *AppIntegration) error
	DeleteOrgAppIntegration(ctx context.Context, orgID string, id string) error
}

type FirestoreConfigStore struct {
	client *firestore.Client
}

func NewFirestoreConfigStore(client *firestore.Client) *FirestoreConfigStore {
	return &FirestoreConfigStore{client: client}
}

func (s *FirestoreConfigStore) GetGlobalConfig(ctx context.Context) (*GlobalConfig, error) {
	organizationID, err := core.GetOrganizationID(ctx)
	if err != nil {
		return nil, err
	}
	col := organizationCollection(organizationID, s.client, "configs")
	doc, err := col.Doc("global").Get(ctx)
	if err != nil {
		return nil, err
	}
	var cfg GlobalConfig
	if err := doc.DataTo(&cfg); err != nil {
		return nil, err
	}
	cfg.DocumentID = doc.Ref.ID
	return &cfg, nil
}

func (s *FirestoreConfigStore) SetGlobalConfig(ctx context.Context, cfg *GlobalConfig) error {
	organizationID, err := core.GetOrganizationID(ctx)
	if err != nil {
		return err
	}
	col := organizationCollection(organizationID, s.client, "configs")
	_, err = col.Doc("global").Set(ctx, cfg)
	return err
}

func (s *FirestoreConfigStore) ListGlobalAppIntegrations(ctx context.Context) ([]AppIntegration, error) {
	var list []AppIntegration
	iter := s.client.Collection("app_integrations").OrderBy("displayOrder", firestore.Asc).Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var ai AppIntegration
		if err := doc.DataTo(&ai); err != nil {
			return nil, err
		}
		ai.ID = doc.Ref.ID
		list = append(list, ai)
	}
	return list, nil
}

func (s *FirestoreConfigStore) SaveGlobalAppIntegration(ctx context.Context, integration *AppIntegration) error {
	if integration.ID == "" {
		return fmt.Errorf("integration ID is required")
	}
	_, err := s.client.Collection("app_integrations").Doc(integration.ID).Set(ctx, integration)
	return err
}

func (s *FirestoreConfigStore) DeleteGlobalAppIntegration(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("integration ID is required")
	}
	_, err := s.client.Collection("app_integrations").Doc(id).Delete(ctx)
	return err
}

func (s *FirestoreConfigStore) ListOrgAppIntegrations(ctx context.Context, orgID string) ([]AppIntegration, error) {
	col := organizationCollection(orgID, s.client, "app_integrations")
	var list []AppIntegration
	iter := col.OrderBy("displayOrder", firestore.Asc).Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var ai AppIntegration
		if err := doc.DataTo(&ai); err != nil {
			return nil, err
		}
		ai.ID = doc.Ref.ID
		list = append(list, ai)
	}
	return list, nil
}

func (s *FirestoreConfigStore) SaveOrgAppIntegration(ctx context.Context, orgID string, integration *AppIntegration) error {
	if integration.ID == "" {
		return fmt.Errorf("integration ID is required")
	}
	col := organizationCollection(orgID, s.client, "app_integrations")
	_, err := col.Doc(integration.ID).Set(ctx, integration)
	return err
}

func (s *FirestoreConfigStore) DeleteOrgAppIntegration(ctx context.Context, orgID string, id string) error {
	if id == "" {
		return fmt.Errorf("integration ID is required")
	}
	col := organizationCollection(orgID, s.client, "app_integrations")
	_, err := col.Doc(id).Delete(ctx)
	return err
}
