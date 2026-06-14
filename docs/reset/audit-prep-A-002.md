# A-002 Capability System — Pre-Scope Brief

**Owner of A-002:** Builder
**Support:** Leader (this brief)
**Spec:** `docs/sprint4/agent-orchestration-design.md` + `docs/sprint5/agent-creation-management-design.md` (same shape as A-001)
**Audit precedent:** `docs/reset/audit/A-001-audit.md` (the A-001 audit doc is the model for the A-002 audit doc)

## Deliverables (per the team contract)
- Capability Definitions
- Assignment Validation

## Existing code floor (READ FIRST)

| File | Purpose | Status |
|------|---------|--------|
| `src/internal/handler/capability.go` | HTTP: `GET /v1/agents/:id/capabilities` (per-agent), `GET /v1/capabilities` (catalog) | Reads only. No mutation endpoints (per design — capabilities are seeded via migrations). |
| `src/internal/service/capability.go` | `CapabilitiesForRole`, `TaskRequiresCapability`, `AgentHasCapability`, `ValidateAgentHasCapabilities` (TASK-403 validation seam), `FindCompatibleAgents`, `AssignmentScore` | Well-developed. Validation seam is the entry point used by `AssignmentService`. |
| `src/internal/model/capability.go` | 9 catalog caps + 12 agent-type design caps, `AssignableCapabilities()` (5-item subset), `RoleCapabilities` and `AgentTypeCapabilities` maps, validation helpers | Comprehensive. |
| `migrations/016_agent_registry.sql` | Catalog seed (5 assignable + leadership) | Done. |
| `migrations/017_create_agent_capabilities.sql` | `agent_capabilities` join (per-agent grant) | Done. |
| `migrations/018_*.sql` | Likely `task_required_capabilities` join | Read to confirm. |

Total: ~700 LOC of service + ~150 LOC of handler + 167 LOC of model + 2 migrations. Strong floor.

## What I expect to find (gaps to verify)

### 1. Spec drift in `docs/api-spec.md` §Capabilities
Same shape as A-001's 12-point drift fix. Likely drift items:
- Status enum (catalog vs assignable)
- Cursor pagination on `/v1/capabilities` (handler supports it; spec may not)
- Proficiency is `*int` in code (nullable) — spec likely declares it as `int` (non-null)
- `granted_at` is required in response — spec may declare it optional
- Filter by category may or may not be on the spec

Read `docs/api-spec.md` §Capabilities. Cross-ref with the handler.

### 2. Documentation task is in the catalog but NOT in the assignable set (LIKELY BUG)
- `TaskRequiresCapability("documentation")` returns `["documentation"]` (service.go line 62).
- `AssignableCapabilities()` does NOT include `CapDocumentation` (model.go line 75-83).
- `IsAssignableCapability("documentation")` returns `false`.

Consequence: a `task_type=documentation` task requires "documentation" capability, but the validation seam rejects it as a non-assignable capability. **A documentation task can never be assigned.**

Same likely-bug shapes: `data_pipeline` and `project_management` task types (line 64, 66 of service.go) — they require `data_engineering` and `project_management` respectively, which are also in the catalog but not in the assignable set.

**Verify this in the validation seam** (`ValidateAgentHasCapabilities` and the assignment write path in `AssignmentService`). If confirmed, the fix is one of:
- (a) Add `CapDocumentation`, `CapProjectMgmt`, `CapDataEngineering` to `AssignableCapabilities()` (broaden the constraint set)
- (b) Map documentation tasks to a different required-cap list (e.g., "coding") so the catalog stays narrow
- (c) Tighten the role/agent-type defaults so documentation tasks only route to techwriter agents

Recommend (a) — least surprising, matches the catalog intent.

### 3. IDOR surface (F-D002-004) — SAME PATTERN AS A-001
`handler/capability.go:60` uses the `projectIDFromContext(c)` helper, which reads the X-Project-ID header and trusts it. The `ListAgentCapabilities` service call passes `callerProjectID` for the cross-tenant check (F-014 fix). The `ListCapabilities` (catalog) read does NOT scope by project at all — it returns the global catalog. The catalog being global is correct (it's a public taxonomy), but the lack of a project-scope filter on the per-agent read is the IDOR surface.

Sprint 6+ fix (out of scope for the combined Sprint 4+5): the `project_memberships` table + `requireProjectMember` middleware. For A-002, document the surface in the audit doc and log as D-002 OPEN.

### 4. Tests
- Service layer: `service/capability_test.go` — coverage of `TaskRequiresCapability` switch cases, `AgentHasCapability` happy/edge, `ValidateAgentHasCapabilities` happy/edge.
- Model: `model/capability_test.go` — coverage of `AssignableCapabilities`, `IsAssignableCapability`, `DefaultCapabilitiesForRole`, `DefaultCapabilitiesForType`.
- Handler: `handler/capability_test.go` — coverage of `ListAgentCapabilities` (404, 400 missing header, happy), `ListCapabilities` (happy, empty result).
- Integration: extend `internal/integration/integration_test.go` to cover: create agent with grant → assign documentation task → expect fail (or success if (a) is chosen). This is also the D-003 seam.

## Audit doc shape (mirror A-001)
`docs/reset/audit/A-002-audit.md`:
- Evidence: every existing file, line range, public API surface
- Drift inventory: 12 items, each with `code` `spec` `fix`
- Pre-push gate: tests + build + Guardian sign-off + secret-scan
- Hand-backs: anything that crosses into A-003 / B-001 / C-002

## Suggested PR shape
- Commit 1: `docs(api-spec): fix capability drift (N items)` (docs only — mirror A-001)
- Commit 2: `fix(capability): include documentation/pm/data caps in assignable set` (closes the likely bug, item 2 above)
- Commit 3: `test(capability): table-driven coverage for the catalog + validation seam`
- Commit 4: `docs(audit): A-002 capability system audit + pre-push gate`

If the spec drift is non-trivial or the validation seam needs a deeper rewrite, this PR can be 2 PRs. Builder's call.

## When this must land
After the A-002-01..05 hand-backs clear the test gate. Hand-backs branch: `fix/A002-handbacks`. A-002 work proper: `feat/A002-capability-system`. The A-002 PR can be folded with the hand-backs branch at Builder's discretion (rebase-merge the hand-backs in as the first 1-5 commits of the A-002 PR) for a single clean history.

## What I (Leader) will do
- Review the audit doc when ready.
- Cross-ref your spec-drift items against A-001's 12 items (the pattern is likely identical).
- Surface the F-D002-004 IDOR finding in the D-002 report so it doesn't get lost.

## What Guardian (D-002) will do
- Review the security shape of the validation seam.
- Confirm the F-D002-004 surface (the X-Project-ID pattern is the same as A-001 / B-001 / B-002 — they all read the same way).
- Add `internal/handler/capability.go` to the D-002 review checklist.
