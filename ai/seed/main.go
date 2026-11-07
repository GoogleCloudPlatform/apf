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
	"flag"
	"fmt"
	"log"

	"google.com/apf/ai/core"
	adk_db "google.golang.org/adk/session/database"
	"gorm.io/driver/postgres"
)

func main() {
	ctx := context.Background()
	cfg := core.NewConfig()

	cleanFlag := flag.Bool("clean", false, "Clean data from all tables")
	flag.Parse()

	fmt.Println("Connecting to database...")
	db, err := core.ConnectDB(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	if *cleanFlag {
		fmt.Println("Cleaning data from tables...")

		// Order matters due to foreign key constraints if not using CASCADE.
		// events depends on sessions. So delete events first.

		err = db.Exec("DELETE FROM events").Error
		if err != nil {
			log.Printf("Warning: failed to delete from events: %v", err)
		}

		err = db.Exec("DELETE FROM sessions").Error
		if err != nil {
			log.Printf("Warning: failed to delete from sessions: %v", err)
		}

		err = db.Exec("DELETE FROM user_states").Error
		if err != nil {
			log.Printf("Warning: failed to delete from user_states: %v", err)
		}

		err = db.Exec("DELETE FROM app_states").Error
		if err != nil {
			log.Printf("Warning: failed to delete from app_states: %v", err)
		}

		fmt.Println("Data cleaned successfully.")
		return
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get sql db: %v", err)
	}

	sessionService, err := adk_db.NewSessionService(postgres.New(postgres.Config{
		Conn: sqlDB,
	}))
	if err != nil {
		log.Fatalf("Failed to create session service: %v", err)
	}

	fmt.Println("Running AutoMigrate...")
	err = adk_db.AutoMigrate(sessionService)
	if err != nil {
		log.Fatalf("Failed to AutoMigrate: %v", err)
	}

	fmt.Println("Database schema initialized successfully.")
}
