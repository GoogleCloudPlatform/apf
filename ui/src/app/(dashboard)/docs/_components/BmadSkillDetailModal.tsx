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

import React, { useEffect, useState } from "react"
import { BmadSkill } from "./data/bmad-catalog"
import {
  Sparkles,
  Terminal,
  ArrowRight,
  FolderOutput,
  Workflow,
  Copy,
  Check,
  Tag,
  Layers,
  X,
  Box
} from "lucide-react"

interface BmadSkillDetailModalProps {
  skill: BmadSkill | null
  isOpen: boolean
  onClose: () => void
}

export function BmadSkillDetailModal({ skill, isOpen, onClose }: BmadSkillDetailModalProps) {
  const [copiedExample, setCopiedExample] = useState<string | null>(null)

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape" && isOpen) {
        onClose()
      }
    }
    window.addEventListener("keydown", handleKeyDown)
    return () => window.removeEventListener("keydown", handleKeyDown)
  }, [isOpen, onClose])

  if (!isOpen || !skill) return null

  const handleCopy = (text: string) => {
    navigator.clipboard.writeText(text)
    setCopiedExample(text)
    setTimeout(() => setCopiedExample(null), 2000)
  }

  const getPhaseBadgeColor = (phase: string) => {
    switch (phase) {
      case "1-ideation":
        return "bg-amber-500/10 text-amber-600 dark:text-amber-400 border-amber-500/20"
      case "2-planning":
        return "bg-blue-500/10 text-blue-600 dark:text-blue-400 border-blue-500/20"
      case "3-solutioning":
        return "bg-purple-500/10 text-purple-600 dark:text-purple-400 border-purple-500/20"
      case "4-implementation":
        return "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border-emerald-500/20"
      case "0-learning":
        return "bg-indigo-500/10 text-indigo-600 dark:text-indigo-400 border-indigo-500/20"
      default:
        return "bg-slate-500/10 text-slate-600 dark:text-slate-400 border-slate-500/20"
    }
  }

  return (
    <div
      role="dialog"
      aria-modal="true"
      className="fixed inset-0 z-[100] flex items-center justify-center bg-black/60 backdrop-blur-sm p-4 select-none"
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose()
      }}
    >
      <div className="bg-card border border-border rounded-2xl max-w-2xl w-full p-6 shadow-2xl space-y-6 animate-in fade-in zoom-in-95 duration-200 relative max-h-[85vh] overflow-y-auto">
        {/* Close Button */}
        <button
          type="button"
          onClick={onClose}
          className="absolute right-4 top-4 p-2 rounded-lg text-muted-foreground hover:bg-muted hover:text-foreground transition-colors"
          title="Close dialog"
        >
          <X className="h-4 w-4" />
        </button>

        {/* Header */}
        <div className="space-y-3 pb-3 border-b border-border text-left pr-8">
          <div className="flex flex-wrap items-center gap-2">
            <span className={`inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-semibold border ${getPhaseBadgeColor(skill.phase)}`}>
              <Layers className="h-3 w-3" />
              {skill.phaseLabel}
            </span>
            <span className="inline-flex items-center px-2 py-0.5 rounded-md bg-muted text-muted-foreground text-xs font-medium border border-border">
              {skill.module}
            </span>
            <span className="inline-flex items-center px-2 py-0.5 rounded-md bg-primary/10 text-primary text-xs font-bold font-mono">
              Code: {skill.code}
            </span>
            {skill.required && (
              <span className="inline-flex items-center px-2 py-0.5 rounded-md bg-destructive/10 text-destructive text-[11px] font-semibold">
                Required Core Step
              </span>
            )}
            {skill.requires_sandbox && (
              <span
                title="Sandbox execution required"
                className="inline-flex items-center gap-1 px-2 py-0.5 rounded-md bg-amber-500/10 text-amber-700 dark:text-amber-300 text-xs font-medium border border-amber-500/20"
                data-testid="bmad-modal-sandbox-badge"
              >
                <Box className="h-3.5 w-3.5" />
                <span>Sandbox</span>
              </span>
            )}
            {skill.apf_supported === false && !skill.requires_sandbox && (
              <span className="inline-flex items-center px-2 py-0.5 rounded-md bg-amber-500/10 text-amber-700 dark:text-amber-300 text-xs font-medium border border-amber-500/20">
                IDE Based
              </span>
            )}
          </div>

          <div className="space-y-1">
            <h3 className="text-xl font-bold text-foreground flex items-center gap-2">
              <span>{skill.name}</span>
              <code className="text-xs font-normal text-muted-foreground font-mono bg-muted/60 px-2 py-0.5 rounded">
                /{skill.id}
              </code>
            </h3>

            <p className="text-xs text-muted-foreground leading-relaxed">
              {skill.description}
            </p>
          </div>
        </div>

        {/* Workflow Sequence Position */}
        {(skill.precededBy || skill.followedBy) && (
          <div className="space-y-2 p-3.5 rounded-xl bg-slate-50 dark:bg-slate-900/60 border border-border">
            <h4 className="text-xs font-bold text-foreground flex items-center gap-2">
              <Workflow className="h-3.5 w-3.5 text-primary" />
              Workflow Pipeline Position
            </h4>
            <div className="flex flex-col sm:flex-row items-start sm:items-center gap-2 text-xs text-muted-foreground">
              {skill.precededBy && (
                <div className="flex items-center gap-1.5">
                  <span className="text-[11px] font-medium text-muted-foreground">After:</span>
                  <code className="font-mono bg-muted px-1.5 py-0.5 rounded text-[11px] text-foreground">
                    {skill.precededBy}
                  </code>
                </div>
              )}
              {skill.precededBy && skill.followedBy && (
                <ArrowRight className="h-3 w-3 text-muted-foreground hidden sm:inline shrink-0" />
              )}
              {skill.followedBy && (
                <div className="flex items-center gap-1.5">
                  <span className="text-[11px] font-medium text-muted-foreground">Next:</span>
                  <code className="font-mono bg-muted px-1.5 py-0.5 rounded text-[11px] text-foreground">
                    {skill.followedBy}
                  </code>
                </div>
              )}
            </div>
          </div>
        )}

        {/* Output Artifacts Specifications */}
        {(skill.outputLocation || skill.outputs) && (
          <div className="space-y-2 p-3.5 rounded-xl bg-slate-50 dark:bg-slate-900/60 border border-border">
            <h4 className="text-xs font-bold text-foreground flex items-center gap-2">
              <FolderOutput className="h-3.5 w-3.5 text-primary" />
              Output Deliverables & Contracts
            </h4>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 text-xs">
              {skill.outputLocation && (
                <div>
                  <span className="text-[10px] uppercase font-semibold text-muted-foreground block mb-0.5">
                    Storage Location
                  </span>
                  <code className="font-mono text-foreground bg-muted/80 px-1.5 py-0.5 rounded text-[11px]">
                    {skill.outputLocation}
                  </code>
                </div>
              )}
              {skill.outputs && (
                <div>
                  <span className="text-[10px] uppercase font-semibold text-muted-foreground block mb-0.5">
                    Artifact Output
                  </span>
                  <span className="text-xs text-foreground font-medium">
                    {skill.outputs}
                  </span>
                </div>
              )}
            </div>
          </div>
        )}

        {/* CLI Arguments / Parameters */}
        {skill.args && (
          <div className="space-y-2 p-3.5 rounded-xl bg-slate-50 dark:bg-slate-900/60 border border-border">
            <h4 className="text-xs font-bold text-foreground flex items-center gap-2">
              <Terminal className="h-3.5 w-3.5 text-primary" />
              Arguments & Parameters
            </h4>
            <code className="block p-2.5 rounded-lg bg-slate-950 text-slate-200 text-xs font-mono overflow-x-auto">
              {skill.args}
            </code>
          </div>
        )}

        {/* Example Invocations */}
        {skill.examples && skill.examples.length > 0 && (
          <div className="space-y-2">
            <h4 className="text-xs font-bold text-foreground flex items-center gap-2">
              <Sparkles className="h-3.5 w-3.5 text-primary" />
              Example Usages & Prompts
            </h4>
            <div className="space-y-2">
              {skill.examples.map((example, idx) => (
                <div
                  key={idx}
                  className="flex items-center justify-between p-2.5 rounded-lg bg-muted/60 border border-border group hover:bg-muted transition-colors"
                >
                  <code className="text-xs font-mono text-foreground truncate mr-2">
                    {example}
                  </code>
                  <button
                    type="button"
                    onClick={() => handleCopy(example)}
                    className="p-1.5 rounded-md hover:bg-background text-muted-foreground hover:text-foreground transition-colors shrink-0"
                    title="Copy command"
                  >
                    {copiedExample === example ? (
                      <Check className="h-3.5 w-3.5 text-emerald-500" />
                    ) : (
                      <Copy className="h-3.5 w-3.5" />
                    )}
                  </button>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Tags */}
        {skill.tags && skill.tags.length > 0 && (
          <div className="flex flex-wrap items-center gap-1.5 pt-2 border-t border-border">
            <Tag className="h-3 w-3 text-muted-foreground mr-1" />
            {skill.tags.map((t) => (
              <span key={t} className="text-[10px] px-2 py-0.5 rounded-md bg-muted text-muted-foreground font-mono">
                #{t}
              </span>
            ))}
          </div>
        )}

        {/* Footer actions */}
        <div className="flex justify-end pt-2">
          <button
            type="button"
            onClick={onClose}
            className="inline-flex h-9 items-center justify-center rounded-xl bg-primary px-4 text-xs font-semibold text-primary-foreground shadow hover:opacity-90 transition-colors"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  )
}
export default BmadSkillDetailModal
