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

import { AgentBadge } from "@/components/shared/AgentBadge";

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
    <div className="space-y-8">
      <PageHeader
        title={project.name}
        subtitle={
          <span className="flex items-center gap-3">
            <span className="text-gray-500 font-mono text-xs">{project.id.toUpperCase()}</span>
            <span className="h-4 w-px bg-gray-800" />
            {/* Project.status is not in the Sprint 1-3 spec; we infer
                an "active" / "archived" badge from the updated_at
                recency instead so the header still gives a status cue. */}
            {(() => {
              const updatedAt = project.updated_at ? new Date(project.updated_at).getTime() : 0;
              const isArchived = updatedAt > 0 && Date.now() - updatedAt > 30 * 24 * 60 * 60 * 1000;
              return <ProjectStatusBadge status={isArchived ? "archived" : "active"} />;
            })()}
          </span>
        }
        actions={
          <div className="flex items-center gap-2">
            <Link
              href={`/projects/${id}/board`}
              className="rounded-lg bg-emerald-500 px-4 py-2 text-sm font-semibold text-white hover:bg-emerald-600 transition-all shadow-lg shadow-emerald-500/10 active:scale-95"
            >
              Kanban Board
            </Link>
            <div className="h-8 w-px bg-gray-800 mx-1" />
            <Link
              href={`/projects/${id}/edit`}
              className="rounded-lg border border-gray-800 px-3 py-2 text-sm text-gray-400 hover:bg-gray-800 hover:text-gray-200 transition-colors"
            >
              Edit
            </Link>
            <button
              onClick={() => setShowDeleteDialog(true)}
              className="rounded-lg border border-red-900/50 px-3 py-2 text-sm text-red-500 hover:bg-red-950/30 transition-colors"
              type="button"
            >
              Delete
            </button>
          </div>
        }
      />

      <div className="grid gap-8 lg:grid-cols-3">
        {/* Main Content */}
        <div className="lg:col-span-2 space-y-8">
          {/* Description Section */}
          <section className="rounded-2xl border border-gray-800 bg-gray-950/30 p-6">
            <h3 className="text-xs font-bold uppercase tracking-widest text-gray-500 mb-4">Project Overview</h3>
            <p className="text-sm text-gray-300 leading-relaxed italic">
              {project.description || "No description provided for this project."}
            </p>
          </section>

          {/* Task Summary Grid */}
          <section>
            <h3 className="text-xs font-bold uppercase tracking-widest text-gray-500 mb-4">Task Distribution</h3>
            <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 md:grid-cols-6">
              {taskCounts.map(({ key, label, count }) => (
                <div
                  key={key}
                  className="group rounded-xl border border-gray-800 bg-gray-950 p-4 transition-all hover:border-gray-700"
                >
                  <p className="text-2xl font-bold text-gray-100 group-hover:text-emerald-400 transition-colors">{count}</p>
                  <p className="text-[10px] text-gray-500 uppercase tracking-widest mt-1 font-bold">
                    {label}
                  </p>
                </div>
              ))}
            </div>
          </section>

          {/* Recent Tasks List */}
          <section className="space-y-4">
            <div className="flex items-center justify-between">
              <h3 className="text-xs font-bold uppercase tracking-widest text-gray-500">
                Project Tasks
              </h3>
              <Link 
                href={`/projects/${id}/board`}
                className="text-xs text-emerald-500 hover:underline"
              >
                View full board &rarr;
              </Link>
            </div>

            <div className="space-y-3">
              {tasksLoading ? (
                <Skeleton.List count={3} />
              ) : !tasks || tasks.length === 0 ? (
                <EmptyState
                  icon=""
                  title="No tasks yet"
                  className="rounded-xl border border-gray-800 bg-gray-950/50 py-12"
                />
              ) : (
                tasks.slice(0, 10).map((task) => (
                  <div
                    key={task.id}
                    className="group rounded-xl border border-gray-800 bg-gray-950 p-4 hover:border-gray-700 hover:bg-gray-900/40 transition-all flex items-center justify-between gap-4"
                  >
                    <div className="flex items-center gap-4 min-w-0">
                      <span className="text-[10px] font-mono text-gray-600 shrink-0 uppercase">
                        {task.id.slice(0, 4)}
                      </span>
                      <span className="text-sm font-medium text-gray-200 truncate group-hover:text-white transition-colors">
                        {task.title}
                      </span>
                    </div>
                    <div className="flex items-center gap-3 shrink-0">
                      <PriorityBadge priority={task.priority} uppercase={false} className="hidden sm:inline-flex" />
                      <TaskStatusBadge status={task.status} />
                      <div className="h-4 w-px bg-gray-800" />
                      {task.assignee_id ? (
                        <AgentBadge type="developer" />
                      ) : (
                        <div className="h-6 w-6 rounded-full border border-gray-800 bg-gray-900 flex items-center justify-center text-[8px] text-gray-600 italic">
                          ?
                        </div>
                      )}
                    </div>
                  </div>
                ))
              )}
            </div>
          </section>
        </div>

        {/* Sidebar Info */}
        <div className="space-y-6">
          <section className="rounded-2xl border border-gray-800 bg-gray-950/50 p-6 space-y-6">
            <div>
              <h3 className="text-xs font-bold uppercase tracking-widest text-gray-500 mb-4">Project Health</h3>
              <div className="space-y-2">
                <div className="flex justify-between text-xs font-bold">
                  <span className="text-gray-400">Tasks tracked</span>
                  {/* `progress` is not in the Sprint 1-3 Project type;
                      show the count of tasks in this project instead. */}
                  <span className="text-emerald-400">
                    {(project as { task_count?: number }).task_count ?? "—"}
                  </span>
                </div>
                <ProgressBar
                  value={Math.min(100, ((project as { task_count?: number }).task_count ?? 0))}
                  size="md"
                />
              </div>
            </div>

            <div className="space-y-4 pt-4 border-t border-gray-800/50">
              <h3 className="text-xs font-bold uppercase tracking-widest text-gray-500">Metadata</h3>
              <div className="space-y-3">
                {(project as { template?: string }).template && (
                  <div className="flex flex-col gap-1">
                    <span className="text-[10px] text-gray-600 uppercase font-bold tracking-tighter">Architecture</span>
                    <span className="text-sm text-gray-300 font-medium">{(project as { template?: string }).template}</span>
                  </div>
                )}
                <div className="flex flex-col gap-1">
                  <span className="text-[10px] text-gray-600 uppercase font-bold tracking-tighter">Active Agents</span>
                  <div className="flex items-center gap-2">
                    <div className="h-2 w-2 rounded-full bg-emerald-500 animate-pulse" />
                    <span className="text-sm text-gray-300 font-medium">{(project as { active_agents?: number }).active_agents ?? 0} Agents Online</span>
                  </div>
                </div>
                <div className="flex flex-col gap-1">
                  <span className="text-[10px] text-gray-600 uppercase font-bold tracking-tighter">Timeline</span>
                  <div className="space-y-1">
                    <p className="text-[11px] text-gray-400">Created {timeAgo(project.created_at)}</p>
                    <p className="text-[11px] text-gray-400">Last activity {timeAgo(project.updated_at)}</p>
                  </div>
                </div>
              </div>
            </div>

            <div className="pt-4">
              <Link
                href={`/projects/${id}/board`}
                className="flex items-center justify-center gap-2 rounded-xl bg-gray-900 px-4 py-3 text-sm font-bold text-gray-200 hover:bg-gray-800 transition-all border border-gray-800"
              >
                Go to Mission Control &rarr;
              </Link>
            </div>
          </section>
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
