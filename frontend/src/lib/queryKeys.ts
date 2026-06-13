// Centralised React Query keys. Keep keys narrow + serialisable so cache
// invalidation is predictable.

import type {
  AgentListFilters,
  CapabilityListFilters,
  DeliverableListFilters,
  ExecutionListFilters,
  TaskListFilters,
} from "./types"

export const queryKeys = {
  agents: {
    all: ["agents"] as const,
    list: (filters: AgentListFilters = {}) =>
      [...queryKeys.agents.all, "list", filters] as const,
    detail: (id: string) =>
      [...queryKeys.agents.all, "detail", id] as const,
    history: (id: string) =>
      [...queryKeys.agents.all, "history", id] as const,
    metrics: (filters: AgentListFilters = {}) =>
      [...queryKeys.agents.all, "metrics", filters] as const,
    capabilities: (id: string) =>
      [...queryKeys.agents.all, "capabilities", id] as const,
  },
  capabilities: {
    all: ["capabilities"] as const,
    list: (filters: CapabilityListFilters = {}) =>
      [...queryKeys.capabilities.all, "list", filters] as const,
  },
  executions: {
    all: ["executions"] as const,
    list: (filters: ExecutionListFilters = {}) =>
      [...queryKeys.executions.all, "list", filters] as const,
  },
  deliverables: {
    all: ["deliverables"] as const,
    list: (filters: DeliverableListFilters = {}) =>
      [...queryKeys.deliverables.all, "list", filters] as const,
    detail: (id: string) =>
      [...queryKeys.deliverables.all, "detail", id] as const,
    versions: (id: string) =>
      [...queryKeys.deliverables.all, "versions", id] as const,
  },
  projects: {
    all: ["projects"] as const,
    list: (filters?: Record<string, unknown>) =>
      [...queryKeys.projects.all, "list", filters ?? {}] as const,
    detail: (id: string) =>
      [...queryKeys.projects.all, "detail", id] as const,
  },
  dashboard: {
    all: ["dashboard"] as const,
    activity: () => [...queryKeys.dashboard.all, "activity"] as const,
  },
  tasks: {
    all: ["tasks"] as const,
    list: (filters: TaskListFilters = {}) =>
      [...queryKeys.tasks.all, "list", filters] as const,
    detail: (id: string) =>
      [...queryKeys.tasks.all, "detail", id] as const,
    history: (id: string) =>
      [...queryKeys.tasks.all, "history", id] as const,
    executions: (id: string) =>
      [...queryKeys.tasks.all, "executions", id] as const,
    deliverables: (id: string) =>
      [...queryKeys.tasks.all, "deliverables", id] as const,
  },
} as const
