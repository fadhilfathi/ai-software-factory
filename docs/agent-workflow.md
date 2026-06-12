# AI Software Factory — Agent Workflow Design

## Overview

The AI Software Factory uses a multi-agent system where specialized AI agents collaborate to deliver software projects. Each agent has a defined role, capabilities, and interaction patterns.

## Agent Types

### 1. PM Agent (Product Manager)
**Role:** Requirements decomposition, task prioritization, sprint planning
**Model:** GPT-4 or Claude (strong reasoning)
**Capabilities:**
- Parse natural language requirements into structured user stories
- Prioritize tasks using MoSCoW method
- Identify dependencies between tasks
- Generate acceptance criteria
- Create sprint plans

### 2. Architect Agent
**Role:** System design, technology selection, API contracts
**Model:** GPT-4 or Claude (strong reasoning)
**Capabilities:**
- Design system architecture
- Select technology stack
- Define API contracts
- Design database schemas
- Document architectural decisions (ADRs)

### 3. Developer Agent
**Role:** Code implementation, refactoring, bug fixes
**Model:** GPT-4 or Claude (code-optimized)
**Capabilities:**
- Write production-quality code
- Follow project coding standards
- Implement unit tests
- Refactor existing code
- Fix bugs based on descriptions

### 4. Review Agent
**Role:** Code review, quality enforcement, security scanning
**Model:** GPT-4 or Claude (code-optimized)
**Capabilities:**
- Review code for quality issues
- Check for security vulnerabilities
- Verify test coverage
- Enforce coding standards
- Suggest improvements

### 5. QA Agent
**Role:** Test planning, test execution, bug reporting
**Model:** GPT-4 or Claude (reasoning + code)
**Capabilities:**
- Generate test plans from acceptance criteria
- Write automated tests
- Execute test suites
- Report bugs with reproduction steps
- Track test coverage

### 6. DevOps Agent
**Role:** CI/CD, deployment, infrastructure
**Model:** GPT-4 or Claude (code + infrastructure)
**Capabilities:**
- Set up CI/CD pipelines
- Configure deployment environments
- Manage infrastructure as code
- Monitor deployments
- Handle rollbacks
- Always commit and push after a finished sprint

## Workflow States

```
┌─────────────┐
│   REQUEST   │ User submits project description
│   INTAKE    │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│  ANALYSIS   │ PM Agent decomposes requirements
│             │ Generates user stories and tasks
└──────┬──────┘
       │
       ▼
┌─────────────┐
│  PLANNING   │ Architect Agent designs system
│             │ Selects tech stack, defines APIs
└──────┬──────┘
       │
       ▼
┌─────────────┐
│IMPLEMENTATION│ Developer Agents write code
│             │ Runs in parallel for independent tasks
└──────┬──────┘
       │
       ▼
┌─────────────┐
│   REVIEW    │ Review Agent checks code quality
│             │ Security scan, test coverage check
└──────┬──────┘
       │
       ├──▶ FAIL ──▶ Back to IMPLEMENTATION (with feedback)
       │
       ▼ PASS
┌─────────────┐
│   TESTING   │ QA Agent runs test suite
│             │ Verifies acceptance criteria
└──────┬──────┘
       │
       ├──▶ FAIL ──▶ Back to IMPLEMENTATION (with bug report)
       │
       ▼ PASS
┌─────────────┐
│  DEPLOYMENT │ DevOps Agent deploys to staging
│             │ Health checks, smoke tests
└──────┬──────┘
       │
       ▼
┌─────────────┐
│    DONE     │ Feature complete
│             │ Metrics recorded, lessons learned
└─────────────┘
```

## Agent Communication Protocol

### Message Format
```json
{
  "message_id": "msg_abc123",
  "from_agent": "agent_dev_001",
  "to_agent": "agent_review_001",
  "type": "task_complete",
  "project_id": "proj_001",
  "task_id": "task_001",
  "payload": {
    "files_changed": ["src/auth/login.ts", "src/auth/register.ts"],
    "commit_sha": "abc123def456",
    "summary": "Implemented JWT authentication with login and register endpoints"
  },
  "timestamp": "2026-06-10T10:45:00Z"
}
```

### Message Types
| Type | From | To | Description |
|------|------|----|-------------|
| `task_assigned` | Orchestrator | Agent | New task ready for execution |
| `task_complete` | Agent | Orchestrator | Task finished successfully |
| `task_failed` | Agent | Orchestrator | Task failed, needs retry |
| `review_request` | Orchestrator | Review Agent | Code needs review |
| `review_complete` | Review Agent | Orchestrator | Review finished |
| `feedback` | Review Agent | Developer Agent | Specific code feedback |
| `deploy_request` | Orchestrator | DevOps Agent | Ready for deployment |
| `deploy_complete` | DevOps Agent | Orchestrator | Deployment finished |
| `test_request` | Orchestrator | QA Agent | Run test suite |
| `test_complete` | QA Agent | Orchestrator | Tests finished |

