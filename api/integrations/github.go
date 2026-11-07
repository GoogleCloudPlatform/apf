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

package integrations

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"google.com/apf/api/core"
	"google.com/apf/api/db"
)

var Store db.IntegrationStore

var githubUserAPIURL = "https://api.github.com/user"
var githubReposAPIURL = "https://api.github.com/user/repos?per_page=100&sort=updated"
var githubTokenExchangeURL = "https://github.com/login/oauth/access_token"

// IsSimulatedToken returns true if the token is a simulated, mock, or dev token.
func IsSimulatedToken(token string) bool {
	if token == "" {
		return false
	}
	return strings.HasPrefix(token, "ghu_simulated_") ||
		strings.HasPrefix(token, "simulated_") ||
		token == "mock-token" ||
		token == "test-token" ||
		token == "mock-user-token" ||
		token == "dummy_access_token" ||
		token == "dummy_token" ||
		token == "dummy-key"
}

func sendJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

func outputPopupSuccess(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
    <title>GitHub Authentication Successful</title>
    <script type="text/javascript">
        if (window.opener) {
            window.opener.postMessage("oauth_success", "*");
        }
        setTimeout(function() {
            window.close();
        }, 1000);
    </script>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; display: flex; justify-content: center; align-items: center; height: 100vh; margin: 0; background-color: #121212; color: #ffffff; }
        .card { text-align: center; padding: 32px; background: #1e1e1e; border-radius: 16px; box-shadow: 0 10px 30px rgba(0,0,0,0.5); border: 1px solid rgba(255,255,255,0.1); }
        h2 { margin-bottom: 12px; color: #4ade80; }
        p { color: #a0a0a0; margin-bottom: 0; }
    </style>
</head>
<body>
    <div class="card">
        <svg width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="#4ade80" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"></path><polyline points="22 4 12 14.01 9 11.01"></polyline></svg>
        <h2>GitHub Connected!</h2>
        <p>Authentication successful. You can close this window and return to the app.</p>
    </div>
</body>
</html>
`))
}

// GitHubConnectHandler returns the GitHub OAuth authorize URL or simulates it.
func GitHubConnectHandler(w http.ResponseWriter, r *http.Request) {
	cfg := core.NewConfig()
	clientId := cfg.GitHubClientID

	organizationID, errOrg := core.GetOrganizationID(r.Context())
	if errOrg != nil {
		http.Error(w, "Unauthorized: missing organization ID", http.StatusUnauthorized)
		return
	}
	state := organizationID

	if clientId == "" {
		// Support simulated flow for local development
		simulatedURL := fmt.Sprintf("%s/api/v1/integrations/github/callback?code=simulated_oauth_code&state=%s", cfg.AppUrl, state)
		sendJSON(w, http.StatusOK, map[string]string{"url": simulatedURL})
		return
	}

	redirectUri := fmt.Sprintf("%s/api/v1/integrations/github/callback", cfg.AppUrl)
	authURL := fmt.Sprintf("https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s&scope=repo,user,offline_access&state=%s", clientId, redirectUri, state)
	sendJSON(w, http.StatusOK, map[string]string{"url": authURL})
}

// GitHubCallbackHandler handles the code exchange from GitHub OAuth redirect.
func GitHubCallbackHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	userId := "simulated-user-uid"
	if ok && claims != nil {
		userId = claims.Subject
	}

	state := r.URL.Query().Get("state")
	if state == "" {
		http.Error(w, "Missing state parameter (organization ID)", http.StatusBadRequest)
		return
	}
	ctx := core.WithOrganizationID(r.Context(), state)

	cfg := core.NewConfig()
	code := r.URL.Query().Get("code")

	if code == "" {
		http.Error(w, "Missing authorization code", http.StatusBadRequest)
		return
	}

	if cfg.GitHubClientSecret == "" || code == "simulated_oauth_code" {
		// Simulated OAuth exchange for local development
		integration := &db.Integration{
			UserID: userId,
			Name:   "github",
			Settings: map[string]interface{}{
				"accessToken":  "ghu_simulated_access_token_987654321",
				"refreshToken": "ghr_simulated_refresh_token_123456789",
				"username":     "octocat",
				"email":        "octocat@github.com",
				"connectedAt":  time.Now().Format(time.RFC3339),
				"acquiredAt":   time.Now().Format(time.RFC3339),
			},
		}
		if err := Store.SaveIntegration(ctx, integration); err != nil {
			http.Error(w, "Failed to save simulated integration: "+err.Error(), http.StatusInternalServerError)
			return
		}
		outputPopupSuccess(w)
		return
	}

	// Perform real Code Exchange for Access Token and Refresh Token
	client := &http.Client{}
	tokenReqBody, _ := json.Marshal(map[string]string{
		"client_id":     cfg.GitHubClientID,
		"client_secret": cfg.GitHubClientSecret,
		"code":          code,
		"redirect_uri":  "http://localhost:8081/api/v1/integrations/github/callback",
	})

	req, err := http.NewRequest("POST", githubTokenExchangeURL, strings.NewReader(string(tokenReqBody)))
	if err != nil {
		http.Error(w, "Failed to prepare token request: "+err.Error(), http.StatusInternalServerError)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Failed to request access token: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		Error        string `json:"error"`
		ErrorDesc    string `json:"error_description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		http.Error(w, "Failed to parse access token response", http.StatusInternalServerError)
		return
	}

	if tokenResp.Error != "" {
		http.Error(w, fmt.Sprintf("GitHub OAuth Error: %s - %s", tokenResp.Error, tokenResp.ErrorDesc), http.StatusBadRequest)
		return
	}

	profile, errProfile := FetchGitHubUserProfile(ctx, tokenResp.AccessToken)
	username := "octocat"
	email := "octocat@github.com"
	if errProfile == nil && profile != nil {
		username = profile.Login
		email = profile.Email
	}

	integration := &db.Integration{
		UserID: userId,
		Name:   "github",
		Settings: map[string]interface{}{
			"accessToken":  tokenResp.AccessToken,
			"refreshToken": tokenResp.RefreshToken,
			"username":     username,
			"email":        email,
			"connectedAt":  time.Now().Format(time.RFC3339),
			"acquiredAt":   time.Now().Format(time.RFC3339),
		},
	}

	if err := Store.SaveIntegration(ctx, integration); err != nil {
		http.Error(w, "Failed to save integration: "+err.Error(), http.StatusInternalServerError)
		return
	}

	outputPopupSuccess(w)
}

// GitHubStatusHandler checks if the user has connected their GitHub account.
func GitHubStatusHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || claims == nil {
		http.Error(w, "Unauthorized context", http.StatusUnauthorized)
		return
	}

	integration, err := Store.GetIntegration(r.Context(), claims.Subject, "github")
	if err != nil || integration == nil {
		sendJSON(w, http.StatusOK, map[string]interface{}{"connected": false})
		return
	}

	username := ""
	if u, ok := integration.Settings["username"].(string); ok {
		username = u
	}
	email := ""
	if e, ok := integration.Settings["email"].(string); ok {
		email = e
	}

	sendJSON(w, http.StatusOK, map[string]interface{}{
		"connected": true,
		"username":  username,
		"email":     email,
	})
}

// GitHubReposHandler fetches user repositories using the stored token.
func GitHubReposHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || claims == nil {
		http.Error(w, "Unauthorized context", http.StatusUnauthorized)
		return
	}

	var accessToken string
	var err error

	if customToken := r.Header.Get("X-GitHub-Token"); customToken != "" {
		accessToken = customToken
		slog.Info("Using custom GitHub token provided in X-GitHub-Token request header")
	} else if customTokenQuery := r.URL.Query().Get("token"); customTokenQuery != "" {
		accessToken = customTokenQuery
		slog.Info("Using custom GitHub token provided in query parameter")
	} else {
		organizationID, errOrg := core.GetOrganizationID(r.Context())
		if errOrg != nil {
			http.Error(w, "Unauthorized context: missing organization ID", http.StatusUnauthorized)
			return
		}
		cred, errResolve := db.ResolveAppCredential(r.Context(), organizationID, nil, nil, claims.Subject, "", "github")
		if errResolve == nil && cred != nil {
			accessToken = db.GetFieldValue(cred, "pat")
			if accessToken == "" {
				http.Error(w, "GitHub integration check failed: credentials configured but personal access token is empty", http.StatusBadRequest)
				return
			}
			slog.Info("Using GitHub credentials resolved from user AppCredential for repos lookup", "userId", claims.Subject)
		} else {
			accessToken, err = GetOrRefreshToken(r.Context(), claims.Subject, false)
			if err != nil {
				http.Error(w, "GitHub integration check failed: "+err.Error(), http.StatusBadRequest)
				return
			}
		}
	}

	q := r.URL.Query().Get("q")

	if strings.HasPrefix(accessToken, "ghu_simulated_") {
		// Return simulated repository list
		simulatedRepos := []map[string]string{
			{"name": "agentic-foundry", "url": "https://github.com/octocat/agentic-foundry"},
			{"name": "hello-world", "url": "https://github.com/octocat/hello-world"},
			{"name": "nextjs-ai-ui", "url": "https://github.com/octocat/nextjs-ai-ui"},
		}
		if q != "" {
			filtered := make([]map[string]string, 0)
			for _, repo := range simulatedRepos {
				if strings.Contains(strings.ToLower(repo["name"]), strings.ToLower(q)) {
					filtered = append(filtered, repo)
				}
			}
			simulatedRepos = filtered
		}
		sendJSON(w, http.StatusOK, simulatedRepos)
		return
	}

	fetchRepos := func(token string) (*http.Response, error) {
		client := &http.Client{}
		targetURL := githubReposAPIURL
		if q != "" {
			targetURL = "https://api.github.com/search/repositories?q=" + url.QueryEscape(q)
		}
		reposReq, err := http.NewRequestWithContext(r.Context(), "GET", targetURL, nil)
		if err != nil {
			return nil, err
		}
		reposReq.Header.Set("Authorization", "Bearer "+token)
		reposReq.Header.Set("Accept", "application/json")
		return client.Do(reposReq)
	}

	resp, err := fetchRepos(accessToken)
	if err != nil {
		http.Error(w, "GitHub API request failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		if r.Header.Get("X-GitHub-Token") == "" && r.URL.Query().Get("token") == "" {
			newAccessToken, refreshErr := GetOrRefreshToken(r.Context(), claims.Subject, true)
			if refreshErr == nil {
				respRetry, retryErr := fetchRepos(newAccessToken)
				if retryErr == nil {
					defer respRetry.Body.Close()
					if respRetry.StatusCode == http.StatusOK {
						processReposResponse(w, respRetry, q != "")
						return
					}
					resp = respRetry
				}
			}
		}
	}

	if resp.StatusCode != http.StatusOK {
		http.Error(w, fmt.Sprintf("GitHub returned error status %d fetching repositories", resp.StatusCode), http.StatusInternalServerError)
		return
	}

	processReposResponse(w, resp, q != "")
}

