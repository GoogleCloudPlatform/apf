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

'use me';
import React from 'react';

export interface GitChoiceOption {
  id: string;
  title: string;
  description: string;
  isRecommended: boolean;
  actionType: 'save_snapshot_and_sync' | 'discard_local_edits' | 'open_github_pr' | 'visual_card_chooser' | 'undo_sync';
  safetyGuarantee?: string;
}

interface GitChoiceModalProps {
  isOpen: boolean;
  onClose: () => void;
  title: string;
  description: string;
  choices: GitChoiceOption[];
  onSelectChoice: (choice: GitChoiceOption) => void;
  isLoading?: boolean;
}

export const GitChoiceModal: React.FC<GitChoiceModalProps> = ({
  isOpen,
  onClose,
  title,
  description,
  choices,
  onSelectChoice,
  isLoading = false,
}) => {
  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4 backdrop-blur-sm">
      <div className="w-full max-w-xl rounded-xl border border-border bg-card p-6 text-card-foreground shadow-2xl">
        <div className="mb-4">
          <div className="flex items-center gap-2">
            <span className="flex h-8 w-8 items-center justify-center rounded-full bg-blue-500/20 text-blue-600 dark:text-blue-400 font-bold text-lg">
              🛡️
            </span>
            <h2 className="text-xl font-semibold text-foreground">{title}</h2>
          </div>
          <p className="mt-2 text-sm text-muted-foreground">{description}</p>
        </div>

        <div className="mt-6 space-y-4">
          {choices.map((choice) => (
            <div
              key={choice.id}
              onClick={() => !isLoading && onSelectChoice(choice)}
              className={`group cursor-pointer rounded-lg border p-4 transition-all ${
                choice.isRecommended
                  ? 'border-blue-500/60 bg-blue-500/10 hover:border-blue-500 hover:bg-blue-500/20'
                  : 'border-border bg-muted/40 hover:border-muted-foreground/50 hover:bg-muted'
              }`}
            >
              <div className="flex items-start justify-between">
                <div className="flex items-center gap-2">
                  <h3 className="font-semibold text-foreground group-hover:text-blue-600 dark:group-hover:text-blue-400">
                    {choice.title}
                  </h3>
                  {choice.isRecommended && (
                    <span className="rounded-full bg-blue-500/20 px-2.5 py-0.5 text-xs font-medium text-blue-700 dark:text-blue-300 border border-blue-400/30">
                      Recommended
                    </span>
                  )}
                </div>
              </div>

              <p className="mt-1 text-xs text-muted-foreground leading-relaxed">
                {choice.description}
              </p>

              {choice.safetyGuarantee && (
                <div className="mt-3 flex items-center gap-1.5 text-xs font-medium text-emerald-600 dark:text-emerald-400">
                  <svg className="h-4 w-4 fill-current" viewBox="0 0 20 20">
                    <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
                  </svg>
                  <span>{choice.safetyGuarantee}</span>
                </div>
              )}
            </div>
          ))}
        </div>

        <div className="mt-6 flex justify-end gap-3 border-t border-border pt-4">
          <button
            type="button"
            onClick={onClose}
            disabled={isLoading}
            className="rounded-lg border border-input bg-background px-4 py-2 text-sm font-medium text-muted-foreground hover:bg-muted hover:text-foreground disabled:opacity-50"
          >
            Cancel
          </button>
        </div>
      </div>
    </div>
  );
};
