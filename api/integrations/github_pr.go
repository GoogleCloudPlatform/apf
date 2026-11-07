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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
)

type GitHubError struct {
	StatusCode int
	Message    string
	Errors     []string
}

func (e *GitHubError) Error() string {
	if len(e.Errors) > 0 {
		return fmt.Sprintf("GitHub API error (%d): %s - %s", e.StatusCode, e.Message, strings.Join(e.Errors, ", "))
	}
	return fmt.Sprintf("GitHub API error (%d): %s", e.StatusCode, e.Message)
}

func IsDuplicatePullRequestError(err error) bool {
	var ghErr *GitHubError
	if errors.As(err, &ghErr) {
		if ghErr.StatusCode == http.StatusUnprocessableEntity {
			for _, subErr := range ghErr.Errors {
				if isDuplicateMsg(subErr) {
					return true
				}
			}
		}
	}
	return isDuplicateMsg(err.Error())
}

func isDuplicateMsg(msg string) bool {
	m := strings.ToLower(msg)
	return strings.Contains(m, "a pull request already exists") ||
		strings.Contains(m, "pull request is already in progress") ||
		strings.Contains(m, "already exists") ||
		strings.Contains(m, "already open") ||
		strings.Contains(m, "already in progress")
}

type GitHubPRClient struct {
	Token string
}

type PullRequestPayload struct {
	Title string `json:"title"`
	Head  string `json:"head"`
	Base  string `json:"base"`
	Body  string `json:"body"`
}

type PullRequestResponse struct {
	HTMLURL string `json:"html_url"`
	Number  int    `json:"number"`
}

var githubAPIURL = "https://api.github.com"

func (c *GitHubPRClient) CreatePullRequest(ctx context.Context, owner, repo string, pr PullRequestPayload) (*PullRequestResponse, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/pulls", githubAPIURL, owner, repo)
	bodyBytes, err := json.Marshal(pr)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", owner)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		var errData map[string]interface{}
		if errDecode := json.NewDecoder(resp.Body).Decode(&errData); errDecode == nil {
			if msg, ok := errData["message"].(string); ok {
				var detailMsgs []string
				if errs, ok := errData["errors"].([]interface{}); ok {
					for _, item := range errs {
						switch errVal := item.(type) {
						case map[string]interface{}:
							if subMsg, ok := errVal["message"].(string); ok {
								detailMsgs = append(detailMsgs, subMsg)
							} else if code, ok := errVal["code"].(string); ok {
								if field, ok := errVal["field"].(string); ok {
									detailMsgs = append(detailMsgs, fmt.Sprintf("%s field: %s", field, code))
								} else {
									detailMsgs = append(detailMsgs, code)
								}
							} else {
								if bytes, err := json.Marshal(errVal); err == nil {
									detailMsgs = append(detailMsgs, string(bytes))
								}
							}
						case string:
							detailMsgs = append(detailMsgs, errVal)
						default:
							detailMsgs = append(detailMsgs, fmt.Sprintf("%v", errVal))
						}
					}
				} else if errsStr, ok := errData["errors"].(string); ok {
					detailMsgs = append(detailMsgs, errsStr)
				}
				if len(detailMsgs) > 0 {
					return nil, &GitHubError{
						StatusCode: resp.StatusCode,
						Message:    msg,
						Errors:     detailMsgs,
					}
				}
				return nil, &GitHubError{
					StatusCode: resp.StatusCode,
					Message:    msg,
				}
			}
		}
		return nil, &GitHubError{
			StatusCode: resp.StatusCode,
			Message:    "unknown response body",
		}
	}

	var prResp PullRequestResponse
	if err := json.NewDecoder(resp.Body).Decode(&prResp); err != nil {
		return nil, err
	}
	return &prResp, nil
}

func (c *GitHubPRClient) ListPullRequests(ctx context.Context, owner, repo, head string) ([]PullRequestResponse, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/pulls?head=%s&state=open", githubAPIURL, owner, repo, head)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", owner)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API list pulls error (%d)", resp.StatusCode)
	}

	var prs []PullRequestResponse
	if err := json.NewDecoder(resp.Body).Decode(&prs); err != nil {
		return nil, err
	}
	return prs, nil
}

func (c *GitHubPRClient) CheckBranchExists(ctx context.Context, owner, repo, branch string) (bool, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/branches/%s", githubAPIURL, owner, repo, branch)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", owner)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	}
	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}
	return false, fmt.Errorf("GitHub API check branch error (%d)", resp.StatusCode)
}

// MergeBranch merges head branch into base branch.
// Returns nil on success (201 Created or 204 No Content).
func (c *GitHubPRClient) MergeBranch(ctx context.Context, owner, repo string, base, head, commitMessage string) error {
	url := fmt.Sprintf("%s/repos/%s/%s/merges", githubAPIURL, owner, repo)
	payload := map[string]string{
		"base":           base,
		"head":           head,
		"commit_message": commitMessage,
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", owner)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		var errData map[string]interface{}
		if errDecode := json.NewDecoder(resp.Body).Decode(&errData); errDecode == nil {
			if msg, ok := errData["message"].(string); ok {
				return &GitHubError{
					StatusCode: resp.StatusCode,
					Message:    msg,
				}
			}
		}
		return &GitHubError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("unexpected status code: %d", resp.StatusCode),
		}
	}

	return nil
}

