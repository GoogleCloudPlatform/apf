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
import { render, screen, fireEvent, cleanup } from "@testing-library/react"
import { describe, it, expect, vi, afterEach } from "vitest"
import "@testing-library/jest-dom/vitest"
import ProjectHeader from "./ProjectHeader"

vi.mock("next/navigation", () => ({
  useRouter: () => ({
    push: vi.fn(),
  }),
}))

describe("ProjectHeader component", () => {
  afterEach(() => {
    cleanup()
  })

  it("renders My Projects and Shared Projects tabs", () => {
    render(
      <ProjectHeader
        activeTab="my"
        setActiveTab={vi.fn()}
        viewMode="card"
        setViewMode={vi.fn()}
      />
    )

    expect(screen.getByText("My Projects")).toBeInTheDocument()
    expect(screen.getByText("Shared Projects")).toBeInTheDocument()
  })

  it("calls onSharedProjectsClick and does NOT call setActiveTab('shared') when onSharedProjectsClick is provided", () => {
    const setActiveTabMock = vi.fn()
    const onSharedProjectsClickMock = vi.fn()

    render(
      <ProjectHeader
        activeTab="my"
        setActiveTab={setActiveTabMock}
        viewMode="card"
        setViewMode={vi.fn()}
        onSharedProjectsClick={onSharedProjectsClickMock}
      />
    )

    fireEvent.click(screen.getByText("Shared Projects"))

    expect(onSharedProjectsClickMock).toHaveBeenCalledTimes(1)
    expect(setActiveTabMock).not.toHaveBeenCalledWith("shared")
  })

  it("calls setActiveTab('shared') when onSharedProjectsClick is not provided", () => {
    const setActiveTabMock = vi.fn()

    render(
      <ProjectHeader
        activeTab="my"
        setActiveTab={setActiveTabMock}
        viewMode="card"
        setViewMode={vi.fn()}
      />
    )

    fireEvent.click(screen.getByText("Shared Projects"))

    expect(setActiveTabMock).toHaveBeenCalledWith("shared")
  })
})
