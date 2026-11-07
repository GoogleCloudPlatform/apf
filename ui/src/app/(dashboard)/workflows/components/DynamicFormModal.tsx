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

import React, { useState } from "react"
import { Loader2 } from "lucide-react"

interface DynamicFormModalProps {
  nodeId: string
  interruptId: string
  schema: any
  onSubmit: (data: Record<string, any>) => void
  onClose: () => void
  isLoading?: boolean
}

export function DynamicFormModal({
  nodeId,
  interruptId,
  schema,
  onSubmit,
  onClose,
  isLoading = false,
}: DynamicFormModalProps) {
  const [formData, setFormData] = useState<Record<string, any>>({})
  
  const properties = schema.properties || {}
  const requiredFields = schema.required || []

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    // Validate required fields
    for (const key of requiredFields) {
      if (formData[key] === undefined || formData[key] === "") {
        alert(`Field "${key}" is required.`)
        return
      }
    }
    onSubmit(formData)
  }

  const handleFieldChange = (key: string, value: any, type: string) => {
    let normalizedValue = value
    if (type === "INTEGER" || type === "NUMBER") {
      normalizedValue = value === "" ? "" : Number(value)
    } else if (type === "BOOLEAN") {
      normalizedValue = Boolean(value)
    }
    setFormData((prev) => ({ ...prev, [key]: normalizedValue }))
  }

  return (
    <div className="fixed inset-0 bg-black/60 backdrop-blur-xs flex items-center justify-center z-50 select-none animate-in fade-in duration-150">
      <div className="bg-background border border-border p-6 rounded-xl shadow-2xl max-w-md w-full max-h-[90vh] flex flex-col overflow-hidden animate-in zoom-in-95 duration-150">
        <div className="flex items-center justify-between border-b border-border pb-3 mb-4 shrink-0">
          <div>
            <h3 className="text-sm font-bold text-foreground">Input Required</h3>
            <p className="text-[10px] text-muted-foreground mt-0.5 font-mono">Node: {nodeId}</p>
          </div>
          <button onClick={onClose} disabled={isLoading} className="text-muted-foreground hover:text-foreground text-xs font-semibold disabled:opacity-50">Cancel</button>
        </div>
        
        <form onSubmit={handleSubmit} className="flex-1 overflow-y-auto space-y-4 pr-1">
          {Object.entries(properties).map(([key, prop]: [string, any]) => {
            const isRequired = requiredFields.includes(key)
            const type = (prop.type || "STRING").toUpperCase()
            const desc = prop.description || ""

            return (
              <div key={key} className="space-y-1.5">
                <label className="text-[11px] font-bold text-foreground flex items-center gap-1">
                  {key}
                  {isRequired && <span className="text-red-500">*</span>}
                </label>
                {desc && <p className="text-[10px] text-muted-foreground leading-normal">{desc}</p>}
                
                {type === "BOOLEAN" ? (
                  <div className="flex items-center h-8">
                    <input
                      type="checkbox"
                      checked={!!formData[key]}
                      disabled={isLoading}
                      onChange={(e) => handleFieldChange(key, e.target.checked, type)}
                      className="h-4 w-4 rounded border-border text-primary focus:ring-primary/20 disabled:opacity-50"
                    />
                  </div>
                ) : (
                  <input
                    type={type === "INTEGER" || type === "NUMBER" ? "number" : "text"}
                    value={formData[key] ?? ""}
                    disabled={isLoading}
                    onChange={(e) => handleFieldChange(key, e.target.value, type)}
                    className="w-full px-3 py-2 border border-border rounded-lg bg-background text-xs text-foreground focus:outline-hidden focus:ring-2 focus:ring-primary/25 focus:border-primary transition-all font-sans disabled:opacity-50"
                    placeholder={`Enter ${key}...`}
                    required={isRequired}
                  />
                )}
              </div>
            )
          })}
          
          <div className="border-t border-border pt-4 mt-6 flex justify-end gap-2 shrink-0 select-none">
            <button
              type="button"
              onClick={onClose}
              disabled={isLoading}
              className="px-3 py-1.5 text-xs font-semibold text-muted-foreground hover:text-foreground hover:bg-muted rounded-lg transition-colors disabled:opacity-50"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={isLoading}
              className="px-4 py-1.5 text-xs font-bold text-primary-foreground bg-primary hover:bg-primary/90 rounded-lg shadow-sm transition-colors disabled:opacity-50 flex items-center gap-1.5"
            >
              {isLoading && <Loader2 className="h-3 w-3 animate-spin" />}
              {isLoading ? "Submitting..." : "Submit Response"}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
