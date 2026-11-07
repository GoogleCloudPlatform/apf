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
import { Trash2, AlertCircle, Loader2 } from "lucide-react"
import { cn } from "@/lib/utils"
import { Artifact } from "@/lib/types"
import { FileIcon } from "./FileIcon"
import { Node } from "./FileExplorer"

export interface FilteredFileItem {
  name: string
  path?: string
  status?: string
  artifactId?: string
}

export interface FilteredFileListingProps {
  artifacts?: Artifact[]
  fileTree?: Node
  activePhase?: string
  onFileSelect?: (file: { name: string; path?: string }) => void
  selectedFilePath?: string | null
  onDeleteArtifact?: (artifactId: string) => Promise<unknown> | void
  onBatchDelete?: (items: FilteredFileItem[]) => Promise<unknown> | void
  onStageForSync?: (paths: string[]) => void
  emptyMessage?: string
  checkedPaths?: Set<string>
  onToggleCheck?: (path: string, item: FilteredFileItem) => void
  onToggleSelectAll?: () => void
}

/**
 * Maps artifact status to a clean colored indicator dot class and title.
 * Status colors:
 * - Complete: Emerald dot
 * - In Progress: Amber dot
 * - Staged for Review: Blue dot
 * - Not Started / other: Slate dot
 */
export function getArtifactStatusDot(status?: string): { dotClass: string; title: string } {
  const norm = (status || "").trim().toLowerCase()
  switch (norm) {
    case "complete":
    case "completed":
    case "approved":
      return { dotClass: "bg-emerald-500", title: "Complete" }
    case "in progress":
    case "in_progress":
    case "running":
      return { dotClass: "bg-amber-500", title: "In Progress" }
    case "staged for review":
    case "staged_for_review":
    case "in review":
    case "review":
      return { dotClass: "bg-blue-500", title: "Staged for Review" }
    case "not started":
    case "not_started":
    default:
      return { dotClass: "bg-slate-400 dark:bg-slate-500", title: status || "Not Started" }
  }
}

/**
 * Recursively extracts all leaf files from a tree node
 */
export function extractFiles(node?: Node): FilteredFileItem[] {
  if (!node) return []
  const files: FilteredFileItem[] = []

  function traverse(current: Node) {
    if (current.type === "file") {
      files.push({
        name: current.name,
        path: current.path,
      })
    }
    if (current.children) {
      current.children.forEach(traverse)
    }
  }

  traverse(node)
  return files
}

