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
import { render, screen, fireEvent, cleanup } from "@testing-library/react"
import { describe, it, expect, vi, afterEach } from "vitest"
import PhaseSwitchCard from "./PhaseSwitchCard"

describe("PhaseSwitchCard", () => {
  afterEach(() => {
    cleanup()
  })
  it("renders target phase badge, skill ID, and reason correctly", () => {
    const handleConfirm = vi.fn()

    render(
      <PhaseSwitchCard
        targetPhase="PLANNING"
        targetPhaseLabel="Planning"
        skillId="create-prd"
        reason="Creating a PRD is handled by John PM in the Planning tab."
        onConfirmSwitch={handleConfirm}
      />
    )

    expect(screen.getByText(/Recommended Phase Switch: Plan/i)).toBeDefined()
    expect(screen.getByText(/Skill: create-prd/i)).toBeDefined()
    expect(screen.getByText(/Creating a PRD is handled by John PM in the Planning tab\./i)).toBeDefined()
    expect(screen.getByRole("button", { name: /Switch to Plan Tab & Continue/i })).toBeDefined()
  })

  it("calls onConfirmSwitch with targetPhase and default or suggested prompt when button is clicked", () => {
    const handleConfirm = vi.fn()

    render(
      <PhaseSwitchCard
        targetPhase="PLANNING"
        skillId="create-prd"
        suggestedPrompt="Let's create a PRD"
        onConfirmSwitch={handleConfirm}
      />
    )

    const button = screen.getByRole("button", { name: /Switch to Plan Tab & Continue/i })
    fireEvent.click(button)

    expect(handleConfirm).toHaveBeenCalledTimes(1)
    expect(handleConfirm).toHaveBeenCalledWith("PLANNING", "Let's create a PRD")
  })
})
