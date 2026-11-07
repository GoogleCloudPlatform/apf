// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package workspace

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gofrs/flock"
)

const (
	defaultRemoteName = "origin"
	rebaseAuthorName  = "Foundry Rebase Swarm"
	rebaseAuthorEmail = "rebase@foundry.google.com"
)

var (
	ErrWriteAccessDenied = errors.New("write access to repository not granted")
)

type GitFileStatusItem struct {
	Path      string `json:"path"`
	Staged    string `json:"staged"`
	Worktree  string `json:"worktree"`
	IsChecked bool   `json:"isChecked"`
}

type GitClientInterface interface {
	Pull(ctx context.Context, organizationID string, repoCfg *GitRepoConfig) error
	LocalCommit(ctx context.Context, organizationID string, repoCfg *GitRepoConfig, filePath string, message string, authorName string, authorEmail string) error
	StagePath(ctx context.Context, organizationID string, repoCfg *GitRepoConfig, relPath string) error
	Push(ctx context.Context, organizationID string, repoCfg *GitRepoConfig) error
	EnsureRepoInitialized(ctx context.Context, organizationID string, repoCfg *GitRepoConfig) error
	CreateSnapshot(ctx context.Context, organizationID string, repoCfg *GitRepoConfig, description string) (*GitSnapshot, error)
	RestoreSnapshot(ctx context.Context, organizationID string, repoCfg *GitRepoConfig, snapshotID string) error
	ListSnapshots(ctx context.Context, organizationID string, repoCfg *GitRepoConfig) ([]*GitSnapshot, error)
	GetWorkspaceStatus(ctx context.Context, organizationID string, repoCfg *GitRepoConfig) ([]GitFileStatusItem, error)
	FastForwardBaseToSpec(ctx context.Context, organizationID string, repoCfg *GitRepoConfig) error
	CheckBranchMergeability(ctx context.Context, organizationID string, repoCfg *GitRepoConfig, baseBranch, specBranch string) (bool, []string, error)
}

type GitRepoConfig struct {
	RepoURL         string
	RepoAccessToken string
	BaseBranch      string
	SpecBranch      string
	UserId          string
	ProjectId       string
	SpecLocation    string // e.g. "docs/spec/"
	AuthorEmail     string
	AuthorName      string
}

type NFSWritePermittedChecker interface {
	CheckFolderWritePermitted(ctx context.Context, workspacePath string) error
}

type GitClient struct {
	WorkspaceMountPath string
	NFSStore           NFSWritePermittedChecker
	runner             GitCmdRunner
}

func NewGitClient(workspaceMountPath string) *GitClient {
	return &GitClient{
		WorkspaceMountPath: workspaceMountPath,
		runner:             &DefaultGitCmdRunner{},
	}
}

func (c *GitClient) WithNFSStore(store NFSWritePermittedChecker) *GitClient {
	c.NFSStore = store
	return c
}

func (c *GitClient) WithRunner(runner GitCmdRunner) *GitClient {
	c.runner = runner
	return c
}

func (c *GitClient) runGit(ctx context.Context, dir string, token string, args ...string) (string, error) {
	if c.runner == nil {
		c.runner = &DefaultGitCmdRunner{}
	}
	stdout, _, err := c.runner.Run(ctx, dir, token, args...)
	return stdout, err
}

func (c *GitClient) getClonePath(organizationID string, repoCfg *GitRepoConfig) string {
	return filepath.Join(c.WorkspaceMountPath, organizationID, repoCfg.UserId, repoCfg.ProjectId)
}

func (c *GitClient) acquireLock(ctx context.Context, organizationID string, repoCfg *GitRepoConfig) (*flock.Flock, error) {
	clonePath := c.getClonePath(organizationID, repoCfg)
	lockPath := filepath.Join(c.WorkspaceMountPath, organizationID, repoCfg.UserId, repoCfg.ProjectId+".lock")

	_ = os.MkdirAll(filepath.Dir(lockPath), 0755)
	fileLock := flock.New(lockPath)

	lockCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	locked, err := fileLock.TryLockContext(lockCtx, 250*time.Millisecond)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire workspace lock for %s: %w", clonePath, err)
	}
	if !locked {
		return nil, fmt.Errorf("workspace lock timeout for %s: resource currently locked by another concurrent session", clonePath)
	}

	return fileLock, nil
}

func (c *GitClient) pruneStaleGitLocks(clonePath string) {
	lockFile := filepath.Join(clonePath, ".git", "index.lock")
	if info, err := os.Stat(lockFile); err == nil {
		if time.Since(info.ModTime()) > 5*time.Minute {
			slog.Warn("Detected stale git index lock file, purging", "path", lockFile, "age", time.Since(info.ModTime()))
			_ = os.Remove(lockFile)
		}
	}
	refLocks, _ := filepath.Glob(filepath.Join(clonePath, ".git", "refs", "heads", "*.lock"))
	for _, refLock := range refLocks {
		if info, err := os.Stat(refLock); err == nil && time.Since(info.ModTime()) > 5*time.Minute {
			slog.Warn("Detected stale git branch reference lock file, purging", "path", refLock)
			_ = os.Remove(refLock)
		}
	}
}

