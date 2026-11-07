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

import React from "react"
import { ArrowRightCircle, Sparkles, Compass } from "lucide-react"
import { cn } from "@/lib/utils"
import { phaseConfig, PhaseType } from "@/app/(dashboard)/projects/[id]/workspace/_utils/workspaceHelpers"

export interface PhaseSwitchCardProps {
  targetPhase: string
  targetPhaseLabel?: string
  skillId?: string
  reason?: string
  suggestedPrompt?: string
  onConfirmSwitch: (targetPhase: string, promptToExecute?: string) => void
}

export default function PhaseSwitchCard({
  targetPhase,
  targetPhaseLabel,
  skillId,
  reason,
  suggestedPrompt,
  onConfirmSwitch,
}: PhaseSwitchCardProps) {
  const normalizedPhase = targetPhase.toUpperCase() as PhaseType
  const phaseInfo = phaseConfig[normalizedPhase] || {
    label: targetPhaseLabel || targetPhase,
    icon: Compass,
  }
  const PhaseIcon = phaseInfo.icon || Compass

  const label = phaseInfo.label || targetPhase

  return (
    <div className="my-2 rounded-xl border border-primary/30 bg-primary/5 p-4 shadow-sm transition-all hover:border-primary/50">
      <div className="flex items-start gap-3">
        <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
          <PhaseIcon className="h-5 w-5" />
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <span className="inline-flex items-center gap-1 rounded-md bg-primary/15 px-2 py-0.5 text-xs font-bold text-primary">
              <Sparkles className="h-3 w-3" />
              Recommended Phase Switch: {label}
            </span>
            {skillId && (
              <span className="text-xs font-medium text-muted-foreground">
                Skill: {skillId}
              </span>
            )}
          </div>

          {reason && (
            <p className="mt-1.5 text-xs leading-relaxed text-foreground/90">
              {reason}
            </p>
          )}

          <div className="mt-3 flex items-center gap-2">
            <button
              type="button"
              onClick={() => onConfirmSwitch(targetPhase, suggestedPrompt || (skillId ? `Run ${skillId}` : undefined))}
              className="inline-flex items-center gap-1.5 rounded-lg bg-primary px-3.5 py-1.5 text-xs font-semibold text-primary-foreground shadow transition-all hover:bg-primary/90 active:scale-95"
            >
              <span>Switch to {label} Tab & Continue</span>
              <ArrowRightCircle className="h-4 w-4" />
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
