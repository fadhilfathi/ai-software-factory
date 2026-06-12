"use client";

/**
 * Single-deliverable viewer.
 *
 * TASK-409 — Deliverable Viewer.
 * Route: /deliverables/[id]
 *
 * Renders a deliverable's metadata + sanitized markdown body. F-008
 * trust boundary lives inside <MarkdownRenderer>, which configures
 * rehype-sanitize with a strict allowlist before react-markdown ever
 * touches the content.
 *
 * If you have raw content from a hostile source, you can verify
 * sanitization against `MarkdownRenderer.test.tsx` — it covers the
 * canonical `<script>alert('xss')</script>` payload from the security
 * review, plus inline event handlers, javascript: URLs, and embedded
 * plugin tags.
 */

import { use } from "react";
import Link from "next/link";

import { useDeliverable } from "@/lib/hooks";
import { timeAgo } from "@/lib/utils";
import type { DeliverableKind } from "@/lib/types";

import { PageHeader } from "@/components/layout/PageHeader";
import { ErrorBlock } from "@/components/ui/ErrorBlock";
import { Skeleton } from "@/components/ui/Skeleton";
import { Badge } from "@/components/ui/Badge";
import { ProjectPickerGate } from "@/components/agents/ProjectPickerGate";
import { MarkdownRenderer } from "@/components/deliverables/MarkdownRenderer";

const KIND_COLORS: Record<string, "blue" | "emerald" | "violet" | "amber" | "gray"> = {
  code: "blue",
  doc: "emerald",
  design: "violet",
  test_report: "amber",
  config: "gray",
  report: "amber",
  other: "gray",
};

const KIND_LABEL: Record<string, string> = {
  code: "Code",
  doc: "Document",
  design: "Design",
  test_report: "Test Report",
  config: "Config",
  report: "Report",
  other: "Other",
};

export default function DeliverableDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  return (
    <ProjectPickerGate>
      <DeliverableDetail params={params} />
    </ProjectPickerGate>
  );
}

function DeliverableDetail({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const { data, isLoading, isError, error, refetch } = useDeliverable(id);

  if (isError) {
    return (
      <div className="space-y-4">
        <ErrorBlock
          title="Failed to load deliverable"
          error={error}
          onRetry={() => refetch()}
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

  if (isLoading || !data) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-10 w-2/3" />
        <Skeleton className="h-6 w-1/3" />
        <Skeleton className="h-96" />
      </div>
    );
  }

  const kindColor = KIND_COLORS[data.kind ?? "other"] ?? "gray";
  const kindLabel = KIND_LABEL[data.kind ?? "other"] ?? data.kind ?? "Other";
  const version = data.version ?? data.latest_version ?? 1;

  return (
    <div className="space-y-6">
      <PageHeader
        title={data.title}
        subtitle={
          <span className="flex flex-wrap items-center gap-2">
            <Badge color={kindColor}>{kindLabel}</Badge>
            <Badge color="gray" variant="outline">
              v{version}
            </Badge>
            <span className="text-gray-500">·</span>
            <span>created {timeAgo(data.created_at)}</span>
            {data.updated_at && data.updated_at !== data.created_at ? (
              <>
                <span className="text-gray-500">·</span>
                <span>updated {timeAgo(data.updated_at)}</span>
              </>
            ) : null}
          </span>
        }
        actions={
          <div className="flex items-center gap-2">
            <Link
              href="/deliverables"
              className="rounded-md border border-gray-700 px-3 py-1.5 text-xs font-medium text-gray-300 hover:bg-gray-800"
            >
              ← All deliverables
            </Link>
            <Link
              href={`/deliverables/${data.id}/versions`}
              className="rounded-md border border-emerald-700 bg-emerald-900/20 px-3 py-1.5 text-xs font-medium text-emerald-300 hover:bg-emerald-900/40"
            >
              View version history
            </Link>
          </div>
        }
      />

      {/* Metadata strip */}
      <div className="grid gap-3 sm:grid-cols-2">
        <MetadataRow label="Deliverable ID" value={data.id} mono />
        <MetadataRow
          label="Task"
          value={
            <Link
              href={`/tasks/${data.task_id}`}
              className="font-mono text-emerald-400 hover:text-emerald-300 hover:underline"
            >
              {data.task_id}
            </Link>
          }
        />
        <MetadataRow
          label="Agent"
          value={
            <Link
              href={`/agents/${data.agent_id}`}
              className="font-mono text-emerald-400 hover:text-emerald-300 hover:underline"
            >
              {data.agent_id}
            </Link>
          }
        />
        {data.description ? (
          <MetadataRow label="Description" value={data.description} />
        ) : null}
      </div>

      {/* Markdown body — F-008 trust boundary lives inside MarkdownRenderer. */}
      <article className="rounded-lg border border-gray-800 bg-gray-900/40 p-6">
        {data.content ? (
          <MarkdownRenderer content={data.content} />
        ) : (
          <p className="text-sm italic text-gray-500">
            (No content on this deliverable.)
          </p>
        )}
      </article>
    </div>
  );
}

function MetadataRow({
  label,
  value,
  mono = false,
}: {
  label: string;
  value: React.ReactNode;
  mono?: boolean;
}) {
  return (
    <div className="rounded-md border border-gray-800 bg-gray-900/30 p-3">
      <p className="text-[10px] font-semibold uppercase tracking-wide text-gray-500">
        {label}
      </p>
      <div
        className={
          mono
            ? "mt-1 break-all font-mono text-xs text-gray-300"
            : "mt-1 break-words text-sm text-gray-300"
        }
      >
        {value}
      </div>
    </div>
  );
}
