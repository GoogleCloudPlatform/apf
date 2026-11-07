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

import React from "react"
import { cn } from "@/lib/utils"

interface ProjectIdentityProps {
  icon: string
  name: string
  id?: string
  isNew?: boolean
  onIconClick?: () => void
  size?: "sm" | "lg"
  className?: string
}

export default function ProjectIdentity({
  icon,
  name,
  id,
  isNew = false,
  onIconClick,
  size = "sm",
  className,
}: ProjectIdentityProps) {
  const isClickable = !!onIconClick

  const boxSizes = {
    sm: "h-9 w-9 text-xl",
    lg: "h-12 w-12 text-2xl",
  }

  const nameSizes = {
    sm: "text-sm",
    lg: "text-xl",
  }

  const idSizes = {
    sm: "text-[9px]",
    lg: "text-xs",
  }

  const gapSizes = {
    sm: "gap-2.5",
    lg: "gap-3",
  }

  return (
    <div className={cn("flex items-center min-w-0 shrink-0", gapSizes[size], className)}>
      <div
        onClick={onIconClick}
        className={cn(
          "flex items-center justify-center rounded-lg bg-primary/10 text-primary font-bold select-none",
          boxSizes[size],
          isClickable && "cursor-pointer hover:bg-primary/20 transition-colors"
        )}
        title={isClickable ? "Click to choose icon" : undefined}
      >
        {icon || "🚀"}
      </div>
      <div className="min-w-0 flex flex-col justify-center gap-1 whitespace-nowrap">
        <h1 className={cn("font-bold text-foreground truncate leading-tight", nameSizes[size])}>
          {isNew ? "New Project" : name}
        </h1>
        {!isNew && id && (
          <span className={cn("font-mono text-muted-foreground leading-none", idSizes[size])}>
            ID: {id}
          </span>
        )}
      </div>
    </div>
  )
}
