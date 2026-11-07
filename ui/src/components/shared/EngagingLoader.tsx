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

import { useEffect, useState, useRef } from "react"
import { AlertCircle, Loader2 } from "lucide-react"

interface EngagingLoaderProps {
  hasError?: boolean
  errorMessage?: string | null
  initialMessage?: string
}

// Phrase list matching widgets/engaging_loader.dart
const loaderMessages = [
  "Crunching data streams...",
  "Optimizing neural pathways...",
  "Syncing with the grid...",
  "Retrieving quantum states...",
  "Calibrating sensors...",
  "Brewing digital coffee...",
  "Defragmenting reality...",
  "Loading awesomeness...",
  "Forging agentic workflows...",
  "Casting neural models...",
  "Smelting data cores...",
  "Tempering prompt structures...",
  "Molding cognitive architectures...",
  "Assembling carbon units...",
  "Warming up the GPUs...",
  "Solving P vs NP...",
  "Reticulating splines...",
  "Bending spacetime...",
  "Mining for insights...",
  "Distilling knowledge...",
  "Synthesizing responses...",
  "Refactoring the matrix...",
  "Calculating state deltas...",
  "Applying differential updates...",
  "Mapping the delta stream...",
  "Resolving change vectors...",
  "Merging delta branches...",
]

// Light pastel colors matching the original palette
const pastelColors = [
  "text-pink-400 dark:text-pink-300",
  "text-blue-400 dark:text-blue-300",
  "text-green-400 dark:text-green-300",
  "text-amber-400 dark:text-amber-300",
  "text-purple-400 dark:text-purple-300",
  "text-orange-400 dark:text-orange-300",
]

export function EngagingLoader({
  hasError = false,
  errorMessage,
  initialMessage,
}: EngagingLoaderProps) {
  const [currentText, setCurrentText] = useState("")
  const [colorClass, setColorClass] = useState(pastelColors[0])
  const timerRef = useRef<NodeJS.Timeout | null>(null)

  const loadRandomState = () => {
    const randomIdx = Math.floor(Math.random() * loaderMessages.length)
    const randomColorIdx = Math.floor(Math.random() * pastelColors.length)
    
    setCurrentText(loaderMessages[randomIdx])
    setColorClass(pastelColors[randomColorIdx])
  }

  const startCycling = () => {
    // Random duration between 4 and 8 seconds
    const delay = 4000 + Math.floor(Math.random() * 4000)
    timerRef.current = setTimeout(() => {
      loadRandomState()
      startCycling()
    }, delay)
  }

  // Handle cycle logic on mount and when states mutate
  useEffect(() => {
    if (hasError) {
      if (timerRef.current) clearTimeout(timerRef.current)
      setCurrentText(errorMessage || "Oops! there was some error please try again.")
    } else if (initialMessage) {
      setCurrentText(initialMessage)
      const randomColorIdx = Math.floor(Math.random() * pastelColors.length)
      setColorClass(pastelColors[randomColorIdx])
      startCycling()
    } else {
      loadRandomState()
      startCycling()
    }

    return () => {
      if (timerRef.current) clearTimeout(timerRef.current)
    }
  }, [hasError, errorMessage, initialMessage])

  return (
    <div className="flex items-center gap-3.5 p-4 rounded-xl border border-border/40 bg-card/60 w-fit max-w-md animate-fade-in">
      {hasError ? (
        <AlertCircle className="h-5 w-5 shrink-0 text-destructive" />
      ) : (
        <Loader2 className={`h-5 w-5 shrink-0 animate-spin ${colorClass}`} />
      )}
      
      <span className="text-xs font-medium text-foreground leading-normal break-words">
        {currentText}
      </span>
    </div>
  )
}
export default EngagingLoader
