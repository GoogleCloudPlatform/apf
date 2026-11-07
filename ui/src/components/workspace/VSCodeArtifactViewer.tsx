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

"use client"

import React, { useState, useEffect, useMemo, useCallback } from "react"
import Editor, { DiffEditor, useMonaco } from "@monaco-editor/react"
import ReactMarkdown from "react-markdown"
import remarkGfm from "remark-gfm"
import { 
  Code2, 
  Eye, 
  Columns, 
  GitCompare, 
  Save, 
  RotateCcw, 
  Copy, 
  Check, 
  WrapText, 
  ExternalLink,
  Sparkles,
  Loader2
} from "lucide-react"
import { cn } from "@/lib/utils"
import { useThemeStore } from "@/stores/theme"
import { MermaidDiagram } from "./MermaidDiagram"

export type ViewerMode = "editor" | "preview" | "split" | "diff"

interface VSCodeArtifactViewerProps {
  content: string
  originalContent?: string
  filePath?: string
  artifactType?: string
  isReadOnly?: boolean
  initialMode?: ViewerMode
  onContentChange?: (newContent: string) => void
  onSave?: (savedContent: string) => void
  isSaving?: boolean
}

// Helper to determine default viewer mode from file extension or artifactType (.md and .html open in preview mode)
export function getDefaultViewerMode(filePath?: string, artifactType?: string): ViewerMode {
  if (!filePath && !artifactType) return "editor"
  const ext = filePath ? filePath.split(".").pop()?.toLowerCase() : ""
  if (
    ext === "md" ||
    ext === "markdown" ||
    ext === "html" ||
    ext === "htm" ||
    artifactType?.toLowerCase() === "html" ||
    artifactType?.toLowerCase() === "markdown" ||
    artifactType?.toLowerCase() === "md" ||
    artifactType?.toLowerCase() === "prd" ||
    artifactType?.toLowerCase() === "architecture" ||
    artifactType?.toLowerCase() === "ux" ||
    artifactType?.toLowerCase() === "epics"
  ) {
    return "preview"
  }
  return "editor"
}

// Helper to determine Monaco language from file extension or artifactType
function getLanguageFromPath(filePath?: string, artifactType?: string): string {
  if (artifactType?.toLowerCase() === "html") return "html"
  if (!filePath) return "markdown"
  const ext = filePath.split(".").pop()?.toLowerCase()
  switch (ext) {
    case "md":
    case "markdown":
      return "markdown"
    case "json":
      return "json"
    case "ts":
      return "typescript"
    case "tsx":
      return "typescript"
    case "js":
    case "jsx":
      return "javascript"
    case "go":
      return "go"
    case "py":
      return "python"
    case "yaml":
    case "yml":
      return "yaml"
    case "html":
    case "htm":
      return "html"
    case "css":
      return "css"
    case "sh":
    case "bash":
      return "shell"
    default:
      return "markdown"
  }
}

function extractTextContent(node: React.ReactNode): string {
  if (typeof node === "string") return node
  if (typeof node === "number") return String(node)
  if (!node) return ""
  if (Array.isArray(node)) return node.map(extractTextContent).join("")
  if (React.isValidElement(node) && node.props && (node.props as { children?: React.ReactNode }).children) {
    return extractTextContent((node.props as { children?: React.ReactNode }).children)
  }
  return ""
}

interface PreBlockProps extends React.ComponentPropsWithoutRef<"pre"> {
  node?: any
  children?: React.ReactNode
}

function CustomPreBlock({ children, node, ...props }: PreBlockProps) {
  const [isCopied, setIsCopied] = useState(false)

  // Extract code child props if available
  const codeElement = React.isValidElement(children) ? children : null
  const className = (codeElement?.props as { className?: string })?.className || ""
  const match = /language-(\w+)/.exec(className)
  const language = match ? match[1] : ""
  const codeString = extractTextContent(children).replace(/\n$/, "")

  // Check if dark theme is active in DOM
  const isDarkTheme = typeof document !== "undefined" && document.documentElement.classList.contains("dark")

  if (language === "mermaid") {
    return <MermaidDiagram chart={codeString} isDark={isDarkTheme} />
  }

  const handleCopyCode = async () => {
    try {
      await navigator.clipboard.writeText(codeString)
      setIsCopied(true)
      setTimeout(() => setIsCopied(false), 2000)
    } catch (e) {
      console.error("Failed to copy code snippet", e)
    }
  }

  return (
    <div className="relative group/code my-4 rounded-xl border border-border bg-slate-900 text-slate-100 overflow-hidden font-mono text-xs not-prose">
      <div className="flex items-center justify-between px-3.5 py-1.5 bg-slate-950/80 border-b border-border/40 text-[10px] text-slate-400">
        <span className="font-semibold uppercase tracking-wider">{language || "text"}</span>
        <button
          onClick={handleCopyCode}
          className="flex items-center gap-1 hover:text-slate-200 transition-colors p-1 rounded hover:bg-slate-800"
          title="Copy code snippet"
        >
          {isCopied ? (
            <>
              <Check className="h-3 w-3 text-green-400" />
              <span className="text-green-400 font-sans">Copied!</span>
            </>
          ) : (
            <>
              <Copy className="h-3 w-3" />
              <span className="font-sans">Copy</span>
            </>
          )}
        </button>
      </div>
      <pre className="p-4 overflow-x-auto leading-relaxed bg-transparent my-0 rounded-none border-none text-slate-100 font-mono text-xs">
        <code>{codeString}</code>
      </pre>
    </div>
  )
}

