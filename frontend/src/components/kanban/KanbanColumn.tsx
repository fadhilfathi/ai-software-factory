"use client";

import { useSortable } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { type Task } from "@/lib/types";
import { TaskCard } from "./TaskCard";

type KanbanColumnProps = {
  title: string;
  statusKey: string;
  tasks: Task[];
  isOver?: boolean;
  setNodeRef?: (node: HTMLElement | null) => void;
  onAddTask?: () => void;
};

const STATUS_COLORS: Record<string, string> = {
  backlog: "border-l-gray-500",
  ready: "border-l-cyan-500",
  in_progress: "border-l-emerald-500",
  review: "border-l-violet-500",
  done: "border-l-gray-500",
  blocked: "border-l-red-500",
};

function SortableTaskCard({ task }: { task: Task }) {
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id: task.id });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
  };

  return (
    <div ref={setNodeRef} style={style} {...attributes} {...listeners}>
      <TaskCard task={task} isDragging={isDragging} />
    </div>
  );
}

export function KanbanColumn({
  title,
  statusKey,
  tasks,
  isOver,
  setNodeRef,
  onAddTask,
}: KanbanColumnProps) {
  return (
    <div
      ref={setNodeRef}
      className={`flex w-[280px] shrink-0 flex-col rounded-lg border border-gray-800 bg-gray-900/50 ${
        isOver ? "border-emerald-500/50" : ""
      }`}
    >
      <div
        className={`flex items-center justify-between rounded-t-lg border-b border-gray-800 px-3 py-2.5 border-l-2 ${
          STATUS_COLORS[statusKey] ?? "border-l-gray-500"
        }`}
      >
        <div className="flex items-center gap-2">
          <h3 className="text-sm font-semibold text-gray-200">{title}</h3>
          <span className="flex h-5 min-w-5 items-center justify-center rounded-full bg-gray-800 px-1.5 text-[10px] font-medium text-gray-400">
            {tasks.length}
          </span>
        </div>
        {onAddTask && (
          <button
            onClick={onAddTask}
            className="flex h-6 w-6 items-center justify-center rounded text-gray-500 hover:bg-gray-800 hover:text-gray-200 transition-colors"
            aria-label={`Add task to ${title}`}
            type="button"
          >
            +
          </button>
        )}
      </div>

      <div className="flex-1 space-y-2 overflow-y-auto p-3">
        {tasks.length === 0 ? (
          <p className="py-8 text-center text-xs text-gray-600">No tasks</p>
        ) : (
          tasks.map((task) => <SortableTaskCard key={task.id} task={task} />)
        )}
      </div>
    </div>
  );
}
