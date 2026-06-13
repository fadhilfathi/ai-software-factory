"use client";

/**
 * Version history + diff viewer.
 *
 * TASK-409 — Deliverable Viewer.
 * Route: /deliverables/[id]/versions
 *
 * Two responsibilities:
 *   1. Show every version of this deliverable in DESC order (latest first)
 *   2. Let the user diff any two versions side-by-side
 *
 * The diff itself is delegated to <VersionDiff>; this page is
 * responsible for the list + a click-to-pick affordance. The hook
 * returns versions in ASC order; the brief requires the list to
 * render in DESC, so we sort here too.
 */

import { use, useMemo } from "react";
import Link from "next/link";

import {
  useDeliverable,
  useDeliverableVersions,
} from "@/lib/hooks";
import { timeAgo, cn } from "@/lib/utils";

import { PageHeader } from "@/components/layout/PageHeader";
import { ErrorBlock } from "@/components/ui/ErrorBlock";
import { Skeleton } from "@/components/ui/Skeleton";
import { EmptyState } from "@/components/ui/EmptyState";
import { Badge } from "@/components/ui/Badge";
import { ProjectPickerGate } from "@/components/agents/ProjectPickerGate";
import { VersionDiff } from "@/components/deliverables/VersionDiff";

export default function DeliverableVersionsPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  return (
    <ProjectPickerGate>
      <DeliverableVersions params={params} />
    </ProjectPickerGate>
  );
}

function DeliverableVersions({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);

  const deliverableQuery = useDeliverable(id);
  const versionsQuery = useDeliverableVersions(id);

  /** ASC → DESC so the list reads newest-first. */
  const versions = useMemo(() => {
    const raw = versionsQuery.data ?? [];
    return [...raw].sort((a, b) => b.version - a.version);
  }, [versionsQuery.data]);

  const isLoading =
    deliverableQuery.isLoading || versionsQuery.isLoading;
  const isError = deliverableQuery.isError || versionsQuery.isError;
  const error = deliverableQuery.error ?? versionsQuery.error;
  const refetch = () => {
    deliverableQuery.refetch();
    versionsQuery.refetch();
  };

  if (isError) {
    return (
      <div className="space-y-4">
        <ErrorBlock
          title="Failed to load version history"
          error={error}
          onRetry={refetch}
        />
        <Link
          href="/deliverables"
          className="inline-block text-sm text-emerald-400 hover:text-emerald-300"
        >
          ← Back to deliverables
        </Link>
      </div>
    );
  }

  if (isLoading) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-10 w-2/3" />
        <Skeleton className="h-6 w-1/3" />
        <Skeleton className="h-64" />
      </div>
    );
  }

  const deliverable = deliverableQuery.data;

  return (
    <div className="space-y-6">
      <PageHeader
        title={
          deliverable
            ? `Version history — ${deliverable.title}`
            : "Version history"
        }
        subtitle={
          deliverable ? (
            <span className="flex flex-wrap items-center gap-2">
              <Badge color="gray" variant="outline">
                {versions.length} version{versions.length === 1 ? "" : "s"}
              </Badge>
              <span className="text-gray-500">·</span>
              <Link
                href={`/deliverables/${deliverable.id}`}
                className="text-emerald-400 hover:text-emerald-300 hover:underline"
              >
                ← Back to current view
              </Link>
            </span>
          ) : null
        }
      />

      {versions.length === 0 ? (
        <EmptyState
          icon="🕰"
          title="No versions recorded"
          description="This deliverable does not have a version history yet."
        />
      ) : (
        <div className="grid gap-6 lg:grid-cols-[18rem_minmax(0,1fr)]">
          {/* Version list — DESC order (latest first) */}
          <aside className="space-y-2">
            <h2 className="text-xs font-semibold uppercase tracking-wide text-gray-500">
              Versions
            </h2>
            <ol className="space-y-1">
              {versions.map((v) => {
                const isCurrent = v.version === (deliverable?.version ?? -1);
                return (
                  <li key={v.id}>
                    <div
                      className={cn(
                        "rounded-md border px-3 py-2",
                        isCurrent
                          ? "border-emerald-700 bg-emerald-900/20"
                          : "border-gray-800 bg-gray-900/30",
                      )}
                    >
                      <div className="flex items-center justify-between">
                        <span className="font-mono text-sm text-gray-100">
                          v{v.version}
                        </span>
                        {isCurrent ? (
                          <Badge color="emerald" variant="outline">
                            current
                          </Badge>
                        ) : null}
                      </div>
                      <p className="mt-1 truncate text-xs text-gray-400">
                        {v.title}
                      </p>
                      <p className="mt-0.5 text-[10px] text-gray-500">
                        {timeAgo(v.created_at)}
                        {v.created_by ? ` · by ${shortId(v.created_by)}` : null}
                      </p>
                    </div>
                  </li>
                );
              })}
            </ol>
          </aside>

          {/* Diff panel — VersionDiff owns the from/to pickers. */}
          <section className="space-y-3">
            <h2 className="text-xs font-semibold uppercase tracking-wide text-gray-500">
              Compare versions
            </h2>
            <VersionDiff versions={versions} />
          </section>
        </div>
      )}
    </div>
  );
}

function shortId(id: string): string {
  if (id.length <= 8) return id;
  return `${id.slice(0, 4)}…${id.slice(-4)}`;
}
