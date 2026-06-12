/**
 * React Query hooks for each API domain.
 * Matches actual backend API routes and response shapes.
 *
 * Response shapes from the Go backend:
 *   - List endpoints return `PaginatedResponse<T>`: { data: T[], pagination: {...} }
 *   - Single-item endpoints return the item directly (no wrapper)
 */
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "./api";
import { queryKeys } from "./queryKeys";
import type {
  Project,
  CreateProjectPayload,
  Task,
  CreateTaskPayload,
  UpdateTaskPayload,
  UpdateTaskStatusPayload,
  Agent,
  DashboardMetrics,
  ActivityItem,
  User,
  LoginPayload,
  RegisterPayload,
  CodeGeneratePayload,
  Review,
  ReviewPayload,
  Deployment,
  DeploymentPayload,
  PaginatedResponse,
} from "./types";

// ─── Projects ────────────────────────────────────────────────────────────────

export function useProjects(filters?: Record<string, string | undefined>) {
  return useQuery({
    queryKey: queryKeys.projects.list(filters),
    queryFn: () =>
      api.get<PaginatedResponse<Project>>("/v1/projects", { params: filters }),
  });
}

/** Single project detail. Backend returns the project object directly. */
export function useProject(id: string) {
  return useQuery({
    queryKey: queryKeys.projects.detail(id),
    queryFn: () => api.get<Project>(`/v1/projects/${id}`),
    enabled: !!id,
  });
}

export function useCreateProject() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: CreateProjectPayload) =>
      api.post<Project>("/v1/projects", payload),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.projects.all });
    },
  });
}

export function useUpdateProject() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, ...payload }: { id: string } & Partial<CreateProjectPayload>) =>
      api.put<Project>(`/v1/projects/${id}`, payload),
    onSuccess: (_, vars) => {
      qc.invalidateQueries({ queryKey: queryKeys.projects.all });
      qc.invalidateQueries({ queryKey: queryKeys.projects.detail(vars.id) });
    },
  });
}

export function useDeleteProject() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.delete<void>(`/v1/projects/${id}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.projects.all });
    },
  });
}

// ─── Tasks ───────────────────────────────────────────────────────────────────

/** Tasks for a project. Select unwraps paginated response into the flat array. */
export function useTasks(projectId?: string) {
  return useQuery({
    queryKey: queryKeys.tasks.list({ project_id: projectId }),
    queryFn: () =>
      api.get<PaginatedResponse<Task>>(`/v1/projects/${projectId}/tasks`),
    enabled: !!projectId,
    select: (res) => res.data,
  });
}

export function useTask(id: string) {
  return useQuery({
    queryKey: queryKeys.tasks.detail(id),
    queryFn: () => api.get<Task>(`/v1/tasks/${id}`),
    enabled: !!id,
  });
}

/** Backend: POST /v1/projects/{projectId}/tasks */
export function useCreateTask() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ projectId, ...payload }: { projectId: string } & CreateTaskPayload) =>
      api.post<Task>(`/v1/projects/${projectId}/tasks`, payload),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.tasks.all });
    },
  });
}

/** Backend: PUT /v1/tasks/{id} — general task update */
export function useUpdateTask() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, ...payload }: { id: string } & UpdateTaskPayload) =>
      api.put<Task>(`/v1/tasks/${id}`, payload),
    onSuccess: (_, vars) => {
      qc.invalidateQueries({ queryKey: queryKeys.tasks.all });
      qc.invalidateQueries({ queryKey: queryKeys.tasks.detail(vars.id) });
    },
  });
}

/** Backend: PATCH /v1/tasks/{id}/status — Kanban status transition */
export function useUpdateTaskStatus() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, ...payload }: { id: string } & UpdateTaskStatusPayload) =>
      api.patch<Task>(`/v1/tasks/${id}/status`, payload),
    onMutate: async (variables) => {
      await qc.cancelQueries({ queryKey: queryKeys.tasks.all });

      const previousTasksList = qc.getQueriesData({ queryKey: queryKeys.tasks.all });
      const previousTaskDetail = qc.getQueryData(queryKeys.tasks.detail(variables.id));

      qc.setQueriesData({ queryKey: queryKeys.tasks.all }, (old: any) => {
        if (!old || !old.data) return old;
        return {
          ...old,
          data: old.data.map((task: Task) =>
            task.id === variables.id ? { ...task, status: variables.status } : task
          ),
        };
      });

      if (previousTaskDetail) {
        qc.setQueryData(queryKeys.tasks.detail(variables.id), {
          ...(previousTaskDetail as any),
          status: variables.status,
        });
      }

      return { previousTasksList, previousTaskDetail };
    },
    onError: (_, variables, context) => {
      if (context?.previousTasksList) {
        context.previousTasksList.forEach(([queryKey, data]) => {
          qc.setQueryData(queryKey, data);
        });
      }
      if (context?.previousTaskDetail) {
        qc.setQueryData(queryKeys.tasks.detail(variables.id), context.previousTaskDetail);
      }
    },
    onSettled: (_, __, variables) => {
      qc.invalidateQueries({ queryKey: queryKeys.tasks.all });
      qc.invalidateQueries({ queryKey: queryKeys.tasks.detail(variables.id) });
    },
  });
}

export function useDeleteTask() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.delete<void>(`/v1/tasks/${id}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.tasks.all });
    },
  });
}

