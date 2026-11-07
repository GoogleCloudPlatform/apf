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

// @vitest-environment jsdom
import React from "react"
import { render, screen, fireEvent, cleanup, waitFor, act } from "@testing-library/react"
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import { Message } from "@/lib/types"
import ChatPane from "./ChatPane"

const mockSendMessage = vi.fn().mockResolvedValue(true)
const mockCancelMessage = vi.fn()
const mockRetryConnection = vi.fn()
const mockTriggerError = vi.fn()

let mockUseChatStreamState = {
  messages: [
    { id: "1", sender: "Agent", text: "Hello! How can I help you?", isUser: false, timestamp: new Date().toISOString() }
  ] as Message[],
  isLoading: false,
  isThinking: false,
  latestThought: null as any,
  thinkingSteps: [] as any[],
  error: null,
  sendMessage: mockSendMessage,
  cancelMessage: mockCancelMessage,
  retryConnection: mockRetryConnection,
  triggerError: mockTriggerError,
}

vi.mock("@/hooks/useChatStream", () => ({
  useChatStream: () => mockUseChatStreamState
}))

vi.mock("@/stores/auth", () => ({
  useAuthStore: () => ({
    user: { name: "Test User" },
  })
}))

vi.mock("@/stores/activity", () => ({
  useActivityStore: () => ({
    activities: [],
    isPanelOpen: false,
  }),
  getPersonaColorClass: () => "bg-blue-500",
}))

const mockStartListening = vi.fn()
const mockStopListening = vi.fn()
let mockSpeechState = {
  isListening: false,
  isSupported: true,
  transcript: "",
  interimTranscript: "",
  error: null as string | null,
  startListening: mockStartListening,
  stopListening: mockStopListening,
  resetTranscript: vi.fn(),
}

vi.mock("@/hooks/useSpeechRecognition", () => ({
  useSpeechRecognition: () => mockSpeechState
}))

const mockToastError = vi.fn()
const mockToastSuccess = vi.fn()
vi.mock("sonner", () => ({
  toast: {
    error: (...args: any[]) => mockToastError(...args),
    success: (...args: any[]) => mockToastSuccess(...args),
  },
}))

const mockUploadSessionArtifact = vi.fn().mockImplementation((projectId, sessionId, file) =>
  Promise.resolve({
    id: "art-123",
    projectId,
    sessionId,
    name: file.name,
    mimeType: file.type || "application/octet-stream",
    size: file.size,
    storageType: "local_file",
    storagePath: `/artifacts/${file.name}`,
    createdAt: new Date().toISOString(),
  })
)

vi.mock("@/lib/api", () => ({
  api: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  },
  uploadSessionArtifact: (...args: any[]) => mockUploadSessionArtifact(...args),
}))

// Mock scroll and URL functions for jsdom
window.HTMLElement.prototype.scrollIntoView = vi.fn()
window.HTMLElement.prototype.scrollTo = vi.fn()
if (typeof window.URL.createObjectURL === "undefined") {
  window.URL.createObjectURL = vi.fn((file: any) => `blob:http://localhost/${file.name || "test"}`)
} else {
  vi.spyOn(window.URL, "createObjectURL").mockImplementation((file: any) => `blob:http://localhost/${file.name || "test"}`)
}
if (typeof window.URL.revokeObjectURL === "undefined") {
  window.URL.revokeObjectURL = vi.fn()
} else {
  vi.spyOn(window.URL, "revokeObjectURL").mockImplementation(vi.fn())
}

