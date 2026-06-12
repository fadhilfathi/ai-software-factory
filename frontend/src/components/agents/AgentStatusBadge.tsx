import { cn } from "@/lib/utils"
import type { AgentStatus } from "@/lib/types"

const STATUS_LABELS: Record<AgentStatus, string> = {
  initializing: "Initializing",
  idle: "Idle",
  busy: "Busy",
  paused: "Paused",
  error: "Error",
  retired: "Retired",
}

const STATUS_TONES: Record<AgentStatus, string> = {
  initializing:
    "bg-slate-100 text-slate-700 ring-slate-200 dark:bg-slate-800 dark:text-slate-200 dark:ring-slate-700",
  idle: "bg-emerald-50 text-emerald-700 ring-emerald-200 dark:bg-emerald-900/30 dark:text-emerald-200 dark:ring-emerald-800",
  busy: "bg-sky-50 text-sky-700 ring-sky-200 dark:bg-sky-900/30 dark:text-sky-200 dark:ring-sky-800",
  paused:
    "bg-amber-50 text-amber-700 ring-amber-200 dark:bg-amber-900/30 dark:text-amber-200 dark:ring-amber-800",
  error: "bg-rose-50 text-rose-700 ring-rose-200 dark:bg-rose-900/30 dark:text-rose-200 dark:ring-rose-800",
  retired:
    "bg-zinc-100 text-zinc-600 ring-zinc-200 dark:bg-zinc-800 dark:text-zinc-300 dark:ring-zinc-700",
}

export function AgentStatusBadge({
  status,
  className,
}: {
  status: AgentStatus
  className?: string
}) {
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-medium ring-1 ring-inset",
        STATUS_TONES[status] ?? STATUS_TONES.idle,
        className,
      )}
    >
      <span
        className={cn(
          "h-1.5 w-1.5 rounded-full",
          status === "busy" && "bg-sky-500 animate-pulse",
          status === "idle" && "bg-emerald-500",
          status === "paused" && "bg-amber-500",
          status === "error" && "bg-rose-500",
          status === "initializing" && "bg-slate-400 animate-pulse",
          status === "retired" && "bg-zinc-400",
        )}
      />
      {STATUS_LABELS[status] ?? status}
    </span>
  )
}