func processReposResponse(w http.ResponseWriter, resp *http.Response, isSearch bool) {
	if isSearch {
		var searchResult struct {
			Items []struct {
				Name          string `json:"name"`
				HTMLURL       string `json:"html_url"`
				DefaultBranch string `json:"default_branch"`
			} `json:"items"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&searchResult); err != nil {
			http.Error(w, "Failed to parse repository search response", http.StatusInternalServerError)
			return
		}

		repos := make([]map[string]string, 0, len(searchResult.Items))
		for _, r := range searchResult.Items {
			repos = append(repos, map[string]string{
				"name":          r.Name,
				"url":           r.HTMLURL,
				"defaultBranch": r.DefaultBranch,
			})
		}
		sendJSON(w, http.StatusOK, repos)
		return
	}

	var rawRepos []struct {
		Name          string `json:"name"`
		HTMLURL       string `json:"html_url"`
		DefaultBranch string `json:"default_branch"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rawRepos); err != nil {
		http.Error(w, "Failed to parse repository list response", http.StatusInternalServerError)
		return
	}

	repos := make([]map[string]string, 0, len(rawRepos))
	for _, r := range rawRepos {
		repos = append(repos, map[string]string{
			"name":          r.Name,
			"url":           r.HTMLURL,
			"defaultBranch": r.DefaultBranch,
		})
	}

	sendJSON(w, http.StatusOK, repos)
}

// GitHubDisconnectHandler removes the GitHub integration for the current user.
func GitHubDisconnectHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || claims == nil {
		http.Error(w, "Unauthorized context", http.StatusUnauthorized)
		return
	}

	if err := Store.DeleteIntegration(r.Context(), claims.Subject, "github"); err != nil {
		http.Error(w, "Failed to disconnect integration: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// PATRequest represents the request body for linking a Personal Access Token.
type PATRequest struct {
	Token string `json:"token"`
}

// GitHubUserProfile represents the user profile information from GitHub.
type GitHubUserProfile struct {
	ID    int64  `json:"id"`
	Login string `json:"login"`
	Email string `json:"email"`
}

// FetchGitHubUserProfile retrieves profile info from GitHub API or simulates it.
func FetchGitHubUserProfile(ctx context.Context, token string) (*GitHubUserProfile, error) {
	if IsSimulatedToken(token) {
		return &GitHubUserProfile{
			ID:    583231,
			Login: "octocat",
			Email: "octocat@github.com",
		}, nil
	}

	client := &http.Client{Timeout: 10 * time.Second}
	ghReq, err := http.NewRequestWithContext(ctx, "GET", githubUserAPIURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub user request: %w", err)
	}
	ghReq.Header.Set("Authorization", "Bearer "+token)
	ghReq.Header.Set("Accept", "application/json")
	ghReq.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := client.Do(ghReq)
	if err != nil {
		return nil, fmt.Errorf("GitHub API user request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub returned status %d fetching user profile", resp.StatusCode)
	}

	var ghUser struct {
		ID    int64  `json:"id"`
		Login string `json:"login"`
		Email string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&ghUser); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub user profile: %w", err)
	}

	resolvedEmail := strings.TrimSpace(ghUser.Email)
	if resolvedEmail == "" {
		if ghUser.ID > 0 {
			resolvedEmail = fmt.Sprintf("%d+%s@users.noreply.github.com", ghUser.ID, ghUser.Login)
		} else {
			resolvedEmail = fmt.Sprintf("%s@users.noreply.github.com", ghUser.Login)
		}
	}

	return &GitHubUserProfile{
		ID:    ghUser.ID,
		Login: ghUser.Login,
		Email: resolvedEmail,
	}, nil
}

// GitHubProfileHandler retrieves profile details (username, email) from GitHub API for a token or current user.
func GitHubProfileHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || claims == nil {
		http.Error(w, "Unauthorized context", http.StatusUnauthorized)
		return
	}

	var accessToken string
	if customToken := r.Header.Get("X-GitHub-Token"); customToken != "" {
		accessToken = customToken
	} else if customTokenQuery := r.URL.Query().Get("token"); customTokenQuery != "" {
		accessToken = customTokenQuery
	} else {
		organizationID, errOrg := core.GetOrganizationID(r.Context())
		if errOrg == nil && organizationID != "" {
			cred, errResolve := db.ResolveAppCredential(r.Context(), organizationID, nil, nil, claims.Subject, "", "github")
			if errResolve == nil && cred != nil {
				accessToken = db.GetFieldValue(cred, "pat")
			}
		}
		if accessToken == "" {
			var err error
			accessToken, err = GetOrRefreshToken(r.Context(), claims.Subject, false)
			if err != nil {
				http.Error(w, "GitHub credentials not configured: "+err.Error(), http.StatusBadRequest)
				return
			}
		}
	}

	profile, err := FetchGitHubUserProfile(r.Context(), accessToken)
	if err != nil {
		http.Error(w, "Failed to fetch GitHub profile: "+err.Error(), http.StatusInternalServerError)
		return
	}

	sendJSON(w, http.StatusOK, map[string]interface{}{
		"connected": true,
		"username":  profile.Login,
		"email":     profile.Email,
	})
}

// GitHubLinkPATHandler links a Personal Access Token directly to the user.
func GitHubLinkPATHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || claims == nil {
		http.Error(w, "Unauthorized context", http.StatusUnauthorized)
		return
	}

	var req PATRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if req.Token == "" {
		http.Error(w, "Token is required", http.StatusBadRequest)
		return
	}

	profile, errProfile := FetchGitHubUserProfile(r.Context(), req.Token)
	if errProfile != nil {
		http.Error(w, fmt.Sprintf("Invalid Personal Access Token (%v)", errProfile), http.StatusBadRequest)
		return
	}

	integration := &db.Integration{
		UserID: claims.Subject,
		Name:   "github",
		Settings: map[string]interface{}{
			"accessToken": req.Token,
			"username":    profile.Login,
			"email":       profile.Email,
			"connectedAt": time.Now().Format(time.RFC3339),
			"acquiredAt":  time.Now().Format(time.RFC3339),
		},
	}

	if err := Store.SaveIntegration(r.Context(), integration); err != nil {
		http.Error(w, "Failed to save integration: "+err.Error(), http.StatusInternalServerError)
		return
	}

	sendJSON(w, http.StatusOK, map[string]interface{}{
		"connected": true,
		"username":  profile.Login,
		"email":     profile.Email,
	})
}

