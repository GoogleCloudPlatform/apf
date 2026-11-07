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
  ZoomIn,
  ZoomOut,
  RotateCcw,
  AlertTriangle,
  RefreshCw,
  Loader2,
} from "lucide-react"
import { cn } from "@/lib/utils"
import { api } from "@/lib/api"

export interface ImageViewerProps {
  src: string
  alt?: string
  onDimensionsLoaded?: (dimensions: { width: number; height: number } | null) => void
  className?: string
  onRetry?: () => void
}

const MIN_ZOOM = 0.25 // 25%
const MAX_ZOOM = 5.0  // 500%
const ZOOM_STEP = 0.25 // ±25%

export function ImageViewer({
  src,
  alt = "Image preview",
  onDimensionsLoaded,
  className,
  onRetry,
}: ImageViewerProps) {
  const [zoom, setZoom] = useState(1.0)
  const [pan, setPan] = useState({ x: 0, y: 0 })
  const [isDragging, setIsDragging] = useState(false)
  const [dragStart, setDragStart] = useState({ x: 0, y: 0 })
  const [isLoading, setIsLoading] = useState(true)
  const [hasError, setHasError] = useState(false)
  const [retryKey, setRetryKey] = useState(0)
  const [displaySrc, setDisplaySrc] = useState<string>("")

  const containerRef = useRef<HTMLDivElement>(null)
  const activeBlobUrlRef = useRef<string | null>(null)

  // Fetch image content with authentication headers if src is an API path, or use directly if blob/data
  useEffect(() => {
    let isMounted = true

    if (activeBlobUrlRef.current) {
      URL.revokeObjectURL(activeBlobUrlRef.current)
      activeBlobUrlRef.current = null
    }

    setIsLoading(true)
    setHasError(false)
    setZoom(1.0)
    setPan({ x: 0, y: 0 })

    if (!src) {
      setDisplaySrc("")
      setIsLoading(false)
      return
    }

    if (src.startsWith("blob:") || src.startsWith("data:")) {
      setDisplaySrc(src)
      return
    }

    // Authenticated API request
    api
      .get(src)
      .then(async (res) => {
        if (!res.ok) {
          throw new Error(`Failed to load image: HTTP status ${res.status}`)
        }
        const blob = await res.blob()
        if (!isMounted) return
        const objUrl = URL.createObjectURL(blob)
        activeBlobUrlRef.current = objUrl
        setDisplaySrc(objUrl)
      })
      .catch((err) => {
        if (!isMounted) return
        console.error("ImageViewer failed to fetch image:", err)
        setIsLoading(false)
        setHasError(true)
        onDimensionsLoaded?.(null)
      })

    return () => {
      isMounted = false
      if (activeBlobUrlRef.current) {
        URL.revokeObjectURL(activeBlobUrlRef.current)
        activeBlobUrlRef.current = null
      }
    }
  }, [src, retryKey])

  const handleZoomIn = () => {
    setZoom((prev) => Math.min(MAX_ZOOM, Math.round((prev + ZOOM_STEP) * 100) / 100))
  }

  const handleZoomOut = () => {
    setZoom((prev) => Math.max(MIN_ZOOM, Math.round((prev - ZOOM_STEP) * 100) / 100))
  }

  const handleResetZoom = () => {
    setZoom(1.0)
    setPan({ x: 0, y: 0 })
  }

  const handleMouseDown = (e: React.MouseEvent) => {
    // Only start drag on primary button
    if (e.button !== 0) return
    e.preventDefault()
    setIsDragging(true)
    setDragStart({
      x: e.clientX - pan.x,
      y: e.clientY - pan.y,
    })
  }

  // Handle global mouse movement and release while dragging
  useEffect(() => {
    if (!isDragging) return

    const handleGlobalMouseMove = (e: MouseEvent) => {
      setPan({
        x: e.clientX - dragStart.x,
        y: e.clientY - dragStart.y,
      })
    }

    const handleGlobalMouseUp = () => {
      setIsDragging(false)
    }

    window.addEventListener("mousemove", handleGlobalMouseMove)
    window.addEventListener("mouseup", handleGlobalMouseUp)

    return () => {
      window.removeEventListener("mousemove", handleGlobalMouseMove)
      window.removeEventListener("mouseup", handleGlobalMouseUp)
    }
  }, [isDragging, dragStart])

  const handleWheel = (e: React.WheelEvent) => {
    if (e.deltaY < 0) {
      handleZoomIn()
    } else if (e.deltaY > 0) {
      handleZoomOut()
    }
  }

  const handleImageLoad = (e: React.SyntheticEvent<HTMLImageElement>) => {
    const img = e.currentTarget
    const dims = {
      width: img.naturalWidth,
      height: img.naturalHeight,
    }
    setIsLoading(false)
    setHasError(false)
    onDimensionsLoaded?.(dims)
  }

  const handleImageError = () => {
    setIsLoading(false)
    setHasError(true)
    onDimensionsLoaded?.(null)
  }

  const handleRetry = () => {
    setHasError(false)
    setIsLoading(true)
    setRetryKey((prev) => prev + 1)
    onRetry?.()
  }

  return (
    <div
      ref={containerRef}
      data-testid="image-viewer-container"
      className={cn(
        "relative flex h-full w-full select-none items-center justify-center overflow-hidden bg-slate-100 dark:bg-slate-900",
        className
      )}
      style={{
        backgroundImage: `
          linear-gradient(45deg, rgba(0, 0, 0, 0.05) 25%, transparent 25%),
          linear-gradient(-45deg, rgba(0, 0, 0, 0.05) 25%, transparent 25%),
          linear-gradient(45deg, transparent 75%, rgba(0, 0, 0, 0.05) 75%),
          linear-gradient(-45deg, transparent 75%, rgba(0, 0, 0, 0.05) 75%)
        `,
        backgroundSize: "20px 20px",
        backgroundPosition: "0 0, 0 10px, 10px -10px, -10px 0px",
      }}
      onMouseDown={handleMouseDown}
      onWheel={handleWheel}
    >
      {/* Loading Spinner */}
      {isLoading && !hasError && (
        <div
          data-testid="image-viewer-loading"
          className="absolute inset-0 z-20 flex flex-col items-center justify-center gap-2 bg-background/50 backdrop-blur-xs text-muted-foreground"
        >
          <Loader2 className="h-6 w-6 animate-spin text-primary" />
          <span className="text-xs font-medium">Loading image...</span>
        </div>
      )}

      {/* Error State */}
      {hasError ? (
        <div
          data-testid="image-viewer-error"
          className="z-20 flex max-w-sm flex-col items-center justify-center gap-3 rounded-xl border border-destructive/20 bg-destructive/5 p-6 text-center text-destructive select-none"
        >
          <AlertTriangle className="h-8 w-8 opacity-70" />
          <div>
            <h3 className="text-sm font-bold text-foreground">Failed to load image</h3>
            <p className="mt-1 text-xs text-muted-foreground leading-normal">
              Could not display the image preview. The file may be missing or corrupted.
            </p>
          </div>
          <button
            type="button"
            data-testid="image-retry-btn"
            onClick={handleRetry}
            className="inline-flex h-8 items-center gap-1.5 rounded-lg border border-destructive px-3 text-xs font-bold text-destructive hover:bg-destructive/10 transition-colors"
          >
            <RefreshCw className="h-3.5 w-3.5" />
            Retry
          </button>
        </div>
      ) : (
        /* Image Transform Canvas */
        <div
          className={cn(
            "flex h-full w-full items-center justify-center p-4",
            isDragging ? "cursor-grabbing" : "cursor-grab"
          )}
        >
          <div
            style={{
              transform: `translate(${pan.x}px, ${pan.y}px) scale(${zoom})`,
              transformOrigin: "center center",
              transition: isDragging ? "none" : "transform 100ms ease-out",
            }}
            className="flex items-center justify-center max-h-full max-w-full"
          >
            <img
              key={`${displaySrc || src}-${retryKey}`}
              src={displaySrc || src}
              alt={alt}
              data-testid="image-viewer-img"
              onLoad={handleImageLoad}
              onError={handleImageError}
              className="max-h-[calc(100vh-200px)] max-w-full rounded-md object-contain shadow-sm pointer-events-none"
              draggable={false}
            />
          </div>
        </div>
      )}

      {/* Floating Zoom & Pan Controls Toolbar */}
      {!hasError && (
        <div
          data-testid="image-viewer-controls"
          className="absolute bottom-4 left-1/2 z-30 flex -translate-x-1/2 items-center gap-1 rounded-full border border-border bg-background/90 px-3 py-1.5 shadow-lg backdrop-blur-md"
        >
          <button
            type="button"
            data-testid="zoom-out-btn"
            title="Zoom out"
            aria-label="Zoom out"
            disabled={zoom <= MIN_ZOOM}
            onClick={(e) => {
              e.stopPropagation()
              handleZoomOut()
            }}
            className="flex h-7 w-7 items-center justify-center rounded-full text-muted-foreground hover:bg-muted hover:text-foreground transition-colors disabled:opacity-40"
          >
            <ZoomOut className="h-3.5 w-3.5" />
          </button>

          <span
            data-testid="zoom-level"
            className="min-w-[48px] text-center font-mono text-xs font-semibold text-foreground select-none"
          >
            {Math.round(zoom * 100)}%
          </span>

          <button
            type="button"
            data-testid="zoom-in-btn"
            title="Zoom in"
            aria-label="Zoom in"
            disabled={zoom >= MAX_ZOOM}
            onClick={(e) => {
              e.stopPropagation()
              handleZoomIn()
            }}
            className="flex h-7 w-7 items-center justify-center rounded-full text-muted-foreground hover:bg-muted hover:text-foreground transition-colors disabled:opacity-40"
          >
            <ZoomIn className="h-3.5 w-3.5" />
          </button>

          <div className="mx-1 h-3.5 w-[1px] bg-border" />

          <button
            type="button"
            data-testid="reset-zoom-btn"
            title="Reset zoom"
            aria-label="Reset zoom"
            onClick={(e) => {
              e.stopPropagation()
              handleResetZoom()
            }}
            className="flex h-7 w-7 items-center justify-center rounded-full text-muted-foreground hover:bg-muted hover:text-foreground transition-colors"
          >
            <RotateCcw className="h-3.5 w-3.5" />
          </button>
        </div>
      )}
    </div>
  )
}

export default ImageViewer
