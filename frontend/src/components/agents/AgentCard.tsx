import Link from "next/link"

import { cn, formatPercent, timeAgo } from "@/lib/utils"
import { AgentStatusBadge } from "./AgentStatusBadge"
import { CapabilityChip } from "./CapabilityChip"
import type { Agent, CapabilityCatalogItem } from "@/lib/types"

export function AgentCard({
  agent,
  capabilities,
}: {
  agent: Agent
  capabilities?: CapabilityCatalogItem[]
}) {
  const capIndex = new Map((capabilities ?? []).map((c) => [c.name, c]))
  const activeAssignments = agent.active_assignments ?? 0
  const successRate = agent.success_rate
  const tasksCompleted = agent.tasks_completed
  const lastActive = agent.last_active_at
    ? timeAgo(agent.last_active_at)
    : "—"

  return (
    <Link
      href={`/agents/${agent.id}`}
      className={cn(
        "group flex flex-col gap-3 rounded-lg border border-slate-200 bg-white p-4 shadow-sm transition",
        "hover:border-sky-300 hover:shadow",
        "dark:border-slate-800 dark:bg-slate-900 dark:hover:border-sky-700",
      )}
    >
      <header className="flex items-start justify-between gap-2">
        <div className="min-w-0">
          <h3 className="truncate text-sm font-semibold text-slate-900 group-hover:text-sky-700 dark:text-slate-100 dark:group-hover:text-sky-300">
            {agent.name}
          </h3>
          <p className="truncate text-xs text-slate-500 dark:text-slate-400">
            {agent.role}
            {agent.metadata?.model ? (
              <span className="ml-1.5 text-slate-400">
                · {String(agent.metadata.model)}
              </span>
            ) : null}
          </p>
        </div>
        <AgentStatusBadge status={agent.status} />
      </header>

      {agent.capabilities.length > 0 ? (
        <div className="flex flex-wrap gap-1.5">
          {agent.capabilities.slice(0, 5).map((name) => {
            const cat = capIndex.get(name)
            return (
              <CapabilityChip
                key={name}
                name={name}
                displayName={cat?.display_name}
                category={cat?.category}
              />
            )
          })}
          {agent.capabilities.length > 5 ? (
            <span className="inline-flex items-center rounded-full bg-slate-100 px-2 py-0.5 text-xs text-slate-500 dark:bg-slate-800 dark:text-slate-400">
              +{agent.capabilities.length - 5} more
            </span>
          ) : null}
        </div>
      ) : (
        <p className="text-xs italic text-slate-400">No capabilities</p>
      )}

      <footer className="mt-auto grid grid-cols-3 gap-2 border-t border-slate-100 pt-3 text-xs dark:border-slate-800">
        <div>
          <div className="text-slate-400">Active</div>
          <div className="font-medium text-slate-700 dark:text-slate-200">
            {activeAssignments}
          </div>
        </div>
        <div>
          <div className="text-slate-400">Success</div>
          <div className="font-medium text-slate-700 dark:text-slate-200">
            {successRate != null ? formatPercent(successRate) : "—"}
          </div>
        </div>
        <div>
          <div className="text-slate-400">
            {tasksCompleted != null ? "Done" : "Last seen"}
          </div>
          <div className="font-medium text-slate-700 dark:text-slate-200">
            {tasksCompleted != null ? tasksCompleted : lastActive}
          </div>
        </div>
      </footer>
    </Link>
  )
}