### Handoff Between Agents

```
Developer Agent
    │
    ├── Commits code to feature branch
    ├── Creates PR with description
    ├── Sends task_complete to Orchestrator
    │
    ▼
Orchestrator
    │
    ├── Validates commit meets task requirements
    ├── Assigns Review Agent
    ├── Sends review_request
    │
    ▼
Review Agent
    │
    ├── Reviews code changes
    ├── Checks security, quality, coverage
    ├── If pass: sends review_complete (approved)
    ├── If fail: sends feedback to Developer Agent
    │
    ▼
QA Agent (on approval)
    │
    ├── Runs automated test suite
    ├── Verifies acceptance criteria
    ├── If pass: sends test_complete (passed)
    ├── If fail: sends bug report to Developer Agent
    │
    ▼
DevOps Agent (on test pass)
    │
    ├── Deploys to staging
    ├── Runs health checks
    ├── Sends deploy_complete
    │
    ▼
Orchestrator
    │
    ├── Updates project status
    ├── Notifies user
    ├── Picks next task from queue
```

## Task Decomposition

### Decomposition Flow
```
High-Level Request: "Build user authentication"
            │
            ▼
    PM Agent Decomposition
            │
            ├── User Story 1: User Registration
            │   ├── Task 1.1: Design registration API (Architect)
            │   ├── Task 1.2: Implement registration endpoint (Developer)
            │   ├── Task 1.3: Write registration tests (QA)
            │   └── Task 1.4: Review registration code (Review)
            │
            ├── User Story 2: User Login
            │   ├── Task 2.1: Design login API (Architect)
            │   ├── Task 2.2: Implement login endpoint (Developer)
            │   ├── Task 2.3: Write login tests (QA)
            │   └── Task 2.4: Review login code (Review)
            │
            └── User Story 3: Token Management
                ├── Task 3.1: Design token refresh flow (Architect)
                ├── Task 3.2: Implement token refresh (Developer)
                ├── Task 3.3: Write token tests (QA)
                └── Task 3.4: Review token code (Review)
```

### Parallel Execution
```
Independent Tasks Run Simultaneously:

Time ─────────────────────────────────────────────────▶

Task 1.1 (Architect): ████████████
Task 1.2 (Developer):             ████████████████████
Task 1.3 (QA):                                    ████████████
Task 1.4 (Review):                                    ████████

Task 2.1 (Architect): ████████████
Task 2.2 (Developer):             ████████████████████
Task 2.3 (QA):                                    ████████████
Task 2.4 (Review):                                    ████████

Total wall time: ~40 minutes (vs 160 minutes sequential)
```

### Dependency Management
```
Task Dependencies:
- 1.1 → 1.2 (Developer needs API design)
- 1.2 → 1.3 (QA needs implementation)
- 1.2 → 1.4 (Review needs implementation)
- 2.1 → 2.2 (same pattern)
- 2.2 → 2.3, 2.4 (same pattern)

No dependencies between Story 1 and Story 2 → parallel execution
```

## Quality Gates

### Gate 1: Code Review
**Trigger:** PR created by Developer Agent
**Checklist:**
- [ ] Code compiles without errors
- [ ] No critical security vulnerabilities
- [ ] Test coverage >= 80%
- [ ] Coding standards followed
- [ ] No duplicate code > 10 lines

**Decision:**
- ✅ Approved → Proceed to testing
- ❌ Changes Requested → Back to Developer with feedback

### Gate 2: Test Execution
**Trigger:** Review approved
**Checklist:**
- [ ] All unit tests pass
- [ ] All integration tests pass
- [ ] Acceptance criteria verified
- [ ] Performance within thresholds
- [ ] No regressions in existing features

**Decision:**
- ✅ All Pass → Proceed to deployment
- ❌ Failures → Back to Developer with bug report

### Gate 3: Deployment
**Trigger:** Tests passed
**Checklist:**
- [ ] Build successful
- [ ] Staging environment healthy
- [ ] Health checks passing
- [ ] Monitoring alerts configured
- [ ] Rollback plan documented

**Decision:**
- ✅ All Green → Deploy to staging
- ❌ Issues → Block deployment, notify user

### Gate 4: Production Promotion
**Trigger:** Staging verification complete (manual approval required)
**Checklist:**
- [ ] Staging tests pass
- [ ] User acceptance (manual)
- [ ] Performance benchmarks met
- [ ] Security scan clean
- [ ] Rollback tested

**Decision:**
- ✅ Human Approves → Deploy to production
- ❌ Human Rejects → Back to development

## Error Handling

