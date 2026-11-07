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

package tools

import (
	"fmt"
	"os"
	"path/filepath"

	"google.golang.org/adk/tool"
)

// FileWriterInput defines the arguments for the FileWriterTool.
type FileWriterInput struct {
	FilePath string `json:"filePath"`
	Content  string `json:"content"`
}

// FileWriterOutput defines the return values for the FileWriterTool.
type FileWriterOutput struct {
	Message  string `json:"message"`
	FilePath string `json:"filePath"`
}

// WriteFileToLocal writes content to a file on the local filesystem.
func WriteFileToLocal(ctx tool.Context, filePath string, content string) (*FileWriterOutput, error) {
	dir := filepath.Dir(filePath)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create directory %q: %w", dir, err)
	}

	err = os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to write file %q: %w", filePath, err)
	}

	return &FileWriterOutput{
		Message:  "File written successfully",
		FilePath: filePath,
	}, nil
}

// FileWriterTool is an ADK tool for writing files to the local filesystem.
type FileWriterTool struct{}

// Name returns the name of the tool.
func (t *FileWriterTool) Name() string {
	return "file_writer"
}

// Description returns a description of the tool.
func (t *FileWriterTool) Description() string {
	return "Writes content to a file on the local filesystem."
}

// Execute runs the tool with the given arguments.
func (t *FileWriterTool) Execute(ctx tool.Context, input *FileWriterInput) (*FileWriterOutput, error) {
	return WriteFileToLocal(ctx, input.FilePath, input.Content)
}
