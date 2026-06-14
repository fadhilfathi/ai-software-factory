# F-D002-001 — Webhook SSRF Fix Brief

**Finding ID:** F-D002-001
**Severity:** HIGH (pre-B-002)
**Owner of fix:** Builder (in B-002 wave — Agent Communication)
**Support:** Leader (this brief)
**Discovered:** 2026-06-14, during D-002 prep
**Detail:** see `docs/reset/d002-security-checklist.md` §F-D002-001

## What the bug is

`src/internal/service/webhook.go:115` — `validateWebhookURL` returns `nil`
for every input. Comment is honest: "TODO: implement full SSRF protection".

Exploit path (after B-002 ships the webhook dispatcher): any authenticated
user with developer+ role can `POST /webhooks { "url": "http://169.254.169.254/latest/meta-data/iam/security-credentials/..." }` and have the server fetch AWS instance metadata, internal services on `localhost` or `127.0.0.1`, link-local addresses, etc. Cloud credential exfil + lateral movement.

## What the fix needs to do

At registration time, the URL must be proven safe to call from a server
context. At dispatch time, the URL must be re-validated (or the validated
IP must be used for the request — TOCTOU mitigation).

### Minimum bar (for B-002 to land)

1. **Scheme allowlist**: `https` only. Reject `http`, `ftp`, `file`, `gopher`,
   `dict`, `ldap`, and any other scheme.
2. **Port allowlist**: 443 only. Reject everything else. (If 80 is needed
   for dev, gate it behind an env flag `WEBHOOK_ALLOW_INSECURE_SCHEME=true`.)
3. **Hostname resolution**: resolve the URL hostname via `net.DefaultResolver.LookupIP(ctx, "ip", host)`. Reject if resolution fails (no
   IP addresses) or if resolution times out (10s deadline).
4. **IP blocklist** (reject if any resolved IP is in any of these ranges):
   - `127.0.0.0/8` (loopback)
   - `10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16` (RFC 1918 private)
   - `169.254.0.0/16` (link-local, **AWS / GCP / Azure metadata**)
   - `100.64.0.0/10` (CGNAT)
   - `224.0.0.0/4` (multicast)
   - `240.0.0.0/4` (reserved / future use)
   - `0.0.0.0/8` ("this network")
   - IPv6: `::1/128`, `fc00::/7`, `fe80::/10`, `::/128`
5. **Length cap**: URL max 2048 bytes; host max 253 chars.
6. **Error UX**: 400 with a clear message (don't leak internal ranges; say
   "URL not reachable from server context").
7. **Unit tests**: every range above, plus the public-IP happy path
   (use `1.1.1.1`, `8.8.8.8`). Use a stub DNS resolver if you don't want
   the test to hit the network.
8. **Integration test** (CI only, not local): real lookup of a known
   public IP works; real lookup of a private IP is rejected.

### Stretch (full bar, recommended for B-002 to be truly safe)

9. **TOCTOU mitigation**: at dispatch time, re-resolve the URL hostname
   and confirm the IP is still in the public range. Use a custom
   `http.Transport` with a custom `DialContext` that pins to the validated
   IP (not the hostname) to defeat DNS rebinding.
10. **HTTPS enforcement on the actual request** (not just registration):
    the dispatcher's HTTP client should refuse to follow redirects to
    non-https URLs.

## Library options

### Option A: `github.com/nathan-osman/go-ssrf` (the comment's recommendation)
- Lightweight, single-purpose, well-tested.
- API: `SafeResolve(ctx, url, allowedSchemes, allowedPorts) (*ResolvedURL, error)`.
- It does NOT do TOCTOU mitigation — you still need a custom Dialer.
- License: MIT. Last commit: recent.

### Option B: `github.com/oxff00/gosafe` (alternative)
- Similar surface, less maintained.

### Option C: handwritten
- ~150 lines for the minimum bar. Easy to audit. No third-party dep.
- Recommended for this repo because:
  - The validation rules are project-specific (no localhost, no AWS metadata)
  - Handwritten code is easier to review in PR
  - The team is small and one off-by-one in an IP range would be bad — better to
    have the code right in front of reviewers
  - No new dependency to vet

**Recommended: Option C (handwritten) for the minimum bar, in a new file
`src/internal/service/webhook_safety.go`. Add a small unit test file
`webhook_safety_test.go` covering each range.**

## When this must land

**Before B-002 ships the webhook dispatcher.** Concretely:
- B-001 (Execution Engine) is the immediate next step for Builder.
- B-002 (Agent Communication) will wire the dispatcher to consume
  `webhook.URL`. B-002 PR is the trigger.

## Suggested PR shape (for Builder)

- File: `src/internal/service/webhook_safety.go` (new, ~150 lines)
- File: `src/internal/service/webhook_safety_test.go` (new, table-driven, 1 test per range + happy path)
- Edit: `src/internal/service/webhook.go` — replace the stub
  `validateWebhookURL` with a call to the new validator.
- Edit: `src/internal/handler/webhook.go` — surface 400 with a clear
  message.
- Optional (stretch): `src/internal/dispatch/webhook_dispatcher.go` —
  custom `DialContext` pinning to validated IP.

Commit message: `fix(webhook): implement SSRF protection for webhook URL validation (F-D002-001)`

## Review checklist for Guardian (D-002)

- [ ] Scheme allowlist enforced
- [ ] Port allowlist enforced (or env-gated for dev)
- [ ] Every private range in the blocklist has a unit test
- [ ] Public-IP happy path test
- [ ] Length caps in place
- [ ] Error UX doesn't leak internal ranges
- [ ] (stretch) TOCTOU mitigation considered and either implemented or
      explicitly deferred with rationale