// ─── Agents ──────────────────────────────────────────────────────────────────

/** Backend: GET /v1/agents. Unwraps paginated response into flat array. */
export function useAgents() {
  return useQuery({
    queryKey: queryKeys.agents.list(),
    queryFn: () =>
      api.get<PaginatedResponse<Agent>>("/v1/agents").then((res) => res.data),
  });
}

export function useAgent(id: string) {
  return useQuery({
    queryKey: queryKeys.agents.detail(id),
    queryFn: () => api.get<Agent>(`/v1/agents/${id}`),
    enabled: !!id,
  });
}

/** Backend: POST /v1/agents/spawn */
export function useCreateAgent() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: {
      type: string;
      project_id?: string;
      config?: { model?: string; temperature?: number };
    }) => api.post<Agent>("/v1/agents/spawn", payload),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.agents.all });
    },
  });
}

export function useAgentMetrics(filters?: Record<string, string | undefined>) {
  return useQuery({
    queryKey: queryKeys.agents.metrics(filters),
    queryFn: () =>
      api.get<{ metrics: unknown }>("/v1/agents/metrics", { params: filters }),
  });
}

/** Backend: POST /v1/agents/{id}/assign */
export function useAssignTask() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      agentId,
      taskId,
      priority,
      context,
    }: {
      agentId: string;
      taskId: string;
      priority?: string;
      context?: Record<string, unknown>;
    }) => api.post(`/v1/agents/${agentId}/assign`, { task_id: taskId, priority, context }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.agents.all });
    },
  });
}

// ─── Dashboard ───────────────────────────────────────────────────────────────

export function useDashboardMetrics() {
  return useQuery({
    queryKey: queryKeys.dashboard.metrics(),
    queryFn: () => api.get<{ metrics: DashboardMetrics }>("/v1/dashboard"),
  });
}

/** @deprecated Use useDashboardMetrics instead */
export const useDashboard = useDashboardMetrics;

/**
 * Activity feed. select unwraps to flat array for consumer components.
 * Note: no backend endpoint for this yet — will error gracefully.
 */
export function useActivity() {
  return useQuery({
    queryKey: queryKeys.dashboard.activity(),
    queryFn: () => api.get<{ activity: ActivityItem[] }>("/v1/dashboard/activity"),
    select: (res) => res.activity,
  });
}

/** Alias for useActivity — resolves dashboard page reference. */
export const useRecentActivity = useActivity;

// ─── Auth ────────────────────────────────────────────────────────────────────

/** Backend: POST /v1/auth/login */
export function useLogin() {
  return useMutation({
    mutationFn: (payload: LoginPayload) =>
      api.post<{ user: User; access_token: string }>("/v1/auth/login", payload),
  });
}

/** Backend: POST /v1/users/register */
export function useRegister() {
  return useMutation({
    mutationFn: (payload: RegisterPayload) =>
      api.post<{ user: User; access_token: string }>("/v1/users/register", payload),
  });
}

/** Backend: GET /v1/users/me */
export function useCurrentUser() {
  return useQuery({
    queryKey: ["users", "me"],
    queryFn: () => api.get<{ user: User }>("/v1/users/me"),
  });
}

// ─── Code Generation ─────────────────────────────────────────────────────────

/** Backend: POST /v1/code/generate */
export function useGenerateCode() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: CodeGeneratePayload) =>
      api.post<{ result: string }>("/v1/code/generate", payload),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.code.generate });
    },
  });
}

// ─── Reviews ─────────────────────────────────────────────────────────────────

/** Backend: GET /v1/reviews/{id} */
export function useReview(id: string) {
  return useQuery({
    queryKey: queryKeys.reviews.detail(id),
    queryFn: () => api.get<Review>(`/v1/reviews/${id}`),
    enabled: !!id,
  });
}

/** Backend: POST /v1/reviews */
export function useCreateReview() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: ReviewPayload) =>
      api.post<Review>("/v1/reviews", payload),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.reviews.all });
    },
  });
}

// ─── Deployments ─────────────────────────────────────────────────────────────

/** Backend: GET /v1/deployments/{id} */
export function useDeployment(id: string) {
  return useQuery({
    queryKey: queryKeys.deployments.detail(id),
    queryFn: () => api.get<Deployment>(`/v1/deployments/${id}`),
    enabled: !!id,
  });
}

/** Backend: POST /v1/deployments */
export function useCreateDeployment() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: DeploymentPayload) =>
      api.post<Deployment>("/v1/deployments", payload),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.deployments.all });
    },
  });
}

// ─── Webhooks ────────────────────────────────────────────────────────────────

/** Backend: POST /v1/webhooks */
export function useRegisterWebhook() {
  return useMutation({
    mutationFn: (payload: { url: string; events: string[]; secret?: string }) =>
      api.post<{ id: string; url: string }>("/v1/webhooks", payload),
  });
}