func (c *GitClient) recoverCorruptedDirectory(clonePath string) {
	if files, err := os.ReadDir(clonePath); err == nil && len(files) > 0 {
		slog.Warn("Clone target directory exists and is non-empty but missing .git. Purging orphaned folder to enable clean clone.", "path", clonePath)
		_ = os.RemoveAll(clonePath)
		_ = os.MkdirAll(clonePath, 0755)
	}
}

// EnsureRepoInitialized makes sure the repo is cloned and at the correct branch.
// It should be called under lock.
func (c *GitClient) EnsureRepoInitialized(ctx context.Context, organizationID string, repoCfg *GitRepoConfig) error {
	clonePath := c.getClonePath(organizationID, repoCfg)
	if c.NFSStore != nil {
		if err := c.NFSStore.CheckFolderWritePermitted(ctx, clonePath); err != nil {
			return err
		}
	}
	err := os.MkdirAll(clonePath, 0755)
	if err != nil {
		return fmt.Errorf("failed to create workspace directory: %w", err)
	}

	c.pruneStaleGitLocks(clonePath)

	isRepo := false
	if _, errRev := c.runGit(ctx, clonePath, "", "rev-parse", "--is-inside-work-tree"); errRev == nil {
		isRepo = true
	}

	if !isRepo {
		if repoCfg.RepoURL != "" {
			c.recoverCorruptedDirectory(clonePath)
			slog.Info("No git workspace found, cloning repo", "url", repoCfg.RepoURL, "path", clonePath)
			err = c.cloneRepo(ctx, repoCfg, clonePath)
			if err != nil {
				return err
			}
		} else {
			slog.Info("No git workspace found and no remote URL configured, initializing local repo", "path", clonePath)
			base := "main"
			if repoCfg.BaseBranch != "" {
				base = repoCfg.BaseBranch
			}
			if _, errInit := c.runGit(ctx, clonePath, "", "init", "-b", base); errInit != nil {
				if _, errFallback := c.runGit(ctx, clonePath, "", "init"); errFallback != nil {
					return fmt.Errorf("failed to initialize repository: %w", errFallback)
				}
				_, _ = c.runGit(ctx, clonePath, "", "symbolic-ref", "HEAD", "refs/heads/"+base)
			}
		}
	}

	// Apply performance configurations (Protocol v2, skipping negotiation, commit-graphs)
	_, _ = c.runGit(ctx, clonePath, "", "config", "protocol.version", "2")
	_, _ = c.runGit(ctx, clonePath, "", "config", "fetch.negotiationAlgorithm", "skipping")
	_, _ = c.runGit(ctx, clonePath, "", "config", "fetch.writeCommitGraph", "true")
	_, _ = c.runGit(ctx, clonePath, "", "config", "core.commitGraph", "true")
	_, _ = c.runGit(ctx, clonePath, "", "config", "gc.writeCommitGraph", "true")

	if err := c.ensureBranches(ctx, repoCfg, clonePath); err != nil {
		return err
	}

	// Always ensure skills and bmad configuration updates are injected when initialized
	_ = c.EnsureSkillsAndBmad(ctx, organizationID, repoCfg)

	// Ensure .gitignore has required zero-trust exclusions
	_ = ensureGitignoreWithBmad(clonePath)

	// In a newly initialized empty repo, create initial root commit with .gitignore and .agents in background context
	if _, errHead := c.runGit(ctx, clonePath, "", "rev-parse", "HEAD"); errHead != nil {
		go func(path string, cfg *GitRepoConfig) {
			bgCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			_, _ = c.runGit(bgCtx, path, "", "add", "-f", "--", ".gitignore", ".agents")
			initName := "System Bot"
			initEmail := "bot@foundry.google.com"
			if cfg != nil && cfg.AuthorName != "" {
				initName = cfg.AuthorName
			}
			if cfg != nil && cfg.AuthorEmail != "" {
				initEmail = cfg.AuthorEmail
			}
			if _, errCommit := c.runGit(bgCtx, path, "",
				"-c", "user.name="+initName,
				"-c", "user.email="+initEmail,
				"commit", "-m", "chore: initialize repository with zero-trust workspace configuration",
			); errCommit != nil {
				slog.Warn("Background initial git commit encountered error or timed out", "path", path, "error", errCommit)
			}
		}(clonePath, repoCfg)
	}

	return nil
}

func (c *GitClient) EnsureSkillsAndBmad(ctx context.Context, organizationID string, repoCfg *GitRepoConfig) error {
	clonePath := c.getClonePath(organizationID, repoCfg)
	return InjectSkillsForPath(ctx, clonePath, organizationID, repoCfg.UserId, repoCfg.ProjectId)
}

