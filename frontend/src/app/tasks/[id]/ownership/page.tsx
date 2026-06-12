"use client"

import { use } from "react"
import Link from "next/link"

import { ProjectPickerGate } from "@/components/agents/ProjectPickerGate"
import { AgentStatusBadge } from "@/components/agents/AgentStatusBadge"
import { CapabilityChip } from "@/components/agents/CapabilityChip"
import { AssignmentHistoryTimeline } from "@/components/tasks/AssignmentHistoryTimeline"
import { ErrorBlock } from "@/components/ui/ErrorBlock"
import { Spinner } from "@/components/ui/Spinner"
import { useAgent, useTask, useTaskHistory } from "@/lib/hooks"
import { cn, timeAgo } from "@/lib/utils"

type Params = Promise<{ id: string }>

/**
 * /tasks/:id/ownership — read-only ownership + history view.
 *
 * Layout:
 *   - Task summary (title, status, priority, due date, project)
 *   - Current owner card (with a "Reassign" CTA)
 *   - Assignment history timeline (driven by GET /v1/tasks/:id/history)
 *
 * Note: this page does NOT implement unassign. The Lead's brief lists
 * the assign body as `{agent_id, capabilities_required?, notes?}` with
 * agent_id required. The spec §3.1 body is `{agent_id?, strategy?, reason?}`
 * and may support a release flow via `strategy: "release"`, but the
 * Lead's shape does not. The unassign gap is flagged in the report.
 */
export default function TaskOwnershipPage({ params }: { params: Params }) {
  return (
    <ProjectPickerGate
      title="Select a project"
      description="Task ownership history is project-scoped. Choose a project to continue."
    >
      <TaskOwnershipPageInner params={params} />
    </ProjectPickerGate>
  )
}

