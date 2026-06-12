"use client";

/**
 * Deliverable list / browser.
 *
 * TASK-409 — Deliverable Viewer.
 * Route: /deliverables
 *
 * Filters (URL query params, deep-linkable):
 *   - task_id   (uuid)   — only deliverables for a given task
 *   - agent_id  (uuid)   — only deliverables for a given agent
 *
 * Pagination: cursor-based via the backend's `next_cursor`. We keep the
 * cursor in URL state so back/forward works as expected.
 *
 * Project scoping: the `useDeliverables` hook is gated on `projectId`
 * from `useProjectFilters`, so when no project is picked the hook stays
 * disabled and the `ProjectPickerGate` shows the picker UI instead.
 */

import { useCallback, Suspense } from "react";
import { useRouter, useSearchParams, usePathname } from "next/navigation";

import { Badge } from "@/components/ui/Badge";
import { EmptyState } from "@/components/ui/EmptyState";
import { ErrorBlock } from "@/components/ui/ErrorBlock";
import { Skeleton } from "@/components/ui/Skeleton";
import { ProjectPickerGate } from "@/components/agents/ProjectPickerGate";
import { DeliverableCard } from "@/components/deliverables/DeliverableCard";
import { useDeliverables } from "@/lib/hooks";

import type { Deliverable } from "@/lib/types";

const PAGE_LIMIT = 24;

function DeliverablesBrowser() {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();

  const taskId = searchParams.get("task_id") ?? undefined;
  const agentId = searchParams.get("agent_id") ?? undefined;
  const cursor = searchParams.get("cursor") ?? undefined;

  const {
    data,
    isLoading,
    isError,
    error,
    refetch,
  } = useDeliverables({
    task_id: taskId,
    agent_id: agentId,
    limit: PAGE_LIMIT,
    cursor,
  });

  const items: Deliverable[] = data?.data ?? [];
  const nextCursor = data?.page_info?.next_cursor ?? null;
  const hasActiveFilters = Boolean(taskId || agentId);

  /**
   * Push a single param change (or removal) onto the URL, preserving the
   * other filters. Resets the cursor on any filter change because the
   * current page is no longer meaningful.
   */
  const setParam = useCallback(
    (key: "task_id" | "agent_id" | "cursor", value: string | null) => {
      const next = new URLSearchParams(searchParams.toString());
      if (value === null || value === "") {
        next.delete(key);
      } else {
        next.set(key, value);
      }
      // Any filter change → drop the cursor; we can't resume a page
      // across a different result set.
      if (key !== "cursor") {
        next.delete("cursor");
      }
      const qs = next.toString();
      router.push(qs ? `${pathname}?${qs}` : pathname);
    },
    [router, pathname, searchParams],
  );

  const clearFilters = useCallback(() => {
    router.push(pathname);
  }, [router, pathname]);

  return (
    <div className="space-y-6">
      {/* Filter row */}
      <div className="rounded-lg border border-gray-800 bg-gray-900/40 p-4">
        <div className="grid gap-3 sm:grid-cols-2">
          <FilterInput
            label="Task ID"
            placeholder="e.g. 8b2a…"
            value={taskId ?? ""}
            onChange={(v) => setParam("task_id", v || null)}
          />
          <FilterInput
            label="Agent ID"
            placeholder="e.g. c4f0…"
            value={agentId ?? ""}
            onChange={(v) => setParam("agent_id", v || null)}
          />
        </div>
        {hasActiveFilters ? (
          <div className="mt-3 flex items-center justify-between">
            <div className="flex flex-wrap items-center gap-2 text-xs text-gray-400">
              <span>Active filters:</span>
              {taskId ? (
                <Badge color="cyan" variant="outline">
                  task_id: {shortId(taskId)}
                </Badge>
              ) : null}
              {agentId ? (
                <Badge color="violet" variant="outline">
                  agent_id: {shortId(agentId)}
                </Badge>
              ) : null}
            </div>
            <button
              type="button"
              onClick={clearFilters}
              className="text-xs font-medium text-emerald-400 hover:text-emerald-300"
            >
              Clear filters
            </button>
          </div>
        ) : null}
      </div>

      {/* Status / results header */}
      <div className="flex items-center justify-between text-sm text-gray-400">
        <span>
          {isLoading
            ? "Loading deliverables…"
            : data
              ? `${items.length} deliverable${items.length === 1 ? "" : "s"}${
                  hasActiveFilters ? " (filtered)" : ""
                }`
              : null}
        </span>
        {data && nextCursor ? (
          <span className="text-xs text-gray-500">
            more results available
          </span>
        ) : null}
      </div>

      {/* Error */}
      {isError ? (
        <ErrorBlock
          title="Failed to load deliverables"
          error={error}
          onRetry={() => refetch()}
        />
      ) : null}

      {/* Loading skeleton */}
      {isLoading ? (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {Array.from({ length: 6 }).map((_, i) => (
            <Skeleton.Card key={i} className="h-40" />
          ))}
        </div>
      ) : null}

      {/* Empty state */}
      {!isLoading && !isError && items.length === 0 ? (
        <EmptyState
          icon="📦"
          title={hasActiveFilters ? "No matching deliverables" : "No deliverables yet"}
          description={
            hasActiveFilters
              ? "Try removing one of the filters above."
              : "Deliverables appear here once agents start producing them."
          }
          action={
            hasActiveFilters ? (
              <button
                type="button"
                onClick={clearFilters}
                className="rounded-md border border-emerald-700 px-3 py-1.5 text-xs font-medium text-emerald-300 hover:bg-emerald-900/20"
              >
                Clear filters
              </button>
            ) : null
          }
        />
      ) : null}

      {/* Result grid */}
      {!isLoading && !isError && items.length > 0 ? (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {items.map((d) => (
            <DeliverableCard
              key={d.id}
              deliverable={d}
              onClick={() => router.push(`/deliverables/${d.id}`)}
            />
          ))}
        </div>
      ) : null}

      {/* Pagination */}
      {!isLoading && !isError && (nextCursor || cursor) ? (
        <div className="flex items-center justify-between border-t border-gray-800 pt-4">
          <button
            type="button"
            onClick={() => setParam("cursor", null)}
            disabled={!cursor}
            className="rounded-md border border-gray-700 px-3 py-1.5 text-xs font-medium text-gray-300 hover:bg-gray-800 disabled:cursor-not-allowed disabled:opacity-40"
          >
            ← First page
          </button>
          <button
            type="button"
            onClick={() => nextCursor && setParam("cursor", nextCursor)}
            disabled={!nextCursor}
            className="rounded-md border border-emerald-700 bg-emerald-900/20 px-3 py-1.5 text-xs font-medium text-emerald-300 hover:bg-emerald-900/40 disabled:cursor-not-allowed disabled:opacity-40"
          >
            Next page →
          </button>
        </div>
      ) : null}
    </div>
  );
}

