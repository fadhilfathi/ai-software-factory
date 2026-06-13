"use client"

import { useMemo, useState } from "react"
import Link from "next/link"

import { ProjectPickerGate } from "@/components/agents/ProjectPickerGate"
import { AgentStatusBadge } from "@/components/agents/AgentStatusBadge"
import { ErrorBlock } from "@/components/ui/ErrorBlock"
import { Spinner } from "@/components/ui/Spinner"
import { useProjectFilters } from "@/hooks/useProjectFilters"
import { useTasks, useTaskHistory } from "@/lib/hooks"
import type { Task, AssignmentEvent, Agent } from "@/lib/types"
import { cn, timeAgo } from "@/lib/utils"
import { useQuery } from "@tanstack/react-query"
import { queryKeys } from "@/lib/queryKeys"
import { api } from "@/lib/api"

/**
 * /assignments — read-only assignment dashboard.
 *
 * Option B per Lead's brief (2026-06-12). There is no GET /v1/assignments
 * endpoint (sprint4 spec only exposes per-task /v1/tasks/:id/history), so
 * the dashboard composes the view by:
 *
 *   1. GET /v1/projects/:projectId/tasks  (list all tasks for the project)
 *   2. For each task, GET /v1/tasks/:id/history  (N+1 — see gap flag)
 *   3. Optionally, for each unique agent_id, GET /v1/agents/:id  (M+1)
 *
 * The N+1 and M+1 are intentional and the Lead explicitly asked for this
 * to be flagged. The page renders progressively and shows partial data
 * while history/agent lookups are in flight.
 */
export default function AssignmentsDashboardPage() {
  return (
    <ProjectPickerGate
      title="Select a project"
      description="The assignment dashboard is project-scoped. Choose a project to continue."
    >
      <AssignmentsDashboardPageInner />
    </ProjectPickerGate>
  )
}

type StatusFilter = "all" | "assigned" | "unassigned" | "stale"

function AssignmentsDashboardPageInner() {
  // useTasks reads the projectId from the URL store (set by the gate),
  // so we don't have to pass it explicitly. We still read it here so we
  // can show a "Pick a project" hint when the gate hasn't set one yet.
  const { projectId } = useProjectFilters()
  const tasksQuery = useTasks()
  const [statusFilter, setStatusFilter] = useState<StatusFilter>("all")
  const [search, setSearch] = useState("")

  const tasks = tasksQuery.data ?? []

  return (
    <div className="mx-auto max-w-6xl px-6 py-8">
      <header className="mb-6 flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold text-slate-900 dark:text-slate-100">
            Assignment dashboard
          </h1>
          <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">
            Read-only view of every task in the current project and its
            ownership history. Data is loaded progressively.
          </p>
        </div>
        <Link
          href="/tasks"
          className="rounded-md border border-slate-300 px-3 py-1.5 text-sm hover:bg-slate-100 dark:border-slate-700 dark:hover:bg-slate-800"
        >
          All tasks →
        </Link>
      </header>

      {/* Filters */}
      <div className="mb-4 flex flex-wrap items-center gap-2">
        <input
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="Search by title…"
          className={cn(
            "flex-1 min-w-[200px] rounded-md border border-slate-300 bg-white px-3 py-1.5 text-sm",
            "dark:border-slate-700 dark:bg-slate-900",
          )}
        />
        <div className="flex gap-1">
          {(
            [
              ["all", "All"],
              ["assigned", "Assigned"],
              ["unassigned", "Unassigned"],
              ["stale", "Stale (>7d)"],
            ] as const
          ).map(([key, label]) => (
            <button
              key={key}
              type="button"
              onClick={() => setStatusFilter(key)}
              className={cn(
                "rounded-md border px-2.5 py-1 text-xs font-medium",
                statusFilter === key
                  ? "border-sky-500 bg-sky-50 text-sky-700 dark:bg-sky-900/30 dark:text-sky-200"
                  : "border-slate-300 text-slate-600 hover:bg-slate-100 dark:border-slate-700 dark:text-slate-300 dark:hover:bg-slate-800",
              )}
            >
              {label}
            </button>
          ))}
        </div>
      </div>

      {!projectId ? (
        <div className="rounded-md border border-dashed border-slate-300 p-6 text-center text-sm text-slate-500 dark:border-slate-700">
          Pick a project to see its assignment dashboard.
        </div>
      ) : tasksQuery.isLoading ? (
        <div className="flex items-center gap-2 px-2 py-8 text-sm text-slate-500">
          <Spinner size="sm" /> Loading tasks…
        </div>
      ) : tasksQuery.isError ? (
        <ErrorBlock
          title="Could not load tasks"
          message={
            (tasksQuery.error as Error | undefined)?.message ??
            "An error occurred while loading the task list."
          }
        />
      ) : tasks.length === 0 ? (
        <div className="rounded-md border border-dashed border-slate-300 p-8 text-center text-sm text-slate-500 dark:border-slate-700">
          No tasks in this project yet.
        </div>
      ) : (
        <AssignmentsTable
          tasks={tasks}
          search={search}
          statusFilter={statusFilter}
        />
      )}

      <p className="mt-6 text-xs text-slate-400">
        ⚠ Dashboard composes the view client-side by calling
        GET /v1/tasks/:id/history for every task (N+1) and
        GET /v1/agents/:id for every assigned row (M+1). A native
        GET /v1/assignments endpoint would replace this. See task
        report.
      </p>
    </div>
  )
}

