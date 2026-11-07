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
import { 
  Cable, 
  Key, 
  Github, 
  Gitlab, 
  Trello, 
  Figma, 
  Lock, 
  ShieldCheck, 
  ExternalLink 
} from "lucide-react"

interface IntegrationsSectionProps {
  id?: string
  sectionRef?: React.RefObject<HTMLDivElement | null>
}

export function IntegrationsSection({ id = "tour-docs-integrations", sectionRef }: IntegrationsSectionProps) {
  const integrationList = [
    {
      name: "GitHub",
      type: "VCS & PR Automation",
      icon: Github,
      desc: "Synchronize branches, create pull requests, trigger GitHub Actions workflows, and comment on code reviews automatically."
    },
    {
      name: "GitLab",
      type: "VCS & Pipeline Integration",
      icon: Gitlab,
      desc: "Integrate with GitLab merge requests, personal access tokens, and internal self-hosted enterprise repositories."
    },
    {
      name: "Jira / Trello",
      type: "Task Management",
      icon: Trello,
      desc: "Sync BMad user stories and epics directly with your team's project boards, status columns, and sprint cycles."
    },
    {
      name: "Figma",
      type: "Design Systems",
      icon: Figma,
      desc: "Extract UI components, color styles, layout coordinates, and design tokens directly into frontend scaffolding."
    }
  ]

  return (
    <div id={id} ref={sectionRef} className="space-y-6 scroll-mt-10">
      {/* Section Header */}
      <div className="space-y-2 border-b border-border pb-4 select-none">
        <div className="flex items-center gap-3">
          <Cable className="h-6 w-6 text-primary" />
          <h2 className="text-base font-bold text-foreground">Integrations & Credentials</h2>
        </div>
        <p className="text-[11px] text-muted-foreground leading-relaxed">
          How to connect your developer tools and secure your API credentials globally across projects.
        </p>
      </div>

      {/* Security Note Card */}
      <div className="p-4 rounded-xl border border-primary/20 bg-primary/5 space-y-2">
        <div className="flex items-center gap-2 text-primary font-bold text-xs">
          <ShieldCheck className="h-4 w-4" />
          <span>Global Master Credentials Model</span>
        </div>
        <p className="text-[11px] text-muted-foreground leading-relaxed">
          Personal access tokens configured under <a href="/settings" className="text-primary hover:underline font-semibold">Settings &rarr; Application Credentials</a> are encrypted and tied to your user profile. When a project enables an integration, it securely inherits your personal master key without duplicating credentials across repos.
        </p>
      </div>

      {/* Integrations Grid */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {integrationList.map((item) => {
          const Icon = item.icon
          return (
            <div key={item.name} className="p-4 rounded-xl border border-border bg-card space-y-2 shadow-sm flex flex-col justify-between">
              <div className="space-y-2">
                <div className="flex items-center gap-3">
                  <div className="h-8 w-8 rounded-lg bg-muted flex items-center justify-center text-foreground shrink-0">
                    <Icon className="h-4 w-4" />
                  </div>
                  <div className="min-w-0 flex-1">
                    <h4 className="text-xs font-bold text-foreground truncate">{item.name}</h4>
                    <span className="text-[10px] text-muted-foreground font-medium block truncate">{item.type}</span>
                  </div>
                </div>
                <p className="text-xs text-muted-foreground leading-relaxed">
                  {item.desc}
                </p>
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}
