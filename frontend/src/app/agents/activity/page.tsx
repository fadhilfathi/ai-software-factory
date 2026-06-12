"use client";

/**
 * Agent Activity Dashboard.
 *
 * TASK-410 — Agent Activity Dashboard.
 * Route: /agents/activity
 *
 * Layout (top → bottom):
 *   1. Header + filter bar (time range, agent multi-select)
 *   2. High-level metrics strip (MetricCards: total/active/completed/executions)
 *   3. TasksPerAgentChart — bar chart of task counts per agent
 *   4. Per-agent card grid — active, completed last 7d, completed last 30d
 *   5. ExecutionHistoryTimeline — project-wide executions in the window
 *
 * Data strategy (single fetch, client-side aggregation — no N+1):
 *   - GET /v1/projects/:id/tasks     → useTasks()
 *   - GET /v1/executions              → useExecutions()
 *   - GET /v1/agents                  → useAgents()
 *   Aggregate by assignee_id / agent_id in the page.
 *
 * Filters live in URL query params for deep-linkable / shareable views.
 */

import { use, useCallback, useMemo, Suspense } from "react";
import { useRouter, useSearchParams, usePathname } from "next/navigation";

import { ProjectPickerGate } from "@/components/agents/ProjectPickerGate";
import { ErrorBlock } from "@/components/ui/ErrorBlock";
import { Skeleton } from "@/components/ui/Skeleton";
import { EmptyState } from "@/components/ui/EmptyState";
import { MetricCard } from "@/components/shared/MetricCard";
import { Badge } from "@/components/ui/Badge";
import {
  TasksPerAgentChart,
  type AgentTaskCount,
} from "@/components/activity/TasksPerAgentChart";
import { AgentActivityCard } from "@/components/activity/AgentActivityCard";
import { ExecutionHistoryTimeline } from "@/components/activity/ExecutionHistoryTimeline";

import { useTasks, useExecutions, useAgents } from "@/lib/hooks";
import { useProjectFilters } from "@/hooks/useProjectFilters";
import type { Agent, AgentStatus, Task, Execution, TaskStatus } from "@/lib/types";

const TIME_RANGES = [
  { key: "7d", label: "Last 7 days", days: 7 },
  { key: "30d", label: "Last 30 days", days: 30 },
  { key: "90d", label: "Last 90 days", days: 90 },
  { key: "all", label: "All time", days: null },
] as const;

type RangeKey = (typeof TIME_RANGES)[number]["key"];

const ACTIVE_STATUSES: TaskStatus[] = ["in_progress", "blocked"];
const DONE_STATUS: TaskStatus = "done";

export default function AgentsActivityPage() {
  return (
    <ProjectPickerGate>
      <Suspense
        fallback={
          <div className="space-y-4">
            <Skeleton className="h-10 w-1/2" />
            <Skeleton className="h-32" />
            <Skeleton className="h-64" />
          </div>
        }
      >
        <AgentsActivity />
      </Suspense>
    </ProjectPickerGate>
  );
}

