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

	// Create worktrees directory
	worktreesDir := filepath.Join(".air", "worktrees")
	if err := os.MkdirAll(worktreesDir, 0755); err != nil {
		return fmt.Errorf("failed to create worktrees directory: %w", err)
	}

	// Create worktrees for each packet
	for _, name := range packets {
		wtPath := filepath.Join(worktreesDir, name)
		branch := "air/" + name

		// Check if worktree already exists
		if _, err := os.Stat(wtPath); err == nil {
			fmt.Printf("Worktree %s already exists\n", name)
			continue
		}

		// Create worktree
		createCmd := exec.Command("git", "worktree", "add", wtPath, "-b", branch)
		createCmd.Stdout = os.Stdout
		createCmd.Stderr = os.Stderr
		if err := createCmd.Run(); err != nil {
			return fmt.Errorf("failed to create worktree for %s: %w", name, err)
		}
		fmt.Printf("Created worktree: %s (branch: %s)\n", wtPath, branch)
	}

	// Start tmux session
	sessionName := "air"

	// Kill existing session if present
	exec.Command("tmux", "kill-session", "-t", sessionName).Run()

	// Absolute path to context file (for shell command substitution)
	contextPath := filepath.Join(projectRoot, ".air", "context.md")

	// Create new session with first packet
	firstPacket := packets[0]
	firstWtPath := filepath.Join(projectRoot, ".air", "worktrees", firstPacket)

	// Create session
	tmuxNew := exec.Command("tmux", "new-session", "-d", "-s", sessionName, "-n", firstPacket, "-c", firstWtPath)
	if err := tmuxNew.Run(); err != nil {
		return fmt.Errorf("failed to create tmux session: %w", err)
	}

	// .air directory path (for --add-dir to grant access from worktree)
	airDir := filepath.Join(projectRoot, ".air")

	// Permission mode flag
	permissionFlag := ""
	if !noAutoAccept {
		permissionFlag = "--permission-mode acceptEdits"
	}

	// Build claude command:
	// - --add-dir grants access to .air/ from the worktree
	// - --permission-mode acceptEdits auto-accepts file edits
	// - --append-system-prompt injects workflow context
	// - Initial prompt tells agent to read their packet
	claudeCmd := fmt.Sprintf(
		`claude --add-dir '%s' %s --append-system-prompt "$(cat '%s')" "Read %s/packets/%s.md and implement it."`,
		airDir,
		permissionFlag,
		contextPath,
		airDir,
		firstPacket,
	)
	exec.Command("tmux", "send-keys", "-t", sessionName+":"+firstPacket, claudeCmd, "Enter").Run()

	// Create windows for remaining packets
	for _, name := range packets[1:] {
		wtPath := filepath.Join(projectRoot, ".air", "worktrees", name)

		// Create window
		exec.Command("tmux", "new-window", "-t", sessionName, "-n", name, "-c", wtPath).Run()

		// Send claude command
		claudeCmd := fmt.Sprintf(
			`claude --add-dir '%s' %s --append-system-prompt "$(cat '%s')" "Read %s/packets/%s.md and implement it."`,
			airDir,
			permissionFlag,
			contextPath,
			airDir,
			name,
		)
		exec.Command("tmux", "send-keys", "-t", sessionName+":"+name, claudeCmd, "Enter").Run()
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