### Agent Failure Recovery
```
Agent Fails
    │
    ├── Transient Error (timeout, rate limit)
    │   └── Retry (max 3 attempts, exponential backoff)
    │
    ├── Code Error (syntax, logic)
    │   ├── Capture error details
    │   ├── Send to Developer Agent with context
    │   └── Developer Agent fixes and retries
    │
    ├── Quality Gate Failure
    │   ├── Capture review/test feedback
    │   ├── Send to original Developer Agent
    │   └── Developer Agent addresses feedback
    │
    └── Persistent Failure (3+ retries)
        ├── Mark task as blocked
        ├── Notify human user
        ├── Provide failure summary
        └── Wait for human intervention
```

### Retry Logic
```python
RETRY_CONFIG = {
    "max_attempts": 3,
    "backoff_multiplier": 2,
    "initial_delay": 5,  # seconds
    "max_delay": 300,    # 5 minutes
    "retryable_errors": [
        "timeout",
        "rate_limit",
        "temporary_failure",
        "connection_error"
    ],
    "non_retryable_errors": [
        "invalid_input",
        "permission_denied",
        "resource_not_found"
    ]
}
```

### Escalation to Human
```
Escalation Triggers:
1. Agent fails 3 times on same task
2. Quality gate fails 3 times
3. Security vulnerability detected
4. Budget/cost threshold exceeded
5. User explicitly requested human review

Escalation Actions:
1. Mark task as "needs_human"
2. Send notification to project owner
3. Include full context (what was tried, what failed)
4. Pause agent work on related tasks
5. Wait for human response
```

## Monitoring & Observability

### Agent Health Metrics
| Metric | Description | Alert Threshold |
|--------|-------------|-----------------|
| Task Completion Rate | % of tasks completed successfully | < 80% |
| Average Task Duration | Mean time to complete a task | > 2x estimate |
| Error Rate | % of tasks that fail | > 20% |
| Retry Rate | % of tasks requiring retry | > 30% |
| Token Usage | LLM tokens consumed per task | > 100K |
| Cost per Task | Average cost per task | > $5 |

### Task Completion Rates
```
Project Dashboard Metrics:
- Tasks Created: 50
- Tasks In Progress: 12
- Tasks Completed: 35
- Tasks Blocked: 3
- Completion Rate: 92%
- Average Duration: 45 minutes
- On-Time Delivery: 88%
```

### Bottleneck Identification
```
Bottleneck Detection:
1. Queue Depth > 10 → Scale up agent workers
2. Review Wait Time > 30 min → Add review agent capacity
3. Test Suite Duration > 30 min → Optimize tests
4. Deployment Failures > 2x/week → Investigate infrastructure
5. Agent Idle Time > 50% → Rebalance task assignments
```

### Dashboard Views
1. **Project Overview:** Overall progress, active agents, recent completions
2. **Agent Performance:** Individual agent metrics, utilization, costs
3. **Task Board:** Kanban view of all tasks with status
4. **Quality Metrics:** Test coverage, review scores, bug trends
5. **Cost Tracking:** Token usage, API costs, total project spend

## ASCII Workflow Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                    AI SOFTWARE FACTORY                          │
│                    Agent Workflow System                         │
└─────────────────────────────────────────────────────────────────┘

  USER                    ORCHESTRATOR                   AGENTS
    │                          │                           │
    │  1. Submit Request       │                           │
    │─────────────────────────▶│                           │
    │                          │                           │
    │                          │  2. Spawn PM Agent        │
    │                          │──────────────────────────▶│
    │                          │                           │
    │                          │  3. Requirements          │
    │                          │◀──────────────────────────│
    │                          │                           │
    │                          │  4. Spawn Architect       │
    │                          │──────────────────────────▶│
    │                          │                           │
    │                          │  5. Architecture Design   │
    │                          │◀──────────────────────────│
    │                          │                           │
    │                          │  6. Spawn Developers (x3) │
    │                          │──────────────────────────▶│
    │                          │                           │
    │                          │  7. Code Generated        │
    │                          │◀──────────────────────────│
    │                          │                           │
    │                          │  8. Spawn Review Agent    │
    │                          │──────────────────────────▶│
    │                          │                           │
    │                          │  9. Review Complete       │
    │                          │◀──────────────────────────│
    │                          │                           │
    │                          │  10. Spawn QA Agent       │
    │                          │──────────────────────────▶│
    │                          │                           │
    │                          │  11. Tests Pass           │
    │                          │◀──────────────────────────│
    │                          │                           │
    │                          │  12. Spawn DevOps Agent   │
    │                          │──────────────────────────▶│
    │                          │                           │
    │                          │  13. Deployment Complete  │
    │                          │◀──────────────────────────│
    │                          │                           │
    │  14. Project Complete     │                           │
    │◀─────────────────────────│                           │
    │                          │                           │
```