type BranchCompareFile struct {
	Filename  string `json:"filename"`
	Status    string `json:"status"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
}

type BranchCompareResponse struct {
	Status       string              `json:"status"`
	AheadBy      int                 `json:"ahead_by"`
	BehindBy     int                 `json:"behind_by"`
	TotalCommits int                 `json:"total_commits"`
	Files        []BranchCompareFile `json:"files"`
	Mergeable    bool                `json:"mergeable"`
}

func (c *GitHubPRClient) CompareBranches(ctx context.Context, owner, repo, base, head string) (*BranchCompareResponse, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/compare/%s...%s", githubAPIURL, owner, repo, base, head)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", owner)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API compare failed with status code %d", resp.StatusCode)
	}

	var res BranchCompareResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	res.Mergeable = res.Status != "conflict"
	return &res, nil
}

func SetGitHubAPIURLForTesting(url string) {
	githubAPIURL = url
}

func (c *GitHubPRClient) CheckPermissions(ctx context.Context, owner, repo string) (read bool, write bool, pr bool, err error) {
	// 1. Check Read and Write via GET /repos/{owner}/{repo}
	url := fmt.Sprintf("%s/repos/%s/%s", githubAPIURL, owner, repo)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, false, false, err
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", owner)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, false, false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusNotFound {
		return false, false, false, nil
	}
	if resp.StatusCode != http.StatusOK {
		return false, false, false, fmt.Errorf("failed to fetch repository details (GitHub status %d)", resp.StatusCode)
	}

	var repoData struct {
		Permissions struct {
			Pull bool `json:"pull"`
			Push bool `json:"push"`
		} `json:"permissions"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&repoData); err != nil {
		return false, false, false, err
	}

	read = true // If we got 200 OK, we definitely have read access
	userHasWrite := repoData.Permissions.Push

	if !read || c.Token == "" {
		return read, false, false, nil
	}

	// Verify if the token actually has write access (handles scoped PATs where user has write but token doesn't)
	write = false
	if userHasWrite {
		writeUrl := fmt.Sprintf("%s/repos/%s/%s/contents/permission-check-dummy-file-do-not-create", githubAPIURL, owner, repo)
		writeReq, err := http.NewRequestWithContext(ctx, "PUT", writeUrl, bytes.NewReader([]byte("{}")))
		if err != nil {
			return read, false, false, err
		}
		writeReq.Header.Set("Authorization", "Bearer "+c.Token)
		writeReq.Header.Set("Accept", "application/vnd.github+json")
		writeReq.Header.Set("X-GitHub-Api-Version", "2022-11-28")
		writeReq.Header.Set("Content-Type", "application/json")
		writeReq.Header.Set("User-Agent", owner)

		writeResp, err := http.DefaultClient.Do(writeReq)
		if err != nil {
			return read, false, false, err
		}
		defer writeResp.Body.Close()

		if writeResp.StatusCode == http.StatusUnprocessableEntity {
			write = true
		} else {
			slog.Info("CheckPermissions: Token write check rejected", "status", writeResp.StatusCode)
			write = false
		}
	}


	// 2. Check PR permission via dummy POST /repos/{owner}/{repo}/pulls
	prUrl := fmt.Sprintf("%s/repos/%s/%s/pulls", githubAPIURL, owner, repo)
	dummyPayload := map[string]string{
		"title": "Permission Check Dummy PR",
		"head":  "non-existent-head-branch-for-permission-check",
		"base":  "non-existent-base-branch-for-permission-check",
	}
	bodyBytes, err := json.Marshal(dummyPayload)
	if err != nil {
		return read, write, false, err
	}

	prReq, err := http.NewRequestWithContext(ctx, "POST", prUrl, bytes.NewReader(bodyBytes))
	if err != nil {
		return read, write, false, err
	}
	prReq.Header.Set("Authorization", "Bearer "+c.Token)
	prReq.Header.Set("Accept", "application/vnd.github+json")
	prReq.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	prReq.Header.Set("Content-Type", "application/json")
	prReq.Header.Set("User-Agent", owner)

	prResp, err := http.DefaultClient.Do(prReq)
	if err != nil {
		return read, write, false, err
	}
	defer prResp.Body.Close()

	if prResp.StatusCode == http.StatusUnprocessableEntity {
		pr = true
	} else if prResp.StatusCode == http.StatusForbidden || prResp.StatusCode == http.StatusNotFound || prResp.StatusCode == http.StatusUnauthorized {
		pr = false
	} else {
		if prResp.StatusCode == http.StatusCreated {
			pr = true
		}
	}

	return read, write, pr, nil
}


