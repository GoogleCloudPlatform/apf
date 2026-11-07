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
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/cloudsqlconn"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"google.com/apf/api/core"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	sessionDBInstance *gorm.DB
	sessionDBOnce     sync.Once
)

// SessionDatabase wraps gorm.DB instance for Session DB connections.
type SessionDatabase struct {
	*gorm.DB
}

// SessionDBConfig holds connection configuration for Session DB.
type SessionDBConfig struct {
	ConnectionName string
	Host           string
	Port           int
	User           string
	Password       string
	DBName         string
	SSLMode        string
	DSN            string
}

// LoadSessionDBConfig loads database configuration from the provided core.Configuration object.
func LoadSessionDBConfig(cfg *core.Configuration) (SessionDBConfig, error) {
	if cfg == nil {
		return SessionDBConfig{}, fmt.Errorf("configuration is required")
	}

	connName := strings.TrimSpace(cfg.CloudSQLConnectionName)
	if connName == "" {
		return SessionDBConfig{}, fmt.Errorf("CloudSQLConnectionName is required")
	}

	user := strings.TrimSpace(cfg.CloudSQLUser)
	if user == "" {
		return SessionDBConfig{}, fmt.Errorf("CloudSQLUser is required")
	}

	password := cfg.CloudSQLPassword
	dbname := strings.TrimSpace(cfg.CloudSQLDatabase)
	if dbname == "" {
		return SessionDBConfig{}, fmt.Errorf("CloudSQLDatabase is required")
	}

	var host string
	var port int

	useDialer := strings.Count(connName, ":") == 2
	if !useDialer {
		if strings.Contains(connName, ":") {
			parts := strings.Split(connName, ":")
			if parts[0] == "" {
				return SessionDBConfig{}, fmt.Errorf("invalid host in CloudSQLConnectionName: %q", connName)
			}
			host = parts[0]
			p, err := strconv.Atoi(parts[1])
			if err != nil || p <= 0 {
				return SessionDBConfig{}, fmt.Errorf("invalid port in CloudSQLConnectionName %q: %w", connName, err)
			}
			port = p
		} else {
			return SessionDBConfig{}, fmt.Errorf("invalid CloudSQLConnectionName %q: must be in project:region:instance format for Cloud SQL or host:port for direct TCP", connName)
		}
	}

	return SessionDBConfig{
		ConnectionName: connName,
		Host:           host,
		Port:           port,
		User:           user,
		Password:       password,
		DBName:         dbname,
		SSLMode:        "disable",
	}, nil
}

// LoadSessionDBConfigFromEnv loads configuration using the global configuration class.
func LoadSessionDBConfigFromEnv() (SessionDBConfig, error) {
	return LoadSessionDBConfig(core.NewConfig())
}

// BuildDSN constructs a PostgreSQL DSN string from configuration if not already provided.
func (c SessionDBConfig) BuildDSN() string {
	if c.DSN != "" {
		return c.DSN
	}
	sslMode := c.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, sslMode)
}

// ConnectDB connects to the database based on the core.Configuration class, mirroring AI tier database connectivity.
func ConnectDB(ctx context.Context, cfg *core.Configuration) (*gorm.DB, error) {
	if cfg == nil {
		return nil, fmt.Errorf("configuration is required")
	}

	dbConfig, err := LoadSessionDBConfig(cfg)
	if err != nil {
		return nil, err
	}
	sdb, err := NewSessionDatabase(dbConfig)
	if err != nil {
		return nil, err
	}
	return sdb.DB, nil
}

// NewSessionDatabase initializes and returns a GORM connection pool for Session DB.
func NewSessionDatabase(cfg SessionDBConfig) (*SessionDatabase, error) {
	useDialer := strings.Count(cfg.ConnectionName, ":") == 2

	var db *gorm.DB
	var err error

	if useDialer {
		ctx := context.Background()
		dialer, err := cloudsqlconn.NewDialer(ctx)
		if err != nil {
			return nil, fmt.Errorf("cloudsqlconn.NewDialer: %w", err)
		}

		sslMode := cfg.SSLMode
		if sslMode == "" {
			sslMode = "disable"
		}

		dsn := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=%s", cfg.User, cfg.Password, cfg.DBName, sslMode)
		config, err := pgx.ParseConfig(dsn)
		if err != nil {
			return nil, fmt.Errorf("failed to parse config: %w", err)
		}

		config.DialFunc = func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.Dial(ctx, cfg.ConnectionName)
		}

		dbURI := stdlib.RegisterConnConfig(config)

		db, err = gorm.Open(postgres.New(postgres.Config{
			DriverName: "pgx",
			DSN:        dbURI,
		}), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to open session database connection using dialer: %w", err)
		}
		log.Printf("Successfully connected to Cloud SQL instance using dialer: %s", cfg.ConnectionName)
	} else {
		dsn := cfg.BuildDSN()
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to open session database connection: %w", err)
		}
		log.Printf("Successfully connected to database using standard TCP: %s:%d", cfg.Host, cfg.Port)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve sql.DB handle from GORM: %w", err)
	}

	// Connection pool parameters
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	log.Println("Session Database connection pool initialized successfully")
	return &SessionDatabase{DB: db}, nil
}

// GetSessionDBSingleton returns a shared singleton GORM DB instance initialized from the configuration class.
func GetSessionDBSingleton(cfgs ...*core.Configuration) (*gorm.DB, error) {
	var err error
	sessionDBOnce.Do(func() {
		var cfg *core.Configuration
		if len(cfgs) > 0 && cfgs[0] != nil {
			cfg = cfgs[0]
		} else {
			cfg = core.NewConfig()
		}
		dbConfig, errCfg := LoadSessionDBConfig(cfg)
		if errCfg != nil {
			err = fmt.Errorf("failed to load session db configuration: %w", errCfg)
			return
		}
		sdb, errInit := NewSessionDatabase(dbConfig)
		if errInit == nil && sdb != nil {
			sessionDBInstance = sdb.DB
		} else {
			err = errInit
		}
	})
	if err != nil {
		sessionDBOnce = sync.Once{}
		return nil, err
	}
	if sessionDBInstance == nil {
		return nil, fmt.Errorf("failed to initialize Session Database")
	}
	return sessionDBInstance, nil
}

// SetSessionDBSingleton sets the shared singleton GORM DB instance (used in tests or setup).
func SetSessionDBSingleton(db *gorm.DB) {
	sessionDBInstance = db
}
