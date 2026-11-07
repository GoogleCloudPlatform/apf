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
import RepoChangeWarningModal from "./RepoChangeWarningModal"

describe("RepoChangeWarningModal", () => {
  afterEach(() => {
    cleanup()
  })

  it("does not render when isOpen is false", () => {
    const { container } = render(
      <RepoChangeWarningModal
        isOpen={false}
        onClose={vi.fn()}
        onConfirm={vi.fn()}
        newRepoUrl="https://github.com/example/repo-b"
      />
    )
    expect(container).toBeEmptyDOMElement()
  })

  it("renders warning when changing from another repo", () => {
    const onCloseMock = vi.fn()
    const onConfirmMock = vi.fn()

    render(
      <RepoChangeWarningModal
        isOpen={true}
        onClose={onCloseMock}
        onConfirm={onConfirmMock}
        initialRepoUrl="https://github.com/example/repo-a"
        newRepoUrl="https://github.com/example/repo-b"
      />
    )

    expect(screen.getByText("Change Repository")).toBeInTheDocument()
    expect(screen.getByText("https://github.com/example/repo-a")).toBeInTheDocument()
    expect(screen.getByText("https://github.com/example/repo-b")).toBeInTheDocument()
    expect(screen.getByText(/will delete your currently cached local workspace data/i)).toBeInTheDocument()

    fireEvent.click(screen.getByRole("button", { name: "Cancel" }))
    expect(onCloseMock).toHaveBeenCalledTimes(1)

    fireEvent.click(screen.getByRole("button", { name: "Proceed" }))
    expect(onConfirmMock).toHaveBeenCalledTimes(1)
  })

  it("renders warning when changing from empty repo", () => {
    render(
      <RepoChangeWarningModal
        isOpen={true}
        onClose={vi.fn()}
        onConfirm={vi.fn()}
        initialRepoUrl=""
        newRepoUrl="https://github.com/example/repo-new"
      />
    )

    expect(screen.getByText("Connect Repository")).toBeInTheDocument()
    expect(screen.getByText("https://github.com/example/repo-new")).toBeInTheDocument()
    expect(screen.getByText(/will replace your current workspace data/i)).toBeInTheDocument()
  })
})
