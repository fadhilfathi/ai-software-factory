# B-002 Agent Communication — Full Dispatch (held until CI green)

**Owner:** Builder
**Support:** Leader (this brief) + Guardian (review at audit time)
**Status:** HELD until Pre-B-002 CI green-up lands
**Spec/design briefs:** `docs/reset/audit-prep-B-002.md` (commit b1226f4), `docs/reset/fix-f-d002-001-webhook-ssrf.md` (commit b1226f4), `docs/reset/pre-b002-ssrf-surface.md` (commit a0fe508)
**Audit checklist (for Guardian at review time):** `docs/reset/audit/B-002-audit-checklist.md` (commit c2c504e)

---

## 6-commit shape

### B-002 c1: `docs(api-spec): F-D002-001 SSRF fix shape + webhook dispatcher + messaging`

Mirror B-001 c1 (12-item spec drift, all closed). Scope:
- §Webhooks section: document the new validateWebhookURL behavior (scheme allowlist, port allowlist, length cap, DNS resolution + IP blocklist)
- §Webhooks section: document the dispatcher contract (HMAC, retry, DLQ, custom DialContext TOCTOU mitigation — flagged as stretch, not min bar)
- §Messages section: document the new endpoints (POST /v1/messages, GET /v1/messages/:id, POST /v1/messages/:id/reply, GET /v1/messages/.../history)
- §Webhook CRUD section: clarify admin-only role on POST/DELETE (per F-D002-005)
- The X-Project-ID surface (F-D002-004) — note "service-layer check is the v1 control; full fix is Sprint 6+ project_memberships"
- 8-12 spec drift items, all closed in c1..c6

### B-002 c2: `feat(webhooks): F-D002-001 SSRF fix — validateWebhookURL handwritten safety.go + IP blocklist + DialContext TOCTOU mitigation`

THE v1 GA GATE. Per the audit-prep-B-002 + fix-f-d002-001 + pre-b002-ssrf-surface briefs, scope:
- New file `src/internal/service/webhooks_safety.go` (~150 lines):
  - Scheme allowlist (https, http for testing)
  - Port allowlist (443/8443 for prod, 80/8080 for testing)
  - Length cap (2048 chars, configurable)
  - `LookupHost` that walks ALL resolved IPs and rejects if ANY is in a blocked range
  - IPv4 blocklist: 0.0.0.0/8, 10.0.0.0/8, 127.0.0.0/8, 169.254.0.0/16, 172.16.0.0/12, 192.168.0.0/16, 100.64.0.0/10, 224.0.0.0/4, 240.0.0.0/4
  - IPv6 blocklist: ::/128, ::1/128, fe80::/10, fc00::/7, ff00::/8
  - Error UX: return `service.Error` with codes (SSRF_BLOCKED, INVALID_SCHEME, INVALID_PORT, TOO_LONG, etc.)
- `src/internal/service/webhook.go:115` — replace stub `return nil` with the real validator (1 line change)
- Tests in `src/internal/service/webhooks_safety_test.go` (table-driven, ~30-40 cases):
  - 127.0.0.1 (loopback) → blocked
  - 10.0.0.1 (private) → blocked
  - 169.254.169.254 (AWS metadata) → blocked
  - [::1] (IPv6 loopback) → blocked
  - [fc00::1] (IPv6 ULA) → blocked
  - Legit public URL → passes
  - ftp:// scheme → blocked
  - 22 port → blocked
  - 9000-char URL → blocked
  - DNS-rebinding hostname that resolves to BOTH 1.2.3.4 and 10.0.0.1 → blocked (all-IPs check)
- **Stretch (not min bar)**: custom `DialContext` in dispatcher (the TOCTOU mitigation). Per Guardian's pre-B-002 audit, can be Sprint 6+.

### B-002 c3: `feat(dispatcher): HMAC + retry + dead-letter routing + custom DialContext TOCTOU mitigation`

Mirror B-001 c2+c3 bundling (B-001 had to bundle driveWorker + state-machine because they were coupled; B-002 c3 will do the same for the dispatcher pieces):
- `src/internal/dispatch/webhook_dispatcher.go` (new, ~250 lines):
  - HMAC signature: `X-Signature` header on every delivery, computed with bcrypt-hashed secret, constant-time compare
  - Retry policy: exponential backoff (1s, 2s, 4s, 8s, 16s) on 5xx + network errors; 4xx is terminal (no retry)
  - DLQ: terminal failure (max attempts exhausted) drops to in-memory dead-letter (Sprint 4+5; Sprint 6+ for persisted table)
  - Shutdown: `Stop(ctx)` waits for in-flight deliveries
  - **Stretch**: custom `DialContext` pinning to validated IP (the TOCTOU mitigation)
