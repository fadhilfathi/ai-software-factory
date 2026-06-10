"use client";

import {
  createContext,
  useContext,
  useState,
  useCallback,
  type ReactNode,
} from "react";

type Notification = {
  id: string;
  type: "agent_task_done" | "gate_passed" | "gate_failed" | "agent_error" | "budget_alert" | "daily_summary";
  title: string;
  message: string;
  read: boolean;
  createdAt: string;
};

type NotificationContextValue = {
  queue: Notification[];
  unreadCount: number;
  push: (notification: Omit<Notification, "id" | "read" | "createdAt">) => void;
  dismiss: (id: string) => void;
  markAllRead: () => void;
};

const NotificationContext = createContext<NotificationContextValue | null>(null);

export function NotificationProvider({ children }: { children: ReactNode }) {
  const [queue, setQueue] = useState<Notification[]>([]);

  const push = useCallback(
    (n: Omit<Notification, "id" | "read" | "createdAt">) => {
      const notification: Notification = {
        ...n,
        id: `notif_${crypto.randomUUID().slice(0, 8)}`,
        read: false,
        createdAt: new Date().toISOString(),
      };
      setQueue((prev) => [notification, ...prev]);
    },
    [],
  );

  const dismiss = useCallback((id: string) => {
    setQueue((prev) => prev.filter((n) => n.id !== id));
  }, []);

  const markAllRead = useCallback(() => {
    setQueue((prev) => prev.map((n) => ({ ...n, read: true })));
  }, []);

  const unreadCount = queue.filter((n) => !n.read).length;

  return (
    <NotificationContext.Provider value={{ queue, unreadCount, push, dismiss, markAllRead }}>
      {children}
    </NotificationContext.Provider>
  );
}

export function useNotifications(): NotificationContextValue {
  const ctx = useContext(NotificationContext);
  if (!ctx) throw new Error("useNotifications must be used within <NotificationProvider>");
  return ctx;
}
