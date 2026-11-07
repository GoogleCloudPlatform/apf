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
import React from "react"
import { render, screen, fireEvent, cleanup, waitFor } from "@testing-library/react"
import { describe, it, expect, vi, afterEach, beforeEach } from "vitest"
import { ImageViewer } from "./ImageViewer"
import { api } from "@/lib/api"

vi.mock("@/lib/api", () => ({
  api: {
    get: vi.fn((url: string) => {
      if (url.includes("invalid")) {
        return Promise.resolve({ ok: false, status: 404 })
      }
      return Promise.resolve({
        ok: true,
        blob: () => Promise.resolve(new Blob(["fake-image-data"], { type: "image/png" })),
      })
    }),
  },
}))

if (typeof window.URL.createObjectURL === "undefined") {
  window.URL.createObjectURL = vi.fn((blob: any) => "blob:http://localhost/mock-blob-image")
} else {
  vi.spyOn(window.URL, "createObjectURL").mockImplementation(() => "blob:http://localhost/mock-blob-image")
}
if (typeof window.URL.revokeObjectURL === "undefined") {
  window.URL.revokeObjectURL = vi.fn()
} else {
  vi.spyOn(window.URL, "revokeObjectURL").mockImplementation(() => {})
}

afterEach(() => {
  cleanup()
  vi.clearAllMocks()
})

