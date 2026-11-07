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

export interface FunctionResponse {
  name: string;
  id: string;
  response: Record<string, any>;
}

export interface PhaseSwitchSuggestion {
  targetPhase: string;       // e.g., "ONBOARDING" | "ANALYSIS" | "PLANNING" | "SOLUTIONING" | "IMPLEMENTATION"
  targetPhaseLabel?: string;
  skillId?: string;
  reason?: string;
  suggestedPrompt?: string;
}

export interface Message {
  id?: string;
  sender: string;
  text: string;
  prompt?: string;
  displayText?: string;
  displayPrompt?: string;
  isUser: boolean;
  actions?: string[];
  phaseSwitchSuggestion?: PhaseSwitchSuggestion;
  rawContent?: string;
  phase?: string;
  timestamp?: string;
  isNotification?: boolean;
  isTransient?: boolean;
  notificationDetails?: Record<string, any>;
  isHistorical?: boolean;
  awaitingInput?: boolean;
  event?: any;
  functionResponse?: FunctionResponse;
  thinkingSteps?: Message[];
  thinkingText?: string;
  thinkingDurationSec?: number;
  artifacts?: Artifact[];
  artifactIds?: string[];
}

export type ArtifactStatus = 'Not Started' | 'In Progress' | 'Staged for Review' | 'Complete';

export interface Artifact {
  id: string;
  projectId?: string;
  workflowId?: string;
  title?: string;
  path?: string;
  phase?: string;
  status?: ArtifactStatus;
  owner?: string;
  content?: string;
  required?: boolean;
  isCustom?: boolean;
  artifactType?: string;
  createdAt?: string;
  updatedAt?: string;
  sessionId?: string;
  name?: string;
  mimeType?: string;
  size?: number;
  storageType?: string;
  storagePath?: string;
}

export interface PhaseAction {
  id: string;
  name: string;
  prompt: string;
  displayPrompt?: string;
}

export interface ArtifactTemplate {
  id: string;
  title: string;
  required: boolean;
}

export interface FeatureFlag<T = Record<string, any>> {
  id: string;
  enabled: boolean;
  properties: T;
  updatedAt?: string;
}

export class Skill {
  displayName: string;
  name: string;
  description: string;

  constructor(init: Partial<Skill> | string) {
    if (typeof init === 'string') {
      this.name = init;
      this.displayName = init;
      this.description = '';
    } else {
      this.displayName = init.displayName || init.name || '';
      this.name = init.name || '';
      this.description = init.description || '';
    }
  }
}

export class Tool {
  displayName: string;
  name: string;
  description: string;

  constructor(init: Partial<Tool> | string) {
    if (typeof init === 'string') {
      this.name = init;
      this.displayName = init;
      this.description = '';
    } else {
      this.displayName = init.displayName || init.name || '';
      this.name = init.name || '';
      this.description = init.description || '';
    }
  }
}

export class Node {
  id: string;
  name: string;
  displayName?: string;
  input?: Record<string, any>;
  output?: Record<string, any>;

  constructor(init: Partial<Node>) {
    this.id = init.id || '';
    this.name = init.name || '';
    this.displayName = init.displayName;
    this.input = init.input;
    this.output = init.output;
  }
}

export class AdkNode extends Node {
  type: 'agent' | 'function' | 'join' | 'custom' | 'trigger' | 'loop' | 'noop' | 'waypoint' | 'manual_trigger' | 'form_trigger';
  description?: string;
  rerunOnResume?: boolean;
  waitForOutput?: boolean;
  timeout?: number;
  retryConfig?: {
    maxAttempts?: number;
    initialDelay?: number;
    maxDelay?: number;
    backoffFactor?: number;
    jitter?: number;
    exceptions?: string[];
  };
  // Promoted type-specific fields as optional in base class for backward compatibility
  model?: string;
  instruction?: string;
  parameterBinding?: 'state' | 'node_input';
  authConfig?: Record<string, any>;
  parallelWorker?: boolean;
  skills?: Skill[];
  tools?: Tool[];
  agentId?: string;

