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

import { useRouter } from "next/navigation"
import { useAuthStore } from "@/stores/auth"
import { 
  ShieldAlert, 
  Lock, 
  ArrowLeft, 
  Building2, 
  LayoutGrid, 
  User, 
  Shield 
} from "lucide-react"

export interface UnauthorizedScreenProps {
  /** Title header text for access restriction */
  title?: string
  /** Detailed explanatory message for the user */
  message?: string
  /** Specific required permission or role name if available */
  requiredPermission?: string
  /** Whether to render the 'Go Back' browser history button */
  showGoBack?: boolean
  /** Whether to render the 'Switch Organization' button */
  showSwitchOrg?: boolean
  /** Whether to render the 'Return to Projects' button */
  showHome?: boolean
  /** Whether to render as full-screen viewport container (default: true) */
  fullPage?: boolean
}

export function UnauthorizedScreen({
  title = "Access Restricted",
  message = "You do not have the required permissions or role to access this section. Please contact your organization administrator if you believe this is an error.",
  requiredPermission,
  showGoBack = true,
  showSwitchOrg = true,
  showHome = true,
  fullPage = true,
}: UnauthorizedScreenProps) {
  const router = useRouter()
  const { email, selectedOrgId, isSuperAdmin, permissions } = useAuthStore()

  const handleGoBack = () => {
    if (window.history.length > 1) {
      router.back()
    } else {
      router.push("/projects")
    }
  }

  const content = (
    <div className="w-full max-w-[480px] rounded-2xl border border-border bg-card p-8 shadow-2xl text-card-foreground">
      <div className="flex flex-col items-center text-center">
        
        {/* Warning Icon Container */}
        <div className="relative flex h-16 w-16 items-center justify-center rounded-2xl border border-amber-500/30 bg-amber-500/10 text-amber-500 dark:border-amber-400/30 dark:bg-amber-400/10 dark:text-amber-400 mb-6 shadow-sm select-none">
          <ShieldAlert className="h-8 w-8" />
          <div className="absolute -bottom-1 -right-1 flex h-6 w-6 items-center justify-center rounded-full bg-destructive text-destructive-foreground ring-2 ring-card">
            <Lock className="h-3 w-3" />
          </div>
        </div>

        {/* Title & Description */}
        <h1 className="text-xl font-bold tracking-tight text-foreground mb-2">
          {title}
        </h1>
        <p className="text-xs text-muted-foreground leading-relaxed mb-6">
          {message}
        </p>

        {/* Account & Role Context Diagnostics */}
        <div className="w-full rounded-xl border border-border bg-muted/40 p-4 text-left mb-6 space-y-3 select-none">
          <div className="text-[10px] font-bold tracking-wider text-muted-foreground uppercase flex items-center gap-1.5 border-b border-border/60 pb-2">
            <User className="h-3.5 w-3.5 text-primary" />
            Active Session Context
          </div>

          <div className="space-y-2 text-xs">
            {/* User Email */}
            <div className="flex items-center justify-between">
              <span className="text-muted-foreground font-medium">Account:</span>
              <span className="font-semibold text-foreground truncate max-w-[240px]">
                {email || "Unknown User"}
              </span>
            </div>

            {/* Selected Organization */}
            <div className="flex items-center justify-between">
              <span className="text-muted-foreground font-medium">Organization:</span>
              <span className="font-mono text-[11px] text-foreground truncate max-w-[200px]">
                {selectedOrgId || "None selected"}
              </span>
            </div>

            {/* User Status / Role */}
            <div className="flex items-center justify-between">
              <span className="text-muted-foreground font-medium">Platform Role:</span>
              <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-md text-[10px] font-bold uppercase tracking-wider bg-primary/10 text-primary border border-primary/20">
                <Shield className="h-3 w-3" />
                {isSuperAdmin ? "Super Admin" : "Standard Member"}
              </span>
            </div>

            {/* Required Permission (if supplied) */}
            {requiredPermission && (
              <div className="flex items-center justify-between pt-1 border-t border-border/40">
                <span className="text-muted-foreground font-medium">Required Permission:</span>
                <span className="font-mono text-[10px] font-bold text-destructive bg-destructive/10 px-1.5 py-0.5 rounded border border-destructive/20">
                  {requiredPermission}
                </span>
              </div>
            )}
          </div>
        </div>

        {/* Action Controls */}
        <div className="flex flex-col sm:flex-row items-center justify-center gap-2.5 w-full">
          {showGoBack && (
            <button
              type="button"
              onClick={handleGoBack}
              className="flex-1 w-full h-10 inline-flex items-center justify-center gap-2 rounded-xl border border-input bg-background hover:bg-muted text-xs font-bold text-foreground transition-all shadow-xs"
            >
              <ArrowLeft className="h-3.5 w-3.5" />
              Go Back
            </button>
          )}

          {showHome && (
            <button
              type="button"
              onClick={() => router.push("/projects")}
              className="flex-1 w-full h-10 inline-flex items-center justify-center gap-2 rounded-xl bg-primary text-primary-foreground text-xs font-bold shadow-sm transition-all hover:opacity-90"
            >
              <LayoutGrid className="h-3.5 w-3.5" />
              Projects
            </button>
          )}

          {showSwitchOrg && (
            <button
              type="button"
              onClick={() => router.push("/select-org")}
              className="flex-1 w-full h-10 inline-flex items-center justify-center gap-2 rounded-xl border border-input bg-muted/40 hover:bg-muted text-xs font-bold text-foreground transition-all shadow-xs"
            >
              <Building2 className="h-3.5 w-3.5" />
              Switch Org
            </button>
          )}
        </div>

      </div>
    </div>
  )

  if (fullPage) {
    return (
      <main className="flex min-h-screen items-center justify-center bg-slate-50 dark:bg-slate-950 p-6">
        {content}
      </main>
    )
  }

  return (
    <div className="flex-1 flex items-center justify-center p-6 min-h-[calc(100vh-6rem)]">
      {content}
    </div>
  )
}

export default UnauthorizedScreen
