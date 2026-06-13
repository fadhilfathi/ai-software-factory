"use client"

import { useMemo, useState } from "react"
import { diffWords, diffLines, type Change } from "diff"
import { cn } from "@/lib/utils"
import type { DeliverableVersion } from "@/lib/types"

type DiffMode = "words" | "lines"

/**
 * VersionDiff — text diff between two versions of a deliverable.
 *
 * Per Lead's brief (TASK-409): use the `diff` library with
 * `diffWords` or `diffLines` (our call). Default is word-level since
 * deliverable content is usually small prose; the user can toggle to
 * line-level for structural comparison.
 *
 * Render rules:
 *   - Added parts → green background
 *   - Removed parts → red background with strikethrough
 *   - Unchanged parts → normal text
 *
 * If either side is missing, we show a one-sided "this is new" or
 * "this is gone" hint rather than crash.
 */
type VersionDiffProps = {
  versions: DeliverableVersion[]
  /** Optional default for the "from" version. */
  initialFrom?: number
  /** Optional default for the "to" version. */
  initialTo?: number
  /** Compact card style for the version-history page. */
  compact?: boolean
  className?: string
}

export function VersionDiff({
  versions,
  initialFrom,
  initialTo,
  compact = false,
  className,
}: VersionDiffProps) {
  // Versions come DESC from the API (newest first). We pick the
  // highest-numbered one as the default "to" and the next one down
  // as the default "from" so the diff is meaningful without the user
  // touching the dropdowns.
  const sorted = useMemo(
    () => [...versions].sort((a, b) => a.version - b.version),
    [versions],
  )
  const defaultTo = initialTo ?? sorted[sorted.length - 1]?.version
  const defaultFrom =
    initialFrom ??
    (defaultTo !== undefined
      ? sorted.find((v) => v.version < defaultTo)?.version
      : undefined)

  const [fromVer, setFromVer] = useState<number | undefined>(defaultFrom)
  const [toVer, setToVer] = useState<number | undefined>(defaultTo)
  const [mode, setMode] = useState<DiffMode>("words")

  const from = sorted.find((v) => v.version === fromVer)
  const to = sorted.find((v) => v.version === toVer)

  const diffs: Change[] = useMemo(() => {
    if (!from || !to) return []
    return mode === "words" ? diffWords(from.content, to.content) : diffLines(from.content, to.content)
  }, [from, to, mode])

  return (
    <div
      data-testid="version-diff"
      className={cn(
        "rounded-md border border-slate-200 dark:border-slate-800",
        className,
      )}
    >
      {/* Controls */}
      <div
        className={cn(
          "flex flex-wrap items-center gap-2 border-b border-slate-200 bg-slate-50 p-3 dark:border-slate-800 dark:bg-slate-900/50",
          compact && "p-2",
        )}
      >
        <div className="flex items-center gap-1.5">
          <label className="text-xs font-medium text-slate-500">From</label>
          <select
            value={fromVer ?? ""}
            onChange={(e) => setFromVer(e.target.value ? Number(e.target.value) : undefined)}
            className="rounded border border-slate-300 bg-white px-2 py-1 text-xs dark:border-slate-700 dark:bg-slate-900"
          >
            <option value="">— pick —</option>
            {sorted.map((v) => (
              <option key={v.id} value={v.version} disabled={v.version === toVer}>
                v{v.version}
              </option>
            ))}
          </select>
        </div>
        <span className="text-slate-400">→</span>
        <div className="flex items-center gap-1.5">
          <label className="text-xs font-medium text-slate-500">To</label>
          <select
            value={toVer ?? ""}
            onChange={(e) => setToVer(e.target.value ? Number(e.target.value) : undefined)}
            className="rounded border border-slate-300 bg-white px-2 py-1 text-xs dark:border-slate-700 dark:bg-slate-900"
          >
            <option value="">— pick —</option>
            {sorted.map((v) => (
              <option key={v.id} value={v.version} disabled={v.version === fromVer}>
                v{v.version}
              </option>
            ))}
          </select>
        </div>
        <div className="ml-auto flex items-center gap-1 rounded border border-slate-300 p-0.5 text-xs dark:border-slate-700">
          <button
            type="button"
            onClick={() => setMode("words")}
            className={cn(
              "rounded px-2 py-0.5",
              mode === "words"
                ? "bg-sky-600 text-white"
                : "text-slate-500 hover:text-slate-700 dark:hover:text-slate-300",
            )}
            aria-pressed={mode === "words"}
          >
            Words
          </button>
          <button
            type="button"
            onClick={() => setMode("lines")}
            className={cn(
              "rounded px-2 py-0.5",
              mode === "lines"
                ? "bg-sky-600 text-white"
                : "text-slate-500 hover:text-slate-700 dark:hover:text-slate-300",
            )}
            aria-pressed={mode === "lines"}
          >
            Lines
          </button>
        </div>
      </div>

      {/* Diff body */}
      <div className={cn("p-4", compact && "p-3")}>
        {!from && !to ? (
          <p className="text-sm text-slate-500">
            Pick two versions to compare.
          </p>
        ) : !from ? (
          <p className="text-sm text-slate-500">
            No &quot;from&quot; version selected. Showing v{to?.version} as new content.
          </p>
        ) : !to ? (
          <p className="text-sm text-slate-500">
            No &quot;to&quot; version selected. Showing v{from.version} as the original.
          </p>
        ) : from.content === to.content ? (
          <p className="text-sm italic text-slate-500">
            v{from.version} and v{to.version} have identical content.
          </p>
        ) : (
          <pre className="whitespace-pre-wrap break-words font-sans text-sm leading-6">
            {diffs.map((part, i) => {
              if (part.added) {
                return (
                  <span
                    key={i}
                    data-testid="diff-added"
                    className="rounded bg-emerald-100 px-0.5 text-emerald-900 dark:bg-emerald-900/40 dark:text-emerald-100"
                  >
                    {part.value}
                  </span>
                )
              }
              if (part.removed) {
                return (
                  <span
                    key={i}
                    data-testid="diff-removed"
                    className="rounded bg-rose-100 px-0.5 text-rose-900 line-through dark:bg-rose-900/40 dark:text-rose-100"
                  >
                    {part.value}
                  </span>
                )
              }
              return (
                <span key={i} data-testid="diff-unchanged">
                  {part.value}
                </span>
              )
            })}
          </pre>
        )}
      </div>
    </div>
  )
}
