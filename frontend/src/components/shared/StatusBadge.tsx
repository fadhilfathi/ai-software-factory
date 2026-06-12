import { Badge } from "@/components/ui/Badge";
import type { BadgeColor } from "@/components/ui/Badge";
import type { ProjectStatus, TaskStatus, AgentStatus_ } from "@/lib/types";

/* ─── Project Status ─── */

const PROJECT_STATUS_MAP: Record<ProjectStatus, { label: string; color: BadgeColor }> = {
  initializing: { label: "Initializing", color: "blue" },
  in_progress: { label: "In Progress", color: "emerald" },
  completed: { label: "Completed", color: "violet" },
  archived: { label: "Archived", color: "gray" },
};

type StatusBadgeProjectProps = {
  status: ProjectStatus;
  className?: string;
};

export function ProjectStatusBadge({ status, className }: StatusBadgeProjectProps) {
  const config = PROJECT_STATUS_MAP[status];
  return (
    <Badge color={config.color} className={className}>
      {config.label}
    </Badge>
  );
}

/* ─── Task Status ─── */

const TASK_STATUS_MAP: Record<TaskStatus, { label: string; color: BadgeColor }> = {
  backlog: { label: "Backlog", color: "gray" },
  ready: { label: "Ready", color: "cyan" },
  in_progress: { label: "In Progress", color: "emerald" },
  review: { label: "Review", color: "violet" },
  done: { label: "Done", color: "gray" },
  blocked: { label: "Blocked", color: "red" },
};

type StatusBadgeTaskProps = {
  status: TaskStatus;
  className?: string;
};

export function TaskStatusBadge({ status, className }: StatusBadgeTaskProps) {
  const config = TASK_STATUS_MAP[status];
  return (
    <Badge color={config.color} className={className}>
      {config.label}
    </Badge>
  );
}

/* ─── Agent Status ─── */

const AGENT_STATUS_MAP: Record<AgentStatus_, { label: string; color: BadgeColor }> = {
  spawning: { label: "Spawning", color: "blue" },
  idle: { label: "Idle", color: "gray" },
  working: { label: "Working", color: "emerald" },
  completed: { label: "Completed", color: "violet" },
  failed: { label: "Failed", color: "red" },
};

type StatusBadgeAgentProps = {
  status: AgentStatus_;
  className?: string;
};

export function AgentStatusBadge({ status, className }: StatusBadgeAgentProps) {
  const config = AGENT_STATUS_MAP[status];
  return (
    <Badge color={config.color} variant="outline" className={className}>
      {config.label}
    </Badge>
  );
}
