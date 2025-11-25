# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This repository contains a systematic workflow for managing multiple AI coding agents for professional software development. It provides methodology, templates, and shell utilities to orchestrate concurrent AI agents working on decomposed work packets using git worktrees.

Core philosophy: **Your attention is the bottleneck, not AI capacity.** The workflow optimizes for human cognitive bandwidth by batching supervisory work into discrete rounds rather than continuous context-switching.

## Repository Structure

```
.
├── README.md                # Project overview and quick start guide
├── WORKFLOW.md              # Core methodology and patterns
├── TEMPLATES.md             # Reusable templates for work packets, sessions, dashboards
├── CLAUDE.example.md        # Example CLAUDE.md for project repositories
├── EXAMPLE-WALKTHROUGH.md   # Detailed walkthrough of the workflow in practice
├── agent-helpers.sh         # Shell utilities for worktree/packet management
└── AGENT-HELPERS.md         # Complete documentation for shell utilities
```

## Key Concepts

### The Rounds Pattern

Work flows through three types of rounds:

1. **Setup Round (15-45 min)**: Decompose work into parallelizable packets, create worktrees, dispatch agents
2. **Execution Rounds (20-30 min)**: Check agent status, unblock with minimal intervention, update tracking
3. **Integration Round (30-60 min)**: Review completed work, run tests, merge, plan next cycle

### Dependency Classification

- **Independent**: No dependencies, fully parallelizable
- **Sequential**: Output of A required before B starts
- **Soft Dependencies**: Run in parallel with agreed interface contracts, reconcile later

### Git Worktree Structure

Each agent operates in an isolated worktree on its own feature branch:
```
project/
├── .git/                    # Shared git state
├── main/                    # Main working directory
├── worktrees/
│   ├── agent-auth/          # Agent 1: Authentication work
│   ├── agent-api/           # Agent 2: API development
│   └── agent-tests/         # Agent 3: Test coverage
```

## Shell Utilities

The `agent-helpers.sh` script provides commands for managing the workflow:

### Setup and Usage

```bash
# Load utilities
source agent-helpers.sh

# View all commands
agent-help
```

### Common Commands

```bash
# Worktree management
agent-create <name> [base-branch]   # Create new agent worktree on feature/<name>
agent-list                          # List all agent worktrees with status
agent-remove <name>                 # Remove worktree after merging
agent-session <n1> <n2>...          # Open tmux with multiple agents
agent-status                        # Quick status check across all agents

# Work packet management
packet-create <name>                # Create work packet from template
packet-list                         # List all work packets

# Session management
session-init [name]                 # Initialize session log (defaults to today's date)
```

### Typical Workflow

```bash
# 1. Create work packets
packet-create auth
packet-create api

# 2. Create corresponding worktrees
agent-create auth
agent-create api

# 3. Launch agents in tmux
agent-session auth api

# 4. During execution, check status periodically
agent-status
agent-list

# 5. After integration, clean up
agent-remove auth
agent-remove api
```

## Documentation Files

### README.md

Human-readable introduction to the project for newcomers. Includes:
- Problem/solution overview
- Quick start guide (install, setup, first session)
- Real-world example with metrics
- Core concepts (work packets, rounds pattern, dependencies)
- What's included in the repository
- Shell utilities summary
- Risk calibration table
- When to use this workflow
- Tips for success and anti-patterns

### WORKFLOW.md

Contains the complete methodology including:
- Core principles (attention bottleneck, context limits, isolation)
- Detailed rounds pattern explanation
- Dependency management strategies
- Environment setup with git worktrees
- Session workflow guidance
- Multi-project workflow options
- Risk calibration (autonomous/standard/high-touch modes)
- Anti-patterns to avoid

### TEMPLATES.md

Provides templates for:
- **Work Packet Template**: Structure for defining agent assignments with objectives, acceptance criteria, boundaries, interface contracts, and signal protocol
- **Tracking Dashboard Template**: Live status tracking for all active agents
- **Session Log Template**: Session planning and round-by-round logging
- **CLAUDE.md Additions**: Template for adding concurrent workflow support to project repositories
- **Quick Reference Card**: One-page printable reference

### CLAUDE.example.md

Example CLAUDE.md file showing how to configure a project repository to support concurrent agent workflows. Includes:
- Project-specific commands (dev, build, test, lint)
- Architecture overview and key directories
- Code style and testing requirements
- Concurrent workflow protocols (boundaries, signaling, coordination files)
- Common patterns and anti-patterns

### EXAMPLE-WALKTHROUGH.md

Detailed realistic example of building a user management feature with 3 concurrent agents over 3 hours. Shows:
- Work decomposition into db/api/tests packets
- Setup round with dependency mapping
- Multiple execution rounds with agent status checks
- Human interventions (decisions, unblocking)
- Integration round with merge review
- Key learnings and metrics

### AGENT-HELPERS.md

Complete reference documentation for the shell utilities. Includes:
- Installation (basic and persistent)
- Configuration options
- Command reference for all utilities (agent-create, agent-list, agent-remove, agent-session, agent-status, packet-create, packet-list, session-init, agent-help)
- Common workflows (starting sessions, execution rounds, integration)
- Troubleshooting guide
- Advanced usage (custom directories, base branches, scripting)
- Tips and best practices

## When Working on This Repository

### Editing Documentation

When updating workflow documentation:
- Keep README.md welcoming and focused on quick value for newcomers
- Keep WORKFLOW.md focused on methodology and patterns
- Keep TEMPLATES.md focused on reusable structures
- Ensure EXAMPLE-WALKTHROUGH.md stays realistic and practical
- Update CLAUDE.example.md to reflect any new workflow features
- Keep AGENT-HELPERS.md synchronized with changes to agent-helpers.sh

### Editing Shell Utilities

When modifying `agent-helpers.sh`:
- Follow bash best practices (quote variables, check arguments)
- Keep functions focused on single responsibilities
- Update `agent-help` documentation if adding/changing commands
- Update AGENT-HELPERS.md with any new commands or changed behavior
- Test worktree operations (create, list, remove) thoroughly
- Ensure compatibility with both bash and zsh

### Consistency Guidelines

- Use "agent" terminology consistently (not "worker" or "assistant")
- Use "work packet" for decomposed units of work
- Use "rounds" terminology for the supervision pattern
- Status emojis: 🟢 Running, 🟡 Blocked, 🔵 Done, 🔴 Drifting, ⚪ Idle/Paused
- Time estimates: Setup 15-45min, Execution 20-30min, Integration 30-60min

## Configuration Directories

The workflow uses these hidden directories (mentioned in templates):

```
.claude/
├── packets/         # Work packet definitions for active work
├── sessions/        # Session logs
└── dashboard.md     # Live agent tracking (during active sessions)
```

These are typically not committed to repositories using this workflow, but are created locally when running concurrent agent sessions.

## This is a Methodology Repository

This repository does not have:
- Build commands (no package.json, no Makefile)
- Tests to run (it's documentation, not code)
- Dependencies to install
- Development servers to start

Instead, it provides:
- A systematic approach to managing multiple AI agents
- Templates for planning and tracking
- Shell utilities to reduce mechanical overhead
- Examples demonstrating the workflow in practice
