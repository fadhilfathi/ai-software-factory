# Pre-B-002 SSRF Surface Audit

| Field        | Value                                                                |
|--------------|----------------------------------------------------------------------|
| Owner        | Guardian (slot `019ec4fe-604f-7551-9a76-38a621ddd256`)               |
| Status       | **Informational — pre-B-002 prep**                                   |
| Reviewed     | commit `08014aa` (HEAD at audit start)                               |
| Date         | 2026-06-14                                                           |
| Purpose      | Map the SSRF surface for Builder's B-002 c2 (the F-D002-001 fix)     |
| Cross-ref    | `docs/reset/audit/B-002-audit-checklist.md` §2, `docs/reset/fix-f-d002-001-webhook-ssrf.md` |

This is a one-page mapping of every input path that flows into the
webhook URL validator, the future dispatcher, and the storage layer. It
is written BEFORE B-002 lands so Builder's c2 PR can satisfy §2.1 / §2.2
of the B-002 audit checklist on the first try.

---

## 1. The validator (THE STUB — `src/internal/service/webhook.go:115`)

```go
// validateWebhookURL validates the webhook URL to prevent SSRF
func validateWebhookURL(rawURL string) error {
    // In production, use a proper URL validator with allow-list
    // For now, basic validation: must be HTTPS, no private IPs
    // This is a simplified check - production should use a library like
    // github.com/nathan-osman/go-ssrf or similar
    return nil // TODO: implement full SSRF protection
}
```

**Truth**: the function returns `nil` for every input. No validation, no
logging, no observable side-effect. The TODO comment is honest; the
production-readiness claim in the comment is false.

**Single call site**: `src/internal/service/webhook.go:39`:

```go
if err := validateWebhookURL(req.URL); err != nil {
    errs.Add("url", err.Error())
}
```

Grep confirms there are no other call sites in the repo. The validator
is invoked at exactly one place: webhook registration.

---

## 2. The input paths into the validator

The validator takes a single string argument (`rawURL string`). That
string flows in from the chain below. Each link in the chain is
annotated with whether it pre-validates the URL.

### 2.1 Today (pre-B-002)

```
HTTP POST /v1/webhooks
   body: { "url": "...", "events": [...], "secret": "..." }
        │
        ▼
handler/webhook.go:35  Register(c)
   c.ShouldBindJSON(&req)        ← no pre-validation
        │ req.URL = <user string>
        ▼
handler/webhook.go:42  svc.RegisterWebhook(...)
        │  service.RegisterWebhookRequest{URL: req.URL, ...}
        ▼
service/webhook.go:32  RegisterWebhook(req)
   1. validation.NotEmpty(req.URL, ...)   ← only checks "non-empty"
   2. len(req.Events) == 0                ← events only
   3. validateWebhookURL(req.URL)         ← THE STUB (returns nil)
   4. event-type allowlist                ← events only
        │  model.Webhook{URL: req.URL, ...}
        ▼
store.Webhooks().Create(webhook)
   writes row with URL = req.URL
        │
        ▼
DB column `webhooks.url` (Postgres, TEXT)
   URL is now persisted, NEVER read again (no dispatcher yet)
```

**Per-link audit**:

| Link                                | Pre-validates URL? | Notes                                                                                         |
|-------------------------------------|--------------------|------------------------------------------------------------------------------------------------|
| `c.ShouldBindJSON(&req)`            | No                 | Gin's JSON binder. Type-checks `url: string` but does not look at the value.                    |
| `svc.RegisterWebhook(...)` call     | No                 | The handler just plumbs the field through. No string trim, no lowercasing, no length cap.       |
| `validation.NotEmpty(req.URL, ...)` | No                 | Only rejects empty string. Does NOT check length, scheme, or shape.                            |
| `len(req.Events) == 0` check        | n/a (events)       | Checks events, not URL.                                                                        |
| `validateWebhookURL(req.URL)`       | **NO (STUB)**      | Returns `nil`. This is the bug.                                                                |
| `event-type allowlist`              | n/a (events)       | Checks events, not URL.                                                                        |
| `store.Webhooks().Create(webhook)`  | No                 | Schema allows any string in the `url` TEXT column. No DB-level check constraint.                |
| DB column `webhooks.url`            | n/a                | Stored as-is. Reachable in the future by the dispatcher (not yet built).                        |

