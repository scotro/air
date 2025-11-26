#!/bin/bash

# Concurrent AI Agent Workflow Helpers
# Source this file or add to your shell profile:
#   source /path/to/agent-helpers.sh

# Configuration
WORKTREE_DIR="worktrees"
PACKETS_DIR=".claude/packets"

# ============================================================================
# Project Initialization
# ============================================================================

# Initialize a project for concurrent agent workflow
# Usage: agent-init
agent-init() {
    # Check if we're in a git repo
    if ! git rev-parse --git-dir > /dev/null 2>&1; then
        echo "Error: Not a git repository. Run 'git init' first."
        return 1
    fi

    echo "Initializing project for concurrent AI agent workflow..."
    echo ""

    # Prompt for language
    echo "Select your primary language:"
    echo "  1) Go"
    echo "  2) TypeScript/JavaScript"
    echo "  3) Python"
    echo "  4) Rust"
    echo "  5) Other (no permissions preset)"
    echo ""
    printf "Choice [1-5]: "
    read -r lang_choice

    # Set up permissions based on language
    local permissions=""
    case "$lang_choice" in
        1)
            permissions='"Bash(go run*)", "Bash(go build*)", "Bash(go test*)", "Bash(go vet*)", "Bash(go mod*)", "Bash(golangci-lint*)"'
            echo "Setting up Go permissions..."
            ;;
        2)
            permissions='"Bash(npm*)", "Bash(npx*)", "Bash(pnpm*)", "Bash(yarn*)", "Bash(tsc*)", "Bash(node*)"'
            echo "Setting up TypeScript/JavaScript permissions..."
            ;;
        3)
            permissions='"Bash(python*)", "Bash(python3*)", "Bash(pip*)", "Bash(pip3*)", "Bash(pytest*)", "Bash(ruff*)", "Bash(uv*)"'
            echo "Setting up Python permissions..."
            ;;
        4)
            permissions='"Bash(cargo*)", "Bash(rustc*)", "Bash(rustfmt*)", "Bash(clippy*)"'
            echo "Setting up Rust permissions..."
            ;;
        *)
            permissions=""
            echo "No language-specific permissions set."
            ;;
    esac

    # Create directories
    mkdir -p .claude/packets
    mkdir -p .claude/commands
    echo "Created .claude/packets/ and .claude/commands/"

    # Create settings.json
    if [[ -n "$permissions" ]]; then
        cat > .claude/settings.json << EOF
{
    "permissions": {
        "allow": [
            $permissions
        ]
    }
}
EOF
        echo "Created .claude/settings.json with permissions"
    fi

    # Create workflow slash commands
    _create_workflow_commands

    # Add to .gitignore if not already present
    if [[ -f .gitignore ]]; then
        if ! grep -q "worktrees/" .gitignore 2>/dev/null; then
            echo "" >> .gitignore
            echo "# AI Agent Workflow" >> .gitignore
            echo "worktrees/" >> .gitignore
            echo ".claude/sessions/" >> .gitignore
            echo "Updated .gitignore"
        fi
    else
        cat > .gitignore << 'EOF'
# AI Agent Workflow
worktrees/
.claude/sessions/
EOF
        echo "Created .gitignore"
    fi

    # Create or update CLAUDE.md
    if [[ -f CLAUDE.md ]]; then
        if ! grep -q "Concurrent Workflow Support" CLAUDE.md 2>/dev/null; then
            echo "" >> CLAUDE.md
            _append_workflow_section >> CLAUDE.md
            echo "Added concurrent workflow section to CLAUDE.md"
        else
            echo "CLAUDE.md already has workflow section"
        fi
    else
        _create_claude_md
        echo "Created CLAUDE.md"
    fi

    echo ""
    echo "Project initialized! Next steps:"
    echo "  1. Edit CLAUDE.md to add project-specific details"
    echo "  2. Run 'packet-create <name>' to create work packets"
    echo "  3. Run 'agent-create <name>' to create agent worktrees"
    echo "  4. Run 'agent-session <n1> <n2>...' to launch agents"
}

