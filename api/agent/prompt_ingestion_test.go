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

package agent

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"strings"
	"testing"

	"google.com/apf/api/db"
)

type mockMemoryArtifactService struct {
	files map[string][]byte
}

func newMockMemoryArtifactService() *mockMemoryArtifactService {
	return &mockMemoryArtifactService{
		files: make(map[string][]byte),
	}
}

func (m *mockMemoryArtifactService) GetArtifact(ctx context.Context, organizationID string, userId string, workspaceID string, path string) ([]byte, error) {
	return nil, nil
}

func (m *mockMemoryArtifactService) WriteArtifact(ctx context.Context, organizationID string, userId string, workspaceID string, path string, data []byte) error {
	return nil
}

func (m *mockMemoryArtifactService) DeleteArtifact(ctx context.Context, organizationID string, userId string, workspaceID string, path string) error {
	return nil
}

func (m *mockMemoryArtifactService) ListArtifacts(ctx context.Context, organizationID string, userId string, workspaceID string, prefix string) ([]string, error) {
	return nil, nil
}

func (m *mockMemoryArtifactService) ListTopLevelFiles(ctx context.Context, organizationID string, userId string, workspaceID string) ([]string, error) {
	return nil, nil
}

func (m *mockMemoryArtifactService) SaveSessionArtifact(ctx context.Context, projectId, sessionId, filename string, r io.Reader) (string, string, int64, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return "", "", 0, err
	}
	path := "/mock/" + projectId + "/" + sessionId + "/" + filename
	m.files[path] = data
	return "local_file", path, int64(len(data)), nil
}

func (m *mockMemoryArtifactService) ReadSessionArtifact(ctx context.Context, storageType string, storagePath string) ([]byte, error) {
	if data, exists := m.files[storagePath]; exists {
		return data, nil
	}
	return nil, errors.New("file not found: " + storagePath)
}

