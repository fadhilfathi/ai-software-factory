"use client";

import Link from "next/link";
import { type Project } from "@/lib/types";
import { ProjectStatusBadge } from "@/components/shared/StatusBadge";
import { ProgressBar } from "@/components/ui/ProgressBar";
import { timeAgo } from "@/lib/utils";
import { AgentBadge } from "@/components/shared/AgentBadge";

type ProjectCardProps = {
  project: Project;
};

export function ProjectCard({ project }: ProjectCardProps) {
  return (
    <Link
      href={`/projects/${project.id}`}
      className="group relative flex flex-col rounded-xl border border-gray-800 bg-gray-950 p-5 transition-all hover:border-gray-700 hover:bg-gray-900/40 hover:shadow-xl hover:shadow-emerald-500/5"
    >
      <div className="flex items-start justify-between gap-4">
        <div className="flex-1 min-w-0">
          <h3 className="text-base font-semibold text-gray-100 truncate group-hover:text-emerald-400 transition-colors">
            {project.name}
          </h3>
          <p className="mt-1 text-xs text-gray-500 line-clamp-2 leading-relaxed">
            {project.description || "No description provided."}
          </p>
        </div>
        <ProjectStatusBadge status={project.status} className="shrink-0" />
      </div>

      <div className="mt-6 space-y-3">
        <div className="flex items-center justify-between text-[10px] uppercase tracking-wider font-bold text-gray-500">
          <span>Overall Progress</span>
          <span className="text-gray-300">{project.progress ?? 0}%</span>
        </div>
        <ProgressBar 
          value={project.progress ?? 0} 
          color={project.status === "completed" ? "violet" : "emerald"} 
          size="sm" 
        />
      </div>

      <div className="mt-auto pt-6 flex items-center justify-between">
        <div className="flex -space-x-2">
          {project.active_agents && project.active_agents > 0 ? (
            Array.from({ length: Math.min(project.active_agents, 3) }).map((_, i) => (
              <div 
                key={i} 
                className="h-6 w-6 rounded-full border-2 border-gray-950 bg-gray-800 flex items-center justify-center"
                title="Active Agent"
              >
                <div className="h-1.5 w-1.5 rounded-full bg-emerald-500 animate-pulse" />
              </div>
            ))
          ) : (
            <span className="text-[10px] text-gray-600">No active agents</span>
          )}
          {project.active_agents && project.active_agents > 3 && (
            <div className="h-6 w-6 rounded-full border-2 border-gray-950 bg-gray-900 flex items-center justify-center text-[8px] font-bold text-gray-400">
              +{project.active_agents - 3}
            </div>
          )}
        </div>
        
        <div className="flex items-center gap-2 text-[10px] text-gray-500">
          <span className="h-1 w-1 rounded-full bg-gray-700" />
          <span>Updated {timeAgo(project.updated_at)}</span>
        </div>
      </div>
      
      {/* Decorative corner element */}
      <div className="absolute top-0 right-0 h-16 w-16 overflow-hidden rounded-tr-xl pointer-events-none opacity-0 group-hover:opacity-100 transition-opacity">
        <div className="absolute top-[-24px] right-[-24px] h-12 w-12 rotate-45 bg-emerald-500/10" />
      </div>
    </Link>
  );
}
