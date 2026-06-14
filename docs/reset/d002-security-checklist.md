# D-002 Security Review — Leader Support Checklist

**Owner of D-002:** Guardian. **Support:** Leader.

This is the prep I (Leader) will keep updated as the review progresses. Guardian
is the final approver; I curate the checklist, cross-reference prior work, and
help chase down open items.

## Source material (already in repo)
- `docs/threat-model.md` — full STRIDE-style threat model, 1.0 draft, 2026-06-10
- `docs/auth-design.md` — auth + session design
- `docs/security.md` — security overview
- `docs/sprint4/security-review.md` + `security-report.md` + `security-audit-design.md` — Sprint 4 findings
- `src/internal/middleware/middleware.go` + `middleware_test.go`
- `src/internal/handler/auth.go` + `internal/handler/user.go` + `internal/handler/webhook.go`
- `src/internal/router/router.go` + `router_role_matrix_test.go` (role matrix is
  a known hard spot per Sprint 4 quality-gate)
- `src/internal/validation/validate.go`
- `src/internal/service/auth.go` + `auth_test.go`

## Review checklist (5 buckets)

### 1. Authorization
- [ ] Role matrix in `router_role_matrix_test.go` covers every endpoint in
      `router.go` (no 404-from-auth gaps).
- [ ] Agent lifecycle endpoints enforce that the caller is the owning user OR
      has `admin` role.
- [ ] Assignment endpoints enforce project membership.
- [ ] Deliverable reads/writes enforce project membership.
- [ ] Webhook endpoints verify HMAC signature (no shared secret in source).
- [ ] Admin-only paths (user mgmt, role mgmt) gate correctly.

### 2. Access review (per-asset)
For each asset in `threat-model.md` §2.1 (D-001..D-008), confirm the access
control is implemented AND tested:
- [ ] D-001 Source code — gated behind project ACL
- [ ] D-002 Build artifacts — same
- [ ] D-003 Agent execution logs — restricted; never exposed via public API
- [ ] D-004 Credentials & tokens — encrypted at rest; never logged
- [ ] D-005 Project configuration — env vars never serialized in API response
- [ ] D-006 Audit logs — append-only; admin-only read
- [ ] D-007 Agent state & memory — same scope as the project
- [ ] D-008 Deployment infra state — restricted

### 3. Threat analysis
- [ ] STRIDE walkthrough per service in `threat-model.md` §3-7
- [ ] OWASP Top-10 (2021) check on the handler layer
- [ ] SSRF on outbound calls (webhooks, agent runtime)
- [ ] Injection on every query path (`store/postgres/*.go` — parameter
      binding, no string concat)
- [ ] IDOR on path-param endpoints (project_id, agent_id, task_id)
- [ ] Privilege escalation via role matrix gaps

### 4. Secret hygiene
- [ ] No API keys, tokens, credentials in any committed file (gitleaks
      must be green; this is also E-002)
- [ ] No AI conversation data in repo
- [ ] No temp execution logs
- [ ] `.env.example` lists variables but not values
- [ ] `.gitignore` covers: `.env*`, `*.key`, `*.pem`, `*.p12`, `secrets/`

### 5. Dependencies
- [ ] `go.mod` — no known-vulnerable versions (run `govulncheck ./...`)
- [ ] `frontend/package.json` — `npm audit` clean or all findings triaged
- [ ] Lockfiles committed (`go.sum`, `package-lock.json`)

## Output deliverable
`docs/reset/security-review.md` — Guardian owns; Leader supports.

Structure:
- Executive summary
- Findings table: `ID | Severity | Asset | Description | Status (OPEN/RESOLVED)`
- For RESOLVED: link to the commit that closed it
- For OPEN: mitigation plan + target sprint

## Status (filled by Leader as work progresses)
2026-06-14: prep started, checklist drafted, awaiting Builder A-group ship
to make sure the surface is stable before deep review.

## Preliminary findings (Leader pre-audit)

### F-D002-001 — Webhook SSRF guard is a no-op stub (HIGH, pre-B-002)
- Location: `src/internal/service/webhook.go:115` — `validateWebhookURL` returns
  `nil` for every input. The function comment is honest: "TODO: implement
  full SSRF protection".
- Exploit path (current): none — no dispatcher reads `webhook.URL` to make an
  outbound HTTP call. `Grep` for `webhook.URL` + `http.` shows only the model
  test fixture.
- Exploit path (after B-002 ships): any authenticated user (developer+) can
  `POST /webhooks { "url": "http://169.254.169.254/latest/meta-data/iam/..." }`
  and have the server fetch AWS / GCP / Azure instance metadata, internal
  services on `localhost` or `127.0.0.1`, link-local addresses, etc.
- Severity when B-002 lands: **HIGH** (cloud credential exfil, lateral movement).
- Recommendation: implement `validateWebhookURL` BEFORE B-002 wires the
  dispatcher. Options: (a) `github.com/nathan-osman/go-ssrf`, (b) handwritten
  allowlist + DNS resolution check + IP range block (RFC 1918, link-local,
  loopback, multicast, IPv6 ULA, IPv6 link-local).
- Owner of fix: Builder (webhook service) — surfaced from D-002 prep.

### F-D002-002 — API key F-002 fix verified (CLOSED in prior sprint)
- `authService.ValidateAPIKey` now hashes the token before lookup, checks
  revocation + expiry, returns ErrUnauthorized on any failure.
- The middleware (TASK-405 era) closed the previous prefix-only bypass.
- No action; logged for the audit trail.

### F-D002-003 — Auth cookie set with `secure=true` (INFO, dev/prod tension)
- `handler/auth.go:43` — refresh-token cookie is `HttpOnly; Secure;
  SameSite=Strict`. In local dev over HTTP, browsers will not send the
  cookie back, so the refresh flow breaks unless dev is run over HTTPS.
- Mitigation: dev `docker-compose.yml` should either terminate TLS locally
  or use `SameSite=Lax` only when `APP_ENV=development`. Confirm with
  Ops during E-001.

### F-D002-004 — Project-membership check never landed (HIGH, IDOR, carried from Sprint 4)
- Cross-references: F-013 (TASK-419) and F-014 (TASK-420) in
  `docs/sprint4/security-review.md`. Both are documented as
  `FIXED-IN-PATCH ... path-implied (no project_memberships table yet —
  Sprint 5+ follow-up)`.
- Current state: `handler/agent.go:projectIDFromContext` and
  `service/assignment.go:AssignTaskToAgent` (and the rest of the
  path-implied checks) verify `resource.ProjectID == callerProjectID`
  (the X-Project-ID header value). They do NOT verify that the
  authenticated user is a member of `callerProjectID`.
- Attack: any authenticated user can submit
  `X-Project-ID: <known-project-UUID>` and the service will allow
  read/write to resources in that project. The 128-bit UUID is not
  brute-forceable, but it leaks via URLs, screenshots, social
  engineering, or compromised log files.
- Real fix (Sprint 6+ — OUT OF SCOPE for this combined Sprint 4+5
  per the "ignore unfinished Sprint 4/5" directive):
  1. Add the `project_memberships(user_id, project_id, role)` table
     (data model §4.1 already plans it).
  2. Add a `requireProjectMember` middleware (or service-layer check)
     that joins `project_memberships` on `(caller.id, header.project_id)`.
  3. Replace the `callerProjectID == uuid.Nil` early-return with a
     `caller is not a member of callerProjectID` check.
  4. Add per-UUID lookups to a `project_id` filter (already done in
     service layer for the touched paths).
- This sprint: log as D-002 OPEN finding. Surface to Guardian as the
  top item for the security review report. No code change.

