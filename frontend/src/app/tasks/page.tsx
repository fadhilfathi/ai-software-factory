"use client";

import { useState } from "react";
import { PageHeader } from "@/components/layout/PageHeader";
import { useProjects, useTasks, useUpdateTaskStatus, useCreateTask } from "@/lib/hooks";
import { FilterBar } from "@/components/shared/FilterBar";
import { KanbanBoard } from "@/components/kanban/KanbanBoard";
import type { TaskStatus, Project } from "@/lib/types";

const PRIORITY_OPTIONS = [
  { value: "critical", label: "Critical" },
  { value: "high", label: "High" },
  { value: "medium", label: "Medium" },
  { value: "low", label: "Low" },
];

export default function TaskBoardPage() {
  const { data: projectsData } = useProjects({ limit: "100" });
  const [selectedProjectId, setSelectedProjectId] = useState<string>("");
  const [agentFilter, setAgentFilter] = useState("");
  const [priorityFilter, setPriorityFilter] = useState("");

  // useProjects returns a bare array; legacy callers used to read
  // `.data` from an envelope. Unwrap defensively.
  const projectsList = (Array.isArray(projectsData) ? projectsData : (projectsData as { data?: Project[] } | undefined)?.data ?? []) as Project[];
  const effectiveProjectId = selectedProjectId || (projectsList[0]?.id ?? "");
  const { data: tasks, isLoading: tasksLoading } = useTasks(effectiveProjectId);
  const updateTaskStatus = useUpdateTaskStatus();
  const createTask = useCreateTask();

  const handleStatusChange = (taskId: string, newStatus: TaskStatus) => {
    updateTaskStatus.mutate({ id: taskId, status: newStatus });
  };

  const handleAddTask = (
    status: TaskStatus,
    data: { title: string; description?: string; priority: "low" | "medium" | "high" | "critical" },
  ) => {
    createTask.mutate({
      projectId: effectiveProjectId,
      title: data.title,
      description: data.description,
      priority: data.priority,
    });
  };

  const projectOptions = projectsList.map((p) => ({
    value: p.id,
    label: p.name,
  }));

  // Filter tasks based on agent and priority filters
  const filteredTasks = tasks?.filter((task) => {
    if (priorityFilter && task.priority !== priorityFilter) return false;
    if (agentFilter && task.assignee_id !== agentFilter) return false;
    return true;
  }) ?? [];

  const uniqueAgents = tasks
    ? [...new Set(tasks.map((t) => t.assignee_id).filter(Boolean))]
    : [];

  const agentOptions = uniqueAgents.map((a) => ({
    value: a!,
    label: `@${a?.slice(0, 8)}`,
  }));

  return (
    <div className="space-y-6">
      <PageHeader 
        title="Task Board" 
        subtitle="Global task management across projects"
      />

      {/* Filter Bar */}
      <FilterBar>
        <FilterBar.Select
          value={selectedProjectId}
          onChange={setSelectedProjectId}
          options={projectOptions}
          placeholder="Select Project"
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
      {effectiveProjectId ? (
        <KanbanBoard
          projectId={effectiveProjectId}
          tasks={filteredTasks}
          isLoading={tasksLoading}
          onStatusChange={handleStatusChange}
          onAddTask={handleAddTask}
          isSubmitting={createTask.isPending}
        />
      ) : (
        <div className="flex h-[400px] items-center justify-center rounded-xl border-2 border-dashed border-gray-800 bg-gray-900/20">
          <div className="text-center">
            <p className="text-gray-400">Please select a project to view the task board.</p>
          </div>
        </div>
      )}
    </div>
  );
}
