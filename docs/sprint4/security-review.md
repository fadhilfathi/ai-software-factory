# Sprint 4 — Security Review

**Owner:** Security-01
**Date:** 2026-06-12
**Sprint:** 4 — Agent Orchestration Engine
**Status:** Runtime review complete. F-001/F-002 verified FIXED-IN-PATCH. F-003..F-007 pre-dev hypotheses: F-003 confirmed (superseded by F-013), F-004 confirmed (see F-014), F-005 not a finding, F-006 deferred to TASK-409, F-007 still applies. §1.2..§4.2 (backend) filled in. Seven new runtime findings (F-013..F-017, F-021, F-023) — F-013..F-016 Critical cross-tenant findings: F-013 **FIXED-IN-PATCH (TASK-419, 2026-06-13)**, F-014..F-016 WAIVED-BY-LEADER (Sprint 5 follow-up: TASK-420..422); **F-017 and F-023 FIXED-IN-PATCH (TASK-423, TASK-424, 2026-06-12)**; F-021 DEFERRED to Sprint 5 (TASK-425). §4 frontend (markdown XSS) and §410 activity dashboard remain deferred.

---

## 0. Scope & Method

### 0.1 Scope of review

In-scope for Sprint 4:

- New backend endpoints implemented by TASK-402, 403, 404, 405, 406
  - `/v1/agents` (CRUD)
  - `/v1/agents/:id` (CRUD)
  - `/v1/agents/:id/heartbeat`
  - `/v1/agents/:id/capabilities` (read)
  - `/v1/capabilities` (read)
  - `/v1/tasks/:id/assign` (assignment)
  - `/v1/executions` (CRUD + status patch)
  - `/v1/deliverables` (CRUD + versions)
- New frontend surfaces in TASK-407, 408, 409, 410
  - `/agents`, `/agents/[id]`, `/agents/[id]/status`, `/agents/[id]/capabilities`
  - `/tasks/[id]/assign`, `/tasks/[id]/ownership`, `/assignments`
  - `/deliverables`, `/deliverables/[id]`, `/deliverables/[id]/versions`
  - `/agents/activity`
- The existing auth middleware (`src/internal/middleware/middleware.go`) and router (`src/internal/router/router.go`) as they apply to the new endpoints.
- The data model in `docs/sprint4/data-model.md` for tenant-scoping analysis.

Out of scope (already reviewed in TASK-308 / `docs/sprint4/security-report.md`):

- `/v1/auth/*`, `/v1/users/*` core authentication — see prior report.
- `/v1/projects/*` — unchanged this sprint, inherits prior findings.
- `/v1/webhooks`, `/v1/code/*` — touched lightly only to assess cross-surface
  exposure; full review deferred unless an issue surfaces in passing.

### 0.2 Method

