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
import { api } from "./api"

describe("ApiClient header sanitization", () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it("filters out headers with non-ISO-8859-1 code points (e.g. masked bullets)", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response("{}", { status: 200 }))
    global.fetch = fetchMock

    await api.get("/api/v1/test", {
      headers: {
        "X-Valid-Header": "valid-value-123",
        "X-Masked-Token": "••••••••••••••••••••••••••••••••",
        "X-Another-Valid": "token_abc_xyz",
      },
    })

    expect(fetchMock).toHaveBeenCalledTimes(1)
    const [, config] = fetchMock.mock.calls[0]
    expect(config.headers).toHaveProperty("X-Valid-Header", "valid-value-123")
    expect(config.headers).toHaveProperty("X-Another-Valid", "token_abc_xyz")
    expect(config.headers).not.toHaveProperty("X-Masked-Token")
  })

  it("passes standard valid headers correctly", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response("{}", { status: 200 }))
    global.fetch = fetchMock

    await api.post("/api/v1/test", { name: "test" }, {
      headers: {
        "Custom-Header": "custom-value",
      },
    })

    expect(fetchMock).toHaveBeenCalledTimes(1)
    const [, config] = fetchMock.mock.calls[0]
    expect(config.headers).toHaveProperty("Custom-Header", "custom-value")
    expect(config.headers).toHaveProperty("Content-Type", "application/json")
  })
})

describe("ApiClient FormData support and uploadSessionArtifact", () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it("does not set application/json Content-Type when body is FormData", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response("{}", { status: 200 }))
    global.fetch = fetchMock

    const formData = new FormData()
    formData.append("key", "value")

    await api.post("/api/v1/upload", formData)

    expect(fetchMock).toHaveBeenCalledTimes(1)
    const [, config] = fetchMock.mock.calls[0]
    expect(config.headers["Content-Type"]).toBeUndefined()
    expect(config.body).toBe(formData)
  })

  it("uploadSessionArtifact successfully uploads file and returns artifact", async () => {
    const mockArtifact = {
      id: "art-1",
      projectId: "proj-1",
      sessionId: "session-1",
      name: "diagram.png",
      mimeType: "image/png",
      size: 1024,
    }
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify(mockArtifact), {
        status: 201,
        headers: { "Content-Type": "application/json" },
      })
    )
    global.fetch = fetchMock

    const { uploadSessionArtifact } = await import("./api")
    const testFile = new File(["test-content"], "diagram.png", { type: "image/png" })

    const result = await uploadSessionArtifact("proj-1", "session-1", testFile)

    expect(result).toEqual(mockArtifact)
    expect(fetchMock).toHaveBeenCalledTimes(1)
    const [url, config] = fetchMock.mock.calls[0]
    expect(url).toBe("/api/v1/projects/proj-1/sessions/session-1/artifacts/upload")
    expect(config.method).toBe("POST")
    expect(config.body instanceof FormData).toBe(true)
  })

  it("uploadSessionArtifact encodes special characters in projectId and sessionId", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ id: "art-2" }), {
        status: 201,
        headers: { "Content-Type": "application/json" },
      })
    )
    global.fetch = fetchMock

    const { uploadSessionArtifact } = await import("./api")
    const testFile = new File(["data"], "test.txt", { type: "text/plain" })

    await uploadSessionArtifact("proj/1", "session 1", testFile)

    expect(fetchMock).toHaveBeenCalledTimes(1)
    const [url] = fetchMock.mock.calls[0]
    expect(url).toBe("/api/v1/projects/proj%2F1/sessions/session%201/artifacts/upload")
  })

  it("uploadSessionArtifact throws descriptive error on non-ok response", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ error: "File exceeds 10MB limit" }), {
        status: 400,
        headers: { "Content-Type": "application/json" },
      })
    )
    global.fetch = fetchMock

    const { uploadSessionArtifact } = await import("./api")
    const testFile = new File(["big data"], "large.bin", { type: "application/octet-stream" })

    await expect(uploadSessionArtifact("proj-1", "session-1", testFile)).rejects.toThrow(
      "File exceeds 10MB limit"
    )
  })

  it("uploadSessionArtifact handles non-JSON error responses gracefully", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response("502 Bad Gateway: Proxy Error", {
        status: 502,
        headers: { "Content-Type": "text/plain" },
      })
    )
    global.fetch = fetchMock

    const { uploadSessionArtifact } = await import("./api")
    const testFile = new File(["test"], "doc.txt", { type: "text/plain" })

    await expect(uploadSessionArtifact("proj-1", "session-1", testFile)).rejects.toThrow(
      "502 Bad Gateway: Proxy Error"
    )
  })
})
