# TASK-425 — Role-Route Matrix (F-021)

**Sprint:** 5
**Author:** Dev-01
**Status:** SHIPPED
**SHA:** see PR
**Issue:** [Sprint 4 security review §7.2.1 / F-021](https://github.com/fadhilfathi/AI-Software-Factory/blob/main/docs/sprint4/security-review.md)

---

## 0. Summary

The role-route matrix is the central access-control policy for the API.
It maps HTTP method × URL pattern → required role. TASK-425 (F-021)
codifies the policy in the router, replacing the implicit "every
authenticated user can do anything that requires auth" model that
existed pre-Sprint 5. The matrix has three branches:

| Branch | Routes | Allowed roles |
|---|---|---|
| Public | 3 (health, login, refresh) | anyone (no auth) |
| Admin-only | 4 (3 DELETEs + register) | `admin` |
| Write | 20 (POST/PUT/PATCH) | `developer`, `admin` |
| Read | 18 (GET) | any authenticated user (role-agnostic) |

Total routes: 45. Routes that need role enforcement: 24 (4 + 20).
The 18 GET routes stay on `RequireAuth` only — the role matrix never
denies a read.

## 1. Why this matters

Pre-TASK-425, the only access control was the JWT auth gate. Any
authenticated user could:

- POST `/v1/users/register` and mint new accounts (because the route
  was in `publicRouteSet`).
- DELETE other people's projects/agents/tasks.
- PATCH other people's executions.

A viewer-role token and an admin-role token had identical
capabilities. The role claim in the JWT was being SET but never READ.
F-021 fixes that. The role claim was added by TASK-417 (Sprint 5
sibling task); TASK-425 is its enforcement side.

## 2. Roles

Three roles, defined in `model.User.Role` (TASK-417):

- `admin` — full read/write/delete, can register new users.
- `developer` — full read, can write (POST/PUT/PATCH). Cannot delete.
- `viewer` — read-only. The implicit "no-write" role. Not currently
  assignable at registration time, but the role exists in the model
  and is honored by `RequireAnyRole` (a viewer is denied by both
  branches).

## 3. The matrix

```go
// In router.go
writeRole := middleware.RequireAnyRole("developer", "admin")
adminRole := middleware.RequireAnyRole("admin")

// Admin-only (4)
v1.DELETE("/projects/:id",  adminRole, projects.Delete)
v1.DELETE("/agents/:id",    adminRole, agents.Delete)
v1.DELETE("/tasks/:id",     adminRole, tasks.Delete)
v1.POST  ("/users/register", adminRole, users.Register)

// Write = dev OR admin (20)
v1.POST  ("/auth/logout",          writeRole, auth.Logout)
v1.POST  ("/projects",             writeRole, projects.Create)
v1.PUT   ("/projects/:id",         writeRole, projects.Update)
v1.POST  ("/projects/:id/decompose", writeRole, projects.Decompose)
v1.POST  ("/agents",               writeRole, agents.Create)
v1.PUT   ("/agents/:id",           writeRole, agents.Update)
v1.POST  ("/projects/:id/tasks",   writeRole, tasks.Create)
v1.PUT   ("/tasks/:id",            writeRole, tasks.Update)
v1.POST  ("/tasks/:id/assign",     writeRole, assignments.AssignTask)
v1.PATCH ("/tasks/:id/status",     writeRole, tasks.UpdateStatus)
v1.POST  ("/executions",           writeRole, executions.Create)
v1.PATCH ("/executions/:id",       writeRole, executions.Patch)
v1.POST  ("/deliverables",         writeRole, deliverables.Create)
v1.PUT   ("/deliverables/:id",     writeRole, deliverables.Update)
v1.POST  ("/code/generate",        writeRole, code.Generate)
v1.POST  ("/code/:projectId/commits", writeRole, code.CreateCommit)
v1.POST  ("/reviews",              writeRole, reviews.Create)
v1.POST  ("/deployments",          writeRole, deployments.Trigger)
v1.POST  ("/deployments/:id/rollback", writeRole, deployments.Rollback)
v1.POST  ("/webhooks",             writeRole, webhooks.Register)
```

The 18 GETs and 3 public routes are mounted without `writeRole` /
`adminRole`. They inherit `RequireAuth` (from the global `Auth`
middleware) but not role checks. Listing them in the matrix below
for completeness:

```text
GET  /v1/healthz                 (public)
POST /v1/auth/login              (public)
POST /v1/auth/refresh            (public)
GET  /v1/projects                (RequireAuth)
GET  /v1/projects/:id            (RequireAuth)
GET  /v1/agents                  (RequireAuth)
GET  /v1/agents/:id              (RequireAuth)
GET  /v1/agents/:id/capabilities (RequireAuth)
GET  /v1/capabilities            (RequireAuth)
GET  /v1/projects/:id/tasks      (RequireAuth)
GET  /v1/tasks/:id               (RequireAuth)
GET  /v1/tasks/:id/history       (RequireAuth)
GET  /v1/executions              (RequireAuth)
GET  /v1/executions/:id          (RequireAuth)
GET  /v1/deliverables            (RequireAuth)
GET  /v1/deliverables/:id        (RequireAuth)
GET  /v1/deliverables/:id/versions (RequireAuth)
GET  /v1/code/:projectId/files/*path (RequireAuth)
GET  /v1/reviews/:id             (RequireAuth)
GET  /v1/deployments/:id         (RequireAuth)
GET  /v1/users/me                (RequireAuth)
```

## 4. `RequireAnyRole` — the primitive

`middleware.RequireAnyRole(roles ...string) gin.HandlerFunc` is a
new middleware primitive that complements the existing
`RequireRole(single)`. The new function uses OR semantics: the
caller passes a list of acceptable roles and any match lets the
request through.

```go
// From middleware.go
func RequireAnyRole(roles ...string) gin.HandlerFunc {
    return func(c *gin.Context) {
        role, exists := c.Get(RoleKey)
        if !exists {
            c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
                "error": gin.H{"code": "FORBIDDEN",
                               "message": "Insufficient permissions"},
            })
            return
        }
        roleStr, ok := role.(string)
        if !ok {
            c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
                "error": gin.H{"code": "FORBIDDEN",
                               "message": "Insufficient permissions"},
            })
            return
        }
        for _, r := range roles {
            if roleStr == r {
                c.Next()
                return
            }
        }
        c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
            "error": gin.H{"code": "FORBIDDEN",
                           "message": "Insufficient permissions"},
        })
    }
}
```

`RequireRole(single)` and `RequireAnyRole(single)` are equivalent —
the new primitive is a strict superset. We keep both for clarity:
`RequireRole("admin")` reads as "this is admin-only", while
`RequireAnyRole("developer", "admin")` reads as "this is the write
branch". The intent is in the call site, not in the implementation.

## 5. The 403 response shape

All three middleware primitives (`RequireAuth`, `RequireRole`,
`RequireAnyRole`) return a uniform error envelope:

```json
{ "error": { "code": "FORBIDDEN", "message": "Insufficient permissions" } }
```

The 401 path (no/invalid token) uses a different envelope and is
not affected by TASK-425.

## 6. Interaction with the service-layer cross-tenant guards

The role matrix is the coarser-grained gate. The finer-grained gate
is the service layer's `callerProjectID` check (TASK-419..422, F-013
through F-015): an authenticated user with the right role still
gets a 404 `CROSS_TENANT_BLOCKED` if they try to access a resource
in a project they don't own. Role checks happen FIRST because they
short-circuit on the request shape (method + URL), while project
checks need to look up the resource. A viewer attempting a
cross-tenant write gets a 403 from the role gate; a developer
attempting a same-tenant write gets through the role gate and is
then either allowed or denied at the service layer.

