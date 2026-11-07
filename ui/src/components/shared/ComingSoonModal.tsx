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
import { PartyPopper } from "lucide-react"

interface ComingSoonModalProps {
  isOpen: boolean
  onClose: () => void
  featureName?: string
}

export default function ComingSoonModal({ isOpen, onClose, featureName = "Dynamic workflows" }: ComingSoonModalProps) {
  if (!isOpen) return null

  return (
    <div className="fixed inset-0 z-[100] flex items-center justify-center bg-black/60 backdrop-blur-sm p-4 select-none">
      <div className="bg-card border border-border rounded-2xl max-w-sm w-full p-6 shadow-2xl space-y-6 animate-in fade-in zoom-in-95 duration-200 flex flex-col items-center text-center">
        <div className="h-12 w-12 rounded-full bg-primary/10 flex items-center justify-center text-primary">
          <PartyPopper className="h-6 w-6" />
        </div>
        
        <div className="space-y-2">
          <h3 className="text-lg font-bold text-foreground">Coming Soon!</h3>
          <p className="text-sm text-muted-foreground leading-relaxed">
            {featureName} feature is coming soon!
          </p>
        </div>

        <button
          type="button"
          onClick={onClose}
          className="w-full h-10 inline-flex items-center justify-center rounded-xl bg-primary px-4 text-xs font-semibold text-primary-foreground shadow hover:opacity-90 transition-colors"
        >
          Got it
        </button>
      </div>
    </div>
  )
}
