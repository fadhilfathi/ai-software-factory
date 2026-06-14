# Dispatch — Builder — 2026-06-14

Slot: 019ec4fe-6040-7af0-b408-4a7e6fc718e8
Name: Builder
Model: MiniMax-M3 (aionrs)
Permission: YOLO

## Your slice
Owner of A-001..A-003 (foundation), B-001..B-003 (execution),
C-001..C-002 (observability). 8 task-board items.

Support on D-001 (testing framework, with Guardian) when free.

## First task: A-001 Agent Registry — audit + ship

The existing surface in `src/internal/handler/agent.go`,
`src/internal/service/agent.go`, `src/internal/model/agent.go`,
`src/internal/agentfactory/`, and migrations `005_create_agents.sql`,
`010_update_agents.sql`, `015_update_agents_table.sql`,
`016_agent_registry.sql`, `017_create_agent_capabilities.sql`,
`026_add_agents_runtime.sql` already covers most of CRUD + storage +
status. A `model/agent_type.go` and `model/worker.go` also exist.

Steps:
1. Open each of the above files. Read. Build a gap list vs the spec
   in `docs/sprint4/agent-orchestration-design.md` and
   `docs/sprint5/agent-creation-management-design.md`.
2. Add what's missing. Delete what's wrong. No drive-by refactors.
3. `go build ./...` → must pass.
4. `go test ./internal/...` → must pass. Add tests for any code path
   that lacks coverage. Aim for ≥80% on the touched packages.
5. Request Guardian review: ping Guardian with a diff summary + the
   list of changed files. Wait for `APPROVED A-001` ack.
6. Local commit: `feat(task-A001): ship agent registry`
7. Hand off to Ops: `git push origin main` after Ops runs the
   secret-scan and CI green-light.

Do NOT push to main yourself. Ops owns git operations per the
team contract (E-002 + E-003).

## After A-001 ships
Pick up A-002 Capability System. Same loop. Then A-003. Then B-001.
Keep the worktree pattern if you want isolation; otherwise commit
directly — but always wait for Ops to push.

## Anti-loop
- 2 failed build/test attempts on the same error → log blocker to
  `docs/reset/blockers.md` and ping Leader.
- Don't chase Sprint 4/5 anomalies from `docs/sprint5/lead-updates/`.
  That history is closed. The brief says so.

## Ping me
- Status updates: terse. One line. "A-001 audit done, 3 gaps, working."
- Blocker: one line + blocker file.
- Done: "A-001 ready for Guardian review." + diff summary.

Brief: `docs/reset/2026-06-14-brief.md`
