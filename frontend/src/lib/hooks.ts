/**
 * React Query hooks for each API domain.
 * Uses the centralized api client and queryKeys factory.
 */
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "./api";
import { queryKeys } from "./queryKeys";
import type {
  Project,
  CreateProjectPayload,
  Task,
  UpdateTaskPayload,
  Agent,
  PaginatedResponse,
  APIResponse,
  DashboardMetrics,
  ActivityItem,
} from "./types";

// ─── Projects ────────────────────────────────────────────────────────────────

export function useProjects(filters?: Record<string, string | undefined>) {
  return useQuery({
    queryKey: queryKeys.projects.list(filters),
    queryFn: () =>
      api.get<APIResponse<{ projects: Project[] }>>("/v1/projects", filters as Record<string, string>),
  });
}

// ─── Tasks ───────────────────────────────────────────────────────────────────

export function useTasks(projectId?: string) {
  return useQuery({
    queryKey: queryKeys.tasks.list(projectId),
    queryFn: () =>
      api.get<APIResponse<{ tasks: Task[] }>>(`/v1/tasks`, { project_id: projectId }),
    enabled: !!projectId,
  });
}

export function useTask(id: string) {
  return useQuery({
    queryKey: queryKeys.tasks.detail(id),
    queryFn: () => api.get<APIResponse<{ task: Task }>>(`/v1/tasks/${id}`),
    enabled: !!id,
  });
}

export function useCreateTask() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: CreateProjectPayload) =>
      api.post<APIResponse<{ task: Task }>>("/v1/tasks", payload),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.tasks.all });
    },
  });
}

export function useUpdateTask() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, ...payload }: UpdateTaskPayload & { id: string }) =>
      api.put<APIResponse<{ task: Task }>>(`/v1/tasks/${id}`, payload),
    onSuccess: (_, vars) => {
      qc.invalidateQueries({ queryKey: queryKeys.tasks.all });
      qc.invalidateQueries({ queryKey: queryKeys.tasks.detail(vars.id) });
    },
  });
}

// ─── Agents ──────────────────────────────────────────────────────────────────

export function useAgents() {
  return useQuery({
    queryKey: queryKeys.agents.list(),
    queryFn: () => api.get<APIResponse<{ agents: Agent[] }>>("/v1/agents"),
  });
}

// ─── Dashboard ───────────────────────────────────────────────────────────────

export function useDashboard() {
  return useQuery({
    queryKey: queryKeys.dashboard.metrics(),
    queryFn: () => api.get<APIResponse<{ metrics: DashboardMetrics }>>("/v1/dashboard"),
  });
}

export function useActivity() {
  return useQuery({
    queryKey: queryKeys.dashboard.activity(),
    queryFn: () => api.get<APIResponse<{ activity: ActivityItem[] }>>("/v1/dashboard/activity"),
  });
}
