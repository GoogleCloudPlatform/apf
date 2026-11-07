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

import React, { useState, useRef, useEffect } from "react"
import { 
  X, 
  Copy, 
  Mail, 
  ChevronDown, 
  Link, 
  User as UserIcon,
  Check,
  ArrowLeft
} from "lucide-react"
import { toast } from "sonner"
import { api } from "@/lib/api"
import { useAuthStore } from "@/stores/auth"

export interface ShareProjectModalProps {
  isOpen: boolean
  onClose: () => void
  projectName: string
  projectId: string
  projectLink?: string
  isTiedToRepo?: boolean
}

interface AccessUser {
  id: string
  name: string
  email: string
  role: "Owner" | "Editor" | "Viewer"
  avatarUrl?: string
  isCurrentUser?: boolean
}

interface PendingRecipient {
  email: string
  name: string
  initials: string
}

export function ShareProjectModal({
  isOpen,
  onClose,
  projectName,
  projectId,
  projectLink = window?.location?.href || "https://foundry.google.com/project/requirements-spec",
  isTiedToRepo = false,
}: ShareProjectModalProps) {
  const { email: currentUserEmail } = useAuthStore()
  
  // State for people with access
  const [people, setPeople] = useState<AccessUser[]>([])
  const [isLoading, setIsLoading] = useState(false)

  // Standard UI Dropdowns
  const [activeDropdown, setActiveDropdown] = useState<string | null>(null)

  // Invitation Flow State
  const [pendingChips, setPendingChips] = useState<PendingRecipient[]>([])
  const [searchInput, setSearchInput] = useState("")
  const [pendingRole, setPendingRole] = useState<"Editor" | "Viewer">("Editor")
  const [showRoleDropdown, setShowRoleDropdown] = useState(false)
  const [notifyPeople, setNotifyPeople] = useState(true)
  const [messageText, setMessageText] = useState("")

  const inputRef = useRef<HTMLInputElement>(null)

  // Fetch members from API
  const fetchMembers = async () => {
    if (!projectId) return
    setIsLoading(true)
    try {
      const response = await api.get(`/api/v1/projects/${projectId}/members`)
      if (!response.ok) {
        throw new Error(`Failed to fetch members: ${response.statusText}`)
      }
      const data = await response.json()
      
      const mappedPeople: AccessUser[] = data.map((m: any) => ({
        id: m.id,
        name: m.name || m.email.split('@')[0],
        email: m.email,
        role: m.role as "Owner" | "Editor" | "Viewer",
        isCurrentUser: m.email.toLowerCase() === currentUserEmail?.toLowerCase(),
      }))
      setPeople(mappedPeople)
    } catch (err) {
      console.error("Failed to fetch project members:", err)
      toast.error("Failed to load project members")
    } finally {
      setIsLoading(false)
    }
  }

  // Fetch members when modal opens
  useEffect(() => {
    if (isOpen) {
      fetchMembers()
    }
  }, [isOpen, projectId, currentUserEmail])

  // Focus input when chips change or modal opens
  useEffect(() => {
    if (isOpen && inputRef.current) {
      inputRef.current.focus()
    }
  }, [isOpen, pendingChips.length])

  if (!isOpen) return null

  // Email Validation helper
  const isValidEmail = (emailStr: string) => {
    const emailRegex = /^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/
    return emailRegex.test(emailStr)
  }

  // Convert raw text into a recipient chip
  const convertInputToChip = (text: string, showToastOnError = true) => {
    const cleanText = text.replace(/,$/, "").trim()
    if (!cleanText) return false

    if (!isValidEmail(cleanText)) {
      if (showToastOnError) {
        toast.error("Please enter a valid email address.")
      }
      return false
    }

    const email = cleanText.toLowerCase()
    
    // Avoid duplicate pending chips
    if (pendingChips.some(c => c.email === email)) {
      setSearchInput("")
      return true
    }

    // Avoid adding people who already have access
    if (people.some(p => p.email === email)) {
      toast.error(`${email} already has access.`)
      setSearchInput("")
      return true
    }

    const name = email.split("@")[0].split(".").map(part => part.charAt(0).toUpperCase() + part.slice(1)).join(" ")
    const initials = name.split(" ").map(n => n[0]).join("").slice(0, 2).toUpperCase()

    setPendingChips([...pendingChips, { email, name, initials }])
    setSearchInput("")
    return true
  }

  // Monitor key presses: Space, Comma, Enter
  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === " " || e.key === "," || e.key === "Enter") {
      const converted = convertInputToChip(searchInput)
      if (converted) {
        e.preventDefault()
      }
    } else if (e.key === "Backspace" && !searchInput && pendingChips.length > 0) {
      // Remove last chip on backspace
      setPendingChips(pendingChips.slice(0, -1))
    }
  }

  const handleBlur = () => {
    convertInputToChip(searchInput, false)
  }

  // Monitor paste events
  const handlePaste = (e: React.ClipboardEvent<HTMLInputElement>) => {
    const pasteData = e.clipboardData.getData("text")
    if (isValidEmail(pasteData.trim())) {
      e.preventDefault()
      convertInputToChip(pasteData.trim())
    }
  }

  const handleRemoveChip = (email: string) => {
    setPendingChips(pendingChips.filter(c => c.email !== email))
  }

  const handleRoleChange = async (userId: string, newRole: "Owner" | "Editor" | "Viewer") => {
    if (newRole === "Owner") {
      toast.error("Cannot change role to Owner")
      return
    }

    setIsLoading(true)
    try {
      const response = await api.put(`/api/v1/projects/${projectId}/members/${userId}`, {
        role: newRole,
      })

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}))
        throw new Error(errorData.message || `Failed to update role: ${response.statusText}`)
      }

      setPeople(people.map(p => p.id === userId ? { ...p, role: newRole } : p))
      toast.success(`Updated role to ${newRole}`)
    } catch (err: any) {
      console.error("Failed to update member role:", err)
      toast.error(err.message || "Failed to update member role")
    } finally {
      setIsLoading(false)
      setActiveDropdown(null)
    }
  }

  const handleRemovePerson = async (userId: string) => {
    const person = people.find(p => p.id === userId)
    if (person?.isCurrentUser) {
      toast.error("You cannot remove yourself.")
      return
    }

    setIsLoading(true)
    try {
      const response = await api.delete(`/api/v1/projects/${projectId}/members/${userId}`)

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}))
        throw new Error(errorData.message || `Failed to remove member: ${response.statusText}`)
      }

      setPeople(people.filter(p => p.id !== userId))
      toast.success(`Removed ${person?.name || "user"} from access list.`)
    } catch (err: any) {
      console.error("Failed to remove member:", err)
      toast.error(err.message || "Failed to remove member")
    } finally {
      setIsLoading(false)
    }
  }

  const handleCopyLink = () => {
    navigator.clipboard.writeText(projectLink)
    toast.success("Link copied to clipboard!")
  }

  const handleCopyEmails = () => {
    const emails = people.map(p => p.email).join(", ")
    navigator.clipboard.writeText(emails)
    toast.success("All access member emails copied!")
  }

  const handleClearInvitationState = () => {
    setPendingChips([])
    setSearchInput("")
    setMessageText("")
    setNotifyPeople(true)
  }

  const handleSendInvitation = async () => {
    if (pendingChips.length === 0) return
    setIsLoading(true)
    try {
      const emails = pendingChips.map(c => c.email)
      const response = await api.post(`/api/v1/projects/${projectId}/members`, {
        emails,
        role: pendingRole,
      })

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}))
        throw new Error(errorData.message || `Failed to add members: ${response.statusText}`)
      }

      const result = await response.json()
      
      if (result.status === "ERROR") {
        throw new Error(result.message || "Failed to add members")
      }

      toast.success(`Access granted to ${pendingChips.length} recipient(s) as ${pendingRole}.`)
      handleClearInvitationState()
      await fetchMembers()
    } catch (err: any) {
      console.error("Failed to add project members:", err)
      toast.error(err.message || "Failed to add project members")
    } finally {
      setIsLoading(false)
    }
  }

  const getInitials = (name: string) => {
    return name.split(" ").map(n => n[0]).join("").slice(0, 2).toUpperCase()
  }

  const isInviting = pendingChips.length > 0

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-[2px] p-4 animate-in fade-in duration-150">
      <div className="bg-card border border-border rounded-3xl w-full max-w-[540px] shadow-2xl overflow-hidden animate-in zoom-in-95 duration-200 text-foreground">
        
        {/* Main Content Area */}
        <div className="p-6 space-y-6">
          
          {/* Header */}
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              {isInviting && (
                <button 
                  onClick={handleClearInvitationState}
                  disabled={isLoading}
                  className="p-1.5 rounded-full hover:bg-muted text-foreground transition-colors -ml-1 disabled:opacity-50"
                  title="Go back"
                >
                  <ArrowLeft className="h-5 w-5" />
                </button>
              )}
              <h2 className="text-xl font-medium tracking-tight">
                Share "{projectName}"
              </h2>
            </div>
            <div className="flex items-center gap-1.5 text-muted-foreground">
              {!isInviting && (
                <button 
                  onClick={onClose}
                  disabled={isLoading}
                  className="p-2 rounded-full hover:bg-muted hover:text-foreground transition-colors ml-1 disabled:opacity-50"
                  title="Close"
                >
                  <X className="h-5 w-5" />
                </button>
              )}
            </div>
          </div>

          {/* Invitation / Add Members Section */}
          <div className="space-y-4">
            
            {/* Input Row (with right-side role dropdown when inviting) */}
            <div className="flex items-start gap-3">
              
              {/* Multi-Chip Input Area */}
              <div className="flex-1 relative group">
                <div 
                  className="flex flex-wrap items-center gap-2 border border-input rounded-xl focus-within:border-primary focus-within:ring-1 focus-within:ring-primary transition-all p-2.5 bg-background min-h-[56px]"
                  onClick={() => !isLoading && inputRef.current?.focus()}
                >
                  {/* Floating Outline Label */}
                  <label 
                    className="absolute -top-2.5 left-3 px-1.5 bg-card text-xs font-normal text-primary transition-all group-focus-within:text-primary"
                  >
                    Add people by email address
                  </label>

                  {/* Rendered Recipient Chips */}
                  {pendingChips.map((chip) => (
                    <div 
                      key={chip.email}
                      className="inline-flex items-center gap-1.5 bg-muted border border-border pl-1 pr-1.5 py-0.5 rounded-full text-xs font-normal text-foreground select-none max-w-full shrink-0"
                    >
                      <div className="h-5 w-5 rounded-full bg-primary/10 border border-primary/20 flex items-center justify-center text-[9px] font-bold text-primary overflow-hidden shrink-0">
                        {chip.initials}
                      </div>
                      <span className="truncate max-w-[120px] text-foreground font-medium">{chip.name}</span>
                      <button
                        type="button"
                        disabled={isLoading}
                        onClick={(e) => {
                          e.stopPropagation()
                          handleRemoveChip(chip.email)
                        }}
                        className="hover:bg-muted-foreground/20 rounded-full p-0.5 text-muted-foreground hover:text-foreground transition-colors shrink-0 disabled:opacity-50"
                      >
                        <X className="h-3 w-3" />
                      </button>
                    </div>
                  ))}

                  {/* Text Input Field */}
                  <input
                    ref={inputRef}
                    type="text"
                    value={searchInput}
                    onChange={(e) => setSearchInput(e.target.value)}
                    onKeyDown={handleKeyDown}
                    onPaste={handlePaste}
                    onBlur={handleBlur}
                    disabled={isLoading}
                    placeholder={pendingChips.length === 0 ? "Enter email addresses..." : ""}
                    className="flex-1 bg-transparent border-none focus:outline-none focus:ring-0 text-sm placeholder:text-muted-foreground/50 h-7 min-w-[120px] disabled:opacity-50"
                  />
                </div>
              </div>

              {/* Right-Side Role Selector (Visible only in invitation state) */}
              {isInviting && (
                <div className="relative shrink-0">
                  <button
                    type="button"
                    onClick={() => setShowRoleDropdown(!showRoleDropdown)}
                    disabled={isLoading || isTiedToRepo}
                    title={isTiedToRepo ? "User role depends on their role in associated code repository" : undefined}
                    className="h-[56px] px-4 rounded-xl border border-input bg-background text-sm font-medium text-foreground hover:bg-muted disabled:opacity-50 disabled:cursor-not-allowed transition-colors flex items-center gap-2 select-none"
                  >
                    {pendingRole}
                    <ChevronDown className="h-4 w-4 text-muted-foreground" />
                  </button>

                  {showRoleDropdown && (
                    <div className="absolute right-0 top-full mt-1.5 z-10 w-36 bg-popover border border-border rounded-xl shadow-xl py-1 animate-in fade-in slide-in-from-top-1 duration-100">
                      {(["Viewer", "Editor"] as const).map((role) => (
                        <button
                          key={role}
                          type="button"
                          onClick={() => {
                            setPendingRole(role)
                            setShowRoleDropdown(false)
                          }}
                          className="w-full flex items-center justify-between px-3.5 py-2 text-left text-xs hover:bg-muted text-foreground transition-colors"
                        >
                          {role}
                          {pendingRole === role && <Check className="h-3.5 w-3.5 text-primary shrink-0" />}
                        </button>
                      ))}
                    </div>
                  )}
                </div>
              )}
            </div>

            {/* Notification and Messaging Fields (Inviting State) */}
            {isInviting && (
              <div className="space-y-4 pt-1 animate-in fade-in slide-in-from-top-2 duration-200">
                
                {/* Notify People Checkbox */}
                <div className="flex items-center gap-2.5 select-none">
                  <input
                    type="checkbox"
                    id="notify-people-checkbox"
                    checked={notifyPeople}
                    onChange={(e) => setNotifyPeople(e.target.checked)}
                    disabled={isLoading}
                    className="h-4 w-4 rounded border-input bg-background accent-primary focus:ring-primary/25 disabled:opacity-50"
                  />
                  <label htmlFor="notify-people-checkbox" className="text-sm font-medium text-foreground">
                    Notify people
                  </label>
                </div>

                {/* Message Box */}
                {notifyPeople && (
                  <div className="space-y-1.5">
                    <textarea
                      value={messageText}
                      onChange={(e) => setMessageText(e.target.value)}
                      disabled={isLoading}
                      placeholder="Message"
                      rows={4}
                      className="w-full p-3.5 rounded-xl border border-input bg-background text-sm transition-all focus:outline-none focus:ring-2 focus:ring-primary/25 focus:border-primary placeholder:text-muted-foreground/50 resize-none disabled:opacity-50"
                    />
                  </div>
                )}
              </div>
            )}
          </div>

          {/* Standard View Fields: People List and General Access (Hidden when inviting) */}
          {!isInviting && (
            <>
              {/* People with Access */}
              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <h3 className="text-sm font-medium text-foreground">
                    People with access
                  </h3>
                  <div className="flex items-center gap-1 text-muted-foreground">
                    <button 
                      onClick={handleCopyEmails}
                      disabled={isLoading}
                      className="p-2 rounded-full hover:bg-muted hover:text-foreground transition-colors disabled:opacity-50"
                      title="Copy all emails"
                    >
                      <Copy className="h-4 w-4" />
                    </button>
                    <button 
                      onClick={() => toast.info("Open mail client to email members")}
                      disabled={isLoading}
                      className="p-2 rounded-full hover:bg-muted hover:text-foreground transition-colors disabled:opacity-50"
                      title="Email members"
                    >
                      <Mail className="h-4 w-4" />
                    </button>
                  </div>
                </div>

                {/* List of People */}
                <div className="space-y-4 max-h-[180px] overflow-y-auto pr-1 scrollbar-thin">
                  {isLoading && people.length === 0 ? (
                    <div className="text-center py-8 text-sm text-muted-foreground">Loading members...</div>
                  ) : people.length === 0 ? (
                    <div className="text-center py-8 text-sm text-muted-foreground">No members found.</div>
                  ) : (
                    people.map((person) => (
                      <div key={person.id} className="flex items-center justify-between group/row">
                        <div className="flex items-center gap-3">
                          <div className="h-9 w-9 rounded-full bg-primary/10 border border-primary/20 flex items-center justify-center text-xs font-bold text-primary shrink-0">
                            {person.avatarUrl ? (
                              <img src={person.avatarUrl} alt={person.name} className="h-full w-full rounded-full object-cover" />
                            ) : (
                              getInitials(person.name)
                            )}
                          </div>
                          <div className="min-w-0">
                            <div className="text-sm font-normal text-foreground truncate flex items-center gap-1">
                              {person.name}
                              {person.isCurrentUser && <span className="text-muted-foreground/60 text-xs">(you)</span>}
                            </div>
                            <div className="text-xs text-muted-foreground truncate">
                              {person.email}
                            </div>
                          </div>
                        </div>

                        {/* Role dropdown */}
                        <div className="relative flex items-center gap-1">
                          {person.role === "Owner" || person.isCurrentUser ? (
                            <span className="text-xs text-muted-foreground font-normal px-2 py-1 select-none mr-2">
                              {person.role}
                            </span>
                          ) : (
                            <>
                              <button
                                onClick={() => handleRemovePerson(person.id)}
                                disabled={isLoading}
                                className="p-1.5 rounded-lg text-muted-foreground/40 hover:text-destructive hover:bg-destructive/10 opacity-0 group-hover/row:opacity-100 transition-all mr-1 disabled:opacity-50"
                                title={`Revoke access for ${person.name}`}
                              >
                                <X className="h-3.5 w-3.5" />
                              </button>

                              <button
                                onClick={() => setActiveDropdown(activeDropdown === person.id ? null : person.id)}
                                disabled={isLoading || isTiedToRepo}
                                title={isTiedToRepo ? "User role depends on their role in associated code repository" : undefined}
                                className="inline-flex items-center gap-1 px-2.5 py-1.5 rounded-lg text-xs font-normal hover:bg-muted disabled:opacity-50 disabled:cursor-not-allowed text-foreground transition-colors"
                              >
                                {person.role}
                                <ChevronDown className="h-3.5 w-3.5 text-muted-foreground" />
                              </button>

                              {activeDropdown === person.id && (
                                <div className="absolute right-0 top-full mt-1.5 z-10 w-36 bg-popover border border-border rounded-xl shadow-xl py-1 animate-in fade-in slide-in-from-top-1 duration-100">
                                  {([ "Viewer", "Editor" ] as const).map((role) => (
                                    <button
                                      key={role}
                                      onClick={() => handleRoleChange(person.id, role)}
                                      disabled={isLoading}
                                      className="w-full flex items-center justify-between px-3.5 py-2 text-left text-xs hover:bg-muted text-foreground transition-colors disabled:opacity-50"
                                    >
                                      {role}
                                      {person.role === role && <Check className="h-3.5 w-3.5 text-primary shrink-0" />}
                                    </button>
                                  ))}
                                </div>
                              )}
                            </>
                          )}
                        </div>
                      </div>
                    ))
                  )}
                </div>
              </div>
            </>
          )}

        </div>

        {/* Footer Actions (Dynamic based on state) */}
        <div className="bg-muted/30 px-6 py-4 flex items-center justify-between border-t border-border shrink-0">
          {isInviting ? (
            <>
              {/* Copy Link Button */}
              <button
                type="button"
                onClick={handleCopyLink}
                disabled={isLoading}
                className="inline-flex h-10 w-10 items-center justify-center rounded-full border border-input bg-background hover:bg-muted text-foreground transition-colors shadow-sm disabled:opacity-50"
                title="Copy link"
              >
                <Link className="h-4.5 w-4.5" />
              </button>
              <div className="flex items-center gap-3">
                <button
                  type="button"
                  onClick={handleClearInvitationState}
                  disabled={isLoading}
                  className="inline-flex h-10 items-center justify-center px-5 text-sm font-semibold text-primary hover:bg-muted rounded-full transition-colors disabled:opacity-50"
                >
                  Cancel
                </button>
                <button
                  type="button"
                  onClick={handleSendInvitation}
                  disabled={isLoading || pendingChips.length === 0}
                  className="inline-flex h-10 items-center justify-center rounded-full bg-primary hover:opacity-95 px-6 text-sm font-semibold text-primary-foreground shadow transition-all disabled:opacity-50"
                >
                  Send
                </button>
              </div>
            </>
          ) : (
            <>
              <button
                type="button"
                onClick={handleCopyLink}
                disabled={isLoading}
                className="inline-flex h-10 items-center justify-center gap-2 rounded-full border border-input bg-background hover:bg-muted px-5 text-sm font-medium text-primary transition-colors shadow-sm disabled:opacity-50"
              >
                <Link className="h-4 w-4" />
                Copy link
              </button>
              <button
                type="button"
                onClick={onClose}
                disabled={isLoading}
                className="inline-flex h-10 items-center justify-center rounded-full bg-primary hover:opacity-95 px-6 text-sm font-semibold text-primary-foreground shadow transition-all disabled:opacity-50"
              >
                Done
              </button>
            </>
          )}
        </div>

      </div>
    </div>
  )
}

export default ShareProjectModal