func (c *GitClient) cloneRepo(ctx context.Context, repoCfg *GitRepoConfig, clonePath string) error {
	base := "main"
	if repoCfg.BaseBranch != "" {
		base = repoCfg.BaseBranch
	}
	var cloneErr error
	err := c.withRetries(ctx, func() error {
		_, errClone := c.runGit(ctx, clonePath, repoCfg.RepoAccessToken, "clone", "--branch", base, repoCfg.RepoURL, ".")
		if errClone != nil {
			_, errClone = c.runGit(ctx, clonePath, repoCfg.RepoAccessToken, "clone", repoCfg.RepoURL, ".")
		}
		cloneErr = errClone
		return errClone
	})
	if err != nil {
		if isAuthError(cloneErr) {
			return ErrWriteAccessDenied
		}
		if isRemoteEmptyError(cloneErr) {
			slog.Info("Remote is empty, initializing local repo for push")
			_, _ = c.runGit(ctx, clonePath, "", "init", "-b", base)
			_, _ = c.runGit(ctx, clonePath, "", "remote", "add", defaultRemoteName, repoCfg.RepoURL)
			return nil
		}
		return fmt.Errorf("failed to clone repository from %s: %w", repoCfg.RepoURL, cloneErr)
	}
	return nil
}

func (c *GitClient) ensureBranches(ctx context.Context, repoCfg *GitRepoConfig, clonePath string) error {
	targetBranch := repoCfg.SpecBranch
	if targetBranch == "" || targetBranch == ".invalid" || strings.HasPrefix(targetBranch, ".") {
		targetBranch = repoCfg.BaseBranch
	}
	if targetBranch == "" || targetBranch == ".invalid" || strings.HasPrefix(targetBranch, ".") {
		targetBranch = "main"
	}

	base := repoCfg.BaseBranch
	if base == "" || base == ".invalid" || strings.HasPrefix(base, ".") {
		base = "main"
	}

	currentBranch, _ := c.runGit(ctx, clonePath, "", "symbolic-ref", "--short", "HEAD")
	hasCommits := false
	if _, errHead := c.runGit(ctx, clonePath, "", "rev-parse", "HEAD"); errHead == nil {
		hasCommits = true
	}

	// If already on target branch with commits, short-circuit branch resolution
	if currentBranch == targetBranch && hasCommits {
		return nil
	}

	if repoCfg.RepoURL != "" {
		_ = c.withRetries(ctx, func() error {
			_, err := c.runGit(ctx, clonePath, repoCfg.RepoAccessToken,
				"-c", "protocol.version=2",
				"-c", "fetch.negotiationAlgorithm=skipping",
				"-c", "fetch.writeCommitGraph=true",
				"fetch", "--no-tags", "--prune", defaultRemoteName, "+refs/heads/*:refs/remotes/origin/*",
			)
			return err
		})
	}

	// 1. If remote origin/<targetBranch> exists
	if _, errRemote := c.runGit(ctx, clonePath, "", "show-ref", "--verify", "refs/remotes/origin/"+targetBranch); errRemote == nil {
		if currentBranch != targetBranch || !hasCommits {
			_, errCheckout := c.runGit(ctx, clonePath, "", "checkout", "-f", "-B", targetBranch, "origin/"+targetBranch)
			return errCheckout
		}
		return nil
	}

	// 2. If local <targetBranch> already exists and has commits
	if _, errLocal := c.runGit(ctx, clonePath, "", "show-ref", "--verify", "refs/heads/"+targetBranch); errLocal == nil && hasCommits {
		if currentBranch != targetBranch {
			_, errCheckout := c.runGit(ctx, clonePath, "", "checkout", targetBranch)
			return errCheckout
		}
		return nil
	}

	// 3. If remote origin/<base> exists, branch off origin/<base>
	if _, errRemoteBase := c.runGit(ctx, clonePath, "", "show-ref", "--verify", "refs/remotes/origin/"+base); errRemoteBase == nil {
		_, errCheckout := c.runGit(ctx, clonePath, "", "checkout", "-f", "-B", targetBranch, "origin/"+base)
		return errCheckout
	}

	// 4. If local <base> exists
	if _, errLocalBase := c.runGit(ctx, clonePath, "", "show-ref", "--verify", "refs/heads/"+base); errLocalBase == nil && hasCommits {
		_, errCheckout := c.runGit(ctx, clonePath, "", "checkout", "-f", "-B", targetBranch, base)
		return errCheckout
	}

	// 5. Empty repo with no commits anywhere: set symbolic-ref HEAD
	if !hasCommits {
		_, errSym := c.runGit(ctx, clonePath, "", "symbolic-ref", "HEAD", "refs/heads/"+targetBranch)
		return errSym
	}

	_, errCheckout := c.runGit(ctx, clonePath, "", "checkout", "-B", targetBranch)
	return errCheckout
}

func (c *GitClient) Pull(ctx context.Context, organizationID string, repoCfg *GitRepoConfig) error {
	if c.NFSStore != nil {
		clonePath := c.getClonePath(organizationID, repoCfg)
		if err := c.NFSStore.CheckFolderWritePermitted(ctx, clonePath); err != nil {
			return err
		}
	}
	lock, err := c.acquireLock(ctx, organizationID, repoCfg)
	if err != nil {
		return err
	}
	defer lock.Unlock()

	if err := c.EnsureRepoInitialized(ctx, organizationID, repoCfg); err != nil {
		return err
	}

	clonePath := c.getClonePath(organizationID, repoCfg)
	return c.pullInternal(ctx, organizationID, repoCfg, clonePath)
}

