# Concurrent AI Agent Workflow

**A systematic approach to managing multiple AI coding agents for professional software development.**

## The Problem

When working with AI coding assistants like Claude Code, you're limited by your attention, not the AI's capacity. Context-switching between multiple ongoing tasks destroys productivity, and trying to supervise AI work in real-time creates a bottleneck.

## The Solution

This workflow treats AI agents like capable junior engineers working in parallel:
- Each agent gets a clearly defined work packet with explicit boundaries
- Agents work in isolated git worktrees on separate branches
- You supervise in batches (rounds) rather than hovering continuously
- Work is integrated systematically after verification

**Result:** Complete 3+ parallel work streams in the time it would take to do one sequentially.

## Quick Start

### 1. Install the Helpers

```bash
# Clone this repository
git clone <repo-url> ~/ai-workflow
cd ~/ai-workflow

# Load the shell utilities
source agent-helpers.sh

# Add to your shell profile for persistence
echo "source ~/ai-workflow/agent-helpers.sh" >> ~/.zshrc  # or ~/.bashrc
```

### 2. Set Up Your First Session

```bash
# In your project directory
cd ~/my-project

# Create work packets for parallel tasks
packet-create auth       # Authentication feature
packet-create api        # REST API endpoints

# Edit the packets to define objectives and boundaries
$EDITOR .claude/packets/auth.md
$EDITOR .claude/packets/api.md

# Create isolated worktrees for each agent
agent-create auth
agent-create api

# Launch agents in tmux
agent-session auth api
```

### 3. Run Execution Rounds

Every 20-30 minutes:
```bash
agent-status    # Quick check on all agents
agent-list      # See which are running/idle
```

In each agent's terminal, ask:
```
"Summarize your progress and any blockers"
```

Unblock agents with minimal input, then let them continue.

### 4. Integration Round

When agents signal completion:
```bash
# Review each agent's work
cd worktrees/agent-auth
git diff main --stat
npm test && npm run lint

# Merge completed work
git checkout main
git merge feature/auth

# Clean up
agent-remove auth
```

## Real-World Example

See [EXAMPLE-WALKTHROUGH.md](EXAMPLE-WALKTHROUGH.md) for a detailed walkthrough of building a user management feature with 3 concurrent agents in 3 hours.

**Metrics from that session:**
- 3 agents working in parallel
- 4 execution rounds
- 1 integration round
- Only 2 meaningful human interventions needed
- Database, API, and tests completed concurrently

## Core Concepts

### Work Packets

A work packet is a unit of work for one agent. It includes:
- **Objective**: What "done" looks like in one sentence
- **Acceptance Criteria**: Specific, verifiable conditions
- **Boundaries**: What's in scope and out of scope
- **Interface Contracts**: For agents with soft dependencies

Template: [TEMPLATES.md](TEMPLATES.md)

### The Rounds Pattern

**Setup Round (15-45 min)**: Decompose work, create worktrees, dispatch agents

**Execution Rounds (20-30 min)**: Check status, unblock agents, update tracking
- Run every 20-30 minutes
- Provide decisions as constraints, not discussions
- Defer non-critical questions

**Integration Round (30-60 min)**: Review, test, merge, plan next cycle

### Dependencies

**Independent**: No dependencies, fully parallelizable
```
[Auth Service] ←──────→ [Email Templates]
```

**Sequential**: Must complete in order
```
[Database Schema] → [Repository] → [API]
```

**Soft Dependencies**: Can run in parallel with agreed contracts
```
[API Design] ~~~~ [Frontend Components]
(both proceed with agreed interface, reconcile later)
```

## What's Included

| File | Purpose |
|------|---------|
| [WORKFLOW.md](WORKFLOW.md) | Complete methodology and patterns |
| [TEMPLATES.md](TEMPLATES.md) | Work packet, session, dashboard templates |
| [EXAMPLE-WALKTHROUGH.md](EXAMPLE-WALKTHROUGH.md) | Detailed realistic example |
| [CLAUDE.example.md](CLAUDE.example.md) | Template for project CLAUDE.md files |
| [agent-helpers.sh](agent-helpers.sh) | Shell utilities for automation |
| [AGENT-HELPERS.md](AGENT-HELPERS.md) | Shell utilities documentation |

## Shell Utilities

The `agent-helpers.sh` script provides commands to reduce mechanical overhead:

```bash
# Worktree management
agent-create <name>       # Create agent worktree
agent-list                # List all agents with status
agent-remove <name>       # Clean up after merge
agent-session <n1> <n2>   # Launch multiple agents in tmux

# Work packets
packet-create <name>      # Create from template
packet-list               # Show all packets

# Sessions
session-init [name]       # Initialize session log

# Monitoring
agent-status              # Check all agents quickly
agent-help                # Show all commands
```

See [AGENT-HELPERS.md](AGENT-HELPERS.md) for complete documentation.

## Risk Calibration

Adjust supervision intensity based on risk:

| Risk Level | Work Type | Check Frequency |
|------------|-----------|-----------------|
| **Low** | Docs, tests, linting | Every hour |
| **Medium** | Features, refactoring | Every 20-30 min |
| **High** | Security, core logic, migrations | Every 10-15 min |

## When to Use This Workflow

**Good fit:**
- Building features with multiple parallel work streams
- Large refactorings that can be decomposed
- Expanding test coverage across multiple modules
- Documentation updates across multiple areas

**Not needed:**
- Single, straightforward tasks
- Exploratory work without clear decomposition
- Very small projects

## Tips for Success

1. **Invest in setup**: A clear 20-minute setup round enables hours of parallel work
2. **Write specific packets**: Vague boundaries cause drift and wasted tokens
3. **Start with 2-3 agents**: Master supervision before scaling up
4. **Don't hover**: Checking every 5 minutes breaks your focus and doesn't help agents
5. **Integration is where quality happens**: Don't skip review rounds

## Anti-Patterns to Avoid

❌ **Under-specifying work packets** - Causes agent drift
❌ **Too many agents at once** - Degrades supervision quality
❌ **Skipping integration rounds** - Compounds errors
❌ **No clear boundaries** - Agents touch everything
❌ **Ignoring drift early** - Wastes time and tokens

## Adding to Your Projects

To use this workflow in your projects:

1. Copy the relevant sections from [CLAUDE.example.md](CLAUDE.example.md) to your project's `CLAUDE.md`
2. Add concurrent workflow protocols (boundaries, signaling, coordination files)
3. Customize the quick commands and architecture sections for your project

## Philosophy

> **Your attention is the bottleneck, not AI capacity.**

This workflow optimizes for human cognitive bandwidth by:
- Batching supervisory work into discrete rounds (not continuous context-switching)
- Isolating agents to prevent interference
- Treating agents like capable junior engineers (clear direction, explicit boundaries, regular check-ins)
- Making supervision async (agents work while you focus elsewhere)

## Learn More

- **Methodology**: Read [WORKFLOW.md](WORKFLOW.md) for the complete approach
- **Templates**: Browse [TEMPLATES.md](TEMPLATES.md) for all reusable structures
- **Example**: Walk through [EXAMPLE-WALKTHROUGH.md](EXAMPLE-WALKTHROUGH.md) to see it in action
- **Shell Utilities**: See [AGENT-HELPERS.md](AGENT-HELPERS.md) for command reference

## License

[Add your license here]

## Contributing

[Add contribution guidelines here]