function TaskOwnershipPageInner({ params }: { params: Params }) {
  const { id } = use(params)
  const taskQuery = useTask(id)
  const historyQuery = useTaskHistory(id)
  // useAgent is only enabled when we have a current owner id; we resolve
  // that from the task payload. Per Sprint 1-3 docs, the current owner
  // is exposed as `assignee_id` on the Task payload. (Sprint 4 spec
  // renames this to `assigned_agent_id`; the frontend reads both shapes
  // to stay compatible with whichever backend is wired up.)
  const rawAgentId =
    taskQuery.data?.assignee_id ?? (taskQuery.data as { assigned_agent_id?: string | null } | undefined)?.assigned_agent_id ?? null
  const currentAgentId: string | undefined = rawAgentId ?? undefined
  const agentQuery = useAgent(currentAgentId)

  if (taskQuery.isLoading) {
    return (
      <div className="flex items-center gap-2 px-6 py-12 text-sm text-slate-500">
        <Spinner size="sm" /> Loading task…
      </div>
    )
  }
  if (taskQuery.isError || !taskQuery.data) {
    return (
      <div className="mx-auto max-w-3xl px-6 py-12">
        <ErrorBlock
          title="Task not found"
          message={
            (taskQuery.error as Error | undefined)?.message ??
            "Could not load the task. It may have been deleted or you may not have access."
          }
          actions={
            <Link
              href="/tasks"
              className="rounded-md border border-slate-300 px-3 py-1.5 text-sm hover:bg-slate-100 dark:border-slate-700 dark:hover:bg-slate-800"
            >
              Back to tasks
            </Link>
          }
        />
      </div>
    )
  }

  const task = taskQuery.data
  const isLoadingHistory = historyQuery.isLoading
  const isErrorHistory = historyQuery.isError
  const events = historyQuery.data ?? []

  // Find the most recent "assign" / "reassign" event with a non-null
  // agent_id — that's the implied current owner, even if the task
  // payload's assigned_agent_id is stale.
  const latestAssignEvent = events.find(
    (e) => (e.event_type === "assign" || e.event_type === "reassign") && e.agent_id,
  )
  const lastEventAt = events[0]?.created_at ?? null

  return (
    <div className="mx-auto max-w-3xl px-6 py-8">
      <nav className="mb-2 flex items-center gap-2 text-sm text-slate-500">
        <Link href="/tasks" className="hover:text-slate-700 dark:hover:text-slate-300">
          Tasks
        </Link>
        <span>/</span>
        <Link
          href={`/tasks/${task.id}`}
          className="hover:text-slate-700 dark:hover:text-slate-300"
        >
          {task.id.slice(0, 8)}…
        </Link>
        <span>/</span>
        <span className="text-slate-700 dark:text-slate-300">Ownership</span>
      </nav>

      <header className="mb-6">
        <h1 className="text-2xl font-semibold text-slate-900 dark:text-slate-100">
          {task.title}
        </h1>
        <div className="mt-3 flex flex-wrap gap-2 text-xs text-slate-500">
          <span className="rounded-md bg-slate-100 px-2 py-0.5 dark:bg-slate-800">
            {task.status}
          </span>
          <span className="rounded-md bg-slate-100 px-2 py-0.5 dark:bg-slate-800">
            {task.priority}
          </span>
          {((task as { due_date?: string | null }).due_date) ? (
            <span className="rounded-md bg-slate-100 px-2 py-0.5 dark:bg-slate-800">
              due {new Date((task as { due_date?: string }).due_date!).toLocaleDateString()}
            </span>
          ) : null}
          {lastEventAt ? (
            <span className="rounded-md bg-slate-100 px-2 py-0.5 dark:bg-slate-800">
              last change {timeAgo(lastEventAt)}
            </span>
          ) : null}
        </div>
      </header>

      {/* Current owner */}
      <section className="mb-8">
        <h2 className="mb-3 text-sm font-semibold uppercase tracking-wide text-slate-500">
          Current owner
        </h2>
        {currentAgentId && agentQuery.data ? (
          <CurrentOwnerCard agent={agentQuery.data} />
        ) : currentAgentId && agentQuery.isLoading ? (
          <div className="flex items-center gap-2 rounded-md border border-slate-200 bg-white p-4 text-sm text-slate-500 dark:border-slate-800 dark:bg-slate-900">
            <Spinner size="sm" /> Loading owner…
          </div>
        ) : (
          <div
            className={cn(
              "rounded-md border border-dashed border-slate-300 p-6 text-center",
              "dark:border-slate-700",
            )}
          >
            <p className="text-sm text-slate-500 dark:text-slate-400">
              This task is currently unassigned.
            </p>
            <Link
              href={`/tasks/${task.id}/assign`}
              className="mt-3 inline-block rounded-md bg-sky-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-sky-700"
            >
              Assign now
            </Link>
          </div>
        )}

        <div className="mt-3 flex items-center gap-2">
          <Link
            href={`/tasks/${task.id}/assign`}
            className="rounded-md bg-sky-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-sky-700"
          >
            {currentAgentId ? "Reassign" : "Assign agent"}
          </Link>
          <Link
            href={`/tasks/${task.id}`}
            className="text-sm text-slate-500 hover:text-slate-700 dark:hover:text-slate-300"
          >
            Back to task
          </Link>
        </div>

        {latestAssignEvent && latestAssignEvent.agent_id && latestAssignEvent.agent_id !== currentAgentId ? (
          <p className="mt-3 text-xs text-amber-600 dark:text-amber-400">
            Note: the task payload shows no current owner, but the most recent
            assignment event is for{" "}
            <code className="font-mono">
              {latestAssignEvent.agent_id.slice(0, 8)}…
            </code>
            . The two views may have drifted; refresh to re-sync.
          </p>
        ) : null}
      </section>

      {/* Required capabilities */}
      {(() => {
        const caps =
          task.required_capabilities ??
          (task as { capabilities_required?: string[] }).capabilities_required ??
          []
        return caps.length > 0 ? (
          <section className="mb-8">
            <h2 className="mb-3 text-sm font-semibold uppercase tracking-wide text-slate-500">
              Required capabilities
            </h2>
            <div className="flex flex-wrap gap-1.5">
              {caps.map((c) => (
                <CapabilityChip key={c} name={c} />
              ))}
            </div>
          </section>
        ) : null
      })()}

      {/* Assignment history */}
      <section>
        <h2 className="mb-3 text-sm font-semibold uppercase tracking-wide text-slate-500">
          Assignment history
        </h2>
        {isLoadingHistory ? (
          <div className="flex items-center gap-2 text-sm text-slate-500">
            <Spinner size="sm" /> Loading history…
          </div>
        ) : isErrorHistory ? (
          <ErrorBlock
            title="History unavailable"
            message={
              (historyQuery.error as Error | undefined)?.message ??
              "The assignment history endpoint returned an error."
            }
          />
        ) : (
          <AssignmentHistoryTimeline events={events} />
        )}
      </section>
    </div>
  )
}

function CurrentOwnerCard({
  agent,
}: {
  agent: {
    id: string
    name: string
    role: string
    status:
      | "idle"
      | "busy"
      | "initializing"
      | "retired"
      | "error"
      | "paused"
    capabilities: string[]
  }
}) {
  return (
    <Link
      href={`/agents/${agent.id}`}
      className={cn(
        "flex items-center justify-between gap-4 rounded-md border border-slate-200 bg-white p-4 transition",
        "hover:border-sky-300 hover:bg-sky-50/50",
        "dark:border-slate-800 dark:bg-slate-900 dark:hover:border-sky-700 dark:hover:bg-sky-900/20",
      )}
    >
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <p className="truncate font-medium text-slate-900 dark:text-slate-100">
            {agent.name}
          </p>
          <AgentStatusBadge status={agent.status} />
        </div>
        <p className="mt-0.5 truncate text-xs text-slate-500">{agent.role}</p>
        {agent.capabilities.length > 0 ? (
          <div className="mt-2 flex flex-wrap gap-1">
            {agent.capabilities.slice(0, 6).map((c) => (
              <CapabilityChip key={c} name={c} />
            ))}
            {agent.capabilities.length > 6 ? (
              <span className="text-xs text-slate-400">
                +{agent.capabilities.length - 6} more
              </span>
            ) : null}
          </div>
        ) : null}
      </div>
      <span className="text-sm text-slate-400">View agent →</span>
    </Link>
  )
}
