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
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreatePullRequest_Success(t *testing.T) {
	origAPIURL := githubAPIURL
	defer func() { githubAPIURL = origAPIURL }()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected method POST, got %s", r.Method)
		}
		if r.URL.Path != "/repos/test-owner/test-repo/pulls" {
			t.Errorf("Expected path /repos/test-owner/test-repo/pulls, got %s", r.URL.Path)
		}

		var payload PullRequestPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("Failed to decode payload: %v", err)
		}
		if payload.Title != "Test PR" {
			t.Errorf("Expected title 'Test PR', got '%s'", payload.Title)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(PullRequestResponse{
			HTMLURL: "https://github.com/test-owner/test-repo/pull/1",
			Number:  1,
		})
	}))
	defer ts.Close()

	githubAPIURL = ts.URL

	client := &GitHubPRClient{Token: "test-token"}
	resp, err := client.CreatePullRequest(context.Background(), "test-owner", "test-repo", PullRequestPayload{
		Title: "Test PR",
	})

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("Expected non-nil response")
	}
	if resp.HTMLURL != "https://github.com/test-owner/test-repo/pull/1" {
		t.Errorf("Expected HTMLURL 'https://github.com/test-owner/test-repo/pull/1', got '%s'", resp.HTMLURL)
	}
	if resp.Number != 1 {
		t.Errorf("Expected PR number 1, got %d", resp.Number)
	}
}

func TestCreatePullRequest_SimpleError(t *testing.T) {
	origAPIURL := githubAPIURL
	defer func() { githubAPIURL = origAPIURL }()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"message": "Bad Request"}`))
	}))
	defer ts.Close()

	githubAPIURL = ts.URL

	client := &GitHubPRClient{Token: "test-token"}
	_, err := client.CreatePullRequest(context.Background(), "test-owner", "test-repo", PullRequestPayload{})

	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	expected := "GitHub API error (400): Bad Request"
	if err.Error() != expected {
		t.Errorf("Expected error '%s', got '%s'", expected, err.Error())
	}
}

func TestCreatePullRequest_ErrorsArray_Message(t *testing.T) {
	origAPIURL := githubAPIURL
	defer func() { githubAPIURL = origAPIURL }()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(`{
			"message": "Validation Failed",
			"errors": [
				{"resource": "PullRequest", "code": "custom", "message": "A pull request already exists for sumeetsing:main."}
			]
		}`))
	}))
	defer ts.Close()

	githubAPIURL = ts.URL

	client := &GitHubPRClient{Token: "test-token"}
	_, err := client.CreatePullRequest(context.Background(), "test-owner", "test-repo", PullRequestPayload{})

	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	expected := "GitHub API error (422): Validation Failed - A pull request already exists for sumeetsing:main."
	if err.Error() != expected {
		t.Errorf("Expected error '%s', got '%s'", expected, err.Error())
	}
}

func TestCreatePullRequest_ErrorsArray_CodeField(t *testing.T) {
	origAPIURL := githubAPIURL
	defer func() { githubAPIURL = origAPIURL }()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(`{
			"message": "Validation Failed",
			"errors": [
				{"resource": "PullRequest", "field": "head", "code": "invalid"},
				{"resource": "PullRequest", "code": "missing_field"}
			]
		}`))
	}))
	defer ts.Close()

	githubAPIURL = ts.URL

	client := &GitHubPRClient{Token: "test-token"}
	_, err := client.CreatePullRequest(context.Background(), "test-owner", "test-repo", PullRequestPayload{})

	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	expected := "GitHub API error (422): Validation Failed - head field: invalid, missing_field"
	if err.Error() != expected {
		t.Errorf("Expected error '%s', got '%s'", expected, err.Error())
	}
}

func TestCreatePullRequest_ErrorsString(t *testing.T) {
	origAPIURL := githubAPIURL
	defer func() { githubAPIURL = origAPIURL }()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{
			"message": "Error occurred",
			"errors": "some detailed string error"
		}`))
	}))
	defer ts.Close()

	githubAPIURL = ts.URL

	client := &GitHubPRClient{Token: "test-token"}
	_, err := client.CreatePullRequest(context.Background(), "test-owner", "test-repo", PullRequestPayload{})

	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	expected := "GitHub API error (400): Error occurred - some detailed string error"
	if err.Error() != expected {
		t.Errorf("Expected error '%s', got '%s'", expected, err.Error())
	}
}

func TestCreatePullRequest_UnknownResponse(t *testing.T) {
	origAPIURL := githubAPIURL
	defer func() { githubAPIURL = origAPIURL }()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`Internal Server Error`))
	}))
	defer ts.Close()

	githubAPIURL = ts.URL

	client := &GitHubPRClient{Token: "test-token"}
	_, err := client.CreatePullRequest(context.Background(), "test-owner", "test-repo", PullRequestPayload{})

	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	expected := "GitHub API error (500): unknown response body"
	if err.Error() != expected {
		t.Errorf("Expected error '%s', got '%s'", expected, err.Error())
	}
}

