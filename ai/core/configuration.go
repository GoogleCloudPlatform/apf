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
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/lmittmann/tint"
)

// Configuration represents the structure for our application's configuration.
type Configuration struct {
	GeminiAPIKey    string `json:"GEMINI_API_KEY"`
	DefaultLLMModel string `json:"DEFAULT_LLM_MODEL"`

	SessionServiceType     string `json:"SESSION_SERVICE_TYPE"`
	CloudSQLConnectionName string `json:"CLOUDSQL_CONNECTION_NAME"`
	CloudSQLUser           string `json:"CLOUDSQL_USER"`
	CloudSQLPassword       string `json:"CLOUDSQL_PASSWORD"`
	CloudSQLDatabase       string `json:"CLOUDSQL_DATABASE"`

	ArtifactServiceType string `json:"ARTIFACT_SERVICE_TYPE"`
	GCSArtifactBucket   string `json:"GCS_ARTIFACT_BUCKET"`

	ServiceName       string `json:"SERVICE_NAME"`
	Port              string `json:"SERVICE_PORT"`
	AppUrl            string `json:"APP_URL"`
	AgentLoaderName   string `json:"AGENT_LOADER_NAME"`
	GithubAccessToken string `json:"GITHUB_ACCESS_TOKEN"`
	GithubMcpUrl      string `json:"GITHUB_MCP_URL"`

	WorkspaceMountPath   string `json:"WORKSPACE_MOUNT_PATH"`
	ContainerConcurrency int    `json:"CONTAINER_CONCURRENCY"`

	LogLevel  string `json:"LOG_LEVEL"`
	DebugMode bool   `json:"DEBUG_MODE"`

	GitHubClientID     string `json:"GITHUB_CLIENT_ID"`
	GitHubClientSecret string `json:"GITHUB_CLIENT_SECRET"`

	BmadCfg BmadConfiguration `json:"BMAD_CONFIGURATION"`
	RepoCfg RepoConfiguration `json:"REPO_CONFIGURATION"`
}

// NewConfig creates a new Configuration instance by reading from environment
// variables, with sensible defaults.
func NewConfig() *Configuration {
	// godotenv.Overload will load a .env file if it exists and overwrite existing env vars. It's not an error if it doesn't.
	if err := godotenv.Overload(); err != nil {
		log.Println("No .env file found, using environment variables and defaults.")
	}

	concurrencyStr := getEnv("CONTAINER_CONCURRENCY", getEnv("CONCURRENCY", "80"))
	concurrency, err := strconv.Atoi(concurrencyStr)
	if err != nil || concurrency <= 0 {
		concurrency = 80
	}

	bmadCfg := NewBmadConfig()
	config := &Configuration{

		SessionServiceType:     getEnv("SESSION_SERVICE_TYPE", ""), // Database or InMemory
		CloudSQLConnectionName: getEnv("CLOUDSQL_CONNECTION_NAME", ""),
		CloudSQLUser:           getEnv("CLOUDSQL_USER", ""),
		CloudSQLPassword:       getEnv("CLOUDSQL_PASSWORD", ""),
		CloudSQLDatabase:       getEnv("CLOUDSQL_DATABASE", ""),

		ArtifactServiceType: getEnv("ARTIFACT_SERVICE_TYPE", "File"),
		GCSArtifactBucket:   getEnv("GCS_ARTIFACT_BUCKET", ""),

		DefaultLLMModel:     getEnv("DEFAULT_LLM_MODEL", "gemini-3.7-flash"),

		ServiceName:          getEnv("SERVICE_NAME", "apf-ai"),
		Port:                 getEnv("PORT", "8080"),
		AppUrl:               getEnv("APP_URL", "http://localhost:8080"),
		AgentLoaderName:      getEnv("AGENT_LOADER_NAME", "super"),
		GithubAccessToken:    getEnv("GITHUB_ACCESS_TOKEN", ""),
		GithubMcpUrl:         getEnv("GITHUB_MCP_URL", "https://api.githubcopilot.com/mcp/"),
		WorkspaceMountPath:   getEnv("WORKSPACE_MOUNT_PATH", ""),
		ContainerConcurrency: concurrency,

		BmadCfg: *bmadCfg,

		LogLevel:  getEnv("LOG_LEVEL", "DEBUG"),
		DebugMode: getEnv("DEBUG_MODE", "false") == "true",

		GitHubClientID:     getEnv("GITHUB_CLIENT_ID", ""),
		GitHubClientSecret: getEnv("GITHUB_CLIENT_SECRET", ""),
	}
	initializeLogger(config)
	return config
}

// getEnv is a helper function to read an environment variable or return a default value.
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// ResolveGeminiAPIKey decrypts jweEncrypted payload if present, returning the database-backed Gemini key resolved from user or project credentials.
func (c *Configuration) ResolveGeminiAPIKey(jweEncrypted string) string {
	if jweEncrypted != "" {
		decryptor, err := NewSecretsDecryptor()
		if err == nil {
			secrets, err := decryptor.Decrypt(jweEncrypted)
			if err == nil && secrets != nil && secrets.GeminiApiKey != "" {
				return secrets.GeminiApiKey
			}
		}
	}
	return ""
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

	// // Create a new logger
	// logger := slog.New(tint.NewHandler(w, nil))

	// Set global logger with custom options
	slog.SetDefault(slog.New(
		tint.NewHandler(w, &tint.Options{
			Level:      level,
			TimeFormat: time.Kitchen,
		}),
	))

	// // Configure the logger handler with the specific level
	// handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
	// 	Level: level,
	// })

	// Now all slog calls use this configured level
	slog.Info("Application started", "logLevel", config.LogLevel)

}
