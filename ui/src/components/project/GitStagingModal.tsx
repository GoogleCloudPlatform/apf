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

import React, { useState, useEffect } from 'react';
import { GitPullRequest, HelpCircle, Loader2, ChevronDown, ChevronRight, ArrowLeftRight, ArrowRight, ArrowLeft, RefreshCw, AlertTriangle } from 'lucide-react';

export interface GitFileChange {
  path: string;
  status: string; // 'modified' | 'added' | 'deleted' | 'untracked'
  isChecked: boolean;
}

export interface GitSyncChecklistStep {
  id: string;
  label: string;
  status: 'completed' | 'in_progress' | 'failed' | 'not_started';
  detail?: string;
  prUrl?: string;
}

interface GitStagingModalProps {
  isOpen: boolean;
  onClose: () => void;
  currentFilePath?: string | null;
  changedFiles: GitFileChange[];
  onConfirmSync: (
    selectedPaths: string[],
    actionType: 'sync' | 'request_review' | 'merge',
    onProgress?: (steps: GitSyncChecklistStep[]) => void,
    direction?: 'push' | 'pull'
  ) => Promise<void> | void;
  onPreviewChanges?: () => Promise<{ mergeable?: boolean; files?: any[] } | void> | void;
  onRefetchStatus?: () => Promise<GitFileChange[] | void> | void;
  isLoading?: boolean;
  userBranch?: string;
  targetBranch?: string;
}

