"use client";

import { useState } from "react";
import Link from "next/link";
import { PageHeader } from "@/components/layout/PageHeader";
import { useAgents } from "@/lib/hooks";
import { cn } from "@/lib/utils";
import { Skeleton } from "@/components/ui/Skeleton";
import { EmptyState } from "@/components/ui/EmptyState";
import { ErrorBlock } from "@/components/ui/ErrorBlock";
import { FilterBar, SearchInput } from "@/components/shared/FilterBar";
import { MetricCard } from "@/components/shared/MetricCard";
import { AgentCard } from "@/components/agents/AgentCard";
import type { AgentStatus_ } from "@/lib/types";

const ROLE_TABS = [
  { value: "all", label: "All Agents", icon: "🤖" },
  { value: "pm", label: "PM", color: "#10b981" },
  { value: "architect", label: "Architect", color: "#0ea5e9" },
  { value: "developer", label: "Developer", color: "#6366f1" },
  { value: "reviewer", label: "Review", color: "#f97316" },
  { value: "qa", label: "QA", color: "#ec4899" },
  { value: "devops", label: "DevOps", color: "#8b5cf6" },
];

export default function AgentDashboardPage() {
  const [activeTab, setActiveTab] = useState("all");
  const [statusFilter, setStatusFilter] = useState("");
  const [search, setSearch] = useState("");

  const filters: Record<string, string | undefined> = {};
  if (statusFilter) filters.status = statusFilter;
  if (activeTab !== "all") filters.role = activeTab;
  if (search) filters.search = search;
  filters.limit = "24";

  const { data, isLoading, isError, error } = useAgents(filters);
  const agents = data?.data ?? [];

  // Summary stats (mocked or derived if available)
  const totalTasks = agents.reduce((acc, a) => acc + (a.tasks_completed ?? 0), 0);
  const activeCount = agents.filter(a => a.status === "working").length;

  return (
    <div className="space-y-8">
      <PageHeader
        title="Agent Registry"
        subtitle="Real-time monitoring and management of the factory's AI workforce."
        actions={
          <Link
            href="/agents/new"
            className="inline-flex items-center gap-2 rounded-xl bg-emerald-500 px-5 py-2.5 text-sm font-bold text-white hover:bg-emerald-600 transition-all shadow-lg shadow-emerald-500/20 active:scale-95"
          >
            + Spawn New Agent
          </Link>
        }
      />

      {/* Global Metrics */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <MetricCard 
          label="Total Agents" 
          value={agents.length} 
          trend="+2 this week" 
          trendUp 
        />
        <MetricCard 
          label="Active Now" 
          value={activeCount} 
          trend="85% utilization" 
          trendNeutral 
        />
        <MetricCard 
          label="Tasks Completed" 
          value={totalTasks} 
          trend="↑ 12% vs last week" 
          trendUp 
        />
        <MetricCard 
          label="Avg Cost/Task" 
          value="$0.42" 
          trend="↓ $0.05 saved" 
          trendUp 
        />
      </div>

      {/* Agent Role Tabs */}
      <div className="flex items-center gap-1 border-b border-gray-800 pb-px">
        {ROLE_TABS.map((tab) => (
          <button
            key={tab.value}
            onClick={() => setActiveTab(tab.value)}
            className={cn(
              "flex items-center gap-2 px-4 py-3 text-sm font-bold transition-all border-b-2",
              activeTab === tab.value
                ? "border-emerald-500 text-emerald-400 bg-emerald-500/5"
                : "border-transparent text-gray-500 hover:text-gray-300 hover:bg-gray-900/40"
            )}
          >
            {tab.color && (
              <div 
                className="h-2 w-2 rounded-full" 
                style={{ backgroundColor: tab.color }} 
              />
            )}
            {tab.label}
          </button>
        ))}
      </div>

      {/* Search & Filters */}
      <FilterBar>
        <SearchInput
          value={search}
          onChange={setSearch}
          placeholder="Search agents by name or model..."
          className="flex-1"
        />
        <FilterBar.Select
          value={statusFilter}
          onChange={setStatusFilter}
          options={[
            { value: "idle", label: "Idle Only" },
            { value: "working", label: "Working Only" },
            { value: "failed", label: "Failed/Error" },
          ]}
          placeholder="All Statuses"
        />
      </FilterBar>

      {isError && (
        <ErrorBlock
          message={(error as Error)?.message ?? "Unknown error"}
          title="Failed to load agent registry"
        />
      )}

      {/* Agents Grid */}
      {isLoading ? (
        <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
          {Array.from({ length: 8 }).map((_, i) => (
            <Skeleton key={i} className="h-48 w-full rounded-xl" />
          ))}
        </div>
      ) : agents.length === 0 ? (
        <EmptyState
          icon="🤖"
          title="No agents found in the registry"
          description="Try adjusting your filters or spawn a new agent to expand the workforce."
        />
      ) : (
        <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
          {agents.map((agent) => (
            <AgentCard key={agent.id} agent={agent} />
          ))}
        </div>
      )}

      {/* Detailed Performance Link */}
      <div className="flex justify-center pt-8">
        <Link
          href="/dashboard"
          className="text-xs font-bold uppercase tracking-widest text-gray-500 hover:text-emerald-500 transition-colors"
        >
          View Factory Performance Report &rarr;
        </Link>
      </div>
    </div>
  );
}
