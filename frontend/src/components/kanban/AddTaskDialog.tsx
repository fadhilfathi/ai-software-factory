"use client";

import { useState, useEffect } from "react";
import { Input } from "@/components/form/Input";
import { Textarea } from "@/components/form/Textarea";
import { Select } from "@/components/form/Select";
import type { TaskPriority } from "@/lib/types";

const PRIORITY_OPTIONS: { value: string; label: string }[] = [
  { value: "low", label: "Low" },
  { value: "medium", label: "Medium" },
  { value: "high", label: "High" },
  { value: "critical", label: "Critical" },
];

type AddTaskDialogProps = {
  open: boolean;
  onClose: () => void;
  onSubmit: (data: {
    title: string;
    description?: string;
    priority: TaskPriority;
  }) => void;
  isSubmitting?: boolean;
};

export function AddTaskDialog({
  open,
  onClose,
  onSubmit,
  isSubmitting = false,
}: AddTaskDialogProps) {
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [priority, setPriority] = useState<string>("medium");

  useEffect(() => {
    if (!open) {
      setTitle("");
      setDescription("");
      setPriority("medium");
    }
  }, [open]);

  useEffect(() => {
    if (!open) return;
    const handler = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    document.addEventListener("keydown", handler);
    return () => document.removeEventListener("keydown", handler);
  }, [open, onClose]);

  if (!open) return null;

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!title.trim()) return;
    onSubmit({
      title: title.trim(),
      description: description.trim() || undefined,
      priority: priority as TaskPriority,
    });
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm">
      <div
        role="dialog"
        aria-modal="true"
        className="mx-4 w-full max-w-md rounded-lg border border-gray-800 bg-gray-950 p-6 shadow-xl"
      >
        <h3 className="text-lg font-semibold text-gray-200">Add Task</h3>

        <form onSubmit={handleSubmit} className="mt-4 space-y-4">
          <Input
            label="Title"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            placeholder="Task title"
            required
          />

          <Textarea
            label="Description"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="Optional description"
            rows={3}
          />

          <Select
            label="Priority"
            value={priority}
            onChange={(e) => setPriority(e.target.value)}
            options={PRIORITY_OPTIONS}
          />

          <div className="flex justify-end gap-3 pt-2">
            <button
              type="button"
              onClick={onClose}
              disabled={isSubmitting}
              className="rounded-lg border border-gray-800 px-4 py-2 text-sm font-medium text-gray-300 hover:bg-gray-800 transition-colors disabled:opacity-50"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={!title.trim() || isSubmitting}
              className="rounded-lg bg-emerald-500 px-4 py-2 text-sm font-medium text-white hover:bg-emerald-600 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {isSubmitting ? "Adding..." : "Add Task"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
