# Sprint 4 Security Review — Agent Orchestration Engine

**Reviewer:** Security Agent  
**Date:** 2026-06-12  
**Scope:** Backend handlers, services, middleware, postgres stores, frontend components

---

## 1. Route Protection

### Public vs Protected Routes (`src/internal/router/router.go:12-21`)

| Route | Protected | Notes |
|---|---|---|
| `GET /v1/healthz` | No | Intentional — health check |
| `POST /v1/auth/login` | No | Intentional — authentication |
| `POST /v1/auth/refresh` | No | Intentional — token refresh |
| `POST /v1/users/register` | No | Intentional — user signup |
| All other routes | **Yes** | JWT middleware enforces `Bearer` token |

**Pass.** The public route set is minimal and correct.

---

## 2. Input Validation

### Finding 2.1 [MEDIUM] — No enum validation on agent status/role filters
`handler/agent.go:87-91` casts query parameters directly to `model.AgentStatus(s)` and `filter.Role = r` without validating they are known enum values. Invalid strings propagate unchecked.

**Location:** `src/internal/handler/agent.go:87-91`  
**Fix:** Use `validation.AllowedStrings()` before casting to `model.AgentStatus` and `model.AgentType`.

### Finding 2.2 [MEDIUM] — No enum validation on agent create/update fields
`handler/agent.go:63-69` passes `req.Type`, `req.Role`, `req.Model`, `req.Provider` directly without validation.  
`handler/agent.go:152-153` casts `model.AgentStatus(req.Status)` without checking validity.

**Location:** `src/internal/handler/agent.go:63-69`, `src/internal/handler/agent.go:152-153`  
**Fix:** Validate agent type against `model.AgentType` constants and status against `model.AgentStatus` constants.

### Finding 2.3 [MEDIUM] — Execution status cast without validation
`handler/execution.go:137` casts `model.ExecutionStatus(req.Status)` without checking it's one of `pending/running/completed/failed`.

**Location:** `src/internal/handler/execution.go:137`  
**Fix:** Add `validation.AllowedStrings()` check before casting.

### Finding 2.4 [LOW] — UUID parse errors silently ignored in filters
`handler/execution.go:73-78` and `handler/deliverable.go:77-82` silently drop UUID parse errors on filter query params instead of returning a validation error.

**Location:** `src/internal/handler/execution.go:73-78`, `src/internal/handler/deliverable.go:77-82`  
**Fix:** Either return a validation error or skip the filter silently (the latter is acceptable for optional filters — low severity).

### Finding 2.5 [LOW] — Code handler uses raw string IDs without UUID validation
`handler/code.go:71-73` accepts `projectID` and `filePath` from URL params but only checks for empty string. `projectID` is not UUID-validated.

**Location:** `src/internal/handler/code.go:71-73`, `src/internal/handler/code.go:114-115`  
**Fix:** Validate `projectID` with `uuid.Parse()`.

### Finding 2.6 [LOW] — No length limits on string inputs
Agent Name, Role, Model, Provider, and Deliverable Title/Content have no maximum length validation. Large inputs could cause DoS or storage issues.

**Location:** `src/internal/service/agent.go:44-45` (only checks non-empty), `src/internal/service/deliverable.go` (no validation)  
**Fix:** Add `validation.MaxLength()` for user-facing string fields.

---

## 3. UUID Injection / Panic Potential

### Finding 3.1 [INFO] — UUID parsing is consistent
All handler endpoints for project, task, agent, execution, and deliverable CRUD properly validate UUID parameters with `uuid.Parse()` and return `VALIDATION_ERROR` on failure.

**Pass.** The only gaps are the filter-level silent drops noted in 2.4.

### Finding 3.2 [INFO] — Middleware type assertion
`middleware/middleware.go:66` uses `c.Set(UserIDKey, userID)` where `userID` is already a `string`. The later type assertion `uid.(string)` in handlers is safe as long as the middleware sets the value. No panic vector found.

---

## 4. SQL Injection

All postgres store queries (agent, agent_run, execution, deliverable, project, task) use parameterized bindings (`$1`, `$2`, ...) via pgx. Dynamic `WHERE` clauses are built with hardcoded column names — user values are always bound through the args slice.

**Files reviewed:**
- `src/internal/store/postgres/agent_store.go`
- `src/internal/store/postgres/agent_run_store.go`
- `src/internal/store/postgres/project_store.go`
- `src/internal/store/postgres/task_store.go`

