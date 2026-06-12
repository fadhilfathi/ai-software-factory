"use client";

import { use } from "react";
import Link from "next/link";
import { PageHeader } from "@/components/layout/PageHeader";
import { useProject, useTasks, useUpdateTaskStatus, useCreateTask } from "@/lib/hooks";
import { KanbanBoard } from "@/components/kanban/KanbanBoard";
import { ErrorBlock } from "@/components/ui/ErrorBlock";
import type { TaskStatus } from "@/lib/types";

export default function KanbanBoardPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const { data: project, isLoading: projectLoading, isError: projectError } = useProject(id);
  const { data: tasks, isLoading: tasksLoading } = useTasks(id);
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
      projectId: id,
      title: data.title,
      description: data.description,
      priority: data.priority,
    });
  };

  if (projectError) {
    return (
      <div>
        <PageHeader title="Board" />
        <ErrorBlock.Page
          message="Could not load this project."
          backHref="/projects"
        />
      </div>
    );
  }

  return (
    <div>
      <PageHeader
        title={projectLoading ? "Loading..." : `${project?.name} Board`}
        subtitle={!projectLoading && project && `ID: ${project.id}`}
        actions={
          <Link
            href={`/projects/${id}`}
            className="text-sm text-gray-400 hover:text-gray-200 transition-colors"
          >
            &larr; Back to Project
          </Link>
        }
      />

      <KanbanBoard
        projectId={id}
        tasks={tasks ?? []}
        isLoading={projectLoading || tasksLoading}
        onStatusChange={handleStatusChange}
        onAddTask={handleAddTask}
        isSubmitting={createTask.isPending}
      />
    </div>
  );
}
