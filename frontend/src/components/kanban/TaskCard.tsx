"use client";

import { type Task } from "@/lib/types";
import { TaskStatusBadge } from "@/components/shared/StatusBadge";
import { PriorityBadge } from "@/components/shared/PriorityBadge";

type TaskCardProps = {
  task: Task;
  isDragging?: boolean;
  dragHandleProps?: Record<string, unknown>;
};

export function TaskCard({ task, isDragging, dragHandleProps }: TaskCardProps) {
  return (
    <div
      className={`rounded-lg border bg-gray-950 p-3 transition-colors ${
        isDragging
          ? "border-emerald-500/50 shadow-lg shadow-emerald-500/10 opacity-90"
          : "border-gray-800 hover:border-gray-700"
      }`}
      {...dragHandleProps}
    >
      <div className="flex items-start justify-between gap-2">
        <span className="text-sm font-medium text-gray-200 leading-snug line-clamp-2">
          {task.title}
        </span>
      </div>
      {task.description && (
        <p className="mt-1 text-[11px] text-gray-500 line-clamp-2">
          {task.description}
        </p>
      )}
      <div className="mt-2 flex items-center gap-2 flex-wrap">
        <PriorityBadge priority={task.priority} uppercase={false} />
        {task.assignee_id && (
          <span className="inline-flex items-center gap-1 rounded bg-gray-800 px-1.5 py-0.5 text-[10px] text-gray-400">
            @{task.assignee_id.slice(0, 8)}
          </span>
        )}
      </div>
    </div>
  );
}
