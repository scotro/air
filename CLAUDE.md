# CLAUDE.md

## Project Overview

Air (AI Runner) is a Go CLI tool that orchestrates multiple Claude Code agents working in parallel on decomposed tasks using git worktrees.

## Commands

```bash
go build -o bin/air ./cmd/air/    # Build
go test ./...                      # Test
```

## Architecture

```
cmd/air/           # CLI commands (cobra)
├── main.go        # Entry point
├── root.go        # Root command
├── init.go        # air init
├── plan.go        # air plan, plan list/show/archive/restore
├── run.go         # air run
├── status.go      # air status
├── integrate.go   # air integrate
├── clean.go       # air clean
└── agent.go       # air agent (coordination commands)
internal/          # (future) shared packages
```

## Key Concepts

- **Plans**: Work units defined in `~/.air/<project>/plans/*.md`
- **Context**: Workflow instructions in `~/.air/<project>/context.md`, injected via `--append-system-prompt`
- **Worktrees**: Isolated git worktrees in `~/.air/<project>/worktrees/` for parallel work
- **Branches**: Named `air/<plan-name>`
- **Channels**: Coordination points in `~/.air/<project>/channels/` for concurrent plans with dependencies
- **Modes**: `ModeSingle` (one git repo) vs `ModeWorkspace` (parent dir with repo children)

## Design Principles

1. Non-invasive: Never touch `.claude/` or `CLAUDE.md` in user projects
2. Wrapper: We wrap `claude` with context, we don't configure it
3. Our namespace: Everything lives in `~/.air/<project>/`

## Testing

Tests must be parallel-safe and avoid polluting global state. Use the subprocess sandbox pattern:

```go
func TestFeature(t *testing.T) {
    t.Parallel()
    env := setupTestRepo(t)    // Creates temp dir + fake HOME
    defer env.cleanup()

    out, err := env.run(t, nil, "init")  // Runs air as subprocess
    // ... assertions on output
}
```

**DO NOT:**
- Call `os.Chdir()` - pollutes process-wide working directory
- Call `os.Setenv()` - pollutes process-wide environment
- Modify global variables

**Instead:**
- Use `setupTestRepo(t)` or `setupTestWorkspace(t)` for test isolation
- Pass env vars via `env.run(t, map[string]string{"KEY": "val"}, args...)`
- Test behavior through command output, not internal function calls

See `air_test.go` for the `testEnv` helper implementation.
