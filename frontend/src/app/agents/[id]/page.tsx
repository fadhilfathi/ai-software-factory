"use client"

import { use, useState } from "react"
import Link from "next/link"
import { useRouter } from "next/navigation"

import {
  useAgent,
  useAgentCapabilities,
  useAgentHistory,
  useDeleteAgent,
  useDeliverables,
  useExecutions,
} from "@/lib/hooks"
import { cn, formatPercent, formatUptime, timeAgo } from "@/lib/utils"
import type { AgentCapability, ExecutionStatus } from "@/lib/types"

import { PageHeader } from "@/components/layout/PageHeader"
import { MetricCard } from "@/components/shared/MetricCard"
import { ActivityTimeline } from "@/components/shared/ActivityTimeline"
import { ConfirmDialog } from "@/components/shared/ConfirmDialog"
import { ErrorBlock } from "@/components/ui/ErrorBlock"
import { Skeleton } from "@/components/ui/Skeleton"
import { EmptyState } from "@/components/ui/EmptyState"
import { AgentStatusBadge } from "@/components/agents/AgentStatusBadge"
import { CapabilityChip } from "@/components/agents/CapabilityChip"
import { ProjectPickerGate } from "@/components/agents/ProjectPickerGate"

type Tab = "overview" | "capabilities" | "executions" | "deliverables"

const TAB_LABELS: Array<{ value: Tab; label: string }> = [
  { value: "overview", label: "Overview" },
  { value: "capabilities", label: "Capabilities" },
  { value: "executions", label: "Executions" },
  { value: "deliverables", label: "Deliverables" },
]

const EXECUTION_STATUS_TONE: Record<ExecutionStatus, string> = {
  pending: "bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-200",
  running: "bg-sky-50 text-sky-700 dark:bg-sky-900/30 dark:text-sky-200",
  succeeded:
    "bg-emerald-50 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-200",
  failed: "bg-rose-50 text-rose-700 dark:bg-rose-900/30 dark:text-rose-200",
  cancelled:
    "bg-amber-50 text-amber-700 dark:bg-amber-900/30 dark:text-amber-200",
}

export default function AgentDetailPage({
  params,
}: {
  params: Promise<{ id: string }>
}) {
  return (
    <ProjectPickerGate>
      <AgentDetail params={params} />
    </ProjectPickerGate>
  )
}

