# TASK-116: Redis Refresh Token Store - Technical Specification

## Overview
To improve authentication security and allow for token revocation, we need to transition from in-memory or insecure refresh token handling to a persistent, TTL-managed store using Redis.

## Design
1.  **Storage Key Structure**: `refresh_token:<user_id>:<token_hash>`
2.  **Value**: `user_id` (used for validation lookup).
3.  **Expiration (TTL)**: Configurable, matching the refresh token's intended lifespan (e.g., 7 days).
4.  **Operations**:
    - `SET(key, value, EX <ttl>)`: On successful login/refresh token generation.
    - `GET(key)`: On refresh token validation.
    - `DEL(key)`: On logout/token revocation.

## Integration
- Update `AuthService` to interface with a Redis client.
- Create a `RedisStore` implementation for the `Store` interface (or a specialized `TokenStore` interface).
