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
import BaseBranchConfirmationModal from "./BaseBranchConfirmationModal"

describe("BaseBranchConfirmationModal", () => {
  afterEach(() => {
    cleanup()
  })

  it("does not render when isOpen is false", () => {
    const { container } = render(
      <BaseBranchConfirmationModal
        isOpen={false}
        onClose={vi.fn()}
        onConfirm={vi.fn()}
        baseBranch="custom-branch"
        repoUrl="https://github.com/example/repo"
        isPending={false}
      />
    )
    expect(container).toBeEmptyDOMElement()
  })

  it("renders branch name, repo URL, and responds to button actions", () => {
    const onCloseMock = vi.fn()
    const onConfirmMock = vi.fn()

    render(
      <BaseBranchConfirmationModal
        isOpen={true}
        onClose={onCloseMock}
        onConfirm={onConfirmMock}
        baseBranch="custom-branch"
        repoUrl="https://github.com/example/repo"
        isPending={false}
      />
    )

    expect(screen.getByText("Base Branch Not Found")).toBeInTheDocument()
    expect(screen.getAllByText("custom-branch").length).toBeGreaterThan(0)
    expect(screen.getByText("https://github.com/example/repo")).toBeInTheDocument()

    fireEvent.click(screen.getByRole("button", { name: "Cancel" }))
    expect(onCloseMock).toHaveBeenCalledTimes(1)

    fireEvent.click(screen.getByRole("button", { name: "Confirm & Create Branch" }))
    expect(onConfirmMock).toHaveBeenCalledTimes(1)
  })
})