  // Trigger specific (optional in base)
  triggerInterval?: 'seconds' | 'minutes' | 'hours' | 'days' | 'weeks' | 'months' | 'custom';
  intervalValue?: number;
  triggerAtMinute?: number;
  triggerAtHour?: number;
  triggerOnWeekdays?: number[];
  triggerAtDayOfMonth?: number;
  cronExpression?: string;

  // Loop specific (optional in base)
  exitCondition?: Record<string, any>;

  constructor(init: Partial<AdkNode>) {
    super(init);
    this.type = init.type || 'custom';
    this.description = init.description;
    this.rerunOnResume = init.rerunOnResume;
    this.waitForOutput = init.waitForOutput;
    this.timeout = init.timeout;
    this.retryConfig = init.retryConfig;
    this.model = init.model;
    this.instruction = init.instruction;
    this.parameterBinding = init.parameterBinding;
    this.authConfig = init.authConfig;
    this.parallelWorker = init.parallelWorker;
    this.skills = init.skills ? init.skills.map(s => new Skill(s)) : undefined;
    this.tools = init.tools ? init.tools.map(t => new Tool(t)) : undefined;
    this.agentId = init.agentId;
    this.triggerInterval = init.triggerInterval;
    this.intervalValue = init.intervalValue;
    this.triggerAtMinute = init.triggerAtMinute;
    this.triggerAtHour = init.triggerAtHour;
    this.triggerOnWeekdays = init.triggerOnWeekdays;
    this.triggerAtDayOfMonth = init.triggerAtDayOfMonth;
    this.cronExpression = init.cronExpression;
    this.exitCondition = init.exitCondition;
  }
}

export class Agent extends AdkNode {
  model: string;
  instruction: string;
  skills: Skill[];
  tools: Tool[];
  agentId?: string;

  constructor(init: Partial<Agent>) {
    super(init);
    this.type = 'agent';
    this.model = init.model || 'gemini-2.5-flash';
    this.instruction = init.instruction || '';
    this.skills = init.skills ? init.skills.map(s => new Skill(s)) : [];
    this.tools = init.tools ? init.tools.map(t => new Tool(t)) : [];
    this.agentId = init.agentId;
  }
}

export class FunctionNode extends AdkNode {
  parameterBinding: 'state' | 'node_input';
  authConfig: Record<string, any>;

  constructor(init: Partial<FunctionNode>) {
    super(init);
    this.type = 'function';
    this.parameterBinding = init.parameterBinding || 'state';
    this.authConfig = init.authConfig || {};
  }
}

export class JoinNode extends AdkNode {
  constructor(init: Partial<JoinNode>) {
    super(init);
    this.type = 'join';
    this.waitForOutput = init.waitForOutput ?? true; // JoinNode defaults to waitForOutput = true
  }
}

export class CustomNode extends AdkNode {
  parallelWorker: boolean;

  constructor(init: Partial<CustomNode>) {
    super(init);
    this.type = 'custom';
    this.parallelWorker = !!init.parallelWorker;
  }
}

export class LoopNode extends AdkNode {
  exitCondition: Record<string, any>;

  constructor(init: Partial<LoopNode>) {
    super(init);
    this.type = 'loop';
    this.exitCondition = init.exitCondition || {};
  }
}

export class ScheduledTrigger extends AdkNode {
  triggerInterval: 'seconds' | 'minutes' | 'hours' | 'days' | 'weeks' | 'months' | 'custom';
  intervalValue: number;
  triggerAtMinute: number;
  triggerAtHour: number;
  triggerOnWeekdays: number[];
  triggerAtDayOfMonth: number;
  cronExpression: string;

