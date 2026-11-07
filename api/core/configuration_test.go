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
	"os"
	"testing"
)

func TestConfigurationIAP(t *testing.T) {
	// Clean up environment variables after test
	defer func() {
		os.Unsetenv("ENVIRONMENT")
		os.Unsetenv("IAP_AUDIENCE")
		os.Unsetenv("JWT_SECRET")
		os.Unsetenv("BYPASS_IAP_VALIDATION")
	}()

	t.Run("Production mode (bypass false) with missing audience (should panic)", func(t *testing.T) {
		os.Setenv("ENVIRONMENT", "production")
		os.Setenv("JWT_SECRET", "some-secret-key-that-is-at-least-32-chars-long")
		os.Setenv("BYPASS_IAP_VALIDATION", "false")
		os.Unsetenv("IAP_AUDIENCE")

		defer func() {
			if r := recover(); r == nil {
				t.Errorf("Expected NewConfig to panic due to missing IAP_AUDIENCE in strict mode, but it did not")
			}
		}()

		_ = NewConfig()
	})

	t.Run("Explicit bypass true (no panic, bypass active in local)", func(t *testing.T) {
		os.Setenv("ENVIRONMENT", "local")
		os.Unsetenv("IAP_AUDIENCE")
		os.Setenv("BYPASS_IAP_VALIDATION", "true")

		config := NewConfig()

		if config.IapAudience != "" {
			t.Errorf("Expected IapAudience to be empty, got %q", config.IapAudience)
		}
		if !config.BypassIAPValidation {
			t.Errorf("Expected BypassIAPValidation to be true, got %t", config.BypassIAPValidation)
		}
	})

	t.Run("Explicit environment values (production mode)", func(t *testing.T) {
		os.Setenv("ENVIRONMENT", "production")
		os.Setenv("IAP_AUDIENCE", "/projects/123456/global/backendServices/abcde")
		os.Setenv("JWT_SECRET", "some-secret-key-that-is-at-least-32-chars-long")
		os.Setenv("BYPASS_IAP_VALIDATION", "false")

		config := NewConfig()

		expectedAudience := "/projects/123456/global/backendServices/abcde"
		if config.IapAudience != expectedAudience {
			t.Errorf("Expected IapAudience to be %q, got %q", expectedAudience, config.IapAudience)
		}
		if config.BypassIAPValidation {
			t.Errorf("Expected BypassIAPValidation to be false, got %t", config.BypassIAPValidation)
		}
	})

	t.Run("Default test behavior (should default to bypass true during test runs in test env)", func(t *testing.T) {
		os.Setenv("ENVIRONMENT", "test")
		os.Unsetenv("BYPASS_IAP_VALIDATION")
		os.Unsetenv("IAP_AUDIENCE")

		config := NewConfig()

		if !config.BypassIAPValidation {
			t.Error("Expected BypassIAPValidation to default to true during test runs")
		}
	})
}

