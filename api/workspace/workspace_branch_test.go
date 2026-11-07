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
	"fmt"
	"hash/crc32"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"google.com/apf/api/core"
	"google.com/apf/api/db"
)

func TestGenerateDynamicBranchName(t *testing.T) {
	subjectID := "accounts.google.com:107764536614723260958"
	expectedCRC := fmt.Sprintf("%08x", crc32.ChecksumIEEE([]byte(subjectID)))

	tests := []struct {
		name      string
		prefix    string
		email     string
		subjectID string
		expected  string
	}{
		{
			name:      "valid email with prefix",
			prefix:    "specs",
			email:     "sumeetsing@google.com",
			subjectID: subjectID,
			expected:  fmt.Sprintf("specs/sumeetsing-%s", expectedCRC),
		},
		{
			name:      "valid email without prefix",
			prefix:    "",
			email:     "sumeetsing@google.com",
			subjectID: subjectID,
			expected:  fmt.Sprintf("sumeetsing-%s", expectedCRC),
		},
		{
			name:      "empty email returns empty string",
			prefix:    "specs",
			email:     "",
			subjectID: subjectID,
			expected:  "",
		},
		{
			name:      "email missing @ symbol returns sanitized email with prefix",
			prefix:    "specs",
			email:     "admin",
			subjectID: subjectID,
			expected:  "specs/admin",
		},
		{
			name:      "subject same as email and is valid email returns specs/username-crc32",
			prefix:    "specs",
			email:     "org-owner@the-high-council.com",
			subjectID: "org-owner@the-high-council.com",
			expected:  fmt.Sprintf("specs/org-owner-%08x", crc32.ChecksumIEEE([]byte("org-owner@the-high-council.com"))),
		},
		{
			name:      "subject same as email but not valid email returns specs/crc32",
			prefix:    "specs",
			email:     "user123",
			subjectID: "user123",
			expected:  fmt.Sprintf("specs/%08x", crc32.ChecksumIEEE([]byte("user123"))),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateDynamicBranchName(tt.prefix, tt.email, tt.subjectID)
			if got != tt.expected {
				t.Errorf("GenerateDynamicBranchName(%q, %q, %q) = %q, expected %q",
					tt.prefix, tt.email, tt.subjectID, got, tt.expected)
			}
		})
	}
}

func TestResolveUserSpecBranchName_WithEmail(t *testing.T) {
	subjectID := "accounts.google.com:107764536614723260958"
	claims := &core.UserClaims{
		Subject:  subjectID,
		Username: "sumeetsing@google.com",
	}
	ctx := context.WithValue(context.Background(), core.UserClaimsKey, claims)

	expectedCRC := fmt.Sprintf("%08x", crc32.ChecksumIEEE([]byte(subjectID)))
	expectedBranch := fmt.Sprintf("specs/sumeetsing-%s", expectedCRC)

	branchName := ResolveUserSpecBranchName(ctx, "specs", claims.Subject, "")
	if branchName != expectedBranch {
		t.Errorf("expected %q, got %q", expectedBranch, branchName)
	}
}

func TestResolveUserSpecBranchName_FallbackUserStore(t *testing.T) {
	origUserStore := UserStore
	defer func() { UserStore = origUserStore }()

	mockStore := &mockGitUserStore{
		user: &db.User{
			Email: "admin@google.com",
		},
	}
	UserStore = mockStore

	subjectID := "user-without-email-claim"
	claims := &core.UserClaims{
		Subject:        subjectID,
		OrganizationID: "test-org",
	}
	ctx := context.WithValue(context.Background(), core.UserClaimsKey, claims)
	ctx = core.WithOrganizationID(ctx, "test-org")

	expectedCRC := fmt.Sprintf("%08x", crc32.ChecksumIEEE([]byte(subjectID)))
	expectedBranch := fmt.Sprintf("specs/admin-%s", expectedCRC)

	branchName := ResolveUserSpecBranchName(ctx, "specs", claims.Subject, "")
	if branchName != expectedBranch {
		t.Errorf("expected %q, got %q", expectedBranch, branchName)
	}
}

func TestResolveUserSpecBranchName_SubjectHasEmail(t *testing.T) {
	subjectID := "testuser@example.com"
	claims := &core.UserClaims{
		Subject: subjectID,
	}
	ctx := context.WithValue(context.Background(), core.UserClaimsKey, claims)

	expectedCRC := fmt.Sprintf("%08x", crc32.ChecksumIEEE([]byte(subjectID)))
	expectedBranch := fmt.Sprintf("specs/testuser-%s", expectedCRC)

	branchName := ResolveUserSpecBranchName(ctx, "specs", claims.Subject, "")
	if branchName != expectedBranch {
		t.Errorf("expected %q, got %q", expectedBranch, branchName)
	}
}

