# AI Software Factory — Wireframes

Low-fidelity structural wireframes for all main platform screens. Each HTML file is a self-contained, interactive mockup that can be opened in a browser.

## Screens

| # | Screen | Route | Description |
|---|--------|-------|-------------|
| 01 | Dashboard | `/dashboard` | Main overview: KPI metrics, agent status, activity feed, project progress |
| 02 | Projects List | `/projects` | Grid of project cards with status, progress bars, agent dots, filters |
| 03 | Project Detail | `/projects/[id]` | Workflow pipeline, task list, agent activity timeline, quality gates |
| 04 | Task Board | `/tasks` | Kanban board with 4 columns: TODO, In Progress, Review, Done |
| 05 | New Project | `/projects/new` | Multi-step form: describe, configure, review. Agent & integration selection |
| 06 | Agent Performance | `/agents` | Per-agent stats, utilization by project, task history, activity timeline |
| 07 | Settings | `/settings` | General config, integrations (GitHub/AWS/Slack), notifications, API keys |
| 08 | Mobile Dashboard | `/dashboard` (375px) | Responsive mobile layout with bottom nav, stacked metrics, activity |

## Design Decisions

- **Sidebar navigation** (260px) on desktop, bottom tab bar on mobile
- **Agent-specific colors** used consistently: PM=#8B5CF6, Architect=#0EA5E9, Developer=#10B981, Review=#F97316, QA=#EC4899, DevOps=#6366F1
- **Workflow pipeline** visualization on Project Detail shows Request→Analysis→Planning→Implementation→Review→Testing→Deploy
- **Quality gates** displayed as status cards on Project Detail (Gate 1-4 from agent-workflow.md)
- **Kanban board** maps directly to the agent workflow states
- **Metrics cards** surface the key KPIs from the vision doc: active projects, completed count, success rate, total spend

## Responsive Breakpoints

- Desktop: 1024px+ (sidebar + content)
- Tablet: 768px (collapsed sidebar)
- Mobile: 375px (bottom nav, stacked layout)

## Accessibility Notes

- All interactive elements have visible focus states
- Color is never the sole indicator of status (badges + text + icons)
- Form labels are explicitly associated with inputs
- Color contrast meets WCAG 2.2 AA (checked against design system tokens)

## Next Steps

These wireframes should inform the component library and page implementations in the Next.js app. Key components to extract:

1. SidebarNav — reusable navigation component
2. MetricCard — KPI display with value, label, and trend
3. ProgressCard — labeled progress bar
4. ActivityFeed — timeline of agent events
5. TaskCard — kanban card with agent, priority, project
6. PipelineVisual — workflow step indicator
7. QualityGateCard — gate status with checklist
8. AgentBadge — colored dot + label for agent types
9. StatusBadge — status pill (active/done/queued/failed)
10. SettingsSection — form group with label, input, hint
