package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ============================================================================
// air agent signal tests
// ============================================================================

func TestAgentSignal_CreatesChannelFile(t *testing.T) {
	tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create channels directory
	channelsDir := filepath.Join(tmpDir, ".air", "channels")
	os.MkdirAll(channelsDir, 0755)

	// Run signal command
	out, err := runAirWithEnv(t, tmpDir, map[string]string{
		"AIR_AGENT_ID":    "test-agent",
		"AIR_WORKTREE":    tmpDir,
		"AIR_CHANNELS_DIR": channelsDir,
	}, "agent", "signal", "test-channel")

	if err != nil {
		t.Fatalf("air agent signal failed: %v\n%s", err, out)
	}

	// Check channel file was created
	channelPath := filepath.Join(channelsDir, "test-channel.json")
	if _, err := os.Stat(channelPath); os.IsNotExist(err) {
		t.Error("channel file was not created")
	}

	// Verify JSON structure
	data, err := os.ReadFile(channelPath)
	if err != nil {
		t.Fatalf("failed to read channel file: %v", err)
	}

	var payload ChannelPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("failed to parse channel JSON: %v", err)
	}

	if payload.Agent != "test-agent" {
		t.Errorf("expected agent 'test-agent', got %q", payload.Agent)
	}
	if payload.SHA == "" {
		t.Error("SHA should not be empty")
	}
	if payload.Worktree != tmpDir {
		t.Errorf("expected worktree %q, got %q", tmpDir, payload.Worktree)
	}
}

func TestAgentSignal_FailsIfAlreadySignaled(t *testing.T) {
	tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	channelsDir := filepath.Join(tmpDir, ".air", "channels")
	os.MkdirAll(channelsDir, 0755)

	env := map[string]string{
		"AIR_AGENT_ID":    "test-agent",
		"AIR_WORKTREE":    tmpDir,
		"AIR_CHANNELS_DIR": channelsDir,
	}

	// First signal should succeed
	_, err := runAirWithEnv(t, tmpDir, env, "agent", "signal", "test-channel")
	if err != nil {
		t.Fatalf("first signal failed: %v", err)
	}

	// Second signal should fail
	_, err = runAirWithEnv(t, tmpDir, env, "agent", "signal", "test-channel")
	if err == nil {
		t.Error("expected error when signaling already-signaled channel")
	}
}

func TestAgentSignal_FailsWithoutAgentID(t *testing.T) {
	tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	channelsDir := filepath.Join(tmpDir, ".air", "channels")
	os.MkdirAll(channelsDir, 0755)

	// Don't set AIR_AGENT_ID
	_, err := runAirWithEnv(t, tmpDir, map[string]string{
		"AIR_CHANNELS_DIR": channelsDir,
	}, "agent", "signal", "test-channel")

	if err == nil {
		t.Error("expected error without AIR_AGENT_ID")
	}
}

func TestAgentSignal_CreatesSubdirectories(t *testing.T) {
	tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	channelsDir := filepath.Join(tmpDir, ".air", "channels")
	os.MkdirAll(channelsDir, 0755)

	// Signal with a path that includes subdirectory
	_, err := runAirWithEnv(t, tmpDir, map[string]string{
		"AIR_AGENT_ID":    "test-agent",
		"AIR_WORKTREE":    tmpDir,
		"AIR_CHANNELS_DIR": channelsDir,
	}, "agent", "signal", "done/test-agent")

	if err != nil {
		t.Fatalf("signal with subdirectory failed: %v", err)
	}

	// Check subdirectory was created
	channelPath := filepath.Join(channelsDir, "done", "test-agent.json")
	if _, err := os.Stat(channelPath); os.IsNotExist(err) {
		t.Error("channel file in subdirectory was not created")
	}
}

// ============================================================================
// air agent wait tests
// ============================================================================

func TestAgentWait_ReturnsImmediatelyIfChannelExists(t *testing.T) {
	tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	channelsDir := filepath.Join(tmpDir, ".air", "channels")
	os.MkdirAll(channelsDir, 0755)

	// Pre-create a channel file
	payload := ChannelPayload{
		SHA:       "abc123",
		Worktree:  "/test/path",
		Agent:     "producer",
		Timestamp: time.Now(),
	}
	data, _ := json.MarshalIndent(payload, "", "  ")
	os.WriteFile(filepath.Join(channelsDir, "pre-existing.json"), data, 0644)

	// Wait should return immediately
	start := time.Now()
	out, err := runAirWithEnv(t, tmpDir, map[string]string{
		"AIR_CHANNELS_DIR": channelsDir,
	}, "agent", "wait", "pre-existing")

	if err != nil {
		t.Fatalf("wait failed: %v\n%s", err, out)
	}

	// Should complete quickly (under 1 second)
	if time.Since(start) > 1*time.Second {
		t.Error("wait took too long for pre-existing channel")
	}

	// Should print the payload
	if !strings.Contains(out, "abc123") {
		t.Error("wait output missing SHA")
	}
	if !strings.Contains(out, "producer") {
		t.Error("wait output missing agent")
	}
}

