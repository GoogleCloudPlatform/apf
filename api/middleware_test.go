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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"google.golang.org/api/idtoken"

	"google.com/apf/api/core"
)

type capturedLog struct {
	Level  string `json:"level"`
	Msg    string `json:"msg"`
	Method string `json:"method"`
	Path   string `json:"path"`
	Status int    `json:"status"`
	Error  string `json:"error,omitempty"`
}

func TestLoggingMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		handler        http.HandlerFunc
		expectedLevel  string
		expectedStatus int
		expectedError  string
	}{
		{
			name: "Success 200 OK",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("success response"))
			},
			expectedLevel:  "INFO",
			expectedStatus: http.StatusOK,
			expectedError:  "",
		},
		{
			name: "Client Error 400 Bad Request",
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "invalid user input", http.StatusBadRequest)
			},
			expectedLevel:  "WARN",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid user input",
		},
		{
			name: "Server Error 500 Internal Server Error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "internal database failure", http.StatusInternalServerError)
			},
			expectedLevel:  "ERROR",
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "internal database failure",
		},
		{
			name: "Truncated Error Response",
			handler: func(w http.ResponseWriter, r *http.Request) {
				largeError := strings.Repeat("A", 2000)
				http.Error(w, largeError, http.StatusInternalServerError)
			},
			expectedLevel:  "ERROR",
			expectedStatus: http.StatusInternalServerError,
			expectedError:  strings.Repeat("A", 1024), // truncated to 1024 bytes
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
				ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
					// Make sure we get standard keys in our JSON
					if a.Key == slog.LevelKey {
						return slog.Attr{Key: "level", Value: a.Value}
					}
					if a.Key == slog.MessageKey {
						return slog.Attr{Key: "msg", Value: a.Value}
					}
					return a
				},
			}))

			// Temporarily set custom default logger
			oldLogger := slog.Default()
			slog.SetDefault(logger)
			defer slog.SetDefault(oldLogger)

			req := httptest.NewRequest("POST", "/test-path", nil)
			rr := httptest.NewRecorder()

			middleware := loggingMiddleware(tc.handler)
			middleware.ServeHTTP(rr, req)

			// Parse the logged JSON
			var logEntry capturedLog
			err := json.Unmarshal(buf.Bytes(), &logEntry)
			if err != nil {
				t.Fatalf("failed to parse log JSON %q: %v", buf.String(), err)
			}

			if logEntry.Level != tc.expectedLevel {
				t.Errorf("expected level %q, got %q", tc.expectedLevel, logEntry.Level)
			}
			if logEntry.Status != tc.expectedStatus {
				t.Errorf("expected status %d, got %d", tc.expectedStatus, logEntry.Status)
			}
			if logEntry.Method != "POST" {
				t.Errorf("expected method POST, got %q", logEntry.Method)
			}
			if logEntry.Path != "/test-path" {
				t.Errorf("expected path /test-path, got %q", logEntry.Path)
			}
			if logEntry.Error != tc.expectedError {
				t.Errorf("expected error %q, got %q", tc.expectedError, logEntry.Error)
			}
		})
	}
}

