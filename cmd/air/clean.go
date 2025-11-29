package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var cleanCmd = &cobra.Command{
	Use:   "clean [names...]",
	Short: "Remove worktrees and optionally delete branches",
	Long: `Remove worktrees, kill the tmux session, and optionally delete branches.

With no arguments, removes all worktrees.
With arguments, removes only the specified worktrees.

By default, plans are archived. Use --keep-plans to preserve them for rerunning
after error recovery.`,
	RunE: runClean,
}

var cleanAll bool
var keepPlans bool

func init() {
	cleanCmd.Flags().BoolVar(&cleanAll, "branches", false, "Also delete air/* branches")
	cleanCmd.Flags().BoolVar(&keepPlans, "keep-plans", false, "Keep plans for rerunning (don't archive)")
}

// worktreeInfo holds info about a worktree for cleanup
type worktreeInfo struct {
	name     string // plan name
	repoName string // repo name (empty for single mode)
	repoPath string // path to repo (for git commands)
	wtPath   string // full worktree path
}

// cleanOptions controls the behavior of cleanWorkspace
type cleanOptions struct {
	deleteBranches bool // delete git branches (vs leave them)
	deletePlans    bool // delete plans entirely (vs archive them)
	keepPlans      bool // keep plans in place (don't archive or delete)
	quiet          bool // minimal output
	cleanAll       bool // cleaning all items (vs specific names)
}