  constructor(init: Partial<ScheduledTrigger>) {
    super(init);
    this.type = 'trigger';
    this.triggerInterval = init.triggerInterval || 'custom';
    this.intervalValue = init.intervalValue ?? 1;
    this.triggerAtMinute = init.triggerAtMinute ?? 0;
    this.triggerAtHour = init.triggerAtHour ?? 9;
    this.triggerOnWeekdays = init.triggerOnWeekdays || [1];
    this.triggerAtDayOfMonth = init.triggerAtDayOfMonth ?? 1;
    this.cronExpression = init.cronExpression || '*/5 * * * *';
  }
}

export class ManualTrigger extends AdkNode {
  constructor(init: Partial<ManualTrigger>) {
    super(init);
    this.type = 'manual_trigger';
  }
}

export class FormTrigger extends AdkNode {
  pass_through?: boolean;

  constructor(init: Partial<FormTrigger>) {
    super(init);
    this.type = 'form_trigger';
    // FormTrigger must be rerun on resume to process the resumed inputs
    if (init.rerunOnResume === undefined) {
      this.rerunOnResume = true;
    }
    // FormTrigger in UI should always use passthrough
    this.pass_through = true;
  }
}

export interface AdkEdge {
  id: string;
  fromNode: string; // ID of source node or "START"
  toNode: string;
  route?: string | string[] | boolean | number;
}

export interface VisualNode {
  id: string;
  type: string;
  position: { x: number; y: number };
  orientation?: 'LR' | 'RL' | 'TB' | 'BT';
  data?: Record<string, any>;
}

export interface VisualEdge {
  id: string;
  source: string;
  target: string;
  sourceHandle?: string;
  targetHandle?: string;
  route?: any;
}

export interface VisualLayout {
  nodes: VisualNode[];
  edges: VisualEdge[];
}

export interface WorkflowTemplate {
  id: string;
  name: string;
  displayName?: string;
  description?: string;
  framework: string; // "BMAD" | "ADK"
  // Legacy phase-based fields (required to prevent breaking other views)
  phases: Record<string, PhaseAction[]>;
  artifacts: Record<string, ArtifactTemplate[]>;
  dependencies: Record<string, string[]>;
  // ADK 2.0 Graph-based fields (pure logical execution graph)
  nodes?: AdkNode[];
  edges?: AdkEdge[];
  // Separate visual layout of the designer canvas
  layout?: VisualLayout;
}

export interface IntegrationSettings {
  aiOrchestrator: string;
}

export interface ProjectIntegration {
  integration_id: string;
  enabled: boolean;
  settings: Record<string, any>;
}

export interface BmadConfig {
  specFolder: string;
  implementationArtifactsPath: string;
  planningArtifactsPath: string;
  testingArtifactsPath: string;
}

export interface Project {
  id: string;
  name: string;
  description: string;
  status: string;
  artifacts: Artifact[];
  icon: string;
  repoAccessToken: string;
  baseBranch: string;
  governanceGates: Record<string, string>;
  integrationSettings?: IntegrationSettings;
  app_integrations?: ProjectIntegration[];
  linkedWorkflows?: string[];
  isReadOnly?: boolean;
  createdAt?: string;
  visibility?: string;
  bmadConfig?: BmadConfig;
  specLocation?: string;
  planningArtifactsLocation?: string;
  implementationArtifactsLocation?: string;
  testingArtifactsLocation?: string;
}

export interface WorkspaceData {
  project: Project;
  workflows: WorkflowTemplate[];
  liveArtifacts: Artifact[];
}

export interface UserPreferences {
  theme: 'light' | 'dark';
}

export interface User {
  id: string;
  preferences: UserPreferences;
}

export interface CredentialAttribute {
  name: string;
  displayName: string;
  enabled: boolean;
  required?: boolean;
  helpText: string;
  isSecret: boolean;
  value: string;
}

export interface AppCredential {
  integrationId: string;
  fields: CredentialAttribute[];
  lastLogin?: string;
}

export interface UserProject {
  userId: string;
  projectId: string;
  appId: string;
  enabled: boolean;
  credentialId: string;
  isOverridden: boolean;
  fields: CredentialAttribute[];
  isReadOnly?: boolean;
}

