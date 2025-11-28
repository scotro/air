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
	Long: `Remove worktrees and optionally delete their branches.

With no arguments, removes all worktrees.
With arguments, removes only the specified worktrees.`,
	RunE: runClean,
}

var cleanAll bool

func init() {
	cleanCmd.Flags().BoolVar(&cleanAll, "branches", false, "Also delete air/* branches")
}

// cleanOptions controls the behavior of cleanWorkspace
type cleanOptions struct {
	deleteBranches bool // delete git branches (vs leave them)
	deletePlans    bool // delete plans entirely (vs archive them)
	quiet          bool // minimal output
	cleanAll       bool // cleaning all items (vs specific names)
}

// cleanWorkspace performs the actual cleanup of worktrees, channels, agents, plans, and branches.
// This is the shared implementation used by both `air clean` and `air plan` (start fresh).
func cleanWorkspace(names []string, opts cleanOptions) error {
	worktreesDir := getWorktreesDir()

	// Remove worktrees
	for _, name := range names {
		wtPath := filepath.Join(worktreesDir, name)

		// Check if worktree exists before trying to remove
		if _, err := os.Stat(wtPath); os.IsNotExist(err) {
			continue
		}

		removeCmd := exec.Command("git", "worktree", "remove", wtPath, "--force")
		if !opts.quiet {
			removeCmd.Stdout = os.Stdout
			removeCmd.Stderr = os.Stderr
		}

		if err := removeCmd.Run(); err != nil {
			if !opts.quiet {
				fmt.Printf("Warning: failed to remove worktree %s: %v\n", name, err)
			}
			// Try to remove directory directly
			os.RemoveAll(wtPath)
		} else if !opts.quiet {
			fmt.Printf("Removed worktree: %s\n", name)
		}
	}

	// Prune worktrees
	exec.Command("git", "worktree", "prune").Run()

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
	if opts.deletePlans {
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
		for _, name := range names {
			branch := "air/" + name
			deleteCmd := exec.Command("git", "branch", "-D", branch)
			if err := deleteCmd.Run(); err != nil {
				if !opts.quiet {
					fmt.Printf("Warning: failed to delete branch %s\n", branch)
				}
			} else if !opts.quiet {
				fmt.Printf("Deleted branch: %s\n", branch)
			}
		}
	}

	return nil
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
	worktreesDir := getWorktreesDir()

	// Get all existing worktrees
	entries, err := os.ReadDir(worktreesDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No worktrees to clean.")
			return nil
		}
		return fmt.Errorf("failed to read worktrees: %w", err)
	}

	// Build set of existing worktrees
	existing := make(map[string]bool)
	for _, entry := range entries {
		if entry.IsDir() {
			existing[entry.Name()] = true
		}
	}

	if len(existing) == 0 {
		fmt.Println("No worktrees to clean.")
		return nil
	}

	// Determine which worktrees to clean
	var names []string
	isCleanAll := len(args) == 0
	if !isCleanAll {
		// Clean specific worktrees
		for _, name := range args {
			if !existing[name] {
				return fmt.Errorf("worktree '%s' not found", name)
			}
			names = append(names, name)
		}
	} else {
		// Clean all worktrees
		for name := range existing {
			names = append(names, name)
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

	// Perform cleanup
	err = cleanWorkspace(names, cleanOptions{
		deleteBranches: deleteBranches,
		deletePlans:    false, // archive, don't delete
		quiet:          false,
		cleanAll:       isCleanAll,
	})
	if err != nil {
		return err
	}

	fmt.Println("\nCleanup complete.")
	return nil
}
