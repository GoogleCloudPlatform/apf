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
	"fmt"
	"log"
	"os"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

func main() {
	ctx := context.Background()

	projectId := os.Getenv("GOOGLE_CLOUD_PROJECT")
	databaseId := os.Getenv("FIRESTORE_DATABASE_ID")
	if projectId == "" || databaseId == "" {
		log.Fatalf("GOOGLE_CLOUD_PROJECT or FIRESTORE_DATABASE_ID environment variable is not set")
	}

	client, err := firestore.NewClientWithDatabase(ctx, projectId, databaseId)
	if err != nil {
		log.Fatalf("Failed to create firestore client: %v", err)
	}
	defer client.Close()

	iter := client.Collection("projects").Documents(ctx)
	count := 0
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatalf("Failed to iterate projects: %v", err)
		}

		var data map[string]interface{}
		if err := doc.DataTo(&data); err != nil {
			log.Printf("Failed to decode project %s: %v", doc.Ref.ID, err)
			continue
		}

		visibility, ok := data["visibility"].(string)
		if !ok || visibility == "" {
			_, err := doc.Ref.Update(ctx, []firestore.Update{
				{Path: "visibility", Value: "public"},
			})
			if err != nil {
				log.Printf("Failed to update project %s: %v", doc.Ref.ID, err)
			} else {
				fmt.Printf("Migrated project %s to public visibility\n", doc.Ref.ID)
				count++
			}
		}
	}
	fmt.Printf("Completed migration. Migrated %d projects.\n", count)
}
