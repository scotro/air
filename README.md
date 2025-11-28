# Air - AI Runner

Orchestrate multiple Claude Code agents working in parallel on decomposed tasks.

## Requirements

- **Git** - for worktree management
- **tmux** - for running multiple agents in parallel
- **Claude Code** - the [Claude CLI](https://docs.anthropic.com/en/docs/claude-code)

## Install

### Quick install (recommended)

```bash
curl -sSL https://raw.githubusercontent.com/scotro/air/main/install.sh | sh
```

Installs to `~/.local/bin` by default. Add it to your PATH if needed:

```bash
export PATH="$PATH:$HOME/.local/bin"
```

### Manual download

Download the binary for your platform from [GitHub Releases](https://github.com/scotro/air/releases).

### From source (requires Go)

```bash
go install github.com/scotro/air/cmd/air@latest
```

## Getting Started

Follow the **[Tutorial](tutorial-gateway.md)** to build a working API gateway with a simple observability dashboard in 30 minutes using parallel agents. You'll learn the full workflow: `init` → `plan` → `run` → `integrate` → `clean`.

## Usage

### Initialize a project

```bash
cd ~/my-project
air init
```

#### Multi-repo workspaces

Air supports coordinating work across multiple repositories:

```
~/my-workspace/
├── schema/             # Git repo
├── sdk/                # Git repo
├── project-a/          # Git repo
└── project-b/          # Git repo
```

```bash
cd ~/my-workspace
air init
```

Air auto-detects repos as direct children and enables cross-repo planning and coordination.


### Plan work

```bash
air plan
```

Claude helps decompose your work into parallelizable plans stored in `~/.air/<project>/plans/`.

```bash
air plan list            # View plans
air plan show <name>     # View specific plan
air plan archive <name>  # Archive a plan
air plan restore <name>  # Restore archived plan
```

### Run agents

```bash
air run <plan1> <plan2> ...
air run all           # Run all plans
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

1. `air plan` launches Claude with orchestration context to create plans
2. `air run` creates isolated git worktrees and starts agents in tmux
3. Each agent receives workflow context via `--append-system-prompt`
4. Agents work on their plans, signal DONE when complete
5. `air integrate` helps merge completed work back to main

## Directory structure

```
~/.air/<project>/
├── context.md      # Workflow instructions (injected to all agents)
├── plans/          # Plan definitions
├── channels/       # Coordination signals for concurrent plans
└── worktrees/      # Git worktrees for each agent
```

## License

MIT
