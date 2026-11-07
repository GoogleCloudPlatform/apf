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

import { BaseNode } from "./BaseNode"
import { Bot } from "lucide-react"

interface LlmNodeProps {
  id: string
  data: {
    name: string
    displayName?: string
    model?: string
    description?: string
    [key: string]: any
  }
  selected?: boolean
}

export const LlmNode = ({ id, data, selected }: LlmNodeProps) => {
  return (
    <BaseNode
      id={id}
      selected={selected}
      data={data}
      icon={Bot}
      iconColorClass="text-blue-500 dark:text-blue-400"
      themeColorClass="border-blue-500 ring-blue-500/20"
      bgClass="bg-blue-50/70 border-blue-100 dark:bg-slate-900/90 dark:border-slate-800/90"
      label={data.displayName || data.name}
      subtitle={data.model || "gemini-2.5-flash"}
    />
  )
}
