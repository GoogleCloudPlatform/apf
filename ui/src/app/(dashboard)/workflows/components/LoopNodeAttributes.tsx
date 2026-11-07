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
import { LogicBuilder } from "./LogicBuilder"

interface LoopNodeAttributesProps {
  data: {
    name: string
    exitCondition?: Record<string, any>
    [key: string]: any
  }
  onChange: (key: string, value: any) => void
}

export const LoopNodeAttributes = ({ data, onChange }: LoopNodeAttributesProps) => {
  const [textValue, setTextValue] = useState("")
  const [error, setError] = useState("")
  const [isBuilderOpen, setIsBuilderOpen] = useState(false)

  // Sync from prop to local state on load or change
  useEffect(() => {
    if (data.exitCondition) {
      setTextValue(JSON.stringify(data.exitCondition, null, 2))
    } else {
      // Default placeholder/example
      const defaultCondition = {
        "==": [
          { "var": `_loop_${data.name || "loop_items"}_count` },
          3
        ]
      }
      setTextValue(JSON.stringify(defaultCondition, null, 2))
    }
  }, [data.exitCondition, data.name])

  const handleTextChange = (val: string) => {
    setTextValue(val)
    if (!val.trim()) {
      setError("")
      onChange("exitCondition", undefined)
      return
    }

    try {
      const parsed = JSON.parse(val)
      setError("")
      onChange("exitCondition", parsed)
    } catch (e: any) {
      setError(`Invalid JSON: ${e.message}`)
    }
  }

  return (
    <div className="space-y-2 border-t border-border/50 pt-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <label className="text-[10px] font-bold text-muted-foreground uppercase tracking-wide">
            Exit Condition
          </label>
          <button
            type="button"
            onClick={() => setIsBuilderOpen(true)}
            className="text-[9px] px-1.5 py-0.5 rounded border border-primary/20 bg-primary/5 text-primary hover:bg-primary/10 transition-colors font-semibold"
          >
            Logic Builder
          </button>
        </div>
        {error && (
          <span className="text-[9px] text-destructive font-semibold">
            {error}
          </span>
        )}
      </div>
      <textarea
        value={textValue}
        onChange={(e) => handleTextChange(e.target.value)}
        placeholder='{\n  "==": [\n    { "var": "_loop_count" },\n    3\n  ]\n}'
        rows={6}
        className={`flex w-full rounded-lg border bg-transparent px-3 py-2 text-xs shadow-sm focus-visible:outline-none focus-visible:ring-1 resize-y font-mono ${
          error ? "border-destructive focus-visible:ring-destructive" : "border-input focus-visible:ring-primary"
        }`}
      />
      <p className="text-[9px] text-muted-foreground/80 leading-normal">
        Specify when to EXIT the loop. The loop will exit when this condition evaluates to <b>true</b>.
        <br />
        Example: <code>{`{"==": [{"var": "_loop_${data.name || "loop_items"}_count"}, 3]}`}</code> exits after 3 iterations (count reaches 3).
      </p>

      <LogicBuilder
        isOpen={isBuilderOpen}
        onClose={() => setIsBuilderOpen(false)}
        onSave={(logic) => {
          const jsonStr = JSON.stringify(logic, null, 2)
          setTextValue(jsonStr)
          setError("")
          onChange("exitCondition", logic)
        }}
        initialLogic={data.exitCondition}
        loopNodeName={data.name}
      />
    </div>
  )
}
