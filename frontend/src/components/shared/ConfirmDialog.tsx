"use client";

import { useEffect, useRef } from "react";
import { cn } from "@/lib/utils";

type ConfirmDialogProps = {
  open: boolean;
  onConfirm: () => void;
  onCancel: () => void;
  title: string;
  message?: string;
  /**
   * Alias of `message` for Sprint 4 pages that prefer the "description"
   * naming. If both are given, `message` wins (back-compat).
   */
  description?: string;
  confirmLabel?: string;
  cancelLabel?: string;
  variant?: "danger" | "default";
  /**
   * Shorthand for `variant="danger"` for the Sprint 4 page styles.
   * Ignored if `variant` is also given.
   */
  destructive?: boolean;
  loading?: boolean;
};

export function ConfirmDialog({
  open,
  onConfirm,
  onCancel,
  title,
  message,
  description,
  confirmLabel = "Confirm",
  cancelLabel = "Cancel",
  variant,
  destructive,
  loading = false,
}: ConfirmDialogProps) {
  const text = message ?? description;
  const resolvedVariant = variant ?? (destructive ? "danger" : "default");
  const dialogRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    const handler = (e: KeyboardEvent) => {
      if (e.key === "Escape") onCancel();
    };
    document.addEventListener("keydown", handler);
    return () => document.removeEventListener("keydown", handler);
  }, [open, onCancel]);

  if (!open) return null;

  const confirmStyles =
    resolvedVariant === "danger"
      ? "bg-red-500 text-white hover:bg-red-600"
      : "bg-emerald-500 text-white hover:bg-emerald-600";

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm">
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        className="mx-4 w-full max-w-md rounded-lg border border-gray-800 bg-gray-950 p-6 shadow-xl"
      >
        <h3 className="text-lg font-semibold text-gray-200">{title}</h3>
        <p className="mt-2 text-sm text-gray-400">{text}</p>
        <div className="mt-6 flex justify-end gap-3">
          <button
            onClick={onCancel}
            disabled={loading}
            className="rounded-lg border border-gray-800 px-4 py-2 text-sm font-medium text-gray-300 hover:bg-gray-800 transition-colors disabled:opacity-50"
            type="button"
          >
            {cancelLabel}
          </button>
          <button
            onClick={onConfirm}
            disabled={loading}
            className={cn(
              "rounded-lg px-4 py-2 text-sm font-medium transition-colors disabled:opacity-50",
              confirmStyles,
            )}
            type="button"
          >
            {loading ? `${confirmLabel}...` : confirmLabel}
          </button>
        </div>
      </div>
    </div>
  );
}
