"use client";

import { useState } from "react";
import {
  DndContext,
  DragOverlay,
  closestCorners,
  useSensor,
  useSensors,
  PointerSensor,
  type DragEndEvent,
  type DragStartEvent,
} from "@dnd-kit/core";
import { SortableContext, verticalListSortingStrategy } from "@dnd-kit/sortable";
import type { Task, TaskStatus } from "@/lib/types";
import { KanbanColumn } from "./KanbanColumn";
import { TaskCard } from "./TaskCard";
import { AddTaskDialog } from "./AddTaskDialog";

const COLUMNS: { status: TaskStatus; title: string }[] = [
  { status: "backlog", title: "Backlog" },
  { status: "ready", title: "Ready" },
  { status: "in_progress", title: "In Progress" },
  { status: "review", title: "Review" },
  { status: "done", title: "Done" },
  { status: "blocked", title: "Blocked" },
];

type KanbanBoardProps = {
  projectId: string;
  tasks: Task[];
  isLoading?: boolean;
  onStatusChange: (taskId: string, newStatus: TaskStatus) => void;
  onAddTask: (status: TaskStatus, data: {
    title: string;
    description?: string;
    priority: "low" | "medium" | "high" | "critical";
  }) => void;
  isSubmitting?: boolean;
};

export function KanbanBoard({
  projectId,
  tasks,
  isLoading,
  onStatusChange,
  onAddTask,
  isSubmitting,
}: KanbanBoardProps) {
  const [activeTask, setActiveTask] = useState<Task | null>(null);
  const [addDialogColumn, setAddDialogColumn] = useState<TaskStatus | null>(null);

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 5 } }),
  );

  const tasksByColumn = COLUMNS.reduce(
    (acc, col) => {
      acc[col.status] = tasks.filter((t) => t.status === col.status);
      return acc;
    },
    {} as Record<TaskStatus, Task[]>,
  );

  const handleDragStart = (event: DragStartEvent) => {
    const task = tasks.find((t) => t.id === event.active.id);
    if (task) setActiveTask(task);
  };

  const handleDragEnd = (event: DragEndEvent) => {
    setActiveTask(null);
    const { active, over } = event;
    if (!over) return;

    const taskId = active.id as string;
    const task = tasks.find((t) => t.id === taskId);
    if (!task) return;

    // Find which column the item was dropped on
    const overTask = tasks.find((t) => t.id === over.id);
    let targetStatus: TaskStatus | null = null;

    if (overTask) {
      targetStatus = overTask.status;
    } else {
      // Dropped on a column (the column's id is its status)
      targetStatus = over.id as TaskStatus;
    }

    if (targetStatus && targetStatus !== task.status) {
      onStatusChange(taskId, targetStatus);
    }
  };

  if (isLoading) {
    return (
      <div className="flex gap-4 overflow-x-auto pb-4">
        {COLUMNS.map((col) => (
          <div
            key={col.status}
            className="flex w-[280px] shrink-0 flex-col rounded-lg border border-gray-800 bg-gray-900/50"
          >
            <div className="border-b border-gray-800 px-3 py-2.5">
              <div className="h-4 w-24 animate-pulse rounded bg-gray-800" />
            </div>
            <div className="space-y-2 p-3">
              {[1, 2, 3].map((i) => (
                <div
                  key={i}
                  className="h-20 animate-pulse rounded-lg border border-gray-800 bg-gray-950"
                />
              ))}
            </div>
          </div>
        ))}
      </div>
    );
  }

  return (
    <>
      <DndContext
        sensors={sensors}
        collisionDetection={closestCorners}
        onDragStart={handleDragStart}
        onDragEnd={handleDragEnd}
      >
        <div className="flex gap-4 overflow-x-auto pb-4">
          {COLUMNS.map((col) => {
            const columnTasks = tasksByColumn[col.status] ?? [];

            return (
              <SortableContext
                key={col.status}
                items={columnTasks.map((t) => t.id)}
                strategy={verticalListSortingStrategy}
              >
                <KanbanColumn
                  title={col.title}
                  statusKey={col.status}
                  tasks={columnTasks}
                  onAddTask={() => setAddDialogColumn(col.status)}
                />
              </SortableContext>
            );
          })}
        </div>

        <DragOverlay>
          {activeTask ? (
            <div className="w-[260px]">
              <TaskCard task={activeTask} isDragging />
            </div>
          ) : null}
        </DragOverlay>
      </DndContext>

      <AddTaskDialog
        open={addDialogColumn !== null}
        onClose={() => setAddDialogColumn(null)}
        onSubmit={(data) => {
          if (addDialogColumn) {
            onAddTask(addDialogColumn, data);
          }
        }}
        isSubmitting={isSubmitting}
      />
    </>
  );
}
