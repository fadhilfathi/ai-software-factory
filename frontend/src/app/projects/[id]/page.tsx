"use client";

import { use, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { PageHeader } from "@/components/layout/PageHeader";
import { useProject, useTasks, useDeleteProject } from "@/lib/hooks";
import { timeAgo, cn } from "@/lib/utils";
import { Skeleton } from "@/components/ui/Skeleton";
import { EmptyState } from "@/components/ui/EmptyState";
import { ErrorBlock } from "@/components/ui/ErrorBlock";
import { ProgressBar } from "@/components/ui/ProgressBar";
import { TaskStatusBadge } from "@/components/shared/StatusBadge";
import { PriorityBadge } from "@/components/shared/PriorityBadge";
import { ProjectStatusBadge } from "@/components/shared/StatusBadge";
import { ConfirmDialog } from "@/components/shared/ConfirmDialog";
import type { TaskStatus } from "@/lib/types";

const TASK_SUMMARY_STATUSES: { key: TaskStatus; label: string }[] = [
  { key: "backlog", label: "Backlog" },
  { key: "ready", label: "Ready" },
  { key: "in_progress", label: "In Progress" },
  { key: "review", label: "Review" },
  { key: "done", label: "Done" },
  { key: "blocked", label: "Blocked" },
];

export default function ProjectDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const router = useRouter();
  const { data: project, isLoading: projectLoading, isError: projectError } = useProject(id);
  const { data: tasks, isLoading: tasksLoading } = useTasks(id);
  const deleteProject = useDeleteProject();
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);

  const taskCounts = TASK_SUMMARY_STATUSES.map((s) => ({
    ...s,
    count: tasks?.filter((t) => t.status === s.key).length ?? 0,
  }));
  const totalTasks = tasks?.length ?? 0;

  const handleDelete = async () => {
    await deleteProject.mutateAsync(id);
    router.push("/projects");
  };

  if (projectLoading) {
    return (
      <div>
        <PageHeader title="Loading..." />
        <div className="space-y-4">
          <Skeleton className="h-24 w-full" />
          <div className="flex gap-3">
            {[1, 2, 3, 4, 5, 6].map((i) => (
              <div key={i} className="flex-1 rounded-lg border border-gray-800 bg-gray-950 p-3">
                <div className="mx-auto mb-2 h-8 w-8 rounded-full bg-gray-800" />
                <div className="h-3 w-12 mx-auto rounded bg-gray-800" />
              </div>
            ))}
          </div>
        </div>
      </div>
    );
  }

  if (projectError || !project) {
    return (
      <div>
        <PageHeader title="Project Not Found" />
        <ErrorBlock.Page
          message="Could not load this project. It may have been deleted or you may not have access."
          backHref="/projects"
        />
      </div>
    );
  }

  return (
    <div>
      <PageHeader
        title={project.name}
        subtitle={
          <span className="flex items-center gap-2">
            <span className="text-gray-500">ID: {project.id}</span>
            <span className="text-gray-600">|</span>
            <ProjectStatusBadge status={project.status} />
          </span>
        }
        actions={
          <div className="flex items-center gap-2">
            <Link
              href={`/projects/${id}/board`}
              className="rounded-lg border border-gray-800 px-3 py-1.5 text-sm text-gray-300 hover:bg-gray-800 transition-colors"
            >
              Kanban Board
            </Link>
            <Link
              href={`/projects/${id}/edit`}
              className="rounded-lg border border-gray-800 px-3 py-1.5 text-sm text-gray-300 hover:bg-gray-800 transition-colors"
            >
              Edit
            </Link>
            <button
              onClick={() => setShowDeleteDialog(true)}
              className="rounded-lg border border-red-800 px-3 py-1.5 text-sm text-red-400 hover:bg-red-950/50 transition-colors"
              type="button"
            >
              Delete
            </button>
            <Link
              href="/projects"
              className="text-sm text-gray-400 hover:text-gray-200 transition-colors ml-2"
            >
              &larr; Back
            </Link>
          </div>
        }
      />

      {/* Description */}
      {project.description && (
        <p className="mb-6 text-sm text-gray-400">{project.description}</p>
      )}

      {/* Task Summary Counts */}
      <div className="mb-6 grid grid-cols-3 gap-3 sm:grid-cols-6">
        {taskCounts.map(({ key, label, count }) => (
          <div
            key={key}
            className="rounded-lg border border-gray-800 bg-gray-950 p-3 text-center"
          >
            <p className="text-lg font-bold text-gray-100">{count}</p>
            <p className="text-[10px] text-gray-500 uppercase tracking-wider mt-0.5">
              {label}
            </p>
          </div>
        ))}
      </div>

      {/* Two column: Task list + Details */}
      <div className="grid gap-6 lg:grid-cols-3">
        {/* Tasks */}
        <div className="lg:col-span-2 space-y-3">
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-semibold text-gray-300 uppercase tracking-wider">
              Tasks
            </h3>
            <span className="text-xs text-gray-500">
              {totalTasks > 0 ? `${totalTasks} total` : ""}
            </span>
          </div>

          {tasksLoading ? (
            <Skeleton.List count={3} />
          ) : !tasks || tasks.length === 0 ? (
            <EmptyState
              icon=""
              title="No tasks yet"
              className="rounded-lg border border-gray-800 bg-gray-950 py-8"
            />
          ) : (
            tasks.map((task) => (
              <div
                key={task.id}
                className="rounded-lg border border-gray-800 bg-gray-950 p-3 hover:border-gray-700 transition-colors"
              >
                <div className="flex items-center justify-between gap-2">
                  <div className="flex items-center gap-2 min-w-0">
                    <span className="text-xs font-mono text-gray-500 shrink-0">
                      {task.id.slice(0, 8)}
                    </span>
                    <span className="text-sm font-medium text-gray-200 truncate">
                      {task.title}
                    </span>
                  </div>
                  <div className="flex items-center gap-2 shrink-0">
                    <PriorityBadge priority={task.priority} />
                    <TaskStatusBadge status={task.status} />
                  </div>
                </div>
                {task.assignee_id && (
                  <span className="mt-1.5 inline-flex items-center gap-1 rounded bg-gray-800 px-2 py-0.5 text-[10px] text-gray-400">
                    @{task.assignee_id.slice(0, 8)}
                  </span>
                )}
              </div>
            ))
          )}
        </div>

        {/* Project Info Sidebar */}
        <div className="space-y-3">
          <h3 className="text-sm font-semibold text-gray-300 uppercase tracking-wider">
            Details
          </h3>

          <div className="space-y-2 rounded-lg border border-gray-800 bg-gray-950 p-3">
            <div className="flex justify-between text-xs">
              <span className="text-gray-500">Progress</span>
              <span className="text-gray-300">{project.progress ?? 0}%</span>
            </div>
            <ProgressBar value={project.progress ?? 0} />
          </div>

          <div className="rounded-lg border border-gray-800 bg-gray-950 p-3 space-y-2 text-xs">
            {project.template && (
              <div className="flex justify-between">
                <span className="text-gray-500">Template</span>
                <span className="text-gray-300">{project.template}</span>
              </div>
            )}
            {project.active_agents !== undefined && (
              <div className="flex justify-between">
                <span className="text-gray-500">Active Agents</span>
                <span className="text-gray-300">{project.active_agents}</span>
              </div>
            )}
            <div className="flex justify-between">
              <span className="text-gray-500">Created</span>
              <span className="text-gray-300">{timeAgo(project.created_at)}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-gray-500">Updated</span>
              <span className="text-gray-300">{timeAgo(project.updated_at)}</span>
            </div>
          </div>

          <Link
            href={`/projects/${id}/board`}
            className="flex items-center justify-center gap-2 rounded-lg border border-gray-800 px-4 py-2 text-sm text-gray-300 hover:bg-gray-800 transition-colors"
          >
            Open Kanban Board
          </Link>
        </div>
      </div>

      <ConfirmDialog
        open={showDeleteDialog}
        onConfirm={handleDelete}
        onCancel={() => setShowDeleteDialog(false)}
        title="Delete Project"
        message={`Are you sure you want to delete "${project.name}"? This action cannot be undone.`}
        confirmLabel="Delete"
        variant="danger"
        loading={deleteProject.isPending}
      />
    </div>
  );
}