**Net result today**: the URL is **completely unchecked** end-to-end.
A user can register `http://169.254.169.254/latest/meta-data/...` and
the row is happily created.

### 2.2 Future (post-B-002 — what the dispatcher will add)

B-002's pre-scope §2 commits Builder to creating
`src/internal/dispatch/webhook_dispatcher.go`. The new link in the
chain is:

```
DB column `webhooks.url`
   WebhookDispatcher.Subscribe(events)
        │  on each event: WebhookDispatcher.Enqueue(event)
        ▼
   for each subscribed webhook w:
        parseURL(w.URL)                 ← re-parse the string from DB
        resolveIP(parseURL.Hostname())  ← re-resolve via DNS
        http.Post(...)                  ← HTTP client opens TCP to that IP
```

**The TOCTOU surface**: a user registers a URL whose hostname resolves
to a public IP at validation time (e.g., 1.2.3.4) and an internal IP
at delivery time (e.g., 10.0.0.1) — the classic DNS-rebinding attack.
**Mitigation**: custom `http.Transport{DialContext: ...}` that pins
to the validated IP and dials that IP directly. The dialer is
`net.Dial("tcp", validatedIP+":443")` — re-resolution is impossible
because the hostname is never passed to `net.Dial`.

This mitigation is **stretch** per the fix brief and audit checklist
§2.1 / §2.2. It is recommended for B-002 but not strictly required
for v1 GA. The fix brief's "minimum bar" + "stretch" sets the bar.

---

## 3. Other input paths (NOT SSRF but related)

The other two body fields in `POST /v1/webhooks` are not SSRF
surfaces, but they share the wire surface and the audit checklist
expects them to be checked. Quick disposition:

| Field     | Flows into ...                        | Pre-validated?         | Risk class |
|-----------|---------------------------------------|------------------------|------------|
| `url`     | `validateWebhookURL` + DB             | **No (STUB)**          | SSRF       |
| `events`  | event-type allowlist                  | Yes (allowlist, 9 events) | None currently; future drift possible (e.g., the dispatcher must validate `event` values in any inbound webhook, not just outbound) |
| `secret`  | bcrypt-hashed, never re-read          | Yes (bcrypt hash stored) | None        |
| `X-Project-ID` header | not consumed by the validator; flows into `callerProjectID` | n/a (F-D002-004, not SSRF) | IDOR |

The audit should NOT scope-creep into these — they are out of scope
for §2.1. Cross-ref F-D002-004 (Sprint 6+) for the X-Project-ID fix.

---

## 4. What the validator must handle (per the fix brief + §2.1)

The fix brief (`docs/reset/fix-f-d002-001-webhook-ssrf.md`) and the
B-002 audit checklist (`docs/reset/audit/B-002-audit-checklist.md` §2.1)
agree on the minimum bar. Consolidating the two for Builder's c2:

### 4.1 Input validation

| Check                       | Rule                                                            | Test cases required                                                                                  |
|-----------------------------|------------------------------------------------------------------|------------------------------------------------------------------------------------------------------|
| Scheme allowlist            | `https` only                                                     | `https://x` → pass; `http://x`, `ftp://x`, `file://x`, `gopher://x`, `dict://x`, `ldap://x` → reject  |
| Port allowlist              | 443 only (80 env-gated for dev via `WEBHOOK_ALLOW_INSECURE_SCHEME`) | `:443` → pass; `:80` → reject in prod; `:22, :3306, :6379, :9200, :11211, :27017` → reject          |
| Length cap (URL)            | ≤ 2048 bytes                                                     | 2049-char URL → reject                                                                                |
| Length cap (host)           | ≤ 253 chars                                                       | 254-char host → reject                                                                                |
| Hostname parse              | `net/url.Parse` must succeed                                      | Malformed URL → reject                                                                                |
| Userinfo stripping          | URLs with `user:pass@host` must be either rejected or stripped   | `https://x:y@1.1.1.1/` → reject or strip userinfo                                                      |
| Fragment/query              | Not load-bearing for SSRF; OK to allow but log                    | `https://1.1.1.1/?x=<script>` → pass (no SSRF concern, but log for audit trail)                       |

### 4.2 DNS resolution + IP blocklist

