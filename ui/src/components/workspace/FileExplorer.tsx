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

import React, { useState, useEffect, useCallback, forwardRef, useImperativeHandle, useRef } from "react"
import { Folder, FolderOpen, ChevronDown, ChevronRight, Loader2, AlertCircle, RefreshCw, GitBranch, Trash2, X, Plus } from "lucide-react"
import { cn } from "@/lib/utils"
import { useParams } from "next/navigation"
import { useQuery } from "@tanstack/react-query"
import { api } from "@/lib/api"
import { Artifact } from "@/lib/types"
import { FilteredFileListing, FilteredFileItem, getArtifactStatusDot, extractFiles } from "./FilteredFileListing"
import { FileIcon } from "./FileIcon"
import { findLastVisibleFolder } from "@/lib/fileEventUtils"

export interface Node {
  name: string
  path: string
  type: "folder" | "file"
  hasChildren?: boolean
  children?: Node[]
  initializing?: boolean
}

export interface FileExplorerHandle {
  refreshFolderForPath: (filePath: string) => Promise<void>
  refreshAll: () => Promise<void>
}

interface FileNodeProps {
  node: Node
  depth: number
  projectId: string
  onFileSelect?: (file: { name: string; path?: string }) => void
  selectedFilePath?: string | null
  expandedPaths: Set<string>
  toggleFolder: (path: string) => void
  childrenCache: Record<string, Node[]>
  setCachedChildren: (path: string, children: Node[]) => void
  liveArtifacts?: Artifact[]
  onUploadToFolder?: (folderPath: string) => void
  onDeleteFolder?: (folder: { name: string; path: string }) => void
  onDeleteFile?: (file: { name: string; path: string; artifactId?: string }) => void
}

