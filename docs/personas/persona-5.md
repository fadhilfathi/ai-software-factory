# Persona 5: David Rodriguez — QA Engineer

## Demographics & Background
- **Name:** David Rodriguez
- **Role:** Senior QA Engineer / Test Architect
- **Company:** EduPlatform (EdTech, 25 engineers, 5 QA)
- **Age:** 33
- **Location:** Chicago, IL
- **Education:** BS Software Engineering from University of Illinois
- **Experience:** 9 years QA, 3 as Test Architect, previously at Salesforce and Coursera

## Goals & Motivations
- Shift quality left — catch bugs in requirements, not production
- Build test infrastructure that scales with the product
- Automate the boring stuff; focus on exploratory and edge-case testing
- Have visibility into what's being built before it reaches QA
- Reduce the regression cycle from days to minutes

## Pain Points
1. **Late involvement** — QA gets features 2 days before release; no time for thorough testing
2. **Flaky tests** — 30% of CI failures are test infrastructure, not product bugs
3. **No shared test data** — Every test creates its own data; cleanup is a nightmare
4. **Manual regression** — Critical paths still tested by hand; doesn't scale
5. **Communication gaps** — Developers don't know what QA tests; QA doesn't know what changed

## How David Interacts with the Platform
- Reviews the PM Agent's acceptance criteria for testability before development starts
- Uses the QA Agent to generate test plans, test cases, and automation scaffolding
- Defines test strategies and quality gates the platform enforces
- Monitors the quality dashboard: coverage, flakiness, defect trends, escape rate
- Collaborates with the Developer Agent to make code more testable

## A Day in the Life

**8:00 AM** — Reviews overnight test results. The QA Agent ran 2,400 tests across 4 services. 3 flaky tests identified — auto-quarantined. 1 genuine failure in the new grading service.

**8:30 AM** — Investigates the failure. The QA Agent generated a minimal reproduction case. David traces it to a timezone edge case in the deadline calculation. Creates a bug ticket; the Developer Agent already has a fix proposed.

**9:30 AM** — Sprint planning. Reviews upcoming stories for testability. Flags 2 stories with vague acceptance criteria — works with the PM Agent to sharpen them before development starts.

**10:30 AM** — Test infrastructure work. The platform's test data factory needs a new entity: "Course Cohort." David defines the factory; the QA Agent generates builders, fixtures, and cleanup logic.

**12:00 PM** — Lunch. Checks the quality dashboard on phone. Coverage up 2% this sprint. Escape rate (bugs found in prod) down to 0.8%.

**1:00 PM** — Exploratory testing session on the new "Peer Review" feature. Uses the platform's test session recorder — captures steps, screenshots, network logs. Finds a permissions edge case. The QA Agent converts the session into an automated regression test.

**2:30 PM** — Reviews the Developer Agent's generated unit tests for the new notification service. Adds 3 integration tests for the email/SMS/push channels. Suggests a contract test for the provider abstraction.

**3:30 PM** — Quality gate review for the release candidate. All gates pass: coverage >85%, 0 critical bugs, flakiness <1%, performance within baseline. Approves promotion to production.

**4:00 PM** — Post-release monitoring. The platform's synthetic tests run every 5 minutes in prod. David watches the dashboard — all green.

**4:30 PM** — Retrospective prep. Exports quality metrics: defect detection rate, mean time to detect, test execution time trends. Data-driven improvement.

## Success Criteria for David
- QA is involved at story refinement, not pre-release
- 90% of regression testing is automated and runs in <10 minutes
- Flaky tests are detected and quarantined automatically
- Developers get test feedback in their IDE, not days later in CI
- Quality metrics drive decisions, not gut feel
- Zero critical escapes to production