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
import { ChevronDown, ChevronRight, Search, X } from "lucide-react"
import { Skill, Tool } from "@/lib/types"
import predefinedAgentsData from "../[id]/agent_definitions.json"

export interface PredefinedAgent {
  id: string
  role: string
  persona: string
  description: string
  color: string
  icon: string
  skills?: Skill[]
  tools?: Tool[]
}

const predefinedAgents = predefinedAgentsData as PredefinedAgent[]

interface ControlNodeItem {
  type: 'agent' | 'function' | 'join' | 'trigger' | 'loop' | 'noop' | 'manual_trigger' | 'form_trigger'
  title: string
  description: string
  icon: string
  bgColor: string
  hoverBorder: string
}

const triggerItems: ControlNodeItem[] = [
  {
    type: 'trigger',
    title: 'Scheduled Trigger',
    description: 'Triggers the workflow periodically based on interval or cron expression.',
    icon: '⏰',
    bgColor: 'bg-purple-500/10 text-purple-700 dark:bg-purple-500/20 dark:text-purple-300',
    hoverBorder: 'hover:border-purple-500/30 hover:bg-purple-500/5',
  },
  {
    type: 'manual_trigger',
    title: 'Manual Trigger',
    description: 'Allows you to start the workflow manually by clicking a button.',
    icon: '🖱️',
    bgColor: 'bg-purple-500/10 text-purple-700 dark:bg-purple-500/20 dark:text-purple-300',
    hoverBorder: 'hover:border-purple-500/30 hover:bg-purple-500/5',
  },
  {
    type: 'form_trigger',
    title: 'Form Trigger',
    description: 'Triggers the workflow via a form submission, capturing input and output data.',
    icon: '📝',
    bgColor: 'bg-purple-500/10 text-purple-700 dark:bg-purple-500/20 dark:text-purple-300',
    hoverBorder: 'hover:border-purple-500/30 hover:bg-purple-500/5',
  },
]

const genericAgentItem: ControlNodeItem = {
  type: 'agent',
  title: 'Generic LLM',
  description: 'A blank agent node that reasons, plans, and invokes tools autonomously.',
  icon: '🤖',
  bgColor: 'bg-blue-500/10 text-blue-700 dark:bg-blue-500/20 dark:text-blue-300',
  hoverBorder: 'hover:border-primary/30 hover:bg-primary/5',
}

const flowControlItems: ControlNodeItem[] = [
  {
    type: 'function',
    title: 'Function Node',
    description: 'Runs deterministic Python sync/async functions or generators.',
    icon: '⚡',
    bgColor: 'bg-amber-500/10 text-amber-700 dark:bg-amber-500/20 dark:text-amber-300',
    hoverBorder: 'hover:border-amber-500/30 hover:bg-amber-500/5',
  },
  {
    type: 'join',
    title: 'Join Node',
    description: 'Waits for all preceding paths to complete before continuing.',
    icon: '⛙',
    bgColor: 'bg-teal-500/10 text-teal-700 dark:bg-teal-500/20 dark:text-teal-300',
    hoverBorder: 'hover:border-teal-500/30 hover:bg-teal-500/5',
  },
  {
    type: 'loop',
    title: 'Loop Node',
    description: 'Iterates over a collection, yielding loop body and done execution paths.',
    icon: '🔁',
    bgColor: 'bg-emerald-500/10 text-emerald-700 dark:bg-emerald-500/20 dark:text-emerald-300',
    hoverBorder: 'hover:border-emerald-500/30 hover:bg-emerald-500/5',
  },
  {
    type: 'noop',
    title: 'Noop Node',
    description: 'No-operation node used purely for flow redirection and routing.',
    icon: '➡️',
    bgColor: 'bg-slate-500/10 text-slate-700 dark:bg-slate-500/20 dark:text-slate-300',
    hoverBorder: 'hover:border-slate-500/30 hover:bg-slate-500/5',
  },
]

