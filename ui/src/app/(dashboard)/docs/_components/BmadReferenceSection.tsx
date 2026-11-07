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

import React, { useState, useMemo } from "react"
import { 
  Sparkles, 
  Search, 
  Layers, 
  Filter, 
  ExternalLink, 
  Code, 
  ArrowRight,
  BookOpen,
  CheckCircle2,
  Workflow,
  Tag,
  FileText,
  SlidersHorizontal,
  ChevronRight,
  Box
} from "lucide-react"
import { 
  BMAD_SKILLS_CATALOG, 
  BMAD_PHASES, 
  BMAD_MODULES, 
  BmadSkill 
} from "./data/bmad-catalog"
import { BmadSkillDetailModal } from "./BmadSkillDetailModal"
import { cn } from "@/lib/utils"

interface BmadReferenceSectionProps {
  id?: string
  sectionRef?: React.RefObject<HTMLDivElement | null>
}

export function BmadReferenceSection({ id = "tour-docs-bmad-reference", sectionRef }: BmadReferenceSectionProps) {
  const [searchQuery, setSearchQuery] = useState("")
  const [selectedPhase, setSelectedPhase] = useState<string>("all")
  const [selectedModule, setSelectedModule] = useState<string>("All")
  const [showIdeSkills, setShowIdeSkills] = useState<boolean>(false)
  const [selectedSkill, setSelectedSkill] = useState<BmadSkill | null>(null)
  const [isModalOpen, setIsModalOpen] = useState(false)

  // Filter skills based on query, phase, module, and IDE skills toggle
  const filteredSkills = useMemo(() => {
    return BMAD_SKILLS_CATALOG.filter((skill) => {
      // IDE-based skills check: hide skills with apf_supported === false by default unless toggle is checked
      if (!showIdeSkills && skill.apf_supported === false) {
        return false
      }

      // Phase check
      if (selectedPhase !== "all" && skill.phase !== selectedPhase) {
        return false
      }

      // Module check
      if (selectedModule !== "All" && skill.module !== selectedModule) {
        return false
      }

      // Query check
      if (searchQuery.trim() !== "") {
        const q = searchQuery.toLowerCase().trim()
        const matchesName = skill.name.toLowerCase().includes(q)
        const matchesId = skill.id.toLowerCase().includes(q)
        const matchesCode = skill.code.toLowerCase().includes(q)
        const matchesDesc = skill.description.toLowerCase().includes(q)
        const matchesTags = skill.tags.some((t) => t.toLowerCase().includes(q))
        const matchesOutputs = (skill.outputs || "").toLowerCase().includes(q)
        return matchesName || matchesId || matchesCode || matchesDesc || matchesTags || matchesOutputs
      }

      return true
    })
  }, [searchQuery, selectedPhase, selectedModule, showIdeSkills])

  const handleOpenDetail = (skill: BmadSkill) => {
    setSelectedSkill(skill)
    setIsModalOpen(true)
  }

  const getPhaseColor = (phase: string) => {
    switch (phase) {
      case "1-ideation":
        return "bg-amber-500/10 text-amber-700 dark:text-amber-300 border-amber-500/20"
      case "2-planning":
        return "bg-blue-500/10 text-blue-700 dark:text-blue-300 border-blue-500/20"
      case "3-solutioning":
        return "bg-purple-500/10 text-purple-700 dark:text-purple-300 border-purple-500/20"
      case "4-implementation":
        return "bg-emerald-500/10 text-emerald-700 dark:text-emerald-300 border-emerald-500/20"
      case "0-learning":
        return "bg-indigo-500/10 text-indigo-700 dark:text-indigo-300 border-indigo-500/20"
      default:
        return "bg-slate-500/10 text-slate-700 dark:text-slate-300 border-slate-500/20"
    }
  }

  return (
    <div id={id} ref={sectionRef} className="space-y-6 scroll-mt-10">
      {/* Section Header */}
      <div className="space-y-2 border-b border-border pb-4 select-none">
        <div className="flex items-center gap-3">
          <Sparkles className="h-6 w-6 text-primary" />
          <h2 className="text-base font-bold text-foreground">BMAD Reference</h2>
        </div>
        <p className="text-[11px] text-muted-foreground leading-relaxed">
          Comprehensive reference for the BMad Method lifecycle: from discovery and PRD creation to architecture spines, sprint planning, build loops, test automation, and autonomous sweeps.
        </p>
      </div>

      {/* Filter and Search Bar Toolbar */}
      <div className="space-y-3">
        <div className="flex flex-col sm:flex-row items-stretch sm:items-center gap-3">
          {/* Search Input */}
          <div className="relative flex-1">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Search skills, commands (e.g. BD, PRD), keywords..."
              className="w-full h-10 pl-9 pr-4 rounded-xl border border-input bg-card text-xs text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all shadow-sm"
              data-testid="bmad-search-input"
            />
            {searchQuery && (
              <button
                type="button"
                onClick={() => setSearchQuery("")}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-xs text-muted-foreground hover:text-foreground font-semibold"
              >
                Clear
              </button>
            )}
          </div>

          {/* Module Filter Dropdown */}
          <div className="flex items-center gap-2">
            <SlidersHorizontal className="h-4 w-4 text-muted-foreground hidden sm:block shrink-0" />
            <select
              value={selectedModule}
              onChange={(e) => setSelectedModule(e.target.value)}
              className="h-10 px-3 rounded-xl border border-input bg-card text-xs font-medium text-foreground focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all shadow-sm cursor-pointer"
              data-testid="bmad-module-select"
            >
              {BMAD_MODULES.map((mod) => (
                <option key={mod} value={mod}>
                  {mod === "All" ? "All Modules" : mod}
                </option>
              ))}
            </select>
          </div>

          {/* Show IDE based skills Toggle */}
          <label
            className="flex items-center gap-2 h-10 px-3.5 rounded-xl border border-input bg-card text-xs font-medium text-foreground hover:bg-muted/50 transition-all shadow-sm cursor-pointer select-none shrink-0"
            data-testid="bmad-toggle-ide-skills-label"
          >
            <input
              type="checkbox"
              checked={showIdeSkills}
              onChange={(e) => setShowIdeSkills(e.target.checked)}
              className="h-4 w-4 rounded border-input text-primary focus:ring-primary/20 accent-primary cursor-pointer"
              data-testid="bmad-toggle-ide-skills"
            />
            <span>Show IDE based skills</span>
          </label>
        </div>

        {/* Phase Filter Tabs */}
        <div className="flex flex-wrap gap-1.5 pt-1">
          {BMAD_PHASES.map((p) => {
            const isSelected = selectedPhase === p.id
            return (
              <button
                key={p.id}
                type="button"
                onClick={() => setSelectedPhase(p.id)}
                className={cn(
                  "px-3 py-1.5 rounded-lg text-xs font-medium transition-all duration-200 border select-none",
                  isSelected
                    ? "bg-primary text-primary-foreground border-primary font-semibold shadow-sm"
                    : "bg-card border-border text-muted-foreground hover:bg-muted hover:text-foreground"
                )}
                data-testid={`bmad-phase-filter-${p.id}`}
              >
                {p.label}
              </button>
            )
          })}
        </div>
      </div>

      {/* Skill Cards Grid */}
      {filteredSkills.length === 0 ? (
        <div className="flex flex-col items-center justify-center p-12 text-center border border-dashed border-border rounded-2xl bg-card/50 space-y-3">
          <BookOpen className="h-10 w-10 text-muted-foreground opacity-40" />
          <h3 className="text-sm font-bold text-foreground">No matching BMAD skills found</h3>
          <p className="text-xs text-muted-foreground max-w-sm">
            Try adjusting your search query or switching the phase filter to find the capability you are looking for.
          </p>
          <button
            type="button"
            onClick={() => {
              setSearchQuery("")
              setSelectedPhase("all")
              setSelectedModule("All")
              setShowIdeSkills(false)
            }}
            className="mt-2 text-xs font-semibold text-primary hover:underline"
          >
            Reset all filters
          </button>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
          {filteredSkills.map((skill) => (
            <div
              key={skill.id}
              onClick={() => handleOpenDetail(skill)}
              className="group relative flex flex-col justify-between p-4 rounded-xl border border-border bg-card hover:border-primary/40 hover:shadow-md transition-all duration-200 cursor-pointer space-y-3"
              data-testid={`bmad-skill-card-${skill.id}`}
            >
              <div className="space-y-2">
                {/* Header badges */}
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <span className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-semibold border ${getPhaseColor(skill.phase)}`}>
                    {skill.phaseLabel}
                  </span>

                  <div className="flex items-center gap-1.5 shrink-0">
                    {skill.requires_sandbox && (
                      <span
                        title="Sandbox execution required"
                        className="inline-flex items-center justify-center p-1 rounded bg-amber-500/10 text-amber-600 dark:text-amber-400 border border-amber-500/20"
                        data-testid={`bmad-icon-sandbox-${skill.id}`}
                      >
                        <Box className="h-3 w-3" />
                      </span>
                    )}
                    {skill.apf_supported === false && !skill.requires_sandbox && (
                      <span className="inline-flex items-center px-1.5 py-0.5 rounded bg-amber-500/10 text-amber-700 dark:text-amber-300 text-[10px] font-semibold border border-amber-500/20">
                        IDE Only
                      </span>
                    )}
                    <span className="inline-flex items-center px-1.5 py-0.5 rounded bg-primary/10 text-primary text-[10px] font-mono font-bold">
                      {skill.code}
                    </span>
                    <span className="text-[10px] text-muted-foreground font-mono bg-muted/60 px-1.5 py-0.5 rounded">
                      /{skill.id}
                    </span>
                  </div>
                </div>

                {/* Title */}
                <h3 className="text-sm font-bold text-foreground group-hover:text-primary transition-colors flex items-center justify-between gap-2">
                  <span className="break-words">{skill.name}</span>
                  <ChevronRight className="h-4 w-4 text-muted-foreground group-hover:text-primary group-hover:translate-x-0.5 transition-all opacity-0 group-hover:opacity-100 shrink-0" />
                </h3>

                {/* Description */}
                <p className="text-xs text-muted-foreground leading-relaxed line-clamp-3 break-words">
                  {skill.description}
                </p>
              </div>

              {/* Footer info */}
              <div className="pt-2 border-t border-border/60 flex items-center justify-between gap-2 text-[11px] text-muted-foreground">
                <span className="truncate flex-1">
                  {skill.outputs ? `Outputs: ${skill.outputs}` : skill.module}
                </span>
                <span className="text-[10px] font-semibold text-primary group-hover:underline shrink-0">
                  View Spec &rarr;
                </span>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Detailed Modal */}
      <BmadSkillDetailModal
        skill={selectedSkill}
        isOpen={isModalOpen}
        onClose={() => {
          setIsModalOpen(false)
          setSelectedSkill(null)
        }}
      />
    </div>
  )
}
