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

export interface LLMResponseOptions {
  skillName: string;
  tabName: string;
  lastAgentMessage: string;
  chatHistory: string[];
  options?: string[];
}

export async function generateLLMResponse(context: LLMResponseOptions): Promise<string> {
  console.log(`[E2E LLM Responder] Invoked for skill: "${context.skillName}" in tab: "${context.tabName}"`);

  const apiKey = process.env.GEMINI_API_KEY || process.env.NEXT_PUBLIC_GEMINI_API_KEY;
  if (apiKey) {
    try {
      const apiResponse = await callGeminiAPI(apiKey, context);
      console.log(`[E2E LLM Responder] Gemini API returned response: "${apiResponse}"`);
      return apiResponse;
    } catch (e) {
      console.warn('Gemini API call failed in E2E LLM responder, falling back to heuristic generator:', e);
    }
  }

  const heuristicResponse = generateHeuristicResponse(context);
  console.log(`[E2E LLM Responder] Heuristic responder generated response: "${heuristicResponse}"`);
  return heuristicResponse;
}

async function callGeminiAPI(apiKey: string, context: LLMResponseOptions): Promise<string> {
  const endpoint = `https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent?key=${apiKey}`;

  const promptText = `
You are acting as a software engineer user participating in an interactive E2E skill workflow in Agentic Foundry.
Current Tab: ${context.tabName}
Active Skill: ${context.skillName}

INSTRUCTIONS FOR GENERATING YOUR RESPONSE:
1. If the input text or available choices list options and one of the options is labeled or indicated as "Recommended" (e.g. "(Recommended) ...", "[Recommended] ...", or "Recommended: ..."), select and respond with that recommended option.
2. If the input text or available choices list options that include "[C]", "Continue", or "Proceed", select and respond with "Continue" or "Proceed".
3. For open-ended questions, provide a clear, concise, and helpful 1-2 sentence user response to progress the workflow.
4. Keep all responses direct, concise, and focused on progressing the workflow. Do NOT use code blocks or extra markdown wrappers.

Latest Agent Output:
"${context.lastAgentMessage}"

${context.options && context.options.length > 0 ? `Available Options: ${JSON.stringify(context.options)}` : ''}
`.trim();

  const body = {
    contents: [
      {
        parts: [{ text: promptText }],
      },
    ],
  };

  const response = await fetch(endpoint, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });

  if (!response.ok) {
    const errorText = await response.text();
    throw new Error(`Gemini API returned status ${response.status}: ${errorText}`);
  }

  const data = await response.json();
  const text = data?.candidates?.[0]?.content?.parts?.[0]?.text?.trim();

  if (text) {
    return text;
  }
  throw new Error('Gemini API returned empty response content');
}

export function generateHeuristicResponse(context: LLMResponseOptions): string {
  if (context.options && context.options.length > 0) {
    const rec = context.options.find((o) => o.toLowerCase().includes('recommended'));
    if (rec) return rec;
    const cont = context.options.find((o) => o.toLowerCase().includes('continue') || o.toLowerCase().includes('proceed') || o.includes('[C]'));
    if (cont) return cont.toLowerCase().includes('proceed') ? 'Proceed' : 'Continue';
    return context.options[0];
  }

  return 'Continue';
}
