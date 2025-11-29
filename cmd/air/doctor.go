package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check environment for required dependencies",
	Long:  `Diagnoses the environment to ensure all required tools are installed and configured correctly.`,
	RunE:  runDoctor,
}

type checkResult struct {
	name    string
	ok      bool
	version string
	message string
}

func runDoctor(cmd *cobra.Command, args []string) error {
	fmt.Println("Checking environment...")
	fmt.Println()

	var results []checkResult
	allOk := true

	// Check git
	results = append(results, checkGit())

	// Check tmux
	results = append(results, checkTmux())

	// Check claude CLI
	results = append(results, checkClaude())

	// Check SSH agent
	results = append(results, checkSSHAgent())

	// Check if in a git repo (optional context)
	results = append(results, checkGitRepo())

	// Check if air is initialized (optional context)
	results = append(results, checkAirInit())

	// Print results
	for _, r := range results {
		if r.ok {
			if r.version != "" {
				fmt.Printf("  ✓ %s %s\n", r.name, r.version)
			} else {
				fmt.Printf("  ✓ %s\n", r.name)
			}
		} else {
			allOk = false
			fmt.Printf("  ✗ %s - %s\n", r.name, r.message)
		}
	}

	fmt.Println()
	if allOk {
		fmt.Println("All checks passed!")
	} else {
		fmt.Println("Some checks failed. Fix the issues above to use air.")
	}

	return nil
}

func checkGit() checkResult {
	out, err := exec.Command("git", "--version").Output()
	if err != nil {
		return checkResult{
			name:    "git",
			ok:      false,
			message: "not found (install from https://git-scm.com)",
		}
	}

	// Parse version from "git version 2.40.0"
	version := strings.TrimSpace(string(out))
	version = strings.TrimPrefix(version, "git version ")

	return checkResult{
		name:    "git",
		ok:      true,
		version: version,
	}
}

func checkTmux() checkResult {
	out, err := exec.Command("tmux", "-V").Output()
	if err != nil {
		return checkResult{
			name:    "tmux",
			ok:      false,
			message: "not found (install: brew install tmux)",
		}
	}

	// Parse version from "tmux 3.3a"
	version := strings.TrimSpace(string(out))
	version = strings.TrimPrefix(version, "tmux ")

	return checkResult{
		name:    "tmux",
		ok:      true,
		version: version,
	}
}

func checkClaude() checkResult {
	out, err := exec.Command("claude", "--version").Output()
	if err != nil {
		return checkResult{
			name:    "claude",
			ok:      false,
			message: "not found (install from https://docs.anthropic.com/en/docs/claude-code)",
		}
	}

	// Parse version - claude outputs version info
	version := strings.TrimSpace(string(out))
	// Take first line if multiline
	if idx := strings.Index(version, "\n"); idx != -1 {
		version = version[:idx]
	}

	return checkResult{
		name:    "claude",
		ok:      true,
		version: version,
	}
}

func checkSSHAgent() checkResult {
	sshAuthSock := os.Getenv("SSH_AUTH_SOCK")
	if sshAuthSock == "" {
		return checkResult{
			name:    "ssh-agent",
			ok:      false,
			message: "SSH_AUTH_SOCK not set (git push may fail)",
		}
	}

	// Check if the socket exists
	if _, err := os.Stat(sshAuthSock); os.IsNotExist(err) {
		return checkResult{
			name:    "ssh-agent",
			ok:      false,
			message: "SSH_AUTH_SOCK socket not found (git push may fail)",
		}
	}

	return checkResult{
		name:    "ssh-agent",
		ok:      true,
		version: "running",
	}
}

func checkGitRepo() checkResult {
	err := exec.Command("git", "rev-parse", "--git-dir").Run()
	if err != nil {
		return checkResult{
			name:    "git repo",
			ok:      false,
			message: "not in a git repository",
		}
	}

	return checkResult{
		name:    "git repo",
		ok:      true,
		version: "detected",
	}
}

func checkAirInit() checkResult {
	if !isInitialized() {
		return checkResult{
			name:    "air init",
			ok:      false,
			message: "not initialized (run 'air init')",
		}
	}

	return checkResult{
		name:    "air init",
		ok:      true,
		version: "configured",
	}
}
