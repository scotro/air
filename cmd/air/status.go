package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check status of running agents",
	RunE:  runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	worktreesDir := filepath.Join(".air", "worktrees")

	entries, err := os.ReadDir(worktreesDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No active agents. Run 'air run <packets>' to start.")
			return nil
		}
		return fmt.Errorf("failed to read worktrees: %w", err)
	}

	if len(entries) == 0 {
		fmt.Println("No active agents. Run 'air run <packets>' to start.")
		return nil
	}

	fmt.Println("Agent Status")
	fmt.Println("============")
	fmt.Println()

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		wtPath := filepath.Join(worktreesDir, name)

		// Get branch name
		branchCmd := exec.Command("git", "-C", wtPath, "rev-parse", "--abbrev-ref", "HEAD")
		branchOut, _ := branchCmd.Output()
		branch := strings.TrimSpace(string(branchOut))

		// Get last commit
		logCmd := exec.Command("git", "-C", wtPath, "log", "-1", "--format=%ar: %s")
		logOut, _ := logCmd.Output()
		lastCommit := strings.TrimSpace(string(logOut))

		// Check if claude is running in this worktree
		status := "idle"
		pgrepCmd := exec.Command("pgrep", "-f", "claude.*"+wtPath)
		if err := pgrepCmd.Run(); err == nil {
			status = "running"
		}

		// Get uncommitted changes count
		diffCmd := exec.Command("git", "-C", wtPath, "status", "--porcelain")
		var diffOut bytes.Buffer
		diffCmd.Stdout = &diffOut
		diffCmd.Run()
		changes := len(strings.Split(strings.TrimSpace(diffOut.String()), "\n"))
		if diffOut.Len() == 0 {
			changes = 0
		}

		// Print status
		statusIcon := "⚪"
		if status == "running" {
			statusIcon = "🟢"
		}

		fmt.Printf("%s %s\n", statusIcon, name)
		fmt.Printf("   Branch: %s\n", branch)
		fmt.Printf("   Last commit: %s\n", lastCommit)
		if changes > 0 {
			fmt.Printf("   Uncommitted: %d files\n", changes)
		}
		fmt.Println()
	}

	return nil
}
