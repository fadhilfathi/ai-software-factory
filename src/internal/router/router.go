package router

import (
	"net/http"

	"github.com/fadhilfathi/AI-Software-Factory/internal/config"
	"github.com/fadhilfathi/AI-Software-Factory/internal/handler"
	"github.com/fadhilfathi/AI-Software-Factory/internal/middleware"
	"github.com/fadhilfathi/AI-Software-Factory/internal/service"
	"github.com/gin-gonic/gin"
)

// publicRouteSet is the set of (method, path-prefix) pairs that bypass auth.
// Sprint 4: Auth middleware takes map[string]bool keyed by method+path.
// Matches the convention used in other services in the repo.
//
// TASK-425 (F-021): POST /v1/users/register was previously public so anyone
// could sign up. The role-route matrix moves it under admin-only auth so
// admins control onboarding. Login + refresh stay public (clients need
// them to obtain a token in the first place).
var publicRouteSet = map[string]bool{
	"GET /v1/healthz":       true,
	"HEAD /v1/healthz":      true,
	"POST /v1/auth/login":   true,
	"POST /v1/auth/refresh": true,
	// POST /v1/users/register was REMOVED in TASK-425 (F-021) — it is now
	// admin-only, mounted under RequireAnyRole("admin") below. Sprint 5
	// PR #17 added HEAD /v1/healthz so Docker / wget --spider healthchecks
	// can probe without keeping a connection open.
}

