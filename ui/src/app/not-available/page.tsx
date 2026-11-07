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

import { useState } from "react"
import { useHealthStore } from "@/stores/health"
import { RefreshCw, CloudOff, AlertCircle } from "lucide-react"

/**
 * AppNotAvailableScreen Next.js port.
 * Renders state-driven diagnostic indicators when backend components drop offline.
 */
export default function NotAvailablePage() {
  const { isAvailable, isApiUp, isAiUp, errorMessage, checkHealth } = useHealthStore()
  const [isRetrying, setIsRetrying] = useState(false)

  const handleRetry = async () => {
    if (isRetrying) return
    setIsRetrying(true)
    await checkHealth()
    setIsRetrying(false)
  }

  return (
    <main className="flex min-h-screen items-center justify-center bg-slate-50 dark:bg-slate-950 p-6">
      <div className="w-full max-w-[460px] rounded-xl border border-border bg-card p-8 shadow-xl text-card-foreground">
        <div className="flex flex-col items-center text-center">
          
          {/* Pulse Animated Warning Icon */}
          <div className="relative flex h-20 w-20 items-center justify-center rounded-full border border-destructive/30 bg-destructive/10 text-destructive animate-pulse mb-6">
            <CloudOff className="h-10 w-10" />
          </div>

          {/* Title */}
          <h1 className="text-2xl font-bold tracking-tight text-foreground mb-2">
            Application Not Available
          </h1>

          {/* Friendly Status Alert Message */}
          <p className="text-sm text-muted-foreground leading-relaxed mb-6">
            {errorMessage ||
              "The required backend services are currently unreachable. Please check your system connection and run checks again."}
          </p>

          {/* Diagnostic status indicators box */}
          <div className="w-full rounded-lg border border-border bg-muted/50 p-5 text-left mb-8">
            <h2 className="text-xs font-bold tracking-wider text-muted-foreground uppercase mb-4 flex items-center gap-1.5">
              <AlertCircle className="h-3.5 w-3.5" />
              Connection Diagnostics
            </h2>

            <div className="space-y-4">
              {/* API status row */}
              <div className="flex items-center justify-between">
                <div>
                  <h3 className="text-sm font-semibold text-foreground">API Backend</h3>
                  <p className="text-[11px] text-muted-foreground">
                    {isApiUp ? "Operational and reachable" : "Offline or unreachable"}
                  </p>
                </div>
                <div
                  className={`flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-[10px] font-bold ${
                    isApiUp
                      ? "bg-green-500/10 border-green-500/30 text-green-600 dark:text-green-400"
                      : "bg-destructive/10 border-destructive/30 text-destructive"
                  }`}
                >
                  <span
                    className={`h-1.5 w-1.5 rounded-full ${
                      isApiUp ? "bg-green-500 shadow-[0_0_6px_rgba(34,197,94,0.6)]" : "bg-destructive"
                    }`}
                  />
                  {isApiUp ? "ONLINE" : "OFFLINE"}
                </div>
              </div>

              <div className="h-px bg-border" />

              {/* AI engine status row */}
              <div className="flex items-center justify-between">
                <div>
                  <h3 className="text-sm font-semibold text-foreground">AI Agent Service</h3>
                  <p className="text-[11px] text-muted-foreground">
                    {isAiUp ? "Operational and reachable" : "Offline or unreachable"}
                  </p>
                </div>
                <div
                  className={`flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-[10px] font-bold ${
                    isAiUp
                      ? "bg-green-500/10 border-green-500/30 text-green-600 dark:text-green-400"
                      : "bg-destructive/10 border-destructive/30 text-destructive"
                  }`}
                >
                  <span
                    className={`h-1.5 w-1.5 rounded-full ${
                      isAiUp ? "bg-green-500 shadow-[0_0_6px_rgba(34,197,94,0.6)]" : "bg-destructive"
                    }`}
                  />
                  {isAiUp ? "ONLINE" : "OFFLINE"}
                </div>
              </div>
            </div>
          </div>

          {/* Retry trigger button */}
          <button
            onClick={handleRetry}
            disabled={isRetrying}
            className="flex w-full h-11 items-center justify-center gap-2 rounded-md bg-primary text-primary-foreground text-sm font-bold shadow-md transition-all hover:opacity-90 disabled:opacity-50"
          >
            {isRetrying ? (
              <RefreshCw className="h-4 w-4 animate-spin" />
            ) : (
              <RefreshCw className="h-4 w-4" />
            )}
            {isRetrying ? "RUNNING CHECKS..." : "RETRY CONNECTION"}
          </button>

          <p className="text-[11px] text-muted-foreground/60 mt-4">
            Please contact the foundry engineering team if the problem persists.
          </p>
        </div>
      </div>
    </main>
  )
}
