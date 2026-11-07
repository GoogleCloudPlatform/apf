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

package integrations

import (
	"context"
	"encoding/json"
	"fmt"

	"google.com/apf/api/db"
)

// AppIntegration defines the interface for all application integrations
type AppIntegration interface {
	Validate(ctx context.Context, cred *db.AppCredential) (bool, error)
}

var registryJSON = `{
  "github": "GitHubIntegration",
  "figma": "FigmaIntegration",
  "lucidchart": "LucidchartIntegration",
  "stitch": "StitchIntegration",
  "drawio": "DrawIOIntegration",
  "gemini": "GeminiIntegration"
}`

var integrationMapping map[string]string
var Registry = map[string]AppIntegration{}

func init() {
	if err := json.Unmarshal([]byte(registryJSON), &integrationMapping); err != nil {
		panic(err)
	}
}

// Register registers an implementation with a class/struct key
func Register(className string, impl AppIntegration) {
	Registry[className] = impl
}

// GetIntegration resolves the dynamic integration implementation by its database ID
func GetIntegration(integrationID string) (AppIntegration, error) {
	className, ok := integrationMapping[integrationID]
	if !ok {
		return nil, fmt.Errorf("no mapping found for integration ID: %s", integrationID)
	}
	impl, ok := Registry[className]
	if !ok {
		return nil, fmt.Errorf("no implementation registered for class name: %s", className)
	}
	return impl, nil
}

// Helper to extract a field's value safely
func getFieldValue(cred *db.AppCredential, name string) string {
	for _, f := range cred.Fields {
		if f.Name == name {
			return f.Value
		}
	}
	return ""
}
