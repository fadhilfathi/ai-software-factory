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

import { ProjectCard } from "@/components/shared/ProjectCard";

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
    <div className="space-y-6">
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
                className="rounded-lg bg-emerald-500 px-6 py-2.5 text-sm font-semibold text-white hover:bg-emerald-600 transition-all shadow-lg shadow-emerald-500/20 active:scale-95"
              >
                Create Project
              </Link>
            )
          }
        />
      )}

      {/* Project Grid */}
      {!isLoading && !isError && projects.length > 0 && (
        <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
          {projects.map((project) => (
            <ProjectCard key={project.id} project={project} />
          ))}
        </div>
      )}

      {/* Pagination info */}
      {data?.pagination && data.pagination.total > 0 && (
        <div className="pt-4 border-t border-gray-800/50">
          <PaginationInfo
            total={data.pagination.total}
            page={data.pagination.page}
            pages={data.pagination.pages}
            showing={projects.length}
          />
        </div>
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