func TestConfigurationProductionFailSafe(t *testing.T) {
	// Clean up environment variables after test
	defer func() {
		os.Unsetenv("ENVIRONMENT")
		os.Unsetenv("IAP_AUDIENCE")
		os.Unsetenv("JWT_SECRET")
		os.Unsetenv("BYPASS_IAP_VALIDATION")
	}()

	t.Run("Production: missing IAP_AUDIENCE should panic", func(t *testing.T) {
		os.Setenv("ENVIRONMENT", "production")
		os.Setenv("JWT_SECRET", "some-secret-key-that-is-at-least-32-chars-long")
		os.Unsetenv("IAP_AUDIENCE")
		os.Setenv("BYPASS_IAP_VALIDATION", "false")

		defer func() {
			if r := recover(); r == nil {
				t.Errorf("Expected NewConfig to panic due to missing IAP_AUDIENCE in production, but it did not")
			} else {
				t.Logf("Panicked as expected: %v", r)
			}
		}()

		_ = NewConfig()
	})

	t.Run("Production: missing JWT_SECRET should panic", func(t *testing.T) {
		os.Setenv("ENVIRONMENT", "production")
		os.Setenv("IAP_AUDIENCE", "some-audience")
		os.Unsetenv("JWT_SECRET")
		os.Setenv("BYPASS_IAP_VALIDATION", "false")

		defer func() {
			if r := recover(); r == nil {
				t.Errorf("Expected NewConfig to panic due to missing JWT_SECRET in production, but it did not")
			} else {
				t.Logf("Panicked as expected: %v", r)
			}
		}()

		_ = NewConfig()
	})

	t.Run("Production: bypass IAP enabled should panic", func(t *testing.T) {
		os.Setenv("ENVIRONMENT", "production")
		os.Setenv("IAP_AUDIENCE", "some-audience")
		os.Setenv("JWT_SECRET", "some-secret-key-that-is-at-least-32-chars-long")
		os.Setenv("BYPASS_IAP_VALIDATION", "true")

		defer func() {
			if r := recover(); r == nil {
				t.Errorf("Expected NewConfig to panic because BYPASS_IAP_VALIDATION is true in production, but it did not")
			} else {
				t.Logf("Panicked as expected: %v", r)
			}
		}()

		_ = NewConfig()
	})

	t.Run("Production: valid config should not panic", func(t *testing.T) {
		os.Setenv("ENVIRONMENT", "production")
		os.Setenv("IAP_AUDIENCE", "some-audience")
		os.Setenv("JWT_SECRET", "some-secret-key-that-is-at-least-32-chars-long")
		os.Setenv("BYPASS_IAP_VALIDATION", "false")

		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Expected NewConfig not to panic, but it did: %v", r)
			}
		}()

		config := NewConfig()
		if config.Environment != "production" {
			t.Errorf("Expected Environment to be production, got %q", config.Environment)
		}
		if config.JwtSecret != "some-secret-key-that-is-at-least-32-chars-long" {
			t.Errorf("Expected JwtSecret to be correct, got %q", config.JwtSecret)
		}
	})

	t.Run("Production: JWT_SECRET too short should panic", func(t *testing.T) {
		os.Setenv("ENVIRONMENT", "production")
		os.Setenv("IAP_AUDIENCE", "some-audience")
		os.Setenv("JWT_SECRET", "short-secret") // < 32 chars
		os.Setenv("BYPASS_IAP_VALIDATION", "false")

		defer func() {
			if r := recover(); r == nil {
				t.Errorf("Expected NewConfig to panic due to short JWT_SECRET in production, but it did not")
			} else {
				t.Logf("Panicked as expected: %v", r)
			}
		}()

		_ = NewConfig()
	})

	t.Run("Environment name normalization (case-insensitive & whitespace)", func(t *testing.T) {
		os.Setenv("ENVIRONMENT", "  LoCaL  ")
		os.Unsetenv("IAP_AUDIENCE")
		os.Setenv("BYPASS_IAP_VALIDATION", "true")

		config := NewConfig()
		if config.Environment != "local" {
			t.Errorf("Expected normalized Environment to be 'local', got %q", config.Environment)
		}
	})
}

func TestConfigurationDebugMode(t *testing.T) {
	os.Setenv("ENVIRONMENT", "test")
	defer func() {
		os.Unsetenv("ENVIRONMENT")
		os.Unsetenv("DEBUG_MODE")
		os.Unsetenv("LOG_LEVEL")
	}()

	t.Run("Default DEBUG_MODE is false", func(t *testing.T) {
		os.Unsetenv("DEBUG_MODE")
		os.Setenv("LOG_LEVEL", "INFO")

		config := NewConfig()
		if config.DebugMode {
			t.Errorf("Expected DebugMode to be false by default, got %t", config.DebugMode)
		}
		if config.IsDebug() {
			t.Errorf("Expected IsDebug() to be false when DEBUG_MODE=false and LOG_LEVEL=INFO")
		}
	})

	t.Run("Explicit DEBUG_MODE=true enables IsDebug()", func(t *testing.T) {
		os.Setenv("DEBUG_MODE", "true")
		os.Setenv("LOG_LEVEL", "INFO")

		config := NewConfig()
		if !config.DebugMode {
			t.Errorf("Expected DebugMode to be true when DEBUG_MODE=true")
		}
		if !config.IsDebug() {
			t.Errorf("Expected IsDebug() to be true when DEBUG_MODE=true")
		}
	})

	t.Run("LOG_LEVEL=debug enables IsDebug() even if DEBUG_MODE=false", func(t *testing.T) {
		os.Setenv("DEBUG_MODE", "false")
		os.Setenv("LOG_LEVEL", "debug")

		config := NewConfig()
		if config.DebugMode {
			t.Errorf("Expected DebugMode to remain false when DEBUG_MODE=false")
		}
		if !config.IsDebug() {
			t.Errorf("Expected IsDebug() to be true when LOG_LEVEL=debug")
		}
	})
}

