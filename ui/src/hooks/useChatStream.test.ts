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

// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from "vitest"
import { renderHook, act } from "@testing-library/react"
import { useChatStream } from "./useChatStream"
import { api } from "@/lib/api"
import type { Artifact } from "@/lib/types"

vi.mock("@/lib/sse", () => ({
  connectSSE: vi.fn().mockReturnValue(vi.fn()),
}))

vi.mock("@/lib/api", () => ({
  api: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  },
}))

describe("useChatStream hook - Artifact attachments (Story 18.2)", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(api.post).mockResolvedValue({
      ok: true,
      status: 201,
      json: async () => ({ id: "msg-1" }),
    } as Response)
  })

  it("attaches artifacts and artifactIds to outgoing payload and optimistic message", async () => {
    const { result } = renderHook(() =>
      useChatStream("proj-1", "PLANNING", "sess-1")
    )

    const mockArtifacts: Artifact[] = [
      {
        id: "art-1",
        name: "diagram.png",
        mimeType: "image/png",
        size: 1024,
        storageType: "local_file",
        storagePath: "/artifacts/diagram.png",
      },
      {
        id: "art-2",
        name: "notes.md",
        mimeType: "text/markdown",
        size: 512,
        storageType: "local_file",
        storagePath: "/artifacts/notes.md",
      },
    ]

    let success = false
    await act(async () => {
      success = await result.current.sendMessage(
        "Analyze these files",
        undefined,
        mockArtifacts
      )
    })

    expect(success).toBe(true)
    expect(api.post).toHaveBeenCalledTimes(1)
    const [url, payload] = vi.mocked(api.post).mock.calls[0]
    expect(url).toBe("/api/v1/projects/proj-1/messages?sessionId=sess-1")
    expect(payload).toMatchObject({
      sender: "User",
      text: "Analyze these files",
      prompt: "Analyze these files",
      phase: "PLANNING",
      sessionId: "sess-1",
      artifactIds: ["art-1", "art-2"],
      artifacts: mockArtifacts,
    })

    const optimisticMsg = result.current.messages.find((m) => m.text === "Analyze these files")
    expect(optimisticMsg).toBeDefined()
    expect(optimisticMsg?.artifacts).toEqual(mockArtifacts)
    expect(optimisticMsg?.artifactIds).toEqual(["art-1", "art-2"])
  })

  it("handles attachment-only turn with default prompt text", async () => {
    const { result } = renderHook(() =>
      useChatStream("proj-1", "PLANNING", "sess-1")
    )

    const mockArtifact: Artifact = {
      id: "art-solo",
      name: "screen.png",
      mimeType: "image/png",
      size: 2048,
    }

    let success = false
    await act(async () => {
      success = await result.current.sendMessage("", undefined, [mockArtifact])
    })

    expect(success).toBe(true)
    expect(api.post).toHaveBeenCalledTimes(1)
    const [, payload] = vi.mocked(api.post).mock.calls[0]
    expect(payload).toMatchObject({
      text: "Uploaded attachment(s)",
      prompt: "Uploaded attachment(s)",
      artifactIds: ["art-solo"],
      artifacts: [mockArtifact],
    })
  })

  it("returns false and does not post when text and artifacts are both empty", async () => {
    const { result } = renderHook(() =>
      useChatStream("proj-1", "PLANNING", "sess-1")
    )

    let success = true
    await act(async () => {
      success = await result.current.sendMessage("", undefined, [])
    })

    expect(success).toBe(false)
    expect(api.post).not.toHaveBeenCalled()
  })
})
