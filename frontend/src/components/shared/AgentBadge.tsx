import { type AgentType } from "@/lib/types";

type AgentBadgeProps = {
  type: AgentType;
  className?: string;
};

const AGENT_CONFIG: Record<AgentType, { label: string; color: string }> = {
  pm: { label: "PM", color: "bg-blue-500" },
  architect: { label: "Architect", color: "bg-cyan-500" },
  developer: { label: "Dev", color: "bg-emerald-500" },
  reviewer: { label: "Review", color: "bg-violet-500" },
  qa: { label: "QA", color: "bg-amber-500" },
  devops: { label: "DevOps", color: "bg-orange-500" },
  security: { label: "Security", color: "bg-red-500" },
  techwriter: { label: "Tech Writer", color: "bg-pink-500" },
};

export function AgentBadge({ type, className = "" }: AgentBadgeProps) {
  const config = AGENT_CONFIG[type];
  
  return (
    <span className={`inline-flex items-center gap-1.5 rounded-full bg-gray-800 px-2 py-0.5 text-[10px] font-medium text-gray-300 ${className}`}>
      <span className={`h-1.5 w-1.5 rounded-full ${config.color}`} />
      {config.label}
    </span>
  );
}
