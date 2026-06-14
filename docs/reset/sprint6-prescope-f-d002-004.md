# Sprint 6+ F-D002-004 IDOR — Prep Brief

**Owner of F-D002-004:** Sprint 6 lead (TBD)
**Support:** Leader (this brief) + Guardian (security re-review at end)
**Origin:** D-002 Security Review §5.3 (aafad88 on main) + F-013/F-014/F-015/F-016 Sprint 4 cross-tenant findings (mitigated-partial + waived at TASK-419..422)
**Severity:** HIGH (D-002 sign-off lists this as a real risk, not theater)
**Effort estimate:** 1.5–2 sprints (10–14 working days)

---

## Problem statement

`src/internal/handler/agent.go:299-311` (`projectIDFromContext`) trusts `X-Project-ID` from the request header without verifying that the calling user is a member of that project. A user with a valid session in project A can spoof the header to access project B's agents, capabilities, assignments, executions, and deliverables.

**Sprint 4 mitigation (TASK-419..422)**: service-layer `callerProjectID` check confirms the URL-derived project_id matches the header. **This only blocks reads** — for paths like `GET /v1/agents/:id` where the URL doesn't carry a project_id, the check is bypassed.

**D-002 confirmed (aafad88 §5.3)**: the CREATE path is still exploitable. Example: `POST /v1/agents` with a spoofed `X-Project-ID` creates an agent in another tenant's namespace. The cross-tenant waiver on F-013/14/15/16 means the issue is **known, accepted, and tracked for Sprint 6+**.

---

## True fix — three components

### Component 1: `project_memberships` table (migration 029)

```sql
-- migration 029_create_project_memberships.sql
CREATE TABLE project_memberships (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  role        TEXT NOT NULL CHECK (role IN ('owner', 'admin', 'member', 'viewer')),
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (user_id, project_id)
);
CREATE INDEX idx_project_memberships_user_id ON project_memberships (user_id);
CREATE INDEX idx_project_memberships_project_id ON project_memberships (project_id);
```

Notes:
- Role enum mirrors the existing 4-tier role model (owner > admin > member > viewer) used in `service/assignment.go` for ownership checks
- `UNIQUE (user_id, project_id)` prevents duplicate memberships
- Both indexes needed — read pattern is "list user's projects" + "list project members"
- Backfill: every existing project needs at least one owner. Migration 030 should backfill from a "default owner" set or a CLI bootstrap tool (TBD; likely a `sprint6-bootstrap` script that reads from env or JSON)

### Component 2: `requireProjectMember` middleware

New file: `src/internal/middleware/require_project_member.go`

```go
// RequireProjectMember reads the calling user (from session, JWT, or
// X-User-ID for the in-memory mock) and the project (from X-Project-ID
// or URL param), looks up the membership, and 403s if absent. The role
// is attached to the gin context for downstream handlers.
func RequireProjectMember(membershipSvc *service.MembershipService, minRole string) gin.HandlerFunc {
  return func(c *gin.Context) {
    userID, ok := userIDFromContext(c)
    if !ok {
      c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHENTICATED", "message": "Missing user identity"}})
      return
    }
    projectID, ok := projectIDFromContext(c)
    if !ok {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": "Missing X-Project-ID"}})
      return
    }
    m, err := membershipSvc.GetMembership(c.Request.Context(), userID, projectID)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": gin.H{"code": "FORBIDDEN", "message": "Not a member of this project"}})
      return
    }
    if !roleAtLeast(m.Role, minRole) {
      c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": gin.H{"code": "FORBIDDEN", "message": "Insufficient role"}})
      return
    }
    c.Set("membership", m)
    c.Next()
  }
}
```

Notes:
- `userIDFromContext` should be its own helper, parallel to `projectIDFromContext`. Currently the user identity comes from `X-API-Key` (Service) or session cookie (Web) — need to add a unified extractor
- `minRole` parameter is per-route (e.g. POST /v1/agents requires `admin`, GET /v1/agents requires `viewer`)
- The `c.Set("membership", m)` puts the membership object in the gin context for handlers that need to check the role explicitly
- `roleAtLeast` is a tiny helper: `viewer < member < admin < owner`

### Component 3: `GET /v1/projects` self-list endpoint

D-002 §5.3 recommended a self-list endpoint "so the header set is no longer blind". This is a small but important UX/security win: clients no longer need to guess or hardcode project IDs.

```go
// handler/project.go
// GET /v1/projects
// Lists the projects the calling user is a member of.
func (h *ProjectHandler) List(c *gin.Context) {
  userID, _ := userIDFromContext(c)
  projects, err := h.membershipSvc.ListProjectsForUser(c.Request.Context(), userID)
  if err != nil { respondError(c, service.NewError("INTERNAL_ERROR", http.StatusInternalServerError, "...")); return }
  c.JSON(http.StatusOK, gin.H{"data": projects})
}
```