// syncBaseBranchFromRemote fetches and updates the local base branch (e.g. main)
// from the remote repository to ensure main is synchronized with remote before comparisons or rebases.
// It implements pre-flight short-circuiting to avoid unnecessary fetches when remote is unchanged.
func (c *GitClient) syncBaseBranchFromRemote(ctx context.Context, clonePath string, repoCfg *GitRepoConfig) {
	if repoCfg.RepoURL == "" || repoCfg.BaseBranch == "" {
		return
	}

	remoteBaseRef := "refs/remotes/origin/" + repoCfg.BaseBranch
	localTrackingHash, _ := c.runGit(ctx, clonePath, "", "rev-parse", "--verify", remoteBaseRef)

	// L169: Pre-flight remote ref check (Short-Circuit)
	if localTrackingHash != "" {
		out, err := c.runGit(ctx, clonePath, repoCfg.RepoAccessToken,
			"-c", "protocol.version=2",
			"ls-remote", "--heads", defaultRemoteName, repoCfg.BaseBranch,
		)
		if err == nil && strings.TrimSpace(out) != "" {
			fields := strings.Fields(out)
			if len(fields) >= 1 && fields[0] == localTrackingHash {
				slog.Debug("Pre-flight short-circuit: Base branch is already up-to-date with remote", "branch", repoCfg.BaseBranch, "hash", localTrackingHash)
				// Ensure local base branch ref matches if it exists
				if localBaseHash, errLocal := c.runGit(ctx, clonePath, "", "rev-parse", "--verify", "refs/heads/"+repoCfg.BaseBranch); errLocal == nil && localBaseHash != localTrackingHash {
					_, _ = c.runGit(ctx, clonePath, "", "update-ref", "refs/heads/"+repoCfg.BaseBranch, localTrackingHash)
				}
				return
			}
		}
	}

	// L170 + L171 + L172: Targeted single-branch fetch with --no-tags, skipping negotiation, and commit graph
	refSpecs := fmt.Sprintf("refs/heads/%s:refs/remotes/origin/%s", repoCfg.BaseBranch, repoCfg.BaseBranch)
	_ = c.withRetries(ctx, func() error {
		_, err := c.runGit(ctx, clonePath, repoCfg.RepoAccessToken,
			"-c", "protocol.version=2",
			"-c", "fetch.negotiationAlgorithm=skipping",
			"-c", "fetch.writeCommitGraph=true",
			"fetch", "--no-tags", "--prune", defaultRemoteName, refSpecs,
		)
		return err
	})

	if _, errRef := c.runGit(ctx, clonePath, "", "show-ref", "--verify", "refs/remotes/origin/"+repoCfg.BaseBranch); errRef == nil {
		if remoteBaseHash, errHash := c.runGit(ctx, clonePath, "", "rev-parse", "refs/remotes/origin/"+repoCfg.BaseBranch); errHash == nil && remoteBaseHash != "" {
			_, _ = c.runGit(ctx, clonePath, "", "update-ref", "refs/heads/"+repoCfg.BaseBranch, remoteBaseHash)
		}
	}
}

