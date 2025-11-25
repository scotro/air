# Example Walkthrough: Building a User Management Feature

This walkthrough demonstrates the concurrent AI agent workflow applied to a realistic feature: adding user management (CRUD operations, roles, permissions) to an existing application.

---

## Scenario

You have 3 hours to make significant progress on user management. The work naturally decomposes into:

1. **Database layer** - Schema, migrations, repository pattern
2. **API layer** - REST endpoints for user CRUD
3. **Permission system** - Role-based access control
4. **Tests** - Comprehensive test coverage

Dependencies:
```
[Database] ──→ [API] ──→ [Permissions]
                 ↑
              [Tests] (can start early with mocks)
```

---

## Setup Round (20 minutes)

### 1. Define Work Packets

**Packet: db-users**
```markdown
# Work Packet: db-users

**Objective:** Create database schema and repository layer for users and roles

**Branch:** `feature/db-users`

## Acceptance Criteria
- [ ] Users table with: id, email, name, password_hash, created_at, updated_at
- [ ] Roles table with: id, name, permissions (jsonb)
- [ ] User_roles junction table
- [ ] Migration files created and tested
- [ ] Repository with: createUser, getUserById, getUserByEmail, updateUser, deleteUser
- [ ] All repository methods have unit tests

## Boundaries
**In Scope:** src/db/schema/, src/db/migrations/, src/lib/repositories/user.ts
**Out of Scope:** API routes, authentication logic, UI components

## Interface Contract
Export these types for other agents:
- User, Role, UserWithRoles types
- UserRepository class with documented methods
```

**Packet: api-users**
```markdown
# Work Packet: api-users

**Objective:** Create REST API endpoints for user management

**Branch:** `feature/api-users`

## Acceptance Criteria
- [ ] GET /api/users - List users (paginated)
- [ ] GET /api/users/:id - Get single user
- [ ] POST /api/users - Create user
- [ ] PATCH /api/users/:id - Update user
- [ ] DELETE /api/users/:id - Delete user
- [ ] Proper error responses (400, 401, 404, 500)
- [ ] Input validation on all endpoints

## Boundaries
**In Scope:** src/app/api/users/
**Out of Scope:** Database schema, permission checks (will be added later), UI

## Dependencies
**Soft dependency on db-users:** Start with mock repository, integrate real one in integration round.

## Interface Contract
Assume UserRepository exists with standard CRUD methods.
Use these response shapes:
- Success: { data: User | User[] }
- Error: { error: string, code: string }
```

**Packet: tests-users**
```markdown
# Work Packet: tests-users

**Objective:** Comprehensive test coverage for user management feature

**Branch:** `feature/tests-users`

## Acceptance Criteria
- [ ] Unit tests for UserRepository (mock DB)
- [ ] Integration tests for API endpoints (test DB)
- [ ] Edge cases: duplicate email, invalid input, not found
- [ ] Test utilities: factories, fixtures, helpers

## Boundaries
**In Scope:** src/**/*.test.ts, src/test/
**Out of Scope:** Implementation code (only tests)

## Dependencies
**Soft dependency:** Can write test shells and mocks while implementation proceeds.
```

### 2. Create Worktrees

```bash
source agent-helpers.sh

# Create worktrees
agent-create db-users
agent-create api-users  
agent-create tests-users

# Verify
agent-list
```

Output:
```
Active Agent Worktrees:
========================
agent-db-users       feature/db-users      ⚪ Idle
agent-api-users      feature/api-users     ⚪ Idle
agent-tests-users    feature/tests-users   ⚪ Idle
```

### 3. Initialize Session

```bash
session-init user-management
```

### 4. Dispatch Agents

Open three terminals (or use tmux):

```bash
# Terminal 1
cd worktrees/agent-db-users
claude
# Paste work packet, then: "Read the work packet and begin implementation"

# Terminal 2
cd worktrees/agent-api-users
claude
# "Read the work packet. Start with mock repository since db-users is in progress."

# Terminal 3
cd worktrees/agent-tests-users
claude
# "Read the work packet. Create test utilities and mock factories first."
```

---

## Execution Round 1 (25 minutes in)

### Check Each Agent

**Agent: db-users**
```
You: "Status update - what's your progress?"

Agent: **STATUS: RUNNING**
Progress:
- Created users table schema with all fields
- Created roles table with permissions jsonb
- Working on user_roles junction table
- Migration file drafted, not yet tested

~40% complete, on track for acceptance criteria.
```
Action: No intervention needed.

**Agent: api-users**
```
You: "Status update"

Agent: **STATUS: RUNNING**
Progress:
- Created mock UserRepository for development
- GET /api/users (list) implemented with pagination
- GET /api/users/:id implemented
- Starting POST /api/users

~35% complete. Question: should user creation return the created user or just the ID?
```
Action: Quick decision - "Return the full user object. Continue."

**Agent: tests-users**
```
You: "Status update"

Agent: **STATUS: RUNNING**
Progress:
- Created test utilities: createTestUser factory, setupTestDb helper
- Created mock repository with in-memory store
- Writing unit tests for repository interface

~25% complete.
```
Action: No intervention needed.

