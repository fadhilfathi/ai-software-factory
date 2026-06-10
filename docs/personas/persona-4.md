# Persona 4: Developer — "Jordan Kim"

## Name and Role
**Jordan Kim**, 29 — Full-Stack Engineer, Product Engineering (2 years at company, 5 years total experience)

## Demographics and Background
- **Location**: Denver, CO (fully remote)
- **Education**: BS Computer Science, University of Colorado Boulder; self-taught React/Node
- **Career Path**: Junior dev at agency (2 years, 15 client projects) → Mid-level at Series A fintech (3 years) → Current role at B2B SaaS (2 years)
- **Technical Fluency**: Strong in TypeScript/React/Node/PostgreSQL; comfortable with AWS, Docker, Kubernetes basics; learning Go; writes tests but considers it a chore
- **Team**: 5 engineers (2 frontend, 2 backend, 1 full-stack), 1 PM, 1 designer, 1 QA — works on "Growth Features" squad

## Goals and Motivations
| Goal | Motivation |
|------|------------|
| Ship clean code fast without cutting corners | Pride in craft; avoids late-night debugging sessions |
| Reduce boilerplate: auth, CRUD, API clients, forms, tests | 40% of time on repetitive patterns; wants to focus on business logic |
| Get fast feedback on code quality *before* PR | Hates review ping-pong; "CI red → fix → push → wait" loop |
| Learn from senior engineers without constant pairing | Wants to grow; pairing doesn't scale; code review feedback is delayed |
| Have autonomy to make good local decisions | Micromanaged tech choices kill motivation |

## Pain Points
| Pain Point | Impact | Current Workaround |
|------------|--------|-------------------|
| Setting up new service takes 1-2 days (config, CI, observability, deploy) | Delays first feature commit | Copies from last service; misses updates |
| Writing tests takes as long as the feature | Skips tests when pressured; tech debt | "Test later" (never happens) |
| PR reviews take 24-48 hours; often nitpick style not substance | Context switch to other task; loses flow | Stacks PRs; creates dependency chains |
| Inconsistent patterns across codebase | Cognitive load; bugs from "works on my machine" | Asks Tech Lead in Slack; inconsistent answers |
| Requirements change mid-implementation | Rework; frustration | "Just refactor it" — no time allocated |
| Debugging production issues without good observability | 2-4 hours to reproduce locally | Adds logs, redeploys, waits — slow cycle |

## How They Interact with the Platform
- **Primary Interface**: VS Code extension (primary) + web dashboard + CLI
- **Frequency**: Continuous — platform is "always on" in IDE
- **Key Workflows**:
  1. **Task Pickup**: Sees assigned task in VS Code sidebar (synced from Linear) → clicks "Start" → platform: scaffolds service/module, generates boilerplate per paved road, creates test files, sets up feature flag
  2. **Code Generation**: Types `// TODO: implement user search with filters` → platform generates: API endpoint, database query with pagination, React hook, TypeScript types, unit + integration tests → Jordan reviews, tweaks, accepts
  3. **Real-Time Feedback**: As Jordan types, platform shows: pattern violations (red squiggly), suggested paved road alternatives (lightbulb), security warnings, performance tips — *before save*
  4. **Test-Driven Loop**: Writes failing test → platform suggests implementation → Jordan accepts/modifies → test passes → repeat
  5. **PR Preparation**: Clicks "Prepare PR" → platform: runs all checks, generates PR description from task + changes, suggests reviewers, creates stack if dependent → one-click submit
  6. **Production Debugging**: Error alert in Slack → clicks "Open in Platform" → sees: distributed trace, related logs, similar past incidents, suggested fix from postmortem DB
- **Permissions**: Write code, run tests, deploy to dev/staging, create PRs
- **Integrations**: VS Code (inline), GitHub (PR), Linear (tasks), Datadog (observability), Slack (alerts), npm/GitHub Packages (paved road libraries)

## Day in the Life
**8:30 AM** — Opens VS Code. Platform sidebar: *"Task GRW-342: Add usage metrics endpoint — Ready to start."* Clicks Start. Platform: *Scaffolds `GET /api/v1/usage/metrics` with pagination, filtering, OpenAPI spec, React Query hook, 12 test cases (unit + contract + e2e).* Jordan: "Use cursor-based pagination instead of offset." Platform regenerates.

**9:15 AM** — Implements business logic. Types query builder. Platform inline: *"Paved road: use QueryBuilder class from @company/db — handles SQL injection, typing, pagination."* Jordan accepts. Writes 3 custom filter methods.

**10:30 AM** — Runs tests in watch mode. Platform: *All 12 pass. Coverage: 94%. Mutation testing: 2 survivors (edge cases).* Jordan adds 2 tests. Coverage 98%.

**11:30 AM** — Needs auth check for enterprise orgs. Types `// TODO: verify org has usage-metrics feature flag`. Platform generates: feature flag check, 403 response, audit log, test cases. Jordan adds integration test with mocked flag service.

**12:30 PM** — Lunch. Phone buzz: *PR #2341 ready for review (dependent on your branch).* Platform: *"Your changes don't conflict. Want to stack?"* Jordan: "Yes." Platform creates stacked PR.

**1:30 PM** — Reviews teammate's PR (platform assigned). Platform pre-reviewed: *3 pattern violations, 1 security issue (missing rate limit), all tests pass.* Jordan adds: "Nice! Consider extracting this validation to shared util per ADR-012." Approves.

**3:00 PM** — Deploys to staging. Platform: *Canary deploy to 5% traffic. Health checks passing. Performance: p95 120ms (target <200ms).* Shares staging URL with PM in Slack.

**4:15 PM** — PM feedback: "Filter by date range needs relative presets (last 7d, 30d)." Jordan adds preset enum to OpenAPI spec. Platform regenerates: API, types, React component, tests. 15 min done.

**5:00 PM** — Wraps up. Platform summary: *1 task completed, 147 LOC (82 generated), 14 tests added, 0 pattern violations, 1 PR merged, 1 stacked, staging deployed.* Commits learning: "Cursor pagination pattern saved to personal snippets."

---

*Persona created for AI Software Factory platform — Sprint 1, TASK-002*