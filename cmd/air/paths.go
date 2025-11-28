package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// Mode represents the Air operating mode
type Mode string

const (
	// ModeSingle is the traditional single-repo mode
	ModeSingle Mode = "single"
	// ModeWorkspace is multi-repo workspace mode
	ModeWorkspace Mode = "workspace"
)

// WorkspaceInfo holds information about the current workspace
type WorkspaceInfo struct {
	Mode  Mode     // Operating mode (single or workspace)
	Name  string   // Project/workspace name (directory basename)
	Root  string   // Absolute path to workspace root (cwd)
	Repos []string // List of repo names (empty for single mode, populated for workspace mode)
}

// detectMode determines the Air operating mode based on the current directory.
// - If cwd is a git repo → single mode
// - If cwd is NOT a git repo but has git repo children → workspace mode
// - Otherwise → error
func detectMode() (*WorkspaceInfo, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	name := filepath.Base(cwd)

	// Check if cwd is a git repo
	gitDir := filepath.Join(cwd, ".git")
	if stat, err := os.Stat(gitDir); err == nil && stat.IsDir() {
		return &WorkspaceInfo{
			Mode:  ModeSingle,
			Name:  name,
			Root:  cwd,
			Repos: nil,
		}, nil
	}

	// Check for git repo children
	repos, err := findChildRepos(cwd)
	if err != nil {
		return nil, err
	}

	if len(repos) > 0 {
		return &WorkspaceInfo{
			Mode:  ModeWorkspace,
			Name:  name,
			Root:  cwd,
			Repos: repos,
		}, nil
	}

	return nil, fmt.Errorf("not a git repo and no git repo children found in %s", cwd)
}

// findChildRepos returns a sorted list of immediate child directories that are git repos
func findChildRepos(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var repos []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		// Skip hidden directories
		if e.Name()[0] == '.' {
			continue
		}
		childPath := filepath.Join(dir, e.Name())
		gitDir := filepath.Join(childPath, ".git")
		if stat, err := os.Stat(gitDir); err == nil && stat.IsDir() {
			repos = append(repos, e.Name())
		}
	}

	sort.Strings(repos)
	return repos, nil
}

// getRepoPath returns the absolute path to a repo within the workspace.
// In single mode, returns the workspace root.
// In workspace mode, returns the path to the named repo.
func (w *WorkspaceInfo) getRepoPath(repoName string) (string, error) {
	if w.Mode == ModeSingle {
		if repoName != "" && repoName != w.Name {
			return "", fmt.Errorf("in single-repo mode, cannot reference repo %q", repoName)
		}
		return w.Root, nil
	}

	// Workspace mode: validate repo exists
	for _, r := range w.Repos {
		if r == repoName {
			return filepath.Join(w.Root, repoName), nil
		}
	}
	return "", fmt.Errorf("repo %q not found in workspace (available: %v)", repoName, w.Repos)
}

// getAirDirForWorkspace returns the air directory for this workspace: ~/.air/<name>/
func (w *WorkspaceInfo) getAirDirForWorkspace() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".air", w.Name), nil
}

// getWorktreePath returns the worktree path for a plan.
// In single mode: ~/.air/<project>/worktrees/<plan>/
// In workspace mode: ~/.air/<workspace>/worktrees/<repo>/<plan>/
func (w *WorkspaceInfo) getWorktreePath(repoName, planName string) (string, error) {
	airDir, err := w.getAirDirForWorkspace()
	if err != nil {
		return "", err
	}

	if w.Mode == ModeSingle {
		return filepath.Join(airDir, "worktrees", planName), nil
	}

	// Workspace mode: include repo in path
	if repoName == "" {
		return "", fmt.Errorf("repo name required in workspace mode")
	}
	return filepath.Join(airDir, "worktrees", repoName, planName), nil
}

// getProjectName returns the basename of the current working directory.
// This is used as the project identifier in ~/.air/<project>/
func getProjectName() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Base(cwd), nil
}

// getAirDir returns the air directory for the current project: ~/.air/<project>/
func getAirDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	project, err := getProjectName()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".air", project), nil
}

// mustGetAirDir returns the air directory or panics. Use only when error handling
// has already been done (e.g., after isInitialized check).
func mustGetAirDir() string {
	dir, err := getAirDir()
	if err != nil {
		panic(err)
	}
	return dir
}

// getPlansDir returns ~/.air/<project>/plans/
func getPlansDir() string {
	return filepath.Join(mustGetAirDir(), "plans")
}

// getWorktreesDir returns ~/.air/<project>/worktrees/
func getWorktreesDir() string {
	return filepath.Join(mustGetAirDir(), "worktrees")
}

// getAgentsDir returns ~/.air/<project>/agents/
func getAgentsDir() string {
	return filepath.Join(mustGetAirDir(), "agents")
}

// getChannelsDir returns the channels directory.
// For agent commands (with AIR_CHANNELS_DIR set), returns the env var value.
// For main project commands, computes ~/.air/<project>/channels/
func getChannelsDir() string {
	// Agent context: use env var
	if dir := os.Getenv("AIR_CHANNELS_DIR"); dir != "" {
		return dir
	}
	// Main project context: compute from project name
	return filepath.Join(mustGetAirDir(), "channels")
}

// getContextPath returns ~/.air/<project>/context.md
func getContextPath() string {
	return filepath.Join(mustGetAirDir(), "context.md")
}

// isInitialized checks if the air directory exists for the current project.
func isInitialized() bool {
	dir, err := getAirDir()
	if err != nil {
		return false
	}
	_, err = os.Stat(dir)
	return err == nil
}
