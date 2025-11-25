# Templates

Reusable templates for the concurrent AI agent workflow.

---

## Work Packet Template

Copy this for each unit of work you dispatch to an agent.

```markdown
# Work Packet: [SHORT-NAME]

**Objective:** [One sentence describing what "done" looks like]

**Branch:** `feature/[short-name]`  
**Worktree:** `worktrees/agent-[short-name]`

## Acceptance Criteria

- [ ] [Specific, verifiable condition]
- [ ] [Another condition]
- [ ] All existing tests pass
- [ ] New tests cover the changes
- [ ] No lint errors introduced

## Context

**Key Files:**
- `src/path/to/relevant/code.ts`
- `docs/relevant-documentation.md`

**Background:**
[2-3 sentences of context the agent needs to understand why this work matters]

**Technical Constraints:**
- [Constraint 1: e.g., Must use existing auth middleware]
- [Constraint 2: e.g., No new dependencies without approval]

## Boundaries

**In Scope:**
- [What this agent SHOULD do]

**Out of Scope:**
- [What this agent should NOT touch]
- [Adjacent work that belongs to another packet]

## Interface Contracts

[If this packet has soft dependencies with others, define the interface here]

```typescript
// Example: API contract this agent should implement/consume
interface UserService {
  getUser(id: string): Promise<User>;
  updateUser(id: string, data: Partial<User>): Promise<User>;
}
```

## Signal Protocol

**Signal BLOCKED when:**
- Need a decision on [specific decision type]
- Encounter unexpected [situation type]
- Tests reveal issues in code outside boundaries

**Signal DONE when:**
- All acceptance criteria met
- Ready for integration review

## Notes

[Any additional context, links to related issues, previous attempts, etc.]
```

---

## Tracking Dashboard Template

Maintain this file during active sessions to track all agents.

```markdown
# Session Dashboard

**Date:** YYYY-MM-DD  
**Session Goal:** [High-level objective for this session]

## Active Agents

| ID | Worktree | Packet | Status | Last Check | Progress | Notes |
|----|----------|--------|--------|------------|----------|-------|
| 1 | agent-auth | User Auth | 🟢 Running | 10:30 | ~60% | On track |
| 2 | agent-api | REST API | 🟡 Blocked | 10:30 | ~40% | Needs schema decision |
| 3 | agent-tests | Test Coverage | 🟢 Running | 10:15 | ~30% | Background task |

### Status Legend
- 🟢 Running: Agent is making progress
- 🟡 Blocked: Waiting on human input
- 🔵 Done: Ready for integration
- 🔴 Drifting: Needs redirection
- ⚪ Paused: Intentionally stopped

## Pending Decisions

| Decision | Blocking | Options | Decided |
|----------|----------|---------|---------|
| Auth token format | Agent 2 | JWT vs opaque | |
| Error response shape | Agent 2, 3 | RFC 7807 vs custom | |

## Completed This Session

| Packet | Agent | Branch | Merged | Notes |
|--------|-------|--------|--------|-------|
| | | | | |

## Carry Forward

[Work packets or decisions to address next session]
```

---

## Session Log Template

Start each session with this structure.

```markdown
# Session: YYYY-MM-DD

## Objectives

1. [Primary goal]
2. [Secondary goal]
3. [Stretch goal if time permits]

## Work Packets

### Planned
- [ ] [Packet 1 name] - [brief description]
- [ ] [Packet 2 name] - [brief description]

### Dependency Map
```
[Packet 1] ──→ [Packet 3]
[Packet 2] ──→ [Packet 3]
```

## Round Log

### Setup Round (HH:MM)
- Created worktrees for: [list]
- Dispatched agents: [list]
- Notes: [any setup issues]

### Execution Round 1 (HH:MM)
- Agent 1: [status, action taken]
- Agent 2: [status, action taken]

### Execution Round 2 (HH:MM)
- Agent 1: [status, action taken]
- Agent 2: [status, action taken]

### Integration Round (HH:MM)
- Merged: [branches]
- Issues found: [list]
- Remaining: [work for next session]

## Outcomes

**Completed:**
- [What got done]

**Deferred:**
- [What didn't get done and why]

**Learnings:**
- [What worked well]
- [What to adjust next time]
```

---

## CLAUDE.md Additions

Add this section to your project's CLAUDE.md to support concurrent workflows.

```markdown
## Concurrent Workflow Support

### Work Packet Location
Active work packets are stored in `.claude/packets/`. Read your assigned packet before starting work.

### Boundary Enforcement
You are working in an isolated worktree. Do NOT modify files outside your packet's stated scope. If you need changes outside your boundaries, signal BLOCKED and explain what you need.

### Signaling
When blocked or done, clearly state your status at the start of your response:

**BLOCKED:** [reason and what you need]
**DONE:** [summary of completed work and verification steps taken]

### Integration Preparation
Before signaling DONE:
1. Ensure all tests pass: `npm test` (or equivalent)
2. Run linter: `npm run lint` (or equivalent)  
3. Summarize all files changed
4. Note any decisions made that should be documented
5. List any follow-up work identified

### Coordination Files
Do not modify these files (they're managed by the human orchestrator):
- `.claude/packets/*`
- `.claude/session.md`
- `.claude/dashboard.md`
```

---

## Quick Reference Card

Print this or keep it visible during sessions.

```
┌─────────────────────────────────────────────────────────┐
│           CONCURRENT AI AGENT QUICK REFERENCE           │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  SETUP ROUND (15-45 min)                               │
│  □ Define session objectives                           │
│  □ Decompose into work packets                         │
│  □ Map dependencies                                     │
│  □ Create worktrees                                     │
│  □ Dispatch agents                                      │
│                                                         │
│  EXECUTION ROUND (every 20-30 min)                     │
│  For each agent:                                        │
│  □ Check: Running / Blocked / Done / Drifting          │
│  □ Unblock with minimal input                          │
│  □ Update dashboard                                     │
│                                                         │
│  INTEGRATION ROUND (30-60 min)                         │
│  □ Review against acceptance criteria                  │
│  □ Run integration tests                               │
│  □ Merge completed work                                │
│  □ Document decisions                                   │
│  □ Plan next cycle                                      │
│                                                         │
├─────────────────────────────────────────────────────────┤
│  AGENT STATUS CHECK PROMPTS                            │
│                                                         │
│  "Summarize your progress and any blockers"            │
│  "What files have you modified?"                       │
│  "Are you on track for the acceptance criteria?"       │
│  "What decisions have you made?"                       │
│                                                         │
├─────────────────────────────────────────────────────────┤
│  WORKTREE COMMANDS                                     │
│                                                         │
│  Create:  git worktree add worktrees/name -b branch    │
│  List:    git worktree list                            │
│  Remove:  git worktree remove worktrees/name           │
│                                                         │
├─────────────────────────────────────────────────────────┤
│  RISK CALIBRATION                                      │
│                                                         │
│  Low:  Docs, tests, linting    → check hourly          │
│  Med:  Features, refactoring   → check every 20-30 min │
│  High: Security, core logic    → check every 10-15 min │
│                                                         │
└─────────────────────────────────────────────────────────┘
```
