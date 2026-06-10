import { type ReactNode } from "react";
import { cn } from "@/lib/utils";

type BadgeVariant = "default" | "outline" | "subtle";

export type BadgeColor =
  | "gray"
  | "emerald"
  | "red"
  | "rose"
  | "yellow"
  | "blue"
  | "violet"
  | "cyan"
  | "amber";

type BadgeProps = {
  children: ReactNode;
  variant?: BadgeVariant;
  color?: BadgeColor;
  className?: string;
  size?: "sm" | "md";
};

const VARIANT_STYLES: Record<BadgeVariant, Record<BadgeColor, string>> = {
  default: {
    gray: "bg-gray-800 text-gray-400",
    emerald: "bg-emerald-500/10 text-emerald-400",
    red: "bg-red-500/10 text-red-400",
    rose: "bg-rose-500/10 text-rose-400",
    yellow: "bg-yellow-500/10 text-yellow-400",
    blue: "bg-blue-500/10 text-blue-400",
    violet: "bg-violet-500/10 text-violet-400",
    cyan: "bg-cyan-500/10 text-cyan-400",
    amber: "bg-amber-500/10 text-amber-400",
  },
  outline: {
    gray: "border border-gray-700 text-gray-400",
    emerald: "border border-emerald-500/20 text-emerald-400",
    red: "border border-red-500/20 text-red-400",
    rose: "border border-rose-500/20 text-rose-400",
    yellow: "border border-yellow-500/20 text-yellow-400",
    blue: "border border-blue-500/20 text-blue-400",
    violet: "border border-violet-500/20 text-violet-400",
    cyan: "border border-cyan-500/20 text-cyan-400",
    amber: "border border-amber-500/20 text-amber-400",
  },
  subtle: {
    gray: "text-gray-500",
    emerald: "text-emerald-500",
    red: "text-red-400",
    rose: "text-rose-400",
    yellow: "text-yellow-400",
    blue: "text-blue-400",
    violet: "text-violet-400",
    cyan: "text-cyan-400",
    amber: "text-amber-400",
  },
};

export function Badge({
  children,
  variant = "default",
  color = "gray",
  className,
  size = "sm",
}: BadgeProps) {
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1 rounded-full font-semibold",
        size === "sm" ? "px-2 py-0.5 text-[10px]" : "px-2.5 py-1 text-xs",
        VARIANT_STYLES[variant][color],
        className,
      )}
    >
      {children}
    </span>
  );
}
