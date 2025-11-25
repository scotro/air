# CLAUDE.md

Project configuration for Claude Code agents. This file provides persistent context across sessions.

## Project Overview

[Brief description of what this project does and its core purpose]

## Quick Commands

```bash
# Development
npm run dev          # Start development server
npm run build        # Production build
npm test             # Run test suite
npm run lint         # Run linter
npm run typecheck    # TypeScript checks

# Database (if applicable)
npm run db:migrate   # Run migrations
npm run db:seed      # Seed development data
```

## Architecture

[2-3 sentences on high-level architecture. E.g., "This is a Next.js application with a PostgreSQL database. API routes live in /app/api, business logic in /lib, and React components in /components."]

### Key Directories

```
src/
├── app/           # Next.js app router pages and API routes
├── components/    # React components (UI in /ui, features in /features)
├── lib/           # Business logic and utilities
├── db/            # Database schema, migrations, queries
└── types/         # TypeScript type definitions
```

### Key Files

- `src/lib/auth.ts` - Authentication logic
- `src/lib/db.ts` - Database connection and utilities
- `src/types/index.ts` - Shared type definitions

## Code Style

- Use TypeScript strict mode
- Prefer named exports over default exports
- Use async/await over .then() chains
- Error handling: throw typed errors, catch at boundaries
- Tests: colocate with source files as `*.test.ts`

## Testing Requirements

Before marking work complete:
1. All existing tests must pass: `npm test`
2. New functionality must have tests
3. No TypeScript errors: `npm run typecheck`
4. No lint errors: `npm run lint`

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
- [ ] `npm test` passes
- [ ] `npm run lint` passes  
- [ ] `npm run typecheck` passes
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

```typescript
// Define typed errors
class AuthenticationError extends Error {
  constructor(message: string, public code: string) {
    super(message);
    this.name = 'AuthenticationError';
  }
}

// Throw at source
if (!user) {
  throw new AuthenticationError('User not found', 'USER_NOT_FOUND');
}

// Catch at boundary (API route, page component)
try {
  await authenticateUser(credentials);
} catch (error) {
  if (error instanceof AuthenticationError) {
    return Response.json({ error: error.message, code: error.code }, { status: 401 });
  }
  throw error; // Re-throw unexpected errors
}
```

### Database Queries

```typescript
// Use the query builder, not raw SQL
import { db } from '@/lib/db';
import { users } from '@/db/schema';

const user = await db.query.users.findFirst({
  where: eq(users.id, userId),
  with: { profile: true }
});
```

### API Response Format

```typescript
// Success
return Response.json({ data: result });

// Error
return Response.json({ 
  error: 'Human readable message',
  code: 'MACHINE_READABLE_CODE' 
}, { status: 400 });
```

---

## Things to Avoid

- Don't add new dependencies without explicit approval
- Don't modify database schema without migration
- Don't bypass TypeScript with `any` or `@ts-ignore`
- Don't leave `console.log` statements in production code
- Don't commit `.env` files or secrets
