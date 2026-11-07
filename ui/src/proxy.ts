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

import { NextResponse } from "next/server"
import type { NextRequest } from "next/server"

export function proxy(request: NextRequest) {
  const backendUrl = process.env.API_BACKEND || "http://localhost:8081"
  const { pathname, search } = request.nextUrl

  // Proxy API requests to the dynamic runtime API_BACKEND
  if (pathname.startsWith("/api/v1/")) {
    const targetUrl = `${backendUrl}${pathname}${search}`
    return NextResponse.rewrite(new URL(targetUrl))
  }

  // Proxy standard health check to the dynamic runtime API_BACKEND
  if (pathname === "/health") {
    const targetUrl = `${backendUrl}/health`
    return NextResponse.rewrite(new URL(targetUrl))
  }

  // Proxy Agy health check to the dynamic runtime API_BACKEND
  if (pathname === "/agy/health") {
    const targetUrl = `${backendUrl}/agy/health`
    return NextResponse.rewrite(new URL(targetUrl))
  }

  return NextResponse.next()
}

export const config = {
  matcher: [
    "/api/v1/:path*",
    "/health",
    "/agy/health",
  ],
}