interface NodeToolboxProps {
  onAddControlNode: (type: 'agent' | 'function' | 'join' | 'trigger' | 'loop' | 'noop' | 'manual_trigger' | 'form_trigger') => void
  onAddPredefinedAgent: (agent: PredefinedAgent) => void
}

export function NodeToolbox({ onAddControlNode, onAddPredefinedAgent }: NodeToolboxProps) {
  const [expandedSections, setExpandedSections] = useState({
    agents: true,
    flowControl: true,
    triggers: true,
  })
  const [searchQuery, setSearchQuery] = useState("")

  const toggleSection = (section: 'agents' | 'flowControl' | 'triggers') => {
    setExpandedSections((prev) => ({
      ...prev,
      [section]: !prev[section],
    }))
  }

  const matchesSearch = (title: string, description: string) => {
    if (!searchQuery.trim()) return true
    const query = searchQuery.toLowerCase()
    return (
      title.toLowerCase().includes(query) ||
      description.toLowerCase().includes(query)
    )
  }

  // Filter items
  const filteredTriggers = triggerItems.filter((item) =>
    matchesSearch(item.title, item.description)
  )

  const showGenericAgent = matchesSearch(
    genericAgentItem.title,
    genericAgentItem.description
  )

  const matchesAgentSearch = (agent: PredefinedAgent) => {
    if (!searchQuery.trim()) return true
    const query = searchQuery.toLowerCase()

    const matchesName = agent.role.toLowerCase().includes(query)
    const matchesType = agent.persona.toLowerCase().includes(query)
    const matchesDescription = agent.description.toLowerCase().includes(query)
    const matchesSkills = (agent.skills || []).some(
      (skill) =>
        skill.displayName.toLowerCase().includes(query) ||
        skill.description.toLowerCase().includes(query)
    )

    return matchesName || matchesType || matchesDescription || matchesSkills
  }

  const filteredPredefinedAgents = predefinedAgents.filter(matchesAgentSearch)

  const filteredFlowControl = flowControlItems.filter((item) =>
    matchesSearch(item.title, item.description)
  )

  const isSearching = searchQuery.trim() !== ""

  const showTriggersSection = filteredTriggers.length > 0
  const showAgentsSection = showGenericAgent || filteredPredefinedAgents.length > 0
  const showFlowControlSection = filteredFlowControl.length > 0

  const isTriggersExpanded = isSearching ? showTriggersSection : expandedSections.triggers
  const isAgentsExpanded = isSearching ? showAgentsSection : expandedSections.agents
  const isFlowControlExpanded = isSearching ? showFlowControlSection : expandedSections.flowControl

  const totalMatches =
    filteredTriggers.length +
    (showGenericAgent ? 1 : 0) +
    filteredPredefinedAgents.length +
    filteredFlowControl.length

  return (
    <div className="flex-1 flex flex-col h-full overflow-hidden border-r border-border bg-card select-none">
      {/* Sidebar Header */}
      <div className="p-4 pb-2 shrink-0">
        <h3 className="font-semibold text-xs text-foreground uppercase tracking-wider">
          Agent & Tools
        </h3>
        <p className="text-[10px] text-muted-foreground mt-0.5">
          Drag handles or click to add elements
        </p>
      </div>

      {/* Search Bar */}
      <div className="px-4 pb-3 border-b border-border shrink-0">
        <div className="relative">
          <Search className="absolute left-2.5 top-2.5 h-3.5 w-3.5 text-muted-foreground/70" />
          <input
            type="text"
            placeholder="Search nodes..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-full pl-8 pr-8 py-1.5 bg-muted/50 border border-border rounded-lg text-xs placeholder:text-muted-foreground/70 focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring transition-colors"
          />
          {searchQuery && (
            <button
              onClick={() => setSearchQuery("")}
              className="absolute right-2.5 top-2.5 text-muted-foreground/70 hover:text-foreground transition-colors"
            >
              <X className="h-3.5 w-3.5" />
            </button>
          )}
        </div>
      </div>

      {/* Toolbox Tree Content */}
      <div className="flex-1 overflow-y-auto p-4 space-y-4">
        {isSearching && totalMatches === 0 && (
          <div className="text-center py-8 text-xs text-muted-foreground">
            No nodes match "{searchQuery}"
          </div>
        )}

        {/* SECTION 1: TRIGGERS */}
        {showTriggersSection && (
          <div className="space-y-2">
            <button
              onClick={() => !isSearching && toggleSection("triggers")}
              disabled={isSearching}
              className={`w-full flex items-center gap-1.5 py-1.5 text-xs font-bold text-muted-foreground uppercase tracking-wider transition-colors ${
                isSearching ? "cursor-default" : "hover:text-foreground"
              }`}
            >
              {!isSearching && (
                <>
                  {isTriggersExpanded ? (
                    <ChevronDown className="h-3.5 w-3.5 text-muted-foreground/70" />
                  ) : (
                    <ChevronRight className="h-3.5 w-3.5 text-muted-foreground/70" />
                  )}
                </>
              )}
              Triggers ({filteredTriggers.length})
            </button>

            {isTriggersExpanded && (
              <div className="grid grid-cols-1 gap-2.5 pl-5">
                {filteredTriggers.map((item) => (
                  <button
                    key={item.type}
                    onClick={() => onAddControlNode(item.type)}
                    className={`group w-full p-3 rounded-xl border border-border bg-card hover:border-purple-500/30 hover:bg-purple-500/5 text-left flex items-center hover:items-start gap-3 transition-all`}
                  >
                    <div className={`h-6 w-6 rounded-lg ${item.bgColor} flex items-center justify-center text-xs shrink-0 transition-all duration-300 group-hover:h-9 group-hover:w-9 group-hover:text-lg`}>
                      {item.icon}
                    </div>
                    <div className="min-w-0 transition-all duration-300 group-hover:pt-0.5">
                      <h4 className="text-xs font-bold text-foreground">
                        {item.title}
                      </h4>
                      <p className="text-[9px] text-muted-foreground/80 mt-0 max-h-0 opacity-0 group-hover:max-h-20 group-hover:opacity-100 group-hover:mt-1 transition-all duration-300 ease-in-out overflow-hidden leading-normal">
                        {item.description}
                      </p>
                    </div>
                  </button>
                ))}
              </div>
            )}
          </div>
        )}

        {/* SECTION 2: AGENTS */}
        {showAgentsSection && (
          <div className="space-y-2">
            <button
              onClick={() => !isSearching && toggleSection("agents")}
              disabled={isSearching}
              className={`w-full flex items-center gap-1.5 py-1.5 text-xs font-bold text-muted-foreground uppercase tracking-wider transition-colors ${
                isSearching ? "cursor-default" : "hover:text-foreground"
              }`}
            >
              {!isSearching && (
                <>
                  {isAgentsExpanded ? (
                    <ChevronDown className="h-3.5 w-3.5 text-muted-foreground/70" />
                  ) : (
                    <ChevronRight className="h-3.5 w-3.5 text-muted-foreground/70" />
                  )}
                </>
              )}
              Agents ({(showGenericAgent ? 1 : 0) + filteredPredefinedAgents.length})
            </button>

            {isAgentsExpanded && (
              <div className="space-y-2.5 pl-5">
                {/* Generic LLM */}
                {showGenericAgent && (
                  <button
                    onClick={() => onAddControlNode("agent")}
                    className="group w-full p-3 rounded-xl border border-border bg-card hover:border-primary/30 hover:bg-primary/5 text-left flex items-center hover:items-start gap-3 transition-all"
                  >
                    <div className="h-6 w-6 rounded-lg bg-blue-500/10 text-blue-700 dark:bg-blue-500/20 dark:text-blue-300 flex items-center justify-center text-xs shrink-0 transition-all duration-300 group-hover:h-9 group-hover:w-9 group-hover:text-lg">
                      🤖
                    </div>
                    <div className="min-w-0 transition-all duration-300 group-hover:pt-0.5">
                      <h4 className="text-xs font-bold text-foreground">
                        {genericAgentItem.title}
                      </h4>
                      <p className="text-[9px] text-muted-foreground/80 mt-0 max-h-0 opacity-0 group-hover:max-h-20 group-hover:opacity-100 group-hover:mt-1 transition-all duration-300 ease-in-out overflow-hidden leading-normal">
                        {genericAgentItem.description}
                      </p>
                    </div>
                  </button>
                )}

                {/* Predefined Agents */}
                {filteredPredefinedAgents.map((agent) => (
                  <button
                    key={agent.id}
                    onClick={() => onAddPredefinedAgent(agent)}
                    className="w-full p-3 border border-border rounded-xl bg-card hover:border-primary/30 hover:bg-primary/5 text-left flex items-center hover:items-start gap-3 group relative select-none transition-all duration-200"
                  >
                    <div className={`h-6 w-6 shrink-0 rounded-lg border flex items-center justify-center text-xs transition-all duration-300 group-hover:h-9 group-hover:w-9 group-hover:text-lg ${agent.color}`}>
                      {agent.icon}
                    </div>
                    <div className="min-w-0 flex-1 transition-all duration-300 group-hover:pt-0.5">
                      <h4 className="text-xs font-bold text-foreground truncate">
                        {agent.role}
                      </h4>
                      <p className="text-[9px] text-muted-foreground/80 mt-0 max-h-0 opacity-0 group-hover:max-h-20 group-hover:opacity-100 group-hover:mt-1 transition-all duration-300 ease-in-out overflow-hidden leading-normal">
                        {agent.description}
                      </p>
                    </div>
                  </button>
                ))}
              </div>
            )}
          </div>
        )}

        {/* SECTION 3: FLOW CONTROL */}
        {showFlowControlSection && (
          <div className="space-y-2">
            <button
              onClick={() => !isSearching && toggleSection("flowControl")}
              disabled={isSearching}
              className={`w-full flex items-center gap-1.5 py-1.5 text-xs font-bold text-muted-foreground uppercase tracking-wider transition-colors ${
                isSearching ? "cursor-default" : "hover:text-foreground"
              }`}
            >
              {!isSearching && (
                <>
                  {isFlowControlExpanded ? (
                    <ChevronDown className="h-3.5 w-3.5 text-muted-foreground/70" />
                  ) : (
                    <ChevronRight className="h-3.5 w-3.5 text-muted-foreground/70" />
                  )}
                </>
              )}
              Flow Control ({filteredFlowControl.length})
            </button>

            {isFlowControlExpanded && (
              <div className="grid grid-cols-1 gap-2.5 pl-5">
                {filteredFlowControl.map((item) => (
                  <button
                    key={item.type}
                    onClick={() => onAddControlNode(item.type)}
                    className={`group w-full p-3 rounded-xl border border-border bg-card ${item.hoverBorder} text-left flex items-center hover:items-start gap-3 transition-all`}
                  >
                    <div className={`h-6 w-6 rounded-lg ${item.bgColor} flex items-center justify-center text-xs shrink-0 transition-all duration-300 group-hover:h-9 group-hover:w-9 group-hover:text-lg`}>
                      {item.icon}
                    </div>
                    <div className="min-w-0 transition-all duration-300 group-hover:pt-0.5">
                      <h4 className="text-xs font-bold text-foreground">
                        {item.title}
                      </h4>
                      <p className="text-[9px] text-muted-foreground/80 mt-0 max-h-0 opacity-0 group-hover:max-h-20 group-hover:opacity-100 group-hover:mt-1 transition-all duration-300 ease-in-out overflow-hidden leading-normal">
                        {item.description}
                      </p>
                    </div>
                  </button>
                ))}
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  )
}