func TestIsDuplicatePullRequestError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "standard string error duplicate",
			err:      errors.New("github api error: a pull request already exists for owner:branch"),
			expected: true,
		},
		{
			name:     "paraphrased string error duplicate",
			err:      errors.New("github api error: pull request is already in progress for owner:branch"),
			expected: true,
		},
		{
			name:     "already open string error",
			err:      errors.New("already open"),
			expected: true,
		},
		{
			name:     "generic already exists string error",
			err:      errors.New("already exists"),
			expected: true,
		},
		{
			name: "GitHubError duplicate in errors array",
			err: &GitHubError{
				StatusCode: http.StatusUnprocessableEntity,
				Message:    "Validation Failed",
				Errors:     []string{"A pull request already exists for owner:branch."},
			},
			expected: true,
		},
		{
			name: "GitHubError duplicate in errors array - in progress",
			err: &GitHubError{
				StatusCode: http.StatusUnprocessableEntity,
				Message:    "Validation Failed",
				Errors:     []string{"Pull request is already in progress for owner:branch."},
			},
			expected: true,
		},
		{
			name:     "unrelated error",
			err:      errors.New("some other random error"),
			expected: false,
		},
		{
			name: "GitHubError unrelated error",
			err: &GitHubError{
				StatusCode: http.StatusBadRequest,
				Message:    "Bad Request",
				Errors:     []string{"Invalid field"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsDuplicatePullRequestError(tt.err)
			if result != tt.expected {
				t.Errorf("IsDuplicatePullRequestError() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestCheckPermissions_PROnly_DoesNotGrantWrite(t *testing.T) {
	origAPIURL := githubAPIURL
	defer func() { githubAPIURL = origAPIURL }()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. GET /repos/test-owner/test-repo
		if r.Method == "GET" && r.URL.Path == "/repos/test-owner/test-repo" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"permissions": {"pull": true, "push": false}}`))
			return
		}

		// 2. POST /repos/test-owner/test-repo/pulls
		if r.Method == "POST" && r.URL.Path == "/repos/test-owner/test-repo/pulls" {
			w.WriteHeader(http.StatusUnprocessableEntity) // 422 indicates PR access
			return
		}
	}))
	defer ts.Close()

	githubAPIURL = ts.URL

	client := &GitHubPRClient{Token: "test-token"}
	read, write, pr, err := client.CheckPermissions(context.Background(), "test-owner", "test-repo")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !read {
		t.Errorf("Expected read to be true")
	}
	if write {
		t.Errorf("Expected write to be false (PR permission alone must not grant write access)")
	}
	if !pr {
		t.Errorf("Expected pr to be true")
	}
}

func TestMergeBranch_Success_201(t *testing.T) {
	origAPIURL := githubAPIURL
	defer func() { githubAPIURL = origAPIURL }()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected method POST, got %s", r.Method)
		}
		if r.URL.Path != "/repos/test-owner/test-repo/merges" {
			t.Errorf("Expected path /repos/test-owner/test-repo/merges, got %s", r.URL.Path)
		}

		var payload map[string]string
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("Failed to decode payload: %v", err)
		}
		if payload["base"] != "main" || payload["head"] != "feature" || payload["commit_message"] != "merge msg" {
			t.Errorf("Unexpected payload: %v", payload)
		}

		w.WriteHeader(http.StatusCreated)
	}))
	defer ts.Close()

	githubAPIURL = ts.URL

	client := &GitHubPRClient{Token: "test-token"}
	err := client.MergeBranch(context.Background(), "test-owner", "test-repo", "main", "feature", "merge msg")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestMergeBranch_Success_204(t *testing.T) {
	origAPIURL := githubAPIURL
	defer func() { githubAPIURL = origAPIURL }()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	githubAPIURL = ts.URL

	client := &GitHubPRClient{Token: "test-token"}
	err := client.MergeBranch(context.Background(), "test-owner", "test-repo", "main", "feature", "merge msg")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestMergeBranch_Conflict_409(t *testing.T) {
	origAPIURL := githubAPIURL
	defer func() { githubAPIURL = origAPIURL }()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(`{"message": "Merge conflict"}`))
	}))
	defer ts.Close()

	githubAPIURL = ts.URL

	client := &GitHubPRClient{Token: "test-token"}
	err := client.MergeBranch(context.Background(), "test-owner", "test-repo", "main", "feature", "merge msg")

	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	var ghErr *GitHubError
	if !errors.As(err, &ghErr) {
		t.Fatalf("Expected GitHubError, got %T", err)
	}
	if ghErr.StatusCode != http.StatusConflict {
		t.Errorf("Expected status 409, got %d", ghErr.StatusCode)
	}
	if ghErr.Message != "Merge conflict" {
		t.Errorf("Expected message 'Merge conflict', got '%s'", ghErr.Message)
	}
}

func TestMergeBranch_Error_404(t *testing.T) {
	origAPIURL := githubAPIURL
	defer func() { githubAPIURL = origAPIURL }()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message": "Branch not found"}`))
	}))
	defer ts.Close()

	githubAPIURL = ts.URL

	client := &GitHubPRClient{Token: "test-token"}
	err := client.MergeBranch(context.Background(), "test-owner", "test-repo", "main", "feature", "merge msg")

	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	var ghErr *GitHubError
	if !errors.As(err, &ghErr) {
		t.Fatalf("Expected GitHubError, got %T", err)
	}
	if ghErr.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", ghErr.StatusCode)
	}
	if ghErr.Message != "Branch not found" {
		t.Errorf("Expected message 'Branch not found', got '%s'", ghErr.Message)
	}
}
