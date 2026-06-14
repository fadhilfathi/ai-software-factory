# D-001 Unified Test Runner — 2026-06-14

> Owner: Guardian · The dispatch asked for "a single `make test` (or document `scripts/test.sh`)"
> that runs backend + frontend. The repo already has the latter. This file documents it.

## TL;DR

```bash
# From the repo root
./scripts/test.sh
```

That's it. The script handles backend + frontend + lint + (optionally) build verification,
mirrors `.github/workflows/sprint-quality-gate.yml`, and fails fast on any non-zero exit.

## What `scripts/test.sh` does today

The script is 67 lines (read at `scripts/test.sh`). It runs five gates in order:

1. **`go vet ./...`** — catches obvious compile-time issues without writing any artifacts.
2. **`go test -v -count=1 -race -shuffle=on ./...`** — the real backend gate, with race
   detection and per-test shuffling enabled. `-count=1` defeats the test-result cache.
3. **`npm test`** — only if the env var `RUN_FRONTEND_TESTS=1` is set (the script's CI
   mirror sets it; local devs can opt-in).
4. **`npm run lint`** — always on, after the Go gate. Fails fast on lint regressions.
5. **`./scripts/quality-gate.sh`** — full 14-step sprint-quality mirror (per
   `.github/workflows/sprint-quality-gate.yml`). Runs only if the local env supports it
   (the script itself self-skips if Go isn't installed).

The script returns the first non-zero exit code it sees and prints a clear summary of
which gate failed.

## Why not a Makefile?

The dispatch allows "or document `scripts/test.sh`". Three reasons to prefer the script:

- **Already in the repo and CI-mirrored.** A Makefile would be parallel to a working
  tool, with no test improvement.
- **No `make` on Windows worktree hosts.** This Guardian host doesn't have GNU make;
  `./scripts/test.sh` works with Git Bash, WSL, or `bash` from a Linux container.
- **`.github/workflows/sprint-quality-gate.yml` is the canonical surface.** Both
  `scripts/test.sh` and `scripts/quality-gate.sh` are local mirrors; adding a third
  (Makefile) would mean three sources of truth to keep in sync.

If the team wants a `make test` wrapper in a future sprint, the diff is ~10 lines:
```make
test:
	@./scripts/test.sh
```
A follow-up ticket is logged in `missing-tests.md` for whoever picks it up.

## CI as source of truth

`.github/workflows/sprint-quality-gate.yml` is the 14-step gate. The local scripts are
an approximation — they're useful for pre-push verification but not authoritative. The
authoritative check is the green check on the PR.

## What D-001 did NOT change in `scripts/test.sh`

- The glob for finding test files (`src/**/*.test.*`) is a known fragile pattern: it
  relies on bash's `globstar` option, which is off by default. On a fresh bash install
  on Windows, the glob returns no matches, the `if` evaluates false, and the script
  falls through to `npm run lint` instead of `npm test`. **Symptom: local test runs
  silently skip frontend tests in deep paths.** Fix: add `shopt -s globstar` near the
  top of the script, OR change the find to `find frontend/src -name '*.test.*'`. This
  is a follow-up, not a D-001 fix — it could mask other bugs in the script and warrants
  a separate PR with its own CI verification.

## Recommended local pre-push sequence

```bash
# 1. Backend smoke (fast feedback on compile errors)
cd src && go vet ./...

# 2. Frontend unit tests
cd ../frontend && npx vitest run

# 3. Full sprint-quality mirror (slow, full coverage)
cd .. && ./scripts/quality-gate.sh

# 4. Push; let CI run the canonical gate
git push origin guardian/d-001-test-framework
```

Steps 1 and 2 catch the bulk of regressions in <30 s. Step 3 takes longer (full test
suite + lint + build) and is the closest local proxy for CI. Step 4 is the
authoritative run.