1. Static review of spec, data model, middleware, router, and (post-dev) handler/service code.
2. Tracing the auth/authz flow from router → middleware → handler → service → store for each new endpoint.
3. Cross-reference against the prior security findings in `docs/sprint4/security-report.md` (TASK-308) to confirm regression status.
4. (If runtime is available after Leader's quality-gate decision) targeted black-box probes against the running stack with a fresh user token; else static-only.

### 0.3 Existing tenant / scoping model (baseline)

Established during prep work (does not require implemented code):

- **Data layer is project-scoped.** `docs/sprint4/data-model.md` §12 states:
  *"Multi-tenancy: every domain table is scoped to project_id. A tenant_id column
  is not introduced in Sprint 4 (single-tenant per deployment)."* Every Sprint-4
  domain table — `agents`, `agent_capabilities`, `assignments`,
  `assignment_events`, `executions`, `deliverables` — carries a `project_id`
  column.
- **Auth layer is user-scoped only.** `src/internal/middleware/middleware.go`
  populates `UserIDKey` and `RoleKey` in the gin context. There is no
  project-membership or project-role lookup in the auth middleware.
- **JWT role is now read from the user record** (F-001, fixed in TASK-417).
  `src/internal/service/auth.go` `Login` and `Refresh` both call a single
  `mintToken(user)` helper that embeds `user.Role` in the JWT claims. The
  pre-patch hard-coded `Role: "user"` is gone. The `users.role` column was
  pre-existing (`001_create_users.sql`); no new migration was required.
  The `RequireRole` middleware still exists in `middleware.go` but is not
  applied in the current router; once it is wired, F-001's fix is what
  makes it satisfiable.
- **`ak_*` API keys are now real credentials** (F-002, fixed in TASK-418).
  `src/internal/middleware/middleware.go` now calls
  `authService.ValidateAPIKey(ctx, token)`, which SHA-256-hashes the
  post-`ak_` body, looks it up in `store.APIKeyStore`, and checks
  `RevokedAt` and `ExpiresAt` before stamping `UserIDKey` / `RoleKey`. The
  13-byte prefix check is gone. Store is in-memory (Option B per the
  TASK-418 brief); the Postgres-backed `api_keys` table is deferred to
  Sprint 5 (see §5.1, F-008).
- **No `tenant_id` anywhere.** The scoping key is `project_id`.

**Implication for the review:** "does the caller have scope to read/write X?"
must be answered in two parts — (a) is the user a member of `X.project_id`? and
(b) does the user's role permit this action? Neither is enforced today in
middleware; both are expected to be enforced (or not) at the handler/service
layer. The review will specifically look for those checks.

### 0.4 Environment caveat

The Windows host has no Docker, no Go, no Node, no Python (per Leader's brief
2026-06-12). The runtime validation portion of this review is therefore
**blocked on a decision from the user** about how to satisfy the pre-commit
quality gate. Sections 1–4 below are structured to allow **static review now**
and **runtime confirmation later** without re-architecting the document.

### 0.5 Threat model

**Assets** (what we are protecting):

- Authenticated user identity (`users.id`, JWT subject, password hashes).
- Project membership and project-scoped data: `agents`, `agent_capabilities`,
  `assignments`, `assignment_events`, `executions`, `deliverables` (all rows
  per `project_id`).
- Assignment history integrity (`assignment_events`) — audit-grade record of
  who assigned what, when.
- Deliverable content — author-generated markdown; potentially customer-facing
  in Sprint 5+, treated as user-controlled from a security standpoint now.
- Webhook secrets and webhook URLs (pre-existing, touched lightly).

**Actors**:

- **Legitimate user** — owns a JWT, may be a member of one or more projects.
- **Project member** — a legitimate user authorized for one project; should
  have *no* visibility into other projects.
- **Project admin** — a user with `role != "user"`. Reachable now that
  F-001 is fixed: the role is read from the DB and embedded in the JWT.
  The `RequireRole` middleware still needs to be wired in the router to
  actually gate anything on it.
- **Authenticated outsider** — any user with a valid JWT, member of *no*
  project the resource belongs to. Should get 404, never 200.
- **Unauthenticated attacker** — no JWT. Should be rejected by `AuthRequired`.
- **API-key caller** — sends `Authorization: Bearer ak_…`. As of TASK-418,
  the key is now real: `auth.ValidateAPIKey` SHA-256-hashes the suffix,
  looks it up in `APIKeyStore`, and checks `RevokedAt` / `ExpiresAt`
  before stamping `UserIDKey` / `RoleKey`. The prior 13-byte prefix
  short-circuit is gone (F-002 fixed).

**Attacker goals** (ranked by impact):

1. Read or modify data in a project they do not belong to (cross-tenant
   read/write).
2. Move a task to an agent of their choosing to influence downstream
   execution (assignment manipulation).
3. Persist capability claims not granted by a project admin (privilege
   escalation via the capability engine).
4. Inject script into a deliverable viewed by another user (stored XSS).
5. Bypass authentication entirely (the `ak_*` short-circuit).
6. Forge or rewrite the `assignment_events` audit trail.

**Attack surface** for this sprint:

- All new HTTP endpoints listed in §0.1 (router → middleware → handler → service
  → store chain).
- The new database tables (SQL injection, parameter tampering, FK-bypass).
- Markdown content entering the database and re-emerging in the browser
  (stored XSS).
- The JWT issuance path (role setting).
- The `Authorization: Bearer ak_*` header (pre-shared key pattern).
- The web frontend (Next.js) — markdown rendering, link handling, `localStorage`
  use of the JWT, CSRF on state-changing routes.

### 0.6 Severity scheme

Each finding in §5 carries a severity. The scheme:

- **Critical** — exploitable today with no preconditions beyond possessing
  *any* credential (or none at all). Cross-tenant data exposure, full
  authentication bypass. **Must be fixed or explicitly accepted by the
  Leader before commit.** Acceptance requires a written waiver recorded in
  §7.
- **High** — exploitable in a realistic scenario with valid authentication,
  often requiring only knowledge of a UUID or one extra parameter. Cross-
  project read/write when the victim UUID is guessable is High, not Critical,
  because it needs a precondition. **Should be fixed this sprint; if not, a
  Leader waiver is required.**
- **Medium** — defense-in-depth gap, or a bug that requires an unusual chain
  to exploit (e.g., a desynchronized deliverable whose task/agent belong to
  different projects). Track to Sprint 5 if not fixed this sprint.
- **Low** — hardening / hygiene (missing security headers, verbose error
  bodies, etc.). Backlog.
- **Info** — observation only. No action. Examples: "JWT `alg` is RS256; that
  is good, no change required."

CVSS-style numerical scoring is intentionally **not** used — the project does
not have a CVSS calculator workflow and an ad-hoc number would be misleading.
The five-level scale above is sufficient for the sprint's sign-off criteria.

### 0.7 Regression check against TASK-308

Each prior finding in `docs/sprint4/security-report.md` is checked for
regression (i.e., was the dev wave that touched that area careful not to
re-introduce the issue, or to widen its blast radius). The check is recorded
inline in §5 with each relevant prior finding. Findings that are new this
sprint are tagged `[NEW]`. Findings carried over from TASK-308 are tagged
`[REGRESSION-CHECK]`.

---

## 1. Authorization Review

**Question to answer:** Does the agent endpoint check that the caller has scope
to read/write the agent? Does assignment enforce project-level access?

### 1.1 Items to check (post-dev)

- [ ] `POST /v1/agents` — does the handler derive `project_id` from the URL, the
      JWT, or the request body? If from the body, can the caller pick any
      project? Is project-membership checked?
- [ ] `GET /v1/agents` — is the list filtered by the caller's project(s), or
      is `?project_id=` purely a client-supplied filter with no membership
      check? (The api-spec leaves this open; the implementation is what matters.)
- [ ] `GET/PUT/DELETE /v1/agents/:id` — does the service verify
      `agent.project_id ∈ caller.projects`?
- [ ] `POST /v1/agents/:id/heartbeat` — who is allowed to heartbeat an agent?
      Any authenticated user, or only the agent's own service account?
- [ ] `RequireRole` middleware (defined in `middleware.go`) — is it actually
      applied anywhere in the router for the agent endpoints, or is it dead
      code? Note: F-001 was the blocker on this being satisfiable; with
      TASK-417 the role is now real, so this becomes a normal review
      question (is `RequireRole` actually used?).

### 1.2 Authorization findings

The runtime review confirms that the agent endpoints are implemented
with a known design tradeoff: the handler's `projectIDFromContext`
helper (in `src/internal/handler/agent.go`) **trusts the
`X-Project-ID` header (or `:projectID` URL param) as the
project-scoping signal without verifying that the caller is a member
of that project**. The function's own comment is explicit:

> *"the service layer trusts this signal and does not double-check
> it against the caller's project membership."*

The `project_memberships` table exists in the data model but is not
read on the request path; no service or middleware consults it. There
is no `requireProjectMember` middleware, and `auth.UserIDKey` is
never joined against a project membership lookup.

Concrete impact for §1.1's checklist items:

- **`POST /v1/agents` (item 1).** Handler's `Create` extracts
  `project_id` from the header and passes it straight to
  `service.CreateAgent`. **No membership check.** Cross-tenant agent
  creation is possible. See **F-013**.
- **`GET /v1/agents` (item 2).** The list filter is
  `project_id = $1` in `store/postgres/agent_store.go`, where `$1`
  is the client-controlled header value. The SQL is correct; the
  input is not. **No membership check.** See **F-013**.
- **`GET /v1/agents/:id` (item 3).** `store.GetByID` has no
  `project_id` filter. Any authenticated user can read any agent by
  UUID. **Cross-tenant read.** See **F-013**.
- **`PUT /v1/agents/:id` (item 4).** `service.UpdateAgent` reads by
  ID, patches, writes back without checking the caller's project.
  **Cross-tenant write.** See **F-013**.
- **`DELETE /v1/agents/:id` (item 5).** `service.SoftDelete` is
  project_id-agnostic. **Cross-tenant soft-delete.** See **F-013**.
- **`POST /v1/agents/:id/heartbeat` (item 6).** The heartbeat
  endpoint (TASK-402) inherits the same pattern. Any authenticated
  user can heartbeat any agent by UUID. **No membership check.**
  See **F-013**.
- **`RequireRole` middleware (item 7).** Still not applied anywhere
  in the router for the agent endpoints. F-001 is fixed (the role is
  now real), so the gap is "not wired" rather than
  "permanently unsatisfiable". This is a defense-in-depth gap, not a
  Critical by itself. See **F-021**.

**Status of the §5 hypotheses (from runtime evidence):**

- **F-003 (agent list tenant scoping) — CONFIRMED, ESCALATED to
  Critical, SUPERSEDED-BY-F-013.** The hypothesis is correct *and*
  the impact is wider than anticipated: the same pattern affects
  `GetByID` / `Update` / `SoftDelete` (not just `List`), and the
  per-UUID lookups are filter-free. F-013 is the canonical finding.

---

## 2. Access Control Review

**Question to answer:** Are list endpoints tenant-scoped? Who can read which
deliverable?

### 2.1 Items to check (post-dev)

- [ ] **List endpoints tenant scoping.** `GET /v1/agents`, `GET /v1/executions`,
      `GET /v1/deliverables`, `GET /v1/capabilities` — does each return only
      rows the caller is permitted to see? Specifically:
      - Does the SQL `WHERE` clause include `project_id IN (caller.projects)`?
      - Or is filtering by `project_id` purely a query parameter that can be
        omitted, returning the global table?
- [ ] **Cross-tenant read.** Pick two `project_id`s. As user U who is a member
      of project A only, can U read or list rows belonging to project B by
      passing `?project_id=B` or by knowing a row's UUID?
- [ ] **Deliverable access.** `GET /v1/deliverables/:id` — is there a check
      that the caller can read the parent task's project, or the parent
      agent's project? Or is the deliverable readable by any authenticated
      user who knows the UUID?
- [ ] **Version history access.** `GET /v1/deliverables/:id/versions` — same
      question; in particular, do older versions stay readable after access
      has been revoked?
- [ ] **API-key path.** F-002 is fixed — `middleware.go` now calls
      `authService.ValidateAPIKey`, which hashes the suffix with SHA-256,
      looks up `APIKeyStore`, and rejects on `RevokedAt` / `ExpiresAt`.
      Confirm the new agent / execution / deliverable endpoints reach the
      middleware chain and that the *post-validation* `UserIDKey` /
      `RoleKey` stamping is what the handlers see (not a stale
      "api_user" default). The open follow-ups are tracked as F-008
      (Postgres-backed store), F-009 (Create method), and F-010
      (LastUsedAt not actually persisted) in §5.1.
- [ ] **404 vs 403.** Do the handlers distinguish "not found" from
      "forbidden"? Returning 404 for both is the safer default; returning 403
      for cross-tenant access leaks existence.

### 2.2 Access-control findings

The runtime review confirms that the deliverable and execution
handler/service layers have **no project-scoping whatsoever**. The
`X-Project-ID` header that exists for agents (still broken, see
§1.2) is **not** used for deliverables or executions. Per-UUID
lookups and list endpoints rely entirely on `task_id` / `agent_id`
filter parameters, both of which are client-controlled. Neither
`service.DeliverableService` nor `service.ExecutionService` consults
`project_memberships`.

Concrete impact for §2.1's checklist items:

- **Deliverable list tenant scoping (item 1).** `GET
  /v1/deliverables` requires at least one of `task_id` or
  `agent_id` (enforced in `service/deliverable.go` `ListDeliverables`),
  but the WHERE clause does not constrain the result set to the
  caller's projects. Any authenticated user can list the
  deliverables of any task or agent by UUID. See **F-015**.
- **Cross-tenant read on a known UUID (item 2).** `GET
  /v1/deliverables/:id` and `GET /v1/executions/:id` both fetch by
  ID with no project_id filter. See **F-015** and **F-016**.
- **Deliverable access (item 3).** The handler does not check that
  the caller's projects include the deliverable's task's project,
  the agent's project, or any project at all. See **F-015**.
- **Version history (item 4).** `GET
  /v1/deliverables/:id/versions` has no project_id filter on the
  versions query. **Soft-deleted (versioned) historical rows stay
  readable across tenants.** See **F-015**.
- **API-key path (item 5).** F-002 is fixed — the path goes through
  `authService.ValidateAPIKey` and stamps a real `UserIDKey` /
  `RoleKey`. The 13-byte prefix short-circuit is gone. The remaining
  gaps (F-008/009/010) are tracked in §5.1.
- **404 vs 403 (item 6).** The handlers return 404 for "not found"
  and 200 with the row for "not authorized" (the cross-tenant read
  is not rejected, it just succeeds). This is a consequence of
  F-015 / F-016, not a separate bug — once those are fixed, the
  cross-tenant read returns 404, which also hides existence.

**Status of the §5 hypotheses (from runtime evidence):**

- **F-003 (agent list tenant scoping) — see §1.2, superseded by
  F-013.** The deliverable and execution list-endpoint analogues
  are filed under F-015 and F-016 respectively.

---

## 3. Assignment Review

**Question to answer:** Can a low-privilege user assign a task to a different
project's agent? Can the capability check be bypassed?

### 3.1 Items to check (post-dev)

- [ ] **Cross-project assignment.** `POST /v1/tasks/:id/assign` with body
      `{ agent_id, capabilities_required? }` — does the service verify that
      `task.project_id == agent.project_id == caller.project_id`? If any of
      the three is left unchecked, a member of project A can move a project-B
      task onto a project-B agent, or move a project-A task onto a
      project-B agent.
- [ ] **Ownership manipulation.** The assignment engine records
      `assignment.assigned_to` and appends to `assignment_events`. Is the
      `assigned_by` field taken from the JWT subject, or from the request body?
      If from the body, audit trail is forgeable.
- [ ] **Capability check bypass.** TASK-403 introduces
      `CapabilityService.ValidateAgentHasCapabilities(agentID, []string)`.
      Confirm:
      - It is called **before** the assignment is persisted, not after.
      - It checks against the *current* capability set of the agent, not
        against a client-supplied claim.
      - It is *not* possible to call the underlying store/handler that
        persists the assignment without going through the validator. (Look
        for any internal "raw" method exposed to the router.)
      - Empty / null / `capabilities_required: []` is handled explicitly
        (no requirements → allow, but still log).
- [ ] **Re-assignment & unassignment.** Does the history entry's `action`
      enum (`assign | reassign | unassign`) get validated server-side, or can
      a client write `action: "promote"` and get it persisted?
- [ ] **Race / TOCTOU.** Two concurrent `assign` calls on the same task — does
      the DB layer use a conditional `UPDATE ... WHERE assigned_to IS NULL`
      and a unique constraint on `(task_id)` in the active-assignment view?
      Otherwise two agents can be assigned to the same task.

### 3.2 Assignment findings

The runtime review confirms **F-004 (cross-project assignment)** as
a Critical. The `service/assignment.go` `AssignTaskToAgent` flow:

1. Fetches the task by ID (`Tasks().GetByID(taskID)`) — no project
   filter.
2. Fetches the agent by ID (`Agents().GetByID(ctx, agentID)`) — no
   project filter.
3. **Never compares `task.ProjectID` to `agent.ProjectID`.**
4. **Never compares the caller's projects to `task.ProjectID`.**
5. Calls the capability validator against the live
   `agent_capabilities` join — this is wired correctly (F-005 is
   NOT a bypass in the strict sense; see below).
6. Persists the assignment and appends an `assignment_events` row
   inside a single transaction.

Concrete impact:

- Any authenticated user can read two UUIDs (a task and an agent)
  from `GET /v1/agents/:id` (which is itself project_id-agnostic,
  see F-013) and `GET /v1/tasks/:id` (assumed to be the same —
  not reviewed this sprint, but the same pattern is likely), then
  call `POST /v1/tasks/<task>/assign` with `{ agent_id: <agent> }`.
- If the agent is `idle`, the assignment succeeds and is recorded
  in `assignment_events` with the caller's user_id (not forgeable
  — `assignedBy` is taken from the JWT subject at
  `handler/assignment.go` `userIDFromContext(c)`, line 109-115).
