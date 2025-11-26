# Agent Plan: [SHORT-NAME]

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
- `internal/path/to/relevant/code.go`
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
- [Adjacent work that belongs to another plan]

## Interface Contracts

[If this plan has soft dependencies with others, define the interface here]

```go
// Example: API contract this agent should implement/consume
type UserService interface {
    GetUser(ctx context.Context, id string) (*User, error)
    UpdateUser(ctx context.Context, id string, data *UserUpdate) (*User, error)
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
