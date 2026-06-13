"use client"

import {
  useMutation,
  useQuery,
  useQueryClient,
  type UseMutationResult,
  type UseQueryResult,
} from "@tanstack/react-query"

import { api } from "./api"
import { queryKeys } from "./queryKeys"
import { useProjectFilters } from "@/hooks/useProjectFilters"
import type {
  Agent,
  AgentCapability,
  AgentListFilters,
  AgentMetadata,
  AgentMetrics,
  ApiEnvelope,
  AssignmentEvent,
  AssignTaskPayload,
  AssignTaskResult,
  CapabilityCatalogItem,
  CapabilityListFilters,
  CreateAgentPayload,
  CreateTaskPayload,
  Deliverable,
  DeliverableListFilters,
  DeliverableVersion,
  Execution,
  ExecutionListFilters,
  PaginatedResponse,
  Project,
  Task,
  TaskHistoryResponse,
  TaskListFilters,
  TaskStatus,
  UpdateAgentPayload,
  UpdateTaskStatusPayload,
} from "./types"

/* ---------- Project context helper ---------- */

/**
 * The /v1/agents/* routes (per Lead, 2026-06-12) are project-scoped and
 * require an `X-Project-ID` header sourced from useProjectFilters(). Do NOT
 * send the header with an empty/synthetic value — the backend rejects it.
 * The page-level ProjectPickerGate is responsible for gating the UI so this
 * header is always present when an agent request fires.
 */
function projectHeaders(
  projectId: string | null | undefined,
): { headers?: Record<string, string> } | undefined {
  if (!projectId) return undefined
  return { headers: { "X-Project-ID": projectId } }
}

function withCapabilityArray(
  filters: AgentListFilters,
): AgentListFilters & { capability?: string } {
  // Spec §1.2 takes a single `capability` query param. If the UI passes
  // multiple, send the first and let the user re-pick (we surface this in
  // the UI as "filter to one capability at a time"). Keeping the type loose
  // here so multi-select is supported client-side even though the wire
  // format is single-valued.
  if (Array.isArray(filters.capability)) {
    const [first] = filters.capability
    const { capability: _omit, ...rest } = filters
    return { ...rest, ...(first ? { capability: first } : {}) } as AgentListFilters & {
      capability?: string
    }
  }
  return filters as AgentListFilters & { capability?: string }
}

/* ---------- Agents ---------- */

export function useAgents(
  filters: AgentListFilters = {},
): UseQueryResult<PaginatedResponse<Agent>> {
  const { projectId } = useProjectFilters()
  const wireFilters = withCapabilityArray(filters)
  return useQuery({
    queryKey: queryKeys.agents.list({ ...wireFilters, project_id: projectId ?? undefined }),
    queryFn: () =>
      api.get<PaginatedResponse<Agent>>("/v1/agents", {
        params: { ...wireFilters, project_id: projectId ?? undefined },
        ...projectHeaders(projectId),
      }),
    enabled: !!projectId,
  })
}

export function useAgent(id: string | undefined): UseQueryResult<Agent> {
  const { projectId } = useProjectFilters()
  return useQuery({
    queryKey: queryKeys.agents.detail(id ?? ""),
    queryFn: () =>
      api
        .get<ApiEnvelope<Agent> | Agent>(`/v1/agents/${id}`, {
          ...projectHeaders(projectId),
        })
        .then((res) => ("data" in res ? res.data : res)),
    enabled: !!id && !!projectId,
  })
}

export function useAgentHistory(
  id: string | undefined,
): UseQueryResult<PaginatedResponse<{
  id: string
  type: string
  at: string
  title: string
  description?: string
  [key: string]: unknown
}>> {
  const { projectId } = useProjectFilters()
  return useQuery({
    queryKey: queryKeys.agents.history(id ?? ""),
    queryFn: () =>
      api.get<PaginatedResponse<{
        id: string
        type: string
        at: string
        title: string
        description?: string
        [key: string]: unknown
      }>>(`/v1/agents/${id}/history`, { ...projectHeaders(projectId) }),
    enabled: !!id && !!projectId,
  })
}

