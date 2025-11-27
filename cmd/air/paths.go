package main

import (
	"os"
	"path/filepath"
)

// getProjectName returns the basename of the current working directory.
// This is used as the project identifier in ~/.air/<project>/
func getProjectName() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Base(cwd), nil
}

// getAirDir returns the air directory for the current project: ~/.air/<project>/
func getAirDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	project, err := getProjectName()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".air", project), nil
}

// mustGetAirDir returns the air directory or panics. Use only when error handling
// has already been done (e.g., after isInitialized check).
func mustGetAirDir() string {
	dir, err := getAirDir()
	if err != nil {
		panic(err)
	}
	return dir
}

// getPlansDir returns ~/.air/<project>/plans/
func getPlansDir() string {
	return filepath.Join(mustGetAirDir(), "plans")
}

// getWorktreesDir returns ~/.air/<project>/worktrees/
func getWorktreesDir() string {
	return filepath.Join(mustGetAirDir(), "worktrees")
}

// getAgentsDir returns ~/.air/<project>/agents/
func getAgentsDir() string {
	return filepath.Join(mustGetAirDir(), "agents")
}

// getChannelsDir returns the channels directory.
// For agent commands (with AIR_CHANNELS_DIR set), returns the env var value.
// For main project commands, computes ~/.air/<project>/channels/
func getChannelsDir() string {
	// Agent context: use env var
	if dir := os.Getenv("AIR_CHANNELS_DIR"); dir != "" {
		return dir
	}
	// Main project context: compute from project name
	return filepath.Join(mustGetAirDir(), "channels")
}

// getContextPath returns ~/.air/<project>/context.md
func getContextPath() string {
	return filepath.Join(mustGetAirDir(), "context.md")
}

// isInitialized checks if the air directory exists for the current project.
func isInitialized() bool {
	dir, err := getAirDir()
	if err != nil {
		return false
	}
	_, err = os.Stat(dir)
	return err == nil
}
