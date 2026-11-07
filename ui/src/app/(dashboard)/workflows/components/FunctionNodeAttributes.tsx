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

interface FunctionNodeAttributesProps {
  data: {
    parameterBinding?: string
    [key: string]: any
  }
  onChange: (key: string, value: any) => void
}

export const FunctionNodeAttributes = ({ data, onChange }: FunctionNodeAttributesProps) => (
  <div className="space-y-1.5 border-t border-border/50 pt-4">
    <label className="text-[10px] font-bold text-muted-foreground uppercase tracking-wide">
      Parameter Binding
    </label>
    <select
      value={data.parameterBinding || "state"}
      onChange={(e) => onChange("parameterBinding", e.target.value)}
      className="flex h-9 w-full rounded-lg border border-input bg-card px-3 py-1 text-xs shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-primary"
    >
      <option value="state">Context State (Default)</option>
      <option value="node_input">Node Input (Tool-mode)</option>
    </select>
    <p className="text-[9px] text-muted-foreground/80 leading-normal">
      State: binds parameters from ctx.state. Node Input: binds parameters from previous output.
    </p>
  </div>
)
