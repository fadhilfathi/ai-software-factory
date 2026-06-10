package router

import (
	"net/http"
	"strings"

	"github.com/example/project/internal/handler"
	"github.com/example/project/internal/middleware"
	"github.com/example/project/internal/service"
)

// publicRoutes returns true for paths that do not require authentication.
func publicRoutes(r *http.Request) bool {
	public := map[string]bool{
		"GET /v1/healthz":          true,
		"POST /v1/auth/login":      true,
		"POST /v1/users/register":  true,
	}
	key := r.Method + " " + r.URL.Path
	return public[key] || strings.HasPrefix(r.URL.Path, "/v1/code/") // code file reads are public for now
}

// New builds and returns the configured HTTP handler with all routes registered
// under the /v1 prefix as specified in the API spec.
func New(svc *service.Services) http.Handler {
	mux := http.NewServeMux()

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

	// --- Health ---
	mux.HandleFunc("GET /v1/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// --- Auth ---
	mux.HandleFunc("POST /v1/auth/login", auth.Login)

	// --- Projects ---
	mux.HandleFunc("POST /v1/projects", projects.Create)
	mux.HandleFunc("GET /v1/projects", projects.List)
	mux.HandleFunc("GET /v1/projects/{id}", projects.Get)

	// --- Agents ---
	mux.HandleFunc("POST /v1/agents/spawn", agents.Spawn)
	mux.HandleFunc("GET /v1/agents", agents.List)
	mux.HandleFunc("POST /v1/agents/{id}/assign", agents.AssignTask)

	// --- Tasks ---
	mux.HandleFunc("POST /v1/projects/{projectId}/tasks", tasks.Create)
	mux.HandleFunc("PATCH /v1/tasks/{id}", tasks.UpdateStatus)

	// --- Code ---
	mux.HandleFunc("POST /v1/code/generate", code.Generate)
	mux.HandleFunc("GET /v1/code/{projectId}/files/{path...}", code.GetFile)
	mux.HandleFunc("POST /v1/code/{projectId}/commits", code.CreateCommit)

	// --- Reviews ---
	mux.HandleFunc("POST /v1/reviews", reviews.Create)
	mux.HandleFunc("GET /v1/reviews/{id}", reviews.Get)

	// --- Deployments ---
	mux.HandleFunc("POST /v1/deployments", deployments.Trigger)
	mux.HandleFunc("GET /v1/deployments/{id}", deployments.GetStatus)
	mux.HandleFunc("POST /v1/deployments/{id}/rollback", deployments.Rollback)

	// --- Users ---
	mux.HandleFunc("POST /v1/users/register", users.Register)
	mux.HandleFunc("GET /v1/users/me", users.GetProfile)

	// --- Webhooks ---
	mux.HandleFunc("POST /v1/webhooks", webhooks.Register)

	// --- Middleware chain (outer wraps inner) ---
	var h http.Handler = mux
	h = middleware.CORS(h)
	h = middleware.RequestID(h)
	h = middleware.Recovery(h)
	h = middleware.Logger(h)
	h = middleware.Auth(publicRoutes)(h)

	return h
}
