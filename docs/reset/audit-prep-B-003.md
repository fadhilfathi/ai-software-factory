# B-003 Deliverable Storage — Pre-Scope Brief

**Owner of B-003:** Builder. Support: Ops (E-001 infra smoke for the deliverable store).
**Spec:** `docs/sprint4/agent-orchestration-design.md` §Deliverable + `docs/sprint5/agent-creation-management-design.md` (the TASK-406 + F-023 line of work)
**Audit precedent:** `docs/reset/audit/A-001-audit.md`, `docs/reset/audit-prep-A-002.md`, `docs/reset/audit-prep-A-003.md`

## Deliverables (per the team contract)
- Deliverable Persistence
- Version History

## Existing code floor (READ FIRST)

| File | Purpose | Status |
|------|---------|--------|
| `src/internal/model/deliverable.go` | `Deliverable` (current), `DeliverableVersion` (history, append-only), `DeliverableFilter` (cursor pagination), `MaxDeliverableContentBytes = 1 MiB` (F-023 DoS hardening) | Solid. Version is server-assigned. |
| `src/internal/handler/deliverable.go` | Create/List/Get/Update (PUT creates a new version, not in-place update). 4 endpoints now wrap in `{data: ...}` envelope (per A-002-11). | Reads + 2 writes. |
| `src/internal/service/deliverable.go` | `CreateDeliverable` (initial), `UpdateDeliverable` (writes a new version row, increments `version`, updates the `deliverables` row in one tx), `ListDeliverables` (cursor pagination, requires at least one of TaskID/AgentID filter) | Solid. |
| `src/internal/store/postgres/deliverable.go` | PG-backed store. Schema covered by migrations 009, 022, 023. | Read. |
| `migrations/009_create_deliverables.sql` | Initial `deliverables` table | Read. |
| `migrations/022_deliverable_versioning.sql`, `023_*.sql` | `deliverable_versions` table + UNIQUE(deliverable_id, version) | Read. |
| `internal/handler/deliverable_test.go` | TestDeliverableHandler_List_WithFilters redesigned in A-002-11 (3 deliverables, no cross-tenant 4th create per TASK-421) | Read. |

## Key design notes (from the model file)

- **Append-only invariant**: `deliverable_versions` rows are never UPDATEd. Every PUT writes a NEW row with version+1. The unique constraint (deliverable_id, version) → 409 on duplicate. The current `deliverables` row mirrors the latest version's title and content.
- **Server-assigned version**: callers don't supply a version; the service computes the next version from the current row. So no `IsValidDeliverableVersion` validator exists (none needed).
- **Cursor-based pagination**: ordered by (created_at DESC, id DESC) for stability. `NextCursor` is empty when there are no more pages.
- **DoS hardening**: `MaxDeliverableContentBytes = 1 MiB` (F-023). The handler also uses `http.MaxBytesReader` with headroom.
- **Cross-tenant check**: at the service layer (not the model). The `X-Project-ID` pattern is the F-D002-004 surface.

## What B-003 needs to do

### 1. Spec drift in `docs/api-spec.md` §Deliverables
Same shape as A-001 (12) / A-002 (12) / A-003 (12) — likely another 12 items. Read `docs/api-spec.md` §Deliverables and cross-ref. Likely drift:
- The PUT semantic: spec may say "update the deliverable" but the code says "create a new version" (different mental model)
- The version field in the response: spec may declare it `int` non-null (matches code), or may declare it nullable
- The `NextCursor` shape: spec may use string token vs UUID
- Filter requirement: spec may allow empty filter (code rejects with 400)
- List ordering: spec may declare created_at ASC vs code's DESC
- `created_by` field: nullable in code (omitempty) — spec may declare it required

