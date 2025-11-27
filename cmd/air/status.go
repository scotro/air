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
	channelsDir := filepath.Join(".air", "channels")

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

	// Collect done agents (check both done/<name>.json and <name>.json patterns)
	doneAgents := make(map[string]bool)
	doneDir := filepath.Join(channelsDir, "done")
	if doneEntries, err := os.ReadDir(doneDir); err == nil {
		for _, de := range doneEntries {
			if strings.HasSuffix(de.Name(), ".json") {
				doneAgents[strings.TrimSuffix(de.Name(), ".json")] = true
			}
		}
	}
	// Also check for done signals at root level (fallback for older format)
	if channelEntries, err := os.ReadDir(channelsDir); err == nil {
		for _, ce := range channelEntries {
			if ce.IsDir() {
				continue
			}
			name := strings.TrimSuffix(ce.Name(), ".json")
			// Check if this matches a worktree name (likely a done signal)
			for _, entry := range entries {
				if entry.Name() == name {
					doneAgents[name] = true
				}
			}
		}
	}

	fmt.Println("Agents")
	fmt.Println()

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		wtPath := filepath.Join(worktreesDir, name)

		// Get last commit
		logCmd := exec.Command("git", "-C", wtPath, "log", "-1", "--format=%s (%ar)")
		logOut, _ := logCmd.Output()
		lastCommit := strings.TrimSpace(string(logOut))

		// Check if claude is running in this worktree
		isRunning := false
		pgrepCmd := exec.Command("pgrep", "-f", "claude.*"+wtPath)
		if err := pgrepCmd.Run(); err == nil {
			isRunning = true
		}

		// Get uncommitted changes count
		diffCmd := exec.Command("git", "-C", wtPath, "status", "--porcelain")
		var diffOut bytes.Buffer
		diffCmd.Stdout = &diffOut
		diffCmd.Run()
		changes := 0
		if diffOut.Len() > 0 {
			changes = len(strings.Split(strings.TrimSpace(diffOut.String()), "\n"))
		}

		// Determine status
		isDone := doneAgents[name]

		var statusIcon, statusText string
		if isDone {
			statusIcon = "✓"
			statusText = "done"
		} else {
			// Show all non-done agents as "running" - we can't reliably detect
			// if an agent is waiting for user input vs actively working
			statusIcon = "●"
			statusText = "running"
		}
		_ = isRunning // still used for potential future features

		// Build info line
		info := lastCommit
		if changes > 0 {
			info += fmt.Sprintf(", %d uncommitted", changes)
		}

		fmt.Printf("  %s %-16s %s\n", statusIcon, name, statusText)
		fmt.Printf("    %s\n", info)
	}

	// Show coordination channels (exclude done markers)
	if err := showChannelStatus(doneAgents); err != nil {
		return nil
	}

	return nil
}

func showChannelStatus(doneAgents map[string]bool) error {
	channelsDir := filepath.Join(".air", "channels")

	entries, err := os.ReadDir(channelsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	// Collect coordination channels (exclude done markers and agent-named files)
	var channels []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".json") {
			name := strings.TrimSuffix(entry.Name(), ".json")
			// Skip if this is a done marker (matches an agent name)
			if doneAgents[name] {
				continue
			}
			channels = append(channels, name)
		}
	}

	if len(channels) == 0 {
		return nil
	}

	fmt.Println()
	fmt.Println("Channels")
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

		fmt.Printf("  ✓ %-16s signaled by %s (%s)\n", ch, payload.Agent, shortSHA)
	}

	return nil
}
