import { cn } from "@/lib/utils"
import type { CapabilityCategory } from "@/lib/types"

const CATEGORY_TONES: Record<string, string> = {
  architecture:
    "bg-violet-50 text-violet-700 ring-violet-200 dark:bg-violet-900/30 dark:text-violet-200 dark:ring-violet-800",
  coding: "bg-sky-50 text-sky-700 ring-sky-200 dark:bg-sky-900/30 dark:text-sky-200 dark:ring-sky-800",
  testing:
    "bg-emerald-50 text-emerald-700 ring-emerald-200 dark:bg-emerald-900/30 dark:text-emerald-200 dark:ring-emerald-800",
  security:
    "bg-rose-50 text-rose-700 ring-rose-200 dark:bg-rose-900/30 dark:text-rose-200 dark:ring-rose-800",
  devops:
    "bg-amber-50 text-amber-700 ring-amber-200 dark:bg-amber-900/30 dark:text-amber-200 dark:ring-amber-800",
  leadership:
    "bg-indigo-50 text-indigo-700 ring-indigo-200 dark:bg-indigo-900/30 dark:text-indigo-200 dark:ring-indigo-800",
}

const DEFAULT_TONE =
  "bg-slate-50 text-slate-700 ring-slate-200 dark:bg-slate-800 dark:text-slate-200 dark:ring-slate-700"

export function CapabilityChip({
  name,
  category,
  displayName,
  onRemove,
  className,
}: {
  /** Capability name (canonical id, per spec §2.1) */
  name: string
  /** Optional category for tone */
  category?: CapabilityCategory | string
  /** Optional human label */
  displayName?: string
  /** Optional remove handler — renders an X button when present */
  onRemove?: () => void
  className?: string
}) {
  const tone = (category && CATEGORY_TONES[category]) || DEFAULT_TONE
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-medium ring-1 ring-inset",
        tone,
        className,
      )}
    >
      <span className="truncate max-w-[16ch]">{displayName ?? name}</span>
      {onRemove ? (
        <button
          type="button"
          onClick={onRemove}
          className="-mr-1 ml-0.5 inline-flex h-4 w-4 items-center justify-center rounded-full text-current/70 hover:bg-black/10 dark:hover:bg-white/10"
          aria-label={`Remove ${displayName ?? name}`}
        >
          ×
        </button>
      ) : null}
    </span>
  )
}