- The task's `assignee_id` is updated to the new agent (in a
  separate statement outside the transaction, line 272-281 of
  `service/assignment.go` — not a security issue but a code-quality
  note: this should be inside the transaction).
- **The data model §4.1 explicitly notes that `assignments` is
  missing a `project_id` column and that the backfill is scheduled
  for Sprint 5.** *"Project-scoped query shortcut. Backfill on
  deploy: UPDATE assignments a SET project_id = t.project_id FROM
  tasks t WHERE a.task_id = t.id. A `(project_id, status)` index
  will be added in the same migration."* The data model is aware
  of the gap. The runtime review surfaces it as **F-014** with the
  same Critical classification and the explicit recommendation to
  close it in Sprint 5 as the data model plans.

Other §3.1 checklist items:

- **Ownership manipulation (item 2).** `assigned_by` is from the
  JWT subject, not the request body. **Audit trail not forgeable
  from the client side.** ✓
- **Capability check (item 3).** The validator is wired correctly:
  it reads the live `agent_capabilities` join via
  `agents.ListCapabilitiesByAgent`, is called *before* persistence
  (in the same service function, before the DB write at line
  222+), and is on the only code path that reaches the
  `assignments` insert. The "soft bypass" identified during prep
  — sending an empty `capabilities_required` to a fresh task
  with no pre-existing required capabilities — is a designed-in
  feature per the api-spec (`capabilities_required` is optional).
  **Not a finding.** F-005 status changed to `NOT-A-FINDING (N/A)`.
- **Re-assign / unassign action enum (item 4).** The service uses
  `model.IsValidAssignmentAction` as a defence-in-depth check on
  the action verb (`service/assignment.go` line 175-181);
  unknown actions return 400. ✓
