package main

import (
	"bytes"
	"encoding/json"
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
			fmt.Println("No active agents. Run 'air run <plans>' to start.")
			return nil
		}
		return fmt.Errorf("failed to read worktrees: %w", err)
	}

	if len(entries) == 0 {
		fmt.Println("No active agents. Run 'air run <plans>' to start.")
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

		// Check if this agent has signaled done
		doneChannelPath := filepath.Join(".air", "channels", "done", name+".json")
		if _, err := os.Stat(doneChannelPath); err == nil {
			fmt.Printf("   ✓ Completed\n")
		}
		fmt.Println()
	}

	// Show channel status
	if err := showChannelStatus(); err != nil {
		// Non-fatal - just skip channel status if channels dir doesn't exist
		return nil
	}

	return nil
}

func showChannelStatus() error {
	channelsDir := filepath.Join(".air", "channels")

	entries, err := os.ReadDir(channelsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No channels yet, that's fine
		}
		return err
	}

	// Collect channels (excluding done/ subdirectory)
	var channels []string
	var doneAgents []string

	for _, entry := range entries {
		if entry.IsDir() {
			if entry.Name() == "done" {
				// Read done subdirectory
				doneDir := filepath.Join(channelsDir, "done")
				doneEntries, _ := os.ReadDir(doneDir)
				for _, de := range doneEntries {
					if strings.HasSuffix(de.Name(), ".json") {
						agentName := strings.TrimSuffix(de.Name(), ".json")
						doneAgents = append(doneAgents, agentName)
					}
				}
			}
			continue
		}
		if strings.HasSuffix(entry.Name(), ".json") {
			channels = append(channels, strings.TrimSuffix(entry.Name(), ".json"))
		}
	}

	if len(channels) == 0 && len(doneAgents) == 0 {
		return nil
	}

	fmt.Println("Channels")
	fmt.Println("========")
	fmt.Println()

	for _, ch := range channels {
		channelPath := filepath.Join(channelsDir, ch+".json")
		data, err := os.ReadFile(channelPath)
		if err != nil {
			continue
		}

		var payload ChannelPayload
		if err := json.Unmarshal(data, &payload); err != nil {
			continue
		}

		shortSHA := payload.SHA
		if len(shortSHA) > 8 {
			shortSHA = shortSHA[:8]
		}

		fmt.Printf("✓ %s\n", ch)
		fmt.Printf("   Signaled by: %s (sha: %s)\n", payload.Agent, shortSHA)
		fmt.Println()
	}

	if len(doneAgents) > 0 {
		fmt.Println("Completed Agents")
		fmt.Println("================")
		fmt.Println()
		for _, agent := range doneAgents {
			fmt.Printf("✓ %s\n", agent)
		}
		fmt.Println()
	}

	return nil
}
