package router

import (
	"net/http"

	"github.com/fadhilfathi/AI-Software-Factory/internal/handler"
	"github.com/fadhilfathi/AI-Software-Factory/internal/middleware"
	"github.com/fadhilfathi/AI-Software-Factory/internal/service"
	"github.com/gin-gonic/gin"
)

// publicRoutes returns true for paths that do not require authentication.
func publicRoutes(c *gin.Context) bool {
	public := map[string]bool{
		"GET /v1/healthz":         true,
		"POST /v1/auth/login":     true,
		"POST /v1/auth/refresh":   true,
		"POST /v1/users/register": true,
	}
	key := c.Request.Method + " " + c.FullPath()
	return public[key]
}

// New builds and returns the configured Gin engine with all routes registered
// under the /v1 prefix as specified in the API spec.
func New(svc *service.Services, corsConfig middleware.CORSConfig, rateLimitConfig middleware.RateLimitConfig) *gin.Engine {
	r := gin.New()

	// --- Global Middleware ---
	r.Use(middleware.Recovery())
	r.Use(middleware.Logger())
	r.Use(middleware.RequestID())
	r.Use(middleware.CORS(corsConfig))
	r.Use(middleware.RateLimit(rateLimitConfig))
	r.Use(middleware.Auth(svc.Auth, publicRoutes))

	// --- Handlers ---
	auth := handler.NewAuthHandler(svc.Auth)
	projects := handler.NewProjectHandler(svc.Project)
	agents := handler.NewAgentHandler(svc.Agent)
	tasks := handler.NewTaskHandler(svc.Task)
	code := handler.NewCodeHandler(svc.Code)
	reviews := handler.NewReviewHandler(svc.Review)
	deployments := handler.NewDeploymentHandler(svc.Deployment)
	users := handler.NewUserHandler(svc.User)
	webhooks := handler.NewWebhookHandler(svc.Webhook)

	v1 := r.Group("/v1")
	{
		// --- Health ---
		v1.GET("/healthz", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		// --- Auth ---
		v1.POST("/auth/login", auth.Login)
		v1.POST("/auth/refresh", auth.Refresh)

		// --- Projects ---
		v1.POST("/projects", projects.Create)
		v1.GET("/projects", projects.List)
		v1.GET("/projects/:id", projects.Get)

		// --- Agents ---
		v1.POST("/agents/spawn", agents.Spawn)
		v1.GET("/agents", agents.List)
		v1.POST("/agents/:id/assign", agents.AssignTask)

		// --- Tasks ---
		v1.POST("/projects/:projectId/tasks", tasks.Create)
		v1.PATCH("/tasks/:id", tasks.UpdateStatus)

		// --- Code ---
		v1.POST("/code/generate", code.Generate)
		v1.GET("/code/:projectId/files/*path", code.GetFile)
		v1.POST("/code/:projectId/commits", code.CreateCommit)

		// --- Reviews ---
		v1.POST("/reviews", reviews.Create)
		v1.GET("/reviews/:id", reviews.Get)

		// --- Deployments ---
		v1.POST("/deployments", deployments.Trigger)
		v1.GET("/deployments/:id", deployments.GetStatus)
		v1.POST("/deployments/:id/rollback", deployments.Rollback)

		// --- Users ---
		v1.POST("/users/register", users.Register)
		v1.GET("/users/me", users.GetProfile)

		// --- Webhooks ---
		v1.POST("/webhooks", webhooks.Register)
	}

	return r
}
