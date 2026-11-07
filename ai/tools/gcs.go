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
	"io"
	"strings"

	"cloud.google.com/go/storage"
	"google.golang.org/adk/tool"
)

// GCSReaderInput defines the arguments for the GCSReaderTool.
type GCSReaderInput struct {
	FilePath string `json:"filePath"`
}

// GCSReaderOutput defines the return values for the GCSReaderTool.
type GCSReaderOutput struct {
	Content    string `json:"content"`
	BucketName string `json:"bucketName"`
	ObjectName string `json:"objectName"`
}

// ReadFileFromGCS reads a file from a Google Cloud Storage bucket and returns its content as a string.
func ReadFileFromGCS(ctx tool.Context, filePath string) (*GCSReaderOutput, error) {

	path := strings.TrimPrefix(filePath, "gs://")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return nil, fmt.Errorf("invalid GCS file path %q; expected format [gs://]<bucket-name>/<object-path>", filePath)
	}
	bucketName := parts[0]
	objectName := parts[1]

	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCS client: %w", err)
	}
	defer client.Close()

	obj := client.Bucket(bucketName).Object(objectName)

	reader, err := obj.NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create reader for object %q in bucket %q: %w", objectName, bucketName, err)
	}
	defer reader.Close()

	bytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read data from object %q in bucket %q: %w", objectName, bucketName, err)
	}

	return &GCSReaderOutput{
		Content:    string(bytes),
		BucketName: bucketName,
		ObjectName: objectName,
	}, nil
}

// GCSReaderTool is an ADK tool for reading files from GCS.
type GCSReaderTool struct{}

// Name returns the name of the tool.
func (t *GCSReaderTool) Name() string {
	return "gcs_reader"
}

// Description returns a description of the tool.
func (t *GCSReaderTool) Description() string {
	return "Reads the content of a file from a Google Cloud Storage (GCS) bucket."
}

// Execute runs the tool with the given arguments.
func (t *GCSReaderTool) Execute(ctx tool.Context, input *GCSReaderInput) (*GCSReaderOutput, error) {
	return ReadFileFromGCS(ctx, input.FilePath)
}
