"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { cn } from "@/lib/utils";

const navItems = [
  { href: "/dashboard", label: "Dashboard", icon: "▦" },
  { href: "/projects", label: "Projects", icon: "▣" },
  { href: "/tasks", label: "Tasks", icon: "☰" },
  { href: "/agents", label: "Agents", icon: "◎" },
  { href: "/settings", label: "Settings", icon: "⚙" },
];

export function SidebarNav({ collapsed }: { collapsed: boolean }) {
  const pathname = usePathname();

  return (
    <nav
      className={cn(
        "fixed left-0 top-0 z-30 flex h-full flex-col border-r border-gray-800 bg-gray-950 transition-all duration-200",
        collapsed ? "w-16" : "w-64",
      )}
      role="navigation"
      aria-label="Main navigation"
    >
      {/* Logo / Brand */}
      <div className="flex h-16 items-center justify-center border-b border-gray-800 px-4">
        {collapsed ? (
          <span className="text-xl font-bold text-emerald-400">AF</span>
        ) : (
          <span className="text-lg font-bold text-emerald-400">AI Software Factory</span>
        )}
      </div>

      {/* Nav Items */}
      <div className="flex-1 space-y-1 px-3 py-4">
        {navItems.map((item) => {
          const isActive = pathname.startsWith(item.href);
          return (
            <Link
              key={item.href}
              href={item.href}
              className={cn(
                "flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-colors",
                isActive
                  ? "bg-emerald-500/10 text-emerald-400"
                  : "text-gray-400 hover:bg-gray-800 hover:text-gray-200",
              )}
              aria-current={isActive ? "page" : undefined}
            >
              <span className="flex h-5 w-5 items-center justify-center text-base">{item.icon}</span>
              {!collapsed && <span>{item.label}</span>}
            </Link>
          );
        })}
      </div>
    </nav>
  );
}
