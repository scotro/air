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
	// Detect mode
	info, err := detectMode()
	if err != nil {
		return fmt.Errorf("failed to detect mode: %w", err)
	}

	worktreesDir := getWorktreesDir()
	channelsDir := getChannelsDir()

	// Collect done agents
	doneAgents := make(map[string]bool)
	doneDir := filepath.Join(channelsDir, "done")
	if doneEntries, err := os.ReadDir(doneDir); err == nil {
		for _, de := range doneEntries {
			if strings.HasSuffix(de.Name(), ".json") {
				doneAgents[strings.TrimSuffix(de.Name(), ".json")] = true
			}
		}
	}

	// Collect agents based on mode
	type agentStatus struct {
		name     string
		repoName string // only in workspace mode
		wtPath   string
	}
	var agents []agentStatus

	if info.Mode == ModeWorkspace {
		// Workspace mode: worktrees/<repo>/<plan>/
		repoEntries, err := os.ReadDir(worktreesDir)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Println("No active agents. Run 'air run' to start.")
				return nil
			}
			return fmt.Errorf("failed to read worktrees: %w", err)
		}

		for _, repoEntry := range repoEntries {
			if !repoEntry.IsDir() {
				continue
			}
			repoName := repoEntry.Name()
			repoWorktreeDir := filepath.Join(worktreesDir, repoName)

			planEntries, err := os.ReadDir(repoWorktreeDir)
			if err != nil {
				continue
			}
			for _, planEntry := range planEntries {
				if !planEntry.IsDir() {
					continue
				}
				agents = append(agents, agentStatus{
					name:     planEntry.Name(),
					repoName: repoName,
					wtPath:   filepath.Join(repoWorktreeDir, planEntry.Name()),
				})
			}
		}
	} else {
		// Single mode: worktrees/<plan>/
		entries, err := os.ReadDir(worktreesDir)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Println("No active agents. Run 'air run <plans>' to start.")
				return nil
			}
			return fmt.Errorf("failed to read worktrees: %w", err)
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			agents = append(agents, agentStatus{
				name:   entry.Name(),
				wtPath: filepath.Join(worktreesDir, entry.Name()),
			})
		}
	}

	if len(agents) == 0 {
		fmt.Println("No active agents. Run 'air run' to start.")
		return nil
	}

	// Print header
	if info.Mode == ModeWorkspace {
		fmt.Printf("Workspace: %s\n\n", info.Name)
	}
	fmt.Println("Agents")
	fmt.Println()

	for _, agent := range agents {
		// Get last commit
		logCmd := exec.Command("git", "-C", agent.wtPath, "log", "-1", "--format=%s (%ar)")
		logOut, _ := logCmd.Output()
		lastCommit := strings.TrimSpace(string(logOut))

		// Get uncommitted changes count
		diffCmd := exec.Command("git", "-C", agent.wtPath, "status", "--porcelain")
		var diffOut bytes.Buffer
		diffCmd.Stdout = &diffOut
		diffCmd.Run()
		changes := 0
		if diffOut.Len() > 0 {
			changes = len(strings.Split(strings.TrimSpace(diffOut.String()), "\n"))
		}

		// Determine status
		isDone := doneAgents[agent.name]

		var statusIcon, statusText string
		if isDone {
			statusIcon = "✓"
			statusText = "done"
		} else {
			statusIcon = "●"
			statusText = "running"
		}

		// Build info line
		agentLabel := agent.name
		if info.Mode == ModeWorkspace && agent.repoName != "" {
			agentLabel = fmt.Sprintf("%s [%s]", agent.name, agent.repoName)
		}

		infoLine := lastCommit
		if changes > 0 {
			infoLine += fmt.Sprintf(", %d uncommitted", changes)
		}

		fmt.Printf("  %s %-24s %s\n", statusIcon, agentLabel, statusText)
		fmt.Printf("    %s\n", infoLine)
	}

	// Show coordination channels (exclude done markers)
	if err := showChannelStatus(doneAgents); err != nil {
		return nil
	}

	return nil
}

func showChannelStatus(doneAgents map[string]bool) error {
	channelsDir := getChannelsDir()

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