export function useAgentMetrics(
  filters: AgentListFilters = {},
): UseQueryResult<AgentMetrics> {
  const { projectId } = useProjectFilters()
  const wireFilters = withCapabilityArray(filters)
  return useQuery({
    queryKey: queryKeys.agents.metrics({ ...wireFilters, project_id: projectId ?? undefined }),
    queryFn: () =>
      api.get<AgentMetrics>("/v1/agents/metrics", {
        params: { ...wireFilters, project_id: projectId ?? undefined },
        ...projectHeaders(projectId),
      }),
    enabled: !!projectId,
  })
}

export function useAgentCapabilities(
  id: string | undefined,
): UseQueryResult<AgentCapability[]> {
  // Per Developer-01 (2026-06-12), `/v1/agents/:id/capabilities` is one of
  // the global catalog-style exceptions — the backend does NOT require
  // X-Project-ID on this route (the capabilities themselves are global,
  // even though the parent agent is project-scoped). So we deliberately
  // omit the header here, and the hook is enabled even when the project
  // switcher is in "all projects" mode.
  return useQuery({
    queryKey: queryKeys.agents.capabilities(id ?? ""),
    queryFn: () =>
      api
        .get<ApiEnvelope<AgentCapability[]> | AgentCapability[]>(
          `/v1/agents/${id}/capabilities`,
        )
        .then((res) => (Array.isArray(res) ? res : res.data)),
    enabled: !!id,
  })
}

export function useCreateAgent(): UseMutationResult<
  Agent,
  Error,
  CreateAgentPayload
> {
  const queryClient = useQueryClient()
  const { projectId } = useProjectFilters()
  return useMutation({
    mutationFn: (payload: CreateAgentPayload) =>
      api.post<ApiEnvelope<Agent> | Agent>(
        "/v1/agents",
        payload,
        projectHeaders(projectId),
      ).then((res) => ("data" in res ? res.data : res)),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.agents.all })
    },
  })
}

export function useUpdateAgent(): UseMutationResult<
  Agent,
  Error,
  { id: string; payload: UpdateAgentPayload }
> {
  const queryClient = useQueryClient()
  const { projectId } = useProjectFilters()
  return useMutation({
    mutationFn: ({ id, payload }) =>
      api.put<ApiEnvelope<Agent> | Agent>(
        `/v1/agents/${id}`,
        payload,
        projectHeaders(projectId),
      ).then((res) => ("data" in res ? res.data : res)),
    onSuccess: (agent) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.agents.all })
      queryClient.setQueryData(queryKeys.agents.detail(agent.id), agent)
    },
  })
}

export function useDeleteAgent(): UseMutationResult<void, Error, string> {
  const queryClient = useQueryClient()
  const { projectId } = useProjectFilters()
  return useMutation({
    mutationFn: (id: string) =>
      api.delete<void>(`/v1/agents/${id}`, projectHeaders(projectId)),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.agents.all })
    },
  })
}

/* ---------- Capabilities (catalog) ---------- */

export function useCapabilities(
  filters: CapabilityListFilters = {},
): UseQueryResult<PaginatedResponse<CapabilityCatalogItem>> {
  return useQuery({
    queryKey: queryKeys.capabilities.list(filters),
    queryFn: () =>
      api.get<PaginatedResponse<CapabilityCatalogItem>>(
        "/v1/capabilities",
        { params: filters },
      ),
    staleTime: 5 * 60 * 1000, // catalog changes rarely
  })
}

/* ---------- Executions ---------- */

export function useExecutions(
  filters: ExecutionListFilters = {},
): UseQueryResult<PaginatedResponse<Execution>> {
  const { projectId } = useProjectFilters()
  return useQuery({
    queryKey: queryKeys.executions.list({ ...filters, project_id: projectId ?? undefined }),
    queryFn: () =>
      api.get<PaginatedResponse<Execution>>(
        "/v1/executions",
        {
          params: { ...filters, project_id: projectId ?? undefined },
          ...projectHeaders(projectId),
        },
      ),
    enabled: !!projectId,
  })
}

