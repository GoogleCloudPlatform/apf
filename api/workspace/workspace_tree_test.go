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
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"google.com/apf/api/core"
)

func TestGetProjectFileTreeHandler_SubPath(t *testing.T) {
	tempDir := t.TempDir()
	origConfig := AppConfig
	AppConfig = &core.Configuration{WorkspaceMountPath: tempDir}
	defer func() { AppConfig = origConfig }()

	orgID := "test-organization"
	userID := "user@google.com"
	projectID := "proj-tree-test"

	cloneDir := filepath.Join(tempDir, orgID, userID, projectID)
	srcDir := filepath.Join(cloneDir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(cloneDir, "README.md"), []byte("readme"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "main.go"), []byte("package main"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	t.Run("Root tree request", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/projects/"+projectID+"/files", nil)
		req.SetPathValue("id", projectID)
		claims := &core.UserClaims{Subject: userID}
		req = withTestCtx(req, claims)

		rr := httptest.NewRecorder()
		GetProjectFileTreeHandler(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rr.Code)
		}

		var rootNode FileNode
		if err := json.Unmarshal(rr.Body.Bytes(), &rootNode); err != nil {
			t.Fatalf("failed to parse json: %v", err)
		}

		if len(rootNode.Children) != 2 {
			t.Errorf("expected 2 children at root, got %d", len(rootNode.Children))
		}
	})

	t.Run("SubPath request", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/projects/"+projectID+"/files?path=src", nil)
		req.SetPathValue("id", projectID)
		claims := &core.UserClaims{Subject: userID}
		req = withTestCtx(req, claims)

		rr := httptest.NewRecorder()
		GetProjectFileTreeHandler(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rr.Code)
		}

		var subNode FileNode
		if err := json.Unmarshal(rr.Body.Bytes(), &subNode); err != nil {
			t.Fatalf("failed to parse json: %v", err)
		}

		if subNode.Name != "src" {
			t.Errorf("expected node name 'src', got '%s'", subNode.Name)
		}
		if subNode.Path != "src" {
			t.Errorf("expected node path 'src', got '%s'", subNode.Path)
		}
		if len(subNode.Children) != 1 || subNode.Children[0].Name != "main.go" {
			t.Errorf("expected 1 child 'main.go', got %+v", subNode.Children)
		}
		if subNode.Children[0].Path != "src/main.go" {
			t.Errorf("expected child path 'src/main.go', got '%s'", subNode.Children[0].Path)
		}
	})
}

