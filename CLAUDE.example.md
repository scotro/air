# CLAUDE.md

Project configuration for Claude Code agents. This file provides persistent context across sessions.

## Project Overview

[Brief description of what this project does and its core purpose]

## Quick Commands

```bash
# Development
go run ./cmd/app       # Start development server
go build -o bin/app    # Production build
go test ./...          # Run test suite
golangci-lint run      # Run linter

# Database (if applicable)
go run ./cmd/migrate   # Run migrations
go run ./cmd/seed      # Seed development data
```

## Architecture

[2-3 sentences on high-level architecture. E.g., "This is a Go HTTP service with a PostgreSQL database. HTTP handlers live in /internal/handlers, business logic in /internal/services, and database access in /internal/db."]

### Key Directories

```
cmd/               # Application entry points
├── app/           # Main application
├── migrate/       # Migration CLI
└── seed/          # Seed data CLI
internal/          # Private application code
├── handlers/      # HTTP handlers
├── services/      # Business logic
├── db/            # Database queries and repositories
└── models/        # Data structures
migrations/        # SQL migration files
```

### Key Files

- `internal/services/auth.go` - Authentication logic
- `internal/db/db.go` - Database connection and utilities
- `internal/models/models.go` - Shared data structures

## Code Style

- Use Go idioms: short variable names in tight scope, longer names for wider scope
- Prefer returning errors over panicking
- Use context.Context for cancellation and request-scoped values
- Error handling: return errors up the stack, wrap with context using `fmt.Errorf`
- Tests: colocate with source files as `*_test.go`, use table-driven tests

## Testing Requirements

Before marking work complete:
1. All existing tests must pass: `go test ./...`
2. New functionality must have tests
3. No lint errors: `golangci-lint run`

## Git Conventions

- Branch naming: `feature/short-description`, `fix/issue-description`
- Commit messages: imperative mood, <72 chars ("Add user authentication")
- Squash commits before requesting review

---

## Concurrent Workflow Support

This project uses concurrent AI agents. Follow these protocols when working in a worktree.

### Your Assignment

Check `.claude/packets/` for your work packet. Read it completely before starting.

### Boundary Enforcement

You are working in an isolated worktree. **Do NOT modify files outside your packet's stated scope.** If you discover you need changes outside your boundaries:

1. Signal BLOCKED
2. Explain what change is needed and why
3. Wait for orchestrator to either expand your scope or assign to another agent

### Status Signaling

When your status changes, clearly state it at the start of your response:

```
**STATUS: RUNNING**
[Normal progress update]

**STATUS: BLOCKED**
Reason: [What you need]
Blocking: [What you cannot proceed with]
Can continue: [What you can work on in the meantime, if anything]

**STATUS: DONE**
Completed: [Summary of what was built]
Files changed: [List]
Tests: [Pass/Fail status]
Ready for: [Integration review]
```

### Pre-Completion Checklist

Before signaling DONE:

- [ ] All acceptance criteria from work packet met
- [ ] `go test ./...` passes
- [ ] `golangci-lint run` passes
- [ ] All changed files listed in status update
- [ ] Any decisions made are documented
- [ ] Any follow-up work identified is noted

### Coordination Files (Do Not Modify)

These files are managed by the human orchestrator:
- `.claude/packets/*` - Work packet definitions
- `.claude/sessions/*` - Session logs
- `.claude/dashboard.md` - Agent tracking

### Context Preservation

If you make architectural decisions or discover important context:
1. Document it in a code comment at the relevant location
2. Mention it in your DONE status so it can be added to project docs
3. Do NOT modify this CLAUDE.md file directly

---

## Common Patterns

### Error Handling

```go
// Define sentinel errors and error types
var ErrUserNotFound = errors.New("user not found")

type AuthError struct {
    Message string
    Code    string
}

func (e *AuthError) Error() string {
    return e.Message
}

// Return errors up the stack with context
func GetUser(ctx context.Context, id string) (*User, error) {
    user, err := db.FindUser(ctx, id)
    if err != nil {
        return nil, fmt.Errorf("get user %s: %w", id, err)
    }
    return user, nil
}

// Handle at boundary (HTTP handler)
func handleGetUser(w http.ResponseWriter, r *http.Request) {
    user, err := svc.GetUser(r.Context(), userID)
    if errors.Is(err, ErrUserNotFound) {
        writeError(w, "User not found", "USER_NOT_FOUND", http.StatusNotFound)
        return
    }
    if err != nil {
        writeError(w, "Internal error", "INTERNAL", http.StatusInternalServerError)
        return
    }
    writeJSON(w, user)
}
```

### Database Queries

```go
// Use sqlx or pgx for database access
func (r *UserRepo) FindByID(ctx context.Context, id string) (*User, error) {
    var user User
    err := r.db.GetContext(ctx, &user,
        "SELECT * FROM users WHERE id = $1", id)
    if err == sql.ErrNoRows {
        return nil, ErrUserNotFound
    }
    if err != nil {
        return nil, fmt.Errorf("query user: %w", err)
    }
    return &user, nil
}
```

### API Response Format

```go
// Success
func writeJSON(w http.ResponseWriter, data any) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]any{"data": data})
}

// Error
func writeError(w http.ResponseWriter, msg, code string, status int) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(map[string]string{
        "error": msg,
        "code":  code,
    })
}
```

---

## Things to Avoid

- Don't add new dependencies without explicit approval
- Don't modify database schema without migration
- Don't use `panic` for recoverable errors
- Don't ignore errors with `_`
- Don't leave `fmt.Println` debug statements in production code
- Don't commit `.env` files or secrets