**Verdict: PASS — No injection vectors.**

---

## 5. Authorization

### Finding 5.1 [HIGH] — No access control on agent CRUD
Any authenticated user can create, update, delete, or list any agent. There is no ownership check or project-scoped access control.

**Location:** All `AgentHandler` methods, `AssignmentHandler.AssignTask`, `ExecutionHandler`, `DeliverableHandler`  
**Fix:** Add project-scoped authorization checks (e.g., verify caller owns the project the agent belongs to).

### Finding 5.2 [MEDIUM] — No ownership check on task assignment
`AssignmentService.AssignTaskToAgent` (`src/internal/service/assignment.go:23`) accepts any `taskID` and `agentID` without verifying that the caller has access to the task's project.

**Location:** `src/internal/service/assignment.go:23-83`  
**Fix:** The `CodeService.checkProjectAccess` pattern should be replicated — pass `ctx` and verify project access at the service level.

### Finding 5.3 [MEDIUM] — CodeService has access control but others don't
`CodeService.GenerateCode` and `CodeService.GetFile` call `checkProjectAccess()` to verify the user has access to the project. However, `AssignmentService`, `ExecutionService`, and `DeliverableService` do not perform similar checks.

**Location:** `src/internal/service/code.go:49-51`, `src/internal/service/code.go:88-90`  
**Fix:** Apply `checkProjectAccess` pattern consistently across all services.

---

## 6. Capability Enforcement

### Finding 6.1 [MEDIUM] — Capability check can be bypassed
`AssignmentService.AssignTaskToAgent` determines required capabilities from the agent's role via `CapabilitiesForRole(agent.Role)`, not from the task's requirements. If an agent's capabilities list is empty, it falls back to `agent.Capabilities` (also empty), and returns `CAPABILITY_MISMATCH` only if both are empty. A task requiring "security" could be assigned to a "developer" agent with explicit `["security"]` capability added — but the task type is never checked.

**Location:** `src/internal/service/assignment.go:38-50`, `src/internal/service/capability.go:20-45`  
**Fix:** Determine required capabilities from task type/description rather than agent role. Validate that the task type matches the agent's role.

### Finding 6.2 [LOW] — Capability string matching is fragile
`CapabilityService.AgentHasCapability` does exact string matching. The `model.Capability` constants in `capability.go` define 8 valid capabilities, but the `TaskRequiresCapability` method uses ad-hoc strings like `"coding"`, `"testing"`, and `"project_management"`. These are not validated against the `ValidCapability()` function.

**Location:** `src/internal/service/capability.go:20-44`, `src/internal/model/capability.go:32-38`  
**Fix:** Use `model.ValidCapability()` to validate all capability strings.

---

## 7. Orchestrator Docker Security

### Finding 7.1 [MEDIUM] — Docker container sandboxing is minimal
`src/internal/service/orchestrator.go:80-92` creates containers with resource limits (512MB RAM, 0.5 CPU) but lacks security hardening:

