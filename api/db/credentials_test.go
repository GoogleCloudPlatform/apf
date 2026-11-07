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
	"testing"
)

type mockAppCredentialStoreForResolver struct {
	creds map[string]*AppCredential
}

func (m *mockAppCredentialStoreForResolver) GetCredential(ctx context.Context, organizationID string, userId, appId string) (*AppCredential, error) {
	key := userId + "_" + appId
	if cred, ok := m.creds[key]; ok {
		return cred, nil
	}
	return nil, errors.New("credential not found")
}

func (m *mockAppCredentialStoreForResolver) SaveCredential(ctx context.Context, organizationID string, userId, appId string, cred *AppCredential) error {
	return nil
}

func (m *mockAppCredentialStoreForResolver) DeleteCredential(ctx context.Context, organizationID string, userId, appId string) error {
	return nil
}

type mockUserProjectStoreForResolver struct {
	settings map[string]*UserProject
}

func (m *mockUserProjectStoreForResolver) GetProjectAppSetting(ctx context.Context, organizationID string, userId, projectId, appId string) (*UserProject, error) {
	key := userId + "_" + projectId + "_" + appId
	if setting, ok := m.settings[key]; ok {
		return setting, nil
	}
	return &UserProject{
		UserID:    userId,
		ProjectID: projectId,
		AppID:     appId,
		Enabled:   false,
	}, nil
}

func (m *mockUserProjectStoreForResolver) SaveProjectAppSetting(ctx context.Context, organizationID string, setting *UserProject) error {
	return nil
}

func TestResolveAppCredential(t *testing.T) {
	ctx := context.Background()

	t.Run("Empty state returns ErrCredentialNotFound", func(t *testing.T) {
		appCredStore := &mockAppCredentialStoreForResolver{creds: make(map[string]*AppCredential)}
		upStore := &mockUserProjectStoreForResolver{settings: make(map[string]*UserProject)}

		cred, err := ResolveAppCredential(ctx, "org1", upStore, appCredStore, "user1", "proj1", "github")
		if !errors.Is(err, ErrCredentialNotFound) {
			t.Errorf("Expected ErrCredentialNotFound, got: %v", err)
		}
		if cred != nil {
			t.Errorf("Expected nil credential, got: %v", cred)
		}
	})

	t.Run("Only User Level global credential configured", func(t *testing.T) {
		appCredStore := &mockAppCredentialStoreForResolver{
			creds: map[string]*AppCredential{
				"user1_github": {
					IntegrationID: "github",
					Fields: []CredentialAttribute{
						{Name: "pat", Value: "global-user-pat"},
					},
				},
			},
		}
		upStore := &mockUserProjectStoreForResolver{settings: make(map[string]*UserProject)}

		cred, err := ResolveAppCredential(ctx, "org1", upStore, appCredStore, "user1", "proj1", "github")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if pat := GetFieldValue(cred, "pat"); pat != "global-user-pat" {
			t.Errorf("Expected PAT 'global-user-pat', got: '%s'", pat)
		}
	})

	t.Run("Project Override enabled prioritizes project fields", func(t *testing.T) {
		appCredStore := &mockAppCredentialStoreForResolver{
			creds: map[string]*AppCredential{
				"user1_github": {
					IntegrationID: "github",
					Fields: []CredentialAttribute{
						{Name: "pat", Value: "global-user-pat"},
					},
				},
			},
		}
		upStore := &mockUserProjectStoreForResolver{
			settings: map[string]*UserProject{
				"user1_proj1_github": {
					UserID:       "user1",
					ProjectID:    "proj1",
					AppID:        "github",
					Enabled:      true,
					IsOverridden: true,
					Fields: []CredentialAttribute{
						{Name: "pat", Value: "project-override-pat"},
					},
				},
			},
		}

		cred, err := ResolveAppCredential(ctx, "org1", upStore, appCredStore, "user1", "proj1", "github")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if pat := GetFieldValue(cred, "pat"); pat != "project-override-pat" {
			t.Errorf("Expected PAT 'project-override-pat', got: '%s'", pat)
		}
	})

	t.Run("Project Link to custom CredentialID", func(t *testing.T) {
		appCredStore := &mockAppCredentialStoreForResolver{
			creds: map[string]*AppCredential{
				"user1_github": {
					IntegrationID: "github",
					Fields: []CredentialAttribute{
						{Name: "pat", Value: "global-user-pat"},
					},
				},
				"user1_custom-cred": {
					IntegrationID: "github",
					Fields: []CredentialAttribute{
						{Name: "pat", Value: "custom-linked-pat"},
					},
				},
			},
		}
		upStore := &mockUserProjectStoreForResolver{
			settings: map[string]*UserProject{
				"user1_proj1_github": {
					UserID:       "user1",
					ProjectID:    "proj1",
					AppID:        "github",
					Enabled:      true,
					IsOverridden: false,
					CredentialID: "custom-cred",
				},
			},
		}

		cred, err := ResolveAppCredential(ctx, "org1", upStore, appCredStore, "user1", "proj1", "github")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if pat := GetFieldValue(cred, "pat"); pat != "custom-linked-pat" {
			t.Errorf("Expected PAT 'custom-linked-pat', got: '%s'", pat)
		}
	})

	t.Run("Project Link with empty CredentialID defaults to appId", func(t *testing.T) {
		appCredStore := &mockAppCredentialStoreForResolver{
			creds: map[string]*AppCredential{
				"user1_github": {
					IntegrationID: "github",
					Fields: []CredentialAttribute{
						{Name: "pat", Value: "global-user-pat"},
					},
				},
			},
		}
		upStore := &mockUserProjectStoreForResolver{
			settings: map[string]*UserProject{
				"user1_proj1_github": {
					UserID:       "user1",
					ProjectID:    "proj1",
					AppID:        "github",
					Enabled:      true,
					IsOverridden: false,
					CredentialID: "",
				},
			},
		}

		cred, err := ResolveAppCredential(ctx, "org1", upStore, appCredStore, "user1", "proj1", "github")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if pat := GetFieldValue(cred, "pat"); pat != "global-user-pat" {
			t.Errorf("Expected PAT 'global-user-pat', got: '%s'", pat)
		}
	})

	t.Run("Disabled Project Setting falls back to User level global credential", func(t *testing.T) {
		appCredStore := &mockAppCredentialStoreForResolver{
			creds: map[string]*AppCredential{
				"user1_github": {
					IntegrationID: "github",
					Fields: []CredentialAttribute{
						{Name: "pat", Value: "global-user-pat"},
					},
				},
			},
		}
		upStore := &mockUserProjectStoreForResolver{
			settings: map[string]*UserProject{
				"user1_proj1_github": {
					UserID:       "user1",
					ProjectID:    "proj1",
					AppID:        "github",
					Enabled:      false,
					IsOverridden: true,
					Fields: []CredentialAttribute{
						{Name: "pat", Value: "project-override-pat-but-disabled"},
					},
				},
			},
		}

		cred, err := ResolveAppCredential(ctx, "org1", upStore, appCredStore, "user1", "proj1", "github")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if pat := GetFieldValue(cred, "pat"); pat != "global-user-pat" {
			t.Errorf("Expected PAT 'global-user-pat', got: '%s'", pat)
		}
	})
}
