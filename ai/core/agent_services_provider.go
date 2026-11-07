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
	"context"
	"fmt"
	"log"
	"net"
	"strings"

	"cloud.google.com/go/cloudsqlconn"
	"cloud.google.com/go/storage"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"google.golang.org/adk/artifact"
	"google.golang.org/adk/artifact/gcsartifact"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"google.golang.org/adk/memory"
	"google.golang.org/adk/session"
	adk_db "google.golang.org/adk/session/database"
)

// NewSessionService creates and returns a new session service.
func NewSessionService(ctx context.Context, cfg *Configuration) (session.Service, error) {
	if cfg.SessionServiceType == "Database" {
		sessionService, err := createDatabaseSessionService(ctx, cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create database session service: %w", err)
		}
		return sessionService, nil
	} else {
		return session.InMemoryService(), nil
	}
}

// NewMemoryService creates and returns a new memory service.
func NewMemoryService() memory.Service {
	return memory.InMemoryService()
}

// NewArtifactService creates and returns a new artifact service. Defaults or Falls back to Memory service in case of failures
func NewArtifactService(ctx context.Context, cfg *Configuration) (artifact.Service, error) {
	switch cfg.ArtifactServiceType {

	case "GCS":
		if cfg.GCSArtifactBucket == "" {
			log.Println("GCS artifact service type selected but GCS_ARTIFACT_BUCKET is not set. Falling back to InMemoryArtifactService.")
			return artifact.InMemoryService(), nil
		}

		storageClient, err := storage.NewClient(ctx)
		if err != nil {
			log.Printf("WARNING: Failed to initialize Google Cloud Storage client: %v", err)
			log.Println("Falling back to InMemoryArtifactService.")
			return artifact.InMemoryService(), nil
		}

		// Verify bucket existence and permissions.
		if _, err := storageClient.Bucket(cfg.GCSArtifactBucket).Attrs(ctx); err != nil {
			log.Printf("WARNING: Failed to access GCS bucket '%s': %v", cfg.GCSArtifactBucket, err)
			log.Println("Please check if the bucket exists and if you have the correct permissions.")
			log.Println("Falling back to InMemoryArtifactService.")
			return artifact.InMemoryService(), nil
		}

		service, err := gcsartifact.NewService(ctx, cfg.GCSArtifactBucket)
		log.Printf("Successfully connected to GCS ArtifactService. Bucket: %s", cfg.GCSArtifactBucket)

		if err != nil {
			return nil, fmt.Errorf("failed to create GCS artifact service: %w", err)
		}
		return service, nil
	default: // Fallback to InMemory
		log.Println("No valid artifact service type specified. Using InMemoryArtifactService.")
		return artifact.InMemoryService(), nil
	}
}

// ConnectDB connects to the database based on configuration.
func ConnectDB(ctx context.Context, cfg *Configuration) (*gorm.DB, error) {
	if cfg.CloudSQLConnectionName == "" {
		return nil, fmt.Errorf("CloudSQLConnectionName is required")
	}

	dsn := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable", cfg.CloudSQLUser, cfg.CloudSQLPassword, cfg.CloudSQLDatabase)

	// Check if we should use the Cloud SQL Dialer.
	// Cloud SQL instance connection names are in the format "project:region:instance".
	// If it doesn't match that format (e.g., "localhost:5432"), we assume a direct connection.
	useDialer := strings.Count(cfg.CloudSQLConnectionName, ":") == 2

	var db *gorm.DB
	var err error

	if useDialer {
		dialer, err := cloudsqlconn.NewDialer(ctx)
		if err != nil {
			return nil, fmt.Errorf("cloudsqlconn.NewDialer: %w", err)
		}

		config, err := pgx.ParseConfig(dsn)
		if err != nil {
			return nil, fmt.Errorf("failed to parse config: %w", err)
		}

		config.DialFunc = func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.Dial(ctx, cfg.CloudSQLConnectionName)
		}

		dbURI := stdlib.RegisterConnConfig(config)

		db, err = gorm.Open(postgres.New(postgres.Config{
			DriverName: "pgx",
			DSN:        dbURI,
		}), &gorm.Config{})

		if err == nil {
			log.Printf("Successfully connected to Cloud SQL instance using dialer: %s", cfg.CloudSQLConnectionName)
		}
	} else {
		// Direct connection (e.g., to local database or Cloud SQL Proxy running locally)
		host := cfg.CloudSQLConnectionName
		port := "5432"

		if strings.Contains(cfg.CloudSQLConnectionName, ":") {
			parts := strings.Split(cfg.CloudSQLConnectionName, ":")
			host = parts[0]
			port = parts[1]
		}

		dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, cfg.CloudSQLUser, cfg.CloudSQLPassword, cfg.CloudSQLDatabase)

		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err == nil {
			log.Printf("Successfully connected to database using standard TCP: %s:%s", host, port)
		}
	}

	return db, err
}

func createDatabaseSessionService(ctx context.Context, cfg *Configuration) (session.Service, error) {
	db, err := ConnectDB(ctx, cfg)
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql db: %w", err)
	}

	return adk_db.NewSessionService(postgres.New(postgres.Config{
		Conn: sqlDB,
	}))
}
