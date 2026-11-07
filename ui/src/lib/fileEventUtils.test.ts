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

import { describe, it, expect } from "vitest"
import {
  normalizeFilePath,
  extractFilePathFromFileEvent,
  findLastVisibleFolder,
} from "./fileEventUtils"
import { Node } from "@/components/workspace/FileExplorer"

describe("fileEventUtils", () => {
  describe("normalizeFilePath", () => {
    it("handles backslashes, leading ./ and / and whitespace", () => {
      expect(normalizeFilePath("  docs\\spec\\prd.md  ")).toBe("docs/spec/prd.md")
      expect(normalizeFilePath("./docs/spec/prd.md")).toBe("docs/spec/prd.md")
      expect(normalizeFilePath("/docs/spec/prd.md/")).toBe("docs/spec/prd.md")
      expect(normalizeFilePath("`docs/spec/prd.md`")).toBe("docs/spec/prd.md")
      expect(normalizeFilePath('"AGENTS.md"')).toBe("AGENTS.md")
      expect(normalizeFilePath("")).toBe("")
    })
  })

  describe("extractFilePathFromFileEvent", () => {
    it("extracts from structured NotificationDetails with name=file-write", () => {
      const msg = {
        isNotification: true,
        notificationDetails: {
          name: "file-write",
          params: {
            filePath: "docs/spec/brainstorming/brainstorm.md",
          },
        },
      }
      expect(extractFilePathFromFileEvent(msg)).toBe(
        "docs/spec/brainstorming/brainstorm.md"
      )
    })

    it("extracts from structured NotificationDetails with name=tool-result for file_writer", () => {
      const msg = {
        isNotification: true,
        notificationDetails: {
          name: "tool-result",
          params: {
            tool_name: "file_writer",
            file_name: "AGENTS.md",
            content: JSON.stringify({
              filePath: "AGENTS.md",
              message: "File written successfully",
            }),
          },
        },
      }
      expect(extractFilePathFromFileEvent(msg)).toBe("AGENTS.md")
    })

    it("extracts from structured ADK Event with FunctionResponse", () => {
      const msg = {
        event: {
          content: {
            parts: [
              {
                functionResponse: {
                  name: "file_writer",
                  response: {
                    filePath: "docs/spec/architecture.md",
                    message: "File written successfully",
                  },
                },
              },
            ],
          },
        },
      }
      expect(extractFilePathFromFileEvent(msg)).toBe("docs/spec/architecture.md")
    })

    it("extracts from structured ADK Event with FunctionCall", () => {
      const msg = {
        event: {
          content: {
            parts: [
              {
                functionCall: {
                  name: "file_writer",
                  args: {
                    filePath: "src/utils/helper.ts",
                    content: "export const x = 1",
                  },
                },
              },
            ],
          },
        },
      }
      expect(extractFilePathFromFileEvent(msg)).toBe("src/utils/helper.ts")
    })

    it("extracts from text Format 1: Tool result (file_writer)", () => {
      const text = `Tool result (file_writer):

{
  "filePath": "AGENTS.md",
  "message": "File written successfully"
}`
      expect(extractFilePathFromFileEvent(text)).toBe("AGENTS.md")
      expect(extractFilePathFromFileEvent({ text })).toBe("AGENTS.md")
    })

    it("extracts from text Format 2: File updated line", () => {
      const text =
        "File updated: docs/spec/brainstorming/brainstorm-stock-screening-app-2026-08-19/.memlog.md"
      expect(extractFilePathFromFileEvent(text)).toBe(
        "docs/spec/brainstorming/brainstorm-stock-screening-app-2026-08-19/.memlog.md"
      )
      expect(extractFilePathFromFileEvent({ text })).toBe(
        "docs/spec/brainstorming/brainstorm-stock-screening-app-2026-08-19/.memlog.md"
      )
    })

    it("extracts from text Format 2 with backticks or Markdown bold", () => {
      expect(
        extractFilePathFromFileEvent("**File updated:** `docs/spec/prd.md`")
      ).toBe("docs/spec/prd.md")
      expect(
        extractFilePathFromFileEvent("File written: src/components/Header.tsx")
      ).toBe("src/components/Header.tsx")
    })

    it("returns null for non-file events", () => {
      expect(extractFilePathFromFileEvent(null)).toBeNull()
      expect(extractFilePathFromFileEvent("")).toBeNull()
      expect(
        extractFilePathFromFileEvent("Hello user! What would you like to build?")
      ).toBeNull()
      expect(
        extractFilePathFromFileEvent({
          isNotification: true,
          notificationDetails: {
            name: "thought",
            params: {},
          },
        })
      ).toBeNull()
    })
  })

  describe("findLastVisibleFolder", () => {
    const mockFileTree: Node = {
      name: "root",
      path: "",
      type: "folder",
      children: [
        { name: "docs", path: "docs", type: "folder" },
        { name: "src", path: "src", type: "folder" },
        { name: "AGENTS.md", path: "AGENTS.md", type: "file" },
      ],
    }

    it("returns root for a root-level file", () => {
      const result = findLastVisibleFolder(
        "AGENTS.md",
        mockFileTree,
        new Set(),
        {}
      )
      expect(result).toEqual({
        lastVisibleFolder: "",
        isRoot: true,
        foldersToOpen: [],
      })
    })

    it("returns top-level folder when top-level folder is collapsed", () => {
      const result = findLastVisibleFolder(
        "docs/spec/brainstorming/brainstorm.md",
        mockFileTree,
        new Set(),
        {}
      )
      expect(result).toEqual({
        lastVisibleFolder: "docs",
        isRoot: false,
        foldersToOpen: ["docs"],
      })
    })

    it("returns deeper folder when parent folder is expanded and child folder is cached", () => {
      const expandedPaths = new Set(["docs"])
      const childrenCache = {
        docs: [
          { name: "spec", path: "docs/spec", type: "folder" as const },
          { name: "readme.md", path: "docs/readme.md", type: "file" as const },
        ],
      }

      const result = findLastVisibleFolder(
        "docs/spec/brainstorming/brainstorm.md",
        mockFileTree,
        expandedPaths,
        childrenCache
      )

      expect(result).toEqual({
        lastVisibleFolder: "docs/spec",
        isRoot: false,
        foldersToOpen: ["docs", "docs/spec"],
      })
    })

    it("returns deeply nested folder when all intermediate folders are expanded", () => {
      const expandedPaths = new Set(["docs", "docs/spec"])
      const childrenCache = {
        docs: [{ name: "spec", path: "docs/spec", type: "folder" as const }],
        "docs/spec": [
          {
            name: "brainstorming",
            path: "docs/spec/brainstorming",
            type: "folder" as const,
          },
        ],
      }

      const result = findLastVisibleFolder(
        "docs/spec/brainstorming/brainstorm.md",
        mockFileTree,
        expandedPaths,
        childrenCache
      )

      expect(result).toEqual({
        lastVisibleFolder: "docs/spec/brainstorming",
        isRoot: false,
        foldersToOpen: ["docs", "docs/spec", "docs/spec/brainstorming"],
      })
    })

    it("returns parent folder when subfolder does not exist yet in cache (newly created by agent)", () => {
      const expandedPaths = new Set(["docs"])
      const childrenCache = {
        docs: [{ name: "old_spec", path: "docs/old_spec", type: "folder" as const }],
      }

      // "new_spec" was created by the agent on disk, but not in childrenCache yet
      const result = findLastVisibleFolder(
        "docs/new_spec/sub/file.md",
        mockFileTree,
        expandedPaths,
        childrenCache
      )

      expect(result).toEqual({
        lastVisibleFolder: "docs",
        isRoot: false,
        foldersToOpen: ["docs"],
      })
    })

    it("returns root if top-level folder is not in root tree yet", () => {
      const result = findLastVisibleFolder(
        "new_top_dir/sub/file.md",
        mockFileTree,
        new Set(),
        {}
      )
      expect(result).toEqual({
        lastVisibleFolder: "",
        isRoot: true,
        foldersToOpen: [],
      })
    })
  })
})