export const GitStagingModal: React.FC<GitStagingModalProps> = ({
  isOpen,
  onClose,
  currentFilePath,
  changedFiles,
  onConfirmSync,
  onPreviewChanges,
  onRefetchStatus,
  isLoading = false,
  userBranch = 'specs/user',
  targetBranch = 'main',
}) => {
  const [fileList, setFileList] = useState<GitFileChange[]>([]);
  const [hoveredAction, setHoveredAction] = useState<'sync' | 'request_review' | null>(null);
  const [selectedAction, setSelectedAction] = useState<'sync' | 'request_review' | null>(null);
  const [activeAction, setActiveAction] = useState<'sync' | 'request_review' | 'merge' | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isPreviewing, setIsPreviewing] = useState(false);
  const [isLoadingStatus, setIsLoadingStatus] = useState(false);
  const [isMergeable, setIsMergeable] = useState<boolean>(true);
  const [steps, setSteps] = useState<GitSyncChecklistStep[] | null>(null);
  const [openAccordion, setOpenAccordion] = useState<'files' | 'stages' | null>('files');
  const [syncDirection, setSyncDirection] = useState<'push' | 'pull'>('push');
  const [hasEvaluatedPreview, setHasEvaluatedPreview] = useState(false);

  useEffect(() => {
    if (isOpen) {
      setSteps(null);
      setHoveredAction(null);
      setSelectedAction(null);
      setActiveAction(null);
      setIsSubmitting(false);
      setIsPreviewing(false);
      setIsMergeable(true);
      setHasEvaluatedPreview(false);
      setOpenAccordion('files');
      setSyncDirection('push');

      if (onRefetchStatus) {
        setIsLoadingStatus(true);
        Promise.resolve(onRefetchStatus())
          .then((fresh) => {
            if (fresh && Array.isArray(fresh)) {
              setFileList(fresh);
            }
          })
          .catch((err) => {
            console.error("Failed to fetch git status on modal open:", err);
          })
          .finally(() => {
            setIsLoadingStatus(false);
          });
      }
    }
  }, [isOpen]);

  useEffect(() => {
    if (changedFiles && changedFiles.length > 0) {
      // Ensure current document is checked by default
      const updated = changedFiles.map((file) => {
        const isCurrentDoc = currentFilePath && (
          file.path === currentFilePath ||
          currentFilePath.endsWith(file.path) ||
          file.path.endsWith(currentFilePath)
        );
        return {
          ...file,
          isChecked: isCurrentDoc ? true : file.isChecked ?? true,
        };
      });
      setFileList(updated);
    } else if (!isLoadingStatus) {
      setFileList([]);
    }
  }, [changedFiles, currentFilePath, isOpen, isLoadingStatus]);

  if (!isOpen) return null;

  const toggleCheck = (path: string) => {
    if (isSubmitting || isLoading || isPreviewing) return;
    setFileList((prev) =>
      prev.map((item) =>
        item.path === path ? { ...item, isChecked: !item.isChecked } : item
      )
    );
  };

  const toggleSelectAll = (checked: boolean) => {
    if (isSubmitting || isLoading || isPreviewing) return;
    setFileList((prev) => prev.map((item) => ({ ...item, isChecked: checked })));
  };

  const selectedCount = fileList.filter((f) => f.isChecked).length;

  const handlePreview = async () => {
    setIsPreviewing(true);
    setOpenAccordion('stages');
    const previewSteps: GitSyncChecklistStep[] = [
      { id: 'eval_diff', label: 'Evaluating diffrences', status: 'in_progress', detail: 'Detecting changes between your branch and base branch' },
      { id: 'eval_merge', label: 'Evaluate Rebase & Merge Status', status: 'not_started' },
    ];
    setSteps(previewSteps);

    try {
      let previewFiles: GitFileChange[] | undefined;
      let mergeable = true;

      if (onPreviewChanges) {
        const res = await onPreviewChanges();
        if (res && typeof res === 'object') {
          if ('mergeable' in res) {
            mergeable = Boolean(res.mergeable);
          }
          if ('files' in res && Array.isArray(res.files)) {
            previewFiles = res.files.map((f: any) => ({
              path: f.path || f.filename,
              status: f.status || 'modified',
              isChecked: true,
            }));
          }
        }
      }

      if (previewFiles) {
        setFileList(previewFiles);
      } else {
        setFileList([]);
      }

      setIsMergeable(mergeable);
      setHasEvaluatedPreview(true);

      if (!mergeable) {
        setSteps([
          { id: 'eval_diff', label: 'Evaluating diffrences', status: 'completed', detail: `Found ${previewFiles ? previewFiles.length : 0} remote file(s) changed` },
          { id: 'eval_merge', label: 'Evaluate Rebase & Merge Status', status: 'failed', detail: 'Merge conflict detected between branches' },
        ]);
        return;
      }

      const fileCountMsg = previewFiles && previewFiles.length > 0
        ? `Found ${previewFiles.length} remote file(s) changed`
        : 'Your spec branch is already up to date.';

      setSteps([
        { id: 'eval_diff', label: 'Evaluating diffrences', status: 'completed', detail: fileCountMsg },
        { id: 'eval_merge', label: 'Evaluate Rebase & Merge Status', status: 'completed', detail: 'Rebase and merge confirmed clean & mergeable' },
      ]);
    } catch (err: any) {
      setIsMergeable(false);
      setSteps([
        { id: 'eval_diff', label: 'Evaluating diffrences', status: 'completed', detail: 'Detecting changes between your branch and base branch' },
        { id: 'eval_merge', label: 'Evaluate Rebase & Merge Status', status: 'failed', detail: err?.message || 'Conflict detected during preview evaluation' },
      ]);
    } finally {
      setIsPreviewing(false);
    }
  };

  const handleToggleDirection = async () => {
    const nextDirection = syncDirection === 'push' ? 'pull' : 'push';
    setSyncDirection(nextDirection);
    setSteps(null);
    setHoveredAction(null);
    setSelectedAction(null);
    setActiveAction(null);
    setIsSubmitting(false);
    setIsPreviewing(false);
    setIsMergeable(true);
    setHasEvaluatedPreview(false);
    setOpenAccordion('files');

    if (nextDirection === 'pull') {
      setTimeout(() => {
        handlePreview();
      }, 0);
    } else {
      if (onRefetchStatus) {
        const fresh = await onRefetchStatus();
        if (fresh && Array.isArray(fresh)) {
          setFileList(fresh);
          return;
        }
      }
      if (changedFiles && changedFiles.length > 0) {
        const updated = changedFiles.map((file) => {
          const isCurrentDoc = currentFilePath && (
            file.path === currentFilePath ||
            currentFilePath.endsWith(file.path) ||
            file.path.endsWith(currentFilePath)
          );
          return {
            ...file,
            isChecked: isCurrentDoc ? true : file.isChecked ?? true,
          };
        });
        setFileList(updated);
      } else {
        setFileList([]);
      }
    }
  };

  const handleConfirm = async (actionType: 'sync' | 'request_review' | 'merge' = 'sync') => {
    const selectedPaths = fileList.filter((f) => f.isChecked).map((f) => f.path);
    const resolvedChoice = actionType === 'request_review' ? 'request_review' : 'sync';
    setSelectedAction(resolvedChoice);
    setActiveAction(actionType);
    setIsSubmitting(true);

    // Switch active accordion exclusively to execution stages when action starts
    setOpenAccordion('stages');

    const initialSteps: GitSyncChecklistStep[] = [
      { id: 'snapshot', label: 'Create Safety Snapshot', status: 'in_progress', detail: 'Backing up local work...' },
      { id: 'commit', label: syncDirection === 'pull' ? 'Fetching remote changes' : 'Approving changes locally', status: 'not_started', detail: `Staging ${selectedPaths.length} file(s)` },
      { id: 'fetch', label: syncDirection === 'pull' ? 'Rebasing local worktree' : 'Fetch Remote Changes', status: 'not_started' },
      { id: 'sync', label: actionType === 'request_review' ? 'Created/Updated Pull Request' : (syncDirection === 'pull' ? 'Pulled Latest Changes' : 'Merged Changes'), status: 'not_started' },
    ];
    setSteps(initialSteps);

    try {
      await onConfirmSync(selectedPaths, actionType, (updatedSteps) => {
        setSteps(updatedSteps);
      }, syncDirection);
    } finally {
      setIsSubmitting(false);
      setActiveAction(null);
    }
  };

  const isBusy = isSubmitting || isLoading;
  const currentFlowAction = activeAction || selectedAction || hoveredAction;
  const hasExecutedTransaction = Boolean(steps && steps.some((s) => s.id === 'sync' || s.id.startsWith('sync') || s.id.startsWith('pr') || s.id === 'snapshot' || s.id === 'fetch'));
  const isFinished = hasExecutedTransaction && Boolean(steps && steps.length > 0 && steps.every((s) => s.status === 'completed' || s.status === 'failed'));

  const isFileAccordionOpen = openAccordion === 'files';
  const isStagesAccordionOpen = openAccordion === 'stages';

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4 backdrop-blur-sm">
      <div className="w-full max-w-3xl rounded-xl border border-border bg-card p-6 text-card-foreground shadow-2xl">
        {/* Header */}
        <div className="flex items-center justify-between border-b border-border pb-4">
          <div>
            <h2 className="text-base font-semibold text-foreground flex items-center gap-2">
              <span>🌿</span> {syncDirection === 'push' ? 'Push Changes' : 'Pull Changes'}
            </h2>
            <p className="mt-1 text-xs text-muted-foreground flex items-center gap-1">
              <span>Select the files you want to publish to target branch</span>
              <span
                title="This is a git branch that is configured in the project to act as a central branch to collect assets updated via this application."
                className="cursor-help inline-flex items-center"
              >
                <HelpCircle className="h-3.5 w-3.5 text-muted-foreground/80 hover:text-foreground transition-colors" />
              </span>
            </p>
          </div>
          <button
            type="button"
            onClick={onClose}
            disabled={isBusy || isPreviewing}
            className="text-muted-foreground hover:text-foreground text-lg font-bold disabled:opacity-50"
          >
            ✕
          </button>
        </div>

        {/* Direction Controls Header */}
        <div className="mt-4 flex items-center justify-between rounded-lg bg-muted/40 px-3.5 py-2 border border-border">
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={handleToggleDirection}
              disabled={isBusy || isPreviewing}
              title="Click to toggle sync direction between Push and Pull"
              className="rounded-lg bg-primary px-3.5 py-1.5 text-xs font-semibold text-primary-foreground shadow hover:opacity-90 disabled:opacity-50 flex items-center gap-2 cursor-pointer active:scale-95 transition-all disabled:cursor-not-allowed"
            >
              <ArrowLeftRight className="h-3.5 w-3.5 text-primary-foreground shrink-0" />
              <span>{syncDirection === 'push' ? 'Push' : 'Pull'}</span>
              <ArrowRight className={`h-3.5 w-3.5 shrink-0 transition-transform duration-200 ${syncDirection === 'pull' ? 'rotate-180' : ''}`} />
              <span>({syncDirection === 'push' ? 'Local to Remote' : 'Remote to Local'})</span>
            </button>
          </div>
        </div>

        {/* Branch Flow Visualization Boxes */}
        <div className="mt-3 flex items-center justify-between gap-3 rounded-lg bg-muted/30 p-3 border border-border">
          <div className="flex-1 flex flex-col items-center justify-center p-2.5 rounded-md border border-purple-500/30 bg-purple-500/10 text-center">
            <span className="font-mono text-xs font-bold text-foreground">
              {userBranch}
            </span>
            <span className="text-[10px] font-semibold text-purple-600 dark:text-purple-400 uppercase tracking-wider mt-0.5">
              {syncDirection === 'push' ? 'Your Branch (Source)' : 'Your Branch (Destination)'}
            </span>
          </div>

          <div className="flex flex-col items-center justify-center text-muted-foreground px-1 shrink-0 transition-all duration-200 min-w-[90px]">
            {currentFlowAction === 'request_review' ? (
              <div className="flex flex-col items-center animate-in fade-in zoom-in duration-200">
                <GitPullRequest className="h-5 w-5 text-purple-600 dark:text-purple-400" />
                <span className="text-[9px] font-semibold text-purple-600 dark:text-purple-400 mt-0.5">
                  Request Review
                </span>
              </div>
            ) : currentFlowAction === 'sync' ? (
              <div className="flex flex-col items-center animate-in fade-in zoom-in duration-200">
                <ArrowRight className={`h-5 w-5 text-primary transition-transform duration-200 ${syncDirection === 'pull' ? 'rotate-180' : ''}`} />
                <span className="text-[9px] font-semibold text-primary mt-0.5">
                  {syncDirection === 'push' ? 'Push Changes' : 'Pull Changes'}
                </span>
              </div>
            ) : (
              <div className="flex flex-col items-center text-muted-foreground/70">
                {syncDirection === 'push' ? (
                  <HelpCircle className="h-5 w-5 text-muted-foreground/70" />
                ) : (
                  <ArrowRight className="h-5 w-5 text-muted-foreground/60 rotate-180" />
                )}
                <span className="text-[9px] font-medium text-muted-foreground mt-0.5">
                  {syncDirection === 'push' ? 'Select PR or Push' : 'Pull from Remote'}
                </span>
              </div>
            )}
          </div>

          <div className="flex-1 flex flex-col items-center justify-center p-2.5 rounded-md border border-blue-500/30 bg-blue-500/10 text-center">
            <span className="font-mono text-xs font-bold text-foreground">
              {targetBranch}
            </span>
            <span className="text-[10px] font-semibold text-blue-600 dark:text-blue-400 uppercase tracking-wider mt-0.5">
              {syncDirection === 'push' ? 'Target Branch (Destination)' : 'Target Branch (Source)'}
            </span>
          </div>
        </div>

        {/* Accordion 1: File List (collapsible, max 10 items scrollbar) */}
        <div className="mt-4 rounded-lg border border-border bg-card overflow-hidden shadow-sm">
          <button
            type="button"
            onClick={() => setOpenAccordion(openAccordion === 'files' ? null : 'files')}
            className="w-full flex items-center justify-between p-3 bg-muted/30 hover:bg-muted/60 text-xs font-semibold text-foreground transition-colors border-b border-border/40 select-none"
          >
            <div className="flex items-center gap-2">
              {isFileAccordionOpen ? (
                <ChevronDown className="h-4 w-4 text-muted-foreground" />
              ) : (
                <ChevronRight className="h-4 w-4 text-muted-foreground" />
              )}
              <span>📁 Changed Files</span>
              <span className="text-[10px] font-mono px-2 py-0.5 rounded-full bg-muted text-muted-foreground font-normal">
                {selectedCount} of {fileList.length} selected
              </span>
            </div>
            {!isFileAccordionOpen && (
              <span className="text-[10px] text-muted-foreground font-normal">Click to expand</span>
            )}
          </button>

          {isFileAccordionOpen && (
            <div className="p-3">
              {isLoadingStatus || isLoading ? (
                <div className="flex flex-col items-center justify-center p-8 text-muted-foreground gap-2 select-none">
                  <Loader2 className="h-5 w-5 animate-spin text-primary" />
                  <span className="text-xs font-medium">Analyzing workspace changes...</span>
                </div>
              ) : fileList.length === 0 ? (
                <div className="flex flex-col items-center justify-center p-6 text-muted-foreground gap-1.5 select-none">
                  <span className="text-xs font-medium">No changed files detected.</span>
                </div>
              ) : (
                <>
                  {/* Select All Toggle */}
                  <div className="mb-2.5 flex items-center justify-between px-1 text-xs text-muted-foreground">
                    <label className="flex items-center gap-2 cursor-pointer select-none">
                      <input
                        type="checkbox"
                        checked={fileList.length > 0 && fileList.every((f) => f.isChecked)}
                        onChange={(e) => toggleSelectAll(e.target.checked)}
                        disabled={isBusy || isPreviewing}
                        className="rounded border-border text-primary focus:ring-primary h-4 w-4 disabled:opacity-50"
                      />
                      <span className="font-medium text-foreground">Select All ({fileList.length} items)</span>
                    </label>
                    <span>{selectedCount} selected</span>
                  </div>

                  {/* Changed Files List (max 10 items = ~360px height + scrollbar) */}
                  <div className="space-y-2 max-h-[360px] overflow-y-auto pr-1">
                    {fileList.map((file) => {
                      const isCurrentDoc = currentFilePath && (
                        file.path === currentFilePath ||
                        currentFilePath.endsWith(file.path) ||
                        file.path.endsWith(currentFilePath)
                      );

                      return (
                        <div
                          key={file.path}
                          onClick={() => toggleCheck(file.path)}
                          className={`flex items-center justify-between p-3 rounded-lg border cursor-pointer transition-all ${
                            file.isChecked
                              ? 'border-blue-500/50 bg-blue-500/10 dark:bg-blue-950/30'
                              : 'border-border bg-muted/20 hover:bg-muted/50'
                          } ${isBusy || isPreviewing ? 'opacity-60 pointer-events-none' : ''}`}
                        >
                          <div className="flex items-center gap-3 min-w-0 flex-1">
                            <input
                              type="checkbox"
                              checked={file.isChecked}
                              onChange={() => toggleCheck(file.path)}
                              onClick={(e) => e.stopPropagation()}
                              disabled={isBusy || isPreviewing}
                              className="rounded border-border text-primary focus:ring-primary h-4 w-4 shrink-0 disabled:opacity-50"
                            />
                            <div className="min-w-0 flex-1">
                              <div className="flex items-center gap-2 flex-wrap">
                                <span className="font-mono text-xs font-medium text-foreground break-all">
                                  {file.path}
                                </span>
                                {isCurrentDoc && (
                                  <span className="rounded bg-emerald-500/20 px-2 py-0.5 text-[10px] font-semibold text-emerald-600 dark:text-emerald-400 shrink-0 border border-emerald-500/30">
                                    Current Document
                                  </span>
                                )}
                              </div>
                            </div>
                          </div>

                          <span className="text-[10px] font-mono capitalize px-2 py-0.5 rounded bg-muted text-muted-foreground shrink-0">
                            {file.status || 'modified'}
                          </span>
                        </div>
                      );
                    })}
                  </div>
                </>
              )}
            </div>
          )}
        </div>

        {/* Accordion 2: Execution Stages */}
        {steps && steps.length > 0 && (
          <div className="mt-3 rounded-lg border border-border bg-card overflow-hidden shadow-sm">
            <button
              type="button"
              onClick={() => setOpenAccordion(openAccordion === 'stages' ? null : 'stages')}
              className="w-full flex items-center justify-between p-3 bg-muted/30 hover:bg-muted/60 text-xs font-semibold text-foreground transition-colors border-b border-border/40 select-none"
            >
              <div className="flex items-center gap-2">
                {isStagesAccordionOpen ? (
                  <ChevronDown className="h-4 w-4 text-muted-foreground" />
                ) : (
                  <ChevronRight className="h-4 w-4 text-muted-foreground" />
                )}
                <span>🚀 Execution Stages</span>
                {(isBusy || isPreviewing) && (
                  <span className="text-[10px] text-primary animate-pulse font-normal flex items-center gap-1">
                    <Loader2 className="h-3 w-3 animate-spin" /> Processing...
                  </span>
                )}
              </div>
              <span className="text-[10px] font-mono opacity-70">
                {steps.filter((s) => s.status === 'completed').length} of {steps.length} completed
              </span>
            </button>

            {isStagesAccordionOpen && (
              <div className="p-3.5 space-y-2">
                {steps.map((step) => (
                  <div key={step.id} className="flex items-start gap-3 rounded-md border border-border/50 bg-muted/20 p-2.5">
                    <div className="mt-0.5 shrink-0">
                      {step.status === 'completed' && (
                        <span className="flex h-4 w-4 items-center justify-center rounded-full bg-emerald-500/20 text-emerald-600 dark:text-emerald-400 text-xs font-bold">
                          ✓
                        </span>
                      )}
                      {step.status === 'in_progress' && (
                        <Loader2 className="h-4 w-4 animate-spin text-primary" />
                      )}
                      {step.status === 'failed' && (
                        <span className="flex h-4 w-4 items-center justify-center rounded-full bg-rose-500/20 text-rose-600 dark:text-rose-400 text-xs font-bold">
                          ✕
                        </span>
                      )}
                      {step.status === 'not_started' && (
                        <span className="flex h-4 w-4 items-center justify-center rounded-full border border-border text-[9px] text-muted-foreground font-medium">
                          ○
                        </span>
                      )}
                    </div>
                    <div className="min-w-0 flex-1">
                      <div className="text-xs font-medium text-foreground flex items-center justify-between">
                        <span>{step.label}</span>
                        <span className="text-[10px] capitalize font-mono opacity-70">
                          {step.status === 'not_started' ? 'Not Started' : step.status === 'in_progress' ? 'In Progress' : step.status === 'completed' ? 'Completed' : 'Failed'}
                        </span>
                      </div>
                      {(step.detail || step.prUrl) && (
                        <div className="mt-0.5 font-mono text-[10px] text-muted-foreground flex items-center gap-2 flex-wrap">
                          {step.detail && <span>{step.detail}</span>}
                          {step.prUrl && (
                            <a
                              href={step.prUrl}
                              target="_blank"
                              rel="noopener noreferrer"
                              className="inline-flex items-center gap-1 font-semibold text-purple-600 dark:text-purple-400 hover:underline cursor-pointer"
                            >
                              <span>Open PR #{step.prUrl.split('/').pop()} ↗</span>
                            </a>
                          )}
                        </div>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {/* Conflict Guidance Banner */}
        {!isMergeable && (
          <div className="mt-4 rounded-lg border border-amber-500/40 bg-amber-500/10 p-3 text-xs text-amber-900 dark:text-amber-200 flex items-start gap-2 select-none">
            <AlertTriangle className="h-4 w-4 shrink-0 text-amber-600 dark:text-amber-400 mt-0.5" />
            <div>
              <span className="font-semibold block mb-0.5">Automated Branch Merge Blocked</span>
              <span>
                A merge conflict was detected between remote <code className="bg-amber-500/20 px-1 py-0.5 rounded font-mono">{targetBranch}</code> and your branch <code className="bg-amber-500/20 px-1 py-0.5 rounded font-mono">{userBranch}</code>.
                Automated pull is disabled to prevent data loss. Please create a <strong>Pull Request</strong> or resolve conflicts on GitHub.
              </span>
            </div>
          </div>
        )}

        {/* Footer Actions */}
        <div className="mt-6 flex flex-wrap items-center justify-end gap-2 border-t border-border pt-4">
          {isFinished ? (
            <button
              type="button"
              onClick={onClose}
              className="rounded-lg bg-primary px-5 py-2 text-xs font-semibold text-primary-foreground shadow hover:opacity-90 cursor-pointer"
            >
              Done
            </button>
          ) : (
            <>
              <button
                type="button"
                onClick={onClose}
                disabled={isBusy || isPreviewing || isLoadingStatus}
                className="rounded-lg border border-input bg-background px-3.5 py-2 text-xs font-medium text-muted-foreground hover:bg-muted hover:text-foreground disabled:opacity-50"
              >
                Cancel
              </button>

              {syncDirection === 'push' ? (
                <>
                  <button
                    type="button"
                    onClick={() => handleConfirm('request_review')}
                    onMouseEnter={() => setHoveredAction('request_review')}
                    onMouseLeave={() => setHoveredAction(null)}
                    disabled={isBusy || isPreviewing || isLoadingStatus || selectedCount === 0}
                    className="rounded-lg border border-purple-500/40 bg-purple-500/10 px-3.5 py-2 text-xs font-semibold text-purple-700 dark:text-purple-300 hover:bg-purple-500/20 disabled:opacity-50 flex items-center gap-1.5 cursor-pointer"
                  >
                    {activeAction === 'request_review' ? (
                      <Loader2 className="h-3.5 w-3.5 animate-spin text-purple-600 dark:text-purple-400" />
                    ) : null}
                    <span>Request Review ({selectedCount})</span>
                  </button>

                  <button
                    type="button"
                    onClick={() => handleConfirm('sync')}
                    onMouseEnter={() => setHoveredAction('sync')}
                    onMouseLeave={() => setHoveredAction(null)}
                    disabled={isBusy || isPreviewing || isLoadingStatus || selectedCount === 0}
                    className="rounded-lg bg-primary px-4 py-2 text-xs font-semibold text-primary-foreground shadow hover:opacity-90 disabled:opacity-50 flex items-center gap-1.5 cursor-pointer"
                  >
                    {activeAction === 'sync' ? (
                      <Loader2 className="h-3.5 w-3.5 animate-spin text-primary-foreground" />
                    ) : null}
                    <span>Push Changes ({selectedCount})</span>
                  </button>
                </>
              ) : (
                <>
                  <button
                    type="button"
                    onClick={handlePreview}
                    disabled={isBusy || isPreviewing || isLoadingStatus}
                    className="rounded-lg border border-blue-500/40 bg-blue-500/10 px-3.5 py-2 text-xs font-semibold text-blue-700 dark:text-blue-300 hover:bg-blue-500/20 disabled:opacity-50 flex items-center gap-1.5 cursor-pointer"
                    title="Refresh remote changes diff and check mergeability via GitHub API"
                  >
                    {isPreviewing ? (
                      <Loader2 className="h-3.5 w-3.5 animate-spin text-blue-600 dark:text-blue-400" />
                    ) : (
                      <RefreshCw className="h-3.5 w-3.5 text-blue-600 dark:text-blue-400" />
                    )}
                    <span>Refresh</span>
                  </button>

                  <button
                    type="button"
                    onClick={() => handleConfirm('sync')}
                    onMouseEnter={() => setHoveredAction('sync')}
                    onMouseLeave={() => setHoveredAction(null)}
                    disabled={isBusy || isPreviewing || isLoadingStatus || selectedCount === 0 || fileList.length === 0 || !isMergeable}
                    title={
                      fileList.length === 0 || selectedCount === 0
                        ? 'No changes available to pull'
                        : !isMergeable
                        ? 'Cannot pull: Merge conflict or unmergeable state detected'
                        : 'Pull changes from remote'
                    }
                    className="rounded-lg bg-primary px-4 py-2 text-xs font-semibold text-primary-foreground shadow hover:opacity-90 disabled:opacity-50 flex items-center gap-1.5 cursor-pointer disabled:cursor-not-allowed"
                  >
                    {activeAction === 'sync' ? (
                      <Loader2 className="h-3.5 w-3.5 animate-spin text-primary-foreground" />
                    ) : null}
                    <span>Pull Changes{hasEvaluatedPreview ? ` (${selectedCount})` : ''}</span>
                  </button>
                </>
              )}
            </>
          )}
        </div>
      </div>
    </div>
  );
};
