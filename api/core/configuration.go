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
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/lmittmann/tint"
)

//go:embed agent-doc-map.json
var AgentNameToDocNameMapBytes []byte

// Configuration represents the structure for our application's configuration.
type Configuration struct {
	GoogleCloudProject  string `json:"GOOGLE_CLOUD_PROJECT"`
	GoogleCloudLocation string `json:"GOOGLE_CLOUD_LOCATION"`

	ServiceName string `json:"SERVICE_NAME"`
	Port        string `json:"PORT"`

	AgentBaseURL                    string `json:"AGENT_BASE_URL"`
	WorkflowAgentBaseURL            string `json:"WORKFLOW_AGENT_BASE_URL"`
	AppUrl                          string `json:"APP_URL"`
	UiUrl                           string `json:"UI_URL"`
	WorkspaceMountPath              string `json:"WORKSPACE_MOUNT_PATH"`
	GlobalWorkspacePath             string `json:"GLOBAL_WORKSPACE_PATH"`
	BmadRootPath                    string `json:"BMAD_ROOT_PATH"`
	BmadPlanningArtifactsPath       string `json:"BMAD_PLANNING_ARTIFACTS_PATH"`
	BmadImplementationArtifactsPath string `json:"BMAD_IMPLEMENTATION_ARTIFACTS_PATH"`
	ArtifactServiceType             string `json:"ARTIFACT_SERVICE_TYPE"`
	GcsArtifactBucket               string `json:"GCS_ARTIFACT_BUCKET"`
	LocalArtifactPath               string `json:"LOCAL_ARTIFACT_PATH"`
	DevRepoAccessToken              string `json:"DEV_REPO_ACCESS_TOKEN"`

	FirestoreDbName string `json:"FIRESTORE_DB_NAME"`

	LogLevel  string `json:"LOG_LEVEL"`
	DebugMode bool   `json:"DEBUG_MODE"`

	GitHubClientID     string `json:"GITHUB_CLIENT_ID"`
	GitHubClientSecret string `json:"GITHUB_CLIENT_SECRET"`

	SecretsEncryptionKey string        `json:"SECRETS_ENCRYPTION_KEY"`
	SimulatedUserID      string        `json:"SIMULATED_USER_ID"`
	AgentHTTPTimeout     time.Duration `json:"-"`

	IapAudience         string   `json:"IAP_AUDIENCE"`
	BypassIAPValidation bool     `json:"BYPASS_IAP_VALIDATION"`
	PerfTestKey         string   `json:"PERF_TEST_KEY"`
	Environment         string   `json:"ENVIRONMENT"`
	JwtSecret           string   `json:"JWT_SECRET"`
	SuperAdminEmails    []string `json:"SUPER_ADMIN_EMAILS"`

	AgentNameToDocNameMap map[string]string `json:"AGENT_NAME_TO_DOC_NAME_MAP"`

	SendGridAPIKey    string `json:"SENDGRID_API_KEY"`
	SendGridFromEmail string `json:"SENDGRID_FROM_EMAIL"`
	EmailServiceType  string `json:"EMAIL_SERVICE_TYPE"` // "CONSOLE" or "SENDGRID"

	CloudSQLConnectionName string `json:"CLOUDSQL_CONNECTION_NAME"`
	CloudSQLUser           string `json:"CLOUDSQL_USER"`
	CloudSQLPassword       string `json:"CLOUDSQL_PASSWORD"`
	CloudSQLDatabase       string `json:"CLOUDSQL_DATABASE"`

	// Seeded system permissions
	Permissions           []string `json:"permissions"`
	CustomRolePermissions []string `json:"custom_role_permissions"`
}

