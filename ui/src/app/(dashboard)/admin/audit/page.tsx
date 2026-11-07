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

import { useEffect } from "react"
import { useAuthStore } from "@/stores/auth"
import { useAppStore } from "@/stores/app"
import { Activity, ShieldAlert, Loader2 } from "lucide-react"

export default function AdminAuditPage() {
  const { setPageTitle } = useAppStore()
  const { isSuperAdmin, isCheckingAuth } = useAuthStore()

  // Sync page title
  useEffect(() => {
    setPageTitle("Admin: Audit Trail")
  }, [setPageTitle])

  if (isCheckingAuth) {
    return (
      <div className="flex-1 flex h-[400px] w-full flex-col items-center justify-center gap-3 text-muted-foreground select-none">
        <Loader2 className="h-8 w-8 animate-spin text-primary" />
        <span className="text-sm font-medium">Verifying access...</span>
      </div>
    )
  }

  // Guard: Only super admins can access
  if (!isSuperAdmin) {
    return (
      <div className="flex-1 flex flex-col items-center justify-center p-8 min-h-[calc(100vh-4rem)] select-none">
        <div className="w-full max-w-md bg-destructive/10 border border-destructive/20 rounded-2xl p-8 text-center space-y-6">
          <div className="flex justify-center text-destructive">
            <ShieldAlert className="h-16 w-16 animate-bounce" />
          </div>
          <div className="space-y-2">
            <h3 className="text-xl font-bold text-foreground">
              Access Restricted
            </h3>
            <p className="text-sm text-muted-foreground leading-relaxed">
              This screen is reserved exclusively for Super Administrators.
            </p>
          </div>
        </div>
      </div>
    )
  }

  return (
    <main className="flex-1 overflow-y-auto p-8">
      <div className="w-full max-w-[800px] space-y-8 pb-12">
        <div className="space-y-2 border-b border-border pb-4">
          <div className="flex items-center gap-3">
            <Activity className="h-6 w-6 text-primary" />
            <h2 className="text-lg font-bold text-foreground">Audit Trail Logs</h2>
          </div>
          <p className="text-xs text-muted-foreground leading-relaxed">
            Real-time security log recording all administrative modifications, policy edits, and access changes.
          </p>
        </div>

        <div className="bg-card border border-border rounded-2xl overflow-hidden shadow-sm">
          <table className="w-full border-collapse text-left">
            <thead>
              <tr className="border-b border-border bg-slate-50 dark:bg-slate-900/50 text-[10px] font-bold uppercase tracking-wider text-muted-foreground select-none">
                <th className="p-4">User</th>
                <th className="p-4">Action Event</th>
                <th className="p-4">Resource</th>
                <th className="p-4">Timestamp</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border/60 text-xs">
              <tr>
                <td className="p-4 font-semibold text-foreground">super-admin@agentic-foundry.local</td>
                <td className="p-4 text-muted-foreground">Enabled GitHub Integration</td>
                <td className="p-4 font-mono text-[10px]">Config: app-integrations</td>
                <td className="p-4 text-muted-foreground">Just now</td>
              </tr>
              <tr>
                <td className="p-4 font-semibold text-foreground">admin@agentic-foundry.local</td>
                <td className="p-4 text-muted-foreground">Updated member role to Owner</td>
                <td className="p-4 font-mono text-[10px]">User: member@agentic-foundry.local</td>
                <td className="p-4 text-muted-foreground">10 mins ago</td>
              </tr>
              <tr>
                <td className="p-4 font-semibold text-foreground">super-admin@agentic-foundry.local</td>
                <td className="p-4 text-muted-foreground">Updated Inactivity Session Timeout</td>
                <td className="p-4 font-mono text-[10px]">Policy: session_inactivity</td>
                <td className="p-4 text-muted-foreground">2 hours ago</td>
              </tr>
              <tr>
                <td className="p-4 font-semibold text-foreground">admin@agentic-foundry.local</td>
                <td className="p-4 text-muted-foreground">Provisioned Showcase Project</td>
                <td className="p-4 font-mono text-[10px]">Project: foundry-showcase-project</td>
                <td className="p-4 text-muted-foreground">1 day ago</td>
              </tr>
              <tr>
                <td className="p-4 font-semibold text-foreground">system-agent@agentic-foundry.local</td>
                <td className="p-4 text-muted-foreground">Rotated API Security Tokens</td>
                <td className="p-4 font-mono text-[10px]">System: database_credentials</td>
                <td className="p-4 text-muted-foreground">3 days ago</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </main>
  )
}
