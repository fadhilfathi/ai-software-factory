# Sprint 4 Technical Specifications: Code & Review Services

> **Document Version**: 1.0  
> **Status**: Draft  
> **Owner**: TechLead

---

## 1. Overview

Sprint 4 focuses on **Autonomous Code Generation & Review**. This involves providing agents with the tools to generate, test, and review code within a secure execution sandbox.

The **Code Service** manages the codebase, generation requests, and Git operations.  
The **Review Service** manages quality gates, automated scanning, and multi-agent review workflows.

---

## 2. Code Service

### 2.1 Responsibilities
- Manage code generation requests (`CodeGenRequest`).
- Orchestrate sandbox execution for code validation.
- Provide Git-like operations (commits, branches, diffs).
- Maintain project file state and metadata.
- Perform static analysis and metric extraction.

### 2.2 Data Models

#### CodeGenRequest (Updated)
```go
type CodeGenRequest struct {
    ID            uuid.UUID     `json:"id"`
    ProjectID     uuid.UUID     `json:"project_id"`
    TaskID        uuid.UUID     `json:"task_id"`
    Specification string        `json:"specification"`
    Files         []string      `json:"files"` // Targeted files
    Status        CodeGenStatus `json:"status"`
    ExecutionID   uuid.UUID     `json:"execution_id,omitempty"` // Link to sandbox run
    Output        string        `json:"output,omitempty"`       // Logs from generation/test
    CreatedAt     time.Time     `json:"created_at"`
    UpdatedAt     time.Time     `json:"updated_at"`
}
```

#### Commit & ProjectFile (Existing)
Already defined in `src/internal/model/code.go`. These will be migrated to PostgreSQL.

### 2.3 API Surface

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/v1/code/generate` | `POST` | Request code generation for a task. |
| `/v1/code/:projectId/files` | `GET` | List/Search files in a project. |
| `/v1/code/:projectId/files/*path` | `GET` | Get content and metadata of a specific file. |
| `/v1/code/:projectId/commits` | `POST` | Create a new commit (persists files). |
| `/v1/code/:projectId/diff` | `GET` | Get diff between two SHAs or SHA vs Working Tree. |
| `/v1/code/:projectId/analysis` | `GET` | Get static analysis report (complexity, linting). |

### 2.4 Sandbox Execution Flow
1. **Request**: Agent calls `POST /v1/code/generate`.
2. **Setup**: Code Service prepares a workspace with current project files.
3. **Execution**: Service triggers a gVisor/Firecracker sandbox via `ExecutionService`.
4. **Validation**: Sandbox runs generation logic + immediate unit tests/linting.
5. **Result**: Output and status (`completed`/`failed`) are returned to the Agent.

---

## 3. Review Service

### 3.1 Responsibilities
- Manage the code review lifecycle.
- Enforce quality gates (coverage, security, complexity).
- Orchestrate Reviewer Agents for deep code analysis.
- Provide a feedback loop between Reviewers and Developers.

### 3.2 Data Models

#### Review (Updated)
```go
type Review struct {
    ID           uuid.UUID      `json:"id"`
    ProjectID    uuid.UUID      `json:"project_id"`
    CommitSHA    string         `json:"commit_sha"`
    ReviewerType string         `json:"reviewer_type"` // 'automated' or 'agent'
    ReviewerID   uuid.UUID      `json:"reviewer_id,omitempty"`
    Status       ReviewStatus   `json:"status"`
    Result       ReviewResult   `json:"result,omitempty"` // 'approved', 'changes_requested'
    Score        float64        `json:"score"`            // 0-100 quality score
    Issues       []ReviewIssue  `json:"issues,omitempty"`
    Metrics      *ReviewMetrics `json:"metrics,omitempty"`
    CreatedAt    time.Time      `json:"created_at"`
    UpdatedAt    time.Time      `json:"updated_at"`
}
```

#### ReviewComment
```go
type ReviewComment struct {
    ID        uuid.UUID `json:"id"`
    ReviewID  uuid.UUID `json:"review_id"`
    File      string    `json:"file"`
    Line      int       `json:"line"`
    AuthorID  uuid.UUID `json:"author_id"`
    Content   string    `json:"content"`
    CreatedAt time.Time `json:"created_at"`
}
```

### 3.3 API Surface

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/v1/reviews` | `POST` | Start a new review for a commit. |
| `/v1/reviews/:id` | `GET` | Get review findings, score, and status. |
| `/v1/reviews/:id/comments` | `POST` | Add a comment to a review. |
| `/v1/reviews/:id/comments` | `GET` | List comments for a review. |
| `/v1/reviews/:id/status` | `PATCH` | Manually override or update review status. |
| `/v1/reviews/project/:projectId` | `GET` | List all reviews for a project. |

### 3.4 Quality Gates
Reviews will be automatically marked as `approved` if:
1. `ReviewerType == 'automated'`
2. `Score >= 80`
3. Zero `high` or `critical` severity `ReviewIssue`s.
4. `TestCoverage >= 70%` (if applicable).

---

## 4. Integration with Other Services

- **Agent Orchestrator**: Uses `reviewer` agents for `POST /v1/reviews`.
- **Project Service**: Updates task status to `Review` when code is generated.
- **Notification Service**: Alerts Developer Agent when a review requires changes.
- **Execution Sandbox**: Provides the runtime for automated review tools (linters, scanners).

---

## 5. Database Schema (PostgreSQL)

### 5.1 `code_gen_requests`
- `id` UUID PRIMARY KEY
- `project_id` UUID REFERENCES projects(id)
- `task_id` UUID REFERENCES tasks(id)
- `specification` TEXT
- `files` TEXT[]
- `status` VARCHAR(20)
- `execution_id` UUID
- `output` TEXT
- `created_at` TIMESTAMPTZ
- `updated_at` TIMESTAMPTZ

### 5.2 `commits`
- `sha` VARCHAR(40) PRIMARY KEY
- `project_id` UUID REFERENCES projects(id)
- `branch` VARCHAR(100)
- `message` TEXT
- `author` VARCHAR(100)
- `created_at` TIMESTAMPTZ

### 5.3 `project_files`
- `project_id` UUID REFERENCES projects(id)
- `path` TEXT
- `content` TEXT
- `language` VARCHAR(50)
- `size` INTEGER
- `last_modified` TIMESTAMPTZ
- `modified_by` VARCHAR(100)
- PRIMARY KEY (project_id, path)

### 5.4 `reviews`
- `id` UUID PRIMARY KEY
- `project_id` UUID REFERENCES projects(id)
- `commit_sha` VARCHAR(40)
- `reviewer_type` VARCHAR(20)
- `reviewer_id` UUID
- `status` VARCHAR(20)
- `result` VARCHAR(20)
- `score` FLOAT
- `metrics` JSONB
- `created_at` TIMESTAMPTZ
- `updated_at` TIMESTAMPTZ

### 5.5 `review_issues`
- `id` UUID PRIMARY KEY
- `review_id` UUID REFERENCES reviews(id)
- `severity` VARCHAR(20)
- `file` TEXT
- `line` INTEGER
- `message` TEXT
- `suggestion` TEXT
- `created_at` TIMESTAMPTZ

### 5.6 `review_comments`
- `id` UUID PRIMARY KEY
- `review_id` UUID REFERENCES reviews(id)
- `file` TEXT
- `line` INTEGER
- `author_id` UUID
- `content` TEXT
- `created_at` TIMESTAMPTZ
