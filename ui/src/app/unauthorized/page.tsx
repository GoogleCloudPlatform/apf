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

import { Suspense } from "react"
import { useSearchParams } from "next/navigation"
import { UnauthorizedScreen } from "@/components/shared/UnauthorizedScreen"
import { Loader2 } from "lucide-react"

function UnauthorizedContent() {
  const searchParams = useSearchParams()
  const reason = searchParams.get("reason") || undefined
  const required = searchParams.get("required") || undefined

  let message: string | undefined
  if (reason === "superadmin") {
    message = "This section is restricted exclusively to Super Administrators."
  } else if (reason === "org_required") {
    message = "You must be a member of an active organization to view this resource."
  }

  return (
    <UnauthorizedScreen
      message={message}
      requiredPermission={required}
      fullPage
    />
  )
}

export default function UnauthorizedPage() {
  return (
    <Suspense
      fallback={
        <div className="flex min-h-screen items-center justify-center bg-slate-50 dark:bg-slate-950 p-6">
          <Loader2 className="h-8 w-8 animate-spin text-primary" />
        </div>
      }
    >
      <UnauthorizedContent />
    </Suspense>
  )
}
