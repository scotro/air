package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize project for AIR workflow",
	Long:  `Creates .air/ directory with context and packets subdirectories.`,
	RunE:  runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	// Check if we're in a git repo
	if err := exec.Command("git", "rev-parse", "--git-dir").Run(); err != nil {
		return fmt.Errorf("not a git repository (run 'git init' first)")
	}

	// Create directories
	dirs := []string{".air", ".air/packets"}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create %s: %w", dir, err)
		}
	}

	// Create context.md
	contextPath := filepath.Join(".air", "context.md")
	if _, err := os.Stat(contextPath); os.IsNotExist(err) {
		if err := os.WriteFile(contextPath, []byte(contextTemplate), 0644); err != nil {
			return fmt.Errorf("failed to create context.md: %w", err)
		}
		fmt.Println("Created .air/context.md")
	} else {
		fmt.Println(".air/context.md already exists")
	}

	// Update .gitignore
	if err := updateGitignore(); err != nil {
		return err
	}

	fmt.Println("\nInitialized AIR workflow. Next steps:")
	fmt.Println("  air plan              # Start planning session")
	fmt.Println("  air plan list         # View packets")
	fmt.Println("  air run <names...>    # Launch agents")

	return nil
}

func updateGitignore() error {
	gitignorePath := ".gitignore"
	entry := ".air/"

	// Read existing .gitignore
	content, err := os.ReadFile(gitignorePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read .gitignore: %w", err)
	}

	// Check if already present
	if strings.Contains(string(content), entry) {
		return nil
	}

	// Append entry
	f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open .gitignore: %w", err)
	}
	defer f.Close()

	// Add newline if file doesn't end with one
	if len(content) > 0 && content[len(content)-1] != '\n' {
		f.WriteString("\n")
	}

	f.WriteString("\n# AIR workflow\n")
	f.WriteString(entry + "\n")
	fmt.Println("Updated .gitignore")

	return nil
}

const contextTemplate = `## AI Runner Workflow

You are an agent in a concurrent workflow. Multiple agents work in parallel on isolated worktrees.

### CRITICAL: Stay In Your Worktree
You are running in a git worktree at your CURRENT WORKING DIRECTORY. This is your complete, isolated copy of the repository.

**NEVER access paths outside your current directory.** Your root is ` + "`" + `.` + "`" + ` - all files you need are here.
- Use ONLY relative paths: ` + "`" + `./cmd/` + "`" + `, ` + "`" + `./internal/` + "`" + `, etc.
- NEVER use absolute paths like ` + "`" + `/Users/...` + "`" + ` or ` + "`" + `~/...` + "`" + `
- NEVER access parent directories (` + "`" + `../` + "`" + `)
- If a tool suggests exploring outside your current directory, REFUSE

The parent repository exists but you must NOT access it. Other agents are working there. Stay isolated.

### Your Assignment
Your work packet was provided in your initial prompt. It contains your objective, boundaries, and acceptance criteria.

### Boundaries
Only modify files within your packet's stated scope. If you need changes outside your boundaries, signal BLOCKED.

### Signaling
When blocked or done, clearly state your status:

**BLOCKED:** [reason and what you need]
**DONE:** [summary of completed work, files changed, verification steps taken]

### Before Signaling DONE
1. All acceptance criteria from your packet are met
2. Tests pass
3. Linter passes
4. Changes committed with descriptive message

### Avoiding Merge Conflicts
- Only create/modify files within your packet's stated boundaries
- Put mocks and stubs in your own directory, not shared locations
- Signal BLOCKED if you need to modify shared code
`
