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

type Integration struct {
	ID       string                 `json:"id" firestore:"-"`
	UserID   string                 `json:"userId" firestore:"userId"`
	Name     string                 `json:"name" firestore:"name"`
	Settings map[string]interface{} `json:"settings" firestore:"settings"`
}

type IntegrationStore interface {
	GetIntegration(ctx context.Context, userId, name string) (*Integration, error)
	SaveIntegration(ctx context.Context, integration *Integration) error
	ListUserIntegrations(ctx context.Context, userId string) ([]Integration, error)
	DeleteIntegration(ctx context.Context, userId, name string) error
}

type FirestoreIntegrationStore struct {
	client *firestore.Client
}

func NewFirestoreIntegrationStore(client *firestore.Client) *FirestoreIntegrationStore {
	return &FirestoreIntegrationStore{client: client}
}

func (s *FirestoreIntegrationStore) GetIntegration(ctx context.Context, userId, name string) (*Integration, error) {
	docID := fmt.Sprintf("%s_%s", userId, name)
	organizationID, err := core.GetOrganizationID(ctx)
	if err != nil {
		return nil, err
	}
	col := organizationCollection(organizationID, s.client, "integrations")
	doc, err := col.Doc(docID).Get(ctx)
	if err != nil {
		return nil, err
	}
	var integration Integration
	if err := doc.DataTo(&integration); err != nil {
		return nil, err
	}
	integration.ID = doc.Ref.ID
	return &integration, nil
}

func (s *FirestoreIntegrationStore) SaveIntegration(ctx context.Context, integration *Integration) error {
	docID := fmt.Sprintf("%s_%s", integration.UserID, integration.Name)
	organizationID, err := core.GetOrganizationID(ctx)
	if err != nil {
		return err
	}
	col := organizationCollection(organizationID, s.client, "integrations")
	_, err = col.Doc(docID).Set(ctx, integration)
	return err
}

func (s *FirestoreIntegrationStore) ListUserIntegrations(ctx context.Context, userId string) ([]Integration, error) {
	var integrations []Integration
	organizationID, err := core.GetOrganizationID(ctx)
	if err != nil {
		return nil, err
	}
	col := organizationCollection(organizationID, s.client, "integrations")
	iter := col.Where("userId", "==", userId).Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var i Integration
		if err := doc.DataTo(&i); err != nil {
			return nil, err
		}
		i.ID = doc.Ref.ID
		integrations = append(integrations, i)
	}
	return integrations, nil
}

func (s *FirestoreIntegrationStore) DeleteIntegration(ctx context.Context, userId, name string) error {
	docID := fmt.Sprintf("%s_%s", userId, name)
	organizationID, err := core.GetOrganizationID(ctx)
	if err != nil {
		return err
	}
	col := organizationCollection(organizationID, s.client, "integrations")
	_, err = col.Doc(docID).Delete(ctx)
	return err
}
