# D-002 — Security Review (Sprint 6)

| Field        | Value                                                                |
|--------------|----------------------------------------------------------------------|
| Owner        | Guardian (slot `019ec4fe-604f-7551-9a76-38a621ddd256`)               |
| Status       | **APPROVED with conditions**                                         |
| Reviewed     | commit `8b85d26` (merge of `origin/main` @ `64c0d09` into `guardian/d-002-security-review`) |
| Date         | 2026-06-14                                                           |
| Sign-off     | Conditional: B-002 must close F-D002-001 before merge; E-001 must harden F-D002-003 |

---

## 1. Executive Summary

The D-002 review covers the codebase as of `8b85d26` against the four preliminary findings in `docs/reset/d002-security-checklist.md`, the 18 Sprint 4 findings in `docs/sprint4/security-report.md`, and a fresh walk of the checklist buckets (auth, access, threat, secret hygiene, dependencies).

**Headline results:**

- **4 preliminary findings**: 2 HIGH OPEN (`F-D002-001` webhook SSRF stub, `F-D002-004` IDOR via `X-Project-ID`), 1 CLOSED (`F-D002-002` API key fix verified), 1 INFO OPEN (`F-D002-003` auth cookie `secure=true` dev/prod tension).
- **18 Sprint 4 findings**: 6 fixed in patch (`F-001`, `F-002`, `F-006`, `F-021`, `F-017`, `F-023`), 4 cross-tenant (`F-013/14/15/16`) PARTIALLY MITIGATED by the path-implied fix in `TASK-419..422` and WAIVED-BY-LEADER for the table-level fix, 8 still OPEN with explicit Sprint 6+ targets in the table below.
- **14 new findings** filed from this review's walk of the checklist.
- **Dependencies**: 3 dev-dep advisories in `frontend/package.json` (1 LOW, 2 MODERATE; none runtime-exposed). `govulncheck` is BLOCKED locally (no Go toolchain on this host) — recommend Ops add it to CI in the next gate pass.
- **Test coverage** for the security surface is generally good (role matrix has 7 explicit tests, middleware has 5 `APIKeyMiddleware` subtests covering the `F-002` fix, `TestRoleMatrix_AdminOnly_Register_RejectsViewer` covers `F-021`). Gaps remain in `handler/auth.go`, `handler/webhook.go`, and `handler/project.go` (no `_test.go`).

**Routing:**

- `F-D002-001` → **B-002 wave** (handwritten `webhooks_safety.go`, ~150 lines; brief in `docs/reset/fix-f-d002-001-webhook-ssrf.md`).
- `F-D002-003` → **E-001 follow-up** (feature-flag the `secure` cookie flag; default off in dev, on in prod).
- `F-D002-004` → **Sprint 6+** (`project_memberships` table + `requireProjectMember` middleware; out of scope per brief).
- `F-D002-005..018` → **Sprint 7+** (logged, not blocking; see §5).

---

## 2. Scope & Methodology

**Reviewed:**

- 23 service files in `src/internal/service/`
- 7 handler files in `src/internal/handler/`
- 2 router files in `src/internal/router/` (incl. role matrix tests)
- 4 middleware files in `src/internal/middleware/`
- 1 validation file in `src/internal/validation/`
- `docs/threat-model.md`, `docs/auth-design.md`, `docs/sprint4/security-report.md`
- `frontend/package.json` (npm audit), `src/go.mod` (no local Go)
- All `*_test.go` files in `src/internal/` (32 files; cross-referenced)

**Trust boundaries (from `docs/threat-model.md`):**

1. External network → edge
2. Edge → app
3. App → data

**Threat actors considered:**

- Malicious authenticated user (cross-tenant, IDOR, abuse of API key, rate-limit evasion)
- Script kiddie (SSRF, basic auth brute-force, common web vulns)
- Trusted insider (revoke/expire surface, audit log gaps)

**Not in scope (per brief):**

- `project_memberships` table
- `requireProjectMember` middleware
- Logging, not fixing.

---

## 3. Findings Table

Legend: **Sev** = HIGH | MEDIUM | LOW | INFO. **Status** = OPEN | CLOSED | MITIGATED | WAIVED.

### 3.1 Preliminary findings (from `d002-security-checklist.md`)