/* ---------- Deliverables ---------- */

export function useDeliverables(
  filters: DeliverableListFilters = {},
): UseQueryResult<PaginatedResponse<Deliverable>> {
  const { projectId } = useProjectFilters()
  return useQuery({
    queryKey: queryKeys.deliverables.list({ ...filters, project_id: projectId ?? undefined }),
    queryFn: () =>
      api.get<PaginatedResponse<Deliverable>>(
        "/v1/deliverables",
        {
          params: { ...filters, project_id: projectId ?? undefined },
          ...projectHeaders(projectId),
        },
      ),
    enabled: !!projectId,
  })
}

/**
 * Single deliverable. /v1/deliverables/:id is not in the Sprint 1-3
 * spec (the spec only lists the list endpoint); we still expose the
 * hook so the existing detail page can render. If the backend returns
 * 404 the error surfaces to the page.
 */
export function useDeliverable(
  id: string | undefined,
): UseQueryResult<Deliverable> {
  const { projectId } = useProjectFilters()
  return useQuery({
    queryKey: queryKeys.deliverables.detail(id ?? ""),
    queryFn: () =>
      api
        .get<ApiEnvelope<Deliverable> | Deliverable>(
          `/v1/deliverables/${id}`,
          { ...projectHeaders(projectId) },
        )
        .then((res) => ("data" in res ? res.data : res)),
    enabled: !!id,
  })
}

/**
 * Version history for a single deliverable. The endpoint returns the
 * versions in DESC order (newest first) per the Lead's brief; the
 * page renders them in that order and lets the user pick any two to
 * diff. Append-only by contract — a PUT to /v1/deliverables/:id
 * creates a new row here, never overwrites.
 */
export function useDeliverableVersions(
  id: string | undefined,
): UseQueryResult<DeliverableVersion[]> {
  const { projectId } = useProjectFilters()
  return useQuery({
    queryKey: queryKeys.deliverables.versions(id ?? ""),
    queryFn: async () => {
      const res = await api.get<
        DeliverableVersion[] | { data: DeliverableVersion[] } | PaginatedResponse<DeliverableVersion>
      >(`/v1/deliverables/${id}/versions`, { ...projectHeaders(projectId) })
      if (Array.isArray(res)) return res
      if ("data" in res) {
        return (res as { data: DeliverableVersion[] }).data
      }
      return (res as unknown) as DeliverableVersion[]
    },
    enabled: !!id,
  })
}

/* ---------- Projects ---------- */

/**
 * Lists projects available to the current user. The /v1/projects endpoint
 * is not in api-spec.md (covered by docs/api-spec.md from earlier sprints).
 * If the endpoint is missing, the hook surfaces the error and the project
 * picker falls back to a text-input mode.
 *
 * Accepts an optional filter object. The shape is permissive
 * (`Record<string, ...>`) to stay compatible with the existing
 * /projects and /tasks pages that build a `Record<string, string | undefined>`
 * and pass it straight through. Only `limit` is read by the query; other
 * keys are forwarded as query-string params.
 */
export function useProjects(
  filters: Record<string, string | number | undefined> = {},
): UseQueryResult<Project[]> {
  // /v1/projects query params are string-only on the wire; stringify any
  // numeric values (e.g. limit: 100) before handing off to the api client.
  const params: Record<string, string | undefined> = {}
  for (const [k, v] of Object.entries(filters)) {
    params[k] = v === undefined ? undefined : String(v)
  }
  return useQuery({
    queryKey: queryKeys.projects.list(filters),
    queryFn: () =>
      api
        .get<{ data: Project[] } | Project[]>("/v1/projects", { params })
        .then((res) => (Array.isArray(res) ? res : res.data ?? [])),
    staleTime: 60 * 1000,
  })
}

/* ---------- Single project ---------- */