func TestBuildPromptParts_TextArtifact(t *testing.T) {
	svc := newMockMemoryArtifactService()
	svc.files["/artifacts/notes.md"] = []byte("# Project Notes\nFeature requirements.")

	artifacts := []db.Artifact{
		{
			ID:          "art-1",
			Name:        "notes.md",
			MimeType:    "text/markdown",
			StorageType: "local_file",
			StoragePath: "/artifacts/notes.md",
		},
	}

	parts, err := BuildPromptParts(context.Background(), "Analyze this", artifacts, svc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(parts) != 1 {
		t.Fatalf("expected 1 part (text only), got %d", len(parts))
	}

	expectedDemarcation := "[Attached File: notes.md]\n# Project Notes\nFeature requirements."
	if !strings.Contains(parts[0].Text, "Analyze this") {
		t.Errorf("expected prompt text to be preserved, got %q", parts[0].Text)
	}
	if !strings.Contains(parts[0].Text, expectedDemarcation) {
		t.Errorf("expected demarcation block in text part, got %q", parts[0].Text)
	}
}

func TestBuildPromptParts_ImageArtifact(t *testing.T) {
	svc := newMockMemoryArtifactService()
	rawImage := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x01} // simulated PNG
	svc.files["/artifacts/arch.png"] = rawImage

	artifacts := []db.Artifact{
		{
			ID:          "art-img",
			Name:        "arch.png",
			MimeType:    "image/png",
			StorageType: "local_file",
			StoragePath: "/artifacts/arch.png",
		},
	}

	parts, err := BuildPromptParts(context.Background(), "Explain diagram", artifacts, svc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(parts) != 2 {
		t.Fatalf("expected 2 parts (text + image), got %d", len(parts))
	}

	// Text part
	if parts[0].Text != "Explain diagram" {
		t.Errorf("expected text part 'Explain diagram', got %q", parts[0].Text)
	}

	// Multimodal image part
	if parts[1].InlineData == nil {
		t.Fatalf("expected InlineData on second part")
	}
	if parts[1].InlineData.MimeType != "image/png" {
		t.Errorf("expected mimeType 'image/png', got %q", parts[1].InlineData.MimeType)
	}
	expectedB64 := base64.StdEncoding.EncodeToString(rawImage)
	if parts[1].InlineData.Data != expectedB64 {
		t.Errorf("expected base64 data %q, got %q", expectedB64, parts[1].InlineData.Data)
	}
}

func TestBuildPromptParts_MultipleArtifacts(t *testing.T) {
	svc := newMockMemoryArtifactService()
	svc.files["/artifacts/schema.sql"] = []byte("CREATE TABLE users (id TEXT);")
	rawJpg := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46}
	svc.files["/artifacts/flow.jpg"] = rawJpg

	artifacts := []db.Artifact{
		{
			ID:          "art-sql",
			Name:        "schema.sql",
			MimeType:    "application/sql",
			StorageType: "local_file",
			StoragePath: "/artifacts/schema.sql",
		},
		{
			ID:          "art-jpg",
			Name:        "flow.jpg",
			MimeType:    "image/jpeg",
			StorageType: "local_file",
			StoragePath: "/artifacts/flow.jpg",
		},
	}

	parts, err := BuildPromptParts(context.Background(), "Review schema and flow", artifacts, svc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(parts) != 2 {
		t.Fatalf("expected 2 parts (1 combined text + 1 image), got %d", len(parts))
	}

	// Verify text part contains prompt and schema demarcation
	if !strings.Contains(parts[0].Text, "Review schema and flow") {
		t.Errorf("expected prompt preserved, got %q", parts[0].Text)
	}
	if !strings.Contains(parts[0].Text, "[Attached File: schema.sql]\nCREATE TABLE users (id TEXT);") {
		t.Errorf("expected schema demarcation block, got %q", parts[0].Text)
	}

	// Verify image part
	if parts[1].InlineData == nil || parts[1].InlineData.MimeType != "image/jpeg" {
		t.Errorf("expected image/jpeg inline data, got %+v", parts[1].InlineData)
	}
}

func TestBuildPromptParts_EmptyPromptWithImage(t *testing.T) {
	svc := newMockMemoryArtifactService()
	rawPng := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A}
	svc.files["/artifacts/screenshot.png"] = rawPng

	artifacts := []db.Artifact{
		{
			ID:          "art-scr",
			Name:        "screenshot.png",
			MimeType:    "image/png",
			StorageType: "local_file",
			StoragePath: "/artifacts/screenshot.png",
		},
	}

	parts, err := BuildPromptParts(context.Background(), "", artifacts, svc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(parts) != 1 {
		t.Fatalf("expected 1 part (image only), got %d", len(parts))
	}
	if parts[0].InlineData == nil || parts[0].InlineData.MimeType != "image/png" {
		t.Errorf("expected valid image part, got %+v", parts[0].InlineData)
	}
}

func TestBuildPromptParts_SvgArtifactAsText(t *testing.T) {
	svc := newMockMemoryArtifactService()
	svgContent := `<svg xmlns="http://www.w3.org/2000/svg"><circle r="10"/></svg>`
	svc.files["/artifacts/diagram.svg"] = []byte(svgContent)

	artifacts := []db.Artifact{
		{
			ID:          "art-svg",
			Name:        "diagram.svg",
			MimeType:    "image/svg+xml",
			StorageType: "local_file",
			StoragePath: "/artifacts/diagram.svg",
		},
	}

	parts, err := BuildPromptParts(context.Background(), "Check vector", artifacts, svc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(parts) != 1 {
		t.Fatalf("expected 1 part (text with svg demarcation), got %d", len(parts))
	}
	if !strings.Contains(parts[0].Text, "[Attached File: diagram.svg]\n"+svgContent) {
		t.Errorf("expected SVG content demarcated as text, got %q", parts[0].Text)
	}
}

func TestBuildPromptParts_UnreadableArtifact(t *testing.T) {
	svc := newMockMemoryArtifactService() // empty files map

	artifacts := []db.Artifact{
		{
			ID:          "art-missing",
			Name:        "missing.txt",
			MimeType:    "text/plain",
			StorageType: "local_file",
			StoragePath: "/artifacts/nonexistent.txt",
		},
	}

	parts, err := BuildPromptParts(context.Background(), "Summarize file", artifacts, svc)
	if err != nil {
		t.Fatalf("expected no crash or error, got %v", err)
	}

	if len(parts) != 1 {
		t.Fatalf("expected 1 text part with warning, got %d", len(parts))
	}

	if !strings.Contains(parts[0].Text, `[Warning: Could not read attached file "missing.txt":`) {
		t.Errorf("expected unreadable file warning, got %q", parts[0].Text)
	}
}

func TestBuildPromptParts_MissingStoragePath(t *testing.T) {
	svc := newMockMemoryArtifactService()

	artifacts := []db.Artifact{
		{
			ID:          "art-nopath",
			Name:        "unlinked.md",
			MimeType:    "text/markdown",
			StorageType: "local_file",
			StoragePath: "",
		},
	}

	parts, err := BuildPromptParts(context.Background(), "Check file", artifacts, svc)
	if err != nil {
		t.Fatalf("expected no crash or error, got %v", err)
	}

	if !strings.Contains(parts[0].Text, `[Warning: Could not read attached file "unlinked.md": missing storage path]`) {
		t.Errorf("expected missing storage path warning, got %q", parts[0].Text)
	}
}

func TestBuildPromptParts_NilArtifactService(t *testing.T) {
	artifacts := []db.Artifact{
		{
			ID:          "art-test",
			Name:        "test.txt",
			MimeType:    "text/plain",
			StorageType: "local_file",
			StoragePath: "/some/path/test.txt",
		},
	}

	parts, err := BuildPromptParts(context.Background(), "Analyze", artifacts, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !strings.Contains(parts[0].Text, `[Warning: Could not read attached file "test.txt": artifact service unavailable]`) {
		t.Errorf("expected artifact service unavailable warning, got %q", parts[0].Text)
	}
}

func TestBuildPromptParts_CorruptedEmptyImage(t *testing.T) {
	svc := newMockMemoryArtifactService()
	svc.files["/artifacts/empty.png"] = []byte{}

	artifacts := []db.Artifact{
		{
			ID:          "art-empty",
			Name:        "empty.png",
			MimeType:    "image/png",
			StorageType: "local_file",
			StoragePath: "/artifacts/empty.png",
		},
	}

	parts, err := BuildPromptParts(context.Background(), "Check image", artifacts, svc)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(parts) != 1 {
		t.Fatalf("expected 1 part (text with warning, no image part), got %d", len(parts))
	}

	if !strings.Contains(parts[0].Text, `[Warning: Could not load attached file "empty.png": empty file]`) {
		t.Errorf("expected empty image warning, got %q", parts[0].Text)
	}
}

func TestBuildPromptParts_UnsupportedBinary(t *testing.T) {
	svc := newMockMemoryArtifactService()
	// Invalid UTF-8 binary bytes
	svc.files["/artifacts/program.exe"] = []byte{0x4D, 0x5A, 0x90, 0x00, 0x03, 0x00, 0x00, 0x00, 0xFF, 0xFE, 0xFD}

	artifacts := []db.Artifact{
		{
			ID:          "art-bin",
			Name:        "program.exe",
			MimeType:    "application/octet-stream",
			StorageType: "local_file",
			StoragePath: "/artifacts/program.exe",
		},
	}

	parts, err := BuildPromptParts(context.Background(), "Run tool", artifacts, svc)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(parts) != 1 {
		t.Fatalf("expected 1 part (text with warning), got %d", len(parts))
	}

	if !strings.Contains(parts[0].Text, `[Warning: Unsupported binary file format for "program.exe"]`) {
		t.Errorf("expected unsupported binary warning, got %q", parts[0].Text)
	}
}

func TestBuildPromptParts_NoArtifacts(t *testing.T) {
	parts, err := BuildPromptParts(context.Background(), "Hello agent", nil, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(parts) != 1 {
		t.Fatalf("expected 1 part, got %d", len(parts))
	}
	if parts[0].Text != "Hello agent" {
		t.Errorf("expected 'Hello agent', got %q", parts[0].Text)
	}
}

func TestBuildPromptParts_FallbackUtf8(t *testing.T) {
	svc := newMockMemoryArtifactService()
	svc.files["/artifacts/custom_data.xyz"] = []byte("custom configuration text")

	artifacts := []db.Artifact{
		{
			ID:          "art-xyz",
			Name:        "custom_data.xyz",
			MimeType:    "application/octet-stream",
			StorageType: "local_file",
			StoragePath: "/artifacts/custom_data.xyz",
		},
	}

	parts, err := BuildPromptParts(context.Background(), "Inspect data", artifacts, svc)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !strings.Contains(parts[0].Text, "[Attached File: custom_data.xyz]\ncustom configuration text") {
		t.Errorf("expected fallback UTF-8 to be treated as text demarcation, got %q", parts[0].Text)
	}
}
