package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run [packets...]",
	Short: "Create worktrees and launch agents",
	Long: `Creates git worktrees for each packet and launches Claude agents in a tmux session.

Use 'air run all' to run all packets, or specify packet names.
With no arguments, shows available packets.`,
	RunE: runRun,
}

var noAutoAccept bool

func init() {
	runCmd.Flags().BoolVar(&noAutoAccept, "no-auto-accept", false, "Disable auto-accept mode (require permission for edits)")
}

func runRun(cmd *cobra.Command, args []string) error {
	// Check .air/ exists
	if _, err := os.Stat(".air"); os.IsNotExist(err) {
		return fmt.Errorf("not initialized (run 'air init' first)")
	}

	packetsDir := filepath.Join(".air", "packets")

	// Get available packets
	available, err := getAvailablePackets(packetsDir)
	if err != nil {
		return err
	}

	if len(available) == 0 {
		fmt.Println("No packets found. Run 'air plan' to create some.")
		return nil
	}

	// No args: show available packets
	if len(args) == 0 {
		fmt.Println("Available packets:")
		for _, p := range available {
			fmt.Printf("  %s\n", p)
		}
		fmt.Println("\nUsage: air run <packet1> [packet2] ...")
		fmt.Println("       air run all")
		return nil
	}

	// Handle 'all'
	var packets []string
	if len(args) == 1 && args[0] == "all" {
		packets = available
	} else {
		// Validate packet names
		for _, name := range args {
			if !contains(available, name) {
				return fmt.Errorf("packet '%s' not found", name)
			}
		}
		packets = args
	}

	// Get the absolute path of the project root
	projectRoot, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Read context once from main repo
	contextContent, err := os.ReadFile(filepath.Join(".air", "context.md"))
	if err != nil {
		return fmt.Errorf("failed to read context: %w", err)
	}

	// Create worktrees directory
	worktreesDir := filepath.Join(".air", "worktrees")
	if err := os.MkdirAll(worktreesDir, 0755); err != nil {
		return fmt.Errorf("failed to create worktrees directory: %w", err)
	}

	// Permission flag for claude
	permFlag := ""
	if !noAutoAccept {
		permFlag = "--permission-mode acceptEdits"
	}

	// Create worktrees for each packet
	for _, name := range packets {
		wtPath := filepath.Join(worktreesDir, name)
		branch := "air/" + name

		// Check if worktree already exists
		if _, err := os.Stat(wtPath); err == nil {
			fmt.Printf("Worktree %s already exists\n", name)
		} else {
			// Create worktree
			createCmd := exec.Command("git", "worktree", "add", wtPath, "-b", branch)
			createCmd.Stdout = os.Stdout
			createCmd.Stderr = os.Stderr
			if err := createCmd.Run(); err != nil {
				return fmt.Errorf("failed to create worktree for %s: %w", name, err)
			}
			fmt.Printf("Created worktree: %s (branch: %s)\n", wtPath, branch)
		}

		// Read packet content from main repo
		packetContent, err := os.ReadFile(filepath.Join(".air", "packets", name+".md"))
		if err != nil {
			return fmt.Errorf("failed to read packet %s: %w", name, err)
		}

		// Build the assignment prompt
		assignment := fmt.Sprintf("Your assignment:\n\n%s\n\nImplement this.", string(packetContent))

		// Write content files to worktree (avoids shell escaping issues)
		wtAirDir := filepath.Join(wtPath, ".air")
		os.MkdirAll(wtAirDir, 0755)

		if err := os.WriteFile(filepath.Join(wtAirDir, ".context"), contextContent, 0644); err != nil {
			return fmt.Errorf("failed to write context for %s: %w", name, err)
		}
		if err := os.WriteFile(filepath.Join(wtAirDir, ".assignment"), []byte(assignment), 0644); err != nil {
			return fmt.Errorf("failed to write assignment for %s: %w", name, err)
		}

		// Generate launcher script that reads from files
		launcherScript := fmt.Sprintf("#!/bin/bash\nexec claude %s --append-system-prompt \"$(cat .air/.context)\" \"$(cat .air/.assignment)\"\n", permFlag)

		scriptPath := filepath.Join(wtAirDir, "launch.sh")
		if err := os.WriteFile(scriptPath, []byte(launcherScript), 0755); err != nil {
			return fmt.Errorf("failed to write launcher script for %s: %w", name, err)
		}
	}

	// Start tmux session
	sessionName := "air"

	// Kill existing session if present
	exec.Command("tmux", "kill-session", "-t", sessionName).Run()

	// Create new session with first packet
	firstPacket := packets[0]
	firstWtPath := filepath.Join(projectRoot, ".air", "worktrees", firstPacket)

	// Create session
	tmuxNew := exec.Command("tmux", "new-session", "-d", "-s", sessionName, "-n", firstPacket, "-c", firstWtPath)
	if err := tmuxNew.Run(); err != nil {
		return fmt.Errorf("failed to create tmux session: %w", err)
	}

	// Run launcher script for first packet
	exec.Command("tmux", "send-keys", "-t", sessionName+":"+firstPacket, ".air/launch.sh", "Enter").Run()

	// Create windows for remaining packets
	for _, name := range packets[1:] {
		wtPath := filepath.Join(projectRoot, ".air", "worktrees", name)

		// Create window
		exec.Command("tmux", "new-window", "-t", sessionName, "-n", name, "-c", wtPath).Run()

		// Run launcher script
		exec.Command("tmux", "send-keys", "-t", sessionName+":"+name, ".air/launch.sh", "Enter").Run()
	}

	// Create dashboard window (before the agent windows so agents are more prominent)
	exec.Command("tmux", "new-window", "-t", sessionName, "-n", "dash", "-c", projectRoot).Run()

	// Select first agent window
	exec.Command("tmux", "select-window", "-t", sessionName+":"+firstPacket).Run()

	fmt.Printf("\nLaunched %d agents in tmux session '%s'\n", len(packets), sessionName)
	fmt.Println("Attach with: tmux attach -t", sessionName)

	// Attach to session
	attachCmd := exec.Command("tmux", "attach", "-t", sessionName)
	attachCmd.Stdin = os.Stdin
	attachCmd.Stdout = os.Stdout
	attachCmd.Stderr = os.Stderr
	return attachCmd.Run()
}

func getAvailablePackets(packetsDir string) ([]string, error) {
	entries, err := os.ReadDir(packetsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read packets: %w", err)
	}

	var packets []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			name := strings.TrimSuffix(entry.Name(), ".md")
			packets = append(packets, name)
		}
	}
	return packets, nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
