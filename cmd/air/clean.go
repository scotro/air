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
	Use:   "clean",
	Short: "Remove worktrees and optionally delete branches",
	RunE:  runClean,
}

var cleanAll bool

func init() {
	cleanCmd.Flags().BoolVar(&cleanAll, "branches", false, "Also delete air/* branches")
}

func runClean(cmd *cobra.Command, args []string) error {
	worktreesDir := filepath.Join(".air", "worktrees")

	entries, err := os.ReadDir(worktreesDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No worktrees to clean.")
			return nil
		}
		return fmt.Errorf("failed to read worktrees: %w", err)
	}

	if len(entries) == 0 {
		fmt.Println("No worktrees to clean.")
		return nil
	}

	// Collect worktree names before removing
	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
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

	fmt.Println("\nCleanup complete. Packets preserved in .air/packets/")
	return nil
}