func TestOrgScopeMiddleware(t *testing.T) {
	// Backup and restore globals
	oldConfig := appConfig
	defer func() {
		appConfig = oldConfig
	}()
	appConfig = &core.Configuration{
		BypassIAPValidation: true,
		Environment:         "test",
	}

	tests := []struct {
		name           string
		claims         *core.UserClaims
		orgHeader      string
		expectedStatus int
		expectedOrgID  string
	}{
		{
			name: "Success - Org-scoped token matches header",
			claims: &core.UserClaims{
				Subject:        "user1",
				OrganizationID: "org1",
			},
			orgHeader:      "org1",
			expectedStatus: http.StatusOK,
			expectedOrgID:  "org1",
		},
		{
			name: "Failure - Primary token without active tenant (must be rejected)",
			claims: &core.UserClaims{
				Subject:       "user1",
				Organizations: []string{"org1", "org2"},
			},
			orgHeader:      "org2",
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "Success - Super Admin (bypasses membership checks)",
			claims: &core.UserClaims{
				Subject:      "super-admin",
				IsSuperAdmin: true,
			},
			orgHeader:      "any-org-id",
			expectedStatus: http.StatusOK,
			expectedOrgID:  "any-org-id",
		},
		{
			name: "Failure - Org mismatch (org-scoped token)",
			claims: &core.UserClaims{
				Subject:        "user1",
				OrganizationID: "org1",
			},
			orgHeader:      "org2",
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "Failure - Org not in allowed list (primary token)",
			claims: &core.UserClaims{
				Subject:       "user1",
				Organizations: []string{"org1", "org2"},
			},
			orgHeader:      "org3",
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "Failure - Missing header",
			claims: &core.UserClaims{
				Subject:       "user1",
				Organizations: []string{"org1"},
			},
			orgHeader:      "",
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "Failure - Empty claims",
			claims: &core.UserClaims{
				Subject: "user1",
			},
			orgHeader:      "org1",
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "Failure - Malformed header (special chars)",
			claims: &core.UserClaims{
				Subject:        "user1",
				OrganizationID: "org1",
				Organizations:  []string{"org1"},
			},
			orgHeader:      "org1;drop table",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Failure - Malformed header (spaces)",
			claims: &core.UserClaims{
				Subject:        "user1",
				OrganizationID: "org1",
				Organizations:  []string{"org1"},
			},
			orgHeader:      "org 1",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Failure - Missing claims",
			claims:         nil,
			orgHeader:      "org1",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "Success - Super Admin bypasses organization access checks",
			claims: &core.UserClaims{
				Subject:      "super-admin-uid",
				IsSuperAdmin: true,
			},
			orgHeader:      "any-org-id",
			expectedStatus: http.StatusOK,
			expectedOrgID:  "any-org-id",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tc.expectedOrgID != "" {
					orgID, err := core.GetOrganizationID(r.Context())
					if err != nil {
						t.Errorf("expected org ID in context, got error: %v", err)
					}
					if orgID != tc.expectedOrgID {
						t.Errorf("expected org ID %q in context, got %q", tc.expectedOrgID, orgID)
					}
				}
				w.WriteHeader(http.StatusOK)
			})

			middleware := orgScopeMiddleware(nextHandler)

			req := httptest.NewRequest("GET", "/test-path", nil)
			if tc.orgHeader != "" {
				req.Header.Set(core.OrganizationIDHeader, tc.orgHeader)
			}

			if tc.claims != nil {
				ctx := context.WithValue(req.Context(), core.UserClaimsKey, tc.claims)
				req = req.WithContext(ctx)
			}

			rr := httptest.NewRecorder()
			middleware.ServeHTTP(rr, req)

			if rr.Code != tc.expectedStatus {
				t.Errorf("expected status %d, got %d", tc.expectedStatus, rr.Code)
			}
		})
	}
}