func (c *GitClient) pullInternal(ctx context.Context, organizationID string, repoCfg *GitRepoConfig, clonePath string) error {
	// Auto-commit dirty changes locally before executing pull
	statusOut, errStatus := c.runGit(ctx, clonePath, "", "status", "--porcelain=v1")
	if errStatus == nil && strings.TrimSpace(statusOut) != "" {
		specFiles := c.getSpecPathsFromPorcelain(statusOut, repoCfg)
		if len(specFiles) > 0 {
			slog.Warn("Worktree has dirty changes blocking pull. Attempting auto-commit of dirty specifications.")
			for _, specFile := range specFiles {
				_, _ = c.runGit(ctx, clonePath, "", "add", "--", specFile)
			}
			_, _ = c.runGit(ctx, clonePath, "", "commit", "-m", "Auto-commit changes before pull")
		}
	}

	// Fetch remote references for base branch and spec branch
	if repoCfg.RepoURL != "" {
		c.syncBaseBranchFromRemote(ctx, clonePath, repoCfg)
		if repoCfg.SpecBranch != "" && repoCfg.SpecBranch != repoCfg.BaseBranch {
			remoteSpecRef := "refs/remotes/origin/" + repoCfg.SpecBranch
			localSpecTrackingHash, _ := c.runGit(ctx, clonePath, "", "rev-parse", "--verify", remoteSpecRef)

			needsSpecFetch := true
			// L169: Pre-flight remote ref check for spec branch
			if localSpecTrackingHash != "" {
				out, err := c.runGit(ctx, clonePath, repoCfg.RepoAccessToken,
					"-c", "protocol.version=2",
					"ls-remote", "--heads", defaultRemoteName, repoCfg.SpecBranch,
				)
				if err == nil && strings.TrimSpace(out) != "" {
					fields := strings.Fields(out)
					if len(fields) >= 1 && fields[0] == localSpecTrackingHash {
						slog.Debug("Pre-flight short-circuit: Spec branch is already up-to-date with remote", "branch", repoCfg.SpecBranch, "hash", localSpecTrackingHash)
						needsSpecFetch = false
					}
				}
			}

			if needsSpecFetch {
				refSpecs := fmt.Sprintf("refs/heads/%s:refs/remotes/origin/%s", repoCfg.SpecBranch, repoCfg.SpecBranch)
				_ = c.withRetries(ctx, func() error {
					_, err := c.runGit(ctx, clonePath, repoCfg.RepoAccessToken,
						"-c", "protocol.version=2",
						"-c", "fetch.negotiationAlgorithm=skipping",
						"-c", "fetch.writeCommitGraph=true",
						"fetch", "--no-tags", "--prune", defaultRemoteName, refSpecs,
					)
					return err
				})
			}
		}
	}

	localHeadHash, errHead := c.runGit(ctx, clonePath, "", "rev-parse", "HEAD")
	if errHead != nil || localHeadHash == "" {
		slog.Info("Local branch has no commits yet, skipping rebase", "branch", repoCfg.SpecBranch)
		return nil
	}

	// Resolve upstream commit:
	// Priority 1: If remote spec branch has updates not in localHead, rebase on remoteSpecRef.
	// Priority 2: If local base branch (main) has updates not in localHead, rebase on localBaseRef.
	var upstreamTarget string

	if repoCfg.SpecBranch != "" {
		remoteSpecRef := "refs/remotes/origin/" + repoCfg.SpecBranch
		if remoteSpecHash, errRef := c.runGit(ctx, clonePath, "", "rev-parse", remoteSpecRef); errRef == nil && remoteSpecHash != "" && remoteSpecHash != localHeadHash {
			bases, _ := c.runGit(ctx, clonePath, "", "merge-base", localHeadHash, remoteSpecHash)
			if bases != remoteSpecHash {
				upstreamTarget = remoteSpecRef
			}
		}
	}

	if upstreamTarget == "" && repoCfg.BaseBranch != "" {
		localBaseRef := "refs/heads/" + repoCfg.BaseBranch
		if baseHash, errRef := c.runGit(ctx, clonePath, "", "rev-parse", localBaseRef); errRef == nil && baseHash != "" && baseHash != localHeadHash {
			bases, _ := c.runGit(ctx, clonePath, "", "merge-base", localHeadHash, baseHash)
			if bases != baseHash {
				upstreamTarget = localBaseRef
			}
		}
	}

	if upstreamTarget == "" {
		slog.Info("Local branch is already up-to-date with upstream (base and remote spec). No rebase needed.")
		return nil
	}

	// Check if fast-forward is possible
	mergeBase, _ := c.runGit(ctx, clonePath, "", "merge-base", localHeadHash, upstreamTarget)
	if mergeBase == localHeadHash {
		slog.Info("Fast-forwarding local branch to upstream target", "target", upstreamTarget)
		targetHash, _ := c.runGit(ctx, clonePath, "", "rev-parse", upstreamTarget)
		_, errReset := c.runGit(ctx, clonePath, "", "reset", "--hard", targetHash)
		return errReset
	}

	// Divergence detected: Rebase local unique commits on top of upstream target
	slog.Info("Divergence detected. Rebasing local unique commits on top of upstream target",
		"local_head", localHeadHash,
		"upstream", upstreamTarget,
	)

	preRebaseHash := localHeadHash
	_, errRebase := c.runGit(ctx, clonePath, "", "rebase", upstreamTarget)
	if errRebase != nil {
		slog.Warn("Rebase failed or encountered conflict, rolling back to pre-rebase state", "error", errRebase)
		_, _ = c.runGit(ctx, clonePath, "", "rebase", "--abort")
		_, _ = c.runGit(ctx, clonePath, "", "reset", "--hard", preRebaseHash)
		if repoCfg.SpecBranch != "" {
			_, _ = c.runGit(ctx, clonePath, "", "update-ref", "refs/heads/"+repoCfg.SpecBranch, preRebaseHash)
		}
		return fmt.Errorf("rebase failed due to conflict: %w", errRebase)
	}

	slog.Info("Rebase pull successful")
	return nil
}