export interface AppIntegration {
  id: string;
  name: string;
  description: string;
  authType: string;
  icon: string;
  fields: CredentialAttribute[];
  isEnabled: boolean;
  displayOrder: number;
  labels?: string[];
}

export interface Session {
  id: string;
  userId: string;
  projectId: string;
  phase: string;
  createdAt: string; // ISO String
  lastActiveAt: string; // ISO String
  status: string;
  title?: string;
}

export const FrameworkType = {
  bmad: "BMAD",
} as const;

/**
 * Evaluates if a phase satisfies dependency gating requirements.
 * A phase is satisfied if all of its required templates are completed/staged, or if it has no required templates.
 */
export function isPhaseSatisfied(
  workflow: WorkflowTemplate,
  phase: string,
  liveArtifacts: Artifact[] = []
): boolean {
  if (!workflow?.id || !workflow?.framework) return false;
  const artifacts = liveArtifacts || [];
  const templates = workflow.artifacts?.[phase] || [];
  const requiredTemplates = templates.filter((t) => t.required);
  if (requiredTemplates.length === 0) {
    return true;
  }
  return artifacts.some(
    (a) =>
      a.phase === phase &&
      (a.status === "Complete" || a.status === "Staged for Review")
  );
}

/**
 * Evaluates if a phase is complete for visual status tracking.
 * A phase is complete if any of its required artifacts are complete or staged for review.
 */
export function isPhaseComplete(
  workflow: WorkflowTemplate,
  phase: string,
  liveArtifacts: Artifact[] = []
): boolean {
  if (!workflow?.id || !workflow?.framework) return false;
  const artifacts = liveArtifacts || [];
  const templates = workflow.artifacts?.[phase] || [];
  const requiredTemplates = templates.filter((t) => t.required);

  if (requiredTemplates.length > 0) {
    return artifacts.some(
      (a) =>
        a.phase === phase &&
        (a.status === "Complete" || a.status === "Staged for Review")
    );
  } else {
    // For optional phases, evaluate progress only on active/started artifacts
    const activeArtifacts = artifacts.filter(
      (a) => a.phase === phase && a.status !== "Not Started"
    );
    if (activeArtifacts.length === 0) {
      return false; // Unstarted
    }
    return activeArtifacts.some(
      (a) => a.status === "Complete" || a.status === "Staged for Review"
    );
  }
}

/**
 * Evaluates if a phase is currently in progress for visual status tracking.
 */
export function isPhaseInProgress(
  workflow: WorkflowTemplate,
  phase: string,
  liveArtifacts: Artifact[] = []
): boolean {
  if (!workflow?.id || !workflow?.framework) return false;
  const artifacts = liveArtifacts || [];
  if (isPhaseComplete(workflow, phase, artifacts)) return false;
  return artifacts.some(
    (a) => a.phase === phase && a.status === "In Progress"
  );
}

/**
 * Helper to determine if an active artifact corresponds to an artifact template definition.
 */
