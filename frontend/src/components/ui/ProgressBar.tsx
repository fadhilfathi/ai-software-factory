import { cn } from "@/lib/utils";

type ProgressBarProps = {
  value: number;
  max?: number;
  size?: "sm" | "md";
  color?: "emerald" | "violet" | "blue" | "amber";
  showLabel?: boolean;
  labelPosition?: "inside" | "right" | "none";
  className?: string;
};

const COLOR_CLASSES = {
  emerald: "bg-emerald-500",
  violet: "bg-violet-500",
  blue: "bg-blue-500",
  amber: "bg-amber-500",
};

const SIZE_CLASSES = {
  sm: "h-2",
  md: "h-2.5",
};

export function ProgressBar({
  value,
  max = 100,
  size = "sm",
  color = "emerald",
  showLabel = false,
  labelPosition = "none",
  className,
}: ProgressBarProps) {
  const pct = Math.min(Math.max(0, (value / max) * 100), 100);

  return (
    <div className={cn("flex items-center gap-2", className)}>
      <div
        className={cn(
          "flex-1 overflow-hidden rounded-full bg-gray-800",
          SIZE_CLASSES[size],
        )}
        role="progressbar"
        aria-valuenow={value}
        aria-valuemin={0}
        aria-valuemax={max}
      >
        <div
          className={cn(
            "h-full rounded-full transition-all duration-500",
            COLOR_CLASSES[color],
          )}
          style={{ width: `${pct}%` }}
        />
      </div>
      {showLabel && labelPosition === "right" && (
        <span className="text-xs text-gray-400 shrink-0 font-medium">
          {Math.round(pct)}%
        </span>
      )}
      {showLabel && labelPosition === "inside" && pct > 15 && (
        <span className="absolute left-2 text-[10px] font-medium text-white">
          {Math.round(pct)}%
        </span>
      )}
    </div>
  );
}
