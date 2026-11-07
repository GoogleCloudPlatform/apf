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
import { VSCodeArtifactViewer } from "./VSCodeArtifactViewer"

afterEach(() => {
  cleanup()
})

// Mock Monaco Editor and DiffEditor for jsdom environment
vi.mock("@monaco-editor/react", () => ({
  default: ({ value, onChange }: any) => (
    <div data-testid="monaco-editor">
      <textarea
        data-testid="monaco-textarea"
        value={value}
        onChange={(e) => onChange(e.target.value)}
      />
    </div>
  ),
  DiffEditor: ({ original, modified }: any) => (
    <div data-testid="monaco-diff-editor">
      <div data-testid="diff-original">{original}</div>
      <div data-testid="diff-modified">{modified}</div>
    </div>
  ),
}))

// Mock Mermaid diagram component
vi.mock("./MermaidDiagram", () => ({
  MermaidDiagram: ({ chart }: { chart: string }) => (
    <div data-testid="mermaid-diagram">{chart}</div>
  ),
}))

describe("VSCodeArtifactViewer", () => {
  const sampleMarkdown = "# Title\n\nThis is a sample markdown text.\n\n```mermaid\ngraph TD\nA --> B\n```"

  it("renders in editor mode by default and allows editing content", () => {
    const handleContentChange = vi.fn()
    render(
      <VSCodeArtifactViewer
        content={sampleMarkdown}
        onContentChange={handleContentChange}
      />
    )

    // Monaco editor should be rendered by default
    expect(screen.getByTestId("monaco-editor")).toBeDefined()

    // Simulate edit
    const textarea = screen.getByTestId("monaco-textarea")
    fireEvent.change(textarea, { target: { value: "# Updated Title" } })

    expect(handleContentChange).toHaveBeenCalledWith("# Updated Title")
  })

  it("switches to preview mode with formatted text and mermaid support", () => {
    render(<VSCodeArtifactViewer content={sampleMarkdown} />)

    // Click Preview button
    const previewBtn = screen.getByTitle("Rendered Markdown Preview")
    fireEvent.click(previewBtn)

    expect(screen.getByRole("heading", { level: 1, name: "Title" })).toBeDefined()
    expect(screen.getByText("This is a sample markdown text.")).toBeDefined()
    expect(screen.getByTestId("mermaid-diagram")).toBeDefined()
  })

  it("switches to split view mode showing editor and preview", () => {
    render(<VSCodeArtifactViewer content={sampleMarkdown} />)

    const splitBtn = screen.getByTitle("Side-by-Side Editor & Preview")
    fireEvent.click(splitBtn)

    expect(screen.getByTestId("monaco-editor")).toBeDefined()
    expect(screen.getByRole("heading", { level: 1, name: "Title" })).toBeDefined()
  })

  it("shows diff view when changes exist", () => {
    render(
      <VSCodeArtifactViewer
        content="# Modified Content"
        originalContent="# Original Content"
      />
    )

    const diffBtn = screen.getByTitle("Compare Changes (Diff)")
    fireEvent.click(diffBtn)

    expect(screen.getByTestId("monaco-diff-editor")).toBeDefined()
    expect(screen.getByTestId("diff-original").textContent).toBe("# Original Content")
    expect(screen.getByTestId("diff-modified").textContent).toBe("# Modified Content")
  })

  it("triggers save callback when save button is clicked on edited content", () => {
    const handleSave = vi.fn()
    render(
      <VSCodeArtifactViewer
        content="# Draft Spec"
        originalContent="# Base Spec"
        onSave={handleSave}
      />
    )

    const saveBtn = screen.getByTitle("Save changes to artifact")
    fireEvent.click(saveBtn)

    expect(handleSave).toHaveBeenCalledWith("# Draft Spec")
  })

  it("renders inline code inside paragraph without div elements inside p", () => {
    const inlineMarkdown = "Paragraph with `inline code` and **`bold inline`**."
    const { container } = render(<VSCodeArtifactViewer content={inlineMarkdown} />)

    const previewBtn = screen.getByTitle("Rendered Markdown Preview")
    fireEvent.click(previewBtn)

    const p = container.querySelector("p")
    expect(p).not.toBeNull()
    // Verify no <div> elements exist inside <p>
    const divsInP = p?.querySelectorAll("div")
    expect(divsInP?.length).toBe(0)

    const codeElements = p?.querySelectorAll("code")
    expect(codeElements?.length).toBe(2)
    expect(screen.getByText("inline code")).toBeDefined()
    expect(screen.getByText("bold inline")).toBeDefined()
  })

  it("renders code blocks with language header and copy button", () => {
    const codeBlockMarkdown = "```typescript\nconst num: number = 42;\n```"
    render(<VSCodeArtifactViewer content={codeBlockMarkdown} />)

    const previewBtn = screen.getByTitle("Rendered Markdown Preview")
    fireEvent.click(previewBtn)

    expect(screen.getByText(/typescript/i)).toBeDefined()
    expect(screen.getByText("const num: number = 42;")).toBeDefined()
    expect(screen.getByTitle("Copy code snippet")).toBeDefined()
  })

  it("renders HTML content in an iframe in preview mode for html files", () => {
    const sampleHtml = "<html><body><h1>Hello HTML World</h1><p>Rendered paragraph</p></body></html>"
    render(
      <VSCodeArtifactViewer
        content={sampleHtml}
        filePath="public/index.html"
      />
    )

    // Click Preview button
    const previewBtn = screen.getByTitle("Rendered HTML Preview")
    fireEvent.click(previewBtn)

    const iframe = screen.getByTitle("HTML Preview") as HTMLIFrameElement
    expect(iframe).toBeDefined()
    expect(iframe.getAttribute("srcdoc")).toBe(sampleHtml)
  })

  it("renders HTML live preview iframe in split mode for html files", () => {
    const sampleHtml = "<div class='card'><h2>Live Preview</h2></div>"
    render(
      <VSCodeArtifactViewer
        content={sampleHtml}
        filePath="test.html"
      />
    )

    const splitBtn = screen.getByTitle("Side-by-Side Editor & Preview")
    fireEvent.click(splitBtn)

    expect(screen.getByTestId("monaco-editor")).toBeDefined()
    const iframe = screen.getByTitle("HTML Live Preview") as HTMLIFrameElement
    expect(iframe).toBeDefined()
    expect(iframe.getAttribute("srcdoc")).toBe(sampleHtml)
  })

  it("renders HTML in preview mode when artifactType is html", () => {
    const sampleHtml = "<h1>Artifact HTML</h1>"
    render(
      <VSCodeArtifactViewer
        content={sampleHtml}
        artifactType="html"
      />
    )

    const previewBtn = screen.getByTitle("Rendered HTML Preview")
    fireEvent.click(previewBtn)

    const iframe = screen.getByTitle("HTML Preview") as HTMLIFrameElement
    expect(iframe).toBeDefined()
    expect(iframe.getAttribute("srcdoc")).toBe(sampleHtml)
  })

  it("renders HTML in preview mode when content starts with doctype", () => {
    const sampleHtml = "<!DOCTYPE html><html><body><h1>Doctype HTML</h1></body></html>"
    render(
      <VSCodeArtifactViewer
        content={sampleHtml}
      />
    )

    const previewBtn = screen.getByTitle("Rendered HTML Preview")
    fireEvent.click(previewBtn)

    const iframe = screen.getByTitle("HTML Preview") as HTMLIFrameElement
    expect(iframe).toBeDefined()
    expect(iframe.getAttribute("srcdoc")).toBe(sampleHtml)
  })

  it("automatically opens .md files in preview mode", () => {
    const mdContent = "# Auto Preview Markdown\n\nThis is markdown rendered preview."
    render(
      <VSCodeArtifactViewer
        content={mdContent}
        filePath="docs/specs/prd/PRD.md"
      />
    )

    // Should render the rendered markdown heading and text immediately without clicking Preview button
    expect(screen.getByRole("heading", { level: 1, name: "Auto Preview Markdown" })).toBeDefined()
    expect(screen.getByText("This is markdown rendered preview.")).toBeDefined()
    // Monaco editor should not be visible in preview mode
    expect(screen.queryByTestId("monaco-editor")).toBeNull()
  })

  it("automatically opens .html files in preview mode", () => {
    const htmlContent = "<h1>Auto Preview HTML</h1>"
    render(
      <VSCodeArtifactViewer
        content={htmlContent}
        filePath="public/dashboard.html"
      />
    )

    // Should render the HTML iframe immediately without clicking Preview button
    const iframe = screen.getByTitle("HTML Preview") as HTMLIFrameElement
    expect(iframe).toBeDefined()
    expect(iframe.getAttribute("srcdoc")).toBe(htmlContent)
    // Monaco editor should not be visible in preview mode
    expect(screen.queryByTestId("monaco-editor")).toBeNull()
  })

  it("opens code files (.ts, .py, .go) in editor mode by default", () => {
    render(
      <VSCodeArtifactViewer
        content="const x: number = 10;"
        filePath="src/utils/math.ts"
      />
    )

    // Monaco editor should be rendered
    expect(screen.getByTestId("monaco-editor")).toBeDefined()
  })

  it("switches to preview mode when switching from a code file to a .md file", () => {
    const { rerender } = render(
      <VSCodeArtifactViewer
        content="export const a = 1;"
        filePath="src/index.ts"
      />
    )

    expect(screen.getByTestId("monaco-editor")).toBeDefined()

    // Switch to markdown file
    rerender(
      <VSCodeArtifactViewer
        content="# Switching To Markdown"
        filePath="README.md"
      />
    )

    expect(screen.getByRole("heading", { level: 1, name: "Switching To Markdown" })).toBeDefined()
    expect(screen.queryByTestId("monaco-editor")).toBeNull()
  })
})
