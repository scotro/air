package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

// ChannelPayload represents the data written to a channel file when signaled
type ChannelPayload struct {
	SHA       string    `json:"sha"`
	Worktree  string    `json:"worktree"`
	Agent     string    `json:"agent"`
	Timestamp time.Time `json:"timestamp"`
}

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Agent coordination commands (used during agent execution)",
	Long:  `Commands for coordinating between concurrent agents. These are called by agents during execution, not by users directly.`,
}

var agentSignalCmd = &cobra.Command{
	Use:   "signal <channel>",
	Short: "Signal a channel with the current commit",
	Long:  `Signals a channel to notify waiting agents. Captures the current HEAD commit SHA and writes it to the channel file.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runAgentSignal,
}

var agentWaitCmd = &cobra.Command{
	Use:   "wait <channel>",
	Short: "Wait for a channel to be signaled",
	Long:  `Blocks until the specified channel is signaled, then prints the channel payload.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runAgentWait,
}

var agentCherryPickCmd = &cobra.Command{
	Use:   "cherry-pick <channel>",
	Short: "Cherry-pick the commit from a signaled channel",
	Long:  `Reads the commit SHA from a signaled channel and cherry-picks it into the current worktree.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runAgentCherryPick,
}

var agentDoneCmd = &cobra.Command{
	Use:   "done",
	Short: "Signal that this agent is complete",
	Long:  `Signals completion by writing to the done/<agent-id> channel.`,
	Args:  cobra.NoArgs,
	RunE:  runAgentDone,
}

func init() {
	agentCmd.AddCommand(agentSignalCmd)
	agentCmd.AddCommand(agentWaitCmd)
	agentCmd.AddCommand(agentCherryPickCmd)
	agentCmd.AddCommand(agentDoneCmd)
}

// getChannelsDir returns the path to the channels directory
func getChannelsDir() (string, error) {
	// First try AIR_CHANNELS_DIR
	if dir := os.Getenv("AIR_CHANNELS_DIR"); dir != "" {
		return dir, nil
	}

	// Fall back to AIR_PROJECT_ROOT/.air/channels
	if root := os.Getenv("AIR_PROJECT_ROOT"); root != "" {
		return filepath.Join(root, ".air", "channels"), nil
	}

	// Last resort: try current directory (for testing)
	return filepath.Join(".air", "channels"), nil
}

// getChannelPath returns the full path to a channel file
func getChannelPath(channel string) (string, error) {
	dir, err := getChannelsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, channel+".json"), nil
}

// readChannel reads and parses a channel file
func readChannel(channel string) (*ChannelPayload, error) {
	path, err := getChannelPath(channel)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var payload ChannelPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse channel %s: %w", channel, err)
	}

	return &payload, nil
}

// writeChannel writes a payload to a channel file
func writeChannel(channel string, payload *ChannelPayload) error {
	path, err := getChannelPath(channel)
	if err != nil {
		return err
	}

	// Create parent directories if needed (for done/<id> channels)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create channel directory: %w", err)
	}

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write channel file: %w", err)
	}

	return nil
}

// channelExists checks if a channel has been signaled
func channelExists(channel string) bool {
	path, err := getChannelPath(channel)
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

// getCurrentSHA returns the current HEAD commit SHA
func getCurrentSHA() (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD SHA: %w", err)
	}
	// Trim newline
	sha := string(output)
	if len(sha) > 0 && sha[len(sha)-1] == '\n' {
		sha = sha[:len(sha)-1]
	}
	return sha, nil
}

func runAgentSignal(cmd *cobra.Command, args []string) error {
	channel := args[0]

	// Require AIR_AGENT_ID
	agentID := os.Getenv("AIR_AGENT_ID")
	if agentID == "" {
		return fmt.Errorf("AIR_AGENT_ID environment variable is required")
	}

	// Check if channel already signaled
	if channelExists(channel) {
		return fmt.Errorf("channel '%s' has already been signaled", channel)
	}

	// Get current HEAD SHA
	sha, err := getCurrentSHA()
	if err != nil {
		return err
	}

	// Get worktree path
	worktree := os.Getenv("AIR_WORKTREE")
	if worktree == "" {
		// Fall back to current directory
		worktree, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}
	}

	// Build and write payload
	payload := &ChannelPayload{
		SHA:       sha,
		Worktree:  worktree,
		Agent:     agentID,
		Timestamp: time.Now().UTC(),
	}

	if err := writeChannel(channel, payload); err != nil {
		return err
	}

	fmt.Printf("Signaled channel '%s' (sha: %s)\n", channel, sha[:8])
	return nil
}

func runAgentWait(cmd *cobra.Command, args []string) error {
	channel := args[0]

	fmt.Printf("Waiting for channel '%s'...\n", channel)

	// Poll until channel exists
	for !channelExists(channel) {
		time.Sleep(2 * time.Second)
	}

	// Read and print payload
	payload, err := readChannel(channel)
	if err != nil {
		return err
	}

	// Print payload as JSON
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}

	fmt.Printf("Channel '%s' signaled:\n%s\n", channel, string(data))
	return nil
}

func runAgentCherryPick(cmd *cobra.Command, args []string) error {
	channel := args[0]

	// Read channel payload
	payload, err := readChannel(channel)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("channel '%s' has not been signaled yet", channel)
		}
		return err
	}

	fmt.Printf("Cherry-picking commit %s from %s...\n", payload.SHA[:8], payload.Agent)

	// Since worktrees share the same git object store, we can cherry-pick directly by SHA
	cherryPickCmd := exec.Command("git", "cherry-pick", payload.SHA)
	cherryPickCmd.Stdout = os.Stdout
	cherryPickCmd.Stderr = os.Stderr

	if err := cherryPickCmd.Run(); err != nil {
		return fmt.Errorf("cherry-pick failed (you may need to resolve conflicts manually): %w", err)
	}

	fmt.Printf("Successfully cherry-picked commit %s\n", payload.SHA[:8])
	return nil
}

func runAgentDone(cmd *cobra.Command, args []string) error {
	// Require AIR_AGENT_ID
	agentID := os.Getenv("AIR_AGENT_ID")
	if agentID == "" {
		return fmt.Errorf("AIR_AGENT_ID environment variable is required")
	}

	// Signal done/<agent-id> channel
	channel := "done/" + agentID

	// Reuse signal logic
	return runAgentSignal(cmd, []string{channel})
}