| Check                       | Rule                                                            | Test cases required                                                                                  |
|-----------------------------|------------------------------------------------------------------|------------------------------------------------------------------------------------------------------|
| `LookupIP` (ipv4 + ipv6)    | `net.DefaultResolver.LookupIP(ctx, "ip", host)` with 10s timeout | Resolution failure → reject; timeout → reject                                                          |
| All-IPs walk                | **Reject if ANY resolved IP is blocked**                          | Hostname that resolves to BOTH 1.2.3.4 AND 10.0.0.1 → reject (the all-IPs check is critical)             |
| IPv4 blocklist              | `0.0.0.0/8`, `10.0.0.0/8`, `127.0.0.0/8`, `169.254.0.0/16`, `172.16.0.0/12`, `192.168.0.0/16`, `100.64.0.0/10`, `224.0.0.0/4`, `240.0.0.0/4` | One test per range. `169.254.169.254` specifically (AWS metadata) is the must-pass test.              |
| IPv6 blocklist              | `::/128`, `::1/128`, `fe80::/10`, `fc00::/7`, `ff00::/8`         | One test per range. `[::1]`, `[fc00::1]` specifically.                                                 |
| Public-IP happy path        | `1.1.1.1` (Cloudflare), `8.8.8.8` (Google)                       | `https://1.1.1.1/webhook` → pass                                                                       |

### 4.3 Error UX

| Check                       | Rule                                                            | Notes                                                                                                  |
|-----------------------------|------------------------------------------------------------------|--------------------------------------------------------------------------------------------------------|
| Error code                  | `SSRF_BLOCKED` / `INVALID_SCHEME` / `INVALID_PORT` / `URL_TOO_LONG` | Mirror the existing `service.Error` pattern; the handler surfaces 400 with the code.                 |
| No internal range leak      | Don't say "169.254.x.x is reserved"; say "URL not reachable from server context" | Defense-in-depth: the operator should not learn our blocklist                                          |
| No secret leak              | If the URL contains a secret (e.g., `?token=xyz`), don't echo the secret in the error | Mirror the `service/webhook.go:115` "no secret in error messages" rule from the audit checklist §2.4 |

---

## 5. What the dispatcher must handle (per §2.2)

B-002 will create `src/internal/dispatch/webhook_dispatcher.go`. The
audit checklist §2.2 requires:

| Check                       | Rule                                                            | Test cases required                                                                                  |
|-----------------------------|------------------------------------------------------------------|------------------------------------------------------------------------------------------------------|
| Custom `DialContext`        | `http.Transport{DialContext: ...}` dials the **validated IP**, not the hostname | Serve on `127.0.0.1` and assert the dialer does not fall through to a public resolver. Mock the resolver. |
| HMAC signature              | `X-Signature` header on every delivery, constant-time compare on the receiver side | Compute the HMAC manually with the webhook's secret, compare to the dispatcher's `X-Signature` header |
| Retry on 5xx                | Exponential backoff (1s, 2s, 4s, 8s, 16s); 4xx is terminal      | Mock a 500 → assert retry; mock a 4xx → assert no retry                                                |
| Max attempts                | From `Webhook.RetryPolicy.MaxAttempts`                           | Mock max attempts exhausted → assert DLQ                                                               |
| DLQ on terminal failure     | In-memory for Sprint 4+5; persisted table Sprint 6+             | Mock max attempts + final 4xx → assert delivery is in the DLQ                                          |
| Graceful shutdown           | `Stop(ctx)` waits for in-flight deliveries (or ctx cancel)       | Call `Stop` mid-flight, assert the in-flight delivery completes or ctx expires                        |

The dispatcher is the B-002 deliverable; the audit must walk each
item above. The SSRF surface is the validator side of this
(`webhook_safety.go`); the dispatcher side is the TOCTOU mitigation
plus the delivery machinery.

---

## 6. Pre-existing surface map (file-by-file)

For the B-002 audit, here is the file-by-file inventory of the
current SSRF surface. Each file is read-only at this point (B-002
edits them).

| File | Lines | Status | SSRF surface? |
|------|-------|--------|----------------|
| `src/internal/handler/webhook.go` | 64 | read | yes — accepts the `url` field from the body, passes to service |
| `src/internal/service/webhook.go` | 121 | read | **THE STUB at line 115** — single call site at line 39 |
| `src/internal/model/webhook.go` | 77 | read | n/a — defines `Webhook` row shape; no validation |
| `src/internal/store/postgres/webhook.go` (or memory equiv) | (read by Builder) | read | n/a — schema column, no DB-level check |
| `src/internal/dispatch/webhook_dispatcher.go` | **NOT YET WRITTEN** | future | yes — the post-B-002 fetch surface |
| `src/internal/service/webhook_safety.go` | **NOT YET WRITTEN** | future | yes — the new validator |

