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

import React, { useState, useEffect } from "react"
import { Plus, Trash2, X, Info } from "lucide-react"
import { cn } from "@/lib/utils"

interface BuilderRule {
  id: string
  field: string
  operator: string
  value: string
  valueType: "string" | "number" | "boolean" | "variable"
}

interface BuilderGroup {
  conjunction: "and" | "or"
  rules: BuilderRule[]
}

interface LogicBuilderProps {
  isOpen: boolean
  onClose: () => void
  onSave: (logic: Record<string, any>) => void
  initialLogic?: Record<string, any>
  loopNodeName?: string
}

const OPERATORS = [
  { value: "==", label: "equals (==)" },
  { value: "!=", label: "does not equal (!=)" },
  { value: ">", label: "greater than (>)" },
  { value: "<", label: "less than (<)" },
  { value: ">=", label: "greater than or equal (>=)" },
  { value: "<=", label: "less than or equal (<=)" },
]

const VALUE_TYPES = [
  { value: "string", label: "Text" },
  { value: "number", label: "Number" },
  { value: "boolean", label: "Boolean" },
  { value: "variable", label: "Variable" },
]

export const LogicBuilder = ({
  isOpen,
  onClose,
  onSave,
  initialLogic,
  loopNodeName = "loop_items",
}: LogicBuilderProps) => {
  const [group, setGroup] = useState<BuilderGroup>({
    conjunction: "and",
    rules: [],
  })

  const loopCountVar = `_loop_${loopNodeName}_count`

  // Load initial logic on open
  useEffect(() => {
    if (isOpen) {
      if (initialLogic) {
        setGroup(deserializeFromJsonLogic(initialLogic))
      } else {
        // Default with one rule using the loop count variable
        setGroup({
          conjunction: "and",
          rules: [
            {
              id: Math.random().toString(36).substring(7),
              field: loopCountVar,
              operator: "==",
              value: "3",
              valueType: "number",
            },
          ],
        })
      }
    }
  }, [isOpen, initialLogic, loopCountVar])

  if (!isOpen) return null

  const handleAddRule = () => {
    const newRule: BuilderRule = {
      id: Math.random().toString(36).substring(7),
      field: "",
      operator: "==",
      value: "",
      valueType: "string",
    }
    setGroup((prev) => ({
      ...prev,
      rules: [...prev.rules, newRule],
    }))
  }

  const handleRemoveRule = (id: string) => {
    setGroup((prev) => ({
      ...prev,
      rules: prev.rules.filter((r) => r.id !== id),
    }))
  }

  const handleUpdateRule = (id: string, updates: Partial<BuilderRule>) => {
    setGroup((prev) => ({
      ...prev,
      rules: prev.rules.map((r) => {
        if (r.id === id) {
          const updated = { ...r, ...updates }
          // Reset value if type changes to boolean to ensure it's "true" or "false"
          if (updates.valueType === "boolean") {
            updated.value = "true"
          } else if (updates.valueType && updates.valueType !== r.valueType) {
            updated.value = ""
          }
          return updated
        }
        return r
      }),
    }))
  }

  const isGroupValid = (): boolean => {
    if (group.rules.length === 0) return false
    return group.rules.every(r => r.field.trim() !== "" && isValidFieldName(r.field))
  }

  const handleApply = () => {
    const serialized = serializeToJsonLogic(group)
    onSave(serialized)
    onClose()
  }

  const serializedPreview = serializeToJsonLogic(group)

  return (
    <div className="fixed inset-0 bg-slate-950/40 dark:bg-slate-950/60 flex items-center justify-center z-[9999] backdrop-blur-sm p-4">
      <div className="bg-background border border-border rounded-xl shadow-2xl w-full max-w-6xl max-h-[90vh] flex flex-col overflow-hidden animate-in fade-in zoom-in-95 duration-150">
        
        {/* Header */}
        <div className="flex items-center justify-between border-b px-6 py-4">
          <div>
            <h3 className="text-sm font-bold text-foreground">Logic Builder</h3>
            <p className="text-[11px] text-muted-foreground mt-0.5">
              Build conditions visually. Generates JsonLogic formatted JSON.
            </p>
          </div>
          <button
            onClick={onClose}
            className="rounded-lg p-1.5 text-muted-foreground hover:bg-muted hover:text-foreground transition-colors"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-6 space-y-6">
          
          {/* Rules Section */}
          <div className="space-y-4">
            {group.rules.length > 1 && (
              <div className="flex items-center gap-2 bg-muted/50 p-2 rounded-lg w-fit">
                <span className="text-[10px] font-bold text-muted-foreground uppercase tracking-wide px-1">
                  Match
                </span>
                <select
                  value={group.conjunction}
                  onChange={(e) => setGroup(prev => ({ ...prev, conjunction: e.target.value as any }))}
                  className="bg-background border border-input rounded px-2 py-0.5 text-xs font-semibold focus:outline-none focus:ring-1 focus:ring-primary"
                >
                  <option value="and">ALL (AND)</option>
                  <option value="or">ANY (OR)</option>
                </select>
                <span className="text-[10px] font-bold text-muted-foreground uppercase tracking-wide px-1">
                  conditions:
                </span>
              </div>
            )}

            {group.rules.length === 0 ? (
              <div className="text-center py-8 border border-dashed rounded-xl bg-muted/20">
                <p className="text-xs text-muted-foreground">No conditions defined yet.</p>
                <button
                  onClick={handleAddRule}
                  className="mt-3 inline-flex items-center gap-1.5 text-xs font-semibold text-primary hover:underline"
                >
                  <Plus className="h-3.5 w-3.5" /> Add your first condition
                </button>
              </div>
            ) : (
              <div className="space-y-3">
                {group.rules.map((rule, index) => {
                  const isFieldInvalid = rule.field.trim() !== "" && !isValidFieldName(rule.field)
                  return (
                    <div 
                      key={rule.id} 
                      className="flex flex-wrap md:flex-nowrap items-center gap-3 p-3 border rounded-lg bg-card/50 hover:bg-card transition-colors relative group"
                    >
                      {/* Index Badge */}
                      <span className="text-[9px] font-bold bg-muted text-muted-foreground w-5 h-5 rounded-full flex items-center justify-center select-none shrink-0">
                        {index + 1}
                      </span>

                      {/* Field (Variable) */}
                      <div className="flex-1 min-w-[150px] space-y-1">
                        <div className="flex items-center justify-between">
                          <span className="text-[9px] font-bold text-muted-foreground uppercase tracking-wider">
                            Variable / Field
                          </span>
                          {rule.field !== loopCountVar && (
                            <button
                              onClick={() => handleUpdateRule(rule.id, { field: loopCountVar })}
                              className="text-[8px] font-bold text-primary hover:underline uppercase"
                              title="Use the current loop iteration counter variable"
                            >
                              Use Loop Count
                            </button>
                          )}
                        </div>
                        <input
                          type="text"
                          value={rule.field}
                          onChange={(e) => handleUpdateRule(rule.id, { field: e.target.value })}
                          placeholder="e.g. status, items_count"
                          className={cn(
                            "w-full rounded border bg-transparent px-2.5 py-1.5 text-xs focus:outline-none focus:ring-1",
                            isFieldInvalid 
                              ? "border-destructive focus:ring-destructive text-destructive" 
                              : "border-input focus:ring-primary text-foreground"
                          )}
                        />
                        {isFieldInvalid && (
                          <p className="text-[8px] text-destructive font-semibold">
                            Invalid name. Use alphanumeric, _, $, or .
                          </p>
                        )}
                      </div>

                      {/* Operator */}
                      <div className="w-[130px] shrink-0 space-y-1">
                        <span className="text-[9px] font-bold text-muted-foreground uppercase tracking-wider">
                          Operator
                        </span>
                        <select
                          value={rule.operator}
                          onChange={(e) => handleUpdateRule(rule.id, { operator: e.target.value })}
                          className="w-full rounded border border-input bg-background px-2 py-1.5 text-xs focus:outline-none focus:ring-1 focus:ring-primary"
                        >
                          {OPERATORS.map((op) => (
                            <option key={op.value} value={op.value}>
                              {op.label}
                            </option>
                          ))}
                        </select>
                      </div>

                      {/* Value Type */}
                      <div className="w-[90px] shrink-0 space-y-1">
                        <span className="text-[9px] font-bold text-muted-foreground uppercase tracking-wider">
                          Type
                        </span>
                        <select
                          value={rule.valueType}
                          onChange={(e) => handleUpdateRule(rule.id, { valueType: e.target.value as any })}
                          className="w-full rounded border border-input bg-background px-2 py-1.5 text-xs focus:outline-none focus:ring-1 focus:ring-primary"
                        >
                          {VALUE_TYPES.map((t) => (
                            <option key={t.value} value={t.value}>
                              {t.label}
                            </option>
                          ))}
                        </select>
                      </div>

                      {/* Value */}
                      <div className="flex-1 min-w-[120px] space-y-1">
                        <span className="text-[9px] font-bold text-muted-foreground uppercase tracking-wider">
                          Value
                        </span>
                        {rule.valueType === "boolean" ? (
                          <select
                            value={rule.value}
                            onChange={(e) => handleUpdateRule(rule.id, { value: e.target.value })}
                            className="w-full rounded border border-input bg-background px-2 py-1.5 text-xs focus:outline-none focus:ring-1 focus:ring-primary"
                          >
                            <option value="true">True</option>
                            <option value="false">False</option>
                          </select>
                        ) : (
                          <input
                            type={rule.valueType === "number" ? "number" : "text"}
                            value={rule.value}
                            onChange={(e) => handleUpdateRule(rule.id, { value: e.target.value })}
                            placeholder={
                              rule.valueType === "variable"
                                ? "Variable name..."
                                : rule.valueType === "number"
                                ? "Number..."
                                : "Value..."
                            }
                            className="w-full rounded border border-input bg-transparent px-2.5 py-1.5 text-xs focus:outline-none focus:ring-1 focus:ring-primary"
                          />
                        )}
                      </div>

                      {/* Remove Button */}
                      <button
                        onClick={() => handleRemoveRule(rule.id)}
                        className="p-1.5 rounded hover:bg-muted text-muted-foreground hover:text-destructive transition-colors shrink-0 mt-4 md:mt-0"
                        title="Remove condition"
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                      </button>
                    </div>
                  )
                })}

                <button
                  onClick={handleAddRule}
                  className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-input bg-background hover:bg-muted text-xs font-semibold text-foreground transition-colors"
                >
                  <Plus className="h-3.5 w-3.5" /> Add Condition
                </button>
              </div>
            )}
          </div>

          {/* JSON Preview Section */}
          <div className="border rounded-xl p-4 bg-muted/30 space-y-2">
            <div className="flex items-center gap-1.5 text-[10px] font-bold text-muted-foreground uppercase tracking-wide">
              <Info className="h-3.5 w-3.5 text-info fill-info/10" />
              Compiled JsonLogic Output Preview
            </div>
            <pre className="p-3 rounded-lg border bg-slate-50 text-slate-900 dark:bg-slate-950 dark:text-slate-200 text-[11px] font-mono overflow-x-auto max-h-[150px]">
              {JSON.stringify(serializedPreview, null, 2)}
            </pre>
          </div>

        </div>

        {/* Footer */}
        <div className="border-t px-6 py-4 flex items-center justify-end gap-3 bg-muted/20">
          <button
            onClick={onClose}
            className="inline-flex h-9 items-center justify-center rounded-lg border border-input bg-background px-4 py-2 text-xs font-semibold text-foreground hover:bg-muted transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={handleApply}
            disabled={!isGroupValid()}
            className="inline-flex h-9 items-center justify-center rounded-lg bg-primary px-4 py-2 text-xs font-semibold text-primary-foreground shadow transition-all hover:opacity-90 disabled:opacity-50"
          >
            Apply Condition
          </button>
        </div>

      </div>
    </div>
  )
}

