# TASK-308: Autonomous Code Security Audit Design

> **Owner**: Security (019eb7a3-fe10-7a32-b147-9740e05efb69) / QA (Support)  
> **Status**: Draft  
> **Date**: 2026-06-12

---

## 1. Objective

Define the security audit strategy for autonomously generated code to prevent vulnerabilities, malicious code injection, and compliance violations.

## 2. Audit Layers

### Layer 1: Static Analysis (SAST)
- **Tooling**: `gosec` for Go, `semgrep` for multi-language rules.
- **Focus**:
  - SQL Injection (parameterized queries check).
  - Hardcoded secrets/credentials.
  - Insecure crypto usage.
  - SSRF-prone URL handling.
  - Improper error handling (leaking PII).

### Layer 2: Dependency Scanning (SCA)
- **Tooling**: `npm audit` / `go list -m all`.
- **Focus**:
  - Vulnerable package versions.
  - License compliance (denying GPL-3.0 if restricted).

### Layer 3: Runtime Behavioral Monitoring (Sandbox)
- **Tooling**: `gVisor` syscall filtering, `falco` or `eBPF` for anomaly detection.
- **Focus**:
  - Unexpected network connections.
  - Unauthorized file system access (outside `/workspace`).
  - Resource exhaustion (CPU/Memory DoS).

### Layer 4: AI-Driven Security Review
- **Tooling**: Specialized `security_reviewer` agent.
- **Focus**:
  - Business logic vulnerabilities (e.g., IDOR, privilege escalation).
  - Obfuscated malicious logic.
  - Compliance with project-specific security standards (e.g., "All handlers must use m.RequireRole").

---

## 3. Security Gate Enforcement

The **Review Service** (TASK-302) will enforce the following hard-fails:
- **Blocker**: Any `HIGH` or `CRITICAL` vulnerability found by SAST.
- **Blocker**: Any hardcoded secrets detected by secret-scanning.
- **Blocker**: Any sandbox escape attempt detected during Test Gate.

---

## 4. Remediation Workflow

1. **Detection**: Audit tool identifies an issue.
2. **Notification**: The `Developer` agent receives the specific finding (file, line, vulnerability type, description).
3. **Automated Fix**: The agent attempts to refactor the code to address the vulnerability.
4. **Re-Audit**: The code is re-submitted to the Review Service.
5. **Escalation**: If the agent fails to fix a `HIGH` vulnerability after 3 attempts, the task is flagged for **Human Intervention**.

---

## 5. Audit Logging & Compliance

- Every audit run is saved as a `Review` record with a permanent link to the `CommitSHA`.
- Audit logs are immutable and stored in the `audit_logs` table.
- Monthly "Security Health Reports" are generated summarizing agent-induced vs. human-induced security trends.
