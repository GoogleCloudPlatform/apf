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

import React, { useState } from "react"
import { Search, Edit3, Lock, AlertTriangle, CheckCircle, Folder, HardDrive, Filter } from "lucide-react"
import { NFSFolderMetadata } from "./EditQuotaModal"
import { formatBytes } from "./NFSVolumeStatsCard"

export interface OrganizationRef {
  id: string
  name: string
}

interface FolderMetadataTableProps {
  folders: NFSFolderMetadata[]
  isLoading: boolean
  onEditFolder: (folder: NFSFolderMetadata) => void
  organizations?: OrganizationRef[]
  getOrgName?: (id: string) => string
}

export default function FolderMetadataTable({
  folders,
  isLoading,
  onEditFolder,
  organizations,
  getOrgName,
}: FolderMetadataTableProps) {
  const [searchQuery, setSearchQuery] = useState("")
  const [statusFilter, setStatusFilter] = useState<string>("all")

  const resolveOrgName = (folder: NFSFolderMetadata) => {
    if (folder.organizationName) return folder.organizationName
    if (getOrgName) {
      const name = getOrgName(folder.organizationId)
      if (name) return name
    }
    if (organizations && organizations.length > 0) {
      const match = organizations.find((o) => o.id === folder.organizationId)
      if (match?.name) return match.name
    }
    return folder.organizationId
  }

  const filteredFolders = folders.filter((f) => {
    const orgName = resolveOrgName(f)
    const matchesSearch =
      f.organizationId?.toLowerCase().includes(searchQuery.toLowerCase()) ||
      orgName?.toLowerCase().includes(searchQuery.toLowerCase()) ||
      f.workspacePath?.toLowerCase().includes(searchQuery.toLowerCase()) ||
      f.folderId?.toLowerCase().includes(searchQuery.toLowerCase())

    if (!matchesSearch) return false

    const isExceeded = f.quotaSizeBytes > 0 && f.usedSizeBytes >= f.quotaSizeBytes
    if (statusFilter === "exceeded") return isExceeded
    if (statusFilter === "normal") return !isExceeded

    return true
  })

  return (
    <div className="bg-card border border-border rounded-2xl overflow-hidden shadow-sm">
      
      {/* Controls Bar */}
      <div className="p-4 md:p-6 border-b border-border flex flex-col md:flex-row md:items-center justify-between gap-4 bg-muted/20">
        <div className="relative flex-1 max-w-md">
          <Search className="absolute left-3.5 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <input
            type="text"
            placeholder="Search by Organization, ID, or path..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-full h-10 pl-10 pr-4 rounded-xl border border-input bg-background text-xs focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all text-foreground"
          />
        </div>

        {/* Status Filter */}
        <div className="flex items-center gap-2">
          <Filter className="h-4 w-4 text-muted-foreground shrink-0" />
          <select
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
            className="h-10 px-3 rounded-xl border border-input bg-background text-xs font-medium text-foreground focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all"
          >
            <option value="all">All Statuses</option>
            <option value="normal">Normal</option>
            <option value="exceeded">Quota Exceeded</option>
          </select>
        </div>
      </div>

      {/* Table */}
      {isLoading ? (
        <div className="flex flex-col items-center justify-center py-20 gap-3 text-muted-foreground">
          <HardDrive className="h-8 w-8 text-primary animate-pulse" />
          <p className="text-xs font-medium">Loading organization metadata...</p>
        </div>
      ) : filteredFolders.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-16 text-center px-6">
          <Folder className="h-10 w-10 text-muted-foreground/60 mb-3" />
          <h3 className="font-bold text-foreground text-sm">No organization workspaces found</h3>
          <p className="text-xs text-muted-foreground mt-1 max-w-sm">
            {searchQuery || statusFilter !== "all"
              ? "No organization workspaces matched your filter criteria."
              : "Run a storage audit to index organization workspace directories."}
          </p>
        </div>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-left border-collapse">
            <thead>
              <tr className="border-b border-border text-[10px] font-bold text-muted-foreground uppercase bg-muted/30">
                <th className="py-4 px-6">Organization</th>
                <th className="py-4 px-6">Workspace Path</th>
                <th className="py-4 px-6">Storage Usage / Quota</th>
                <th className="py-4 px-6 text-center">Status</th>
                <th className="py-4 px-6">Retention</th>
                <th className="py-4 px-6 text-right">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border/60 text-xs">
              {filteredFolders.map((folder) => {
                const used = folder.usedSizeBytes || 0
                const quota = folder.quotaSizeBytes || 1
                const percent = Math.min(Math.round((used / quota) * 100), 100)
                const isExceeded = quota > 0 && used >= quota
                const orgName = resolveOrgName(folder)

                return (
                  <tr key={folder.folderId} className="hover:bg-muted/20 transition-colors">
                    
                    {/* Organization (Name & ID) */}
                    <td className="py-4 px-6">
                      <div className="font-bold text-foreground truncate max-w-xs" title={orgName}>
                        {orgName}
                      </div>
                      <div className="text-[10px] text-muted-foreground/70 font-mono mt-0.5 truncate max-w-xs" title={folder.organizationId}>
                        {folder.organizationId}
                      </div>
                    </td>

                    {/* Path */}
                    <td className="py-4 px-6">
                      <div className="font-mono text-[11px] text-muted-foreground truncate max-w-xs" title={folder.workspacePath}>
                        {folder.workspacePath}
                      </div>
                      <div className="text-[10px] text-muted-foreground/70 mt-0.5">
                        {folder.fileCount} files
                      </div>
                    </td>

                    {/* Usage / Quota */}
                    <td className="py-4 px-6">
                      <div className="flex items-center justify-between text-[11px] font-semibold mb-1">
                        <span className="text-foreground">{formatBytes(used)}</span>
                        <span className="text-muted-foreground">of {formatBytes(quota)} ({percent}%)</span>
                      </div>
                      <div className="h-2 w-36 bg-muted/60 rounded-full overflow-hidden p-0.5 border border-border">
                        <div
                          className={`h-full rounded-full transition-all ${
                            isExceeded
                              ? "bg-destructive"
                              : percent > 80
                              ? "bg-amber-500"
                              : "bg-primary"
                          }`}
                          style={{ width: `${percent}%` }}
                        />
                      </div>
                    </td>

                    {/* Status */}
                    <td className="py-4 px-6 text-center">
                      {isExceeded ? (
                        <span className="inline-flex items-center gap-1 text-[10px] font-bold px-2.5 py-1 rounded-full bg-destructive/10 text-destructive border border-destructive/20">
                          <AlertTriangle className="h-3 w-3" /> Quota Exceeded
                        </span>
                      ) : (
                        <span className="inline-flex items-center gap-1 text-[10px] font-bold px-2.5 py-1 rounded-full bg-emerald-500/10 text-emerald-500 border border-emerald-500/20">
                          <CheckCircle className="h-3 w-3" /> Active
                        </span>
                      )}
                    </td>

                    {/* Retention */}
                    <td className="py-4 px-6 text-muted-foreground font-medium">
                      {folder.retentionPolicy || "90d"}
                    </td>

                    {/* Actions */}
                    <td className="py-4 px-6 text-right">
                      <button
                        onClick={() => onEditFolder(folder)}
                        className="h-8 px-3 rounded-lg border border-border bg-background hover:bg-muted text-muted-foreground hover:text-foreground flex items-center justify-center gap-1.5 ml-auto text-xs font-semibold transition-all"
                      >
                        <Edit3 className="h-3.5 w-3.5" /> Edit
                      </button>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
