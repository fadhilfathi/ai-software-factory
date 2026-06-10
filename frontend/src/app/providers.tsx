"use client";

import { type ReactNode } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { AuthProvider } from "@/providers/AuthProvider";
import { ThemeProvider } from "@/providers/ThemeProvider";
import { UIProvider } from "@/providers/UIProvider";
import { NotificationProvider } from "@/providers/NotificationProvider";
import { VisionProvider } from "@/providers/VisionProvider";
import { RealtimeProvider } from "@/providers/RealtimeProvider";
import { AppLayout } from "@/components/layout/AppLayout";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30 * 1000, // 30s
      gcTime: 5 * 60 * 1000, // 5min
      retry: 3,
      refetchOnWindowFocus: true,
    },
  },
});

export function Providers({ children }: { children: ReactNode }) {
  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider>
        <AuthProvider>
          <NotificationProvider>
            <UIProvider>
              <VisionProvider>
                <RealtimeProvider>
                  <AppLayout>{children}</AppLayout>
                </RealtimeProvider>
              </VisionProvider>
            </UIProvider>
          </NotificationProvider>
        </AuthProvider>
      </ThemeProvider>
    </QueryClientProvider>
  );
}
