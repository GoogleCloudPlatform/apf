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
)

// Global store instances to be initialized by the main application package.
var (
	AppCredentialStoreInstance AppCredentialStore
	UserProjectStoreInstance   UserProjectStore
)

// ErrCredentialNotFound is returned when no database credential records are configured.
var ErrCredentialNotFound = errors.New("app credential not found")

func ResolveAppCredential(ctx context.Context, organizationID string, upStore UserProjectStore, appCredStore AppCredentialStore, userId, projectId, appId string) (*AppCredential, error) {
	if upStore == nil {
		upStore = UserProjectStoreInstance
	}
	if appCredStore == nil {
		appCredStore = AppCredentialStoreInstance
	}

	// 1. Check User-Project level setting
	if projectId != "" && upStore != nil {
		setting, err := upStore.GetProjectAppSetting(ctx, organizationID, userId, projectId, appId)
		if err == nil && setting != nil && setting.Enabled {
			if setting.IsOverridden {
				return &AppCredential{
					IntegrationID: appId,
					Fields:        setting.Fields,
					LastLogin:     setting.UpdatedAt,
				}, nil
			} else {
				credentialID := setting.CredentialID
				if credentialID == "" {
					credentialID = appId
				}
				if appCredStore != nil {
					cred, err := appCredStore.GetCredential(ctx, organizationID, userId, credentialID)
					if err == nil && cred != nil {
						return cred, nil
					}
				}
			}
		}
	}

	// 2. Fall back to global User level credential
	if appCredStore != nil {
		cred, err := appCredStore.GetCredential(ctx, organizationID, userId, appId)
		if err == nil && cred != nil {
			return cred, nil
		}
	}

	return nil, ErrCredentialNotFound
}

// GetFieldValue utility parses and returns the string value of a named field attribute from the credential fields.
func GetFieldValue(cred *AppCredential, name string) string {
	if cred == nil {
		return ""
	}
	for _, field := range cred.Fields {
		if field.Name == name {
			return field.Value
		}
	}
	return ""
}