func TestGetProjectFileTreeHandler_ExcludeSymlinks(t *testing.T) {
	tempDir := t.TempDir()
	origConfig := AppConfig
	AppConfig = &core.Configuration{WorkspaceMountPath: tempDir}
	defer func() { AppConfig = origConfig }()

	orgID := "test-organization"
	userID := "user@google.com"
	projectID := "proj-tree-symlinks"

	cloneDir := filepath.Join(tempDir, orgID, userID, projectID)
	srcDir := filepath.Join(cloneDir, "src")
	extraDir := filepath.Join(cloneDir, "extra")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	if err := os.MkdirAll(extraDir, 0755); err != nil {
		t.Fatalf("failed to create extra dir: %v", err)
	}

	// Create regular files
	if err := os.WriteFile(filepath.Join(cloneDir, "README.md"), []byte("readme"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "main.go"), []byte("package main"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(extraDir, "config.json"), []byte("{}"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	// Create symlink to file in root
	if err := os.Symlink(filepath.Join(cloneDir, "README.md"), filepath.Join(cloneDir, "README.link")); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}
	// Create symlink to directory in root
	if err := os.Symlink(extraDir, filepath.Join(cloneDir, "extra-link")); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}
	// Create broken symlink in root
	if err := os.Symlink(filepath.Join(cloneDir, "non-existent-file"), filepath.Join(cloneDir, "broken.link")); err != nil {
		t.Fatalf("failed to create broken symlink: %v", err)
	}
	// Create symlink to file in nested folder
	if err := os.Symlink(filepath.Join(srcDir, "main.go"), filepath.Join(srcDir, "main.link.go")); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	t.Run("Excludes symlinks from root tree", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/projects/"+projectID+"/files", nil)
		req.SetPathValue("id", projectID)
		claims := &core.UserClaims{Subject: userID}
		req = withTestCtx(req, claims)

		rr := httptest.NewRecorder()
		GetProjectFileTreeHandler(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rr.Code)
		}

		var rootNode FileNode
		if err := json.Unmarshal(rr.Body.Bytes(), &rootNode); err != nil {
			t.Fatalf("failed to parse json: %v", err)
		}

		// Root should have 3 children: README.md, src, extra.
		// README.link, extra-link, broken.link must be excluded.
		if len(rootNode.Children) != 3 {
			t.Fatalf("expected 3 children at root (README.md, src, extra), got %d: %+v", len(rootNode.Children), rootNode.Children)
		}

		for _, child := range rootNode.Children {
			if child.Name == "README.link" || child.Name == "extra-link" || child.Name == "broken.link" {
				t.Errorf("symlink %s was not excluded from root children", child.Name)
			}
			if child.Name == "src" {
				// src should only contain main.go, not main.link.go
				if len(child.Children) != 1 || child.Children[0].Name != "main.go" {
					t.Errorf("expected only 'main.go' in src, got: %+v", child.Children)
				}
			}
		}
	})

	t.Run("Rejects subpath that is a symlink", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/projects/"+projectID+"/files?path=extra-link", nil)
		req.SetPathValue("id", projectID)
		claims := &core.UserClaims{Subject: userID}
		req = withTestCtx(req, claims)

		rr := httptest.NewRecorder()
		GetProjectFileTreeHandler(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400 for symlink path, got %d", rr.Code)
		}
	})
}

