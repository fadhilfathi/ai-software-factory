# Database Schema

## Design Decisions
- UUID PKs over serial (distributed safety)
- ON DELETE CASCADE for child records, SET NULL for user/owner FKs
- UUID[] for task.dependencies (no join table, app-layer FK enforcement)
- 6 composite indexes for board/review/deployment query patterns
- Partitioning and retention strategy from the spec

## Migration Order

| #   | Table             | Dependencies                       |
|-----|-------------------|------------------------------------|
| 001 | users             | —                                  |
| 002 | teams             | users                              |
| 003 | team_members      | teams, users                       |
| 004 | projects          | users, teams                       |
| 005 | agents            | projects                           |
| 006 | tasks             | projects, users, agents (self-ref) |
| 007 | code_artifacts    | tasks, projects                    |
| 008 | reviews           | tasks, projects, users, agents     |
| 009 | deployments       | projects (self-ref)                |
| 010 | notifications     | users                              |
| 011 | audit_logs        | users                              |
| 012 | webhook_configs   | projects                           |
| 013 | Composite indexes | all tables                         |

Two tables derived from ER diagram only (no SQL in the spec):
- team_members — UNIQUE(team_id, user_id) constraint prevents duplicates
- webhook_configs — active boolean, events JSONB, last_used_at timestamp