function AgentDetail({ params }: { params: Promise<{ id: string }> }) {
  const router = useRouter()
  const { id } = use(params)
  const agentQuery = useAgent(id)
  const historyQuery = useAgentHistory(id)
  const deleteAgent = useDeleteAgent()
  const [tab, setTab] = useState<Tab>("overview")
  const [confirmDelete, setConfirmDelete] = useState(false)

  if (agentQuery.isError) {
    return <ErrorBlock error={agentQuery.error} onRetry={() => agentQuery.refetch()} />
  }

  if (agentQuery.isLoading || !agentQuery.data) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-12 w-1/2" />
        <Skeleton className="h-24" />
        <Skeleton className="h-64" />
      </div>
    )
  }

  const agent = agentQuery.data

  async function onDelete() {
    await deleteAgent.mutateAsync(agent.id)
    router.push("/agents")
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title={agent.name}
        description={agent.metadata?.description as string | undefined}
        actions={
          <div className="flex items-center gap-2">
            <Link
              href={`/agents/${agent.id}/status`}
              className="rounded-md border border-slate-300 px-3 py-1.5 text-sm hover:border-slate-400 dark:border-slate-700"
            >
              Status timeline
            </Link>
            <Link
              href={`/agents/${agent.id}/capabilities`}
              className="rounded-md border border-slate-300 px-3 py-1.5 text-sm hover:border-slate-400 dark:border-slate-700"
            >
              Manage capabilities
            </Link>
            <Link
              href={`/agents/${agent.id}/edit`}
              className="rounded-md border border-sky-300 bg-sky-50 px-3 py-1.5 text-sm font-medium text-sky-700 hover:bg-sky-100 dark:border-sky-700 dark:bg-sky-900/30 dark:text-sky-200"
            >
              Edit
            </Link>
            <button
              type="button"
              onClick={() => setConfirmDelete(true)}
              className="rounded-md border border-rose-300 bg-rose-50 px-3 py-1.5 text-sm font-medium text-rose-700 hover:bg-rose-100 dark:border-rose-700 dark:bg-rose-900/30 dark:text-rose-200"
            >
              Delete
            </button>
          </div>
        }
      >
        <div className="flex flex-wrap items-center gap-3">
          <AgentStatusBadge status={agent.status} />
          <span className="text-sm text-slate-500 dark:text-slate-400">
            {agent.role}
          </span>
          {agent.metadata?.model ? (
            <span className="text-xs text-slate-400">
              model: {String(agent.metadata.model)}
            </span>
          ) : null}
        </div>
      </PageHeader>

      {/* Metric row */}
      <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
        <MetricCard
          label="Tasks completed"
          value={agent.tasks_completed != null ? String(agent.tasks_completed) : "—"}
        />
        <MetricCard
          label="Success rate"
          value={agent.success_rate != null ? formatPercent(agent.success_rate) : "—"}
        />
        <MetricCard
          label="Uptime"
          value={
            agent.uptime_seconds != null
              ? formatUptime(agent.uptime_seconds)
              : "—"
          }
        />
        <MetricCard
          label="Last active"
          value={
            agent.last_active_at
              ? timeAgo(agent.last_active_at)
              : "—"
          }
        />
      </div>

      {/* Tabs */}
      <div className="border-b border-slate-200 dark:border-slate-800">
        <nav className="-mb-px flex flex-wrap gap-2" aria-label="Sections">
          {TAB_LABELS.map((t) => {
            const active = tab === t.value
            return (
              <button
                key={t.value}
                type="button"
                onClick={() => setTab(t.value)}
                className={cn(
                  "border-b-2 px-3 py-2 text-sm font-medium transition",
                  active
                    ? "border-sky-500 text-sky-700 dark:text-sky-300"
                    : "border-transparent text-slate-500 hover:border-slate-300 hover:text-slate-700 dark:text-slate-400 dark:hover:border-slate-600 dark:hover:text-slate-200",
                )}
                aria-pressed={active}
              >
                {t.label}
              </button>
            )
          })}
        </nav>
      </div>

      {tab === "overview" ? (
        <OverviewTab agentId={id} />
      ) : tab === "capabilities" ? (
        <CapabilitiesTab agentId={id} />
      ) : tab === "executions" ? (
        <ExecutionsTab agentId={id} />
      ) : (
        <DeliverablesTab agentId={id} />
      )}

      <ConfirmDialog
        open={confirmDelete}
        title="Delete this agent?"
        description={`This will permanently delete "${agent.name}" and remove it from any assignments. This action cannot be undone.`}
        confirmLabel="Delete"
        destructive
        loading={deleteAgent.isPending}
        onCancel={() => setConfirmDelete(false)}
        onConfirm={onDelete}
      />

      {/* Activity history sits at the bottom of the overview column area. The
          un-mocked list binds to useAgentHistory. If the history endpoint
          doesn't exist yet (migration 020 not landed), the hook errors and
          we render an EmptyState hint. */}
      {tab === "overview" ? (
        <section>
          <h2 className="mb-3 text-sm font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">
            Recent activity
          </h2>
          {historyQuery.isError ? (
            <EmptyState
              title="Activity feed unavailable"
              description="The history endpoint isn't available yet. The agent_state_events table lands in migration 020 — this feed will populate once it does."
            />
          ) : historyQuery.isLoading ? (
            <div className="space-y-2">
              <Skeleton className="h-12" />
              <Skeleton className="h-12" />
              <Skeleton className="h-12" />
            </div>
          ) : (historyQuery.data?.data ?? []).length === 0 ? (
            <EmptyState
              title="No activity yet"
              description="This agent hasn't done anything recordable yet."
            />
          ) : (
            <ActivityTimeline
              items={(historyQuery.data?.data ?? []).map((e) => ({
                id: e.id,
                text: [e.title, e.description].filter(Boolean).join(" — "),
                type:
                  e.type === "execution_failed"
                    ? "error"
                    : e.type === "execution_completed"
                      ? "success"
                      : "info",
                timestamp: e.at,
              }))}
            />
          )}
        </section>
      ) : null}
    </div>
  )
}

/* ---------- Tabs ---------- */

function OverviewTab({ agentId: _agentId }: { agentId: string }) {
  // The metric row + activity history are rendered in the parent so the
  // tab content stays small. This tab exists as an explicit "no extra
  // content" marker for the future when overview grows richer widgets.
  return null
}

