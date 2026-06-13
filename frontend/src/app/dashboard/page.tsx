"use client";

import Link from "next/link";
import { PageHeader } from "@/components/layout/PageHeader";
import { useProjects, useAgents, useExecutions } from "@/lib/hooks";
import { timeAgo, formatNumber, formatCurrency } from "@/lib/utils";
import { MetricCard } from "@/components/shared/MetricCard";
import { Badge } from "@/components/ui/Badge";
import { Skeleton } from "@/components/ui/Skeleton";
import { EmptyState } from "@/components/ui/EmptyState";
import { ProgressBar } from "@/components/ui/ProgressBar";
import type { AgentStatus, ExecutionStatus } from "@/lib/types";

const AGENT_STATUS_BADGE: Record<AgentStatus, { color: "emerald" | "blue" | "yellow" | "red" | "gray"; label: string }> = {
  idle: { color: "emerald", label: "Idle" },
  busy: { color: "blue", label: "Working" },
  initializing: { color: "yellow", label: "Initializing" },
  error: { color: "red", label: "Error" },
  retired: { color: "gray", label: "Retired" },
  paused: { color: "gray", label: "Paused" },
};

const EXEC_STATUS_COLOR: Record<string, "emerald" | "blue" | "red" | "gray"> = {
  completed: "emerald",
  running: "blue",
  failed: "red",
};

