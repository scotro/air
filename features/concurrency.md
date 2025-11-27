# Air Concurrency Support - Requirements Document

## Overview

Extend Air to support **concurrent plans** with explicit coordination between agents. Currently, Air supports parallel execution of independent plans. This enhancement adds the ability for agents to wait on and signal each other at pre-planned integration points.

## Goals

1. Enable agents to depend on work completed by other agents
2. Keep coordination explicit and pre-planned (no ad-hoc communication)
3. Maintain simplicity - leverage existing patterns, minimal new tooling
4. Preserve backward compatibility - independent plans continue to work unchanged

## Non-Goals

- Real-time agent-to-agent messaging
- Dynamic dependency discovery at runtime
- Automatic conflict resolution
- Complex workflow orchestration (DAG visualization, etc.)

---

## Concepts

### Channels

A **channel** is a named coordination point. Channels have simple semantics:

- **Single-signal**: A channel can only be signaled once
- **Latched**: Once signaled, the channel stays signaled (late waiters get the value immediately)
- **Broadcast**: Multiple agents can wait on the same channel

Channel state is stored as files in `~/.air/<project>/channels/`:
```
~/.air/<project>/channels/
├── core-ready.json
├── strings-ready.json
└── done/
    ├── core.json
    └── strings.json
```

### Channel Payload

When an agent signals a channel, it writes:
```json
{
  "sha": "abc12345",
  "branch": "air/agent-name",
  "worktree": "/home/user/.air/project/worktrees/agent-name",
  "agent": "agent-name",
  "timestamp": "2025-01-15T10:30:00Z"
}
```

### Integration Points

An **integration point** is where one agent's work flows into another's:
1. Producer agent commits work and signals a channel
2. Consumer agent waits on the channel
3. Consumer merges the branch into its worktree
4. Consumer continues with its plan

### Done Channels

Completion is signaled via a special `done/<agent-id>` channel. This allows `air status` to track which agents have finished.

---

## Plan File Format

Plans may include an optional **Dependencies** section. Plans without this section are independent (parallel-only, backward compatible).

```markdown
# Plan: strings

**Objective:** Implement SET and GET string commands

## Dependencies

**Waits on:**
- `core-ready` - Core command registry must exist before registering commands

**Signals:**
- `strings-ready` - Signal when string commands are implemented

**Sequence:**
1. Run `air agent wait core-ready` before beginning implementation
2. Run `air agent merge core-ready` to pull in core module
3. Implement string commands
4. Run `air agent signal strings-ready`
5. Run `air agent done`

## Boundaries
...

## Acceptance Criteria
...
```

### Design Principles for Plans

1. **Minimize integration points** - Each integration point is a potential conflict. Prefer independent work.
2. **Non-overlapping files** - Agents consuming the same channel should work on disjoint file sets.
3. **Early integration, late signaling** - Wait/merge early, signal late (after work is complete).

---

## CLI Commands

### User-Facing (unchanged)

```
air init          # Initialize project
air plan          # Start planning session
air run [plans]   # Launch agents
air status        # Show agent/channel status
air integrate     # Final merge
air clean         # Clean up worktrees
```

### Agent Commands (new)

Scoped under `air agent` to distinguish from user commands:

```
air agent wait <channel>        # Block until channel is signaled, print payload
air agent signal <channel>      # Signal channel with current HEAD commit
air agent merge <channel>       # Merge branch from channel's source
air agent done                  # Signal completion (done/<agent-id> channel)
```

#### `air agent wait <channel>`

- Blocks (polls) until `~/.air/<project>/channels/<channel>.json` exists
- Prints payload JSON to stdout on success
- Exit code 0 on success

#### `air agent signal <channel>`

- Requires `AIR_AGENT_ID` and `AIR_WORKTREE` environment variables
- Captures current HEAD SHA from worktree
- Creates `~/.air/<project>/channels/<channel>.json` with payload
- Fails if channel already signaled (single-signal semantics)
- Exit code 0 on success, non-zero if already signaled

#### `air agent merge <channel>`

- Reads payload from `~/.air/<project>/channels/<channel>.json`
- Merges the source branch into current worktree (includes transitive dependencies)
- Fails if merge has conflicts (user intervention required)
- Exit code 0 on success, non-zero on conflict

#### `air agent done`

- Equivalent to `air agent signal done/<agent-id>`
- Uses `AIR_AGENT_ID` environment variable

---

## Environment Variables

Set by `air run` for each agent:

| Variable | Description | Example |
|----------|-------------|---------|
| `AIR_AGENT_ID` | Plan/agent name | `strings` |
| `AIR_WORKTREE` | Absolute path to agent's worktree | `~/.air/project/worktrees/strings` |
| `AIR_PROJECT_ROOT` | Absolute path to main project | `/path/to/project` |
| `AIR_CHANNELS_DIR` | Path to channels directory | `~/.air/project/channels` |

---

## Directory Structure

```
~/.air/<project>/
├── context.md              # Agent context (injected to all agents)
├── plans/
│   ├── core.md
│   └── strings.md
├── channels/               # Channel state (created at runtime)
│   ├── core-ready.json
│   └── done/
│       ├── core.json
│       └── strings.json
└── worktrees/              # Git worktrees (created by air run)
    ├── core/
    └── strings/
```

---

## Context Updates

### Planner Context (`orchestrationContext` in plan.go)

Add guidance for the planner on:
- When to use concurrent vs parallel plans
- How to structure the Dependencies section
- Channel naming conventions
- Design principles (minimal integration, non-overlapping files)

### Agent Context (`contextTemplate` in init.go)

Add guidance for agents on:
- How to interpret the Dependencies section
- When and how to use `air agent` commands
- What to do on merge conflicts (signal BLOCKED)

---

## Error Handling

| Scenario | Behavior |
|----------|----------|
| `wait` on non-existent channel | Block until signaled (or timeout?) |
| `signal` on already-signaled channel | Error, exit non-zero |
| `merge` conflict | Error, exit non-zero, agent should signal BLOCKED |
| Missing environment variables | Error with helpful message |

### Open Question: Timeout

Should `air agent wait` have a timeout? Options:
1. No timeout - wait forever (agent can be killed manually)
2. Configurable timeout via flag or env var
3. Default timeout (e.g., 30 minutes) with override

**Recommendation:** Start with no timeout. Add later if needed.

---

## Status Display

`air status` should show:
- Which agents are running/done
- Which channels are signaled
- Dependency relationships (which agent is waiting on what)

Example output:
```
Agents:
  core      ✓ done
  strings   ● running (waiting on core-ready)
  lists     ● running

Channels:
  core-ready    ✓ signaled by core (abc1234)
  strings-ready ○ pending
  lists-ready   ○ pending
```

---

## Testing Strategy

1. **Unit tests for `air agent` commands**
   - Signal creates correct file
   - Wait blocks then returns when file exists
   - Cherry-pick applies commit correctly
   - Error cases (already signaled, missing env vars, conflicts)

2. **Integration test**
   - Create two plans with dependency
   - Run both agents
   - Verify coordination works end-to-end

---

## Migration / Backward Compatibility

- Plans without Dependencies section work exactly as before
- `air run` sets new env vars, but agents that don't use them are unaffected
- No changes to existing plan format (Dependencies is additive)

---

## Future Considerations (Out of Scope)

- **Cycle detection**: Validate dependency graph before running
- **Visualization**: Show dependency DAG in status
- **Auto-retry**: Retry merge with different strategy on conflict
- **Partial ordering**: Run independent subgraphs in parallel automatically
