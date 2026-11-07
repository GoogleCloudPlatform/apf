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

import React, { useState, useEffect, useRef } from "react"
import { useRouter } from "next/navigation"
import { Search, Loader2, AlertCircle, ChevronDown, Check } from "lucide-react"
import { api } from "@/lib/api"
import AlertModal from "@/components/shared/AlertModal"
import { useOnboardingStore } from "@/stores/onboarding"

interface Repo {
  name: string
  url: string
  defaultBranch?: string
}

interface SearchableRepoDropdownProps {
  value: string
  onChange: (url: string) => void
  onSelectRepo?: (url: string, defaultBranch?: string) => void
  disabled?: boolean
  customToken?: string
  isOverridden?: boolean
}

export default function SearchableRepoDropdown({
  value,
  onChange,
  onSelectRepo,
  disabled = false,
  customToken,
  isOverridden,
}: SearchableRepoDropdownProps) {
  const router = useRouter()
  const [isOpen, setIsOpen] = useState(false)
  const [searchQuery, setSearchQuery] = useState("")
  const [debouncedQuery, setDebouncedQuery] = useState("")
  const [repos, setRepos] = useState<Repo[]>([])
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [showErrorModal, setShowErrorModal] = useState(false)
  const [modalErrorMessage, setModalErrorMessage] = useState("")
  const containerRef = useRef<HTMLDivElement>(null)

  // 300ms Debouncing search input
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedQuery(searchQuery)
    }, 300)
    return () => clearTimeout(timer)
  }, [searchQuery])

  // Fetch repos when debounced search query changes or when opening dropdown for the first time
  useEffect(() => {
    if (!isOpen) return

    // Do not search if query exists but has fewer than 3 characters
    if (debouncedQuery.length > 0 && debouncedQuery.length < 3) {
      setRepos([])
      return
    }

    const hasRealToken = (isOverridden ?? true) && customToken && !/[•·]/.test(customToken) && customToken.trim() !== ""

    async function fetchRepos() {
      setIsLoading(true)
      setError(null)
      try {
        const res = await api.get(
          `/api/v1/integrations/github/repos?q=${encodeURIComponent(debouncedQuery)}`,
          hasRealToken ? { headers: { "X-GitHub-Token": customToken } } : undefined
        )
        if (!res.ok) {
          const text = await res.text()
          const errMsg = text || "Failed to fetch repositories"
          setModalErrorMessage(errMsg)
          setShowErrorModal(true)
          setError(errMsg)
          return
        }
        const data = await res.json()
        setRepos(data || [])
      } catch (err: any) {
        console.error(err)
        setError(err.message || "Failed to load repositories")
      } finally {
        setIsLoading(false)
      }
    }

    fetchRepos()
  }, [debouncedQuery, isOpen, customToken, isOverridden])

  // Handle click outside dropdown to close it
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(event.target as Node)) {
        setIsOpen(false)
      }
    }
    document.addEventListener("mousedown", handleClickOutside)
    return () => document.removeEventListener("mousedown", handleClickOutside)
  }, [])

  const selectedRepo = repos.find((r) => r.url === value) || { name: value ? value.replace("https://github.com/", "") : "Select a Repository...", url: value }

  return (
    <div className="relative w-full" ref={containerRef}>
      <label className="text-[10px] font-bold text-muted-foreground uppercase tracking-wider block mb-1.5">
        GitHub Repository
      </label>
      <button
        type="button"
        id="tour-project-github-repo"
        disabled={disabled}
        onClick={() => setIsOpen(!isOpen)}
        className="flex h-9 w-full items-center justify-between rounded border border-input bg-card px-3 text-xs focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary disabled:opacity-50 disabled:bg-muted/30"
      >
        <span className="truncate text-foreground">
          {value ? selectedRepo.name : "Select a Repository..."}
        </span>
        <ChevronDown className="h-4 w-4 text-muted-foreground shrink-0" />
      </button>

      {isOpen && (
        <div
          id="tour-project-github-repo-dropdown"
          className="absolute z-50 mt-1 w-full rounded-lg border border-border bg-card p-2 shadow-lg max-h-64 overflow-y-auto space-y-2"
        >
          {/* Search query input field */}
          <div className="relative">
            <Search className="absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
            <input
              type="text"
              placeholder="Search repositories..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full h-8 pl-8 pr-3 rounded border border-input bg-muted/20 text-xs focus:outline-none focus:ring-1 focus:ring-primary focus:border-primary"
            />
          </div>

          {/* Loading status */}
          {isLoading && (
            <div className="flex items-center justify-center py-6 gap-2 text-muted-foreground text-xs">
              <Loader2 className="h-4 w-4 animate-spin text-primary" />
              Searching repos...
            </div>
          )}

          {/* Error warning message */}
          {error && (
            <div className="flex items-center gap-2 p-2 rounded text-xs text-destructive bg-destructive/5 border border-destructive/10 leading-normal">
              <AlertCircle className="h-4 w-4 shrink-0" />
              <div className="flex-1">{error}</div>
            </div>
          )}

          {/* Repos result matches list */}
          {!isLoading && !error && (
            <div className="space-y-0.5">
              {repos.length === 0 ? (
                <div className="text-center py-6 text-muted-foreground text-xs">
                  No repositories found
                </div>
              ) : (
                repos.map((repo) => {
                  const isSelected = repo.url === value
                  return (
                    <button
                      key={repo.url}
                      type="button"
                      onClick={() => {
                        onChange(repo.url)
                        if (onSelectRepo) {
                          onSelectRepo(repo.url, repo.defaultBranch)
                        }
                        setIsOpen(false)
                        useOnboardingStore.getState().advanceStep("catalog", 13)
                      }}
                      className="flex w-full items-center justify-between rounded px-2.5 py-1.5 text-left text-xs hover:bg-muted/50 text-foreground transition-colors"
                    >
                      <div className="flex flex-col min-w-0">
                        <span className="font-medium truncate">{repo.name}</span>
                        <span className="text-[10px] text-muted-foreground truncate">{repo.url}</span>
                      </div>
                      {isSelected && <Check className="h-4 w-4 text-primary shrink-0 ml-2" />}
                    </button>
                  )
                })
              )}
            </div>
          )}
        </div>
      )}

      {/* Reusable AlertModal Component */}
      {showErrorModal && (
        <AlertModal
          type="error"
          title="GitHub Authentication Failed"
          description="The repository search failed. Your Personal Access Token (PAT) may be invalid, expired, or missing required permissions."
          details={modalErrorMessage || undefined}
          onClose={() => setShowErrorModal(false)}
        />
      )}
    </div>
  )
}
