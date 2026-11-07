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

import React from "react";

interface SkeletonLoaderProps {
  mode: "list" | "button";
  count?: number;
}

export default function SkeletonLoader({ mode, count = 1 }: SkeletonLoaderProps) {
  if (mode === "button") {
    return (
      <div className="flex flex-wrap gap-1.5 py-1">
        <div className="inline-flex h-6 w-28 border border-border/40 bg-muted/30 rounded-full animate-pulse" />
      </div>
    );
  }

  const items = Array.from({ length: Math.max(1, count) });

  return (
    <div className="space-y-1.5 py-1 w-full">
      {items.map((_, i) => (
        <div
          key={i}
          className="flex items-center justify-between w-full h-8 px-2.5 rounded-md border border-border/40 bg-muted/20 gap-2 select-none"
        >
          <div className="flex items-center gap-2 min-w-0 flex-1">
            {/* Icon Block */}
            <div className="h-3.5 w-3.5 bg-muted/50 rounded shrink-0 animate-pulse" />
            {/* Text Line Block */}
            <div className="h-3 bg-muted/50 rounded flex-1 max-w-[120px] animate-pulse" />
            {/* Badge/Status Block */}
            <div className="h-4 w-12 bg-muted/50 rounded-sm shrink-0 animate-pulse" />
          </div>
        </div>
      ))}
    </div>
  );
}