func TestGetProjectFileTreeHandler_HiddenFilesAndGit(t *testing.T) {
	tempDir := t.TempDir()
	origConfig := AppConfig
	AppConfig = &core.Configuration{WorkspaceMountPath: tempDir}
	defer func() { AppConfig = origConfig }()

	orgID := "test-organization"
	userID := "user@google.com"
	projectID := "proj-tree-hidden-files"

	cloneDir := filepath.Join(tempDir, orgID, userID, projectID)
	gitDir := filepath.Join(cloneDir, ".git")
	gitObjectsDir := filepath.Join(gitDir, "objects")
	githubDir := filepath.Join(cloneDir, ".github", "workflows")
	agentsDir := filepath.Join(cloneDir, ".agents")
	srcDir := filepath.Join(cloneDir, "src")

	for _, d := range []string{gitObjectsDir, githubDir, agentsDir, srcDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}
	}

	// Create visible and hidden files
	if err := os.WriteFile(filepath.Join(cloneDir, "README.md"), []byte("readme"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cloneDir, ".gitignore"), []byte("node_modules\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cloneDir, ".env"), []byte("SECRET=123\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(githubDir, "ci.yml"), []byte("name: CI\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentsDir, "config.json"), []byte("{}\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(gitDir, "config"), []byte("[core]\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(gitObjectsDir, "sample_obj"), []byte("binary"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	t.Run("Includes hidden files and folders while excluding .git in root tree", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/projects/"+projectID+"/files", nil)
		req.SetPathValue("id", projectID)
		claims := &core.UserClaims{Subject: userID}
		req = withTestCtx(req, claims)

		rr := httptest.NewRecorder()
		GetProjectFileTreeHandler(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rr.Code)
		}

		var rootNode FileNode
		if err := json.Unmarshal(rr.Body.Bytes(), &rootNode); err != nil {
			t.Fatalf("failed to parse json: %v", err)
		}

		childMap := make(map[string]FileNode)
		for _, child := range rootNode.Children {
			childMap[child.Name] = child
		}

		// .git must NOT be present
		if _, exists := childMap[".git"]; exists {
			t.Errorf("expected .git directory to be excluded from root file tree")
		}

		// Hidden files .gitignore and .env MUST be present
		if node, exists := childMap[".gitignore"]; !exists || node.Type != "file" {
			t.Errorf("expected .gitignore to be present as a file, got: %+v", node)
		}
		if node, exists := childMap[".env"]; !exists || node.Type != "file" {
			t.Errorf("expected .env to be present as a file, got: %+v", node)
		}

		// Hidden folders .github and .agents MUST be present
		if node, exists := childMap[".github"]; !exists || node.Type != "folder" {
			t.Errorf("expected .github to be present as a folder, got: %+v", node)
		}
		if node, exists := childMap[".agents"]; !exists || node.Type != "folder" {
			t.Errorf("expected .agents to be present as a folder, got: %+v", node)
		}

		// Regular files / folders must still be present
		if node, exists := childMap["README.md"]; !exists || node.Type != "file" {
			t.Errorf("expected README.md to be present as a file, got: %+v", node)
		}
		if node, exists := childMap["src"]; !exists || node.Type != "folder" {
			t.Errorf("expected src to be present as a folder, got: %+v", node)
		}
	})

	t.Run("Supports subpath query for hidden directories", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/projects/"+projectID+"/files?path=.github/workflows", nil)
		req.SetPathValue("id", projectID)
		claims := &core.UserClaims{Subject: userID}
		req = withTestCtx(req, claims)

		rr := httptest.NewRecorder()
		GetProjectFileTreeHandler(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rr.Code)
		}

		var subNode FileNode
		if err := json.Unmarshal(rr.Body.Bytes(), &subNode); err != nil {
			t.Fatalf("failed to parse json: %v", err)
		}

		if subNode.Name != "workflows" {
			t.Errorf("expected node name 'workflows', got '%s'", subNode.Name)
		}
		if len(subNode.Children) != 1 || subNode.Children[0].Name != "ci.yml" {
			t.Errorf("expected 1 child 'ci.yml', got %+v", subNode.Children)
		}
	})

	t.Run("Rejects subpath query for .git directory", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/projects/"+projectID+"/files?path=.git", nil)
		req.SetPathValue("id", projectID)
		claims := &core.UserClaims{Subject: userID}
		req = withTestCtx(req, claims)

		rr := httptest.NewRecorder()
		GetProjectFileTreeHandler(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400 for .git path query, got %d", rr.Code)
		}
	})
}

