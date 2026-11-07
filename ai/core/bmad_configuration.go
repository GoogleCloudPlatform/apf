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
	"log"

	"github.com/joho/godotenv"
	"google.golang.org/adk/session"
)

type StateGetter interface {
	State() session.State
}

// NewBmadConfigFromState resolves BmadConfiguration from state or configuration defaults.
func NewBmadConfigFromState(ctx StateGetter, cfg *Configuration) *BmadConfiguration {
	state := ctx.State()
	bmadCfg := cfg.BmadCfg

	if val, err := state.Get("specLocation"); err == nil {
		bmadCfg.BmadOutputPath = getStringVal(val, bmadCfg.BmadOutputPath)
	}
	if val, err := state.Get("planningArtifactsLocation"); err == nil {
		bmadCfg.BmadPlanningArtifactsPath = getStringVal(val, bmadCfg.BmadPlanningArtifactsPath)
	}
	if val, err := state.Get("implementationArtifactsLocation"); err == nil {
		bmadCfg.BmadImplementationArtifactsPath = getStringVal(val, bmadCfg.BmadImplementationArtifactsPath)
	}
	if val, err := state.Get("testingArtifactsLocation"); err == nil {
		bmadCfg.BmadTestingArtifactsPath = getStringVal(val, bmadCfg.BmadTestingArtifactsPath)
	}

	return &bmadCfg
}

// Configuration represents the structure for our application's configuration.
type BmadConfiguration struct {
	BmadRootPath                    string `json:"BMAD_ROOT_PATH"`
	BmadPlanningArtifactsPath       string `json:"BMAD_PLANNING_ARTIFACTS_PATH"`
	BmadImplementationArtifactsPath string `json:"BMAD_IMPLEMENTATION_ARTIFACTS_PATH"`
	BmadTestingArtifactsPath        string `json:"BMAD_TESTING_ARTIFACTS_PATH"`
	BmadOutputPath                  string `json:"BMAD_OUTPUT_PATH"`
}

func (b *BmadConfiguration) GetBmadRootPath() string {
	if b != nil {
		return b.BmadRootPath
	}
	return ""
}

func (b *BmadConfiguration) GetSpecLocation() string {
	if b != nil && b.BmadOutputPath != "" {
		return b.BmadOutputPath
	}
	return "docs/spec"
}

func (b *BmadConfiguration) GetPlanningArtifactsLocation() string {
	if b != nil && b.BmadPlanningArtifactsPath != "" {
		return b.BmadPlanningArtifactsPath
	}
	return "docs/spec/plans"
}

func (b *BmadConfiguration) GetImplementationArtifactsLocation() string {
	if b != nil && b.BmadImplementationArtifactsPath != "" {
		return b.BmadImplementationArtifactsPath
	}
	return "docs/spec/implementation"
}

func (b *BmadConfiguration) GetTestingArtifactsLocation() string {
	if b != nil && b.BmadTestingArtifactsPath != "" {
		return b.BmadTestingArtifactsPath
	}
	return "docs/spec/test"
}

// NewBmadConfig creates a new BmadConfiguration instance reading from environment variables.
func NewBmadConfig() *BmadConfiguration {
	if err := godotenv.Overload(); err != nil {
		log.Println("No .env file found, using environment variables and defaults.")
	}

	config := &BmadConfiguration{
		BmadRootPath:                    getEnv("BMAD_ROOT_PATH", ""),
		BmadPlanningArtifactsPath:       getEnv("BMAD_PLANNING_ARTIFACTS_PATH", "docs/spec/plans"),
		BmadImplementationArtifactsPath: getEnv("BMAD_IMPLEMENTATION_ARTIFACTS_PATH", "docs/spec/implementation"),
		BmadTestingArtifactsPath:        getEnv("BMAD_TESTING_ARTIFACTS_PATH", "docs/spec/test"),
		BmadOutputPath:                  getEnv("BMAD_OUTPUT_PATH", "docs/spec"),
	}
	return config
}
