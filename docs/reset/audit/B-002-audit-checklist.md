# B-002 Audit Checklist for Guardian (D-002 follow-on)

**When to use:** after Builder ships B-002 and pings Lead with the SHIP SHA. This checklist walks the audit doc Guardian will write. Pre-staged so the review is fast and consistent with A-001 / A-002 / A-003 / B-001 audit pattern.

**Source materials:**
- B-002 pre-scope: `docs/reset/audit-prep-B-002.md` (Builder's working brief)
- F-D002-001 fix brief: `docs/reset/fix-f-d002-001-webhook-ssrf.md`
- D-002 review: `docs/reset/security-review.md` §5.1, §5.2, §5.4
- Audit precedent: `docs/reset/audit/A-001-audit.md`, `docs/reset/audit/A-002-audit.md`, `docs/reset/audit/A-003-audit.md`, `docs/reset/audit/B-001-audit.md`

---

## Audit doc shell

```markdown
# B-002 Agent Communication — Audit

**Date:** YYYY-MM-DD
**Reviewer:** Guardian
**Build SHA:** <commit>
**Spec version:** <commit or tag>
**Pre-scope:** docs/reset/audit-prep-B-002.md
**F-D002-001 fix brief:** docs/reset/fix-f-d002-001-webhook-ssrf.md
**Cross-ref:** docs/reset/security-review.md §5.1, §5.2, §5.4

## 1. Evidence — what was shipped
[file-by-file inventory with line ranges, mirroring A-001 §1]

## 2. F-D002-001 SSRF fix verification
[walk the validator + dispatcher; see §2 below]

## 3. Spec drift inventory
[code vs. spec, N items, each with `code` `spec` `fix`]

## 4. Cross-tenant (F-D002-004) surface
[messaging + webhook CRUD endpoints, mirror B-001 §3]

## 5. Pre-push gate
[tests green, gitleaks clean, validate-infra green, go vet clean, no staticcheck SA4009/SA4010]

## 6. Hand-backs
[items that cross into C-001 (dashboard) or C-002 (recovery)]

## 7. Sign-off
PASS / PARTIAL-PASS / FAIL with conditions
```

---

## §2 — F-D002-001 SSRF fix verification (the gate)

This is the v1 GA condition from D-002 sign-off. The audit MUST walk the validator and the dispatcher, not just rubber-stamp "the test passes".

### 2.1 `src/internal/service/webhook_safety.go` (new, ~150 lines expected)

Walk every code path:

- [ ] **Scheme allowlist** — only `https` (or `http` for testing?). Anything else (`ftp`, `file`, `gopher`, `dict`, `ldap`) → 400.
- [ ] **Port allowlist** — restrict to 443/8443 (and 80/8080 for testing?). Or block well-known dangerous ports (22, 23, 25, 53, 3306, 5432, 6379, 9200, 11211, 27017). Decide which and document.
- [ ] **Length cap** — reject URLs > 2048 chars (standard browser cap). Configurable?
- [ ] **DNS resolution** — `LookupHost` on the hostname. If it returns multiple IPs, **walk all of them** and reject if ANY resolves to a blocked range. (Naive impl rejects only the first; this is a real bypass.)
- [ ] **IPv4 blocklist** — at minimum:
  - `0.0.0.0/8` (current network)
  - `10.0.0.0/8` (private)
  - `127.0.0.0/8` (loopback)
  - `169.254.0.0/16` (link-local — includes AWS metadata at 169.254.169.254)
  - `172.16.0.0/12` (private)
  - `192.168.0.0/16` (private)
  - `100.64.0.0/10` (CGNAT)
  - `224.0.0.0/4` (multicast)
  - `240.0.0.0/4` (reserved/broadcast)
- [ ] **IPv6 blocklist** — at minimum:
  - `::/128` (unspecified)
  - `::1/128` (loopback)
  - `fe80::/10` (link-local — IPv6 equivalent of 169.254.x.x)
  - `fc00::/7` (ULA — IPv6 equivalent of 10.x.x.x)
  - `ff00::/8` (multicast)
- [ ] **Error UX** — return a `validate.Error` with a clear code (`SSRF_BLOCKED`, `INVALID_SCHEME`, etc.) and a message that the operator can act on. Mirror the existing `service.Error` pattern.
- [ ] **Tests** — `webhook_safety_test.go` must cover at minimum:
  - 127.0.0.1 (loopback) → blocked
  - 10.0.0.1 (private) → blocked
  - 169.254.169.254 (AWS metadata) → blocked
  - [::1] (IPv6 loopback) → blocked
  - [fc00::1] (IPv6 ULA) → blocked
  - Legit public URL (e.g. `https://api.example.com/webhook`) → passes
  - ftp:// scheme → blocked
  - 22 port → blocked
  - 9000-char URL → blocked
  - DNS-rebinding hostname that resolves to BOTH 1.2.3.4 and 10.0.0.1 → blocked (the all-IPs check)

### 2.2 `src/internal/dispatch/webhook_dispatcher.go` (new)

- [ ] **Custom `DialContext`** — passes the *validated* IP, not the hostname. The HTTP client calls `http.Transport{DialContext: ...}` and the dialer does `net.Dial("tcp", validatedIP+":443")`. This is the TOCTOU mitigation; without it, a DNS-rebinding attacker could resolve to a safe IP at validation time and an unsafe IP at request time.
- [ ] **HMAC signature** — `X-Signature` header on every delivery, computed with the webhook's bcrypt-hashed secret. Constant-time compare on the receiver side. The `service/webhook.go:ValidateWebhookSecret` is the existing primitive; verify the dispatcher uses it.
- [ ] **Retry policy** — exponential backoff on 5xx and network errors; 4xx is terminal (no retry, the receiver rejected). Max attempts from `Webhook.RetryPolicy`. Backoff base from config (default 1s, 2s, 4s, 8s, 16s).
- [ ] **DLQ routing** — terminal failure (max attempts exhausted) drops to dead-letter. The DLQ is in-memory for Sprint 4+5 (per D-002 §5.5), Sprint 6+ for the persisted table.
- [ ] **Shutdown** — `Stop(ctx)` waits for in-flight deliveries to finish (or context cancel, whichever first). Mirror the pattern in `dispatch/dispatcher.go:Stop`.
- [ ] **Tests** — `webhook_dispatcher_test.go`:
  - 200 response → success, no retry
  - 500 response → retry with backoff
  - 4xx response → terminal, no retry
  - Max attempts exhausted → DLQ
  - Custom DialContext pins to the IP that was validated (verify by serving on 127.0.0.1 and asserting the dialer doesn't fall through to a public resolver)
  - HMAC signature is correct (compute manually, compare to header)

### 2.3 `src/internal/handler/webhook.go` (existing, edited)

- [ ] **400 on validator failure** — the handler now calls the real validator; failure must surface as 400 with the validator's error code.
- [ ] **F-D002-005** — POST /v1/webhooks is admin-only. Verify the route has the admin role check. If not, this is a hand-back.
- [ ] **F-D002-016** — `handler/webhook_test.go` was absent pre-D-002; verify the new file exists and covers CRUD.

### 2.4 `src/internal/service/webhook.go` (existing, edited)

- [ ] **Stub replaced** — `validateWebhookURL` no longer returns `nil` unconditionally. It calls the new validator.
- [ ] **No secret in error messages** — if the validator fails on a URL that contains a secret, the error must not echo the secret.

---

## §3 — Spec drift inventory template

Mirror A-001 / A-002 / A-003 / B-001 audit pattern. Expect 8–12 items:

| # | Spec § | Code location | Code says | Spec says | Fix |
|---|--------|---------------|-----------|-----------|-----|
| 1 | §4.2 Webhook Registration | `handler/webhook.go:XX` | accepts URL only | spec says `events` array is required | tighten validation |
| 2 | ... | ... | ... | ... | ... |

Common drift spots to look for:
- `Webhook.RetryPolicy.MaxAttempts` default (code may be 3, spec may say 5)
- HMAC signature header name (X-Signature? X-Webhook-Signature? X-Hub-Signature-256?)
- Event-type enum (TaskCreated vs task.created vs task_created)
- `GET /v1/messages` pagination (spec may say cursor-based, code may be offset-based)
- Response envelope (`{data: ...}` per A-002-11 fix — verify the new endpoints comply)

---

## §4 — Cross-tenant (F-D002-004) surface

The webhook CRUD + messaging endpoints share the X-Project-ID surface. Walk every new handler:

- [ ] `POST /v1/messages` — calls `service.Messaging.Send` which must check `callerProjectID == req.ProjectID` (mirror of `service/assignment.go:AssignTaskToAgent` pattern)
- [ ] `GET /v1/messages/:id` — same check
- [ ] `POST /v1/messages/:id/reply` — same check
- [ ] `GET /v1/messages/.../history` — same check
- [ ] `POST /v1/webhooks` (admin) — verify the role check is correct, not just the project check
- [ ] `GET /v1/webhooks` — caller sees only their project's webhooks
- [ ] `DELETE /v1/webhooks/:id` — same
- [ ] `messaging_test.go` — has a cross-tenant negative test for each endpoint (mirror B-001 `TestAssignmentService_AssignTaskToAgent_CrossTenant` pattern)

Document any unchecked surface as a hand-back. The Sprint 6+ fix (project_memberships) is out of scope for B-002, but the partial-mitigation-by-theater finding should be re-confirmed.

---

## §5 — Pre-push gate (Lead verifies before audit)

The audit doc should walk each of these:

- [ ] `go test ./...` green with `-race -shuffle=on` (E-003)
- [ ] `gitleaks detect --no-banner` clean (E-002)
- [ ] `python scripts/validate-infra.py` exit 0 (E-001)
- [ ] `gofmt -l` empty (no gofmt violations)
- [ ] `go vet ./...` clean
- [ ] `staticcheck -checks=SA4009,SA4010 ./...` (if installed; otherwise note "skipped, see Ops follow-up")
- [ ] No new dependencies in `go.mod` without a justification line in the commit body

---

## §6 — Hand-back template

Items that cross into C-track or Sprint 6+:

- [ ] **C-001 (Monitoring Dashboard)** — the webhook dispatcher needs a "deliveries" view (per-event, per-webhook, status, retry count). Document the data shape the dashboard will need.
- [ ] **C-002 (Recovery System)** — if the dispatcher is part of the agent's runtime, a crashed agent should not leave deliveries half-fired. Cross-ref `audit-prep-C-002.md` §Cross-agent handoffs.
- [ ] **D-002 sign-off** — F-D002-001 is closed at B-002. F-D002-004 is still open (Sprint 6+). F-D002-005 is closed if admin role check is in place.
- [ ] **F-D002-016** — `handler/webhook_test.go` was absent; verify it now exists and is in the test count.
- [ ] **Sprint 7 non-blocking** — F-D002-015 (auth_test.go), F-D002-017 (project.go) — out of scope, log but don't block.

---

## §7 — Sign-off rubric

| Outcome | Conditions |
|---------|------------|
| **PASS** | All §2 / §3 / §4 / §5 checks green. F-D002-001 closed. No new hand-backs. |
| **PARTIAL-PASS** | All checks green except ≤2 hand-backs that can land in a follow-up PR before v1 GA. |
| **FAIL** | F-D002-001 not closed OR cross-tenant surface has an unchecked route OR pre-push gate is red. |

The D-002 sign-off is **APPROVED-WITH-CONDITIONS** until B-002 ships with PASS. B-002 with FAIL blocks v1 GA.

---

## Quick-reference: known D-002 findings that B-002 affects

- **F-D002-001** (HIGH webhook SSRF) — closes at B-002 (commit 2 per the pre-scope)
- **F-D002-004** (HIGH X-Project-ID IDOR) — partial-mitigated by service-layer check, Sprint 6+ for full fix
- **F-D002-005** (admin-only POST /v1/webhooks) — closes at B-002 (verify in §4)
- **F-D002-016** (handler/webhook_test.go absent) — closes at B-002 (commit 5 per the pre-scope)
- **F-D002-018** (delivery audit log) — Sprint 7 non-blocking

If B-002 closes F-D002-001 / 005 / 016, the D-002 sign-off is a clean v1 GA. The other 12 LOW/INFO findings remain on the Sprint 7 hardening list.