| ID         | Sev   | Asset                       | Description                                                                                              | Status   | Owner / Target       |
|------------|-------|-----------------------------|----------------------------------------------------------------------------------------------------------|----------|----------------------|
| F-D002-001 | HIGH  | `service/webhook.go:115`    | `validateWebhookURL` is a stub that returns `nil` for any input. Comment is honest: "TODO: implement full SSRF protection". The handler accepts any URL, including loopback, RFC 1918, link-local, and 169.254.0.0/16 (cloud metadata). | OPEN     | B-002 wave (in scope) |
| F-D002-002 | CLOSED| `service/auth.go:232-270`   | `ValidateAPIKey` correctly SHA-256 hashes the presented key, compares to the `key_hash` column, checks `RevokedAt == nil` and `ExpiresAt` (with `nil` expiry = no expiry). Confirmed in code + `TestAPIKeyMiddleware` 5 subtests. | CLOSED @ TASK-418 | — |
| F-D002-003 | INFO  | `handler/auth.go:43`        | Refresh-cookie `SetCookie` uses `secure=true, httpOnly=true`. The `secure=true` flag breaks local HTTP dev unless the operator is on HTTPS. No env-var toggle observed. | OPEN     | E-001 follow-up       |
| F-D002-004 | HIGH  | `handler/agent.go:299-311` (and analogous in 5 other handlers) | `projectIDFromContext` reads `X-Project-ID` header with priority 1 over the URL path. Header is TRUSTED with no membership check. An authenticated user in Project A can spoof `X-Project-ID: <B-UUID>` and: (a) read Project B's agents — the service-layer `callerProjectID` check in `service/agent.go` correctly rejects (returns `CROSS_TENANT_BLOCKED` 404), so reads are blocked; (b) **CREATE** an agent in Project B by sending the spoofed header on `POST /v1/agents` — the `ProjectID` is set from the request, the agent lives in B's namespace, but the caller has no membership in B. Pollutes B's namespace, can be used to set up sleeper accounts, evades per-project rate limits. | OPEN (partially mitigated) | Sprint 6+ (`project_memberships` + `requireProjectMember`); out of scope per brief |

### 3.2 Sprint 4 findings (from `sprint4/security-report.md`) — status update

| ID    | Sev     | Description                                                        | Status                                                 | Evidence                                                                                  |
|-------|---------|--------------------------------------------------------------------|--------------------------------------------------------|-------------------------------------------------------------------------------------------|
| F-001 | HIGH    | Hard-coded JWT role on mint                                        | FIXED                                                  | `service/auth.go:mintToken` reads role from DB row, not constant (TASK-417)                |
| F-002 | HIGH    | `ak_*` API key bypass via header pre-check                         | FIXED + tested                                         | `service/auth.go:ValidateAPIKey` + `middleware_test.go:TestAPIKeyMiddleware` (5 subtests)  |
| F-006 | MEDIUM  | Markdown XSS in deliverable rendering                              | FIXED                                                  | Renderer escape pass (TASK-409)                                                           |
| F-008 | HIGH    | Webhook SSRF — **same as F-D002-001**                              | OPEN (rolled into F-D002-001)                          | F-D002-001 above                                                                          |
| F-013 | CRITICAL| Cross-tenant agent read (Project A reads Project B)                | MITIGATED-PARTIAL (path-implied) + WAIVED-BY-LEADER    | `service/agent.go:GetAgent` checks `a.ProjectID != callerProjectID` → 404                 |
| F-014 | CRITICAL| Cross-tenant agent write                                           | MITIGATED-PARTIAL + WAIVED-BY-LEADER                   | Same pattern in `UpdateAgent` (calls `GetAgent` first)                                   |
| F-015 | CRITICAL| Cross-tenant assignment                                            | MITIGATED-PARTIAL + WAIVED-BY-LEADER                   | `service/assignment.go:AssignTaskToAgent` triple-checks `callerProjectID == task.ProjectID == agent.ProjectID` |
| F-016 | CRITICAL| Cross-tenant deliverable                                           | MITIGATED-PARTIAL + WAIVED-BY-LEADER                   | `service/deliverable.go:CreateDeliverable` checks `task.ProjectID != callerProjectID`     |
| F-017 | MEDIUM  | Assignment notes not persisted                                     | FIXED                                                  | `service/assignment.go:282` — `Notes: notes` is set on the event row                      |
| F-021 | HIGH    | `POST /v1/auth/register` was unauthenticated                       | FIXED + tested                                         | `router/router.go` gates register behind `RequireRole(admin)`; `TestRoleMatrix_AdminOnly_Register_RejectsViewer` confirms |
| F-023 | MEDIUM  | Deliverable content size unbounded                                 | FIXED                                                  | `service/deliverable.go:122-127, 292-297` — `MaxDeliverableContentBytes` enforced         |

