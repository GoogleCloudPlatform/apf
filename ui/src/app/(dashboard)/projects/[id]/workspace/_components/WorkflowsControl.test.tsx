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
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import "@testing-library/jest-dom/vitest"
import * as matchers from "@testing-library/jest-dom/matchers"
expect.extend(matchers as any)

import WorkflowsControl from "./WorkflowsControl"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"

vi.mock("next/navigation", () => ({
  useParams: () => ({ id: "test-project-id" }),
}))

vi.mock("@/lib/api", () => ({
  api: {
    get: vi.fn(),
    post: vi.fn(),
  },
}))

import { api } from "@/lib/api"

const mockWorkflows = [
  {
    id: "wf-1",
    name: "test-workflow",
    displayName: "Test Workflow",
    framework: "ADK 2.0",
    description: "A test workflow",
  },
]

const renderWithQueryClient = (ui: React.ReactElement) => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  })
  return render(<QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>)
}

describe("WorkflowsControl", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    ;(api.get as any).mockResolvedValue({
      ok: true,
      json: async () => mockWorkflows,
    })
  })

  afterEach(() => {
    cleanup()
  })

  it("renders linked workflows and opens Coming Soon modal on clicking Invoke Workflow", async () => {
    renderWithQueryClient(<WorkflowsControl linkedWorkflowIds={["wf-1"]} />)

    const workflowTitle = await screen.findByText("Test Workflow")
    expect(workflowTitle).toBeInTheDocument()

    const invokeButton = screen.getByTitle("Invoke Workflow")
    expect(invokeButton).toBeInTheDocument()

    fireEvent.click(invokeButton)

    expect(screen.getByText("Coming Soon!")).toBeInTheDocument()
    expect(screen.getByText("Invoke Workflow feature is coming soon!")).toBeInTheDocument()
  })
})
