/**
 * Copyright 2026 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { Node } from "@/components/workspace/FileExplorer"

export interface FolderMatchResult {
  /** The last folder along the file path that is currently visible in the tree view ("" for root) */
  lastVisibleFolder: string
  /** Whether the last visible folder is the root workspace folder */
  isRoot: boolean
  /** All parent folders up to lastVisibleFolder that should be expanded */
  foldersToOpen: string[]
}

/**
 * Normalizes a file or folder path by replacing backslashes, trimming whitespace,
 * and removing leading ./ or /
 */
export function normalizeFilePath(filePath: string): string {
  if (!filePath) return ""
  return filePath
    .trim()
    .replace(/^[`"']+|[`"']+$/g, "")
    .replace(/\\/g, "/")
    .replace(/^\.?\//, "")
    .replace(/\/+$/, "")
}

/**
 * Extracts a target file path from thought stream messages or tool execution outputs.
 * Prioritizes structured fields (NotificationDetails, ADK Event FunctionResponse / FunctionCall,
 * tool-result parameters) before falling back to JSON and text string matching.
 *
 * Supported formats:
 * 1. Notification with name="file-write" and params.filePath
 * 2. Notification with name="tool-result", params.tool_name="file_writer", params.file_name / params.content
 * 3. ADK Event parts with functionResponse (file_writer) or functionCall (file_writer)
 * 4. Text Format 1: Tool result (file_writer):\n\n{\n  "filePath": "AGENTS.md",\n  "message": "File written successfully"\n}
 * 5. Text Format 2: File updated: docs/spec/brainstorming/brainstorm-stock-screening-app-2026-08-19/.memlog.md
 */
export function extractFilePathFromFileEvent(input: any): string | null {
  if (!input) return null

  // If input is a raw JSON string, attempt to parse it first into an object
  let data = input
  if (typeof input === "string") {
    try {
      const parsed = JSON.parse(input)
      if (parsed && typeof parsed === "object") {
        data = parsed
      }
    } catch {
      // Retain data as string if not JSON
    }
  }

  // 1. Structured inspection of NotificationDetails
  if (typeof data === "object" && data !== null) {
    const details = data.notificationDetails
    if (details) {
      const params = details.params || {}
      if (params.filePath && typeof params.filePath === "string") {
        return normalizeFilePath(params.filePath)
      }
      if (params.path && typeof params.path === "string") {
        return normalizeFilePath(params.path)
      }

      // Check tool-result notification parameters
      if (
        details.name === "tool-result" ||
        params.tool_name === "file_writer" ||
        params.tool_name === "write_to_file"
      ) {
        if (params.file_name && typeof params.file_name === "string") {
          return normalizeFilePath(params.file_name)
        }
        if (params.content) {
          if (typeof params.content === "object" && params.content?.filePath) {
            return normalizeFilePath(params.content.filePath)
          }
          if (typeof params.content === "string") {
            try {
              const parsedContent = JSON.parse(params.content)
              if (parsedContent?.filePath) return normalizeFilePath(parsedContent.filePath)
              if (parsedContent?.path) return normalizeFilePath(parsedContent.path)
            } catch {
              const fpMatch =
                params.content.match(/"filePath"\s*:\s*"([^"]+)"/i) ||
                params.content.match(/"path"\s*:\s*"([^"]+)"/i)
              if (fpMatch && fpMatch[1]) return normalizeFilePath(fpMatch[1])
            }
          }
        }
      }
    }

    // 2. Structured inspection of ADK Event parts
    const event = data.event
    if (event?.content?.parts && Array.isArray(event.content.parts)) {
      for (const part of event.content.parts) {
        const funcResp = part.functionResponse || part.function_response
        if (funcResp) {
          const resp = funcResp.response || {}
          if (resp.filePath && typeof resp.filePath === "string") {
            return normalizeFilePath(resp.filePath)
          }
          if (resp.path && typeof resp.path === "string") {
            return normalizeFilePath(resp.path)
          }
        }

        const funcCall = part.functionCall || part.function_call
        if (funcCall && (funcCall.name === "file_writer" || funcCall.name === "write_to_file")) {
          const args = funcCall.args || {}
          if (args.filePath && typeof args.filePath === "string") {
            return normalizeFilePath(args.filePath)
          }
          if (args.path && typeof args.path === "string") {
            return normalizeFilePath(args.path)
          }
        }
      }
    }

    // 3. Structured FunctionResponse field on Message
    if (data.functionResponse?.response?.filePath) {
      return normalizeFilePath(String(data.functionResponse.response.filePath))
    }
    if (data.functionResponse?.response?.path) {
      return normalizeFilePath(String(data.functionResponse.response.path))
    }
  }

  // 4. Fallback text parsing for string messages or raw text payloads
  const textToScan: string =
    typeof input === "string"
      ? input
      : typeof data === "object" && data !== null
      ? `${data.text || ""} ${data.displayText || ""} ${data.rawContent || ""}`
      : ""

  if (!textToScan.trim()) return null

  // Format 1: Tool result (file_writer): ... {"filePath": "..."}
  const toolResultRegex = /Tool result\s*\((?:file_writer|file_create|write_to_file|writer)\):\s*([\s\S]*)/i
  const toolMatch = textToScan.match(toolResultRegex)
  if (toolMatch && toolMatch[1]) {
    const rawBody = toolMatch[1].trim()
    try {
      const parsed = JSON.parse(rawBody)
      if (parsed?.filePath && typeof parsed.filePath === "string") {
        return normalizeFilePath(parsed.filePath)
      }
      if (parsed?.path && typeof parsed.path === "string") {
        return normalizeFilePath(parsed.path)
      }
    } catch {
      const fpMatch =
        rawBody.match(/"filePath"\s*:\s*"([^"]+)"/i) || rawBody.match(/"path"\s*:\s*"([^"]+)"/i)
      if (fpMatch && fpMatch[1]) {
        return normalizeFilePath(fpMatch[1])
      }
    }
  }

  // Generic file_writer JSON match in text
  const fileWriterJsonMatch = textToScan.match(/file_writer[\s\S]*?"filePath"\s*:\s*"([^"]+)"/i)
  if (fileWriterJsonMatch && fileWriterJsonMatch[1]) {
    return normalizeFilePath(fileWriterJsonMatch[1])
  }

  // Format 2: File updated: <path> / File written: <path> / File created: <path>
  const fileUpdatedRegex = /(?:\*\*)?(?:File updated|File written|File created):?(?:\*\*)?:?\s*`?([^\s\n\r`"\\*]+)`?/i
  const fileUpdatedMatch = textToScan.match(fileUpdatedRegex)
  if (fileUpdatedMatch && fileUpdatedMatch[1]) {
    return normalizeFilePath(fileUpdatedMatch[1])
  }

  return null
}