- Tests in `src/internal/dispatch/webhook_dispatcher_test.go`:
  - 200 response → success
  - 500 response → retry with backoff
  - 4xx response → terminal
  - Max attempts exhausted → DLQ
  - Custom DialContext pins to validated IP (if implemented)

### B-002 c4: `test(comm): table-driven F-D002-001 fix coverage`

Mirror B-001 c4 (table-driven state-machine coverage). Scope:
- `src/internal/service/webhooks_safety_test.go` (already in c2; expand if needed)
- New file `src/internal/handler/comm_envelope_test.go` (if not already covered by A-002-11): test that all 5 sites (agent CRUD, task create, deliverable create, plus the new webhook endpoints) return `{data: ...}` envelope
- New file `src/internal/handler/webhook_test.go` (closes F-D002-016 — was on Sprint 7 backlog, drops to closed)
- Cross-tenant 2 cases (Review-equivalent + Cancel-equivalent for messaging) returning 404

### B-002 c5: `feat(messaging): messaging endpoints + cross-tenant surface`

The 4 new endpoints + cross-tenant checks (mirrors B-001 c3 reviewer/cancel):
- `POST /v1/messages` — create + send
- `GET /v1/messages/:id` — read
- `POST /v1/messages/:id/reply` — reply
- `GET /v1/messages/.../history` — paginated history
- Each handler calls `service.Messaging.Send/Get/Reply/History` which checks `callerProjectID == req.ProjectID` (mirror `service/assignment.go:AssignTaskToAgent` F-014 pattern)
- Tests in `src/internal/handler/messaging_test.go` (mirror `execution_test.go` Review+Cancel test style):
  - Happy path for each endpoint
  - Cross-tenant 2 cases per endpoint (foreign project returns 404)
  - 6 reviewer-equivalent + 4 cancel-equivalent test cases

### B-002 c6: `docs(audit): B-002 audit + pre-push gate + Guardian sign-off`

Closes B-002. Mirror the A-001 / A-002 / A-003 / B-001 / D-003 audit format. File: `docs/reset/audit/B-002-audit-2026-06-14.md`

Sections (mirror B-001 c5 §1–§7):
1. **Evidence** — file-by-file inventory of c1..c5 with line ranges
2. **F-D002-001 SSRF verification** — the gate, walk validator + dispatcher; check all 10 tests pass
3. **Spec drift inventory** — 8-12 items, mirror A-001 pattern
4. **Cross-tenant (F-D002-004) surface** — every messaging + webhook CRUD endpoint, the partial-mitigation status
5. **Pre-push gate** — go test, gitleaks, validate-infra, gofmt, go vet, staticcheck (note: no local Go; CI is source of truth)
6. **Hand-backs** — coordination flags still in flight:
   - C-001 dashboard: webhook deliveries view
   - C-002 recovery: webhook delivery state machine
   - Sprint 6+ F-D002-004 project_memberships (full fix)
   - Sprint 6+ TOCTOU mitigation (custom DialContext — stretch, deferred)
7. **Sign-off** — PASS / PARTIAL-PASS / FAIL with conditions

---

## Hand-back expected at end of B-002

- F-D002-001 closed (v1 GA condition satisfied)
- F-D002-005 closed (admin role check on POST + DELETE webhook CRUD; already in F-D002-005 SHIP @ 1ba7cf7)
- F-D002-016 closed (handler/webhook_test.go created)
- F-D002-004 partial mitigation confirmed (full fix is Sprint 6+)
- v1 GA re-sign-off from D-002 review (clean APPROVED)

---

## Cross-agent dependencies

- Guardian: standing by for the B-002 audit review. Audit checklist is pre-staged at `docs/reset/audit/B-002-audit-checklist.md` (commit c2c504e). Will produce `docs/reset/audit/B-002-audit-2026-06-14.md` mirroring the A-001/B-001/D-003 pattern.
- Leader: dispatches in full when CI is green. Has already pre-staged all the briefs (audit-prep-B-002, fix-f-d002-001, pre-b002-ssrf-surface, B-002-audit-checklist). No new Lead work needed for B-002 c1..c6 itself.
- Ops: logged out, lane closed. No Ops work for B-002.

---

## E-003 path (Builder's standard)

Worktree `feat/B002-agent-communication` from current main:
- 6 commits in c1..c6 order
- Push branch, merge origin/main (v2 rule, no rebase), push branch, ff-merge to main, push main
- Cross-agent main drift is expected (Guardian/Ops may push during B-002); handle via --no-ff from parent worktree if needed
- Watch CI on every push
- Ping the moment of every push (reaffirmed protocol)

---

## When this dispatches

This brief is held until Pre-B-002 CI green-up lands. Once Builder pings the green-up SHA, Leader sends this dispatch as a single message.
