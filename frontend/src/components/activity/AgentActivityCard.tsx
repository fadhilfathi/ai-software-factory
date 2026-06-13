"use client";

/**
 * Per-agent activity card.
 *
 * TASK-410 — Agent Activity Dashboard.
 * One card per agent. Shows:
 *   - agent name + status badge
 *   - active tasks count
 *   - completed tasks in last 7 days
 *   - completed tasks in last 30 days
 *   - link to the agent detail page
 *
 * The data is pre-aggregated by the parent page (single fetch + client
 * aggregation = no N+1).
 */

import Link from "next/link";

import { Badge } from "@/components/ui/Badge";
import { cn } from "@/lib/utils";
import { AgentStatus } from "@/lib/types";

export type AgentActivityStats = {
  agentId: string;
  agentName: string;
  status: AgentStatus;
  totalTasks: number;
  activeTasks: number;
  completedLast7d: number;
  completedLast30d: number;
  recentExecutions: number;
};

const STATUS_TONE: Record<AgentStatus, { bg: string; label: string }> = {
  initializing: { bg: "slate", label: "Initializing" },
  idle: { bg: "emerald", label: "Idle" },
  busy: { bg: "blue", label: "Busy" },
  paused: { bg: "amber", label: "Paused" },
  error: { bg: "rose", label: "Error" },
  retired: { bg: "gray", label: "Retired" },
};

export function AgentActivityCard({
  stats,
  empty = false,
}: {
  stats: AgentActivityStats;
  empty?: boolean;
}) {
  const tone = STATUS_TONE[stats.status] ?? STATUS_TONE.idle;

  return (
    <article
      className={cn(
        "rounded-lg border bg-gray-900/30 p-4 transition",
        empty
          ? "border-dashed border-gray-800 opacity-60"
          : "border-gray-800 hover:border-gray-700",
      )}
    >
      <header className="mb-3 flex items-start justify-between gap-2">
        <div className="min-w-0 flex-1">
          <h3 className="truncate text-sm font-semibold text-gray-100">
            {stats.agentName}
          </h3>
          <p className="truncate font-mono text-[10px] text-gray-500">
            {stats.agentId}
          </p>
        </div>
        <Badge color={tone.bg as never} variant="outline">
          {tone.label}
        </Badge>
      </header>

      <dl className="grid grid-cols-2 gap-3 text-sm">
        <Stat
          label="Active tasks"
          value={stats.activeTasks}
          accent={stats.activeTasks > 0 ? "amber" : "slate"}
        />
        <Stat label="Total tasks" value={stats.totalTasks} accent="slate" />
        <Stat
          label="Completed (7d)"
          value={stats.completedLast7d}
          accent="emerald"
        />
        <Stat
          label="Completed (30d)"
          value={stats.completedLast30d}
          accent="emerald"
        />
        <Stat
          label="Executions (window)"
          value={stats.recentExecutions}
          accent="violet"
        />
        <Stat
          label="—"
          value={empty ? 0 : 0}
          accent="slate"
          hide={empty}
        />
      </dl>

      <footer className="mt-3 flex items-center justify-between border-t border-gray-800 pt-3">
        <Link
          href={`/agents/${stats.agentId}`}
          className="text-xs font-medium text-emerald-400 hover:text-emerald-300"
        >
          View agent →
        </Link>
        {stats.activeTasks > 0 ? (
          <span className="text-[10px] text-amber-400">
            {stats.activeTasks} in flight
          </span>
        ) : null}
      </footer>
    </article>
  );
}

function Stat({
  label,
  value,
  accent,
  hide = false,
}: {
  label: string;
  value: number;
  accent: "amber" | "emerald" | "slate" | "violet";
  hide?: boolean;
}) {
  if (hide) return null;
  const colorClass = {
    amber: "text-amber-300",
    emerald: "text-emerald-300",
    slate: "text-gray-300",
    violet: "text-violet-300",
  }[accent];
  return (
    <div>
      <dt className="text-[10px] font-semibold uppercase tracking-wide text-gray-500">
        {label}
      </dt>
      <dd className={cn("mt-0.5 text-xl font-semibold tabular-nums", colorClass)}>
        {value}
      </dd>
    </div>
  );
}
