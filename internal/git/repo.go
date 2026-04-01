package git

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// TokenFunc generates a GitHub installation token for cloning private repos.
type TokenFunc func(ctx context.Context, installID int64) (string, error)

// RepoManager handles cloning, fetching, and worktree lifecycle for repositories.
type RepoManager struct {
	basePath  string
	tokenFunc TokenFunc

	mu        sync.Mutex
	repoLocks map[string]*sync.Mutex
}

// NewRepoManager creates a RepoManager. If basePath is empty, all operations
// are no-ops (exploration disabled).
func NewRepoManager(basePath string, tokenFunc TokenFunc) *RepoManager {
	return &RepoManager{
		basePath:  basePath,
		tokenFunc: tokenFunc,
		repoLocks: make(map[string]*sync.Mutex),
	}
}

// IsAvailable checks if the git binary exists on the system.
func (rm *RepoManager) IsAvailable() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

// Enabled returns true if the manager has a base path configured and git is available.
func (rm *RepoManager) Enabled() bool {
	return rm.basePath != "" && rm.IsAvailable()
}

// Prepare ensures the repo is cloned/fetched and creates a worktree for the given branch.
// Returns the worktree path. Caller must defer Cleanup(worktreePath).
// Returns ("", nil) if exploration is disabled (no base path or git unavailable).
func (rm *RepoManager) Prepare(ctx context.Context, repo string, branch string, installID int64) (string, bool, error) {
	if !rm.Enabled() {
		return "", false, nil
	}

	repoDir := filepath.Join(rm.basePath, repo)
	mu := rm.repoMutex(repo)
	mu.Lock()
	defer mu.Unlock()

	firstClone, err := rm.ensureCloned(ctx, repoDir, repo, installID)
	if err != nil {
		return "", false, fmt.Errorf("clone/fetch %s: %w", repo, err)
	}

	wtPath, err := os.MkdirTemp("", "mole-wt-*")
	if err != nil {
		return "", false, fmt.Errorf("creating temp dir: %w", err)
	}

	// Remove the temp dir so git worktree add can create it
	os.Remove(wtPath)

	if err := rm.gitCmd(ctx, repoDir, "worktree", "add", wtPath, "origin/"+branch); err != nil {
		os.RemoveAll(wtPath)
		return "", false, fmt.Errorf("worktree add: %w", err)
	}

	return wtPath, firstClone, nil
}

// Cleanup removes a worktree directory and prunes the git worktree list.
func (rm *RepoManager) Cleanup(worktreePath string) {
	if worktreePath == "" {
		return
	}
	if err := os.RemoveAll(worktreePath); err != nil {
		slog.Warn("failed to remove worktree dir", "path", worktreePath, "error", err)
	}
}

// CleanupStale removes orphaned worktree directories left by crashed reviews.
// Should be called at startup.
func (rm *RepoManager) CleanupStale() {
	if rm.basePath == "" {
		return
	}

	matches, _ := filepath.Glob(filepath.Join(os.TempDir(), "mole-wt-*"))
	for _, m := range matches {
		slog.Info("cleaning up stale worktree", "path", m)
		os.RemoveAll(m)
	}

	// Prune worktree references in all repos
	owners, _ := os.ReadDir(rm.basePath)
	for _, owner := range owners {
		if !owner.IsDir() {
			continue
		}
		repos, _ := os.ReadDir(filepath.Join(rm.basePath, owner.Name()))
		for _, repo := range repos {
			if !repo.IsDir() {
				continue
			}
			repoDir := filepath.Join(rm.basePath, owner.Name(), repo.Name())
			_ = rm.gitCmd(context.Background(), repoDir, "worktree", "prune")
		}
	}
}

func (rm *RepoManager) repoMutex(repo string) *sync.Mutex {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if mu, ok := rm.repoLocks[repo]; ok {
		return mu
	}
	mu := &sync.Mutex{}
	rm.repoLocks[repo] = mu
	return mu
}

func (rm *RepoManager) ensureCloned(ctx context.Context, repoDir, repo string, installID int64) (firstClone bool, err error) {
	if _, err := os.Stat(repoDir); err == nil {
		// Repo exists, fetch updates
		if ferr := rm.fetchWithAuth(ctx, repoDir, repo, installID); ferr != nil {
			return false, ferr
		}
		return false, nil
	}

	// First clone
	if err := rm.cloneWithAuth(ctx, repoDir, repo, installID); err != nil {
		return false, err
	}
	return true, nil
}

func (rm *RepoManager) cloneWithAuth(ctx context.Context, repoDir, repo string, installID int64) error {
	token, err := rm.tokenFunc(ctx, installID)
	if err != nil {
		return fmt.Errorf("generating token: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(repoDir), 0o755); err != nil {
		return fmt.Errorf("creating parent dir: %w", err)
	}

	url := fmt.Sprintf("https://github.com/%s.git", repo)
	cmd := exec.CommandContext(ctx, "git", "clone", "--bare", url, repoDir)
	cmd.Env = rm.authEnv(token)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone: %s: %w", strings.TrimSpace(string(out)), err)
	}

	slog.Info("repository cloned", "repo", repo, "path", repoDir)
	return nil
}

func (rm *RepoManager) fetchWithAuth(ctx context.Context, repoDir, repo string, installID int64) error {
	token, err := rm.tokenFunc(ctx, installID)
	if err != nil {
		return fmt.Errorf("generating token: %w", err)
	}

	cmd := exec.CommandContext(ctx, "git", "-C", repoDir, "fetch", "--all", "--prune")
	cmd.Env = rm.authEnv(token)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git fetch: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

func (rm *RepoManager) gitCmd(ctx context.Context, repoDir string, args ...string) error {
	fullArgs := append([]string{"-C", repoDir}, args...)
	cmd := exec.CommandContext(ctx, "git", fullArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git %s: %s: %w", args[0], strings.TrimSpace(string(out)), err)
	}
	return nil
}

// authEnv returns environment variables that configure git to use a token
// for authentication via credential helper, avoiding the token in CLI args.
func (rm *RepoManager) authEnv(token string) []string {
	helper := fmt.Sprintf("!f() { echo username=x-access-token; echo password=%s; }; f", token)

	env := os.Environ()
	env = append(env,
		"GIT_TERMINAL_PROMPT=0",
		fmt.Sprintf("GIT_CONFIG_COUNT=%d", 1),
		"GIT_CONFIG_KEY_0=credential.helper",
		fmt.Sprintf("GIT_CONFIG_VALUE_0=%s", helper),
	)
	return env
}
