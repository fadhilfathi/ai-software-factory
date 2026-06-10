# Project Structure

```
src/
├── go.mod                 # Go module definition
├── cmd/
│   └── main.go           # Application entry point (HTTP API server)
├── internal/             # Private application packages
│   ├── config/           # Configuration management
│   │   └── config.go
│   ├── handler/          # HTTP request handlers (one file per resource)
│   │   ├── types.go      # Shared response types (APIResponse, ErrorResponse, Pagination)
│   │   ├── auth.go       # POST /v1/auth/login
│   │   ├── project.go    # CRUD /v1/projects
│   │   ├── agent.go      # Spawn, list, assign /v1/agents
│   │   ├── task.go       # Create, update /v1/tasks
│   │   ├── code.go       # Generate, files, commits /v1/code
│   │   ├── review.go     # Request, results /v1/reviews
│   │   ├── deployment.go # Trigger, status, rollback /v1/deployments
│   │   ├── user.go       # Register, profile /v1/users
│   │   └── webhook.go    # Register /v1/webhooks
│   ├── logger/           # Structured logging setup using zap
│   │   └── logger.go
│   ├── middleware/       # HTTP middleware chain
│   │   └── middleware.go # CORS, RequestID, Recovery, Logger, Auth (JWT + API Key)
│   └── router/           # Route mapping + middleware wiring
│       └── router.go
└── pkg/                  # Public packages
    └── errors/           # Custom error types with wrapping support
        └── errors.go
```

## Module Organization

- **module path**: `github.com/example/project`
- **Go version**: 1.22
- **Dependencies**: `go.uber.org/zap` (structured logging)

## Package Layout

### `cmd/`
HTTP API server. Reads config, initializes logger, wires the router, and starts listening.

### `internal/handler/`
One file per API resource. Each handler is a struct with methods matching HTTP verbs:
- Validates input, returns structured `ErrorResponse` for bad requests
- Returns spec-compliant JSON responses (standard envelope + pagination)
- All routes are under the `/v1` prefix as specified by the API spec

### `internal/middleware/`
Middleware chain (outer→inner): **CORS → RequestID → Recovery → Logger → Auth**
- `CORS` — permissive headers for development
- `RequestID` — attaches X-Request-ID to every response
- `Recovery` — catches panics and returns 500
- `Logger` — logs method, path, remote addr, and duration
- `Auth` — checks JWT (`Bearer eyJ...`) or API Key (`Bearer ak_...`) on all routes except public ones (healthz, login, register)

### `internal/router/`
Registers all routes with Go 1.22's enhanced `http.ServeMux` pattern matching (method + path with `{param}` and `{param...}` wildcards). Wraps the mux with the full middleware chain.

## API Endpoints

| Method | Path | Handler | Auth |
|--------|------|---------|------|
| GET | /v1/healthz | inline | No |
| POST | /v1/auth/login | `AuthHandler.Login` | No |
| POST | /v1/users/register | `UserHandler.Register` | No |
| POST | /v1/projects | `ProjectHandler.Create` | Yes |
| GET | /v1/projects | `ProjectHandler.List` | Yes |
| GET | /v1/projects/{id} | `ProjectHandler.Get` | Yes |
| POST | /v1/agents/spawn | `AgentHandler.Spawn` | Yes |
| GET | /v1/agents | `AgentHandler.List` | Yes |
| POST | /v1/agents/{id}/assign | `AgentHandler.AssignTask` | Yes |
| POST | /v1/projects/{projectId}/tasks | `TaskHandler.Create` | Yes |
| PATCH | /v1/tasks/{id} | `TaskHandler.UpdateStatus` | Yes |
| POST | /v1/code/generate | `CodeHandler.Generate` | Yes |
| GET | /v1/code/{projectId}/files/{path...} | `CodeHandler.GetFile` | Yes |
| POST | /v1/code/{projectId}/commits | `CodeHandler.CreateCommit` | Yes |
| POST | /v1/reviews | `ReviewHandler.Create` | Yes |
| GET | /v1/reviews/{id} | `ReviewHandler.Get` | Yes |
| POST | /v1/deployments | `DeploymentHandler.Trigger` | Yes |
| GET | /v1/deployments/{id} | `DeploymentHandler.GetStatus` | Yes |
| POST | /v1/deployments/{id}/rollback | `DeploymentHandler.Rollback` | Yes |
| GET | /v1/users/me | `UserHandler.GetProfile` | Yes |
| POST | /v1/webhooks | `WebhookHandler.Register` | Yes |

Full request/response schema details: see `docs/api-spec.md`.

## Usage

```bash
# Build the API server
cd src/
go build -o bin/project ./cmd/main.go

# Run the server (default: localhost:8080)
./bin/project

# Or override port
PORT=3001 ./bin/project

# Run with a custom log level
LOG_LEVEL=debug ./bin/project
```