function FilterInput({
  label,
  placeholder,
  value,
  onChange,
}: {
  label: string;
  placeholder: string;
  value: string;
  onChange: (v: string) => void;
}) {
  return (
    <label className="block">
      <span className="mb-1 block text-xs font-medium text-gray-400">
        {label}
      </span>
      <input
        type="text"
        inputMode="text"
        autoComplete="off"
        spellCheck={false}
        placeholder={placeholder}
        value={value}
        onChange={(e) => onChange(e.target.value.trim())}
        className="w-full rounded-md border border-gray-700 bg-gray-950 px-3 py-2 text-sm text-gray-100 placeholder:text-gray-600 focus:border-emerald-500 focus:outline-none focus:ring-1 focus:ring-emerald-500"
      />
    </label>
  );
}

function shortId(id: string): string {
  if (id.length <= 8) return id;
  return `${id.slice(0, 4)}…${id.slice(-4)}`;
}

/**
 * Outer page wrapper. The browser itself uses `useSearchParams`, which
 * Next 15 requires to be inside a Suspense boundary (otherwise the
 * static prerender fails). The gate sits outside the boundary because
 * it doesn't need URL state.
 */
export default function DeliverablesPage() {
  return (
    <ProjectPickerGate>
      <div className="space-y-6">
        <header>
          <h1 className="text-2xl font-bold text-gray-100">Deliverables</h1>
          <p className="mt-1 text-sm text-gray-400">
            Markdown artifacts produced by agents. Filter by task or agent,
            then open a card to view the rendered content or its version
            history.
          </p>
        </header>
        <Suspense
          fallback={
            <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
              {Array.from({ length: 6 }).map((_, i) => (
                <Skeleton.Card key={i} className="h-40" />
              ))}
            </div>
          }
        >
          <DeliverablesBrowser />
        </Suspense>
      </div>
    </ProjectPickerGate>
  );
}
