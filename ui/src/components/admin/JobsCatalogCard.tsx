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

import { Terminal, Play, ShieldCheck, Wrench } from "lucide-react"
import { JobCatalogItem } from "./JobRunModal"

interface JobsCatalogCardProps {
  jobs: JobCatalogItem[]
  onSelectJob: (job: JobCatalogItem) => void
  isLoading: boolean
}

export default function JobsCatalogCard({
  jobs,
  onSelectJob,
  isLoading,
}: JobsCatalogCardProps) {
  return (
    <div className="rounded-2xl border border-border bg-card p-6 shadow-sm space-y-5 text-foreground">
      <div className="flex items-center justify-between">
        <div className="space-y-1">
          <h3 className="text-base font-bold text-foreground flex items-center gap-2">
            <Terminal className="h-5 w-5 text-primary" />
            Registered System Jobs
          </h3>
          <p className="text-xs text-muted-foreground">
            Available batch operations registered in the APF container runtime.
          </p>
        </div>
        <span className="text-xs font-semibold px-2.5 py-1 rounded-full bg-primary/10 text-primary border border-primary/20">
          {jobs.length} Available
        </span>
      </div>

      {isLoading ? (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {[1, 2, 3].map((i) => (
            <div
              key={i}
              className="h-36 rounded-xl border border-border/50 bg-muted/30 animate-pulse"
            />
          ))}
        </div>
      ) : jobs.length === 0 ? (
        <div className="p-8 text-center rounded-xl border border-dashed border-border bg-muted/20">
          <Wrench className="h-8 w-8 text-muted-foreground mx-auto mb-2 opacity-50" />
          <p className="text-sm font-semibold text-foreground">No Registered Jobs Found</p>
          <p className="text-xs text-muted-foreground mt-1">
            Ensure system housekeeping jobs are compiled into the binary.
          </p>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {jobs.map((job) => {
            const hasDryRun = job.parameters?.some((p) => p.name === "dryRun") ?? true

            return (
              <div
                key={job.name}
                className="flex flex-col justify-between p-4 rounded-xl border border-border bg-card hover:border-primary/40 hover:shadow-md transition-all space-y-4"
              >
                <div className="space-y-2">
                  <div className="flex items-center justify-between gap-2">
                    <span className="font-mono text-xs font-bold text-primary bg-primary/10 px-2 py-0.5 rounded border border-primary/20 truncate">
                      {job.name}
                    </span>
                    {hasDryRun && (
                      <span className="inline-flex items-center gap-1 text-[10px] font-medium text-emerald-600 dark:text-emerald-400 bg-emerald-500/10 px-2 py-0.5 rounded-full border border-emerald-500/20">
                        <ShieldCheck className="h-3 w-3" />
                        Dry Run
                      </span>
                    )}
                  </div>
                  <p className="text-xs text-muted-foreground line-clamp-3 leading-relaxed">
                    {job.description || "No description provided for this job."}
                  </p>
                </div>

                <div className="pt-2 border-t border-border/50 flex justify-end">
                  <button
                    onClick={() => onSelectJob(job)}
                    className="w-full inline-flex items-center justify-center gap-1.5 px-3 py-1.5 rounded-lg bg-primary hover:bg-primary/90 text-primary-foreground text-xs font-semibold shadow-sm transition-colors"
                  >
                    <Play className="h-3.5 w-3.5" />
                    Configure & Run
                  </button>
                </div>
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}
