import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

type ErrorBlockProps = {
  message: string;
  title?: string;
  actions?: ReactNode;
  className?: string;
};

export function ErrorBlock({
  message,
  title,
  actions,
  className,
}: ErrorBlockProps) {
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
      <p className="text-sm text-red-400/90">{message}</p>
      {actions && <div className="mt-3 flex gap-2">{actions}</div>}
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
  // Use dynamic import to avoid forcing next/link on all consumers
  const Link = require("next/link").default;
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
