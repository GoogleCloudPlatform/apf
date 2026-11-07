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

import { useState, useEffect, useRef, useMemo } from "react"
import { ChevronDown, Brain, CheckCircle2 } from "lucide-react"
import ReactMarkdown from "react-markdown"
import remarkGfm from "remark-gfm"
import { cn } from "@/lib/utils"
import { Message } from "@/lib/types"

export interface ThinkingStep {
  id: string
  summary: string
  content: string
  status: "streaming" | "completed"
  durationSec?: number
  isFileReader?: boolean
}

interface ThinkingBlockProps {
  thinkingSteps?: Message[]
  thinkingText?: string
  isStreaming: boolean
  durationSec?: number
  onScrollBottom?: () => void
}

export function formatToolTitle(toolName: string): string {
  if (!toolName) return "Tool result"
  const stripped = toolName.replace(/^[`'"]+|[`'"]+$/g, "").trim()
  if (!stripped) return "Tool result"
  const withSpaces = stripped
    .replace(/([a-z0-9])([A-Z])/g, "$1 $2")
    .replace(/[_-]+/g, " ")
    .trim()
  if (!withSpaces) return "Tool result"
  return withSpaces.charAt(0).toUpperCase() + withSpaces.slice(1)
}

export function getStepSummary(step: Message | string): string {
  if (typeof step !== "string") {
    const params = step.notificationDetails?.params
    if (params) {
      const isFileReader =
        params.is_file_reader === true ||
        (typeof params.tool_name === "string" &&
          /read_file|view_file|file_reader|^read$/i.test(params.tool_name))
      if (isFileReader) {
        const fileName = params.file_name || params.path || params.filePath
        if (fileName && typeof fileName === "string") {
          return `Read file: ${fileName}`
        }
        return "Read file content"
      }
      if (typeof params.tool_name === "string" && params.tool_name.trim()) {
        return formatToolTitle(params.tool_name)
      }
    }
  }

  const text = (typeof step === "string" ? step : step.text) || ""
  const fileReadMatch = text.match(/^(?:\*\*)?Read file:\s*`?([^`\n*]+)`?(?:\*\*)?/i)
  if (fileReadMatch && fileReadMatch[1].trim()) {
    return `Read file: ${fileReadMatch[1].trim()}`
  }

  const toolResultMatch = text.match(
    /^(?:\*\*)?Tool result\s*(?:\(\s*[`'"]?([^`'")]*)[`'"]?\s*\)|:\s*[`'"]?([^`'"\n]+)[`'"]?)(?:.*)?/i
  )
  if (toolResultMatch) {
    const rawName = (toolResultMatch[1] || toolResultMatch[2] || "").trim()
    if (rawName) {
      return formatToolTitle(rawName)
    }
  }

  const firstLine = text.split("\n")[0].trim()
  const cleaned = firstLine.replace(/^[#*-\s]+/, "").replace(/\*\*$/, "")

  const cleanedToolMatch = cleaned.match(
    /^Tool result\s*(?:\(\s*[`'"]?([^`'")]*)[`'"]?\s*\)|:\s*[`'"]?([^`'"\n]+)[`'"]?)(?:.*)?/i
  )
  if (cleanedToolMatch) {
    const rawName = (cleanedToolMatch[1] || cleanedToolMatch[2] || "").trim()
    if (rawName) {
      return formatToolTitle(rawName)
    }
  }

  if (cleaned.length > 70) {
    return cleaned.substring(0, 67) + "..."
  }
  return cleaned || "Completed thought step"
}

function getStepContent(step: Message | string): string {
  if (typeof step !== "string") {
    const params = step.notificationDetails?.params
    if (params) {
      const isFileReader =
        params.is_file_reader === true ||
        (typeof params.tool_name === "string" &&
          /read_file|view_file|file_reader|^read$/i.test(params.tool_name))
      if (isFileReader && params.content && typeof params.content === "string") {
        const content = params.content.trim()
        const fileName = params.file_name || params.path || params.filePath || ""
        const isMarkdown = /\.mdx?$/i.test(fileName)

        if (isMarkdown) {
          return content
        }

        if (!content.startsWith("```")) {
          return `\`\`\`\n${content}\n\`\`\``
        }
        return content
      }
    }
  }

  const text = (typeof step === "string" ? step : step.text) || ""
  const fileReadHeaderMatch = text.match(/^(?:\*\*)?Read file:\s*`?([^`\n*]+)`?(?:\*\*)?\r?\n([\s\S]*)$/i)
  if (fileReadHeaderMatch && fileReadHeaderMatch[2]) {
    const fileName = fileReadHeaderMatch[1].trim()
    const innerContent = fileReadHeaderMatch[2].trim()
    if (/\.mdx?$/i.test(fileName)) {
      const codeBlockMatch = innerContent.match(/^```[a-z]*\r?\n([\s\S]*?)\r?\n```$/i)
      return codeBlockMatch ? codeBlockMatch[1] : innerContent
    }
    return innerContent
  }

  return text
}

// ------------------------------------------------------------------
// Fast Left-to-Right Line-by-Line File Content Typewriter Animation
// ------------------------------------------------------------------
function AnimatedFileContent({
  content,
  isMarkdown,
  onScrollBottom,
}: {
  content: string
  isMarkdown: boolean
  onScrollBottom?: () => void
}) {
  const { lang, lines } = useMemo(() => {
    if (!isMarkdown) {
      const match = content.match(/^```([a-z0-9_-]*)\r?\n([\s\S]*?)\r?\n```$/i)
      if (match) {
        return { lang: match[1], lines: match[2].split("\n") }
      }
    }
    return { lang: "", lines: content.split("\n") }
  }, [content, isMarkdown])

  const [lineIndex, setLineIndex] = useState(0)
  const [charIndex, setCharIndex] = useState(0)
  const [isCompleted, setIsCompleted] = useState(false)
  const scrollRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    setLineIndex(0)
    setCharIndex(0)
    setIsCompleted(false)
  }, [content])

  useEffect(() => {
    if (isCompleted || lineIndex >= lines.length) {
      setIsCompleted(true)
      return
    }

    const timer = setTimeout(() => {
      const currentLine = lines[lineIndex] || ""
      const stepChars = Math.max(56, Math.ceil(currentLine.length * 2))

      if (charIndex + stepChars < currentLine.length) {
        setCharIndex((c) => c + stepChars)
      } else {
        setLineIndex((l) => l + 1)
        setCharIndex(0)
      }
    }, 24)

    return () => clearTimeout(timer)
  }, [lineIndex, charIndex, isCompleted, lines])

  useEffect(() => {
    if (!isCompleted) {
      if (scrollRef.current) {
        scrollRef.current.scrollTop = scrollRef.current.scrollHeight
      }
      onScrollBottom?.()
    }
  }, [lineIndex, charIndex, isCompleted, onScrollBottom])

  const printedBody = useMemo(() => {
    if (isCompleted) {
      return lines.join("\n")
    }
    const completedLines = lines.slice(0, lineIndex).join("\n")
    const activeLine =
      lineIndex < lines.length
        ? (lineIndex > 0 ? "\n" : "") + lines[lineIndex].substring(0, charIndex) + " ▋"
        : ""
    return completedLines + activeLine
  }, [lines, lineIndex, charIndex, isCompleted])

  const markdownInput = useMemo(() => {
    if (isMarkdown) return printedBody
    return `\`\`\`${lang}\n${printedBody}\n\`\`\``
  }, [isMarkdown, lang, printedBody])

  return (
    <div
      ref={scrollRef}
      className={cn(
        "prose prose-slate dark:prose-invert prose-xs max-w-none break-words text-[11px] leading-relaxed text-gray-600 dark:text-gray-300 max-h-64 overflow-y-auto",
        !isMarkdown && "font-mono"
      )}
    >
      <ReactMarkdown remarkPlugins={[remarkGfm]}>
        {markdownInput}
      </ReactMarkdown>
    </div>
  )
}

// ------------------------------------------------------------------
// Parent Timeline Container (Sequential Multi-Step Accordions)
// ------------------------------------------------------------------
export function ThinkingBlock({
  thinkingSteps,
  thinkingText,
  isStreaming,
  durationSec,
  onScrollBottom,
}: ThinkingBlockProps) {
  // Convert Message[] or raw thinkingText to normalized ThinkingStep[]
  const steps: ThinkingStep[] = (() => {
    if (thinkingSteps && thinkingSteps.length > 0) {
      return thinkingSteps.map((m, idx) => {
        const isLast = idx === thinkingSteps.length - 1
        const summary = getStepSummary(m)
        const isFileReader =
          Boolean(m.notificationDetails?.params?.is_file_reader) ||
          /^(?:\*\*)?Read file:/i.test(m.text || "") ||
          /^Read file:/i.test(summary)
        return {
          id: m.id || `step-${idx}`,
          summary,
          content: getStepContent(m),
          status: isStreaming && isLast ? "streaming" : "completed",
          durationSec: m.thinkingDurationSec,
          isFileReader,
        }
      })
    }
    if (thinkingText) {
      const summary = getStepSummary(thinkingText)
      const isFileReader = /^Read file:/i.test(summary)
      return [
        {
          id: "step-single",
          summary,
          content: getStepContent(thinkingText),
          status: isStreaming ? "streaming" : "completed",
          durationSec: durationSec,
          isFileReader,
        },
      ]
    }
    return []
  })()

  if (steps.length === 0) return null

  const activeIndex = steps.length - 1

  return (
    <div className="my-2.5 space-y-1.5 border-l-2 border-gray-200 dark:border-gray-800 pl-3 w-full">
      {steps.map((step, idx) => {
        const isLatest = idx === activeIndex
        return (
          <ThinkingStepItem
            key={step.id}
            step={step}
            // Auto-expand only if it's the latest step AND actively streaming
            defaultOpen={isLatest && step.status === "streaming"}
            onScrollBottom={onScrollBottom}
          />
        )
      })}
    </div>
  )
}

// ------------------------------------------------------------------
// Individual Step Accordion Item
// ------------------------------------------------------------------
function ThinkingStepItem({
  step,
  defaultOpen,
  onScrollBottom,
}: {
  step: ThinkingStep
  defaultOpen: boolean
  onScrollBottom?: () => void
}) {
  const [isOpen, setIsOpen] = useState(defaultOpen)
  const [userToggled, setUserToggled] = useState(false)
  const scrollRef = useRef<HTMLDivElement>(null)

  // Automatically collapse when defaultOpen transitions from true -> false
  useEffect(() => {
    if (!userToggled) {
      setIsOpen(defaultOpen)
    }
  }, [defaultOpen, userToggled])

  // Automatically scroll parent container when accordion opens or toggles
  useEffect(() => {
    if (isOpen) {
      const timer = setTimeout(() => {
        onScrollBottom?.()
      }, 30)
      return () => clearTimeout(timer)
    }
  }, [isOpen, onScrollBottom])

  const isStreaming = step.status === "streaming"

  // Live Auto-Scroll to bottom while actively streaming
  useEffect(() => {
    if (isStreaming && isOpen) {
      if (scrollRef.current) {
        scrollRef.current.scrollTop = scrollRef.current.scrollHeight
      }
      onScrollBottom?.()
    }
  }, [step.content, isStreaming, isOpen, onScrollBottom])

  return (
    <div className="rounded-md border border-gray-200/80 dark:border-gray-800/80 bg-gray-50/40 dark:bg-gray-900/30 text-xs transition-colors overflow-hidden">
      {/* Step Header */}
      <button
        type="button"
        onClick={() => {
          setIsOpen(!isOpen)
          setUserToggled(true)
        }}
        className="flex w-full items-center justify-between px-3 py-2 text-left font-medium text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-100 select-none transition-colors"
      >
        <div className="flex items-center gap-2 overflow-hidden pr-2">
          {isStreaming ? (
            <Brain className="h-3.5 w-3.5 shrink-0 animate-pulse text-amber-500" />
          ) : (
            <CheckCircle2 className="h-3.5 w-3.5 shrink-0 text-emerald-500/80 dark:text-emerald-400/80" />
          )}
          <span className="truncate">
            {step.summary || (isStreaming ? "Thinking..." : "Completed thought step")}
          </span>
        </div>

        <div className="flex items-center gap-2 shrink-0 text-[11px] text-gray-400">
          {step.durationSec && <span>{step.durationSec}s</span>}
          <ChevronDown
            className={cn(
              "h-3.5 w-3.5 transition-transform duration-200 shrink-0",
              isOpen ? "rotate-180" : ""
            )}
          />
        </div>
      </button>

      {/* Step Content */}
      {isOpen && (
        <div className="border-t border-gray-200/60 dark:border-gray-800/60 px-3 py-2.5 animate-fade-in">
          {step.isFileReader && step.status === "streaming" ? (
            <AnimatedFileContent
              content={step.content}
              isMarkdown={/\.mdx?$/i.test(step.summary)}
              onScrollBottom={onScrollBottom}
            />
          ) : (
            <div
              ref={scrollRef}
              className={cn(
                "prose prose-slate dark:prose-invert prose-xs max-w-none break-words text-[11px] leading-relaxed text-gray-600 dark:text-gray-300 max-h-64 overflow-y-auto",
                !/\.mdx?$/i.test(step.summary) && "font-mono"
              )}
            >
              <ReactMarkdown remarkPlugins={[remarkGfm]}>
                {step.content}
              </ReactMarkdown>
            </div>
          )}
        </div>
      )}
    </div>
  )
}

export default ThinkingBlock
