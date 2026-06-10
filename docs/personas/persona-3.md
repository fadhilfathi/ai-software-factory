# Persona 3: Tech Lead — "Priya Sharma"

## Name and Role
**Priya Sharma**, 38 — Staff Engineer / Tech Lead, Platform Infrastructure (leads 8 engineers across 2 squads)

## Demographics and Background
- **Location**: Seattle, WA (hybrid 3 days/week)
- **Education**: MS Distributed Systems, University of Washington; BS CS, IIT Bombay
- **Career Path**: SDE at Amazon (5 years, DynamoDB team) → Senior Engineer at Snowflake (3 years) → Staff Engineer at current Series C startup (2 years)
- **Technical Fluency**: Expert in distributed systems, database internals, cloud architecture; writes production code 30% of time; reviews 15+ PRs/week
- **Scope**: Owns "Platform Foundations" — shared libraries, CI/CD, observability, database layer, API gateway, auth infrastructure

## Goals and Motivations
| Goal | Motivation |
|------|------------|
| Enforce architectural standards without being a bottleneck | 40 PRs/week waiting on her review; team velocity suffers |
| Reduce production incidents caused by inconsistent patterns | 3 SEV-2 incidents last quarter from "creative" implementations |
| Enable teams to move fast with guardrails, not gates | "Approval culture" slows innovation; wants paved roads |
| Codify tribal knowledge into executable standards | Onboarding takes 3 months; senior knowledge walks out door |
| Spend 50% time on strategic technical initiatives | Currently 80% reactive (reviews, incidents, fires) |

## Pain Points
| Pain Point | Impact | Current Workaround |
|------------|--------|-------------------|
| Reviews same patterns repeatedly (error handling, logging, transactions) | 15 hrs/week on mechanical reviews | Shared checklist doc; rarely followed |
| Teams choose different libraries for same problem | 4 HTTP clients, 3 config libraries, 2 logging wrappers | "Approved list" wiki; outdated in 2 months |
| No automated enforcement of ADR decisions | Decisions ignored; architecture drifts | Monthly "architecture audit" — always late |
| New hires copy bad patterns from old code | Perpetuates tech debt | Pair programming (expensive, doesn't scale) |
| Incident postmortems reveal known anti-patterns | Preventable; damages trust | "Don't do X" tribal knowledge; not documented |
| Cross-team dependencies block sprints | 30% sprint capacity on coordination | Weekly sync meeting; action items lost |

## How They Interact with the Platform
- **Primary Interface**: Tech Lead workspace (web + VS Code extension + CLI)
- **Frequency**: Continuous — IDE integration for real-time feedback; deep reviews 2-3 hrs/day
- **Key Workflows**:
  1. **Architecture Governance**: Defines "Paved Road" specs (language versions, libraries, patterns) → platform enforces via: PR gate checks, agent code generation templates, lint rules, dependency policies
  2. **ADR Lifecycle**: Writes ADR in platform → auto-generates: implementation checklist, lint rules, agent instructions, migration scripts → tracks adoption across repos
  3. **Code Review Augmentation**: Platform pre-reviews every PR: checks pattern compliance, suggests refactors, flags security/performance risks → Priya reviews only "judgment calls" (architecture, trade-offs)
  4. **Agent Instruction**: Configures Developer agents: "Use Repository pattern for data access", "All async operations must have timeout + retry with exponential backoff" → agents follow automatically
  5. **Incident → Prevention**: Postmortem creates "Prevention Rule" → platform backfills to existing code + adds to agent instructions + generates regression test
- **Permissions**: Write ADRs, configure agent rules, approve/reject architectural exceptions, manage paved road catalog
- **Integrations**: GitHub (PR checks, CODEOWNERS), GitLab, VS Code/JetBrains (inline suggestions), Datadog/PagerDuty (incident → rule), Backstage (service catalog), Slack (review requests)

## Day in the Life
**7:45 AM** — Commute. Checks phone: *3 PRs need architectural review*. Platform pre-reviewed: 2 auto-approved (follow paved road), 1 flagged: *"Uses custom retry logic instead of standard Resilience4j wrapper — adds 200 LOC, no test coverage."* Taps "Request Change" with link to paved road doc.

**9:00 AM** — Standup. Squad mentions: "New service needs config management." Priya: "Use platform's Config Agent — generates type-safe config from schema, handles secrets, feature flags." Engineer: "Done in 10 min."

**10:30 AM** — Deep work: Writing ADR-047 "Event-Driven Architecture for Billing Events". Platform: *Generates Kafka topic naming convention, Avro schema template, consumer error-handling boilerplate, migration script for existing webhook handlers, lint rule forbidding direct HTTP calls between services.* Commits ADR; platform propagates to 12 repos.

**12:30 PM** — Lunch with engineering manager. Discusses: "Team wants to adopt tRPC." Priya opens platform: "Show me tRPC vs. gRPC comparison for our use case." Platform runs analysis: *tRPC: 40% less boilerplate, TypeScript-first, but no Go/Java support. gRPC: polyglot, mature, but 3x boilerplate.* Decision: tRPC for TypeScript services, gRPC for polyglot. Platform generates starter kits for both.

**2:00 PM** — Incident retrospective (SEV-2: cascade failure from missing circuit breaker). Platform creates Prevention Rule: *"All outbound HTTP calls must have circuit breaker + timeout."* Backfills: finds 47 call sites missing breakers → generates PRs for each repo. Priya reviews 3 critical ones; approves batch merge for rest.

**3:30 PM** — New hire onboarding. Points to platform: "Your first task: platform will guide you through building a service on the paved road. Ask it questions." New hire ships hello-world service with tests, observability, CI/CD in 2 hours.

**5:00 PM** — Reviews platform's "Architecture Health" report: *ADR-047 adoption: 60% (target 80% in 2 weeks). 12 pattern violations in PRs this week (down from 34). 3 new paved road requests from teams.* Updates quarterly OKRs.

---

*Persona created for AI Software Factory platform — Sprint 1, TASK-002*