# AI Software Factory — Threat Model

**Document Version:** 1.0
**Date:** 2026-06-10
**Status:** Draft
**Author:** Security Agent
**Classification:** Internal — Security Sensitive

---

## 1. Executive Summary

This threat model identifies security risks, attack vectors, and mitigations for the AI Software Factory platform. The analysis covers the multi-agent orchestration system, microservice architecture, and data flows as defined in the functional requirements, non-functional requirements, service architecture, and agent workflow design documents.

### Scope

**In Scope:**
- All 9 microservices (API Gateway, Project Service, Agent Orchestrator, Code Service, Review Service, QA Service, Deploy Service, Notification Service, User Service)
- Agent Orchestrator and 6 agent types (PM, Architect, Developer, Review, QA, DevOps)
- Inter-service communication (gRPC, NATS messaging)
- External integrations (Git providers, CI/CD, Cloud providers, Communication platforms)
- User authentication and authorization flows
- Data storage and processing

**Out of Scope:**
- Underlying cloud infrastructure (assumed hardened per NFR-011, NFR-014)
- Third-party SaaS security (GitHub, Slack, etc. — covered by NFR-040)
- Physical security of data centers

### Risk Classification

| Level | Definition | Response Timeline |
|-------|------------|-------------------|
| **Critical** | Immediate exploit possible, high impact | Fix before deployment |
| **High** | Exploitable with moderate effort, significant impact | Fix within 1 sprint |
| **Medium** | Exploitable under specific conditions, moderate impact | Fix within 2 sprints |
| **Low** | Theoretical or low-impact vectors | Track, fix when convenient |
|| **Informational** | Hardening opportunities, defense-in-depth | Document for future |

---

## 2. Asset Inventory

### 2.1 Data Assets

| Asset ID | Asset | Classification | Description |
|----------|-------|----------------|-------------|
| D-001 | Source Code Repositories | Confidential | User project code, commit history, branches |
| D-002 | Build Artifacts | Confidential | Compiled binaries, container images, deployment packages |
| D-003 | Agent Execution Logs | Internal | LLM prompts, responses, tool calls, reasoning traces |
| D-004 | User Credentials & Tokens | Restricted | OAuth tokens, API keys, SSH keys, PATs for external integrations |
| D-005 | Project Configuration | Confidential | Build configs, deployment manifests, environment variables, secrets |
| D-006 | Audit & Compliance Logs | Restricted | Security events, access logs, audit trails |
| D-007 | Agent State & Memory | Internal | Conversation history, learned patterns, project context |
| D-008 | Deployment Infrastructure State | Confidential | Kubernetes manifests, Terraform state, cloud resource configs |

### 2.2 System Assets

| Asset ID | Asset | Type | Description |
|----------|-------|------|-------------|
| S-001 | API Gateway | Service | Entry point, auth termination, rate limiting, routing |
| S-002 | Agent Orchestrator | Service | Task scheduling, agent lifecycle, state management |
| S-003 | Code Service | Service | Code generation, modification, analysis, sandbox execution |
| S-004 | Review Service | Service | Automated code review, security scanning, quality gates |
| S-005 | QA Service | Service | Test generation, execution, result aggregation |
| S-006 | Deploy Service | Service | CI/CD pipeline orchestration, infrastructure provisioning |
| S-007 | Notification Service | Service | Multi-channel alerting (Slack, Email, Webhook, Teams) |
| S-008 | User Service | Service | Authentication, authorization, user management |
| S-009 | Project Service | Service | Project CRUD, metadata, settings, webhooks |
| S-010 | NATS Message Bus | Infrastructure | Inter-service async communication, event streaming |
| S-011 | PostgreSQL Cluster | Database | Primary relational data store |
| S-012 | Redis Cluster | Cache/Queue | Session store, rate limiting, task queues |
| S-013 | Object Storage (S3-compatible) | Storage | Artifacts, logs, large binaries |
| S-014 | Vector Database | Database | Embeddings for code search, RAG context |
| S-015 | LLM Provider Endpoints | External | OpenAI, Anthropic, local models (Ollama, vLLM) |

---

## 3. Trust Boundaries

| Boundary ID | Name | Description | Controls |
|-------------|------|-------------|----------|
| TB-01 | **Internet → API Gateway** | Public internet to platform entry point | WAF, TLS 1.3, Rate limiting, DDoS protection |
| TB-02 | **API Gateway → Internal Services** | Authenticated requests to microservices | mTLS, JWT validation, Service mesh |
| TB-03 | **Agent Orchestrator → Agent Workers** | Orchestrator spawns/controls agent processes | Process isolation, Capability-based sandbox, Resource limits |
| TB-04 | **Services → External Integrations** | Outbound calls to Git, CI/CD, Cloud, Comms | OAuth 2.0, Short-lived tokens, Secret rotation |
| TB-05 | **Services → Data Stores** | Database, Cache, Object storage, Vector DB | Network policies, Encryption at rest, IAM roles |
| TB-06 | **Agent Workers → LLM Providers** | Prompts/responses to external model APIs | Request sanitization, PII redaction, Token budgets |
| TB-07 | **User → API Gateway** | Human operators and CI/CD systems | OIDC/OAuth2, MFA, Device trust, Session management |
| TB-08 | **Inter-Service (gRPC/NATS)** | East-west traffic between microservices | mTLS, SPIFFE/SPIRE, Authorization policies |
| TB-09 | **Deploy Service → Cloud Providers** | Infrastructure provisioning and deployment | IRSA/Workload Identity, Least-privilege roles, Policy-as-code |
|| TB-10 | **Code Service Sandbox → Host** | Code execution isolation | gVisor/Firecracker, Seccomp, No network, Read-only FS |

