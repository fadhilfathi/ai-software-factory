# Authentication Design: JWT-based Flow

## Overview
This document outlines the authentication design for the AI Software Factory backend, implemented in Go using the Gin framework. It replaces the legacy Node.js/Fastify approach with a modern, secure JWT-based system.

## Authentication Flow

### 1. User Registration / Login
- **Endpoint:** `POST /api/v1/auth/login`
- **Mechanism:**
  - Client sends credentials (username/password) over HTTPS.
  - Backend verifies credentials against the database.
  - On success, the backend generates a pair of tokens: **Access Token** and **Refresh Token**.
  - **Access Token:** Short-lived (e.g., 15 minutes) for API access.
  - **Refresh Token:** Long-lived (e.g., 7 days) for generating new Access Tokens.
- **Storage:** Access token returned in JSON response. Refresh token stored in an `HttpOnly`, `Secure`, `SameSite=Strict` cookie to prevent XSS.

### 2. Protected API Access
- **Endpoint:** Any route under `/api/v1/...`
- **Mechanism:**
  - Client includes the Access Token in the `Authorization: Bearer <token>` header.
  - Backend `authMiddleware` validates the token (signature, expiration, algorithm).
  - If valid, the user context (e.g., `userID`, `role`) is populated in the Gin context for downstream handlers.

### 3. Refreshing Tokens
- **Endpoint:** `POST /api/v1/auth/refresh`
- **Mechanism:**
  - Client sends a request to refresh the token. The browser automatically sends the Refresh Token via the `HttpOnly` cookie.
  - Backend validates the Refresh Token (potentially against a whitelist in Redis for immediate revocation).
  - Backend generates a new Access Token pair.

## Technical Details

### Dependencies
- `github.com/gin-gonic/gin`
- `github.com/golang-jwt/jwt/v5`

### Token Structure (Claims)
```go
type Claims struct {
    UserID string `json:"uid"`
    Role   string `json:"role"`
    jwt.RegisteredClaims
}
```

### Secret Management
- `JWT_ACCESS_SECRET`: Loaded from environment variables (must be strong).
- `JWT_REFRESH_SECRET`: Separate secret for Refresh Tokens.

## Security Considerations
1. **HTTPS:** Mandatory for all communications.
2. **Algorithm Validation:** Explicitly enforce `SigningMethodHS256` or `RS256`.
3. **Short Expiration:** Minimize window of opportunity for stolen Access Tokens.
4. **Revocation:** Use a Redis-based whitelist for Refresh Tokens to allow immediate user logout/suspension.