func TestConfigurationBmadRootPath(t *testing.T) {
	os.Setenv("ENVIRONMENT", "test")
	defer func() {
		os.Unsetenv("ENVIRONMENT")
		os.Unsetenv("BMAD_ROOT_PATH")
	}()

	t.Run("BMAD_ROOT_PATH expands leading tilde", func(t *testing.T) {
		os.Setenv("BMAD_ROOT_PATH", "~/.apf/config/bmad")
		config := NewConfig()
		home, err := os.UserHomeDir()
		if err != nil {
			t.Fatalf("Failed to get home dir: %v", err)
		}
		expected := home + "/.apf/config/bmad"
		if config.BmadRootPath != expected {
			t.Errorf("Expected BmadRootPath to be %q, got %q", expected, config.BmadRootPath)
		}
	})
}

func TestGetWorkspaceMountPath(t *testing.T) {
	t.Run("Valid mount path expands tilde", func(t *testing.T) {
		cfg := &Configuration{WorkspaceMountPath: "~/workspaces"}
		home, err := os.UserHomeDir()
		if err != nil {
			t.Fatalf("Failed to get home dir: %v", err)
		}
		expected := home + "/workspaces"
		if got := cfg.GetWorkspaceMountPath(); got != expected {
			t.Errorf("Expected %q, got %q", expected, got)
		}
	})

	t.Run("Absolute mount path without tilde returns directly", func(t *testing.T) {
		cfg := &Configuration{WorkspaceMountPath: "/opt/foundry/workspaces"}
		expected := "/opt/foundry/workspaces"
		if got := cfg.GetWorkspaceMountPath(); got != expected {
			t.Errorf("Expected %q, got %q", expected, got)
		}
	})

	t.Run("Empty mount path panics", func(t *testing.T) {
		cfg := &Configuration{WorkspaceMountPath: ""}
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("Expected GetWorkspaceMountPath to panic on empty path, but it did not")
			}
		}()
		_ = cfg.GetWorkspaceMountPath()
	})

	t.Run("Nil configuration panics", func(t *testing.T) {
		var cfg *Configuration
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("Expected nil configuration to panic, but it did not")
			}
		}()
		_ = cfg.GetWorkspaceMountPath()
	})
}

func TestDefaultBypassAndEnvVal(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	t.Run("Simulate production command line without test flags", func(t *testing.T) {
		os.Args = []string{"apf-server", "--port=8081"}
		bypass := defaultBypassVal()
		if bypass != "false" {
			t.Errorf("expected 'false', got %q", bypass)
		}
		envVal := defaultEnvVal()
		if envVal != "production" {
			t.Errorf("expected 'production', got %q", envVal)
		}
	})

	t.Run("Simulate test command line", func(t *testing.T) {
		os.Args = []string{"apf-server", "-test.v", "-test.run=TestFoo"}
		bypass := defaultBypassVal()
		if bypass != "true" {
			t.Errorf("expected 'true', got %q", bypass)
		}
		envVal := defaultEnvVal()
		if envVal != "test" {
			t.Errorf("expected 'test', got %q", envVal)
		}
	})
}

func TestConfigurationSuperAdminEmails(t *testing.T) {
	os.Setenv("ENVIRONMENT", "test")
	defer func() {
		os.Unsetenv("ENVIRONMENT")
		os.Unsetenv("SUPER_ADMIN_EMAILS")
	}()

	t.Run("Parses comma separated emails with whitespace", func(t *testing.T) {
		os.Setenv("SUPER_ADMIN_EMAILS", " admin1@example.com , admin2@example.com , , admin3@example.com ")
		cfg := NewConfig()
		if len(cfg.SuperAdminEmails) != 3 {
			t.Fatalf("expected 3 super admin emails, got %d: %v", len(cfg.SuperAdminEmails), cfg.SuperAdminEmails)
		}
		if cfg.SuperAdminEmails[0] != "admin1@example.com" || cfg.SuperAdminEmails[1] != "admin2@example.com" || cfg.SuperAdminEmails[2] != "admin3@example.com" {
			t.Errorf("unexpected emails: %v", cfg.SuperAdminEmails)
		}
	})

	t.Run("Empty SUPER_ADMIN_EMAILS yields empty slice", func(t *testing.T) {
		os.Unsetenv("SUPER_ADMIN_EMAILS")
		cfg := NewConfig()
		if len(cfg.SuperAdminEmails) != 0 {
			t.Errorf("expected 0 emails, got %d", len(cfg.SuperAdminEmails))
		}
	})
}