// healthzHandler responds to GET and HEAD on /v1/healthz.
// Sprint 5: HEAD support added so Docker / wget --spider healthchecks pass.
func healthzHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func New(svc *service.Services, cfg *config.Config, corsConfig middleware.CORSConfig, rateLimitConfig middleware.RateLimitConfig) *gin.Engine {
	r := gin.New()

	r.Use(middleware.Recovery())
	r.Use(middleware.Logger())
	r.Use(middleware.RequestID())
	r.Use(middleware.CORS(corsConfig))
	r.Use(middleware.RateLimit(rateLimitConfig))
	r.Use(middleware.Auth(svc.Auth, publicRouteSet))

	// TASK-425 (F-021): the role-route matrix. Two role guards are needed:
	//
	//   - writeRole: developer OR admin can write (POST/PUT/PATCH).
	//     Viewer is the implicit "read-only" role and is denied.
	//   - adminRole: admin only (DELETE + POST /v1/users/register).
	//
	// GET routes stay under RequireAuth alone — every authenticated user can
	// read. The service layer still enforces project-scope (TASK-419..422
	// cross-tenant guards), so a viewer who guesses another tenant's UUID
	// gets a 404 CROSS_TENANT_BLOCKED, not a 403. Role checks happen
	// BEFORE project-scope checks because the role matrix is the
	// coarser-grained gate.
	writeRole := middleware.RequireAnyRole("developer", "admin")
	adminRole := middleware.RequireAnyRole("admin")

	auth := handler.NewAuthHandler(svc.Auth, cfg.Auth.CookieSecure)
	projects := handler.NewProjectHandler(svc.Project)
	agents := handler.NewAgentHandler(svc.Agent)
	capabilities := handler.NewCapabilityHandler(svc.Agent) // TASK-403: capability routes moved off AgentHandler
	tasks := handler.NewTaskHandler(svc.Task)
	assignments := handler.NewAssignmentHandler(svc.Assignment)
	executions := handler.NewExecutionHandler(svc.Execution, svc.Log)
	deliverables := handler.NewDeliverableHandler(svc.Deliverable)
	code := handler.NewCodeHandler(svc.Code)
	reviews := handler.NewReviewHandler(svc.Review)
	deployments := handler.NewDeploymentHandler(svc.Deployment)
	users := handler.NewUserHandler(svc.User)
	webhooks := handler.NewWebhookHandler(svc.Webhook)

	v1 := r.Group("/v1")
	{
		v1.GET("/healthz", healthzHandler)
		v1.HEAD("/healthz", healthzHandler)
		v1.POST("/auth/login", auth.Login)
		v1.POST("/auth/refresh", auth.Refresh)

		// --- Write = developer OR admin (20 routes). ---
		v1.POST("/auth/logout", writeRole, auth.Logout)

		v1.POST("/projects", writeRole, projects.Create)
		v1.PUT("/projects/:id", writeRole, projects.Update)
		v1.POST("/projects/:id/decompose", writeRole, projects.Decompose)

		v1.POST("/agents", writeRole, agents.Create)
		v1.PUT("/agents/:id", writeRole, agents.Update)

		v1.POST("/projects/:id/tasks", writeRole, tasks.Create)
		v1.PUT("/tasks/:id", writeRole, tasks.Update)
		v1.POST("/tasks/:id/assign", writeRole, assignments.AssignTask)
		v1.PATCH("/tasks/:id/status", writeRole, tasks.UpdateStatus)

		v1.POST("/executions", writeRole, executions.Create)
		v1.PATCH("/executions/:id", writeRole, executions.Patch)
		// B-001 reviewer action: the only path into COMPLETED.
		v1.PATCH("/executions/:id/review", writeRole, executions.Review)

		v1.POST("/deliverables", writeRole, deliverables.Create)
		v1.PUT("/deliverables/:id", writeRole, deliverables.Update)

		v1.POST("/code/generate", writeRole, code.Generate)
		v1.POST("/code/:projectId/commits", writeRole, code.CreateCommit)

		v1.POST("/reviews", writeRole, reviews.Create)

		v1.POST("/deployments", writeRole, deployments.Trigger)
		v1.POST("/deployments/:id/rollback", writeRole, deployments.Rollback)

		// F-D002-005 (D-002 review §3.3): POST /v1/webhooks
		// is admin-only. The previous `writeRole` let
		// developers register webhooks, which combined with
		// the X-Project-ID IDOR (F-D002-004) to let a
		// developer in project A register a webhook in
		// project B. Tightened to `adminRole` so only admins
		// can register; the F-D002-004 fix (project_memberships)
		// is the second leg.
		v1.POST("/webhooks", adminRole, webhooks.Register)

		// --- Admin-only (4 routes): DELETE + /v1/users/register. ---
		v1.DELETE("/projects/:id", adminRole, projects.Delete)
		v1.DELETE("/agents/:id", adminRole, agents.Delete)
		v1.DELETE("/tasks/:id", adminRole, tasks.Delete)
		// B-001 operator cancel: only the operator can hard-cancel an execution.
		v1.DELETE("/executions/:id", adminRole, executions.Cancel)
		v1.POST("/users/register", adminRole, users.Register)

		// --- Read = RequireAuth only (18 routes). No role check. ---
		v1.GET("/projects", projects.List)
		v1.GET("/projects/:id", projects.Get)

		v1.GET("/agents", agents.List)
		v1.GET("/agents/:id", agents.Get)
		// TASK-403: capability routes moved to CapabilityHandler.
		v1.GET("/agents/:id/capabilities", capabilities.ListAgentCapabilities)
		v1.GET("/capabilities", capabilities.ListCatalogCapabilities)

		v1.GET("/projects/:id/tasks", tasks.List)
		v1.GET("/tasks/:id", tasks.Get)
		v1.GET("/tasks/:id/history", assignments.ListHistory)

		v1.GET("/executions", executions.List)
		v1.GET("/executions/:id", executions.GetByID)

		v1.GET("/deliverables", deliverables.List)
		v1.GET("/deliverables/:id", deliverables.Get)
		v1.GET("/deliverables/:id/versions", deliverables.ListVersions)

		v1.GET("/code/:projectId/files/*path", code.GetFile)

		v1.GET("/reviews/:id", reviews.Get)
		v1.GET("/deployments/:id", deployments.GetStatus)

		v1.GET("/users/me", users.GetProfile)
	}

	return r
}