# Internal: Create workflow slash commands
_create_workflow_commands() {
    # /workflow-setup command
    cat > .claude/commands/workflow-setup.md << 'EOF'
I want to set up concurrent agents to work on a task.

Ask me: **What do you want to build?**

Then ask: **Should I create the work packets automatically, or do you want to create them manually with `packet-create <name>`?**

If automatic: analyze my goal, create 2-4 packets in `.claude/packets/`, and output the agent-create and agent-session commands to run.

If manual: suggest packet names and what each should cover, then let me create them myself.
EOF

    # /workflow-status command
    cat > .claude/commands/workflow-status.md << 'EOF'
Check the status of all active agents in this workflow session.

Run these commands and summarize:
1. `git worktree list` - Show all active worktrees
2. For each agent worktree, show recent commits

Present a summary table:
| Agent | Branch | Last Commit | Status |
|-------|--------|-------------|--------|

Then suggest which agents might need attention based on commit activity.
EOF

    # /workflow-integrate command
    cat > .claude/commands/workflow-integrate.md << 'EOF'
Help me integrate completed agent work.

For each feature branch, I need to:
1. Review the changes (`git diff main..feature/<name>`)
2. Run tests
3. Merge to main

Please:
1. List all feature branches from agent worktrees
2. For each, provide the merge command
3. Remind me to run tests before and after merging
4. Provide cleanup commands for worktrees after merge

Integration checklist:
- [ ] All agents have committed their changes
- [ ] Tests pass on each branch
- [ ] Changes reviewed
- [ ] Merged to main
- [ ] Worktrees cleaned up
EOF

    echo "Created workflow slash commands in .claude/commands/"
}

# Internal: Append workflow section to existing CLAUDE.md
_append_workflow_section() {
    cat << 'EOF'

## Concurrent Workflow Support

### Work Packet Location
Active work packets are stored in `.claude/packets/`. Read your assigned packet before starting work.

### Boundary Enforcement
You are working in an isolated worktree. Do NOT modify files outside your packet's stated scope. If you need changes outside your boundaries, signal BLOCKED and explain what you need.

### Signaling
When blocked or done, clearly state your status:
- **BLOCKED:** [reason and what you need]
- **DONE:** [summary of completed work]

### Agent Completion Protocol
Before signaling DONE:
1. Ensure all tests pass
2. Run linter
3. Commit your changes with a descriptive message
4. Summarize files changed and decisions made

### Avoiding Merge Conflicts
You are one of several agents working in parallel. To avoid conflicts:
- Only create files within your packet's stated boundaries
- Put mocks/stubs in your own package directory, not shared locations
- Signal BLOCKED if you need changes outside your scope
EOF
}

# Internal: Create new CLAUDE.md
_create_claude_md() {
    cat > CLAUDE.md << 'EOF'
# CLAUDE.md

## Project Overview
[Describe your project here]

## Commands
- Build: `[your build command]`
- Test: `[your test command]`
- Lint: `[your lint command]`

## Code Style
[Describe coding conventions]

## Concurrent Workflow Support

### Work Packet Location
Active work packets are stored in `.claude/packets/`. Read your assigned packet before starting work.

### Boundary Enforcement
You are working in an isolated worktree. Do NOT modify files outside your packet's stated scope. If you need changes outside your boundaries, signal BLOCKED and explain what you need.

### Signaling
When blocked or done, clearly state your status:
- **BLOCKED:** [reason and what you need]
- **DONE:** [summary of completed work]

### Agent Completion Protocol
Before signaling DONE:
1. Ensure all tests pass
2. Run linter
3. Commit your changes with a descriptive message
4. Summarize files changed and decisions made

### Avoiding Merge Conflicts
You are one of several agents working in parallel. To avoid conflicts:
- Only create files within your packet's stated boundaries
- Put mocks/stubs in your own package directory, not shared locations
- Signal BLOCKED if you need changes outside your scope
EOF
}

# ============================================================================
# Worktree Management
# ============================================================================

