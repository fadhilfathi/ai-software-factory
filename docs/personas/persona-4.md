# Persona 4: Priya Sharma — Senior Developer

## Demographics & Background
- **Name:** Priya Sharma
- **Role:** Senior Software Engineer
- **Company:** HealthTech Solutions (Healthcare SaaS, 30 engineers)
- **Age:** 31
- **Location:** Boston, MA (remote)
- **Education:** BS Computer Science from University of Washington
- **Experience:** 7 years full-stack, 2 at current company, previously at Amazon

## Goals & Motivations
- Write clean, maintainable code without constant context switching
- Own features end-to-end: design, implementation, testing, deployment
- Learn new patterns by seeing them applied correctly
- Minimize boilerplate and repetitive tasks
- Have confidence that changes won't break production

## Pain Points
1. **Endless boilerplate** — Same CRUD patterns, same validation, same error handling every feature
2. **Context switching tax** — Jira, GitHub, CI/CD, docs, Slack — 5 tools for one feature
3. **Unclear requirements** — Vague tickets lead to rework and frustration
4. **Testing is an afterthought** — No time to write tests; QA catches bugs days later
5. **Deployment anxiety** — Manual steps, flaky pipelines, fear of Friday deploys

## How Priya Interacts with the Platform
- Receives well-scoped tasks from the PM Agent with clear acceptance criteria
- Uses the Developer Agent to generate scaffolding, tests, and boilerplate
- Reviews and refines agent-generated code — adds business logic, handles edge cases
- Runs the Review Agent locally before pushing for instant feedback
- Deploys via the DevOps Agent; monitors through the platform dashboard

## A Day in the Life

**8:30 AM** — Opens the platform. Assigned task: "Add Webhook Retry Logic for Payment Callbacks." The Developer Agent has already generated the service structure, unit tests, and integration test harness.

**9:00 AM** — Reviews the generated code. The pattern matches the team's standards. Priya adds the exponential backoff logic, idempotency key handling, and dead-letter queue integration — the domain-specific parts AI can't guess.

**10:30 AM** — Runs the Review Agent. Catches a potential race condition in the retry logic. Fixes it. Runs the full test suite — all green.

**11:00 AM** — Pushes to feature branch. The DevOps Agent picks it up: runs CI, deploys to preview environment, runs contract tests against the payments service.

**11:30 AM** — Coffee break. Checks phone — preview deployment is live. QA Agent is already running automated tests against it.

**1:00 PM** — Pair programming session. Another engineer needs help with a GraphQL resolver. They use the platform together — the Developer Agent generates the resolver pattern; they add the business logic.

**2:30 PM** — Reviews a PR from a junior engineer. The Review Agent already flagged 3 issues. Priya adds 2 more suggestions about error handling. Approves.

**4:00 PM** — Feature is in staging. Product Manager validates the acceptance criteria in the preview env. All pass. Priya merges to main.

**4:30 PM** — The DevOps Agent promotes to production with blue-green deploy. Zero downtime. Priya monitors the dashboard — error rate flat, latency improved.

**5:00 PM** — Updates the task: "Done." The platform auto-generates the changelog entry. Logs off feeling productive.

## Success Criteria for Priya
- Spends 70% of time on business logic, 30% on plumbing (reversed from current)
- Gets instant feedback on code quality before CI
- Deploys are boring and safe — any day of the week
- Learns new patterns by seeing them in generated code
- Owns features completely without waiting on other teams