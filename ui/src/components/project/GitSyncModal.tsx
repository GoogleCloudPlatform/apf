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

import React from 'react';
import {
  GitSyncRunner,
  GitSyncChecklistResponse,
  GitSyncChecklistStep,
  DynamicBranchInfo,
  ConflictResolutionInfo,
} from './GitSyncRunner';
import { GitChoiceOption } from './GitChoiceModal';

export type {
  GitSyncChecklistResponse,
  GitSyncChecklistStep,
  DynamicBranchInfo,
  ConflictResolutionInfo,
};

interface GitSyncModalProps {
  isOpen: boolean;
  onClose: () => void;
  data: GitSyncChecklistResponse | null;
  onSelectChoice?: (choice: GitChoiceOption) => void;
  onResolveVisualConflict?: (fileChoices: Record<string, 'mine' | 'remote'>) => void;
  onUndoSync?: () => void;
}

export const GitSyncModal: React.FC<GitSyncModalProps> = ({
  isOpen,
  onClose,
  data,
  onSelectChoice,
  onResolveVisualConflict,
  onUndoSync,
}) => {
  if (!isOpen || !data) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4 backdrop-blur-sm">
      <div className="w-full max-w-3xl rounded-xl border border-border bg-card p-6 text-card-foreground shadow-2xl">
        {/* Header */}
        <div className="flex items-center justify-between border-b border-border pb-4 mb-4">
          <div>
            <h2 className="text-lg font-semibold text-foreground flex items-center gap-2">
              <span>🔄</span> Git Workspace Synchronization
            </h2>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="text-muted-foreground hover:text-foreground text-lg font-bold"
          >
            ✕
          </button>
        </div>

        {/* Embedded GitSyncRunner Component */}
        <GitSyncRunner
          data={data}
          onSelectChoice={onSelectChoice}
          onResolveVisualConflict={onResolveVisualConflict}
          onUndoSync={onUndoSync}
        />

        {/* Footer */}
        <div className="mt-6 flex justify-end border-t border-border pt-4">
          <button
            type="button"
            onClick={onClose}
            className="rounded-md border border-input bg-background px-4 py-2 text-xs font-medium text-muted-foreground hover:bg-muted hover:text-foreground"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  );
};
