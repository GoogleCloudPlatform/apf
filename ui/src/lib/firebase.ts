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

import { initializeApp, getApps, getApp, FirebaseApp } from "firebase/app"
import {
  getAuth,
  Auth,
  User,
  GoogleAuthProvider as FirebaseGoogleAuthProvider,
  signInWithEmailLink as firebaseSignInWithEmailLink,
  updatePassword as firebaseUpdatePassword,
  signInWithPopup as firebaseSignInWithPopup,
  AuthProvider,
  connectAuthEmulator,
} from "firebase/auth"

// Define environment variables
const FIREBASE_API_KEY = process.env.NEXT_PUBLIC_FIREBASE_API_KEY
const FIREBASE_AUTH_DOMAIN = process.env.NEXT_PUBLIC_FIREBASE_AUTH_DOMAIN
const FIREBASE_PROJECT_ID = process.env.NEXT_PUBLIC_FIREBASE_PROJECT_ID

// Robust Mock Mode detection:
// 1. Explicitly enabled via NEXT_PUBLIC_ENABLE_MOCK_AUTH === "true"
// 2. Or in development mode when Firebase configuration is missing.
// Never silently fall back in production.
const isMockMode =
  process.env.NEXT_PUBLIC_ENABLE_MOCK_AUTH === "true" ||
  ((process.env.NODE_ENV === "development" || process.env.NODE_ENV === "test") &&
    (!FIREBASE_API_KEY || !FIREBASE_AUTH_DOMAIN || !FIREBASE_PROJECT_ID))

// Define Mock interfaces to match Firebase SDK shapes
export interface MockUser {
  email: string | null
  tenantId: string | null
  getIdToken: () => Promise<string>
}

export interface MockAuth {
  currentUser: MockUser | null
  tenantId: string | null
  mockEmail?: string | null // Pluggable mock email for invitation reconciliation
}

// Global instances
let firebaseApp: FirebaseApp | null = null
let realAuth: Auth | null = null

// Initialize real Firebase if not in mock mode and running in browser
if (!isMockMode && typeof window !== "undefined") {
  try {
    if (!FIREBASE_API_KEY || !FIREBASE_PROJECT_ID) {
      throw new Error("Firebase API Key or Project ID is missing from environment.")
    }
    const firebaseConfig = {
      apiKey: FIREBASE_API_KEY,
      authDomain: FIREBASE_AUTH_DOMAIN,
      projectId: FIREBASE_PROJECT_ID,
    }
    
    if (getApps().length === 0) {
      firebaseApp = initializeApp(firebaseConfig)
    } else {
      firebaseApp = getApp()
    }
    realAuth = getAuth(firebaseApp)
    
    const emulatorHost = process.env.NEXT_PUBLIC_FIREBASE_AUTH_EMULATOR_HOST
    if (realAuth && emulatorHost) {
      console.log("[Firebase Client] Connecting Auth SDK to local emulator:", emulatorHost)
      connectAuthEmulator(realAuth, `http://${emulatorHost}`, { disableWarnings: true })
    }
  } catch (e) {
    console.warn("Firebase SDK initialization skipped (this is expected if using IAP instead of Firebase):", e)
  }
}

// Mock Auth Client State
class MockAuthClient implements MockAuth {
  private _tenantId: string | null = null
  private _currentUser: MockUser | null = null
  public mockEmail: string | null = null

  get tenantId() {
    return this._tenantId
  }

  set tenantId(val: string | null) {
    this._tenantId = val
  }

  get currentUser() {
    return this._currentUser
  }

  set currentUser(user: MockUser | null) {
    this._currentUser = user
  }

  // Helper to simulate logging in
  simulateLogin(email: string, tenantId: string) {
    this._currentUser = {
      email,
      tenantId,
      getIdToken: async () => `simulated-jwt-token-for-${email}`,
    }
  }
}

const mockAuthInstance = new MockAuthClient()

// Exported auth reference (can be real Auth or MockAuthClient)
export const auth: Auth | MockAuth = (isMockMode ? mockAuthInstance : realAuth) as any

export const isFirebaseMock = isMockMode

// Pluggable wrapper for signInWithEmailLink with strong typing
export async function signInWithEmailLink(
  authClient: Auth | MockAuth,
  email: string,
  emailLink: string
): Promise<{ user: User | MockUser }> {
  if (isMockMode) {
    console.log("[Mock Auth] signInWithEmailLink", email, emailLink)
    const client = authClient as MockAuthClient
    client.simulateLogin(email, client.tenantId || "mock-tenant-id")
    if (!client.currentUser) {
      throw new Error("Failed to create simulated mock user.")
    }
    return { user: client.currentUser }
  } else {
    if (!authClient) {
      throw new Error("Firebase Auth Client is not initialized. Please verify your configuration.")
    }
    return firebaseSignInWithEmailLink(authClient as Auth, email, emailLink)
  }
}

// Pluggable wrapper for updatePassword with strong typing
export async function updatePassword(user: User | MockUser, password: string): Promise<void> {
  if (isMockMode) {
    console.log("[Mock Auth] updatePassword executed successfully")
    return Promise.resolve()
  } else {
    if (!user) {
      throw new Error("No authenticated user session to update password.")
    }
    return firebaseUpdatePassword(user as User, password)
  }
}

// Pluggable wrapper for signInWithPopup (Google Auth) with strong typing
export async function signInWithPopup(
  authClient: Auth | MockAuth,
  provider: AuthProvider | any
): Promise<{ user: User | MockUser }> {
  if (isMockMode) {
    console.log("[Mock Auth] signInWithPopup with Google")
    const client = authClient as MockAuthClient
    // Use the seeded mockEmail if available, otherwise generate a random one
    const simulatedEmail = client.mockEmail || `google-user-${Math.floor(Math.random() * 1000)}@example.com`
    client.simulateLogin(simulatedEmail, client.tenantId || "mock-tenant-id")
    if (!client.currentUser) {
      throw new Error("Failed to create simulated Google mock user.")
    }
    return { user: client.currentUser }
  } else {
    if (!authClient) {
      throw new Error("Firebase Auth Client is not initialized. Please verify your configuration.")
    }
    return firebaseSignInWithPopup(authClient as Auth, provider)
  }
}

// Mock/Real GoogleAuthProvider
export const GoogleAuthProvider = isMockMode
  ? class MockGoogleAuthProvider {
      static PROVIDER_ID = "google.com"
    }
  : FirebaseGoogleAuthProvider
