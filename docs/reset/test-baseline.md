# D-001 Test Baseline — 2026-06-14

> Owner: Guardian · Branch: `guardian/d-001-test-framework` · Worktree: `guardian-d001-wt/`
> Run timestamp (UTC): 2026-06-14

## Scope

D-001 calls for running the existing backend + frontend test suites from a clean checkout,
capturing pass/fail counts and durations, and recording the result here. The full CI is the
source of truth for the gate; this baseline gives us a quick read of local-machine parity.

## Frontend (Next.js 14 + Vitest) — LOCAL GREEN

```
$ node --version  →  v26.1.0
$ npm  --version  →  11.13.0
$ cd frontend && npx vitest run --reporter=verbose
```

| Test File | Tests | Pass | Fail | Notes |
|---|---:|---:|---:|---|
| `src/hooks/useKanbanDrag.test.ts` | 3 | 3 | 0 | DnD utility — pure logic |
| `src/lib/hooks.test.tsx` | 2 | 2 | 0 | React Query wrapper |
| `src/components/deliverables/MarkdownRenderer.test.tsx` | 6 | 6 | 0 | Render + sanitisation |
| **TOTAL** | **11** | **11** | **0** | **4.78 s wall** |

Verbatim output archived at `out/frontend-test-baseline.txt` and
`out/vitest-stdout.txt` / `out/vitest-stderr.txt`.

Only stderr output is a single deprecation notice from a transitive dependency
(unrelated to our code). No test was skipped or marked `.todo`.

## Backend (Go) — CANNOT RUN LOCALLY (BLOCKER)

The Go toolchain is not installed on this Guardian worktree host:

```
$ go version
'go' is not recognized as an internal or external command
```

Searched standard install locations:

- `C:\Program Files\Go\` — not present
- `C:\Program Files (x86)\Go\` — not present
- `where go` — empty
- `choco list --local-only` — Go not installed
- `GOPATH` / `GOROOT` env vars — unset

**Per the D-001 operating rules, I am not installing Go locally without Lead approval.** The
CI workflow (`.github/workflows/sprint-quality-gate.yml`) installs Go 1.23 via
`actions/setup-go@v5` and is the source of truth for backend verification.

Once the green-light CI run completes, this file will be amended with a pass/fail summary.
For now, the test inventory below is taken from a static file walk — no execution.

### Static backend test inventory (from `**/*_test.go` in worktree)

31 test files, 7,775 lines, covering 9 of 14 internal packages + 3 top-level packages
(only `model` has 8). All Dispatch-priority packages have tests.

| Package | Test files | Lines | Status |
|---|---:|---:|---|
| `internal/agentfactory` | 1 | 413 | ✓ |
| `internal/aion` | 1 | 351 | ✓ |
| `internal/dispatch` | 2 | 549 | ✓ |
| `internal/events` | 2 | 198 | ✓ (thin) |
| `internal/handler` | 5 | 1,798 | ✓ |
| `internal/integration` | 1 | 407 | ✓ |
| `internal/middleware` | 1 | 277 | ✓ |
| `internal/model` | 8 | 681 | ✓ |
| `internal/router` | 2 | 219 | ✓ |
| `internal/service` | 7 | 2,777 | ✓ |
| `internal/config` | 0 | 0 | ✗ (added by D-001) |
| `internal/logger` | 0 | 0 | ✗ follow-up |
| `internal/store` | 0 | 0 | ✗ follow-up (covered by integration) |
| `internal/validation` | 0 | 0 | ✗ follow-up |
| `cmd/`, `db/`, `pkg/errors/` | 0 | 0 | ✗ follow-up |

Full per-file inventory in `coverage-matrix.md`.

## Environment notes

- OneDrive path: `C:\Users\fadhi\OneDrive\Documents\ai-software-factory\`
- Worktree: `C:\Users\fadhi\OneDrive\Documents\ai-software-factory\guardian-d001-wt\`
- Branch: `guardian/d-001-test-framework`
- Base SHA: `bd91463` (main, `chore(ops): harden .gitignore + capture dispatch residual`)
- Repo HEAD on main: `bd91463`

## What the CI will tell us that this baseline cannot

- `go test ./...` result (compile + race + coverage)
- Postgres-backed integration tests
- CodeQL/gosec static analysis
- Frontend build (`next build`)

When the CI run lands, append the green check here and link the run.
