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

func runClean(cmd *cobra.Command, args []string) error {
	worktreesDir := filepath.Join(".air", "worktrees")

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
	if len(args) > 0 {
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

	// Remove worktrees
	for _, name := range names {
		wtPath := filepath.Join(worktreesDir, name)

		removeCmd := exec.Command("git", "worktree", "remove", wtPath, "--force")
		removeCmd.Stdout = os.Stdout
		removeCmd.Stderr = os.Stderr

		if err := removeCmd.Run(); err != nil {
			fmt.Printf("Warning: failed to remove worktree %s: %v\n", name, err)
			// Try to remove directory directly
			os.RemoveAll(wtPath)
		} else {
			fmt.Printf("Removed worktree: %s\n", name)
		}
	}

	// Prune worktrees
	exec.Command("git", "worktree", "prune").Run()

	// Archive plans
	archivedDir := filepath.Join(".air", "plans", "archive")
	if err := os.MkdirAll(archivedDir, 0755); err != nil {
		return fmt.Errorf("failed to create archive directory: %w", err)
	}

	for _, name := range names {
		planFile := filepath.Join(".air", "plans", name+".md")
		archivedFile := filepath.Join(archivedDir, name+".md")

		if err := os.Rename(planFile, archivedFile); err != nil {
			if !os.IsNotExist(err) {
				fmt.Printf("Warning: failed to archive plan %s: %v\n", name, err)
			}
		} else {
			fmt.Printf("Archived plan: %s\n", name)
		}
	}

	// Delete branches if requested
	if cleanAll {
		fmt.Println("\nDeleting branches...")
		for _, name := range names {
			branch := "air/" + name
			deleteCmd := exec.Command("git", "branch", "-D", branch)
			if err := deleteCmd.Run(); err != nil {
				fmt.Printf("Warning: failed to delete branch %s\n", branch)
			} else {
				fmt.Printf("Deleted branch: %s\n", branch)
			}
		}
	} else {
		// Ask about branches
		fmt.Print("\nDelete air/* branches? [y/N] ")
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		if response == "y" || response == "yes" {
			for _, name := range names {
				branch := "air/" + name
				deleteCmd := exec.Command("git", "branch", "-D", branch)
				if err := deleteCmd.Run(); err != nil {
					fmt.Printf("Warning: failed to delete branch %s\n", branch)
				} else {
					fmt.Printf("Deleted branch: %s\n", branch)
				}
			}
		}
	}

	fmt.Println("\nCleanup complete.")
	return nil
}
