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

import fs from 'fs';
import path from 'path';
import type { TestInfo } from '@playwright/test';

export interface RecordedMessage {
  index: number;
  timestamp: string;
  sender: 'user' | 'agent' | 'system';
  text: string;
  matchedFailures: string[];
}

export class FailureMonitor {
  private patterns: RegExp[] = [];
  private recordedMessages: RecordedMessage[] = [];
  private detectedFailures: RecordedMessage[] = [];

  constructor(configFilePath?: string) {
    const defaultPath = path.join(__dirname, '../config/failure-patterns.json');
    const filePath = configFilePath || defaultPath;
    this.loadPatterns(filePath);
  }

  private loadPatterns(filePath: string) {
    try {
      if (fs.existsSync(filePath)) {
        const raw = fs.readFileSync(filePath, 'utf-8');
        const patternStrings: string[] = JSON.parse(raw);
        this.patterns = patternStrings.map((p) => {
          // Parse inline flag like (?i) if present or default to 'i'
          let cleaned = p;
          let flags = 'i';
          if (p.startsWith('(?i)')) {
            cleaned = p.substring(4);
          }
          return new RegExp(cleaned, flags);
        });
      }
    } catch (e) {
      console.warn('Failed to load failure patterns file, using default fallback regexes:', e);
      this.patterns = [
        /internal server error/i,
        /failed to execute/i,
        /tool execution failed/i,
        /unhandled exception/i,
        /connection refused/i,
      ];
    }
  }

  public recordAndCheck(sender: 'user' | 'agent' | 'system', text: string): RecordedMessage {
    // Prevent duplicate recording of identical message content from same sender if called repeatedly in loop
    const last = this.recordedMessages[this.recordedMessages.length - 1];
    if (last && last.sender === sender && last.text === text) {
      return last;
    }

    const index = this.recordedMessages.length + 1;
    const timestamp = new Date().toISOString();
    const matchedFailures: string[] = [];

    for (const pattern of this.patterns) {
      if (pattern.test(text)) {
        matchedFailures.push(pattern.source);
      }
    }

    const recorded: RecordedMessage = {
      index,
      timestamp,
      sender,
      text,
      matchedFailures,
    };

    this.recordedMessages.push(recorded);
    if (matchedFailures.length > 0) {
      this.detectedFailures.push(recorded);
    }

    return recorded;
  }

  public getRecordedMessages(): RecordedMessage[] {
    return this.recordedMessages;
  }

  public getDetectedFailures(): RecordedMessage[] {
    return this.detectedFailures;
  }

  public enrichTestReport(testInfo: TestInfo, skillName: string, tabName: string) {
    const sanitize = (str: string) => str.toLowerCase().replace(/[^a-z0-9]/g, '-');
    const safeTab = sanitize(tabName);
    const safeSkill = sanitize(skillName);

    // 1. Attach Full Chat Transcript JSON
    testInfo.attach(`chat-transcript-${safeTab}-${safeSkill}.json`, {
      body: JSON.stringify(this.recordedMessages, null, 2),
      contentType: 'application/json',
    });

    // 2. Attach Detected Failures JSON
    testInfo.attach(`detected-failures-${safeTab}-${safeSkill}.json`, {
      body: JSON.stringify(this.detectedFailures, null, 2),
      contentType: 'application/json',
    });

    // 3. Attach Human-Readable Summary Report
    const summaryLines = [
      `=== SKILL E2E INTERACTION REPORT ===`,
      `Tab: ${tabName}`,
      `Quick Action / Skill: ${skillName}`,
      `Total Messages Exchanged: ${this.recordedMessages.length}`,
      `Total Failures Detected: ${this.detectedFailures.length}`,
      ``,
    ];

    if (this.detectedFailures.length > 0) {
      summaryLines.push(`--- DETECTED FAILURE MESSAGES ---`);
      this.detectedFailures.forEach((f) => {
        summaryLines.push(
          `[Step ${f.index}] [${f.sender.toUpperCase()}] [Matched Patterns: ${f.matchedFailures.join(', ')}]\nContent: ${f.text}\n`
        );
      });
    } else {
      summaryLines.push(`No failure patterns matched during skill execution.`);
    }

    testInfo.attach(`failure-report-${safeTab}-${safeSkill}.txt`, {
      body: summaryLines.join('\n'),
      contentType: 'text/plain',
    });
  }
}
