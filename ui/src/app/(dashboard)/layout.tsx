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

import LeftBar from "@/components/shared/LeftBar"
import TopBar from "@/components/shared/TopBar"
import { useActivityStore } from "@/stores/activity"
import { ActivityFeedPanel } from "@/components/workspace/ActivityFeedPanel"
import OnboardingTour from "@/components/shared/OnboardingTour"

interface DashboardLayoutProps {
  children: React.ReactNode
}

/**
 * Dashboard Shell layout container.
 * Groups pages requiring LeftBar and TopBar components.
 */
export default function DashboardLayout({ children }: DashboardLayoutProps) {
  const { isPanelOpen } = useActivityStore()

  return (
    <div className="flex h-screen w-screen overflow-hidden bg-background text-foreground">
      {/* 1. Left Sidebar Navigation Column */}
      <LeftBar />

      {/* 2. Main content viewport section */}
      <div className="flex-1 flex flex-col h-full overflow-hidden">
        {/* Top Header Panel */}
        <TopBar />

        {/* Outer content container supporting right logs dock panel */}
        <div className="flex-1 flex w-full h-full overflow-hidden relative">
          <main className="flex-1 h-full overflow-y-auto min-w-0">
            {children}
          </main>

          {/* Collapsible right-docked side panel activities stream list */}
          {isPanelOpen && <ActivityFeedPanel />}
        </div>
      </div>

      {/* Onboarding Walkthrough */}
      <OnboardingTour />
    </div>
  )
}
