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
    metrics: (filters?: Record<string, string | undefined>) =>
      ["agents", "metrics", filters] as const,
    history: (agentId: string) => ["agents", "history", agentId] as const,
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
};
