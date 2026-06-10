# Persona 5: QA Engineer — "Samara Okafor"

## Name and Role
**Samara Okafor**, 35 — Senior QA Engineer / Test Architect (leads quality for 3 product areas, 15 engineers)

## Demographics and Background
- **Location**: Atlanta, GA (hybrid 2 days/week)
- **Education**: BS Computer Engineering, Georgia Tech; ISTQB Advanced Test Manager; AWS Certified Developer
- **Career Path**: Manual QA at enterprise software co (3 years) → SDET at fintech startup (4 years, built test framework from scratch) → Senior QA at current Series C (2 years)
- **Technical Fluency**: Expert in test architecture, automation frameworks (Playwright, Cypress, k6, Postman), CI/CD pipelines, contract testing, chaos engineering; writes TypeScript/Go/Python; reads all codebases
- **Scope**: Owns "Quality Platform" — shared test infrastructure, standards, tooling; embeds with 3 squads (2 weeks each rotation)

## Goals and Motivations
| Goal | Motivation |
|------|------------|
| Shift quality left: catch bugs at spec, not production | Production bugs cost 100x more; team burns out on hotfixes |
| Eliminate manual regression testing | 2 days/sprint on manual regression; zero value add |
| Make testing so easy developers do it by default | "QA bottleneck" narrative; wants shared ownership |
| Provide real-time quality signal to PM/Tech Lead | Decisions made without quality data; surprises at launch |
| Build reusable test patterns as platform capabilities | Reinventing test setup per project; 30% waste |

## Pain Points
| Pain Point | Impact | Current Workaround |
|------------|--------|-------------------|
| Test plans written after code; miss requirements gaps | Bugs found in staging; expensive rework | "Review PRs for testability" — too late |
| Flaky tests erode trust; 20% false positives | Engineers ignore CI failures; merge anyway | "Quarantine flaky tests" — never fixed |
| No contract testing between frontend/backend | Integration bugs only in e2e (slow, brittle) | Manual API testing in Postman |
| Test data management is a nightmare | Tests pollute each other; CI non-deterministic | Snapshots/seed scripts; brittle maintenance |
| Performance testing only before major releases | Production incidents from gradual degradation | "Load test once a quarter" |
| QA seen as gatekeeper, not enabler | Adversarial dynamic; devs hide changes | "Shift left" workshops; cultural, not systemic |

## How They Interact with the Platform
- **Primary Interface**: QA workspace (web) + VS Code extension + CLI for test authoring
- **Frequency**: Daily 3-4 hours authoring/maintaining; continuous monitoring via dashboard
- **Key Workflows**:
  1. **Spec-Driven Test Generation**: Reads validated spec in platform → auto-generates: unit test templates, contract tests (Pact), API test scenarios, e2e user journeys, performance benchmarks → Samara reviews, adds edge cases, publishes to shared test library
  2. **Test Plan as Code**: Writes test plan in platform (markdown + executable annotations) → platform: tracks coverage against spec, generates traceability matrix, alerts on gaps, produces launch readiness report
  3. **Flake Detection & Healing**: Platform monitors all test runs → detects flakes statistically → auto-bisects to root cause → suggests fix (selector, timing, data) → Samara approves → platform applies across repos
  4. **Test Data Factory**: Defines data models in platform → generates: type-safe factories, database seeders, API mock servers, anonymized production snapshots → versioned, branch-scoped, ephemeral
  5. **Continuous Quality Dashboard**: Real-time view per squad: spec coverage, test pass rate, flake rate, performance trends, defect escape rate → PM/Tech Lead subscribe to alerts
  6. **Production → Test Loop**: Incident in Datadog → platform creates "Regression Test Candidate" → Samara converts to contract/e2e test → adds to paved road → prevents recurrence
- **Permissions**: Write test plans, manage test infrastructure, configure quality gates, approve/reject deployments to staging/prod
- **Integrations**: GitHub/GitLab (PR checks, deployment gates), Playwright/Cypress/k6 (execution), Pact (contract), Datadog/New Relic (observability), Linear/Jira (defects), Slack (alerts), TestRail/Xray (legacy migration)

## Day in the Life
**7:30 AM** — Opens Quality Dashboard. Squad "Growth" shows: *Spec coverage: 87% (target 90%). Flake rate: 3.2% (up from 1.8%). 2 regression test candidates from last week's incidents.* Clicks "Investigate Flakes."

**8:00 AM** — Platform flake analysis: *`checkout-flow.test.ts` fails 12% on "Stripe webhook timeout" — root cause: test uses fixed 5s wait, Staging Stripe latency varies.* Platform suggests: "Replace with polling waiter + configurable timeout." Samara approves; platform patches 4 repos using same pattern.

**9:30 AM** — Sprint planning with Growth squad. PM presents "Usage Metrics API" spec. Samara: "Platform shows 3 uncovered acceptance criteria: error rate threshold, pagination edge cases, rate limit headers." Adds to test plan. Platform generates contract tests from OpenAPI spec.

**11:00 AM** — Authors performance benchmark for new API. Platform: *Generates k6 script from OpenAPI: ramp to 1000 RPS, measures p50/p95/p99, error rate, compares to baseline.* Commits to `performance/` folder; CI runs nightly.

**12:30 PM** — Lunch. Checks phone: *Deployment gate for "Billing V2" to staging — Quality check: 94% spec coverage, 0 critical defects, 2 flaky tests (quarantined), performance within baseline.* Taps "Approve."

**1:30 PM** — Incident retrospective (SEV-1: billing webhook duplicate charges). Platform created Regression Test Candidate: *"Idempotency key validation missing for Stripe webhook."* Samara converts to contract test: adds to `webhook-contracts` paved road package. Platform backfills to 3 other webhook handlers.

**3:00 PM** — Reviews Developer agent test output for "Usage Metrics" task. Platform generated: 12 unit, 8 contract, 5 e2e. Samara adds: 3 mutation test survivors (edge cases), 1 chaos test (DB connection pool exhaustion). Approves; platform merges to test library.

**4:30 PM** — Updates "Quality Gates" config for next quarter: *Raise spec coverage target to 95%. Add mutation testing threshold (80%). Require contract test for all new APIs.* Platform validates against current repos: *3 repos would fail — creates remediation tasks.*

**5:00 PM** — End of day. Platform summary: *2 flakes fixed, 1 regression test added, 1 performance baseline updated, 3 quality gate changes proposed, 1 deployment approved.* Exports weekly quality report for engineering all-hands.

---

*Persona created for AI Software Factory platform — Sprint 1, TASK-002*