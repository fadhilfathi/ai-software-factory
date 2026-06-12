"use client";

import { use } from "react";
import Link from "next/link";
import { PageHeader } from "@/components/layout/PageHeader";
import { useDeliverable } from "@/lib/hooks";
import { timeAgo } from "@/lib/utils";
import { Skeleton } from "@/components/ui/Skeleton";
import { ErrorBlock } from "@/components/ui/ErrorBlock";
import { Badge } from "@/components/ui/Badge";

export default function DeliverableViewerPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const { data: deliverable, isLoading, isError } = useDeliverable(id);

  if (isLoading) {
    return (
      <div>
        <PageHeader title="Loading deliverable..." />
        <div className="space-y-4">
          <Skeleton className="h-24 w-full" />
          <Skeleton className="h-64 w-full" />
        </div>
      </div>
    );
  }

  if (isError || !deliverable) {
    return (
      <div>
        <PageHeader title="Deliverable Not Found" />
        <ErrorBlock.Page
          message="Could not load this deliverable."
          backHref="/tasks"
        />
      </div>
    );
  }

  return (
    <div>
      <PageHeader
        title={deliverable.title}
        subtitle={
          <span className="flex items-center gap-2 text-sm text-gray-500">
            <Badge color="gray" variant="outline">v{deliverable.version}</Badge>
            <span>{timeAgo(deliverable.created_at)}</span>
            {deliverable.agent_name && (
              <>
                <span className="text-gray-600">|</span>
                <span>by {deliverable.agent_name}</span>
              </>
            )}
          </span>
        }
        actions={
          <Link
            href={`/tasks/${deliverable.task_id}`}
            className="text-sm text-gray-400 hover:text-gray-200 transition-colors"
          >
            &larr; Back to Task
          </Link>
        }
      />

      <div className="rounded-lg border border-gray-800 bg-gray-950 p-6">
        {deliverable.content ? (
          <pre className="whitespace-pre-wrap break-words text-sm text-gray-300 font-sans leading-relaxed">
            {deliverable.content}
          </pre>
        ) : (
          <p className="text-sm text-gray-500">No content.</p>
        )}
      </div>
    </div>
  );
}
