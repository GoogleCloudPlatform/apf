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

package main

import (
	"context"
	"log"

	"google.com/apf/api/db"
)

func seedAppIntegrations(ctx context.Context) {
	// Seed Application Integrations list
	appIntegrations := []db.AppIntegration{
		{
			ID:           "gemini",
			Name:         "Gemini Models",
			Description:  "Google Gemini AI integration for model calls and agent reasoning.",
			AuthType:     "apikey",
			Icon:         "gemini",
			IsEnabled:    true,
			DisplayOrder: 1,
			Labels:       []string{"AI Model"},
			Fields: []db.CredentialAttribute{
				{
					Name:        "gemini_api_key",
					DisplayName: "Gemini API Key",
					Enabled:     true,
					Required:    true,
					HelpText:    "Enter your Google Gemini API Key.",
					IsSecret:    true,
				},
			},
		},
		{
			ID:           "github",
			Name:         "GitHub",
			Description:  "GitHub integration for code synchronization and repository management.",
			AuthType:     "personal_access_token",
			Icon:         "github",
			IsEnabled:    true,
			DisplayOrder: 2,
			Labels:       []string{"Code Repo", "Issue Tracking"},
			Fields: []db.CredentialAttribute{
				{
					Name:        "pat",
					DisplayName: "Personal Access Token",
					Enabled:     true,
					Required:    true,
					HelpText:    "Enter your GitHub Personal Access Token (PAT) with repo and user scopes.",
					IsSecret:    true,
				},
				{
					Name:        "email",
					DisplayName: "Git Author Email",
					Enabled:     true,
					Required:    true,
					HelpText:    "Your GitHub-verified author email used for Git commits and sync.",
					IsSecret:    false,
				},
			},
		},
		{
			ID:           "figma",
			Name:         "Figma",
			Description:  "Figma integration for importing frames, layout grids, and assets.",
			AuthType:     "apikey",
			Icon:         "figma",
			IsEnabled:    false,
			DisplayOrder: 3,
			Labels:       []string{"Design"},
			Fields: []db.CredentialAttribute{
				{
					Name:        "accessToken",
					DisplayName: "Figma API Key / Personal Access Token",
					Enabled:     true,
					Required:    true,
					HelpText:    "Enter your Figma Personal Access Token from Figma Account Settings.",
					IsSecret:    true,
				},
			},
		},
		{
			ID:           "stitch",
			Name:         "Stitch",
			Description:  "Stitch integration for backend services data pipelines.",
			AuthType:     "apikey",
			Icon:         "stitch",
			IsEnabled:    false,
			DisplayOrder: 4,
			Labels:       []string{"Design"},
			Fields: []db.CredentialAttribute{
				{
					Name:        "accessToken",
					DisplayName: "Stitch API key / service key",
					Enabled:     true,
					Required:    true,
					HelpText:    "Enter your Google Stitch API project key.",
					IsSecret:    true,
				},
			},
		},
		{
			ID:           "drawio",
			Name:         "Draw.io",
			Description:  "Draw.io integration for general design specifications and flowcharts.",
			AuthType:     "apikey",
			Icon:         "drawio",
			IsEnabled:    false,
			DisplayOrder: 5,
			Labels:       []string{"Design"},
			Fields: []db.CredentialAttribute{
				{
					Name:        "accessToken",
					DisplayName: "Draw.io API key",
					Enabled:     true,
					Required:    true,
					HelpText:    "Enter your Draw.io API key.",
					IsSecret:    true,
				},
			},
		},
		{
			ID:           "gitlab",
			Name:         "GitLab",
			Description:  "GitLab integration for repository management and CI/CD pipelines.",
			AuthType:     "personal_access_token",
			Icon:         "gitlab",
			IsEnabled:    false,
			DisplayOrder: 6,
			Labels:       []string{"Code Repo", "Issue Tracking"},
			Fields: []db.CredentialAttribute{
				{
					Name:        "pat",
					DisplayName: "Personal Access Token",
					Enabled:     true,
					Required:    true,
					HelpText:    "Enter your GitLab Personal Access Token (PAT) with api scope.",
					IsSecret:    true,
				},
			},
		},
		{
			ID:           "jira",
			Name:         "Jira",
			Description:  "Jira integration for enterprise issue tracking and agile project management.",
			AuthType:     "personal_access_token",
			Icon:         "jira",
			IsEnabled:    false,
			DisplayOrder: 7,
			Labels:       []string{"Issue Tracking"},
			Fields: []db.CredentialAttribute{
				{
					Name:        "pat",
					DisplayName: "API Token / Personal Access Token",
					Enabled:     true,
					Required:    true,
					HelpText:    "Enter your Jira Personal Access Token or API Token.",
					IsSecret:    true,
				},
			},
		},
		{
			ID:           "linear",
			Name:         "Linear",
			Description:  "Linear integration for modern software project planning and issue tracking.",
			AuthType:     "apikey",
			Icon:         "linear",
			IsEnabled:    false,
			DisplayOrder: 8,
			Labels:       []string{"Issue Tracking"},
			Fields: []db.CredentialAttribute{
				{
					Name:        "accessToken",
					DisplayName: "API Key",
					Enabled:     true,
					Required:    true,
					HelpText:    "Enter your Linear Personal Access Token or API Key.",
					IsSecret:    true,
				},
			},
		},
	}

	for i := range appIntegrations {
		ai := &appIntegrations[i]
		if err := configStore.SaveGlobalAppIntegration(ctx, ai); err != nil {
			log.Printf("Failed to seed app integration %s: %v", ai.ID, err)
		} else {
			log.Printf("Successfully seeded app integration %s (enabled: %t)", ai.ID, ai.IsEnabled)
		}
	}
}
