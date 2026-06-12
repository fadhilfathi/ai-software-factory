"use client";

import { timeAgo, cn } from "@/lib/utils";

type TimelineItem = {
  id: string;
  text: string;
  type: "success" | "error" | "info" | "warning";
  timestamp: string;
};

type ActivityTimelineProps = {
  items: TimelineItem[];
  loading?: boolean;
};

const TYPE_COLORS = {
  success: "bg-emerald-500",
  error: "bg-red-500",
  info: "bg-blue-500",
  warning: "bg-amber-500",
};

export function ActivityTimeline({ items, loading }: ActivityTimelineProps) {
  if (loading) {
    return (
      <div className="space-y-4">
        {[1, 2, 3].map((i) => (
          <div key={i} className="flex gap-4 animate-pulse">
            <div className="h-2 w-2 rounded-full bg-gray-800 mt-2" />
            <div className="flex-1 space-y-2">
              <div className="h-3 w-1/2 bg-gray-800 rounded" />
              <div className="h-2 w-1/4 bg-gray-900 rounded" />
            </div>
          </div>
        ))}
      </div>
    );
  }

  if (items.length === 0) {
    return <p className="text-xs text-gray-500 py-4 text-center">No recent activity.</p>;
  }

  return (
    <div className="space-y-6">
      {items.map((item, i) => (
        <div key={item.id} className="relative flex gap-4">
          {/* Connector Line */}
          {i !== items.length - 1 && (
            <div className="absolute left-[3.5px] top-4 bottom-[-24px] w-px bg-gray-800" />
          )}
          
          <div className={cn(
            "relative z-10 h-2 w-2 rounded-full mt-1.5 shrink-0",
            TYPE_COLORS[item.type]
          )} />
          
          <div className="flex-1 min-w-0">
            <p className="text-sm text-gray-200 leading-snug">
              {item.text}
            </p>
            <p className="mt-1 text-[10px] font-bold text-gray-500 uppercase tracking-tighter">
              {timeAgo(item.timestamp)}
            </p>
          </div>
        </div>
      ))}
    </div>
  );
}