// NewConfig creates a new Configuration instance by reading from environment
// variables, with sensible defaults.
func NewConfig() *Configuration {
	// godotenv.Load will load a .env file if it exists. It's not an error if it doesn't.
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables and defaults.")
	}

	timeoutStr := getEnv("AGENT_HTTP_TIMEOUT", "15m")
	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		log.Printf("Warning: Invalid AGENT_HTTP_TIMEOUT '%s', defaulting to 15m: %v", timeoutStr, err)
		timeout = 15 * time.Minute
	}

	superAdminEmailsStr := getEnv("SUPER_ADMIN_EMAILS", "")
	var superAdminEmails []string
	if superAdminEmailsStr != "" {
		for _, email := range strings.Split(superAdminEmailsStr, ",") {
			trimmed := strings.TrimSpace(email)
			if trimmed != "" {
				superAdminEmails = append(superAdminEmails, trimmed)
			}
		}
	}

	config := &Configuration{
		GoogleCloudProject:  getEnv("GOOGLE_CLOUD_PROJECT", ""),
		GoogleCloudLocation: getEnv("GOOGLE_CLOUD_LOCATION", "us-central1"),

		ServiceName: getEnv("SERVICE_NAME", "apf-ai"),
		Port:        getEnv("PORT", "8081"),

		AgentBaseURL:                    getEnv("AGENT_BASE_URL", "http://localhost:8080"),
		WorkflowAgentBaseURL:            getEnv("WORKFLOW_AGENT_BASE_URL", "http://localhost:8087"),
		AppUrl:                          getEnv("APP_URL", "http://localhost:8081"),
		UiUrl:                           getEnv("UI_URL", "http://localhost:8082"),
		WorkspaceMountPath:              getEnv("WORKSPACE_MOUNT_PATH", "~/workspaces"),
		GlobalWorkspacePath:             ExpandPath(getEnv("GLOBAL_WORKSPACE_PATH", filepath.Join(getEnv("WORKSPACE_MOUNT_PATH", "~/workspaces"), "global"))),
		BmadRootPath:                    ExpandPath(getEnv("BMAD_ROOT_PATH", "")),
		BmadPlanningArtifactsPath:       getEnv("BMAD_PLANNING_ARTIFACTS_PATH", "plans"),
		BmadImplementationArtifactsPath: getEnv("BMAD_IMPLEMENTATION_ARTIFACTS_PATH", "implementation"),
		// Possible values are FILE and GCS
		ArtifactServiceType: getEnv("ARTIFACT_SERVICE_TYPE", "FILE"),
		GcsArtifactBucket:   getEnv("GCS_ARTIFACT_BUCKET", getEnv("GCS_BUCKET_NAME", "")),
		LocalArtifactPath:   getEnv("LOCAL_ARTIFACT_PATH", ".foundry/artifacts"),
		DevRepoAccessToken:  getEnv("DEV_REPO_ACCESS_TOKEN", getEnv("GITHUB_ACCESS_TOKEN", "")),

		FirestoreDbName: getEnv("FIRESTORE_DB_NAME", "apf-local-firestore"),
		LogLevel:        getEnv("LOG_LEVEL", "INFO"),
		DebugMode:       getEnv("DEBUG_MODE", "false") == "true",

		GitHubClientID:     getEnv("GITHUB_CLIENT_ID", ""),
		GitHubClientSecret: getEnv("GITHUB_CLIENT_SECRET", ""),

		SecretsEncryptionKey: getEnv("SECRETS_ENCRYPTION_KEY", ""),
		SimulatedUserID:      getEnv("SIMULATED_USER_ID", "simulated-admin-uid"),
		IapAudience:          getEnv("IAP_AUDIENCE", ""),
		BypassIAPValidation:  getEnv("BYPASS_IAP_VALIDATION", defaultBypassVal()) == "true",
		PerfTestKey:          getEnv("PERF_TEST_KEY", ""),
		Environment:          getEnv("ENVIRONMENT", defaultEnvVal()),
		JwtSecret:            getEnv("JWT_SECRET", ""),
		SuperAdminEmails:     superAdminEmails,
		AgentHTTPTimeout:     timeout,

		SendGridAPIKey:    getEnv("SENDGRID_API_KEY", ""),
		SendGridFromEmail: getEnv("SENDGRID_FROM_EMAIL", "noreply@agenticproductfoundry.com"),
		EmailServiceType:  getEnv("EMAIL_SERVICE_TYPE", "CONSOLE"),

		CloudSQLConnectionName: getEnv("CLOUDSQL_CONNECTION_NAME", ""),
		CloudSQLUser:           getEnv("CLOUDSQL_USER", ""),
		CloudSQLPassword:       getEnv("CLOUDSQL_PASSWORD", ""),
		CloudSQLDatabase:       getEnv("CLOUDSQL_DATABASE", ""),

		Permissions:           AllPermissions(),
		CustomRolePermissions: CustomRolePermissions(),
	}

	// Normalize Environment to lowercase and trim spaces
	config.Environment = strings.ToLower(strings.TrimSpace(config.Environment))

	isProduction := config.Environment != "local" && config.Environment != "test"
	if isProduction {
		if config.IapAudience == "" {
			panic("Configuration error: IAP_AUDIENCE must be set in production mode")
		}
		if config.JwtSecret == "" {
			panic("Configuration error: JWT_SECRET must be set in production mode")
		}
		if len(config.JwtSecret) < 32 {
			panic("Configuration error: JWT_SECRET must be at least 32 characters long in production mode")
		}
		if config.BypassIAPValidation {
			panic("Configuration error: BYPASS_IAP_VALIDATION cannot be true in production mode")
		}
	} else {
		if !config.BypassIAPValidation && config.IapAudience == "" {
			panic("Configuration error: IAP_AUDIENCE must be set when BYPASS_IAP_VALIDATION is false (strict mode)")
		}
	}

	var agentNameToDocNameMap map[string]string
	if err := json.Unmarshal(AgentNameToDocNameMapBytes, &agentNameToDocNameMap); err != nil {
		log.Printf("Error unmarshaling AgentNameToDocNameMapBytes: %v", err)
	}
	config.AgentNameToDocNameMap = agentNameToDocNameMap

	initializeLogger(config)
	return config
}