// cleanWorkspaceWorktrees performs the actual cleanup of worktrees, channels, agents, plans, and branches.
// This is the shared implementation used by both `air clean` and `air plan` (start fresh).
// For workspace mode, pass worktreeInfo with repoPath set; for single mode, repoPath can be empty.
func cleanWorkspaceWorktrees(worktrees []worktreeInfo, opts cleanOptions) error {
	// Remove worktrees
	for _, wt := range worktrees {
		// Check if worktree exists before trying to remove
		if _, err := os.Stat(wt.wtPath); os.IsNotExist(err) {
			continue
		}

		// Run git worktree remove from the correct repo
		removeCmd := exec.Command("git", "worktree", "remove", wt.wtPath, "--force")
		if wt.repoPath != "" {
			removeCmd.Dir = wt.repoPath
		}
		if !opts.quiet {
			removeCmd.Stdout = os.Stdout
			removeCmd.Stderr = os.Stderr
		}

		label := wt.name
		if wt.repoName != "" {
			label = fmt.Sprintf("%s [%s]", wt.name, wt.repoName)
		}

		if err := removeCmd.Run(); err != nil {
			if !opts.quiet {
				fmt.Printf("Warning: failed to remove worktree %s: %v\n", label, err)
			}
			// Try to remove directory directly
			os.RemoveAll(wt.wtPath)
		} else if !opts.quiet {
			fmt.Printf("Removed worktree: %s\n", label)
		}
	}

	// Prune worktrees in all repos
	prunedRepos := make(map[string]bool)
	for _, wt := range worktrees {
		repoPath := wt.repoPath
		if repoPath == "" {
			repoPath = "."
		}
		if !prunedRepos[repoPath] {
			pruneCmd := exec.Command("git", "worktree", "prune")
			pruneCmd.Dir = repoPath
			pruneCmd.Run()
			prunedRepos[repoPath] = true
		}
	}

	// Collect names for channel/agent/plan cleanup
	names := make([]string, len(worktrees))
	for i, wt := range worktrees {
		names[i] = wt.name
	}

	// Clean up channels and agent data
	channelsDir := getChannelsDir()
	agentsDir := getAgentsDir()
	if opts.cleanAll {
		// Cleaning all - remove entire channels and agents directories
		if err := os.RemoveAll(channelsDir); err != nil {
			if !opts.quiet {
				fmt.Printf("Warning: failed to remove channels directory: %v\n", err)
			}
		} else if !opts.quiet {
			fmt.Println("Cleared channels directory")
		}
		if err := os.RemoveAll(agentsDir); err != nil {
			if !opts.quiet {
				fmt.Printf("Warning: failed to remove agents directory: %v\n", err)
			}
		} else if !opts.quiet {
			fmt.Println("Cleared agents directory")
		}
	} else {
		// Cleaning specific items - remove their done/<name>.json and agent data
		for _, name := range names {
			doneFile := filepath.Join(channelsDir, "done", name+".json")
			if err := os.Remove(doneFile); err == nil && !opts.quiet {
				fmt.Printf("Removed done channel: %s\n", name)
			}
			agentDir := filepath.Join(agentsDir, name)
			if err := os.RemoveAll(agentDir); err == nil && !opts.quiet {
				fmt.Printf("Removed agent data: %s\n", name)
			}
		}
	}

	// Handle plans
	plansDir := getPlansDir()
	if opts.keepPlans {
		// Keep plans in place (for error recovery / rerun)
		if !opts.quiet {
			fmt.Println("Plans preserved for rerun")
		}
	} else if opts.deletePlans {
		// Delete plans entirely
		for _, name := range names {
			planFile := filepath.Join(plansDir, name+".md")
			if err := os.Remove(planFile); err != nil {
				if !os.IsNotExist(err) && !opts.quiet {
					fmt.Printf("Warning: failed to delete plan %s: %v\n", name, err)
				}
			} else if !opts.quiet {
				fmt.Printf("Deleted plan: %s\n", name)
			}
		}
	} else {
		// Archive plans
		archivedDir := filepath.Join(plansDir, "archive")
		if err := os.MkdirAll(archivedDir, 0755); err != nil {
			return fmt.Errorf("failed to create archive directory: %w", err)
		}

		for _, name := range names {
			planFile := filepath.Join(plansDir, name+".md")
			archivedFile := filepath.Join(archivedDir, name+".md")

			if err := os.Rename(planFile, archivedFile); err != nil {
				if !os.IsNotExist(err) && !opts.quiet {
					fmt.Printf("Warning: failed to archive plan %s: %v\n", name, err)
				}
			} else if !opts.quiet {
				fmt.Printf("Archived plan: %s\n", name)
			}
		}
	}

	// Delete branches if requested
	if opts.deleteBranches {
		if !opts.quiet {
			fmt.Println("\nDeleting branches...")
		}
		for _, wt := range worktrees {
			branch := "air/" + wt.name
			deleteCmd := exec.Command("git", "branch", "-D", branch)
			if wt.repoPath != "" {
				deleteCmd.Dir = wt.repoPath
			}

			label := branch
			if wt.repoName != "" {
				label = fmt.Sprintf("%s [%s]", branch, wt.repoName)
			}

			if err := deleteCmd.Run(); err != nil {
				if !opts.quiet {
					fmt.Printf("Warning: failed to delete branch %s\n", label)
				}
			} else if !opts.quiet {
				fmt.Printf("Deleted branch: %s\n", label)
			}
		}
	}

	return nil
}

// cleanWorkspace is the legacy interface for single-repo mode cleanup.
// Kept for backward compatibility with existing callers.
func cleanWorkspace(names []string, opts cleanOptions) error {
	worktreesDir := getWorktreesDir()
	worktrees := make([]worktreeInfo, len(names))
	for i, name := range names {
		worktrees[i] = worktreeInfo{
			name:   name,
			wtPath: filepath.Join(worktreesDir, name),
		}
	}
	return cleanWorkspaceWorktrees(worktrees, opts)
}

// getExistingWorktrees returns the names of existing worktrees
func getExistingWorktrees() []string {
	worktreesDir := getWorktreesDir()
	entries, err := os.ReadDir(worktreesDir)
	if err != nil {
		return nil
	}

	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	return names
}

// getExistingPlans returns the names of existing plans (excluding archive/)
func getExistingPlans() []string {
	plansDir := getPlansDir()
	entries, err := os.ReadDir(plansDir)
	if err != nil {
		return nil
	}

	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			name := strings.TrimSuffix(entry.Name(), ".md")
			names = append(names, name)
		}
	}
	return names
}