func TestUploadProjectFilesHandler(t *testing.T) {
	tempDir := t.TempDir()
	origConfig := AppConfig
	AppConfig = &core.Configuration{WorkspaceMountPath: tempDir}
	defer func() { AppConfig = origConfig }()

	orgID := "test-organization"
	userID := "user@google.com"
	projectID := "proj-upload-test"

	cloneDir := filepath.Join(tempDir, orgID, userID, projectID)
	if err := os.MkdirAll(cloneDir, 0755); err != nil {
		t.Fatalf("failed to create clone dir: %v", err)
	}

	createMultipartRequest := func(url string, files map[string]string) *http.Request {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		for filename, content := range files {
			part, err := writer.CreateFormFile("files", filename)
			if err != nil {
				t.Fatalf("failed to create form file: %v", err)
			}
			if _, err := io.WriteString(part, content); err != nil {
				t.Fatalf("failed to write form file content: %v", err)
			}
		}
		writer.Close()

		req := httptest.NewRequest("POST", url, body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.SetPathValue("id", projectID)
		claims := &core.UserClaims{Subject: userID}
		return withTestCtx(req, claims)
	}

	t.Run("Upload files to root directory", func(t *testing.T) {
		req := createMultipartRequest("/api/v1/projects/"+projectID+"/files/upload", map[string]string{
			"root_file1.txt": "hello root 1",
			"root_file2.json": `{"test": true}`,
		})

		rr := httptest.NewRecorder()
		UploadProjectFilesHandler(rr, req)

		if rr.Code != http.StatusOK && rr.Code != http.StatusCreated {
			t.Fatalf("expected status 200/201, got %d: %s", rr.Code, rr.Body.String())
		}

		content1, err := os.ReadFile(filepath.Join(cloneDir, "root_file1.txt"))
		if err != nil || string(content1) != "hello root 1" {
			t.Errorf("expected root_file1.txt content 'hello root 1', got err: %v, content: %s", err, string(content1))
		}
		content2, err := os.ReadFile(filepath.Join(cloneDir, "root_file2.json"))
		if err != nil || string(content2) != `{"test": true}` {
			t.Errorf("expected root_file2.json content '{\"test\": true}', got err: %v, content: %s", err, string(content2))
		}
	})

	t.Run("Upload files to subfolder (creates folder if missing)", func(t *testing.T) {
		req := createMultipartRequest("/api/v1/projects/"+projectID+"/files/upload?targetPath=src/assets", map[string]string{
			"logo.svg": "<svg></svg>",
		})

		rr := httptest.NewRecorder()
		UploadProjectFilesHandler(rr, req)

		if rr.Code != http.StatusOK && rr.Code != http.StatusCreated {
			t.Fatalf("expected status 200/201, got %d: %s", rr.Code, rr.Body.String())
		}

		content, err := os.ReadFile(filepath.Join(cloneDir, "src", "assets", "logo.svg"))
		if err != nil || string(content) != "<svg></svg>" {
			t.Errorf("expected logo.svg content '<svg></svg>', got err: %v, content: %s", err, string(content))
		}
	})

	t.Run("Rejects path traversal in targetPath", func(t *testing.T) {
		traversalPaths := []string{
			"../../etc",
			"../sibling",
			"/etc",
			"foo\\bar",
			".git",
			".git/hooks",
		}

		for _, tp := range traversalPaths {
			req := createMultipartRequest("/api/v1/projects/"+projectID+"/files/upload?targetPath="+tp, map[string]string{
				"evil.txt": "evil content",
			})

			rr := httptest.NewRecorder()
			UploadProjectFilesHandler(rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Errorf("expected status 400 for targetPath '%s', got %d: %s", tp, rr.Code, rr.Body.String())
			}
		}
	})

	t.Run("Rejects path traversal in filename", func(t *testing.T) {
		evilFilenames := []string{
			"../../foo.txt",
			"../evil.sh",
			"/root/evil.txt",
			"..",
			".git",
		}

		for _, fn := range evilFilenames {
			req := createMultipartRequest("/api/v1/projects/"+projectID+"/files/upload", map[string]string{
				fn: "evil content",
			})

			rr := httptest.NewRecorder()
			UploadProjectFilesHandler(rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Errorf("expected status 400 for filename '%s', got %d: %s", fn, rr.Code, rr.Body.String())
			}
		}
	})

	t.Run("Rejects upload if destination filename conflicts with existing directory", func(t *testing.T) {
		subDir := filepath.Join(cloneDir, "subfolder")
		if err := os.MkdirAll(subDir, 0755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}

		req := createMultipartRequest("/api/v1/projects/"+projectID+"/files/upload", map[string]string{
			"subfolder": "cannot overwrite directory with file",
		})

		rr := httptest.NewRecorder()
		UploadProjectFilesHandler(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400 when filename matches existing dir, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("Rejects upload if destination is a symlink", func(t *testing.T) {
		outsideDir := t.TempDir()
		outsideFile := filepath.Join(outsideDir, "secret.txt")
		if err := os.WriteFile(outsideFile, []byte("secret"), 0644); err != nil {
			t.Fatalf("failed to write outside file: %v", err)
		}
		symlinkTarget := filepath.Join(cloneDir, "symlink_file.txt")
		if err := os.Symlink(outsideFile, symlinkTarget); err != nil {
			t.Fatalf("failed to create symlink: %v", err)
		}

		req := createMultipartRequest("/api/v1/projects/"+projectID+"/files/upload", map[string]string{
			"symlink_file.txt": "overwrite attempt",
		})

		rr := httptest.NewRecorder()
		UploadProjectFilesHandler(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400 when uploading to symlink, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("Rejects request with no files", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		writer.Close()

		req := httptest.NewRequest("POST", "/api/v1/projects/"+projectID+"/files/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.SetPathValue("id", projectID)
		claims := &core.UserClaims{Subject: userID}
		req = withTestCtx(req, claims)

		rr := httptest.NewRecorder()
		UploadProjectFilesHandler(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400 for empty files upload, got %d: %s", rr.Code, rr.Body.String())
		}
	})
}

func TestDeleteProjectFolderHandler(t *testing.T) {
	tempDir := t.TempDir()
	origConfig := AppConfig
	AppConfig = &core.Configuration{WorkspaceMountPath: tempDir}
	defer func() { AppConfig = origConfig }()

	orgID := "test-organization"
	userID := "user@google.com"
	projectID := "proj-delete-folder-test"

	cloneDir := filepath.Join(tempDir, orgID, userID, projectID)
	targetSubdir := filepath.Join(cloneDir, "src", "temp")
	if err := os.MkdirAll(targetSubdir, 0755); err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(targetSubdir, "temp.txt"), []byte("temp"), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cloneDir, "README.md"), []byte("readme"), 0644); err != nil {
		t.Fatalf("failed to write readme: %v", err)
	}

	t.Run("Successfully deletes subfolder and its contents", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/v1/projects/"+projectID+"/folders?path=src/temp", nil)
		req.SetPathValue("id", projectID)
		claims := &core.UserClaims{Subject: userID}
		req = withTestCtx(req, claims)

		rr := httptest.NewRecorder()
		DeleteProjectFolderHandler(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
		}

		if _, err := os.Stat(targetSubdir); !os.IsNotExist(err) {
			t.Errorf("expected targetSubdir to be removed, but stat returned: %v", err)
		}
	})

	t.Run("Returns 404 for non-existent folder", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/v1/projects/"+projectID+"/folders?path=non/existent/folder", nil)
		req.SetPathValue("id", projectID)
		claims := &core.UserClaims{Subject: userID}
		req = withTestCtx(req, claims)

		rr := httptest.NewRecorder()
		DeleteProjectFolderHandler(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("Rejects deleting root directory attempt", func(t *testing.T) {
		invalidPaths := []string{"", ".", "/", "..", "../", "../../", "foo\\bar"}
		for _, p := range invalidPaths {
			req := httptest.NewRequest("DELETE", "/api/v1/projects/"+projectID+"/folders?path="+p, nil)
			req.SetPathValue("id", projectID)
			claims := &core.UserClaims{Subject: userID}
			req = withTestCtx(req, claims)

			rr := httptest.NewRecorder()
			DeleteProjectFolderHandler(rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Errorf("expected status 400 for deleting root attempt '%s', got %d: %s", p, rr.Code, rr.Body.String())
			}
		}
	})

	t.Run("Rejects deleting .git directory", func(t *testing.T) {
		gitDir := filepath.Join(cloneDir, ".git")
		if err := os.MkdirAll(gitDir, 0755); err != nil {
			t.Fatalf("failed to create .git dir: %v", err)
		}

		req := httptest.NewRequest("DELETE", "/api/v1/projects/"+projectID+"/folders?path=.git", nil)
		req.SetPathValue("id", projectID)
		claims := &core.UserClaims{Subject: userID}
		req = withTestCtx(req, claims)

		rr := httptest.NewRecorder()
		DeleteProjectFolderHandler(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400 for .git folder deletion, got %d: %s", rr.Code, rr.Body.String())
		}

		// Ensure .git still exists
		if _, err := os.Stat(gitDir); os.IsNotExist(err) {
			t.Errorf(".git directory was erroneously deleted")
		}
	})

	t.Run("Rejects deleting file instead of folder", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/v1/projects/"+projectID+"/folders?path=README.md", nil)
		req.SetPathValue("id", projectID)
		claims := &core.UserClaims{Subject: userID}
		req = withTestCtx(req, claims)

		rr := httptest.NewRecorder()
		DeleteProjectFolderHandler(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400 for file path, got %d: %s", rr.Code, rr.Body.String())
		}
	})
}

