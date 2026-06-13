"use client";

/**
 * Execution history timeline (per agent or project-wide).
 *
 * TASK-410 — Agent Activity Dashboard.
 * Vertical timeline grouped by date (day) with the agent's executions
 * stacked under each day header. Reuses the dot+line visual pattern
 * from the existing `LifecycleTimeline` (agents) and the task
 * `AssignmentHistoryTimeline` (TASK-408).
 *
 * `Execution.status` is a small enum (pending/running/completed/failed)
 * — each gets a colour dot and a short status label.
 */

import { useMemo } from "react";

import { Execution, ExecutionStatus } from "@/lib/types";
import { cn, timeAgo } from "@/lib/utils";

const STATUS_TONE: Record<ExecutionStatus, { dot: string; label: string }> = {
  pending: { dot: "bg-slate-400", label: "Pending" },
  running: { dot: "bg-sky-500", label: "Running" },
  succeeded: { dot: "bg-emerald-500", label: "Succeeded" },
  failed: { dot: "bg-rose-500", label: "Failed" },
  cancelled: { dot: "bg-amber-500", label: "Cancelled" },
};

export function ExecutionHistoryTimeline({
  executions,
  agentsById,
  emptyHint = "No executions recorded in this window.",
  groupByAgent = false,
}: {
  executions: Execution[];
  agentsById?: Record<string, string>;
  emptyHint?: string;
  /**
   * When true, group by agent (e.g. project-wide view with a sticky
   * agent label per cluster). When false (default), group by day.
   */
  groupByAgent?: boolean;
}) {
  const groups = useMemo(
    () => groupExecutions(executions, groupByAgent),
    [executions, groupByAgent],
  );

  if (executions.length === 0) {
    return (
      <div className="rounded-md border border-dashed border-gray-800 bg-gray-900/30 p-6 text-center text-sm text-gray-500">
        {emptyHint}
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {groups.map((group) => (
        <section key={group.key}>
          <h3 className="mb-2 text-xs font-semibold uppercase tracking-wide text-gray-500">
            {group.label}{" "}
            <span className="ml-1 font-normal normal-case text-gray-600">
              ({group.items.length})
            </span>
          </h3>
          <ol className="relative space-y-4 border-l border-gray-800 pl-5">
            {group.items.map((exec) => {
              const tone = STATUS_TONE[exec.status] ?? STATUS_TONE.pending;
              return (
                <li
                  key={exec.id}
                  className="relative rounded-md border border-gray-800 bg-gray-900/30 p-3"
                >
                  <span
                    className={cn(
                      "absolute -left-[27px] top-4 inline-block h-2.5 w-2.5 rounded-full ring-4 ring-gray-950",
                      tone.dot,
                    )}
                    aria-hidden
                  />
                  <div className="flex flex-wrap items-baseline justify-between gap-2">
                    <div className="min-w-0 flex-1">
                      <p className="truncate font-mono text-xs text-gray-300">
                        exec · {exec.id.slice(0, 8)}…
                      </p>
                      {agentsById && exec.agent_id ? (
                        <p className="mt-0.5 truncate text-[10px] text-gray-500">
                          agent: {agentsById[exec.agent_id] ?? exec.agent_id}
                        </p>
                      ) : null}
                    </div>
                    <span
                      className={cn(
                        "rounded-md border px-2 py-0.5 text-[10px] font-medium uppercase tracking-wide",
                        tone.dot.replace("bg-", "border-"),
                        "text-gray-200",
                      )}
                    >
                      {tone.label}
                    </span>
                  </div>
                  <div className="mt-2 flex flex-wrap gap-3 text-[10px] text-gray-500">
                    <span>started {timeAgo(exec.started_at)}</span>
                    {exec.completed_at ? (
                      <span>completed {timeAgo(exec.completed_at)}</span>
                    ) : null}
                    {exec.task_id ? (
                      <span className="font-mono">
                        task {exec.task_id.slice(0, 8)}…
                      </span>
                    ) : null}
                  </div>
                </li>
              );
            })}
          </ol>
        </section>
      ))}
    </div>
  );
}

type Group = { key: string; label: string; items: Execution[] };

function groupExecutions(
  executions: Execution[],
  groupByAgent: boolean,
): Group[] {
  if (groupByAgent) {
    const map = new Map<string, Execution[]>();
    for (const e of executions) {
      const k = e.agent_id ?? "unknown";
      const arr = map.get(k) ?? [];
      arr.push(e);
      map.set(k, arr);
    }
    return Array.from(map.entries())
      .map(([key, items]) => ({
        key,
        label: `Agent ${key.slice(0, 8)}…`,
        items: items.sort(byStartedDesc),
      }))
      .sort((a, b) => a.label.localeCompare(b.label));
  }

  // Group by day (DESC).
  const map = new Map<string, Execution[]>();
  for (const e of executions) {
    const day = (e.started_at ?? e.completed_at ?? "").slice(0, 10);
    const k = day || "unknown";
    const arr = map.get(k) ?? [];
    arr.push(e);
    map.set(k, arr);
  }
  return Array.from(map.entries())
    .map(([key, items]) => ({
      key,
      label: key === "unknown" ? "Unknown date" : formatDayLabel(key),
      items: items.sort(byStartedDesc),
    }))
    .sort((a, b) => (a.key < b.key ? 1 : -1));
}

function byStartedDesc(a: Execution, b: Execution): number {
  const aT = a.started_at ?? "";
  const bT = b.started_at ?? "";
  return aT < bT ? 1 : aT > bT ? -1 : 0;
}

function formatDayLabel(iso: string): string {
  // iso is "YYYY-MM-DD"
  const d = new Date(`${iso}T00:00:00`);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleDateString(undefined, {
    weekday: "short",
    year: "numeric",
    month: "short",
    day: "numeric",
  });
}