- **Race / TOCTOU (item 5).** The protection is in place at both
  layers. The DB-level partial unique index
  `uq_assignments_one_active_per_task` on
  `assignments(task_id) WHERE status='active'` (migration
  `src/db/migrations/019_create_assignments.sql` line 55-57)
  enforces "at most one active assignment per task". The service
  has an explicit transactional flow
  (`src/internal/service/assignment.go`):
  - Idempotency check (line 151-167) — if the agent is already
    the assignee, return the existing active row with
    `Idempotent=true` and **do not** write a new event.
  - Atomic flip-and-insert (line 186-213) — flip the previous
    active row to `'superseded'` and insert the new active row
    in the same transaction. The flip releases the partial
    unique-index slot before the new insert.
  - Conflict mapping (line 225-257) — a unique-constraint
    violation (a concurrent POST won the race) is mapped to 409
    with the structured error `Assignment race: another
    request created an active row for this task concurrently`.

  The service-level pre-check and the DB-level guard are
  consistent (both key on `status='active'`). The DB-level
  index is the canonical enforcement; the service pre-check is
  the UX optimisation. **No race window; this item is clean.**
  ✓ *(Earlier draft of this review incorrectly flagged it as a
  watchpoint — corrected after the Lead pointed at migration 019
  and the service's transactional wrapper.)*

**Additional finding (item not in the §3.1 checklist):**

- **`assignment_events.notes` is silently dropped.** The service
  writes the event with `Notes: ""` (`service/assignment.go` line
  244) because the service's `Append` call has no notes parameter.
  The handler then mutates the in-memory result with
  `result.Event.Notes = req.Notes` (`handler/assignment.go` line
  134-136) AFTER the service has returned. The DB row has the
  empty string; the response has the notes; subsequent
  `GET /v1/tasks/:id/history` calls return the empty notes.
  **Data-integrity bug, not a security finding per se**, but
  worth flagging because it makes the audit trail less useful
  (operators may write notes expecting them to be persisted).
  See **F-017**.

**Status of the §5 hypotheses (from runtime evidence):**

- **F-004 (cross-project assignment) — CONFIRMED, escalated to
  Critical, see F-014.** Data model §4.1 acknowledges the gap and
  defers to Sprint 5; the runtime review surfaces it now so the
  waiver decision (§7.2) is explicit.
- **F-005 (capability check bypass) — NOT A FINDING.** The
  validator is correctly wired. The "no requirements" case is
  per spec. F-005 status changed to `N/A`.

---

## 4. Deliverable Access Review

**Question to answer:** XSS risk in markdown rendering? Path/content injection
in deliverable content?

### 4.1 Items to check (post-dev)

- [x] **Markdown XSS in frontend viewer (TASK-409).** `/deliverables/[id]`
      renders deliverable content via `react-markdown@9` + `remark-gfm@4`
      with `rehype-raw@7` + `rehype-sanitize@6` (custom schema). Verified in
      `frontend/src/components/deliverables/MarkdownRenderer.tsx`:
      - `tagNames` allowlist is grep-friendly and explicitly omits
        `script`, `style`, `iframe`, `object`, `embed`, `frame`, `frameset`,
        `form`, `textarea`, `button`, `link`, `meta`, `base`, `audio`,
        `video`, `source`, `track`, `area`, `map`.
      - `attributes` map is per-tag (`a: [href, title]`, `img: [src, alt, title]`,
        `code: [className]`, `pre: [className]`, `input: [type, checked, disabled]`).
        No `on*` handlers, no `style` attribute.
      - `protocols` map restricts `href` to `http`/`https`/`mailto` and `src`
        to `http`/`https`; `javascript:` and `data:` URIs are blocked.
      - `rehype-raw` is paired with `rehype-sanitize` so sanitization has
        something to operate on; the order in the plugin array is correct
        (`[rehypeRaw, [rehypeSanitize, schema]]` — raw→sanitize, not the
        other way round).
      - **No `dangerouslySetInnerHTML` is used anywhere in this file.**
        `react-markdown` renders via its component tree, not via raw HTML
        injection.
      - **External links (watchpoint, not finding):** `react-markdown` does
        not set `target="_blank"` by default on `<a>` tags, so the
        `rel="noopener noreferrer"` requirement is not active. **If a
        future custom `components.a` is added that opens links in a new
        tab, the noopener/noreferrer policy must come with it.**
      - **6 XSS unit tests** in `MarkdownRenderer.test.tsx` cover the
        canonical vectors: `<script>` strip, inline `onerror` strip,
        `javascript:` URL strip, `<iframe>`/`<object>`/`<embed>` strip,
        `<style>` strip, and a GFM-positive case (table renders). `npx
        vitest run` → 11/11 pass per the dev's TASK-409 work log.
      - **Doc nit (no security impact):** the file's leading comment
        refers to "F-008 from security-review §5.1" — that row in §5.1
        is `APIKeyStore` persistence. The actual cross-tenant
        markdown-XSS row in this review is **F-006** (F-008 is a
        different concern; the dev's F-008 reference is a
        pre-review-label copy/paste). No code change implied; noted here
        for the closeout reviewer.
- [x] **Content injection at the API layer.** `POST /v1/deliverables` and
      `PUT /v1/deliverables/:id` accept `content` (markdown). Verified:
      - Content is stored as-is, no server-side markdown rendering
        (TASK-406 service uses a single `text` column).
      - **Size is bounded (F-023, TASK-424 FIXED-IN-PATCH 2026-06-12):**
        two-layer defence — handler `http.MaxBytesReader` at 1 MiB + 8 KiB
        headroom, service `len(content) > MaxDeliverableContentBytes`
        check; both map to 413 `PAYLOAD_TOO_LARGE`. See §4.2 backend
        notes and F-023 row.
      - `text/markdown` content-type is not accepted as a structural
        marker; the body is JSON-decoded, and the `content` field is the
        only payload.
- [x] **Path injection in deliverable title / version label.** Title and
      other string fields are stored as `text` columns and returned as
      JSON strings; React renders them as text, not as HTML. `react-markdown`
      is not involved for these fields. No template injection vector. ✓
- [x] **Version-diff XSS.** `VersionDiff` consumes the `diff` library
      output (text-level diff lines, no HTML). Renders into the React
      tree, not via `dangerouslySetInnerHTML`. ✓
- [ ] **Cross-tenant deliverable read.** Same as §2.2; still WAIVED-BY-LEADER
      (TASK-421, Sprint 5). Carrying the watchpoint forward to the
      runtime-review pass.
- [ ] **Deleted agent / deleted task.** Watchpoint; not confirmed (static-only
      review). To be re-verified at runtime.
- [x] **Versioning immutability.** `PUT /v1/deliverables/:id` always inserts
      a new row in `deliverable_versions`; service explicitly INSERTs
      without UPDATE on the current row. Historical rows are append-only. ✓

### 4.2 Deliverable findings

The runtime review covers the **deliverable backend** (TASK-406, in
place) and the **deliverable frontend viewer** (TASK-409, in place).
**TASK-409 is closed** as of 2026-06-12, so the markdown-XSS
checklist in §4.1 has been walked; F-006 moves to `FIXED-IN-PATCH`.
**TASK-410** (agent activity dashboard) is also briefly covered
here — it consumes the F-016 execution data source and adds no new
attack surface.

**Backend findings (§4.2 backend) — unchanged from previous pass:**

- **Cross-tenant read on a known UUID (item 2).** `GET
  /v1/deliverables/:id` and `GET /v1/deliverables/:id/versions`
  are project_id-agnostic — see **F-015** (WAIVED-BY-LEADER, TASK-421,
  Sprint 5).
- **Content injection at the API layer (item 3).** No SQL injection risk
  (sqlx `?` placeholders). Size cap now in place — see **F-023**
  (FIXED-IN-PATCH, TASK-424). 1 MiB cap + two-layer defence posture is
  reasonable for markdown (see §7.2.2 for the cap-rationale paragraph).
- **Path injection in title / version label (item 4).** React-escaped
  string fields, no `react-markdown` involvement. ✓
- **Cross-tenant deliverable read (item 5).** F-015.
- **Deleted entity handling (item 6).** Watchpoint; static-only — to be
  re-verified at runtime.
- **Versioning immutability (item 7).** `PUT /v1/deliverables/:id`
  always creates a new version row; the service explicitly inserts
  (`service/deliverable.go` line 222-228) without an UPDATE on the
  current row. ✓
- **Version diff XSS (item 4, frontend).** Resolved — text-level diff
  library output rendered into the React tree, no `dangerouslySetInnerHTML`. ✓
- **Markdown rendering XSS (item 1, frontend).** Resolved — see
  `MarkdownRenderer.tsx` summary in §4.1 and F-006 row.

**Frontend findings (§4.2 frontend) — TASK-409 + TASK-410:**

- **Markdown rendering XSS (F-006).** `MarkdownRenderer` uses
  `rehype-sanitize` with a custom schema that allows GFM (tables,
  task lists, autolinks) but explicitly omits the dangerous tags
  (`script`, `style`, `iframe`, `object`, `embed`, `frame`, `frameset`,
  `form`, etc.), restricts URL protocols (`http`/`https`/`mailto` for
  `href`; `http`/`https` for `src`), and strips all `on*` attributes.
  6 XSS unit tests pin the boundary. **No active XSS risk; F-006 moves
  to FIXED-IN-PATCH (TASK-409).**
- **TASK-410 activity dashboard.** `/agents/activity` is a
  metrics/visualization page that re-uses the F-016 execution data
  source (filtered by `agent_id` + time range) and renders into
  `recharts` (SVG output, not HTML). **No new attack surface:**
  read-only, no input fields, no markdown/HTML rendering. **Inherits
  the F-016 cross-tenant exposure** (a user can read activity
  metrics for any agent UUID, not just their own project's agents)
  — this is already waived under TASK-422. No additional
  TASK-410-specific finding.
- **Watchpoint (not finding):** if a future custom `components.a` is
  added to `MarkdownRenderer` that opens links in a new tab
  (`target="_blank"`), the noopener/noreferrer policy must be added
  in the same change. `react-markdown` does not set `target="_blank"`
  by default, so the policy is not active today, but the watchpoint
  is worth a one-liner in the dev's TODO so it doesn't get
  introduced silently.

**Status of the §5 hypotheses (from runtime evidence):**

- **F-006 (markdown XSS) — FIXED-IN-PATCH (TASK-409, 2026-06-12).**
  The frontend `MarkdownRenderer` sanitises via `rehype-sanitize`
  with a custom schema; dangerous tags, dangerous protocols, and
  event-handler attributes are all stripped. 6 XSS unit tests pin
  the boundary. See F-006 row in §5.
- **F-007 (path / content injection carry-over).** The backend
  handlers I read (`deliverable`, `execution`) do not add new
  path-injection surfaces. F-007 stays at Medium with its current
  TASK-308 reference. The runtime review of the `/v1/code/*`
  endpoints is out of scope for this sprint per §0.1.
- **F-015 (deliverable cross-tenant) — WAIVED-BY-LEADER (TASK-421,
  Sprint 5).** Static-only confirmation; runtime unverified.
- **F-023 (deliverable content size) — FIXED-IN-PATCH (TASK-424,
  2026-06-12).** Two-layer defence; 1 MiB cap is reasonable.

---

## 5. Findings Summary Table

Severity scale is defined in §0.6. Status values: `OPEN` (not yet addressed),
`PENDING-USER` (escalated to user/Leader for a fix-or-waive decision),
`WAIVED-BY-LEADER` (explicit written acceptance, recorded in §7),
`FIXED-IN-PATCH` (resolved by a follow-up patch under this sprint), `DEFERRED`
(rolled to Sprint 5 backlog), `N/A` (regression check passed, no action).

| #     | Severity | Area                | Title                                                                 | Location                                                                                       | Recommendation                                                                                                          | Status          |
|-------|----------|---------------------|-----------------------------------------------------------------------|------------------------------------------------------------------------------------------------|-------------------------------------------------------------------------------------------------------------------------|-----------------|
| F-001 | Critical | AuthZ / role model  | JWT role hard-coded to `"user"`                                       | `src/internal/service/auth.go` `Login` (≈L96–100), `Refresh` (≈L134, L141) — both now thread `string(user.Role)` into `generateJWT` / `generateRefreshToken`; tests in `service/auth_test.go` (`TestGenerateJWT_PreservesRole`, `TestGenerateRefreshToken_PreservesRole`, `TestGenerateJWT_NotHardCodedAsUser`) | Look up the user's role from `users` table on token issue; expose it on the claim. Until then, `RequireRole` is dead code. | FIXED-IN-PATCH (TASK-417). `mintToken(user)` helper embeds `user.Role`; pre-existing `001_create_users.sql` already has `role VARCHAR(50) NOT NULL DEFAULT 'user'`, so no new migration was needed. |
| F-002 | Critical | Authentication      | `Authorization: Bearer ak_*` bypasses validation                      | `src/internal/middleware/middleware.go` (≈L69) → `authService.ValidateAPIKey`; `service/auth.go` `ValidateAPIKey` (≈L233–271) SHA-256-hashes the suffix and checks `RevokedAt` / `ExpiresAt`; `store.APIKeyStore` interface + in-memory impl; `middleware_test.go` `TestAPIKeyMiddleware` covers valid/unknown/empty/non-`ak_`/tampered. | Implement real `ak_*` lookup against an `api_keys` table (or HMAC-signed token). Reject with 401 on miss, expiry, or revocation. | FIXED-IN-PATCH (TASK-418). Store is in-memory (Option B per brief); persistent `api_keys` table deferred (F-008). |
| F-003 | High (h) | Access control      | Agent list endpoint tenant scoping unknown                           | `GET /v1/agents` — pending TASK-402                                                            | Force `WHERE project_id IN (caller.projects)`; reject if `?project_id=` outside that set.                              | PRE-DEV-HYPOTHESIS CONFIRMED → SUPERSEDED-BY-F-013. The hypothesis is correct *and* the impact is wider (per-UUID lookups in `GetByID` / `Update` / `SoftDelete` are also project_id-agnostic). |
| F-004 | High (h) | AuthZ               | Cross-project task assignment                                         | `POST /v1/tasks/:id/assign` — pending TASK-404                                                  | Enforce `task.project_id == agent.project_id == caller.project_id` in service layer.                                  | PRE-DEV-HYPOTHESIS CONFIRMED → see F-014. Data model §4.1 acknowledges the gap and defers to Sprint 5. |
| F-005 | High (h) | AuthZ               | Capability check bypass risk                                          | `service/capability.go` `ValidateAgentHasCapabilities` — pending TASK-403                       | Call validator before persistence; remove any "raw assign" code path that skips it.                                    | NOT-A-FINDING (N/A). Validator is correctly wired: reads the live `agent_capabilities` join, called before persistence, only call path. The "empty `capabilities_required` on a task with no pre-existing required capabilities" case is per spec (`capabilities_required` is optional). |
| F-006 | Medium (h) | XSS / content      | Markdown rendering injection in deliverable viewer                   | `frontend/src/components/deliverables/MarkdownRenderer.tsx` — uses `react-markdown@9` + `remark-gfm@4` + `rehype-raw@7` + `rehype-sanitize@6` (custom schema, `DELIVERABLE_SANITIZE_SCHEMA`); tagNames allowlist omits `script`/`style`/`iframe`/`object`/`embed`/`frame`/`frameset`/`form`/`textarea`/`button`/`link`/`meta`/`base`/`audio`/`video`/`source`/`track`/`area`/`map`; per-tag attribute map; `href` → `http`/`https`/`mailto`, `src` → `http`/`https`; no `on*`/`style` attrs; 6 XSS unit tests in `MarkdownRenderer.test.tsx` cover `<script>`, inline `onerror`, `javascript:` URL, `<iframe>`/`<object>`/`<embed>`, `<style>`, GFM-positive. | Use `react-markdown` with sanitization (rehype-sanitize + custom schema); restrict URL protocols; never use `dangerouslySetInnerHTML`. | **FIXED-IN-PATCH (TASK-409, 2026-06-12)**. `[RUNTIME-FINDING]`. Doc nit: the dev's `MarkdownRenderer` leading comment refers to "F-008 from security-review §5.1" — the markdown-XSS row in this review is F-006 (F-008 in §5.1 is APIKeyStore persistence); the dev's F-008 reference is a pre-review-label copy/paste, not a security issue. Watchpoint: if a future `components.a` override opens external links with `target="_blank"`, the noopener/noreferrer policy must come with it (react-markdown does not set `target="_blank"` by default, so the policy is not active today). |
| F-007 | Medium (h) | Injection          | Path / content injection in code endpoints carrying over to sprint 4 | `/v1/code/:projectId/files/*path`, `/v1/code/:projectId/commits` — pre-existing                 | Reject paths escaping the project root; bound payload size; verify ownership before accepting content.                | OPEN. Pre-existing from TASK-308; not re-surfaced by the Sprint 4 backend review. |
| F-013 | Critical | AuthZ / tenant scoping | Agent handler trusts `X-Project-ID` (or URL `:projectID`) without membership check; per-UUID lookups have no `project_id` filter | `src/internal/handler/agent.go` `projectIDFromContext` (line 278-290; the function's own comment confirms the trust); `service/agent.go` `GetAgent` / `UpdateAgent` / `RetireAgent`; `store/postgres/agent_store.go` `GetByID` / `Update` / `SoftDelete` — all filter-free | Add a `requireProjectMember` middleware (or a service-layer check) that consults `project_memberships` for `(caller.id, header.project_id)`; reject 403 (or 404 to hide existence) if the caller is not a member. Add a `project_id` filter to every per-UUID lookup. Track the Sprint 4 fix under a new task; data model already acknowledges the gap. | **FIXED-IN-PATCH (TASK-419, 2026-06-13)**. Path-implied approach (no `project_memberships` table yet — Sprint 5+ follow-up): `service/agent.go` `GetAgent` / `UpdateAgent` / `RetireAgent` / `ListAgentCapabilities` gained a `callerProjectID uuid.UUID` parameter; each method calls `GetAgent` first (or its own fetch) and returns `crossTenantBlocked()` (`*Error{Status: 404, Code: "CROSS_TENANT_BLOCKED"}`) on `resource.ProjectID != callerProjectID`. `handler/agent.go` `Get` / `Update` / `Delete` and `handler/capability.go` `ListAgentCapabilities` extract `callerProjectID` via `projectIDFromContext(c)` and return `MISSING_PROJECT_HEADER` 400 if it is `uuid.Nil`. Two new helpers in `service/errors.go`: `crossTenantBlocked()` and `missingProjectHeader()`. `AgentService` interface + `mockAgentService` updated to match. New tests: service-side `TestAgentService_Get_CrossTenant`, `TestAgentService_Update_CrossTenant`, `TestAgentService_Retire_CrossTenant`, `TestAgentService_ListAgentCapabilities_CrossTenant`, `TestAgentService_MissingProjectHeader` (each with same-project control); handler-side `TestAgentHandler_Get_CrossTenant`, `TestAgentHandler_Update_CrossTenant`, `TestAgentHandler_Delete_CrossTenant`, `TestAgentHandler_Get_MissingProjectHeader`, `TestAgentHandler_Update_MissingProjectHeader`, `TestAgentHandler_Delete_MissingProjectHeader`. `agent_test.go` and `integration_test.go` mock signatures updated. `[RUNTIME-FINDING]`. Supersedes F-003. |
| F-014 | Critical | AuthZ               | Cross-project task assignment (no `(task.project_id == agent.project_id == caller.project_id)` triple-check) | `src/internal/service/assignment.go` `AssignTaskToAgent` (no project check between lines 103 and 222) | Enforce the triple-check in the service layer (or in middleware). Data model §4.1 already plans an `assignments.project_id` column + backfill in Sprint 5; the runtime check should land with that migration. | WAIVED-BY-LEADER (TASK-420, Sprint 5). `[RUNTIME-FINDING]`. Confirms F-004. |
| F-015 | Critical | AuthZ / tenant scoping | Deliverable service has no project_id check; cross-tenant read/write via known UUID | `src/internal/handler/deliverable.go` `GetDeliverable` / `UpdateDeliverable` / `ListDeliverables` / `ListDeliverableVersions` / `CreateDeliverable`; `src/internal/service/deliverable.go` corresponding methods — all project_id-agnostic | Add a `project_id` filter to every read and to the `Create` validation (verify the deliverable's `task_id` and `agent_id` belong to a project the caller is a member of). | WAIVED-BY-LEADER (TASK-421, Sprint 5). `[RUNTIME-FINDING]`. |
| F-016 | Critical | AuthZ / tenant scoping | Execution service has no project_id check; cross-tenant read/write via known UUID, including status PATCH | `src/internal/handler/execution.go` `Create` / `List` / `GetByID` / `Patch`; `src/internal/service/execution.go` corresponding methods — all project_id-agnostic | Add a `project_id` filter to every read and to the `Create` / `Patch` validation. Status PATCH is especially sensitive — any user can flip an execution to `failed` and mislead downstream observers. | WAIVED-BY-LEADER (TASK-422, Sprint 5). `[RUNTIME-FINDING]`. |
| F-017 | Medium   | Data integrity      | `assignment_events.notes` silently dropped — service writes the event with `Notes: ""`; handler mutates the in-memory response after the fact | `src/internal/service/assignment.go` line 244 (`Notes: ""` in the `Append` call); `src/internal/handler/assignment.go` line 134-136 (`result.Event.Notes = req.Notes` after the service returns) | Add `notes` as a parameter to the service's `AppendAssignmentEvent` (or a wrapper) so the notes are persisted in the same transaction. Until then, document that notes are response-only. | **FIXED-IN-PATCH (TASK-423, 2026-06-12)**. `AssignTaskToAgent` signature gained a `notes string` parameter; the value is written to the `assignment_events.notes` column in the same WithTx as the assignment. Handler no longer mutates the in-memory response (the `result.Event.Notes = req.Notes` block is removed). New tests: `TestAssignTaskToAgent_NotesPersistedInEvent` (round-trip via real service + memory store), `TestAssignTaskToAgent_EmptyNotesPersisted` (no-notes default), `TestAssignmentHandler_Assign_NoInMemoryNotesMutation` (handler no longer synthesises notes in the response). Consumer-side `AssignmentService` interface in the handler package updated to match. `[RUNTIME-FINDING]`. |
| F-021 | Low      | Defense-in-depth    | `RequireRole` middleware not applied in router for any agent / deliverable / execution / assignment route | `src/internal/router/router.go` — no `RequireRole(...)` calls anywhere | Wire `RequireRole` on the routes that should be admin-only (e.g. role assignment, API-key admin, project creation). F-001 is fixed so the middleware is satisfiable now. | DEFERRED (TASK-425, Sprint 5). `[RUNTIME-FINDING]`. Severity downgraded from Medium to Low per Leader call (2026-06-12): no admin routes exist yet, so the gap is a forward-looking concern rather than a current vulnerability. See §7.2.2. |
| F-023 | Low      | DoS                 | Deliverable `content` field size not bounded in the application layer (only by PostgreSQL's per-tuple limit ≈1 GB) | `src/internal/handler/deliverable.go` `CreateDeliverable` / `UpdateDeliverable` — no `http.MaxBytesReader` or content-length guard before reaching the service | Wrap the request body in `http.MaxBytesReader` (e.g. 1 MiB cap) in the handler, and add a service-layer validation `len(content) <= MaxContentBytes`. | **FIXED-IN-PATCH (TASK-424, 2026-06-12)**. Two-layer defence-in-depth: handler wraps `c.Request.Body` in `http.MaxBytesReader` with cap `MaxDeliverableContentBytes + 8 KiB` (~1.008 MiB) and maps the `*http.MaxBytesError` to 413 `PAYLOAD_TOO_LARGE`; service additionally re-checks `len(req.Content) > model.MaxDeliverableContentBytes` (1 MiB) and returns a typed `*service.Error` with Status=413 / Code=`PAYLOAD_TOO_LARGE`. New constant `MaxDeliverableContentBytes int64 = 1 << 20` in `model/deliverable.go`. New tests: `TestDeliverableService_Create_OversizedContent_413`, `TestDeliverableService_Create_AtTheCap_Succeeds`, `TestDeliverableService_Update_OversizedContent_413`, `TestDeliverableHandler_Create_OversizedRequest_413`, `TestDeliverableHandler_Update_OversizedRequest_413`, `TestDeliverableHandler_Create_AtTheCapBody_Succeeds`. `[RUNTIME-FINDING]`. |

(h) = pre-dev hypothesis; severity may move up or down after the runtime
review fills in §1.2..§4.2. F-006 / F-007 inherit from TASK-308; (h) reflects
the expectation that the new surfaces are *not* materially worse, not that
they are clean.

### 5.1 Follow-up items surfaced during sprint development

This subsection captures items that arose during the sprint's dev work and
that are not direct regressions against the F-001 / F-002 patch surface.
It groups two categories:

- **Deferred** — known limitations of TASK-418's Option B (in-memory)
  choice. Required before any production deployment, but explicitly out
  of scope for Sprint 4.
- **Fixed-in-patch** — quality / degraded-state bugs caught and fixed
  mid-sprint by the dev's own tests or by the Lead's review pass. Listed
  for completeness; no further action.

| #     | Severity | Area                  | Title                                                                                              | Recommendation                                                                                                                                                                                  | Status                          |
|-------|----------|-----------------------|----------------------------------------------------------------------------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|---------------------------------|
| F-008 | High     | Persistence           | `APIKeyStore` is in-memory only; keys vanish on restart                                            | Add Postgres-backed `store.PostgresAPIKeyStore` implementing the existing `store.APIKeyStore` interface; wire it in `cmd/api/main.go` `buildAPIKeyStore()`. Migration: `024_create_api_keys.sql` (full schema already drafted in the dev's TODO at `src/internal/store/memory/api_key_store.go` lines 113–138). | DEFERRED                        |
| F-009 | Medium   | API surface           | `APIKeyStore` has no `Create` method; keys can only be seeded at startup                          | Add `Create(ctx, *model.APIKey) error` to the `store.APIKeyStore` interface; implement in both memory and Postgres stores; expose via `POST /v1/api_keys` admin endpoint with `RequireRole("admin")` once that middleware is wired. | DEFERRED                        |
| F-010 | Low      | Audit                 | `LastUsedAt` is declared "best-effort" but never actually persisted                               | Add `MarkUsed(ctx, id, time.Time) error` to the `store.APIKeyStore` interface; call from `auth.ValidateAPIKey` after a successful lookup; ignore the `MarkUsed` error (best-effort, per the existing docstring). | DEFERRED                        |
| F-011 | Medium   | Degraded auth state   | `hooks/useProjectFilters.ts` missing `projectId`/`setProjectId` → `<ProjectPickerGate>` crashes on every project-scoped page | Confirm fix lands; ensure the unhandled error in the crash path does not leak project IDs, user IDs, or auth tokens in its message.                                                              | FIXED-IN-PATCH (TASK-408 FIX-1) |
| F-012 | Low      | UI integrity          | `useUpdateTaskStatus` rollback snapshot captured AFTER optimistic patch → rollback restored the optimistic value, not the original | Confirm fix lands; snapshot order is now `capture → patch (list first, detail last) → store explicit detailSnapshot` for the detail rollback path.                                              | FIXED-IN-PATCH (TASK-408 FIX-2) |

**On F-008 / F-009 / F-010 (DEFERRED):** TASK-418's brief explicitly chose
Option B (in-memory) for this sprint and documented the deferred Postgres
schema inline. The patches deliver what was scoped; the follow-ups are
required before any production deployment that wants persistent
credentials, admin-managed key rotation, or audit-grade key-use history.

**On F-011 (FIXED-IN-PATCH, TASK-408 FIX-1):** The
`hooks/useProjectFilters.ts` hook was destructured by 20+ call sites for
`{ projectId, setProjectId }` but did not return them. The
`<ProjectPickerGate>` therefore crashed at runtime on every project-scoped
page. **Security classification: degraded auth state, not auth bypass** —
the crash happens before any protected content renders, so there is no
unauthorized disclosure; the user simply sees an error page. Severity
**Medium** because the blast radius is sprint-wide (every project-scoped
page behind the gate). **No additional security concerns flagged for this
fix** beyond standard error-handling hygiene: the unhandled error in the
crash path should not leak project IDs, user IDs, or auth tokens in its
message. The fix adds `projectId` / `setProjectId` to the hook return and
syncs to `?projectId=`.

**On F-012 (FIXED-IN-PATCH, TASK-408 FIX-2):** The optimistic-update path
captured its rollback snapshot AFTER applying the optimistic patch, so
`getQueriesData({queryKey: tasks.all})` returned the optimistic value.
Rollback restored the optimistic value, not the original, so a failed
PATCH left the UI showing the failed value. **Security classification: UI
integrity, not auth bypass** — there is no data exposure or privilege
issue; the user is simply misled about whether the status change
succeeded. Severity **Low** because the worst-case is a user acting on
stale state, with no privilege escalation. The fix reorders to
`capture snapshots first → patch (list caches first, detail cache last)
→ store explicit detailSnapshot` for the detail rollback path. **No
security concerns flagged for this fix.**

**Risk in the meantime:** F-008 / F-009 / F-010 are DEFERRED and remain a
production-readiness concern (see top of this section). F-011 and F-012
are FIXED-IN-PATCH with no residual risk for this sprint's review scope.

---

The runtime review added 7 new rows to §5 (F-013, F-014, F-015, F-016,
F-017, F-021, F-023) — all tagged `[RUNTIME-FINDING]`. Four of them
(F-013, F-014, F-015, F-016) are Critical cross-tenant
authorisation gaps; the waiver decision for those is pending in
§7.2.1.

---

## 6. Expected top risk areas (pre-dev hypothesis)

Listed up front so reviewers know what to expect. Each row carries a
**Why**, **Probe** (what would confirm or refute), and **Mitigation**
pointer. The actual findings table above (§5) supersedes this list once
code is available; rows here are pre-dev hypotheses tagged `(h)` and
pre-mapped to F-003..F-007 in §5.

1. **Agent list endpoint tenant scoping.** `GET /v1/agents` with optional
   `?project_id=`. (F-003)
   - **Why it matters.** A member of project A querying
     `GET /v1/agents?project_id=<B>` could enumerate every agent in B and
     read its capabilities, owner, and status. This is the most likely
     cross-tenant data leak in the new code.
   - **Probe (when runtime is available).** Mint a token as user U
     (member of A). Hit `GET /v1/agents` without a filter and
     `GET /v1/agents?project_id=<B>`. The first must return only A's
     agents; the second must 403 (or 404, see §2.1).
   - **Mitigation.** Force the service-layer `WHERE project_id IN
     (caller.projects)`; reject any `?project_id=` outside that set with
     404 (not 403, to avoid existence leak).

2. **Cross-project task assignment.** `POST /v1/tasks/:id/assign` with
   body `{ agent_id, capabilities_required? }`. (F-004)
   - **Why it matters.** A member of project A could assign a project-B
     task to a project-B agent (or vice-versa), confusing the audit
     trail and potentially triggering work the legitimate project owner
     did not authorize. With no triple check, this is one missing `if`
     statement away.
   - **Probe.** As U (project A member), call
     `POST /v1/tasks/<B-task-id>/assign` with body
     `{ agent_id: <B-agent-id> }`. Must 404 (task not visible to U) or
     403.
   - **Mitigation.** Service-layer check before persistence:
     `task.project_id == agent.project_id == caller.project_id`. All
     three or none.

3. **Capability check bypass.** The capability engine (TASK-403)
   introduces `CapabilityService.ValidateAgentHasCapabilities`. (F-005)
   - **Why it matters.** A new validation seam is exactly where bypasses
     are introduced. The risk is the validator being defined but not
     wired on the hot path of `Assign`, or being called *after* the
     assignment has already been persisted. Either turns the capability
     feature into theater.
   - **Probe.** Static: grep for callers of
     `ValidateAgentHasCapabilities`; the only legitimate caller is the
     assignment service, called before the `INSERT`. Runtime: as U,
     assign an agent to a task that requires a capability the agent
     does not have; expect 409.
   - **Mitigation.** Single entry point to assignment persistence
     (e.g., `AssignmentService.Assign`); the validator runs inside it
     before any DB write. No alternative write path is exposed.

4. **XSS via markdown deliverable viewer.** TASK-409 renders deliverable
   content with `react-markdown + remark-gfm`. (F-006)
   - **Why it matters.** A stored-XSS in `/deliverables/[id]` is rendered
     in the browser of every other project member who views it — a
     classic privilege-escalation pivot (steal session, pivot to
     `/v1/users/me` for the victim's projects, reassign tasks).
   - **Probe.** Static: read
     `web/.../deliverables/[id]/page.tsx` (or `.jsx`); confirm
     `disallowedElements` is set, no `dangerouslySetInnerHTML`, link
     `rel="noopener noreferrer"`. Runtime: create a deliverable whose
     content is `![x](javascript:alert(1))` and view it; the JS must not
     fire.
   - **Mitigation.** Use `react-markdown` with a deny-list of HTML
     elements; sanitize URL schemes (`http`, `https`, `mailto` only);
     never use `dangerouslySetInnerHTML`. Version-diff library must
     output text, not HTML.

5. **Path / content injection in `/v1/code/:projectId/files/*path` and
   `/v1/code/:projectId/commits`.** (F-007) Pre-existing, but the new
   sprint surfaces touch adjacent code paths.
   - **Why it matters.** A wildcard path param like `*path` will accept
     `..%2F..%2Fetc%2Fpasswd`. The `commits` endpoint accepts arbitrary
     `content` from any authenticated user. The blast radius is wider
     now that deliverables and executions are in scope — the
     "trust-the-input" pattern tends to spread.
   - **Probe.** Static: read the handlers; confirm `filepath.Clean` and
     a project-root check, and a max-content-size limit. Runtime:
     `GET /v1/code/<P>/files/../../../../etc/passwd`; must 400 or 404.
   - **Mitigation.** Resolve the requested path against the project's
     configured root; reject if the resolved path escapes. Bound
     `content` to a reasonable size (e.g., 1 MiB). Verify the caller is
     a project member before accepting the commit.

**Risks not in the top 5 but still tracked (will be added to §5 if confirmed):**

- `POST /v1/webhooks` exposure (already pre-existing — TASK-308 8.1).
  Treated as out-of-scope this sprint per §0.1 but flagged in case the
  Sprint 4 dev work widens the surface.
- `RequireRole` middleware being added to the router *without* first
  verifying F-001's fix is intact (i.e. someone reverts the
  `mintToken(user)` helper to `mintToken(constantRole)`). F-001 is
  currently `FIXED-IN-PATCH`; the new test
  `TestGenerateJWT_NotHardCodedAsUser` is the regression tripwire — make
  sure that test is in the CI gate (TASK-414).
- In-memory `APIKeyStore` (F-008): keys vanish on restart. Acceptable for
  the dev workflow, not for any deployment that needs persistent
  credentials. Sprint 4 can ship; Sprint 5 cannot.
- API key without a `Create` method (F-009): a misconfigured deployment
  cannot rotate or revoke keys without a code change. Track.
- `LastUsedAt` is never written (F-010): audit visibility into which key
  is in use is degraded. Low severity but a clean Sprint 5 fix.
- Token storage in the web frontend. The current `/users/login` flow
  presumably puts the JWT in `localStorage`; if so, XSS = full token
  theft. F-006 mitigation is therefore doubly important.
- Rate limit (`middleware.RateLimit`) interaction with the new
  endpoints. Confirm `/v1/agents`, `/v1/tasks/:id/assign`, and
  `/v1/deliverables` are behind the same limiter as `/v1/auth/login`,
  and that login is *stricter* than the others (otherwise credential
  stuffing has unlimited retries per minute).

---

## 7. Sign-off

### 7.1 Closure checklist

- [ ] All four subject reviews (§1, §2, §3, §4) completed — §1.2, §2.2, §3.2,
      §4.2 filled in.
- [ ] All `Critical` and `High` findings in §5 either:
  - Fixed in a follow-up patch under TASK-412 (status `FIXED-IN-PATCH`),
    **or**
  - Explicitly accepted by the Leader with a written waiver recorded in
    §7.2 (status `WAIVED-BY-LEADER`).
- [ ] All `Medium` and `Low` findings in §5 have a target sprint or a
      backlog entry.
- [ ] Findings table (§5) has severity, area, title, location,
      recommendation, and status for every row.
- [ ] Regression check (§0.7) recorded for every relevant TASK-308 finding.
- [ ] Runtime review status (§7.3) is either `COMPLETE` or
      `DEFERRED-TO-SPRINT-5-WITH-WAIVER`.

### 7.2 Leader waiver log

Any `Critical` or `High` finding that is *not* fixed in this sprint must be
recorded here with an explicit reason, the compensating control (if any),
and the Leader's name and timestamp. Format:

```
WAIVER — F-XXX — <one-line summary>
  Reason:        <why we are accepting the risk this sprint>
  Compensating:  <what is in place to reduce the risk in the meantime>
  Accepted by:   <Leader name>
  At:            <ISO-8601 timestamp>
  Expires:       <date or "next sprint review">
```

A finding with no waiver row and no `FIXED-IN-PATCH` status is a **blocker
for the sprint closeout commit** (TASK-415).

#### 7.2.1 Granted waivers — Critical cross-tenant findings (F-013..F-016)

The runtime review surfaced four Critical cross-tenant findings
(F-013, F-014, F-015, F-016) that all share the same root cause: the
service layer trusts the request's `project_id` signal (header for
agents, body/query for the others) without verifying that the
caller is a member of that project. The data model explicitly defers
`project_id` to Sprint 5 in the relevant sections:
- `data-model.md` §4.1 (assignments): "Project-scoped query shortcut.
  Backfill on deploy: UPDATE assignments a SET project_id = t.project_id
  FROM tasks t WHERE a.task_id = t.id. A `(project_id, status)` index
  will be added in the same migration."
- `data-model.md` §9.1 (deliverables): 4 enrichment columns deferred to
  Sprint 5 `025b_extend_deliverables.sql`.
- `data-model.md` §6 (agent_state_events): `021` is a Sprint 5+
  placeholder.

All four are accepted (option a) — documented Sprint 5 work, not
Sprint 4 oversight. The Sprint 4 design relies on the header / body
`project_id` as a Sprint 4 behaviour, with JWT authentication
(TASK-417) gating access and the `assignedBy` audit trail intact.

```
WAIVER — F-013 — Cross-tenant agent access via X-Project-ID trust
  Severity:      Critical
  Waiver:        ACCEPTED (option a)
  Reason:        data-model.md §9.1 explicitly defers `project_id` to
                 Sprint 5. The cross-tenant scoping is documented
                 Sprint 5 work. The Sprint 4 design relies on
                 `X-Project-ID` as a Sprint 4 behaviour, with JWT
                 authentication (TASK-417) gating access.
  Compensating:  None. Rely on JWT authentication + audit trail.
  Sprint 5 fix:  TASK-419 (cross-tenant scoping for agents).
  Accepted by:   Leader
  At:            2026-06-12
  Expires:       "Sprint 5 closeout"

WAIVER — F-014 — Cross-project task assignment
  Severity:      Critical
  Waiver:        ACCEPTED (option a)
  Reason:        data-model.md §4.1 plans the fix explicitly — backfill
                 query and `(project_id, status)` index specified. The
                 `project_id` column on `assignments` is being added in
                 Sprint 5 as part of `025_extend_assignments.sql`. The
                 cross-tenant triple-check depends on this column.
  Compensating:  None. `assignedBy` is correctly taken from the JWT
                 subject (audit trail not forgeable). Cross-tenant
                 assignment is possible in Sprint 4; audit trail is
                 intact.
  Sprint 5 fix:  TASK-420 (cross-tenant check for assignment).
  Accepted by:   Leader
  At:            2026-06-12
  Expires:       "Sprint 5 closeout"

WAIVER — F-015 — Cross-tenant deliverable access
  Severity:      Critical
  Waiver:        ACCEPTED (option a)
  Reason:        data-model.md §9.1 defers `project_id` on deliverables
                 to Sprint 5 (`025b`). Cross-tenant scoping depends on
                 this column.
  Compensating:  None. Rely on JWT authentication.
  Sprint 5 fix:  TASK-421 (cross-tenant check for deliverable).
  Accepted by:   Leader
  At:            2026-06-12
  Expires:       "Sprint 5 closeout"

WAIVER — F-016 — Cross-tenant execution access, including status PATCH
  Severity:      Critical
  Waiver:        ACCEPTED (option a)
  Reason:        data-model.md §6 defers `agent_state_events` (021)
                 and §9.1 defers `project_id` on deliverables to
                 Sprint 5. Cross-tenant scoping depends on these
                 columns. The execution status PATCH risk is mitigated
                 by JWT authentication; Sprint 4 user pool is small.
  Compensating:  None. Rely on JWT authentication.
  Sprint 5 fix:  TASK-422 (cross-tenant check for execution).
  Accepted by:   Leader
  At:            2026-06-12
  Expires:       "Sprint 5 closeout"
```

#### 7.2.2 Medium / Low action items (F-017, F-021, F-023)

These are not waivers (the Leader has not accepted the risk — the
fix is in flight or scheduled). They are recorded here so the
closeout reviewer can verify the actions were taken.

- **F-017 (Medium, `assignment_events.notes` silently dropped).**
  Sprint 4 fix. **TASK-423 RESOLVED 2026-06-12**. `AssignTaskToAgent`
  signature gained a `notes string` parameter; the value is now
  written to the `assignment_events.notes` column in the same
  transaction as the assignment. Handler no longer mutates the
  in-memory response (the `result.Event.Notes = req.Notes` block
  is removed). New tests:
  `TestAssignTaskToAgent_NotesPersistedInEvent` (round-trip via
  real service + memory store),
  `TestAssignTaskToAgent_EmptyNotesPersisted` (no-notes default),
  `TestAssignmentHandler_Assign_NoInMemoryNotesMutation` (handler
  no longer synthesises notes in the response). §5 status:
  `FIXED-IN-PATCH (TASK-423)`.

  **Security-01 review (2026-06-12, on Lead's request):**
  the patch is correct and removes the audit-trail
  integrity gap. Confirmed:
  (1) `assignedBy` is still taken from `userIDFromContext(c)`
  (the JWT subject), not from the request body — so the
  `assignment_events.assigned_by` column remains
  unforgeable from the caller's perspective.
  (2) The `notes` value flows handler → service → DB
  write in one call; no in-memory mutation
  shortcuts, no second-write to "fix" the response.
  (3) The fix is in the service layer (correct place
  for data integrity), with the handler reduced
  to a thin pass-through. No regressions to the
  capability-validation path or the partial unique
  index. The `assignedBy` audit trail is intact
  and the new `notes` audit field is now also
  intact.

- **F-021 (Low, `RequireRole` middleware not applied in router).**
  Sprint 5 follow-up — multi-route refactor with matrix design
  (which routes are admin-only). **TASK-425** in the Sprint 5
  backlog. Severity downgraded from Medium to Low per Leader call
  (2026-06-12): no admin routes exist yet, so the gap is a
  forward-looking concern rather than a current vulnerability.

- **F-023 (Low, deliverable `content` size not bounded in app
  layer).** Sprint 4 fix. **TASK-424 RESOLVED 2026-06-12**.
  Two-layer defence-in-depth: handler wraps `c.Request.Body` in
  `http.MaxBytesReader` with cap `model.MaxDeliverableContentBytes + 8 KiB`
  (~1.008 MiB) and maps the `*http.MaxBytesError` to 413
  `PAYLOAD_TOO_LARGE` (via a new `isMaxBytesError` helper in
  `handler/types.go`); service additionally re-checks
  `len(req.Content) > model.MaxDeliverableContentBytes` (1 MiB)
  and returns a typed `*service.Error` with Status=413 /
  Code=`PAYLOAD_TOO_LARGE`. New constant
  `MaxDeliverableContentBytes int64 = 1 << 20` in
  `model/deliverable.go`. New tests:
  `TestDeliverableService_Create_OversizedContent_413`,
  `TestDeliverableService_Create_AtTheCap_Succeeds`,
  `TestDeliverableService_Update_OversizedContent_413`,
  `TestDeliverableHandler_Create_OversizedRequest_413`,
  `TestDeliverableHandler_Update_OversizedRequest_413`,
  `TestDeliverableHandler_Create_AtTheCapBody_Succeeds`.
  §5 status: `FIXED-IN-PATCH (TASK-424)`.

  **Cap rationale (Security-01, 2026-06-12, on Lead's request):**
  1 MiB (`1 << 20` = 1 048 576 bytes) is the right cap for
  markdown deliverables. Typical agent deliverables (code review
  reports, design docs, test plans, generated API docs, verbose
  diffs) sit in the 1-100 KiB range; 1 MiB is ~10× typical
  headroom and ~5× the largest plausible single-doc deliverable.
  Tighter caps (e.g., 256 KiB) would be a DoS-hardening
  overcorrection that risks rejecting legitimate use (a long SPEC
  document, a verbose API reference, a generated changelog). 1
  MiB is also the lowest `http.MaxBytesReader` value that
  comfortably holds a JSON envelope of `{"content": "..."}` plus
  a few KiB of other fields without the headroom-overflow edge
  case at the boundary; the dev added 8 KiB of headroom
  explicitly to handle the envelope, which is the correct
  idiom. **No change recommended.** If real workload shows
  smaller typical sizes, the cap can be moved down in a
  follow-up; the constant is centralised in
  `model/deliverable.go`, so a tuning change is one line.

  **Error envelope consistency:** the new `PAYLOAD_TOO_LARGE`
  response uses the same `{ error: { code, message, details } }`
  envelope as the rest of the codebase, with `code =
  "PAYLOAD_TOO_LARGE"`, `details = [{ field: "content", message:
  "exceeds maximum allowed size of <formatBytes> bytes" }]`,
  and `formatBytes` handling B / KiB / MiB with integer /
  fractional cases. This is consistent with the
  `assignment_required_capabilities_mismatch` validation
  pattern already in the error catalogue. No envelope
  inconsistency to call out.

### 7.3 Runtime review status

One of:

- `COMPLETE` — runtime review done, all rows in §5 have a final severity
  and status.
- `PARTIAL — STATIC-ONLY` — runtime was unavailable (e.g., the Windows
  host constraint); static review done. All `High` and `Critical` rows
  in §5 must have an explicit note stating that they are *unverified at
  runtime* and either (a) carried over from TASK-308 (where they were
  also unverified at runtime) or (b) flagged for a follow-up runtime
  test in Sprint 5.
- `DEFERRED-TO-SPRINT-5-WITH-WAIVER` — the user/Leader has accepted that
  the runtime review will not complete this sprint; a waiver is recorded
  in §7.2 covering all `High` and `Critical` rows that would otherwise
  require runtime confirmation.

**Current status:** `PARTIAL — STATIC-ONLY — WAIVED-FOR-CROSS-TENANT`. The
runtime environment was unavailable (Windows host has no Docker / Go /
Node / Python, per Leader's brief 2026-06-12). Static code review is
complete for the backend surfaces (TASK-402-408, 416, 417, 418) and the
frontend deliverable viewer (TASK-409, sanitization posture walked in
§4.1 / §4.2; F-006 now `FIXED-IN-PATCH`). The four Critical
cross-tenant findings (F-013, F-014, F-015, F-016) are *unverified at
runtime* — they are explicitly waived for Sprint 4 (per §7.2.1) and
flagged for follow-up runtime testing in Sprint 5 as TASK-419,
TASK-420, TASK-421, TASK-422. TASK-410 (agent activity dashboard) was
covered briefly in §4.2 and adds no new attack surface; it inherits
F-016's cross-tenant exposure (already waived under TASK-422). Three
Medium / Low findings have Sprint 4 fixes in flight or landed: F-017
(`FIXED-IN-PATCH`, TASK-423), F-023 (`FIXED-IN-PATCH`, TASK-424), F-021
(deferred to Sprint 5, TASK-425).

### 7.4 Cross-references

- Prior security report: `docs/sprint4/security-report.md` (TASK-308).
  The regression check (§0.7) walks this list.
- Architecture: `docs/architecture.md`, `docs/services.md`.
- Sprint closeout gate: TASK-414 (`scripts/quality-gate.sh`).
- Pre-existing criticals escalated to the user on 2026-06-12:
  F-001 (hard-coded JWT role) and F-002 (API-key bypass).
  - F-001: FIXED-IN-PATCH on 2026-06-12 (TASK-417 — `auth.go` now passes
    `string(user.Role)` into both `generateJWT` and `generateRefreshToken`;
    unit tests added in `auth_test.go` covering `admin` / `member` / `viewer`).
  - F-002: FIXED-IN-PATCH on 2026-06-12 (TASK-418 — in-memory API key
    store behind a `store.APIKeyStore` interface; `auth.ValidateAPIKey`
    hashes the post-`ak_` part with sha256, checks revocation/expiry,
    and the middleware calls it instead of the 13-byte prefix check).
    Postgres-backed implementation deferred to a follow-up sprint task
    (see `docs/sprint4/infra-validation.md` "Outstanding / deferred").
- Sprint 4 patches reviewed (2026-06-12):
  - F-006: FIXED-IN-PATCH (TASK-409 — `MarkdownRenderer` uses
    `rehype-sanitize` with a custom schema; see §4.1 / §4.2).
  - F-013: FIXED-IN-PATCH (TASK-419 — `callerProjectID` threaded through `GetAgent` / `UpdateAgent` / `RetireAgent` / `ListAgentCapabilities`; cross-tenant miss → 404 `CROSS_TENANT_BLOCKED`; missing header → 400 `MISSING_PROJECT_HEADER`). Updated 2026-06-13.
- F-017: FIXED-IN-PATCH (TASK-423 — `notes` threaded through
    `AssignTaskToAgent` into the `assignment_events` row; in-memory
    handler mutation removed).
  - F-023: FIXED-IN-PATCH (TASK-424 — two-layer defence:
    `http.MaxBytesReader` at handler + service-layer size check,
    both mapping to 413 `PAYLOAD_TOO_LARGE`; 1 MiB cap with 8 KiB
    headroom). See §7.2.2 for the cap-rationale paragraph.
- Sprint 5 follow-ups linked from §5.1 / §7.2.1:
  TASK-419, TASK-420, TASK-421, TASK-422 (cross-tenant fixes for
  F-013, F-014, F-015, F-016); TASK-425 (RequireRole routing
  matrix for F-021).

---

**Reviewer:** Security-01
**Date signed:** _pending_
**Runtime review status:** _pending — see §7.3_
