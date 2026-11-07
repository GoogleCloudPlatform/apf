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

import { useState, useEffect } from "react"
import { usePathname, useRouter } from "next/navigation"
import { useAppStore } from "@/stores/app"
import { useAuthStore } from "@/stores/auth"
import Image from "next/image"
import api from "@/lib/api"
import { 
  FolderKanban, 
  Layers, 
  Settings, 
  Bug, 
  ChevronLeft, 
  ChevronRight,
  Users,
  Workflow,
  Cable,
  Building,
  Shield,
  Info,
  Activity,
  UserPlus,
  ArrowLeft,
  HardDrive,
  Cpu,
  Terminal
} from "lucide-react"

interface MenuItem {
  id: number
  label: string
  icon: React.ComponentType<{ className?: string }>
  path?: string
  action?: () => void
  activeState?: boolean
  visible: boolean
  isToggle?: boolean
}

export function LeftBar() {
  const router = useRouter()
  const pathname = usePathname()
  
  const { 
    lastSelectedProjectId, 
    isDebugBarOpen, 
    toggleDebugBar,
    isLeftBarExpanded: isExpanded,
    setIsLeftBarExpanded: setIsExpanded
  } = useAppStore()
  const { isOwner, isAdmin, isSuperAdmin, permissions } = useAuthStore()

  const [appVersion, setAppVersion] = useState<string>("")

  useEffect(() => {
    let isMounted = true
    api.get("/api/v1/version")
      .then(res => res.json())
      .then(data => {
        if (isMounted && data?.version) {
          const v = data.version.trim()
          setAppVersion(v.startsWith("v") ? v : `v${v}`)
        }
      })
      .catch(() => {
        // Ignore fetch errors cleanly
      })
    return () => {
      isMounted = false
    }
  }, [])

  const userPermissions = permissions || []
  const hasPermission = (perm: string) => userPermissions.includes(perm)

  const isWorkspaceScreen = pathname.endsWith("/workspace")
  const hasActiveProject = lastSelectedProjectId !== null
  const isAdminMode = pathname.startsWith("/admin")

  // Determine active item indexing for styles
  let selectedIndex = -1
  if (isAdminMode) {
    if (pathname.startsWith("/admin/overview")) selectedIndex = 10
    else if (pathname.startsWith("/admin/autojoin")) selectedIndex = 14
    else if (pathname.startsWith("/admin/nfs")) selectedIndex = 16
    else if (pathname.startsWith("/admin/jobs")) selectedIndex = 20
    else if (pathname.startsWith("/admin/audit")) selectedIndex = 12
    else if (pathname.startsWith("/admin/ai-usage")) selectedIndex = 18
  } else {
    if (pathname.startsWith("/settings")) selectedIndex = 3
    else if (pathname.startsWith("/iam")) selectedIndex = 4
    else if (pathname.startsWith("/integrations")) selectedIndex = 6
    else if (pathname.endsWith("/workspace")) selectedIndex = 1
    else if (pathname.startsWith("/workflows")) selectedIndex = 5
    else if (pathname.startsWith("/projects")) selectedIndex = 0
  }

  const navigateTo = (path: string) => {
    router.push(path)
  }

  const topMenuItems: MenuItem[] = isAdminMode ? [
    {
      id: 10,
      label: "Overview",
      icon: Info,
      path: "/admin/overview",
      visible: true
    },
    {
      id: 14,
      label: "Autojoin Rules",
      icon: UserPlus,
      path: "/admin/autojoin",
      visible: true
    },
    {
      id: 16,
      label: "NFS Storage",
      icon: HardDrive,
      path: "/admin/nfs",
      visible: true
    },
    {
      id: 20,
      label: "System Jobs",
      icon: Terminal,
      path: "/admin/jobs",
      visible: true
    },
    {
      id: 12,
      label: "Audit Trail",
      icon: Activity,
      path: "/admin/audit",
      visible: true
    },
    {
      id: 18,
      label: "AI Usage & Cost",
      icon: Cpu,
      path: "/admin/ai-usage",
      visible: true
    }
  ] : [
    {
      id: 0,
      label: "Projects",
      icon: FolderKanban,
      path: "/projects",
      visible: hasPermission("project:list")
    },
    {
      id: 5,
      label: "Workflows",
      icon: Workflow,
      path: "/workflows",
      visible: hasPermission("project:list")
    },
    {
      id: 1,
      label: "Workspace",
      icon: Layers,
      path: `/projects/${lastSelectedProjectId}/workspace`,
      visible: hasActiveProject
    },
    {
      id: 6,
      label: "Integrations",
      icon: Cable,
      path: "/integrations",
      visible: hasPermission("org:update")
    },
    {
      id: 4,
      label: "IAM",
      icon: Users,
      path: "/iam",
      visible: hasPermission("user:list")
    }
  ]

  const bottomMenuItems: MenuItem[] = isAdminMode ? [
    {
      id: 15,
      label: "Exit Admin",
      icon: ArrowLeft,
      path: "/projects",
      visible: true
    },
    {
      id: 3,
      label: "Settings",
      icon: Settings,
      path: "/settings",
      visible: true
    }
  ] : [
    {
      id: 2,
      label: "Debug Console",
      icon: Bug,
      action: toggleDebugBar,
      activeState: isDebugBarOpen,
      visible: isWorkspaceScreen,
      isToggle: true
    },
    {
      id: 7,
      label: "Admin Console",
      icon: Shield,
      path: "/admin",
      visible: isSuperAdmin || isAdmin || isOwner
    },
    {
      id: 3,
      label: "Settings",
      icon: Settings,
      path: "/settings",
      visible: true
    }
  ]

  const renderMenuItem = (item: MenuItem) => {
    if (!item.visible) return null

    const isSelected = item.isToggle ? item.activeState : selectedIndex === item.id
    const Icon = item.icon

    return (
      <button
        key={item.id}
        id={`tour-leftbar-${item.label.toLowerCase().replace(/\s+/g, '-')}`}
        onClick={() => item.isToggle ? item.action?.() : navigateTo(item.path!)}
        className={`flex items-center w-full h-12 transition-all duration-200 relative ${
          isExpanded ? "px-5 gap-4 justify-start" : "px-0 justify-center gap-0"
        } ${
          isSelected
            ? "bg-secondary text-secondary-foreground font-semibold"
            : "text-muted-foreground hover:bg-muted hover:text-foreground"
        }`}
      >
        {/* Selected left vertical indicator stripe */}
        {isSelected && (
          <span className="absolute left-0 top-0 bottom-0 w-1 bg-primary rounded-r" />
        )}
        
        <Icon className={`h-5 w-5 shrink-0 ${isSelected ? "text-primary" : ""}`} />
        
        {isExpanded && (
          <span className="text-sm truncate animate-fade-in">
            {item.label}
          </span>
        )}
      </button>
    )
  }

  return (
    <aside
      id="tour-leftbar"
      className={`relative flex flex-col h-screen bg-slate-100 dark:bg-slate-900 border-r border-border transition-all duration-300 ${
        isExpanded ? "w-64" : "w-18"
      }`}
    >
      {/* Branding / Logo Area */}
      <div className={`flex h-16 items-center border-b border-border overflow-hidden transition-all duration-300 ${
        isExpanded ? "px-5 gap-3" : "justify-center px-0 gap-0"
      }`}>
        <div className="flex h-8 w-8 shrink-0 items-center justify-center">
          <Image
            src="/images/agentic-foundry-logo.png"
            alt="Agentic Foundry Logo"
            width={32}
            height={32}
            className="object-contain"
            priority
          />
        </div>
        {isExpanded && (
          <div className="flex flex-col justify-center min-w-0 animate-fade-in pt-1">
            <span className="font-bold text-foreground text-sm tracking-wide truncate leading-tight">
              Product Foundry
            </span>
            {appVersion && (
              <span className="text-[10px] font-medium text-muted-foreground tracking-normal truncate leading-none mt-0.5">
                {appVersion}
              </span>
            )}
          </div>
        )}
      </div>

      {/* Navigation List Menu */}
      <nav className="flex-1 flex flex-col justify-between overflow-hidden">
        {/* Top Scrollable Items */}
        <div className="flex-1 py-4 space-y-1 overflow-y-auto">
          {topMenuItems.map(renderMenuItem)}
        </div>

        {/* Bottom Docked Items */}
        {bottomMenuItems.some(item => item.visible) && (
          <div className="py-2 space-y-1 border-t border-border/50">
            {bottomMenuItems.map(renderMenuItem)}
          </div>
        )}
      </nav>

      {/* Footer Collapse Sidebar Action button */}
      <div className={`p-3 border-t border-border flex transition-all duration-300 ${
        isExpanded ? "justify-end" : "justify-center"
      }`}>
        <button
          onClick={() => setIsExpanded(!isExpanded)}
          className="flex h-8 w-8 items-center justify-center rounded-lg text-muted-foreground hover:bg-muted hover:text-foreground"
        >
          {isExpanded ? (
            <ChevronLeft className="h-4 w-4" />
          ) : (
            <ChevronRight className="h-4 w-4" />
          )}
        </button>
      </div>
    </aside>
  )
}
export default LeftBar