func TestResolveUserSpecBranchName_LegacyBranchFallback(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "legacy-branch-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	subjectID := "legacy-user-123"
	email := "legacy@example.com"

	// Create legacy local branch ref on disk
	legacyBranch := "specs/legacy"
	legacyRefPath := filepath.Join(tempDir, ".git", "refs", "heads", "specs", "legacy")
	_ = os.MkdirAll(filepath.Dir(legacyRefPath), 0755)
	_ = os.WriteFile(legacyRefPath, []byte("0123456789abcdef0123456789abcdef01234567\n"), 0644)

	claims := &core.UserClaims{
		Subject:  subjectID,
		Username: email,
	}
	ctx := context.WithValue(context.Background(), core.UserClaimsKey, claims)

	resolved := ResolveUserSpecBranchName(ctx, "specs", subjectID, tempDir)
	if resolved != legacyBranch {
		t.Errorf("expected legacy branch %q to be preserved, got %q", legacyBranch, resolved)
	}
}

func TestResolveSpecBranch_FilesystemInspect(t *testing.T) {
	origStore := db.ProjectStoreInstance
	origSessionStore := SessionStore
	defer func() {
		db.ProjectStoreInstance = origStore
		SessionStore = origSessionStore
	}()

	tmpDir, err := os.MkdirTemp("", "git-fs-branch-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	orgID := "org-test"
	userID := "user-123"
	projID := "proj-456"

	wsDir := filepath.Join(tmpDir, orgID, userID, projID)
	customFSBranch := "custom-feature-branch"
	headPath := filepath.Join(wsDir, ".git", "HEAD")
	_ = os.MkdirAll(filepath.Dir(headPath), 0755)
	_ = os.WriteFile(headPath, []byte("ref: refs/heads/"+customFSBranch+"\n"), 0644)

	origAppConfig := AppConfig
	AppConfig = &core.Configuration{WorkspaceMountPath: tmpDir}
	defer func() { AppConfig = origAppConfig }()

	db.ProjectStoreInstance = &mockGitProjectStore{
		project: &db.Project{
			ID: projID,
		},
	}
	SessionStore = &mockGitSessionStore{sessions: []db.Session{}}

	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/projects/%s/git/sync", projID), nil)
	ctx := core.WithOrganizationID(context.Background(), orgID)
	req = req.WithContext(ctx)

	_, spec, err := resolveSpecBranch(req.Context(), orgID, projID, userID, req)
	if err != nil {
		t.Fatalf("resolveSpecBranch returned error: %v", err)
	}
	if spec != customFSBranch {
		t.Errorf("expected spec branch to be inspected from filesystem %q, got %q", customFSBranch, spec)
	}
}

func TestResolveSpecBranch_FilesystemOnMain_ResolvesDynamicSpecBranch(t *testing.T) {
	origStore := db.ProjectStoreInstance
	origSessionStore := SessionStore
	defer func() {
		db.ProjectStoreInstance = origStore
		SessionStore = origSessionStore
	}()

	tmpDir, err := os.MkdirTemp("", "git-fs-main-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	orgID := "org-test"
	userID := "sumeetsing@google.com"
	projID := "proj-456"

	wsDir := filepath.Join(tmpDir, orgID, userID, projID)
	headPath := filepath.Join(wsDir, ".git", "HEAD")
	_ = os.MkdirAll(filepath.Dir(headPath), 0755)
	_ = os.WriteFile(headPath, []byte("ref: refs/heads/main\n"), 0644)

	origAppConfig := AppConfig
	AppConfig = &core.Configuration{WorkspaceMountPath: tmpDir}
	defer func() { AppConfig = origAppConfig }()

	db.ProjectStoreInstance = &mockGitProjectStore{
		project: &db.Project{
			ID: projID,
		},
	}
	SessionStore = &mockGitSessionStore{sessions: []db.Session{}}

	claims := &core.UserClaims{
		Subject:  userID,
		Username: userID,
	}
	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/projects/%s/git/sync", projID), nil)
	ctx := core.WithOrganizationID(context.Background(), orgID)
	ctx = context.WithValue(ctx, core.UserClaimsKey, claims)
	req = req.WithContext(ctx)

	base, spec, err := resolveSpecBranch(req.Context(), orgID, projID, userID, req)
	if err != nil {
		t.Fatalf("resolveSpecBranch returned error: %v", err)
	}
	if base != "main" {
		t.Errorf("expected base branch 'main', got %q", base)
	}
	if spec == "main" {
		t.Fatalf("expected dynamic spec branch, but got 'main'")
	}
	expectedCRC := fmt.Sprintf("%08x", crc32.ChecksumIEEE([]byte(userID)))
	expectedSpec := fmt.Sprintf("specs/sumeetsing-%s", expectedCRC)
	if spec != expectedSpec {
		t.Errorf("expected spec branch %q, got %q", expectedSpec, spec)
	}

	// Also test ResolveSpecBranchNoRequest
	baseNoReq, specNoReq, errNoReq := ResolveSpecBranchNoRequest(ctx, orgID, projID, userID, "")
	if errNoReq != nil {
		t.Fatalf("ResolveSpecBranchNoRequest returned error: %v", errNoReq)
	}
	if baseNoReq != "main" {
		t.Errorf("expected base branch 'main', got %q", baseNoReq)
	}
	if specNoReq != expectedSpec {
		t.Errorf("expected spec branch %q, got %q", expectedSpec, specNoReq)
	}
}

type mockGitUserStore struct {
	db.UserStore
	user *db.User
	err  error
}

func (m *mockGitUserStore) GetUser(ctx context.Context, organizationID, id string) (*db.User, error) {
	return m.user, m.err
}
