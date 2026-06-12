package router

import (
	"net/http"

	"github.com/fadhilfathi/AI-Software-Factory/internal/handler"
	"github.com/fadhilfathi/AI-Software-Factory/internal/middleware"
	"github.com/fadhilfathi/AI-Software-Factory/internal/service"
	"github.com/gin-gonic/gin"
)

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

func New(svc *service.Services, corsConfig middleware.CORSConfig, rateLimitConfig middleware.RateLimitConfig) *gin.Engine {
	r := gin.New()

	r.Use(middleware.Recovery())
	r.Use(middleware.Logger())
	r.Use(middleware.RequestID())
	r.Use(middleware.CORS(corsConfig))
	r.Use(middleware.RateLimit(rateLimitConfig))
	r.Use(middleware.Auth(svc.Auth, publicRoutes))

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
		v1.GET("/healthz", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		v1.POST("/auth/login", auth.Login)
		v1.POST("/auth/refresh", auth.Refresh)

		v1.POST("/projects", projects.Create)
		v1.GET("/projects", projects.List)
		v1.GET("/projects/:id", projects.Get)
		v1.PUT("/projects/:id", projects.Update)
		v1.DELETE("/projects/:id", projects.Delete)

		v1.POST("/agents/spawn", agents.Spawn)
		v1.GET("/agents", agents.List)
		v1.POST("/agents/:id/assign", agents.AssignTask)

		v1.POST("/projects/:projectId/tasks", tasks.Create)
		v1.GET("/projects/:projectId/tasks", tasks.List)
		v1.GET("/tasks/:id", tasks.Get)
		v1.PUT("/tasks/:id", tasks.Update)
		v1.DELETE("/tasks/:id", tasks.Delete)
		v1.PATCH("/tasks/:id/status", tasks.UpdateStatus)

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