# Create a new agent worktree
# Usage: agent-create <name> [base-branch]
agent-create() {
    local name="$1"
    local base="${2:-main}"
    
    if [[ -z "$name" ]]; then
        echo "Usage: agent-create <name> [base-branch]"
        return 1
    fi
    
    mkdir -p "$WORKTREE_DIR"
    git worktree add "$WORKTREE_DIR/agent-$name" -b "feature/$name" "$base"
    
    echo "Created worktree: $WORKTREE_DIR/agent-$name"
    echo "Branch: feature/$name"
    echo ""
    echo "To start agent:"
    echo "  cd $WORKTREE_DIR/agent-$name && claude"
}

# List all agent worktrees with status
# Usage: agent-list
agent-list() {
    echo "Active Agent Worktrees:"
    echo "========================"

    # Process worktrees without subshell to avoid zsh compatibility issues
    local worktree_output
    worktree_output=$(git worktree list)

    while IFS= read -r line; do
        # Skip lines that don't contain our agent worktrees
        case "$line" in
            *worktrees/agent-*) ;;
            *) continue ;;
        esac

        # Parse the line: /path/to/worktree  commit [branch]
        local wt_path wt_branch wt_name wt_status
        wt_path="${line%% *}"
        wt_branch="${line##*\[}"
        wt_branch="${wt_branch%\]*}"
        wt_name="${wt_path##*/}"

        # Check if claude is running in this worktree
        if pgrep -f "claude.*$wt_path" > /dev/null 2>&1; then
            wt_status="🟢 Running"
        else
            wt_status="⚪ Idle"
        fi

        printf "%-20s %-25s %s\n" "$wt_name" "$wt_branch" "$wt_status"
    done <<< "$worktree_output"
}

# Remove an agent worktree (after merging)
# Usage: agent-remove <name> [--force]
agent-remove() {
    local name="$1"
    local force="$2"
    
    if [[ -z "$name" ]]; then
        echo "Usage: agent-remove <name> [--force]"
        return 1
    fi
    
    local worktree="$WORKTREE_DIR/agent-$name"
    local branch="feature/$name"
    
    if [[ ! -d "$worktree" ]]; then
        echo "Worktree not found: $worktree"
        return 1
    fi
    
    if [[ "$force" == "--force" ]]; then
        git worktree remove "$worktree" --force
    else
        git worktree remove "$worktree"
    fi
    
    # Optionally delete the branch
    read -p "Delete branch $branch? [y/N] " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        git branch -d "$branch" 2>/dev/null || git branch -D "$branch"
    fi
}

