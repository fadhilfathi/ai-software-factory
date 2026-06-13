import { cn } from "@/lib/utils"
import type { AgentStatus } from "@/lib/types"

const STATUS_TONE: Record<AgentStatus, string> = {
  initializing: "bg-slate-400",
  idle: "bg-emerald-500",
  busy: "bg-sky-500",
  paused: "bg-amber-500",
  error: "bg-rose-500",
  retired: "bg-zinc-400",
}

const STATUS_LABEL: Record<AgentStatus, string> = {
  initializing: "Initializing",
  idle: "Idle",
  busy: "Busy",
  paused: "Paused",
  error: "Error",
  retired: "Retired",
}

export type LifecycleEvent = {
  id: string
  status: AgentStatus
  at: string
  reason?: string
  actor?: string
}

/**
 * Vertical lifecycle timeline. Per the Lead's brief, the underlying
 * `agent_state_events` table is a Sprint 4 / migration 020 addition. Until
 * that migration lands and the history endpoint populates, this component
 * renders a sensible default timeline seeded from the agent's current
 * status + created_at.
 */
export function LifecycleTimeline({
  events,
  emptyHint = "Event log will populate once executions run.",
}: {
  events: LifecycleEvent[]
  emptyHint?: string
}) {
  if (events.length === 0) {
    return (
      <div className="rounded-md border border-dashed border-slate-300 p-6 text-center text-sm text-slate-500 dark:border-slate-700 dark:text-slate-400">
        {emptyHint}
      </div>
    )
  }

  return (
    <ol className="relative space-y-6 border-l border-slate-200 pl-6 dark:border-slate-700">
      {events.map((e, i) => (
        <li key={e.id ?? `${e.status}-${e.at}-${i}`} className="relative">
          <span
            className={cn(
              "absolute -left-[31px] top-1 inline-block h-3 w-3 rounded-full ring-4 ring-white dark:ring-slate-900",
              STATUS_TONE[e.status] ?? "bg-slate-400",
            )}
            aria-hidden
          />
          <div className="flex flex-col gap-0.5">
            <span className="text-sm font-medium text-slate-900 dark:text-slate-100">
              {STATUS_LABEL[e.status] ?? e.status}
            </span>
            <time
              dateTime={e.at}
              className="text-xs text-slate-500 dark:text-slate-400"
            >
              {new Date(e.at).toLocaleString()}
            </time>
            {e.reason ? (
              <p className="mt-1 text-sm text-slate-600 dark:text-slate-300">
                {e.reason}
              </p>
            ) : null}
            {e.actor ? (
              <p className="text-xs text-slate-400">by {e.actor}</p>
            ) : null}
          </div>
        </li>
      ))}
    </ol>
  )
}
