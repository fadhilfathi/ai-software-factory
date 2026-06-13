"use client"

import { use, useMemo } from "react"
import Link from "next/link"

import { useAgent, useAgentHistory } from "@/lib/hooks"
import { cn } from "@/lib/utils"
import type { AgentStatus } from "@/lib/types"

import { PageHeader } from "@/components/layout/PageHeader"
import { ErrorBlock } from "@/components/ui/ErrorBlock"
import { Skeleton } from "@/components/ui/Skeleton"
import { AgentStatusBadge } from "@/components/agents/AgentStatusBadge"
import { LifecycleTimeline, type LifecycleEvent } from "@/components/agents/LifecycleTimeline"
import { ProjectPickerGate } from "@/components/agents/ProjectPickerGate"

export default function AgentStatusPage({
  params,
}: {
  params: Promise<{ id: string }>
}) {
  return (
    <ProjectPickerGate>
      <AgentStatus params={params} />
    </ProjectPickerGate>
  )
}

function AgentStatus({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params)
  const agentQuery = useAgent(id)
  const historyQuery = useAgentHistory(id)

  // Derive lifecycle events from whatever we have. The dedicated
  // agent_state_events table is migration 020, which is not yet applied,
  // so the history endpoint will typically 404. We seed a sensible
  // synthetic timeline from created_at + status + last_active_at so the
  // page is never empty in the meantime.
  const events = useMemo<LifecycleEvent[]>(() => {
    const agent = agentQuery.data
    if (!agent) return []
    const out: LifecycleEvent[] = []
    out.push({
      id: `${agent.id}-created`,
      status: "initializing",
      at: agent.created_at,
      reason: "Agent registered",
      actor: "system",
    })
    if (agent.last_active_at && agent.last_active_at !== agent.created_at) {
      out.push({
        id: `${agent.id}-last-active`,
        status: agent.status === "retired" ? "retired" : "busy",
        at: agent.last_active_at,
        reason: agent.status === "retired" ? "Retired" : "Last seen active",
      })
    }
    // If the history endpoint is live, splice in real events.
    if (historyQuery.data?.data) {
      for (const e of historyQuery.data.data) {
        if (typeof (e as { type?: string }).type === "string" &&
            (e as { type: string }).type === "agent_state_change") {
          const status = (e as { metadata?: { status?: AgentStatus } }).metadata?.status
          if (status) {
            out.push({
              id: (e as { id: string }).id,
              status,
              at: (e as { at: string }).at,
              reason: (e as { description?: string }).description,
            })
          }
        }
      }
    }
    return out.sort((a, b) => a.at.localeCompare(b.at))
  }, [agentQuery.data, historyQuery.data])

  if (agentQuery.isError) {
    return <ErrorBlock error={agentQuery.error} onRetry={() => agentQuery.refetch()} />
  }

  if (agentQuery.isLoading || !agentQuery.data) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-10 w-1/3" />
        <Skeleton className="h-6 w-1/2" />
        <Skeleton className="h-64" />
      </div>
    )
  }

  const agent = agentQuery.data

  return (
    <div className="space-y-6">
      <PageHeader
        title={`Status: ${agent.name}`}
        description="Lifecycle state transitions for this agent."
        actions={
          <Link
            href={`/agents/${agent.id}`}
            className="rounded-md border border-slate-300 px-3 py-1.5 text-sm hover:border-slate-400 dark:border-slate-700"
          >
            Back to overview
          </Link>
        }
      />

      <div className="flex flex-wrap items-center gap-3">
        <span className="text-sm text-slate-500 dark:text-slate-400">
          Current:
        </span>
        <AgentStatusBadge status={agent.status} />
        <span className="text-xs text-slate-400">
          registered {new Date(agent.created_at).toLocaleString()}
        </span>
      </div>

      <section
        className={cn(
          "rounded-lg border border-slate-200 bg-white p-6",
          "dark:border-slate-800 dark:bg-slate-900",
        )}
      >
        <h2 className="mb-4 text-sm font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">
          Lifecycle
        </h2>
        <LifecycleTimeline
          events={events}
          emptyHint="Event log will populate once executions run."
        />
      </section>

      <p className="text-xs text-slate-500">
        Detailed state-change events come from <code>agent_state_events</code>{" "}
        (migration 020). Until that migration lands, the timeline above is
        synthesised from the agent's <code>created_at</code>,{" "}
        <code>last_active_at</code>, and current <code>status</code>.
      </p>
    </div>
  )
}
