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

import { useState } from "react"
import { SchemaEditor } from "../../components/SchemaEditor"

interface FormTriggerAttributesProps {
  data: {
    input?: Record<string, any>
    output?: Record<string, any>
    [key: string]: any
  }
  onChange: (field: string, val: any) => void
}

export const FormTriggerAttributes = ({ data, onChange }: FormTriggerAttributesProps) => {
  const [activeTab, setActiveTab] = useState<"general" | "input_output">("general")

  return (
    <div className="space-y-4">
      {/* Tab Navigation Headers */}
      <div className="flex border-b border-border/60 shrink-0">
        <button
          type="button"
          onClick={() => setActiveTab("general")}
          className={`flex-1 pb-2 text-[10px] font-bold uppercase tracking-wider border-b-2 transition-all ${
            activeTab === "general"
              ? "border-primary text-primary"
              : "border-transparent text-muted-foreground hover:text-foreground"
          }`}
        >
          General
        </button>
        <button
          type="button"
          onClick={() => setActiveTab("input_output")}
          className={`flex-1 pb-2 text-[10px] font-bold uppercase tracking-wider border-b-2 transition-all ${
            activeTab === "input_output"
              ? "border-primary text-primary"
              : "border-transparent text-muted-foreground hover:text-foreground"
          }`}
        >
          Form Inputs
        </button>
      </div>

      {/* Tab Contents */}
      {activeTab === "general" && (
        <div className="space-y-3 pt-2">
          <h4 className="text-[10px] font-bold text-muted-foreground uppercase tracking-wider">
            Trigger Configuration
          </h4>
          <p className="text-xs text-muted-foreground/80 leading-normal bg-muted/30 p-3 rounded-lg border border-border/40">
            Form triggers allow you to start or pause a workflow by submitting a form. You can configure the form fields under <b>Form Inputs</b>. The form submission payload is passed through directly to downstream nodes as passthrough.
          </p>
        </div>
      )}

      {activeTab === "input_output" && (
        <div className="space-y-6">
          <SchemaEditor
            label="Form Inputs (Schema)"
            schema={data.input}
            onChange={(val) => onChange("input", val)}
          />
        </div>
      )}
    </div>
  )
}
