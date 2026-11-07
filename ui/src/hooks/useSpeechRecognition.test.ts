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
import { renderHook, act } from "@testing-library/react"
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import { useSpeechRecognition } from "./useSpeechRecognition"

describe("useSpeechRecognition", () => {
  let mockRecognitionInstance: any

  beforeEach(() => {
    mockRecognitionInstance = {
      continuous: false,
      interimResults: false,
      lang: "en-US",
      onstart: null,
      onresult: null,
      onerror: null,
      onend: null,
      start: vi.fn(() => {
        mockRecognitionInstance.onstart?.(new Event("start"))
      }),
      stop: vi.fn(() => {
        mockRecognitionInstance.onend?.(new Event("end"))
      }),
      abort: vi.fn(),
    }

    class MockSpeechRecognition {
      constructor() {
        return mockRecognitionInstance
      }
    }

    ;(window as any).SpeechRecognition = MockSpeechRecognition
  })

  afterEach(() => {
    delete (window as any).SpeechRecognition
    delete (window as any).webkitSpeechRecognition
    vi.clearAllMocks()
  })

  it("detects browser support when SpeechRecognition is available", () => {
    const { result } = renderHook(() => useSpeechRecognition())
    expect(result.current.isSupported).toBe(true)
  })

  it("detects lack of browser support when SpeechRecognition is missing", () => {
    delete (window as any).SpeechRecognition
    delete (window as any).webkitSpeechRecognition

    const { result } = renderHook(() => useSpeechRecognition())
    expect(result.current.isSupported).toBe(false)
  })

  it("starts listening and updates isListening state", () => {
    const { result } = renderHook(() => useSpeechRecognition())

    act(() => {
      result.current.startListening()
    })

    expect(mockRecognitionInstance.start).toHaveBeenCalledTimes(1)
    expect(result.current.isListening).toBe(true)
  })

  it("stops listening when stopListening is called", () => {
    const { result } = renderHook(() => useSpeechRecognition())

    act(() => {
      result.current.startListening()
    })
    expect(result.current.isListening).toBe(true)

    act(() => {
      result.current.stopListening()
    })
    expect(mockRecognitionInstance.stop).toHaveBeenCalledTimes(1)
    expect(result.current.isListening).toBe(false)
  })

  it("handles speech recognition results and triggers onResult callback", () => {
    const onResult = vi.fn()
    const { result } = renderHook(() => useSpeechRecognition({ onResult }))

    act(() => {
      result.current.startListening()
    })

    act(() => {
      const mockResultEvent = {
        results: [
          {
            isFinal: true,
            0: { transcript: "Hello world" },
            length: 1,
          },
        ],
      }
      mockRecognitionInstance.onresult?.(mockResultEvent)
    })

    expect(result.current.transcript).toBe("Hello world")
    expect(onResult).toHaveBeenCalledWith("Hello world", true)
  })

  it("handles speech recognition errors and triggers onError callback", () => {
    const onError = vi.fn()
    const { result } = renderHook(() => useSpeechRecognition({ onError }))

    act(() => {
      result.current.startListening()
    })

    act(() => {
      mockRecognitionInstance.onerror?.({ error: "not-allowed" })
    })

    expect(result.current.error).toContain("Microphone permission denied")
    expect(result.current.isListening).toBe(false)
    expect(onError).toHaveBeenCalledWith(expect.stringContaining("Microphone permission denied"))
  })
})
