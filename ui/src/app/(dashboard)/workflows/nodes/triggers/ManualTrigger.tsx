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

import { TriggerNode } from "../../components/TriggerNode"
import { MousePointerClick } from "lucide-react"

interface ManualTriggerProps {
  id: string
  data: {
    name: string
    displayName?: string
    [key: string]: any
  }
  selected?: boolean
}

export const ManualTrigger = ({ id, data, selected }: ManualTriggerProps) => {
  return (
    <TriggerNode
      id={id}
      data={data}
      selected={selected}
      icon={MousePointerClick}
      themeClass="border-purple-500/70 ring-purple-500/20"
      iconClass="text-purple-400"
    />
  )
}
