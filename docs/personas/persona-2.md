# Persona 2: Product Manager — "Marcus Rivera"

## Name and Role
**Marcus Rivera**, 34 — Senior Product Manager, B2B SaaS platform (50-person product org)

## Demographics and Background
- **Location**: Austin, TX (remote-first company)
- **Education**: BS Human-Computer Interaction, Carnegie Mellon; Product School certification
- **Career Path**: Associate PM at Microsoft (2 years) → PM at Series A startup (3 years) → Senior PM at current company (2 years)
- **Technical Fluency**: Writes SQL for analytics; reads API specs; creates detailed PRDs; cannot write production code
- **Scope**: Owns "Core Platform" product area — auth, billing, org management, APIs (4 engineers, 1 designer, 1 QA)

## Goals and Motivations
| Goal | Motivation |
|------|------------|
| Reduce spec-to-ship cycle from 6 weeks to 2 weeks | Competitors shipping faster; customer churn from missing features |
| Spend 70% time on discovery/strategy, 30% on execution | Currently inverted; drowns in Jira grooming and standups |
| Enable self-serve prototyping for Sales/CS/Marketing | Stakeholders bypass PM for "quick builds" → shadow IT |
| Maintain single source of truth for requirements | Specs drift from implementation; no traceability |
| Build measurable product outcomes, not output | OKRs tied to activation/retention, not story points |

## Pain Points
| Pain Point | Impact | Current Workaround |
|------------|--------|-------------------|
| Writing detailed specs takes 2-3 days per feature | Bottleneck for 4 engineers; context switching | Writes "lite specs" → engineers fill gaps → rework |
| Engineers push back on specs feasibility *after* sprint starts | Sprint disruption; carries spillover | Pre-sprint tech review meetings (adds 1 week) |
| No way to validate ideas with real users before commit | Builds features nobody uses (30% waste) | Fake-door tests in Figma; low fidelity |
| Requirements change mid-sprint; no impact analysis | Scope creep invisible until retro | "Just add it" culture; velocity lies |
| QA finds requirement gaps, not bugs | Late discovery; expensive fixes | Acceptance criteria review meeting (often skipped) |
| Stakeholders demand dates before discovery done | Commits to impossible timelines | Pads estimates; erodes credibility |

## How They Interact with the Platform
- **Primary Interface**: Product workspace (web) — spec editor, roadmap view, analytics
- **Frequency**: Daily 2-4 hours; lives in platform during discovery & planning
- **Key Workflows**:
  1. **Idea → Validated Spec**: Types problem statement → platform generates: user stories, acceptance criteria, data model, API contract, UI wireframes, effort estimate → Marcus refines → one-click "Validate with Users" deploys interactive prototype to staging
  2. **Spec Review Loop**: Shares spec link with Tech Lead + Designer → inline comments → platform tracks resolution → auto-updates linked Linear tickets
  3. **Sprint Planning**: Drags validated specs into sprint → platform auto-breaks into tasks, assigns to Developer agents, generates test plans for QA
  4. **Progress Tracking**: Real-time view: spec → code → test → deploy; blockers surfaced automatically
  5. **Launch & Measure**: Feature flags rollout; platform tracks adoption, error rates, user feedback → feeds back into spec for v2
- **Permissions**: Write specs, approve prototypes, prioritize backlog, trigger deploys to staging
- **Integrations**: Linear (bi-directional), Amplitude/GA (analytics), Figma (design handoff), Slack (notifications), Gong (customer calls → insight extraction)

## Day in the Life
**8:00 AM** — Opens platform. "Billing V2" spec shows: *Designer approved wireframes; Tech Lead flagged webhook retry logic — needs idempotency key design*. Clicks "Resolve with Tech Lead" → platform opens collaborative session with Tech Lead agent.

**9:30 AM** — Discovery call with 3 enterprise customers. Records in Gong. Platform auto-extracts: "Need usage-based billing with custom metrics" → suggests spec additions. Marcus accepts 2, rejects 1 ("out of scope for V2").

**11:00 AM** — Sprint planning. Drags 5 validated specs into "Sprint 23". Platform: *Auto-generated 27 tasks, assigned 3 Developer agents + 1 QA agent. Estimated cycle time: 8 days. Risk: webhook idempotency (high complexity).* Marcus adjusts priority, moves 1 spec to next sprint.

**1:00 PM** — Lunch. Checks phone: *Developer agent #2 blocked on "Stripe webhook signature verification — library version conflict."* Platform suggests fix. Marcus approves; agent unblocks in 3 min.

**2:30 PM** — Stakeholder demo. Sales asks: "Can we show usage-based billing to Acme Corp Friday?" Marcus clicks "Deploy to Demo Env" → shareable URL in 90 sec. Adds feature flag for Acme-only.

**4:00 PM** — Reviews "Spec Health" dashboard: 12 active specs, 3 stale (>14 days no update), 2 with drifting acceptance criteria. Platform suggests: "Archive stale specs? Merge duplicate AC?" Marcus acts on 4 items in 5 min.

**5:30 PM** — End of day. Platform summarizes: *3 specs validated, 1 sprint planned, 2 blockers resolved, 1 demo deployed, 14 Linear tickets synced*. Exports weekly PM report for Monday leadership sync.

---

*Persona created for AI Software Factory platform — Sprint 1, TASK-002*