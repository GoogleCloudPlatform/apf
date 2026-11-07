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

"use client"

import * as React from "react"
import * as ResizablePrimitive from "react-resizable-panels"
import { GripVertical } from "lucide-react"
import { cn } from "@/lib/utils"

const ResizablePanelGroup = ({
  className,
  ...props
}: React.ComponentProps<typeof ResizablePrimitive.PanelGroup>) => (
  <ResizablePrimitive.PanelGroup
    className={cn(
      "flex h-full w-full data-[panel-group-direction=vertical]:flex-col",
      className
    )}
    {...props}
  />
)

const ResizablePanel = ResizablePrimitive.Panel

interface ResizableHandleProps
  extends React.ComponentProps<typeof ResizablePrimitive.PanelResizeHandle> {
  withHandle?: boolean
}

/**
 * Custom resizable split pane handle.
 * Converted to functional component to bypass library ref interface issues.
 */
const ResizableHandle = ({
  withHandle,
  className,
  ...props
}: ResizableHandleProps) => (
  <ResizablePrimitive.PanelResizeHandle
    className={cn(
      "relative flex w-1.5 items-center justify-center bg-border transition-colors hover:bg-primary/30 focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring focus-visible:ring-offset-1 data-[panel-group-direction=vertical]:h-1.5 data-[panel-group-direction=vertical]:w-full data-[panel-group-direction=vertical]:border-y data-[panel-group-direction=vertical]:border-y-border data-[panel-group-direction=vertical]:after:left-0 data-[panel-group-direction=vertical]:after:right-0 data-[panel-group-direction=vertical]:after:top-1/2 data-[panel-group-direction=vertical]:after:h-2 data-[panel-group-direction=vertical]:after:-translate-y-1/2 data-[panel-group-direction=horizontal]:border-x data-[panel-group-direction=horizontal]:border-x-border data-[panel-group-direction=horizontal]:after:bottom-0 data-[panel-group-direction=horizontal]:after:top-0 data-[panel-group-direction=horizontal]:after:left-1/2 data-[panel-group-direction=horizontal]:after:w-2 data-[panel-group-direction=horizontal]:after:-translate-x-1/2",
      className
    )}
    {...props}
  >
    {withHandle && (
      <div className="z-10 flex h-4 w-3 items-center justify-center rounded-sm border bg-border text-muted-foreground hover:bg-muted focus:bg-muted">
        <GripVertical className="h-2.5 w-2.5" />
      </div>
    )}
  </ResizablePrimitive.PanelResizeHandle>
)

export { ResizablePanelGroup, ResizablePanel, ResizableHandle }
export default ResizablePanelGroup
