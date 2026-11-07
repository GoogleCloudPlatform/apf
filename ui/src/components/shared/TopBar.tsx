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

import { useState } from "react"
import { useAppStore } from "@/stores/app"
import { useAuthStore } from "@/stores/auth"
import { useThemeStore } from "@/stores/theme"
import { useActivityStore } from "@/stores/activity"
import { useOnboardingStore } from "@/stores/onboarding"
import { usePathname, useSearchParams, useRouter } from "next/navigation"
import { useQuery } from "@tanstack/react-query"
import { Project, FeedbackType } from "@/lib/types"
import { api } from "@/lib/api"
import { useFeature } from "@/hooks/useFeature"
import { FeedbackModal } from "./FeedbackModal"
import { 
  HelpCircle, 
  Bell, 
  Sun, 
  Moon, 
  LogOut, 
  User as UserIcon,
  Sparkles,
  Building2,
  Tag
} from "lucide-react"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"

export function TopBar() {
  const pathname = usePathname()
  const searchParams = useSearchParams()
  const router = useRouter()
  const rawProjectId = pathname.split("/projects/")[1]?.split("/")[0] || null
  const projectId = rawProjectId && rawProjectId.trim() !== "" ? rawProjectId : null
  const isValidProjectId = projectId && projectId !== "new" && projectId !== "create" && /^[a-zA-Z0-9_-]+$/.test(projectId)

  const { data: project } = useQuery<Project>({
    queryKey: ["project", projectId],
    queryFn: async () => {
      if (!projectId) throw new Error("No project ID")
      const res = await api.get(`/api/v1/projects/${projectId}`)
      if (!res.ok) throw new Error("Failed to load project details")
      return res.json()
    },
    enabled: !!isValidProjectId
  })

  const { pageTitle } = useAppStore()
  const { email, displayName, logout, switchOrganization, organizations, selectedOrgId } = useAuthStore()
  const { themeMode, toggleTheme } = useThemeStore()
  const { unreadCount, togglePanel } = useActivityStore()
  const { startTour, getSectionFirstStepIndex } = useOnboardingStore()

  const { enabled: isBannersEnabled, properties: bannerProps } = useFeature<{
    show_alpha_notice?: boolean
    show_version?: boolean
  }>("app_banners")
  const showAlphaNotice = isBannersEnabled && Boolean(bannerProps?.show_alpha_notice)
  const showVersion = isBannersEnabled && Boolean(bannerProps?.show_version)

  const { data: versionData } = useQuery<{ version: string }>({
    queryKey: ["app-version"],
    queryFn: async () => {
      const res = await api.get("/api/v1/version")
      if (!res.ok) throw new Error("Failed to load version")
      return res.json()
    },
    enabled: showVersion,
    staleTime: 10 * 60 * 1000,
  })

  const rawVersion = typeof versionData?.version === "string" ? versionData.version.trim() : (versionData?.version ? String(versionData.version) : "")
  const appVersion = rawVersion ? (rawVersion.startsWith("v") ? rawVersion : `v${rawVersion}`) : ""

  const [isFeedbackOpen, setIsFeedbackOpen] = useState(false)
  const [feedbackType, setFeedbackType] = useState<FeedbackType>("general")

  const handleOpenFeedback = (type: FeedbackType) => {
    setFeedbackType(type)
    setIsFeedbackOpen(true)
  }

  const userInitial = email && email.length > 0 ? email[0].toUpperCase() : "?"
  const activeOrg = organizations.find(o => o.id === selectedOrgId)
  const orgName = activeOrg ? activeOrg.name : ""

  return (
    <header className="flex h-16 w-full items-center justify-between px-6 border-b border-border bg-slate-100 dark:bg-slate-900 shrink-0">
      {/* Title Subtitle Breadcrumbs block */}
      <div className="flex items-center gap-4">
        <span className="text-sm font-semibold text-muted-foreground tracking-wide uppercase">
          {pageTitle}
        </span>
      </div>

      {/* Alpha Banner Notice & Version Label */}
      {(showAlphaNotice || (showVersion && Boolean(appVersion))) && (
        <div className="flex items-center gap-2">
          {/* Alpha Banner Notice */}
          {showAlphaNotice && (
            <div className="flex items-center gap-2 px-3.5 py-1.5 rounded-lg border border-border bg-white dark:bg-slate-800 shadow-sm text-xs text-foreground font-medium">
              <Sparkles className="h-4 w-4 text-blue-500 shrink-0" />
              <span>
                You&apos;re using the alpha version of APF.{" "}
                <a
                  href="https://github.com/GoogleCloudPlatform/apf"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-600 dark:text-blue-400 hover:underline font-medium cursor-pointer"
                  data-testid="banner-learn-more-link"
                >
                  Learn more
                </a>{" "}
                or{" "}
                <a
                  href="#"
                  onClick={(e) => {
                    e.preventDefault()
                    handleOpenFeedback("bug")
                  }}
                  className="text-blue-600 dark:text-blue-400 hover:underline font-medium cursor-pointer"
                  data-testid="banner-file-bug-link"
                >
                  file a bug
                </a>
              </span>
            </div>
          )}

          {/* Application Version Label */}
          {showVersion && appVersion && (
            <div
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-border bg-white dark:bg-slate-800 shadow-sm text-xs text-foreground font-medium"
              data-testid="topbar-version-label"
            >
              <Tag className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
              <span>{appVersion}</span>
            </div>
          )}
        </div>
      )}



      {/* Action Operations Tray */}
      <div className="flex items-center gap-3">
        {/* Quick Tour Action button */}
        <button
          onClick={() => {
            if (pathname.includes("/workspace")) {
              startTour("workspace")
            } else if (pathname.includes("/settings")) {
              startTour("settings")
            } else if (pathname.includes("/details")) {
              startTour("catalog", getSectionFirstStepIndex("catalog", "project-creation"))
            } else {
              startTour("catalog")
            }
          }}
          className="flex h-9 w-9 items-center justify-center rounded-lg text-purple-600 dark:text-purple-400 hover:bg-purple-500/10 transition-all relative group"
          title="Start Quick Tour"
        >
          <Sparkles className="h-4.5 w-4.5 animate-pulse" />
        </button>

        {/* Help Action button */}
        <button
          onClick={() => router.push("/docs")}
          className="flex h-9 w-9 items-center justify-center rounded-lg text-muted-foreground hover:bg-muted hover:text-foreground transition-colors"
          title="Help & Documentation"
          data-testid="topbar-docs-search-button"
        >
          <HelpCircle className="h-4.5 w-4.5" />
        </button>

        {/* Dynamic Notification Bell and Badge count */}
        <button
          onClick={togglePanel}
          className="relative flex h-9 w-9 items-center justify-center rounded-lg text-muted-foreground hover:bg-muted hover:text-foreground transition-colors"
          title="Toggle System Activities Feed"
        >
          <Bell className="h-4.5 w-4.5" />
          {unreadCount > 0 && (
            <span className="absolute top-1.5 right-1.5 flex h-4 min-w-[16px] items-center justify-center rounded-full bg-destructive px-1 text-[9px] font-bold text-destructive-foreground ring-2 ring-slate-100 dark:ring-slate-900">
              {unreadCount}
            </span>
          )}
        </button>

        {/* Theme dynamic toggle switch button */}
        <button
          onClick={toggleTheme}
          className="flex h-9 w-9 items-center justify-center rounded-lg text-muted-foreground hover:bg-muted hover:text-foreground transition-colors"
          title={`Switch Theme (Current: ${themeMode})`}
        >
          {themeMode === "light" ? (
            <Moon className="h-4.5 w-4.5" />
          ) : (
            <Sun className="h-4.5 w-4.5" />
          )}
        </button>

        {/* Vertical boundary spacer */}
        <span className="h-5 w-px bg-border my-auto mx-1" />

        {/* Account drop-down menu block */}
        <DropdownMenu>
          <DropdownMenuTrigger className="focus:outline-none select-none">
            <div className="flex h-8 w-8 items-center justify-center rounded-full bg-primary text-primary-foreground font-bold text-sm hover:opacity-90 transition-opacity">
              {userInitial}
            </div>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-56 mt-2 rounded-lg p-1.5 shadow-lg">
            <DropdownMenuLabel className="px-2.5 py-2">
              <div className="flex flex-col gap-0.5">
                <span className="text-sm font-bold text-foreground truncate">
                  {displayName || "Foundry User"}
                </span>
                {email && (
                  <span className="text-xs text-muted-foreground truncate">
                    {email}
                  </span>
                )}
              </div>
            </DropdownMenuLabel>
            <DropdownMenuSeparator className="my-1" />
            <DropdownMenuItem
              onClick={() => {
                switchOrganization()
                router.push("/select-org")
              }}
              className="flex items-center gap-2 px-2.5 py-2 text-foreground hover:bg-accent hover:text-accent-foreground focus:bg-accent focus:text-accent-foreground rounded cursor-pointer"
            >
              <Building2 className="h-4 w-4 shrink-0 text-muted-foreground" />
              <div className="flex flex-col min-w-0 text-left">
                <span className="text-sm font-medium truncate">
                  {orgName || "Select Organization"}
                </span>
                <span className="text-[10px] text-muted-foreground font-medium">
                  Switch Organization
                </span>
              </div>
            </DropdownMenuItem>
            <DropdownMenuItem
              onClick={() => logout()}
              className="flex items-center gap-2 px-2.5 py-2 text-sm text-destructive hover:bg-destructive/10 hover:text-destructive focus:bg-destructive/10 focus:text-destructive rounded cursor-pointer"
            >
              <LogOut className="h-4 w-4" />
              <span>Logout</span>
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>

      <FeedbackModal
        isOpen={isFeedbackOpen}
        onClose={() => setIsFeedbackOpen(false)}
        initialType={feedbackType}
        projectId={projectId}
      />
    </header>
  )
}
export default TopBar

