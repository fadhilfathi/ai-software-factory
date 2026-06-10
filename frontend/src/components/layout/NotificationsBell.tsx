"use client";

import { useNotifications } from "@/providers/NotificationProvider";
import { cn } from "@/lib/utils";

export function NotificationsBell() {
  const { unreadCount } = useNotifications();

  return (
    <button
      className="relative flex h-9 w-9 items-center justify-center rounded-lg text-gray-400 hover:bg-gray-800 hover:text-gray-200 transition-colors"
      aria-label={`Notifications${unreadCount > 0 ? ` (${unreadCount} unread)` : ""}`}
      type="button"
    >
      <span className="text-lg">🔔</span>
      {unreadCount > 0 && (
        <span
          className={cn(
            "absolute -right-1 -top-1 flex h-5 min-w-5 items-center justify-center rounded-full bg-emerald-500 px-1.5 text-[10px] font-bold text-white",
            unreadCount > 99 && "px-1 text-[9px]",
          )}
        >
          {unreadCount > 99 ? "99+" : unreadCount}
        </span>
      )}
    </button>
  );
}