export function FilteredFileListing({
  artifacts,
  fileTree,
  activePhase,
  onFileSelect,
  selectedFilePath,
  onDeleteArtifact,
  onBatchDelete,
  onStageForSync,
  emptyMessage,
  checkedPaths: controlledCheckedPaths,
  onToggleCheck: controlledOnToggleCheck,
  onToggleSelectAll: controlledOnToggleSelectAll,
}: FilteredFileListingProps) {
  const [internalCheckedPaths, setInternalCheckedPaths] = useState<Set<string>>(new Set())
  const checkedPaths = controlledCheckedPaths ?? internalCheckedPaths

  const [artifactToDelete, setArtifactToDelete] = useState<FilteredFileItem | null>(null)
  const [isBatchDeleteModalOpen, setIsBatchDeleteModalOpen] = useState(false)
  const [isDeleting, setIsDeleting] = useState(false)
  const [deleteError, setDeleteError] = useState<string | null>(null)
  const modalRef = useRef<HTMLDivElement>(null)

  const resolvedEmptyMessage =
    emptyMessage || (activePhase ? `No files tracked for ${activePhase} phase.` : "No files found.")

  // Escape key listener to close delete modals
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape" && !isDeleting) {
        if (artifactToDelete) {
          setArtifactToDelete(null)
          setDeleteError(null)
        }
        if (isBatchDeleteModalOpen) {
          setIsBatchDeleteModalOpen(false)
          setDeleteError(null)
        }
      }
    }
    window.addEventListener("keydown", handleKeyDown)
    return () => window.removeEventListener("keydown", handleKeyDown)
  }, [artifactToDelete, isBatchDeleteModalOpen, isDeleting])

  // Derive file items list
  let items: FilteredFileItem[] = []

  if (artifacts !== undefined) {
    items = artifacts.map((a) => {
      const normalizedPath = a.path ? a.path.replace(/\\/g, "/") : ""
      return {
        name: a.title || (normalizedPath ? normalizedPath.split("/").pop() : "") || "Untitled",
        path: a.path || a.title,
        status: a.status,
        artifactId: a.id,
      }
    })
  } else if (fileTree) {
    items = extractFiles(fileTree)
  }

  // Sort items alphabetically by name
  items.sort((a, b) => a.name.localeCompare(b.name, undefined, { sensitivity: "base" }))

  const allItemKeys = items.map((i) => i.path || i.name)
  const isAllSelected = items.length > 0 && allItemKeys.every((k) => checkedPaths.has(k))
  const isSomeSelected = checkedPaths.size > 0 && !isAllSelected

  const handleToggleCheck = (key: string, item: FilteredFileItem) => {
    if (controlledOnToggleCheck) {
      controlledOnToggleCheck(key, item)
    } else {
      setInternalCheckedPaths((prev) => {
        const next = new Set(prev)
        if (next.has(key)) {
          next.delete(key)
        } else {
          next.add(key)
        }
        return next
      })
    }
  }

  const handleToggleSelectAll = () => {
    if (controlledOnToggleSelectAll) {
      controlledOnToggleSelectAll()
    } else {
      setInternalCheckedPaths((prev) => {
        if (allItemKeys.every((k) => prev.has(k))) {
          const next = new Set(prev)
          allItemKeys.forEach((k) => next.delete(k))
          return next
        } else {
          const next = new Set(prev)
          allItemKeys.forEach((k) => next.add(k))
          return next
        }
      })
    }
  }

  const handleClearSelection = () => {
    if (controlledOnToggleSelectAll && isAllSelected) {
      controlledOnToggleSelectAll()
    } else {
      setInternalCheckedPaths(new Set())
    }
  }

  const selectedItems = items.filter((i) => checkedPaths.has(i.path || i.name))

  if (items.length === 0) {
    return (
      <div className="text-center py-12 text-xs text-muted-foreground/60 italic px-4">
        {resolvedEmptyMessage}
      </div>
    )
  }

  const handleConfirmDelete = async () => {
    if (!artifactToDelete || !onDeleteArtifact) return
    setIsDeleting(true)
    setDeleteError(null)
    try {
      const target = artifactToDelete.artifactId || artifactToDelete.path || artifactToDelete.name
      await onDeleteArtifact(target)

      // Close preview if currently previewed file matches deleted file
      if (selectedFilePath) {
        const normSelected = selectedFilePath.replace(/\\/g, "/").replace(/^\.?\//, "")
        const normPath = (artifactToDelete.path || "").replace(/\\/g, "/").replace(/^\.?\//, "")
        if (
          normSelected === normPath ||
          selectedFilePath === artifactToDelete.name ||
          selectedFilePath === artifactToDelete.artifactId ||
          (normPath && normSelected.endsWith("/" + normPath)) ||
          (normPath && normPath.endsWith("/" + normSelected))
        ) {
          onFileSelect?.({ name: "", path: "" })
        }
      }

      setArtifactToDelete(null)
    } catch (err: any) {
      setDeleteError(err.message || "Failed to delete file")
    } finally {
      setIsDeleting(false)
    }
  }

  const handleConfirmBatchDelete = async () => {
    if (selectedItems.length === 0) return
    setIsDeleting(true)
    setDeleteError(null)
    try {
      if (onBatchDelete) {
        await onBatchDelete(selectedItems)
      } else if (onDeleteArtifact) {
        for (const item of selectedItems) {
          const target = item.artifactId || item.path || item.name
          if (target) {
            await onDeleteArtifact(target)
          }
        }
      }

      // Close preview if currently previewed file is among deleted batch items
      if (selectedFilePath) {
        const normSelected = selectedFilePath.replace(/\\/g, "/").replace(/^\.?\//, "")
        const isPreviewedItemDeleted = selectedItems.some((item) => {
          const normPath = (item.path || "").replace(/\\/g, "/").replace(/^\.?\//, "")
          return (
            normSelected === normPath ||
            selectedFilePath === item.name ||
            selectedFilePath === item.artifactId ||
            (normPath && normSelected.endsWith("/" + normPath)) ||
            (normPath && normPath.endsWith("/" + normSelected))
          )
        })
        if (isPreviewedItemDeleted) {
          onFileSelect?.({ name: "", path: "" })
        }
      }

      setIsBatchDeleteModalOpen(false)
      handleClearSelection()
    } catch (err: any) {
      setDeleteError(err.message || "Failed to delete selected artifacts")
    } finally {
      setIsDeleting(false)
    }
  }

  return (
    <>
      <div
        className="space-y-0.5 flex-1 min-h-0 overflow-y-auto"
        role="list"
        aria-label="Filtered Files"
      >
        {items.map((file) => {
          const itemKey = file.path || file.name
          const isChecked = checkedPaths.has(itemKey)
          const isSelected = Boolean(
            selectedFilePath &&
              (file.path === selectedFilePath ||
                file.name === selectedFilePath ||
                (file.path &&
                  selectedFilePath.replace(/\\/g, "/").replace(/^\.?\//, "") ===
                    file.path.replace(/\\/g, "/").replace(/^\.?\//, "")))
          )

          const statusDot = file.status ? getArtifactStatusDot(file.status) : null

          // Directory path subtitle
          const normalizedPath = (file.path || "").replace(/\\/g, "/").replace(/^\.?\//, "")
          const lastSlash = normalizedPath.lastIndexOf("/")
          const dirSubtitle = lastSlash > 0 ? normalizedPath.substring(0, lastSlash) : ""

          const handleItemClick = () => {
            onFileSelect?.({ name: file.name, path: file.path })
          }

          const handleKeyDown = (e: React.KeyboardEvent) => {
            if (e.key === "Enter" || e.key === " ") {
              e.preventDefault()
              handleItemClick()
            }
          }

          return (
            <div
              key={file.artifactId || file.path || file.name}
              role="listitem"
              tabIndex={0}
              aria-selected={isSelected}
              onClick={handleItemClick}
              onKeyDown={handleKeyDown}
              className={cn(
                "w-full flex items-center justify-between py-1 px-1.5 rounded text-xs transition-colors text-left focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring cursor-pointer select-none group/item",
                isSelected
                  ? "bg-primary/10 text-primary font-semibold"
                  : isChecked
                  ? "bg-muted/60 text-foreground"
                  : "text-foreground/80 hover:bg-muted/40 hover:text-foreground"
              )}
              title={file.path || file.name}
            >
              <div className="flex items-center gap-1.5 min-w-0 flex-1">
                {/* Multi-Select Checkbox */}
                <input
                  type="checkbox"
                  checked={isChecked}
                  onChange={(e) => {
                    e.stopPropagation()
                    handleToggleCheck(itemKey, file)
                  }}
                  onClick={(e) => {
                    e.stopPropagation()
                  }}
                  aria-label={`Select ${file.name}`}
                  className="h-3.5 w-3.5 rounded border-border text-primary focus:ring-0 focus:ring-offset-0 focus-visible:ring-0 focus:outline-none focus-visible:outline-none outline-none ring-0 cursor-pointer shrink-0"
                />

                <FileIcon filename={file.name} className="h-4 w-4 shrink-0" />
                <span
                  className={cn(
                    "truncate font-medium",
                    dirSubtitle ? "max-w-[75%] shrink-0" : ""
                  )}
                >
                  {file.name}
                </span>
                {dirSubtitle && (
                  <span className="truncate text-[10px] text-muted-foreground/60 font-normal ml-1 min-w-0 flex-1">
                    {dirSubtitle}
                  </span>
                )}
              </div>

              <div className="flex items-center gap-1 shrink-0 ml-2">
                {/* Colored Status Dot Indicator */}
                {statusDot && (
                  <span
                    className={cn("h-2 w-2 rounded-full shrink-0", statusDot.dotClass)}
                    title={`Status: ${statusDot.title}`}
                    aria-label={`Status: ${statusDot.title}`}
                  />
                )}

                {/* Delete button (hover/focus reveal) */}
                {onDeleteArtifact && (file.artifactId || file.path || file.name) && (
                  <button
                    type="button"
                    onClick={(e) => {
                      e.stopPropagation()
                      setDeleteError(null)
                      setArtifactToDelete(file)
                    }}
                    onKeyDown={(e) => {
                      e.stopPropagation()
                    }}
                    className="opacity-0 group-hover/item:opacity-100 focus-visible:opacity-100 p-0.5 rounded text-muted-foreground/60 hover:text-destructive hover:bg-destructive/10 transition-opacity"
                    title={`Delete ${file.name}`}
                    aria-label={`Delete ${file.name}`}
                  >
                    <Trash2 className="h-3 w-3" />
                  </button>
                )}
              </div>
            </div>
          )
        })}
      </div>

      {/* Single Artifact Delete Confirmation Modal */}
      {artifactToDelete && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-background/80 backdrop-blur-sm"
          role="dialog"
          aria-modal="true"
          aria-labelledby="delete-dialog-title"
          onClick={() => {
            if (!isDeleting) {
              setArtifactToDelete(null)
              setDeleteError(null)
            }
          }}
        >
          <div
            ref={modalRef}
            className="w-full max-w-sm rounded-lg border border-border bg-card p-4 shadow-lg space-y-4"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex items-center gap-2 text-destructive">
              <AlertCircle className="h-5 w-5 shrink-0" />
              <h3 id="delete-dialog-title" className="text-sm font-semibold">
                Delete Artifact
              </h3>
            </div>
            <p className="text-xs text-muted-foreground">
              Are you sure you want to delete{" "}
              <strong className="text-foreground">{artifactToDelete.name}</strong>? This action
              cannot be undone.
            </p>

            {deleteError && (
              <div className="p-2 rounded bg-destructive/10 text-destructive text-[11px]">
                {deleteError}
              </div>
            )}

            <div className="flex justify-end gap-2 pt-2">
              <button
                type="button"
                disabled={isDeleting}
                onClick={() => {
                  setArtifactToDelete(null)
                  setDeleteError(null)
                }}
                className="px-3 py-1.5 rounded text-xs text-muted-foreground hover:bg-muted font-medium transition-colors disabled:opacity-50"
              >
                Cancel
              </button>
              <button
                type="button"
                disabled={isDeleting}
                onClick={handleConfirmDelete}
                className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded text-xs bg-destructive text-destructive-foreground hover:bg-destructive/90 font-medium transition-colors disabled:opacity-50"
              >
                {isDeleting && <Loader2 className="h-3 w-3 animate-spin" />}
                Delete
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Batch Delete Confirmation Modal */}
      {isBatchDeleteModalOpen && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-background/80 backdrop-blur-sm"
          role="dialog"
          aria-modal="true"
          aria-labelledby="batch-delete-dialog-title"
          onClick={() => {
            if (!isDeleting) {
              setIsBatchDeleteModalOpen(false)
              setDeleteError(null)
            }
          }}
        >
          <div
            className="w-full max-w-sm rounded-lg border border-border bg-card p-4 shadow-lg space-y-4"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex items-center gap-2 text-destructive">
              <AlertCircle className="h-5 w-5 shrink-0" />
              <h3 id="batch-delete-dialog-title" className="text-sm font-semibold">
                Delete {selectedItems.length} {selectedItems.length === 1 ? "Artifact" : "Artifacts"}
              </h3>
            </div>
            <p className="text-xs text-muted-foreground">
              Are you sure you want to delete the following{" "}
              <strong className="text-foreground">{selectedItems.length}</strong> selected item(s)? This action cannot be undone.
            </p>

            <div className="max-h-36 overflow-y-auto space-y-1 p-2 rounded bg-muted/40 border border-border/50 text-xs">
              {selectedItems.map((item) => (
                <div key={item.artifactId || item.path || item.name} className="truncate text-foreground/80 flex items-center gap-1.5">
                  <FileIcon filename={item.name} className="h-3.5 w-3.5 shrink-0" />
                  <span className="truncate">{item.name}</span>
                </div>
              ))}
            </div>

            {deleteError && (
              <div className="p-2 rounded bg-destructive/10 text-destructive text-[11px]">
                {deleteError}
              </div>
            )}

            <div className="flex justify-end gap-2 pt-2">
              <button
                type="button"
                disabled={isDeleting}
                onClick={() => {
                  setIsBatchDeleteModalOpen(false)
                  setDeleteError(null)
                }}
                className="px-3 py-1.5 rounded text-xs text-muted-foreground hover:bg-muted font-medium transition-colors disabled:opacity-50"
              >
                Cancel
              </button>
              <button
                type="button"
                disabled={isDeleting}
                onClick={handleConfirmBatchDelete}
                className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded text-xs bg-destructive text-destructive-foreground hover:bg-destructive/90 font-medium transition-colors disabled:opacity-50"
              >
                {isDeleting && <Loader2 className="h-3 w-3 animate-spin" />}
                Delete ({selectedItems.length})
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  )
}
export default FilteredFileListing