function CapabilitiesTab({ agentId }: { agentId: string }) {
  const agentQuery = useAgent(agentId)
  const richQuery = useAgentCapabilities(agentId)
  const capIndex = new Map((richQuery.data ?? []).map((c: AgentCapability) => [c.name, c]))
  const capabilities = agentQuery.data?.capabilities ?? []

  if (capabilities.length === 0) {
    return (
      <EmptyState
        title="No capabilities"
        description="This agent has no capabilities assigned."
        action={
          <Link
            href={`/agents/${agentId}/capabilities`}
            className="inline-flex items-center gap-1.5 rounded-md bg-sky-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-sky-700"
          >
            Manage capabilities
          </Link>
        }
      />
    )
  }

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap gap-2">
        {capabilities.map((name) => {
          const rich = capIndex.get(name)
          return (
            <CapabilityChip
              key={name}
              name={name}
              displayName={rich?.display_name}
              category={rich?.category}
            />
          )
        })}
      </div>
      <p className="text-xs text-slate-500">
        <Link
          href={`/agents/${agentId}/capabilities`}
          className="text-sky-600 hover:underline dark:text-sky-400"
        >
          Manage capabilities →
        </Link>
      </p>
    </div>
  )
}

function ExecutionsTab({ agentId }: { agentId: string }) {
  const executionsQuery = useExecutions({ agent_id: agentId, limit: 20 })

  if (executionsQuery.isError) {
    return (
      <EmptyState
        title="Executions unavailable"
        description="The executions endpoint is not yet wired (TASK-405 pending)."
      />
    )
  }
  if (executionsQuery.isLoading) {
    return (
      <div className="space-y-2">
        <Skeleton className="h-10" />
        <Skeleton className="h-10" />
        <Skeleton className="h-10" />
      </div>
    )
  }
  const items = executionsQuery.data?.data ?? []
  if (items.length === 0) {
    return (
      <EmptyState
        title="No executions yet"
        description="This agent has not been assigned to a task yet."
      />
    )
  }
  return (
    <ul className="divide-y divide-slate-200 rounded-md border border-slate-200 dark:divide-slate-800 dark:border-slate-800">
      {items.map((e) => (
        <li
          key={e.id}
          className="flex items-center justify-between gap-3 px-4 py-2.5 text-sm"
        >
          <span className="font-mono text-xs text-slate-500">
            {e.id.slice(0, 8)}…
          </span>
          <span
            className={cn(
              "rounded-full px-2 py-0.5 text-xs",
              EXECUTION_STATUS_TONE[e.status] ?? EXECUTION_STATUS_TONE.pending,
            )}
          >
            {e.status}
          </span>
          <span className="text-xs text-slate-400">
            {timeAgo(e.started_at)}
          </span>
        </li>
      ))}
    </ul>
  )
}

function DeliverablesTab({ agentId }: { agentId: string }) {
  const deliverablesQuery = useDeliverables({ agent_id: agentId, limit: 20 })

  if (deliverablesQuery.isError) {
    return (
      <EmptyState
        title="Deliverables unavailable"
        description="The deliverables endpoint is not yet wired (TASK-406 pending)."
      />
    )
  }
  if (deliverablesQuery.isLoading) {
    return (
      <div className="space-y-2">
        <Skeleton className="h-10" />
        <Skeleton className="h-10" />
      </div>
    )
  }
  const items = deliverablesQuery.data?.data ?? []
  if (items.length === 0) {
    return (
      <EmptyState
        title="No deliverables yet"
        description="This agent has not produced any deliverables."
      />
    )
  }
  return (
    <ul className="divide-y divide-slate-200 rounded-md border border-slate-200 dark:divide-slate-800 dark:border-slate-800">
      {items.map((d) => (
        <li key={d.id} className="px-4 py-2.5 text-sm">
          <div className="flex items-center justify-between gap-2">
            <span className="font-medium text-slate-900 dark:text-slate-100">
              {d.title}
            </span>
            <span className="rounded-full bg-slate-100 px-2 py-0.5 text-xs text-slate-600 dark:bg-slate-800 dark:text-slate-300">
              {d.kind} · v{d.latest_version}
            </span>
          </div>
          {d.description ? (
            <p className="mt-0.5 text-xs text-slate-500">{d.description}</p>
          ) : null}
        </li>
      ))}
    </ul>
  )
}
