"use client"

import { useMemo, useState } from "react"

import { useAgents } from "@/lib/hooks"
import { cn } from "@/lib/utils"
import type { Agent, AgentStatus } from "@/lib/types"

import { AgentStatusBadge } from "./AgentStatusBadge"
import { CapabilityChip } from "./CapabilityChip"

export type AgentPickerProps = {
  value: string | null
  onChange: (agentId: string | null) => void
  /**
   * If set, the picker only shows agents whose `capabilities` array is a
   * superset of this list (client-side filter — see Lead's brief: the API
   * doesn't have a multi-cap filter).
   */
  requiredCapabilities?: string[]
  /** Optional list of agent ids to hide (e.g. retired or busy). */
  excludeIds?: string[]
  /** Status filter; defaults to "idle" per Lead's brief. */
  statusFilter?: AgentStatus | AgentStatus[] | "all"
  /** Render helper for the empty state. */
  emptyHint?: string
  /** Limit for the underlying list query. */
  limit?: number
  className?: string
}

const DEFAULT_STATUS: AgentStatus[] = ["idle", "initializing"]

export function AgentPicker({
  value,
  onChange,
  requiredCapabilities = [],
  excludeIds = [],
  statusFilter = DEFAULT_STATUS,
  emptyHint = "No matching agents.",
  limit = 100,
  className,
}: AgentPickerProps) {
  const [search, setSearch] = useState("")
  const [open, setOpen] = useState(false)

  // Map the statusFilter to a `status` query string. The useAgents hook
  // only takes a single status, so we just pick the first non-"all" one.
  const statusQuery =
    statusFilter === "all"
      ? undefined
      : Array.isArray(statusFilter)
      ? statusFilter[0]
      : statusFilter

  const agentsQuery = useAgents({ status: statusQuery, limit })

  const filtered = useMemo(() => {
    const list = (agentsQuery.data?.data ?? []) as Agent[]
    const searchLc = search.trim().toLowerCase()
    const requiredSet = new Set(requiredCapabilities)
    const excludeSet = new Set(excludeIds)
    return list
      .filter((a) => !excludeSet.has(a.id))
      .filter((a) => {
        if (requiredSet.size === 0) return true
        const aSet = new Set(a.capabilities)
        for (const r of requiredSet) {
          if (!aSet.has(r)) return false
        }
        return true
      })
      .filter((a) => {
        if (!searchLc) return true
        return (
          a.name.toLowerCase().includes(searchLc) ||
          a.role.toLowerCase().includes(searchLc)
        )
      })
  }, [agentsQuery.data, requiredCapabilities, excludeIds, search])

  const selected = useMemo(
    () =>
      (agentsQuery.data?.data ?? []).find((a) => a.id === value) ?? null,
    [agentsQuery.data, value],
  )

  return (
    <div className={cn("relative", className)}>
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        className={cn(
          "flex w-full items-center justify-between gap-3 rounded-md border border-slate-300 bg-white px-3 py-2 text-left text-sm shadow-sm transition hover:border-slate-400",
          "dark:border-slate-700 dark:bg-slate-900 dark:hover:border-slate-600",
        )}
        aria-haspopup="listbox"
        aria-expanded={open}
      >
        {selected ? (
          <div className="flex min-w-0 flex-1 items-center gap-2">
            <span className="truncate font-medium text-slate-900 dark:text-slate-100">
              {selected.name}
            </span>
            <span className="truncate text-xs text-slate-500">{selected.role}</span>
            <AgentStatusBadge status={selected.status} className="ml-auto" />
          </div>
        ) : (
          <span className="text-slate-500 dark:text-slate-400">
            Select an agent…
          </span>
        )}
        <span aria-hidden className="text-slate-400">▾</span>
      </button>

      {open ? (
        <>
          <button
            type="button"
            className="fixed inset-0 z-10 cursor-default"
            aria-label="Close agent menu"
            onClick={() => setOpen(false)}
          />
          <div className="absolute z-20 mt-1 w-full rounded-md border border-slate-200 bg-white shadow-lg dark:border-slate-700 dark:bg-slate-900">
            <div className="border-b border-slate-200 p-2 dark:border-slate-700">
              <input
                type="text"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="Search by name or role…"
                className={cn(
                  "w-full rounded-md border border-slate-300 bg-white px-2 py-1 text-sm",
                  "dark:border-slate-700 dark:bg-slate-900",
                )}
              />
            </div>
            <ul role="listbox" className="max-h-72 overflow-auto p-1">
              {agentsQuery.isLoading ? (
                <li className="px-3 py-2 text-sm text-slate-500">Loading…</li>
              ) : filtered.length === 0 ? (
                <li className="px-3 py-2 text-sm text-slate-500">{emptyHint}</li>
              ) : (
                filtered.map((a) => {
                  const isSelected = a.id === value
                  return (
                    <li key={a.id}>
                      <button
                        type="button"
                        onClick={() => {
                          onChange(a.id)
                          setOpen(false)
                        }}
                        className={cn(
                          "flex w-full items-center justify-between gap-2 rounded px-2 py-1.5 text-left text-sm transition",
                          "hover:bg-slate-100 dark:hover:bg-slate-800",
                          isSelected && "bg-slate-100 dark:bg-slate-800",
                        )}
                        role="option"
                        aria-selected={isSelected}
                      >
                        <span className="flex min-w-0 flex-1 flex-col">
                          <span className="flex items-center gap-2">
                            <span className="truncate font-medium text-slate-900 dark:text-slate-100">
                              {a.name}
                            </span>
                            <span className="truncate text-xs text-slate-500">
                              {a.role}
                            </span>
                          </span>
                          {a.capabilities.length > 0 ? (
                            <span className="mt-0.5 flex flex-wrap gap-1">
                              {a.capabilities.slice(0, 4).map((c) => (
                                <CapabilityChip key={c} name={c} />
                              ))}
                              {a.capabilities.length > 4 ? (
                                <span className="text-xs text-slate-400">
                                  +{a.capabilities.length - 4} more
                                </span>
                              ) : null}
                            </span>
                          ) : null}
                        </span>
                        <AgentStatusBadge status={a.status} />
                      </button>
                    </li>
                  )
                })
              )}
            </ul>
          </div>
        </>
      ) : null}
    </div>
  )
}
