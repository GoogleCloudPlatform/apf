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

import React from "react"
import { X, Trash2 } from "lucide-react"
import { cn, isValidPythonIdentifier } from "@/lib/utils"
import { DesignerNode, DesignerEdge } from "./WorkflowDesigner"
import { LlmNodeAttributes } from "./LlmNodeAttributes"
import { FunctionNodeAttributes } from "./FunctionNodeAttributes"
import { JoinNodeAttributes } from "./JoinNodeAttributes"
import { ScheduledTriggerAttributes } from "../nodes/triggers/ScheduledTriggerAttributes"
import { ManualTriggerAttributes } from "../nodes/triggers/ManualTriggerAttributes"
import { FormTriggerAttributes } from "../nodes/triggers/FormTriggerAttributes"
import { CustomNodeAttributes } from "./CustomNodeAttributes"
import { LoopNodeAttributes } from "./LoopNodeAttributes"

interface WorkflowPropertiesPanelProps {
  selectedNodeId: string | null
  selectedEdgeId: string | null
  nodes: DesignerNode[]
  edges: DesignerEdge[]

  // General workflow settings
  workflowDisplayName: string
  setWorkflowDisplayName: (val: string) => void
  workflowName: string
  setWorkflowName: (val: string) => void
  workflowDescription: string
  setWorkflowDescription: (val: string) => void
  framework: string
  setFramework: (val: string) => void

  // Update handlers
  handleUpdateNodeData: (field: string, val: any) => void
  handleUpdateEdgeRoute: (val: string) => void
  handleDeleteSelected: () => void
  onPaneClick: () => void

  readOnly?: boolean
}