The audit must confirm (a) the stub is replaced by a real validator,
(b) the new file `webhook_safety.go` is in the diff, (c) the new
`webhook_safety_test.go` covers every range, (d) the dispatcher
exists and uses a custom `DialContext`.

---

## 7. What the audit MUST walk (checklist for B-002 c2)

This is the v1 GA condition from the D-002 sign-off. The B-002 audit
cannot be PASS without these:

- [ ] **§2.1 validator** — `webhook_safety.go` exists, ~150 lines, handwritten. Scheme allowlist, port allowlist, length cap, hostname resolution, IP blocklist (IPv4 + IPv6), all-IPs walk, error UX, unit tests.
- [ ] **§2.1 tests** — `webhook_safety_test.go` covers every IP range, every blocked scheme, length caps, public-IP happy path. **At minimum** the test cases listed in §4.1 + §4.2 above.
- [ ] **§2.2 dispatcher** — `webhook_dispatcher.go` exists. Custom `DialContext`, HMAC signature, retry policy, DLQ routing, graceful shutdown.
- [ ] **§2.2 dispatcher tests** — `webhook_dispatcher_test.go` covers 200 success, 500 retry, 4xx terminal, max attempts → DLQ, custom `DialContext` pins to validated IP, HMAC signature is correct.
- [ ] **§2.3 handler** — `handler/webhook.go` surfaces 400 on validator failure. F-D002-005 (admin role) confirmed (closed at 1ba7cf7). F-D002-016 (handler test gap) closed — `handler/webhook_test.go` exists.
- [ ] **§2.4 service** — `service/webhook.go:validateWebhookURL` no longer returns `nil` unconditionally. No secret in error messages.

The D-002 sign-off is APPROVED-WITH-CONDITIONS until B-002 ships
with PASS on all of the above.

---

## 8. Hand-backs to file at audit time

If B-002 c2 misses any item in §7, the audit is PARTIAL-PASS at best
and FAIL at worst. Common hand-backs the audit may surface:

- **Validator misses an IP range** (e.g., forgot `100.64.0.0/10`
  CGNAT). The test that catches this is a unit test against
  `100.64.0.1`. The fix is a one-line addition to the blocklist.
- **Validator only walks the first IP** (the all-IPs check is
  missed). The test that catches this is a hostname that resolves
  to BOTH a public and a private IP. The fix is to walk all
  resolved IPs and reject if ANY is blocked.
- **Dispatcher doesn't pin to validated IP** (TOCTOU mitigation
  skipped). The test that catches this is a `127.0.0.1` server
  that the dispatcher should NOT be able to reach if the
  registration validator blocked loopback. The fix is a custom
  `DialContext`.
- **Error message leaks the blocklist** (e.g., "169.254.x.x is
  reserved"). The fix is to use a generic "URL not reachable from
  server context" message.
- **Test uses real DNS** (slow, flaky, can't run in CI sandbox).
  The fix is to inject a `Resolver` interface and use a stub in
  tests.

These are the likely PARTIAL-PASS items. None of them should block
B-002 from landing if the audit finds them on first pass — they are
small, follow-up-PR-able fixes.

---

## 9. Sign-off

This is a pre-B-002 informational audit. It does not produce code
changes; it produces the surface map that B-002 c2 will be reviewed
against.

**Guardian's recommendation**:

- B-002 c2 should follow the suggested PR shape in
  `docs/reset/audit-prep-B-002.md` §Suggested PR shape (6 commits).
- The F-D002-001 fix is the must-do (commit 2 of the shape).
- The audit will fail if §2.1, §2.2, §2.3, §2.4 of the B-002 audit
  checklist are not all green.
- Stretch (TOCTOU mitigation, HTTPS enforcement on the actual
  request) is recommended but not required for v1 GA.

**Pre-existing constraint**: there is no `go` toolchain on this
audit host, so the new `webhook_safety.go` cannot be unit-tested
locally. CI is the source of truth. Builder should ensure the new
test file is self-contained (no real DNS lookups) so CI runs
deterministically.

— Guardian (slot `019ec4fe-604f-7551-9a76-38a621ddd256`), 2026-06-14
