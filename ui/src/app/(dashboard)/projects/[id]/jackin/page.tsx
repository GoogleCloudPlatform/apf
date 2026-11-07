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

import { useEffect, useState, use } from "react"
import { useRouter } from "next/navigation"
import { useQuery } from "@tanstack/react-query"
import { toast } from "sonner"
import { 
  ArrowLeft, 
  Plus, 
  Mic, 
  Paperclip, 
  Image as ImageIcon,
  FolderOpen, 
  Search, 
  CornerDownLeft,
  X,
  Sparkles,
  Loader2
} from "lucide-react"
import { useAppStore } from "@/stores/app"
import { useAuthStore } from "@/stores/auth"
import { api } from "@/lib/api"
import { Project } from "@/lib/types"

interface PageParams {
  id: string
}

// Static Headjack Sugestions Data matching gemini_chat_data.dart
const geminiChatPlaybook = {
  default_chips: [
    {
      text: "Analyze",
      id: "analyze",
      icon: "💡",
      suggestions: [
        "Brainstorm",
        "Technical Research",
        "Domain Research",
        "Market Research",
        "Product Brief",
        "Product FAQ"
      ],
    },
    {
      text: "Plan",
      id: "plan",
      icon: "📐",
      suggestions: ["Create PRD", "Create UX Design"],
    },
    {
      text: "Solution",
      id: "solution",
      icon: "📚",
      suggestions: [
        "Create Architecture",
        "Create Epics and Stories",
        "Check Implementation Readiness"
      ],
    },
    {
      text: "Implement",
      id: "implement",
      icon: "✏️",
      suggestions: [
        "Sprint Planning",
        "Create Story",
        "Dev Story",
        "Code Review",
        "Correct Course",
        "Retrospective"
      ],
    },
    {
      text: "Launch",
      id: "launch",
      icon: "🚀",
      suggestions: [],
    }
  ],
  menu_items: [
    { text: "Upload files", icon: Paperclip },
    { text: "Add from Drive", icon: FolderOpen },
    { text: "Upload Image", icon: ImageIcon }
  ]
}

