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

interface ConflictCardResolverProps {
  conflictingFiles: string[];
  onResolve: (fileChoices: Record<string, 'mine' | 'remote'>) => void;
  onCancel: () => void;
}

export const ConflictCardResolver: React.FC<ConflictCardResolverProps> = ({
  conflictingFiles,
  onResolve,
  onCancel,
}) => {
  const [selections, setSelections] = useState<Record<string, 'mine' | 'remote'>>(() => {
    const initial: Record<string, 'mine' | 'remote'> = {};
    conflictingFiles.forEach((file) => {
      initial[file] = 'mine'; // Default recommendation to Keep Mine
    });
    return initial;
  });

  const handleSelect = (file: string, choice: 'mine' | 'remote') => {
    setSelections((prev) => ({ ...prev, [file]: choice }));
  };

  const handleApply = () => {
    onResolve(selections);
  };

  return (
    <div className="rounded-lg border border-amber-500/40 bg-card p-5 shadow-lg text-card-foreground">
      <div className="mb-4">
        <h3 className="text-base font-semibold text-amber-600 dark:text-amber-400">
          ⚡ Visual File Conflict Chooser
        </h3>
        <p className="mt-1 text-xs text-muted-foreground">
          Select which version to preserve for each conflicting file below.
        </p>
      </div>

      <div className="space-y-4 max-h-72 overflow-y-auto pr-1">
        {conflictingFiles.map((filePath) => (
          <div key={filePath} className="rounded-md border border-border bg-muted/30 p-3">
            <div className="mb-2 font-mono text-xs text-foreground truncate font-medium">
              📄 {filePath}
            </div>

            <div className="grid grid-cols-2 gap-3">
              <button
                type="button"
                onClick={() => handleSelect(filePath, 'mine')}
                className={`flex flex-col items-center rounded-md border p-2.5 text-xs transition-all ${
                  selections[filePath] === 'mine'
                    ? 'border-blue-500 bg-blue-500/10 font-semibold text-blue-600 dark:text-blue-300 ring-1 ring-blue-500'
                    : 'border-border bg-background text-muted-foreground hover:border-muted-foreground/50 hover:text-foreground'
                }`}
              >
                <span>Keep My Version</span>
                <span className="mt-0.5 text-[10px] opacity-70 font-normal">Preserve local draft</span>
              </button>

              <button
                type="button"
                onClick={() => handleSelect(filePath, 'remote')}
                className={`flex flex-col items-center rounded-md border p-2.5 text-xs transition-all ${
                  selections[filePath] === 'remote'
                    ? 'border-purple-500 bg-purple-500/10 font-semibold text-purple-600 dark:text-purple-300 ring-1 ring-purple-500'
                    : 'border-border bg-background text-muted-foreground hover:border-muted-foreground/50 hover:text-foreground'
                }`}
              >
                <span>Keep Remote Version</span>
                <span className="mt-0.5 text-[10px] opacity-70 font-normal">Overwrite with repository</span>
              </button>
            </div>
          </div>
        ))}
      </div>

      <div className="mt-5 flex justify-end gap-2 border-t border-border pt-3">
        <button
          type="button"
          onClick={onCancel}
          className="rounded-md border border-input bg-background px-3 py-1.5 text-xs text-muted-foreground hover:bg-muted hover:text-foreground"
        >
          Back
        </button>
        <button
          type="button"
          onClick={handleApply}
          className="rounded-md bg-blue-600 px-4 py-1.5 text-xs font-semibold text-white hover:bg-blue-500 shadow"
        >
          Apply Choices & Resolve
        </button>
      </div>
    </div>
  );
};
