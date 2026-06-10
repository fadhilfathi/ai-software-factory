"use client";

import { PageHeader } from "@/components/layout/PageHeader";
import { useProjects, useRecentActivity } from "@/lib/hooks";
import { timeAgo, formatNumber, formatCurrency } from "@/lib/utils";
import { MetricCard } from "@/components/shared/MetricCard";
import { Skeleton } from "@/components/ui/Skeleton";
import { EmptyState } from "@/components/ui/EmptyState";
import { ProgressBar } from "@/components/ui/ProgressBar";
import Link from "next/link";

export default function DashboardPage() {
  const { data: projectsData, isLoading: projectsLoading } = useProjects({ limit: "100" });
  const { data: activity, isLoading: activityLoading } = useRecentActivity();

  const projects = projectsData?.data ?? [];
  const activeProjects = projects.filter((p) => p.status === "in_progress");
  const completedProjects = projects.filter((p) => p.status === "completed");
  const totalSpend = projects.length * 1240;

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

      {/* Active Projects + Activity Feed */}
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
                      {project.progress ?? 0}%
                    </span>
                  </div>
                  <ProgressBar
                    value={project.progress ?? 0}
                    className="mt-2"
                  />
                  {project.active_agents !== undefined && project.active_agents > 0 && (
                    <p className="mt-1.5 text-[10px] text-gray-500">
                      {project.active_agents} agent{project.active_agents !== 1 ? "s" : ""} active
                    </p>
                  )}
                </Link>
              ))}
            </div>
          )}
        </div>

        {/* Activity Feed */}
        <div className="rounded-lg border border-gray-800 bg-gray-950 p-4">
          <h2 className="mb-4 text-sm font-semibold text-gray-300 uppercase tracking-wider">
            Recent Activity
          </h2>

          {activityLoading ? (
            <Skeleton.Activity count={3} />
          ) : !activity || activity.length === 0 ? (
            <p className="py-8 text-center text-sm text-gray-500">No recent activity</p>
          ) : (
            <div className="space-y-3">
              {activity.slice(0, 8).map((item) => (
                <div key={item.id} className="flex items-start gap-3 text-sm text-gray-400">
                  <span className="mt-0.5 flex h-5 w-5 items-center justify-center rounded-full bg-gray-800 text-[10px] font-bold text-gray-300">
                    {item.agent_type?.[0]?.toUpperCase() ?? "?"}
                  </span>
                  <div className="flex-1 min-w-0">
                    <p className="truncate">{item.text}</p>
                    <p className="mt-0.5 text-xs text-gray-600">
                      {timeAgo(item.created_at)}
                    </p>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
