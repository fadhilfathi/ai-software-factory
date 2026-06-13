"use client"

import Link from "next/link"
import { cn, timeAgo } from "@/lib/utils"
import type { Deliverable } from "@/lib/types"

type DeliverableCardProps = {
  deliverable: Deliverable
  /** Optional click handler that overrides the default link. */
  onClick?: () => void
  className?: string
}

const KIND_COLORS: Record<string, string> = {
  code: "bg-sky-100 text-sky-800 dark:bg-sky-900/40 dark:text-sky-200",
  doc: "bg-emerald-100 text-emerald-800 dark:bg-emerald-900/40 dark:text-emerald-200",
  design: "bg-violet-100 text-violet-800 dark:bg-violet-900/40 dark:text-violet-200",
  test_report: "bg-amber-100 text-amber-800 dark:bg-amber-900/40 dark:text-amber-200",
  config: "bg-slate-100 text-slate-800 dark:bg-slate-800 dark:text-slate-200",
  other: "bg-slate-100 text-slate-600 dark:bg-slate-800 dark:text-slate-300",
}

/**
 * DeliverableCard — one row in the deliverables browser.
 *
 * Shows: title, version chip, optional `kind` chip, optional
 * description, created/updated timestamps, and the truncated agent id
 * (the full agent name is fetched by the list page via the `useAgent`
 * hook if a projectId is set; we keep the card self-contained to
 * avoid N+1 in the list).
 *
 * Clicking the card routes to `/deliverables/[id]` (the markdown
 * viewer). The version-history link is in the top-right of the card.
 */
export function DeliverableCard({
  deliverable,
  onClick,
  className,
}: DeliverableCardProps) {
  const currentVersion = deliverable.version ?? deliverable.latest_version ?? 1
  const kind = deliverable.kind ?? "other"
  const kindColor = KIND_COLORS[kind] ?? KIND_COLORS.other

  const body = (
    <div
      className={cn(
        "flex flex-col gap-2 rounded-md border border-slate-200 bg-white p-4 transition hover:border-sky-300 hover:bg-sky-50/30 dark:border-slate-800 dark:bg-slate-900 dark:hover:border-sky-700 dark:hover:bg-sky-900/20",
        className,
      )}
    >
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <span
              className={cn(
                "rounded px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wide",
                kindColor,
              )}
            >
              {kind}
            </span>
            <span className="rounded bg-slate-100 px-1.5 py-0.5 font-mono text-[10px] text-slate-600 dark:bg-slate-800 dark:text-slate-300">
              v{currentVersion}
            </span>
          </div>
          <h3 className="mt-1.5 truncate font-medium text-slate-900 dark:text-slate-100">
            {deliverable.title}
          </h3>
          {deliverable.description ? (
            <p className="mt-0.5 line-clamp-2 text-xs text-slate-500">
              {deliverable.description}
            </p>
          ) : null}
        </div>
        <Link
          href={`/deliverables/${deliverable.id}/versions`}
          onClick={(e) => e.stopPropagation()}
          className="shrink-0 rounded-md border border-slate-300 px-2 py-1 text-[11px] text-slate-600 hover:bg-slate-100 dark:border-slate-700 dark:text-slate-300 dark:hover:bg-slate-800"
        >
          History
        </Link>
      </div>

      <div className="mt-1 flex flex-wrap items-center gap-3 text-[11px] text-slate-500">
        <span>
          task:{" "}
          <Link
            href={`/tasks/${deliverable.task_id}`}
            onClick={(e) => e.stopPropagation()}
            className="font-mono text-slate-600 hover:text-sky-700 dark:text-slate-400 dark:hover:text-sky-300"
          >
            {deliverable.task_id.slice(0, 8)}…
          </Link>
        </span>
        <span className="text-slate-300 dark:text-slate-600">·</span>
        <span>
          agent:{" "}
          <Link
            href={`/agents/${deliverable.agent_id}`}
            onClick={(e) => e.stopPropagation()}
            className="font-mono text-slate-600 hover:text-sky-700 dark:text-slate-400 dark:hover:text-sky-300"
          >
            {deliverable.agent_id.slice(0, 8)}…
          </Link>
        </span>
        <span className="ml-auto text-slate-400">
          created {timeAgo(deliverable.created_at)}
          {deliverable.updated_at && deliverable.updated_at !== deliverable.created_at
            ? ` · updated ${timeAgo(deliverable.updated_at)}`
            : null}
        </span>
      </div>
    </div>
  )

  if (onClick) {
    return (
      <button
        type="button"
        onClick={onClick}
        className="text-left w-full focus:outline-none focus-visible:ring-2 focus-visible:ring-sky-500"
      >
        {body}
      </button>
    )
  }

  return (
    <Link
      href={`/deliverables/${deliverable.id}`}
      className="block focus:outline-none focus-visible:ring-2 focus-visible:ring-sky-500"
    >
      {body}
    </Link>
  )
}