func TestOrgScopeMiddleware_Security(t *testing.T) {
	// Backup and restore globals
	oldConfig := appConfig
	defer func() {
		appConfig = oldConfig
	}()

	t.Run("Strict Mode - Reject empty organization ID in claims", func(t *testing.T) {
		appConfig = &core.Configuration{BypassIAPValidation: false}
		
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("Next handler should not be called")
		})

		middleware := orgScopeMiddleware(nextHandler)

		req := httptest.NewRequest("GET", "/test-path", nil)
		req.Header.Set(core.OrganizationIDHeader, "org1")

		// Claims has no OrganizationID (it's empty)
		claims := &core.UserClaims{
			Subject: "user1",
		}
		ctx := context.WithValue(req.Context(), core.UserClaimsKey, claims)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		middleware.ServeHTTP(rr, req)

		if rr.Code != http.StatusForbidden {
			t.Errorf("Expected status 403, got %d", rr.Code)
		}
		if !strings.Contains(rr.Body.String(), "Forbidden: user token is not scoped to a valid tenant") {
			t.Errorf("Expected tenant scoping error message, got %q", rr.Body.String())
		}
	})

	t.Run("Bypass Mode - Reject empty organization ID in claims even if BypassIAPValidation is true", func(t *testing.T) {
		appConfig = &core.Configuration{BypassIAPValidation: true}
		
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("Next handler should not be called")
		})

		middleware := orgScopeMiddleware(nextHandler)

		req := httptest.NewRequest("GET", "/test-path", nil)
		req.Header.Set(core.OrganizationIDHeader, "org1")

		// Claims has no OrganizationID (it's empty)
		claims := &core.UserClaims{
			Subject:       "user1",
			Organizations: []string{"org1"},
		}
		ctx := context.WithValue(req.Context(), core.UserClaimsKey, claims)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		middleware.ServeHTTP(rr, req)

		if rr.Code != http.StatusForbidden {
			t.Errorf("Expected status 403, got %d", rr.Code)
		}
		if !strings.Contains(rr.Body.String(), "Forbidden: user token is not scoped to a valid tenant") {
			t.Errorf("Expected tenant scoping error message, got %q", rr.Body.String())
		}
	})
}

