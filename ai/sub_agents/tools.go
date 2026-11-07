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

package subagents

import (
	"google.com/apf/ai/core"
	"google.com/apf/ai/tools"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

func GetGcsTool() (tool.Tool, error) {

	gcsReaderTool := &tools.GCSReaderTool{}

	gcsTool, err := functiontool.New(
		functiontool.Config{
			Name:        gcsReaderTool.Name(),
			Description: gcsReaderTool.Description(),
		},
		gcsReaderTool.Execute)

	if err != nil {
		return nil, err
	}
	return gcsTool, nil
}

func GetFileReaderTool() (tool.Tool, error) {

	fileReaderTool := &tools.FileReaderTool{}

	fileTool, err := functiontool.New(
		functiontool.Config{
			Name:        fileReaderTool.Name(),
			Description: fileReaderTool.Description(),
		},
		fileReaderTool.Execute)
	if err != nil {
		return nil, err
	}
	return fileTool, nil
}

func GetFileWriterTool() (tool.Tool, error) {

	fileWriterTool := &tools.FileWriterTool{}

	fileTool, err := functiontool.New(
		functiontool.Config{
			Name:        fileWriterTool.Name(),
			Description: fileWriterTool.Description(),
		},
		fileWriterTool.Execute)
	if err != nil {
		return nil, err
	}
	return fileTool, nil
}

func GetDirectoryReaderTool() (tool.Tool, error) {

	directoryReaderTool := &tools.DirectoryReaderTool{}

	dirTool, err := functiontool.New(
		functiontool.Config{
			Name:        directoryReaderTool.Name(),
			Description: directoryReaderTool.Description(),
		},
		directoryReaderTool.Execute)
	if err != nil {
		return nil, err
	}
	return dirTool, nil
}

func GetSuggestPhaseSwitchTool() (tool.Tool, error) {
	suggestPhaseSwitchTool := &tools.SuggestPhaseSwitchTool{}

	switchTool, err := functiontool.New(
		functiontool.Config{
			Name:        suggestPhaseSwitchTool.Name(),
			Description: suggestPhaseSwitchTool.Description(),
		},
		suggestPhaseSwitchTool.Execute)
	if err != nil {
		return nil, err
	}
	return switchTool, nil
}

func GetUVRunnerTool(cfg *core.Configuration) (tool.Tool, error) {
	uvRunner := tools.NewUVRunnerTool(cfg)

	uvTool, err := functiontool.New(
		functiontool.Config{
			Name:        uvRunner.Name(),
			Description: uvRunner.Description(),
		},
		uvRunner.Execute)
	if err != nil {
		return nil, err
	}
	return uvTool, nil
}

func GetFindTool() (tool.Tool, error) {
	findTool := &tools.FindTool{}

	fTool, err := functiontool.New(
		functiontool.Config{
			Name:        findTool.Name(),
			Description: findTool.Description(),
		},
		findTool.Execute)
	if err != nil {
		return nil, err
	}
	return fTool, nil
}
