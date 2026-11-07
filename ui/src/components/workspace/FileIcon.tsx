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

import React, { useState } from "react"
import { File } from "lucide-react"
import { cn } from "@/lib/utils"
import { getFileIconMeta, FileIconMeta } from "@/lib/fileIconUtils"

export interface FileIconProps extends React.HTMLAttributes<HTMLSpanElement> {
  filename: string
  className?: string
  showBadge?: boolean
}

export const FileIcon: React.FC<FileIconProps> = ({
  filename,
  className = "h-4 w-4",
  showBadge = false,
  ...props
}) => {
  const [imgError, setImgError] = useState(false)
  const meta: FileIconMeta = getFileIconMeta(filename)

  if (showBadge) {
    return (
      <span
        className={cn(
          "inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-[10px] font-mono border",
          meta.badgeBgClass,
          className
        )}
        {...props}
      >
        {!imgError && meta.iconPath ? (
          <img
            src={meta.iconPath}
            alt={meta.language}
            className="h-3 w-3 shrink-0"
            onError={() => setImgError(true)}
          />
        ) : (
          <File className={cn("h-3 w-3 shrink-0", meta.textColorClass)} />
        )}
        <span>.{meta.extension || "file"}</span>
      </span>
    )
  }

  return (
    <span
      className={cn("inline-flex items-center justify-center shrink-0", meta.textColorClass, className)}
      title={`${filename} (${meta.language})`}
      {...props}
    >
      {!imgError && meta.iconPath ? (
        <img
          src={meta.iconPath}
          alt={meta.language}
          className={cn("object-contain shrink-0", className)}
          onError={() => setImgError(true)}
        />
      ) : (
        <File className={cn("shrink-0", className, meta.textColorClass)} />
      )}
    </span>
  )
}

export default FileIcon
