package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

var integrateCmd = &cobra.Command{
	Use:   "integrate",
	Short: "Start integration session to merge completed work",
	RunE:  runIntegrate,
}

func runIntegrate(cmd *cobra.Command, args []string) error {
	// Check .air/ exists
	if _, err := os.Stat(".air"); os.IsNotExist(err) {
		return fmt.Errorf("not initialized (run 'air init' first)")
	}

	// Read context
	contextPath := filepath.Join(".air", "context.md")
	context, err := os.ReadFile(contextPath)
	if err != nil {
		return fmt.Errorf("failed to read context: %w", err)
	}

	// Build integration prompt
	integrationPrompt := string(context) + "\n\n" + integrationContext

	// Launch claude with initial prompt
	claudeCmd := exec.Command("claude",
		"--append-system-prompt", integrationPrompt,
		"Begin integration. Show me the status of agent branches and guide me through merging.")
	claudeCmd.Stdin = os.Stdin
	claudeCmd.Stdout = os.Stdout
	claudeCmd.Stderr = os.Stderr

	return claudeCmd.Run()
}

const integrationContext = `## Integration Mode

You are helping integrate completed agent work. Run these commands to assess the situation:

1. ` + "`" + `git worktree list` + "`" + ` - Show active worktrees
2. ` + "`" + `git branch -a | grep air/` + "`" + ` - Show agent branches

For each branch ready to merge:
1. Review changes: ` + "`" + `git diff main..air/<name>` + "`" + `
2. Check for conflicts: ` + "`" + `git merge-tree $(git merge-base main air/<name>) main air/<name>` + "`" + `
3. Provide the merge command: ` + "`" + `git merge air/<name> --no-ff -m "Merge <name>"` + "`" + `

Remind the user:
- Run tests before and after merging
- Merge from the main project directory, not from a worktree
- After successful merge, clean up with: ` + "`" + `air clean` + "`" + `

If there are merge conflicts, help resolve them.
`
