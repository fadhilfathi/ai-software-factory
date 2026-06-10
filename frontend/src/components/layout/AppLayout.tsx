"use client";

import { type ReactNode } from "react";
import { SidebarNav } from "./SidebarNav";
import { MobileNav } from "./MobileNav";
import { Breadcrumb } from "./Breadcrumb";
import { NotificationsBell } from "./NotificationsBell";
import { useUI } from "@/providers/UIProvider";
import { cn } from "@/lib/utils";
import { ToastContainer } from "@/components/ui/ToastContainer";

type AppLayoutProps = {
  children: ReactNode;
};

export function AppLayout({ children }: AppLayoutProps) {
  const { sidebarCollapsed, setSidebarCollapsed } = useUI();

  return (
    <div className="min-h-screen bg-gray-900">
      {/* Desktop Sidebar */}
      <SidebarNav collapsed={sidebarCollapsed} />

      {/* Toggle button (desktop only) */}
      <button
        onClick={() => setSidebarCollapsed(!sidebarCollapsed)}
        className={cn(
          "fixed top-4 z-40 hidden h-8 w-8 items-center justify-center rounded-md bg-gray-800 text-gray-400 hover:bg-gray-700 hover:text-gray-200 transition-colors md:flex",
          sidebarCollapsed ? "left-[4.5rem]" : "left-[17rem]",
        )}
        aria-label={sidebarCollapsed ? "Expand sidebar" : "Collapse sidebar"}
        type="button"
      >
        {sidebarCollapsed ? "→" : "←"}
      </button>

      {/* Main content area */}
      <div
        className={cn(
          "transition-all duration-200",
          sidebarCollapsed ? "md:ml-16" : "md:ml-64",
        )}
      >
        {/* Top bar */}
        <header className="sticky top-0 z-20 flex h-16 items-center justify-end gap-4 border-b border-gray-800 bg-gray-900/80 px-6 backdrop-blur-sm">
          <NotificationsBell />
        </header>

        {/* Page content */}
        <main className="px-6 py-6">
          <Breadcrumb />
          {children}
        </main>
      </div>

      {/* Mobile bottom nav */}
      <MobileNav />

      <ToastContainer />
    </div>
  );
}
