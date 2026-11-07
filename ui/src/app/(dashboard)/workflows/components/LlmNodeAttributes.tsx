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

import { useState, useEffect } from "react"
import { Plus, Trash2, ChevronDown, ChevronRight, Settings, ArrowLeftRight, Wrench } from "lucide-react"
import skillDefinitions from "../[id]/skill_definitions.json"
import agentDefinitions from "../[id]/agent_definitions.json"
import toolDefinitions from "../[id]/tool_definitions.json"
import { isValidPythonIdentifier } from "@/lib/utils"
import { KeyValueEditor } from "./KeyValueEditor"

interface LlmNodeAttributesProps {
  data: {
    name: string
    displayName?: string
    agentId?: string
    description?: string
    model?: string
    instruction?: string
    skills?: { displayName: string; name: string; description: string }[]
    tools?: { displayName: string; name: string; description: string }[]
    input?: Record<string, any>
    output?: Record<string, any>
    [key: string]: any
  }
  onChange: (key: string, value: any) => void
  existingNames?: string[]
}

export const LlmNodeAttributes = ({ data, onChange, existingNames }: LlmNodeAttributesProps) => {
  const [activeTab, setActiveTab] = useState<"general" | "skills_tools" | "input_output">("general")
  const [isSkillsExpanded, setIsSkillsExpanded] = useState(true)
  const [isToolsExpanded, setIsToolsExpanded] = useState(true)

  const name = data.name || ""
  const isValidName = isValidPythonIdentifier(name)
  const isDuplicate = existingNames ? existingNames.includes(name) : false

  let nameError = ""
  if (!name) {
    nameError = "ADK Name is required"
  } else if (!isValidName) {
    nameError = "Must be a valid Python identifier (starts with letter/underscore, alphanumeric/underscore only, and not a Python keyword)"
  } else if (isDuplicate) {
    nameError = "Name must be unique across all nodes in the workflow"
  }


  const agentDef = data.agentId ? agentDefinitions.find((a) => a.id === data.agentId) : null
  const agentSkills: any[] = (agentDef as any)?.skills || []
  const allAvailableSkills = [
    ...skillDefinitions,
    ...agentSkills.filter(
      (as: any) => !skillDefinitions.some((sd) => sd.name === as.name)
    ),
  ]
  const skillsToShow = allAvailableSkills

  const agentTools: any[] = (agentDef as any)?.tools || []
  const allAvailableTools = [
    ...toolDefinitions,
    ...agentTools.filter(
      (at: any) => !toolDefinitions.some((td) => td.name === at.name)
    ),
  ]
  const toolsToShow = allAvailableTools

  const isSkillChecked = (skill: { name: string }) => {
    return (data.skills || []).some((s: any) =>
      typeof s === "string" ? s === skill.name : s.name === skill.name
    )
  }

  const handleSkillToggle = (skill: { displayName?: string; name: string; description?: string }) => {
    const currentSkills = data.skills || []
    const exists = isSkillChecked(skill)
    let updatedSkills: { displayName: string; name: string; description: string }[]
    if (exists) {
      updatedSkills = currentSkills
        .filter((s: any) => (typeof s === "string" ? s !== skill.name : s.name !== skill.name))
        .map((s: any) =>
          typeof s === "string"
            ? { displayName: s, name: s, description: "" }
            : s
        )
    } else {
      updatedSkills = [
        ...currentSkills.map((s: any) =>
          typeof s === "string"
            ? { displayName: s, name: s, description: "" }
            : s
        ),
        {
          displayName: skill.displayName || skill.name,
          name: skill.name,
          description: skill.description || "",
        },
      ]
    }
    onChange("skills", updatedSkills)
  }

  const isToolChecked = (tool: { name: string }) => {
    return (data.tools || []).some((t: any) =>
      typeof t === "string" ? t === tool.name : t.name === tool.name
    )
  }

  const handleToolToggle = (tool: { displayName?: string; name: string; description?: string }) => {
    const currentTools = data.tools || []
    const exists = isToolChecked(tool)
    let updatedTools: { displayName: string; name: string; description: string }[]
    if (exists) {
      updatedTools = currentTools
        .filter((t: any) => (typeof t === "string" ? t !== tool.name : t.name !== tool.name))
        .map((t: any) =>
          typeof t === "string"
            ? { displayName: t, name: t, description: "" }
            : t
        )
    } else {
      updatedTools = [
        ...currentTools.map((t: any) =>
          typeof t === "string"
            ? { displayName: t, name: t, description: "" }
            : t
        ),
        {
          displayName: tool.displayName || tool.name,
          name: tool.name,
          description: tool.description || "",
        },
      ]
    }
    onChange("tools", updatedTools)
  }

  return (
    <div className="space-y-4">
      {/* Tab Navigation Headers */}
      <div className="flex gap-5 border-b border-border/60 shrink-0">
        <button
          type="button"
          onClick={() => setActiveTab("general")}
          className={`pb-2 text-[10px] font-bold uppercase tracking-wider border-b-2 transition-all flex items-center gap-1.5 ${
            activeTab === "general"
              ? "border-primary text-primary"
              : "border-transparent text-muted-foreground hover:text-foreground"
          }`}
        >
          <Settings className="h-3.5 w-3.5" />
          General
        </button>
        <button
          type="button"
          onClick={() => setActiveTab("input_output")}
          className={`pb-2 text-[10px] font-bold uppercase tracking-wider border-b-2 transition-all flex items-center gap-1.5 ${
            activeTab === "input_output"
              ? "border-primary text-primary"
              : "border-transparent text-muted-foreground hover:text-foreground"
          }`}
        >
          <ArrowLeftRight className="h-3.5 w-3.5" />
          Input & Output
        </button>
        <button
          type="button"
          onClick={() => setActiveTab("skills_tools")}
          className={`pb-2 text-[10px] font-bold uppercase tracking-wider border-b-2 transition-all flex items-center gap-1.5 ${
            activeTab === "skills_tools"
              ? "border-primary text-primary"
              : "border-transparent text-muted-foreground hover:text-foreground"
          }`}
        >
          <Wrench className="h-3.5 w-3.5" />
          Skills & Tools
        </button>
      </div>

      {/* Tab Contents */}
      {activeTab === "general" && (
        <div className="space-y-4">
          {/* 1. Display Name */}
          <div className="space-y-1.5">
            <label className="text-[10px] font-bold text-muted-foreground uppercase tracking-wide">
              Display Name
            </label>
            <input
              type="text"
              value={data.displayName || ""}
              onChange={(e) => onChange("displayName", e.target.value)}
              placeholder="Friendly name for this node"
              className="flex h-9 w-full rounded-lg border border-input bg-transparent px-3 py-1 text-xs shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-primary"
            />
            <p className="text-[9px] text-muted-foreground/80 leading-normal">
              A friendly title displayed on the design canvas.
            </p>
          </div>

          {/* 2. ADK Name */}
          <div className="space-y-1.5">
            <label className="text-[10px] font-bold text-muted-foreground uppercase tracking-wide">
              ADK Name
            </label>
            <input
              type="text"
              value={data.name || ""}
              onChange={(e) => onChange("name", e.target.value)}
              placeholder="e.g. agent_pm"
              className={`flex h-9 w-full rounded-lg border bg-transparent px-3 py-1 text-xs shadow-sm focus-visible:outline-none focus-visible:ring-1 ${
                nameError
                  ? "border-destructive focus-visible:ring-destructive"
                  : "border-input focus-visible:ring-primary"
              }`}
            />
            {nameError ? (
              <p className="text-[9px] text-destructive font-semibold mt-1 leading-normal">
                {nameError}
              </p>
            ) : (
              <p className="text-[9px] text-muted-foreground/80 leading-normal">
                Must be a unique Python identifier (starts with a letter, no spaces or dashes).
              </p>
            )}
          </div>

          {/* 3. Description */}
          <div className="space-y-1.5">
            <label className="text-[10px] font-bold text-muted-foreground uppercase tracking-wide">
              Description
            </label>
            <textarea
              value={data.description || ""}
              onChange={(e) => onChange("description", e.target.value)}
              placeholder="What does this node do?"
              rows={3}
              className="flex w-full rounded-lg border border-input bg-transparent px-3 py-2 text-xs shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-primary resize-none"
            />
          </div>

          {/* 4. Gemini Model */}
          <div className="space-y-1.5 border-t border-border/50 pt-4">
            <label className="text-[10px] font-bold text-muted-foreground uppercase tracking-wide">
              Gemini Model
            </label>
            <select
              value={data.model || "gemini-2.5-flash"}
              onChange={(e) => onChange("model", e.target.value)}
              className="flex h-9 w-full rounded-lg border border-input bg-card px-3 py-1 text-xs shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-primary"
            >
              <option value="gemini-2.5-flash">Gemini 2.5 Flash (Recommended)</option>
              <option value="gemini-2.5-pro">Gemini 2.5 Pro</option>
            </select>
          </div>

          {/* 5. System Instructions */}
          <div className="space-y-1.5">
            <label className="text-[10px] font-bold text-muted-foreground uppercase tracking-wide">
              System Instructions
            </label>
            <textarea
              value={data.instruction || ""}
              onChange={(e) => onChange("instruction", e.target.value)}
              placeholder="Instructions for the agent's persona and logic..."
              rows={6}
              className="flex w-full rounded-lg border border-input bg-transparent px-3 py-2 text-xs shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-primary resize-none font-mono text-[10px] leading-relaxed"
            />
          </div>

          {/* 6. Advanced ADK Primitives */}
          <div className="border-t border-border/50 pt-4 space-y-4">
            <h4 className="text-[10px] font-bold text-muted-foreground uppercase tracking-wide">
              Advanced ADK Primitives
            </h4>

            {/* Rerun on Resume */}
            <div className="flex items-center justify-between">
              <div className="space-y-0.5">
                <label className="text-[11px] font-semibold text-foreground">
                  Rerun on Resume
                </label>
                <p className="text-[9px] text-muted-foreground/80">
                  Rerun node from scratch when resumed.
                </p>
              </div>
              <input
                type="checkbox"
                checked={!!data.rerunOnResume}
                onChange={(e) => onChange("rerunOnResume", e.target.checked)}
                className="h-4 w-4 rounded border-gray-300 text-primary focus:ring-primary cursor-pointer"
              />
            </div>

            {/* Wait for Output */}
            <div className="flex items-center justify-between">
              <div className="space-y-0.5">
                <label className="text-[11px] font-semibold text-foreground">
                  Wait for Output
                </label>
                <p className="text-[9px] text-muted-foreground/80">
                  Pause state until node explicitly yields output.
                </p>
              </div>
              <input
                type="checkbox"
                checked={!!data.waitForOutput}
                onChange={(e) => onChange("waitForOutput", e.target.checked)}
                className="h-4 w-4 rounded border-gray-300 text-primary focus:ring-primary cursor-pointer"
              />
            </div>

            {/* Timeout */}
            <div className="space-y-1.5">
              <label className="text-[10px] font-bold text-muted-foreground uppercase tracking-wide">
                Timeout (Seconds)
              </label>
              <input
                type="number"
                value={data.timeout || ""}
                onChange={(e) => onChange("timeout", e.target.value)}
                placeholder="No Timeout"
                className="flex h-9 w-full rounded-lg border border-input bg-transparent px-3 py-1 text-xs shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-primary"
              />
            </div>
          </div>
        </div>
      )}

      {activeTab === "skills_tools" && (
        <div className="space-y-6">
          {/* Skills Section */}
          <div className="space-y-3">
            <button
              type="button"
              onClick={() => setIsSkillsExpanded(!isSkillsExpanded)}
              className="flex items-center justify-between w-full text-[10px] font-bold text-muted-foreground uppercase tracking-wider border-b border-border pb-1.5 hover:text-foreground transition-colors text-left"
            >
              <span>Skills</span>
              {isSkillsExpanded ? (
                <ChevronDown className="h-3.5 w-3.5 text-muted-foreground/70" />
              ) : (
                <ChevronRight className="h-3.5 w-3.5 text-muted-foreground/70" />
              )}
            </button>
            
            {isSkillsExpanded && (
              skillsToShow.length === 0 ? (
                <p className="text-[9px] text-muted-foreground/60 italic p-2 text-center">No skills available.</p>
              ) : (
                <div className="space-y-2.5">
                  {skillsToShow.map((skill) => {
                    const isChecked = isSkillChecked(skill)
                    return (
                      <label
                        key={skill.name}
                        className="flex items-start gap-2.5 p-1.5 rounded hover:bg-muted/50 transition-colors cursor-pointer select-none"
                      >
                        <input
                          type="checkbox"
                          checked={isChecked}
                          onChange={() => handleSkillToggle(skill)}
                          className="h-3.5 w-3.5 rounded border-gray-300 text-primary focus:ring-primary mt-0.5 cursor-pointer"
                        />
                        <div className="min-w-0">
                          <p className="text-[11px] font-semibold text-foreground leading-tight">
                            {skill.displayName}
                          </p>
                          <p className="text-[9px] text-muted-foreground leading-normal mt-0.5">
                            {skill.description}
                          </p>
                        </div>
                      </label>
                    )
                  })}
                </div>
              )
            )}
          </div>

          {/* Tools Section */}
          <div className="space-y-3">
            <button
              type="button"
              onClick={() => setIsToolsExpanded(!isToolsExpanded)}
              className="flex items-center justify-between w-full text-[10px] font-bold text-muted-foreground uppercase tracking-wider border-b border-border pb-1.5 hover:text-foreground transition-colors text-left"
            >
              <span>Tools</span>
              {isToolsExpanded ? (
                <ChevronDown className="h-3.5 w-3.5 text-muted-foreground/70" />
              ) : (
                <ChevronRight className="h-3.5 w-3.5 text-muted-foreground/70" />
              )}
            </button>

            {isToolsExpanded && (
              toolsToShow.length === 0 ? (
                <p className="text-[9px] text-muted-foreground/60 italic p-2 text-center">No tools available.</p>
              ) : (
                <div className="space-y-2.5">
                  {toolsToShow.map((tool) => {
                    const isChecked = isToolChecked(tool)
                    return (
                      <label
                        key={tool.name}
                        className="flex items-start gap-2.5 p-1.5 rounded hover:bg-muted/50 transition-colors cursor-pointer select-none"
                      >
                        <input
                          type="checkbox"
                          checked={isChecked}
                          onChange={() => handleToolToggle(tool)}
                          className="h-3.5 w-3.5 rounded border-gray-300 text-primary focus:ring-primary mt-0.5 cursor-pointer"
                        />
                        <div className="min-w-0">
                          <p className="text-[11px] font-semibold text-foreground leading-tight">
                            {tool.displayName}
                          </p>
                          <p className="text-[9px] text-muted-foreground leading-normal mt-0.5">
                            {tool.description}
                          </p>
                        </div>
                      </label>
                    )
                  })}
                </div>
              )
            )}
          </div>
        </div>
      )}

      {activeTab === "input_output" && (
        <div className="space-y-6">
          <KeyValueEditor
            label="Input Data"
            values={data.input}
            onChange={(val) => onChange("input", val)}
          />
          <div className="border-t border-border/50 my-4" />
          <KeyValueEditor
            label="Output Data"
            values={data.output}
            onChange={(val) => onChange("output", val)}
          />
        </div>
      )}
    </div>
  )
}
