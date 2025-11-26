# CLAUDE.md

## Project Overview

AIR (AI Runner) is a Go CLI tool that orchestrates multiple Claude Code agents working in parallel on decomposed tasks using git worktrees.

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
├── plan.go        # air plan, plan list, plan show
├── run.go         # air run
├── status.go      # air status
├── integrate.go   # air integrate
└── clean.go       # air clean
internal/          # (future) shared packages
```

## Key Concepts

- **Packets**: Work units defined in `.air/packets/*.md`
- **Context**: Workflow instructions in `.air/context.md`, injected via `--append-system-prompt`
- **Worktrees**: Isolated git worktrees in `.air/worktrees/` for parallel work
- **Branches**: Named `air/<packet-name>`

## Design Principles

1. Non-invasive: Never touch `.claude/` or `CLAUDE.md` in user projects
2. Wrapper: We wrap `claude` with context, we don't configure it
3. Our namespace: Everything lives in `.air/`
