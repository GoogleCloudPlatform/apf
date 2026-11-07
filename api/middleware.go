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
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"google.com/apf/api/core"
)

// resolveClaims parses credentials (IAP or Bearer Token) and extracts user claims, validating against Org Session JWT first.
func resolveClaims(r *http.Request) (*core.UserClaims, error) {
	claims, err := resolveClaimsInternal(r)
	if err != nil {
		return nil, err
	}
	if claims != nil && appConfig != nil && !claims.IsSuperAdmin {
		email := claims.Username
		if email != "" {
			for _, adminEmail := range appConfig.SuperAdminEmails {
				if strings.EqualFold(email, adminEmail) {
					claims.IsSuperAdmin = true
					hasAdmin := false
					for _, role := range claims.Roles {
						if role == "admin" {
							hasAdmin = true
							break
						}
					}
					if !hasAdmin {
						claims.Roles = append(claims.Roles, "admin")
					}
					break
				}
			}
		}
	}
	return claims, nil
}

func resolveClaimsInternal(r *http.Request) (*core.UserClaims, error) {
	iapAssertion := r.Header.Get("X-Goog-IAP-JWT-Assertion")
	iapEmailHeader := r.Header.Get("x-goog-authenticated-user-email")

	authHeader := r.Header.Get("Authorization")
	var sessionToken string
	if authHeader != "" {
		parts := strings.Split(authHeader, " ")
		if len(parts) == 2 && parts[0] == "Bearer" {
			sessionToken = parts[1]
			if sessionToken == "undefined" || sessionToken == "null" || sessionToken == "iap-authenticated" {
				sessionToken = ""
			}
		}
	}

	bypassIAP := true
	expectedAudience := ""
	if appConfig != nil {
		bypassIAP = appConfig.BypassIAPValidation
		expectedAudience = appConfig.IapAudience
	}

	var primaryClaims *core.UserClaims

	perfTestHeader := r.Header.Get("X-Perf-Test-Key")
	if perfTestHeader == "" {
		perfTestHeader = r.Header.Get("PERF_TEST_KEY")
	}

	// 1. Resolve Primary Identity (PERF_TEST_KEY or IAP)
	if appConfig != nil && appConfig.PerfTestKey != "" && perfTestHeader != "" && perfTestHeader == appConfig.PerfTestKey {
		slog.Info("Authenticating request via valid PERF_TEST_KEY header")
		email := strings.TrimPrefix(iapEmailHeader, "accounts.google.com:")
		if email == "" {
			email = "developer@agentic-foundry.local"
		}
		orgID := r.Header.Get(core.OrganizationIDHeader)
		if orgID == "" {
			orgID = "org-high-council"
		}
		roles := []string{string(core.RoleOrgAdmin), string(core.RoleOrgOwner), string(core.RoleProjectCreator)}

		primaryClaims = &core.UserClaims{
			Subject:        email,
			Username:       email,
			Roles:          roles,
			OrganizationID: orgID,
			Organizations:  []string{orgID},
		}
	} else if bypassIAP {
		// Bypass Mode (Local Dev)
		env := "production"
		if appConfig != nil {
			env = appConfig.Environment
		}
		if env != "local" && env != "test" {
			return nil, fmt.Errorf("security violation: IAP bypass is only allowed in local or test environments")
		}

		if iapAssertion != "" {
			if strings.HasPrefix(iapAssertion, "simulated-jwt-") {
				simClaims, err := globalAuthenticator.VerifyToken(iapAssertion)
				if err != nil {
					return nil, fmt.Errorf("invalid simulated IAP token: %w", err)
				}
				primaryClaims = simClaims
			}
		}

		if primaryClaims == nil && iapEmailHeader != "" {
			slog.Warn("Bypassing IAP: trusting unverified header x-goog-authenticated-user-email")
			email := strings.TrimPrefix(iapEmailHeader, "accounts.google.com:")
			roles := []string{string(core.RoleOrgAdmin)}
			if os.Getenv("PERF_TEST_MODE") == "true" {
				roles = append(roles, string(core.RoleOrgOwner))
			}

			// Dynamically scope mock tenant to match the requested organization, or use safe default
			orgID := r.Header.Get(core.OrganizationIDHeader)
			if orgID == "" {
				orgID = "mock-tenant-123"
			}

			primaryClaims = &core.UserClaims{
				Subject:        email,
				Username:       email,
				Roles:          roles,
				Token:          iapAssertion,
				OrganizationID: orgID,
				Organizations:  []string{orgID},
			}
		}
	} else {
		// Strict Mode (Production)
		if iapAssertion == "" {
			return nil, fmt.Errorf("missing IAP assertion")
		}

		// Temporarily decode and log JWT claims to debug audience mismatch
		if parts := strings.Split(iapAssertion, "."); len(parts) > 1 {
			decoded, err := base64.RawURLEncoding.DecodeString(parts[1])
			if err == nil {
				slog.Info("DEBUG JWT Claims", "payload", string(decoded), "expectedAudience", expectedAudience)
			} else {
				slog.Warn("DEBUG JWT Claims decode failed", "err", err)
			}
		}

		payload, err := globalTokenValidator.Validate(r.Context(), iapAssertion, expectedAudience)
		if err != nil {
			return nil, fmt.Errorf("IAP token validation failed: %w", err)
		}
		if payload == nil {
			return nil, fmt.Errorf("IAP token validation returned empty payload")
		}

		subject := payload.Subject
		email, _ := payload.Claims["email"].(string)

		var tenantID string
		if firebaseClaim, ok := payload.Claims["firebase"].(map[string]interface{}); ok {
			tenantID, _ = firebaseClaim["tenant"].(string)
		}

		// Strictly no PERF_TEST_MODE in production/strict mode
		roles := []string{"user"}
		isSuperAdmin := false
		if tenantID == "" {
			if isa, ok := payload.Claims["is_super_admin"].(bool); ok && isa {
				isSuperAdmin = true
				roles = []string{"admin"}
			} else if appConfig != nil && email != "" {
				for _, adminEmail := range appConfig.SuperAdminEmails {
					if strings.EqualFold(email, adminEmail) {
						isSuperAdmin = true
						roles = []string{"admin"}
						break
					}
				}
			}
		}

		primaryClaims = &core.UserClaims{
			Subject:        subject,
			Username:       email,
			Roles:          roles,
			OrganizationID: tenantID,
			Organizations:  []string{tenantID},
			Token:          iapAssertion,
			IsSuperAdmin:   isSuperAdmin,
		}
	}

	// 2. Resolve Session / Bearer Token
	if sessionToken != "" {
		jwtSecret := ""
		if appConfig != nil {
			jwtSecret = appConfig.JwtSecret
		}
		if jwtSecret == "" {
			if !bypassIAP {
				return nil, fmt.Errorf("secure configuration error: JWT_SECRET must be configured in production")
			}
			jwtSecret = "dev-secret-key-that-must-be-changed-in-production-12345"
		}

		// Try Org Session JWT first
		orgClaims, err := VerifyOrgSessionToken(sessionToken, jwtSecret)
		if err == nil {
			// It is a valid Org Session JWT.
			// If we have a primary identity (from IAP), we MUST verify they match (Session Hijacking protection)
			if primaryClaims != nil {
				cleanSubject := strings.TrimPrefix(primaryClaims.Subject, "accounts.google.com:")
				cleanEmail := strings.TrimPrefix(primaryClaims.Username, "accounts.google.com:")
				cleanOrgSub := strings.TrimPrefix(orgClaims.Subject, "accounts.google.com:")

				if cleanOrgSub == "" || cleanSubject == "" {
					return nil, fmt.Errorf("invalid empty identity in claims")
				}

				if cleanOrgSub != cleanSubject && cleanOrgSub != cleanEmail {
					return nil, fmt.Errorf("identity mismatch between IAP and session token")
				}
			}

			// Return merged claims. Crucially: propagate primaryClaims.Subject as source of truth if IAP is active!
			subject := orgClaims.Subject
			username := orgClaims.Subject
			isSuperAdmin := orgClaims.IsSuperAdmin
			if primaryClaims != nil {
				subject = primaryClaims.Subject
				if primaryClaims.Username != "" {
					username = primaryClaims.Username
				}
				isSuperAdmin = primaryClaims.IsSuperAdmin
			} else {
				// Local / Bypass Mode: mirror super admin status if orgClaims indicate super admin
				if orgClaims.IsSuperAdmin || orgClaims.Subject == "simulated-super-admin-uid" || orgClaims.Subject == "super-admin" {
					isSuperAdmin = true
				}
			}

			return &core.UserClaims{
				Subject:        subject, // Source of truth identity
				Username:       username,
				OrganizationID: orgClaims.OrgID,
				Organizations:  []string{orgClaims.OrgID},
				Roles:          orgClaims.Roles,
				Permissions:    orgClaims.Permissions,
				Token:          sessionToken,
				IsSuperAdmin:   isSuperAdmin,
			}, nil
		}

		// If it's not a valid Org Session JWT, and we are in bypass mode, it might be a Firebase Token.
		if bypassIAP {
			firebaseClaims, err := globalAuthenticator.VerifyToken(sessionToken)
			if err == nil {
				// It's a valid Firebase token.
				if primaryClaims != nil {
					cleanSubject := strings.TrimPrefix(primaryClaims.Subject, "accounts.google.com:")
					cleanEmail := strings.TrimPrefix(primaryClaims.Username, "accounts.google.com:")
					cleanFbSub := strings.TrimPrefix(firebaseClaims.Subject, "accounts.google.com:")
					cleanFbEmail := strings.TrimPrefix(firebaseClaims.Username, "accounts.google.com:")

					if cleanFbSub == "" || cleanSubject == "" {
						return nil, fmt.Errorf("invalid empty identity in claims")
					}

					emailMatch := false
					if cleanFbEmail != "" && cleanEmail != "" && cleanFbEmail == cleanEmail {
						emailMatch = true
					}
					if cleanFbEmail != "" && cleanSubject != "" && cleanFbEmail == cleanSubject {
						emailMatch = true
					}

					if cleanFbSub != cleanSubject && cleanFbSub != cleanEmail && !emailMatch {
						return nil, fmt.Errorf("identity mismatch between IAP and Firebase token")
					}
				}
				firebaseClaims.Token = sessionToken
				return firebaseClaims, nil
			}
			return nil, fmt.Errorf("invalid token: %w", err)
		}

		// In strict mode, if they passed a token but it's not a valid Org Session JWT, reject.
		return nil, fmt.Errorf("invalid session token: %w", err)
	}

	// 3. No session token. If we have primary claims (IAP), return them.
	if primaryClaims != nil {
		return primaryClaims, nil
	}

	return nil, fmt.Errorf("no credentials provided")
}