func TestAgentWait_BlocksUntilSignaled(t *testing.T) {
	tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	channelsDir := filepath.Join(tmpDir, ".air", "channels")
	os.MkdirAll(channelsDir, 0755)

	// Start wait in background
	done := make(chan struct{})
	var waitErr error
	var waitOut string

	go func() {
		waitOut, waitErr = runAirWithEnv(t, tmpDir, map[string]string{
			"AIR_CHANNELS_DIR": channelsDir,
		}, "agent", "wait", "delayed-channel")
		close(done)
	}()

	// Wait a bit to ensure the wait command is blocking
	time.Sleep(500 * time.Millisecond)

	select {
	case <-done:
		t.Fatal("wait returned before channel was signaled")
	default:
		// Good - still blocking
	}

	// Now signal the channel
	payload := ChannelPayload{
		SHA:       "delayed123",
		Worktree:  "/test/path",
		Agent:     "delayed-producer",
		Timestamp: time.Now(),
	}
	data, _ := json.MarshalIndent(payload, "", "  ")
	os.WriteFile(filepath.Join(channelsDir, "delayed-channel.json"), data, 0644)

	// Wait should now complete
	select {
	case <-done:
		if waitErr != nil {
			t.Errorf("wait failed after signal: %v\n%s", waitErr, waitOut)
		}
		if !strings.Contains(waitOut, "delayed123") {
			t.Error("wait output missing expected SHA")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("wait did not complete after channel was signaled")
	}
}

// ============================================================================
// air agent done tests
// ============================================================================

func TestAgentDone_SignalsDoneChannel(t *testing.T) {
	tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	channelsDir := filepath.Join(tmpDir, ".air", "channels")
	os.MkdirAll(channelsDir, 0755)

	out, err := runAirWithEnv(t, tmpDir, map[string]string{
		"AIR_AGENT_ID":    "my-agent",
		"AIR_WORKTREE":    tmpDir,
		"AIR_CHANNELS_DIR": channelsDir,
	}, "agent", "done")

	if err != nil {
		t.Fatalf("agent done failed: %v\n%s", err, out)
	}

	// Check done/<agent-id> channel was created
	channelPath := filepath.Join(channelsDir, "done", "my-agent.json")
	if _, err := os.Stat(channelPath); os.IsNotExist(err) {
		t.Error("done channel was not created")
	}

	// Verify agent ID in payload
	data, _ := os.ReadFile(channelPath)
	var payload ChannelPayload
	json.Unmarshal(data, &payload)
	if payload.Agent != "my-agent" {
		t.Errorf("expected agent 'my-agent', got %q", payload.Agent)
	}
}

func TestAgentDone_FailsWithoutAgentID(t *testing.T) {
	tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	channelsDir := filepath.Join(tmpDir, ".air", "channels")
	os.MkdirAll(channelsDir, 0755)

	_, err := runAirWithEnv(t, tmpDir, map[string]string{
		"AIR_CHANNELS_DIR": channelsDir,
	}, "agent", "done")

	if err == nil {
		t.Error("expected error without AIR_AGENT_ID")
	}
}

// ============================================================================
// air agent merge tests
// ============================================================================

func TestAgentMerge_FailsIfChannelNotSignaled(t *testing.T) {
	tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	channelsDir := filepath.Join(tmpDir, ".air", "channels")
	os.MkdirAll(channelsDir, 0755)

	_, err := runAirWithEnv(t, tmpDir, map[string]string{
		"AIR_CHANNELS_DIR": channelsDir,
	}, "agent", "merge", "nonexistent")

	if err == nil {
		t.Error("expected error for unsignaled channel")
	}
}

func TestAgentMerge_MergesBranchFromSameRepo(t *testing.T) {
	// This tests the scenario where worktrees share the same git object store
	tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	channelsDir := filepath.Join(tmpDir, ".air", "channels")
	os.MkdirAll(channelsDir, 0755)

	// Create a feature branch with a commit
	exec.Command("git", "-C", tmpDir, "checkout", "-b", "air/producer").Run()
	testFile := filepath.Join(tmpDir, "new-feature.txt")
	os.WriteFile(testFile, []byte("new feature content"), 0644)
	exec.Command("git", "-C", tmpDir, "add", "new-feature.txt").Run()
	exec.Command("git", "-C", tmpDir, "commit", "-m", "Add new feature").Run()

	// Get the SHA of the commit
	shaCmd := exec.Command("git", "-C", tmpDir, "rev-parse", "HEAD")
	shaOut, _ := shaCmd.Output()
	sha := strings.TrimSpace(string(shaOut))

	// Create a consumer branch from main (before the feature)
	exec.Command("git", "-C", tmpDir, "checkout", "main").Run()
	exec.Command("git", "-C", tmpDir, "checkout", "-b", "air/consumer").Run()

	// Verify the file doesn't exist on this branch
	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Fatal("test file should not exist on consumer branch")
	}

	// Create channel pointing to the producer branch
	payload := ChannelPayload{
		SHA:       sha,
		Branch:    "air/producer",
		Worktree:  tmpDir,
		Agent:     "producer",
		Timestamp: time.Now(),
	}
	data, _ := json.MarshalIndent(payload, "", "  ")
	os.WriteFile(filepath.Join(channelsDir, "feature-ready.json"), data, 0644)

	// Merge the branch
	out, err := runAirWithEnv(t, tmpDir, map[string]string{
		"AIR_CHANNELS_DIR": channelsDir,
	}, "agent", "merge", "feature-ready")

	if err != nil {
		t.Fatalf("merge failed: %v\n%s", err, out)
	}

	// Verify the file now exists
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Error("merged file should exist")
	}

	// Verify content
	content, _ := os.ReadFile(testFile)
	if string(content) != "new feature content" {
		t.Errorf("unexpected file content: %s", content)
	}
}

