"use client";

import { useUI } from "@/providers/UIProvider";
import { cn } from "@/lib/utils";

const TOAST_STYLES = {
  success: "border-emerald-800 bg-emerald-950/50 text-emerald-400",
  error: "border-red-800 bg-red-950/50 text-red-400",
  info: "border-blue-800 bg-blue-950/50 text-blue-400",
  warning: "border-amber-800 bg-amber-950/50 text-amber-400",
};

export function ToastContainer() {
  const { toasts, removeToast } = useUI();

  if (toasts.length === 0) return null;

  return (
    <div className="fixed bottom-20 right-4 z-50 flex flex-col gap-2 md:bottom-4">
      {toasts.map((toast) => (
        <div
          key={toast.id}
          className={cn(
            "animate-in fade-in slide-in-from-right-2 rounded-lg border px-4 py-2 text-sm shadow-lg",
            TOAST_STYLES[toast.type],
          )}
          role="alert"
        >
          <div className="flex items-center justify-between gap-3">
            <span>{toast.message}</span>
            <button
              onClick={() => removeToast(toast.id)}
              className="text-current opacity-60 hover:opacity-100 transition-opacity"
              type="button"
              aria-label="Dismiss"
            >
              &times;
            </button>
          </div>
        </div>
      ))}
    </div>
  );
}
