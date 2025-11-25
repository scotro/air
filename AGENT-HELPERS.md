# Agent Helpers Shell Utilities

Complete reference for the `agent-helpers.sh` shell utilities that automate the mechanical aspects of the concurrent AI agent workflow.

## Installation

### Basic Installation

```bash
# Clone the repository
git clone <repo-url> ~/ai-workflow

# Load utilities in current shell
source ~/ai-workflow/agent-helpers.sh
```

### Persistent Installation

Add to your shell profile (~/.zshrc or ~/.bashrc):

```bash
# For zsh
echo "source ~/ai-workflow/agent-helpers.sh" >> ~/.zshrc
source ~/.zshrc

# For bash
echo "source ~/ai-workflow/agent-helpers.sh" >> ~/.bashrc
source ~/.bashrc
```

Verify installation:
```bash
agent-help
```

## Configuration

The utilities use these default directories (configurable by editing the script):

```bash
WORKTREE_DIR="worktrees"        # Where agent worktrees are created
PACKETS_DIR=".claude/packets"   # Where work packets are stored
```

## Command Reference

### Worktree Management

#### agent-create

Create a new agent worktree with an isolated git branch.

**Usage:**
```bash
agent-create <name> [base-branch]
```

**Parameters:**
- `name` (required): Short identifier for the agent/feature (e.g., "auth", "api")
- `base-branch` (optional): Branch to base the new branch on (default: "main")

**Creates:**
- Worktree at: `worktrees/agent-<name>`
- Feature branch: `feature/<name>`

**Example:**
```bash
# Create agent for authentication work
agent-create auth

# Create agent based on develop branch
agent-create api develop
```

**Output:**
```
Created worktree: worktrees/agent-auth
Branch: feature/auth

To start agent:
  cd worktrees/agent-auth && claude
```

**What it does:**
1. Creates `worktrees/` directory if it doesn't exist
2. Runs `git worktree add` to create isolated working tree
3. Creates and checks out `feature/<name>` branch
4. Displays instructions for launching agent

---

#### agent-list

List all active agent worktrees with their status.

**Usage:**
```bash
agent-list
```

**Output:**
```
Active Agent Worktrees:
========================
agent-auth           feature/auth              🟢 Running
agent-api            feature/api               ⚪ Idle
agent-tests          feature/tests             🟢 Running
```

**Status Indicators:**
- 🟢 Running: Claude process detected in worktree
- ⚪ Idle: No active Claude process

**Note:** Status detection is heuristic-based (checks for `claude` process in the worktree path) and may not always be 100% accurate.

---

#### agent-remove

Remove an agent worktree after work is completed and merged.

**Usage:**
```bash
agent-remove <name> [--force]
```

**Parameters:**
- `name` (required): Agent identifier (without "agent-" prefix)
- `--force` (optional): Force removal even if there are uncommitted changes

**Interactive prompts:**
- Asks whether to delete the feature branch after removing worktree

**Example:**
```bash
# Normal removal (will fail if uncommitted changes)
agent-remove auth

# Force removal
agent-remove auth --force
```

**Interactive session:**
```
Delete branch feature/auth? [y/N] y
```

**What it does:**
1. Removes the worktree at `worktrees/agent-<name>`
2. Prompts whether to delete the feature branch
3. Deletes branch with `-d` (safe) or `-D` (force) if confirmed

**Best practice:** Only remove after merging to main and verifying the merge.

---

#### agent-session

Open a tmux session with multiple agent worktrees in separate windows.

**Usage:**
```bash
agent-session <name1> [name2] [name3] ...
```

**Parameters:**
- `name1, name2, ...` (required): Agent identifiers (at least one)

**Example:**
```bash
# Launch three agents in tmux
agent-session auth api tests
```

**What it creates:**
- Tmux session named `agents-HHMM` (e.g., `agents-1430`)
- One window per agent, named after the agent
- Each window starts in the agent's worktree directory
- Additional "dashboard" window in the project root

**Tmux window layout:**
```
agents-1430
├─ auth          (worktrees/agent-auth/)
├─ api           (worktrees/agent-api/)
├─ tests         (worktrees/agent-tests/)
└─ dashboard     (project root)
```

**Working with tmux:**
```bash
# Switch windows: Ctrl-b, then number (0, 1, 2...)
# Or: Ctrl-b, then 'n' for next, 'p' for previous

# Detach from session: Ctrl-b, then 'd'

# Reattach to session:
tmux attach-session -t agents-1430

# List sessions:
tmux list-sessions

# Kill session:
tmux kill-session -t agents-1430
```

**Note:** Requires tmux to be installed. Install with:
```bash
# macOS
brew install tmux

# Ubuntu/Debian
sudo apt install tmux
```

