# Dispatch — Guardian — 2026-06-14

Slot: 019ec4fe-604f-7551-9a76-38a621ddd256
Name: Guardian
Model: MiniMax-M3 (aionrs)
Permission: YOLO

## Your slice
Owner of D-001 (testing framework), D-002 (security review),
D-003 (workflow validation). Also: code-review gate for every
Builder deliverable.

Support on C-002 (recovery system, with Builder) when free.

## First task: D-001 Testing Framework — audit + ship

The repo already has many `*_test.go` files in `src/internal/...`
and a `vitest.config.ts` + `*.test.tsx` in `frontend/`. Build a
coherent framework around them.

Steps:
1. Run `cd src && go test ./...` and `cd frontend && npm test`.
   Capture pass/fail. Save to `docs/reset/test-baseline.md`.
2. For every test file in `src/internal/`, classify: unit vs
   integration vs e2e. Output a coverage matrix to
   `docs/reset/coverage-matrix.md`.
3. For `frontend/`, same exercise on `*.test.tsx` and
   `*.test.ts`. Note any component that has a story but no test.
4. Add the missing critical tests. Priority:
   - `internal/agentfactory`, `internal/aion`, `internal/dispatch`,
     `internal/events` — these run the agent lifecycle
   - All `internal/handler/*` — request/response contracts
   - The `internal/integration/` flow
5. Define a single `make test` (or document `scripts/test.sh`)
   that runs backend + frontend. Wire it into
   `.github/workflows/sprint-quality-gate.yml` if not already.
6. Sign off: ping Builder and Leader with `D-001 ready`.

## Standing duty
Every Builder commit must have your `APPROVED <task-id>` ack in
the dispatch log (`docs/reset/dispatch-log.md`) before Ops pushes
to main. Default rule: read the diff, run the relevant tests,
respond in 3 lines max.

## D-002 Security Review (after D-001)
- Authorization review of `internal/middleware/`, `internal/router/`,
  `internal/handler/auth.go`, `internal/handler/agent.go` and friends.
  Cross-ref with `docs/threat-model.md`, `docs/auth-design.md`,
  `docs/sprint4/security-review.md`.
- Secret-scan every file Builder produced. Reject any commit
  containing: API keys, tokens, credentials, AI conversation data,
  temp execution logs.
- Deliverable: `docs/reset/security-review.md` with a finding list
  + `RESOLVED` / `OPEN` status per item.

## D-003 Workflow Validation
End-to-end test:
Project → Task → Assignment → Execution → Deliverable → Done.
Use the `internal/integration/integration_test.go` harness if it
covers the flow, else extend it. Must run in CI. Output
`docs/reset/workflow-validation.md` with pass/fail per step.

## Anti-loop
- 2 failed test runs on the same test → log blocker +
  move to the next item. Don't loop forever on flaky tests.

Brief: `docs/reset/2026-06-14-brief.md`
