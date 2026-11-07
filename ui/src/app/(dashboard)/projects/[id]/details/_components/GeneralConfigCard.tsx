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
import { Settings2, ChevronDown } from "lucide-react"

interface GeneralConfigCardProps {
  name: string
  setName: (n: string) => void
  description: string
  setDescription: (d: string) => void
  icon: string
  onChooseIcon: () => void
  visibility: string
  setVisibility: (v: string) => void
  disableVisibility?: boolean
}

export default function GeneralConfigCard({
  name,
  setName,
  description,
  setDescription,
  icon,
  onChooseIcon,
  visibility,
  setVisibility,
  disableVisibility = false,
}: GeneralConfigCardProps) {
  return (
    <div id="tour-general-config-card" className="rounded-xl border border-border bg-card p-6 shadow-sm space-y-5">
      <div className="flex items-center gap-2 border-b border-border pb-3 mb-2">
        <Settings2 className="h-4 w-4 text-primary" />
        <h2 className="text-sm font-semibold text-foreground">General Configurations</h2>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-5">
        {/* Project name input */}
        <div className="space-y-1.5">
          <label className="text-xs font-semibold text-muted-foreground uppercase" htmlFor="projectName">
            Project Name
          </label>
          <input
            id="projectName"
            type="text"
            placeholder="Enter project title"
            value={name}
            onChange={(e) => setName(e.target.value)}
            className="w-full h-10 px-3.5 rounded-md border border-input bg-transparent text-sm focus:outline-none focus:ring-2 focus:ring-primary/40 focus:border-primary"
            required
          />
        </div>

        {/* Icon visual selector block */}
        <div className="space-y-1.5">
          <label className="text-xs font-semibold text-muted-foreground uppercase">
            Project Icon / Emoji
          </label>
          <button
            type="button"
            onClick={onChooseIcon}
            className="flex items-center justify-between w-full h-10 px-3.5 rounded-md border border-input bg-transparent text-sm hover:bg-muted/30"
          >
            <div className="flex items-center gap-3">
              <span className="text-lg">{icon}</span>
              <span className="text-xs text-muted-foreground">Choose character symbol</span>
            </div>
            <ChevronDown className="h-4 w-4 text-muted-foreground" />
          </button>
        </div>

        {/* Project description textarea (Span 2) */}
        <div className="space-y-1.5 md:col-span-2">
          <label className="text-xs font-semibold text-muted-foreground uppercase" htmlFor="projectDesc">
            Project Description
          </label>
          <textarea
            id="projectDesc"
            rows={3}
            placeholder="Enter project scope details..."
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            className="w-full p-3.5 rounded-md border border-input bg-transparent text-sm focus:outline-none focus:ring-2 focus:ring-primary/40 focus:border-primary resize-none"
          />
        </div>

        {/* Visibility (Span 2) */}
        <div className="space-y-2 md:col-span-2">
          <label className="text-xs font-semibold text-muted-foreground uppercase">
            Visibility
          </label>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <label className={`flex items-start gap-3 p-3.5 rounded-lg border transition-all ${
              disableVisibility ? "opacity-60 cursor-not-allowed" : "cursor-pointer"
            } ${
              visibility === "private"
                ? "border-primary bg-primary/5"
                : disableVisibility ? "border-input" : "border-input hover:bg-muted/30"
            }`}>
              <input
                type="radio"
                name="visibility"
                value="private"
                checked={visibility === "private"}
                onChange={() => setVisibility("private")}
                disabled={disableVisibility}
                className="mt-1 h-4 w-4 text-primary focus:ring-primary disabled:opacity-50"
              />
              <div className="space-y-0.5">
                <span className="text-sm font-medium text-foreground">Visible only to you</span>
                <p className="text-xs text-muted-foreground">
                  If others have access to the associated git repo, they will still be able to join the project.
                </p>
              </div>
            </label>

            <label className={`flex items-start gap-3 p-3.5 rounded-lg border transition-all ${
              disableVisibility ? "opacity-60 cursor-not-allowed" : "cursor-pointer"
            } ${
              visibility === "public"
                ? "border-primary bg-primary/5"
                : disableVisibility ? "border-input" : "border-input hover:bg-muted/30"
            }`}>
              <input
                type="radio"
                name="visibility"
                value="public"
                checked={visibility === "public"}
                onChange={() => setVisibility("public")}
                disabled={disableVisibility}
                className="mt-1 h-4 w-4 text-primary focus:ring-primary disabled:opacity-50"
              />
              <div className="space-y-0.5">
                <span className="text-sm font-medium text-foreground">Visible to everyone in the org</span>
                <p className="text-xs text-muted-foreground">
                  Anyone in your organization can view and join this project.
                </p>
              </div>
            </label>
          </div>
          {disableVisibility && (
            <p className="text-xs text-muted-foreground mt-1">
              Visibility of the showcase project is locked to private.
            </p>
          )}
        </div>

      </div>
    </div>
  )
}
