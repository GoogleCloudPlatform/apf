# Technical Research: On-Demand Workspace Provisioning & Seeding Engines

## 1. Executive Summary
To implement the **Foundry Showcase** feature, we must evaluate different technical architectures for provisioning pre-configured project workspaces on-demand. The system must be fast (provisioning in < 10 seconds), highly secure (preventing path traversal and multitenancy context leaks), and strictly idempotent.

## 2. Evaluation of Seeding Architectures
We evaluated three different technical approaches for storing and loading showcase templates:

### Option A: Remote Git Cloning
- **Description**: Store the showcase spec templates in a public GitHub repository and clone them at runtime during provisioning.
- **Pros**: Easy to update templates without rebuilding the backend API.
- **Cons**: High latency (>5 seconds due to network bounds), dependency on external network uptime, and potential API rate-limiting issues on GitHub.

### Option B: Local Disk Templates
- **Description**: Store templates as separate markdown and JSON files on the server's local disk.
- **Pros**: Fast read times, easy to edit locally.
- **Cons**: Increases deployment complexity; templates must be packaged and deployed to the server's disk, risking path resolution issues across different environments (local dev vs. cloud run).

### Option C: Go Embedded Filesystem (Selected)
- **Description**: Use Go 1.16+ `embed` package to bundle JSON metadata and Markdown files directly into the compiled Go binary.
- **Pros**: 
  - **Sub-millisecond read latency** (compiled into memory).
  - **Zero deployment overhead** (embedded inside a single stateless binary).
  - **Absolute reliability** (no external network or disk dependencies).
- **Cons**: Changing templates requires rebuilding the backend binary.

---

## 3. Security Risk Assessment & Mitigations

### 3.1 Path Traversal Attacks
- **Risk**: Since the reset handler clears local filesystem directories based on the `projectId`, a malicious request with a project ID like `../../` could wipe out the entire server filesystem.
- **Mitigation**: Implement strict sanitization on the `projectId` parameter. We will mathematically restrict resets to project IDs prefixed with `showcase-` and reject any strings containing `..`, `/`, or `\`.

### 3.2 Multitenancy Context Bleed
- **Risk**: Ephemeral showcase data from one user leaking into another user's organization workspace.
- **Mitigation**: Seed the project using a deterministic ID containing the user's secure organization ID and user ID: `showcase-{orgId}-{userId}`, and enforce hard Firestore multitenancy query boundaries using the request's context claims.