describe("ImageViewer", () => {
  const defaultSrc = "/api/v1/projects/proj-1/artifacts/diagram.png/content"

  it("renders loading state initially and then renders image element", async () => {
    render(<ImageViewer src={defaultSrc} alt="Architecture Diagram" />)

    expect(screen.getByTestId("image-viewer-loading")).toBeDefined()
    const img = (await screen.findByTestId("image-viewer-img")) as HTMLImageElement
    expect(img).toBeDefined()
    expect(img.getAttribute("alt")).toBe("Architecture Diagram")
  })

  it("measures and reports natural dimensions on image load", () => {
    const onDimensionsLoaded = vi.fn()
    render(
      <ImageViewer
        src={defaultSrc}
        alt="Diagram"
        onDimensionsLoaded={onDimensionsLoaded}
      />
    )

    const img = screen.getByTestId("image-viewer-img") as HTMLImageElement
    Object.defineProperty(img, "naturalWidth", { value: 1920, configurable: true })
    Object.defineProperty(img, "naturalHeight", { value: 1080, configurable: true })

    fireEvent.load(img)

    expect(onDimensionsLoaded).toHaveBeenCalledWith({ width: 1920, height: 1080 })
    expect(screen.queryByTestId("image-viewer-loading")).toBeNull()
  })

  it("handles zoom in, zoom out, and reset zoom interactions", () => {
    render(<ImageViewer src={defaultSrc} alt="Diagram" />)

    const img = screen.getByTestId("image-viewer-img")
    fireEvent.load(img)

    const zoomLevel = screen.getByTestId("zoom-level")
    expect(zoomLevel.textContent).toBe("100%")

    const zoomInBtn = screen.getByTestId("zoom-in-btn")
    const zoomOutBtn = screen.getByTestId("zoom-out-btn")
    const resetBtn = screen.getByTestId("reset-zoom-btn")

    // Zoom in: 100% -> 125%
    fireEvent.click(zoomInBtn)
    expect(zoomLevel.textContent).toBe("125%")

    // Zoom in again: 125% -> 150%
    fireEvent.click(zoomInBtn)
    expect(zoomLevel.textContent).toBe("150%")

    // Zoom out: 150% -> 125%
    fireEvent.click(zoomOutBtn)
    expect(zoomLevel.textContent).toBe("125%")

    // Reset zoom: 125% -> 100%
    fireEvent.click(resetBtn)
    expect(zoomLevel.textContent).toBe("100%")
  })

  it("enforces zoom boundary limits between 25% and 500%", () => {
    render(<ImageViewer src={defaultSrc} alt="Diagram" />)

    const zoomInBtn = screen.getByTestId("zoom-in-btn")
    const zoomOutBtn = screen.getByTestId("zoom-out-btn")
    const zoomLevel = screen.getByTestId("zoom-level")

    // Zoom out to minimum 25%
    for (let i = 0; i < 6; i++) {
      fireEvent.click(zoomOutBtn)
    }
    expect(zoomLevel.textContent).toBe("25%")
    expect(zoomOutBtn.hasAttribute("disabled")).toBe(true)

    // Zoom in to maximum 500%
    for (let i = 0; i < 25; i++) {
      fireEvent.click(zoomInBtn)
    }
    expect(zoomLevel.textContent).toBe("500%")
    expect(zoomInBtn.hasAttribute("disabled")).toBe(true)
  })

  it("supports panning via mouse drag events and ignores non-primary mouse clicks", () => {
    render(<ImageViewer src={defaultSrc} alt="Diagram" />)

    const container = screen.getByTestId("image-viewer-container")
    const img = screen.getByTestId("image-viewer-img")
    fireEvent.load(img)

    // Non-primary click (button 1/2) should not start drag
    fireEvent.mouseDown(container, { button: 2, clientX: 100, clientY: 100 })
    fireEvent.mouseMove(window, { clientX: 150, clientY: 180 })
    const transformWrapper = img.parentElement as HTMLElement
    expect(transformWrapper.style.transform).toContain("translate(0px, 0px)")

    // Mouse down at (100, 100)
    fireEvent.mouseDown(container, { button: 0, clientX: 100, clientY: 100 })

    // Global mouse move to (150, 180) -> delta (+50, +80)
    fireEvent.mouseMove(window, { clientX: 150, clientY: 180 })
    expect(transformWrapper.style.transform).toContain("translate(50px, 80px)")

    // Global mouse up ends dragging
    fireEvent.mouseUp(window)

    // Further move should not change transform
    fireEvent.mouseMove(window, { clientX: 200, clientY: 200 })
    expect(transformWrapper.style.transform).toContain("translate(50px, 80px)")
  })

  it("supports zooming via mouse wheel events", () => {
    render(<ImageViewer src={defaultSrc} alt="Diagram" />)

    const container = screen.getByTestId("image-viewer-container")
    const img = screen.getByTestId("image-viewer-img")
    fireEvent.load(img)

    const zoomLevel = screen.getByTestId("zoom-level")
    expect(zoomLevel.textContent).toBe("100%")

    // Wheel up zooms in
    fireEvent.wheel(container, { deltaY: -100 })
    expect(zoomLevel.textContent).toBe("125%")

    // Wheel down zooms out
    fireEvent.wheel(container, { deltaY: 100 })
    expect(zoomLevel.textContent).toBe("100%")
  })

  it("displays error state when image fails to load and allows retry", () => {
    const onDimensionsLoaded = vi.fn()
    const onRetry = vi.fn()
    render(
      <ImageViewer
        src="/invalid-image.png"
        alt="Invalid"
        onDimensionsLoaded={onDimensionsLoaded}
        onRetry={onRetry}
      />
    )

    const img = screen.getByTestId("image-viewer-img")
    fireEvent.error(img)

    expect(screen.getByTestId("image-viewer-error")).toBeDefined()
    expect(screen.getByText("Failed to load image")).toBeDefined()
    expect(onDimensionsLoaded).toHaveBeenCalledWith(null)
    expect(screen.queryByTestId("image-viewer-controls")).toBeNull()

    const retryBtn = screen.getByTestId("image-retry-btn")
    fireEvent.click(retryBtn)

    expect(onRetry).toHaveBeenCalled()
    expect(screen.queryByTestId("image-viewer-error")).toBeNull()
    expect(screen.getByTestId("image-viewer-loading")).toBeDefined()
  })
})