func (c *GitClient) LocalCommit(ctx context.Context, organizationID string, repoCfg *GitRepoConfig, filePath string, message string, authorName string, authorEmail string) error {
	if c.NFSStore != nil {
		clonePath := c.getClonePath(organizationID, repoCfg)
		if err := c.NFSStore.CheckFolderWritePermitted(ctx, clonePath); err != nil {
			return err
		}
	}
	lock, err := c.acquireLock(ctx, organizationID, repoCfg)
	if err != nil {
		return err
	}
	defer lock.Unlock()

	if err := c.EnsureRepoInitialized(ctx, organizationID, repoCfg); err != nil {
		return err
	}

	clonePath := c.getClonePath(organizationID, repoCfg)
	if filePath != "" {
		if _, err := c.runGit(ctx, clonePath, "", "add", "--", filePath); err != nil {
			return fmt.Errorf("failed to stage file %s: %w", filePath, err)
		}
	}

	// Check if anything is staged to commit
	if _, errDiff := c.runGit(ctx, clonePath, "", "diff", "--cached", "--quiet"); errDiff == nil {
		slog.Info("Workspace clean or nothing staged, nothing to commit", "file", filePath)
		return nil
	}

	if authorName == "" {
		if repoCfg != nil && repoCfg.AuthorName != "" {
			authorName = repoCfg.AuthorName
		} else {
			authorName = "System Bot"
		}
	}
	if authorEmail == "" {
		if repoCfg != nil && repoCfg.AuthorEmail != "" {
			authorEmail = repoCfg.AuthorEmail
		} else {
			authorEmail = "bot@foundry.google.com"
		}
	}

	_, errCommit := c.runGit(ctx, clonePath, "",
		"-c", "user.name="+authorName,
		"-c", "user.email="+authorEmail,
		"commit", "-m", message,
	)
	if errCommit != nil {
		return fmt.Errorf("failed to commit: %w", errCommit)
	}

	slog.Info("Local commit successful", "file", filePath, "message", message)
	return nil
}

func (c *GitClient) StagePath(ctx context.Context, organizationID string, repoCfg *GitRepoConfig, relPath string) error {
	if c.NFSStore != nil {
		clonePath := c.getClonePath(organizationID, repoCfg)
		if err := c.NFSStore.CheckFolderWritePermitted(ctx, clonePath); err != nil {
			return err
		}
	}
	lock, err := c.acquireLock(ctx, organizationID, repoCfg)
	if err != nil {
		return err
	}
	defer lock.Unlock()

	if err := c.EnsureRepoInitialized(ctx, organizationID, repoCfg); err != nil {
		return err
	}

	clonePath := c.getClonePath(organizationID, repoCfg)
	if _, err := c.runGit(ctx, clonePath, "", "add", "-f", "--", relPath); err != nil {
		if _, errAdd := c.runGit(ctx, clonePath, "", "add", "--", relPath); errAdd != nil {
			return fmt.Errorf("failed to stage path %s: %w", relPath, errAdd)
		}
	}
	return nil
}

func (c *GitClient) Push(ctx context.Context, organizationID string, repoCfg *GitRepoConfig) error {
	if c.NFSStore != nil {
		clonePath := c.getClonePath(organizationID, repoCfg)
		if err := c.NFSStore.CheckFolderWritePermitted(ctx, clonePath); err != nil {
			return err
		}
	}
	lock, err := c.acquireLock(ctx, organizationID, repoCfg)
	if err != nil {
		return err
	}
	defer lock.Unlock()

	if err := c.EnsureRepoInitialized(ctx, organizationID, repoCfg); err != nil {
		return err
	}

	clonePath := c.getClonePath(organizationID, repoCfg)
	return c.pushWithRecovery(ctx, organizationID, repoCfg, clonePath)
}

func (c *GitClient) pushWithRecovery(ctx context.Context, organizationID string, repoCfg *GitRepoConfig, clonePath string) error {
	return c.pushWithRecoveryDepth(ctx, organizationID, repoCfg, clonePath, 0)
}

func (c *GitClient) pushWithRecoveryDepth(ctx context.Context, organizationID string, repoCfg *GitRepoConfig, clonePath string, depth int) error {
	if depth >= 3 {
		return fmt.Errorf("git push failed: non-fast-forward push rejected after %d rebase retries", depth)
	}

	refSpecs := []string{
		fmt.Sprintf("refs/heads/%s:refs/heads/%s", repoCfg.SpecBranch, repoCfg.SpecBranch),
	}
	// If base branch does not exist on remote origin yet, push local base branch to initialize remote repository
	if repoCfg.BaseBranch != "" && repoCfg.BaseBranch != repoCfg.SpecBranch {
		if _, errRemoteBase := c.runGit(ctx, clonePath, "", "show-ref", "--verify", "refs/remotes/origin/"+repoCfg.BaseBranch); errRemoteBase != nil {
			if _, errLocalBase := c.runGit(ctx, clonePath, "", "show-ref", "--verify", "refs/heads/"+repoCfg.BaseBranch); errLocalBase == nil {
				refSpecs = append(refSpecs, fmt.Sprintf("refs/heads/%s:refs/heads/%s", repoCfg.BaseBranch, repoCfg.BaseBranch))
			}
		}
	}

	args := append([]string{"push", defaultRemoteName}, refSpecs...)
	var pushErr error
	err := c.withRetries(ctx, func() error {
		_, errPush := c.runGit(ctx, clonePath, repoCfg.RepoAccessToken, args...)
		pushErr = errPush
		return errPush
	})

	if err != nil {
		if isAuthError(pushErr) {
			return ErrWriteAccessDenied
		}

		if isNonFastForwardError(pushErr) {
			slog.Warn("Push rejected (non-fast-forward), attempting fetch, rebase, and retry", "retryDepth", depth+1)
			errPull := c.pullInternal(ctx, organizationID, repoCfg, clonePath)
			if errPull == nil {
				errRetry := c.pushWithRecoveryDepth(ctx, organizationID, repoCfg, clonePath, depth+1)
				if errRetry == nil {
					return nil
				}
			}
			// If spec branch rebase failed or still rejected non-fast-forward, force push spec staging branch
			if repoCfg.SpecBranch != "" && repoCfg.SpecBranch != repoCfg.BaseBranch {
				slog.Info("Rebase push rejected on spec staging branch. Force pushing spec branch to remote origin.", "specBranch", repoCfg.SpecBranch)
				forceArgs := append([]string{"push", "--force", defaultRemoteName}, refSpecs...)
				_, errForce := c.runGit(ctx, clonePath, repoCfg.RepoAccessToken, forceArgs...)
				if errForce == nil {
					slog.Info("Successfully force-pushed spec staging branch", "branch", repoCfg.SpecBranch)
					return nil
				}
				return fmt.Errorf("failed to force push spec branch %s: %w", repoCfg.SpecBranch, errForce)
			}
			return fmt.Errorf("failed to pull/rebase after push rejection: %w", errPull)
		}

		return fmt.Errorf("failed to push changes: %w", pushErr)
	}

	slog.Info("Push successful", "branch", repoCfg.SpecBranch)
	return nil
}

