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
import { render, screen, cleanup } from "@testing-library/react"
import { describe, it, expect, vi, afterEach } from "vitest"
import { ThinkingBlock, getStepSummary, formatToolTitle } from "./ThinkingBlock"
import { Message } from "@/lib/types"

describe("formatToolTitle", () => {
  it("formats snake_case tool name to Sentence case with spaces", () => {
    expect(formatToolTitle("transfer_to_agent")).toBe("Transfer to agent")
    expect(formatToolTitle("execute_command")).toBe("Execute command")
    expect(formatToolTitle("web_search")).toBe("Web search")
  })

  it("handles quotes and backticks around tool names", () => {
    expect(formatToolTitle("'transfer_to_agent'")).toBe("Transfer to agent")
    expect(formatToolTitle("`transfer_to_agent`")).toBe("Transfer to agent")
    expect(formatToolTitle('"transfer_to_agent"')).toBe("Transfer to agent")
  })

  it("handles kebab-case tool names", () => {
    expect(formatToolTitle("transfer-to-agent")).toBe("Transfer to agent")
    expect(formatToolTitle("bmad-help")).toBe("Bmad help")
  })

  it("handles camelCase tool names", () => {
    expect(formatToolTitle("transferToAgent")).toBe("Transfer To Agent")
  })

  it("handles empty or falsy inputs", () => {
    expect(formatToolTitle("")).toBe("Tool result")
    expect(formatToolTitle("   ")).toBe("Tool result")
  })
})

describe("getStepSummary", () => {
  it("transforms Tool result('transfer_to_agent') received to 'Transfer to agent'", () => {
    expect(getStepSummary("Tool result('transfer_to_agent') received")).toBe("Transfer to agent")
    expect(getStepSummary("Tool result ('transfer_to_agent') received")).toBe("Transfer to agent")
    expect(getStepSummary("Tool result (`transfer_to_agent`) received")).toBe("Transfer to agent")
    expect(getStepSummary('Tool result ("transfer_to_agent") received')).toBe("Transfer to agent")
    expect(getStepSummary("Tool result (transfer_to_agent) received")).toBe("Transfer to agent")
  })

  it("transforms Tool result with colon or multiline content", () => {
    expect(getStepSummary("Tool result (`transfer_to_agent`):\n```\noutput\n```")).toBe("Transfer to agent")
    expect(getStepSummary("Tool result('transfer_to_agent'): output")).toBe("Transfer to agent")
    expect(getStepSummary("**Tool result('transfer_to_agent') received**")).toBe("Transfer to agent")
  })

  it("extracts tool name from notificationDetails.params.tool_name", () => {
    const msg: Message = {
      sender: "Agent",
      text: "Some arbitrary text",
      isUser: false,
      notificationDetails: {
        name: "tool-result",
        params: {
          tool_name: "transfer_to_agent",
          content: "switched agent successfully",
        },
      },
    }
    expect(getStepSummary(msg)).toBe("Transfer to agent")
  })

  it("prioritizes file reader tool if params indicate is_file_reader or file_name", () => {
    const msg: Message = {
      sender: "Agent",
      text: "Read file: src/app.ts",
      isUser: false,
      notificationDetails: {
        name: "tool-result",
        params: {
          is_file_reader: true,
          file_name: "src/app.ts",
          tool_name: "read_file",
        },
      },
    }
    expect(getStepSummary(msg)).toBe("Read file: src/app.ts")
  })

  it("handles Read file text pattern", () => {
    expect(getStepSummary("Read file: `config.json`")).toBe("Read file: config.json")
    expect(getStepSummary("**Read file: index.ts**")).toBe("Read file: index.ts")
  })

  it("falls back to cleaned first line if not a tool result or file reader", () => {
    expect(getStepSummary("Thinking about the architecture...")).toBe("Thinking about the architecture...")
  })
})

describe("ThinkingBlock Component", () => {
  afterEach(() => {
    cleanup()
  })

  it("renders converted title for Tool result steps in accordion items", () => {
    const steps: Message[] = [
      {
        id: "step-1",
        sender: "Agent",
        text: "Tool result('transfer_to_agent') received",
        isUser: false,
      },
    ]

    render(<ThinkingBlock thinkingSteps={steps} isStreaming={false} />)

    expect(screen.getByText("Transfer to agent")).not.toBeNull()
  })

  it("renders converted title when thinkingText is provided", () => {
    render(
      <ThinkingBlock
        thinkingText="Tool result (`transfer_to_agent`) received"
        isStreaming={false}
      />
    )

    expect(screen.getByText("Transfer to agent")).not.toBeNull()
  })
})