function AgentsActivity() {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const { projectId } = useProjectFilters();

  const rangeKey = parseRange(searchParams.get("range"));
  const agentFilter = parseAgentFilter(searchParams.get("agents"));

  /* ---- Data fetches (single each, project-scoped) ---- */
  const tasksQuery = useTasks({});
  const executionsQuery = useExecutions({});
  const agentsQuery = useAgents({});

  /* ---- URL param setters ---- */
  const setParam = useCallback(
    (key: "range" | "agents", value: string | null) => {
      const next = new URLSearchParams(searchParams.toString());
      if (value === null || value === "") {
        next.delete(key);
      } else {
        next.set(key, value);
      }
      const qs = next.toString();
      router.push(qs ? `${pathname}?${qs}` : pathname);
    },
    [router, pathname, searchParams],
  );

  const toggleAgent = useCallback(
    (id: string) => {
      const set = new Set(agentFilter);
      if (set.has(id)) set.delete(id);
      else set.add(id);
      setParam("agents", set.size > 0 ? Array.from(set).join(",") : null);
    },
    [agentFilter, setParam],
  );

  const setRange = useCallback(
    (r: RangeKey) => setParam("range", r === "7d" ? null : r),
    [setParam],
  );

  /* ---- Cutoff for time-range window ---- */
  const cutoff = useMemo(() => {
    const r = TIME_RANGES.find((x) => x.key === rangeKey);
    if (!r || r.days === null) return null;
    return new Date(Date.now() - r.days * 24 * 60 * 60 * 1000).toISOString();
  }, [rangeKey]);

  const cutoff7d = useMemo(
    () => new Date(Date.now() - 7 * 24 * 60 * 60 * 1000).toISOString(),
    [],
  );
  const cutoff30d = useMemo(
    () => new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString(),
    [],
  );

  /* ---- Normalise errors ---- */
  const isError =
    tasksQuery.isError || executionsQuery.isError || agentsQuery.isError;
  const error = tasksQuery.error ?? executionsQuery.error ?? agentsQuery.error;
  const isLoading =
    tasksQuery.isLoading || executionsQuery.isLoading || agentsQuery.isLoading;
  const refetch = () => {
    tasksQuery.refetch();
    executionsQuery.refetch();
    agentsQuery.refetch();
  };

  /* ---- Agent lookup tables ---- */
  const agentsById = useMemo(() => {
    const map: Record<string, Agent> = {};
    for (const a of agentsQuery.data?.data ?? []) {
      map[a.id] = a;
    }
    return map;
  }, [agentsQuery.data]);

  const allTasks: Task[] = tasksQuery.data ?? [];
  const allExecutions: Execution[] = executionsQuery.data?.data ?? [];
  const allAgents: Agent[] = agentsQuery.data?.data ?? [];

  /* ---- Apply agent filter (and prune) ---- */
  const tasksInScope = useMemo(() => {
    if (agentFilter.size === 0) return allTasks;
    return allTasks.filter(
      (t) => t.assignee_id && agentFilter.has(t.assignee_id),
    );
  }, [allTasks, agentFilter]);

  const executionsInScope = useMemo(() => {
    if (agentFilter.size === 0) return allExecutions;
    return allExecutions.filter(
      (e) => e.agent_id && agentFilter.has(e.agent_id),
    );
  }, [allExecutions, agentFilter]);

  /* ---- Aggregate per-agent stats ---- */
  const agentStats = useMemo(() => {
    return allAgents.map((agent) => {
      const myTasks = tasksInScope.filter((t) => t.assignee_id === agent.id);
      const myExecs = executionsInScope.filter((e) => e.agent_id === agent.id);

      const activeTasks = myTasks.filter((t) =>
        ACTIVE_STATUSES.includes(t.status as TaskStatus),
      ).length;
      const completedLast7d = myTasks.filter(
        (t) =>
          t.status === DONE_STATUS &&
          t.updated_at &&
          t.updated_at >= cutoff7d,
      ).length;
      const completedLast30d = myTasks.filter(
        (t) =>
          t.status === DONE_STATUS &&
          t.updated_at &&
          t.updated_at >= cutoff30d,
      ).length;

      const recentExecutions = cutoff
        ? myExecs.filter((e) => e.started_at && e.started_at >= cutoff).length
        : myExecs.length;

      return {
        agentId: agent.id,
        agentName: agent.name,
        status: (agent.status ?? "idle") as AgentStatus,
        totalTasks: myTasks.length,
        activeTasks,
        completedLast7d,
        completedLast30d,
        recentExecutions,
      };
    });
  }, [
    allAgents,
    tasksInScope,
    executionsInScope,
    cutoff,
    cutoff7d,
    cutoff30d,
  ]);

  /* ---- Chart data (same shape, top 12 by total) ---- */
  const chartData: AgentTaskCount[] = useMemo(() => {
    return agentStats.map((s) => ({
      agentId: s.agentId,
      agentName: s.agentName,
      total: s.totalTasks,
      active: s.activeTasks,
      completed: s.completedLast7d,
    }));
  }, [agentStats]);

  /* ---- Top-level metrics ---- */
  const metrics = useMemo(() => {
    const inWindow = cutoff
      ? tasksInScope.filter(
          (t) => t.created_at && t.created_at >= cutoff,
        )
      : tasksInScope;
    const activeCount = tasksInScope.filter((t) =>
      ACTIVE_STATUSES.includes(t.status as TaskStatus),
    ).length;
    const completedInWindow = cutoff
      ? tasksInScope.filter(
          (t) => t.status === DONE_STATUS && t.updated_at && t.updated_at >= cutoff,
        ).length
      : tasksInScope.filter((t) => t.status === DONE_STATUS).length;
    const executionsInWindow = cutoff
      ? executionsInScope.filter(
          (e) => e.started_at && e.started_at >= cutoff,
        ).length
      : executionsInScope.length;

    return {
      totalTasks: inWindow.length,
      activeTasks: activeCount,
      completedInWindow,
      executionsInWindow,
    };
  }, [tasksInScope, executionsInScope, cutoff]);

  /* ---- Render ---- */
  return (
    <div className="space-y-6">
      <header>
        <h1 className="text-2xl font-bold text-gray-100">Agent activity</h1>
        <p className="mt-1 text-sm text-gray-400">
          A read-only view of what the team is working on. Per-agent
          productivity, task distribution, and execution history.
        </p>
      </header>

      {/* Filter bar */}
      <div className="rounded-lg border border-gray-800 bg-gray-900/40 p-4">
        <div className="flex flex-wrap items-end gap-4">
          <div>
            <span className="mb-1 block text-xs font-medium text-gray-400">
              Time range
            </span>
            <div className="flex flex-wrap gap-1 rounded-md border border-gray-800 bg-gray-950 p-1">
              {TIME_RANGES.map((r) => (
                <button
                  key={r.key}
                  type="button"
                  onClick={() => setRange(r.key)}
                  className={
                    rangeKey === r.key
                      ? "rounded bg-emerald-600/20 px-3 py-1 text-xs font-medium text-emerald-200"
                      : "rounded px-3 py-1 text-xs font-medium text-gray-400 hover:bg-gray-800"
                  }
                >
                  {r.label}
                </button>
              ))}
            </div>
          </div>
          <div className="min-w-0 flex-1">
            <span className="mb-1 block text-xs font-medium text-gray-400">
              Agents{" "}
              {agentFilter.size > 0 ? (
                <button
                  type="button"
                  onClick={() => setParam("agents", null)}
                  className="ml-2 text-[10px] text-emerald-400 hover:text-emerald-300"
                >
                  clear
                </button>
              ) : null}
            </span>
            <div className="flex flex-wrap gap-1">
              {allAgents.length === 0 ? (
                <span className="text-xs text-gray-500">
                  No agents in this project.
                </span>
              ) : (
                allAgents.map((a) => {
                  const on = agentFilter.has(a.id);
                  return (
                    <button
                      key={a.id}
                      type="button"
                      onClick={() => toggleAgent(a.id)}
                      className={
                        on
                          ? "rounded border border-emerald-700 bg-emerald-900/30 px-2 py-0.5 text-[11px] font-medium text-emerald-200"
                          : "rounded border border-gray-700 bg-gray-900/40 px-2 py-0.5 text-[11px] font-medium text-gray-400 hover:bg-gray-800"
                      }
                    >
                      {a.name}
                    </button>
                  );
                })
              )}
            </div>
          </div>
        </div>
      </div>

      {/* Error */}
      {isError ? (
        <ErrorBlock
          title="Failed to load activity data"
          error={error}
          onRetry={refetch}
        />
      ) : null}

      {/* Loading skeleton */}
      {isLoading ? (
        <div className="space-y-6">
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
            {Array.from({ length: 4 }).map((_, i) => (
              <Skeleton.Card key={i} className="h-24" />
            ))}
          </div>
          <Skeleton className="h-72" />
        </div>
      ) : null}

      {/* Content */}
      {!isLoading && !isError ? (
        <>
          {/* Metrics strip */}
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
            <MetricCard
              label="Tasks in window"
              value={metrics.totalTasks}
            />
            <MetricCard
              label="Active (now)"
              value={metrics.activeTasks}
            />
            <MetricCard
              label="Completed (window)"
              value={metrics.completedInWindow}
            />
            <MetricCard
              label="Executions (window)"
              value={metrics.executionsInWindow}
            />
          </div>

          {/* Bar chart */}
          <section>
            <h2 className="mb-2 text-xs font-semibold uppercase tracking-wide text-gray-500">
              Tasks per agent
            </h2>
            <TasksPerAgentChart data={chartData} />
          </section>

          {/* Per-agent card grid */}
          <section>
            <div className="mb-2 flex items-baseline justify-between">
              <h2 className="text-xs font-semibold uppercase tracking-wide text-gray-500">
                Per-agent summary
              </h2>
              <span className="text-[10px] text-gray-500">
                showing {agentStats.length} agent
                {agentStats.length === 1 ? "" : "s"}
              </span>
            </div>
            {agentStats.length === 0 ? (
              <EmptyState
                icon="🤖"
                title="No agents in this project"
                description="The activity dashboard needs at least one agent to summarise."
              />
            ) : (
              <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
                {agentStats.map((s) => (
                  <AgentActivityCard key={s.agentId} stats={s} />
                ))}
              </div>
            )}
          </section>

          {/* Execution timeline */}
          <section>
            <div className="mb-2 flex items-baseline justify-between">
              <h2 className="text-xs font-semibold uppercase tracking-wide text-gray-500">
                Execution history
              </h2>
              <Badge color="gray" variant="outline">
                {executionsInScope.length} in window
              </Badge>
            </div>
            <ExecutionHistoryTimeline
              executions={executionsInScope}
              agentsById={Object.fromEntries(
                Object.entries(agentsById).map(([id, a]) => [id, a.name]),
              )}
              groupByAgent={false}
              emptyHint={
                cutoff
                  ? `No executions recorded in the last ${labelFor(rangeKey)}.`
                  : "No executions recorded yet."
              }
            />
          </section>
        </>
      ) : null}
    </div>
  );
}

/* ---------- helpers ---------- */

function parseRange(raw: string | null): RangeKey {
  if (!raw) return "7d";
  const valid = TIME_RANGES.find((r) => r.key === raw);
  return (valid?.key ?? "7d") as RangeKey;
}

function parseAgentFilter(raw: string | null): Set<string> {
  if (!raw) return new Set();
  return new Set(
    raw
      .split(",")
      .map((s) => s.trim())
      .filter(Boolean),
  );
}

function labelFor(r: RangeKey): string {
  return TIME_RANGES.find((x) => x.key === r)?.label ?? "window";
}
