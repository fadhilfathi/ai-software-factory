"use client";

import { use, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { PageHeader } from "@/components/layout/PageHeader";
import { useAgent, useDeleteAgent } from "@/lib/hooks";
import { timeAgo } from "@/lib/utils";
import { Skeleton } from "@/components/ui/Skeleton";
import { ErrorBlock } from "@/components/ui/ErrorBlock";
import { Badge } from "@/components/ui/Badge";
import { ConfirmDialog } from "@/components/shared/ConfirmDialog";
import type { AgentStatus_ } from "@/lib/types";

const STATUS_BADGE: Record<AgentStatus_, { color: "emerald" | "blue" | "yellow" | "red" | "gray"; label: string }> = {
  idle: { color: "emerald", label: "Idle" },
  working: { color: "blue", label: "Working" },
  spawning: { color: "yellow", label: "Spawning" },
  failed: { color: "red", label: "Failed" },
  completed: { color: "gray", label: "Completed" },
};

import { MetricCard } from "@/components/shared/MetricCard";
import { ActivityTimeline } from "@/components/shared/ActivityTimeline";
import { ProgressBar } from "@/components/ui/ProgressBar";

const STATUS_BADGE: Record<AgentStatus_, { color: "emerald" | "blue" | "yellow" | "red" | "gray"; label: string }> = {
  idle: { color: "emerald", label: "Idle" },
  working: { color: "blue", label: "Working" },
  spawning: { color: "yellow", label: "Spawning" },
  failed: { color: "red", label: "Failed" },
  completed: { color: "gray", label: "Completed" },
};

function AgentStatusBadge({ status }: { status: AgentStatus_ }) {
  const cfg = STATUS_BADGE[status];
  return <Badge color={cfg.color}>{cfg.label}</Badge>;
}

export default function AgentDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const router = useRouter();
  const { data: agent, isLoading, isError } = useAgent(id);
  const deleteAgent = useDeleteAgent();
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);

  // Mocked activity for wireframe fidelity
  const mockActivity = [
    { id: "1", text: "Successfully completed task: auth.ts JWT validation", type: "success" as const, timestamp: new Date(Date.now() - 1000 * 60 * 5).toISOString() },
    { id: "2", text: "Started working on: register.ts endpoint implementation", type: "info" as const, timestamp: new Date(Date.now() - 1000 * 60 * 15).toISOString() },
    { id: "3", text: "Task assigned by PM Agent: TASK-105", type: "info" as const, timestamp: new Date(Date.now() - 1000 * 60 * 45).toISOString() },
    { id: "4", text: "Agent initialized and capabilities registered", type: "success" as const, timestamp: new Date(Date.now() - 1000 * 60 * 120).toISOString() },
  ];

  const handleDelete = async () => {
    await deleteAgent.mutateAsync(id);
    router.push("/agents");
  };

  if (isLoading) {
    return (
      <div className="space-y-6">
        <PageHeader title="Loading..." />
        <div className="grid gap-6 lg:grid-cols-4">
          {[1, 2, 3, 4].map(i => <Skeleton key={i} className="h-24 rounded-xl" />)}
        </div>
        <Skeleton className="h-96 w-full rounded-xl" />
      </div>
    );
  }

  if (isError || !agent) {
    return (
      <div>
        <PageHeader title="Agent Not Found" />
        <ErrorBlock.Page
          message="Could not load this agent."
          backHref="/agents"
        />
      </div>
    );
  }

  return (
    <div className="space-y-8">
      <PageHeader
        title={agent.name}
        subtitle={
          <span className="flex items-center gap-3">
            <span className="text-gray-500 font-mono text-xs uppercase">{agent.id.slice(0, 8)}</span>
            <span className="h-4 w-px bg-gray-800" />
            <AgentStatusBadge status={agent.status} />
          </span>
        }
        actions={
          <div className="flex items-center gap-2">
            <Link
              href={`/agents/${id}/edit`}
              className="rounded-lg border border-gray-800 px-3 py-2 text-sm text-gray-400 hover:bg-gray-800 hover:text-gray-200 transition-colors"
            >
              Configure
            </Link>
            <button
              onClick={() => setShowDeleteDialog(true)}
              className="rounded-lg border border-red-900/50 px-3 py-2 text-sm text-red-500 hover:bg-red-950/30 transition-colors"
              type="button"
            >
              Decommission
            </button>
          </div>
        }
      />

      {/* Performance Stats Row */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <MetricCard label="Tasks Completed" value={agent.tasks_completed ?? 0} trend="+3 today" trendUp />
        <MetricCard label="Uptime (hrs)" value={agent.uptime ?? 0} trend="99.9% health" trendNeutral />
        <MetricCard label="Success Rate" value="96%" trend="↑ 2%" trendUp />
        <MetricCard label="Avg Task Cost" value="$0.38" trend="↓ $0.04" trendUp />
      </div>

      <div className="grid gap-8 lg:grid-cols-3">
        {/* Left Column: Details & Capabilities */}
        <div className="space-y-8">
          <section className="rounded-2xl border border-gray-800 bg-gray-950/30 p-6">
            <h3 className="text-xs font-bold uppercase tracking-widest text-gray-500 mb-6">Agent DNA</h3>
            <dl className="space-y-4 text-sm">
              <div className="flex justify-between border-b border-gray-900 pb-2">
                <dt className="text-gray-500">System Role</dt>
                <dd className="text-gray-200 font-bold capitalize">{agent.role}</dd>
              </div>
              <div className="flex justify-between border-b border-gray-900 pb-2">
                <dt className="text-gray-500">Core Architecture</dt>
                <dd className="text-gray-200">{agent.type}</dd>
              </div>
              <div className="flex justify-between border-b border-gray-900 pb-2">
                <dt className="text-gray-500">Model Engine</dt>
                <dd className="text-emerald-400 font-mono text-xs">{agent.model}</dd>
              </div>
              <div className="flex justify-between border-b border-gray-900 pb-2">
                <dt className="text-gray-500">Compute Provider</dt>
                <dd className="text-gray-200">{agent.provider}</dd>
              </div>
              <div className="flex justify-between">
                <dt className="text-gray-500">Born On</dt>
                <dd className="text-gray-400 text-xs">{new Date(agent.created_at).toLocaleDateString()}</dd>
              </div>
            </dl>
          </section>

          <section className="rounded-2xl border border-gray-800 bg-gray-950/30 p-6">
            <h3 className="text-xs font-bold uppercase tracking-widest text-gray-500 mb-6">Capability Matrix</h3>
            <div className="flex flex-wrap gap-2">
              {agent.capabilities?.map((cap) => (
                <Badge key={cap} color="emerald" variant="outline" className="px-3 py-1">
                  {cap}
                </Badge>
              ))}
            </div>
          </section>
        </div>

        {/* Middle Column: Utilization & Tasks */}
        <div className="space-y-8">
          <section className="rounded-2xl border border-gray-800 bg-gray-950/30 p-6">
            <h3 className="text-xs font-bold uppercase tracking-widest text-gray-500 mb-6">Project Utilization</h3>
            <div className="space-y-6">
              {[
                { name: "Auth Service", pct: 72, color: "blue" as const },
                { name: "Payment Gateway", pct: 45, color: "emerald" as const },
                { name: "ML Pipeline", pct: 18, color: "violet" as const },
              ].map(p => (
                <div key={p.name} className="space-y-2">
                  <div className="flex justify-between text-[11px] font-bold">
                    <span className="text-gray-400">{p.name}</span>
                    <span className="text-gray-300">{p.pct}%</span>
                  </div>
                  <ProgressBar value={p.pct} color={p.color} size="sm" />
                </div>
              ))}
            </div>
          </section>

          <section className="rounded-2xl border border-gray-800 bg-gray-950/30 p-6">
            <h3 className="text-xs font-bold uppercase tracking-widest text-gray-500 mb-4">Current Task</h3>
            {agent.current_task_id ? (
              <div className="rounded-xl bg-emerald-500/5 border border-emerald-500/20 p-4">
                <p className="text-xs font-mono text-emerald-500 mb-2">{agent.current_task_id}</p>
                <p className="text-sm text-gray-200 font-medium">Processing codebase updates for security middleware...</p>
                <div className="mt-4 flex items-center gap-2">
                  <div className="h-1.5 w-1.5 rounded-full bg-emerald-500 animate-pulse" />
                  <span className="text-[10px] text-emerald-400 font-bold uppercase">Executing Step 3/5</span>
                </div>
              </div>
            ) : (
              <div className="flex flex-col items-center justify-center py-8 text-center bg-gray-900/20 rounded-xl border border-dashed border-gray-800">
                <p className="text-xs text-gray-500 italic">Agent is currently idle.</p>
                <button className="mt-3 text-[10px] font-bold uppercase text-emerald-500 hover:underline">
                  Assign Task &rarr;
                </button>
              </div>
            )}
          </section>
        </div>

        {/* Right Column: Activity Timeline */}
        <div className="space-y-8">
          <section className="rounded-2xl border border-gray-800 bg-gray-950/30 p-6">
            <h3 className="text-xs font-bold uppercase tracking-widest text-gray-500 mb-6">Recent Log Events</h3>
            <ActivityTimeline items={mockActivity} />
          </section>
        </div>
      </div>

      <ConfirmDialog
        open={showDeleteDialog}
        onConfirm={handleDelete}
        onCancel={() => setShowDeleteDialog(false)}
        title="Decommission Agent"
        message={`Are you sure you want to decommission "${agent.name}"? This will terminate its active processes.`}
        confirmLabel="Confirm Termination"
        variant="danger"
        loading={deleteAgent.isPending}
      />
    </div>
  );
}
