"use client";

import { type Task } from "@/lib/types";
import { Modal } from "@/components/ui/Modal";
import { PriorityBadge } from "@/components/shared/PriorityBadge";
import { TaskStatusBadge } from "@/components/shared/StatusBadge";
import { AgentBadge } from "@/components/shared/AgentBadge";

type TaskDetailModalProps = {
  task: Task | null;
  open: boolean;
  onClose: () => void;
};

export function TaskDetailModal({ task, open, onClose }: TaskDetailModalProps) {
  if (!task) return null;

  return (
    <Modal
      open={open}
      onClose={onClose}
      title="Task Details"
      size="xl"
    >
      <div className="space-y-6">
        <div>
          <div className="flex items-center gap-3 mb-1">
            <span className="text-[10px] font-mono text-gray-500">
              {task.id.toUpperCase()}
            </span>
          </div>
          <h2 className="text-xl font-bold text-gray-100">{task.title}</h2>
        </div>

        <div className="grid grid-cols-2 gap-4 rounded-lg bg-gray-900/50 p-4 border border-gray-800">
          <div>
            <span className="text-xs text-gray-500 block mb-1">Status</span>
            <TaskStatusBadge status={task.status} />
          </div>
          <div>
            <span className="text-xs text-gray-500 block mb-1">Priority</span>
            <PriorityBadge priority={task.priority} />
          </div>
          <div>
            <span className="text-xs text-gray-500 block mb-1">Assignee</span>
            {task.assignee_id ? (
              <AgentBadge type="developer" />
            ) : (
              <span className="text-sm text-gray-400">Unassigned</span>
            )}
          </div>
          <div>
            <span className="text-xs text-gray-500 block mb-1">Project ID</span>
            <span className="text-sm text-gray-400 font-mono">{task.project_id.slice(0, 8)}</span>
          </div>
        </div>

        <div>
          <span className="text-xs text-gray-500 block mb-2 uppercase tracking-wider font-semibold">Description</span>
          <div className="prose prose-invert max-w-none text-gray-300 bg-gray-900/30 p-4 rounded-lg border border-gray-800/50">
            {task.description || "No description provided."}
          </div>
        </div>

        <div className="flex justify-end pt-4 border-t border-gray-800">
          <button
            onClick={onClose}
            className="rounded-lg bg-gray-800 px-4 py-2 text-sm font-medium text-gray-200 hover:bg-gray-700 transition-colors"
          >
            Close
          </button>
        </div>
      </div>
    </Modal>
  );
}
