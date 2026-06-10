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
  AgentType,
  TaskStatus,
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
      api.get<PaginatedResponse<Project>>("/projects", { params: filters }),
    select: (data) => data,
  });
}

export function useProject(id: string) {
  return useQuery({
    queryKey: queryKeys.projects.detail(id),
    queryFn: () => api.get<APIResponse<Project>>(`/projects/${id}`),
    enabled: !!id,
    select: (data) => data.data,
  });
}

export function useCreateProject() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: CreateProjectPayload) =>
      api.post<APIResponse<Project>>("/projects", payload),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.projects.all });
    },
  });
}

// ─── Tasks ───────────────────────────────────────────────────────────────────

export function useTasks(projectId: string) {
  return useQuery({
    queryKey: queryKeys.tasks.list({ project_id: projectId }),
    queryFn: () =>
      api.get<PaginatedResponse<Task>>(`/projects/${projectId}/tasks`),
    enabled: !!projectId,
    select: (data) => data.data,
  });
}

export function useUpdateTask() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      id,
      ...payload
    }: UpdateTaskPayload & { id: string }) =>
      api.patch<APIResponse<Task>>(`/tasks/${id}`, payload),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.tasks.all });
    },
  });
}

// ─── Agents ──────────────────────────────────────────────────────────────────

export function useAgents(filters?: Record<string, string | undefined>) {
  return useQuery({
    queryKey: queryKeys.agents.metrics(filters),
    queryFn: () => api.get<APIResponse<Agent[]>>("/agents", { params: filters }),
    select: (data) => data.data,
  });
}

// ─── Dashboard ────────────────────────────────────────────────────────────────

export function useDashboardMetrics() {
  return useQuery({
    queryKey: ["dashboard", "metrics"],
    queryFn: () => api.get<APIResponse<DashboardMetrics>>("/dashboard/metrics"),
    select: (data) => data.data,
  });
}

export function useRecentActivity() {
  return useQuery({
    queryKey: ["dashboard", "activity"],
    queryFn: () => api.get<PaginatedResponse<ActivityItem>>("/activity", { params: { limit: "10" } }),
    select: (data) => data.data,
  });
}

// ─── Settings ────────────────────────────────────────────────────────────────

export function useSettings() {
  return useQuery({
    queryKey: queryKeys.settings.all,
    queryFn: () => api.get<APIResponse<Record<string, unknown>>>("/settings"),
    select: (data) => data.data,
  });
}

export function useUpdateSettings() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (settings: Record<string, unknown>) =>
      api.patch<APIResponse<Record<string, unknown>>>("/settings", settings),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.settings.all });
    },
  });
}
