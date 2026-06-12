# Technical Specification: Redis-based Refresh Token Store

## Overview
To support secure session management and immediate refresh token revocation, we are implementing a persistent store for refresh tokens using Redis. This resolves the current gap where tokens exist only in memory without revocation capabilities.

## Redis Data Structure
We will use a key-value pattern for efficient lookup and revocation.

### Key Format
`auth:refresh_token:<token_hash>`

- `<token_hash>`: SHA-256 hash of the raw refresh token string.

### Value Format (JSON)
```json
{
  "user_id": "uuid-v4-string",
  "expires_at": "rfc3339-timestamp"
}
```

## TTL (Time-To-Live) Strategy
- Upon storage, the Redis key will be set with an `EXPIRE` command.
- TTL value = `Refresh Token Lifetime` (e.g., 7 days).
- This ensures tokens are automatically cleaned up when they expire.

## Security Practices
1. **Hashing:** Raw refresh tokens are NEVER stored in Redis. Only their SHA-256 hash is stored.
2. **Revocation:** When a user logs out or is suspended, the specific token hash key is `DEL`eted from Redis.
3. **Atomicity:** The `SET` and `EXPIRE` operations should ideally be executed together (e.g., `SET key value EX 604800`) to prevent orphaned keys if a failure occurs.

## Operational Procedures

### 1. Store Refresh Token
- Generate raw token.
- Calculate `hash = sha256(raw_token)`.
- Store key: `SET auth:refresh_token:<hash> <json_value> EX 604800`.

### 2. Validate Refresh Token
- Receive raw token from client.
- Calculate `hash = sha256(raw_token)`.
- Retrieve: `GET auth:refresh_token:<hash>`.
- If key exists:
  - Parse JSON.
  - Check `expires_at`.
  - Proceed with authentication.
- If key does not exist:
  - Reject authentication (token invalid or revoked).

### 3. Revoke Refresh Token
- Receive raw token from client.
- Calculate `hash = sha256(raw_token)`.
- `DEL auth:refresh_token:<hash>`.
