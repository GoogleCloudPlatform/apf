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

import { useState, useEffect } from "react"
import { Plus, Trash2 } from "lucide-react"

interface SchemaField {
  name: string
  type: string
  description: string
}

interface SchemaEditorProps {
  label: string
  schema: Record<string, any> | undefined
  onChange: (newSchema: Record<string, any> | undefined) => void
  disabled?: boolean
}

const SUPPORTED_TYPES = ["string", "integer", "boolean", "number", "array", "object"]

export const SchemaEditor = ({ label, schema = {}, onChange, disabled = false }: SchemaEditorProps) => {
  const [fields, setFields] = useState<SchemaField[]>(() => {
    return Object.entries(schema || {}).map(([name, info]: [string, any]) => ({
      name,
      type: info?.type || "string",
      description: info?.description || "",
    }))
  })

  useEffect(() => {
    // Compile the current local fields to see what they represent semantically
    const currentCompiled: Record<string, any> = {}
    fields.forEach((f) => {
      if (f.name.trim()) {
        currentCompiled[f.name.trim()] = {
          type: f.type,
          description: f.description.trim() || undefined,
        }
      }
    })

    // Compare the compiled local state with the incoming schema
    const compiledSerialized = JSON.stringify(currentCompiled)
    const incomingSerialized = JSON.stringify(schema || {})

    // If they are different, it means the incoming schema changed from the outside
    // (e.g. node switch, undo/redo), so we must reset our local state.
    if (compiledSerialized !== incomingSerialized) {
      const newFields = Object.entries(schema || {}).map(([name, info]: [string, any]) => ({
        name,
        type: info?.type || "string",
        description: info?.description || "",
      }))
      setFields(newFields)
    }
  }, [schema])

  const triggerChange = (updatedFields: SchemaField[]) => {
    const result: Record<string, any> = {}
    updatedFields.forEach((f) => {
      if (f.name.trim()) {
        result[f.name.trim()] = {
          type: f.type,
          description: f.description.trim() || undefined, // omit if empty
        }
      }
    })
    onChange(Object.keys(result).length > 0 ? result : undefined)
  }

  const handleFieldChange = (index: number, key: keyof SchemaField, val: string) => {
    if (disabled) return
    const updated = [...fields]
    updated[index][key] = val
    setFields(updated)
    triggerChange(updated)
  }

  const addField = () => {
    if (disabled) return
    setFields([...fields, { name: "", type: "string", description: "" }])
  }

  const removeField = (index: number) => {
    if (disabled) return
    const updated = fields.filter((_, i) => i !== index)
    setFields(updated)
    triggerChange(updated)
  }

  return (
    <div className="space-y-2.5">
      <div className="flex items-center justify-between">
        <label className="text-[10px] font-bold text-muted-foreground uppercase tracking-wide">
          {label}
        </label>
        {!disabled && (
          <button
            type="button"
            onClick={addField}
            className="text-[9px] px-1.5 py-0.5 rounded border border-primary/20 bg-primary/5 text-primary hover:bg-primary/10 transition-colors font-semibold flex items-center gap-1"
          >
            <Plus className="h-2.5 w-2.5" />
            Add Field
          </button>
        )}
      </div>

      {fields.length === 0 ? (
        <p className="text-[9px] text-muted-foreground/60 italic p-3 border border-dashed border-border rounded-lg text-center bg-muted/5">
          No schema fields configured.
        </p>
      ) : (
        <div className="space-y-2 max-h-64 overflow-y-auto pr-1">
          {fields.map((field, index) => (
            <div
              key={index}
              className="p-2.5 rounded-lg border border-border/50 bg-muted/15 space-y-1.5 relative group hover:border-border transition-colors"
            >
              {/* Row 1: Name, Type, and Delete */}
              <div className="flex items-center gap-1.5">
                <input
                  type="text"
                  value={field.name}
                  onChange={(e) => handleFieldChange(index, "name", e.target.value)}
                  placeholder="Field Name (e.g. run_data)"
                  disabled={disabled}
                  className="flex-1 h-7 rounded-md border border-input bg-transparent px-2 py-0.5 text-xs shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-primary disabled:opacity-50 disabled:cursor-not-allowed font-mono"
                />
                
                <select
                  value={field.type}
                  onChange={(e) => handleFieldChange(index, "type", e.target.value)}
                  disabled={disabled}
                  className="w-24 h-7 rounded-md border border-input bg-card px-1.5 py-0.5 text-[11px] shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-primary disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {SUPPORTED_TYPES.map((t) => (
                    <option key={t} value={t}>
                      {t}
                    </option>
                  ))}
                </select>

                {!disabled && (
                  <button
                    type="button"
                    onClick={() => removeField(index)}
                    className="p-1 hover:bg-destructive/10 text-muted-foreground hover:text-destructive rounded transition-colors shrink-0"
                    title="Remove Field"
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                  </button>
                )}
              </div>

              {/* Row 2: Description */}
              <div className="flex items-center gap-1.5">
                <input
                  type="text"
                  value={field.description}
                  onChange={(e) => handleFieldChange(index, "description", e.target.value)}
                  placeholder="Description (e.g. Preferred units: miles or km)"
                  disabled={disabled}
                  className="flex-1 h-7 rounded-md border border-input/40 bg-transparent px-2 py-0.5 text-[10px] text-muted-foreground placeholder:text-muted-foreground/55 shadow-none focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-primary/50 disabled:opacity-50 disabled:cursor-not-allowed"
                />
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
