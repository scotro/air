package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize project for Air workflow",
	Long:  `Creates ~/.air/<project>/ directory with context and plans subdirectories.`,
	RunE:  runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	// Check if we're in a git repo
	if err := exec.Command("git", "rev-parse", "--git-dir").Run(); err != nil {
		return fmt.Errorf("not a git repository (run 'git init' first)")
	}

	// Get air directory path
	airDir, err := getAirDir()
	if err != nil {
		return fmt.Errorf("failed to determine air directory: %w", err)
	}

	projectName, _ := getProjectName()

	// Check for collision (directory already exists for different project)
	if _, err := os.Stat(airDir); err == nil {
		// Directory exists - check if it's for this project by verifying we're in the right place
		// For now, just warn and continue (re-init is allowed)
		fmt.Printf("Air directory already exists: %s\n", airDir)
	}

	// Create directories
	plansDir := getPlansDir()
	if err := os.MkdirAll(plansDir, 0755); err != nil {
		return fmt.Errorf("failed to create plans directory: %w", err)
	}

	// Create context.md
	contextPath := getContextPath()
	if _, err := os.Stat(contextPath); os.IsNotExist(err) {
		if err := os.WriteFile(contextPath, []byte(contextTemplate), 0644); err != nil {
			return fmt.Errorf("failed to create context.md: %w", err)
		}
		fmt.Printf("Created %s\n", contextPath)
	} else {
		fmt.Printf("context.md already exists at %s\n", contextPath)
	}

	fmt.Printf("\nInitialized Air workflow for '%s'.\n", projectName)
	fmt.Printf("Air directory: %s\n", airDir)
	fmt.Println("\nNext steps:")
	fmt.Println("  air plan              # Start planning session")
	fmt.Println("  air plan list         # View plans")
	fmt.Println("  air run <names...>    # Launch agents")

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
Your plan was provided in your initial prompt. It contains your objective, boundaries, and acceptance criteria.

### Boundaries
Only modify files within your plan's stated scope. If you need changes outside your boundaries, signal BLOCKED.

### Signaling
When blocked or done, clearly state your status:

**BLOCKED:** [reason and what you need]
**DONE:** [summary of completed work, files changed, verification steps taken]

### Before Signaling DONE
1. All acceptance criteria from your plan are met
2. Tests pass
3. Linter passes
4. Changes committed with descriptive message

### Avoiding Merge Conflicts
- Only create/modify files within your plan's stated boundaries
- Put mocks and stubs in your own directory, not shared locations
- Signal BLOCKED if you need to modify shared code

### Coordination Commands (if your plan has a Dependencies section)

If your plan includes a **Dependencies** section, you must coordinate with other agents using these commands:

**Waiting for another agent:**
` + "```" + `bash
air agent wait <channel-name>    # Blocks until the channel is signaled (use 600000ms timeout)
air agent merge <channel>        # Merges the dependency branch into your worktree
` + "```" + `

**Important:** When running ` + "`" + `air agent wait` + "`" + `, use a 10-minute timeout (600000ms). If it times out, simply run it again. Keep retrying until the channel is signaled - the other agent may still be working.

**Signaling other agents:**
` + "```" + `bash
air agent signal <channel-name>  # Signals the channel with your current commit
air agent done                   # Marks you as complete
` + "```" + `

**Important:**
- Follow the **Sequence** in your Dependencies section exactly
- Always commit your changes BEFORE signaling
- If ` + "`" + `merge` + "`" + ` fails with conflicts, signal BLOCKED and describe the conflict
- Run ` + "`" + `air agent done` + "`" + ` as your final action when all work is complete
`
