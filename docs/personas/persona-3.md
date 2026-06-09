# Persona 3: Marcus Johnson — Tech Lead

## Demographics & Background
- **Name:** Marcus Johnson
- **Role:** Tech Lead / Engineering Manager
- **Company:** FinFlow (Fintech, 45 engineers)
- **Age:** 36
- **Location:** New York, NY
- **Education:** MS Computer Science from MIT, BS from Georgia Tech
- **Experience:** 12 years engineering, 4 as Tech Lead, previously at Stripe and Square

## Goals & Motivations
- Maintain architectural integrity across 6+ microservices
- Enable team autonomy while preventing divergence
- Reduce onboarding time for new engineers from months to weeks
- Balance technical debt paydown with feature delivery
- Establish consistent patterns the team can trust and extend

## Pain Points
1. **Inconsistent code quality** — Different engineers apply different patterns, creating maintenance burden
2. **Knowledge silos** — Critical systems owned by 1-2 people; bus factor is terrifying
3. **Architecture drift** — Decisions made in isolation compound into systemic issues
4. **Review bottleneck** — Spends 40% of time in code reviews, not leading
5. **Documentation rot** — Architecture docs are always outdated; no one trusts them

## How Marcus Interacts with the Platform
- Defines architectural guardrails and coding standards the agents enforce
- Reviews the Architect Agent's proposals before they reach the team
- Uses the platform to spin up reference implementations for new patterns
- Monitors code quality metrics across all agent-generated and human code
- Approves or rejects the Review Agent's suggested refactors

## A Day in the Life

**8:00 AM** — Reviews overnight agent activity. The Architect Agent proposed a new event-driven pattern for the payments service. Marcus validates it against the existing architecture and approves with minor constraints.

**9:00 AM** — Team standup. Two engineers are blocked on legacy auth integration. Marcus points them to the platform's generated adapter pattern — saves a day of exploration.

**10:30 AM** — Deep dive on the new notification service. The Developer Agent has generated a clean implementation following the approved architecture. Marcus does a quick architectural review — approves merge.

**12:00 PM** — Architecture review meeting. Uses the platform's dependency graph to show how the new microservice fits. No surprises — the agents maintained consistency.

**1:30 PM** — Onboards a new senior engineer. Instead of weeks of shadowing, the new hire gets a platform-guided tour: generated docs, running services, test suites. Productive by day 2.

**3:00 PM** — Reviews technical debt dashboard. The platform identified 3 services with circular dependencies. Creates refactor tasks; the Architect Agent proposes solutions.

**4:30 PM** — 1:1 with an engineer struggling with a complex migration. Uses the platform to generate a migration plan with rollback strategy. Problem solved.

**5:30 PM** — Updates the architecture decision log. The platform auto-generates ADRs from agent decisions. Marcus just adds context.

## Success Criteria for Marcus
- New engineers are productive in days, not months
- Architecture decisions are documented and enforced automatically
- Code reviews focus on business logic, not style and patterns
- Technical debt is visible and actively managed
- The team ships features faster because the foundation is solid