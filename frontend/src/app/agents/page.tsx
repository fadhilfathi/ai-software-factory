"use client"

import { useEffect, useMemo, useState } from "react"
import Link from "next/link"

import { useAgents, useCapabilities } from "@/lib/hooks"
import { useProjectFilters } from "@/hooks/useProjectFilters"
import { cn } from "@/lib/utils"
import type {
  AgentListFilters,
  AgentStatus,
} from "@/lib/types"

import { PageHeader } from "@/components/layout/PageHeader"
import { FilterBar, SearchInput } from "@/components/shared/FilterBar"
import { ErrorBlock } from "@/components/ui/ErrorBlock"
import { Skeleton } from "@/components/ui/Skeleton"
import { EmptyState } from "@/components/ui/EmptyState"
import { Toggle } from "@/components/ui/Toggle"
import { PaginationInfo } from "@/components/shared/PaginationInfo"
import { AgentCard } from "@/components/agents/AgentCard"
import { CapabilityMultiSelect } from "@/components/agents/CapabilityMultiSelect"
import { ProjectPickerGate } from "@/components/agents/ProjectPickerGate"

const ROLE_TABS: Array<{ value: string; label: string }> = [
  { value: "all", label: "All" },
  { value: "architect", label: "Architects" },
  { value: "developer", label: "Developers" },
  { value: "qa", label: "QA" },
  { value: "devops", label: "DevOps" },
  { value: "security", label: "Security" },
  { value: "lead", label: "Leads" },
]

const STATUS_OPTIONS: Array<{ value: AgentStatus | "all"; label: string }> = [
  { value: "all", label: "All statuses" },
  { value: "initializing", label: "Initializing" },
  { value: "idle", label: "Idle" },
  { value: "busy", label: "Busy" },
  { value: "paused", label: "Paused" },
  { value: "error", label: "Error" },
  { value: "retired", label: "Retired" },
]

const SORT_OPTIONS: Array<{ value: NonNullable<AgentListFilters["sort"]>; label: string }> = [
  { value: "-last_active_at", label: "Most recently active" },
  { value: "last_active_at", label: "Least recently active" },
  { value: "name", label: "Name (A→Z)" },
  { value: "-name", label: "Name (Z→A)" },
  { value: "-created_at", label: "Newest first" },
  { value: "created_at", label: "Oldest first" },
]

const PAGE_SIZE = 24

export default function AgentsListPage() {
  return (
    <ProjectPickerGate>
      <AgentsListContent />
    </ProjectPickerGate>
  )
}