// ============================================================================
// air run env vars tests
// ============================================================================

func TestRun_SetsEnvironmentVariables(t *testing.T) {
	tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	initProject(t, tmpDir)

	// Create test plan
	airDir := getTestAirDir(t, tmpDir)
	os.WriteFile(filepath.Join(airDir, "plans", "test.md"), []byte("# Test\n**Objective:** Test"), 0644)

	runAir(t, tmpDir, "run", "test")

	// Read the launch script (now in agents dir)
	scriptPath := filepath.Join(airDir, "agents", "test", "launch.sh")
	content, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("failed to read launch.sh: %v", err)
	}

	script := string(content)

	// Check for env var exports
	if !strings.Contains(script, "AIR_AGENT_ID=") {
		t.Error("launch.sh missing AIR_AGENT_ID")
	}
	if !strings.Contains(script, "AIR_WORKTREE=") {
		t.Error("launch.sh missing AIR_WORKTREE")
	}
	if !strings.Contains(script, "AIR_PROJECT_ROOT=") {
		t.Error("launch.sh missing AIR_PROJECT_ROOT")
	}
	if !strings.Contains(script, "AIR_CHANNELS_DIR=") {
		t.Error("launch.sh missing AIR_CHANNELS_DIR")
	}

	// Verify AIR_AGENT_ID is set to the plan name
	if !strings.Contains(script, `AIR_AGENT_ID="test"`) {
		t.Error("AIR_AGENT_ID should be set to plan name")
	}
}

func TestRun_CreatesChannelsDirectory(t *testing.T) {
	tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	initProject(t, tmpDir)

	// Create test plan
	airDir := getTestAirDir(t, tmpDir)
	os.WriteFile(filepath.Join(airDir, "plans", "test.md"), []byte("# Test"), 0644)

	runAir(t, tmpDir, "run", "test")

	// Check channels directory was created
	channelsDir := filepath.Join(airDir, "channels")
	if _, err := os.Stat(channelsDir); os.IsNotExist(err) {
		t.Error("channels directory was not created")
	}
}

// ============================================================================
// Helper functions
// ============================================================================

// runAirWithEnv runs the air command with custom environment variables
func runAirWithEnv(t *testing.T, dir string, env map[string]string, args ...string) (string, error) {
	t.Helper()

	cmd := exec.Command(testBinaryPath, args...)
	cmd.Dir = dir

	// Set up environment - filter out AIR_* variables from parent environment
	// to ensure tests have complete control over AIR-specific env vars
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, "AIR_") {
			cmd.Env = append(cmd.Env, e)
		}
	}
	// Add the explicitly provided env vars
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	out, err := cmd.CombinedOutput()
	return string(out), err
}