function CustomInlineCode({ className, children, ...props }: React.ComponentPropsWithoutRef<"code">) {
  return (
    <code
      className={cn(
        "px-1.5 py-0.5 rounded-md bg-muted text-foreground font-mono text-[11px] border border-border/50",
        className
      )}
      {...props}
    >
      {children}
    </code>
  )
}

const markdownComponents = {
  pre: CustomPreBlock,
  code: CustomInlineCode,
  table: ({ children }: { children?: React.ReactNode }) => (
    <div className="my-4 overflow-x-auto rounded-lg border border-border">
      <table className="w-full text-left text-xs border-collapse divide-y divide-border">
        {children}
      </table>
    </div>
  ),
  th: ({ children }: { children?: React.ReactNode }) => (
    <th className="bg-muted/80 px-4 py-2 font-semibold text-foreground border-b border-border">
      {children}
    </th>
  ),
  td: ({ children }: { children?: React.ReactNode }) => (
    <td className="px-4 py-2 border-b border-border/40 text-muted-foreground">
      {children}
    </td>
  ),
}

export function VSCodeArtifactViewer({
  content,
  originalContent,
  filePath,
  artifactType,
  isReadOnly = false,
  initialMode,
  onContentChange,
  onSave,
  isSaving = false,
}: VSCodeArtifactViewerProps) {
  const [currentMode, setCurrentMode] = useState<ViewerMode>(() => {
    if (initialMode) return initialMode
    return getDefaultViewerMode(filePath, artifactType)
  })
  const [editedValue, setEditedValue] = useState(content)
  const [isWordWrap, setIsWordWrap] = useState(true)
  const { themeMode } = useThemeStore()

  // Detect whether dark mode is currently active
  const isDark = useMemo(() => {
    if (typeof document !== "undefined") {
      return document.documentElement.classList.contains("dark")
    }
    return themeMode === "dark"
  }, [themeMode])

  // Sync internal editedValue if incoming content changes and is not dirty
  useEffect(() => {
    setEditedValue(content)
  }, [content])

  // Automatically switch mode when opening/selecting a new file
  useEffect(() => {
    if (initialMode) {
      setCurrentMode(initialMode)
    } else {
      setCurrentMode(getDefaultViewerMode(filePath, artifactType))
    }
  }, [filePath, artifactType, initialMode])

  const isDirty = useMemo(() => {
    return editedValue !== (originalContent ?? content)
  }, [editedValue, originalContent, content])

  const language = useMemo(() => getLanguageFromPath(filePath, artifactType), [filePath, artifactType])

  const isHtml = useMemo(() => {
    if (language === "html") return true
    if (artifactType?.toLowerCase() === "html") return true
    if (filePath) {
      const ext = filePath.split(".").pop()?.toLowerCase()
      if (ext === "html" || ext === "htm") return true
    }
    const trimmed = (editedValue || "").trim().toLowerCase()
    if (
      trimmed.startsWith("<!doctype html") ||
      trimmed.startsWith("<html") ||
      trimmed.startsWith("<!doctype")
    ) {
      return true
    }
    return false
  }, [language, artifactType, filePath, editedValue])

  const handleEditorChange = useCallback((value: string | undefined) => {
    const newVal = value ?? ""
    setEditedValue(newVal)
    onContentChange?.(newVal)
  }, [onContentChange])

  const handleReset = () => {
    const base = originalContent ?? content
    setEditedValue(base)
    onContentChange?.(base)
  }

  const handleSaveClick = () => {
    onSave?.(editedValue)
  }

  // Handle keyboard shortcut Ctrl+S / Cmd+S to save
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "s") {
        e.preventDefault()
        if (onSave && isDirty && !isSaving && !isReadOnly) {
          onSave(editedValue)
        }
      }
    }
    window.addEventListener("keydown", handleKeyDown)
    return () => window.removeEventListener("keydown", handleKeyDown)
  }, [onSave, isDirty, isSaving, isReadOnly, editedValue])

  const monacoTheme = isDark ? "vs-dark" : "vs"

  return (
    <div className="flex flex-col h-full w-full bg-background overflow-hidden relative select-text">
      
      {/* 1. Mode Switcher & Utility Toolbar */}
      <div className="h-10 px-4 border-b border-border bg-card flex items-center justify-between shrink-0 select-none text-xs">
        
        {/* Left: View Mode Segmented Controls (Editor first, then Preview, Split, Diff) */}
        <div className="flex items-center gap-1 bg-muted/60 p-0.5 rounded-lg border border-border/50">
          <button
            onClick={() => setCurrentMode("editor")}
            className={cn(
              "flex items-center gap-1.5 px-2.5 py-1 rounded-md font-medium transition-all",
              currentMode === "editor"
                ? "bg-background text-foreground shadow-sm"
                : "text-muted-foreground hover:text-foreground"
            )}
            title="VS Code / Monaco Raw Editor"
          >
            <Code2 className="h-3.5 w-3.5" />
            <span>Editor</span>
          </button>

          <button
            onClick={() => setCurrentMode("preview")}
            className={cn(
              "flex items-center gap-1.5 px-2.5 py-1 rounded-md font-medium transition-all",
              currentMode === "preview"
                ? "bg-background text-foreground shadow-sm"
                : "text-muted-foreground hover:text-foreground"
            )}
            title={isHtml ? "Rendered HTML Preview" : "Rendered Markdown Preview"}
          >
            <Eye className="h-3.5 w-3.5" />
            <span>Preview</span>
          </button>

          <button
            onClick={() => setCurrentMode("split")}
            className={cn(
              "flex items-center gap-1.5 px-2.5 py-1 rounded-md font-medium transition-all",
              currentMode === "split"
                ? "bg-background text-foreground shadow-sm"
                : "text-muted-foreground hover:text-foreground"
            )}
            title="Side-by-Side Editor & Preview"
          >
            <Columns className="h-3.5 w-3.5" />
            <span>Split</span>
          </button>

          {(originalContent !== undefined || isDirty) && (
            <button
              onClick={() => setCurrentMode("diff")}
              className={cn(
                "flex items-center gap-1.5 px-2.5 py-1 rounded-md font-medium transition-all",
                currentMode === "diff"
                  ? "bg-background text-foreground shadow-sm"
                  : "text-muted-foreground hover:text-foreground"
              )}
              title="Compare Changes (Diff)"
            >
              <GitCompare className="h-3.5 w-3.5" />
              <span>Diff</span>
              {isDirty && (
                <span className="h-1.5 w-1.5 rounded-full bg-primary animate-pulse" />
              )}
            </button>
          )}
        </div>

        {/* Right: Quick actions (Save, Revert, Word Wrap, Dirty status) */}
        <div className="flex items-center gap-2">
          {isDirty && (
            <span className="hidden sm:inline-flex items-center gap-1 text-[11px] font-medium text-amber-500 bg-amber-500/10 px-2 py-0.5 rounded-full border border-amber-500/20">
              <span className="h-1.5 w-1.5 rounded-full bg-amber-500" />
              Unsaved Edits
            </span>
          )}

          {/* Word Wrap Toggle */}
          <button
            onClick={() => setIsWordWrap(!isWordWrap)}
            className={cn(
              "flex items-center gap-1 px-2 py-1 rounded-md border text-[11px] font-medium transition-colors",
              isWordWrap
                ? "border-primary/30 bg-primary/10 text-primary"
                : "border-border text-muted-foreground hover:text-foreground hover:bg-muted"
            )}
            title={isWordWrap ? "Disable Word Wrap" : "Enable Word Wrap"}
          >
            <WrapText className="h-3 w-3" />
            <span className="hidden sm:inline">Wrap</span>
          </button>

          {/* Revert edits button */}
          {isDirty && (
            <button
              onClick={handleReset}
              className="flex items-center gap-1 px-2 py-1 rounded-md border border-border text-muted-foreground hover:text-destructive hover:bg-destructive/10 text-[11px] font-medium transition-colors"
              title="Revert to original content"
            >
              <RotateCcw className="h-3 w-3" />
              <span className="hidden sm:inline">Revert</span>
            </button>
          )}

          {/* Save button */}
          {onSave && isDirty && (
            <button
              onClick={handleSaveClick}
              disabled={isSaving}
              className="flex items-center gap-1.5 px-3 py-1 rounded-md bg-primary text-primary-foreground font-semibold text-[11px] hover:opacity-90 transition-opacity disabled:opacity-50"
              title="Save changes to artifact"
            >
              {isSaving ? (
                <Loader2 className="h-3 w-3 animate-spin" />
              ) : (
                <Save className="h-3 w-3" />
              )}
              <span>Save</span>
            </button>
          )}
        </div>
      </div>

      {/* 2. Main Content Body Area */}
      <div className="flex-1 min-h-0 relative overflow-hidden">
        
        {/* Mode A: VS Code / Monaco Raw Editor */}
        {currentMode === "editor" && (
          <div className="h-full w-full">
            <Editor
              height="100%"
              language={language}
              theme={monacoTheme}
              value={editedValue}
              onChange={handleEditorChange}
              options={{
                readOnly: isReadOnly,
                wordWrap: isWordWrap ? "on" : "off",
                minimap: { enabled: true },
                fontSize: 13,
                lineNumbers: "on",
                scrollBeyondLastLine: false,
                automaticLayout: true,
                tabSize: 2,
                folding: true,
                renderLineHighlight: "all",
                quickSuggestions: true,
              }}
            />
          </div>
        )}

        {/* Mode B: Rendered Markdown / HTML Preview */}
        {currentMode === "preview" && (
          <div className="h-full w-full overflow-hidden">
            {isHtml ? (
              <div className="h-full w-full bg-white flex flex-col">
                <iframe
                  title="HTML Preview"
                  srcDoc={editedValue}
                  sandbox="allow-scripts allow-same-origin allow-forms allow-popups"
                  className="w-full h-full border-0 bg-white"
                />
              </div>
            ) : (
              <div className="h-full w-full overflow-y-auto p-6 md:p-8">
                <div className="max-w-4xl mx-auto prose prose-slate dark:prose-invert prose-sm md:prose-base leading-relaxed break-words">
                  <ReactMarkdown
                    remarkPlugins={[remarkGfm]}
                    components={markdownComponents}
                  >
                    {editedValue}
                  </ReactMarkdown>
                </div>
              </div>
            )}
          </div>
        )}

        {/* Mode C: Side-by-Side Split View */}
        {currentMode === "split" && (
          <div className="flex h-full w-full divide-x divide-border">
            {/* Left Side: Monaco Editor */}
            <div className="w-1/2 h-full">
              <Editor
                height="100%"
                language={language}
                theme={monacoTheme}
                value={editedValue}
                onChange={handleEditorChange}
                options={{
                  readOnly: isReadOnly,
                  wordWrap: isWordWrap ? "on" : "off",
                  minimap: { enabled: false },
                  fontSize: 12,
                  lineNumbers: "on",
                  scrollBeyondLastLine: false,
                  automaticLayout: true,
                  tabSize: 2,
                }}
              />
            </div>

            {/* Right Side: Live Markdown or HTML Preview */}
            <div className="w-1/2 h-full overflow-hidden bg-card/40">
              {isHtml ? (
                <div className="h-full w-full bg-white flex flex-col">
                  <iframe
                    title="HTML Live Preview"
                    srcDoc={editedValue}
                    sandbox="allow-scripts allow-same-origin allow-forms allow-popups"
                    className="w-full h-full border-0 bg-white"
                  />
                </div>
              ) : (
                <div className="h-full w-full overflow-y-auto p-5">
                  <div className="prose prose-slate dark:prose-invert prose-xs leading-normal break-words max-w-none">
                    <ReactMarkdown
                      remarkPlugins={[remarkGfm]}
                      components={markdownComponents}
                    >
                      {editedValue}
                    </ReactMarkdown>
                  </div>
                </div>
              )}
            </div>
          </div>
        )}

        {/* Mode D: Monaco Diff View */}
        {currentMode === "diff" && (
          <div className="h-full w-full">
            <DiffEditor
              height="100%"
              language={language}
              theme={monacoTheme}
              original={originalContent ?? content}
              modified={editedValue}
              options={{
                readOnly: isReadOnly,
                renderSideBySide: true,
                wordWrap: isWordWrap ? "on" : "off",
                automaticLayout: true,
                fontSize: 13,
              }}
            />
          </div>
        )}

      </div>
    </div>
  )
}
export default VSCodeArtifactViewer
