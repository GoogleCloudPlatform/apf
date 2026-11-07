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
import { X } from "lucide-react"
import ScrollArea from "@/components/ui/scroll-area"
import { availableEmojis } from "../_hooks/useProjectDetails"

interface EmojiIconPickerModalProps {
  isOpen: boolean
  onClose: () => void
  selectedIcon: string
  onSelectIcon: (emoji: string) => void
}

export default function EmojiIconPickerModal({
  isOpen,
  onClose,
  selectedIcon,
  onSelectIcon,
}: EmojiIconPickerModalProps) {
  if (!isOpen) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm select-none">
      <div className="w-full max-w-[500px] rounded-xl border border-border bg-card p-6 shadow-2xl text-card-foreground animate-in zoom-in-95 duration-200">
        <div className="flex items-center justify-between border-b border-border pb-3 mb-4">
          <h3 className="text-sm font-bold text-foreground">Select Project Icon Symbol</h3>
          <button
            type="button"
            onClick={onClose}
            className="flex h-6 w-6 items-center justify-center rounded hover:bg-muted text-muted-foreground"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        <ScrollArea className="h-[280px]">
          <div className="grid grid-cols-8 gap-2.5 p-1">
            {availableEmojis.map((emoji) => {
              const isSelected = selectedIcon === emoji
              return (
                <button
                  key={emoji}
                  type="button"
                  onClick={() => {
                    onSelectIcon(emoji)
                    onClose()
                  }}
                  className={`flex h-10 w-10 items-center justify-center rounded-lg text-2xl hover:bg-primary/10 transition-colors border ${
                    isSelected ? "border-primary bg-primary/5" : "border-transparent"
                  }`}
                >
                  {emoji}
                </button>
              )
            })}
          </div>
        </ScrollArea>
      </div>
    </div>
  )
}
