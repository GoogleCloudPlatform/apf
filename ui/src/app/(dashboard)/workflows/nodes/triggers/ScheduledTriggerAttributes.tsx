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

import { useState } from "react"
import { KeyValueEditor } from "../../components/KeyValueEditor"

interface ScheduledTriggerAttributesProps {
  data: {
    triggerInterval?: 'seconds' | 'minutes' | 'hours' | 'days' | 'weeks' | 'months' | 'custom'
    intervalValue?: number
    triggerAtMinute?: number
    triggerAtHour?: number
    triggerOnWeekdays?: number[]
    triggerAtDayOfMonth?: number
    cronExpression?: string
    input?: Record<string, any>
    output?: Record<string, any>
    [key: string]: any
  }
  onChange: (field: string, val: any) => void
}

const WEEKDAYS = [
  { label: "Su", value: 0 },
  { label: "Mo", value: 1 },
  { label: "Tu", value: 2 },
  { label: "We", value: 3 },
  { label: "Th", value: 4 },
  { label: "Fr", value: 5 },
  { label: "Sa", value: 6 },
]

export const ScheduledTriggerAttributes = ({ data, onChange }: ScheduledTriggerAttributesProps) => {
  const [activeTab, setActiveTab] = useState<"general" | "input_output">("general")

  const triggerInterval = data.triggerInterval || 'custom'
  const intervalValue = data.intervalValue ?? 1
  const triggerAtMinute = data.triggerAtMinute ?? 0
  const triggerAtHour = data.triggerAtHour ?? 9
  const triggerOnWeekdays = data.triggerOnWeekdays || [1] // Default to Monday
  const triggerAtDayOfMonth = data.triggerAtDayOfMonth ?? 1
  const cronExpression = data.cronExpression || '*/5 * * * *'

  // Helper to toggle weekday selection
  const handleWeekdayToggle = (day: number) => {
    let updated: number[]
    if (triggerOnWeekdays.includes(day)) {
      updated = triggerOnWeekdays.filter(d => d !== day)
    } else {
      updated = [...triggerOnWeekdays, day].sort()
    }
    // Ensure at least one day is selected
    if (updated.length === 0) updated = [day]
    onChange("triggerOnWeekdays", updated)
  }

  // Generate readable label for Interval Value
  const getIntervalValueLabel = () => {
    switch (triggerInterval) {
      case 'seconds': return "Seconds Between Triggers"
      case 'minutes': return "Minutes Between Triggers"
      case 'hours': return "Hours Between Triggers"
      case 'days': return "Days Between Triggers"
      case 'weeks': return "Weeks Between Triggers"
      case 'months': return "Months Between Triggers"
      default: return "Interval Value"
    }
  }

  return (
    <div className="space-y-4">
      {/* Tab Navigation Headers */}
      <div className="flex border-b border-border/60 shrink-0">
        <button
          type="button"
          onClick={() => setActiveTab("general")}
          className={`flex-1 pb-2 text-[10px] font-bold uppercase tracking-wider border-b-2 transition-all ${
            activeTab === "general"
              ? "border-primary text-primary"
              : "border-transparent text-muted-foreground hover:text-foreground"
          }`}
        >
          General
        </button>
        <button
          type="button"
          onClick={() => setActiveTab("input_output")}
          className={`flex-1 pb-2 text-[10px] font-bold uppercase tracking-wider border-b-2 transition-all ${
            activeTab === "input_output"
              ? "border-primary text-primary"
              : "border-transparent text-muted-foreground hover:text-foreground"
          }`}
        >
          Input & Output
        </button>
      </div>

      {/* Tab Contents */}
      {activeTab === "general" && (
        <div className="space-y-4">
          {/* 1. Trigger Interval Selector */}
          <div className="space-y-1.5">
            <label className="text-[10px] font-bold text-muted-foreground uppercase tracking-wide">
              Trigger Interval
            </label>
            <select
              value={triggerInterval}
              onChange={(e) => {
                const val = e.target.value as any
                onChange("triggerInterval", val)
                // Apply sensible defaults when switching intervals if not set
                if (val !== 'custom' && !data.intervalValue) onChange("intervalValue", 1)
              }}
              className="flex h-9 w-full rounded-lg border border-input bg-card px-3 py-1 text-xs shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-primary"
            >
              <option value="seconds">Seconds</option>
              <option value="minutes">Minutes</option>
              <option value="hours">Hours</option>
              <option value="days">Days</option>
              <option value="weeks">Weeks</option>
              <option value="months">Months</option>
              <option value="custom">Custom (Cron Expression)</option>
            </select>
          </div>

          {/* 2. Interval Value (for all non-custom intervals) */}
          {triggerInterval !== 'custom' && (
            <div className="space-y-1.5">
              <label className="text-[10px] font-bold text-muted-foreground uppercase tracking-wide">
                {getIntervalValueLabel()}
              </label>
              <input
                type="number"
                min={1}
                value={intervalValue}
                onChange={(e) => onChange("intervalValue", Math.max(1, Number(e.target.value)))}
                className="flex h-9 w-full rounded-lg border border-input bg-transparent px-3 py-1 text-xs shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-primary"
              />
            </div>
          )}

          {/* 3. Trigger At Hour (for Daily, Weekly, Monthly) */}
          {['days', 'weeks', 'months'].includes(triggerInterval) && (
            <div className="space-y-1.5">
              <label className="text-[10px] font-bold text-muted-foreground uppercase tracking-wide">
                Trigger at Hour
              </label>
              <select
                value={triggerAtHour}
                onChange={(e) => onChange("triggerAtHour", Number(e.target.value))}
                className="flex h-9 w-full rounded-lg border border-input bg-card px-3 py-1 text-xs shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-primary"
              >
                {Array.from({ length: 24 }).map((_, hour) => {
                  const ampm = hour >= 12 ? "PM" : "AM"
                  const displayHour = hour % 12 === 0 ? 12 : hour % 12
                  return (
                    <option key={hour} value={hour}>
                      {displayHour} {ampm} ({hour}:00)
                    </option>
                  )
                })}
              </select>
            </div>
          )}

          {/* 4. Trigger At Minute (for Hourly, Daily, Weekly, Monthly) */}
          {['hours', 'days', 'weeks', 'months'].includes(triggerInterval) && (
            <div className="space-y-1.5">
              <label className="text-[10px] font-bold text-muted-foreground uppercase tracking-wide">
                Trigger at Minute
              </label>
              <input
                type="number"
                min={0}
                max={59}
                value={triggerAtMinute}
                onChange={(e) => onChange("triggerAtMinute", Math.min(59, Math.max(0, Number(e.target.value))))}
                className="flex h-9 w-full rounded-lg border border-input bg-transparent px-3 py-1 text-xs shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-primary"
              />
              <p className="text-[8px] text-muted-foreground">Minute past the hour (0 - 59)</p>
            </div>
          )}

          {/* 5. Trigger on Weekdays (for Weekly only) */}
          {triggerInterval === 'weeks' && (
            <div className="space-y-1.5">
              <label className="text-[10px] font-bold text-muted-foreground uppercase tracking-wide block">
                Trigger on Weekdays
              </label>
              <div className="flex gap-1 pt-0.5">
                {WEEKDAYS.map((day) => {
                  const isSelected = triggerOnWeekdays.includes(day.value)
                  return (
                    <button
                      key={day.value}
                      type="button"
                      onClick={() => handleWeekdayToggle(day.value)}
                      className={`h-7 w-7 rounded-lg text-[10px] font-bold border transition-all flex items-center justify-center ${
                        isSelected
                          ? "bg-purple-600 border-purple-600 text-white shadow-sm"
                          : "bg-transparent border-input text-muted-foreground hover:bg-muted hover:text-foreground"
                      }`}
                    >
                      {day.label}
                    </button>
                  )
                })}
              </div>
            </div>
          )}

          {/* 6. Trigger at Day of Month (for Monthly only) */}
          {triggerInterval === 'months' && (
            <div className="space-y-1.5">
              <label className="text-[10px] font-bold text-muted-foreground uppercase tracking-wide">
                Trigger at Day of Month
              </label>
              <input
                type="number"
                min={1}
                max={31}
                value={triggerAtDayOfMonth}
                onChange={(e) => onChange("triggerAtDayOfMonth", Math.min(31, Math.max(1, Number(e.target.value))))}
                className="flex h-9 w-full rounded-lg border border-input bg-transparent px-3 py-1 text-xs shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-primary"
              />
              <p className="text-[8px] text-muted-foreground">Day of month (1 - 31). Runs only in months containing this day.</p>
            </div>
          )}

          {/* 7. Custom Cron Expression (for Custom only) */}
          {triggerInterval === 'custom' && (
            <div className="space-y-1.5">
              <label className="text-[10px] font-bold text-muted-foreground uppercase tracking-wide">
                Cron Expression
              </label>
              <input
                type="text"
                value={cronExpression}
                onChange={(e) => onChange("cronExpression", e.target.value)}
                placeholder="e.g. */5 * * * *"
                className="flex h-9 w-full rounded-lg border border-input bg-transparent px-3 py-1 text-xs shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-primary"
              />
              <p className="text-[8px] text-muted-foreground leading-normal">
                Standard crontab expression (5 or 6 fields). You can generate expressions using{" "}
                <a
                  href="https://crontab.guru"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-primary hover:underline font-semibold"
                >
                  crontab.guru
                </a>.
              </p>
            </div>
          )}
        </div>
      )}

      {activeTab === "input_output" && (
        <div className="space-y-6">
          <KeyValueEditor
            label="Input Data (Not Supported for Triggers)"
            values={data.input}
            onChange={(val) => onChange("input", val)}
            disabled
          />
          <div className="border-t border-border/50 my-4" />
          <KeyValueEditor
            label="Output Data"
            values={data.output}
            onChange={(val) => onChange("output", val)}
          />
        </div>
      )}
    </div>
  )
}
