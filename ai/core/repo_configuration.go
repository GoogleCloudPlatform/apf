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

package core

import (
	"log/slog"

	"google.golang.org/adk/agent"
)

// RepoConfiguration represents the structure of repo tied to project.
type RepoConfiguration struct {
	ProjectId                        string `json:"projectId"`
	OrganizationId                   string `json:"organizationId"`
	ProjectName                      string `json:"projectName"`
	SpecLocation                     string `json:"specLocation"`
	PlanningArtifactsLocation        string `json:"planningArtifactsLocation"`
	ImplementationArtifactsLocation   string `json:"implementationArtifactsLocation"`
	IsReadOnly                       bool   `json:"isReadOnly"`
}

func NewRepoConfigFromState(ctx agent.CallbackContext, cfg *Configuration) (*RepoConfiguration, error) {

	state := ctx.State()

	projectId_str := ctx.SessionID()

	// 1. Resolve project ID
	if projectId, err := state.Get("projectId"); err == nil {
		projectId_str = getStringVal(projectId, projectId_str)
	} else {
		slog.Warn("No project id provided: Using session id instead", "error", err)
	}

	organizationId_str := "default-organization"
	if organizationId, err := state.Get("organizationId"); err == nil {
		organizationId_str = getStringVal(organizationId, organizationId_str)
	}

	projectName_str := ""
	if projectName, err := state.Get("projectName"); err == nil {
		projectName_str = getStringVal(projectName, projectName_str)
	}
	if projectName_str == "" {
		projectName_str = projectId_str
	}

	specLocation_str := ""
	if specLoc, err := state.Get("specLocation"); err == nil {
		specLocation_str = getStringVal(specLoc, "")
	}

	planningLocation_str := ""
	if planLoc, err := state.Get("planningArtifactsLocation"); err == nil {
		planningLocation_str = getStringVal(planLoc, "")
	}

	implLocation_str := ""
	if implLoc, err := state.Get("implementationArtifactsLocation"); err == nil {
		implLocation_str = getStringVal(implLoc, "")
	}

	isReadOnly_bool := false
	if isReadOnlyVal, err := state.Get("isReadOnly"); err == nil {
		if val, ok := isReadOnlyVal.(bool); ok {
			isReadOnly_bool = val
		} else if valStr, ok := isReadOnlyVal.(string); ok {
			isReadOnly_bool = valStr == "true"
		}
	}

	return &RepoConfiguration{
		ProjectId:                        projectId_str,
		OrganizationId:                   organizationId_str,
		ProjectName:                      projectName_str,
		SpecLocation:                     specLocation_str,
		PlanningArtifactsLocation:        planningLocation_str,
		ImplementationArtifactsLocation:   implLocation_str,
		IsReadOnly:                       isReadOnly_bool,
	}, nil
}

func getStringVal(val interface{}, fallback string) string {
	if val == nil {
		return fallback
	}
	switch v := val.(type) {
	case string:
		if v != "" {
			return v
		}
	case *string:
		if v != nil && *v != "" {
			return *v
		}
	}
	return fallback
}