Notes:
- Returns the project set with role annotations: `[{id, name, role}, ...]`
- No `X-Project-ID` header needed for this endpoint (it's the bootstrap)
- 200 with empty array if user has no projects (legitimate state, not a 404)
- Becomes the seed for the frontend project-switcher

---

## Cross-cutting changes

Every existing route that calls `projectIDFromContext` must add `RequireProjectMember` to its middleware chain. Current surface (from a grep — may be incomplete):

| File | Route | Method | Current min role |
|------|-------|--------|------------------|
| `handler/agent.go` | `/v1/agents` | POST | admin (create) |
| `handler/agent.go` | `/v1/agents` | GET | viewer |
| `handler/agent.go` | `/v1/agents/:id` | GET | viewer |
| `handler/agent.go` | `/v1/agents/:id` | PATCH | member |
| `handler/agent.go` | `/v1/agents/:id` | DELETE | admin |
| `handler/capability.go` | `/v1/agents/:id/capabilities` | * | viewer (read), member (write) |
| `handler/assignment.go` | `/v1/assignments` | POST | member (assign) |
| `handler/assignment.go` | `/v1/assignments/...` | * | viewer (read) |
| `handler/execution.go` | `/v1/executions/...` | * | viewer (read), member (action) |
| `handler/deliverable.go` | `/v1/deliverables/...` | * | viewer (read), member (write) |
| `handler/webhook.go` | `/v1/webhooks` | POST/DELETE | admin (D-002-005 also flags this) |
| `handler/dispatcher.go` (Sprint 4+5 deliverable) | `/v1/tasks/:id/dispatch` | POST | member |

Sprint 6 should sweep all of these and apply the middleware. The 5-step migration order:

1. **Add `MembershipService` + `MemoryMembershipStore`** (no DB yet)
2. **Add `RequireProjectMember` middleware** (uses `MemoryMembershipStore`)
3. **Apply middleware to all routes** (no behavior change if no membership rows exist — empty store = no access, which will break tests; need a test helper that seeds memberships)
4. **Backfill memberships** for any existing test fixtures
5. **Add `GET /v1/projects`** as the bootstrap endpoint

DB migration (`029_create_project_memberships.sql`) and the backfill (`030_backfill_project_memberships.sql`) come at step 4 — the in-memory store is sufficient for the first 3 steps and unblocks the middleware work without waiting on DB.

---

## Out of scope (but related)

- **F-D002-005 (admin-only webhook create)** — separate finding, separate fix; will land in B-002 alongside F-D002-001 SSRF
- **F-D002-015 (`handler/auth_test.go` absent)** — Sprint 7 non-blocking per D-002
- **F-D002-017 (`handler/project.go` absent)** — Sprint 7 non-blocking; the project.go referenced in the cross-cutting table above is the new file this work creates, not a pre-existing file
- **Audit log table** — recommended as a follow-up (track who reads/writes what project), but not required for the IDOR fix

---

## Risks

1. **Test fixtures will break en masse** when `RequireProjectMember` is applied to all routes. Every existing test that calls a project-scoped endpoint will need to seed a membership first. This is a ~1-day churn to fix the ~200 test cases across handler/* and integration/.
2. **Backfill data integrity** — for any pre-existing project with no membership, the only user with access is the project creator (assumed from the audit log or a separate `projects.created_by` column; if neither exists, every existing user loses access to every project). Recommend a "default owner" CLI tool that takes a `--user <id>` flag per project.
3. **Performance** — every request now does a `MembershipService.GetMembership` lookup. Add a 30s in-memory cache keyed on `(user_id, project_id)` to avoid DB pressure. The `c.Set("membership", m)` pattern lets handlers skip the second lookup if they need the role again.
4. **Cookie vs. header auth consistency** — the web session is a cookie, the API uses `X-API-Key`. `userIDFromContext` needs to handle both. Use a single middleware (`AuthOrSession`) that extracts user ID from either source.

---

## Open questions for the Sprint 6 lead

1. **Roles**: is 4-tier (owner/admin/member/viewer) enough, or do we need a custom role system? Spec is silent.
2. **Cross-project users**: can one user be a `viewer` in project A and an `owner` in project B? Likely yes; the schema supports it.
3. **Public projects**: does any project have public-readable content (for unauthenticated market place browsing)? If yes, the middleware needs an "anonymous read" path. Spec is silent.
4. **Service accounts**: are there non-user identities (cron jobs, webhooks) that need to act on behalf of a project? If yes, the `user_id` column needs to be polymorphic (`actor_id` + `actor_type`).
5. **Audit**: should every `RequireProjectMember` success/failure be logged? The current middleware has a `Logger` that logs requests; the membership check itself is not separately logged.

---

## Hand-back expected at end of Sprint 6

- [ ] `project_memberships` table created (migration 029) + backfilled (migration 030)
- [ ] `MembershipService` + `MemoryMembershipStore` + (Sprint 6+ if time) `PostgresMembershipStore`
- [ ] `RequireProjectMember` middleware in `middleware/`
- [ ] All project-scoped routes have the middleware applied
- [ ] `GET /v1/projects` self-list endpoint live
- [ ] `MembershipService` route added to the API spec (`docs/api-spec.md` §Projects)
- [ ] Audit doc: `docs/audit/F-D002-004-fix-sprint6.md` (mirror A-001/B-001 format)
- [ ] Cross-tenant negative tests (the F-013/14/15/16 waiver) re-run as positive-control tests: same exploit attempts now return 403
- [ ] D-002 sign-off re-issued: "F-D002-004 CLOSED" or "F-D002-004 PARTIALLY-CLOSED, residual risk = X"

---

## Why this matters

The v1 GA sign-off from D-002 is conditional on B-002 closing F-D002-001 + E-001 hardening F-D002-003. The F-D002-004 IDOR is **explicitly deferred to Sprint 6+** with a project-memberships approach. If Sprint 6 doesn't land this, the v2 sign-off will be a downgrade ("APPROVED-WITH-RISK" instead of "APPROVED"). Better to scope it now and budget 1.5–2 sprints of work than to discover at the Sprint 7 review that nothing was scheduled.