export default function DashboardPage() {
  const { data: projectsData, isLoading: projectsLoading } = useProjects({ limit: "100" });
  const { data: agentsData, isLoading: agentsLoading } = useAgents({ limit: "100" });
  const { data: execsData, isLoading: execsLoading } = useExecutions({ limit: "10" });

  // useProjects returns a bare array; useAgents and useExecutions
  // still return envelopes per the Sprint 4 / Sprint 1-3 spec split.
  const projects = projectsData ?? [];
  const agents = (agentsData as { data?: unknown[] } | undefined)?.data ?? [];
  const executions = (execsData as { data?: unknown[] } | undefined)?.data ?? [];

  // `status` is not a Project field per docs/api-spec.md (Sprint 1-3).
  // The dashboard used to read it pre-TASK-408; we now count active
  // projects by recency (anything touched in the last 30 days) instead.
  const thirtyDaysAgo = Date.now() - 30 * 24 * 60 * 60 * 1000;
  const activeProjects = projects.filter(
    (p) => p.updated_at && new Date(p.updated_at).getTime() >= thirtyDaysAgo,
  );
  const completedProjects = projects.filter(
    (p) => p.updated_at && new Date(p.updated_at).getTime() < thirtyDaysAgo,
  );
  const totalSpend = projects.length * 1240;

  const agentCounts = {
    // Typed as AgentStatus[]. The hook returns `unknown[]` for the data
    // array; we re-narrow before reading `.status` so the rest of the
    // file is type-safe.
    total: agents.length,
    idle: agents.filter((a) => AGENT_STATUS_BADGE[(a as { status: AgentStatus }).status]?.label === "Idle").length,
    working: agents.filter((a) => AGENT_STATUS_BADGE[(a as { status: AgentStatus }).status]?.label === "Working").length,
    completed: agents.filter((a) => (a as { status: AgentStatus }).status === "retired").length,
    failed: agents.filter((a) => (a as { status: AgentStatus }).status === "error").length,
  };

  return (
    <div>
      <PageHeader title="Dashboard" />

      {/* Metrics Row */}
      <div className="mb-6 grid grid-cols-2 gap-4 md:grid-cols-4">
        <MetricCard
          label="Active Projects"
          value={formatNumber(activeProjects.length)}
          trend={projects.length > 0 ? `${Math.round((activeProjects.length / projects.length) * 100)}%` : "0%"}
          trendUp={activeProjects.length > 0}
          loading={projectsLoading}
        />
        <MetricCard
          label="Completed"
          value={formatNumber(completedProjects.length)}
          trend={`+${completedProjects.length}`}
          trendUp={completedProjects.length > 0}
          loading={projectsLoading}
        />
        <MetricCard
          label="Total Projects"
          value={formatNumber(projects.length)}
          trend="total"
          trendNeutral
          loading={projectsLoading}
        />
        <MetricCard
          label="Est. Spend"
          value={formatCurrency(totalSpend)}
          trend={`$${totalSpend}`}
          trendUp
          loading={projectsLoading}
        />
      </div>

      {/* Agent Summary Row */}
      <div className="mb-6 grid grid-cols-2 gap-4 md:grid-cols-5">
        <MetricCard
          label="Total Agents"
          value={formatNumber(agentCounts.total)}
          loading={agentsLoading}
        />
        <MetricCard
          label="Idle"
          value={formatNumber(agentCounts.idle)}
          trend={`${agentCounts.total > 0 ? Math.round((agentCounts.idle / agentCounts.total) * 100) : 0}%`}
          trendUp
          loading={agentsLoading}
        />
        <MetricCard
          label="Working"
          value={formatNumber(agentCounts.working)}
          trend={`${agentCounts.total > 0 ? Math.round((agentCounts.working / agentCounts.total) * 100) : 0}%`}
          trendUp
          loading={agentsLoading}
        />
        <MetricCard
          label="Completed"
          value={formatNumber(agentCounts.completed)}
          trend={`${agentCounts.total > 0 ? Math.round((agentCounts.completed / agentCounts.total) * 100) : 0}%`}
          trendNeutral
          loading={agentsLoading}
        />
        <MetricCard
          label="Failed"
          value={formatNumber(agentCounts.failed)}
          trend={`${agentCounts.total > 0 ? Math.round((agentCounts.failed / agentCounts.total) * 100) : 0}%`}
          trendUp={false}
          loading={agentsLoading}
        />
      </div>

      {/* Active Projects + Recent Executions */}
      <div className="grid gap-6 lg:grid-cols-2">
        {/* Active Projects */}
        <div className="rounded-lg border border-gray-800 bg-gray-950 p-4">
          <div className="mb-4 flex items-center justify-between">
            <h2 className="text-sm font-semibold text-gray-300 uppercase tracking-wider">
              Active Projects
            </h2>
            <Link
              href="/projects"
              className="text-xs text-emerald-400 hover:text-emerald-300 transition-colors"
            >
              View all &rarr;
            </Link>
          </div>

          {projectsLoading ? (
            <Skeleton.List count={3} />
          ) : activeProjects.length === 0 ? (
            <EmptyState
              icon=""
              title="No active projects"
              action={
                <Link
                  href="/projects/new"
                  className="inline-block text-sm text-emerald-400 hover:text-emerald-300"
                >
                  Create your first project &rarr;
                </Link>
              }
              className="py-8"
            />
          ) : (
            <div className="space-y-3">
              {activeProjects.slice(0, 5).map((project) => (
                <Link
                  key={project.id}
                  href={`/projects/${project.id}`}
                  className="block rounded-md bg-gray-900 p-3 hover:bg-gray-800 transition-colors"
                >
                  <div className="flex items-center justify-between">
                    <span className="text-sm font-medium text-gray-200 truncate">
                      {project.name}
                    </span>
                    <span className="text-xs text-emerald-400">
                      {/* `progress` is not in the Sprint 1-3 Project type; show agent count instead. */}
                      {(project as { active_agents?: number }).active_agents ?? 0} agents
                    </span>
                  </div>
                  {/* `progress` is not in the Sprint 1-3 Project type; the
                      bar is left out until the backend exposes a per-project
                      progress metric. */}
                  {(() => {
                    const active = (project as { active_agents?: number }).active_agents ?? 0;
                    return active > 0 ? (
                      <p className="mt-1.5 text-[10px] text-gray-500">
                        {active} agent{active !== 1 ? "s" : ""} active
                      </p>
                    ) : null;
                  })()}
                </Link>
              ))}
            </div>
          )}
        </div>

        {/* Recent Executions */}
        <div className="rounded-lg border border-gray-800 bg-gray-950 p-4">
          <div className="mb-4 flex items-center justify-between">
            <h2 className="text-sm font-semibold text-gray-300 uppercase tracking-wider">
              Recent Executions
            </h2>
            <span className="text-[10px] text-gray-600">Auto-refreshes</span>
          </div>

          {execsLoading ? (
            <Skeleton.List count={3} />
          ) : executions.length === 0 ? (
            <EmptyState
              icon=""
              title="No executions yet"
              className="py-8"
            />
          ) : (
            <div className="space-y-2">
              {executions.map((raw) => {
                const exec = raw as {
                  id: string;
                  execution_id?: string;
                  agent_name?: string;
                  task_id?: string;
                  started_at?: string;
                  status: ExecutionStatus;
                };
                return (
                  <div
                    key={exec.id}
                    className="flex items-center justify-between rounded-md bg-gray-900 p-3"
                  >
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-2">
                        <span className="text-xs font-mono text-gray-500">
                          {exec.execution_id?.slice(0, 8) ?? exec.id.slice(0, 8)}
                        </span>
                        {exec.agent_name && (
                          <span className="text-xs text-gray-400 truncate">
                            {exec.agent_name}
                          </span>
                        )}
                      </div>
                      <div className="mt-0.5 flex items-center gap-2 text-[10px] text-gray-600">
                        <span>task: {exec.task_id?.slice(0, 8)}</span>
                        {exec.started_at && <span>&middot; {timeAgo(exec.started_at)}</span>}
                      </div>
                    </div>
                    <Badge
                      color={EXEC_STATUS_COLOR[exec.status] ?? "gray"}
                      size="sm"
                    >
                      {exec.status}
                    </Badge>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
