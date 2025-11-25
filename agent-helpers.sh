#!/bin/bash

# Concurrent AI Agent Workflow Helpers
# Source this file or add to your shell profile:
#   source /path/to/agent-helpers.sh

# Configuration
WORKTREE_DIR="worktrees"
PACKETS_DIR=".claude/packets"

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
    git worktree list | grep -E "worktrees/agent-" | while read -r line; do
        local path=$(echo "$line" | awk '{print $1}')
        local branch=$(echo "$line" | awk '{print $3}' | tr -d '[]')
        local name=$(basename "$path")
        
        # Check if claude is running in this worktree (rough heuristic)
        if pgrep -f "claude.*$path" > /dev/null 2>&1; then
            status="🟢 Running"
        else
            status="⚪ Idle"
        fi
        
        printf "%-20s %-25s %s\n" "$name" "$branch" "$status"
    done
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
    
    # Replace placeholder
    sed -i "s/NAME_PLACEHOLDER/$name/g" "$file"
    
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
EOF
}

echo "Agent helpers loaded. Run 'agent-help' for commands."
