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

interface ManualTriggerAttributesProps {
  data: {
    [key: string]: any
  }
  onChange: (field: string, val: any) => void
}

export const ManualTriggerAttributes = ({ data, onChange }: ManualTriggerAttributesProps) => {
  return (
    <div className="space-y-3 pt-2 border-t border-border/50">
      <h4 className="text-[10px] font-bold text-muted-foreground uppercase tracking-wider">
        Trigger Configuration
      </h4>
      <p className="text-xs text-muted-foreground/80 leading-normal bg-muted/30 p-3 rounded-lg border border-border/40">
        Manual triggers do not require input or output parameters. When you click <b>Invoke</b>, the workflow begins executing immediately from this entry point.
      </p>
    </div>
  )
}