---

## 4. STRIDE Threat Enumeration

### 4.1 Spoofing

| Threat ID | Asset / Boundary | Description | Likelihood | Impact | Risk |
|-----------|------------------|-------------|------------|--------|------|
| T-SP-001 | TB-01, TB-07 | Attacker impersonates legitimate user via stolen credentials (phishing, credential stuffing) | High | Critical | **Critical** |
| T-SP-002 | TB-02, TB-08 | Service-to-service impersonation via stolen mTLS cert or SPIFFE identity | Medium | High | **High** |
| T-SP-003 | TB-04 | Attacker impersonates Git provider webhook to inject malicious payloads | Medium | High | **High** |
| T-SP-004 | TB-06 | Malicious agent worker spoofs orchestrator to exfiltrate prompts/responses | Low | Medium | **Medium** |
| T-SP-005 | TB-09 | Compromised deploy service impersonates cloud provider API to escalate privileges | Low | Critical | **High** |

### 4.2 Tampering

| Threat ID | Asset / Boundary | Description | Likelihood | Impact | Risk |
|-----------|------------------|-------------|------------|--------|------|
| T-TA-001 | D-001, TB-02 | Malicious code injection via compromised agent (prompt injection → malicious PR) | High | Critical | **Critical** |
| T-TA-002 | D-005, S-006 | Deployment manifest tampering (supply chain attack via malicious Terraform/Helm) | Medium | Critical | **Critical** |
| T-TA-003 | D-003, S-002 | Agent execution log tampering to hide malicious activity | Medium | High | **High** |
| T-TA-004 | S-010, TB-08 | NATS message tampering (replay, reorder, injection) | Low | High | **Medium** |
| T-TA-005 | D-002, S-013 | Build artifact tampering (binary injection, container image poisoning) | Medium | Critical | **Critical** |
| T-TA-006 | D-007, TB-03 | Agent memory/state poisoning (context injection, long-term persistence) | Medium | High | **High** |

### 4.3 Repudiation

| Threat ID | Asset / Boundary | Description | Likelihood | Impact | Risk |
|-----------|------------------|-------------|------------|--------|------|
| T-RE-001 | D-006, S-008 | Admin actions not audited (user creation, role changes, secret rotation) | Medium | High | **High** |
| T-RE-002 | S-002, TB-03 | Agent task execution not attributable (which agent did what, when) | High | Medium | **High** |
| T-RE-003 | D-004, TB-04 | External integration actions not logged (who deployed, what Git operations) | Medium | Medium | **Medium** |
| T-RE-004 | D-006, S-011 | Audit log tampering/deletion to cover tracks | Low | Critical | **High** |

### 4.4 Information Disclosure

| Threat ID | Asset / Boundary | Description | Likelihood | Impact | Risk |
|-----------|------------------|-------------|------------|--------|------|
| T-ID-001 | D-004, S-008, S-012 | Secret leakage via logs, error messages, debug endpoints | High | Critical | **Critical** |
| T-ID-002 | D-001, TB-06 | Source code / PII sent to external LLM providers without sanitization | High | High | **High** |
| T-ID-003 | D-003, S-002 | Agent reasoning traces expose internal logic, prompts, tool calls | Medium | Medium | **Medium** |
| T-ID-004 | S-011, S-012, TB-05 | Database/cache misconfiguration exposes data (public S3, open Redis) | Medium | High | **High** |
| T-ID-005 | D-008, TB-09 | Terraform state / kubeconfig leakage via deploy service logs | Medium | High | **High** |
| T-ID-006 | S-014, TB-05 | Vector DB embeddings leak semantic content of private code | Low | Medium | **Medium** |

### 4.5 Denial of Service

| Threat ID | Asset / Boundary | Description | Likelihood | Impact | Risk |
|-----------|------------------|-------------|------------|--------|------|
| T-DS-001 | S-001, TB-01 | API Gateway overwhelmed by volumetric DDoS | High | High | **High** |
| T-DS-002 | S-002, TB-03 | Agent orchestrator resource exhaustion (unbounded agent spawns, token budgets) | High | High | **High** |
| T-DS-003 | S-003, TB-03 | Code service sandbox escape → host DoS (fork bomb, memory exhaustion) | Medium | Critical | **High** |
| T-DS-004 | S-010, TB-08 | NATS jetstream saturation (unacked messages, consumer lag) | Medium | Medium | **Medium** |
| T-DS-005 | S-011, S-012, TB-05 | Database connection pool exhaustion, Redis OOM | Medium | High | **High** |
| T-DS-006 | TB-06 | LLM provider rate limits / quota exhaustion blocks all agents | High | Medium | **High** |

### 4.6 Elevation of Privilege

| Threat ID | Asset / Boundary | Description | Likelihood | Impact | Risk |
|-----------|------------------|-------------|------------|--------|------|
| T-EP-001 | TB-03, S-003 | Sandbox escape from code execution → host/container breakout | Medium | Critical | **Critical** |
| T-EP-002 | TB-02, TB-08 | Service mesh bypass / mTLS cert compromise → lateral movement | Low | Critical | **High** |
| T-EP-003 | D-004, S-008 | Token theft → impersonate service account with elevated scopes | Medium | Critical | **Critical** |
| T-EP-004 | TB-09, S-006 | Deploy service cloud credentials over-privileged → full account takeover | Medium | Critical | **Critical** |
| T-EP-005 | TB-04, S-007 | Webhook secret compromise → impersonate notification service → SSRF | Low | High | **Medium** |
| T-EP-006 | S-002, TB-03 | Prompt injection in PM/Architect agent → privilege escalation via task delegation | High | High | **High** |