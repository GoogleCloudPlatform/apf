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
import { render, screen, cleanup } from "@testing-library/react"
import { describe, it, expect, afterEach } from "vitest"
import * as matchers from "@testing-library/jest-dom/matchers"
expect.extend(matchers)
import { GitSyncRunner, GitSyncChecklistResponse } from "./GitSyncRunner"

describe("GitSyncRunner Component", () => {
  afterEach(() => {
    cleanup()
  })

  const mockData: GitSyncChecklistResponse = {
    status: "SUCCESS",
    message: "Workspace successfully synchronized.",
    dynamicBranches: {
      baseBranch: "main",
      specBranch: "specs/user1",
    },
    steps: [
      { id: "step1", label: "Create Safety Snapshot", status: "completed" },
      { id: "step2", label: "Fast-Forward Workspace", status: "completed" },
    ],
  }

  it("renders process checklist and branch badges", () => {
    render(<GitSyncRunner data={mockData} />)

    expect(screen.getByText(/Base: main/i)).toBeInTheDocument()
    expect(screen.getByText(/Spec: specs\/user1/i)).toBeInTheDocument()
    expect(screen.getByText(/Create Safety Snapshot/i)).toBeInTheDocument()
    expect(screen.getByText(/Fast-Forward Workspace/i)).toBeInTheDocument()
  })
})
