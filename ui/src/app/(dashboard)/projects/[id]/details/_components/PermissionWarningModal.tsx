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

import React from "react"
import { Loader2, CheckCircle2, AlertTriangle, AlertCircle } from "lucide-react"

interface PermissionWarningModalProps {
  isOpen: boolean
  onClose: () => void
  onConfirm: () => void
  permissions: { read: boolean; write: boolean; pullRequest: boolean } | null
  isPending: boolean
}

export default function PermissionWarningModal({
  isOpen,
  onClose,
  onConfirm,
  permissions,
  isPending,
}: PermissionWarningModalProps) {
  if (!isOpen || !permissions) return null

  const items = [
    { 
      label: "Read Access", 
      desc: "Repository reading and browsing capabilities", 
      value: permissions.read 
    },
    { 
      label: "Write Access", 
      desc: "Ability to commit changes, modify files, and update branches", 
      value: permissions.write 
    },
    { 
      label: "Pull Request Access", 
      desc: "Creating, merging, and managing pull requests", 
      value: permissions.pullRequest 
    },
  ]

  let messageText = ""
  if (!permissions.write && !permissions.pullRequest) {
    messageText = "You dont have write access to the code repo. You will not be able to sync your changes and collaborate with others."
  } else if (!permissions.pullRequest) {
    messageText = "You dont have permission to submit pull request. If your orgnaization prevents direct pushes, you will need this permission to collbaorate effectively."
  } else if (!permissions.write) {
    messageText = "You dont have write access to the code repo. You will not be able to sync your changes and collaborate with others."
  } else {
    messageText = "You are attempting to perform an action that requires additional permissions. Please review the missing accesses below before proceeding."
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm select-none p-4">
      <div className="w-full max-w-[440px] rounded-2xl border border-border bg-card p-6 shadow-2xl text-card-foreground animate-in zoom-in-95 duration-200">
        
        <h3 className="text-base font-bold text-foreground mb-4">
          Missing Permissions
        </h3>

        <div className="flex items-start gap-3 mb-6">
          <div className="h-9 w-9 rounded-full bg-amber-500/10 flex items-center justify-center text-amber-500 shrink-0 mt-0.5">
            <AlertTriangle className="h-5 w-5" />
          </div>
          <p className="text-xs text-muted-foreground leading-relaxed text-left">
            {messageText}
          </p>
        </div>
        
        <div className="border border-border rounded-xl overflow-hidden mb-6 bg-background/50 divide-y divide-border/60">
          {items.map((item, idx) => (
            <div key={idx} className="flex items-start gap-3 p-3.5 transition-colors hover:bg-muted/30">
              {item.value ? (
                <CheckCircle2 className="h-5 w-5 text-emerald-500 shrink-0 mt-0.5" />
              ) : (
                <AlertCircle className="h-5 w-5 text-amber-500 shrink-0 mt-0.5" />
              )}
              <div className="flex flex-col text-left">
                <span className={`text-xs font-semibold ${item.value ? "text-foreground" : "text-foreground/90"}`}>
                  {item.label}
                </span>
                <span className="text-[10px] text-muted-foreground/80 mt-0.5 leading-normal">
                  {item.desc}
                </span>
              </div>
            </div>
          ))}
        </div>

        <div className="flex justify-end gap-2.5 pt-2">
          <button
            type="button"
            onClick={onClose}
            className="inline-flex h-9 items-center justify-center rounded-lg border border-input bg-background px-4 py-2 text-xs font-semibold text-foreground hover:bg-muted transition-colors"
          >
            Cancel
          </button>
          
          <button
            type="button"
            onClick={onConfirm}
            disabled={isPending}
            className="inline-flex h-9 items-center justify-center gap-1.5 rounded-lg bg-primary px-4 py-2 text-xs font-semibold text-primary-foreground shadow hover:opacity-90 disabled:opacity-50 transition-colors"
          >
            {isPending ? (
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
            ) : (
              "Save Anyway"
            )}
          </button>
        </div>
      </div>
    </div>
  )
}