// --- Helper Functions for Serialization/Deserialization ---

const serializeToJsonLogic = (group: BuilderGroup): Record<string, any> => {
  if (group.rules.length === 0) return {}

  const serializedRules = group.rules
    .filter(r => r.field.trim() !== "") // Skip rules with empty fields
    .map((rule) => {
      let val: any = rule.value
      if (rule.valueType === "number") {
        val = parseFloat(rule.value)
        if (isNaN(val)) val = 0
      } else if (rule.valueType === "boolean") {
        val = rule.value === "true"
      } else if (rule.valueType === "variable") {
        val = { var: rule.value }
      }

      return {
        [rule.operator]: [{ var: rule.field }, val],
      }
    })

  if (serializedRules.length === 0) return {}
  if (serializedRules.length === 1) return serializedRules[0]

  return {
    [group.conjunction]: serializedRules,
  }
}

const deserializeFromJsonLogic = (logic: Record<string, any>): BuilderGroup => {
  const defaultGroup: BuilderGroup = {
    conjunction: "and",
    rules: [
      {
        id: Math.random().toString(36).substring(7),
        field: "",
        operator: "==",
        value: "",
        valueType: "string",
      },
    ],
  }

  if (!logic || typeof logic !== "object" || Object.keys(logic).length === 0) {
    return defaultGroup
  }

  const op = Object.keys(logic)[0]
  const args = logic[op]

  if (!Array.isArray(args)) {
    return defaultGroup
  }

  if (op === "and" || op === "or") {
    const rules: BuilderRule[] = []

    for (const arg of args) {
      if (typeof arg === "object" && arg !== null) {
        const subOp = Object.keys(arg)[0]
        const subArgs = arg[subOp]

        if (Array.isArray(subArgs) && subArgs.length === 2) {
          const fieldVar = subArgs[0]
          const valObj = subArgs[1]

          if (typeof fieldVar === "object" && fieldVar !== null && "var" in fieldVar) {
            const field = fieldVar.var
            let value = ""
            let valueType: "string" | "number" | "boolean" | "variable" = "string"

            if (typeof valObj === "object" && valObj !== null && "var" in valObj) {
              value = valObj.var
              valueType = "variable"
            } else if (typeof valObj === "number") {
              value = String(valObj)
              valueType = "number"
            } else if (typeof valObj === "boolean") {
              value = String(valObj)
              valueType = "boolean"
            } else {
              value = String(valObj)
              valueType = "string"
            }

            rules.push({
              id: Math.random().toString(36).substring(7),
              field,
              operator: subOp,
              value,
              valueType,
            })
          }
        }
      }
    }

    if (rules.length > 0) {
      return {
        conjunction: op as "and" | "or",
        rules,
      }
    }
  } else {
    // Single rule
    if (args.length === 2) {
      const fieldVar = args[0]
      const valObj = args[1]

      if (typeof fieldVar === "object" && fieldVar !== null && "var" in fieldVar) {
        const field = fieldVar.var
        let value = ""
        let valueType: "string" | "number" | "boolean" | "variable" = "string"

        if (typeof valObj === "object" && valObj !== null && "var" in valObj) {
          value = valObj.var
          valueType = "variable"
        } else if (typeof valObj === "number") {
          value = String(valObj)
          valueType = "number"
        } else if (typeof valObj === "boolean") {
          value = String(valObj)
          valueType = "boolean"
        } else {
          value = String(valObj)
          valueType = "string"
        }

        return {
          conjunction: "and",
          rules: [
            {
              id: Math.random().toString(36).substring(7),
              field,
              operator: op,
              value,
              valueType,
            },
          ],
        }
      }
    }
  }

  return defaultGroup
}

const isValidFieldName = (name: string): boolean => {
  if (!name) return false
  // Allows alphanumeric, underscores, dollar signs, and dots (for nested paths)
  const regex = /^[a-zA-Z0-9_$]+(?:\.[a-zA-Z0-9_$]+)*$/
  return regex.test(name)
}