export function WorkflowPropertiesPanel({
  selectedNodeId,
  selectedEdgeId,
  nodes,
  edges,
  workflowDisplayName,
  setWorkflowDisplayName,
  workflowName,
  setWorkflowName,
  workflowDescription,
  setWorkflowDescription,
  framework,
  setFramework,
  handleUpdateNodeData,
  handleUpdateEdgeRoute,
  handleDeleteSelected,
  onPaneClick,
  readOnly = false,
}: WorkflowPropertiesPanelProps) {
  // Get currently selected items
  const selectedNode = nodes.find((n) => n.id === selectedNodeId)
  const selectedEdge = edges.find((e) => e.id === selectedEdgeId)

  const otherNodeNames = nodes
    .filter((n) => n.id !== selectedNodeId && n.id !== "start" && n.type !== "waypoint" && n.type !== "noop")
    .map((n) => n.data?.name)
    .filter((name): name is string => !!name)

  const selectedNodeName = selectedNode?.data?.name || ""
  const isSelectedNodeNameValid = isValidPythonIdentifier(selectedNodeName)
  const isSelectedNodeNameDuplicate = otherNodeNames.includes(selectedNodeName)

  let selectedNodeNameError = ""
  if (selectedNode && selectedNode.id !== "start" && selectedNode.type !== "waypoint" && selectedNode.type !== "noop") {
    if (!selectedNodeName) {
      selectedNodeNameError = "ADK Name is required"
    } else if (!isSelectedNodeNameValid) {
      selectedNodeNameError = "Must be a valid Python identifier (starts with letter/underscore, alphanumeric/underscore only, and not a Python keyword)"
    } else if (isSelectedNodeNameDuplicate) {
      selectedNodeNameError = "Name must be unique across all nodes in the workflow"
    }
  }

  const hasSelection = selectedNodeId || selectedEdgeId

  return (
    <div className="h-full flex flex-col overflow-hidden bg-card border-l border-border select-none">
      {hasSelection ? (
        <>
          {/* Property Header */}
          <div className="p-4 border-b border-border bg-muted/10 flex items-center justify-between shrink-0">
            <div>
              <h3 className="font-semibold text-xs text-foreground uppercase tracking-wider">
                {selectedNodeId ? "Node Properties" : "Connection Properties"}
              </h3>
              <p className="text-[10px] text-muted-foreground mt-0.5">
                {selectedNodeId ? `ID: ${selectedNodeId}` : `ID: ${selectedEdgeId}`}
              </p>
            </div>
            <button
              onClick={onPaneClick}
              className="p-1.5 hover:bg-muted rounded text-muted-foreground hover:text-foreground"
            >
              <X className="h-4 w-4" />
            </button>
          </div>

          {/* Properties Content Form */}
          <div className="flex-1 overflow-y-auto p-4 space-y-5">
            <div className={cn("space-y-5", readOnly && "pointer-events-none opacity-85")}>
              {selectedNode && selectedNode.id !== "start" ? (
                <>
                  {selectedNode.type === "agent" ? (
                    <LlmNodeAttributes
                      data={selectedNode.data || {}}
                      onChange={handleUpdateNodeData}
                      existingNames={otherNodeNames}
                    />
                  ) : (
                    <>
                      {/* 1. Display Name Field */}
                      <div className="space-y-1.5">
                        <label className="text-[10px] font-bold text-muted-foreground uppercase tracking-wide">
                          Display Name
                        </label>
                        <input
                          type="text"
                          value={selectedNode.data?.displayName || ""}
                          onChange={(e) => handleUpdateNodeData("displayName", e.target.value)}
                          placeholder="Friendly name for this node"
                          disabled={readOnly}
                          className="flex h-9 w-full rounded-lg border border-input bg-transparent px-3 py-1 text-xs shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-primary"
                        />
                        <p className="text-[9px] text-muted-foreground/80 leading-normal">
                          A friendly title displayed on the design canvas.
                        </p>
                      </div>

                      {/* 2. Common Name Field (ADK Name) */}
                      <div className="space-y-1.5">
                        <label className="text-[10px] font-bold text-muted-foreground uppercase tracking-wide">
                          ADK Name
                        </label>
                        <input
                          type="text"
                          value={selectedNode.data?.name || ""}
                          onChange={(e) => handleUpdateNodeData("name", e.target.value)}
                          placeholder="e.g. agent_pm"
                          disabled={readOnly}
                          className={cn(
                            "flex h-9 w-full rounded-lg border bg-transparent px-3 py-1 text-xs shadow-sm focus-visible:outline-none focus-visible:ring-1",
                            selectedNodeNameError
                              ? "border-destructive focus-visible:ring-destructive"
                              : "border-input focus-visible:ring-primary"
                          )}
                        />
                        {selectedNodeNameError ? (
                          <p className="text-[9px] text-destructive font-semibold mt-1 leading-normal">
                            {selectedNodeNameError}
                          </p>
                        ) : (
                          <p className="text-[9px] text-muted-foreground/80 leading-normal">
                            Must be a unique Python identifier (starts with a letter, no spaces or dashes).
                          </p>
                        )}
                      </div>

                      {/* 3. Common Description Field */}
                      <div className="space-y-1.5">
                        <label className="text-[10px] font-bold text-muted-foreground uppercase tracking-wide">
                          Description
                        </label>
                        <textarea
                          value={selectedNode.data?.description || ""}
                          onChange={(e) => handleUpdateNodeData("description", e.target.value)}
                          placeholder="What does this node do?"
                          rows={3}
                          disabled={readOnly}
                          className="flex w-full rounded-lg border border-input bg-transparent px-3 py-2 text-xs shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-primary resize-none"
                        />
                      </div>

                      {/* --- Type Specific Fields for non-agent nodes --- */}
                      {selectedNode.type === "function" && (
                        <FunctionNodeAttributes data={selectedNode.data || {}} onChange={handleUpdateNodeData} />
                      )}
                      {selectedNode.type === "join" && (
                        <JoinNodeAttributes data={selectedNode.data || {}} onChange={handleUpdateNodeData} />
                      )}
                      {selectedNode.type === "trigger" && (
                        <ScheduledTriggerAttributes data={selectedNode.data || {}} onChange={handleUpdateNodeData} />
                      )}
                      {selectedNode.type === "manual_trigger" && (
                        <ManualTriggerAttributes data={selectedNode.data || {}} onChange={handleUpdateNodeData} />
                      )}
                      {selectedNode.type === "form_trigger" && (
                        <FormTriggerAttributes data={selectedNode.data || {}} onChange={handleUpdateNodeData} />
                      )}
                      {selectedNode.type === "loop" && (
                        <LoopNodeAttributes data={selectedNode.data || {}} onChange={handleUpdateNodeData} />
                      )}
                      {selectedNode.type === "custom" && (
                        <CustomNodeAttributes data={selectedNode.data || {}} onChange={handleUpdateNodeData} />
                      )}
                      
                      {/* --- Advanced Graph Execution Primitives for non-agent nodes --- */}
                      <div className="border-t border-border/50 pt-4 space-y-4">
                        <h4 className="text-[10px] font-bold text-muted-foreground uppercase tracking-wider">
                          Advanced ADK Primitives
                        </h4>

                        {/* Rerun on Resume */}
                        <div className="flex items-center justify-between">
                          <div className="space-y-0.5">
                            <label className="text-[11px] font-semibold text-foreground">
                              Rerun on Resume
                            </label>
                            <p className="text-[9px] text-muted-foreground/80">
                              Rerun node from scratch when resumed.
                            </p>
                          </div>
                          <input
                            type="checkbox"
                            checked={!!selectedNode.data?.rerunOnResume}
                            onChange={(e) => handleUpdateNodeData("rerunOnResume", e.target.checked)}
                            disabled={readOnly}
                            className="h-4 w-4 rounded border-gray-300 text-primary focus:ring-primary"
                          />
                        </div>

                        {/* Wait for Output */}
                        <div className="flex items-center justify-between">
                          <div className="space-y-0.5">
                            <label className="text-[11px] font-semibold text-foreground">
                              Wait for Output
                            </label>
                            <p className="text-[9px] text-muted-foreground/80">
                              Pause state until node explicitly yields output.
                            </p>
                          </div>
                          <input
                            type="checkbox"
                            checked={!!selectedNode.data?.waitForOutput}
                            onChange={(e) => handleUpdateNodeData("waitForOutput", e.target.checked)}
                            disabled={readOnly}
                            className="h-4 w-4 rounded border-gray-300 text-primary focus:ring-primary"
                          />
                        </div>

                        {/* Timeout */}
                        <div className="space-y-1.5">
                          <label className="text-[10px] font-bold text-muted-foreground uppercase tracking-wide">
                            Timeout (Seconds)
                          </label>
                          <input
                            type="number"
                            value={selectedNode.data?.timeout || ""}
                            onChange={(e) => handleUpdateNodeData("timeout", e.target.value)}
                            placeholder="No Timeout"
                            disabled={readOnly}
                            className="flex h-9 w-full rounded-lg border border-input bg-transparent px-3 py-1 text-xs shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-primary"
                          />
                        </div>
                      </div>
                    </>
                  )}
                </>
              ) : selectedNode && selectedNode.id === "start" ? (
                <div className="p-4 text-center text-xs text-muted-foreground border rounded-xl border-dashed">
                  Implicit Start node of the workflow. No custom properties can be configured.
                </div>
              ) : selectedEdge ? (
                /* Edge Properties Editor */
                <div className="space-y-4">
                  <div className="space-y-1.5">
                    <label className="text-[10px] font-bold text-muted-foreground uppercase tracking-wide">
                      Route Name (Conditional Branch)
                    </label>
                    <input
                      type="text"
                      value={typeof selectedEdge.label === "string" ? selectedEdge.label : ""}
                      onChange={(e) => handleUpdateEdgeRoute(e.target.value)}
                      placeholder="e.g. success, failure, approved"
                      disabled={readOnly}
                      className="flex h-9 w-full rounded-lg border border-input bg-transparent px-3 py-1 text-xs shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-primary"
                    />
                    <p className="text-[9px] text-muted-foreground/80 leading-normal">
                      Assigns a label to this edge. ADK follows this path only when the source node yields this matching route value. Leave empty for default unconditional flow.
                    </p>
                  </div>
                </div>
              ) : null}
            </div>
          </div>

          {/* Danger Zone / Delete action (Hidden in readOnly mode) */}
          {!readOnly && (
            <div className="p-4 border-t border-border bg-muted/5 shrink-0">
              <button
                onClick={handleDeleteSelected}
                className="w-full inline-flex h-9 items-center justify-center gap-1.5 rounded-lg border border-destructive/30 bg-transparent px-4 py-2 text-xs font-semibold text-destructive hover:bg-destructive/10 transition-colors"
              >
                <Trash2 className="h-3.5 w-3.5" />
                {selectedNodeId ? "Delete Node" : "Delete Connection"}
              </button>
            </div>
          )}
        </>
      ) : (
        /* General Workflow Settings Pane when nothing is selected */
        <div className="h-full flex flex-col overflow-hidden bg-card p-4 space-y-4">
          <div className="border-b border-border pb-3">
            <h3 className="font-semibold text-xs text-foreground uppercase tracking-wider">
              Workflow Details
            </h3>
            <p className="text-[10px] text-muted-foreground mt-0.5">
              Orchestration metadata
            </p>
          </div>

          <div className={cn("space-y-4 flex-1", readOnly && "pointer-events-none opacity-85")}>
            {/* 1. Display Name */}
            <div className="space-y-1.5">
              <label className="text-[10px] font-bold text-muted-foreground uppercase tracking-wide">
                Display Name
              </label>
              <input
                type="text"
                value={workflowDisplayName}
                onChange={(e) => {
                  if (readOnly) return
                  const newVal = e.target.value
                  setWorkflowDisplayName(newVal)
                  const oldSlug = workflowDisplayName.toLowerCase().replace(/[^a-z0-9_]/g, "_")
                  if (!workflowName || workflowName === oldSlug) {
                    const newSlug = newVal.toLowerCase().replace(/[^a-z0-9_]/g, "_")
                    if (/^[a-z_]/.test(newSlug)) {
                      setWorkflowName(newSlug)
                    }
                  }
                }}
                placeholder="Friendly name for the workflow"
                disabled={readOnly}
                className="flex h-9 w-full rounded-lg border border-input bg-transparent px-3 py-1 text-xs shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-primary"
              />
            </div>

            {/* 2. ADK Name */}
            <div className="space-y-1.5">
              <label className="text-[10px] font-bold text-muted-foreground uppercase tracking-wide">
                ADK Name
              </label>
              <input
                type="text"
                value={workflowName}
                onChange={(e) => {
                  if (readOnly) return
                  setWorkflowName(e.target.value)
                }}
                placeholder="e.g. my_awesome_workflow"
                disabled={readOnly}
                className={cn(
                  "flex h-9 w-full rounded-lg border bg-transparent px-3 py-1 text-xs shadow-sm focus-visible:outline-none focus-visible:ring-1",
                  workflowName && !isValidPythonIdentifier(workflowName)
                    ? "border-destructive focus-visible:ring-destructive"
                    : "border-input focus-visible:ring-primary"
                )}
              />
              {workflowName && !isValidPythonIdentifier(workflowName) ? (
                <p className="text-[9px] text-destructive font-semibold mt-1 leading-normal">
                  Must be a valid Python identifier (starts with a letter, no spaces or dashes).
                </p>
              ) : (
                <p className="text-[9px] text-muted-foreground/80 leading-normal">
                  Technical identifier used in code generation.
                </p>
              )}
            </div>

            {/* 3. Description */}
            <div className="space-y-1.5">
              <label className="text-[10px] font-bold text-muted-foreground uppercase tracking-wide">
                Description
              </label>
              <textarea
                value={workflowDescription}
                onChange={(e) => {
                  if (readOnly) return
                  setWorkflowDescription(e.target.value)
                }}
                placeholder="What does this workflow do?"
                rows={4}
                disabled={readOnly}
                className="flex w-full rounded-lg border border-input bg-transparent px-3 py-2 text-xs shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-primary resize-none"
              />
            </div>

            {/* 4. Target Runtime */}
            <div className="space-y-1.5 border-t border-border/50 pt-3">
              <label className="text-[10px] font-bold text-muted-foreground uppercase tracking-wide">
                Target Runtime
              </label>
              <select
                value={framework}
                onChange={(e) => {
                  if (readOnly) return
                  setFramework(e.target.value)
                }}
                disabled={readOnly}
                className="flex h-9 w-full rounded-lg border border-input bg-card px-3 py-1 text-xs shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-primary"
              >
                <option value="ADK 2.0">ADK 2.0</option>
                <option value="Tmux">Tmux</option>
                <option value="Archon">Archon</option>
                <option value="Scion">Scion</option>
              </select>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