/**
 * Matches a file path against the current file tree view from left to right.
 * Finds the last folder along the path that is currently visible in the tree view.
 */
export function findLastVisibleFolder(
  filePath: string,
  fileTree: Node | undefined | null,
  expandedPaths: Set<string>,
  childrenCache: Record<string, Node[]>
): FolderMatchResult {
  const normalized = normalizeFilePath(filePath)
  if (!normalized) {
    return { lastVisibleFolder: "", isRoot: true, foldersToOpen: [] }
  }

  const parts = normalized.split("/").filter(Boolean)
  // If the file is at the root level (e.g. "AGENTS.md" -> parts.length === 1),
  // there are no directory segments before the file.
  if (parts.length <= 1) {
    return { lastVisibleFolder: "", isRoot: true, foldersToOpen: [] }
  }

  // Folder segments from left to right (excluding the filename at the end)
  // e.g. for "docs/spec/brainstorming/file.md":
  // segments = ["docs", "docs/spec", "docs/spec/brainstorming"]
  const folderSegments: string[] = []
  let acc = ""
  for (let i = 0; i < parts.length - 1; i++) {
    acc = acc ? `${acc}/${parts[i]}` : parts[i]
    folderSegments.push(acc)
  }

  let lastVisibleFolder = ""
  const foldersToOpen: string[] = []
  let currentChildren = fileTree?.children

  for (let i = 0; i < folderSegments.length; i++) {
    const segPath = folderSegments[i]
    const segName = parts[i]

    if (!currentChildren || currentChildren.length === 0) {
      break
    }

    const matchingChild = currentChildren.find(
      (c) =>
        (normalizeFilePath(c.path) === segPath || c.name === segName) &&
        c.type === "folder"
    )

    if (!matchingChild) {
      // Child folder is not present in current visible children. Stop matching.
      break
    }

    // This folder is visible!
    lastVisibleFolder = segPath
    foldersToOpen.push(segPath)

    // Check if this folder is expanded in the tree view.
    // If not expanded, its subfolders are hidden/not visible on screen.
    const isExpanded = expandedPaths.has(segPath)
    if (!isExpanded) {
      break
    }

    // If expanded, move down to its children for the next segment.
    currentChildren = childrenCache[segPath] ?? matchingChild.children
  }

  return {
    lastVisibleFolder,
    isRoot: lastVisibleFolder === "",
    foldersToOpen,
  }
}
