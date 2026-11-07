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

import React, { useState } from 'react';
import { Loader2 } from 'lucide-react';
import { GitChoiceModal, GitChoiceOption } from './GitChoiceModal';
import { ConflictCardResolver } from './ConflictCardResolver';

export interface GitSyncChecklistStep {
  id: string;
  label: string;
  status: 'completed' | 'in_progress' | 'failed' | 'skipped';
  detail?: string;
}

export interface DynamicBranchInfo {
  baseBranch: string;
  specBranch: string;
  backupRef?: string;
  shadowBranch?: string;
}

export interface ConflictResolutionInfo {
  conflictingFiles: string[];
  pullRequestUrl?: string;
  cliInstructions?: string;
  nextStepRecommendation: string;
  choices?: GitChoiceOption[];
}

export interface GitSyncChecklistResponse {
  status: 'SUCCESS' | 'DIRTY_WORKTREE_PROMPT' | 'CONFLICT_DETECTED' | 'ERROR';
  message: string;
  snapshotId?: string;
  dynamicBranches: DynamicBranchInfo;
  steps: GitSyncChecklistStep[];
  conflictInfo?: ConflictResolutionInfo;
  availableChoices?: GitChoiceOption[];
}

export interface GitSyncRunnerProps {
  data: GitSyncChecklistResponse | null;
  onSelectChoice?: (choice: GitChoiceOption) => void | Promise<void>;
  onResolveVisualConflict?: (fileChoices: Record<string, 'mine' | 'remote'>) => void | Promise<void>;
  onUndoSync?: () => void | Promise<void>;
  className?: string;
}

