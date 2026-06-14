# B-002 Agent Communication — Pre-Scope Brief

**Owner of B-002:** Builder. Support: Leader (this brief) + Guardian (D-002 review of the F-D002-001 fix).
**Spec:** `docs/sprint4/agent-orchestration-design.md` §Communication + `docs/sprint5/agent-creation-management-design.md` (the F-002 + F-016 + TASK-421 line of work)
**Audit precedent:** `docs/reset/audit/A-001-audit.md`, `docs/reset/audit-prep-A-002.md`, `docs/reset/audit-prep-B-001.md`
**Critical dependency:** F-D002-001 (webhook SSRF) MUST be fixed before the webhook dispatcher ships. Brief at `docs/reset/fix-f-d002-001-webhook-ssrf.md`.

## Deliverables (per the team contract)
- Messaging (agent-to-agent)
- Task Dispatch
- Response Capture

## Existing code floor (READ FIRST)

| File | Purpose | Status |
|------|---------|--------|
| `src/internal/events/bus.go` | Memory-bus pub/sub for in-process events | Read. In-memory only. A-002-10 fixed `TestMemoryBus_RoundTrip`. |
| `src/internal/events/state.go` | Event-state machine (used by C-002 recovery) | Read. |
| `src/internal/dispatch/dispatcher.go` | Worker pool that pulls `WorkerSpec` from `DispatchQueue` and drives `aion.Runtime.Spawn/Wait`. Tenant-agnostic by design — service layer is the cross-tenant gate. | Solid. A-002-05 added `NackResult{Dropped, Retried}`. |
| `src/internal/dispatch/queue.go` | `DispatchQueue` interface + in-memory + Postgres impls. Nack returns `NackResult` for retry/DLQ tracking. | Solid. |
| `src/internal/handler/webhook.go` | Register/List/Delete webhooks. Handler accepts URL + event_types + secret, calls `WebhookService.RegisterWebhook`. | Read. URL validation is `validateWebhookURL` — THE STUB. |
| `src/internal/service/webhook.go` | `RegisterWebhook` validates URL (STUB — F-D002-001), hashes secret with bcrypt, validates event types against allowlist, stores via Postgres `webhooks` table. `ValidateWebhookSecret` does constant-time compare. | **The SSRF fix lives here.** |
| `src/internal/model/webhook.go` | `Webhook` row, event-type constants, HMAC signature, retry policy. | Read. |
| `src/internal/aion/process.go` | `ProcessRuntime` — real aion CLI child process spawn. | Read (also touched by A-002-01 hand-back). |
| Migrations 012, 013, 014, 021, 027 | webhook schema + retry policy + secret storage | Read to confirm. |

**Key observation:** The webhook DISPATCHER (the thing that fires HTTP POSTs to user-supplied URLs) is NOT in this codebase yet. `handler/webhook.go` is a CRUD surface only — registration, list, delete. There's no `dispatch/webhook_dispatcher.go` or equivalent that consumes `webhook.URL` to fire deliveries. This is the piece B-002 will own, and the reason F-D002-001 is a "time bomb" rather than an active exploit.

## What B-002 needs to do

### 1. Fix F-D002-001 (webhook SSRF stub) FIRST

**This is the must-do item.** Use the brief at `docs/reset/fix-f-d002-001-webhook-ssrf.md`:
- Create `src/internal/service/webhook_safety.go` (new, ~150 lines) with the handwritten validator
- Create `src/internal/service/webhook_safety_test.go` (new, table-driven, 1 test per range + happy path)
- Replace the stub in `src/internal/service/webhook.go:validateWebhookURL` with a call to the new validator
- Edit `src/internal/handler/webhook.go` to surface 400 with a clear message on validator failure
- Stretch (recommended): custom `DialContext` for the future dispatcher that pins to the validated IP (TOCTOU mitigation)

### 2. Build the webhook dispatcher (the missing piece)

Create `src/internal/dispatch/webhook_dispatcher.go`:
- `WebhookDispatcher` struct with a worker pool, the validated webhook list, an HTTP client (custom `DialContext` to pin to validated IPs)
- `Enqueue(event WebhookEvent)` — queue an event for delivery
- `Start(ctx, workers int)` + `Stop(ctx)` lifecycle (mirror `dispatch/dispatcher.go`)
- For each event: look up subscribed webhooks, fire HTTP POST with HMAC signature, retry on 5xx with exponential backoff (max attempts from `Webhook.RetryPolicy`), drop to DLQ on terminal failure
- **Use the validated IP for the HTTP request** (TOCTOU mitigation). The HTTP client must be configured to dial the IP directly, not re-resolve the hostname.