### 3.3 New findings (this review's checklist walk)

| ID          | Sev | Asset                                          | Description                                                                                                                       | Status | Target  |
|-------------|-----|------------------------------------------------|-----------------------------------------------------------------------------------------------------------------------------------|--------|---------|
| F-D002-005  | LOW | `router/router.go`                             | `POST /v1/webhooks` is gated by `RequireAnyRole(writeRole)` (developer+admin). Bulk-register of a webhook in another project's namespace is implicitly possible if the caller spoofs `X-Project-ID` (couples to F-D002-004). Should be admin-only. | OPEN   | Sprint 7|
| F-D002-006  | INFO| `service/webhook.go:webhookEventAllowlist`     | Allowed events are a hard-coded slice in code. Adding an event requires a code change + redeploy. Should be config-driven for ops. | OPEN   | Sprint 7|
| F-D002-007  | INFO| `service/assignment.go:AssignTaskToAgent` notes| `Notes` field is unbounded (no `MaxLength` check). 4kB is typical. Low risk (not rendered as HTML), but a noisy client could fill the DB. | OPEN   | Sprint 7|
| F-D002-008  | LOW | `middleware/middleware.go:isPublic`            | `isPublic(path)` does prefix match (`strings.HasPrefix`). A future public path `/v1/audit` would be a prefix of (or mistaken for) `/v1/audit-logs`. Current set is fine; pattern is fragile. | OPEN   | Sprint 7|
| F-D002-009  | LOW | `middleware/middleware.go` rate limit          | Rate limit is per-IP only. An attacker with many IPs (e.g. botnet) can amplify. Per-user rate limit (keyed on `user_id` from JWT/API key) recommended for the v1 GA. | OPEN   | Sprint 7|
| F-D002-010  | INFO| Edge / reverse proxy                            | No CSP, X-Frame-Options, X-Content-Type-Options, Referrer-Policy, or Permissions-Policy observed in `router/router.go` (handlers attach these via middleware only if added). Frontend is served by Next.js; backend JSON has no HTML surface, so impact is low. Should be added at the reverse proxy. | OPEN   | Sprint 7|
| F-D002-011  | INFO| `service/auth.go:RevokeAPIKey` (assumed path)  | API key revoke/expire returns no audit event. Recommend logging `(user_id, key_id_prefix, action=revoke, actor_id, ts)` for compliance. No audit log middleware observed. | OPEN   | Sprint 7|
| F-D002-012  | LOW | `frontend/package.json` — `diff`               | `diff@>=6.0.0 <8.0.2` (resolved via `next` or `jsdiff`); advisory: DoS in `parsePatch`/`applyPatch` when fed untrusted input. **Dev dep only**; not runtime-exposed in the production build (Next.js static export). | OPEN   | Sprint 7|
| F-D002-013  | MOD  | `frontend/package.json` — `next` + `postcss`   | `postcss@<8.5.10` (via `next@<=16.3.0-canary.5`); advisory: XSS via unescaped `</style>` in CSS Stringify Output. **Build-time only**; not runtime-exposed. Fix: pin `postcss` to `^8.5.10` and bump `next` to 9.3.3 (npm-suggested minor). | OPEN   | Sprint 7|
| F-D002-014  | INFO| `src/go.mod`                                    | `govulncheck` not run locally (no Go toolchain on this host). CI's `build-go.yml:0` does not include a `govulncheck` step. **Recommend** Ops add `go install golang.org/x/vuln/cmd/govulncheck@latest && govulncheck ./...` to the gate pass. | OPEN   | Ops (next gate pass) |
| F-D002-015  | INFO| `handler/auth.go` test gap                      | No `handler/auth_test.go`. Login/Refresh/Logout happy + sad paths are uncovered. Recommend table-driven tests (mirror `router_role_matrix_test.go` style). | OPEN   | Sprint 7|
| F-D002-016  | INFO| `handler/webhook.go` test gap                   | No `handler/webhook_test.go`. Register/List/Delete happy + sad paths uncovered. Coupled to F-D002-001 (B-002 will need tests for the SSRF fix). | OPEN   | Sprint 7|
| F-D002-017  | INFO| `handler/project.go` test gap                   | No `handler/project_test.go`. CRUD + member-list happy + sad paths uncovered. | OPEN   | Sprint 7|
| F-D002-018  | INFO| `service/capability.go:SetCapabilities`         | Re-reads capability cache after DB write. If the cache re-read fails, the version on the returned agent doesn't reflect the cap change. Low risk (caller can retry). Recommend: best-effort cache refresh with explicit `cacheStale=true` flag in the result. | OPEN   | Sprint 7 |