export const GitSyncRunner: React.FC<GitSyncRunnerProps> = ({
  data,
  onSelectChoice,
  onResolveVisualConflict,
  onUndoSync,
  className = '',
}) => {
  const [showVisualChooser, setShowVisualChooser] = useState(false);
  const [showCliDetails, setShowCliDetails] = useState(false);
  const [activeChoiceId, setActiveChoiceId] = useState<string | null>(null);

  const handleChoiceClick = async (choice: GitChoiceOption) => {
    if (!onSelectChoice) return;
    setActiveChoiceId(choice.id);
    try {
      await onSelectChoice(choice);
    } finally {
      setActiveChoiceId(null);
    }
  };

  if (!data) return null;

  const isConflict = data.status === 'CONFLICT_DETECTED';
  const isDirtyPrompt = data.status === 'DIRTY_WORKTREE_PROMPT';

  return (
    <div className={`space-y-5 ${className}`}>
      {/* Header Info */}
      <div>
        <p className="text-xs text-muted-foreground">
          {data.message || 'Synchronizing workspace with remote repository.'}
        </p>
      </div>

      {/* Dynamic Branch Badges */}
      {data.dynamicBranches && (
        <div className="flex flex-wrap items-center gap-2 rounded-lg bg-muted/50 p-3 border border-border">
          <span className="text-xs font-semibold text-muted-foreground">Branch Context:</span>
          <span className="rounded bg-blue-500/10 px-2 py-0.5 text-xs text-blue-600 dark:text-blue-300 border border-blue-500/20 font-medium">
            Base: {data.dynamicBranches.baseBranch}
          </span>
          <span className="rounded bg-purple-500/10 px-2 py-0.5 text-xs text-purple-600 dark:text-purple-300 border border-purple-500/20 font-medium">
            Spec: {data.dynamicBranches.specBranch}
          </span>
          {data.dynamicBranches.backupRef && (
            <span className="rounded bg-emerald-500/10 px-2 py-0.5 text-xs text-emerald-600 dark:text-emerald-300 border border-emerald-500/20 font-mono font-medium">
              Snapshot: {data.dynamicBranches.backupRef.replace('refs/foundry/backups/', '')}
            </span>
          )}
        </div>
      )}

      {/* Step Checklist */}
      <div className="space-y-2.5 max-h-56 overflow-y-auto pr-1">
        {(data.steps || []).map((step) => (
          <div
            key={step.id}
            className="flex items-start gap-3 rounded-md border border-border bg-muted/20 p-2.5"
          >
            <div className="mt-0.5">
              {step.status === 'completed' && (
                <span className="text-emerald-500 font-bold">✓</span>
              )}
              {step.status === 'in_progress' && (
                <span className="animate-spin inline-block text-blue-500 font-bold">↻</span>
              )}
              {step.status === 'failed' && (
                <span className="text-rose-500 font-bold">✕</span>
              )}
              {step.status === 'skipped' && (
                <span className="text-muted-foreground font-bold">-</span>
              )}
            </div>
            <div className="flex-1">
              <div className="text-xs font-medium text-foreground">{step.label}</div>
              {step.detail && (
                <div className="mt-0.5 text-[11px] text-muted-foreground font-mono">{step.detail}</div>
              )}
            </div>
          </div>
        ))}
      </div>

      {/* Dirty Worktree Prompt Cards */}
      {isDirtyPrompt && data.availableChoices && onSelectChoice && (
        <div className="rounded-lg border border-blue-500/30 bg-blue-500/10 p-4">
          <h3 className="text-sm font-semibold text-blue-600 dark:text-blue-300 mb-2">
            📝 Uncommitted Workspace Edits Detected
          </h3>
          <p className="text-xs text-muted-foreground mb-3">
            Choose how you would like to handle your local edits before pulling remote changes:
          </p>
          <div className="space-y-2">
            {data.availableChoices.map((choice) => (
              <button
                key={choice.id}
                type="button"
                onClick={() => handleChoiceClick(choice)}
                disabled={activeChoiceId !== null}
                className={`w-full text-left p-3 rounded-md border text-xs transition-all disabled:opacity-60 ${
                  choice.isRecommended
                    ? 'border-blue-500 bg-blue-500/20 font-medium text-blue-700 dark:text-blue-100'
                    : 'border-border bg-card hover:bg-muted text-card-foreground'
                }`}
              >
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    {activeChoiceId === choice.id && (
                      <Loader2 className="h-3.5 w-3.5 animate-spin text-blue-600 dark:text-blue-400 shrink-0" />
                    )}
                    <span className="font-semibold">{choice.title}</span>
                  </div>
                  {choice.isRecommended && (
                    <span className="text-[10px] bg-blue-500/20 text-blue-700 dark:text-blue-300 px-2 py-0.5 rounded-full border border-blue-400/30">
                      Recommended
                    </span>
                  )}
                </div>
                <p className="mt-1 text-[11px] opacity-80">{choice.description}</p>
              </button>
            ))}
          </div>
        </div>
      )}

      {/* Conflict Resolution Banner */}
      {isConflict && data.conflictInfo && !showVisualChooser && (
        <div className="rounded-lg border border-rose-500/40 bg-rose-500/10 p-4">
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-semibold text-rose-600 dark:text-rose-300 flex items-center gap-2">
              ⚠️ Merge Conflict Resolution Required
            </h3>
          </div>
          <p className="mt-1 text-xs text-muted-foreground">
            {data.conflictInfo.nextStepRecommendation}
          </p>

          {data.conflictInfo.conflictingFiles?.length > 0 && (
            <div className="mt-3">
              <span className="text-[11px] font-medium text-muted-foreground">Conflicting Files:</span>
              <ul className="mt-1 space-y-1 font-mono text-[11px] text-rose-700 dark:text-rose-200">
                {data.conflictInfo.conflictingFiles.map((file) => (
                  <li key={file} className="flex items-center gap-1.5">
                    <span>•</span> {file}
                  </li>
                ))}
              </ul>
            </div>
          )}

          {/* Action Buttons */}
          <div className="mt-4 flex flex-wrap gap-2">
            {data.conflictInfo.pullRequestUrl && (
              <a
                href={data.conflictInfo.pullRequestUrl}
                target="_blank"
                rel="noopener noreferrer"
                className="rounded-md bg-rose-600 px-3.5 py-1.5 text-xs font-semibold text-white hover:bg-rose-500 shadow flex items-center gap-1.5"
              >
                <span>🚀 Resolve Conflict on GitHub PR</span>
              </a>
            )}

            <button
              type="button"
              onClick={() => setShowVisualChooser(true)}
              className="rounded-md bg-blue-600 px-3.5 py-1.5 text-xs font-semibold text-white hover:bg-blue-500 shadow"
            >
              ⚡ Visual File Chooser
            </button>

            {onUndoSync && (
              <button
                type="button"
                onClick={onUndoSync}
                className="rounded-md border border-input bg-background px-3.5 py-1.5 text-xs text-muted-foreground hover:bg-muted hover:text-foreground"
              >
                ↩️ 1-Click Undo Sync
              </button>
            )}
          </div>

          {/* Collapsible CLI Accordion */}
          {data.conflictInfo.cliInstructions && (
            <div className="mt-3 border-t border-rose-500/20 pt-2">
              <button
                type="button"
                onClick={() => setShowCliDetails(!showCliDetails)}
                className="text-[11px] text-muted-foreground hover:text-foreground underline"
              >
                {showCliDetails ? 'Hide CLI Commands' : 'Show Advanced CLI Commands'}
              </button>
              {showCliDetails && (
                <pre className="mt-2 rounded bg-muted p-2 font-mono text-[10px] text-foreground overflow-x-auto border border-border">
                  {data.conflictInfo.cliInstructions}
                </pre>
              )}
            </div>
          )}
        </div>
      )}

      {/* Visual Card Chooser Sub-view */}
      {isConflict && showVisualChooser && data.conflictInfo && (
        <div>
          <ConflictCardResolver
            conflictingFiles={data.conflictInfo.conflictingFiles || []}
            onResolve={(choices) => {
              setShowVisualChooser(false);
              if (onResolveVisualConflict) onResolveVisualConflict(choices);
            }}
            onCancel={() => setShowVisualChooser(false)}
          />
        </div>
      )}
    </div>
  );
};