func defaultBypassVal() string {
	// If running under 'go test', default bypass to true for developer convenience
	if len(os.Args) > 0 {
		bin := os.Args[0]
		if strings.HasSuffix(bin, ".test") || strings.Contains(bin, "/_test/") {
			return "true"
		}
	}
	for _, arg := range os.Args {
		if strings.Contains(arg, "-test.run") || strings.HasPrefix(arg, "-test.") {
			return "true"
		}
	}
	return "false"
}

func defaultEnvVal() string {
	if len(os.Args) > 0 {
		bin := os.Args[0]
		if strings.HasSuffix(bin, ".test") || strings.Contains(bin, "/_test/") {
			return "test"
		}
	}
	for _, arg := range os.Args {
		if strings.Contains(arg, "-test.run") || strings.HasPrefix(arg, "-test.") {
			return "test"
		}
	}
	return "production"
}


// getEnv is a helper function to read an environment variable or return a default value.
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// IsDebug returns true if DEBUG_MODE is explicitly enabled or LOG_LEVEL is debug.
func (c *Configuration) IsDebug() bool {
	return c.DebugMode || strings.EqualFold(c.LogLevel, "debug")
}

func initializeLogger(config *Configuration) {

	var level slog.Level
	if config.DebugMode || strings.EqualFold(config.LogLevel, "debug") {
		level = slog.LevelDebug
	} else {
		switch strings.ToLower(config.LogLevel) {
		case "info":
			level = slog.LevelInfo
		case "warn":
			level = slog.LevelWarn
		case "error":
			level = slog.LevelError
		default:
			level = slog.LevelInfo // Default to info if invalid
		}
	}

	w := os.Stderr

	// Set global logger with custom options
	slog.SetDefault(slog.New(
		tint.NewHandler(w, &tint.Options{
			Level:      level,
			TimeFormat: time.Kitchen,
		}),
	))

	slog.Info("Application started", "logLevel", config.LogLevel)

}

// ExpandPath expands leading tilde (~) to the user's home directory.
func ExpandPath(p string) string {
	if strings.HasPrefix(p, "~/") || p == "~" {
		home, err := os.UserHomeDir()
		if err == nil {
			if p == "~" {
				return home
			}
			return filepath.Join(home, p[2:])
		}
	}
	return p
}

// GetWorkspaceMountPath returns the normalized workspace mount path.
// It expands a leading '~' to the user's home directory.
// If WorkspaceMountPath is missing or empty in the configuration, it logs a clear error and panics.
func (c *Configuration) GetWorkspaceMountPath() string {
	if c == nil || strings.TrimSpace(c.WorkspaceMountPath) == "" {
		slog.Error("CRITICAL CONFIGURATION ERROR: WorkspaceMountPath is missing or empty in configuration")
		panic("Configuration error: WorkspaceMountPath is missing or empty in configuration")
	}

	mountPath := strings.TrimSpace(c.WorkspaceMountPath)
	if strings.HasPrefix(mountPath, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			slog.Error("CRITICAL CONFIGURATION ERROR: Failed to resolve user home directory for WorkspaceMountPath", "error", err)
			panic(fmt.Sprintf("Configuration error: failed to resolve user home directory for WorkspaceMountPath: %v", err))
		}
		mountPath = filepath.Join(home, strings.TrimPrefix(mountPath, "~"))
	}

	return mountPath
}

// GetLocalArtifactPath returns the resolved local artifact path.
// If LocalArtifactPath is empty, it defaults to .foundry/artifacts.
func (c *Configuration) GetLocalArtifactPath() string {
	if c == nil || strings.TrimSpace(c.LocalArtifactPath) == "" {
		return ".foundry/artifacts"
	}
	return ExpandPath(strings.TrimSpace(c.LocalArtifactPath))
}