function AgentsListContent() {
  const { projectId } = useProjectFilters()
  const [role, setRole] = useState<string>("all")
  const [status, setStatus] = useState<AgentStatus | "all">("all")
  const [capabilities, setCapabilities] = useState<string[]>([])
  const [search, setSearch] = useState("")
  const [sort, setSort] =
    useState<NonNullable<AgentListFilters["sort"]>>("-last_active_at")
  const [includeRetired, setIncludeRetired] = useState(false)
  const [cursors, setCursors] = useState<string[]>([])
  const [page, setPage] = useState(0)

  // Reset pagination whenever a filter changes.
  useEffect(() => {
    setCursors([])
    setPage(0)
  }, [role, status, capabilities, search, sort, includeRetired])

  const filters: AgentListFilters = {
    role: role === "all" ? undefined : role,
    status: status === "all" ? undefined : status,
    capability: capabilities.length > 0 ? capabilities : undefined,
    search: search.trim() || undefined,
    sort,
    include_retired: includeRetired || undefined,
    cursor: cursors[page],
    limit: PAGE_SIZE,
  }

  const agentsQuery = useAgents(filters)
  const capabilitiesQuery = useCapabilities()
  const catalogIndex = useMemo(() => {
    const m = new Map<string, NonNullable<typeof capabilitiesQuery.data>["data"][number]>()
    for (const c of capabilitiesQuery.data?.data ?? []) m.set(c.name, c)
    return m
  }, [capabilitiesQuery.data])

  const agents = agentsQuery.data?.data ?? []
  const pageInfo = agentsQuery.data?.page_info
  const hasMore = pageInfo?.has_more ?? false

  return (
    <div className="space-y-6">
      <PageHeader
        title="Agents"
        description="Manage the agents in this project. Filter by role, status, or capability."
        actions={
          <Link
            href="/agents/new"
            className="inline-flex items-center gap-1.5 rounded-md bg-sky-600 px-3 py-1.5 text-sm font-medium text-white shadow-sm transition hover:bg-sky-700"
          >
            <span aria-hidden>+</span> New agent
          </Link>
        }
      />

      {/* Role tabs */}
      <div className="border-b border-slate-200 dark:border-slate-800">
        <nav className="-mb-px flex flex-wrap gap-2" aria-label="Filter by role">
          {ROLE_TABS.map((tab) => {
            const active = role === tab.value
            return (
              <button
                key={tab.value}
                type="button"
                onClick={() => setRole(tab.value)}
                className={cn(
                  "border-b-2 px-3 py-2 text-sm font-medium transition",
                  active
                    ? "border-sky-500 text-sky-700 dark:text-sky-300"
                    : "border-transparent text-slate-500 hover:border-slate-300 hover:text-slate-700 dark:text-slate-400 dark:hover:border-slate-600 dark:hover:text-slate-200",
                )}
                aria-pressed={active}
              >
                {tab.label}
              </button>
            )
          })}
        </nav>
      </div>

      <FilterBar>
        <SearchInput
          value={search}
          onChange={setSearch}
          placeholder="Search agents by name…"
          className="min-w-[18rem] flex-1"
        />
        <FilterBar.Select
          value={status}
          onChange={(v) => setStatus(v as AgentStatus | "all")}
          options={STATUS_OPTIONS}
          aria-label="Filter by status"
          className="w-44"
        />
        <FilterBar.Select
          value={sort}
          onChange={(v) =>
            setSort(v as NonNullable<AgentListFilters["sort"]>)
          }
          options={SORT_OPTIONS}
          aria-label="Sort agents"
          className="w-56"
        />
        <CapabilityMultiSelect
          value={capabilities}
          onChange={setCapabilities}
          className="w-72"
        />
        <label className="ml-auto flex items-center gap-2 text-sm text-slate-600 dark:text-slate-300">
          <Toggle
            checked={includeRetired}
            onChange={setIncludeRetired}
            label="Include retired"
          />
          Include retired
        </label>
      </FilterBar>

      {agentsQuery.isError ? (
        <ErrorBlock
          error={agentsQuery.error}
          onRetry={() => agentsQuery.refetch()}
        />
      ) : agentsQuery.isLoading ? (
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
          {Array.from({ length: 8 }).map((_, i) => (
            <Skeleton key={i} className="h-44" />
          ))}
        </div>
      ) : agents.length === 0 ? (
        <EmptyState
          title="No agents match your filters"
          description="Try clearing a filter, including retired agents, or creating a new agent."
          action={
            <Link
              href="/agents/new"
              className="inline-flex items-center gap-1.5 rounded-md bg-sky-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-sky-700"
            >
              Create an agent
            </Link>
          }
        />
      ) : (
        <>
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
            {agents.map((a) => (
              <AgentCard
                key={a.id}
                agent={a}
                capabilities={Array.from(catalogIndex.values())}
              />
            ))}
          </div>

          <div className="flex items-center justify-between border-t border-slate-200 pt-4 dark:border-slate-800">
            <PaginationInfo
              page={page + 1}
              total={agents.length}
              showing={agents.length}
              pages={hasMore ? page + 2 : page + 1}
            />
            <div className="flex items-center gap-2">
              <button
                type="button"
                disabled={page === 0}
                onClick={() => setPage((p) => Math.max(0, p - 1))}
                className="rounded-md border border-slate-300 px-3 py-1.5 text-sm disabled:opacity-50 dark:border-slate-700"
              >
                Previous
              </button>
              <button
                type="button"
                disabled={!hasMore || !pageInfo?.next_cursor}
                onClick={() => {
                  if (!pageInfo?.next_cursor) return
                  setCursors((c) => [...c, pageInfo.next_cursor as string])
                  setPage((p) => p + 1)
                }}
                className="rounded-md border border-slate-300 px-3 py-1.5 text-sm disabled:opacity-50 dark:border-slate-700"
              >
                Next
              </button>
            </div>
          </div>
        </>
      )}

      {!projectId ? (
        <p className="text-xs text-slate-400">
          Tip: agent management is project-scoped. Use the picker at the top
          to switch projects.
        </p>
      ) : null}
    </div>
  )
}
