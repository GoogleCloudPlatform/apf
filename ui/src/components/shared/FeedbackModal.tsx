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

import React, { useState, useEffect } from "react"
import { useAuthStore } from "@/stores/auth"
import { submitFeedback } from "@/lib/api"
import { FeedbackType } from "@/lib/types"
import { toast } from "sonner"
import { X, Loader2, MessageSquarePlus, Bug, Lightbulb, MessageCircle } from "lucide-react"

export interface FeedbackModalProps {
  isOpen: boolean
  onClose: () => void
  initialType?: FeedbackType
  projectId?: string | null
}

export function FeedbackModal({
  isOpen,
  onClose,
  initialType = "general",
  projectId,
}: FeedbackModalProps) {
  const { email: authEmail, selectedOrgId } = useAuthStore()

  const [type, setType] = useState<FeedbackType>(initialType)
  const [subject, setSubject] = useState("")
  const [message, setMessage] = useState("")
  const [userEmail, setUserEmail] = useState(authEmail || "")
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [errorMsg, setErrorMsg] = useState<string | null>(null)

  useEffect(() => {
    if (isOpen) {
      setType(initialType)
      setSubject("")
      setMessage("")
      setUserEmail(authEmail || "")
      setErrorMsg(null)
    }
  }, [isOpen, initialType, authEmail])

  useEffect(() => {
    if (!isOpen) return
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape" && !isSubmitting) {
        onClose()
      }
    }
    window.addEventListener("keydown", handleKeyDown)
    return () => window.removeEventListener("keydown", handleKeyDown)
  }, [isOpen, isSubmitting, onClose])

  if (!isOpen) return null

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setErrorMsg(null)

    const trimmedSubject = subject.trim()
    const trimmedMessage = message.trim()

    if (!trimmedSubject) {
      setErrorMsg("Subject is required.")
      return
    }
    if (trimmedSubject.length > 200) {
      setErrorMsg("Subject must not exceed 200 characters.")
      return
    }
    if (!trimmedMessage) {
      setErrorMsg("Message is required.")
      return
    }
    if (trimmedMessage.length > 4000) {
      setErrorMsg("Message must not exceed 4000 characters.")
      return
    }

    setIsSubmitting(true)
    try {
      await submitFeedback({
        type,
        subject: trimmedSubject,
        message: trimmedMessage,
        userEmail: userEmail.trim() || undefined,
        organizationId: selectedOrgId || undefined,
        projectId: projectId || undefined,
      })

      toast.success("Feedback submitted successfully! Thank you for helping us improve.")
      onClose()
    } catch (err: any) {
      const msg = err?.message || "Failed to submit feedback. Please try again."
      setErrorMsg(msg)
      toast.error(msg)
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm p-4 animate-in fade-in duration-200"
      onClick={(e) => {
        if (e.target === e.currentTarget && !isSubmitting) {
          onClose()
        }
      }}
      data-testid="feedback-modal-backdrop"
    >
      <div
        className="w-full max-w-lg rounded-xl border border-border bg-background shadow-2xl overflow-hidden animate-in zoom-in-95 duration-200"
        role="dialog"
        aria-modal="true"
        aria-labelledby="feedback-modal-title"
      >
        {/* Header */}
        <div className="flex items-center justify-between border-b border-border px-6 py-4 bg-muted/40">
          <div className="flex items-center gap-2.5">
            <div className="p-2 rounded-lg bg-blue-500/10 text-blue-600 dark:text-blue-400">
              <MessageSquarePlus className="h-5 w-5" />
            </div>
            <div>
              <h2 id="feedback-modal-title" className="text-base font-semibold text-foreground">
                Submit Feedback
              </h2>
              <p className="text-xs text-muted-foreground">
                Help us improve APF by sharing bugs, ideas, or comments.
              </p>
            </div>
          </div>
          <button
            type="button"
            onClick={onClose}
            disabled={isSubmitting}
            className="rounded-lg p-1.5 text-muted-foreground hover:bg-accent hover:text-foreground transition-colors disabled:opacity-50"
            aria-label="Close dialog"
            data-testid="feedback-modal-close-btn"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        {/* Form Body */}
        <form onSubmit={handleSubmit} className="p-6 space-y-4">
          {errorMsg && (
            <div className="rounded-md bg-destructive/15 p-3 text-xs font-medium text-destructive border border-destructive/20" role="alert">
              {errorMsg}
            </div>
          )}

          {/* Feedback Type Selector */}
          <div>
            <label className="block text-xs font-medium text-foreground mb-1.5">
              Feedback Type
            </label>
            <div className="grid grid-cols-3 gap-2">
              <button
                type="button"
                onClick={() => setType("bug")}
                className={`flex items-center justify-center gap-1.5 px-3 py-2 text-xs font-medium rounded-lg border transition-all ${
                  type === "bug"
                    ? "border-red-500 bg-red-500/10 text-red-600 dark:text-red-400 font-semibold"
                    : "border-border bg-background hover:bg-muted text-muted-foreground"
                }`}
                data-testid="feedback-type-bug"
              >
                <Bug className="h-3.5 w-3.5" />
                <span>Bug Report</span>
              </button>

              <button
                type="button"
                onClick={() => setType("feature_request")}
                className={`flex items-center justify-center gap-1.5 px-3 py-2 text-xs font-medium rounded-lg border transition-all ${
                  type === "feature_request"
                    ? "border-purple-500 bg-purple-500/10 text-purple-600 dark:text-purple-400 font-semibold"
                    : "border-border bg-background hover:bg-muted text-muted-foreground"
                }`}
                data-testid="feedback-type-feature"
              >
                <Lightbulb className="h-3.5 w-3.5" />
                <span>Feature</span>
              </button>

              <button
                type="button"
                onClick={() => setType("general")}
                className={`flex items-center justify-center gap-1.5 px-3 py-2 text-xs font-medium rounded-lg border transition-all ${
                  type === "general"
                    ? "border-blue-500 bg-blue-500/10 text-blue-600 dark:text-blue-400 font-semibold"
                    : "border-border bg-background hover:bg-muted text-muted-foreground"
                }`}
                data-testid="feedback-type-general"
              >
                <MessageCircle className="h-3.5 w-3.5" />
                <span>General</span>
              </button>
            </div>
          </div>

          {/* Subject Field */}
          <div>
            <label htmlFor="feedback-subject" className="block text-xs font-medium text-foreground mb-1">
              Subject <span className="text-destructive">*</span>
            </label>
            <input
              id="feedback-subject"
              type="text"
              value={subject}
              onChange={(e) => setSubject(e.target.value)}
              placeholder="Short title or summary..."
              maxLength={200}
              className="w-full px-3 py-2 rounded-lg border border-input bg-background text-xs text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
              required
              data-testid="feedback-subject-input"
            />
          </div>

          {/* Message Textarea */}
          <div>
            <label htmlFor="feedback-message" className="block text-xs font-medium text-foreground mb-1">
              Message <span className="text-destructive">*</span>
            </label>
            <textarea
              id="feedback-message"
              rows={4}
              value={message}
              onChange={(e) => setMessage(e.target.value)}
              placeholder="Describe your feedback, steps to reproduce, or suggestions..."
              maxLength={4000}
              className="w-full px-3 py-2 rounded-lg border border-input bg-background text-xs text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring resize-none"
              required
              data-testid="feedback-message-input"
            />
          </div>

          {/* User Email Field */}
          <div>
            <label htmlFor="feedback-email" className="block text-xs font-medium text-foreground mb-1">
              Contact Email (optional)
            </label>
            <input
              id="feedback-email"
              type="email"
              value={userEmail}
              onChange={(e) => setUserEmail(e.target.value)}
              placeholder="your.email@example.com"
              className="w-full px-3 py-2 rounded-lg border border-input bg-background text-xs text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
              data-testid="feedback-email-input"
            />
          </div>

          {/* Actions */}
          <div className="flex items-center justify-end gap-2 pt-2 border-t border-border">
            <button
              type="button"
              onClick={onClose}
              disabled={isSubmitting}
              className="px-4 py-2 text-xs font-medium rounded-lg border border-border bg-background text-foreground hover:bg-muted transition-colors disabled:opacity-50"
              data-testid="feedback-cancel-btn"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={isSubmitting}
              className="flex items-center gap-1.5 px-4 py-2 text-xs font-medium rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50 shadow-sm"
              data-testid="feedback-submit-btn"
            >
              {isSubmitting && <Loader2 className="h-3.5 w-3.5 animate-spin" />}
              <span>Submit Feedback</span>
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