func runClean(cmd *cobra.Command, args []string) error {
	// Detect mode
	info, err := detectMode()
	if err != nil {
		return fmt.Errorf("failed to detect mode: %w", err)
	}

	worktreesDir := getWorktreesDir()

	// Collect worktrees based on mode
	var worktrees []worktreeInfo
	existing := make(map[string]worktreeInfo)

	if info.Mode == ModeWorkspace {
		// Workspace mode: worktrees/<repo>/<plan>/
		repoEntries, err := os.ReadDir(worktreesDir)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Println("No worktrees to clean.")
				return nil
			}
			return fmt.Errorf("failed to read worktrees: %w", err)
		}

		for _, repoEntry := range repoEntries {
			if !repoEntry.IsDir() {
				continue
			}
			repoName := repoEntry.Name()
			repoPath := filepath.Join(info.Root, repoName)
			repoWorktreeDir := filepath.Join(worktreesDir, repoName)

			planEntries, err := os.ReadDir(repoWorktreeDir)
			if err != nil {
				continue
			}

			for _, planEntry := range planEntries {
				if !planEntry.IsDir() {
					continue
				}
				wt := worktreeInfo{
					name:     planEntry.Name(),
					repoName: repoName,
					repoPath: repoPath,
					wtPath:   filepath.Join(repoWorktreeDir, planEntry.Name()),
				}
				worktrees = append(worktrees, wt)
				existing[planEntry.Name()] = wt
			}
		}
	} else {
		// Single mode: worktrees/<plan>/
		entries, err := os.ReadDir(worktreesDir)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Println("No worktrees to clean.")
				return nil
			}
			return fmt.Errorf("failed to read worktrees: %w", err)
		}

		for _, entry := range entries {
			if entry.IsDir() {
				wt := worktreeInfo{
					name:     entry.Name(),
					repoPath: info.Root,
					wtPath:   filepath.Join(worktreesDir, entry.Name()),
				}
				worktrees = append(worktrees, wt)
				existing[entry.Name()] = wt
			}
		}
	}

	if len(worktrees) == 0 {
		fmt.Println("No worktrees to clean.")
		return nil
	}

	// Determine which worktrees to clean
	var toClean []worktreeInfo
	isCleanAll := len(args) == 0
	if !isCleanAll {
		// Clean specific worktrees
		for _, name := range args {
			wt, ok := existing[name]
			if !ok {
				return fmt.Errorf("worktree '%s' not found", name)
			}
			toClean = append(toClean, wt)
		}
	} else {
		// Clean all worktrees
		toClean = worktrees
	}

	// Show what will be cleaned
	if info.Mode == ModeWorkspace {
		fmt.Printf("Workspace: %s\n\n", info.Name)
	}
	fmt.Println("Worktrees to clean:")
	for _, wt := range toClean {
		if wt.repoName != "" {
			fmt.Printf("  %s [%s]\n", wt.name, wt.repoName)
		} else {
			fmt.Printf("  %s\n", wt.name)
		}
	}

	// Determine if we should delete branches
	deleteBranches := cleanAll
	if !cleanAll {
		// Ask about branches
		fmt.Print("\nDelete air/* branches? [y/N] ")
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))
		deleteBranches = response == "y" || response == "yes"
	}

	// Kill tmux session if it exists
	if err := exec.Command("tmux", "kill-session", "-t", "air").Run(); err == nil {
		fmt.Println("Killed tmux session: air")
	}

	// Perform cleanup
	err = cleanWorkspaceWorktrees(toClean, cleanOptions{
		deleteBranches: deleteBranches,
		deletePlans:    false, // archive, don't delete
		keepPlans:      keepPlans,
		quiet:          false,
		cleanAll:       isCleanAll,
	})
	if err != nil {
		return err
	}

	fmt.Println("\nCleanup complete.")
	return nil
}