describe("ChatPane Component", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockUploadSessionArtifact.mockImplementation((projectId, sessionId, file) =>
      Promise.resolve({
        id: "art-123",
        projectId,
        sessionId,
        name: file.name,
        mimeType: file.type || "application/octet-stream",
        size: file.size,
        storageType: "local_file",
        storagePath: `/artifacts/${file.name}`,
        createdAt: new Date().toISOString(),
      })
    )
    mockUseChatStreamState.isThinking = false
    mockUseChatStreamState.latestThought = null
    mockUseChatStreamState.thinkingSteps = []
    mockUseChatStreamState.error = null
    mockUseChatStreamState.messages = [
      { id: "1", sender: "Agent", text: "Hello! How can I help you?", isUser: false, timestamp: new Date().toISOString() }
    ]
    mockSpeechState.isListening = false
    mockSpeechState.isSupported = true
  })

  afterEach(() => {
    cleanup()
  })

  it("renders textarea in chat composer", () => {
    render(
      <ChatPane
        projectId="proj-1"
        activePhase="phase-1"
        sessionId="session-1"
        isConnecting={false}
        connectionError={null}
        onRetry={vi.fn()}
      />
    )

    const textarea = screen.getByPlaceholderText(/Ask a question, enter a command/i)
    expect(textarea).not.toBeNull()
    expect(textarea.tagName.toLowerCase()).toBe("textarea")
    expect(textarea.parentElement?.className).not.toContain("animate-neon-glow-once")
  })

  it("submits message on Enter without Shift", async () => {
    render(
      <ChatPane
        projectId="proj-1"
        activePhase="phase-1"
        sessionId="session-1"
        isConnecting={false}
        connectionError={null}
        onRetry={vi.fn()}
      />
    )

    const textarea = screen.getByPlaceholderText(/Ask a question, enter a command/i) as HTMLTextAreaElement
    fireEvent.change(textarea, { target: { value: "Hello world" } })

    fireEvent.keyDown(textarea, { key: "Enter", shiftKey: false })

    expect(mockSendMessage).toHaveBeenCalledWith("Hello world")
  })

  it("allows new line on Shift+Enter without submitting message", () => {
    render(
      <ChatPane
        projectId="proj-1"
        activePhase="phase-1"
        sessionId="session-1"
        isConnecting={false}
        connectionError={null}
        onRetry={vi.fn()}
      />
    )

    const textarea = screen.getByPlaceholderText(/Ask a question, enter a command/i) as HTMLTextAreaElement
    fireEvent.change(textarea, { target: { value: "Line 1" } })

    fireEvent.keyDown(textarea, { key: "Enter", shiftKey: true })

    expect(mockSendMessage).not.toHaveBeenCalled()
  })

  it("resizes height up to maximum of 6 lines height", async () => {
    render(
      <ChatPane
        projectId="proj-1"
        activePhase="phase-1"
        sessionId="session-1"
        isConnecting={false}
        connectionError={null}
        onRetry={vi.fn()}
      />
    )

    const textarea = screen.getByPlaceholderText(/Ask a question, enter a command/i) as HTMLTextAreaElement

    // Mock scrollHeight and computedStyle for jsdom
    Object.defineProperty(textarea, "scrollHeight", {
      configurable: true,
      get() {
        // Return simulated scrollHeight based on number of lines in value
        const lines = (textarea.value.match(/\n/g) || []).length + 1
        return lines * 20 + 16 + 2 // 20px per line + 16px padding + 2px border
      }
    })

    // Single line
    fireEvent.change(textarea, { target: { value: "Line 1" } })
    expect(textarea.style.height).toBe("38px")

    // 3 lines
    fireEvent.change(textarea, { target: { value: "Line 1\nLine 2\nLine 3" } })
    expect(textarea.style.height).toBe("78px")

    // 8 lines (should cap at 6 lines max height, which is less than total 8-line scrollHeight)
    fireEvent.change(textarea, { target: { value: "1\n2\n3\n4\n5\n6\n7\n8" } })
    const h8 = parseFloat(textarea.style.height)
    expect(h8).toBeLessThan(textarea.scrollHeight)
    expect(h8).toBeGreaterThan(130)
    expect(h8).toBeLessThan(140)
  })

  it("renders notification bubble (latestThought) even when isThinking is false", () => {
    mockUseChatStreamState.isThinking = false
    mockUseChatStreamState.latestThought = {
      id: "notif-1",
      sender: "System",
      text: "System check completed successfully",
      isUser: false,
      isNotification: true,
      isTransient: true,
      timestamp: new Date().toISOString()
    }

    render(
      <ChatPane
        projectId="proj-1"
        activePhase="phase-1"
        sessionId="session-1"
        isConnecting={false}
        connectionError={null}
        onRetry={vi.fn()}
      />
    )

    expect(screen.getByText("System check completed successfully")).not.toBeNull()
  })

  it("auto-scrolls to bottom when thinkingSteps updates during streaming", () => {
    mockUseChatStreamState.isThinking = true
    mockUseChatStreamState.thinkingSteps = [
      {
        id: "step-1",
        sender: "Agent",
        text: "Read file: `config.json`",
        isUser: false,
        timestamp: new Date().toISOString()
      }
    ]

    render(
      <ChatPane
        projectId="proj-1"
        activePhase="phase-1"
        sessionId="session-1"
        isConnecting={false}
        connectionError={null}
        onRetry={vi.fn()}
      />
    )

    expect(window.HTMLElement.prototype.scrollIntoView).toHaveBeenCalled()
  })

  it("renders voice input microphone button when speech recognition is supported", () => {
    render(
      <ChatPane
        projectId="proj-1"
        activePhase="phase-1"
        sessionId="session-1"
        isConnecting={false}
        connectionError={null}
        onRetry={vi.fn()}
      />
    )

    const micButton = screen.getByTitle("Start voice input (Alt+V or Ctrl+Shift+M)")
    expect(micButton).not.toBeNull()
  })

  it("calls startListening when clicking microphone button", () => {
    render(
      <ChatPane
        projectId="proj-1"
        activePhase="phase-1"
        sessionId="session-1"
        isConnecting={false}
        connectionError={null}
        onRetry={vi.fn()}
      />
    )

    const micButton = screen.getByTitle("Start voice input (Alt+V or Ctrl+Shift+M)")
    fireEvent.click(micButton)
    expect(mockStartListening).toHaveBeenCalledTimes(1)
  })

  it("renders live listening indicator banner and allows stopping when isListening is true", () => {
    mockSpeechState.isListening = true

    render(
      <ChatPane
        projectId="proj-1"
        activePhase="phase-1"
        sessionId="session-1"
        isConnecting={false}
        connectionError={null}
        onRetry={vi.fn()}
      />
    )

    expect(screen.getByText(/Listening\.\.\. Speak clearly into your microphone/i)).not.toBeNull()

    const stopBtn = screen.getByText("Stop (ESC / Alt+V)")
    fireEvent.click(stopBtn)
    expect(mockStopListening).toHaveBeenCalledTimes(1)
  })

  it("toggles voice capture via Alt+V or Ctrl+Shift+M keyboard shortcuts", () => {
    render(
      <ChatPane
        projectId="proj-1"
        activePhase="phase-1"
        sessionId="session-1"
        isConnecting={false}
        connectionError={null}
        onRetry={vi.fn()}
      />
    )

    fireEvent.keyDown(window, { key: "v", altKey: true })
    expect(mockStartListening).toHaveBeenCalledTimes(1)
  })

  it("renders Recommended Phase Switch Card exactly once when suggestion is attached", () => {
    mockUseChatStreamState.messages = [
      {
        id: "msg-1",
        sender: "User",
        text: "I want to create a PRD",
        isUser: true,
        timestamp: new Date().toISOString(),
      },
      {
        id: "msg-2",
        sender: "Agent",
        text: "I recommend switching to the Planning phase to create a PRD.",
        isUser: false,
        phaseSwitchSuggestion: {
          targetPhase: "PLANNING",
          skillId: "bmad-prd",
          reason: "PRD creation is handled in Planning phase by John PM.",
        },
        timestamp: new Date().toISOString(),
      },
    ]

    const handleCrossPhaseSwitch = vi.fn()

    render(
      <ChatPane
        projectId="proj-1"
        activePhase="ONBOARDING"
        sessionId="session-1"
        isConnecting={false}
        connectionError={null}
        onRetry={vi.fn()}
        onCrossPhaseSwitch={handleCrossPhaseSwitch}
      />
    )

    const cards = screen.getAllByText(/Recommended Phase Switch: Plan/i)
    expect(cards).toHaveLength(1)

    const switchBtn = screen.getByRole("button", { name: /Switch to Plan Tab & Continue/i })
    expect(switchBtn).toBeDefined()
    fireEvent.click(switchBtn)
    expect(handleCrossPhaseSwitch).toHaveBeenCalledWith("PLANNING", "Run bmad-prd")
  })

  it("renders 'Go to Settings' action button in AlertModal when Gemini API key error occurs", () => {
    mockUseChatStreamState.error = {
      title: "Agent Initialization Failed",
      description: "Gemini API key is required. Please configure your key in Project Settings or User Settings.",
    } as any

    render(
      <ChatPane
        projectId="proj-1"
        activePhase="ONBOARDING"
        sessionId="session-1"
        isConnecting={false}
        connectionError="Your Gemini API key is missing or invalid. Please configure a valid key in Project Settings or User Settings."
        onRetry={vi.fn()}
      />
    )

    const settingsBtn = screen.getByRole("button", { name: /Go to Settings/i })
    expect(settingsBtn).toBeDefined()
  })

  it("adds quick action pill to chat composer when clicking an action chip in message", () => {
    mockUseChatStreamState.messages = [
      {
        id: "msg-1",
        sender: "Agent",
        text: "Here are some recommendations:",
        isUser: false,
        actions: ["bmad-help", "bmad-create-prd"],
        timestamp: new Date().toISOString(),
      },
    ]

    render(
      <ChatPane
        projectId="proj-1"
        activePhase="PLANNING"
        sessionId="session-1"
        isConnecting={false}
        connectionError={null}
        onRetry={vi.fn()}
      />
    )

    // Click the Help action chip
    const chipButton = screen.getByRole("button", { name: /help/i })
    fireEvent.click(chipButton)

    // Verify pill is shown in composer with label "Help"
    expect(screen.getByText("Help")).toBeDefined()
    // Verify remove button is present
    const removeBtn = screen.getByRole("button", { name: /remove quick action/i })
    expect(removeBtn).toBeDefined()
  })

  it("removes quick action pill when clicking the 'x' button", () => {
    mockUseChatStreamState.messages = [
      {
        id: "msg-1",
        sender: "Agent",
        text: "Suggested next action:",
        isUser: false,
        actions: ["bmad-create-prd"],
        timestamp: new Date().toISOString(),
      },
    ]

    render(
      <ChatPane
        projectId="proj-1"
        activePhase="PLANNING"
        sessionId="session-1"
        isConnecting={false}
        connectionError={null}
        onRetry={vi.fn()}
      />
    )

    const chipButton = screen.getByRole("button", { name: /create-prd/i })
    fireEvent.click(chipButton)

    // Verify pill exists
    expect(screen.getByText("Create Prd")).toBeDefined()
    const removeBtn = screen.getByRole("button", { name: /remove quick action/i })

    // Click remove button
    fireEvent.click(removeBtn)

    // Pill should no longer be in the document
    expect(screen.queryByRole("button", { name: /remove quick action/i })).toBeNull()
  })

  it("sends associated quick action prompt to backend when clicking send with empty input", async () => {
    mockUseChatStreamState.messages = [
      {
        id: "msg-1",
        sender: "Agent",
        text: "Options:",
        isUser: false,
        actions: ["bmad-help"],
        timestamp: new Date().toISOString(),
      },
    ]

    render(
      <ChatPane
        projectId="proj-1"
        activePhase="PLANNING"
        sessionId="session-1"
        isConnecting={false}
        connectionError={null}
        onRetry={vi.fn()}
      />
    )

    // Click chip
    const chipButton = screen.getByRole("button", { name: /help/i })
    fireEvent.click(chipButton)

    // Submit form (send button)
    const submitBtn = screen.getAllByRole("button").find(btn => btn.getAttribute("type") === "submit")!
    expect(submitBtn).toBeDefined()
    expect((submitBtn as HTMLButtonElement).disabled).toBe(false)

    fireEvent.click(submitBtn)

    await waitFor(() => {
      expect(mockSendMessage).toHaveBeenCalledWith("bmad-help")
    })

    // Pill should be cleared after send
    expect(screen.queryByRole("button", { name: /remove quick action/i })).toBeNull()
  })

  it("combines quick action prompt and typed text on submission", async () => {
    mockUseChatStreamState.messages = [
      {
        id: "msg-1",
        sender: "Agent",
        text: "Options:",
        isUser: false,
        actions: ["bmad-create-prd"],
        timestamp: new Date().toISOString(),
      },
    ]

    render(
      <ChatPane
        projectId="proj-1"
        activePhase="PLANNING"
        sessionId="session-1"
        isConnecting={false}
        connectionError={null}
        onRetry={vi.fn()}
      />
    )

    // Click chip
    const chipButton = screen.getByRole("button", { name: /create-prd/i })
    fireEvent.click(chipButton)

    // Type additional text in textarea
    const textarea = screen.getByPlaceholderText(/Add extra details/i)
    fireEvent.change(textarea, { target: { value: "focus on user authentication" } })

    // Submit
    const submitBtn = screen.getAllByRole("button").find(btn => btn.getAttribute("type") === "submit")!
    fireEvent.click(submitBtn)

    await waitFor(() => {
      expect(mockSendMessage).toHaveBeenCalledWith("bmad-create-prd focus on user authentication")
    })
  })

  it("sends prompt to API but passes displayPrompt for chat bubble display when submitting quick action with displayPrompt", async () => {
    const quickActions = [
      {
        id: "bmad-project-context",
        name: "Project Context",
        prompt: "/bmad-project-context",
        displayPrompt: "Generate project context by setting up repository instructions.",
      },
    ]

    render(
      <ChatPane
        projectId="proj-1"
        activePhase="ONBOARDING"
        sessionId="session-1"
        isConnecting={false}
        connectionError={null}
        quickActions={quickActions}
        onRetry={vi.fn()}
      />
    )

    // Click quick action button
    const actionBtn = screen.getByRole("button", { name: "Project Context" })
    fireEvent.click(actionBtn)

    // Submit form (send button)
    const submitBtn = screen.getAllByRole("button").find(btn => btn.getAttribute("type") === "submit")!
    fireEvent.click(submitBtn)

    await waitFor(() => {
      expect(mockSendMessage).toHaveBeenCalledWith(
        "/bmad-project-context",
        "Generate project context by setting up repository instructions."
      )
    })
  })

  it("combines display prompt with user text when submitting quick action with displayPrompt and typed text", async () => {
    const quickActions = [
      {
        id: "bmad-project-context",
        name: "Project Context",
        prompt: "/bmad-project-context",
        displayPrompt: "Generate project context by setting up repository instructions.",
      },
    ]

    render(
      <ChatPane
        projectId="proj-1"
        activePhase="ONBOARDING"
        sessionId="session-1"
        isConnecting={false}
        connectionError={null}
        quickActions={quickActions}
        onRetry={vi.fn()}
      />
    )

    // Click quick action button
    const actionBtn = screen.getByRole("button", { name: "Project Context" })
    fireEvent.click(actionBtn)

    // Type additional text in textarea
    const textarea = screen.getByPlaceholderText(/Add extra details/i)
    fireEvent.change(textarea, { target: { value: "focus on backend APIs" } })

    // Submit
    const submitBtn = screen.getAllByRole("button").find(btn => btn.getAttribute("type") === "submit")!
    fireEvent.click(submitBtn)

    await waitFor(() => {
      expect(mockSendMessage).toHaveBeenCalledWith(
        "/bmad-project-context focus on backend APIs",
        "Generate project context by setting up repository instructions. focus on backend APIs"
      )
    })
  })

  it("renders displayPrompt or displayText in the chat bubble instead of prompt", () => {
    mockUseChatStreamState.messages = [
      {
        id: "msg-user-1",
        sender: "User",
        text: "Generate project context by setting up repository instructions.",
        prompt: "/bmad-project-context",
        displayPrompt: "Generate project context by setting up repository instructions.",
        isUser: true,
        timestamp: new Date().toISOString(),
      },
    ]

    render(
      <ChatPane
        projectId="proj-1"
        activePhase="ONBOARDING"
        sessionId="session-1"
        isConnecting={false}
        connectionError={null}
        onRetry={vi.fn()}
      />
    )

    expect(screen.getByText("Generate project context by setting up repository instructions.")).toBeDefined()
    expect(screen.queryByText("/bmad-project-context")).toBeNull()
  })

  it("sets quick action pill via ref.current.injectQuickAction", () => {
    const ref = React.createRef<any>()

    render(
      <ChatPane
        ref={ref}
        projectId="proj-1"
        activePhase="PLANNING"
        sessionId="session-1"
        isConnecting={false}
        connectionError={null}
        onRetry={vi.fn()}
      />
    )

    act(() => {
      ref.current?.injectQuickAction({ label: "Custom Action", prompt: "custom-prompt-123" })
    })

    expect(screen.getByText("Custom Action")).toBeDefined()
    expect(screen.getByRole("button", { name: /remove quick action/i })).toBeDefined()
  })

  it("renders quick actions under the chat prompt text box with flex-wrap layout", () => {
    const quickActions = [
      { id: "bmad-help", name: "Help", prompt: "Analyze current state and recommend next steps." },
      { id: "sprint-planning", name: "Sprint Planning", prompt: "Generate sprint plan." },
      { id: "create-story", name: "Create Story", prompt: "Create story." },
    ]

    render(
      <ChatPane
        projectId="proj-1"
        activePhase="PLANNING"
        sessionId="session-1"
        isConnecting={false}
        connectionError={null}
        quickActions={quickActions}
        onRetry={vi.fn()}
      />
    )

    const container = document.getElementById("tour-workspace-quick-actions")
    expect(container).not.toBeNull()
    expect(container?.className).toContain("flex")
    expect(container?.className).toContain("flex-wrap")
    expect(container?.className).toContain("pl-[52px]")

    const helpBtn = screen.getByRole("button", { name: "Help" })
    expect(helpBtn).toBeDefined()
    expect(helpBtn.className).toContain("animate-neon-glow-once-4s")
    expect(helpBtn.querySelector(".animate-border-snake-4s")).not.toBeNull()
    expect(screen.getByRole("button", { name: "Sprint Planning" })).toBeDefined()
    expect(screen.getByRole("button", { name: "Create Story" })).toBeDefined()
  })

  it("selects and deselects quick action when clicking quick action buttons under prompt text box", () => {
    const quickActions = [
      { id: "bmad-help", name: "Help", prompt: "Analyze current state." },
      { id: "sprint-planning", name: "Sprint Planning", prompt: "Generate sprint plan." },
    ]

    render(
      <ChatPane
        projectId="proj-1"
        activePhase="PLANNING"
        sessionId="session-1"
        isConnecting={false}
        connectionError={null}
        quickActions={quickActions}
        onRetry={vi.fn()}
      />
    )

    const helpBtn = screen.getByRole("button", { name: "Help" })
    fireEvent.click(helpBtn)

    // Pill is injected
    expect(screen.getByRole("button", { name: /remove quick action/i })).toBeDefined()

    // Clicking the same button toggles / deselects it
    fireEvent.click(helpBtn)
    expect(screen.queryByRole("button", { name: /remove quick action/i })).toBeNull()
  })

  it("cancels button animation when user sets focus on the chat text box", () => {
    const quickActions = [
      { id: "bmad-help", name: "Help", prompt: "Analyze current state." },
    ]

    render(
      <ChatPane
        projectId="proj-1"
        activePhase="PLANNING"
        sessionId="session-1"
        isConnecting={false}
        connectionError={null}
        quickActions={quickActions}
        onRetry={vi.fn()}
      />
    )

    const helpBtn = screen.getByRole("button", { name: "Help" })
    expect(helpBtn.className).toContain("animate-neon-glow-once-4s")
    expect(helpBtn.querySelector(".animate-border-snake-4s")).not.toBeNull()

    // Focus the textarea
    const textarea = screen.getByPlaceholderText(/Ask a question, enter a command/i)
    fireEvent.focus(textarea)

    // Animation should now be cancelled
    expect(helpBtn.className).not.toContain("animate-neon-glow-once-4s")
    expect(helpBtn.querySelector(".animate-border-snake-4s")).toBeNull()
  })

  it("disables quick action buttons when isThinking is true", () => {
    mockUseChatStreamState.isThinking = true
    const quickActions = [
      { id: "bmad-help", name: "Help", prompt: "Analyze current state." },
    ]

    render(
      <ChatPane
        projectId="proj-1"
        activePhase="PLANNING"
        sessionId="session-1"
        isConnecting={false}
        connectionError={null}
        quickActions={quickActions}
        onRetry={vi.fn()}
      />
    )

    const helpBtn = screen.getByRole("button", { name: "Help" }) as HTMLButtonElement
    expect(helpBtn.disabled).toBe(true)
  })

  describe("Session Naming", () => {
    it("names session with action button label when initiated via quick action", async () => {
      const onUpdateSessionTitle = vi.fn()
      const quickActions = [
        { id: "bmad-help", name: "Help", prompt: "Analyze current state." },
      ]

      render(
        <ChatPane
          projectId="proj-1"
          activePhase="PLANNING"
          sessionId="session-1"
          sessionCreatedAt="2026-08-14T10:30:00Z"
          onUpdateSessionTitle={onUpdateSessionTitle}
          isConnecting={false}
          connectionError={null}
          quickActions={quickActions}
          onRetry={vi.fn()}
        />
      )

      // Click the Help quick action button
      const helpBtn = screen.getByRole("button", { name: "Help" })
      fireEvent.click(helpBtn)

      // Click Send button
      const sendBtn = screen.getByRole("button", { name: "" }) // Send button with svg icon
      const form = sendBtn.closest("form")
      if (form) {
        fireEvent.submit(form)
      } else {
        fireEvent.click(sendBtn)
      }

      expect(onUpdateSessionTitle).toHaveBeenCalledTimes(1)
      expect(onUpdateSessionTitle).toHaveBeenCalledWith("Help")
    })

    it("names session with first 5-6 words of prompt when initiated without action button", async () => {
      const onUpdateSessionTitle = vi.fn()

      render(
        <ChatPane
          projectId="proj-1"
          activePhase="PLANNING"
          sessionId="session-1"
          sessionCreatedAt="2026-08-14T10:30:00Z"
          onUpdateSessionTitle={onUpdateSessionTitle}
          isConnecting={false}
          connectionError={null}
          onRetry={vi.fn()}
        />
      )

      const textarea = screen.getByPlaceholderText(/Ask a question, enter a command/i)
      fireEvent.change(textarea, { target: { value: "Create user authentication story for login page with google oauth" } })

      const form = textarea.closest("form")
      if (form) {
        fireEvent.submit(form)
      }

      expect(onUpdateSessionTitle).toHaveBeenCalledTimes(1)
      expect(onUpdateSessionTitle).toHaveBeenCalledWith("Create user authentication story for login")
    })

    it("does not rename session if sessionTitle is already set", async () => {
      const onUpdateSessionTitle = vi.fn()

      render(
        <ChatPane
          projectId="proj-1"
          activePhase="PLANNING"
          sessionId="session-1"
          sessionTitle="Existing Session Name"
          sessionCreatedAt="2026-08-14T10:30:00Z"
          onUpdateSessionTitle={onUpdateSessionTitle}
          isConnecting={false}
          connectionError={null}
          onRetry={vi.fn()}
        />
      )

      const textarea = screen.getByPlaceholderText(/Ask a question, enter a command/i)
      fireEvent.change(textarea, { target: { value: "Some new follow up message" } })

      const form = textarea.closest("form")
      if (form) {
        fireEvent.submit(form)
      }

      expect(onUpdateSessionTitle).not.toHaveBeenCalled()
    })

    it("names session on submitMessage via ref when sessionTitle is not set", async () => {
      const onUpdateSessionTitle = vi.fn()
      const ref = React.createRef<any>()

      render(
        <ChatPane
          ref={ref}
          projectId="proj-1"
          activePhase="PLANNING"
          sessionId="session-1"
          sessionCreatedAt="2026-08-14T10:30:00Z"
          onUpdateSessionTitle={onUpdateSessionTitle}
          isConnecting={false}
          connectionError={null}
          onRetry={vi.fn()}
        />
      )

      act(() => {
        ref.current?.submitMessage("Design database schema for products and inventory items")
      })

      expect(onUpdateSessionTitle).toHaveBeenCalledTimes(1)
      expect(onUpdateSessionTitle).toHaveBeenCalledWith("Design database schema for products and")
    })
  })

  describe("Centered Initial State Layout (Jackin Style)", () => {
    it("renders centered greeting with active phase name and centered card when messages are empty", () => {
      mockUseChatStreamState.messages = []
      mockUseChatStreamState.isThinking = false

      const quickActions = [
        { id: "bmad-help", name: "Help", prompt: "Analyze current state." },
        { id: "sprint-planning", name: "Sprint Planning", prompt: "Generate sprint plan." },
      ]

      render(
        <ChatPane
          projectId="proj-1"
          activePhase="PLANNING"
          sessionId="session-1"
          isConnecting={false}
          connectionError={null}
          quickActions={quickActions}
          onRetry={vi.fn()}
        />
      )

      expect(
        screen.getByText("Hi, where should we start today in the planning phase?")
      ).not.toBeNull()
      expect(screen.getByTestId("phase-header-icon")).toBeDefined()

      const textarea = screen.getByPlaceholderText(/Ask a question, enter a command/i)
      expect(textarea).not.toBeNull()

      expect(screen.getByRole("button", { name: "Help" })).toBeDefined()
      expect(screen.getByRole("button", { name: "Sprint Planning" })).toBeDefined()
    })

    it("renders dynamic phase name and icon for Onboarding phase", () => {
      mockUseChatStreamState.messages = []
      mockUseChatStreamState.isThinking = false

      render(
        <ChatPane
          projectId="proj-1"
          activePhase="ONBOARDING"
          sessionId="session-1"
          isConnecting={false}
          connectionError={null}
          onRetry={vi.fn()}
        />
      )

      expect(
        screen.getByText("Hi, where should we start today in the onboarding phase?")
      ).not.toBeNull()
      expect(screen.getByTestId("phase-header-icon")).toBeDefined()
    })

    it("hides agent-init notification messages from rendering as active step or message thread", () => {
      mockUseChatStreamState.messages = []
      mockUseChatStreamState.isThinking = false
      mockUseChatStreamState.latestThought = {
        id: "agent-init-1",
        sender: "System",
        text: "APF Onboarding Agent initialized and ready",
        isUser: false,
        isNotification: true,
        isTransient: true,
        notificationDetails: {
          name: "agent-init",
          params: { agentName: "APF Onboarding Agent", phase: "ONBOARDING" }
        },
        timestamp: new Date().toISOString()
      }

      render(
        <ChatPane
          projectId="proj-1"
          activePhase="ONBOARDING"
          sessionId="session-1"
          isConnecting={false}
          connectionError={null}
          onRetry={vi.fn()}
        />
      )

      expect(screen.queryByText(/System is processing/i)).toBeNull()
      expect(screen.queryByText(/initialized and ready/i)).toBeNull()
      expect(
        screen.getByText("Hi, where should we start today in the onboarding phase?")
      ).not.toBeNull()
    })

    it("submits prompt from centered composer and triggers sendMessage", async () => {
      mockUseChatStreamState.messages = []
      mockUseChatStreamState.isThinking = false

      render(
        <ChatPane
          projectId="proj-1"
          activePhase="PLANNING"
          sessionId="session-1"
          isConnecting={false}
          connectionError={null}
          onRetry={vi.fn()}
        />
      )

      const textarea = screen.getByPlaceholderText(/Ask a question, enter a command/i) as HTMLTextAreaElement
      fireEvent.change(textarea, { target: { value: "Let's create the PRD" } })
      fireEvent.keyDown(textarea, { key: "Enter", shiftKey: false })

      expect(mockSendMessage).toHaveBeenCalledWith("Let's create the PRD")
    })

    it("triggers onRequestNewSession with autoSubmit=false when clicking a quick action button if messages exist", () => {
      mockUseChatStreamState.messages = [
        { id: "1", sender: "User", text: "Previous message", isUser: true, timestamp: new Date().toISOString() }
      ]
      const mockRequestNewSession = vi.fn()
      const quickActions = [
        {
          id: "bmad-prd",
          name: "Create PRD",
          prompt: "/bmad-prd",
          displayPrompt: "Create Product Requirements Document",
        }
      ]

      render(
        <ChatPane
          projectId="proj-1"
          activePhase="PLANNING"
          sessionId="session-1"
          isConnecting={false}
          connectionError={null}
          quickActions={quickActions}
          onRequestNewSession={mockRequestNewSession}
          onRetry={vi.fn()}
        />
      )

      const actionBtn = screen.getByRole("button", { name: "Create PRD" })
      fireEvent.click(actionBtn)

      expect(mockRequestNewSession).toHaveBeenCalledWith(
        expect.objectContaining({
          label: "Create PRD",
          prompt: "/bmad-prd",
          id: "bmad-prd",
          displayPrompt: "Create Product Requirements Document",
        }),
        false
      )
    })

    it("selects quick action in current session without calling onRequestNewSession when messages are empty", () => {
      mockUseChatStreamState.messages = []
      const mockRequestNewSession = vi.fn()
      const quickActions = [
        {
          id: "bmad-prd",
          name: "Create PRD",
          prompt: "/bmad-prd",
          displayPrompt: "Create Product Requirements Document",
        }
      ]

      render(
        <ChatPane
          projectId="proj-1"
          activePhase="PLANNING"
          sessionId="session-1"
          isConnecting={false}
          connectionError={null}
          quickActions={quickActions}
          onRequestNewSession={mockRequestNewSession}
          onRetry={vi.fn()}
        />
      )

      const actionBtn = screen.getByRole("button", { name: "Create PRD" })
      fireEvent.click(actionBtn)

      expect(mockRequestNewSession).not.toHaveBeenCalled()
      expect(screen.getAllByText("Create PRD").length).toBeGreaterThan(0)
    })

    it("triggers onRequestNewSession with autoSubmit=true when clicking an action chip if messages exist", () => {
      mockUseChatStreamState.messages = [
        {
          id: "1",
          sender: "Agent",
          text: "Here is your plan. Choose next step:",
          isUser: false,
          actions: ["bmad-spec"],
          timestamp: new Date().toISOString(),
        }
      ]
      const mockRequestNewSession = vi.fn()
      const quickActions = [
        {
          id: "bmad-spec",
          name: "Create Spec",
          prompt: "/bmad-spec",
          displayPrompt: "Create Technical Specification",
        }
      ]

      render(
        <ChatPane
          projectId="proj-1"
          activePhase="PLANNING"
          sessionId="session-1"
          isConnecting={false}
          connectionError={null}
          quickActions={quickActions}
          onRequestNewSession={mockRequestNewSession}
          onRetry={vi.fn()}
        />
      )

      const chipBtn = screen.getByRole("button", { name: "spec" })
      fireEvent.click(chipBtn)

      expect(mockRequestNewSession).toHaveBeenCalledWith(
        expect.objectContaining({
          label: "Spec",
          prompt: "/bmad-spec",
          id: "bmad-spec",
        }),
        true
      )
    })
  })

  describe("File & Multimodal Image Attachment (Story 18.2)", () => {
    it("renders enabled file attachment button (paperclip) in bottom composer", () => {
      render(
        <ChatPane
          projectId="proj-1"
          activePhase="PLANNING"
          sessionId="session-1"
          isConnecting={false}
          connectionError={null}
          onRetry={vi.fn()}
        />
      )

      const attachBtn = screen.getByRole("button", { name: /attach files/i })
      expect(attachBtn).not.toBeNull()
      expect((attachBtn as HTMLButtonElement).disabled).toBe(false)
      expect(attachBtn.id).toBe("tour-chat-file-upload-button")
    })

    it("renders enabled file attachment button in centered empty state composer", () => {
      mockUseChatStreamState.messages = []
      render(
        <ChatPane
          projectId="proj-1"
          activePhase="PLANNING"
          sessionId="session-1"
          isConnecting={false}
          connectionError={null}
          onRetry={vi.fn()}
        />
      )

      const attachBtn = screen.getByRole("button", { name: /attach files/i })
      expect(attachBtn).not.toBeNull()
      expect((attachBtn as HTMLButtonElement).disabled).toBe(false)
    })

    it("stages valid image file with thumbnail preview and formatted size", () => {
      render(
        <ChatPane
          projectId="proj-1"
          activePhase="PLANNING"
          sessionId="session-1"
          isConnecting={false}
          connectionError={null}
          onRetry={vi.fn()}
        />
      )

      const fileInput = document.querySelector('input[type="file"]') as HTMLInputElement
      expect(fileInput).not.toBeNull()

      const imageFile = new File(["test image content"], "diagram.png", { type: "image/png" })
      Object.defineProperty(imageFile, "size", { value: 2 * 1024 * 1024 }) // 2MB

      fireEvent.change(fileInput, { target: { files: [imageFile] } })

      expect(screen.getByText("diagram.png")).not.toBeNull()
      expect(screen.getByText("(2.0 MB)")).not.toBeNull()
      const imgPreview = screen.getByAltText("diagram.png")
      expect(imgPreview).not.toBeNull()
      expect(imgPreview.getAttribute("src")).toBe("blob:http://localhost/diagram.png")
    })

    it("stages valid code file with text icon and formatted size", () => {
      render(
        <ChatPane
          projectId="proj-1"
          activePhase="PLANNING"
          sessionId="session-1"
          isConnecting={false}
          connectionError={null}
          onRetry={vi.fn()}
        />
      )

      const fileInput = document.querySelector('input[type="file"]') as HTMLInputElement
      const codeFile = new File(["package main"], "main.go", { type: "text/plain" })
      Object.defineProperty(codeFile, "size", { value: 15 * 1024 }) // 15KB

      fireEvent.change(fileInput, { target: { files: [codeFile] } })

      expect(screen.getByText("main.go")).not.toBeNull()
      expect(screen.getByText("(15.0 KB)")).not.toBeNull()
      expect(screen.queryByAltText("main.go")).toBeNull()
    })

    it("enforces 10MB limit and shows toast error for oversized files without staging", () => {
      render(
        <ChatPane
          projectId="proj-1"
          activePhase="PLANNING"
          sessionId="session-1"
          isConnecting={false}
          connectionError={null}
          onRetry={vi.fn()}
        />
      )

      const fileInput = document.querySelector('input[type="file"]') as HTMLInputElement
      const largeFile = new File(["huge data"], "large_video.mp4", { type: "video/mp4" })
      Object.defineProperty(largeFile, "size", { value: 15 * 1024 * 1024 }) // 15MB

      fireEvent.change(fileInput, { target: { files: [largeFile] } })

      expect(mockToastError).toHaveBeenCalledWith("File exceeds 10MB limit")
      expect(screen.queryByText("large_video.mp4")).toBeNull()
    })

    it("removes staged file when clicking remove button and revokes object URL", () => {
      render(
        <ChatPane
          projectId="proj-1"
          activePhase="PLANNING"
          sessionId="session-1"
          isConnecting={false}
          connectionError={null}
          onRetry={vi.fn()}
        />
      )

      const fileInput = document.querySelector('input[type="file"]') as HTMLInputElement
      const imageFile = new File(["test image"], "diagram.png", { type: "image/png" })
      Object.defineProperty(imageFile, "size", { value: 500 * 1024 })

      fireEvent.change(fileInput, { target: { files: [imageFile] } })
      expect(screen.getByText("diagram.png")).not.toBeNull()

      const removeBtn = screen.getByRole("button", { name: "Remove diagram.png" })
      fireEvent.click(removeBtn)

      expect(screen.queryByText("diagram.png")).toBeNull()
      expect(window.URL.revokeObjectURL).toHaveBeenCalledWith("blob:http://localhost/diagram.png")
    })

    it("stages files via drag and drop onto composer form", () => {
      render(
        <ChatPane
          projectId="proj-1"
          activePhase="PLANNING"
          sessionId="session-1"
          isConnecting={false}
          connectionError={null}
          onRetry={vi.fn()}
        />
      )

      const form = document.querySelector("form")!
      const droppedFile = new File(["console.log('hi')"], "index.ts", { type: "text/typescript" })
      Object.defineProperty(droppedFile, "size", { value: 2048 })

      fireEvent.dragOver(form, { dataTransfer: { files: [droppedFile] } })
      fireEvent.drop(form, { dataTransfer: { files: [droppedFile] } })

      expect(screen.getByText("index.ts")).not.toBeNull()
      expect(screen.getByText("(2.0 KB)")).not.toBeNull()
    })

    it("uploads staged files before dispatching message and passes artifacts to sendMessage", async () => {
      render(
        <ChatPane
          projectId="proj-1"
          activePhase="PLANNING"
          sessionId="session-1"
          isConnecting={false}
          connectionError={null}
          onRetry={vi.fn()}
        />
      )

      const fileInput = document.querySelector('input[type="file"]') as HTMLInputElement
      const imgFile = new File(["pixels"], "arch.png", { type: "image/png" })
      Object.defineProperty(imgFile, "size", { value: 1024 * 1024 })

      fireEvent.change(fileInput, { target: { files: [imgFile] } })
      expect(screen.getByText("arch.png")).not.toBeNull()

      const textarea = screen.getByPlaceholderText(/Ask a question, enter a command/i)
      fireEvent.change(textarea, { target: { value: "Explain this architecture" } })

      const form = textarea.closest("form")!
      fireEvent.submit(form)

      await waitFor(() => {
        expect(mockUploadSessionArtifact).toHaveBeenCalledWith("proj-1", "session-1", imgFile)
      })

      await waitFor(() => {
        expect(mockSendMessage).toHaveBeenCalledWith(
          "Explain this architecture",
          undefined,
          [
            expect.objectContaining({
              id: "art-123",
              name: "arch.png",
            }),
          ]
        )
      })

      expect(screen.queryByText("arch.png")).toBeNull()
      expect(window.URL.revokeObjectURL).toHaveBeenCalled()
    })

    it("aborts send and preserves staged files if file upload fails", async () => {
      mockUploadSessionArtifact.mockRejectedValueOnce(new Error("Network upload error"))

      render(
        <ChatPane
          projectId="proj-1"
          activePhase="PLANNING"
          sessionId="session-1"
          isConnecting={false}
          connectionError={null}
          onRetry={vi.fn()}
        />
      )

      const fileInput = document.querySelector('input[type="file"]') as HTMLInputElement
      const imgFile = new File(["pixels"], "broken.png", { type: "image/png" })
      Object.defineProperty(imgFile, "size", { value: 1024 * 1024 })

      fireEvent.change(fileInput, { target: { files: [imgFile] } })

      const textarea = screen.getByPlaceholderText(/Ask a question, enter a command/i)
      fireEvent.change(textarea, { target: { value: "Retry this" } })

      const form = textarea.closest("form")!
      fireEvent.submit(form)

      await waitFor(() => {
        expect(mockToastError).toHaveBeenCalledWith("Network upload error")
      })

      expect(mockSendMessage).not.toHaveBeenCalled()
      expect(screen.getByText("broken.png")).not.toBeNull()
    })

    it("renders attached artifacts in chat message bubbles", () => {
      mockUseChatStreamState.messages = [
        {
          id: "msg-art-1",
          sender: "User",
          text: "Here is the diagram and config",
          isUser: true,
          artifacts: [
            {
              id: "art-1",
              name: "cloud-arch.png",
              mimeType: "image/png",
              size: 2 * 1024 * 1024,
            },
            {
              id: "art-2",
              name: "config.yaml",
              mimeType: "text/yaml",
              size: 5 * 1024,
            },
          ],
          timestamp: new Date().toISOString(),
        },
      ]

      render(
        <ChatPane
          projectId="proj-1"
          activePhase="PLANNING"
          sessionId="session-1"
          isConnecting={false}
          connectionError={null}
          onRetry={vi.fn()}
        />
      )

      expect(screen.getByText("cloud-arch.png")).not.toBeNull()
      expect(screen.getByText("(2.0 MB)")).not.toBeNull()
      expect(screen.getByText("config.yaml")).not.toBeNull()
      expect(screen.getByText("(5.0 KB)")).not.toBeNull()
    })

    it("rejects unsupported file extensions with toast error on file selection or drop", () => {
      render(
        <ChatPane
          projectId="proj-1"
          activePhase="PLANNING"
          sessionId="session-1"
          isConnecting={false}
          connectionError={null}
          onRetry={vi.fn()}
        />
      )

      const fileInput = document.querySelector('input[type="file"]') as HTMLInputElement
      const exeFile = new File(["binary data"], "malware.exe", { type: "application/x-msdownload" })
      Object.defineProperty(exeFile, "size", { value: 1024 })

      fireEvent.change(fileInput, { target: { files: [exeFile] } })

      expect(mockToastError).toHaveBeenCalledWith("Unsupported file type: malware.exe")
      expect(screen.queryByText("malware.exe")).toBeNull()
    })

    it("does not prematurely revoke object URLs when staging multiple files sequentially", () => {
      render(
        <ChatPane
          projectId="proj-1"
          activePhase="PLANNING"
          sessionId="session-1"
          isConnecting={false}
          connectionError={null}
          onRetry={vi.fn()}
        />
      )

      const fileInput = document.querySelector('input[type="file"]') as HTMLInputElement
      const img1 = new File(["img1"], "first.png", { type: "image/png" })
      Object.defineProperty(img1, "size", { value: 1024 })

      fireEvent.change(fileInput, { target: { files: [img1] } })
      expect(screen.getByText("first.png")).not.toBeNull()
      expect(window.URL.revokeObjectURL).not.toHaveBeenCalled()

      const img2 = new File(["img2"], "second.png", { type: "image/png" })
      Object.defineProperty(img2, "size", { value: 2048 })

      fireEvent.change(fileInput, { target: { files: [img2] } })
      expect(screen.getByText("first.png")).not.toBeNull()
      expect(screen.getByText("second.png")).not.toBeNull()
      expect(window.URL.revokeObjectURL).not.toHaveBeenCalled()
    })

    it("resets staged files and revokes object URLs when switching sessions", () => {
      const { rerender } = render(
        <ChatPane
          projectId="proj-1"
          activePhase="PLANNING"
          sessionId="session-1"
          isConnecting={false}
          connectionError={null}
          onRetry={vi.fn()}
        />
      )

      const fileInput = document.querySelector('input[type="file"]') as HTMLInputElement
      const img = new File(["img"], "preview.png", { type: "image/png" })
      Object.defineProperty(img, "size", { value: 1024 })

      fireEvent.change(fileInput, { target: { files: [img] } })
      expect(screen.getByText("preview.png")).not.toBeNull()

      rerender(
        <ChatPane
          projectId="proj-1"
          activePhase="PLANNING"
          sessionId="session-2"
          isConnecting={false}
          connectionError={null}
          onRetry={vi.fn()}
        />
      )

      expect(screen.queryByText("preview.png")).toBeNull()
      expect(window.URL.revokeObjectURL).toHaveBeenCalledWith("blob:http://localhost/preview.png")
    })

    it("ignores drag and drop when thinking or session is connecting", () => {
      mockUseChatStreamState.isThinking = true
      render(
        <ChatPane
          projectId="proj-1"
          activePhase="PLANNING"
          sessionId="session-1"
          isConnecting={false}
          connectionError={null}
          onRetry={vi.fn()}
        />
      )

      const form = document.querySelector("form")!
      const droppedFile = new File(["content"], "notes.md", { type: "text/markdown" })
      Object.defineProperty(droppedFile, "size", { value: 500 })

      fireEvent.drop(form, { dataTransfer: { files: [droppedFile] } })

      expect(screen.queryByText("notes.md")).toBeNull()
    })
  })
})