// authMiddleware intercepts Authorization headers and injects user context.
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Allow OPTIONS (preflight)
		if r.Method == "OPTIONS" {
			next(w, r)
			return
		}

		claims, err := resolveClaims(r)
		if err == nil {
			ctx := context.WithValue(r.Context(), core.UserClaimsKey, claims)
			next(w, r.WithContext(ctx))
			return
		}

		// Only fall back to session cookie if IAP is bypassed (local dev)
		bypassIAP := true
		if appConfig != nil {
			bypassIAP = appConfig.BypassIAPValidation
		}

		if bypassIAP {
			session, sessErr := sessionCookieStore.Get(r, "session-auth")
			if sessErr == nil {
				if claimsCookie, ok := session.Values["claims"].(*core.UserClaims); ok {
					ctx := context.WithValue(r.Context(), core.UserClaimsKey, claimsCookie)
					next(w, r.WithContext(ctx))
					return
				}
			}
		}

		slog.Warn("Authentication failed", "err", err, "path", r.URL.Path, "ip", r.RemoteAddr)
		http.Error(w, "Unauthorized: authentication failed", http.StatusUnauthorized)
	}
}

// optionalAuthMiddleware intercepts Authorization headers and injects user context if present, but does not reject unauthenticated requests.
func optionalAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "OPTIONS" {
			next(w, r)
			return
		}

		claims, err := resolveClaims(r)
		if err == nil {
			ctx := context.WithValue(r.Context(), core.UserClaimsKey, claims)
			next(w, r.WithContext(ctx))
			return
		}

		next(w, r)
	}
}

