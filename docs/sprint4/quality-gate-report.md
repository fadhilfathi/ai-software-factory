# Sprint 4 Quality Gate Report

**Sprint:** 4
**Owner:** DevOps-01 (workflow + script) / CI run (results)
**Workflow:** `.github/workflows/sprint-quality-gate.yml`
**Local runner:** `scripts/quality-gate.sh`
**Closeout commit:** *(filled in by TASK-415)*

> **Status:** ⏳ Pending — this report is populated by the CI workflow run attached
> to the TASK-415 sprint closeout commit. The structure below is the contract; the
> CI run (or DevOps at closeout time) fills in the values marked `<!-- fill -->`.

---

## CI run metadata

| Field | Value |
|---|---|
| Commit SHA | <!-- fill: long SHA --> |
| Branch | <!-- fill: typically `feat/sprint-4` or `main` --> |
| Workflow run | <!-- fill: https://github.com/<org>/<repo>/actions/runs/<id> --> |
| Trigger | <!-- fill: push to main / PR to main / manual dispatch --> |
| Started at | <!-- fill: ISO-8601 UTC --> |
| Finished at | <!-- fill: ISO-8601 UTC --> |
| Duration | <!-- fill: Xm Ys --> |
| Runner | <!-- fill: e.g. ubuntu-latest --> |
| Final result | <!-- fill: ✅ pass / ❌ fail --> |

---

## Step-by-step results

Each row maps 1:1 to a step in `.github/workflows/sprint-quality-gate.yml`.
A ✅ requires exit code 0; a ❌ must be investigated before the closeout commit
is fast-forwarded into `main`.

| # | Step | Exit | Duration | Notes |
|---|---|---|---|---|
| 1/14 | Checkout | <!-- fill --> | <!-- fill --> | |
| 2/14 | Set up Go 1.25 | <!-- fill --> | <!-- fill --> | setup-go@v5 |
| 3/14 | Set up Node 20 | <!-- fill --> | <!-- fill --> | setup-node@v4 |
| 4/14 | Cache Go modules + build cache | <!-- fill --> | <!-- fill --> | cache hit: yes/no (key on `src/**/go.sum`) |
| 5/14 | `go mod download` | <!-- fill --> | <!-- fill --> | |
| 6/14 | `go vet ./...` | <!-- fill --> | <!-- fill --> | |
| 7/14 | `go build ./...` | <!-- fill --> | <!-- fill --> | |
| 8/14 | `go test -count=1 -timeout 5m ./internal/...` | <!-- fill --> | <!-- fill --> | |
| 9/14 | `docker compose config` | <!-- fill --> | <!-- fill --> | |
| 10/14 | `docker compose up -d` | <!-- fill --> | <!-- fill --> | services: db, redis, api, frontend |
| 11/14 | Wait for /v1/healthz (≤120s) | <!-- fill --> | <!-- fill --> | attempts until green |
| 12/14 | `curl -fsS /v1/healthz` | <!-- fill --> | <!-- fill --> | must be 2xx |
| 13/14 | `go test -count=1 -timeout 10m ./internal/integration/...` | <!-- fill --> | <!-- fill --> | (skipped until TASK-411 lands integration tests) |
| 14/14 | `docker compose down -v` | <!-- fill --> | <!-- fill --> | cleanup; `if: always()` |

**Overall:** <!-- fill: ✅ all 14 steps passed / ❌ N step(s) failed -->

---

## Annotations / warnings of note

<!-- fill: any ::warning:: or ::notice:: lines emitted by the run that the team
should be aware of. Empty section is fine if the run is clean. -->

---

## Tester approval

The acceptance criteria require explicit sign-off from Tester-01. The line below
is added by Tester-01 (or delegated QA) after reviewing the CI run:

```
TESTER APPROVED — <ISO-8601 timestamp> <name>
```

Until that line is present, the gate is not considered closed for Sprint 4.

---

## How to re-run locally

```bash
./scripts/quality-gate.sh
```

The script mirrors the 14 workflow steps in the same order. It fails loud on
missing prerequisites (`go`, `node`, `docker`, `curl`, `git`) and on any step
that the workflow would also fail. Useful for catching issues before pushing.

---

## How the gate was built

- **Workflow file:** `.github/workflows/sprint-quality-gate.yml`
  - Single job `quality-gate`, runs on `ubuntu-latest`.
  - Triggers on push to `main` and on PRs targeting `main`.
  - Concurrency group cancels in-flight runs on the same ref.
  - 14 numbered steps, each with a `::error::` annotation on failure.
  - Step 14 (`docker compose down -v`) has `if: always()` so the runner cleans up
    containers even on earlier failure.
- **Local script:** `scripts/quality-gate.sh`
  - Bash, `set -euo pipefail`.
  - Mirrors the 14 steps 1:1.
  - Colourised output (`▶` per step, `✅` at success, `::error::` on failure).
  - Optional local cache-key echo (informational only; the real cache lives in GH Actions).

### Notes on the integration test step (13/14)

`src/internal/integration/` does not exist yet — integration tests are landed
by TASK-411 (Testing & Validation). The workflow step therefore prints a
`::notice::` and exits 0 (no test files to fail). Once the directory is
populated, the same step runs the real suite without any change to the
workflow.

### Known limitations of the local runner

- The local cache key is informational only; the actual cache lives in GH
  Actions. Local builds always re-download modules and rebuild.
- The script does not abort and tear down on every failure — it only tears
  down at the end of a successful run. This is intentional: failed local
  runs leave the stack up so the dev can `docker compose logs` and debug.
  The CI workflow *does* always tear down (step 14 has `if: always()`).
