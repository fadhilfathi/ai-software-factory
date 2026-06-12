"use client";

import { type Agent } from "@/lib/types";
import { AgentStatusBadge } from "@/components/shared/StatusBadge";
import { AgentBadge } from "@/components/shared/AgentBadge";
import { cn } from "@/lib/utils";

type AgentCardProps = {
  agent: Agent;
  onClick?: () => void;
};

export function AgentCard({ agent, onClick }: AgentCardProps) {
  return (
    <div
      onClick={onClick}
      className={cn(
        "group relative flex flex-col rounded-xl border border-gray-800 bg-gray-950 p-5 transition-all hover:border-gray-700 hover:bg-gray-900/40 cursor-pointer",
        agent.status === "working" && "ring-1 ring-emerald-500/20"
      )}
    >
      <div className="flex items-start justify-between gap-4">
        <div className="flex items-center gap-3">
          <div className="relative">
            <div className="h-12 w-12 rounded-full bg-gray-900 border border-gray-800 flex items-center justify-center text-xl shadow-inner">
              {agent.type === "pm" ? "📈" : agent.type === "architect" ? "📐" : "🤖"}
            </div>
            <div className={cn(
              "absolute -bottom-0.5 -right-0.5 h-3.5 w-3.5 rounded-full border-2 border-gray-950",
              agent.status === "working" ? "bg-emerald-500 animate-pulse" : 
              agent.status === "idle" ? "bg-gray-500" : 
              agent.status === "failed" ? "bg-red-500" : "bg-blue-500"
            )} />
          </div>
          <div>
            <h3 className="font-semibold text-gray-100 group-hover:text-emerald-400 transition-colors">
              {agent.name}
            </h3>
            <p className="text-[10px] uppercase tracking-widest font-bold text-gray-500">
              {agent.role}
            </p>
          </div>
        </div>
        <AgentStatusBadge status={agent.status} />
      </div>

      <div className="mt-6 grid grid-cols-2 gap-4 border-t border-gray-900 pt-4">
        <div className="space-y-1">
          <span className="text-[10px] text-gray-600 uppercase font-bold tracking-tight">Tasks Done</span>
          <p className="text-sm font-medium text-gray-300">{agent.tasks_completed ?? 0}</p>
        </div>
        <div className="space-y-1 text-right">
          <span className="text-[10px] text-gray-600 uppercase font-bold tracking-tight">Model</span>
          <p className="text-[10px] font-mono text-gray-400 truncate max-w-[100px] ml-auto">
            {agent.model || "default"}
          </p>
        </div>
      </div>

      <div className="mt-4 space-y-2">
        <div className="flex justify-between text-[10px] text-gray-500 uppercase font-bold">
          <span>Utilization</span>
          <span className="text-gray-400">{agent.status === "working" ? "85%" : "0%"}</span>
        </div>
        <div className="h-1.5 w-full rounded-full bg-gray-900 overflow-hidden">
          <div 
            className={cn(
              "h-full rounded-full transition-all duration-500",
              agent.status === "working" ? "bg-emerald-500 w-[85%]" : "bg-gray-800 w-0"
            )}
          />
        </div>
      </div>

      {agent.current_task_id && (
        <div className="mt-4 rounded-lg bg-gray-900/50 p-2 border border-gray-800/50">
          <span className="text-[10px] text-gray-600 uppercase font-bold block mb-1">Current Task</span>
          <p className="text-[11px] text-gray-300 truncate font-mono">
            {agent.current_task_id}
          </p>
        </div>
      )}
    </div>
  );
}
