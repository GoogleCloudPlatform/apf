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

interface KeyValueEditorProps {
  label: string
  values: Record<string, any> | undefined
  onChange: (newValues: Record<string, any> | undefined) => void
  disabled?: boolean
}

export const KeyValueEditor = ({ label, values = {}, onChange, disabled = false }: KeyValueEditorProps) => {
  const [pairs, setPairs] = useState<{ key: string; value: string }[]>(() => {
    return Object.entries(values || {}).map(([k, v]) => ({
      key: k,
      value: typeof v === "object" ? JSON.stringify(v) : String(v),
    }))
  })

  useEffect(() => {
    const newPairs = Object.entries(values || {}).map(([k, v]) => ({
      key: k,
      value: typeof v === "object" ? JSON.stringify(v) : String(v),
    }))
    const currentKeys = pairs.map(p => p.key).join(",")
    const incomingKeys = newPairs.map(p => p.key).join(",")
    const currentVals = pairs.map(p => p.value).join(",")
    const incomingVals = newPairs.map(p => p.value).join(",")
    if (currentKeys !== incomingKeys || currentVals !== incomingVals) {
      setPairs(newPairs)
    }
  }, [values])

  const handlePairChange = (index: number, field: "key" | "value", val: string) => {
    if (disabled) return
    const updated = [...pairs]
    updated[index][field] = val
    setPairs(updated)

    const result: Record<string, any> = {}
    updated.forEach((p) => {
      if (p.key.trim()) {
        let parsedValue: any = p.value
        const trimmedVal = p.value.trim()
        if (
          (trimmedVal.startsWith("{") && trimmedVal.endsWith("}")) ||
          (trimmedVal.startsWith("[") && trimmedVal.endsWith("]"))
        ) {
          try {
            parsedValue = JSON.parse(trimmedVal)
          } catch (e) {}
        } else if (trimmedVal === "true") {
          parsedValue = true
        } else if (trimmedVal === "false") {
          parsedValue = false
        } else if (!isNaN(Number(trimmedVal)) && trimmedVal !== "") {
          parsedValue = Number(trimmedVal)
        }
        result[p.key.trim()] = parsedValue
      }
    })
    onChange(Object.keys(result).length > 0 ? result : undefined)
  }

  const addPair = () => {
    if (disabled) return
    setPairs([...pairs, { key: "", value: "" }])
  }

  const removePair = (index: number) => {
    if (disabled) return
    const updated = pairs.filter((_, i) => i !== index)
    setPairs(updated)

    const result: Record<string, any> = {}
    updated.forEach((p) => {
      if (p.key.trim()) {
        let parsedValue: any = p.value
        const trimmedVal = p.value.trim()
        if (
          (trimmedVal.startsWith("{") && trimmedVal.endsWith("}")) ||
          (trimmedVal.startsWith("[") && trimmedVal.endsWith("]"))
        ) {
          try {
            parsedValue = JSON.parse(trimmedVal)
          } catch (e) {}
        } else if (trimmedVal === "true") {
          parsedValue = true
        } else if (trimmedVal === "false") {
          parsedValue = false
        } else if (!isNaN(Number(trimmedVal)) && trimmedVal !== "") {
          parsedValue = Number(trimmedVal)
        }
        result[p.key.trim()] = parsedValue
      }
    })
    onChange(Object.keys(result).length > 0 ? result : undefined)
  }

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between">
        <label className="text-[10px] font-bold text-muted-foreground uppercase tracking-wide">
          {label}
        </label>
        {!disabled && (
          <button
            type="button"
            onClick={addPair}
            className="text-[9px] px-1.5 py-0.5 rounded border border-primary/20 bg-primary/5 text-primary hover:bg-primary/10 transition-colors font-semibold flex items-center gap-1"
          >
            <Plus className="h-2.5 w-2.5" />
            Add Parameter
          </button>
        )}
      </div>
      
      {pairs.length === 0 ? (
        <p className="text-[9px] text-muted-foreground/60 italic p-2 border border-dashed border-border rounded-lg text-center">
          No parameters configured.
        </p>
      ) : (
        <div className="space-y-1.5 max-h-48 overflow-y-auto pr-1">
          {pairs.map((pair, index) => (
            <div key={index} className="flex items-center gap-1.5">
              <input
                type="text"
                value={pair.key}
                onChange={(e) => handlePairChange(index, "key", e.target.value)}
                placeholder="Key"
                disabled={disabled}
                className="flex-1 h-8 rounded-md border border-input bg-transparent px-2 py-1 text-xs shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-primary disabled:opacity-50 disabled:cursor-not-allowed"
              />
              <span className="text-muted-foreground text-xs">:</span>
              <input
                type="text"
                value={pair.value}
                onChange={(e) => handlePairChange(index, "value", e.target.value)}
                placeholder="Value"
                disabled={disabled}
                className="flex-[1.5] h-8 rounded-md border border-input bg-transparent px-2 py-1 text-xs shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-primary disabled:opacity-50 disabled:cursor-not-allowed"
              />
              {!disabled && (
                <button
                  type="button"
                  onClick={() => removePair(index)}
                  className="p-1 hover:bg-destructive/10 text-muted-foreground hover:text-destructive rounded transition-colors shrink-0"
                >
                  <Trash2 className="h-3.5 w-3.5" />
                </button>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
