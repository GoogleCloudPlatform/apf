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

import type { Config } from "tailwindcss"

const config: Config = {
  darkMode: ["class"],
  content: [
    "./src/pages/**/*.{js,ts,jsx,tsx,mdx}",
    "./src/components/**/*.{js,ts,jsx,tsx,mdx}",
    "./src/app/**/*.{js,ts,jsx,tsx,mdx}",
  ],
  theme: {
    container: {
      center: true,
      padding: "2rem",
      screens: {
        "2xl": "1400px",
      },
    },
    extend: {
      spacing: {
        "4.5": "1.125rem",
      },
      colors: {
        border: "hsl(var(--border))",
        input: "hsl(var(--input))",
        ring: "hsl(var(--ring))",
        background: "hsl(var(--background))",
        foreground: "hsl(var(--foreground))",
        primary: {
          DEFAULT: "hsl(var(--primary))",
          foreground: "hsl(var(--primary-foreground))",
        },
        secondary: {
          DEFAULT: "hsl(var(--secondary))",
          foreground: "hsl(var(--secondary-foreground))",
        },
        destructive: {
          DEFAULT: "hsl(var(--destructive))",
          foreground: "hsl(var(--destructive-foreground))",
        },
        muted: {
          DEFAULT: "hsl(var(--muted))",
          foreground: "hsl(var(--muted-foreground))",
        },
        accent: {
          DEFAULT: "hsl(var(--accent))",
          foreground: "hsl(var(--accent-foreground))",
        },
        popover: {
          DEFAULT: "hsl(var(--popover))",
          foreground: "hsl(var(--popover-foreground))",
        },
        card: {
          DEFAULT: "hsl(var(--card))",
          foreground: "hsl(var(--card-foreground))",
        },
      },
      borderRadius: {
        lg: "var(--radius)",
        md: "calc(var(--radius) - 2px)",
        sm: "calc(var(--radius) - 4px)",
      },
      keyframes: {
        "accordion-down": {
          from: { height: "0" },
          to: { height: "var(--radix-accordion-content-height)" },
        },
        "accordion-up": {
          from: { height: "var(--radix-accordion-content-height)" },
          to: { height: "0" },
        },
        "neon-glow": {
          "0%": {
            borderColor: "hsl(var(--input))",
            boxShadow: "0 0 0 0 transparent",
          },
          "40%": {
            borderColor: "hsl(var(--primary) / 0.85)",
            boxShadow: "0 0 0 1px hsl(var(--primary) / 0.6), 0 0 12px 1px hsl(var(--primary) / 0.35), 0 0 24px 2px hsl(var(--primary) / 0.15)",
          },
          "60%": {
            borderColor: "hsl(var(--primary))",
            boxShadow: "0 0 0 1.5px hsl(var(--primary)), 0 0 16px 2px hsl(var(--primary) / 0.45), 0 0 30px 3px hsl(var(--primary) / 0.2)",
          },
          "100%": {
            borderColor: "hsl(var(--input))",
            boxShadow: "0 0 0 0 transparent",
          },
        },
        "border-snake": {
          "0%": {
            strokeDashoffset: "100",
            opacity: "0",
          },
          "4%": {
            opacity: "0.75",
          },
          "50%": {
            opacity: "0.85",
          },
          "100%": {
            strokeDashoffset: "-22",
            opacity: "0",
          },
        },
      },
      animation: {
        "accordion-down": "accordion-down 0.2s ease-out",
        "accordion-up": "accordion-up 0.2s ease-out",
        "neon-glow-once": "neon-glow 1.4s cubic-bezier(0.4, 0, 0.2, 1) 3s 1",
        "neon-glow-once-4s": "neon-glow 2.8s cubic-bezier(0.4, 0, 0.2, 1) 4s 1",
        "border-snake-fast": "border-snake 1.2s cubic-bezier(0.4, 0, 0.2, 1) 3s 1 forwards",
        "border-snake-fast-4s": "border-snake 2.6s cubic-bezier(0.4, 0, 0.2, 1) 4s 1 forwards",
      },
    },
  },
  plugins: [require("tailwindcss-animate")],
}

export default config
