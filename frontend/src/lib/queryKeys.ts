/**
 * Centralized React Query key factory.
 * Provides consistent cache keys for all API queries.
 */
export const queryKeys = {
  projects: {
    all: ["projects"] as const,
    list: (filters?: Record<string, string | undefined>) =>
      ["projects", "list", filters] as const,
    detail: (id: string) => ["projects", "detail", id] as const,
  },
  tasks: {
    all: ["tasks"] as const,
    list: (filters?: Record<string, string | undefined>) =>
      ["tasks", "list", filters] as const,
    detail: (id: string) => ["tasks", "detail", id] as const,
  },
  agents: {
    all: ["agents"] as const,
    list: (filters?: Record<string, string | undefined>) =>
      ["agents", "list", filters] as const,
    detail: (id: string) => ["agents", "detail", id] as const,
    metrics: (filters?: Record<string, string | undefined>) =>
      ["agents", "metrics", filters] as const,
    history: (agentId: string) => ["agents", "history", agentId] as const,
  },
  dashboard: {
    metrics: () => ["dashboard"] as const,
    activity: () => ["dashboard", "activity"] as const,
  },
  reviews: {
    all: ["reviews"] as const,
    detail: (id: string) => ["reviews", "detail", id] as const,
  },
  deployments: {
    all: ["deployments"] as const,
    detail: (id: string) => ["deployments", "detail", id] as const,
  },
  vision: {
    document: ["vision", "document"] as const,
    history: ["vision", "history"] as const,
    diff: (from: number, to: number) => ["vision", "diff", from, to] as const,
  },
  settings: {
    all: ["settings"] as const,
    section: (name: string) => ["settings", "section", name] as const,
  },
  designSystem: {
    tokens: ["design-system", "tokens"] as const,
  },
  code: {
    generate: ["code", "generate"] as const,
  },
  executions: {
    all: ["executions"] as const,
    list: (filters?: Record<string, string | undefined>) =>
      ["executions", "list", filters] as const,
    detail: (id: string) => ["executions", "detail", id] as const,
  },
  deliverables: {
    all: ["deliverables"] as const,
    list: (filters?: Record<string, string | undefined>) =>
      ["deliverables", "list", filters] as const,
    detail: (id: string) => ["deliverables", "detail", id] as const,
  },
} as const;
