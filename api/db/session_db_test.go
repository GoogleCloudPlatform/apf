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
	"strings"
	"testing"

	"google.com/apf/api/core"
)

func TestLoadSessionDBConfig_CloudSQLInstance(t *testing.T) {
	appCfg := &core.Configuration{
		CloudSQLConnectionName: "agenticproductfoundry-489717:us-central1:apf-stg-db-instance-6774c64f",
		CloudSQLUser:           "apf-stg-adk-user",
		CloudSQLPassword:       "secret-pass",
		CloudSQLDatabase:       "apf-stg-session-db",
	}

	cfg, err := LoadSessionDBConfig(appCfg)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if cfg.ConnectionName != "agenticproductfoundry-489717:us-central1:apf-stg-db-instance-6774c64f" {
		t.Errorf("Expected ConnectionName to match, got %s", cfg.ConnectionName)
	}
	if strings.Count(cfg.ConnectionName, ":") != 2 {
		t.Errorf("Expected 2 colons in Cloud SQL Connection Name, got %d", strings.Count(cfg.ConnectionName, ":"))
	}
	if cfg.User != "apf-stg-adk-user" {
		t.Errorf("Expected user apf-stg-adk-user, got %s", cfg.User)
	}
	if cfg.DBName != "apf-stg-session-db" {
		t.Errorf("Expected dbname apf-stg-session-db, got %s", cfg.DBName)
	}
}

func TestLoadSessionDBConfig_HostPort(t *testing.T) {
	appCfg := &core.Configuration{
		CloudSQLConnectionName: "127.0.0.1:5433",
		CloudSQLUser:           "postgres",
		CloudSQLPassword:       "password",
		CloudSQLDatabase:       "dev_db",
	}

	cfg, err := LoadSessionDBConfig(appCfg)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if cfg.Host != "127.0.0.1" {
		t.Errorf("Expected Host 127.0.0.1, got %s", cfg.Host)
	}
	if cfg.Port != 5433 {
		t.Errorf("Expected Port 5433, got %d", cfg.Port)
	}
	dsn := cfg.BuildDSN()
	if !strings.Contains(dsn, "host=127.0.0.1 port=5433") {
		t.Errorf("Expected DSN to contain host and port, got %s", dsn)
	}
}

func TestLoadSessionDBConfig_ValidationErrors(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *core.Configuration
		errContains string
	}{
		{
			name:        "nil config",
			cfg:         nil,
			errContains: "configuration is required",
		},
		{
			name: "missing connection name",
			cfg: &core.Configuration{
				CloudSQLConnectionName: "",
				CloudSQLUser:           "user",
				CloudSQLDatabase:       "db",
			},
			errContains: "CloudSQLConnectionName is required",
		},
		{
			name: "missing user",
			cfg: &core.Configuration{
				CloudSQLConnectionName: "host:5432",
				CloudSQLUser:           "",
				CloudSQLDatabase:       "db",
			},
			errContains: "CloudSQLUser is required",
		},
		{
			name: "missing database name",
			cfg: &core.Configuration{
				CloudSQLConnectionName: "host:5432",
				CloudSQLUser:           "user",
				CloudSQLDatabase:       "",
			},
			errContains: "CloudSQLDatabase is required",
		},
		{
			name: "direct TCP missing port",
			cfg: &core.Configuration{
				CloudSQLConnectionName: "localhost",
				CloudSQLUser:           "user",
				CloudSQLDatabase:       "db",
			},
			errContains: "must be in project:region:instance format for Cloud SQL or host:port for direct TCP",
		},
		{
			name: "direct TCP invalid port",
			cfg: &core.Configuration{
				CloudSQLConnectionName: "localhost:invalid_port",
				CloudSQLUser:           "user",
				CloudSQLDatabase:       "db",
			},
			errContains: "invalid port in CloudSQLConnectionName",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := LoadSessionDBConfig(tc.cfg)
			if err == nil {
				t.Fatalf("Expected error containing %q, got nil", tc.errContains)
			}
			if !strings.Contains(err.Error(), tc.errContains) {
				t.Errorf("Expected error containing %q, got %q", tc.errContains, err.Error())
			}
		})
	}
}