export function determineArtifactType(filePath: string): string {
  const pathLower = (filePath || "").toLowerCase().replace(/\\/g, "/");
  const baseName = pathLower.split("/").pop() || "";

  if (pathLower.includes("project-context") || pathLower.includes("agents.md")) {
    return "project-context-doc";
  }
  if (pathLower.includes("project-status")) {
    return "project-status-doc";
  }
  if (
    pathLower.includes("reconcile-brain") ||
    pathLower.includes("/prd/") ||
    pathLower.startsWith("prd/")
  ) {
    return "product-req-doc";
  }
  if (pathLower.includes("brief")) {
    return "product-brief-doc";
  }
  if (pathLower.includes("brainstorm")) {
    return "brainstorming-session-doc";
  }
  if (pathLower.includes("research") || pathLower.includes("recon")) {
    return "research-doc";
  }
  if (pathLower.includes("prfaq")) {
    return "prfaq-doc";
  }
  if (pathLower.includes("prd")) {
    return "product-req-doc";
  }
  if (pathLower.includes("ux") || pathLower.includes("design")) {
    return "ux-design-doc";
  }
  if (
    pathLower.includes("architecture") ||
    pathLower.includes("/adr/") ||
    pathLower.includes("/adrs/") ||
    pathLower.startsWith("adr/") ||
    baseName.startsWith("adr-")
  ) {
    return "architecture-doc";
  }
  if (pathLower.includes("readiness")) {
    return "readiness-report-doc";
  }
  if (pathLower.includes("retro")) {
    return "retrospective-doc";
  }
  if (pathLower.includes("investigation") || baseName.includes("case-")) {
    return "investigation-doc";
  }
  if (
    pathLower.includes("test-summary") ||
    pathLower.includes("test-review") ||
    pathLower.includes("traceability") ||
    pathLower.includes("nfr-audit")
  ) {
    return "test-summary-doc";
  }
  if (
    pathLower.includes("epics-and-stories") ||
    pathLower.includes("epics_and_stories") ||
    baseName === "epics.md" ||
    (pathLower.includes("epic") && !pathLower.includes("story") && !pathLower.includes("stories") && !pathLower.includes("retro"))
  ) {
    return "epics-stories-doc";
  }
  if (
    pathLower.includes("stories") ||
    pathLower.includes("story") ||
    /^[0-9]+-[0-9]+-.*\.md$/.test(baseName)
  ) {
    return "story-doc";
  }
  if (pathLower.includes("sprint")) {
    return "sprint-plan-doc";
  }
  return "";
}

/**
 * Helper to determine if an active artifact corresponds to an artifact template definition.
 */
export function isArtifactRelatedToTemplate(
  artifact: Artifact,
  template: ArtifactTemplate
): boolean {
  if (!artifact || !template) return false;

  const artifactPathOrTitle = artifact.path || artifact.title || artifact.name || "";
  const determinedType = determineArtifactType(artifactPathOrTitle);
  const typeId = determinedType || artifact.artifactType;
  if (typeId) {
    return typeId === template.id;
  }

  if ((artifact.title && template.title && artifact.title === template.title) || (artifact.id && template.id && artifact.id === template.id)) return true;

  const pathLower = (artifact.path || "").toLowerCase();
  const pathParts = pathLower ? pathLower.split("/") : [];
  const normalizedTitle = template.title.toLowerCase().replace(/\s+/g, "_");
  const normalizedId = template.id.toLowerCase().replace(/-/g, "_");

  for (const part of pathParts) {
    if (part === normalizedTitle || part === normalizedId) return true;
    if (part.includes(normalizedTitle) || part.includes(normalizedId)) {
      return true;
    }
  }

  const titleLower = (artifact.title || artifact.name || "").toLowerCase().trim();
  if (
    titleLower &&
    ((normalizedTitle && titleLower.includes(normalizedTitle)) ||
      (template.title && template.title.trim() && titleLower.includes(template.title.toLowerCase().trim())))
  ) {
    return true;
  }

  return false;
}

export type PhaseStatus = 'complete' | 'in-progress' | 'optional-unstarted' | 'required-unstarted';

export function getPhaseStatus(
  workflow: WorkflowTemplate,
  phase: string,
  liveArtifacts: Artifact[] = []
): PhaseStatus {
  if (isPhaseComplete(workflow, phase, liveArtifacts)) {
    return 'complete';
  }
  if (isPhaseInProgress(workflow, phase, liveArtifacts)) {
    return 'in-progress';
  }
  const templates = workflow.artifacts?.[phase] || [];
  const hasRequirements = templates.some((t) => t.required);
  if (!hasRequirements) {
    return 'optional-unstarted';
  }
  return 'required-unstarted';
}

export type FeedbackType = 'bug' | 'feature_request' | 'general';

export interface FeedbackPayload {
  type: FeedbackType;
  subject: string;
  message: string;
  userEmail?: string;
  organizationId?: string;
  projectId?: string;
}

export interface FeedbackResponse {
  id: string;
  status: string;
  message: string;
}

