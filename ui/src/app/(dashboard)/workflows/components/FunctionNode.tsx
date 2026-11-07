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
import { Braces } from "lucide-react"

interface FunctionNodeProps {
  id: string
  data: {
    name: string
    displayName?: string
    parameterBinding?: string
    description?: string
    [key: string]: any
  }
  selected?: boolean
}

export const FunctionNode = ({ id, data, selected }: FunctionNodeProps) => {
  return (
    <BaseNode
      id={id}
      selected={selected}
      data={data}
      icon={Braces}
      iconColorClass="text-amber-500 dark:text-amber-400"
      themeColorClass="border-amber-500 ring-amber-500/20"
      bgClass="bg-amber-50/70 border-amber-100 dark:bg-slate-900/90 dark:border-slate-800/90"
      label={data.displayName || data.name}
      subtitle={data.parameterBinding || "state"}
    />
  )
}
