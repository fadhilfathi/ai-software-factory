import { Badge } from "@/components/ui/Badge";
import type { BadgeColor } from "@/components/ui/Badge";
import type { TaskPriority } from "@/lib/types";

const PRIORITY_MAP: Record<TaskPriority, { label: string; color: BadgeColor }> = {
  critical: { label: "Critical", color: "red" },
  high: { label: "High", color: "rose" },
  medium: { label: "Medium", color: "yellow" },
  low: { label: "Low", color: "gray" },
};

type PriorityBadgeProps = {
  priority: TaskPriority;
  className?: string;
  uppercase?: boolean;
};

export function PriorityBadge({
  priority,
  className,
  uppercase = true,
}: PriorityBadgeProps) {
  const config = PRIORITY_MAP[priority];
  return (
    <Badge color={config.color} className={className}>
      {uppercase ? config.label.toUpperCase() : config.label}
    </Badge>
  );
}