func TestConfigurationAgentHTTPTimeout(t *testing.T) {
	os.Setenv("ENVIRONMENT", "test")
	defer func() {
		os.Unsetenv("ENVIRONMENT")
		os.Unsetenv("AGENT_HTTP_TIMEOUT")
	}()

	t.Run("Parses valid duration string", func(t *testing.T) {
		os.Setenv("AGENT_HTTP_TIMEOUT", "45s")
		cfg := NewConfig()
		if cfg.AgentHTTPTimeout.Seconds() != 45 {
			t.Errorf("expected 45s, got %v", cfg.AgentHTTPTimeout)
		}
	})

	t.Run("Invalid duration string defaults to 15m", func(t *testing.T) {
		os.Setenv("AGENT_HTTP_TIMEOUT", "not-a-valid-duration")
		cfg := NewConfig()
		if cfg.AgentHTTPTimeout.Minutes() != 15 {
			t.Errorf("expected 15m, got %v", cfg.AgentHTTPTimeout)
		}
	})
}

func TestExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get user home dir: %v", err)
	}

	t.Run("Expand tilde only", func(t *testing.T) {
		if got := ExpandPath("~"); got != home {
			t.Errorf("expected %q, got %q", home, got)
		}
	})

	t.Run("Expand tilde prefix path", func(t *testing.T) {
		expected := home + "/my/folder"
		if got := ExpandPath("~/my/folder"); got != expected {
			t.Errorf("expected %q, got %q", expected, got)
		}
	})

	t.Run("Absolute path remains unchanged", func(t *testing.T) {
		path := "/var/log/app"
		if got := ExpandPath(path); got != path {
			t.Errorf("expected %q, got %q", path, got)
		}
	})

	t.Run("Relative path remains unchanged", func(t *testing.T) {
		path := "relative/dir"
		if got := ExpandPath(path); got != path {
			t.Errorf("expected %q, got %q", path, got)
		}
	})
}

func TestInitializeLogger_LogLevels(t *testing.T) {
	for _, level := range []string{"info", "warn", "error", "unknown_level"} {
		t.Run("Log level "+level, func(t *testing.T) {
			cfg := &Configuration{
				LogLevel: level,
			}
			initializeLogger(cfg)
		})
	}
}

func TestConfigurationArtifactPaths(t *testing.T) {
	os.Setenv("ENVIRONMENT", "test")
	defer func() {
		os.Unsetenv("ENVIRONMENT")
		os.Unsetenv("GCS_ARTIFACT_BUCKET")
		os.Unsetenv("LOCAL_ARTIFACT_PATH")
	}()

	t.Run("Default LOCAL_ARTIFACT_PATH and empty GCS_ARTIFACT_BUCKET", func(t *testing.T) {
		os.Unsetenv("GCS_ARTIFACT_BUCKET")
		os.Unsetenv("LOCAL_ARTIFACT_PATH")
		cfg := NewConfig()
		if cfg.GcsArtifactBucket != "" {
			t.Errorf("Expected empty GcsArtifactBucket, got %q", cfg.GcsArtifactBucket)
		}
		if cfg.LocalArtifactPath != ".foundry/artifacts" {
			t.Errorf("Expected .foundry/artifacts, got %q", cfg.LocalArtifactPath)
		}
		if got := cfg.GetLocalArtifactPath(); got != ".foundry/artifacts" {
			t.Errorf("Expected .foundry/artifacts, got %q", got)
		}
	})

	t.Run("Custom GCS_ARTIFACT_BUCKET and LOCAL_ARTIFACT_PATH", func(t *testing.T) {
		os.Setenv("GCS_ARTIFACT_BUCKET", "my-custom-bucket")
		os.Setenv("LOCAL_ARTIFACT_PATH", "~/custom/artifacts")
		cfg := NewConfig()
		if cfg.GcsArtifactBucket != "my-custom-bucket" {
			t.Errorf("Expected my-custom-bucket, got %q", cfg.GcsArtifactBucket)
		}
		home, _ := os.UserHomeDir()
		expectedLocal := home + "/custom/artifacts"
		if got := cfg.GetLocalArtifactPath(); got != expectedLocal {
			t.Errorf("Expected %q, got %q", expectedLocal, got)
		}
	})
}