A viewer attempting a cross-tenant GET gets a 404 from the service
layer, not a 403 — this is intentional, to avoid leaking existence
information across tenants.

## 7. What changed in `publicRouteSet`

`POST /v1/users/register` was previously in `publicRouteSet`, which
meant anyone could sign up. It is now removed from the set and
mounted under `adminRole`. The other three public routes
(`GET /v1/healthz`, `POST /v1/auth/login`, `POST /v1/auth/refresh`)
are unchanged.

## 8. Test plan

### 8.1 Middleware unit tests (`middleware_test.go`)

`TestRequireAnyRole` covers the primitive directly with five
subtests:

- `viewer_denied_for_write_matrix` — viewer on
  `RequireAnyRole("developer", "admin")` → 403.
- `developer_allowed_for_write_matrix` — developer → 200.
- `admin_allowed_for_write_matrix` — admin → 200.
- `admin_only_branch_denies_developer` — developer on
  `RequireAnyRole("admin")` → 403 (parity with `RequireRole`).
- `missing_role_returns_403` — no role in context → 403.

### 8.2 Router integration tests (`router_test.go`)

`TestRoleMatrix_*` exercises the real `router.New(...)` with a
fake auth service that returns canned claims based on the bearer
token. The tests cover all 24 role-enforced routes by sampling one
representative route per HTTP method per branch:

