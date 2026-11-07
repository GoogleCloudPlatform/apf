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
	"fmt"
	"log/slog"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"google.com/apf/api/core"
	"google.com/apf/api/db"
)

// isImageArtifact checks if the artifact is a raster image based on its MIME type or file extension.
func isImageArtifact(name, mimeType string) bool {
	lowerMime := strings.ToLower(mimeType)
	if strings.HasPrefix(lowerMime, "image/") && !strings.Contains(lowerMime, "svg") {
		return true
	}
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".webp", ".gif", ".bmp", ".tiff", ".tif", ".ico":
		return true
	}
	return false
}

// isTextArtifact checks if the artifact is a text or code document based on MIME type or extension.
func isTextArtifact(name, mimeType string) bool {
	lowerMime := strings.ToLower(mimeType)
	if strings.HasPrefix(lowerMime, "text/") || strings.Contains(lowerMime, "svg") {
		return true
	}
	if strings.Contains(lowerMime, "json") ||
		strings.Contains(lowerMime, "yaml") ||
		strings.Contains(lowerMime, "xml") ||
		strings.Contains(lowerMime, "javascript") ||
		strings.Contains(lowerMime, "typescript") ||
		strings.Contains(lowerMime, "sql") ||
		strings.Contains(lowerMime, "toml") {
		return true
	}
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".txt", ".md", ".markdown", ".json", ".yaml", ".yml", ".py", ".go",
		".ts", ".tsx", ".js", ".jsx", ".html", ".htm", ".css", ".scss", ".sass",
		".less", ".csv", ".tsv", ".sql", ".sh", ".bash", ".zsh", ".xml", ".proto",
		".graphql", ".gql", ".toml", ".ini", ".env", ".c", ".cpp", ".cc", ".cxx",
		".h", ".hpp", ".hh", ".java", ".kt", ".kts", ".rs", ".rb", ".php",
		".swift", ".r", ".scala", ".dart", ".lua", ".diff", ".patch", ".log",
		".rst", ".asciidoc", ".adoc", ".svg":
		return true
	}
	return false
}

// resolveImageMimeType resolves the MIME type for an image artifact.
func resolveImageMimeType(name, mimeType string, data []byte) string {
	lowerMime := strings.ToLower(mimeType)
	if strings.HasPrefix(lowerMime, "image/") && !strings.Contains(lowerMime, "svg") {
		return mimeType
	}
	ext := strings.ToLower(filepath.Ext(name))
	if detected := mime.TypeByExtension(ext); detected != "" && strings.HasPrefix(detected, "image/") {
		return detected
	}
	if len(data) > 0 {
		sniffed := http.DetectContentType(data)
		if strings.HasPrefix(sniffed, "image/") {
			return sniffed
		}
	}
	switch ext {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	case ".bmp":
		return "image/bmp"
	case ".tiff", ".tif":
		return "image/tiff"
	case ".ico":
		return "image/x-icon"
	default:
		return "image/png"
	}
}

// BuildPromptParts ingests attached session artifacts, extracts text/code files into prompt demarcation blocks,
// base64-encodes images into multimodal inline data parts, and injects structured warnings for unreadable or unsupported files.
func BuildPromptParts(ctx context.Context, prompt string, artifacts []db.Artifact, svc core.ArtifactService) ([]ProxyMessagePart, error) {
	var textBlocks []string
	if prompt != "" {
		textBlocks = append(textBlocks, prompt)
	}

	var imageParts []ProxyMessagePart

	for _, art := range artifacts {
		filename := strings.TrimSpace(art.Name)
		if filename == "" {
			filename = filepath.Base(filepath.Clean(art.StoragePath))
		}
		if filename == "" || filename == "." || filename == "/" {
			filename = "attached_file"
		}

		if art.StoragePath == "" {
			warning := fmt.Sprintf(`[Warning: Could not read attached file "%s": missing storage path]`, filename)
			slog.Warn("Attached artifact missing storage path", "filename", filename, "id", art.ID)
			textBlocks = append(textBlocks, warning)
			continue
		}

		if svc == nil {
			warning := fmt.Sprintf(`[Warning: Could not read attached file "%s": artifact service unavailable]`, filename)
			slog.Warn("ArtifactService unavailable to read artifact", "filename", filename, "storagePath", art.StoragePath)
			textBlocks = append(textBlocks, warning)
			continue
		}

		data, err := svc.ReadSessionArtifact(ctx, art.StorageType, art.StoragePath)
		if err != nil {
			warning := fmt.Sprintf(`[Warning: Could not read attached file "%s": %v]`, filename, err)
			slog.Warn("Failed to read attached session artifact", "filename", filename, "storagePath", art.StoragePath, "error", err)
			textBlocks = append(textBlocks, warning)
			continue
		}

		// Image artifact branch
		if isImageArtifact(filename, art.MimeType) {
			if len(data) == 0 {
				warning := fmt.Sprintf(`[Warning: Could not load attached file "%s": empty file]`, filename)
				slog.Warn("Attached image artifact is empty", "filename", filename)
				textBlocks = append(textBlocks, warning)
				continue
			}

			imgMimeType := resolveImageMimeType(filename, art.MimeType, data)
			encoded := base64.StdEncoding.EncodeToString(data)
			imageParts = append(imageParts, ProxyMessagePart{
				InlineData: &ProxyInlineData{
					MimeType: imgMimeType,
					Data:     encoded,
				},
			})
			continue
		}

		// Text/code artifact branch
		if isTextArtifact(filename, art.MimeType) {
			if !utf8.Valid(data) {
				warning := fmt.Sprintf(`[Warning: Unsupported binary file format for "%s"]`, filename)
				slog.Warn("Attached text artifact has invalid UTF-8 binary bytes", "filename", filename)
				textBlocks = append(textBlocks, warning)
				continue
			}

			demarcation := fmt.Sprintf("[Attached File: %s]\n%s", filename, string(data))
			textBlocks = append(textBlocks, demarcation)
			continue
		}

		// Fallback for unrecognized extension/MIME: check UTF-8 validity
		if utf8.Valid(data) {
			demarcation := fmt.Sprintf("[Attached File: %s]\n%s", filename, string(data))
			textBlocks = append(textBlocks, demarcation)
		} else {
			warning := fmt.Sprintf(`[Warning: Unsupported binary file format for "%s"]`, filename)
			slog.Warn("Attached file is unsupported non-UTF8 binary format", "filename", filename)
			textBlocks = append(textBlocks, warning)
		}
	}

	parts := make([]ProxyMessagePart, 0, 1+len(imageParts))
	if len(textBlocks) > 0 {
		parts = append(parts, ProxyMessagePart{Text: strings.Join(textBlocks, "\n\n")})
	}
	parts = append(parts, imageParts...)
	if len(parts) == 0 {
		parts = append(parts, ProxyMessagePart{Text: ""})
	}

	return parts, nil
}
