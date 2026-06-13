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
- **Post-fix**: TBD. Will be triggered via `workflow_dispatch` on the
  fix branch after the PR is opened. Result recorded below.

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
