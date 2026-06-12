# TASK-302: Review Service Architectural Design

> **Owner**: QA (019eb7a3-fe02-7413-ba61-0979c0a215c8)  
> **Status**: Draft  
> **Date**: 2026-06-12

---

## 1. Objective

The Review Service is responsible for orchestrating the quality and security validation of autonomously generated code. It serves as the primary gatekeeper before code is considered "ready" for further stages.

## 2. Core Responsibilities

- **Gate Orchestration**: Sequentially or concurrently trigger Linter, Complexity, SAST, and Test gates.
- **Reviewer Agent Management**: Spawn and manage `reviewer` agents for deep logic and architectural analysis.
- **Outcome Aggregation**: Consolidate findings from multiple sources into a single `Review` record.
- **Decision Logic**: Apply business rules to determine if a review is `approved` or `changes_requested`.

---

## 3. Workflow: The Autonomous Review Pipeline

1. **Trigger**: `CodeService` or a `Developer` agent calls `POST /v1/reviews`.
2. **Phase 1: Automated Scanning (Synchronous/Fast)**
   - **Linter**: Runs `golangci-lint` (Go) or `eslint` (TS).
   - **SAST**: Runs `gosec` (Go) or `semgrep`.
   - **Complexity**: Calculates cyclomatic complexity.
3. **Phase 2: Execution Validation (Asynchronous)**
   - **Test Runner**: Triggers `ExecutionService` to run unit/integration tests in a gVisor sandbox.
   - **Coverage**: Extracts coverage metrics from test output.
4. **Phase 3: AI Peer Review (Asynchronous/Deep)**
   - **Reviewer Agent**: A `reviewer` agent is assigned the task to analyze the code for architectural alignment and logic errors.
5. **Finalization**:
   - Scores and issues are aggregated.
   - `ReviewResult` is calculated based on defined thresholds.
   - Status is set to `completed`.

---

## 4. API Design (Refined)

### `POST /v1/reviews`
Initiates a new review.
- **Request**:
  ```json
  {
    "project_id": "uuid",
    "commit_sha": "string",
    "reviewer_type": "automated|agent",
    "target_agent_id": "uuid (optional)"
  }
  ```

### `GET /v1/reviews/:id`
Retrieves the full review report.

### `GET /v1/reviews/project/:projectId`
Lists all reviews for a project.

### `POST /v1/reviews/:id/comments`
Allows agents or humans to add specific feedback.

---

## 5. Decision Engine Thresholds

| Gate | "Approved" Threshold |
|---|---|
| **Linter** | Zero Errors |
| **SAST** | Zero HIGH/CRITICAL findings |
| **Complexity** | Max per-function: 15 |
| **Test Coverage** | Minimum 70% |
| **AI Review Score** | Minimum 80/100 |

---

## 6. Security Considerations (Alignment with TASK-308)

- **Execution Isolation**: All automated scans and tests MUST run within the hardened `gVisor` sandbox defined in TASK-315.
- **Least Privilege**: `reviewer` agents are granted read-only access to the codebase.
- **Audit Logging**: Every review decision and gate failure is recorded in the system audit log.

---

## 7. Implementation Roadmap

1. **Step 1**: Update `internal/model/review.go` to include `target_agent_id` and refine `ReviewMetrics`.
2. **Step 2**: Implement the `GateRunner` interface and individual gate implementations (Lint, SAST, Complexity).
3. **Step 3**: Integrate with `ExecutionService` for the Test Gate.
4. **Step 4**: Implement the `reviewer` agent orchestration logic in `ReviewService`.
5. **Step 5**: Finalize the aggregation and decision logic.