var orgIDRegex = regexp.MustCompile("^[a-zA-Z0-9-_]+$")

// orgScopeMiddleware verifies that the request is scoped to an organization the user is authorized to access.
func orgScopeMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
		if !ok || claims == nil {
			http.Error(w, "Unauthorized: missing user claims", http.StatusUnauthorized)
			return
		}

		if !claims.IsSuperAdmin && claims.OrganizationID == "" {
			http.Error(w, "Forbidden: user token is not scoped to a valid tenant", http.StatusForbidden)
			return
		}

		reqOrgID := r.Header.Get(core.OrganizationIDHeader)
		if reqOrgID == "" {
			http.Error(w, fmt.Sprintf("Forbidden: %s header is required", core.OrganizationIDHeader), http.StatusForbidden)
			return
		}

		if !orgIDRegex.MatchString(reqOrgID) {
			http.Error(w, fmt.Sprintf("Forbidden: Invalid %s format", core.OrganizationIDHeader), http.StatusBadRequest)
			return
		}

		// Validate organization access
		if !claims.HasAccessToOrg(reqOrgID) {
			http.Error(w, "Forbidden: User is not a member of this organization", http.StatusForbidden)
			return
		}

		// Inject validated organization ID into context
		ctx := core.WithOrganizationID(r.Context(), reqOrgID)
		next(w, r.WithContext(ctx))
	}
}