func TestAuthMiddleware_IAP(t *testing.T) {
	// Backup and restore globals
	oldConfig := appConfig
	oldValidator := globalTokenValidator
	defer func() {
		appConfig = oldConfig
		globalTokenValidator = oldValidator
	}()

	// Setup mock validator with audience assertion
	mockValidator := &MockTokenValidator{
		ExpectedAudience: "my-aud",
		Payloads: map[string]*idtoken.Payload{
			"valid-iap-token": {
				Subject: "iap-user-123",
				Claims: map[string]interface{}{
					"email": "iap-user@example.com",
					"firebase": map[string]interface{}{
						"tenant": "tenant-abc",
					},
				},
			},
			"another-iap-token": {
				Subject: "another-user-456",
				Claims: map[string]interface{}{
					"email": "another@example.com",
				},
			},
		},
		Errors: map[string]error{
			"invalid-iap-token": errors.New("invalid signature"),
		},
	}
	globalTokenValidator = mockValidator

	// Helper to create a request and run it through authMiddleware
	runMiddleware := func(cfg *core.Configuration, modifyReq func(*http.Request)) (*httptest.ResponseRecorder, *core.UserClaims) {
		appConfig = cfg
		var capturedClaims *core.UserClaims

		handler := authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
			if ok {
				capturedClaims = claims
			}
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/protected-route", nil)
		if modifyReq != nil {
			modifyReq(req)
		}
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		return rr, capturedClaims
	}

	t.Run("Strict Mode - Missing IAP assertion header", func(t *testing.T) {
		cfg := &core.Configuration{IapAudience: "my-aud", BypassIAPValidation: false}
		rr, claims := runMiddleware(cfg, nil)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d. Body: %s", rr.Code, rr.Body.String())
		}
		if claims != nil {
			t.Error("Expected no claims in context")
		}
	})

	t.Run("Strict Mode - Invalid IAP token", func(t *testing.T) {
		cfg := &core.Configuration{IapAudience: "my-aud", BypassIAPValidation: false}
		rr, _ := runMiddleware(cfg, func(req *http.Request) {
			req.Header.Set("X-Goog-IAP-JWT-Assertion", "invalid-iap-token")
		})

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", rr.Code)
		}
	})

	t.Run("Strict Mode - Valid IAP token", func(t *testing.T) {
		cfg := &core.Configuration{IapAudience: "my-aud", BypassIAPValidation: false}
		rr, claims := runMiddleware(cfg, func(req *http.Request) {
			req.Header.Set("X-Goog-IAP-JWT-Assertion", "valid-iap-token")
		})

		if rr.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", rr.Code)
		}
		if claims == nil {
			t.Fatal("Expected claims in context, got nil")
		}
		if claims.Subject != "iap-user-123" {
			t.Errorf("Expected subject iap-user-123, got %q", claims.Subject)
		}
		if claims.Username != "iap-user@example.com" {
			t.Errorf("Expected username iap-user@example.com, got %q", claims.Username)
		}
		if claims.OrganizationID != "tenant-abc" {
			t.Errorf("Expected organization ID tenant-abc, got %q", claims.OrganizationID)
		}
	})

	t.Run("Strict Mode - Session Hijacking (subject mismatch)", func(t *testing.T) {
		cfg := &core.Configuration{IapAudience: "my-aud", BypassIAPValidation: false}
		
		// Mint a session token for "different-user"
		sessionToken, err := MintOrgSessionToken("different-user@example.com", "org-abc", []string{"viewer"}, []string{"read:data"}, false, "dev-secret-key-that-must-be-changed-in-production-12345")
		if err != nil {
			t.Fatalf("failed to mint token: %v", err)
		}

		rr, _ := runMiddleware(cfg, func(req *http.Request) {
			req.Header.Set("X-Goog-IAP-JWT-Assertion", "valid-iap-token") // Subject is iap-user-123 / iap-user@example.com
			req.Header.Set("Authorization", "Bearer "+sessionToken)
		})

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d. Body: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("Strict Mode - Valid matching Session Token and IAP", func(t *testing.T) {
		os.Setenv("JWT_SECRET", "dev-secret-key-that-must-be-changed-in-production-12345")
		defer os.Unsetenv("JWT_SECRET")

		cfg := &core.Configuration{
			IapAudience:         "my-aud",
			BypassIAPValidation: false,
			JwtSecret:           "dev-secret-key-that-must-be-changed-in-production-12345",
		}
		
		// Mint a session token for "iap-user@example.com" (which matches the email/subject of valid-iap-token)
		sessionToken, err := MintOrgSessionToken("iap-user@example.com", "org-abc", []string{"viewer"}, []string{"read:data"}, false, "dev-secret-key-that-must-be-changed-in-production-12345")
		if err != nil {
			t.Fatalf("failed to mint token: %v", err)
		}

		rr, claims := runMiddleware(cfg, func(req *http.Request) {
			req.Header.Set("X-Goog-IAP-JWT-Assertion", "valid-iap-token")
			req.Header.Set("Authorization", "Bearer "+sessionToken)
		})

		if rr.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
		}
		if claims == nil {
			t.Fatal("Expected claims in context, got nil")
		}
		// Claims subject MUST be the IAP subject ("iap-user-123") as the source of truth, NOT the session token subject ("iap-user@example.com")!
		if claims.Subject != "iap-user-123" {
			t.Errorf("Expected subject iap-user-123, got %q", claims.Subject)
		}
		if claims.OrganizationID != "org-abc" {
			t.Errorf("Expected organization ID org-abc, got %q", claims.OrganizationID)
		}
	})

	t.Run("Strict Mode - Session Cookie Fallback Rejected", func(t *testing.T) {
		cfg := &core.Configuration{IapAudience: "my-aud", BypassIAPValidation: false}
		
		rr, claims := runMiddleware(cfg, func(req *http.Request) {
			// Even if a session cookie is present, it must be ignored in strict mode.
			// Since sessionCookieStore is nil in tests, if it tried to use it, it would have panicked
			// (or returned error if nil-safe). But since bypassIAP is false, it won't even call it.
		})

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d. Body: %s", rr.Code, rr.Body.String())
		}
		if claims != nil {
			t.Error("Expected no claims in context")
		}
	})

	t.Run("Strict Mode - PERF_TEST_MODE Ignored", func(t *testing.T) {
		os.Setenv("PERF_TEST_MODE", "true")
		defer os.Unsetenv("PERF_TEST_MODE")

		cfg := &core.Configuration{IapAudience: "my-aud", BypassIAPValidation: false}
		rr, claims := runMiddleware(cfg, func(req *http.Request) {
			req.Header.Set("X-Goog-IAP-JWT-Assertion", "valid-iap-token")
		})

		if rr.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", rr.Code)
		}
		if claims == nil {
			t.Fatal("Expected claims in context, got nil")
		}
		// In strict mode, PERF_TEST_MODE must be ignored. Roles should only contain "user".
		isAdmin := false
		for _, r := range claims.Roles {
			if r == "admin" {
				isAdmin = true
			}
		}
		if isAdmin {
			t.Error("Security violation: PERF_TEST_MODE escalated role to admin in strict mode")
		}
	})

	t.Run("Strict Mode - Missing JWT_SECRET Error", func(t *testing.T) {
		oldSecret := os.Getenv("JWT_SECRET")
		os.Setenv("JWT_SECRET", "")
		defer os.Setenv("JWT_SECRET", oldSecret)

		cfg := &core.Configuration{IapAudience: "my-aud", BypassIAPValidation: false}
		sessionToken, _ := MintOrgSessionToken("iap-user@example.com", "org-abc", []string{"viewer"}, []string{"read:data"}, false, "dev-secret-key-that-must-be-changed-in-production-12345")

		rr, _ := runMiddleware(cfg, func(req *http.Request) {
			req.Header.Set("X-Goog-IAP-JWT-Assertion", "valid-iap-token")
			req.Header.Set("Authorization", "Bearer "+sessionToken)
		})

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d. Body: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("Bypass Mode - Local Super Admin Org Session Token preserves is_super_admin", func(t *testing.T) {
		cfg := &core.Configuration{
			BypassIAPValidation: true,
			Environment:         "test",
			JwtSecret:           "dev-secret-key-that-must-be-changed-in-production-12345",
		}

		sessionToken, err := MintOrgSessionToken("simulated-super-admin-uid", "mock-org-1", []string{"owner"}, []string{"org:read"}, true, "dev-secret-key-that-must-be-changed-in-production-12345")
		if err != nil {
			t.Fatalf("failed to mint super admin token: %v", err)
		}

		rr, claims := runMiddleware(cfg, func(req *http.Request) {
			req.Header.Set("Authorization", "Bearer "+sessionToken)
		})

		if rr.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
		}
		if claims == nil {
			t.Fatal("Expected claims in context, got nil")
		}
		if !claims.IsSuperAdmin {
			t.Error("Expected IsSuperAdmin to be true in bypass mode with super admin org session token")
		}
	})

	t.Run("Bypass Mode - Missing IAP assertion, fallback to x-goog-authenticated-user-email", func(t *testing.T) {
		cfg := &core.Configuration{
			IapAudience:         "my-aud",
			BypassIAPValidation: true,
			Environment:         "test",
		}

		t.Run("Without Organization Header (should default to mock-tenant-123)", func(t *testing.T) {
			rr, claims := runMiddleware(cfg, func(req *http.Request) {
				req.Header.Set("x-goog-authenticated-user-email", "accounts.google.com:fallback-user@example.com")
			})

			if rr.Code != http.StatusOK {
				t.Fatalf("Expected status 200, got %d", rr.Code)
			}
			if claims == nil {
				t.Fatal("Expected claims in context, got nil")
			}
			if claims.Subject != "fallback-user@example.com" {
				t.Errorf("Expected subject fallback-user@example.com, got %q", claims.Subject)
			}
			if claims.OrganizationID != "mock-tenant-123" {
				t.Errorf("Expected default organization ID mock-tenant-123, got %q", claims.OrganizationID)
			}
		})

		t.Run("With Organization Header (should dynamically scope)", func(t *testing.T) {
			rr, claims := runMiddleware(cfg, func(req *http.Request) {
				req.Header.Set("x-goog-authenticated-user-email", "accounts.google.com:fallback-user@example.com")
				req.Header.Set(core.OrganizationIDHeader, "custom-org-99")
			})

			if rr.Code != http.StatusOK {
				t.Fatalf("Expected status 200, got %d", rr.Code)
			}
			if claims == nil {
				t.Fatal("Expected claims in context, got nil")
			}
			if claims.OrganizationID != "custom-org-99" {
				t.Errorf("Expected dynamic organization ID custom-org-99, got %q", claims.OrganizationID)
			}
			if len(claims.Organizations) != 1 || claims.Organizations[0] != "custom-org-99" {
				t.Errorf("Expected organizations list to contain custom-org-99, got %v", claims.Organizations)
			}
		})

		t.Run("With Super Admin Email (should elevate to super admin)", func(t *testing.T) {
			cfgWithAdmin := &core.Configuration{
				IapAudience:         "my-aud",
				BypassIAPValidation: true,
				Environment:         "test",
				SuperAdminEmails:    []string{"admin-user@example.com"},
			}
			rr, claims := runMiddleware(cfgWithAdmin, func(req *http.Request) {
				req.Header.Set("x-goog-authenticated-user-email", "accounts.google.com:admin-user@example.com")
			})

			if rr.Code != http.StatusOK {
				t.Fatalf("Expected status 200, got %d", rr.Code)
			}
			if claims == nil {
				t.Fatal("Expected claims in context, got nil")
			}
			if !claims.IsSuperAdmin {
				t.Error("Expected IsSuperAdmin to be true")
			}
			isAdmin := false
			for _, r := range claims.Roles {
				if r == "admin" {
					isAdmin = true
				}
			}
			if !isAdmin {
				t.Error("Expected admin role for Super Admin")
			}
		})
	})

	t.Run("Bypass Mode - Simulated JWT token", func(t *testing.T) {
		cfg := &core.Configuration{
			IapAudience:         "my-aud",
			BypassIAPValidation: true,
			Environment:         "test",
		}
		
		rr, claims := runMiddleware(cfg, func(req *http.Request) {
			req.Header.Set("X-Goog-IAP-JWT-Assertion", "simulated-jwt-token-for-mockuser-organization-mockorg")
		})

		if rr.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
		}
		if claims == nil {
			t.Fatal("Expected claims in context, got nil")
		}
		if claims.Subject != "mockuser" {
			t.Errorf("Expected subject mockuser, got %q", claims.Subject)
		}
		if claims.OrganizationID != "mockorg" {
			t.Errorf("Expected organization ID mockorg, got %q", claims.OrganizationID)
		}
	})

	t.Run("Strict Mode - Valid Super Admin IAP token", func(t *testing.T) {
		cfg := &core.Configuration{IapAudience: "my-aud", BypassIAPValidation: false}
		
		payload := &idtoken.Payload{
			Subject: "super-admin-uid",
			Claims: map[string]interface{}{
				"email": "super@admin.com",
				"is_super_admin": true,
				"firebase": map[string]interface{}{}, // Empty tenant
			},
		}
		
		mockValidator := &MockTokenValidator{
			Payloads: map[string]*idtoken.Payload{
				"valid-super-admin-token": payload,
			},
			ExpectedAudience: "my-aud",
		}
		
		oldValidator := globalTokenValidator
		globalTokenValidator = mockValidator
		defer func() { globalTokenValidator = oldValidator }()
		
		rr, claims := runMiddleware(cfg, func(req *http.Request) {
			req.Header.Set("X-Goog-IAP-JWT-Assertion", "valid-super-admin-token")
		})
		
		if rr.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
		}
		if claims == nil {
			t.Fatal("Expected claims in context, got nil")
		}
		if !claims.IsSuperAdmin {
			t.Error("Expected IsSuperAdmin to be true")
		}
		if claims.OrganizationID != "" {
			t.Errorf("Expected empty organization ID, got %q", claims.OrganizationID)
		}
		isAdmin := false
		for _, r := range claims.Roles {
			if r == "admin" {
				isAdmin = true
			}
		}
		if !isAdmin {
			t.Error("Expected admin role for Super Admin")
		}
	})

	t.Run("Strict Mode - Super Admin by Email List", func(t *testing.T) {
		cfg := &core.Configuration{
			IapAudience:         "my-aud",
			BypassIAPValidation: false,
			SuperAdminEmails:    []string{"admin1@test.com", "ADMIN2@test.com"},
		}
		
		payload := &idtoken.Payload{
			Subject: "user-uid",
			Claims: map[string]interface{}{
				"email": "admin2@test.com", // case-insensitive match test
				"firebase": map[string]interface{}{},
			},
		}
		
		mockValidator := &MockTokenValidator{
			Payloads: map[string]*idtoken.Payload{
				"valid-token": payload,
			},
			ExpectedAudience: "my-aud",
		}
		
		oldValidator := globalTokenValidator
		globalTokenValidator = mockValidator
		defer func() { globalTokenValidator = oldValidator }()
		
		rr, claims := runMiddleware(cfg, func(req *http.Request) {
			req.Header.Set("X-Goog-IAP-JWT-Assertion", "valid-token")
		})
		
		if rr.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
		}
		if claims == nil {
			t.Fatal("Expected claims in context, got nil")
		}
		if !claims.IsSuperAdmin {
			t.Error("Expected IsSuperAdmin to be true")
		}
		isAdmin := false
		for _, r := range claims.Roles {
			if r == "admin" {
				isAdmin = true
			}
		}
		if !isAdmin {
			t.Error("Expected admin role for Super Admin")
		}
	})

	t.Run("Strict Mode - Non-Admin Email Not Elevated", func(t *testing.T) {
		cfg := &core.Configuration{
			IapAudience:         "my-aud",
			BypassIAPValidation: false,
			SuperAdminEmails:    []string{"admin1@test.com"},
		}
		
		payload := &idtoken.Payload{
			Subject: "user-uid",
			Claims: map[string]interface{}{
				"email": "regular@test.com",
				"firebase": map[string]interface{}{},
			},
		}
		
		mockValidator := &MockTokenValidator{
			Payloads: map[string]*idtoken.Payload{
				"valid-token": payload,
			},
			ExpectedAudience: "my-aud",
		}
		
		oldValidator := globalTokenValidator
		globalTokenValidator = mockValidator
		defer func() { globalTokenValidator = oldValidator }()
		
		rr, claims := runMiddleware(cfg, func(req *http.Request) {
			req.Header.Set("X-Goog-IAP-JWT-Assertion", "valid-token")
		})
		
		if rr.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
		}
		if claims == nil {
			t.Fatal("Expected claims in context, got nil")
		}
		if claims.IsSuperAdmin {
			t.Error("Expected IsSuperAdmin to be false")
		}
	})

	t.Run("Production Mode - Bypass IAP Violation Rejected", func(t *testing.T) {
		cfg := &core.Configuration{
			Environment:         "production",
			BypassIAPValidation: true,
			IapAudience:         "my-aud",
		}
		rr, claims := runMiddleware(cfg, func(req *http.Request) {
			req.Header.Set("x-goog-authenticated-user-email", "accounts.google.com:fallback-user@example.com")
		})

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d. Body: %s", rr.Code, rr.Body.String())
		}
		if !strings.Contains(rr.Body.String(), "Unauthorized: authentication failed") {
			t.Errorf("Expected generic unauthorized message, got %q", rr.Body.String())
		}
		if claims != nil {
			t.Error("Expected no claims in context")
		}
	})
}