### Update Dashboard

```markdown
| ID | Worktree | Packet | Status | Last Check | Progress |
|----|----------|--------|--------|------------|----------|
| 1 | agent-db-users | db-users | 🟢 Running | 10:25 | ~40% |
| 2 | agent-api-users | api-users | 🟢 Running | 10:25 | ~35% |
| 3 | agent-tests-users | tests-users | 🟢 Running | 10:25 | ~25% |
```

---

## Execution Round 2 (50 minutes in)

**Agent: db-users**
```
Agent: **STATUS: DONE**
Completed:
- Users table: id, email, name, password_hash, created_at, updated_at
- Roles table: id, name, permissions (jsonb), created_at
- User_roles junction with foreign keys
- Migration: 001_create_users_and_roles.sql (tested locally)
- UserRepository: all CRUD methods + getUsersByRole
- Unit tests: 12 tests, all passing

Files changed:
- src/db/schema/users.ts (new)
- src/db/schema/roles.ts (new)
- src/db/migrations/001_create_users_and_roles.sql (new)
- src/lib/repositories/user.ts (new)
- src/lib/repositories/user.test.ts (new)
- src/types/user.ts (new)

Ready for: Integration review
```
Action: Mark for integration. This unblocks api-users.

**Agent: api-users**
```
Agent: **STATUS: BLOCKED**
Reason: Need to switch from mock to real repository

Completed so far:
- All 5 endpoints implemented with mock
- Input validation using zod
- Error handling standardized

Blocking: Integration with real UserRepository
Can continue: Could add more validation edge cases
```
Action: "db-users is complete and merged to main. Pull main into your branch and switch to real repository."

**Agent: tests-users**
```
Agent: **STATUS: RUNNING**
Progress:
- Repository unit tests complete (using mocks)
- Starting API integration test setup
- Created test database configuration

~50% complete
```
Action: No intervention needed.

---

## Execution Round 3 (1 hour 20 minutes in)

**Agent: db-users** - Already done, worktree can be cleaned up.

**Agent: api-users**
```
Agent: **STATUS: DONE**
Completed:
- Integrated real UserRepository
- All endpoints tested manually
- Added additional validation (email format, required fields)
- Proper 404 for missing users

Files changed:
- src/app/api/users/route.ts (new)
- src/app/api/users/[id]/route.ts (new)
- src/lib/validation/user.ts (new)

Ready for: Integration review and test coverage
```

**Agent: tests-users**
```
Agent: **STATUS: RUNNING**
Progress:
- API integration tests written for all endpoints
- Running into issue: test database not resetting between tests

~70% complete
```
Action: "Use transaction rollback pattern - wrap each test in a transaction and rollback after. Check src/test/helpers.ts for existing patterns."

---

## Integration Round (2 hours in)

### 1. Review db-users

```bash
cd worktrees/agent-db-users
git diff main...HEAD --stat
npm test
npm run lint
```

Everything passes. Merge:
```bash
git checkout main
git merge feature/db-users
```

### 2. Review api-users

```bash
cd worktrees/agent-api-users
git rebase main  # Pick up db-users changes
npm test
npm run lint
```

Found issue: one test failing due to missing await. Fix is trivial - make it directly or note for agent.

```bash
git checkout main  
git merge feature/api-users
```

### 3. Check tests-users Progress

Agent resolved the transaction issue and is finishing up. Will complete in next round.

### 4. Clean Up Completed Worktrees

```bash
agent-remove db-users
agent-remove api-users
```

### 5. Plan Next Cycle

Remaining work:
- tests-users: ~30 minutes to complete
- permissions: Not started, now has all dependencies ready

Decision: Start permissions agent while tests-users finishes.

```bash
packet-create permissions
agent-create permissions
cd worktrees/agent-permissions && claude
```

---

## Session Outcomes (3 hours)

**Completed:**
- Database schema and migrations for users/roles
- Full REST API for user CRUD
- 80% test coverage (finishing async)
- Permissions work started

**Metrics:**
- 3 agents run concurrently
- 4 execution rounds
- 1 integration round
- ~2 meaningful human interventions (decisions/unblocking)

**Learnings:**
- db-users finished faster than expected - could have started permissions earlier
- Soft dependency pattern worked well for api-users
- Test agent benefited from late start (more stable interfaces to test against)

---

## Key Takeaways from This Example

1. **Front-load decomposition** - The 20-minute setup round enabled 2+ hours of parallel work

2. **Soft dependencies unlock parallelism** - api-users started with mocks rather than waiting

3. **Quick decisions keep agents moving** - "Return full user object" took 10 seconds but unblocked progress

4. **Integration rounds are where quality happens** - Found the missing await during merge review

5. **Stagger by dependency** - Started tests-users slightly later so it had stable interfaces

6. **Don't over-parallelize** - 3 agents was manageable; 5+ would have degraded supervision quality
