import { cn, timeAgo } from "@/lib/utils"
import type { AssignmentEvent, AssignmentEventType } from "@/lib/types"

const ACTION_TONE: Record<AssignmentEventType, string> = {
  assign: "bg-emerald-50 text-emerald-700 ring-emerald-200 dark:bg-emerald-900/30 dark:text-emerald-200 dark:ring-emerald-800",
  reassign: "bg-sky-50 text-sky-700 ring-sky-200 dark:bg-sky-900/30 dark:text-sky-200 dark:ring-sky-800",
  release: "bg-amber-50 text-amber-700 ring-amber-200 dark:bg-amber-900/30 dark:text-amber-200 dark:ring-amber-800",
  unassign: "bg-zinc-50 text-zinc-600 ring-zinc-200 dark:bg-zinc-800 dark:text-zinc-300 dark:ring-zinc-700",
}

const ACTION_LABEL: Record<AssignmentEventType, string> = {
  assign: "Assigned",
  reassign: "Reassigned",
  release: "Released",
  unassign: "Unassigned",
}

export function AssignmentHistoryTimeline({
  events,
  emptyHint = "No assignment history yet.",
}: {
  events: AssignmentEvent[]
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
    <ol className="relative space-y-5 border-l border-slate-200 pl-6 dark:border-slate-700">
      {events.map((e) => {
        const tone =
          ACTION_TONE[e.event_type] ??
          "bg-slate-100 text-slate-700 ring-slate-200 dark:bg-slate-800 dark:text-slate-200 dark:ring-slate-700"
        return (
          <li key={e.id} className="relative">
            <span
              className={cn(
                "absolute -left-[31px] top-1.5 inline-block h-3 w-3 rounded-full ring-4 ring-white dark:ring-slate-900",
                tone.split(" ")[0], // bg-* class
              )}
              aria-hidden
            />
            <div className="flex flex-col gap-1.5">
              <div className="flex flex-wrap items-center gap-2">
                <span
                  className={cn(
                    "rounded-full px-2 py-0.5 text-xs font-medium ring-1 ring-inset",
                    tone,
                  )}
                >
                  {ACTION_LABEL[e.event_type] ?? e.event_type}
                </span>
                {e.agent_id ? (
                  <span className="text-sm text-slate-900 dark:text-slate-100">
                    {e.agent_name ?? e.agent_id.slice(0, 8) + "…"}
                  </span>
                ) : (
                  <span className="text-sm italic text-slate-500">no agent</span>
                )}
                <time
                  dateTime={e.created_at}
                  className="ml-auto text-xs text-slate-500"
                >
                  {timeAgo(e.created_at)}
                </time>
              </div>
              {e.notes ? (
                <p className="text-sm text-slate-600 dark:text-slate-300">
                  {e.notes}
                </p>
              ) : null}
              {e.actor_id ? (
                <p className="text-xs text-slate-400">by {e.actor_id}</p>
              ) : null}
            </div>
          </li>
        )
      })}
    </ol>
  )
}