### 2. Test coverage (the A-002-15..18-style cascade)
- Service: `service/deliverable_test.go` — coverage of `CreateDeliverable` (happy, validation, content too large, F-023), `UpdateDeliverable` (happy, version increment, 409 on duplicate version race, F-D002-004 cross-tenant), `ListDeliverables` (cursor pagination happy + boundary, filter requirement 400, empty result, cross-tenant)
- Model: `model/deliverable_test.go` — version invariants, ordering invariants, `MaxDeliverableContentBytes` enforcement at the service boundary
- Handler: `handler/deliverable_test.go` — `CreateDeliverable` (happy, 413 too large, 400 missing X-Project-ID, 404 crossTenantBlocked), `UpdateDeliverable` (happy, 409 duplicate, 404 not found, 413 too large), `ListDeliverables` (the A-002-11 redesigned test), `GetDeliverable` (happy, 404 not found)
- Integration: extend `internal/integration/integration_test.go` per D-003 — add the deliverable step in the 15-step T1 lifecycle

### 3. Append-only invariant enforcement
- The unique constraint at the DB level is the enforcement. The service code should never UPDATE a `deliverable_versions` row.
- Add a `staticcheck` or custom check that flags any code path that mutates a `DeliverableVersion` after insert. (Or just rely on the UNIQUE constraint + 23505 mapping to 409 in the handler.)
- The audit doc should call out: "deliverable_versions is append-only by design; the only writes are INSERTs in `UpdateDeliverable` and the unique constraint is the only safety net."

### 4. Version history listing
- `ListDeliverableVersions(deliverableID, cursor, limit)` — already in the service? Verify.
- The response should include the full history, ordered by version DESC.
- Cross-tenant check on the parent deliverable (F-D002-004 surface — a user in project A shouldn't be able to read versions of a deliverable in project B).

### 5. Content storage decision
- The current `Deliverable.Content` is a `string` (markdown body, up to 1 MiB). For larger artifacts (binary files, large datasets), this won't scale.
- **Out of scope for B-003** (Sprint 6+): object-store-backed deliverable content (S3 / GCS). Log in the audit doc as a known limitation.
- For B-003: verify the 1 MiB cap is enforced and the F-023 message is clear.

### 6. F-D002-004 IDOR
Same X-Project-ID surface. Log in the audit doc as D-002 OPEN (Sprint 6+). Add cross-tenant negative tests to the integration suite per D-003.

## Audit doc shape (mirror A-001 / A-002 / A-003)
`docs/reset/audit/B-003-audit.md`:
- Evidence: every existing file, the F-023 DoS hardening, the append-only invariant, the cross-tenant pattern
- Drift inventory: 12 items, each with `code` `spec` `fix`
- Pre-push gate: tests + build + Guardian sign-off + secret-scan
- Hand-backs: anything that crosses into D-003 (the integration test step) or B-002 (the webhook event on `DeliverableCreated`)

## Suggested PR shape
- Commit 1: `docs(api-spec): fix deliverable drift (N items)` (mirror A-001/A-002/A-003 — likely 12 items)
- Commit 2: `test(deliverable): table-driven coverage for Create/Update/List/Get + version history` (extends the A-002-11 redesign)
- Commit 3: `fix(deliverable): [any code gap surfaced by the audit]` (e.g., cross-tenant check on ListDeliverableVersions, content cap message, cursor ordering)
- Commit 4: `docs(audit): B-003 deliverable storage audit + pre-push gate`

## When this must land

After B-001 (Execution Engine) and B-002 (Agent Communication) — the deliverable is the OUTPUT of an execution, and the webhook fires on `DeliverableCreated` (B-002 surface). B-003 ships after B-001 + B-002 have settled the lifecycle and event types.

## What I (Leader) will do

- Review the audit doc.
- Surface the F-D002-004 IDOR in the D-002 report.
- Coordinate the `DeliverableCreated` event type with the B-002 webhook dispatcher (Guardian's D-002 should re-review the webhook surface when B-003 ships).

## What Guardian (D-002 / D-003) will do

- Add the deliverable endpoints to the D-002 review checklist.
- Verify the cross-tenant check on `ListDeliverableVersions` (often missed — easy to scope by parent and forget the deliverable's own project).
- Extend the D-003 integration test with the deliverable step.

## What Ops (E-001) will do (downstream)

- Add a `docker compose up` smoke that creates a deliverable via the API and confirms the row is in the Postgres `deliverables` + `deliverable_versions` tables.
- Add a `gitleaks` config to ignore test fixture content (deliverable content can include URL/password patterns that gitleaks flags as false positives).
