"use client";

import { useEffect, type ReactNode } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { getRealtimeClient, useRealtime } from "@/lib/realtime";
import { useNotifications } from "./NotificationProvider";
import { queryKeys } from "@/lib/queryKeys";

/**
 * RealtimeProvider wires incoming WebSocket events to:
 *   1. React Query cache invalidation (refetches stale data)
 *   2. NotificationProvider push (shows toast/bell alerts)
 *
 * Render this inside <QueryClientProvider> and <NotificationProvider>.
 */
export function RealtimeProvider({ children }: { children: ReactNode }) {
  const qc = useQueryClient();
  const { push } = useNotifications();
  const { state } = useRealtime({ autoConnect: true });

  // Wire auth token from the singleton API client into the WS client
  useEffect(() => {
    const client = getRealtimeClient();
    // Dynamically import to avoid circular dep
    import("@/lib/api").then(({ getAccessToken }) => {
      client.setAccessTokenProvider(() => getAccessToken());
    });
  }, []);

  // ─── Event → Query invalidation map ────────────────────────────────────
  useEffect(() => {
    const client = getRealtimeClient();

    const unsubs = [
      // Agent status changes → refetch agent list
      client.on("agent_status", () => {
        qc.invalidateQueries({ queryKey: queryKeys.agents.all });
      }),

      // Task updates → refetch tasks and the parent project
      client.on("task_update", (event) => {
        qc.invalidateQueries({ queryKey: queryKeys.tasks.all });
        if ("project_id" in event && typeof event.project_id === "string") {
          qc.invalidateQueries({
            queryKey: queryKeys.projects.detail(event.project_id),
          });
        }
      }),

      // Project updates → refetch project list + detail
      client.on("project_update", (event) => {
        qc.invalidateQueries({ queryKey: queryKeys.projects.all });
        if ("project_id" in event && typeof event.project_id === "string") {
          qc.invalidateQueries({
            queryKey: queryKeys.projects.detail(event.project_id),
          });
        }
      }),

      // Activity feed → refresh
      client.on("activity", () => {
        qc.invalidateQueries({ queryKey: queryKeys.dashboard.activity() });
      }),

      // Notifications → push into notification bell
      client.on("notification", (event) => {
        if ("notification" in event && event.notification) {
          const notif = event.notification as {
            type: string;
            title: string;
            message: string;
          };
          push({
            type: notif.type as "agent_task_done" | "gate_passed" | "gate_failed" | "agent_error" | "budget_alert" | "daily_summary",
            title: notif.title,
            message: notif.message,
          });
        }
      }),
    ];

    return () => unsubs.forEach((u) => u());
  }, [qc, push]);

  return <>{children}</>;
}
