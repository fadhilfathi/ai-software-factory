"use client";

import { use, useState } from "react";
import Link from "next/link";
import { PageHeader } from "@/components/layout/PageHeader";
import { useTask, useAgents, useAssignTask, useTaskExecutions, useTaskDeliverables } from "@/lib/hooks";
import { timeAgo } from "@/lib/utils";
import { Skeleton } from "@/components/ui/Skeleton";
import { EmptyState } from "@/components/ui/EmptyState";
import { ErrorBlock } from "@/components/ui/ErrorBlock";
import { Badge } from "@/components/ui/Badge";
import { SpinnerButton } from "@/components/ui/Spinner";
import { TaskStatusBadge } from "@/components/shared/StatusBadge";
import { PriorityBadge } from "@/components/shared/PriorityBadge";

export default function TaskDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const { data: task, isLoading: taskLoading, isError: taskError } = useTask(id);
  const { data: agentsData } = useAgents({ status: "idle", limit: "100" });
  const assignTask = useAssignTask();
  const { data: execsData, isLoading: execsLoading } = useTaskExecutions(id);
  const { data: deliverables, isLoading: delsLoading } = useTaskDeliverables(id);

  const executions = execsData ?? [];
  const idleAgents = agentsData?.data ?? [];
  const isAssigned = !!task?.assignee_id;
  const [selectedAgentId, setSelectedAgentId] = useState("");

  const handleAssign = () => {
    if (!selectedAgentId) return;
    assignTask.mutate({ taskId: id, agent_id: selectedAgentId });
  };

  if (taskLoading) {
    return (
      <div>
        <PageHeader title="Loading..." />
        <div className="space-y-4">
          <Skeleton className="h-32 w-full" />
          <Skeleton className="h-48 w-full" />
          <Skeleton className="h-48 w-full" />
        </div>
      </div>
    );
  }

  if (taskError || !task) {
    return (
      <div>
        <PageHeader title="Task Not Found" />
        <ErrorBlock.Page
          message="Could not load this task."
          backHref="/tasks"
        />
      </div>
    );
  }

  return (
    <div>
      <PageHeader
        title={task.title}
        subtitle={
          <span className="flex items-center gap-2">
            <TaskStatusBadge status={task.status} />
            <span className="text-gray-600">|</span>
            <PriorityBadge priority={task.priority} />
            <span className="text-gray-500">ID: {task.id.slice(0, 8)}</span>
          </span>
        }
        actions={
          <Link
            href={`/projects/${task.project_id}`}
            className="text-sm text-gray-400 hover:text-gray-200 transition-colors"
          >
            &larr; Back to Project
          </Link>
        }
      />

      {task.description && (
        <p className="mb-6 text-sm text-gray-400">{task.description}</p>
      )}

      <div className="grid gap-6 lg:grid-cols-2">
        {/* Assignment Section */}
        <div className="space-y-4">
          <div className="rounded-lg border border-gray-800 bg-gray-950 p-4">
            <h3 className="mb-3 text-sm font-semibold text-gray-300 uppercase tracking-wider">
              Assignment
            </h3>

            {isAssigned ? (
              <div className="space-y-2">
                <p className="text-sm text-gray-400">
                  Assigned to agent: <span className="text-gray-200 font-medium">{task.assignee_id}</span>
                </p>
                <Link
                  href={`/agents/${task.assignee_id}`}
                  className="inline-block text-xs text-emerald-400 hover:text-emerald-300 transition-colors"
                >
                  View Agent &rarr;
                </Link>
              </div>
            ) : (
              <div className="space-y-3">
                <p className="text-sm text-gray-500">
                  {idleAgents.length === 0
                    ? "No idle agents available for assignment."
                    : "Select an idle agent to assign this task."}
                </p>
                {idleAgents.length > 0 && (
                  <>
                    <select
                      value={selectedAgentId}
                      onChange={(e) => setSelectedAgentId(e.target.value)}
                      className="w-full rounded-lg border border-gray-800 bg-gray-900 px-4 py-2 text-sm text-gray-200 focus:outline-none focus:ring-2 focus:ring-emerald-500/50"
                    >
                      <option value="">Select an agent...</option>
                      {idleAgents.map((agent) => (
                        <option key={agent.id} value={agent.id}>
                          {agent.name} ({agent.role})
                        </option>
                      ))}
                    </select>
                    <SpinnerButton
                      onClick={handleAssign}
                      loading={assignTask.isPending}
                      loadingText="Assigning..."
                      disabled={!selectedAgentId || assignTask.isPending}
                    >
                      Assign Task
                    </SpinnerButton>
                  </>
                )}
              </div>
            )}
          </div>

          {/* Execution History */}
          <div className="rounded-lg border border-gray-800 bg-gray-950 p-4">
            <h3 className="mb-3 text-sm font-semibold text-gray-300 uppercase tracking-wider">
              Execution History
            </h3>

            {execsLoading ? (
              <Skeleton.List count={2} />
            ) : !executions || executions.length === 0 ? (
              <EmptyState
                icon=""
                title="No executions yet"
                className="py-6"
              />
            ) : (
              <div className="space-y-2">
                {executions.map((exec) => (
                  <div
                    key={exec.id}
                    className="rounded-lg border border-gray-800 bg-gray-900 p-3"
                  >
                    <div className="flex items-center justify-between">
                      <span className="text-xs font-mono text-gray-500">
                        {(exec as { execution_id?: string }).execution_id?.slice(0, 8) ?? exec.id.slice(0, 8)}
                      </span>
                      <Badge color={exec.status === "succeeded" ? "emerald" : exec.status === "running" ? "blue" : "gray"}>
                        {exec.status}
                      </Badge>
                    </div>
                    <div className="mt-1.5 flex items-center gap-2 text-xs text-gray-500">
                      {(exec as { agent_name?: string }).agent_name && <span>Agent: {(exec as { agent_name?: string }).agent_name}</span>}
                      {(exec as { started_at?: string }).started_at && <span>&middot; {timeAgo((exec as { started_at?: string }).started_at)}</span>}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>

        {/* Deliverables Section */}
        <div className="space-y-4">
          <div className="rounded-lg border border-gray-800 bg-gray-950 p-4">
            <h3 className="mb-3 text-sm font-semibold text-gray-300 uppercase tracking-wider">
              Deliverables
            </h3>

            {delsLoading ? (
              <Skeleton.List count={2} />
            ) : !deliverables || deliverables.length === 0 ? (
              <EmptyState
                icon=""
                title="No deliverables yet"
                className="py-6"
              />
            ) : (
              <div className="space-y-2">
                {deliverables.map((del) => (
                  <div
                    key={del.id}
                    className="rounded-lg border border-gray-800 bg-gray-900 p-3"
                  >
                    <div className="flex items-center justify-between">
                      <span className="text-sm font-medium text-gray-200">
                        {del.title}
                      </span>
                      <Badge color="gray" variant="outline">
                        v{(del as { version?: number }).version ?? 1}
                      </Badge>
                    </div>
                    <div className="mt-1.5 flex items-center justify-between text-xs text-gray-500">
                      <span>{timeAgo(del.created_at)}</span>
                      <Link
                        href={`/deliverables/${del.id}`}
                        className="text-emerald-400 hover:text-emerald-300 transition-colors"
                      >
                        View &rarr;
                      </Link>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