const FileNode: React.FC<FileNodeProps> = ({
  node,
  depth,
  projectId,
  onFileSelect,
  selectedFilePath,
  expandedPaths,
  toggleFolder,
  childrenCache,
  setCachedChildren,
  liveArtifacts,
  onUploadToFolder,
  onDeleteFolder,
  onDeleteFile,
}) => {
  const isFolder = node.type === "folder"
  const isOpen = isFolder && expandedPaths.has(node.path)
  const [isLoadingChildren, setIsLoadingChildren] = useState(false)
  const [fetchError, setFetchError] = useState<string | null>(null)

  const children = childrenCache[node.path] ?? node.children
  const isLoaded = Boolean(children !== undefined)

  const isSelected =
    !isFolder &&
    Boolean(
      selectedFilePath &&
        (node.path === selectedFilePath ||
          node.name === selectedFilePath ||
          (node.path &&
            selectedFilePath.replace(/\\/g, "/").replace(/^\.?\//, "") ===
              node.path.replace(/\\/g, "/").replace(/^\.?\//, "")))
    )

  // Check if this file leaf node matches a tracked Firestore artifact (strict path/title matching)
  const artifactMatch =
    !isFolder && liveArtifacts
      ? liveArtifacts.find((a) => {
          const artPath = (a.path || "").replace(/\\/g, "/").replace(/^\.?\//, "")
          const nodePath = (node.path || "").replace(/\\/g, "/").replace(/^\.?\//, "")
          return (
            artPath === nodePath ||
            (a.title === node.name && (!artPath || !artPath.includes("/")))
          )
        })
      : null

  const fetchChildren = useCallback(async () => {
    setIsLoadingChildren(true)
    setFetchError(null)
    try {
      const queryPath = encodeURIComponent(node.path)
      const res = await api.get(`/api/v1/projects/${projectId}/files?path=${queryPath}`)
      if (!res.ok) {
        const errText = await res.text()
        throw new Error(`Failed to load directory: ${res.status} ${errText}`)
      }
      const data: Node = await res.json()
      setCachedChildren(node.path, data.children || [])
    } catch (err: any) {
      setFetchError(err.message || "Failed to load subfolder")
    } finally {
      setIsLoadingChildren(false)
    }
  }, [node.path, projectId, setCachedChildren])

  // Automatically fetch children if folder is open but not yet loaded
  useEffect(() => {
    if (isFolder && isOpen && !isLoaded && !isLoadingChildren) {
      fetchChildren()
    }
  }, [isFolder, isOpen, isLoaded, isLoadingChildren, fetchChildren])

  const handleToggle = async () => {
    if (!isFolder) return
    toggleFolder(node.path)
    if (!isOpen && !isLoaded && !isLoadingChildren) {
      await fetchChildren()
    }
  }

  const handleClick = (e: React.MouseEvent) => {
    e.stopPropagation()
    if (isFolder) {
      handleToggle()
    } else {
      onFileSelect?.({ name: node.name, path: node.path })
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" || e.key === " ") {
      e.preventDefault()
      if (isFolder) {
        handleToggle()
      } else {
        onFileSelect?.({ name: node.name, path: node.path })
      }
    }
  }

  const handleRefreshSubfolder = async (e: React.MouseEvent) => {
    e.stopPropagation()
    if (!isFolder || isLoadingChildren) return
    if (!isOpen) {
      toggleFolder(node.path)
    }
    await fetchChildren()
  }

  return (
    <div className="select-none" role="none">
      <div
        style={{ paddingLeft: `${depth * 12}px` }}
        className={cn(
          "w-full flex items-center justify-between py-1 px-2 rounded text-xs transition-colors text-left focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring cursor-pointer group/row",
          isFolder
            ? "text-foreground/80 hover:bg-muted/50 hover:text-foreground"
            : isSelected
            ? "bg-primary/10 text-primary font-semibold"
            : "text-foreground/80 hover:bg-muted/40 hover:text-foreground"
        )}
        onClick={handleClick}
        onKeyDown={handleKeyDown}
        tabIndex={0}
        role="treeitem"
        aria-expanded={isFolder ? isOpen : undefined}
        aria-selected={isSelected}
      >
        <div className="flex items-center gap-1.5 min-w-0 flex-1">
          {isFolder ? (
            <>
              <span className="text-muted-foreground/70 shrink-0">
                {isLoadingChildren ? (
                  <Loader2 className="h-3.5 w-3.5 animate-spin text-primary" />
                ) : isOpen ? (
                  <ChevronDown className="h-3.5 w-3.5" />
                ) : (
                  <ChevronRight className="h-3.5 w-3.5" />
                )}
              </span>
              <span className="shrink-0">
                {isOpen ? (
                  <FolderOpen className="h-4 w-4 text-[#B8860B] dark:text-[#E8B86D] fill-[#C89F5D]/20 dark:fill-[#DCB67A]/20" />
                ) : (
                  <Folder className="h-4 w-4 text-[#C89F5D] dark:text-[#DCB67A] fill-[#C89F5D]/15 dark:fill-[#DCB67A]/15" />
                )}
              </span>
            </>
          ) : (
            <>
              <span className="w-3.5 shrink-0" />
              <FileIcon filename={node.name} className="h-4 w-4 shrink-0" />
            </>
          )}
          <span className={cn("truncate", isFolder ? "font-medium" : "")}>{node.name}</span>
        </div>

        <div className="flex items-center gap-1 shrink-0 ml-1">
          {artifactMatch && (
            <span
              className={cn(
                "h-2 w-2 rounded-full shrink-0",
                getArtifactStatusDot(artifactMatch.status).dotClass
              )}
              title={`Status: ${artifactMatch.status || "Not Started"}`}
              aria-label={`Status: ${artifactMatch.status || "Not Started"}`}
            />
          )}

          {isFolder ? (
            <div className="flex items-center gap-0.5 opacity-0 group-hover/row:opacity-100 group-focus-within/row:opacity-100 focus-within:opacity-100 transition-opacity">
              <button
                type="button"
                onClick={(e) => {
                  e.stopPropagation()
                  onUploadToFolder?.(node.path)
                }}
                className="p-0.5 rounded hover:bg-muted text-muted-foreground hover:text-foreground transition-colors shrink-0"
                title={`Upload to ${node.name}`}
                aria-label={`Upload to ${node.name}`}
              >
                <Plus className="h-3 w-3" />
              </button>
              <button
                type="button"
                onClick={handleRefreshSubfolder}
                disabled={isLoadingChildren}
                className="p-0.5 rounded hover:bg-muted text-muted-foreground hover:text-foreground transition-colors shrink-0"
                title={`Refresh ${node.name}`}
                aria-label={`Refresh ${node.name}`}
              >
                <RefreshCw className={cn("h-3 w-3", isLoadingChildren && "animate-spin")} />
              </button>
              <button
                type="button"
                onClick={(e) => {
                  e.stopPropagation()
                  onDeleteFolder?.({ name: node.name, path: node.path })
                }}
                className="p-0.5 rounded hover:bg-destructive/10 text-muted-foreground hover:text-destructive transition-colors shrink-0"
                title={`Delete ${node.name}`}
                aria-label={`Delete ${node.name}`}
              >
                <Trash2 className="h-3 w-3" />
              </button>
            </div>
          ) : (
            <div className="flex items-center gap-0.5 opacity-0 group-hover/row:opacity-100 group-focus-within/row:opacity-100 focus-within:opacity-100 transition-opacity">
              <button
                type="button"
                onClick={(e) => {
                  e.stopPropagation()
                  onDeleteFile?.({
                    name: node.name,
                    path: node.path,
                    artifactId: artifactMatch?.id,
                  })
                }}
                className="p-0.5 rounded hover:bg-destructive/10 text-muted-foreground hover:text-destructive transition-colors shrink-0"
                title={`Delete ${node.name}`}
                aria-label={`Delete ${node.name}`}
              >
                <Trash2 className="h-3 w-3" />
              </button>
            </div>
          )}
        </div>
      </div>

      {isFolder && isOpen && (
        <div className="mt-0.5" role="group">
          {fetchError && (
            <div className="pl-6 py-1 text-[10px] text-destructive flex items-center gap-1">
              <AlertCircle className="h-3 w-3 shrink-0" />
              <span>{fetchError}</span>
            </div>
          )}
          {children && children.length > 0 ? (
            children.map((child) => (
              <FileNode
                key={child.path || child.name}
                node={child}
                depth={depth + 1}
                projectId={projectId}
                onFileSelect={onFileSelect}
                selectedFilePath={selectedFilePath}
                expandedPaths={expandedPaths}
                toggleFolder={toggleFolder}
                childrenCache={childrenCache}
                setCachedChildren={setCachedChildren}
                liveArtifacts={liveArtifacts}
                onUploadToFolder={onUploadToFolder}
                onDeleteFolder={onDeleteFolder}
                onDeleteFile={onDeleteFile}
              />
            ))
          ) : isLoaded && !fetchError ? (
            <div className="pl-8 py-0.5 text-[10px] text-muted-foreground/60 italic">
              Empty directory
            </div>
          ) : null}
        </div>
      )}
    </div>
  )
}

export interface FileExplorerProps {
  onFileSelect?: (file: { name: string; path?: string }) => void
  selectedFilePath?: string | null
  activePhase?: string | null
  liveArtifacts?: Artifact[]
  onDeleteArtifact?: (artifactId: string) => Promise<unknown> | void
  onBatchDelete?: (items: FilteredFileItem[]) => Promise<unknown> | void
  onStageForSync?: (paths: string[]) => void
  onRefreshArtifacts?: () => Promise<unknown> | void
  isLoadingArtifacts?: boolean
  checkedPaths?: Set<string>
  onCheckedPathsChange?: (paths: Set<string>) => void
  showAllFiles?: boolean
  onShowAllFilesChange?: (showAll: boolean) => void
}

export const FileExplorer = forwardRef<FileExplorerHandle, FileExplorerProps>(function FileExplorer({
  onFileSelect,
  selectedFilePath,
  activePhase,
  liveArtifacts,
  onDeleteArtifact,
  onBatchDelete,
  onStageForSync,
  onRefreshArtifacts,
  isLoadingArtifacts = false,
  checkedPaths: controlledCheckedPaths,
  onCheckedPathsChange,
  showAllFiles: controlledShowAllFiles,
  onShowAllFilesChange,
}: FileExplorerProps = {}, ref) {
  const params = useParams()
  const projectId = params?.id as string

  const [internalShowAllFiles, setInternalShowAllFiles] = useState(false)
  const showAllFiles = controlledShowAllFiles ?? internalShowAllFiles
  const setShowAllFiles = useCallback((val: boolean | ((prev: boolean) => boolean)) => {
    if (onShowAllFilesChange) {
      const next = typeof val === "function" ? val(showAllFiles) : val
      onShowAllFilesChange(next)
    } else {
      setInternalShowAllFiles(val)
    }
  }, [showAllFiles, onShowAllFilesChange])

  const [expandedPaths, setExpandedPaths] = useState<Set<string>>(new Set())
  const [childrenCache, setChildrenCache] = useState<Record<string, Node[]>>({})

  const [internalCheckedPaths, setInternalCheckedPaths] = useState<Set<string>>(new Set())
  const checkedPaths = controlledCheckedPaths ?? internalCheckedPaths

  const [isBatchDeleteModalOpen, setIsBatchDeleteModalOpen] = useState(false)
  const [isDeletingBatch, setIsDeletingBatch] = useState(false)
  const [batchDeleteError, setBatchDeleteError] = useState<string | null>(null)

  // Project Tree File Upload State & Handlers
  const fileInputRef = useRef<HTMLInputElement>(null)
  const [uploadTargetPath, setUploadTargetPath] = useState<string | null>(null)
  const [isUploading, setIsUploading] = useState(false)
  const [uploadError, setUploadError] = useState<string | null>(null)

  // Folder Delete State & Handlers
  const [folderToDelete, setFolderToDelete] = useState<{ name: string; path: string } | null>(null)
  const [isDeletingFolder, setIsDeletingFolder] = useState(false)
  const [deleteFolderError, setDeleteFolderError] = useState<string | null>(null)

  // File Delete State & Handlers
  const [fileToDelete, setFileToDelete] = useState<{ name: string; path: string; artifactId?: string } | null>(null)
  const [isDeletingFile, setIsDeletingFile] = useState(false)
  const [deleteFileError, setDeleteFileError] = useState<string | null>(null)

  useEffect(() => {
    setFolderToDelete(null)
    setDeleteFolderError(null)
    setFileToDelete(null)
    setDeleteFileError(null)
    setUploadError(null)
  }, [projectId])

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        if (folderToDelete && !isDeletingFolder) {
          setFolderToDelete(null)
          setDeleteFolderError(null)
        }
        if (fileToDelete && !isDeletingFile) {
          setFileToDelete(null)
          setDeleteFileError(null)
        }
      }
    }
    window.addEventListener("keydown", handleKeyDown)
    return () => window.removeEventListener("keydown", handleKeyDown)
  }, [folderToDelete, isDeletingFolder, fileToDelete, isDeletingFile])

  const handleUploadToRoot = () => {
    setUploadTargetPath("")
    setUploadError(null)
    fileInputRef.current?.click()
  }

  const handleUploadToFolder = (folderPath: string) => {
    setUploadTargetPath(folderPath)
    setUploadError(null)
    fileInputRef.current?.click()
  }

  const handleFileInputChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = e.target.files
    if (!files || files.length === 0 || !projectId) {
      if (fileInputRef.current) {
        fileInputRef.current.value = ""
      }
      return
    }

    const target = uploadTargetPath ?? ""
    setIsUploading(true)
    setUploadError(null)

    try {
      const formData = new FormData()
      for (let i = 0; i < files.length; i++) {
        formData.append("files", files[i])
      }

      const query = target ? `?targetPath=${encodeURIComponent(target)}` : ""
      const res = await api.post(`/api/v1/projects/${projectId}/files/upload${query}`, formData)
      if (!res.ok) {
        let errMessage = `Upload failed (${res.status})`
        try {
          const errData = await res.json()
          errMessage = errData.message || errData.error || errMessage
        } catch {
          const text = await res.text()
          if (text) errMessage = text
        }
        throw new Error(errMessage)
      }

      // Successful upload: refresh root or subtree
      if (target === "") {
        await refetchFiles()
      } else {
        // Expand target folder and refresh its subtree
        setExpandedPaths((prev) => new Set(prev).add(target))
        try {
          const queryPath = encodeURIComponent(target)
          const resSub = await api.get(`/api/v1/projects/${projectId}/files?path=${queryPath}`)
          if (resSub.ok) {
            const data: Node = await resSub.json()
            setCachedChildren(target, data.children || [])
          }
        } catch (err) {
          console.error("Failed to refresh target folder:", target, err)
        }
        await refetchFiles()
      }
    } catch (err: any) {
      setUploadError(err.message || "Failed to upload files")
    } finally {
      setIsUploading(false)
      if (fileInputRef.current) {
        fileInputRef.current.value = ""
      }
    }
  }

  const handleOpenDeleteFolderModal = (folder: { name: string; path: string }) => {
    setFolderToDelete(folder)
    setDeleteFolderError(null)
  }

  const handleConfirmDeleteFolder = async () => {
    if (!folderToDelete || !projectId) return
    setIsDeletingFolder(true)
    setDeleteFolderError(null)

    try {
      const queryPath = encodeURIComponent(folderToDelete.path)
      const res = await api.delete(`/api/v1/projects/${projectId}/folders?path=${queryPath}`)
      if (!res.ok) {
        let errMessage = `Failed to delete folder (${res.status})`
        try {
          const errData = await res.json()
          errMessage = errData.message || errData.error || errMessage
        } catch {
          const text = await res.text()
          if (text) errMessage = text
        }
        throw new Error(errMessage)
      }

      // Cleanup cache & expanded paths
      const deletedPath = folderToDelete.path
      setChildrenCache((prev) => {
        const next = { ...prev }
        delete next[deletedPath]
        Object.keys(next).forEach((k) => {
          if (k.startsWith(deletedPath + "/")) {
            delete next[k]
          }
        })
        return next
      })
      setExpandedPaths((prev) => {
        const next = new Set(prev)
        next.delete(deletedPath)
        Array.from(next).forEach((k) => {
          if (k.startsWith(deletedPath + "/")) {
            next.delete(k)
          }
        })
        return next
      })

      // Clean up checkedPaths if any items under deleted folder were checked
      updateCheckedPaths((prev) => {
        const next = new Set(prev)
        Array.from(next).forEach((p) => {
          if (p === deletedPath || p.startsWith(deletedPath + "/")) {
            next.delete(p)
          }
        })
        return next
      })

      // Close preview if currently previewed file is within deleted folder
      if (selectedFilePath) {
        const normSelected = selectedFilePath.replace(/\\/g, "/").replace(/^\.?\//, "")
        const normDeletedFolder = deletedPath.replace(/\\/g, "/").replace(/^\.?\//, "")
        if (
          normSelected === normDeletedFolder ||
          normSelected.startsWith(normDeletedFolder + "/")
        ) {
          onFileSelect?.({ name: "", path: "" })
        }
      }

      // If this was inside a parent folder, refresh the parent folder
      if (deletedPath.includes("/")) {
        const parentPath = deletedPath.substring(0, deletedPath.lastIndexOf("/"))
        try {
          const queryParent = encodeURIComponent(parentPath)
          const resParent = await api.get(`/api/v1/projects/${projectId}/files?path=${queryParent}`)
          if (resParent.ok) {
            const data: Node = await resParent.json()
            setCachedChildren(parentPath, data.children || [])
          }
        } catch (err) {
          console.error("Failed to refresh parent folder after delete:", parentPath, err)
        }
      }

      await refetchFiles()
      setFolderToDelete(null)
    } catch (err: any) {
      setDeleteFolderError(err.message || "Failed to delete folder")
    } finally {
      setIsDeletingFolder(false)
    }
  }

  const handleOpenDeleteFileModal = (file: { name: string; path: string; artifactId?: string }) => {
    setFileToDelete(file)
    setDeleteFileError(null)
  }

  const handleConfirmDeleteFile = async () => {
    if (!fileToDelete || !projectId) return
    setIsDeletingFile(true)
    setDeleteFileError(null)
    try {
      const targetIdOrPath = fileToDelete.artifactId || fileToDelete.path || fileToDelete.name
      if (onDeleteArtifact) {
        await onDeleteArtifact(targetIdOrPath)
      } else {
        const idOrPath = fileToDelete.artifactId || encodeURIComponent(fileToDelete.path)
        const res = await api.delete(`/api/v1/projects/${projectId}/artifacts/${idOrPath}`)
        if (!res.ok) {
          let errMsg = `Failed to delete file (Status ${res.status})`
          try {
            const data = await res.json()
            if (data.error || data.message) errMsg = data.error || data.message
          } catch (_) {}
          throw new Error(errMsg)
        }
      }

      // Clean up checkedPaths if the deleted file was checked
      const deletedPath = fileToDelete.path
      updateCheckedPaths((prev) => {
        const next = new Set(prev)
        next.delete(deletedPath)
        next.delete(fileToDelete.name)
        return next
      })

      // Close preview if currently previewed file matches deleted file
      if (selectedFilePath) {
        const normSelected = selectedFilePath.replace(/\\/g, "/").replace(/^\.?\//, "")
        const normDeleted = deletedPath.replace(/\\/g, "/").replace(/^\.?\//, "")
        if (
          normSelected === normDeleted ||
          selectedFilePath === fileToDelete.name ||
          selectedFilePath === fileToDelete.artifactId ||
          normSelected.endsWith("/" + normDeleted) ||
          normDeleted.endsWith("/" + normSelected)
        ) {
          onFileSelect?.({ name: "", path: "" })
        }
      }

      // If this file was inside a parent folder, refresh the parent folder
      if (deletedPath.includes("/")) {
        const parentPath = deletedPath.substring(0, deletedPath.lastIndexOf("/"))
        try {
          const queryParent = encodeURIComponent(parentPath)
          const resParent = await api.get(`/api/v1/projects/${projectId}/files?path=${queryParent}`)
          if (resParent.ok) {
            const data: Node = await resParent.json()
            setCachedChildren(parentPath, data.children || [])
          }
        } catch (err) {
          console.error("Failed to refresh parent folder after file delete:", parentPath, err)
        }
      }

      await Promise.all([
        refetchFiles(),
        onRefreshArtifacts ? onRefreshArtifacts() : Promise.resolve(),
      ])
      setFileToDelete(null)
    } catch (err: any) {
      setDeleteFileError(err.message || "Failed to delete file")
    } finally {
      setIsDeletingFile(false)
    }
  }

  const updateCheckedPaths = useCallback(
    (updater: (prev: Set<string>) => Set<string>) => {
      if (onCheckedPathsChange) {
        const next = updater(checkedPaths)
        onCheckedPathsChange(next)
      } else {
        setInternalCheckedPaths(updater)
      }
    },
    [checkedPaths, onCheckedPathsChange]
  )

  const phaseDisplayName = activePhase
    ? activePhase.charAt(0).toUpperCase() + activePhase.slice(1).toLowerCase()
    : ""

  const title = showAllFiles
    ? "All Files"
    : phaseDisplayName
    ? `${phaseDisplayName} Files`
    : "Files"

  // Automatically ensure parent folders of selected file are kept expanded in tree view
  useEffect(() => {
    if (!selectedFilePath) return
    const normalized = selectedFilePath.replace(/\\/g, "/").replace(/^\.?\//, "")
    const parts = normalized.split("/").filter(Boolean)
    if (parts.length > 1) {
      setExpandedPaths((prev) => {
        const next = new Set(prev)
        let acc = ""
        for (let i = 0; i < parts.length - 1; i++) {
          acc = acc ? `${acc}/${parts[i]}` : parts[i]
          next.add(acc)
        }
        return next
      })
    }
  }, [selectedFilePath])

  const toggleFolder = (path: string) => {
    setExpandedPaths((prev) => {
      const next = new Set(prev)
      if (next.has(path)) {
        next.delete(path)
      } else {
        next.add(path)
      }
      return next
    })
  }

  const setCachedChildren = (path: string, children: Node[]) => {
    setChildrenCache((prev) => ({
      ...prev,
      [path]: children,
    }))
  }

  const {
    data: fileTree,
    isLoading: isLoadingFiles,
    error,
    refetch: refetchFiles,
    isRefetching,
  } = useQuery<Node>({
    queryKey: ["project-files", projectId],
    queryFn: async () => {
      const res = await api.get(`/api/v1/projects/${projectId}/files`)
      if (res.status === 202) {
        const data: Node = await res.json()
        return { ...data, initializing: true }
      }
      if (!res.ok) {
        const errText = await res.text()
        throw new Error(`Failed to load files: ${res.status} ${errText}`)
      }
      return res.json()
    },
    enabled: !!projectId,
    staleTime: 0,
    refetchInterval: (query) => {
      return query.state.data?.initializing ? 2000 : false
    },
  })

  // Filter live artifacts matching active phase with trimmed case-insensitivity
  const normalizedActivePhase = (activePhase || "").trim().toUpperCase()
  const phaseArtifacts = (liveArtifacts || []).filter((a) => {
    const p = (a.phase || "").trim().toUpperCase()
    return p === normalizedActivePhase
  })

  const handleRefresh = async () => {
    // Clear subfolder cache on refresh to fetch fresh directory data
    setChildrenCache({})
    await Promise.all([
      refetchFiles(),
      onRefreshArtifacts ? onRefreshArtifacts() : Promise.resolve(),
    ])
  }

  useImperativeHandle(
    ref,
    () => ({
      refreshFolderForPath: async (filePath: string) => {
        const match = findLastVisibleFolder(filePath, fileTree, expandedPaths, childrenCache)
        if (match.isRoot) {
          await refetchFiles()
        } else {
          try {
            const queryPath = encodeURIComponent(match.lastVisibleFolder)
            const res = await api.get(`/api/v1/projects/${projectId}/files?path=${queryPath}`)
            if (res.ok) {
              const data: Node = await res.json()
              setChildrenCache((prev) => ({
                ...prev,
                [match.lastVisibleFolder]: data.children || [],
              }))
            }
          } catch (err) {
            console.error("Failed to refresh folder:", match.lastVisibleFolder, err)
          }

          // Expand the last visible folder and all its parent paths
          setExpandedPaths((prev) => {
            const next = new Set(prev)
            match.foldersToOpen.forEach((p) => next.add(p))
            return next
          })
        }
      },
      refreshAll: handleRefresh,
    }),
    [fileTree, expandedPaths, childrenCache, projectId, refetchFiles, handleRefresh]
  )

  const isRefreshing = isRefetching || isLoadingArtifacts

  const handleToggleCheck = useCallback(
    (path: string, _item?: FilteredFileItem) => {
      updateCheckedPaths((prev) => {
        const next = new Set(prev)
        if (next.has(path)) {
          next.delete(path)
        } else {
          next.add(path)
        }
        return next
      })
    },
    [updateCheckedPaths]
  )

  const handleClearSelection = useCallback(() => {
    updateCheckedPaths(() => new Set())
  }, [updateCheckedPaths])

  const currentSelectableKeys: string[] = React.useMemo(() => {
    if (showAllFiles) return []
    if (liveArtifacts !== undefined) {
      return phaseArtifacts
        .map((a) => a.path || a.title || a.name || "")
        .filter((k): k is string => Boolean(k))
    } else if (fileTree) {
      return extractFiles(fileTree)
        .map((f) => f.path || f.name || "")
        .filter((k): k is string => Boolean(k))
    }
    return []
  }, [showAllFiles, fileTree, liveArtifacts, phaseArtifacts])

  const isAllSelected =
    currentSelectableKeys.length > 0 &&
    currentSelectableKeys.every((k) => checkedPaths.has(k))

  const handleToggleSelectAll = useCallback(() => {
    updateCheckedPaths((prev) => {
      const next = new Set(prev)
      if (currentSelectableKeys.every((k) => next.has(k))) {
        currentSelectableKeys.forEach((k) => next.delete(k))
      } else {
        currentSelectableKeys.forEach((k) => next.add(k))
      }
      return next
    })
  }, [currentSelectableKeys, updateCheckedPaths])

  // Get matching items for batch delete
  const selectedItemsForBatchDelete = React.useMemo<FilteredFileItem[]>(() => {
    const selectedKeys = Array.from(checkedPaths)
    if (liveArtifacts && liveArtifacts.length > 0) {
      return selectedKeys.map((key) => {
        const artifact = liveArtifacts.find(
          (a) => a.path === key || a.title === key || (a.path && key.endsWith(a.path))
        )
        const name = artifact?.title || key.split("/").pop() || key
        return {
          name,
          path: key,
          artifactId: artifact?.id,
        }
      })
    }
    return selectedKeys.map((key) => ({
      name: key.split("/").pop() || key,
      path: key,
      artifactId: undefined,
    }))
  }, [checkedPaths, liveArtifacts])

  const handleConfirmBatchDelete = async () => {
    if (selectedItemsForBatchDelete.length === 0) return
    setIsDeletingBatch(true)
    setBatchDeleteError(null)
    try {
      if (onBatchDelete) {
        await onBatchDelete(selectedItemsForBatchDelete)
      } else if (onDeleteArtifact) {
        for (const item of selectedItemsForBatchDelete) {
          const target = item.artifactId || item.path || item.name
          if (target) {
            await onDeleteArtifact(target)
          }
        }
      }

      // Close preview if currently previewed file is among the batch-deleted items
      if (selectedFilePath) {
        const normSelected = selectedFilePath.replace(/\\/g, "/").replace(/^\.?\//, "")
        const isPreviewedItemDeleted = selectedItemsForBatchDelete.some((item) => {
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
      setBatchDeleteError(err.message || "Failed to delete selected files")
    } finally {
      setIsDeletingBatch(false)
    }
  }

  return (
    <div className="h-full flex flex-col">
      <div className="flex items-center justify-between px-1.5 py-1 border-b border-border/40 mb-1">
        <div className="flex items-center gap-1.5 min-w-0">
          {!showAllFiles && (
            <input
              type="checkbox"
              checked={isAllSelected}
              onChange={handleToggleSelectAll}
              aria-label="Select all files"
              title="Select all / Deselect all"
              className="h-3.5 w-3.5 rounded border-border text-primary focus:ring-0 focus:ring-offset-0 focus-visible:ring-0 focus:outline-none focus-visible:outline-none outline-none ring-0 cursor-pointer shrink-0"
            />
          )}
          <span className="text-[11px] font-semibold text-muted-foreground uppercase tracking-wider truncate">
            {title}
          </span>
        </div>
        <div className="flex items-center gap-1 shrink-0">
          {showAllFiles && (
            <button
              type="button"
              onClick={handleUploadToRoot}
              disabled={isLoadingFiles || isUploading}
              className="p-1 rounded hover:bg-muted text-muted-foreground hover:text-foreground transition-colors disabled:opacity-50"
              title="Upload to root"
              aria-label="Upload to root"
            >
              <Plus className="h-3.5 w-3.5" />
            </button>
          )}
          <button
            type="button"
            onClick={handleRefresh}
            disabled={isLoadingFiles || isRefreshing}
            className="p-1 rounded hover:bg-muted text-muted-foreground hover:text-foreground transition-colors disabled:opacity-50"
            title="Refresh files"
            aria-label="Refresh files"
          >
            <RefreshCw className={cn("h-3.5 w-3.5", isRefreshing && "animate-spin")} />
          </button>
        </div>
      </div>

      {uploadError && (
        <div className="mx-1.5 my-1 p-2 rounded bg-destructive/10 border border-destructive/20 text-destructive text-[11px] flex items-center justify-between gap-2">
          <span className="truncate">{uploadError}</span>
          <button
            type="button"
            onClick={() => setUploadError(null)}
            className="p-0.5 hover:bg-destructive/20 rounded shrink-0"
            title="Dismiss error"
            aria-label="Dismiss upload error"
          >
            <X className="h-3 w-3" />
          </button>
        </div>
      )}

      {/* Multi-Selection Action Bar (only in Phase-Filtered view) */}
      {!showAllFiles && checkedPaths.size > 0 && (
        <div className="flex items-center justify-between px-2 py-1.5 bg-primary/10 border border-primary/20 rounded-md text-xs mb-1.5 animate-in fade-in duration-150 select-none">
          <span className="text-[11px] font-semibold text-primary">
            {checkedPaths.size} {checkedPaths.size === 1 ? "file" : "files"} selected
          </span>
          <div className="flex items-center gap-1">
            {onStageForSync && (
              <button
                type="button"
                onClick={() => onStageForSync(Array.from(checkedPaths))}
                className="inline-flex items-center gap-1 text-[10px] font-medium bg-primary text-primary-foreground hover:bg-primary/90 px-2 py-0.5 rounded shadow-sm transition-colors"
                title="Stage selected files for Git synchronization"
              >
                <GitBranch className="h-3 w-3" />
                <span>Stage for Sync</span>
              </button>
            )}
            <button
              type="button"
              onClick={() => setIsBatchDeleteModalOpen(true)}
              className="p-1 rounded text-destructive hover:bg-destructive/10 transition-colors"
              title="Delete selected files"
              aria-label="Delete selected files"
            >
              <Trash2 className="h-3.5 w-3.5" />
            </button>
            <button
              type="button"
              onClick={handleClearSelection}
              className="p-1 rounded text-muted-foreground hover:text-foreground transition-colors"
              title="Clear selection"
              aria-label="Clear selection"
            >
              <X className="h-3.5 w-3.5" />
            </button>
          </div>
        </div>
      )}

      <div className="space-y-0.5 flex-1 min-h-0 overflow-y-auto">
        {/* Loading Skeleton for Tree View */}
        {showAllFiles && isLoadingFiles && !fileTree ? (
          <div className="flex flex-col items-center justify-center py-12 text-muted-foreground gap-2">
            <Loader2 className="h-5 w-5 animate-spin text-primary" />
            <span className="text-xs">Reading workspace directory...</span>
          </div>
        ) : null}

        {error && showAllFiles && (
          <div className="flex flex-col items-center gap-2 p-4 rounded-lg border border-destructive/20 bg-destructive/5 text-destructive py-8 justify-center text-center">
            <AlertCircle className="h-5 w-5 shrink-0" />
            <span className="text-xs font-medium">Failed to read directory</span>
            <span className="text-[10px] opacity-80">{error.message}</span>
          </div>
        )}

        {/* Tree View (Show All Files = true) */}
        {showAllFiles && !isLoadingFiles && !error && fileTree && (
          fileTree.initializing ? (
            <div className="flex flex-col items-center justify-center py-12 text-muted-foreground gap-2 text-center px-4">
              <Loader2 className="h-5 w-5 animate-spin text-primary" />
              <span className="text-xs font-medium">Initializing workspace repository...</span>
              <span className="text-[10px] opacity-70">Files will appear automatically once cloning completes.</span>
            </div>
          ) : (
            <div
              className="space-y-0.5 flex-1 min-h-0 overflow-y-auto"
              role="tree"
              aria-label="Project Files"
            >
              {fileTree.children && fileTree.children.length > 0 ? (
                fileTree.children.map((child) => (
                  <FileNode
                    key={child.path || child.name}
                    node={child}
                    depth={0}
                    projectId={projectId}
                    onFileSelect={onFileSelect}
                    selectedFilePath={selectedFilePath}
                    expandedPaths={expandedPaths}
                    toggleFolder={toggleFolder}
                    childrenCache={childrenCache}
                    setCachedChildren={setCachedChildren}
                    liveArtifacts={liveArtifacts}
                    onUploadToFolder={handleUploadToFolder}
                    onDeleteFolder={handleOpenDeleteFolderModal}
                    onDeleteFile={handleOpenDeleteFileModal}
                  />
                ))
              ) : (
                <div className="text-center py-12 text-xs text-muted-foreground/60 italic">
                  No files found in workspace.
                </div>
              )}
            </div>
          )
        )}

        {/* Phase Filtered View (Show All Files = false) */}
        {!showAllFiles && (
          <FilteredFileListing
            artifacts={liveArtifacts !== undefined ? phaseArtifacts : undefined}
            fileTree={liveArtifacts === undefined ? fileTree : undefined}
            activePhase={phaseDisplayName}
            onFileSelect={onFileSelect}
            selectedFilePath={selectedFilePath}
            onDeleteArtifact={onDeleteArtifact}
            onBatchDelete={onBatchDelete}
            onStageForSync={onStageForSync}
            checkedPaths={checkedPaths}
            onToggleCheck={handleToggleCheck}
            onToggleSelectAll={handleToggleSelectAll}
            emptyMessage={`No files tracked for ${phaseDisplayName || "current"} phase.`}
          />
        )}
      </div>

      {/* Batch Delete Confirmation Modal */}
      {isBatchDeleteModalOpen && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-background/80 backdrop-blur-sm"
          role="dialog"
          aria-modal="true"
          aria-labelledby="explorer-batch-delete-dialog-title"
          onClick={() => {
            if (!isDeletingBatch) {
              setIsBatchDeleteModalOpen(false)
              setBatchDeleteError(null)
            }
          }}
        >
          <div
            className="w-full max-w-sm rounded-lg border border-border bg-card p-4 shadow-lg space-y-4"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex items-center gap-2 text-destructive">
              <AlertCircle className="h-5 w-5 shrink-0" />
              <h3 id="explorer-batch-delete-dialog-title" className="text-sm font-semibold">
                Delete {selectedItemsForBatchDelete.length} {selectedItemsForBatchDelete.length === 1 ? "File" : "Files"}
              </h3>
            </div>
            <p className="text-xs text-muted-foreground">
              Are you sure you want to delete the following{" "}
              <strong className="text-foreground">{selectedItemsForBatchDelete.length}</strong> selected item(s)? This action cannot be undone.
            </p>

            <div className="max-h-36 overflow-y-auto space-y-1 p-2 rounded bg-muted/40 border border-border/50 text-xs">
              {selectedItemsForBatchDelete.map((item) => (
                <div key={item.path || item.name} className="truncate text-foreground/80 flex items-center gap-1.5">
                  <FileIcon filename={item.name} className="h-3.5 w-3.5 shrink-0" />
                  <span className="truncate">{item.name}</span>
                </div>
              ))}
            </div>

            {batchDeleteError && (
              <div className="p-2 rounded bg-destructive/10 text-destructive text-[11px]">
                {batchDeleteError}
              </div>
            )}

            <div className="flex justify-end gap-2 pt-2">
              <button
                type="button"
                disabled={isDeletingBatch}
                onClick={() => {
                  setIsBatchDeleteModalOpen(false)
                  setBatchDeleteError(null)
                }}
                className="px-3 py-1.5 rounded text-xs text-muted-foreground hover:bg-muted font-medium transition-colors disabled:opacity-50"
              >
                Cancel
              </button>
              <button
                type="button"
                disabled={isDeletingBatch}
                onClick={handleConfirmBatchDelete}
                className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded text-xs bg-destructive text-destructive-foreground hover:bg-destructive/90 font-medium transition-colors disabled:opacity-50"
              >
                {isDeletingBatch && <Loader2 className="h-3 w-3 animate-spin" />}
                Delete ({selectedItemsForBatchDelete.length})
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Folder Delete Confirmation Modal */}
      {folderToDelete && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-background/80 backdrop-blur-sm"
          role="dialog"
          aria-modal="true"
          aria-labelledby="explorer-folder-delete-dialog-title"
          onClick={() => {
            if (!isDeletingFolder) {
              setFolderToDelete(null)
              setDeleteFolderError(null)
            }
          }}
        >
          <div
            className="w-full max-w-sm rounded-lg border border-border bg-card p-4 shadow-lg space-y-4"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex items-center gap-2 text-destructive">
              <AlertCircle className="h-5 w-5 shrink-0" />
              <h3 id="explorer-folder-delete-dialog-title" className="text-sm font-semibold">
                Delete Folder
              </h3>
            </div>
            <p className="text-xs text-muted-foreground">
              Are you sure you want to delete the folder{" "}
              <strong className="text-foreground">"{folderToDelete.name}"</strong>
              {folderToDelete.path ? ` (${folderToDelete.path})` : ""} and all its contents? This action cannot be undone.
            </p>

            {deleteFolderError && (
              <div className="p-2 rounded bg-destructive/10 text-destructive text-[11px]">
                {deleteFolderError}
              </div>
            )}

            <div className="flex justify-end gap-2 pt-2">
              <button
                type="button"
                disabled={isDeletingFolder}
                onClick={() => {
                  setFolderToDelete(null)
                  setDeleteFolderError(null)
                }}
                className="px-3 py-1.5 rounded text-xs text-muted-foreground hover:bg-muted font-medium transition-colors disabled:opacity-50"
              >
                Cancel
              </button>
              <button
                type="button"
                disabled={isDeletingFolder}
                onClick={handleConfirmDeleteFolder}
                className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded text-xs bg-destructive text-destructive-foreground hover:bg-destructive/90 font-medium transition-colors disabled:opacity-50"
              >
                {isDeletingFolder && <Loader2 className="h-3 w-3 animate-spin" />}
                Delete Folder
              </button>
            </div>
          </div>
        </div>
      )}

      {/* File Delete Confirmation Modal */}
      {fileToDelete && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-background/80 backdrop-blur-sm"
          role="dialog"
          aria-modal="true"
          aria-labelledby="explorer-file-delete-dialog-title"
          onClick={() => {
            if (!isDeletingFile) {
              setFileToDelete(null)
              setDeleteFileError(null)
            }
          }}
        >
          <div
            className="w-full max-w-sm rounded-lg border border-border bg-card p-4 shadow-lg space-y-4"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex items-center gap-2 text-destructive">
              <AlertCircle className="h-5 w-5 shrink-0" />
              <h3 id="explorer-file-delete-dialog-title" className="text-sm font-semibold">
                Delete File
              </h3>
            </div>
            <p className="text-xs text-muted-foreground">
              Are you sure you want to delete the file{" "}
              <strong className="text-foreground">"{fileToDelete.name}"</strong>
              {fileToDelete.path && fileToDelete.path !== fileToDelete.name ? ` (${fileToDelete.path})` : ""}? This action cannot be undone.
            </p>

            {deleteFileError && (
              <div className="p-2 rounded bg-destructive/10 text-destructive text-[11px]">
                {deleteFileError}
              </div>
            )}

            <div className="flex justify-end gap-2 pt-2">
              <button
                type="button"
                disabled={isDeletingFile}
                onClick={() => {
                  setFileToDelete(null)
                  setDeleteFileError(null)
                }}
                className="px-3 py-1.5 rounded text-xs text-muted-foreground hover:bg-muted font-medium transition-colors disabled:opacity-50"
              >
                Cancel
              </button>
              <button
                type="button"
                disabled={isDeletingFile}
                onClick={handleConfirmDeleteFile}
                className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded text-xs bg-destructive text-destructive-foreground hover:bg-destructive/90 font-medium transition-colors disabled:opacity-50"
              >
                {isDeletingFile && <Loader2 className="h-3 w-3 animate-spin" />}
                Delete File
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Hidden File Input for Root & Folder Uploads */}
      <input
        type="file"
        multiple
        ref={fileInputRef}
        onChange={handleFileInputChange}
        className="hidden"
        aria-label="Upload files"
      />
    </div>
  )
})

export default FileExplorer