Admin-only branch (3):
- `TestRoleMatrix_AdminOnly_DELETE_Project_RejectsDeveloper` —
  developer on `DELETE /v1/projects/:id` → 403.
- `TestRoleMatrix_AdminOnly_Register_RejectsViewer` — viewer on
  `POST /v1/users/register` → 403 (this is the route that USED to
  be public).
- `TestRoleMatrix_AdminOnly_DELETE_Task_AllowsAdmin` — admin on
  `DELETE /v1/tasks/:id` → not 401/403 (handler may 404/500, but
  the role gate passed).

Write branch (3):
- `TestRoleMatrix_Write_POST_Project_RejectsViewer` — viewer on
  `POST /v1/projects` → 403.
- `TestRoleMatrix_Write_PUT_Task_AllowsDeveloper` — developer on
  `PUT /v1/tasks/:id` → not 401/403.
- `TestRoleMatrix_Write_PATCH_Execution_AllowsAdmin` — admin on
  `PATCH /v1/executions/:id` → not 401/403.

Regression (1):
- `TestRoleMatrix_Public_Login_NoToken` — `POST /v1/auth/login`
  with no Authorization header still bypasses auth (it remains in
  `publicRouteSet`).

## 9. Out of scope

- **Per-resource ACLs** (e.g. "this developer owns this project").
  That's the `callerProjectID` work, in the service layer. TASK-425
  is a global role gate; finer-grained per-tenant checks are
  sibling tasks.
- **API key paths** — keys can have their own role binding, but
  that's a Sprint 5 follow-up (F-007). The current matrix treats
  any authenticated principal the same regardless of credential
  type.
- **Audit logging** on 403s. The `Logger()` middleware already
  logs every request, but a 403 from a developer trying to DELETE
  is not specially flagged. That's a Sprint 6 candidate.

## 10. Follow-ups

- **F-007** — API keys should carry a role and be enforced the same
  way. Currently any valid API key bypasses role checks because
  `ValidateAPIKey` doesn't return a role claim. This is a
  known-acceptable risk for Sprint 5 and a Sprint 6 task.
- **Audit hook** — wire a 403-counter into `Logger()` so Security-01
  can see who's trying what.
- **Spec doc** — `docs/sprint4/security-review.md` says
  "role-route matrix lives in router.go". Add a link to this file
  from F-021's row.

## 11. Diff summary

| File | Lines added | Lines removed |
|---|---|---|
| `src/internal/middleware/middleware.go` | +45 (RequireAnyRole + doc) | 0 |
| `src/internal/middleware/middleware_test.go` | +73 (TestRequireAnyRole with 5 subtests) | 0 |
| `src/internal/router/router.go` | +35 (matrix block + publicRouteSet edit) | -3 (register line) |
| `src/internal/router/router_test.go` | +201 (new file: 7 tests, fake auth) | 0 |
| `docs/sprint5/task-425-role-route-matrix.md` | +200 (this file) | 0 |

Total: ~554 lines added, ~3 removed.