export function useProject(
  id: string | undefined,
): UseQueryResult<Project> {
  return useQuery({
    queryKey: queryKeys.projects.detail(id ?? ""),
    queryFn: () =>
      api
        .get<ApiEnvelope<Project> | Project>(`/v1/projects/${id}`)
        .then((res) => ("data" in res ? res.data : res)),
    enabled: !!id,
  })
}

export function useDeleteProject(): UseMutationResult<void, Error, string> {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api.delete<void>(`/v1/projects/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.projects.all })
    },
  })
}

/**
 * Create a new project. The Sprint 1-3 spec doesn't include a
 * /v1/projects POST endpoint; if the backend rejects this we surface
 * the error to the page so the new-project form can show a message.
 */
export function useCreateProject(): UseMutationResult<
  Project,
  Error,
  Partial<Omit<Project, "id" | "created_at" | "updated_at">> & {
    name: string
  }
> {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload) =>
      api
        .post<ApiEnvelope<Project> | Project>(`/v1/projects`, payload)
        .then((res) => ("data" in res ? res.data : res)),
    onSuccess: (project) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.projects.all })
      queryClient.setQueryData(queryKeys.projects.detail(project.id), project)
    },
  })
}

/**
 * Update a project. Sprint 1-3 spec doesn't include a PATCH /v1/projects/:id;
 * we expose the hook so the existing edit page has a stable target.
 *
 * Two call shapes:
 *   - `useUpdateProject(id)` — id captured by closure (preferred for pages)
 *   - `useUpdateProject()`   — id passed in the mutation payload (legacy)
 */
export function useUpdateProject(
  id?: string,
): UseMutationResult<Project, Error, Partial<Project> & { id?: string }> {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload) => {
      const targetId = id ?? payload.id
      if (!targetId) throw new Error("useUpdateProject: missing project id")
      return api
        .patch<ApiEnvelope<Project> | Project>(`/v1/projects/${targetId}`, payload)
        .then((res) => ("data" in res ? res.data : res))
    },
    onSuccess: (project) => {
      queryClient.setQueryData(queryKeys.projects.detail(project.id), project)
      queryClient.invalidateQueries({ queryKey: queryKeys.projects.all })
    },
  })
}

/* ---------- Tasks ---------- */

/**
 * List tasks for a project. Accepts either a string shorthand (treated as
 * `{ project_id }`) for backward compat with the existing /tasks and
 * /projects/:id pages, or a full TaskListFilters object.
 */
export function useTasks(
  filters: TaskListFilters | string = {},
): UseQueryResult<Task[]> {
  const { projectId: switcherProjectId } = useProjectFilters()
  const f: TaskListFilters =
    typeof filters === "string"
      ? { project_id: filters }
      : { ...filters, project_id: filters.project_id ?? switcherProjectId ?? undefined }
  return useQuery({
    queryKey: queryKeys.tasks.list(f),
    queryFn: async () => {
      // Sprint 1-3 spec: tasks are project-scoped at the URL level.
      if (!f.project_id) return []
      const res = await api.get<
        { data: Task[] } | Task[] | PaginatedResponse<Task>
      >(`/v1/projects/${f.project_id}/tasks`, {
        params: f,
        ...projectHeaders(switcherProjectId),
      })
      if (Array.isArray(res)) return res
      if ("data" in res) {
        return (res as { data: Task[] }).data
      }
      return (res as unknown) as Task[]
    },
    enabled: !!f.project_id,
  })
}

export function useTask(id: string | undefined): UseQueryResult<Task> {
  const { projectId } = useProjectFilters()
  return useQuery({
    queryKey: queryKeys.tasks.detail(id ?? ""),
    queryFn: () =>
      api
        .get<ApiEnvelope<Task> | Task>(`/v1/tasks/${id}`, { ...projectHeaders(projectId) })
        .then((res) => ("data" in res ? res.data : res)),
    enabled: !!id,
  })
}

export function useCreateTask(): UseMutationResult<
  Task,
  Error,
  CreateTaskPayload
> {
  const queryClient = useQueryClient()
  const { projectId } = useProjectFilters()
  return useMutation({
    mutationFn: (payload: CreateTaskPayload) => {
      const { projectId: pid, ...body } = payload
      return api
        .post<ApiEnvelope<Task> | Task>(
          `/v1/projects/${pid}/tasks`,
          body,
          projectHeaders(projectId),
        )
        .then((res) => ("data" in res ? res.data : res))
    },
    onSuccess: (task) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.tasks.all })
      queryClient.setQueryData(queryKeys.tasks.detail(task.id), task)
    },
  })
}

export function useUpdateTaskStatus(): UseMutationResult<
  Task,
  Error,
  UpdateTaskStatusPayload
  // The optimistic update here also rolls back on error. We snapshot
  // *all* list caches for the current project (the actual key used by
  // the page is `tasks.list({ project_id })`, which the snapshot walks
  // via the `tasks.all` parent key) plus the single detail cache.
> {
  const queryClient = useQueryClient()
  const { projectId } = useProjectFilters()
  return useMutation({
    mutationFn: ({ id, status }) => {
      const headers = projectHeaders(projectId)
      // Only include the third `headers` argument when there's a real
      // X-Project-ID to send — otherwise we leave it off entirely so
      // call-site tests (which assert on the exact arg list) don't
      // see a trailing `undefined`.
      return (
        headers
          ? api.patch<ApiEnvelope<Task> | Task>(
              `/v1/tasks/${id}/status`,
              { status },
              headers,
            )
          : api.patch<ApiEnvelope<Task> | Task>(
              `/v1/tasks/${id}/status`,
              { status },
            )
      ).then((res) => ("data" in res ? res.data : res))
    },
    onMutate: async ({ id, status }) => {
      await queryClient.cancelQueries({ queryKey: queryKeys.tasks.all })
      // IMPORTANT: capture snapshots BEFORE applying the optimistic
      // patch. The detail cache is a child of `tasks.all` (the snapshot
      // key walks the parent), so if we patched the detail first the
      // snapshot would contain the optimistic value and the rollback
      // would re-apply the optimistic state instead of restoring the
      // original.
      const detailKey = queryKeys.tasks.detail(id)
      const prevLists = queryClient.getQueriesData<unknown>({
        queryKey: queryKeys.tasks.all,
      })
      const prevDetail = queryClient.getQueryData<Task | undefined>(detailKey)
      // Snapshot the original detail separately so the rollback can
      // restore it directly even though the snapshots loop will
      // also touch the detail key.
      const detailSnapshot: [readonly unknown[], unknown] = [
        detailKey,
        prevDetail,
      ]
      const snapshots: Array<[readonly unknown[], unknown]> = []
      for (const [key, data] of prevLists) {
        if (data === undefined) continue
        snapshots.push([key, data])
        if (Array.isArray(data)) {
          queryClient.setQueryData<Task[]>(
            key,
            data.map((t) => (t.id === id ? { ...t, status } : t)),
          )
        } else if (data && typeof data === "object" && "data" in data) {
          const env = data as { data: Task[]; pagination?: unknown }
          if (Array.isArray(env.data)) {
            queryClient.setQueryData(key, {
              ...env,
              data: env.data.map((t) => (t.id === id ? { ...t, status } : t)),
            })
          }
        }
      }
      // Apply the optimistic patch to the detail cache LAST so the
      // list snapshot loop (which already ran) has captured the
      // pre-patch state of any list entries that contain this task.
      if (prevDetail) {
        queryClient.setQueryData<Task>(detailKey, { ...prevDetail, status })
      }
      return { prevDetail, detailKey, snapshots, detailSnapshot }
    },
    onError: () => {
      // Rollback is handled in onSettled.
    },
    onSuccess: (task) => {
      queryClient.setQueryData(queryKeys.tasks.detail(task.id), task)
    },
    onSettled: (_data, _err, _vars, context) => {
      if (context?.detailSnapshot) {
        // Restore the detail FIRST with the original (pre-patch) value.
        // This must run before the snapshot loop because the loop
        // contains the pre-patch state of the list, but the detail
        // entry in that loop is the value that existed at onMutate time
        // (also pre-patch). Restoring from the explicit snapshot makes
        // the intent obvious.
        queryClient.setQueryData(
          context.detailSnapshot[0],
          context.detailSnapshot[1],
        )
      }
      if (context?.snapshots) {
        for (const [key, value] of context.snapshots) {
          queryClient.setQueryData(key, value)
        }
      }
    },
  })
}

/**
 * POST /v1/tasks/:id/assign.
 *
 * Per Lead's brief (2026-06-12), body shape is
 *   { agent_id, capabilities_required?, notes? }
 * Per Sprint 4 spec §3.1, the body is
 *   { agent_id?, strategy?, reason? }
 * We follow the Lead's brief; if the backend rejects unknown fields,
 * surface in the gap list.
 */
export function useAssignTask(): UseMutationResult<
  AssignTaskResult,
  Error,
  AssignTaskPayload
> {
  const queryClient = useQueryClient()
  const { projectId } = useProjectFilters()
  return useMutation({
    mutationFn: ({ taskId, ...body }) =>
      api
        .post<
          ApiEnvelope<AssignTaskResult> | AssignTaskResult
        >(`/v1/tasks/${taskId}/assign`, body, projectHeaders(projectId))
        .then((res) => ("data" in res ? res.data : res)),
    onSuccess: (result) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.tasks.all })
      queryClient.invalidateQueries({ queryKey: queryKeys.agents.all })
      // The returned task is the freshest version; cache it.
      queryClient.setQueryData(queryKeys.tasks.detail(result.task.id), result.task)
    },
  })
}

/* ---------- Per-task executions / deliverables (sub-resources) ---------- */

export function useTaskExecutions(
  taskId: string | undefined,
): UseQueryResult<Execution[]> {
  const { projectId } = useProjectFilters()
  return useQuery({
    queryKey: queryKeys.tasks.executions(taskId ?? ""),
    queryFn: async () => {
      const res = await api.get<
        Execution[] | PaginatedResponse<Execution>
      >("/v1/executions", {
        params: { task_id: taskId, project_id: projectId ?? undefined },
        ...projectHeaders(projectId),
      })
      return Array.isArray(res) ? res : res.data
    },
    enabled: !!taskId && !!projectId,
  })
}

export function useTaskDeliverables(
  taskId: string | undefined,
): UseQueryResult<Deliverable[]> {
  const { projectId } = useProjectFilters()
  return useQuery({
    queryKey: queryKeys.tasks.deliverables(taskId ?? ""),
    queryFn: async () => {
      const res = await api.get<
        Deliverable[] | PaginatedResponse<Deliverable>
      >("/v1/deliverables", {
        params: { task_id: taskId, project_id: projectId ?? undefined },
        ...projectHeaders(projectId),
      })
      return Array.isArray(res) ? res : res.data
    },
    enabled: !!taskId && !!projectId,
  })
}

/* ---------- Task history (TASK-404) ---------- */

export function useTaskHistory(
  id: string | undefined,
): UseQueryResult<AssignmentEvent[]> {
  const { projectId } = useProjectFilters()
  return useQuery({
    queryKey: queryKeys.tasks.history(id ?? ""),
    queryFn: async () => {
      const res = await api.get<
        TaskHistoryResponse | { data: AssignmentEvent[] } | AssignmentEvent[]
      >(`/v1/tasks/${id}/history`, { ...projectHeaders(projectId) })
      if (Array.isArray(res)) return res
      if ("data" in res) {
        return (res as { data: AssignmentEvent[] }).data
      }
      return (res as unknown) as AssignmentEvent[]
    },
    enabled: !!id,
  })
}

/* ---------- Re-exports ---------- */

export type {
  Agent,
  AgentCapability,
  AgentMetadata,
  AssignmentEvent,
  AssignTaskPayload,
  AssignTaskResult,
  CapabilityCatalogItem,
  CreateTaskPayload,
  Task,
  TaskHistoryResponse,
  TaskListFilters,
  TaskStatus,
  UpdateTaskStatusPayload,
}
