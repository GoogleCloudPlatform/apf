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

import { Loader2 } from "lucide-react"

/**
 * Global dynamic route suspense loader screen page fallback.
 */
export default function LoadingPage() {
  return (
    <div className="flex h-screen w-screen flex-col items-center justify-center bg-background text-foreground gap-4">
      <Loader2 className="h-10 w-10 animate-spin text-primary" />
      <span className="text-sm text-muted-foreground font-medium animate-pulse">
        Transitioning workspace paths...
      </span>
    </div>
  )
}
