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

import React, { useEffect, useRef, useState, useId } from "react"
import mermaid from "mermaid"
import { AlertCircle, Loader2 } from "lucide-react"

interface MermaidDiagramProps {
  chart: string
  isDark?: boolean
}

function initMermaid(isDark: boolean) {
  try {
    mermaid.initialize({
      startOnLoad: false,
      theme: isDark ? "dark" : "default",
      securityLevel: "loose",
      fontFamily: "var(--font-sans, inherit)",
      darkMode: isDark,
    })
  } catch (err) {
    console.error("Failed to initialize mermaid:", err)
  }
}

export function MermaidDiagram({ chart, isDark = false }: MermaidDiagramProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const [svgContent, setSvgContent] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const rawId = useId()
  const cleanId = `mermaid-${rawId.replace(/:/g, "")}`

  useEffect(() => {
    let isCancelled = false
    initMermaid(isDark)

    const renderChart = async () => {
      if (!chart.trim()) {
        setSvgContent("")
        setIsLoading(false)
        return
      }

      setIsLoading(true)
      setError(null)

      try {
        const id = `${cleanId}-${Date.now()}`
        const { svg } = await mermaid.render(id, chart)
        if (!isCancelled) {
          setSvgContent(svg)
          setError(null)
          setIsLoading(false)
        }
      } catch (err: any) {
        if (!isCancelled) {
          console.warn("Mermaid render error:", err)
          setError(err?.message || "Invalid diagram syntax")
          setIsLoading(false)
        }
      }
    }

    renderChart()

    return () => {
      isCancelled = true
    }
  }, [chart, isDark, cleanId])

  if (isLoading) {
    return (
      <div className="flex items-center justify-center p-6 my-4 rounded-xl border border-border bg-muted/20 text-muted-foreground text-xs gap-2">
        <Loader2 className="h-4 w-4 animate-spin text-primary" />
        <span>Rendering diagram...</span>
      </div>
    )
  }

  if (error) {
    return (
      <div className="my-4 rounded-xl border border-destructive/30 bg-destructive/10 p-4 text-xs font-mono text-destructive">
        <div className="flex items-center gap-1.5 font-sans font-semibold mb-2">
          <AlertCircle className="h-4 w-4 shrink-0" />
          <span>Diagram Render Error</span>
        </div>
        <pre className="overflow-x-auto text-[11px] p-2 bg-background/60 rounded border border-border">
          {chart}
        </pre>
      </div>
    )
  }

  return (
    <div
      ref={containerRef}
      className="mermaid-wrapper my-6 p-4 rounded-xl border border-border bg-card/60 overflow-x-auto flex justify-center shadow-sm"
      dangerouslySetInnerHTML={{ __html: svgContent || "" }}
    />
  )
}
export default MermaidDiagram
