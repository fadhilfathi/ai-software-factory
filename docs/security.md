# AI Software Factory — Security Architecture

## 1. Introduction

Security is a foundational pillar of the AI Software Factory. This document formalizes the platform's security architecture, detailing the controls, patterns, and technologies used to protect user data, agent execution environments, and system integrity.

The architecture follows the **Zero Trust** principle: no entity (user, service, or agent) is trusted by default, regardless of their location on the network.

---

## 2. Identity & Access Management (IAM)

### 2.1 Authentication

The platform supports two primary authentication mechanisms, managed by the `User Service` and the `AuthService` interface.

- **JWT Sessions (Interactive):** Users authenticate via `POST /auth/login` to receive a short-lived (15m) JWT Access Token and a long-lived (7d) Refresh Token.
  - **Access Tokens:** Statistically validated by the `Auth` middleware using `HS256`.
  - **Refresh Tokens:** Stored in `HttpOnly`, `Secure`, `SameSite=Strict` cookies. Validated against a persistent store for revocation support.
- **API Keys (Non-interactive):** Service accounts and CLI tools use static API keys with the `ak_` prefix.
  - Stored as SHA-256 hashes in the database.
  - Scoped to specific projects or organizations.

### 2.2 Authorization (RBAC)

Authorization is enforced at the service boundary using Role-Based Access Control (RBAC).

- **Middleware:** `RequireRole(role string)` is used to protect sensitive routes.
- **Roles:**
  - `admin`: Full system access.
  - `user`: Access to own projects and teams.
  - `viewer`: Read-only access to assigned projects.
  - `api_user`: Scoped access for API keys.
- **Verification:** The `Auth` middleware verifies user existence and "active" status in the database on **every** request to allow for immediate account suspension.

---

## 3. Network Security

### 3.1 Transport Security (TLS)
- **External:** All external traffic is mandatory over **TLS 1.3**.
- **Internal:** Service-to-service communication via gRPC uses **mTLS** (Mutual TLS) to ensure both encryption and service identity verification.

### 3.2 Edge Protection
- **CORS:** Configurable Cross-Origin Resource Sharing (CORS) policies restrict frontend access to approved domains.
- **Rate Limiting:** Enforced per-IP and per-User to prevent DDoS and brute-force attacks.
  - Default: 100 requests/minute with a burst of 20.
  - Headers: `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset`.

---

## 4. Session Management & Revocation

### 4.1 Refresh Token Revocation
To prevent "forever sessions" if a token is compromised, the platform implements a **Redis-backed revocation store**.

- **Flow:**
  1. On login, the SHA-256 hash of the refresh token is stored in Redis with a TTL matching the token expiry.
  2. On every `/auth/refresh` call, the store is checked.
  3. On logout or password change, the hash is deleted from Redis, immediately invalidating the session.
- **Hashing:** Raw refresh tokens are never stored in Redis, only their cryptographic hashes.

For technical details, see [Redis Refresh Store Spec](./redis-refresh-store-spec.md).

---

## 5. Data Protection

### 5.1 Encryption
- **At Rest:** All sensitive data in PostgreSQL (e.g., secrets, PII) and artifacts in Object Storage are encrypted using **AES-256**.
- **Secrets:** Environment-specific secrets (e.g., LLM API keys) are managed via a secure Vault or Cloud KMS and injected into services at runtime.

### 5.2 PII Redaction & Logging
- **Log Masking:** Emails and other PII are hashed (using `sha256(email)[:8]`) before being written to structured logs (`zap`).
- **Panic Recovery:** The `Recovery` middleware catches panics to prevent stack traces (which may contain sensitive data) from leaking to clients.

---

## 6. Agent Security

### 6.1 Execution Sandbox (gVisor)
Agents execute code in a highly isolated environment to prevent sandbox escapes and lateral movement. The platform utilizes **gVisor (`runsc`)** as the primary container runtime for untrusted code execution.

| Control | Implementation | Purpose |
|---------|----------------|---------|
| **Runtime Isolation** | gVisor (`runsc`) | Intercepts syscalls to prevent kernel exploits |
| **Resource Limits** | Memory/CPU Cgroups | Prevents Denial-of-Service via resource exhaustion |
| **Read-only FS** | `ReadonlyRootfs: true` | Prevents environment persistence and tampering |
| **Capability Drop** | `CapDrop: ["ALL"]` | Removes all root-like privileges within the container |
| **Network Isolation** | `NetworkMode: "none"` | Prevents data exfiltration and lateral attacks |
| **Privilege Control** | `no-new-privileges` | Prevents processes from gaining new privileges |

### 6.2 Resource Quotas
- **Token Budgets:** LLM usage is capped per agent run to prevent cost-based DoS.
- **Runtime Timeouts:** Executions are wrapped in Go `context.WithTimeout` (default 5m) to prevent hanging processes.

---

## 7. Auditing & Compliance

### 7.1 Audit Logging
All security-sensitive actions are recorded in an immutable audit log:
- User login/logout.
- Project creation/deletion.
- Role changes and permission updates.
- Agent spawn and code generation events.

### 7.2 Vulnerability Management
- **CI/CD Scanning:** Every Pull Request is scanned using SAST (Static Analysis Security Testing) and SCA (Software Composition Analysis) for vulnerabilities in dependencies.
- **Dependency Updates:** Automated tools ensure Go and Node.js packages are kept up to date with security patches.

---

> **Document Version:** 1.0 | **Last Updated:** 2026-06-12
>
> For authentication implementation details, see [Auth Design](./auth-design.md).
