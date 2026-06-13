# Sprint 5 — Infrastructure Fixes

This file documents infrastructure fixes applied during Sprint 5. Each
entry records the problem, the fix, the alternatives considered, and
the verification result.

---

## TASK-429 — Fix Deploy workflow `buildx cache-to` (2026-06-13)

### Problem

The post-merge `Deploy` workflow on `ebeba6b` (Sprint 4 closeout)
failed at the `build-push-api` step. Run reference:
[run #27459209340](https://github.com/fadhilfathi/ai-software-factory/actions/runs/27459209340).

The runner reported:

```
ERROR: failed to build: Cache export is not supported for the docker driver.
Learn more at https://docs.docker.com/go/build-cache-backends/
```

### Root cause

`docker/build-push-action@v6` invokes `docker buildx build`, which by
default uses the `docker` driver (the local Docker daemon). The `docker`
driver does not support cache *export* — it supports cache *import*
(`cache-from: type=gha` works) but not *export* (`cache-to: type=gha`).

The `cache-to: type=gha,mode=max` line was being rejected because the
underlying buildx driver could not honor a cache export to the GHA
backend.

To support full cache export, the workflow must switch buildx to the
`docker-container` driver (a BuildKit container), which DOES support
cache export. That requires a `docker/setup-buildx-action@v3` step
configured with `driver: docker-container` before the build steps.

### Fix (this commit)

Option (a) — **drop the `cache-to` lines entirely** from both
`Build & push API` and `Build & push Frontend`. Cache reads
(`cache-from: type=gha`) still work, so the first build after a
clean cache is still fast — the workflow pulls cached layers from the
GHA backend. Cache *writes* are dropped, so subsequent builds within
the same workflow run will rebuild some layers that would otherwise
have been re-cached, but the cross-run speedup (which is the bigger
win) is preserved.

A brief inline comment in `deploy.yml` records the rationale and
points to the proper follow-up fix.

Also added a `workflow_dispatch` trigger so the Deploy workflow can
be re-run from the GitHub Actions UI without needing a new push.

### Alternatives considered

- **Option (b) — pin `cache-version`**: does not address the root
  cause. The `cache-version` parameter controls the cache backend
  format version, not the buildx driver. Pinning the version would
  not change the driver's cache-export capability and the same error
  would recur.
- **Option (c) — switch to `type=local`**: works in principle (the
  `docker` driver supports local cache export) but the cache is
  written to a path on the runner filesystem, which is ephemeral
  on GitHub-hosted runners. Cache would not persist across runs,
  so the speedup would not accumulate. Rejected.

### Proper follow-up (filed)

The proper fix is to add a `docker/setup-buildx-action@v3` step with
`driver: docker-container` (or `kubernetes`/`remote`) before the
build steps, so that full cache export is supported. The `cache-to:
type=gha,mode=max` line can then be re-added. This is a small change
but it requires a test run on a real runner to confirm the
`docker-container` driver works in the GHA-hosted environment
(runner has a `docker-container` driver by default in v3, but the
workflow should be explicit about it for reproducibility).

Filed as a Sprint 5+ follow-up. Not blocking TASK-429.

### Verification

- **Pre-fix**: Run #27459209340 on `ebeba6b` — Deploy workflow red
  at `build-push-api` with the `Cache export is not supported` error.
- **Post-fix run #1**: Run #27461419661 on `fc4db30` (squash-merge of
  PR #2, triggered automatically by the push to `main`)
  - ✓ **Build & Push Images** job — **SUCCESS** in 1m36s
    - `Build & push API` (step 4) — 41s, success
    - `Build & push Frontend` (step 5) — 48s, success
  - ❌ **Deploy Stack** job — FAILED in 1m15s
    - All buildx / image-push steps clean
    - Step 6 (`Deploy`) failed because the `api` container panic'd on
      startup: `panic: required environment variable JWT_SECRET is not set`
      (`internal/config/config.go:99`).
    - `api-1` was restart-looping; `db-1` and `redis-1` were healthy.

### Separate finding (NOT a TASK-429 regression)

The Deploy failure above is **not caused by the buildx fix** — the
`Build & Push Images` job, which is the only thing this commit changed,
succeeded cleanly. The failure is a pre-existing bug in the
`docker-compose.yml` `api` service: its `environment:` block
(lines 64-73) does not set `JWT_SECRET` (or `JWT_ACCESS_SECRET` /
`JWT_REFRESH_SECRET` if the API has split them). The Go API's
`config.Load()` panics on `getEnvRequired("JWT_SECRET")` because the
env var is unset.

This failure would have happened on any Deploy run, including the
Sprint 4 closeout run #27459209340, but the `buildx cache-to` error
struck first and masked it. The same pre-fix run only reached
`build-push-api` and never attempted the Deploy step.

This is a **separate Sprint 5 follow-up task** (not part of TASK-429,
whose scope was the buildx `cache-to` problem only):

1. Determine which JWT secret(s) the API requires (check
   `internal/config/config.go` for all `getEnvRequired` / `os.Getenv`
   calls).
2. Add the missing `JWT_SECRET` (and any other required vars) to the
   `api` service's `environment:` block in `docker-compose.yml`.
3. Decide on a secret-management strategy:
   - **Local dev**: pull from a `.env` file via Compose's `env_file:`
     directive.
   - **GHA secrets**: the `Deploy` workflow does not currently inject
     GitHub Actions secrets into the container environment. May need
     to add an `env:` block to the `Deploy` step that sources
     `${{ secrets.JWT_SECRET }}` etc.
4. Verify with a re-run of the Deploy workflow on `main`.

### Files changed

- `.github/workflows/deploy.yml`
  - Added `workflow_dispatch` trigger
  - Dropped `cache-to: type=gha,mode=max` from both build steps
  - Added inline comments explaining the rationale
- `docs/sprint5/infra-fixes.md` (this file, new)

### Reference

- Failed run: https://github.com/fadhilfathi/ai-software-factory/actions/runs/27459209340
- Closeout commit: `ebeba6b32a0c03ca5ab2095264eb46dd40264268`
- `docker buildx` cache backends: https://docs.docker.com/go/build-cache-backends/
- `docker/setup-buildx-action`: https://github.com/docker/setup-buildx-action

## Sprint 5 validation (TASK-512)

Added to the api service in `docker-compose.yml` (carryover from TASK-430 + Sprint 5 additions):

```yaml
# TASK-430 (carryover): JWT secret signs and verifies JWTs.
# PATH C AMENDMENT: a dev-only default is provided for non-prod use
# (CI, local dev, integration tests). In production the operator MUST
# override JWT_SECRET with a 32+ char random value.
JWT_SECRET:       ${JWT_SECRET:-dev_only_secret_32_chars_minimum_for_local_testing}
# Sprint 5: Aion Agent Runtime (TASK-501).
AION_BINARY:           ${AION_BINARY:-/usr/local/bin/aion}
AION_MODEL:            ${AION_MODEL:-MiniMax-M3}
AION_PROVIDER:         ${AION_PROVIDER:-aionrs}
AION_PERMISSION_MODE:  ${AION_PERMISSION_MODE:-YOLO}
AION_MAX_CONCURRENT:   ${AION_MAX_CONCURRENT:-4}
AION_WAIT_TIMEOUT:     ${AION_WAIT_TIMEOUT:-300}
AION_E2E:              ${AION_E2E:-0}
# Sprint 5: Agent runtime mode (TASK-512 / security-01 C1).
AGENT_RUNTIME:        ${AGENT_RUNTIME:-aion}
AGENT_WORKER_SANDBOX: ${AGENT_WORKER_SANDBOX:-in-process}
AGENT_WORKER_RESTART: ${AGENT_WORKER_RESTART:-kill-in-flight}
```

### Validation matrix (target state for Sprint 5 closeout)

| Step | Command | Expected | Sprint 5 status |
|---|---|---|---|
| 1. Compose config | `docker compose config` | Exit 0, no warnings | TBD (CI gate step 9) |
| 2. Stack up | `docker compose up -d` | All services healthy | TBD (CI gate step 10) |
| 3. API healthz | `curl -fsS http://localhost:8080/v1/healthz` | 200 OK | TBD (CI gate step 12) |
| 4. Aion worker spawn | `docker compose exec api aion spawn --model MiniMax-M3 --provider aionrs` | Exit 0, agent PID tracked | DEFERRED to TASK-507 (Aion CLI spawn) |

### Worker blast radius (security-01 C1)

`AGENT_WORKER_SANDBOX=in-process` is **ENFORCED** for Sprint 5. The Aion worker runs inside the API process and is therefore fully trusted with the host capabilities of the api container. The threat model (security-01 §0.5 C1) is documented in `docs/sprint5/security-review.md`. Sprint 6 follow-up: move the worker to a sidecar with restricted syscalls (seccomp profile + drop CAP_SYS_ADMIN).

### Sprint 4 carryovers retired

The following Sprint 4 sandbox tuning knobs (runc-based) were removed in the Sprint 5 rewrite:

- `AGENT_MEMORY_MB`
- `AGENT_CPU_LIMIT`
- `AGENT_RUNTIME=runc` (replaced with `=aion`)

These do not apply to the in-process Aion worker model.

### Files changed in TASK-512

- `docker-compose.yml`
  - Added `JWT_SECRET` (carryover from TASK-430; **PATH C AMENDMENT** uses `${VAR:-dev_default}` so CI / local dev / non-prod compose can boot without a secret. In production the operator MUST override with a 32+ char random value — see .env.example)
  - Added 7 `AION_*` primary vars (TASK-501 compat)
  - Added 3 `AGENT_*` runtime mode vars (Lead brief / security-01 C1)
- `.env.example`
  - Replaced `# --- Agent Sandbox ---` section with `# --- Agent Runtime (Sprint 5) ---`
  - Removed stale `AGENT_MEMORY_MB` / `AGENT_CPU_LIMIT` / `AGENT_RUNTIME=runc`
  - Added the same env-var set with the same defaults and an explanation per var
- `docs/sprint5/infra-fixes.md` (this file)
  - Added the "Sprint 5 validation" section (above)

### NO COMMIT (TASK-512 work-in-progress)

Per §3 rule 7 of the sprint brief, the TASK-512 changes are left uncommitted on the `devops01/sprint5-task-512` branch. DevOps-01 (closeout owner) will fold them into the `feat(sprint-5)` closeout commit once:

1. The Sprint Quality Gate on `ci.yml` and `sprint-quality-gate.yml` is green (Lead's Rule 5).
2. Security-01 has signed off on the threat model update.
3. Tester-01 has run the integration test pack against the new env-var set.