func (c *GitClient) GetWorkspaceStatus(ctx context.Context, organizationID string, repoCfg *GitRepoConfig) ([]GitFileStatusItem, error) {
	lock, err := c.acquireLock(ctx, organizationID, repoCfg)
	if err != nil {
		return nil, err
	}
	defer lock.Unlock()

	if err := c.EnsureRepoInitialized(ctx, organizationID, repoCfg); err != nil {
		return nil, err
	}

	clonePath := c.getClonePath(organizationID, repoCfg)
	itemsMap := make(map[string]GitFileStatusItem)

	// 1. Collect uncommitted working tree changes
	statusOut, errStatus := c.runGit(ctx, clonePath, "", "status", "--porcelain=v1")
	if errStatus == nil && statusOut != "" {
		for _, line := range strings.Split(statusOut, "\n") {
			line = strings.TrimRight(line, "\r\n")
			if len(line) < 3 {
				continue
			}
			stagedCode := line[0]
			worktreeCode := line[1]
			path := strings.TrimSpace(line[3:])
			if strings.Contains(path, " -> ") {
				parts := strings.Split(path, " -> ")
				path = strings.TrimSpace(parts[len(parts)-1])
			}

			itemsMap[path] = GitFileStatusItem{
				Path:      path,
				Staged:    mapStatusCode(stagedCode),
				Worktree:  mapStatusCode(worktreeCode),
				IsChecked: true,
			}
		}
	} else if errStatus != nil {
		return nil, fmt.Errorf("failed to get git status: %w", errStatus)
	}

	// 2. Collect committed changes on specBranch ahead of baseBranch
	if repoCfg.BaseBranch != "" && repoCfg.SpecBranch != "" && repoCfg.BaseBranch != repoCfg.SpecBranch {
		// Synchronize base branch (main) with remote before comparing
		c.syncBaseBranchFromRemote(ctx, clonePath, repoCfg)

		baseRef := "refs/heads/" + repoCfg.BaseBranch
		if _, errCheck := c.runGit(ctx, clonePath, "", "rev-parse", "--verify", baseRef); errCheck != nil {
			baseRef = "refs/remotes/origin/" + repoCfg.BaseBranch
			if _, errOrigin := c.runGit(ctx, clonePath, "", "rev-parse", "--verify", baseRef); errOrigin != nil {
				baseRef = repoCfg.BaseBranch
			}
		}

		specRef := "refs/heads/" + repoCfg.SpecBranch
		if _, errCheck := c.runGit(ctx, clonePath, "", "rev-parse", "--verify", specRef); errCheck != nil {
			specRef = "HEAD"
		}

		diffOut, errDiff := c.runGit(ctx, clonePath, "", "diff", "--name-status", baseRef+"..."+specRef)
		if errDiff == nil && diffOut != "" {
			for _, line := range strings.Split(diffOut, "\n") {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				parts := strings.Fields(line)
				if len(parts) < 2 {
					continue
				}
				statusCode := parts[0][0]
				path := parts[len(parts)-1]

				if existing, exists := itemsMap[path]; exists {
					if existing.Staged == "unmodified" || existing.Staged == "" {
						existing.Staged = "committed"
					}
					itemsMap[path] = existing
				} else {
					itemsMap[path] = GitFileStatusItem{
						Path:      path,
						Staged:    "committed",
						Worktree:  mapStatusCode(statusCode),
						IsChecked: true,
					}
				}
			}
		}
	}

	items := make([]GitFileStatusItem, 0, len(itemsMap))
	for _, item := range itemsMap {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Path < items[j].Path
	})

	return items, nil
}