// projectAccessMiddleware verifies that the user has access to the project.
func projectAccessMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectId := r.PathValue("id")
		claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
		if !ok {
			http.Error(w, "Unauthorized context", http.StatusUnauthorized)
			return
		}

		if claims.IsSuperAdmin {
			next(w, r)
			return
		}

		organizationID, errOrg := core.GetOrganizationID(r.Context())
		if errOrg != nil {
			http.Error(w, "Unauthorized context: missing organization ID", http.StatusUnauthorized)
			return
		}

		project, err := store.GetProject(r.Context(), organizationID, projectId)
		if err != nil {
			http.Error(w, "Project not found", http.StatusNotFound)
			return
		}

		// Super Admin bypasses ownership/membership checks, but project must exist
		if claims.IsSuperAdmin {
			next(w, r)
			return
		}

		isMember := false
		if project.Members != nil && (*project.Members)[claims.Subject] != "" {
			isMember = true
		}
		if project.OwnerId != nil && *project.OwnerId != claims.Subject && !isMember {
			http.Error(w, "Forbidden: Not an owner or member of this project", http.StatusForbidden)
			return
		}

		next(w, r)
	}
}

// corsMiddleware handles CORS headers for all requests.
func corsMiddleware(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Only allow CORS in local environment (detected by emulator host or explicit env var)
		isLocal := os.Getenv("FIRESTORE_EMULATOR_HOST") != "" || os.Getenv("ALLOW_CORS") == "true"

		if isLocal {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, x-goog-authenticated-user-email, x-goog-authenticated-user-id, X-Organization-Id")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
		}

		next.ServeHTTP(w, r)
	}
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
	body       []byte
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

func (lrw *loggingResponseWriter) Write(b []byte) (int, error) {
	if lrw.statusCode >= 400 {
		if len(lrw.body) < 1024 {
			limit := 1024 - len(lrw.body)
			if len(b) < limit {
				limit = len(b)
			}
			lrw.body = append(lrw.body, b[:limit]...)
		}
	}
	return lrw.ResponseWriter.Write(b)
}

func (lrw *loggingResponseWriter) Flush() {
	if flusher, ok := lrw.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lrw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		start := time.Now()

		next.ServeHTTP(lrw, r)

		duration := time.Since(start)
		if lrw.statusCode >= 500 {
			args := []any{
				"method", r.Method,
				"path", r.URL.Path,
				"status", lrw.statusCode,
				"duration", duration,
			}
			if len(lrw.body) > 0 {
				args = append(args, "error", strings.TrimSpace(string(lrw.body)))
			}
			slog.Error("HTTP Request", args...)
		} else if lrw.statusCode >= 400 {
			args := []any{
				"method", r.Method,
				"path", r.URL.Path,
				"status", lrw.statusCode,
				"duration", duration,
			}
			if len(lrw.body) > 0 {
				args = append(args, "error", strings.TrimSpace(string(lrw.body)))
			}
			slog.Warn("HTTP Request", args...)
		} else {
			slog.Info("HTTP Request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", lrw.statusCode,
				"duration", duration,
			)
		}
	})
}

// orgPermissionMiddleware verifies that the user has the required granular permission in the active organization.
func orgPermissionMiddleware(requiredPermission string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			orgID := r.PathValue("orgId")
			if orgID == "" {
				http.Error(w, "Bad Request: Missing organization ID", http.StatusBadRequest)
				return
			}

			claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
			if !ok || claims == nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Super Admin bypasses database permission checks
			if claims.IsSuperAdmin {
				ctx := context.WithValue(r.Context(), core.OrganizationIDKey, orgID)
				next(w, r.WithContext(ctx))
				return
			}

			// 1. Fetch active user from database to verify role
			user, err := userStore.GetUser(r.Context(), orgID, claims.Subject)
			if err != nil {
				http.Error(w, "Forbidden: User not found in organization", http.StatusForbidden)
				return
			}

			// 2. Verify user has the required permission using core.HasPermission helper
			if !core.HasPermission(r.Context(), []string{string(user.Role)}, requiredPermission) {
				http.Error(w, fmt.Sprintf("Forbidden: Missing required permission %q", requiredPermission), http.StatusForbidden)
				return
			}

			// Inject organization ID into context
			ctx := context.WithValue(r.Context(), core.OrganizationIDKey, orgID)
			next(w, r.WithContext(ctx))
		}
	}
}
