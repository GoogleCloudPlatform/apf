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

interface CustomNodeAttributesProps {
  data: {
    parallelWorker?: boolean
    [key: string]: any
  }
  onChange: (key: string, value: any) => void
}

export const CustomNodeAttributes = ({ data, onChange }: CustomNodeAttributesProps) => (
  <div className="flex items-center justify-between border-t border-border/50 pt-4">
    <div className="space-y-0.5">
      <label className="text-[11px] font-semibold text-foreground">
        Parallel Worker
      </label>
      <p className="text-[9px] text-muted-foreground/80">
        Spin up parallel executions for iterable inputs.
      </p>
    </div>
    <input
      type="checkbox"
      checked={!!data.parallelWorker}
      onChange={(e) => onChange("parallelWorker", e.target.checked)}
      className="h-4 w-4 rounded border-gray-300 text-primary focus:ring-primary"
    />
  </div>
)
