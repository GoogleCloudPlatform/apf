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

	"google.golang.org/api/idtoken"
)

// TokenValidator defines interface for validating Google ID tokens/IAP assertions.
type TokenValidator interface {
	Validate(ctx context.Context, token string, audience string) (*idtoken.Payload, error)
}

// GoogleTokenValidator is the production implementation wrapping standard Google idtoken validation.
type GoogleTokenValidator struct{}

func (g *GoogleTokenValidator) Validate(ctx context.Context, token string, audience string) (*idtoken.Payload, error) {
	return idtoken.Validate(ctx, token, audience)
}

// MockTokenValidator is a simulation validator for local testing.
type MockTokenValidator struct {
	Payloads         map[string]*idtoken.Payload
	Errors           map[string]error
	ExpectedAudience string
}

func (m *MockTokenValidator) Validate(ctx context.Context, token string, audience string) (*idtoken.Payload, error) {
	if m.ExpectedAudience != "" && audience != m.ExpectedAudience {
		return nil, fmt.Errorf("mock validation failed: audience mismatch (got %q, want %q)", audience, m.ExpectedAudience)
	}
	if m.Errors != nil {
		if err, ok := m.Errors[token]; ok {
			return nil, err
		}
	}
	if m.Payloads != nil {
		if payload, ok := m.Payloads[token]; ok {
			return payload, nil
		}
	}
	return nil, fmt.Errorf("mock validation failed: token %q not configured", token)
}

var globalTokenValidator TokenValidator = &GoogleTokenValidator{}
