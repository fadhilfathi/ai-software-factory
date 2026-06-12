"use client";

import { type Task } from "@/lib/types";
import { PriorityBadge } from "@/components/shared/PriorityBadge";
import { AgentBadge } from "@/components/shared/AgentBadge";

type TaskCardProps = {
  task: Task;
  isDragging?: boolean;
  dragHandleProps?: Record<string, unknown>;
};

export function TaskCard({ task, isDragging, dragHandleProps }: TaskCardProps) {
  return (
    <div
      className={`group rounded-lg border bg-gray-950 p-3 transition-all ${
        isDragging
          ? "border-emerald-500/50 shadow-lg shadow-emerald-500/10 opacity-90 scale-[1.02]"
          : "border-gray-800 hover:border-gray-700 hover:bg-gray-900/40"
      }`}
      {...dragHandleProps}
    >
      <div className="flex items-start justify-between gap-2">
        <span className="text-sm font-medium text-gray-200 leading-snug line-clamp-2 group-hover:text-white transition-colors">
          {task.title}
        </span>
        <span className="text-[10px] font-mono text-gray-600 shrink-0">
          {task.id.slice(0, 4).toUpperCase()}
        </span>
      </div>
      
      {task.description && (
        <p className="mt-1.5 text-[11px] text-gray-500 line-clamp-2 leading-relaxed">
          {task.description}
        </p>
      )}

      <div className="mt-3 flex items-center justify-between gap-2">
        <div className="flex items-center gap-1.5 flex-wrap">
          <PriorityBadge priority={task.priority} uppercase={false} className="px-1.5 py-0" />
        </div>
        
        {task.assignee_id ? (
          <AgentBadge type="developer" />
        ) : (
          <span className="text-[10px] text-gray-600 italic">Unassigned</span>
        )}
      </div>
    </div>
  );
}
