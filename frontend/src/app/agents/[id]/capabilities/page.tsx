"use client"

import { use, useEffect, useMemo, useState } from "react"
import Link from "next/link"

import {
  useAgent,
  useAgentCapabilities,
  useCapabilities,
  useUpdateAgent,
} from "@/lib/hooks"
import type { AgentCapability } from "@/lib/types"
import { cn } from "@/lib/utils"

import { PageHeader } from "@/components/layout/PageHeader"
import { ErrorBlock } from "@/components/ui/ErrorBlock"
import { Skeleton } from "@/components/ui/Skeleton"
import { EmptyState } from "@/components/ui/EmptyState"
import { SpinnerButton } from "@/components/ui/Spinner"
import { ProjectPickerGate } from "@/components/agents/ProjectPickerGate"
import { CapabilityMultiSelect } from "@/components/agents/CapabilityMultiSelect"

export default function AgentCapabilitiesPage({
  params,
}: {
  params: Promise<{ id: string }>
}) {
  return (
    <ProjectPickerGate>
      <AgentCapabilities params={params} />
    </ProjectPickerGate>
  )
}

function AgentCapabilities({
  params,
}: {
  params: Promise<{ id: string }>
}) {
  const { id } = use(params)
  const agentQuery = useAgent(id)
  const richQuery = useAgentCapabilities(id)
  const catalogQuery = useCapabilities()
  const updateAgent = useUpdateAgent()

  // Local form state for the "add" dropdown. We commit the full
  // capabilities array on save (replace semantics, per spec §1.4).
  const [draftCapabilities, setDraftCapabilities] = useState<string[]>([])
  const [hydrated, setHydrated] = useState(false)

  // Seed draft from the agent's current capabilities exactly once.
  useEffect(() => {
    if (agentQuery.data && !hydrated) {
      setDraftCapabilities(agentQuery.data.capabilities ?? [])
      setHydrated(true)
    }
  }, [agentQuery.data, hydrated])

  // Build a lookup from the catalog so the displayed chip uses the
  // human-friendly display_name.
  const catalogByName = useMemo(() => {
    const m = new Map<string, { display_name: string; category: string }>()
    for (const c of catalogQuery.data?.data ?? []) {
      m.set(c.name, { display_name: c.display_name, category: c.category })
    }
    return m
  }, [catalogQuery.data])

  // Rich capability rows (with proficiency) for the read-only matrix.
  const richByName = useMemo(() => {
    const m = new Map<string, AgentCapability>()
    for (const c of richQuery.data ?? []) m.set(c.name, c)
    return m
  }, [richQuery.data])

  const dirty = useMemo(() => {
    if (!agentQuery.data) return false
    const a = [...(agentQuery.data.capabilities ?? [])].sort()
    const b = [...draftCapabilities].sort()
    return a.length !== b.length || a.some((v, i) => v !== b[i])
  }, [agentQuery.data, draftCapabilities])

  async function onSave() {
    if (!agentQuery.data) return
    await updateAgent.mutateAsync({
      id: agentQuery.data.id,
      payload: {
        capabilities: draftCapabilities,
        ...(agentQuery.data.version != null
          ? { version: agentQuery.data.version }
          : {}),
      },
    })
  }

  if (agentQuery.isError) {
    return (
      <ErrorBlock
        title="Could not load agent"
        message={
          (agentQuery.error as Error | undefined)?.message ??
          "An error occurred while loading the agent."
        }
        actions={
          <button
            type="button"
            onClick={() => agentQuery.refetch()}
            className="rounded-md border border-slate-300 px-3 py-1.5 text-sm hover:bg-slate-100 dark:border-slate-700 dark:hover:bg-slate-800"
          >
            Retry
          </button>
        }
      />
    )
  }

  if (agentQuery.isLoading || !agentQuery.data) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-10 w-1/3" />
        <Skeleton className="h-32" />
      </div>
    )
  }

  const agent = agentQuery.data

  return (
    <div className="space-y-6">
      <PageHeader
        title={`Capabilities: ${agent.name}`}
        actions={
          <>
            <p className="text-xs text-slate-500">
              Changes are committed via PUT /v1/agents/:id (replace semantics).
            </p>
            <Link
              href={`/agents/${agent.id}`}
              className="rounded-md border border-slate-300 px-3 py-1.5 text-sm hover:border-slate-400 dark:border-slate-700"
            >
              Back to overview
            </Link>
          </>
        }
      />

      {updateAgent.isError ? (
        <ErrorBlock
          message={
            (updateAgent.error as Error & { status?: number })?.status === 409
              ? "This agent was modified by someone else. Refresh and try again."
              : (updateAgent.error as Error | undefined)?.message ??
                "Failed to update agent capabilities."
          }
        />
      ) : null}

      <section
        className={cn(
          "rounded-lg border border-slate-200 bg-white p-6 space-y-4",
          "dark:border-slate-800 dark:bg-slate-900",
        )}
      >
        <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">
          Assigned capabilities
        </h2>

        {draftCapabilities.length === 0 ? (
          <EmptyState
            title="No capabilities assigned"
            description="Add a capability from the dropdown below."
          />
        ) : (
          <ul className="divide-y divide-slate-100 dark:divide-slate-800">
            {draftCapabilities.map((name) => {
              const rich = richByName.get(name)
              const cat = catalogByName.get(name)
              return (
                <li
                  key={name}
                  className="flex items-center justify-between gap-3 py-2.5"
                >
                  <div className="min-w-0">
                    <div className="truncate text-sm font-medium text-slate-900 dark:text-slate-100">
                      {cat?.display_name ?? name}
                    </div>
                    <div className="truncate text-xs text-slate-500">
                      {name} · {cat?.category ?? "—"}
                    </div>
                  </div>

                  {/* Proficiency: read-only. The PUT body doesn't carry
                      per-capability proficiency, so we display what the
                      server returned and don't let the user move it. */}
                  <div className="flex items-center gap-2">
                    <ProficiencyBar value={rich?.proficiency} />
                    <button
                      type="button"
                      onClick={() =>
                        setDraftCapabilities((cur) =>
                          cur.filter((n) => n !== name),
                        )
                      }
                      className="rounded-md border border-rose-200 bg-rose-50 px-2 py-1 text-xs font-medium text-rose-700 hover:bg-rose-100 dark:border-rose-700 dark:bg-rose-900/30 dark:text-rose-200"
                      aria-label={`Remove ${name}`}
                    >
                      Remove
                    </button>
                  </div>
                </li>
              )
            })}
          </ul>
        )}

        <div className="border-t border-slate-200 pt-4 dark:border-slate-800">
          <label
            htmlFor="capability-add"
            className="block text-sm font-medium text-slate-700 dark:text-slate-200"
          >
            Add capabilities
          </label>
          <p className="mb-2 text-xs text-slate-500">
            Pick from the catalog. The catalog is loaded via GET /v1/capabilities.
          </p>
          <CapabilityMultiSelect
            value={draftCapabilities}
            onChange={setDraftCapabilities}
            excludeIds={[]}
          />
        </div>

        <div className="flex items-center justify-end gap-2 border-t border-slate-200 pt-4 dark:border-slate-800">
          <span
            className={cn(
              "mr-auto text-xs",
              dirty ? "text-amber-600" : "text-slate-400",
            )}
          >
            {dirty ? "Unsaved changes" : "No pending changes"}
          </span>
          <button
            type="button"
            disabled={!dirty || updateAgent.isPending}
            onClick={() => {
              if (!agentQuery.data) return
              setDraftCapabilities(agentQuery.data.capabilities ?? [])
            }}
            className="rounded-md border border-slate-300 px-3 py-1.5 text-sm disabled:opacity-50 dark:border-slate-700"
          >
            Reset
          </button>
          <SpinnerButton
            type="button"
            loading={updateAgent.isPending}
            disabled={!dirty}
            onClick={onSave}
          >
            Save
          </SpinnerButton>
        </div>
      </section>
    </div>
  )
}

function ProficiencyBar({ value }: { value?: number }) {
  // Defensive: proficiency is response-only and may be missing. Render an
  // empty bar with a "—" in that case so the layout doesn't jump.
  const filled = typeof value === "number" ? Math.max(0, Math.min(5, value)) : 0
  return (
    <div
      className="flex items-center gap-1"
      title={
        typeof value === "number"
          ? `Proficiency: ${value}/5`
          : "Proficiency unavailable"
      }
    >
      <div className="flex gap-0.5">
        {Array.from({ length: 5 }).map((_, i) => (
          <span
            key={i}
            className={cn(
              "h-2 w-3 rounded-sm",
              i < filled
                ? "bg-sky-500"
                : "bg-slate-200 dark:bg-slate-700",
            )}
          />
        ))}
      </div>
      <span className="w-6 text-right text-xs text-slate-500">
        {typeof value === "number" ? `${value}/5` : "—"}
      </span>
    </div>
  )
}