function AssignmentsTable({
  tasks,
  search,
  statusFilter,
}: {
  tasks: Task[]
  search: string
  statusFilter: StatusFilter
}) {
  // Filter first (cheap) before triggering per-task history lookups.
  const visible = useMemo(() => {
    const s = search.trim().toLowerCase()
    return tasks.filter((t) => {
      if (s && !t.title.toLowerCase().includes(s)) return false
      return true
    })
  }, [tasks, search])

  return (
    <div
      className={cn(
        "overflow-hidden rounded-md border border-slate-200 bg-white",
        "dark:border-slate-800 dark:bg-slate-900",
      )}
    >
      <table className="w-full text-sm">
        <thead className="bg-slate-50 text-left text-xs uppercase tracking-wide text-slate-500 dark:bg-slate-800/50">
          <tr>
            <th className="px-4 py-2 font-medium">Task</th>
            <th className="px-4 py-2 font-medium">Status</th>
            <th className="px-4 py-2 font-medium">Owner</th>
            <th className="px-4 py-2 font-medium">Last assignment</th>
            <th className="px-4 py-2 font-medium text-right">Actions</th>
          </tr>
        </thead>
        <tbody>
          {visible.map((task) => (
            <AssignmentRow
              key={task.id}
              task={task}
              statusFilter={statusFilter}
            />
          ))}
          {visible.length === 0 ? (
            <tr>
              <td
                colSpan={5}
                className="px-4 py-6 text-center text-sm text-slate-500"
              >
                No tasks match the current filter.
              </td>
            </tr>
          ) : null}
        </tbody>
      </table>
    </div>
  )
}

function AssignmentRow({
  task,
  statusFilter,
}: {
  task: Task
  statusFilter: StatusFilter
}) {
  // N+1 — one history query per task. The page is intentionally a
  // read-only view; this is the cost of not having GET /v1/assignments.
  const historyQuery = useTaskHistory(task.id)
  const events: AssignmentEvent[] = historyQuery.data ?? []
  const lastAssign = events.find(
    (e) => (e.event_type === "assign" || e.event_type === "reassign") && e.agent_id,
  )
  // Sprint 1-3 uses `assignee_id`; Sprint 4 renames it to
  // `assigned_agent_id`. Read both to stay backend-agnostic.
  const ownerId =
    task.assignee_id ??
    (task as { assigned_agent_id?: string | null }).assigned_agent_id ??
    null
  const isAssigned = !!ownerId
  const isStale =
    !isAssigned && events.length === 0
      ? false
      : !isAssigned &&
        events[0] &&
        Date.now() - new Date(events[0].created_at).getTime() >
          7 * 24 * 60 * 60 * 1000

  // Local filter logic (per-row)
  if (statusFilter === "assigned" && !isAssigned) return null
  if (statusFilter === "unassigned" && isAssigned) return null
  if (statusFilter === "stale" && !isStale) return null

  return (
    <tr className="border-t border-slate-100 dark:border-slate-800">
      <td className="px-4 py-3 align-top">
        <Link
          href={`/tasks/${task.id}`}
          className="font-medium text-slate-900 hover:text-sky-700 dark:text-slate-100 dark:hover:text-sky-300"
        >
          {task.title}
        </Link>
        <p className="mt-0.5 font-mono text-[10px] text-slate-400">
          {task.id.slice(0, 8)}…
        </p>
      </td>
      <td className="px-4 py-3 align-top">
        <span className="rounded-md bg-slate-100 px-2 py-0.5 text-xs text-slate-600 dark:bg-slate-800 dark:text-slate-300">
          {task.status}
        </span>
      </td>
      <td className="px-4 py-3 align-top">
        {isAssigned && ownerId ? (
          <OwnerCell agentId={ownerId} />
        ) : (
          <span className="text-xs italic text-slate-400">unassigned</span>
        )}
      </td>
      <td className="px-4 py-3 align-top text-xs text-slate-500">
        {historyQuery.isLoading ? (
          <Spinner size="sm" />
        ) : lastAssign ? (
          <div className="flex flex-col">
            <span>
              {lastAssign.event_type === "reassign" ? "Reassigned" : "Assigned"}
            </span>
            <span className="text-slate-400">{timeAgo(lastAssign.created_at)}</span>
          </div>
        ) : events.length > 0 ? (
          <span className="italic">{events[0].event_type}</span>
        ) : (
          <span className="italic text-slate-400">no history</span>
        )}
      </td>
      <td className="px-4 py-3 text-right align-top">
        <div className="flex justify-end gap-2 text-xs">
          <Link
            href={`/tasks/${task.id}/ownership`}
            className="rounded-md border border-slate-300 px-2 py-1 hover:bg-slate-100 dark:border-slate-700 dark:hover:bg-slate-800"
          >
            History
          </Link>
          <Link
            href={`/tasks/${task.id}/assign`}
            className="rounded-md bg-sky-600 px-2 py-1 font-medium text-white hover:bg-sky-700"
          >
            {isAssigned ? "Reassign" : "Assign"}
          </Link>
        </div>
      </td>
    </tr>
  )
}

function OwnerCell({ agentId }: { agentId: string }) {
  // M+1 — one agent lookup per assigned row. Same gap rationale as
  // history N+1; we just need a name + status to render the badge.
  const agentQuery = useQuery<Agent>({
    queryKey: queryKeys.agents.detail(agentId),
    queryFn: () => api.get<Agent>(`/v1/agents/${agentId}`),
    enabled: !!agentId,
    retry: 0,
    staleTime: 30 * 1000,
  })
  if (agentQuery.isLoading) return <Spinner size="sm" />
  if (agentQuery.isError || !agentQuery.data) {
    return <span className="font-mono text-xs text-slate-400">{agentId.slice(0, 8)}…</span>
  }
  const a = agentQuery.data
  return (
    <div className="flex items-center gap-2">
      <span className="font-medium text-slate-900 dark:text-slate-100">
        {a.name}
      </span>
      <AgentStatusBadge status={a.status} />
    </div>
  )
}