### 3. Wire the webhook events into the events bus

- New event types in `model/webhook.go`: `TaskCreated`, `TaskAssigned`, `TaskCompleted`, `ExecutionCompleted`, `DeliverableCreated`, etc.
- `events/bus.go`: add a publisher for these events
- `dispatch/webhook_dispatcher.go`: subscribe to the bus and Enqueue on relevant events

### 4. Add task dispatch and response capture (the messaging part)

- `src/internal/messaging/` (new package): agent-to-agent message envelope (`From`, `To`, `Body`, `ReplyTo`)
- `src/internal/messaging/memory_bus.go`: in-memory impl using the existing events bus
- `src/internal/handler/messaging.go`: POST `/v1/messages` (send), GET `/v1/messages/:id` (get), POST `/v1/messages/:id/reply` (reply)
- `src/internal/service/messaging.go`: dispatch logic with X-Project-ID cross-tenant check (F-D002-004 surface)
- Response capture: a `message_response` table with `(message_id, from, body, received_at)`; list endpoint for thread history

### 5. IDOR surface (F-D002-004)

Same X-Project-ID pattern. The webhook CRUD endpoints + the messaging endpoints all need the cross-tenant check. The new dispatcher is tenant-agnostic by design (mirror of the existing dispatch pattern) — the SERVICE is the gate. Log in the audit doc as D-002 OPEN (Sprint 6+ fix).

### 6. Tests

- `webhook_safety_test.go`: every IP range, scheme/port allowlist, length cap, error UX
- `webhook_dispatcher_test.go`: HMAC signature, retry on 5xx, DLQ on terminal failure, custom DialContext pins to validated IP (DNS rebinding defense)
- `messaging_test.go`: send + reply, thread history, cross-tenant blocked
- Integration: extend `internal/integration/integration_test.go` per D-003 — webhook fires on task-completed event; message round-trip works

## Audit doc shape (mirror A-001 / A-002)
`docs/reset/audit/B-002-audit.md`:
- Evidence: every existing file, the F-D002-001 fix, the new dispatcher
- Drift inventory: 12 items, each with `code` `spec` `fix`
- Pre-push gate: tests + build + Guardian sign-off + secret-scan
- Hand-backs: anything that crosses into C-001 (dashboard) or C-002 (recovery)

## Suggested PR shape
- Commit 1: `docs(api-spec): fix agent communication drift (N items)` (mirror A-001/A-002)
- Commit 2: `fix(webhook): implement SSRF protection (F-D002-001)` — the must-do item
- Commit 3: `feat(dispatch): webhook dispatcher with HMAC + custom DialContext` — the missing piece
- Commit 4: `feat(messaging): agent-to-agent message envelope + thread history`
- Commit 5: `test(webhook, messaging): coverage for SSRF, HMAC, retry, cross-tenant, message round-trip`
- Commit 6: `docs(audit): B-002 agent communication audit + pre-push gate`

If 6 commits is too many, fold 5 into 6. The must-do is commit 2 (F-D002-001). Commit 3 needs commit 2 to be safe.

## When this must land

After A-003 (Assignment Engine) ships, and ideally after D-002 (Security Review) has reviewed the F-D002-001 fix brief. The D-002 review checklist should be extended to include the new dispatcher + messaging endpoints.

## What I (Leader) will do

- Review the audit doc.
- Coordinate the F-D002-001 sign-off with Guardian (D-002 owns the security review; the SSRF fix is the highest-priority D-002 OPEN item that B-002 will close).
- Cross-ref the messaging IDOR surface in the D-002 report.

## What Guardian (D-002) will do

- Re-review the `webhook_safety.go` code (likely needs to walk the IP blocklist line-by-line to confirm no off-by-one).
- Re-review the `webhook_dispatcher.go` for TOCTOU mitigation (custom `DialContext`).
- Add the messaging endpoints to the D-002 review checklist (they share the F-D002-004 surface).

## What Ops (E-001) will do (downstream)

- Confirm the webhook dispatcher is exercised by the `docker compose up` smoke (the smoke should fire a test webhook to `webhook.site` or a local listener and confirm it lands).
- Add a `gitleaks` config to ignore test-only webhook URLs/secrets in fixtures.
