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
import { BarChart2, Calendar, Lightbulb, Code, ArrowUpRight } from "lucide-react"
import { 
  WorkflowTemplate, 
  Artifact, 
  isPhaseComplete, 
  isPhaseInProgress,
  determineArtifactType
} from "@/lib/types"

export type PhaseType = "ONBOARDING" | "ANALYSIS" | "PLANNING" | "SOLUTIONING" | "IMPLEMENTATION"

export type PhaseStage = "locked" | "notStarted" | "inProgress" | "complete"

export const phaseConfig: Record<PhaseType, { label: string; icon: React.ComponentType<any> }> = {
  ONBOARDING: { label: "Onboard", icon: ArrowUpRight },
  ANALYSIS: { label: "Analyze", icon: BarChart2 },
  PLANNING: { label: "Plan", icon: Calendar },
  SOLUTIONING: { label: "Solution", icon: Lightbulb },
  IMPLEMENTATION: { label: "Implement", icon: Code },
}

export function getPhaseStage(
  phase: string,
  satisfiedPhases: Record<string, boolean>,
  workflow: WorkflowTemplate,
  liveArtifacts: Artifact[]
): PhaseStage {
  if (typeof window !== "undefined") {
    const search = window.location.search || ""
    if (search.includes("force=true") || search.includes("forcePhase=true") || window.localStorage?.getItem("apf_force_phase") === "true") {
      if (isPhaseComplete(workflow, phase, liveArtifacts)) return "complete"
      if (isPhaseInProgress(workflow, phase, liveArtifacts)) return "inProgress"
      return "notStarted"
    }
  }
  const deps = workflow?.dependencies?.[phase] || []
  let isLocked = false
  for (const dep of deps) {
    if (!satisfiedPhases || !satisfiedPhases[dep]) {
      isLocked = true
      break
    }
  }
  if (isLocked) return "locked"
  if (isPhaseComplete(workflow, phase, liveArtifacts)) return "complete"
  if (isPhaseInProgress(workflow, phase, liveArtifacts)) return "inProgress"
  return "notStarted"
}

/**
 * Normalizes a file path or title for loose matching by stripping extension, whitespace, and separators.
 */
export function normalizeArtifactName(str: string): string {
  if (!str) return ""
  return str
    .toLowerCase()
    .replace(/\\/g, "/")
    .replace(/^\.?\//, "")
    .replace(/\.md$/, "")
    .replace(/[-_\s/]+/g, "")
}

/**
 * Checks if a given filename is generic (like index.md or readme.md).
 */
export function isGenericFilename(filename: string): boolean {
  const lower = (filename || "").toLowerCase()
  return lower === "index.md" || lower === "readme.md" || lower === "index" || lower === "readme"
}

/**
 * Determines whether an Artifact matches a given file path from a file-write event.
 */
export function isArtifactMatchingFilePath(
  artifact: { title?: string; path?: string; id?: string; artifactType?: string },
  filePath: string
): boolean {
  if (!artifact || !filePath) return false

  const cleanFile = filePath.trim().replace(/\\/g, "/").replace(/^\.?\//, "").toLowerCase()
  const cleanArtPath = (artifact.path || "").trim().replace(/\\/g, "/").replace(/^\.?\//, "").toLowerCase()
  const cleanTitle = (artifact.title || "").trim().toLowerCase()
  const cleanId = (artifact.id || "").trim().toLowerCase()

  if (!cleanFile) return false

  // 1. Direct path equality or suffix match
  if (cleanArtPath) {
    if (cleanArtPath === cleanFile) return true
    if (cleanArtPath.endsWith("/" + cleanFile) || cleanFile.endsWith("/" + cleanArtPath)) return true
    if (cleanArtPath.endsWith(cleanFile) || cleanFile.endsWith(cleanArtPath)) return true
  }

  // 2. Basename / filename matching
  const fileBaseName = cleanFile.split("/").pop() || ""
  const artBaseName = cleanArtPath.split("/").pop() || ""
  const titleBaseName = cleanTitle.split("/").pop() || ""

  const isGeneric = isGenericFilename(fileBaseName)

  if (fileBaseName && !isGeneric) {
    if (artBaseName && fileBaseName === artBaseName) return true
    if (cleanTitle === fileBaseName || titleBaseName === fileBaseName) return true
    if (cleanTitle.endsWith("/" + fileBaseName) || fileBaseName.endsWith("/" + cleanTitle)) return true
    if (cleanId === fileBaseName || cleanId.endsWith("_" + fileBaseName) || cleanId.endsWith("-" + fileBaseName)) return true

    // Normalized name matching (e.g. "PRD" <-> "prd.md", "UX Design" <-> "ux-design.md", "Architecture" <-> "architecture.md")
    const normFileBase = normalizeArtifactName(fileBaseName)
    const normTitle = normalizeArtifactName(cleanTitle)
    const normArtBase = normalizeArtifactName(artBaseName)

    if (normFileBase && (normTitle === normFileBase || normArtBase === normFileBase)) {
      return true
    }
  }

  // 3. Title matching
  if (cleanTitle && cleanFile) {
    if (cleanTitle === cleanFile) return true
    if (cleanFile.endsWith("/" + cleanTitle) || cleanFile.endsWith(cleanTitle)) return true
  }

  // 4. Artifact type matching if applicable
  const fileArtType = determineArtifactType(filePath)
  if (fileArtType) {
    const artType = artifact.artifactType || determineArtifactType(artifact.path || artifact.title || "")
    if (artType === fileArtType) {
      if (fileBaseName && (artBaseName === fileBaseName || cleanTitle.includes(fileBaseName.replace(/\.md$/, "")))) {
        return true
      }
    }
  }

  return false
}

/**
 * Checks if the currently active preview document matches the updated file path.
 */
export function isPreviewDocumentMatching(
  activeArtifactTitle: string | null,
  liveArtifacts: Artifact[],
  filePath: string
): boolean {
  if (!activeArtifactTitle || !filePath) return false

  // 1. Check matching artifact in liveArtifacts list
  const matchedArtifact = liveArtifacts.find(
    (a) => a.title === activeArtifactTitle || a.id === activeArtifactTitle || a.path === activeArtifactTitle
  )
  if (matchedArtifact && isArtifactMatchingFilePath(matchedArtifact, filePath)) {
    return true
  }

  // 2. Check all artifacts in liveArtifacts to see if any artifact matching filePath also corresponds to activeArtifactTitle
  for (const art of liveArtifacts) {
    if (
      isArtifactMatchingFilePath(art, filePath) &&
      (art.title === activeArtifactTitle || art.id === activeArtifactTitle || art.path === activeArtifactTitle)
    ) {
      return true
    }
  }

  // 3. Fallback: check directly against activeArtifactTitle string
  return isArtifactMatchingFilePath({ title: activeArtifactTitle, path: activeArtifactTitle }, filePath)
}
