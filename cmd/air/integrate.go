package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var integrateCmd = &cobra.Command{
	Use:   "integrate",
	Short: "Start integration session to merge completed work",
	RunE:  runIntegrate,
}

func runIntegrate(cmd *cobra.Command, args []string) error {
	// Check initialization
	if !isInitialized() {
		return fmt.Errorf("not initialized (run 'air init' first)")
	}

	// Read context
	context, err := os.ReadFile(getContextPath())
	if err != nil {
		return fmt.Errorf("failed to read context: %w", err)
	}

	// Build integration prompt
	integrationPrompt := string(context) + "\n\n" + integrationContext

	// Launch claude with initial prompt
	claudeCmd := buildIntegrateCommand(integrationPrompt)
	claudeCmd.Stdin = os.Stdin
	claudeCmd.Stdout = os.Stdout
	claudeCmd.Stderr = os.Stderr

	return claudeCmd.Run()
}

// buildIntegrateCommand constructs the claude command for integration mode.
// Extracted for testability - allows verifying command args are correctly structured.
func buildIntegrateCommand(integrationPrompt string) *exec.Cmd {
	// Allowed tools for integration: read-only git commands, air commands, and file inspection
	allowedTools := `Bash(git worktree:*) Bash(git branch:*) Bash(git log:*) Bash(git diff:*) Bash(git merge-tree:*) Bash(git merge-base:*) Bash(air plan:*) Bash(cat:*) Bash(ls:*)`

	return exec.Command("claude",
		"--allowedTools", allowedTools,
		"--append-system-prompt", integrationPrompt,
		"Begin integration. Show me the status of agent branches and guide me through merging.")
}

const integrationContext = `## Integration Mode

You are helping integrate completed agent work into the main branch.

### Step 1: Assess the situation

Run these commands to understand the current state:
- ` + "`" + `git branch -a | grep air/` + "`" + ` - Show agent branches
- ` + "`" + `git worktree list` + "`" + ` - Show active worktrees

### Step 2: Determine merge order from dependencies

Use ` + "`" + `air plan list` + "`" + ` to see all plans, then ` + "`" + `air plan show <name>` + "`" + ` to read each one. Look for the **Dependencies** sections - plans that "Signal" a channel must be merged before plans that "Wait on" that channel.

Build a topological merge order. For example:
- setup (no dependencies) → merge first
- core (waits on setup) → merge second
- strings, hashes, ttl (wait on core) → merge last (order among these doesn't matter)

### Step 3: Present the merge strategy

Show the user:
1. The recommended merge order with rationale
2. A preview of what each branch changes: ` + "`" + `git log --oneline HEAD..air/<name>` + "`" + `
3. Conflict check for the first branch: ` + "`" + `git merge-tree $(git merge-base HEAD air/<name>) HEAD air/<name>` + "`" + `

Then ask: **"Would you like me to handle the merging for you?"**

### Step 4a: If user wants you to handle it

For each branch in order:
1. Check for conflicts: ` + "`" + `git merge-tree $(git merge-base HEAD air/<name>) HEAD air/<name>` + "`" + `
2. If clean, execute: ` + "`" + `git merge air/<name> --no-ff -m "Merge <name>"` + "`" + `
3. If conflicts detected, STOP and help resolve before continuing
4. After each successful merge, briefly confirm and move to the next

After all merges complete:
- Summarize what was merged
- Offer to run tests if a test command exists (check for Makefile, go.mod, package.json)
- Remind user: ` + "`" + `air clean` + "`" + ` removes worktrees and will ask about deleting branches

### Step 4b: If user wants to do it themselves

Provide the merge commands in the correct order. For each branch show:
` + "`" + `git merge air/<name> --no-ff -m "Merge <name>"` + "`" + `

### Handling conflicts

If a merge has conflicts:
1. Show which files conflict
2. Help resolve them interactively
3. After resolution: ` + "`" + `git add <files>` + "`" + ` then ` + "`" + `git commit` + "`" + `
4. Continue with remaining branches
`
