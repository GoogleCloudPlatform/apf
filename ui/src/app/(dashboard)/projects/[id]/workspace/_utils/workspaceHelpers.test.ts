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

import { describe, it, expect } from "vitest"
import { 
  isArtifactMatchingFilePath, 
  isPreviewDocumentMatching, 
  normalizeArtifactName, 
  isGenericFilename 
} from "./workspaceHelpers"
import { Artifact } from "@/lib/types"

describe("workspaceHelpers artifact and preview matching", () => {
  describe("normalizeArtifactName", () => {
    it("strips file extensions, leading slashes, and special characters", () => {
      expect(normalizeArtifactName("docs/specs/prd.md")).toBe("docsspecsprd")
      expect(normalizeArtifactName("PRD")).toBe("prd")
      expect(normalizeArtifactName("UX Design")).toBe("uxdesign")
      expect(normalizeArtifactName("product-brief.md")).toBe("productbrief")
      expect(normalizeArtifactName("")).toBe("")
    })
  })

  describe("isGenericFilename", () => {
    it("identifies generic filenames", () => {
      expect(isGenericFilename("index.md")).toBe(true)
      expect(isGenericFilename("readme.md")).toBe(true)
      expect(isGenericFilename("INDEX.MD")).toBe(true)
      expect(isGenericFilename("prd.md")).toBe(false)
      expect(isGenericFilename("architecture.md")).toBe(false)
    })
  })

  describe("isArtifactMatchingFilePath", () => {
    it("matches exact path equality", () => {
      const artifact: Artifact = {
        id: "art-1",
        title: "PRD",
        path: "docs/specs/prd.md",
        phase: "PLANNING",
        status: "Complete",
        owner: "agent",
        content: "# PRD",
        artifactType: "product-req-doc",
      }

      expect(isArtifactMatchingFilePath(artifact, "docs/specs/prd.md")).toBe(true)
      expect(isArtifactMatchingFilePath(artifact, "/docs/specs/prd.md")).toBe(true)
      expect(isArtifactMatchingFilePath(artifact, "docs\\specs\\prd.md")).toBe(true)
    })

    it("matches subpath / suffix path equality", () => {
      const artifact: Artifact = {
        id: "art-1",
        title: "prd.md",
        path: "docs/spec/plans/prd.md",
        phase: "PLANNING",
        status: "Complete",
        owner: "agent",
        content: "# PRD",
        artifactType: "product-req-doc",
      }

      expect(isArtifactMatchingFilePath(artifact, "plans/prd.md")).toBe(true)
      expect(isArtifactMatchingFilePath(artifact, "prd.md")).toBe(true)
    })

    it("matches filename against title or basename", () => {
      const artifact: Artifact = {
        id: "proj1_prd.md",
        title: "PRD",
        path: "docs/specs/prd.md",
        phase: "PLANNING",
        status: "Complete",
        owner: "agent",
        content: "# PRD",
        artifactType: "product-req-doc",
      }

      expect(isArtifactMatchingFilePath(artifact, "docs/specs/prd.md")).toBe(true)
      expect(isArtifactMatchingFilePath(artifact, "prd.md")).toBe(true)
    })

    it("matches multi-word titles like 'UX Design' or 'Architecture'", () => {
      const uxArtifact: Artifact = {
        id: "art-ux",
        title: "UX Design",
        path: "docs/specs/ux-design.md",
        phase: "PLANNING",
        status: "Complete",
        owner: "agent",
        content: "# UX",
        artifactType: "ux-design-doc",
      }

      expect(isArtifactMatchingFilePath(uxArtifact, "docs/specs/ux-design.md")).toBe(true)
      expect(isArtifactMatchingFilePath(uxArtifact, "ux-design.md")).toBe(true)
    })

    it("returns false for different artifacts", () => {
      const artifact: Artifact = {
        id: "art-arch",
        title: "Architecture",
        path: "docs/specs/architecture.md",
        phase: "SOLUTIONING",
        status: "Complete",
        owner: "agent",
        content: "# Architecture",
        artifactType: "architecture-doc",
      }

      expect(isArtifactMatchingFilePath(artifact, "docs/specs/prd.md")).toBe(false)
      expect(isArtifactMatchingFilePath(artifact, "docs/specs/ux.md")).toBe(false)
      expect(isArtifactMatchingFilePath(artifact, "")).toBe(false)
    })
  })

  describe("isPreviewDocumentMatching", () => {
    const liveArtifacts: Artifact[] = [
      {
        id: "art-prd",
        title: "PRD",
        path: "docs/specs/prd.md",
        phase: "PLANNING",
        status: "Complete",
        owner: "agent",
        content: "# PRD Content",
        artifactType: "product-req-doc",
      },
      {
        id: "art-arch",
        title: "Architecture",
        path: "docs/specs/architecture.md",
        phase: "SOLUTIONING",
        status: "Complete",
        owner: "agent",
        content: "# Architecture Content",
        artifactType: "architecture-doc",
      },
    ]

    it("returns true when open preview document matches updated file", () => {
      expect(isPreviewDocumentMatching("PRD", liveArtifacts, "docs/specs/prd.md")).toBe(true)
      expect(isPreviewDocumentMatching("PRD", liveArtifacts, "prd.md")).toBe(true)
      expect(isPreviewDocumentMatching("Architecture", liveArtifacts, "docs/specs/architecture.md")).toBe(true)
    })

    it("returns false when open preview document does not match updated file", () => {
      expect(isPreviewDocumentMatching("Architecture", liveArtifacts, "docs/specs/prd.md")).toBe(false)
      expect(isPreviewDocumentMatching("PRD", liveArtifacts, "docs/specs/architecture.md")).toBe(false)
    })

    it("returns false when preview is not open (null)", () => {
      expect(isPreviewDocumentMatching(null, liveArtifacts, "docs/specs/prd.md")).toBe(false)
      expect(isPreviewDocumentMatching("", liveArtifacts, "docs/specs/prd.md")).toBe(false)
    })
  })
})
