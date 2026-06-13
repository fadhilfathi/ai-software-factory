package router

import (
	"net/http"

	"github.com/fadhilfathi/AI-Software-Factory/internal/handler"
	"github.com/fadhilfathi/AI-Software-Factory/internal/middleware"
	"github.com/fadhilfathi/AI-Software-Factory/internal/service"
	"github.com/gin-gonic/gin"
)

// publicRouteSet is the set of (method, path-prefix) pairs that bypass auth.
// Sprint 4: Auth middleware takes map[string]bool keyed by method+path.
// Matches the convention used in other services in the repo.
var publicRouteSet = map[string]bool{
	"GET /v1/healthz":         true,
	"HEAD /v1/healthz":        true,
	"POST /v1/auth/login":     true,
	"POST /v1/auth/refresh":   true,
	"POST /v1/users/register": true,
}

// healthzHandler responds to GET and HEAD on /v1/healthz.
// Sprint 5: HEAD support added so Docker / wget --spider healthchecks pass.
func healthzHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func New(svc *service.Services, corsConfig middleware.CORSConfig, rateLimitConfig middleware.RateLimitConfig) *gin.Engine {
	r := gin.New()

	r.Use(middleware.Recovery())
	r.Use(middleware.Logger())
	r.Use(middleware.RequestID())
	r.Use(middleware.CORS(corsConfig))
	r.Use(middleware.RateLimit(rateLimitConfig))
	r.Use(middleware.Auth(svc.Auth, publicRouteSet))

	auth := handler.NewAuthHandler(svc.Auth)
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
		v1.POST("/auth/logout", auth.Logout)

		v1.POST("/projects", projects.Create)
		v1.GET("/projects", projects.List)
		v1.GET("/projects/:id", projects.Get)
		v1.PUT("/projects/:id", projects.Update)
		v1.DELETE("/projects/:id", projects.Delete)
		v1.POST("/projects/:id/decompose", projects.Decompose)

		v1.POST("/agents", agents.Create)
		v1.GET("/agents", agents.List)
		v1.GET("/agents/:id", agents.Get)
		v1.PUT("/agents/:id", agents.Update)
		v1.DELETE("/agents/:id", agents.Delete)
		// TASK-403: capability routes moved to CapabilityHandler.
		v1.GET("/agents/:id/capabilities", capabilities.ListAgentCapabilities)
		v1.GET("/capabilities", capabilities.ListCatalogCapabilities)

		v1.POST("/projects/:id/tasks", tasks.Create)
		v1.GET("/projects/:id/tasks", tasks.List)
		v1.GET("/tasks/:id", tasks.Get)
		v1.PUT("/tasks/:id", tasks.Update)
		v1.DELETE("/tasks/:id", tasks.Delete)
		// TASK-404: assignment endpoints. POST creates/updates the
		// assignment + appends to assignment_events. GET returns
		// the history DESC by assigned_at.
		v1.POST("/tasks/:id/assign", assignments.AssignTask)
		v1.GET("/tasks/:id/history", assignments.ListHistory)
		v1.PATCH("/tasks/:id/status", tasks.UpdateStatus)

		// TASK-405: Sprint 4 Execution Tracking System.
		// PATCH is mounted on the resource itself (per
		// api-spec.md §5) rather than on a /status sub-path.
		v1.POST("/executions", executions.Create)
		v1.GET("/executions", executions.List)
		v1.GET("/executions/:id", executions.GetByID)
		v1.PATCH("/executions/:id", executions.Patch)

		// TASK-406: Sprint 4 Deliverable Storage. 5 routes:
		// POST/GET list/GET single/PUT on the resource itself,
		// plus GET versions on the history sub-resource.
		v1.POST("/deliverables", deliverables.Create)
		v1.GET("/deliverables", deliverables.List)
		v1.GET("/deliverables/:id", deliverables.Get)
		v1.PUT("/deliverables/:id", deliverables.Update)
		v1.GET("/deliverables/:id/versions", deliverables.ListVersions)

		v1.POST("/code/generate", code.Generate)
		v1.GET("/code/:projectId/files/*path", code.GetFile)
		v1.POST("/code/:projectId/commits", code.CreateCommit)

		v1.POST("/reviews", reviews.Create)
		v1.GET("/reviews/:id", reviews.Get)

		v1.POST("/deployments", deployments.Trigger)
		v1.GET("/deployments/:id", deployments.GetStatus)
		v1.POST("/deployments/:id/rollback", deployments.Rollback)

		v1.POST("/users/register", users.Register)
		v1.GET("/users/me", users.GetProfile)

		v1.POST("/webhooks", webhooks.Register)
	}

	return r
}
