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

import { useQuery } from "@tanstack/react-query"
import api from "@/lib/api"
import { FeatureFlag } from "@/lib/types"

export async function fetchFeature<T = Record<string, any>>(
  featureId: string
): Promise<FeatureFlag<T>> {
  if (!featureId) {
    return {
      id: "",
      enabled: false,
      properties: {} as T,
    }
  }
  const response = await api.get(`/api/v1/features/${encodeURIComponent(featureId)}`)
  if (!response.ok) {
    if (response.status === 404) {
      return {
        id: featureId,
        enabled: false,
        properties: {} as T,
      }
    }
    throw new Error(`Failed to fetch feature flag '${featureId}': ${response.statusText}`)
  }
  const data = await response.json()
  return (data ?? {
    id: featureId,
    enabled: false,
    properties: {} as T,
  }) as FeatureFlag<T>
}

export function useFeatureFlag<T = Record<string, any>>(
  featureId: string,
  defaultProperties?: T
) {
  const { data, isLoading, isError, error } = useQuery<FeatureFlag<T>>({
    queryKey: ["feature", featureId],
    queryFn: () => fetchFeature<T>(featureId),
    enabled: Boolean(featureId),
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
    retry: false,
    refetchOnWindowFocus: false,
  })

  if (isError || !data) {
    return {
      enabled: false,
      properties: defaultProperties ?? ({} as T),
      isLoading,
      isError,
      error,
    }
  }

  const safeProperties =
    data.properties && typeof data.properties === "object" ? data.properties : ({} as T)

  return {
    enabled: Boolean(data.enabled),
    properties: { ...(defaultProperties ?? {}), ...safeProperties },
    isLoading: false,
    isError: false,
    error: null,
  }
}

export const useFeature = useFeatureFlag