---

## 4. RESOLVED — Commit links

| ID    | Commit / PR      | Note                                                                                  |
|-------|------------------|---------------------------------------------------------------------------------------|
| F-001 | TASK-417         | `mintToken` reads role from DB row, not hard-coded constant.                          |
| F-002 | TASK-418         | `ValidateAPIKey` SHA-256 + `RevokedAt`/`ExpiresAt` checks; covered by `TestAPIKeyMiddleware` (5 subtests). |
| F-006 | TASK-409         | Markdown XSS escape pass in renderer.                                                 |
| F-008 | (rolled into F-D002-001) | Same root cause; F-D002-001 is the active ticket.                           |
| F-013 | TASK-419         | `service/agent.go:GetAgent` — `a.ProjectID != callerProjectID` → 404.                 |
| F-014 | TASK-420         | `service/agent.go:UpdateAgent` — calls `GetAgent` first.                              |
| F-015 | TASK-421         | `service/assignment.go:AssignTaskToAgent` — triple check.                             |
| F-016 | TASK-422         | `service/deliverable.go:CreateDeliverable` — cross-tenant guard.                      |
| F-017 | (code fix, no TASK)| `service/assignment.go:282` — `Notes: notes` persisted to event row.                |
| F-021 | TASK-425         | `router/router.go` — register gated by `RequireRole(admin)`; `TestRoleMatrix_AdminOnly_Register_RejectsViewer` confirms. |
| F-023 | (code fix, no TASK)| `service/deliverable.go:122-127, 292-297` — `MaxDeliverableContentBytes` cap.      |
| A-001-followup | `64c0d09` | Role length tightened to 80 chars (code + DB migration `027`).                        |

---

## 5. OPEN — Mitigation plans

### 5.1 In-scope (B-002 wave)

- **F-D002-001 (HIGH, webhook SSRF stub)**
  - **Fix brief**: `docs/reset/fix-f-d002-001-webhook-ssrf.md`.
  - **Recommended path**: Option C (handwritten). New file `src/internal/service/webhooks_safety.go` (~150 lines).
  - **Minimum bar**: scheme allowlist (https only), port 443 only, DNS resolve + IP blocklist covering loopback, RFC 1918, link-local (169.254.0.0/16), CGNAT (100.64.0.0/10), multicast, reserved, IPv6 ULA (fc00::/7) and link-local (fe80::/10), length cap on URL/host.
  - **Stretch**: TOCTOU mitigation via custom `DialContext` that pins the resolved IP for the duration of the request.
  - **Tests**: table-driven, with `127.0.0.1`, `10.0.0.1`, `169.254.169.254` (AWS metadata), `[::1]`, `[fc00::1]`, and a legit URL all as cases.
  - **Coupled findings**: F-D002-016 (handler test gap) — fix should ship with tests.

### 5.2 In-scope (E-001 follow-up)

- **F-D002-003 (INFO, secure cookie dev/prod)**
  - **Fix**: add `AUTH_COOKIE_SECURE` env var (default `true` in prod, `false` in dev). Read in `handler/auth.go` at startup, store in handler config struct.
  - **Tests**: parametrize `TestAuthMiddleware` with `secure=true/false`; confirm the `Set-Cookie` header carries the flag.
  - **Doc**: update `docs/auth-design.md` §Refresh Tokens to note the env var.

### 5.3 Out-of-scope (Sprint 6+)