---

#### agent-status

Quick status check across all agents showing recent git activity.

**Usage:**
```bash
agent-status
```

**Output:**
```
Agent Status Check - 14:30
==================================

[agent-auth]
  Last commit: 5 minutes ago: Implement JWT token validation
  Changes: 3 files changed, 145 insertions(+), 12 deletions(-)

[agent-api]
  Last commit: 2 minutes ago: Add user CRUD endpoints
  Changes: 5 files changed, 287 insertions(+), 5 deletions(-)

[agent-tests]
  Last commit: 10 minutes ago: Add integration test setup
  Changes: 2 files changed, 98 insertions(+)
```

**What it shows:**
- Timestamp of status check
- For each agent:
  - Last commit time and message
  - Git diff stats since previous commit

**Use case:** Quick scan during execution rounds to see which agents are making progress.

---

### Work Packet Management

#### packet-create

Create a new work packet from the standard template.

**Usage:**
```bash
packet-create <name>
```

**Parameters:**
- `name` (required): Work packet identifier (e.g., "auth", "api")

**Creates:**
- File at: `.claude/packets/<name>.md`
- Populated from template with placeholders replaced

**Example:**
```bash
packet-create auth
```

**Output:**
```
Created packet: .claude/packets/auth.md
Edit the packet, then run: agent-create auth
```

**Template structure:**
The created file includes sections for:
- Objective
- Branch and worktree information
- Acceptance criteria
- Context (key files, background, constraints)
- Boundaries (in scope / out of scope)
- Signal protocol (when to signal BLOCKED or DONE)

**Next steps:**
1. Edit the packet file to fill in details
2. Create corresponding worktree with `agent-create`
3. Dispatch agent with the packet as initial context

**Error handling:**
```bash
packet-create auth
# Output: Packet already exists: .claude/packets/auth.md
```

Won't overwrite existing packets.

---

#### packet-list

List all work packets with their objectives.

**Usage:**
```bash
packet-list
```

**Output:**
```
Work Packets:
=============
auth                 Implement JWT-based authentication system
api                  Create REST API endpoints for user management
tests                Comprehensive test coverage for user features
```

**What it shows:**
- Packet name (filename without .md)
- Objective extracted from the `**Objective:**` line

**Use case:** Quick overview of all planned work packets during setup round.

**Note:** Returns error if `.claude/packets/` directory doesn't exist yet.

---

### Session Management

#### session-init

Initialize a new session log from template.

**Usage:**
```bash
session-init [name]
```

