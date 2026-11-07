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
	"log/slog"
	"os"

	"google.golang.org/adk/tool"
)

// FileReaderInput defines the arguments for the FileReaderTool.
type FileReaderInput struct {
	FilePath string `json:"filePath"`
}

// FileReaderOutput defines the return values for the FileReaderTool.
type FileReaderOutput struct {
	Content  string `json:"content"`
	FilePath string `json:"filePath"`
}

// ReadFileFromLocal reads a file from the local filesystem and returns its content.
func ReadFileFromLocal(ctx tool.Context, filePath string) (*FileReaderOutput, error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		slog.Warn("File does not exist", "filePath", filePath)
		return &FileReaderOutput{
			Content:  "",
			FilePath: filePath,
		}, nil
	}

	bytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %q: %w", filePath, err)
	}

	return &FileReaderOutput{
		Content:  string(bytes),
		FilePath: filePath,
	}, nil
}

// FileReaderTool is an ADK tool for reading files from the local filesystem.
type FileReaderTool struct{}

// Name returns the name of the tool.
func (t *FileReaderTool) Name() string {
	return "file_reader"
}

// Description returns a description of the tool.
func (t *FileReaderTool) Description() string {
	return "Reads the content of a file from the local filesystem."
}

// Execute runs the tool with the given arguments.
func (t *FileReaderTool) Execute(ctx tool.Context, input *FileReaderInput) (*FileReaderOutput, error) {
	return ReadFileFromLocal(ctx, input.FilePath)
}