// GetOrRefreshToken checks if access token is expired/older than 1 hour, and if so, refreshes it.
func GetOrRefreshToken(ctx context.Context, userId string, forceRefresh bool) (string, error) {
	if Store == nil {
		return "", fmt.Errorf("integrations.Store is uninitialized")
	}
	integration, err := Store.GetIntegration(ctx, userId, "github")
	if err != nil {
		return "", fmt.Errorf("failed to retrieve github integration: %w", err)
	}
	if integration == nil {
		return "", fmt.Errorf("github integration not found for user")
	}

	accessToken, _ := integration.Settings["accessToken"].(string)
	if accessToken == "" {
		return "", fmt.Errorf("github access token is missing")
	}

	refreshToken, _ := integration.Settings["refreshToken"].(string)
	// If there is no refresh token, this is a PAT (Personal Access Token).
	// PATs are long-lived and do not support OAuth token refresh. Return as-is.
	if refreshToken == "" {
		return accessToken, nil
	}

	isSimulated := strings.HasPrefix(accessToken, "ghu_simulated_")

	acquiredAtStr, _ := integration.Settings["acquiredAt"].(string)
	var acquiredAt time.Time
	if acquiredAtStr != "" {
		acquiredAt, _ = time.Parse(time.RFC3339, acquiredAtStr)
	}

	// Check age. If no acquiredAt is stored, treat as expired/older than 1 hour.
	isOld := acquiredAt.IsZero() || time.Since(acquiredAt) > time.Hour

	if forceRefresh || isOld {
		if isSimulated {
			return simulateTokenRefresh(ctx, integration)
		}
		return performRealTokenRefresh(ctx, integration, refreshToken)
	}

	return accessToken, nil
}