**Parameters:**
- `name` (optional): Session name (default: today's date YYYY-MM-DD)

**Creates:**
- File at: `.claude/sessions/<name>.md`
- Session log with timestamp in Setup Round section
- Opens file in editor (uses $EDITOR environment variable, defaults to vim)

**Example:**
```bash
# Create session with today's date
session-init

# Create named session
session-init user-management-sprint
```

**Template structure:**
- Objectives section
- Work Packets (planned, dependency map)
- Round Log (Setup, Execution rounds, Integration)
- Outcomes (completed, deferred, learnings)

**Workflow:**
1. Run `session-init` at start of work session
2. Fill in objectives and planned packets
3. Update Round Log throughout session
4. Fill in Outcomes at end of session

**Use case:** Maintain history of what was accomplished in each work session for later reference.

---

### Utility Commands

#### agent-help

Display help text with all available commands.

**Usage:**
```bash
agent-help
```

**Output:**
```
Concurrent AI Agent Workflow Helpers
=====================================

Worktree Management:
  agent-create <name> [base]  Create new agent worktree
  agent-list                  List all agent worktrees
  agent-remove <name>         Remove agent worktree
  agent-session <n1> <n2>...  Open tmux with multiple agents
  agent-status                Quick status check across agents

Work Packets:
  packet-create <name>        Create new work packet from template
  packet-list                 List all work packets

Sessions:
  session-init [name]         Initialize a new session log

Examples:
  # Start a new feature with two parallel agents
  packet-create auth
  packet-create api
  agent-create auth
  agent-create api
  agent-session auth api

  # Check on agents during execution round
  agent-status
  agent-list
```

---

## Common Workflows

### Starting a New Concurrent Session

```bash
# 1. Initialize session
session-init

# 2. Create work packets
packet-create auth
packet-create api
packet-create tests

# 3. Edit packets to define work
$EDITOR .claude/packets/auth.md
$EDITOR .claude/packets/api.md
$EDITOR .claude/packets/tests.md

# 4. Create worktrees
agent-create auth
agent-create api
agent-create tests

# 5. Launch agents in tmux
agent-session auth api tests

# 6. In each tmux window, start Claude and paste the work packet
```

### Execution Round Check-in

```bash
# Quick status across all agents
agent-status

# See which agents are active
agent-list

# In each agent terminal, ask for status
# "Summarize your progress and any blockers"
```

### Integration and Cleanup

```bash
# 1. Review each completed agent
cd worktrees/agent-auth
git diff main --stat
npm test && npm run lint
git log main..HEAD

# 2. Switch to main and merge
cd ../..  # Back to project root
git checkout main
git merge feature/auth
git push

# 3. Clean up worktree
agent-remove auth

# 4. Repeat for other completed agents
```

### Mid-Session: Adding a New Agent

```bash
# Create packet for newly identified work
packet-create permissions
$EDITOR .claude/packets/permissions.md

# Create worktree
agent-create permissions

# Launch in new tmux window
cd worktrees/agent-permissions
claude
```

---

## Troubleshooting

### Issue: "fatal: invalid reference: feature/name"

**Cause:** Git can't create the branch (might already exist or invalid name)

**Solution:**
```bash
# Check existing branches
git branch -a

# Use different name or delete old branch
git branch -D feature/old-name
```

---

### Issue: "worktree already exists"

**Cause:** Worktree directory already exists from previous session

**Solution:**
```bash
# Remove the worktree first
agent-remove old-name

# Or manually remove
git worktree remove worktrees/agent-old-name
rm -rf worktrees/agent-old-name
```

---

### Issue: "Cannot remove worktree, uncommitted changes"

**Cause:** Agent worktree has uncommitted work

**Solution:**
```bash
# Option 1: Commit the changes
cd worktrees/agent-name
git add .
git commit -m "Work in progress"
cd ../..
agent-remove name

# Option 2: Force removal (loses changes!)
agent-remove name --force
```

---

### Issue: agent-status shows no output

**Cause:** No agent worktrees exist yet

**Solution:**
```bash
# Create some agents first
agent-create auth
agent-create api

# Or check if worktrees directory exists
ls -la worktrees/
```

---

### Issue: tmux command not found

**Cause:** tmux not installed

**Solution:**
```bash
# macOS
brew install tmux

# Ubuntu/Debian
sudo apt install tmux

# Or launch agents manually without tmux
cd worktrees/agent-auth && claude &
cd worktrees/agent-api && claude &
```

---

## Advanced Usage

### Custom Worktree Directory

Edit the script to change default location:

```bash
# In agent-helpers.sh, change:
WORKTREE_DIR="worktrees"

# To:
WORKTREE_DIR="../shared-worktrees"
```

Then reload:
```bash
source ~/ai-workflow/agent-helpers.sh
```

---

### Base Branch Other Than Main

```bash
# Create agent from develop branch
agent-create feature1 develop

# Create agent from another feature branch
agent-create feature2 feature/base-feature
```

---

### Multiple Projects

```bash
# Keep separate worktree directories per project
cd ~/project-a
WORKTREE_DIR="worktrees-a" agent-create auth

cd ~/project-b
WORKTREE_DIR="worktrees-b" agent-create api
```

Or use separate worktree directories:
```bash
# Edit agent-helpers.sh per project
# Or pass as environment variable
WORKTREE_DIR=my-worktrees agent-create auth
```

---

### Scripting with Helpers

```bash
#!/bin/bash
source ~/ai-workflow/agent-helpers.sh

# Automated setup
AGENTS=("auth" "api" "tests")

for agent in "${AGENTS[@]}"; do
    packet-create "$agent"
    agent-create "$agent"
done

agent-session "${AGENTS[@]}"
```

---

## Tips and Best Practices

### Naming Conventions

Use short, descriptive names:
- ✅ `auth`, `api`, `tests`, `db-schema`
- ❌ `the-authentication-feature`, `fix`, `work`

### When to Create vs Remove

**Create worktrees:**
- At start of setup round
- When new work is identified mid-session

**Remove worktrees:**
- After successful merge to main
- When work is abandoned (use --force)
- At end of session for incomplete work (commit first!)

### Monitoring Strategy

**During active development:**
```bash
# Every 20-30 minutes
agent-status
agent-list
```

**For low-risk background tasks:**
```bash
# Every hour
agent-status
```

### Integration Best Practices

Always test before removing:
```bash
cd worktrees/agent-name
npm test
npm run lint
npm run typecheck  # if applicable
git diff main --stat

# Only then:
cd ../..
git merge feature/name
agent-remove name
```

---

## See Also

- [WORKFLOW.md](WORKFLOW.md) - Complete methodology
- [TEMPLATES.md](TEMPLATES.md) - Work packet and session templates
- [EXAMPLE-WALKTHROUGH.md](EXAMPLE-WALKTHROUGH.md) - Realistic example
- [README.md](README.md) - Project overview and quick start