export default function HeadjackPage({ params }: { params: Promise<PageParams> }) {
  const { id: projectId } = use(params)
  const router = useRouter()
  const { setPageTitle } = useAppStore()
  const { displayName } = useAuthStore()

  const [promptText, setPromptText] = useState("")
  const [selectedChipId, setSelectedChipId] = useState<string | null>(null)
  const [isMenuOpen, setIsMenuOpen] = useState(false)

  // Sync dyn header title
  useEffect(() => {
    setPageTitle("The Matrix")
  }, [setPageTitle])

  // Query: Get current project info
  const {
    data: project,
    isLoading: isLoadingProject,
    error: projectError,
  } = useQuery<Project>({
    queryKey: ["project", projectId],
    queryFn: async () => {
      const res = await api.get(`/api/v1/projects/${projectId}`)
      if (!res.ok) throw new Error("Failed to load project details")
      return res.json()
    },
  })

  const handleCancel = () => {
    router.push("/projects")
  }

  const handleSend = () => {
    if (!promptText.trim()) return
    // Mock submit action matching current Flutter code
    toast.info(`Prompt submitted: "${promptText.trim()}"`)
    setPromptText("")
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault()
      handleSend()
    }
  }

  const activeChip = geminiChatPlaybook.default_chips.find((c) => c.id === selectedChipId)
  const shouldMoveUp = selectedChipId !== null || promptText.length > 0

  if (isLoadingProject) {
    return (
      <div className="flex h-[350px] w-full flex-col items-center justify-center gap-3 text-muted-foreground select-none">
        <Loader2 className="h-8 w-8 animate-spin text-primary" />
        <span className="text-sm font-medium">Entering The Matrix...</span>
      </div>
    )
  }

  return (
    <div className="flex flex-col h-full bg-slate-50 dark:bg-slate-950">
      
      {/* Upper sub-app bar header row */}
      <div className="h-16 px-6 border-b border-border bg-card flex items-center justify-between select-none">
        <div className="flex items-center gap-3">
          <button
            onClick={handleCancel}
            className="flex h-8 w-8 items-center justify-center rounded-lg text-muted-foreground hover:bg-muted border border-input bg-background"
          >
            <ArrowLeft className="h-4 w-4" />
          </button>
          
          <div className="flex items-center gap-2.5">
            <span className="text-xl">{project?.icon || "🚀"}</span>
            <div>
              <h2 className="text-sm font-bold text-foreground leading-tight">
                {project?.name}
              </h2>
              <span className="font-mono text-[9px] text-muted-foreground">
                ID: {projectId}
              </span>
            </div>
          </div>
        </div>

        <span className="text-[10px] font-bold text-primary uppercase tracking-wider border border-primary/20 bg-primary/5 px-2.5 py-1 rounded-full flex items-center gap-1">
          <Sparkles className="h-3 w-3" />
          Gemini Playground
        </span>
      </div>

      {/* Primary chat workspace container view */}
      <div className="flex-1 flex flex-col justify-center items-center px-6 relative overflow-y-auto">
        
        {/* Translates the Transform.translate offset loop offset-y -160 */}
        <div className={`w-full max-w-3xl flex flex-col items-center gap-8 transition-transform duration-500 transform ${
          shouldMoveUp ? "-translate-y-12" : "-translate-y-20"
        }`}>
          
          {/* Main prompt headings greeting */}
          <div className={`text-center space-y-1.5 select-none transition-opacity duration-300 ${
            shouldMoveUp ? "opacity-30 scale-95" : "opacity-100"
          }`}>
            <h3 className="text-sm text-muted-foreground font-medium">
              Hi {displayName || "Foundry User"}
            </h3>
            <h1 className="text-3xl font-bold tracking-tight text-foreground">
              Where should we start?
            </h1>
          </div>

          {/* Central actions prompt input card block */}
          <div className="w-full rounded-2xl border border-border bg-card shadow-lg p-4 flex flex-col gap-3.5 hover:border-muted-foreground/20 transition-all duration-300">
            
            {/* Top input text fields (when floating up) */}
            {shouldMoveUp && (
              <div className="flex items-center gap-3 px-1 border-b border-border/60 pb-3 animate-slide-down">
                {activeChip && (
                  <span className="inline-flex items-center gap-1.5 text-xs font-semibold bg-secondary text-secondary-foreground px-3 py-1 rounded-full border border-border">
                    <span>{activeChip.icon}</span>
                    <span>{activeChip.text}</span>
                    <button
                      onClick={() => setSelectedChipId(null)}
                      className="ml-1 flex h-4.5 w-4.5 items-center justify-center rounded-full hover:bg-muted-foreground/20 text-muted-foreground"
                    >
                      <X className="h-3 w-3" />
                    </button>
                  </span>
                )}
                
                <input
                  type="text"
                  placeholder={activeChip ? "What do you want to learn?" : "Enter a prompt for Foundry"}
                  value={promptText}
                  onChange={(e) => setPromptText(e.target.value)}
                  onKeyDown={handleKeyDown}
                  className="flex-1 bg-transparent text-sm focus:outline-none placeholder:text-muted-foreground"
                  autoFocus
                />
              </div>
            )}

            {/* Actions list row bar */}
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2 relative">
                {/* Popover/Dropdown Plus uploads selector button */}
                <button
                  type="button"
                  onClick={() => setIsMenuOpen(!isMenuOpen)}
                  className={`flex h-8 w-8 items-center justify-center rounded-full hover:bg-muted text-muted-foreground border border-input bg-background transition-transform duration-200 ${
                    isMenuOpen ? "rotate-45" : ""
                  }`}
                >
                  <Plus className="h-4.5 w-4.5" />
                </button>

                {isMenuOpen && (
                  <div className="absolute bottom-10 left-0 w-48 rounded-lg border border-border bg-card p-1 shadow-xl text-card-foreground flex flex-col z-50 animate-in zoom-in-95 duration-150">
                    {geminiChatPlaybook.menu_items.map((item, idx) => {
                      const Icon = item.icon
                      return (
                        <button
                          key={idx}
                          type="button"
                          onClick={() => {
                            setIsMenuOpen(false)
                            toast.info(`Action triggered: ${item.text}`)
                          }}
                          className="flex items-center gap-2 px-3 py-2 text-xs rounded hover:bg-muted text-left transition-colors text-foreground"
                        >
                          <Icon className="h-3.5 w-3.5 text-muted-foreground" />
                          <span>{item.text}</span>
                        </button>
                      )
                    })}
                  </div>
                )}

                {/* Sub-prompt main text field input (when collapsed/default landing mode) */}
                {!shouldMoveUp && (
                  <div className="flex-1 flex items-center min-w-[280px] sm:min-w-[400px]">
                    <input
                      type="text"
                      placeholder="Enter a prompt for Foundry"
                      value={promptText}
                      onChange={(e) => setPromptText(e.target.value)}
                      onKeyDown={handleKeyDown}
                      className="w-full h-8 px-2 bg-transparent text-sm focus:outline-none placeholder:text-muted-foreground"
                    />
                  </div>
                )}
              </div>

              <div className="flex items-center gap-1">
                {/* Mic input action placeholder */}
                <button
                  type="button"
                  className="flex h-8 w-8 items-center justify-center rounded-full hover:bg-muted text-muted-foreground"
                  title="Voice Input (Coming soon)"
                >
                  <Mic className="h-4 w-4" />
                </button>

                {/* Submit action button */}
                {shouldMoveUp && (
                  <button
                    onClick={handleSend}
                    className="flex h-8 w-8 items-center justify-center rounded-full bg-primary text-primary-foreground shadow-md hover:opacity-90"
                    title="Submit Prompt"
                  >
                    <CornerDownLeft className="h-4 w-4" />
                  </button>
                )}
              </div>
            </div>
          </div>

          {/* Bottom presets grid OR suggestions rows checklist lists */}
          {!selectedChipId ? (
            /* Emojis Default playbooks chips grid row */
            <div className="flex flex-wrap justify-center gap-2.5 w-full select-none animate-slide-up">
              {geminiChatPlaybook.default_chips.map((chip) => (
                <button
                  key={chip.id}
                  onClick={() => chip.suggestions.length > 0 ? setSelectedChipId(chip.id) : toast.info(`Action triggered: ${chip.text}`)}
                  className="flex items-center gap-2 bg-card hover:bg-muted border border-border px-5 py-3 rounded-full text-sm font-semibold text-foreground transition-all shadow-sm"
                >
                  <span>{chip.icon}</span>
                  <span>{chip.text}</span>
                </button>
              ))}
            </div>
          ) : (
            /* Active suggestions checklist list rows */
            <div className="w-full space-y-2 animate-slide-up">
              <div className="flex justify-between items-center px-1 mb-2">
                <span className="text-xs font-bold text-muted-foreground uppercase tracking-wider">
                  {activeChip?.text} Sugestions Playbook
                </span>
                <button
                  onClick={() => setSelectedChipId(null)}
                  className="text-[10px] font-semibold text-primary hover:underline"
                >
                  Back to Milestones
                </button>
              </div>
              
              <div className="space-y-2 select-none">
                {activeChip?.suggestions.map((suggestion, idx) => (
                  <button
                    key={idx}
                    onClick={() => {
                      setPromptText(`Help me run ${suggestion.toLowerCase()}`)
                      toast.info(`Selected task: ${suggestion}`)
                    }}
                    className="flex items-center w-full h-12 px-6 rounded-xl border border-border bg-card hover:bg-muted text-foreground hover:border-muted-foreground/20 text-sm font-medium transition-all shadow-sm text-left gap-3"
                  >
                    <div className="h-2 w-2 rounded-full bg-primary" />
                    <span>{suggestion}</span>
                  </button>
                ))}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
