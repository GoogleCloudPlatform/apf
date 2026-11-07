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

interface JoinNodeAttributesProps {
  data: Record<string, any>
  onChange: (key: string, value: any) => void
}

export const JoinNodeAttributes = ({ data, onChange }: JoinNodeAttributesProps) => (
  <div className="space-y-1.5 border-t border-border/50 pt-4">
    <p className="text-[10px] text-muted-foreground leading-normal italic">
      Join nodes consolidate multiple execution paths. There are no node-specific attributes to configure.
    </p>
  </div>
)
