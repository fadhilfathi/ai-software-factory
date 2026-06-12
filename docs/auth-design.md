# Authentication Design: JWT-based Flow

For the high-level security architecture of the platform, including network security, agent isolation, and auditing, see the [Security Architecture](./security.md) document.

## Overview
This document outlines the authentication design for the AI Software Factory backend, implemented in Go using the Gin framework. The system uses JWT for stateless session management with a Redis-backed store for refresh token revocation.

## Authentication Flow

### 1. User Registration / Login
- **Endpoint:** `POST /api/v1/auth/login`
- **Mechanism:**
  - Client sends credentials (username/password) over HTTPS.
  - Backend verifies credentials against the database using `bcrypt` for constant-time comparison.
  - On success, the backend generates a pair of tokens: **Access Token** and **Refresh Token**.
  - **Access Token:** Short-lived (15 minutes) for API access.
  - **Refresh Token:** Long-lived (7 days) for generating new Access Tokens.
- **Storage:** Access token returned in JSON response. Refresh token stored in an `HttpOnly`, `Secure`, `SameSite=Strict` cookie to prevent XSS.

### 2. Protected API Access
- **Endpoint:** Any route under `/api/v1/...` (excluding public paths like `/auth/login`, `/healthz`).
- **Mechanism:**
  - Client includes the Access Token in the `Authorization: Bearer <token>` header.
  - Backend `Auth` middleware validates the token using `AuthService.ValidateToken`.
  - **User Verification:** The middleware verifies that the user still exists and is active in the database on every request.
  - If valid, the user context (`user_id`, `role`) is populated in the Gin context for downstream handlers.

### 3. Role-Based Access Control (RBAC)
- **Middleware:** `RequireRole(requiredRole string)`
- **Mechanism:**
  - Checks the `role` stored in the Gin context (populated during token validation).
  - If the role does not match `requiredRole`, returns `403 Forbidden`.

### 4. Refreshing Tokens
- **Endpoint:** `POST /api/v1/auth/refresh`
- **Mechanism:**
  - Client sends a request to refresh the token. The browser automatically sends the Refresh Token via the `HttpOnly` cookie.
  - Backend validates the Refresh Token (signature, expiration, and audience).
  - **Revocation Check:** The token is checked against the Redis revocation store (see [Redis Refresh Store](./redis-refresh-store.md)).
  - Backend generates a new pair of Access and Refresh tokens.

## Technical Details

### Dependencies
- `github.com/gin-gonic/gin`
- `github.com/golang-jwt/jwt/v5`
- `golang.org/x/crypto/bcrypt`

### AuthService Interface
The `AuthService` is defined as an interface to support testability and multiple store backends.

```go
type AuthService interface {
    Login(req LoginRequest) (*LoginResult, *Error)
    Refresh(refreshToken string) (*LoginResult, *Error)
    ValidateToken(tokenString string) (*Claims, error)
    ValidateRefreshToken(refreshToken string) (uuid.UUID, error)
}
```

### Token Structure (Claims)
```go
type Claims struct {
    UserID string `json:"uid"`
    Role   string `json:"role"`
    jwt.RegisteredClaims
}
```

### Secret Management
- `JWT_ACCESS_SECRET`: Loaded from environment variables (min 32 characters).
- `JWT_REFRESH_SECRET`: Separate secret for Refresh Tokens (currently sharing the same secret but partitioned by audience).

## Security Considerations
1. **HTTPS:** Mandatory for all communications.
2. **Algorithm Validation:** Explicitly enforces `SigningMethodHS256`.
3. **User Status Check:** Verifies user existence on every authenticated request to allow immediate suspension.
4. **Revocation:** Uses a Redis-based store for Refresh Tokens to allow immediate user logout/suspension (see [Redis Refresh Store Spec](./redis-refresh-store-spec.md)).
5. **PII Protection:** Email addresses are hashed before being logged for debugging failed login attempts.