# Open a tmux session with multiple agent worktrees
# Usage: agent-session <name1> <name2> ...
agent-session() {
    if [[ $# -lt 1 ]]; then
        echo "Usage: agent-session <name1> [name2] [name3] ..."
        return 1
    fi
    
    local session_name="agents-$(date +%H%M)"
    
    # Create new tmux session with first agent
    local first="$1"
    shift
    tmux new-session -d -s "$session_name" -c "$WORKTREE_DIR/agent-$first"
    tmux rename-window -t "$session_name" "$first"
    
    # Add windows for remaining agents
    for name in "$@"; do
        tmux new-window -t "$session_name" -n "$name" -c "$WORKTREE_DIR/agent-$name"
    done
    
    # Add a dashboard window
    tmux new-window -t "$session_name" -n "dashboard" -c "$(pwd)"
    
    # Attach to session
    tmux attach-session -t "$session_name"
}

# ============================================================================
# Work Packet Management
# ============================================================================

# Create a new work packet from template
# Usage: packet-create <name>
packet-create() {
    local name="$1"
    
    if [[ -z "$name" ]]; then
        echo "Usage: packet-create <name>"
        return 1
    fi
    
    mkdir -p "$PACKETS_DIR"
    local file="$PACKETS_DIR/$name.md"
    
    if [[ -f "$file" ]]; then
        echo "Packet already exists: $file"
        return 1
    fi
    
    cat > "$file" << 'EOF'
# Work Packet: NAME_PLACEHOLDER

**Objective:** [One sentence describing what "done" looks like]

**Branch:** `feature/NAME_PLACEHOLDER`  
**Worktree:** `worktrees/agent-NAME_PLACEHOLDER`

## Acceptance Criteria

- [ ] [Specific, verifiable condition]
- [ ] All existing tests pass
- [ ] New tests cover the changes
- [ ] No lint errors introduced

## Context

**Key Files:**
- `src/path/to/relevant/code`

**Background:**
[2-3 sentences of context]

**Technical Constraints:**
- [Constraint 1]

## Boundaries

**In Scope:**
- [What this agent SHOULD do]

**Out of Scope:**
- [What this agent should NOT touch]

## Signal Protocol

**Signal BLOCKED when:**
- Need a decision on [specific decision type]
- Encounter unexpected issues outside boundaries

**Signal DONE when:**
- All acceptance criteria met
- Ready for integration review
EOF
    
    # Replace placeholder (portable sed -i for macOS and Linux)
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' "s/NAME_PLACEHOLDER/$name/g" "$file"
    else
        sed -i "s/NAME_PLACEHOLDER/$name/g" "$file"
    fi

    echo "Created packet: $file"
    echo "Edit the packet, then run: agent-create $name"
}

# List all work packets
# Usage: packet-list
packet-list() {
    if [[ ! -d "$PACKETS_DIR" ]]; then
        echo "No packets directory found"
        return 1
    fi
    
    echo "Work Packets:"
    echo "============="
    for file in "$PACKETS_DIR"/*.md; do
        if [[ -f "$file" ]]; then
            local name=$(basename "$file" .md)
            local objective=$(grep "^\*\*Objective:\*\*" "$file" | sed 's/\*\*Objective:\*\* //')
            printf "%-20s %s\n" "$name" "$objective"
        fi
    done
}

# ============================================================================
# Session Helpers
# ============================================================================

# Initialize a new session
# Usage: session-init [name]
session-init() {
    local name="${1:-$(date +%Y-%m-%d)}"
    local file=".claude/sessions/$name.md"
    
    mkdir -p ".claude/sessions"
    
    cat > "$file" << EOF
# Session: $name

## Objectives

1. [Primary goal]
2. [Secondary goal]

## Work Packets

### Planned
- [ ] [Packet 1] - [description]
- [ ] [Packet 2] - [description]

### Dependency Map
\`\`\`
[Packet 1] ──→ [Packet 3]
[Packet 2] ──→ [Packet 3]
\`\`\`

## Round Log

### Setup Round ($(date +%H:%M))
- Created worktrees for: 
- Dispatched agents: 
- Notes: 

## Outcomes

**Completed:**
- 

**Deferred:**
- 

**Learnings:**
- 
EOF
    
    echo "Created session: $file"
    ${EDITOR:-vim} "$file"
}

# Quick status check across all agents
# Usage: agent-status
agent-status() {
    echo "Agent Status Check - $(date +%H:%M)"
    echo "=================================="
    
    for worktree in "$WORKTREE_DIR"/agent-*; do
        if [[ -d "$worktree" ]]; then
            local name=$(basename "$worktree")
            echo ""
            echo "[$name]"
            
            # Show recent git activity
            cd "$worktree"
            local changes=$(git diff --stat HEAD~1 2>/dev/null | tail -1)
            local last_commit=$(git log -1 --format="%ar: %s" 2>/dev/null)
            
            echo "  Last commit: $last_commit"
            echo "  Changes: $changes"
            cd - > /dev/null
        fi
    done
}

# ============================================================================
# Utility
# ============================================================================

# Print help
agent-help() {
    cat << 'EOF'
Concurrent AI Agent Workflow Helpers
=====================================

Project Setup:
  agent-init                  Initialize project for workflow (run once)

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

Quick Start:
  agent-init                  # First time: initialize project
  packet-create <name>        # Create work packets
  agent-create <name>         # Create worktrees
  agent-session <name>        # Launch in tmux

During Session:
  agent-status                # Check all agents
  agent-list                  # See running status

After Completion:
  git merge feature/<name>    # Merge from main directory
  agent-remove <name>         # Clean up worktrees
EOF
}
