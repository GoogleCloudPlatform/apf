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

import { useState, useEffect, useRef, useCallback } from "react"

// Minimal TypeScript definitions for Web Speech API to support browsers without `@types/dom-speech-recognition`
interface SpeechRecognitionEvent extends Event {
  readonly resultIndex: number
  readonly results: SpeechRecognitionResultList
}

interface SpeechRecognitionResultList {
  readonly length: number
  item(index: number): SpeechRecognitionResult
  [index: number]: SpeechRecognitionResult
}

interface SpeechRecognitionResult {
  readonly isFinal: boolean
  readonly length: number
  item(index: number): SpeechRecognitionAlternative
  [index: number]: SpeechRecognitionAlternative
}

interface SpeechRecognitionAlternative {
  readonly transcript: string
  readonly confidence: number
}

interface SpeechRecognitionErrorEvent extends Event {
  readonly error: string
  readonly message?: string
}

interface SpeechRecognitionInstance extends EventTarget {
  continuous: boolean
  interimResults: boolean
  lang: string
  onstart: ((this: SpeechRecognitionInstance, ev: Event) => any) | null
  onresult: ((this: SpeechRecognitionInstance, ev: SpeechRecognitionEvent) => any) | null
  onerror: ((this: SpeechRecognitionInstance, ev: SpeechRecognitionErrorEvent) => any) | null
  onend: ((this: SpeechRecognitionInstance, ev: Event) => any) | null
  start(): void
  stop(): void
  abort(): void
}

interface SpeechRecognitionConstructor {
  new (): SpeechRecognitionInstance
}

declare global {
  interface Window {
    SpeechRecognition?: SpeechRecognitionConstructor
    webkitSpeechRecognition?: SpeechRecognitionConstructor
  }
}

export interface UseSpeechRecognitionOptions {
  onResult?: (transcript: string, isFinal: boolean) => void
  onError?: (error: string) => void
  onEnd?: () => void
  continuous?: boolean
  interimResults?: boolean
  lang?: string
}

export interface UseSpeechRecognitionReturn {
  isListening: boolean
  isSupported: boolean
  transcript: string
  interimTranscript: string
  error: string | null
  startListening: () => void
  stopListening: () => void
  resetTranscript: () => void
}

export function useSpeechRecognition(options: UseSpeechRecognitionOptions = {}): UseSpeechRecognitionReturn {
  const {
    onResult,
    onError,
    onEnd,
    continuous = true,
    interimResults = true,
    lang = "en-US",
  } = options

  const [isListening, setIsListening] = useState<boolean>(false)
  const [isSupported, setIsSupported] = useState<boolean>(false)
  const [transcript, setTranscript] = useState<string>("")
  const [interimTranscript, setInterimTranscript] = useState<string>("")
  const [error, setError] = useState<string | null>(null)

  const recognitionRef = useRef<SpeechRecognitionInstance | null>(null)
  const optionsRef = useRef(options)

  useEffect(() => {
    optionsRef.current = options
  }, [options])

  useEffect(() => {
    if (typeof window !== "undefined") {
      const SpeechRec = window.SpeechRecognition || window.webkitSpeechRecognition
      setIsSupported(!!SpeechRec)
    }
  }, [])

  const stopListening = useCallback(() => {
    if (recognitionRef.current) {
      try {
        recognitionRef.current.stop()
      } catch {
        // Ignore if recognition already stopped
      }
    }
    setIsListening(false)
  }, [])

  const resetTranscript = useCallback(() => {
    setTranscript("")
    setInterimTranscript("")
    setError(null)
  }, [])

  const startListening = useCallback(() => {
    if (typeof window === "undefined") return

    const SpeechRec = window.SpeechRecognition || window.webkitSpeechRecognition
    if (!SpeechRec) {
      const unsupportedErr = "Speech recognition is not supported in this browser."
      setError(unsupportedErr)
      optionsRef.current.onError?.(unsupportedErr)
      return
    }

    // Stop existing instance if already listening
    if (recognitionRef.current) {
      try {
        recognitionRef.current.abort()
      } catch {
        // Ignore abort errors
      }
    }

    const recognition = new SpeechRec()
    recognition.continuous = continuous
    recognition.interimResults = interimResults
    recognition.lang = lang

    recognition.onstart = () => {
      setIsListening(true)
      setError(null)
    }

    recognition.onresult = (event: SpeechRecognitionEvent) => {
      let finalStr = ""
      let interimStr = ""

      for (let i = 0; i < event.results.length; i++) {
        const result = event.results[i]
        if (result.isFinal) {
          finalStr += result[0].transcript
        } else {
          interimStr += result[0].transcript
        }
      }

      const combined = (finalStr + " " + interimStr).trim()
      setTranscript(finalStr)
      setInterimTranscript(interimStr)

      optionsRef.current.onResult?.(combined, !interimStr && Boolean(finalStr))
    }

    recognition.onerror = (event: SpeechRecognitionErrorEvent) => {
      let errorMessage = "Speech recognition error occurred."
      switch (event.error) {
        case "not-allowed":
        case "permission-denied":
          errorMessage = "Microphone permission denied. Please allow microphone access in browser settings."
          break
        case "no-speech":
          errorMessage = "No speech was detected. Please try speaking again."
          break
        case "audio-capture":
          errorMessage = "No microphone was found or microphone is not working."
          break
        case "network":
          errorMessage = "Network error occurred during speech recognition."
          break
        case "aborted":
          // Normal abort, don't surface as error
          return
      }

      setError(errorMessage)
      setIsListening(false)
      optionsRef.current.onError?.(errorMessage)
    }

    recognition.onend = () => {
      setIsListening(false)
      optionsRef.current.onEnd?.()
    }

    recognitionRef.current = recognition

    try {
      recognition.start()
    } catch (err: any) {
      const startErr = err?.message || "Failed to start speech recognition."
      setError(startErr)
      setIsListening(false)
      optionsRef.current.onError?.(startErr)
    }
  }, [continuous, interimResults, lang])

  useEffect(() => {
    return () => {
      if (recognitionRef.current) {
        try {
          recognitionRef.current.abort()
        } catch {
          // Ignore unmount abort errors
        }
      }
    }
  }, [])

  return {
    isListening,
    isSupported,
    transcript,
    interimTranscript,
    error,
    startListening,
    stopListening,
    resetTranscript,
  }
}
