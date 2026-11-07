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

	"cloud.google.com/go/firestore"
	"google.com/apf/api/core"
)

func testCtx() context.Context {
	return context.WithValue(context.Background(), core.OrganizationIDKey, "test-organization")
}

func organizationCollectionRef(client *firestore.Client, collectionName string) *firestore.CollectionRef {
	return client.Collection("organizations").Doc("test-organization").Collection(collectionName)
}
