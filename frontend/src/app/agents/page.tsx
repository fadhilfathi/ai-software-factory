"use client";

import { useState } from "react";
import { PageHeader } from "@/components/layout/PageHeader";
import { useAgents, useProjects } from "@/lib/hooks";
import { timeAgo, cn } from "@/lib/utils";
import { Skeleton } from "@/components/ui/Skeleton";
import { EmptyState } from "@/components/ui/EmptyState";
import { ErrorBlock } from "@/components/ui/ErrorBlock";
import { AgentStatusBadge } from "@/components/shared/StatusBadge";
import { FilterBar } from "@/components/shared/FilterBar";
import type { Agent } from "@/lib/types";

const AGENT_ICONS: Record<string, string> = {
  pm: "🎯",
  developer: "💻",
  reviewer: "🔍",
  devops: "⚙️",
};

const AGENT_NAMES: Record<string, string> = {
  pm: "Project Manager",
  developer: "Developer",
  reviewer: "Reviewer",
  devops: "DevOps",
};

export default function AgentPerformancePage() {
  const { data: agents, isLoading, isError, error } = useAgents();
  const { data: projectsData } = useProjects({ limit: "100" });
  const [selected, setSelected] = useState<Agent | null>(null);
  const [projectFilter, setProjectFilter] = useState("");

  const projects = projectsData?.data ?? [];
  const projectMap = new Map(projects.map((p) => [p.id, p.name]));

  const filteredAgents = agents
    ? projectFilter
      ? agents.filter((a) => a.project_id === projectFilter)
      : agents
    : [];

  const handleSelect = (agent: Agent) => {
    setSelected((prev) => (prev?.id === agent.id ? null : agent));
  };

  const projectOptions = projects.map((p) => ({
    value: p.id,
    label: p.name,
  }));

  return (
    <div>
      <PageHeader title="Agent Performance" />

      {/* Filters */}
      <FilterBar>
        <FilterBar.Select
          value={projectFilter}
          onChange={setProjectFilter}
          options={projectOptions}
          placeholder="All Projects"
        />
      </FilterBar>

      {/* Error State */}
      {isError && (
        <ErrorBlock
          message={(error as Error)?.message ?? "Unknown error"}
          title="Failed to load agents"
          className="mb-6"
        />
      )}

      {/* Loading State */}
      {isLoading && (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {Array.from({ length: 6 }).map((_, i) => (
            <div
              key={i}
              className="animate-pulse rounded-lg border border-gray-800 bg-gray-950 p-4"
            >
              <div className="flex items-center gap-3">
                <div className="h-10 w-10 rounded-full bg-gray-800" />
                <div className="flex-1 space-y-1.5">
                  <div className="h-4 w-24 rounded bg-gray-800" />
                  <div className="h-3 w-16 rounded bg-gray-800" />
                </div>
              </div>
              <div className="mt-3 space-y-2">
                <div className="h-1.5 rounded-full bg-gray-800" />
                <div className="grid grid-cols-3 gap-2">
                  {[1, 2, 3].map((j) => (
                    <div key={j} className="h-8 rounded bg-gray-800" />
                  ))}
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Empty State */}
      {!isLoading && !isError && filteredAgents.length === 0 && (
        <EmptyState
          icon="🤖"
          title="No agents"
          description={
            projectFilter
              ? "No agents assigned to this project."
              : "No agents have been spawned yet."
          }
        />
      )}

      {/* Agent Cards Grid */}
      {!isLoading && !isError && filteredAgents.length > 0 && (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {filteredAgents.map((agent) => {
            const uptimeHours = agent.uptime ? Math.floor(agent.uptime / 3600) : 0;
            const utilization = agent.tasks_completed
              ? Math.min(Math.round((agent.tasks_completed / (agent.tasks_completed + 3)) * 100), 100)
              : 0;

            return (
              <button
                key={agent.id}
                onClick={() => handleSelect(agent)}
                className={cn(
                  "rounded-lg border p-4 text-left transition-all",
                  selected?.id === agent.id
                    ? "border-emerald-500 bg-emerald-500/5"
                    : "border-gray-800 bg-gray-950 hover:border-gray-700",
                )}
                type="button"
              >
                <div className="flex items-center gap-3">
                  <div className="flex h-10 w-10 items-center justify-center rounded-full bg-gray-800 text-lg">
                    {AGENT_ICONS[agent.type] ?? "🤖"}
                  </div>
                  <div className="min-w-0">
                    <p className="font-medium text-gray-200 truncate">
                      {AGENT_NAMES[agent.type] ?? agent.type}
                    </p>
                    <p className="text-xs text-gray-500">@{agent.type}</p>
                  </div>
                  <div className="ml-auto shrink-0">
                    <AgentStatusBadge status={agent.status} />
                  </div>
                </div>

                {/* Utilization bar */}
                <div className="mt-3 flex gap-0.5">
                  {Array.from({ length: 20 }).map((_, i) => (
                    <div
                      key={i}
                      className={cn(
                        "h-1.5 flex-1 rounded-full",
                        i / 20 < utilization / 100 ? "bg-emerald-500" : "bg-gray-800",
                      )}
                    />
                  ))}
                </div>

                <div className="mt-3 grid grid-cols-3 gap-2 text-center">
                  <div>
                    <p className="text-lg font-bold text-gray-200">{agent.tasks_completed ?? 0}</p>
                    <p className="text-[10px] text-gray-500">Tasks</p>
                  </div>
                  <div>
                    <p className="text-lg font-bold text-gray-200">
                      {uptimeHours > 0 ? `${uptimeHours}h` : "<1h"}
                    </p>
                    <p className="text-[10px] text-gray-500">Uptime</p>
                  </div>
                  <div>
                    <p className="text-lg font-bold text-gray-200">{utilization}%</p>
                    <p className="text-[10px] text-gray-500">Util.</p>
                  </div>
                </div>

                {agent.project_id && (
                  <p className="mt-2 text-[10px] text-gray-600 truncate">
                    Project: {projectMap.get(agent.project_id) ?? agent.project_id.slice(0, 8)}
                  </p>
                )}
              </button>
            );
          })}
        </div>
      )}

      {/* Detail Panel */}
      {selected && (
        <div className="mt-6 rounded-lg border border-gray-800 bg-gray-950 p-6">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-lg font-semibold text-gray-200">
              {AGENT_NAMES[selected.type] ?? selected.type} Details
            </h3>
            <AgentStatusBadge status={selected.status} />
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-3">
              <div>
                <p className="text-xs text-gray-500">Agent ID</p>
                <p className="text-sm text-gray-200 font-mono">{selected.id}</p>
              </div>
              {selected.project_id && (
                <div>
                  <p className="text-xs text-gray-500">Project</p>
                  <p className="text-sm text-gray-200">
                    {projectMap.get(selected.project_id) ?? selected.project_id}
                  </p>
                </div>
              )}
              {selected.current_task && (
                <div>
                  <p className="text-xs text-gray-500">Current Task</p>
                  <p className="text-sm text-gray-200">{selected.current_task}</p>
                </div>
              )}
            </div>
            <div className="space-y-3">
              {selected.config && (
                <div>
                  <p className="text-xs text-gray-500">Model</p>
                  <p className="text-sm text-gray-200">
                    {selected.config.model ?? "Default"}
                  </p>
                </div>
              )}
              <div>
                <p className="text-xs text-gray-500">Created</p>
                <p className="text-sm text-gray-200">{timeAgo(selected.created_at)}</p>
              </div>
              <div>
                <p className="text-xs text-gray-500">Last Updated</p>
                <p className="text-sm text-gray-200">{timeAgo(selected.updated_at)}</p>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
