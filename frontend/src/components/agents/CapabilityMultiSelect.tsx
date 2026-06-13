"use client"

import { useState } from "react"

import { useCapabilities } from "@/lib/hooks"
import { cn } from "@/lib/utils"
import { CapabilityChip } from "./CapabilityChip"

export function CapabilityMultiSelect({
  value,
  onChange,
  excludeIds,
  placeholder = "Filter by capability…",
  className,
}: {
  /** Capability names currently selected. */
  value: string[]
  onChange: (next: string[]) => void
  /** Optional list of capability names to exclude (e.g. already assigned). */
  excludeIds?: string[]
  placeholder?: string
  className?: string
}) {
  const [open, setOpen] = useState(false)
  const { data: capPage, isLoading } = useCapabilities()
  const capabilities = capPage?.data ?? []
  const excluded = new Set(excludeIds ?? [])

  const selected = value
    .map((name) => capabilities.find((c) => c.name === name))
    .filter((c): c is NonNullable<typeof c> => Boolean(c))

  function toggle(name: string) {
    if (value.includes(name)) {
      onChange(value.filter((v) => v !== name))
    } else {
      onChange([...value, name])
    }
  }

  return (
    <div className={cn("relative", className)}>
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        className={cn(
          "flex w-full items-center justify-between gap-2 rounded-md border border-slate-300 bg-white px-3 py-1.5 text-left text-sm shadow-sm transition hover:border-slate-400",
          "dark:border-slate-700 dark:bg-slate-900 dark:hover:border-slate-600",
        )}
        aria-haspopup="listbox"
        aria-expanded={open}
      >
        <span className="flex flex-1 flex-wrap items-center gap-1.5 truncate">
          {selected.length === 0 ? (
            <span className="text-slate-500 dark:text-slate-400">{placeholder}</span>
          ) : (
            selected.map((c) => (
              <CapabilityChip
                key={c.name}
                name={c.name}
                displayName={c.display_name}
                category={c.category}
                onRemove={() => toggle(c.name)}
              />
            ))
          )}
        </span>
        <span aria-hidden className="text-slate-400">▾</span>
      </button>

      {open ? (
        <>
          {/* click-outside catcher */}
          <button
            type="button"
            className="fixed inset-0 z-10 cursor-default"
            aria-label="Close capability menu"
            onClick={() => setOpen(false)}
          />
          <div
            role="listbox"
            className="absolute z-20 mt-1 max-h-64 w-full overflow-auto rounded-md border border-slate-200 bg-white p-1 shadow-lg dark:border-slate-700 dark:bg-slate-900"
          >
            {isLoading ? (
              <div className="px-3 py-2 text-sm text-slate-500">Loading…</div>
            ) : capabilities.length === 0 ? (
              <div className="px-3 py-2 text-sm text-slate-500">
                No capabilities in catalog.
              </div>
            ) : (
              capabilities.map((c) => {
                const isSelected = value.includes(c.name)
                const isExcluded = excluded.has(c.name)
                return (
                  <button
                    type="button"
                    key={c.name}
                    onClick={() => !isExcluded && toggle(c.name)}
                    disabled={isExcluded}
                    className={cn(
                      "flex w-full items-center justify-between rounded px-2 py-1.5 text-left text-sm transition",
                      isExcluded
                        ? "cursor-not-allowed opacity-50"
                        : "hover:bg-slate-100 dark:hover:bg-slate-800",
                      isSelected && "bg-slate-100 dark:bg-slate-800",
                    )}
                    role="option"
                    aria-selected={isSelected}
                  >
                    <span className="flex items-center gap-2">
                      <input
                        type="checkbox"
                        checked={isSelected}
                        readOnly
                        className="h-3.5 w-3.5 accent-sky-600"
                      />
                      <span className="truncate">{c.display_name}</span>
                      <span className="text-xs text-slate-400">({c.name})</span>
                    </span>
                    <span className="text-xs text-slate-400">{c.category}</span>
                  </button>
                )
              })
            )}
          </div>
        </>
      ) : null}
    </div>
  )
}
