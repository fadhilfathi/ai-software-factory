# Quality & Security Gates for Autonomous Code

This document defines the automated quality and security gates that all autonomously generated code must pass before being merged or promoted.

## 1. Overview

Autonomous code generation introduces risks regarding code quality, security vulnerabilities, and maintainability. To mitigate these risks, we implement a multi-layered validation pipeline.

### The Pipeline
1. **Linter Gate**: Static analysis for style and common errors.
2. **Complexity Gate**: Ensuring code remains maintainable and not overly complex.
3. **Security Gate (SAST)**: Identification of potential security vulnerabilities.
4. **Test Gate**: Validation of functional correctness and regression testing.
5. **AI Review Gate**: Architectural and logic review by specialized Reviewer agents.

---

## 2. Quality Gates

### 2.1 Linter Gate
All code must adhere to project-specific styling and best practices.
- **Go**: `golangci-lint` (default presets + `gocritic`, `revive`).
- **TypeScript/React**: `eslint` (with `@typescript-eslint/recommended`, `react-hooks/recommended`).
- **Pass Criteria**: Zero errors, zero warnings in designated "strict" categories.

### 2.2 Complexity Gate
Limits the cognitive load required to understand and maintain the code.
- **Metric**: Cyclomatic Complexity (using `gocyclo` for Go, `eslint-plugin-complexity` for TS).
- **Pass Criteria**: 
  - Maximum per-function complexity: **10**.
  - Average per-file complexity: **5**.

### 2.3 Test Gate
Autonomous code is incomplete without tests.
- **Requirement**: Every new feature or bug fix must include corresponding unit tests.
- **Coverage**: Minimum **80%** statement coverage for the changed lines.
- **Pass Criteria**: 
  - All tests pass.
  - Coverage threshold met.
  - No "flaky" tests detected.

---

## 3. Security Gates

### 3.1 SAST (Static Analysis Security Testing)
Scans the source code for known security patterns (e.g., hardcoded secrets, SQL injection, insecure cryptography).
- **Tools**: `gosec` (Go), `semgrep` (General purpose), `npm audit` (Dependencies).
- **Pass Criteria**: 
  - Zero `HIGH` or `CRITICAL` severity findings.
  - All `MEDIUM` findings must be reviewed and documented by a human or a high-confidence security agent.

### 3.2 Secret Scanning
Prevents accidental leakage of API keys, tokens, or credentials.
- **Tool**: `gitleaks` or `trufflehog`.
- **Pass Criteria**: Zero secrets detected in the commit history or working tree.

### 3.3 Sandbox Integrity
Code execution for testing and generation must occur in a hardened environment.
- **Isolation**: `gVisor` or `Firecracker` microVMs.
- **Network**: Deny all by default (unless specifically required and scoped).
- **Filesystem**: Read-only root filesystem, scoped writable workspace.

---

## 4. AI Review Gate

Specialized Reviewer agents perform a deep dive into the code.
- **Criteria**:
  - **Correctness**: Does it fulfill the requirement?
  - **Architecture**: Does it follow the established patterns?
  - **Readability**: Is the code clear and well-documented?
- **Pass Criteria**:
  - **Review Score >= 80/100**.
  - Zero `blocker` issues identified.

---

## 5. Enforcement & Remediation

### Automatic Blocking
If any gate fails (except for AI Review Gate which is advisory-first), the code generation request is marked as `failed`, and the agent is notified with the specific failure logs.

### Agent Loop
1. Agent generates code.
2. Gates run.
3. Gates fail → Agent receives logs → Agent attempts fix (max 3 retries).
4. Gates pass → Code moves to AI Review.
5. AI Review pass → Final Approval.

---

## 6. Metrics & Monitoring

We track the following metrics to evaluate the effectiveness of our gates:
- **Gate Pass Rate**: Percentage of generation requests that pass all gates on first try.
- **Escape Rate**: Number of bugs/vulnerabilities found in "passed" code by humans.
- **Mean Time to Repair (MTTR)**: How long it takes agents to fix gate failures.
