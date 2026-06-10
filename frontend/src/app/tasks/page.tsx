"use client";

import { useState, useCallback } from "react";
import { PageHeader } from "@/components/layout/PageHeader";
import { useProjects, useTasks, useUpdateTask } from "@/lib/hooks";
import { useKanbanDrag } from "@/hooks/useKanbanDrag";
import { cn } from "@/lib/utils";
import { FilterBar } from "@/components/shared/FilterBar";
import { PriorityBadge } from "@/components/shared/PriorityBadge";
import { TaskStatusBadge } from "@/components/shared/StatusBadge";
import { Skeleton } from "@/components/ui/Skeleton";
import { EmptyState } from "@/components/ui/EmptyState";
import type { Task, TaskStatus } from "@/lib/types";

const COLUMNS: { key: TaskStatus; label: string }[] = [
  { key: "backlog", label: "Backlog" },
  { key: "todo", label: "Todo" },
  { key: "in_progress", label: "In Progress" },
  { key: "review", label: "Review" },
  { key: "done", label: "Done" },
];

const COLUMN_COLORS: Record<TaskStatus, string> = {
  backlog: "border-gray-800",
  todo: "border-blue-500/30",
  in_progress: "border-emerald-500/30",
  review: "border-violet-500/30",
  done: "border-gray-600/30",
};

const PRIORITY_OPTIONS = [
  { value: "critical", label: "Critical" },
  { value: "high", label: "High" },
  { value: "medium", label: "Medium" },
  { value: "low", label: "Low" },
];

export default function TaskBoardPage() {
  const { data: projectsData } = useProjects({ limit: "100" });
  const [selectedProject, setSelectedProject] = useState<string>("");
  const [agentFilter, setAgentFilter] = useState("");
  const [priorityFilter, setPriorityFilter] = useState("");

  const projectId = selectedProject || (projectsData?.data?.[0]?.id ?? "");
  const { data: tasks, isLoading } = useTasks(projectId);
  const updateTask = useUpdateTask();
  const { activeDrag, startDrag, endDrag } = useKanbanDrag();

  // Group tasks by status
  const tasksByColumn = tasks?.reduce(
    (acc, task) => {
      if (!acc[task.status]) acc[task.status] = [];
      if (priorityFilter && task.priority !== priorityFilter) return acc;
      if (agentFilter && task.assignee_agent_id !== agentFilter) return acc;
      acc[task.status].push(task);
      return acc;
    },
    {} as Record<TaskStatus, Task[]>,
  ) ?? ({} as Record<TaskStatus, Task[]>);

  for (const col of COLUMNS) {
    if (!tasksByColumn[col.key]) tasksByColumn[col.key] = [];
  }

  const handleMoveTask = useCallback(
    (taskId: string, newStatus: TaskStatus) => {
      updateTask.mutate({ id: taskId, status: newStatus });
    },
    [updateTask],
  );

  const uniqueAgents = tasks
    ? [...new Set(tasks.map((t) => t.assignee_agent_id).filter(Boolean))]
    : [];

  const projectOptions = (projectsData?.data ?? []).map((p) => ({
    value: p.id,
    label: p.name,
  }));

  const agentOptions = uniqueAgents.map((a) => ({
    value: a!,
    label: `@${a?.slice(0, 8)}`,
  }));

  return (
    <div>
      <PageHeader title="Task Board" />

      {/* Filter Bar */}
      <FilterBar>
        <FilterBar.Select
          value={selectedProject}
          onChange={setSelectedProject}
          options={projectOptions}
          placeholder="All Projects"
        />
        <FilterBar.Select
          value={agentFilter}
          onChange={setAgentFilter}
          options={agentOptions}
          placeholder="All Agents"
        />
        <FilterBar.Select
          value={priorityFilter}
          onChange={setPriorityFilter}
          options={PRIORITY_OPTIONS}
          placeholder="All Priorities"
        />
      </FilterBar>

      {/* Kanban Board */}
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5">
        {COLUMNS.map((column) => (
          <div
            key={column.key}
            className={cn(
              "rounded-lg border bg-gray-950/80",
              COLUMN_COLORS[column.key],
            )}
            onDragOver={(e) => {
              e.preventDefault();
              e.currentTarget.style.borderColor = "rgba(52, 211, 153, 0.5)";
            }}
            onDragLeave={(e) => {
              e.currentTarget.style.borderColor = "";
            }}
            onDrop={(e) => {
              e.preventDefault();
              e.currentTarget.style.borderColor = "";
              const taskId = e.dataTransfer.getData("text/task-id");
              if (taskId) {
                handleMoveTask(taskId, column.key);
              }
              endDrag();
            }}
          >
            {/* Column Header */}
            <div className="flex items-center justify-between border-b border-gray-800 px-4 py-3">
              <h3 className="text-sm font-semibold text-gray-300 uppercase tracking-wider">
                {column.label}
              </h3>
              <span className="flex h-5 w-5 items-center justify-center rounded-full bg-gray-800 text-[10px] font-bold text-gray-400">
                {(tasksByColumn[column.key] as Task[])?.length ?? 0}
              </span>
            </div>

            {/* Task Cards */}
            <div className="space-y-2 p-3 min-h-[120px]">
              {isLoading ? (
                <div className="space-y-2">
                  {[1, 2].map((i) => (
                    <Skeleton key={i} className="h-20 w-full" />
                  ))}
                </div>
              ) : (tasksByColumn[column.key] ?? []).length === 0 ? (
                <div className="flex items-center justify-center py-8">
                  <p className="text-xs text-gray-600">No tasks</p>
                </div>
              ) : (
                (tasksByColumn[column.key] ?? []).map((task) => (
                  <div
                    key={task.id}
                    draggable
                    onDragStart={(e) => {
                      e.dataTransfer.setData("text/task-id", task.id);
                      startDrag({ id: task.id, column: column.key });
                    }}
                    className={cn(
                      "rounded-md border border-gray-800 bg-gray-900 p-3 hover:border-gray-700 transition-colors cursor-grab active:cursor-grabbing select-none",
                      activeDrag?.id === task.id && "opacity-50 border-emerald-500",
                    )}
                  >
                    <div className="flex items-center justify-between gap-2">
                      <span className="text-xs font-mono text-gray-500">
                        {task.id.slice(0, 8)}
                      </span>
                      <PriorityBadge priority={task.priority} />
                    </div>
                    <p className="mt-1 text-sm text-gray-200 line-clamp-2">{task.title}</p>
                    {task.assignee_agent_id && (
                      <span className="mt-2 inline-flex items-center gap-1 rounded bg-gray-800 px-2 py-0.5 text-[10px] text-gray-400">
                        @{task.assignee_agent_id.slice(0, 8)}
                      </span>
                    )}
                  </div>
                ))
              )}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
