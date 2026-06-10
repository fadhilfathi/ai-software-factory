"use client";

import { Suspense, useState } from "react";
import { PageHeader } from "@/components/layout/PageHeader";
import { useProjects } from "@/lib/hooks";
import { useProjectFilters } from "@/hooks/useProjectFilters";
import { timeAgo, cn } from "@/lib/utils";
import { ProjectStatusBadge } from "@/components/shared/StatusBadge";
import { Skeleton } from "@/components/ui/Skeleton";
import { EmptyState } from "@/components/ui/EmptyState";
import { ErrorBlock } from "@/components/ui/ErrorBlock";
import { FilterBar, SearchInput } from "@/components/shared/FilterBar";
import { PaginationInfo } from "@/components/shared/PaginationInfo";
import { ProgressBar } from "@/components/ui/ProgressBar";
import Link from "next/link";

const STATUS_OPTIONS = [
  { value: "initializing", label: "Initializing" },
  { value: "in_progress", label: "In Progress" },
  { value: "completed", label: "Completed" },
  { value: "archived", label: "Archived" },
];

function ProjectsListContent() {
  const { filters, setFilter } = useProjectFilters({ status: "all" });
  const [search, setSearch] = useState("");

  const queryFilters: Record<string, string | undefined> = {};
  if (filters.status && filters.status !== "all") queryFilters.status = filters.status;
  if (search) queryFilters.search = search;

  const { data, isLoading, isError, error } = useProjects(queryFilters);

  const projects = data?.data ?? [];
  const showEmptyMessage = !search && (!filters.status || filters.status === "all");

  return (
    <div>
      {/* Filter Bar */}
      <FilterBar>
        <SearchInput
          value={search}
          onChange={setSearch}
          placeholder="Search projects..."
        />
        <FilterBar.Select
          value={filters.status ?? ""}
          onChange={(v) => setFilter("status", v === "" ? undefined : v)}
          options={STATUS_OPTIONS}
          placeholder="All Status"
        />
      </FilterBar>

      {/* Error State */}
      {isError && (
        <ErrorBlock
          message={(error as Error)?.message ?? "Unknown error"}
          title="Failed to load projects"
        />
      )}

      {/* Loading State */}
      {isLoading && <Skeleton.CardGrid count={6} />}

      {/* Empty State */}
      {!isLoading && !isError && projects.length === 0 && (
        <EmptyState
          icon="📁"
          title={
            search || (filters.status && filters.status !== "all")
              ? "No projects match your filters"
              : "No projects yet"
          }
          description={
            search || (filters.status && filters.status !== "all")
              ? "Try adjusting your search or filter criteria."
              : "Create your first project to get started."
          }
          action={
            showEmptyMessage && (
              <Link
                href="/projects/new"
                className="rounded-lg bg-emerald-500 px-4 py-2 text-sm font-medium text-white hover:bg-emerald-600 transition-colors"
              >
                Create Project
              </Link>
            )
          }
        />
      )}

      {/* Project Grid */}
      {!isLoading && !isError && projects.length > 0 && (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {projects.map((project) => (
            <Link
              key={project.id}
              href={`/projects/${project.id}`}
              className="rounded-lg border border-gray-800 bg-gray-950 p-4 hover:border-gray-700 transition-colors group"
            >
              <div className="flex items-start justify-between gap-2">
                <h3 className="font-medium text-gray-200 truncate group-hover:text-emerald-400 transition-colors">
                  {project.name}
                </h3>
                <ProjectStatusBadge status={project.status} />
              </div>
              {project.description && (
                <p className="mt-1.5 text-xs text-gray-500 line-clamp-2">
                  {project.description}
                </p>
              )}
              <ProgressBar value={project.progress ?? 0} className="mt-3" />
              <div className="mt-3 flex items-center justify-between text-xs text-gray-500">
                <span>{timeAgo(project.updated_at)}</span>
                {project.active_agents !== undefined && project.active_agents > 0 && (
                  <span>{project.active_agents} agent{project.active_agents !== 1 ? "s" : ""}</span>
                )}
              </div>
            </Link>
          ))}
        </div>
      )}

      {/* Pagination info */}
      {data?.pagination && data.pagination.total > 0 && (
        <PaginationInfo
          total={data.pagination.total}
          page={data.pagination.page}
          pages={data.pagination.pages}
          showing={projects.length}
        />
      )}
    </div>
  );
}

export default function ProjectsListPage() {
  return (
    <div>
      <PageHeader
        title="Projects"
        actions={
          <Link
            href="/projects/new"
            className="inline-flex items-center gap-2 rounded-lg bg-emerald-500 px-4 py-2 text-sm font-medium text-white hover:bg-emerald-600 transition-colors"
          >
            + New Project
          </Link>
        }
      />
      <Suspense fallback={<Skeleton.CardGrid count={6} />}>
        <ProjectsListContent />
      </Suspense>
    </div>
  );
}
