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

package workspace

import (
	"context"
	"net/http"

	"google.com/apf/api/core"
)

func init() {
	// Execute background tasks synchronously during tests to eliminate data races
	SetDefaultAsyncRunner(func(fn func()) {
		fn()
	})
}

func testCtx() context.Context {
	return context.WithValue(context.Background(), core.OrganizationIDKey, "test-organization")
}

func withTestCtx(req *http.Request, claims *core.UserClaims) *http.Request {
	ctx := context.WithValue(req.Context(), core.UserClaimsKey, claims)
	ctx = context.WithValue(ctx, core.OrganizationIDKey, "test-organization")
	return req.WithContext(ctx)
}
