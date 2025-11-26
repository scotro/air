# AIR - AI Runner

Orchestrate multiple Claude Code agents working in parallel on decomposed tasks.

## Install

```bash
go install github.com/scotro/ai-runner/cmd/air@latest
```

Requires `$GOPATH/bin` (usually `~/go/bin`) in your PATH.

## Usage

### Initialize a project

```bash
cd ~/my-project
air init
```

Creates `.air/` directory. Does not touch `.claude/` or `CLAUDE.md`.

### Plan work packets

```bash
air plan
```

Claude helps decompose your work into parallelizable packets stored in `.air/packets/`.

```bash
air plan list            # View packets
air plan show <name>     # View specific packet
air plan archive <name>  # Archive a packet
air plan restore <name>  # Restore archived packet
```

### Run agents

```bash
air run <packet1> <packet2> ...
air run all           # Run all packets
```

Creates worktrees, starts tmux session, launches Claude agents automatically.

### Monitor and integrate

```bash
air status            # Check agent progress
air integrate         # Guide through merging
air clean             # Remove all worktrees
air clean <name>      # Remove specific worktree
```

## How it works

1. `air plan` launches Claude with orchestration context to create work packets
2. `air run` creates isolated git worktrees and starts agents in tmux
3. Each agent receives workflow context via `--append-system-prompt`
4. Agents work on their packets, signal DONE when complete
5. `air integrate` helps merge completed work back to main

## Directory structure

```
.air/               # Entire directory is gitignored
├── context.md      # Workflow instructions (injected to all agents)
├── packets/        # Work packet definitions
└── worktrees/      # Git worktrees for each agent
```

## License

MIT
