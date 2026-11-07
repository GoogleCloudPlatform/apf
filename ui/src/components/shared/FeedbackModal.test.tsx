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

// @ts-nocheck
// @vitest-environment jsdom
import React from "react"
import { render, screen, fireEvent, waitFor, cleanup } from "@testing-library/react"
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import * as matchers from "@testing-library/jest-dom/matchers"
expect.extend(matchers)
import { FeedbackModal } from "./FeedbackModal"
import * as apiModule from "@/lib/api"
import { toast } from "sonner"

vi.mock("sonner", () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
  },
}))

vi.mock("@/stores/auth", () => ({
  useAuthStore: () => ({
    email: "user@example.com",
    selectedOrgId: "org-123",
  }),
}))

describe("FeedbackModal Component", () => {
  const defaultProps = {
    isOpen: true,
    onClose: vi.fn(),
    initialType: "general",
    projectId: "proj-456",
  }

  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    cleanup()
  })

  it("does not render when isOpen is false", () => {
    render(<FeedbackModal {...defaultProps} isOpen={false} />)
    expect(screen.queryByText("Submit Feedback")).not.toBeInTheDocument()
  })

  it("renders modal form elements and pre-fills email from auth store", () => {
    render(<FeedbackModal {...defaultProps} />)

    expect(screen.getByRole("heading", { name: "Submit Feedback" })).toBeInTheDocument()
    expect(screen.getByTestId("feedback-subject-input")).toBeInTheDocument()
    expect(screen.getByTestId("feedback-message-input")).toBeInTheDocument()

    const emailInput = screen.getByTestId("feedback-email-input")
    expect(emailInput).toHaveValue("user@example.com")
  })

  it("allows switching feedback types", () => {
    render(<FeedbackModal {...defaultProps} initialType="general" />)

    const bugBtn = screen.getByTestId("feedback-type-bug")
    const featureBtn = screen.getByTestId("feedback-type-feature")

    fireEvent.click(bugBtn)
    expect(bugBtn.className).toContain("border-red-500")

    fireEvent.click(featureBtn)
    expect(featureBtn.className).toContain("border-purple-500")
  })

  it("validates empty subject and message fields", async () => {
    render(<FeedbackModal {...defaultProps} />)

    const form = screen.getByTestId("feedback-subject-input").closest("form")!
    fireEvent.submit(form)

    expect(screen.getByRole("alert")).toHaveTextContent("Subject is required.")
  })

  it("submits feedback payload successfully and closes modal", async () => {
    const submitSpy = vi.spyOn(apiModule, "submitFeedback").mockResolvedValue({
      id: "fb-123",
      status: "success",
      message: "Feedback submitted successfully",
    })

    render(<FeedbackModal {...defaultProps} initialType="bug" />)

    fireEvent.change(screen.getByTestId("feedback-subject-input"), {
      target: { value: "UI rendering issue" },
    })
    fireEvent.change(screen.getByTestId("feedback-message-input"), {
      target: { value: "Canvas flickers on zoom." },
    })

    const form = screen.getByTestId("feedback-subject-input").closest("form")!
    fireEvent.submit(form)

    await waitFor(() => {
      expect(submitSpy).toHaveBeenCalledWith({
        type: "bug",
        subject: "UI rendering issue",
        message: "Canvas flickers on zoom.",
        userEmail: "user@example.com",
        organizationId: "org-123",
        projectId: "proj-456",
      })
    })

    expect(toast.success).toHaveBeenCalledWith(
      "Feedback submitted successfully! Thank you for helping us improve."
    )
    expect(defaultProps.onClose).toHaveBeenCalled()
  })

  it("closes modal on backdrop click, cancel button, or Escape key press", () => {
    const onCloseMock = vi.fn()
    const { rerender } = render(<FeedbackModal {...defaultProps} onClose={onCloseMock} />)

    // Backdrop click
    const backdrop = screen.getByTestId("feedback-modal-backdrop")
    fireEvent.click(backdrop)
    expect(onCloseMock).toHaveBeenCalledTimes(1)

    // Cancel button
    const cancelBtn = screen.getByTestId("feedback-cancel-btn")
    fireEvent.click(cancelBtn)
    expect(onCloseMock).toHaveBeenCalledTimes(2)

    // Close X button
    const closeBtn = screen.getByTestId("feedback-modal-close-btn")
    fireEvent.click(closeBtn)
    expect(onCloseMock).toHaveBeenCalledTimes(3)

    // Escape key
    fireEvent.keyDown(window, { key: "Escape" })
    expect(onCloseMock).toHaveBeenCalledTimes(4)
  })
})