func (c *GitClient) FastForwardBaseToSpec(ctx context.Context, organizationID string, repoCfg *GitRepoConfig) error {
	if repoCfg.BaseBranch == "" || repoCfg.SpecBranch == "" || repoCfg.BaseBranch == repoCfg.SpecBranch {
		return nil
	}

	lock, err := c.acquireLock(ctx, organizationID, repoCfg)
	if err != nil {
		return err
	}
	defer lock.Unlock()

	clonePath := c.getClonePath(organizationID, repoCfg)
	specHash, err := c.runGit(ctx, clonePath, "", "rev-parse", "refs/heads/"+repoCfg.SpecBranch)
	if err != nil {
		return fmt.Errorf("failed to get spec branch reference: %w", err)
	}

	if _, err := c.runGit(ctx, clonePath, "", "update-ref", "refs/heads/"+repoCfg.BaseBranch, specHash); err != nil {
		return fmt.Errorf("failed to set base branch reference to spec commit: %w", err)
	}

	slog.Info("Fast-forwarded local base branch to spec commit", "baseBranch", repoCfg.BaseBranch, "specBranch", repoCfg.SpecBranch, "commit", specHash)
	return nil
}

func (c *GitClient) CheckBranchMergeability(ctx context.Context, organizationID string, repoCfg *GitRepoConfig, baseBranch, specBranch string) (bool, []string, error) {
	if baseBranch == "" {
		baseBranch = repoCfg.BaseBranch
	}
	if specBranch == "" {
		specBranch = repoCfg.SpecBranch
	}

	lock, err := c.acquireLock(ctx, organizationID, repoCfg)
	if err != nil {
		return false, nil, err
	}
	defer lock.Unlock()

	clonePath := c.getClonePath(organizationID, repoCfg)
	out, err := c.runGit(ctx, clonePath, "", "merge-tree", "refs/heads/"+baseBranch, "refs/heads/"+specBranch)
	if err != nil {
		var conflicts []string
		for _, l := range strings.Split(out, "\n") {
			if strings.Contains(l, "changed in both") || strings.Contains(l, "CONFLICT") {
				conflicts = append(conflicts, strings.TrimSpace(l))
			}
		}
		return false, conflicts, nil
	}

	return true, nil, nil
}

func (c *GitClient) getSpecPathsFromPorcelain(statusOut string, repoCfg *GitRepoConfig) []string {
	var paths []string
	lines := strings.Split(statusOut, "\n")
	for _, line := range lines {
		if len(line) < 3 {
			continue
		}
		path := strings.TrimSpace(line[3:])
		if path == ".gitignore" || strings.HasPrefix(path, ".agents") || strings.HasPrefix(path, "docs/spec") {
			paths = append(paths, path)
		} else if repoCfg.SpecLocation != "" && (path == repoCfg.SpecLocation || strings.HasPrefix(path, repoCfg.SpecLocation)) {
			paths = append(paths, path)
		}
	}
	return paths
}

func mapStatusCode(code byte) string {
	switch code {
	case 'M':
		return "modified"
	case 'A':
		return "added"
	case 'D':
		return "deleted"
	case 'R':
		return "renamed"
	case 'C':
		return "copied"
	case '?':
		return "untracked"
	default:
		return "unmodified"
	}
}

func isAuthError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "write access to repository not granted") ||
		strings.Contains(msg, "authentication failed") ||
		strings.Contains(msg, "status code: 403") ||
		strings.Contains(msg, "status code: 401") ||
		strings.Contains(msg, "permission to") && strings.Contains(msg, "denied")
}

func isNonFastForwardError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "non-fast-forward") ||
		strings.Contains(msg, "fetch first") ||
		strings.Contains(msg, "rejected")
}

func isRemoteEmptyError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "couldn't find remote ref") ||
		strings.Contains(msg, "remote repository is empty") ||
		strings.Contains(msg, "empty repository")
}

func isTransientError(err error) bool {
	if err == nil {
		return false
	}
	if isAuthError(err) || isRemoteEmptyError(err) || isNonTransientError(err) {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "eof") ||
		strings.Contains(msg, "try again") ||
		strings.Contains(msg, "slow down") ||
		strings.Contains(msg, "429") ||
		strings.Contains(msg, "503") ||
		strings.Contains(msg, "502")
}

func isNonTransientError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "repository not found") ||
		strings.Contains(msg, "couldn't find remote ref") ||
		strings.Contains(msg, "not a git repository") ||
		strings.Contains(msg, "does not exist") ||
		strings.Contains(msg, "cannot open")
}

func (c *GitClient) withRetries(ctx context.Context, op func() error) error {
	backoffs := []time.Duration{100 * time.Millisecond, 200 * time.Millisecond, 500 * time.Millisecond}
	var lastErr error
	for i := 0; i <= len(backoffs); i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		err := op()
		if err == nil {
			return nil
		}
		lastErr = err
		if !isTransientError(err) {
			return err
		}
		if i < len(backoffs) {
			slog.Warn("Git operation failed with transient error, retrying...", "attempt", i+1, "error", err)
			time.Sleep(backoffs[i])
		}
	}
	return lastErr
}
