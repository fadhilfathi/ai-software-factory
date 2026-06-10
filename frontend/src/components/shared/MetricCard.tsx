import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

type MetricCardProps = {
  label: string;
  value: string | number;
  trend?: string;
  trendUp?: boolean;
  trendNeutral?: boolean;
  icon?: ReactNode;
  loading?: boolean;
  className?: string;
};

export function MetricCard({
  label,
  value,
  trend,
  trendUp,
  trendNeutral,
  icon,
  loading = false,
  className,
}: MetricCardProps) {
  return (
    <div
      className={cn(
        "rounded-lg border border-gray-800 bg-gray-950 p-4",
        className,
      )}
    >
      <div className="flex items-start justify-between">
        <p className="text-sm text-gray-400">{label}</p>
        {icon && <div className="text-gray-500">{icon}</div>}
      </div>
      <div className="mt-1 flex items-baseline gap-2">
        <span className="text-2xl font-bold text-gray-100">
          {loading ? "..." : value}
        </span>
        {trend && (
          <span
            className={cn(
              "text-sm",
              trendNeutral
                ? "text-gray-500"
                : trendUp
                  ? "text-emerald-400"
                  : "text-red-400",
            )}
          >
            {trend}
          </span>
        )}
      </div>
    </div>
  );
}
