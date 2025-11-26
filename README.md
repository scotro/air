# Concurrent AI Agent Workflow

Manage multiple Claude Code agents working in parallel on decomposed tasks using git worktrees.

## Install

```bash
git clone <repo-url> ~/ai-workflow
~/ai-workflow/install.sh
```

Then restart your terminal or `source ~/.zshrc`.

## Initialize a Project

```bash
cd ~/my-project
agent-init
```

This creates `.claude/` with settings and workflow commands.

## Create Work Packets

Use Claude to help decompose your work:

```
> /workflow-setup
```

Or create packets manually:

```bash
packet-create auth
packet-create api
# Edit .claude/packets/*.md with objectives and boundaries
```

## Launch Agents

```bash
agent-create auth
agent-create api
agent-session auth api
```

In each tmux window, start Claude and point it to the packet:

```
> Read .claude/packets/auth.md and implement it.
```

## Integrate

After agents signal DONE:

```bash
# From main project directory (not worktree)
git merge feature/auth
git merge feature/api
agent-remove auth
agent-remove api
```

## Commands

```
agent-init              Initialize project for workflow
agent-create <name>     Create agent worktree
agent-list              List agents with status
agent-remove <name>     Clean up after merge
agent-session <n>...    Launch agents in tmux
agent-status            Quick status check
packet-create <name>    Create work packet
packet-list             List packets
agent-help              Show all commands
```

## Learn More

- [WORKFLOW.md](WORKFLOW.md) - Full methodology
- [AGENT-HELPERS.md](AGENT-HELPERS.md) - Shell command docs

## License

MIT License. Use it however you want. No warranty.