- **F-D002-004 (HIGH, IDOR via X-Project-ID)**
  - **True fix**: `project_memberships` table `(user_id, project_id, role, created_at)` + `requireProjectMember` middleware.
  - **Why deferred**: per Lead's brief, the path-implied fix in `TASK-419..422` is the v1 control; the table-level fix is Sprint 6+ work.
  - **Until then**: header trust remains. Recommend a `/v1/projects` self-list endpoint (where users see which projects they're a member of) so the header set is no longer "blind" — Sprint 6+.

### 5.4 Logged, not blocking (Sprint 7+)

- F-D002-005 — admin-only on `POST /v1/webhooks`.
- F-D002-006 — webhook event allowlist in config.
- F-D002-007 — `MaxLength` on assignment `Notes`.
- F-D002-008 — exact-match instead of `HasPrefix` in `isPublic`.
- F-D002-009 — per-user rate limit.
- F-D002-010 — security headers at the reverse proxy.
- F-D002-011 — API key revoke/expire audit log.
- F-D002-012..013 — npm dev-dep version pins.
- F-D002-015..017 — handler test coverage.
- F-D002-018 — capability cache race window.

---

## 6. Dependencies output

### 6.1 npm audit (run 2026-06-14, local Node 11.13.0)

Local Node version is below the npm audit DB's supported range — the report below is what `npm audit --json` returned; CI's Node version is canonical.

| Package | Range                              | Severity | Title                                                        | Fix path                                      |
|---------|------------------------------------|----------|--------------------------------------------------------------|-----------------------------------------------|
| `diff`  | `>=6.0.0 <8.0.2`                   | LOW      | DoS in `parsePatch`/`applyPatch` on untrusted input          | Bump to `^9.0.0` (major)                      |
| `next`  | `>=9.3.4-canary.0 <16.3.0-canary.5`| MODERATE | Depends on vulnerable `postcss`                              | Pin `next` to `9.3.3` (npm-suggested minor)   |
| `postcss`| `<8.5.10`                         | MODERATE | XSS via unescaped `</style>` in CSS Stringify Output         | Pin `postcss` to `^8.5.10`                    |

**Totals**: 0 critical, 0 high, 2 moderate, 1 low, 0 info.

**Exposure**: all three are **dev/build dependencies** consumed by the Next.js build pipeline. They are not present in the production runtime (Next.js static export emits HTML + JS bundles; the build is the only consumer of `postcss` and `diff`). Risk is low; fix during the next frontend dep-refresh.

### 6.2 govulncheck (BLOCKED locally)

Local host has no Go toolchain. Recommendation: Ops add to CI gate pass:

```yaml
- name: govulncheck
  run: |
    go install golang.org/x/vuln/cmd/govulncheck@latest
    govulncheck ./...
```

Out of scope for this review's deliverable but flagged for the next E-003 gate pass.

### 6.3 Repository secret hygiene

- No `.env` files tracked.
- No API keys, JWT secrets, or DB credentials in code or migration files.
- `migrations/027_tighten_agent_role_length.sql` is benign (DDL only).
- Webhook secret hashing (`bcrypt`) is correct; no plaintext secret paths.

---

## 7. Test coverage observations

- **Role matrix**: 7 explicit tests in `router/router_role_matrix_test.go` (admin-only `POST /v1/auth/register`, `DELETE /v1/agents/:id`, `POST /v1/agents/:id/retire`; write-any on `POST /v1/projects/:id/tasks`, `POST /v1/agents`, `PUT /v1/agents/:id`). **Good**.
- **API key middleware**: 5 subtests in `middleware_test.go:TestAPIKeyMiddleware` covering valid, revoked, expired, missing, malformed. **Good** — closes the F-002 fix surface.
- **Service-layer cross-tenant checks**: covered by `service/agent_test.go`, `service/assignment_test.go`, `service/deliverable_test.go`, `service/execution_test.go` (multiple cross-tenant cases).
- **Gaps** (F-D002-015..017): `handler/auth_test.go`, `handler/webhook_test.go`, `handler/project_test.go` are absent. Recommend adding in Sprint 7.

---

## 8. Sign-off

**Code-level review status: APPROVED with the following conditions:**

1. **B-002 (F-D002-001 webhook SSRF)**: must land before the v1 GA. The stub is a real SSRF vector (loopback, 169.254.169.254, RFC 1918). Lead can route the fix brief to Builder's B-002 wave.
2. **E-001 (F-D002-003 secure cookie)**: must add the `AUTH_COOKIE_SECURE` env var. Devs using HTTP should not be forced onto HTTPS, but prod must be locked down.
3. **Sprint 6+ (F-D002-004 IDOR)**: out of scope per Lead's brief, but logged here so it doesn't get lost. The true fix is the `project_memberships` table + `requireProjectMember` middleware.

The 14 new findings (F-D002-005..018) are non-blocking. None of them are HIGH or CRITICAL. Most are test-coverage gaps and dev-dep advisories. Recommend a Sprint 7 hardening pass to close them in batch.

**This review is filed as `docs/reset/security-review.md` on branch `guardian/d-002-security-review` at `8b85d26`.** Lead to push to main under E-003 or route as preferred.

— Guardian (slot `019ec4fe-604f-7551-9a76-38a621ddd256`), 2026-06-14
