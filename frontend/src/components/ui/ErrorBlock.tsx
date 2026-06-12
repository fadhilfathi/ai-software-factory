import Link from "next/link";
import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

type ErrorBlockProps = {
  message?: string;
  error?: Error | null;
  title?: string;
  actions?: ReactNode;
  onRetry?: () => void;
  className?: string;
};

export function ErrorBlock({
  message,
  error,
  title,
  actions,
  onRetry,
  className,
}: ErrorBlockProps) {
  // Resolve message in order: explicit `message` prop → `error.message`
  // → generic fallback. The fallback keeps `message` optional in the
  // type while still always showing something to the user.
  const resolved = message ?? error?.message ?? "An unexpected error occurred.";
  return (
    <div
      className={cn(
        "rounded-lg border border-red-800 bg-red-950/50 p-4",
        className,
      )}
    >
      {title && (
        <p className="mb-1 text-sm font-semibold text-red-400">{title}</p>
      )}
      <p className="text-sm text-red-400/90">{resolved}</p>
      <div className="mt-3 flex gap-2">
        {onRetry ? (
          <button
            type="button"
            onClick={onRetry}
            className="rounded-md border border-red-700 px-3 py-1.5 text-xs font-medium text-red-300 hover:bg-red-900/30"
          >
            Retry
          </button>
        ) : null}
        {actions}
      </div>
    </div>
  );
}

type ErrorPageProps = {
  message: string;
  backHref: string;
  backLabel?: string;
};

/**
 * Full-page error state with a "Back" link. Designed for dynamic routes
 * (project detail, agent detail) where the resource fails to load.
 */
ErrorBlock.Page = function ErrorPage({
  message,
  backHref,
  backLabel = "← Back",
}: ErrorPageProps) {
  return (
    <div>
      <div className="rounded-lg border border-red-800 bg-red-950/50 p-6 text-center">
        <p className="text-sm text-red-400">{message}</p>
        <Link
          href={backHref}
          className="mt-4 inline-block text-sm text-emerald-400 hover:text-emerald-300"
        >
          {backLabel}
        </Link>
      </div>
    </div>
  );
};