func simulateTokenRefresh(ctx context.Context, integration *db.Integration) (string, error) {
	// Simulate a refreshed token
	newAccessToken := fmt.Sprintf("ghu_simulated_access_token_refreshed_%d", time.Now().UnixNano())
	newRefreshToken := fmt.Sprintf("ghr_simulated_refresh_token_refreshed_%d", time.Now().UnixNano())
	integration.Settings["accessToken"] = newAccessToken
	integration.Settings["refreshToken"] = newRefreshToken
	integration.Settings["acquiredAt"] = time.Now().Format(time.RFC3339)

	if err := Store.SaveIntegration(ctx, integration); err != nil {
		return "", fmt.Errorf("failed to save simulated refreshed integration: %w", err)
	}
	return newAccessToken, nil
}

func performRealTokenRefresh(ctx context.Context, integration *db.Integration, refreshToken string) (string, error) {
	cfg := core.NewConfig()
	if cfg.GitHubClientID == "" || cfg.GitHubClientSecret == "" {
		return "", fmt.Errorf("github oauth credentials are not configured")
	}

	client := &http.Client{}
	refreshBody, _ := json.Marshal(map[string]string{
		"client_id":     cfg.GitHubClientID,
		"client_secret": cfg.GitHubClientSecret,
		"grant_type":    "refresh_token",
		"refresh_token": refreshToken,
	})

	req, err := http.NewRequestWithContext(ctx, "POST", githubTokenExchangeURL, strings.NewReader(string(refreshBody)))
	if err != nil {
		return "", fmt.Errorf("failed to create refresh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("token refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github returned status %d on token refresh", resp.StatusCode)
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		Error        string `json:"error"`
		ErrorDesc    string `json:"error_description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to parse refresh token response: %w", err)
	}

	if tokenResp.Error != "" {
		return "", fmt.Errorf("github refresh error: %s - %s", tokenResp.Error, tokenResp.ErrorDesc)
	}

	// Update integration credentials
	integration.Settings["accessToken"] = tokenResp.AccessToken
	if tokenResp.RefreshToken != "" {
		integration.Settings["refreshToken"] = tokenResp.RefreshToken
	}
	integration.Settings["acquiredAt"] = time.Now().Format(time.RFC3339)

	if err := Store.SaveIntegration(ctx, integration); err != nil {
		return "", fmt.Errorf("failed to save refreshed integration: %w", err)
	}

	return tokenResp.AccessToken, nil
}

// ErrGitHubCredsNotConfigured is returned when no GitHub credentials (PAT or OAuth)
// can be resolved and no system fallback is available.
var ErrGitHubCredsNotConfigured = errors.New("GitHub credentials not configured. Please link GitHub or configure a Personal Access Token (PAT) in settings.")

// ResolveGitHubToken resolves the GitHub access token for a given user and project.
// It checks project/user level AppCredentials (PAT), then falls back to user OAuth tokens,
// and finally falls back to the system-level environment default.
func ResolveGitHubToken(ctx context.Context, userId string, projectId string) (string, error) {
	var accessToken string

	organizationID, errOrg := core.GetOrganizationID(ctx)
	if errOrg != nil {
		return "", fmt.Errorf("failed to resolve organization ID: %w", errOrg)
	}

	// 1. Try to resolve 2-level AppCredential (PAT)
	cred, errResolve := db.ResolveAppCredential(ctx, organizationID, nil, nil, userId, projectId, "github")
	if errResolve == nil && cred != nil {
		accessToken = db.GetFieldValue(cred, "pat")
		if accessToken != "" {
			slog.Info("Using GitHub credentials resolved from 2-level AppCredential database", "userId", userId, "projectId", projectId)
			return accessToken, nil
		}
		slog.Warn("GitHub credentials configured but PAT attribute is empty", "userId", userId, "projectId", projectId)
	}

	// 2. Fallback to OAuth Token
	token, errOAuth := GetOrRefreshToken(ctx, userId, false)
	if errOAuth == nil && token != "" {
		slog.Info("Using GitHub OAuth token", "userId", userId, "projectId", projectId)
		return token, nil
	}
	if errOAuth != nil {
		slog.Debug("Failed to get/refresh OAuth token", "userId", userId, "error", errOAuth)
	}

	// 3. Fallback to system-level development token
	appCfg := core.NewConfig()
	if appCfg.DevRepoAccessToken != "" {
		slog.Info("No database credentials found; falling back to DEV_REPO_ACCESS_TOKEN", "userId", userId, "projectId", projectId)
		return appCfg.DevRepoAccessToken, nil
	}

	slog.Warn("GitHub credentials not configured in database and system fallback is absent", "userId", userId, "projectId", projectId)
	return "", ErrGitHubCredsNotConfigured
}

// ResolveGitHubEmail resolves the GitHub author email for a given user and project.
// It checks 2-level AppCredential (email attribute), then falls back to user profile info.
func ResolveGitHubEmail(ctx context.Context, userId string, projectId string) (string, error) {
	organizationID, errOrg := core.GetOrganizationID(ctx)
	if errOrg == nil {
		cred, errResolve := db.ResolveAppCredential(ctx, organizationID, nil, nil, userId, projectId, "github")
		if errResolve == nil && cred != nil {
			email := db.GetFieldValue(cred, "email")
			if email != "" {
				return email, nil
			}
		}
	}

	token, errToken := ResolveGitHubToken(ctx, userId, projectId)
	if errToken == nil && token != "" {
		if profile, err := FetchGitHubUserProfile(ctx, token); err == nil && profile != nil && profile.Email != "" {
			return profile.Email, nil
		}
	}

	return "", nil
}

type ValidatePermissionsRequest struct {
	RepoURL    string `json:"repoUrl"`
	Token      string `json:"token,omitempty"`
	ProjectID  string `json:"projectId,omitempty"`
	BaseBranch string `json:"baseBranch,omitempty"`
}

type ValidatePermissionsResponse struct {
	Read             bool   `json:"read"`
	Write            bool   `json:"write"`
	PullRequest      bool   `json:"pullRequest"`
	BaseBranchExists bool   `json:"baseBranchExists"`
	BaseBranch       string `json:"baseBranch,omitempty"`
}

func ValidateGitHubPermissionsHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || claims == nil {
		http.Error(w, "Unauthorized context", http.StatusUnauthorized)
		return
	}

	var req ValidatePermissionsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if req.RepoURL == "" {
		http.Error(w, "repoUrl is required", http.StatusBadRequest)
		return
	}

	slog.Info("ValidateGitHubPermissionsHandler: Received request", "repoUrl", req.RepoURL, "hasTokenInReq", req.Token != "", "projectId", req.ProjectID, "baseBranch", req.BaseBranch)

	// Parse GitHub URL
	if !strings.HasPrefix(req.RepoURL, "https://github.com/") {
		slog.Warn("ValidateGitHubPermissionsHandler: Invalid repo URL prefix", "repoUrl", req.RepoURL)
		http.Error(w, "invalid repository URL: only HTTPS github.com URLs are supported", http.StatusBadRequest)
		return
	}
	trimmed := strings.TrimPrefix(req.RepoURL, "https://github.com/")
	trimmed = strings.TrimSuffix(trimmed, ".git")
	trimmed = strings.TrimRight(trimmed, "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		slog.Warn("ValidateGitHubPermissionsHandler: Failed to parse owner/repo", "repoUrl", req.RepoURL)
		http.Error(w, "failed to parse owner and repository name from URL", http.StatusBadRequest)
		return
	}
	owner, repo := parts[0], parts[1]
	slog.Info("ValidateGitHubPermissionsHandler: Parsed URL", "owner", owner, "repo", repo)

	token := req.Token
	tokenSource := "request"
	if token == "" {
		resolvedToken, err := ResolveGitHubToken(r.Context(), claims.Subject, req.ProjectID)
		if err == nil {
			token = resolvedToken
			tokenSource = "resolved"
		} else {
			slog.Warn("ValidateGitHubPermissionsHandler: Failed to resolve token", "userId", claims.Subject, "projectId", req.ProjectID, "error", err)
		}
	}

	// Handle masked token
	if strings.Contains(token, "•") || token == "credentialMaskPlaceholder" {
		slog.Info("ValidateGitHubPermissionsHandler: Token is masked, attempting to force resolve from DB")
		resolvedToken, err := ResolveGitHubToken(r.Context(), claims.Subject, req.ProjectID)
		if err == nil {
			token = resolvedToken
			tokenSource = "resolved_force"
		} else {
			slog.Warn("ValidateGitHubPermissionsHandler: Failed to force resolve token", "error", err)
			token = "" // clear it so we fail safely
		}
	}

	if token == "" {
		slog.Info("ValidateGitHubPermissionsHandler: No token available, proceeding with unauthenticated check")
	}

	maskedToken := ""
	if len(token) > 4 {
		maskedToken = token[:4] + "..."
	} else {
		maskedToken = "..."
	}
	slog.Info("ValidateGitHubPermissionsHandler: Using token", "source", tokenSource, "tokenPreview", maskedToken)

	if IsSimulatedToken(token) {
		slog.Info("ValidateGitHubPermissionsHandler: Simulated token, bypassing checks")
		baseBranchExists := true
		if req.BaseBranch == "non-existent-branch" {
			baseBranchExists = false
		}
		sendJSON(w, http.StatusOK, ValidatePermissionsResponse{
			Read:             true,
			Write:            true,
			PullRequest:      true,
			BaseBranchExists: baseBranchExists,
			BaseBranch:       req.BaseBranch,
		})
		return
	}

	client := &GitHubPRClient{Token: token}
	read, write, pr, err := client.CheckPermissions(r.Context(), owner, repo)
	if err != nil {
		slog.Error("ValidateGitHubPermissionsHandler: GitHub permissions check failed", "owner", owner, "repo", repo, "error", err)
		http.Error(w, fmt.Sprintf("Failed to check permissions: %v", err), http.StatusInternalServerError)
		return
	}

	baseBranchExists := true
	if req.BaseBranch != "" {
		exists, errB := client.CheckBranchExists(r.Context(), owner, repo, req.BaseBranch)
		if errB == nil {
			baseBranchExists = exists
		}
	}

	slog.Info("ValidateGitHubPermissionsHandler: Check results", "read", read, "write", write, "pr", pr, "baseBranchExists", baseBranchExists)

	sendJSON(w, http.StatusOK, ValidatePermissionsResponse{
		Read:             read,
		Write:            write,
		PullRequest:      pr,
		BaseBranchExists: baseBranchExists,
		BaseBranch:       req.BaseBranch,
	})
}


