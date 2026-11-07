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

// @vitest-environment jsdom
import React from "react"
import { render, screen, cleanup, fireEvent } from "@testing-library/react"
import { describe, it, expect, vi, afterEach } from "vitest"
import FolderMetadataTable from "./FolderMetadataTable"
import { NFSFolderMetadata } from "./EditQuotaModal"

describe("FolderMetadataTable", () => {
  afterEach(() => {
    cleanup()
  })

  const mockFolders: NFSFolderMetadata[] = [
    {
      folderId: "folder-123",
      organizationId: "org-alpha-id",
      organizationName: "Alpha Corporation",
      projectId: "proj-1",
      workspacePath: "/nfs/data/tenants/org-alpha/workspace",
      quotaSizeBytes: 10 * 1024 * 1024 * 1024,
      usedSizeBytes: 2 * 1024 * 1024 * 1024,
      fileCount: 42,
      retentionPolicy: "90d",
    },
    {
      folderId: "folder-456",
      organizationId: "org-beta-id",
      projectId: "proj-2",
      workspacePath: "/nfs/data/tenants/org-beta/workspace",
      quotaSizeBytes: 5 * 1024 * 1024 * 1024,
      usedSizeBytes: 5 * 1024 * 1024 * 1024,
      fileCount: 150,
      retentionPolicy: "30d",
    },
  ]

  const mockOrganizations = [
    { id: "org-alpha-id", name: "Alpha Corporation" },
    { id: "org-beta-id", name: "Beta Logistics" },
  ]

  it("renders the table column name as 'Organization' instead of 'Organization ID'", () => {
    render(
      <FolderMetadataTable
        folders={mockFolders}
        isLoading={false}
        onEditFolder={vi.fn()}
        organizations={mockOrganizations}
      />
    )

    // Verify header column name
    const orgHeader = screen.getByRole("columnheader", { name: /^Organization$/i })
    expect(orgHeader).not.toBeNull()
    expect(screen.queryByRole("columnheader", { name: /^Organization ID$/i })).toBeNull()
  })

  it("renders the organization column content split into 2 rows: top row with Organization Name and bottom row with Organization ID in smaller text", () => {
    render(
      <FolderMetadataTable
        folders={mockFolders}
        isLoading={false}
        onEditFolder={vi.fn()}
        organizations={mockOrganizations}
      />
    )

    // For folder 1 (Alpha Corporation & org-alpha-id)
    expect(screen.getByText("Alpha Corporation")).not.toBeNull()
    expect(screen.getByText("org-alpha-id")).not.toBeNull()

    // For folder 2 (Beta Logistics resolved via organizations list & org-beta-id)
    expect(screen.getByText("Beta Logistics")).not.toBeNull()
    expect(screen.getByText("org-beta-id")).not.toBeNull()
  })

  it("filters folders by organization name as well as organization ID", () => {
    render(
      <FolderMetadataTable
        folders={mockFolders}
        isLoading={false}
        onEditFolder={vi.fn()}
        organizations={mockOrganizations}
      />
    )

    const searchInput = screen.getByPlaceholderText(/Search by Organization/i)

    // Search by Organization Name
    fireEvent.change(searchInput, { target: { value: "Beta Logistics" } })
    expect(screen.getByText("Beta Logistics")).not.toBeNull()
    expect(screen.queryByText("Alpha Corporation")).toBeNull()

    // Search by Organization ID
    fireEvent.change(searchInput, { target: { value: "org-alpha-id" } })
    expect(screen.getByText("Alpha Corporation")).not.toBeNull()
    expect(screen.queryByText("Beta Logistics")).toBeNull()
  })
})
