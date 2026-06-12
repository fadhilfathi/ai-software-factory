# AI Software Factory Backend

This is the backend component of the [AI Software Factory](..), a multi-agent platform for autonomous software development. Built with **Go 1.25+** and the **Gin** framework.

## Project Structure

```
src/
├── cmd/
│   └── main.go           # Application entry point (HTTP API server)
├── internal/             # Private application packages
│   ├── config/           # Configuration management
│   ├── handler/          # Gin handlers (one file per resource)
│   ├── logger/           # Structured logging setup (zap)
│   ├── middleware/       # Gin middleware (Auth, CORS, Recovery, Logger)
│   └── router/           # Gin router definition + route mapping
└── pkg/                  # Public packages
    └── errors/           # Custom error types
```

## Getting Started

1. Ensure prerequisites (Go 1.25+, PostgreSQL, Redis) are met.
2. Build the API:
   ```bash
   go build -o bin/api ./cmd/main.go
   ```
3. Run the server:
   ```bash
   ./bin/api
   ```

## Documentation

- [System Architecture](../docs/architecture.md)
- [Developer Guide](../docs/developer-guide.md)
- [API Specification](../docs/api-spec.md)

