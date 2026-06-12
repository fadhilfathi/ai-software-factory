# Repository Structure (MVP)

This document outlines the directory structure for the AI Software Factory project, optimized for a Go/Gin backend and Next.js frontend.

## Root Structure

```text
/
├── .github/           # GitHub Actions (CI/CD)
├── docs/              # Project documentation
├── frontend/          # Next.js/TypeScript frontend
│   └── src/           # Frontend source code
├── src/               # Go/Gin backend source code
│   ├── cmd/           # Application entry point
│   ├── db/            # Database migrations and schema
│   └── internal/      # Private application code
└── scripts/           # Build and deployment scripts
```

## Backend Structure (`src/internal`)

The backend follows idiomatic Go/Gin patterns:

```text
src/internal/
├── config/            # Configuration loading
├── handler/           # Gin handlers (HTTP requests -> service calls)
├── logger/            # Application logging setup
├── middleware/        # Gin middleware (Auth, CORS, etc.)
├── model/             # Data models (structs)
├── router/            # Gin router definition
├── service/           # Business logic layer
├── store/             # Data access layer
└── validation/        # Request/Input validation logic
```

## Implementation Notes

1.  **Gin Migration**: The `router` and `handler` packages are being updated to use `github.com/gin-gonic/gin`.
2.  **Handler Signatures**: All handlers in `handler/` must be refactored to use `func(c *gin.Context)`.
3.  **Middleware**: Middleware in `middleware/` must be compatible with Gin's `gin.HandlerFunc` signature.
