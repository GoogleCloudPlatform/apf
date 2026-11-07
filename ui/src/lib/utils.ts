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

import { type ClassValue, clsx } from "clsx"
import { twMerge } from "tailwind-merge"

/**
 * Standard utility to combine Tailwind classnames and merge overrides gracefully.
 */
export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

/**
 * Formats ISO date strings into clean visual summaries: "Jan 04, 02:30 PM".
 */
export function formatSessionDate(dateString: string | Date): string {
  if (!dateString) return ""
  const dt = new Date(dateString)
  if (isNaN(dt.getTime())) return ""

  const months = [
    "Jan", "Feb", "Mar", "Apr", "May", "Jun",
    "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"
  ]
  const month = months[dt.getMonth()]
  const day = dt.getDate().toString().padStart(2, "0")
  
  let hour = dt.getHours()
  const ampm = hour >= 12 ? "PM" : "AM"
  hour = hour % 12
  hour = hour ? hour : 12 // the hour '0' should be '12'
  const hourStr = hour.toString().padStart(2, "0")
  
  const minute = dt.getMinutes().toString().padStart(2, "0")
  
  return `${month} ${day}, ${hourStr}:${minute} ${ampm}`
}

/**
 * Capitalizes a word or string segment.
 */
export function capitalize(str: string): string {
  if (!str) return ""
  return str.charAt(0).toUpperCase() + str.slice(1).toLowerCase()
}

/**
 * Formats byte size into readable string: "15.0 KB", "2.4 MB", "500 B".
 */
export function formatFileSize(bytes?: number | null): string {
  if (bytes === undefined || bytes === null || !Number.isFinite(bytes) || bytes <= 0) return "0 B"
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

const IMAGE_EXTENSIONS = new Set([
  "png", "jpg", "jpeg", "gif", "webp", "svg", "bmp", "ico", "avif"
])

/**
 * Detects whether a given filename, path, mimeType, or artifactType represents an image file.
 */
export function isImageFile(
  filenameOrPath?: string | null,
  mimeType?: string | null,
  artifactType?: string | null
): boolean {
  if (artifactType && artifactType.toLowerCase() === "image") {
    return true
  }
  if (mimeType && mimeType.toLowerCase().startsWith("image/")) {
    return true
  }
  if (filenameOrPath) {
    const cleanPath = filenameOrPath.split("?")[0].split("#")[0]
    const ext = cleanPath.split(".").pop()?.toLowerCase()
    if (ext && IMAGE_EXTENSIONS.has(ext)) {
      return true
    }
  }
  return false
}

export function getApiBaseUrl(): string {
  if (typeof window !== "undefined") {
    if (window.location.hostname === "localhost" || window.location.hostname === "127.0.0.1") {
      return "http://localhost:8081"
    }
  }
  return ""
}

const PYTHON_KEYWORDS = new Set([
  "False", "None", "True", "and", "as", "assert", "async", "await", "break",
  "class", "continue", "def", "del", "elif", "else", "except", "finally",
  "for", "from", "global", "if", "import", "in", "is", "lambda", "nonlocal",
  "not", "or", "pass", "raise", "return", "try", "while", "with", "yield"
])

export function isValidPythonIdentifier(name: string): boolean {
  if (!name) return false
  const pythonIdentifierRegex = /^[a-zA-Z_][a-zA-Z0-9_]*$/
  return pythonIdentifierRegex.test(name) && !PYTHON_KEYWORDS.has(name)
}

export const DEFAULT_SPEC_LOCATION = "docs/specs"
export const BMAD_PLANNING_ARTIFACTS_PATH = "plans"
export const BMAD_IMPLEMENTATION_ARTIFACTS_PATH = "implementation"

import type { Project } from "./types"

export function getSpecLocation(project?: Project | null): string {
  if (!project?.app_integrations) return DEFAULT_SPEC_LOCATION
  const githubIntegration = project.app_integrations.find(
    (i) => i.integration_id === "github" && i.enabled
  )
  return githubIntegration?.settings?.specLocation || DEFAULT_SPEC_LOCATION
}