| Configuration | Current | Recommended |
|---|---|---|
| `ReadOnlyRootFilesystem` | Not set | `true` |
| `CapDrop` | Not set | `ALL` (drop all capabilities) |
| `SecurityOpt` | Not set | seccomp or AppArmor profile |
| `NetworkMode` | Default (bridge) | `none` (agent containers shouldn't need network) |
| `Image` | Hardcoded `"ai-software-factory-agent:latest"` | Use pinned digest or version tag |
| `AutoRemove` | `true` | OK — good for cleanup |

**Location:** `src/internal/service/orchestrator.go:80-92`  
**Fix:** Add read-only rootfs, drop all capabilities, consider network mode `none`, pin image version.

### Finding 7.2 [LOW] — AGENT_ID env var passes UUID
`orchestrator.go:83` passes `AGENT_ID` as an environment variable. This is fine functionally but could be logged by the container runtime. No secrets are leaked here.

---

## 8. Webhook SSRF Protection

### Finding 8.1 [MEDIUM] — SSRF validation is unimplemented
`src/internal/service/webhook.go:112-118`:
```go
func validateWebhookURL(rawURL string) error {
    return nil // TODO: implement full SSRF protection
}
```
Any URL is accepted for webhook registration, including internal network addresses (`localhost`, `10.x.x.x`, `169.254.x.x`, etc.).

**Location:** `src/internal/service/webhook.go:42-44`, `src/internal/service/webhook.go:112-117`  
**Fix:** Implement URL validation that rejects private IPs, loopback addresses, and internal DNS names. Consider using a dedicated SSRF prevention library.

---

## 9. Frontend Security

### Finding 9.1 [PASS] — No XSS vectors found
All user-controlled data is rendered via JSX, which auto-escapes HTML entities. Specific checks:
- `deliverables/[id]/page.tsx:73` — `{deliverable.content}` inside `<pre>` — safe (JSX escapes)
- `agents/[id]/page.tsx:154` — `{cap}` inside `<Badge>` — safe
- `agents/page.tsx:153` — `{agent.name}` inside `<Link>` — safe
- No `dangerouslySetInnerHTML` usage found in any component

### Finding 9.2 [PASS] — Token handling
- Access token stored in module-level variable (`src/lib/api.ts:12`) — lost on page refresh (correct)
- Refresh token stored in HttpOnly, Secure, SameSite=Strict cookie (set by backend in `handler/auth.go:43`)
- Auto-refresh on 401 implemented (`src/lib/api.ts:78-87`)
- No localStorage/sessionStorage token storage

### Finding 9.3 [INFO] — AuthProvider initial fetch
`AuthProvider.tsx:39` calls `GET /auth/me` on mount to restore session. This is correct for cookie-based auth.

---

## Finding Summary

| ID | Severity | Category | Description | Location |
|---|---|---|---|---|
| 5.1 | **HIGH** | Authorization | No access control on agents, executions, deliverables | All handler/*.go |
| 5.3 | **MEDIUM** | Authorization | CodeService has project access checks; AssignmentService/ExecutionService do not | `service/assignment.go`, `service/execution.go` |
| 6.1 | **MEDIUM** | Capabilities | Task type not validated against agent capabilities | `service/assignment.go:38-50` |
| 7.1 | **MEDIUM** | Docker | Container sandboxing is minimal (no RO rootfs, no cap drop, no seccomp) | `service/orchestrator.go:80-92` |
| 8.1 | **MEDIUM** | SSRF | Webhook URL validation is a TODO stub | `service/webhook.go:112-118` |
| 2.1 | **MEDIUM** | Input Validation | No enum validation on agent status/role filters | `handler/agent.go:87-91` |
| 2.2 | **MEDIUM** | Input Validation | No enum validation on agent create/update | `handler/agent.go:63-69,152-153` |
| 2.3 | **MEDIUM** | Input Validation | No enum validation on execution status | `handler/execution.go:137` |
| 5.2 | **MEDIUM** | Authorization | No ownership check on task assignment | `service/assignment.go:23` |
| 2.4 | **LOW** | Input Validation | UUID parse errors silently ignored in filters | `handler/execution.go:73-78`, `handler/deliverable.go:77-82` |
| 2.5 | **LOW** | Input Validation | Code handler uses raw string IDs | `handler/code.go:71-73,114-115` |
| 2.6 | **LOW** | Input Validation | No length limits on string inputs | `service/agent.go:44-45` |
| 6.2 | **LOW** | Capabilities | Fragile capability string matching | `service/capability.go:20-44` |
| 7.2 | **LOW** | Docker | Minor container config hardening | `service/orchestrator.go:83` |
| 3.1 | **INFO** | UUID Handling | UUID parsing consistent across CRUD endpoints | All handlers |
| 4 | **PASS** | SQL Injection | All queries parameterized | All store/postgres/ |
| 9.1 | **PASS** | XSS | No XSS vectors in React components | All frontend .tsx |
| 9.2 | **PASS** | Token Handling | Secure token storage (in-memory + HttpOnly cookie) | `lib/api.ts`, `handler/auth.go` |

---

## Recommendations (Priority Order)

1. **P0:** Add project-scoped authorization checks across all handlers/services (5.1, 5.2, 5.3)
2. **P1:** Implement SSRF protection for webhook URLs (8.1)
3. **P1:** Harden Docker container sandboxing in orchestrator (7.1)
4. **P1:** Add enum validation for agent status, agent type, execution status, and deployment environment on all handlers (2.1, 2.2, 2.3)
5. **P1:** Fix task capability matching to validate against task type rather than agent role (6.1)
6. **P2:** Add input length limits for user-facing string fields (2.6)
7. **P2:** Handle UUID parse errors explicitly in filter query params (2.4)
8. **P2:** Validate projectID as UUID in code handler (2.5)
